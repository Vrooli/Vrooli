// Package clifreshness keeps a development checkout's root CLI from serving a
// command surface older than the checkout itself.
//
// It is deliberately narrow. The only signal is the checkout's command
// contract, `cli/manifest.json`, compared with the copy this build embedded.
// That file is small and authored, so the check costs one file read and one
// comparison, works on every platform, and needs no git state — which matters
// because the worktree is shared by many agents and is almost never clean.
//
// A refresh builds only the repository's own CLI into `.vrooli/build/vrooli`.
// It never replaces the managed image in the runtime home: that image is owned
// by root activation and is the rollback target, so an out-of-band replacement
// would destroy the rollback contract. When the checkout does not compile, the
// command runs on the build it already had.
package clifreshness

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	embeddedmanifest "github.com/vrooli/vrooli/cli"
)

// surface is the part of a command manifest that decides whether a build can
// serve an invocation: the command paths and their flag names. Descriptions,
// governance metadata and prose never participate, so editing them cannot force
// a rebuild.
type surface struct {
	Commands []commandPath
}

type commandPath struct {
	Path  string
	Flags []string
}

// CommandSurfaceBehind reports whether this build's command surface is older
// than root's authored manifest.
func CommandSurfaceBehind(root string) (bool, error) {
	checkout, err := checkoutSurfaceDigest(root)
	if err != nil {
		return false, err
	}
	embedded, err := EmbeddedSurfaceDigest()
	if err != nil {
		return false, err
	}
	return checkout != embedded, nil
}

// CheckoutSurfaceDigest is the canonical digest of root's authored command
// surface. It is the only value a local build records, so a sibling process can
// tell whether the published build already serves this checkout.
func CheckoutSurfaceDigest(root string) (string, error) { return checkoutSurfaceDigest(root) }

// EmbeddedSurfaceDigest is the canonical digest of the command surface this
// binary was compiled with.
func EmbeddedSurfaceDigest() (string, error) {
	return surfaceDigest(embeddedmanifest.Bytes())
}

func checkoutSurfaceDigest(root string) (string, error) {
	manifestPath := filepath.Join(root, "cli", "manifest.json")
	source, err := os.ReadFile(manifestPath)
	if err != nil {
		return "", err
	}
	digest, err := surfaceDigest(source)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", manifestPath, err)
	}
	return digest, nil
}

func surfaceDigest(data []byte) (string, error) {
	parsed, err := parseSurface(data)
	if err != nil {
		return "", err
	}
	canonical, err := json.Marshal(parsed)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(canonical)
	return hex.EncodeToString(sum[:]), nil
}

func parseSurface(data []byte) (surface, error) {
	var document struct {
		Groups []manifestGroup `json:"groups"`
	}
	if err := json.Unmarshal(data, &document); err != nil {
		return surface{}, err
	}
	collected := surface{}
	collectGroups(document.Groups, "", &collected)
	sort.Slice(collected.Commands, func(i, j int) bool { return collected.Commands[i].Path < collected.Commands[j].Path })
	return collected, nil
}

type manifestGroup struct {
	Name     string          `json:"name"`
	Groups   []manifestGroup `json:"groups"`
	Commands []manifestItem  `json:"commands"`
}

type manifestItem struct {
	Name  string         `json:"name"`
	Flags []manifestFlag `json:"flags"`
}

type manifestFlag struct {
	Name string `json:"name"`
}

func collectGroups(groups []manifestGroup, prefix string, collected *surface) {
	for _, group := range groups {
		path := joinPath(prefix, group.Name)
		collectGroups(group.Groups, path, collected)
		for _, command := range group.Commands {
			flags := make([]string, 0, len(command.Flags))
			for _, flag := range command.Flags {
				flags = append(flags, flag.Name)
			}
			sort.Strings(flags)
			collected.Commands = append(collected.Commands, commandPath{Path: joinPath(path, command.Name), Flags: flags})
		}
	}
}

func joinPath(prefix, name string) string {
	if prefix == "" {
		return name
	}
	return prefix + " " + name
}

// equalSurfaces treats adding a command or a flag as a difference, because the
// older build cannot serve it. Removing one is also a difference: the older
// build still offers something the checkout no longer declares, and running it
// would act on a command that no longer exists.
func equalSurfaces(a, b surface) bool {
	if len(a.Commands) != len(b.Commands) {
		return false
	}
	for i := range a.Commands {
		if a.Commands[i].Path != b.Commands[i].Path {
			return false
		}
		if len(a.Commands[i].Flags) != len(b.Commands[i].Flags) {
			return false
		}
		for j := range a.Commands[i].Flags {
			if a.Commands[i].Flags[j] != b.Commands[i].Flags[j] {
				return false
			}
		}
	}
	return true
}
