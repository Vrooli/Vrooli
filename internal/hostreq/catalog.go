package hostreq

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"runtime"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
	"github.com/vrooli/vrooli/internal/safeguards"
	"github.com/vrooli/vrooli/internal/tools"
)

type requirementCatalog struct {
	tools      map[string]hostreqkit.ToolManifest
	safeguards map[string]hostreqkit.SafeguardManifest
}

func loadRequirementCatalog() (requirementCatalog, error) {
	toolItems := make(map[string]hostreqkit.ToolManifest)
	if err := readCatalog(tools.Manifests, "tool.json", func(data []byte) error {
		var item hostreqkit.ToolManifest
		if err := json.Unmarshal(data, &item); err != nil {
			return err
		}
		if item.Name == "" {
			return fmt.Errorf("tool manifest has no name")
		}
		toolItems[item.Name] = item
		return nil
	}); err != nil {
		return requirementCatalog{}, fmt.Errorf("load tool catalog: %w", err)
	}
	safeguardItems := make(map[string]hostreqkit.SafeguardManifest)
	if err := readCatalog(safeguards.Manifests, "safeguard.json", func(data []byte) error {
		var item hostreqkit.SafeguardManifest
		if err := json.Unmarshal(data, &item); err != nil {
			return err
		}
		if item.Name == "" {
			return fmt.Errorf("safeguard manifest has no name")
		}
		safeguardItems[item.Name] = item
		return nil
	}); err != nil {
		return requirementCatalog{}, fmt.Errorf("load safeguard catalog: %w", err)
	}
	return requirementCatalog{tools: toolItems, safeguards: safeguardItems}, nil
}

func readCatalog(fsys fs.FS, filename string, consume func([]byte) error) error {
	return fs.WalkDir(fsys, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || entry.Name() != filename {
			return nil
		}
		data, err := fs.ReadFile(fsys, path)
		if err != nil {
			return err
		}
		return consume(data)
	})
}

func (c requirementCatalog) details(kind hostreqspec.Kind, name, platform string) (hostreqspec.Privilege, hostreqspec.Bundling, error) {
	if kind == hostreqspec.KindSafeguard {
		item, ok := c.safeguards[name]
		if !ok {
			return "", "", fmt.Errorf("unknown safeguard %q", name)
		}
		return item.Privilege, item.Bundling, nil
	}
	item, ok := c.tools[name]
	if !ok {
		return "", "", fmt.Errorf("unknown tool %q", name)
	}
	return toolPrivilege(item, platform), item.Bundling, nil
}

func toolPrivilege(item hostreqkit.ToolManifest, platform string) hostreqspec.Privilege {
	if item.Privilege != "" {
		return item.Privilege
	}
	if item.SourceType() != "package" {
		return hostreqspec.PrivilegeUser
	}
	platform = strings.ToLower(strings.TrimSpace(platform))
	if platform == "" {
		platform = runtime.GOOS
	}
	if platform == "darwin" {
		platform = "macos"
	}
	if platform == "linux" || platform == "windows" {
		return hostreqspec.PrivilegeElevated
	}
	return hostreqspec.PrivilegeUser
}
