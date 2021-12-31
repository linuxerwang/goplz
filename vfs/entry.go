package vfs

import (
	"fmt"
	"io"
	"log"
	"sync"

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
	// GetChild returns the child entry.
	GetChild(key string) Entry
	// SetChild sets the child entry with key.
	SetChild(key string, entry Entry)
	// DeleteChild deletes child entry by key.
	DeleteChild(key string)
	Children() []fuse.DirEntry
	// Print prints the entry.
	Print(w io.Writer, prefix string)
}

type entry struct {
	virtual  string
	actual   string
	readonly bool

	parent     Entry
	children   map[string]Entry
	childrenMu sync.RWMutex
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

func (e *entry) GetChild(key string) Entry {
	e.childrenMu.RLock()
	defer e.childrenMu.RUnlock()

	return e.children[key]
}

func (e *entry) SetChild(key string, entry Entry) {
	e.childrenMu.Lock()
	defer e.childrenMu.Unlock()

	e.children[key] = entry
}

func (e *entry) DeleteChild(key string) {
	e.childrenMu.Lock()
	defer e.childrenMu.Unlock()

	delete(e.children, key)
}

func (e *entry) Children() []fuse.DirEntry {
	e.childrenMu.RLock()
	defer e.childrenMu.RUnlock()

	entries := make([]fuse.DirEntry, 0, len(e.children))
	for _, c := range e.children {
		attr, err := c.Attr()
		if err != nil {
			log.Print(err)
			continue
		}
		entries = append(entries, fuse.DirEntry{
			Name: c.Virtual(),
			Mode: attr.Mode,
		})
	}
	return entries
}

func (e *entry) Print(w io.Writer, prefix string) {
	virtual := e.Virtual()
	if virtual == "" || virtual == "." {
		virtual = "TOP"
	}
	ftype := "F"
	attr, _ := e.Attr()
	if attr.IsDir() {
		ftype = "D"
	}
	io.WriteString(w, fmt.Sprintf("%s[%s] %s => %s\n", prefix, ftype, virtual, e.Actual()))
	prefix += "    "

	e.childrenMu.RLock()
	defer e.childrenMu.RUnlock()

	for _, c := range e.children {
		c.Print(w, prefix)
	}
}

// Make sure *entry implements Entry.
var _ = (Entry)((*entry)(nil))
