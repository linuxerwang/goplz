package gopathfs

import (
	"log"
	"os"
	"path/filepath"

	"github.com/hanwen/go-fuse/fuse"
)

// OpenDir overrides the parent's OpenDir method.
func (gpf *GoPathFs) OpenDir(virtual string, context *fuse.Context) ([]fuse.DirEntry, fuse.Status) {
	if verbose {
		log.Printf("open virtual directory %s\n", virtual)
	}
	entry, remPath := gpf.vfs.MatchPath(virtual)
	if len(remPath) != 0 {
		return nil, fuse.ENOENT
	}

	return entry.Children(), fuse.OK
}

// Mkdir overrides the parent's Mkdir method.
func (gpf *GoPathFs) Mkdir(virtual string, mode uint32, context *fuse.Context) fuse.Status {
	if verbose {
		log.Printf("make virtual directory %s\n", virtual)
	}
	entry, remPath := gpf.vfs.MatchPath(virtual)
	if len(remPath) == 0 {
		return fuse.EINVAL
	}
	if entry.Readonly() {
		return fuse.EROFS
	}

	actual := filepath.Join(entry.Actual(), filepath.Join(remPath...))
	if err := os.MkdirAll(actual, os.ModePerm); err != nil {
		return fuse.EINVAL
	}
	gpf.vfs.Track(virtual, actual, false)
	return fuse.OK
}

// Rmdir overrides the parent's Rmdir method.
func (gpf *GoPathFs) Rmdir(virtual string, context *fuse.Context) fuse.Status {
	if verbose {
		log.Printf("delete vitual directory %s\n", virtual)
	}
	if err := gpf.vfs.Untrack(virtual); err != nil {
		return fuse.EINVAL
	}
	return fuse.OK
}
