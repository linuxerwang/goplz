package vfs

import (
	"bytes"
	"io"
	"log"
	"os"
	"strings"

	"github.com/hanwen/go-fuse/fuse"
	cli "github.com/urfave/cli/v2"
	"golang.org/x/sys/unix"
)

var (
	verbose       bool
	pathSeparator = string(os.PathSeparator)

	defaultDirAttr = fuse.Attr{
		Mode: fuse.S_IFDIR | 0755,
	}
)

// Init initialize the gopathfs package.
func Init(ctx *cli.Context) {
	verbose = ctx.Bool("verbose")
}

// FileSystem is an interface to abstract file systems.
type FileSystem interface {
	// MatchPath finds the actual file for the given virtual file, returning
	// the deepest Entry object matched in the virtual file system and the
	// remaining unmatched paths.
	MatchPath(virtual string) (Entry, []string)

	// Track tracks the mapping from virtual file to the actual file.
	Track(virtual, actual string, readonly bool)

	// Untrack removes the mapping from the given virtual file.
	Untrack(virtual string) error

	String() string
}

type fileSystem struct {
	root   entry
	actual string
}

// Make sure *fileSystem implements FileSystem.
var _ = (FileSystem)((*fileSystem)(nil))

func (fs *fileSystem) MatchPath(virtual string) (matched Entry, remPath []string) {
	if virtual == "" || virtual == "." {
		return &fs.root, nil
	}

	dirs := strings.Split(virtual, pathSeparator)
	var idx int
	var p Entry = &fs.root
	for ; idx < len(dirs); idx++ {
		if c := p.GetChild(dirs[idx]); c != nil {
			p = c
		} else {
			break
		}
	}

	return p, dirs[idx:]
}

func (fs *fileSystem) Track(virtual, actual string, readonly bool) {
	if verbose {
		log.Printf("track file %s => %s\n", virtual, actual)
	}
	parent, remPath := fs.MatchPath(virtual)
	for i, rp := range remPath {
		e := entry{
			virtual:  rp,
			parent:   parent,
			children: map[string]Entry{},
			readonly: readonly,
		}
		if i == len(remPath)-1 {
			e.actual = actual
		}
		parent.SetChild(rp, &e)
		parent = &e
	}
}

func (fs *fileSystem) Untrack(virtual string) error {
	if verbose {
		log.Printf("untrack file %s\n", virtual)
	}
	parent, remPath := fs.MatchPath(virtual)
	if len(remPath) > 0 {
		return os.ErrNotExist
	}
	if err := os.RemoveAll(parent.Actual()); err != nil {
		return err
	}
	if parent.Parent() == nil {
		return nil
	}
	parent.Parent().DeleteChild(parent.Virtual())
	return nil
}

func (fs *fileSystem) String() string {
	var buf bytes.Buffer
	fs.printEntry(&fs.root, &buf, "")
	return buf.String()
}

func (fs *fileSystem) printEntry(e Entry, w io.Writer, prefix string) {
	e.Print(w, prefix)
}

// New creates and returns a new FileSystem tracking the actual file.
func New(actual string) (FileSystem, error) {
	fs := fileSystem{
		actual: actual,
		root: entry{
			virtual:  "",
			actual:   actual,
			parent:   nil,
			children: map[string]Entry{},
		},
	}

	for _, v := range []string{"bin", "pkg", "src"} {
		fs.root.children[v] = &entry{
			virtual:  v,
			actual:   "",
			parent:   &fs.root,
			children: map[string]Entry{},
		}
	}
	return &fs, nil
}

func getRealDirAttr(actual string) (*fuse.Attr, error) {
	st := unix.Stat_t{}
	if err := unix.Lstat(actual, &st); err != nil {
		return nil, err
	}
	return unixAttrToFuseAttr(&st), nil
}
