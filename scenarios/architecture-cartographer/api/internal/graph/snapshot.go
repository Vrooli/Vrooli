package graph

// Clone returns a deep copy of the snapshot. Cartographer treats
// GraphSnapshot as immutable, but consumers occasionally need a
// value-copy they can hold onto across goroutines without aliasing the
// caller's slices.
func (s GraphSnapshot) Clone() GraphSnapshot {
	out := s
	if s.Languages != nil {
		out.Languages = append([]Language(nil), s.Languages...)
	}
	if s.Files != nil {
		out.Files = append([]FileNode(nil), s.Files...)
	}
	if s.Packages != nil {
		out.Packages = append([]PackageNode(nil), s.Packages...)
	}
	if s.Symbols != nil {
		out.Symbols = append([]SymbolNode(nil), s.Symbols...)
	}
	if s.Imports != nil {
		imps := make([]ImportEdge, len(s.Imports))
		for i, e := range s.Imports {
			imps[i] = e
			if e.SymbolIDs != nil {
				imps[i].SymbolIDs = append([]string(nil), e.SymbolIDs...)
			}
			if e.SymbolKinds != nil {
				imps[i].SymbolKinds = append([]string(nil), e.SymbolKinds...)
			}
		}
		out.Imports = imps
	}
	return out
}
