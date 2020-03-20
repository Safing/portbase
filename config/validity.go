package config

import (
	"github.com/tevino/abool"
)

// ValidityFlag is a flag that signifies if the configuration has been changed. It is not safe for concurrent use.
type ValidityFlag struct {
	flag *abool.AtomicBool
}

// NewValidityFlag returns a flag that signifies if the configuration has been changed.
func NewValidityFlag() *ValidityFlag {
	vf := &ValidityFlag{}
	vf.Refresh()
	return vf
}

// IsValid returns if the configuration is still valid.
func (vf *ValidityFlag) IsValid() bool {
	return vf.flag.IsSet()
}

// Refresh refreshes the flag and makes it reusable.
func (vf *ValidityFlag) Refresh() {
	validityFlagLock.RLock()
	defer validityFlagLock.RUnlock()

	vf.flag = validityFlag
}
