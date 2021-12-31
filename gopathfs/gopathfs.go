package gopathfs

import (
	"log"
	"path/filepath"

	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/pathfs"
	"github.com/rjeczalik/notify"
	cli "github.com/urfave/cli/v2"

	"github.com/linuxerwang/goplz/conf"
	"github.com/linuxerwang/goplz/mapping"
	"github.com/linuxerwang/goplz/vfs"
)

var (
	verbose bool
)

// Init initialize the gopathfs package.
func Init(ctx *cli.Context) {
	verbose = ctx.Bool("verbose")
}

type changeCallbackFunc func(notify.EventInfo)

// GoPathFs implements a virtual tree for src folder of GOPATH.
type GoPathFs struct {
	pathfs.FileSystem
	cfg            *conf.Config
	vfs            vfs.FileSystem
	mapper         mapping.SourceMapper
	changeCallback changeCallbackFunc
	notifyCh       chan notify.EventInfo
}

// Access overrides the parent's Access method.
func (gpf *GoPathFs) Access(name string, mode uint32, context *fuse.Context) fuse.Status {
	return fuse.OK
}

// GetAttr overrides the parent's GetAttr method.
func (gpf *GoPathFs) GetAttr(name string, context *fuse.Context) (*fuse.Attr, fuse.Status) {
	entry, relpath := gpf.vfs.MatchPath(name)
	if len(relpath) != 0 {
		return nil, fuse.ENOENT
	}
	attr, err := entry.Attr()
	if err != nil {
		return nil, fuse.ENOENT
	}
	return attr, fuse.OK
}

// OnMount overrides the parent's OnMount method.
func (gpf *GoPathFs) OnMount(nodeFs *pathfs.PathNodeFs) {
	root := filepath.Join(gpf.cfg.Workspace, "...")
	if verbose {
		log.Printf("Watching directory %s for changes.", root)
	}
	if err := notify.Watch(root, gpf.notifyCh, notify.Create|notify.Remove|notify.Rename); err != nil {
		log.Fatal(err)
	}

	go func() {
		for ei := range gpf.notifyCh {
			gpf.changeCallback(ei)
		}
	}()
}

// OnUnmount overwrites the parent's OnUnmount method.
func (gpf *GoPathFs) OnUnmount() {
	notify.Stop(gpf.notifyCh)
}

// NewGoPathFs returns a new GoPathFs.
func NewGoPathFs(cfg *conf.Config, fs vfs.FileSystem, mapper mapping.SourceMapper, changeCallback changeCallbackFunc) *GoPathFs {
	gpfs := GoPathFs{
		FileSystem:     pathfs.NewDefaultFileSystem(),
		cfg:            cfg,
		vfs:            fs,
		mapper:         mapper,
		changeCallback: changeCallback,
		notifyCh:       make(chan notify.EventInfo, 1000),
	}
	gpfs.SetDebug(true)
	return &gpfs
}
