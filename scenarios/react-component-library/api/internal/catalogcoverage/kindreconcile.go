package catalogcoverage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// KindMismatch is an advisory reconciliation result. The catalog declaration
// remains authoritative; this heuristic never rewrites asset.kind.
type KindMismatch struct {
	AssetID      string
	DeclaredKind string
	DerivedKind  string
	Message      string
}

// ReconcileKinds derives a conservative kind from the implementation root and
// its exported source shape. It is intentionally a finding-producing oracle,
// not an auto-corrector: unusual assets should be reviewed rather than
// silently moved between maturity ladders.
func ReconcileKinds(root string, assets []Asset, impls []Implementation) ([]KindMismatch, error) {
	_ = root
	declared := map[string]string{}
	for _, asset := range assets {
		declared[asset.ID] = asset.Kind
	}
	var out []KindMismatch
	for _, impl := range impls {
		if impl.CatalogID == "" || declared[impl.CatalogID] == "" {
			continue
		}
		derived := derivedKind(impl.Root)
		if impl.Path != "" && impl.Latest != "" {
			source := filepath.Join(filepath.Dir(impl.Path), "versions", impl.Latest)
			files, _ := filepath.Glob(filepath.Join(source, "*.tsx"))
			if len(files) == 0 {
				files, _ = filepath.Glob(filepath.Join(source, "*.ts"))
			}
			if len(files) > 0 {
				data, err := os.ReadFile(files[0])
				if err != nil {
					return nil, err
				}
				text := string(data)
				if strings.Contains(text, "<") && (impl.Root == "components" || impl.Root == "primitives") {
					derived = map[bool]string{true: "component", false: derived}[impl.Root == "components"]
				}
			}
		}
		if declared[impl.CatalogID] != derived {
			out = append(out, KindMismatch{AssetID: impl.CatalogID, DeclaredKind: declared[impl.CatalogID], DerivedKind: derived, Message: fmt.Sprintf("declared kind %q disagrees with derived kind %q", declared[impl.CatalogID], derived)})
		}
	}
	return out, nil
}

func derivedKind(root string) string {
	switch root {
	case "foundations":
		return "foundation"
	case "hooks":
		return "runtime-hook"
	case "services":
		return "runtime-service"
	case "primitives":
		return "primitive"
	case "components":
		return "component"
	default:
		return "unknown"
	}
}
