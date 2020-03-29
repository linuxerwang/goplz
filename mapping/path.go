package mapping

import "strings"

import "os"

const pathSeparator = string(os.PathSeparator)

// ContainsDir returns true if path contains dir.
func ContainsDir(path, dir string) bool {
	if path == dir {
		return true
	}
	if strings.HasPrefix(path, dir+pathSeparator) {
		return true
	}
	if strings.HasSuffix(path, pathSeparator+dir) {
		return true
	}
	if strings.Contains(path, pathSeparator+dir+pathSeparator) {
		return true
	}
	return false
}
