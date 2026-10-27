package main

import (
	"embed"
	"io/fs"

	"github.com/donseba/go-importmap"
)

// The themes' JavaScript packages, listed in importmap.json, are vendored
// into assets/vendor with their licences and a lock file of their integrity
// (task vendor), and embedded in the program, so it needs no network.
//
//go:generate go run github.com/donseba/go-importmap/cmd/importmap vendor -config importmap.json -out assets/vendor

//go:embed assets
var assetFiles embed.FS

// importMap is the import map of the vendored packages. The site serves it
// at /assets/ (webthings.Config.ImportMap), and {{importmap}} in a layout's
// head maps their names, so a theme's site.js can import "htmx"; core adds
// go-partial's browser clients, such as "partial-interactions", which the
// body's clientAttrs "reloadOnError" needs. FromFS checks every vendored
// file against the lock.
func importMap() (*importmap.ImportMap, error) {
	files, err := fs.Sub(assetFiles, "assets")
	if err != nil {
		return nil, err
	}

	return importmap.FromFS(files, "vendor/"+importmap.LockFile)
}
