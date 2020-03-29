package mapping

import (
	"path/filepath"

	"github.com/linuxerwang/goplz/conf"
	pb "github.com/linuxerwang/goplz/conf/proto"
)

type sourceMapping struct {
	actualDir string
	excludes  []string
	filters   []*sourceFilter
}

func (sm *sourceMapping) Map(actual string) (string, bool, MatchStatus) {
	for _, e := range sm.excludes {
		if ContainsDir(actual, e) {
			return "", false, Excluded
		}
	}
	if !filepath.HasPrefix(actual, sm.actualDir) {
		return "", false, Unmatched
	}
	for _, f := range sm.filters {
		if virtual, readonly, st := f.Map(actual); virtual != "" {
			return virtual, readonly, st
		}
	}
	return "", false, Unmatched
}

func newSourceMapping(cfg *conf.Config, sm *pb.SourceMapping) *sourceMapping {
	smapping := sourceMapping{
		actualDir: sm.FromActualDir,
	}
	for _, f := range sm.Filter {
		smapping.filters = append(smapping.filters, newSourceFilter(cfg, f))
	}
	for _, e := range sm.Exclude {
		smapping.excludes = append(smapping.excludes, e)
	}
	return &smapping
}
