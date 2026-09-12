package apply

import (
	"context"
	"encoding/json"
	"path"
	"strings"
)

// iconAsset is one derived icon variant apply knows how to install: the stored
// asset filename plus the PWA manifest metadata for it. The lookup key into the
// assets domain is the filename stem (matched case-insensitively by the
// composition-root adapter), so the stem is derived from filename.
type iconAsset struct {
	filename string // e.g. "maskable-icon-192.png"
	sizes    string // manifest "sizes" (e.g. "192x192")
	purpose  string // manifest "purpose" ("" → omitted; "maskable" for launcher)
}

func (i iconAsset) stem() string {
	return strings.TrimSuffix(i.filename, path.Ext(i.filename))
}

// iconSet is the canonical platform icon set, in a stable order so a re-apply
// rebuilds a byte-identical manifest icons array (idempotent, no duplicates).
var iconSet = []iconAsset{
	{filename: "favicon-16.png", sizes: "16x16"},
	{filename: "favicon-32.png", sizes: "32x32"},
	{filename: "favicon-196.png", sizes: "196x196"},
	{filename: "apple-touch-icon.png", sizes: "180x180"},
	{filename: "maskable-icon-192.png", sizes: "192x192", purpose: "maskable"},
	{filename: "maskable-icon-512.png", sizes: "512x512", purpose: "maskable"},
}

// applyIcons copies whichever derived icon assets the brand has into ui/public
// and merges the manifest icon metadata (icons array + theme/background color).
// Each copied file is its own action so a preview reports the full plan. The
// element is skipped only when the brand has neither derived icons nor colors —
// there would be nothing to write.
func (s *service) applyIcons(ctx context.Context, brand BrandView, scenario string, write bool) ([]Action, *Skip, error) {
	var actions []Action
	var present []iconAsset
	for _, ic := range iconSet {
		content, found, err := s.assets.Read(ctx, brand.ID, ic.stem())
		if err != nil {
			return nil, nil, err
		}
		if !found {
			continue
		}
		rel := path.Join(publicDir, content.Filename)
		if write {
			if err := s.workspace.WriteFile(ctx, scenario, rel, content.Bytes); err != nil {
				return nil, nil, err
			}
		}
		actions = append(actions, Action{Type: ActionAsset, File: rel, Element: ElementIcons})
		present = append(present, ic)
	}

	if len(present) == 0 && !brand.Colors.HasAny() {
		return nil, &Skip{Element: ElementIcons, Reason: "no derived icons or colors to install"}, nil
	}

	if write {
		existing, err := s.workspace.ReadFile(ctx, scenario, manifestPath)
		if err != nil {
			return nil, nil, err
		}
		merged, err := mergeIconsManifest(existing, brand, present)
		if err != nil {
			return nil, nil, err
		}
		if err := s.workspace.WriteFile(ctx, scenario, manifestPath, merged); err != nil {
			return nil, nil, err
		}
	}
	actions = append(actions, Action{Type: ActionJSON, File: manifestPath, Element: ElementIcons})
	return actions, nil, nil
}

// mergeIconsManifest merges the PWA icon metadata onto an existing manifest,
// preserving every unmanaged key (and the identity keys mergeManifest wrote).
// The icons array is rebuilt from `present` (replacing, never appending) so a
// re-apply produces no duplicate entries. theme_color/background_color come from
// the brand's colors when set.
func mergeIconsManifest(existing []byte, brand BrandView, present []iconAsset) ([]byte, error) {
	manifest := map[string]any{}
	if len(existing) > 0 {
		_ = json.Unmarshal(existing, &manifest)
	}

	if len(present) > 0 {
		arr := make([]any, 0, len(present))
		for _, ic := range present {
			entry := map[string]any{
				"src":   "/" + ic.filename,
				"sizes": ic.sizes,
				"type":  "image/png",
			}
			if ic.purpose != "" {
				entry["purpose"] = ic.purpose
			}
			arr = append(arr, entry)
		}
		manifest["icons"] = arr
		manifest["_brand_icons_version"] = brand.Version
	}
	if brand.Colors.Primary != "" {
		manifest["theme_color"] = brand.Colors.Primary
	}
	if brand.Colors.Background != "" {
		manifest["background_color"] = brand.Colors.Background
	}

	out, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}
