package start

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	osexec "os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
	"github.com/hanwen/go-fuse/fuse/pathfs"
	"github.com/rjeczalik/notify"
	"github.com/urfave/cli"

	"github.com/linuxerwang/goplz/conf"
	"github.com/linuxerwang/goplz/exec"
	"github.com/linuxerwang/goplz/gopathfs"
	"github.com/linuxerwang/goplz/mapping"
	"github.com/linuxerwang/goplz/vfs"
)

var (
	verbose bool
)

// StartCmd is for subcommand "start".
var StartCmd = &cli.Command{
	Name:  "start",
	Usage: "start goplz",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "detach",
			Value: true,
			Usage: "if true, detach from parent process.",
		},
		&cli.BoolFlag{
			Name:  "detached",
			Value: false,
			Usage: "True means the current process has been detached from parent process. Do not set it manually, it's only used by goplz to detach itself.",
		},
	},
	Action: func(ctx *cli.Context) error {
		verbose = ctx.Bool("verbose")

		fmt.Printf("Starting goplz ...\n")

		cfg := conf.Cfg()
		if _, err := os.Stat(cfg.GoplzPid); !os.IsNotExist(err) {
			b, err := ioutil.ReadFile(cfg.GoplzPid)
			if err != nil {
				panic(err)
			}
			pid, err := strconv.Atoi(string(b))
			if err != nil {
				panic(err)
			}

			// The process for pid might already not exist.
			p, _ := os.FindProcess(pid)
			if err := p.Signal(syscall.Signal(0)); err == nil {
				// The process for pid  still exists.
				fmt.Println("goplz already started, start IDE.")
				startIDE(cfg)
				return nil
			}

			log.Printf("Process with ID %d in file %s does not exist, remove.\n", pid, cfg.GoplzPid)
			os.Remove(cfg.GoplzPid)
		}

		detach := ctx.Bool("detach")
		detached := ctx.Bool("detached")
		if detach && !detached {
			pid, err := detachProcess()
			if err != nil {
				fmt.Println(err)
				os.Exit(2)
			}
			fmt.Printf("goplz is running detached. To stop it, run `goplz stop` or `kill -SIGQUIT %d`.\n", pid)
			return nil
		}

		gopathfs.Init(ctx)
		vfs.Init(ctx)

		mapper := mapping.New(cfg)
		fs := createVirtualFS(cfg, mapper)

		startGopathFS(cfg, detach, fs, mapper)

		return nil
	},
}

func startIDE(cfg *conf.Config) {
	cmd := fmt.Sprintf("%s %s/src", cfg.Settings.IdeCmd, cfg.Settings.VirtualGoPath)
	if err := exec.RunCommand(cmd); err != nil {
		fmt.Println("Error to run IDE, ", err)
	}
}

func detachProcess() (int, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return 0, err
	}
	args := append(os.Args, "--detached")
	cmd := osexec.Command(args[0], args[1:]...)
	cmd.Dir = cwd
	err = cmd.Start()
	if err != nil {
		return 0, err
	}
	pid := cmd.Process.Pid
	cmd.Process.Release()
	return pid, nil
}

func createVirtualFS(cfg *conf.Config, mapper mapping.SourceMapper) vfs.FileSystem {
	fs, err := vfs.New(".")
	if err != nil {
		panic(err)
	}

	filepath.Walk(".", func(actual string, info os.FileInfo, err error) error {
		virtual, readonly, st := mapper.Map(actual)
		if st == mapping.Excluded {
			return filepath.SkipDir
		}
		if virtual != "" {
			fs.Track(virtual, actual, readonly)
		}
		return nil
	})

	if verbose {
		log.Printf("File System:\n%s", fs)
	}
	return fs
}

func startGopathFS(cfg *conf.Config, detach bool, fs vfs.FileSystem, mapper mapping.SourceMapper) {
	absWorkspace, err := filepath.Abs(cfg.Workspace)
	if err != nil {
		panic(err)
	}

	// Create a FUSE virtual file system on cfg.Settings.VirtualGoPath.
	var gpfs *pathfs.PathNodeFs
	gpfs = pathfs.NewPathNodeFs(gopathfs.NewGoPathFs(cfg, fs, mapper, func(ei notify.EventInfo) {
		actual, _ := filepath.Rel(absWorkspace, ei.Path())
		if verbose {
			log.Println("file changed:", actual, ei.Event(), ei.Sys())
		}

		virtual, readonly, st := mapper.Map(actual)
		if st == mapping.Excluded || st == mapping.Unmatched {
			log.Printf("file %s is excluded or unmatched\n", actual)
			return
		}

		switch ei.Event() {
		case notify.Create:
			fi, err := os.Stat(actual)
			if err != nil {
				log.Printf("Failed to stat actual file %s\n", actual)
				return
			}
			if fi.IsDir() {
				filepath.Walk(actual, func(ac string, info os.FileInfo, err error) error {
					vi, readonly, st := mapper.Map(ac)
					if st == mapping.Excluded {
						return filepath.SkipDir
					}
					if vi != "" {
						fs.Track(vi, ac, readonly)
					}
					return nil
				})
			} else {
				fs.Track(virtual, actual, readonly)
			}
		case notify.Remove, notify.Rename:
			fs.Untrack(virtual)
		}
		// TODO: is the line needed?
		// gpfs.EntryNotify(filepath.Dir(virtual), filepath.Base(virtual))
	}), nil)

	fmt.Printf("Fuse mount %s\n", cfg.Settings.VirtualGoPath)
	server, _, err := nodefs.MountRoot(cfg.Settings.VirtualGoPath, gpfs.Root(), nil)
	if err != nil {
		fmt.Printf("Mount fail: %v\n", err)
		os.Exit(2)
	}
	fmt.Printf("Mounted Please source folder to %s. \nYou need to set %s as your GOPATH. \n\n Ctrl+C to exit.\n", cfg.Settings.VirtualGoPath, cfg.Settings.VirtualGoPath)

	if detach {
		if err := ioutil.WriteFile(cfg.GoplzPid,
			[]byte(fmt.Sprintf("%d", os.Getpid())), os.ModePerm); err != nil {

			fmt.Printf("Failed to write to file %s: %v\n", cfg.GoplzPid, err)
			os.Exit(2)
		}
	}

	// Handle ctl+c.
	setGracefullExit(cfg, server)

	// If a Go IDE is specified, start it with the proper GOPATH.
	if cfg.Settings.IdeCmd != "" {
		go func() {
			server.WaitMount()
			fmt.Println("\nStarting IDE ...")
			startIDE(cfg)
		}()
	}

	server.Serve()

	makeSureUnmount(cfg)
	os.Remove(cfg.GoplzPid)
}

func setGracefullExit(cfg *conf.Config, server *fuse.Server) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGQUIT)
	go func() {
		<-c
		for {
			fmt.Printf("\nUnmount %s.\n", cfg.Settings.VirtualGoPath)
			if err := server.Unmount(); err != nil {
				fmt.Println("Error to unmount,", err)
				time.Sleep(time.Second)
				continue
			}
			break
		}
	}()
}

func makeSureUnmount(cfg *conf.Config) {
	// Check if the mount point is still mounted (only works on linux).
	for {
		time.Sleep(time.Second)
		if b, err := ioutil.ReadFile("/proc/mounts"); err == nil {
			if idx := strings.Index(string(b), cfg.Settings.VirtualGoPath); idx < 0 {
				break
			}
			osexec.Command("fusermount", "-u", cfg.Settings.VirtualGoPath).CombinedOutput()
		}
	}
}
