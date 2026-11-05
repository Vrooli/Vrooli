package uimanifest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LoadAt resolves files for an explicit scenario root. It never searches another
// scenario or the process working directory for a missing target manifest.
func LoadAt(root string) (Manifest, error) {
	root = filepath.Clean(root)
	if filepath.Base(root) == "bas" {
		root = filepath.Dir(root)
	}
	mf := Manifest{Files: map[string]FileDeclaration{}}
	base := filepath.Join(root, "ui", "manifest.json")
	data, err := os.ReadFile(base)
	if os.IsNotExist(err) {
		var svc serviceJSON
		if raw, e := os.ReadFile(filepath.Join(root, ".vrooli", "service.json")); e == nil {
			if e = json.Unmarshal(raw, &svc); e != nil {
				return mf, e
			}
			if id := svc.Generation.Template.ID; id != "" {
				for dir := root; filepath.Dir(dir) != dir; dir = filepath.Dir(dir) {
					candidate := filepath.Join(dir, "templates", "scenarios", id, "ui", "manifest.json")
					if raw, e := os.ReadFile(candidate); e == nil {
						data, err = raw, nil
						break
					}
				}
			}
		}
	}
	if err == nil {
		if e := json.Unmarshal(data, &mf); e != nil {
			return mf, fmt.Errorf("UI manifest: %w", e)
		}
	} else if !os.IsNotExist(err) {
		return mf, err
	}
	if mf.Files == nil {
		mf.Files = map[string]FileDeclaration{}
	}
	defaults := map[string]string{"selectorRegistry": "ui/src/consts/selectors.ts", "librarySelectors": "ui/src/consts/selectors.library.ts", "appEntry": "ui/src/main.tsx"}
	// Compatibility applies only inside this target and only without a declaration.
	if _, declared := mf.Files["selectorRegistry"]; !declared {
		if _, e := os.Stat(filepath.Join(root, "ui/src/constants/selectors.ts")); e == nil {
			defaults["selectorRegistry"] = "ui/src/constants/selectors.ts"
		}
	}
	for key, p := range defaults {
		if _, ok := mf.Files[key]; !ok {
			mf.Files[key] = FileDeclaration{Path: p}
		}
	}
	return loadOverlay(mf, filepath.Join(root, ".vrooli", "ui-manifest.json"))
}

func validateFilePaths(mf Manifest) error {
	for key, file := range mf.Files {
		p := filepath.Clean(file.Path)
		if p == "." || filepath.IsAbs(p) || p == ".." || strings.HasPrefix(p, ".."+string(filepath.Separator)) {
			return fmt.Errorf("UI file %s escapes scenario root", key)
		}
	}
	return nil
}

func (m Manifest) SelectorManifestPath() string {
	return strings.TrimSuffix(strings.TrimSuffix(m.ResolveFile("selectorRegistry", "ui/src/consts/selectors.ts"), ".tsx"), ".ts") + ".manifest.json"
}
