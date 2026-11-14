package graph

// SnapshotIndex is a reusable lookup view over one immutable graph snapshot.
// It keeps detector packages from rebuilding the same package/file maps and
// reproducing package resolution rules independently.
type SnapshotIndex struct {
	packages map[string]PackageNode
	files    map[string]FileNode
}

// IndexSnapshot builds the package and file indexes used by graph detectors.
func IndexSnapshot(snapshot GraphSnapshot) SnapshotIndex {
	index := SnapshotIndex{
		packages: make(map[string]PackageNode, len(snapshot.Packages)),
		files:    make(map[string]FileNode, len(snapshot.Files)),
	}
	for _, pkg := range snapshot.Packages {
		index.packages[pkg.ID] = pkg
	}
	for _, file := range snapshot.Files {
		index.files[file.ID] = file
	}
	return index
}

// Package returns a package by ID, or the zero value when absent.
func (index SnapshotIndex) Package(id string) PackageNode { return index.packages[id] }

// File returns a file by ID, or the zero value when absent.
func (index SnapshotIndex) File(id string) FileNode { return index.files[id] }

// ResolvePackage resolves either a package ID or a file ID to its owner.
// Unknown or ownerless IDs return the zero PackageNode.
func (index SnapshotIndex) ResolvePackage(id string) PackageNode {
	if pkg, ok := index.packages[id]; ok {
		return pkg
	}
	if file, ok := index.files[id]; ok {
		return index.packages[file.PackageID]
	}
	return PackageNode{}
}

// SymbolPath returns the owning file path, falling back to the package path
// when the symbol has no file entry.
func (index SnapshotIndex) SymbolPath(symbol SymbolNode) string {
	if file := index.File(symbol.FileID); file.ID != "" {
		return file.Path
	}
	return index.Package(symbol.PackageID).RepoPath
}
