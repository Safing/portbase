package utils

import (
	"fmt"
	"os"
	"runtime"
)

const isWindows = runtime.GOOS == "windows"

// EnsureDirectory ensures that the given directory exists and that is has the given permissions set.
// If path is a file, it is deleted and a directory created.
// If a directory is created, also all missing directories up to the required one are created with the given permissions.
func EnsureDirectory(path string, perm os.FileMode) error {
	// open path
	f, err := os.Stat(path)
	if err == nil {
		// file exists
		if f.IsDir() {
			// directory exists, check permissions
			if isWindows {
				// TODO: set correct permission on windows
				// acl.Chmod(path, perm)
			} else if f.Mode().Perm() != perm {
				return os.Chmod(path, perm)
			}
			return nil
		}
		err = os.Remove(path)
		if err != nil {
			return fmt.Errorf("could not remove file %s to place dir: %s", path, err)
		}
	}
	// file does not exist (or has been deleted)
	if err == nil || os.IsNotExist(err) {
		err = os.MkdirAll(path, perm)
		if err != nil {
			return fmt.Errorf("could not create dir %s: %s", path, err)
		}
		return nil
	}
	// other error opening path
	return fmt.Errorf("failed to access %s: %s", path, err)
}
