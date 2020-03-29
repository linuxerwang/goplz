package conf

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/golang/protobuf/proto"
	"gopkg.in/gcfg.v1"

	pb "github.com/linuxerwang/goplz/conf/proto"
)

const (
	goplzPidFile = ".goplzpid"
	goplzRcFile  = ".goplzrc"
	plzCfgFile   = ".plzconfig"
)

type initializer func(cfg *Config)

var (
	workspace string

	cfg Config

	initializers = make([]initializer, 0, 10)
)

func init() {
	var err error
	workspace, err = os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get the current working directory, %v", err)
	}

	workspace, err = filepath.EvalSymlinks(workspace)
	if err != nil {
		log.Fatalf("Failed to get the current working directory, %v", err)
	}
}

// RegisterInitializer registers the given initializer.
func RegisterInitializer(i initializer) {
	initializers = append(initializers, i)
}

// Config contains the goplz configuration.
type Config struct {
	Settings *pb.Settings

	GoImportPath  string
	Workspace     string
	GoplzConf     string
	GoplzPid      string
	PlzConf       string
	VirtualSrcDir string
}

// GetExistingProcess returns the existing goplz process for the workspace.
func (cfg *Config) GetExistingProcess() *os.Process {
	if _, err := os.Stat(cfg.GoplzPid); err != nil {
		log.Printf("There is no file .goplzpid in workspace %s.\n", cfg.Workspace)
		return nil
	}

	b, err := ioutil.ReadFile(cfg.GoplzPid)
	if err != nil {
		log.Printf("Failed to read from .goplzpid, %v.\n", err)
		return nil
	}

	pid, err := strconv.ParseInt(strings.TrimSpace(string(b)), 10, 32)
	if err != nil {
		log.Printf("Invalid pid in .goplzpid, %v.\n", string(b))
		return nil
	}

	p, err := os.FindProcess(int(pid))
	if err != nil {
		log.Printf("Failed to find process %d.\n", pid)
		os.Remove(cfg.GoplzPid)
		return nil
	}
	return p
}

// HasGoplzRc returns true if the .goplzrc file exists.
func HasGoplzRc() bool {
	if _, err := os.Stat(goplzRcFile); err != nil {
		if os.IsNotExist(err) {
			return false
		}
		panic(err)
	}
	return true
}

// CreateGoplzRc creates the .goplzrc file.
func CreateGoplzRc(virtualGoPath string) {
	autoVirtualGoPath := false
	if virtualGoPath == "" {
		autoVirtualGoPath = true
		virtualGoPath = filepath.Join(filepath.Dir(workspace), fmt.Sprintf(".%s-gopath", filepath.Base(workspace)))
	}

	if _, err := os.Stat(virtualGoPath); err != nil {
		if os.IsNotExist(err) {
			if err = os.MkdirAll(virtualGoPath, os.ModePerm); err != nil {
				panic(err)
			}
		} else {
			panic(err)
		}
	}

	settings := pb.Settings{
		IdeCmd:        "/usr/bin/code",
		VirtualGoPath: virtualGoPath,
		SourceMapping: []*pb.SourceMapping{
			{
				FromActualDir: "plz-out/gen",
				Filter: []*pb.SourceFilter{
					{
						Match:        "plz-out/gen/third_party/go/pkg",
						ToVirtualDir: "pkg",
						Strip:        "plz-out/gen/third_party/go/pkg",
						Readonly:     true,
					},
					{
						Match:        "plz-out/gen/third_party/go/src",
						ToVirtualDir: "src",
						Strip:        "plz-out/gen/third_party/go/src",
					},
					{
						Match:        ".*\\.a$",
						ToVirtualDir: "pkg",
						Strip:        "plz-out/gen",
						Prepend:      "linux_amd64/{{.GoImportPath}}",
						ExcludeRegexp: []string{
							"^third_party/.*",
						},
						Readonly: true,
					},
					{
						Match:        ".*\\.pb.go$",
						ToVirtualDir: "src",
						Strip:        "plz-out/gen",
						Prepend:      "{{.GoImportPath}}",
						ExcludeRegexp: []string{
							"^plz-out/gen/third_party/.*",
						},
						Readonly: true,
					},
				},
			},
		},
		Exclude: []string{
			".git",
		},
	}
	saveCfg(goplzRcFile, &settings)
	if !autoVirtualGoPath {
		fmt.Printf("Created goplz config file %s, please set virtual_go_path.\n", goplzRcFile)
		fmt.Println("After that, run `goplz start`.")
	} else {
		fmt.Printf("Initialized goplz, the virtual GOPATH is at %s.\n", virtualGoPath)
	}
}

// Cfg returns the goplz config.
func Cfg() *Config {
	// The command has to be executed in a Please workspace.
	if _, err := os.Stat(filepath.Join(workspace, plzCfgFile)); err != nil {
		log.Fatalf("Error, the command has to be run in a Please workspace, %v", err)
	}

	cfg.Workspace = workspace
	cfg.GoplzConf = filepath.Join(workspace, goplzRcFile)
	cfg.GoplzPid = filepath.Join(workspace, goplzPidFile)
	cfg.PlzConf = filepath.Join(workspace, plzCfgFile)

	plzCfg := struct {
		Go struct {
			ImportPath string
		}
	}{}
	if err := gcfg.FatalOnly(gcfg.ReadFileInto(&plzCfg, plzCfgFile)); err != nil {
		fmt.Printf("Failed to parse plz config file %s, %+v.\n", plzCfgFile, err)
		os.Exit(1)
	}

	cfg.GoImportPath = strings.TrimSpace(plzCfg.Go.ImportPath)
	if cfg.GoImportPath == "" {
		fmt.Printf("Can not find Go ImportPath in %s.\n", plzCfgFile)
		os.Exit(1)
	}
	fmt.Printf("Go Import Path: %s\n", cfg.GoImportPath)

	settings := &pb.Settings{}
	parseCfg(goplzRcFile, settings)
	if settings.VirtualGoPath == "REPLACE_ME" {
		fmt.Printf("virtual_go_path was not set.")
		os.Exit(1)
	}

	fmt.Printf("Virtual Go Path: %s\n", settings.VirtualGoPath)
	cfg.VirtualSrcDir = filepath.Join(settings.VirtualGoPath, "src")

	cfg.Settings = settings

	for _, i := range initializers {
		i(&cfg)
	}
	return &cfg
}

func parseCfg(fn string, cfg proto.Message) {
	b, err := ioutil.ReadFile(fn)
	if err != nil {
		log.Fatalf("Error to read config file %s, %v.\n", fn, err)
	}

	if err := proto.UnmarshalText(string(b), cfg); err != nil {
		log.Fatalf("Error to parse config file %s, %v.\n", fn, err)
	}
}

func saveCfg(fn string, cfg proto.Message) {
	if err := ioutil.WriteFile(fn, []byte(proto.MarshalTextString(cfg)), os.ModePerm); err != nil {
		log.Fatalf("Error to save config file %s, %v.\n", fn, err)
	}
}
