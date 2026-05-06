package memberflow

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var (
	metadataStartRE = regexp.MustCompile(`^\s*<!--\s*prompt-manager-graph:\s*$`)
	metadataEndRE   = regexp.MustCompile(`^\s*-->\s*$`)
)

func LoadOperatingGraphBlocks(repoRoot string) ([]OperatingGraphBlock, error) {
	docsDir := filepath.Join(repoRoot, "docs")
	var blocks []OperatingGraphBlock
	err := filepath.WalkDir(docsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(strings.ToLower(path), ".md") {
			return nil
		}
		rel, _ := filepath.Rel(repoRoot, path)
		found, err := ExtractOperatingGraphBlocks(path, rel)
		if err != nil {
			return err
		}
		blocks = append(blocks, found...)
		return nil
	})
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	sort.Slice(blocks, func(i, j int) bool {
		if blocks[i].Metadata.ID != blocks[j].Metadata.ID {
			return blocks[i].Metadata.ID < blocks[j].Metadata.ID
		}
		return blocks[i].Source.Path < blocks[j].Source.Path
	})
	return blocks, nil
}

func ExtractOperatingGraphBlocks(path, sourcePath string) ([]OperatingGraphBlock, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(data), "\n")
	var blocks []OperatingGraphBlock
	inFence := false
	for i := 0; i < len(lines); i++ {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), "```") {
			inFence = !inFence
			continue
		}
		if inFence || !metadataStartRE.MatchString(lines[i]) {
			continue
		}
		metaStart := i + 1
		var metaLines []string
		i++
		for i < len(lines) && !metadataEndRE.MatchString(lines[i]) {
			metaLines = append(metaLines, lines[i])
			i++
		}
		if i >= len(lines) {
			return nil, fmt.Errorf("%s:%d: unterminated prompt-manager-graph metadata", sourcePath, metaStart)
		}
		meta, err := parseOperatingGraphMetadata(metaLines)
		if err != nil {
			return nil, fmt.Errorf("%s:%d: %w", sourcePath, metaStart, err)
		}
		i++
		for i < len(lines) && strings.TrimSpace(lines[i]) == "" {
			i++
		}
		if i >= len(lines) || strings.TrimSpace(lines[i]) != "```mermaid" {
			return nil, fmt.Errorf("%s:%d: prompt-manager-graph metadata must be followed by a mermaid fence", sourcePath, metaStart)
		}
		fenceLine := i + 1
		i++
		var mermaid []string
		for i < len(lines) && strings.TrimSpace(lines[i]) != "```" {
			mermaid = append(mermaid, lines[i])
			i++
		}
		if i >= len(lines) {
			return nil, fmt.Errorf("%s:%d: unterminated mermaid fence", sourcePath, fenceLine)
		}
		graph, err := ParseOperatingMermaid(meta.ID, mermaid, fenceLine+1)
		if err != nil {
			return nil, fmt.Errorf("%s:%d: %w", sourcePath, fenceLine, err)
		}
		blocks = append(blocks, OperatingGraphBlock{
			Metadata: meta,
			Graph:    graph,
			Source: OperatingGraphSource{
				Path:      sourcePath,
				Line:      metaStart,
				FenceLine: fenceLine,
			},
		})
	}
	return blocks, nil
}

func parseOperatingGraphMetadata(lines []string) (OperatingGraphMetadata, error) {
	meta := OperatingGraphMetadata{Extra: map[string]string{}}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			return meta, fmt.Errorf("metadata line %q must be key: value", line)
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		switch key {
		case "id":
			meta.ID = value
		case "scope":
			meta.Scope = value
		case "team":
			meta.Team = value
		case "mode":
			meta.Mode = value
		case "status":
			meta.Status = value
		case "allow":
			return meta, fmt.Errorf("metadata field %q is not supported; remove it and resolve graph drift explicitly", key)
		default:
			meta.Extra[key] = value
		}
	}
	if meta.ID == "" || meta.Scope == "" || meta.Team == "" || meta.Mode == "" {
		return meta, fmt.Errorf("metadata requires id, scope, team, and mode")
	}
	switch meta.Mode {
	case "explanatory", "checkable", "contract":
	default:
		return meta, fmt.Errorf("unsupported mode %q", meta.Mode)
	}
	return meta, nil
}

func filterOperatingGraphBlocks(blocks []OperatingGraphBlock, teamFilter, idFilter string) []OperatingGraphBlock {
	var out []OperatingGraphBlock
	for _, block := range blocks {
		if teamFilter != "" && block.Metadata.Team != teamFilter {
			continue
		}
		if idFilter != "" && block.Metadata.ID != idFilter {
			continue
		}
		out = append(out, block)
	}
	return out
}
