package hfmeta

import (
	"fmt"
	"path"
	"regexp"
	"slices"
	"strings"

	"image-tools/internal/fetch"
)

var importIDSanitizer = regexp.MustCompile(`[^a-z0-9._-]+`)

func (m Metadata) ImportID(fallback string) string {
	base := m.RepoID
	if base == "" {
		base = m.Source
	}
	base = strings.ToLower(path.Base(strings.TrimRight(base, "/")))
	base = strings.TrimSuffix(base, path.Ext(base))
	base = strings.Trim(importIDSanitizer.ReplaceAllString(base, "-"), "-._")
	if base == "" {
		base = fallback
	}
	return "imported-" + base
}
func (m Metadata) ImportName(fallback string) string {
	base := m.RepoID
	if base == "" {
		base = path.Base(strings.TrimRight(m.Source, "/"))
	}
	if strings.TrimSpace(base) == "" {
		return fallback
	}
	return base
}
func (m Metadata) LicenseLabel() string {
	if strings.TrimSpace(m.License) == "" {
		return "unverified"
	}
	return m.License
}
func (m Metadata) IsLocal() bool {
	return strings.HasPrefix(m.Source, "/") || strings.HasPrefix(m.Source, ".")
}

// ImportSource retains local references and pins repository revisions supplied by
// inspection. Each catalog chooses which single-file weight formats it accepts.
func (m Metadata) ImportSource(extensions ...string) fetch.Source {
	if m.IsLocal() {
		return fetch.Source{LocalPath: m.Source}
	}
	if m.Layout == LayoutDiffusersRepo && m.RepoID != "" {
		return fetch.Source{Repo: fetch.RepoSpec{RepoID: m.RepoID, Revision: m.Revision}}
	}
	if m.Layout != LayoutSingleFile {
		return fetch.Source{}
	}
	filename, url := "", ""
	if hasURLScheme(m.Source) {
		filename, url = path.Base(strings.SplitN(m.Source, "?", 2)[0]), m.Source
	} else if m.RepoID != "" {
		for _, f := range m.Files {
			if !strings.Contains(strings.Trim(f.Path, "/"), "/") && slices.Contains(extensions, strings.ToLower(path.Ext(f.Path))) {
				filename = path.Base(f.Path)
				rev := m.Revision
				if rev == "" {
					rev = "main"
				}
				url = fmt.Sprintf("https://huggingface.co/%s/resolve/%s/%s", m.RepoID, rev, filename)
				break
			}
		}
	}
	if url == "" {
		return fetch.Source{}
	}
	kind := fetch.ArtifactGeneric
	switch strings.ToLower(path.Ext(filename)) {
	case ".safetensors":
		kind = fetch.ArtifactSafetensors
	case ".gguf":
		kind = fetch.ArtifactGGUF
	case ".onnx":
		kind = fetch.ArtifactONNX
	}
	return fetch.Source{Assets: []fetch.Asset{{URL: url, Filename: filename, Kind: kind}}}
}
