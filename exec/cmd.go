package exec

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/linuxerwang/goplz/conf"
)

var (
	cfg *conf.Config
)

func init() {
	conf.RegisterInitializer(func(c *conf.Config){
		cfg = c
	})
}

// RunCommand executes the given command.
func RunCommand(command string) error {
	parts := strings.Split(command, " ")
	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Env = replaceGoPathEnv()
	return cmd.Run()
}

func replaceGoPathEnv() []string {
	environ := []string{fmt.Sprintf("GOPATH=%s", cfg.Settings.VirtualGoPath)}
	env := os.Environ()
	for _, e := range env {
		if strings.HasPrefix(e, "GOPATH=") {
			continue
		}
		environ = append(environ, e)
	}
	return environ
}
