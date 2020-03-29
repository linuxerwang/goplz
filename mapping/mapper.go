package mapping

import (
	"github.com/linuxerwang/goplz/conf"
	pb "github.com/linuxerwang/goplz/conf/proto"
)

const (
	Excluded MatchStatus = iota
	Matched
	Unmatched
)

type MatchStatus int

// SourceMapper is an interface for source code mapping.
type SourceMapper interface {
	// Map returns the virtual file mapped to the given actual file. It returns
	// true if the mapping is valid according the predefined mapping rules. It
	// also returns the match status.
	Map(actual string) (string, bool, MatchStatus)
}

type sourceMapper struct {
	excludes []string
	mappings []*sourceMapping
}

func (sm *sourceMapper) Map(actual string) (string, bool, MatchStatus) {
	for _, e := range sm.excludes {
		if ContainsDir(actual, e) {
			return "", false, Excluded
		}
	}
	for _, mapping := range sm.mappings {
		if virtual, readonly, st := mapping.Map(actual); virtual != "" {
			return virtual, readonly, st
		}
	}
	return "", false, Unmatched
}

// Make sure sourceMapper implements SourceMapper.
var _ = (SourceMapper)(&sourceMapper{})

// New creates and returns a new SourceMapping.
func New(cfg *conf.Config) SourceMapper {
	smapper := sourceMapper{
		excludes: cfg.Settings.Exclude,
	}
	for _, sm := range cfg.Settings.SourceMapping {
		smapper.mappings = append(smapper.mappings, newSourceMapping(cfg, sm))
	}
	// Default mapping must be at the last.
	smapper.mappings = append(smapper.mappings, newSourceMapping(cfg, defaultMapping()))
	return &smapper
}

func defaultMapping() *pb.SourceMapping {
	return &pb.SourceMapping{
		FromActualDir: "",
		Filter: []*pb.SourceFilter{
			{
				Match:        ".*",
				ToVirtualDir: "src",
				Prepend:      "{{.GoImportPath}}",
			},
		},
		Exclude: []string{
			"plz-out",
		},
	}
}
