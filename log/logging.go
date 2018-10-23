// Copyright Safing ICS Technologies GmbH. Use of this source code is governed by the AGPL license that can be found in the LICENSE file.

package log

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tevino/abool"
)

// concept
/*
- Logging function:
  - check if file-based levelling enabled
    - if yes, check if level is active on this file
  - check if level is active
  - send data to backend via big buffered channel
- Backend:
  - wait until there is time for writing logs
  - write logs
  - configurable if logged to folder (buffer + rollingFileAppender) and/or console
  - console: log everything above INFO to stderr
- Channel overbuffering protection:
  - if buffer is full, trigger write
- Anti-Importing-Loop:
  - everything imports logging
  - logging is configured by main module and is supplied access to configuration and taskmanager
*/

type severity uint32

type logLine struct {
	msg   string
	level severity
	time  time.Time
	file  string
	line  int
}

const (
	TraceLevel    severity = 1
	DebugLevel    severity = 2
	InfoLevel     severity = 3
	WarningLevel  severity = 4
	ErrorLevel    severity = 5
	CriticalLevel severity = 6
)

var (
	logBuffer             chan *logLine
	forceEmptyingOfBuffer chan bool

	logLevelInt = uint32(3)
	logLevel    = &logLevelInt

	fileLevelsActive = abool.NewBool(false)
	fileLevels       = make(map[string]severity)
	fileLevelsLock   sync.Mutex

	logsWaiting     = make(chan bool, 1)
	logsWaitingFlag = abool.NewBool(false)

	shutdownSignal    = make(chan struct{}, 0)
	shutdownWaitGroup sync.WaitGroup

	started    = abool.NewBool(false)
	testErrors = abool.NewBool(false)
)

func SetFileLevels(levels map[string]severity) {
	fileLevelsLock.Lock()
	fileLevels = levels
	fileLevelsLock.Unlock()
	fileLevelsActive.Set()
}

func UnSetFileLevels() {
	fileLevelsActive.UnSet()
}

func SetLogLevel(level severity) {
	atomic.StoreUint32(logLevel, uint32(level))
}

func ParseLevel(level string) severity {
	switch strings.ToLower(level) {
	case "trace":
		return 1
	case "debug":
		return 2
	case "info":
		return 3
	case "warning":
		return 4
	case "error":
		return 5
	case "critical":
		return 6
	}
	return 0
}

func Start() error {

	if !started.SetToIf(false, true) {
		return nil
	}

	logBuffer = make(chan *logLine, 8192)
	forceEmptyingOfBuffer = make(chan bool, 4)

	initialLogLevel := ParseLevel(logLevelFlag)
	if initialLogLevel > 0 {
		atomic.StoreUint32(logLevel, uint32(initialLogLevel))
	} else {
		return fmt.Errorf("invalid log level \"%s\", falling back to level info", logLevelFlag)
	}

	// get and set file loglevels
	fileLogLevels := fileLogLevelsFlag
	if len(fileLogLevels) > 0 {
		newFileLevels := make(map[string]severity)
		for _, pair := range strings.Split(fileLogLevels, ",") {
			splitted := strings.Split(pair, "=")
			if len(splitted) != 2 {
				return fmt.Errorf("invalid file log level \"%s\", ignoring", pair)
			}
			fileLevel := ParseLevel(splitted[1])
			if fileLevel == 0 {
				return fmt.Errorf("invalid file log level \"%s\", ignoring", pair)
			}
			newFileLevels[splitted[0]] = fileLevel
		}
		SetFileLevels(newFileLevels)
	}

	fmt.Println(fmt.Sprintf("%s%s ▶ BOF%s", InfoLevel.color(), time.Now().Format("060102 15:04:05.000"), endColor()))
	go writer()

	return nil
}

// Shutdown writes remaining log lines and then stops the logger.
func Shutdown() {
	close(shutdownSignal)
	shutdownWaitGroup.Wait()
}
