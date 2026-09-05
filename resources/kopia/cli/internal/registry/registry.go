// Package registry keeps the resource-kopia import path stable while the
// registry model is shared with the Vrooli control plane.
package registry

import kopiaregistry "github.com/vrooli/vrooli/packages/kopiaregistry-go"

const (
	BackendFilesystem = kopiaregistry.BackendFilesystem
	BackendS3         = kopiaregistry.BackendS3
)

type (
	Entry    = kopiaregistry.Entry
	Registry = kopiaregistry.Registry
)

var New = kopiaregistry.New
