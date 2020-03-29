package mapping

import (
	"bytes"
	"log"
	"path/filepath"
	"regexp"
	"text/template"

	"github.com/linuxerwang/goplz/conf"
	pb "github.com/linuxerwang/goplz/conf/proto"
)

type sourceFilter struct {
	cfg          *conf.Config
	match        *regexp.Regexp
	toVirtualDir string
	strip        string
	prepend      *template.Template
	excludes     []*regexp.Regexp
	buf          *bytes.Buffer
	readonly     bool
}

func (sf *sourceFilter) Map(from string) (string, bool, MatchStatus) {
	if !sf.match.MatchString(from) {
		return "", false, Unmatched
	}
	for _, e := range sf.excludes {
		if e.MatchString(from) {
			return "", false, Unmatched
		}
	}
	if sf.strip != "" {
		var err error
		from, err = filepath.Rel(sf.strip, from)
		if err != nil {
			return "", false, Unmatched
		}
	}
	prepend := ""
	if sf.prepend != nil {
		sf.buf.Reset()
		if err := sf.prepend.Execute(sf.buf, sf.cfg); err != nil {
			log.Print(err)
			return "", false, Unmatched
		}
		prepend = sf.buf.String()
	}
	return filepath.Join(sf.toVirtualDir, prepend, from), sf.readonly, Matched
}

func newSourceFilter(cfg *conf.Config, f *pb.SourceFilter) *sourceFilter {
	sf := sourceFilter{
		cfg:          cfg,
		match:        regexp.MustCompile(f.Match),
		toVirtualDir: f.ToVirtualDir,
		strip:        f.Strip,
		prepend:      template.Must(template.New("prepend").Parse(f.Prepend)),
		buf:          &bytes.Buffer{},
		readonly:     f.Readonly,
	}
	for _, e := range f.ExcludeRegexp {
		sf.excludes = append(sf.excludes, regexp.MustCompile(e))
	}
	return &sf
}
