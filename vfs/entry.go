package vfs

import (
	"github.com/hanwen/go-fuse/fuse"
)

// Entry is an interface for the virtual directory or file.
type Entry interface {
	// Virtual returns the virtual file of this entry.
	Virtual() string
	// Actual returns the actual file mapped by this virtual file.
	Actual() string
	// Attr returns the attrs of this entry.
	Attr() (*fuse.Attr, error)
	// Readonly returns true if this virtual file is readonly.
	Readonly() bool
	// Parent returns the parent entry.
	Parent() Entry
	// Children returns the child entries.
	Children() map[string]Entry
}

type entry struct {
	virtual  string
	actual   string
	readonly bool

	parent   Entry
	children map[string]Entry
}

func (e *entry) Virtual() string {
	return e.virtual
}

func (e *entry) Actual() string {
	return e.actual
}

func (e *entry) Parent() Entry {
	return e.parent
}

func (e *entry) Children() map[string]Entry {
	return e.children
}

func (e *entry) Attr() (attr *fuse.Attr, err error) {
	attr = &defaultDirAttr
	if e.actual != "" {
		attr, err = getRealDirAttr(e.actual)
		if err != nil {
			return
		}
	}
	if e.readonly {
		// Reset the W bits.
		attr.Mode &^= 0b010_010_010
	}
	return attr, nil
}

func (e *entry) Readonly() bool {
	return e.readonly
}

// Make sure *entry implements Entry.
var _ = (Entry)((*entry)(nil))
