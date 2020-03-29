package gopathfs

import (
	"log"
	"os"
	"path/filepath"

	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
	"github.com/linuxerwang/goplz/mapping"
	"golang.org/x/sys/unix"
)

// Open overrides the parent's Open method.
func (gpf *GoPathFs) Open(virtual string, flags uint32, context *fuse.Context) (file nodefs.File, st fuse.Status) {
	if verbose {
		log.Printf("open virtual file %s\n", virtual)
	}
	entry, remPath := gpf.vfs.MatchPath(virtual)
	if len(remPath) != 0 {
		return nil, fuse.ENOENT
	}

	flag := int(flags)
	if entry.Readonly() {
		flag = os.O_RDONLY
	}
	f, err := os.OpenFile(entry.Actual(), flag, 0)
	if err != nil {
		log.Printf("Failed to open virtual file: %s => %s, %+v.\n", virtual, entry.Actual(), err)
		return nil, fuse.EIO
	}

	return nodefs.NewLoopbackFile(f), fuse.OK
}

// Create overrides the parent's Create method.
func (gpf *GoPathFs) Create(virtual string, flags uint32, mode uint32,
	context *fuse.Context) (file nodefs.File, st fuse.Status) {

	if verbose {
		log.Printf("create virtual file %s\n", virtual)
	}
	entry, remPath := gpf.vfs.MatchPath(virtual)
	if len(remPath) != 1 {
		if verbose {
			log.Printf("Failed to create virtual file %s, extra remaining paths %v\n", virtual, remPath)
		}
		return nil, fuse.EINVAL
	}
	if entry.Readonly() {
		return nil, fuse.EROFS
	}

	actual := filepath.Join(entry.Actual(), remPath[0])
	f, err := os.OpenFile(actual, int(flags), os.FileMode(mode))
	if err != nil {
		log.Printf("Failed to create virtual file %s => %s, %v\n", virtual, entry.Actual(), err)
		return nil, fuse.EINVAL
	}
	gpf.vfs.Track(virtual, actual, false)
	return nodefs.NewLoopbackFile(f), fuse.OK
}

// Unlink overrides the parent's Unlink method.
func (gpf *GoPathFs) Unlink(virtual string, context *fuse.Context) (st fuse.Status) {
	if verbose {
		log.Printf("unlink virtual file %s\n", virtual)
	}
	entry, remPath := gpf.vfs.MatchPath(virtual)
	if len(remPath) != 0 {
		return fuse.ENOENT
	}
	if entry.Readonly() {
		return fuse.EROFS
	}

	if err := unix.Unlink(entry.Actual()); err != nil {
		log.Printf("Failed to unlink virtual file %s => %s, %v\n", virtual, entry.Actual(), err)
		return fuse.EINVAL
	}

	if err := gpf.vfs.Untrack(virtual); err != nil {
		log.Printf("Failed to untrack virtual file %s, %v\n", virtual, err)
		return fuse.EINVAL
	}
	return fuse.OK
}

// Rename overrides the parent's Rename method.
func (gpf *GoPathFs) Rename(oldVirtual string, newVirtual string, context *fuse.Context) (st fuse.Status) {
	if verbose {
		log.Printf("rename virtual file %s to %s\n", oldVirtual, newVirtual)
	}

	entry, remPath := gpf.vfs.MatchPath(oldVirtual)
	if len(remPath) != 0 {
		return fuse.ENOENT
	}
	if entry.Readonly() {
		log.Printf("failed to rename readonly virtual file %s to %s", oldVirtual, newVirtual)
		return fuse.EROFS
	}

	dir, remPath := gpf.vfs.MatchPath(newVirtual)
	newActual := filepath.Join(dir.Actual(), filepath.Join(remPath...))
	if verbose {
		log.Printf("rename actual file %s to %s", entry.Actual(), newActual)
	}
	if err := os.Rename(entry.Actual(), newActual); err != nil {
		log.Printf("Failed to rename %s to %s, %v", entry.Actual(), newActual, err)
		return fuse.EINVAL
	}

	if err := gpf.vfs.Untrack(oldVirtual); err != nil {
		log.Printf("Failed to untrack virtual file %s, %v\n", oldVirtual, err)
	}
	filepath.Walk(dir.Actual(), func(actual string, info os.FileInfo, err error) error {
		virtual, readonly, st := gpf.mapper.Map(actual)
		if st == mapping.Excluded {
			return filepath.SkipDir
		}
		if virtual != "" {
			gpf.vfs.Track(virtual, actual, readonly)
		}
		return nil
	})
	return fuse.OK
}
