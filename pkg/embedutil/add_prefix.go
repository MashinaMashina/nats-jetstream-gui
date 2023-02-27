package embedutil

import (
	"embed"
	"io/fs"
	"path"
)

type PrefixFS struct {
	embed.FS
	prefix string
}

func NewPrefixFS(prefix string, fs embed.FS) PrefixFS {
	return PrefixFS{
		FS:     fs,
		prefix: prefix,
	}
}

func (fs PrefixFS) Open(name string) (fs.File, error) {
	return fs.FS.Open(path.Join(fs.prefix, name))
}
