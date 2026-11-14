package resources

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vrooli/binaryfetch"
	platform "github.com/vrooli/platform-go"
)

type openRGBMachLoaderImage struct {
	metadata       platform.MachOLoaderSlice
	runpaths       []string
	foreignRunpath bool
	runpathLoad    bool
	children       []string
}

func openRGBMachSystemPath(name string) bool {
	return path.Clean(name) == name && (strings.HasPrefix(name, "/usr/lib/") || strings.HasPrefix(name, "/System/Library/"))
}

func openRGBMachWithin(root, name string) bool {
	rel, err := filepath.Rel(root, name)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func openRGBMachRelativePath(root, image, executable, name string) (string, error) {
	var base, rest string
	switch {
	case strings.HasPrefix(name, "@loader_path/"):
		base, rest = filepath.Dir(image), strings.TrimPrefix(name, "@loader_path/")
	case strings.HasPrefix(name, "@executable_path/"):
		base, rest = filepath.Dir(executable), strings.TrimPrefix(name, "@executable_path/")
	default:
		return "", fmt.Errorf("unqualified Mach-O private path %q", name)
	}
	result := filepath.Clean(filepath.Join(base, filepath.FromSlash(rest)))
	if !openRGBMachWithin(root, result) {
		return "", fmt.Errorf("Mach-O private path escapes bundle: %q", name)
	}
	return result, nil
}

// This proves declared Mach-O loader edges, not arbitrary runtime dlopen calls.
// The caller independently proves the complete pinned immutable bundle. Every
// supported load-command form comes from the shared held-reader parser.
func (p openRGBReleaseProfile) darwinLoaderSystemImages(ctx context.Context, bundle string) ([]string, error) {
	if p.OS != "darwin" {
		return nil, errors.New("Mach-O loader proof requires a Darwin profile")
	}
	root, err := filepath.EvalSymlinks(bundle)
	if err != nil || !filepath.IsAbs(root) {
		return nil, errors.Join(errors.New("Mach-O loader bundle is not canonical"), err)
	}
	executable := filepath.Join(root, filepath.FromSlash(p.Payload))
	images := map[string]*openRGBMachLoaderImage{}
	err = filepath.WalkDir(root, func(name string, entry os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if e = ctx.Err(); e != nil {
			return e
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		file, e := os.Open(name)
		if e != nil {
			return e
		}
		defer file.Close()
		var magic [4]byte
		if _, e = file.ReadAt(magic[:], 0); e != nil {
			return nil
		}
		if magic != [4]byte{0xcf, 0xfa, 0xed, 0xfe} {
			return nil
		}
		info, e := file.Stat()
		if e != nil {
			return e
		}
		metadata, e := platform.ReadMachOLoaderMetadata(file, info.Size())
		if e != nil {
			return fmt.Errorf("Mach-O loader metadata %s: %w", name, e)
		}
		if len(metadata.Slices) != 1 || len(images) >= 256 {
			return errors.New("unsupported Mach-O bundle image inventory")
		}
		m := metadata.Slices[0]
		cpu := uint32(0x1000007)
		if p.Arch == "arm64" {
			cpu = 0x100000c
		}
		if m.CPU != cpu || len(m.DyldEnvironment) != 0 || m.Dylinker != "" && (m.FileType != 2 || m.Dylinker != "/usr/lib/dyld") {
			return fmt.Errorf("Mach-O image has foreign architecture, loader or embedded environment: %s", name)
		}
		images[name] = &openRGBMachLoaderImage{metadata: m}
		return nil
	})
	if err != nil {
		return nil, err
	}
	main := images[executable]
	if main == nil || main.metadata.FileType != 2 || main.metadata.Dylinker != "/usr/lib/dyld" {
		return nil, errors.New("Mach-O entry image has no admitted system loader")
	}
	var allRunpaths []string
	for name, image := range images {
		for _, rpath := range image.metadata.RPaths {
			if strings.HasPrefix(rpath, "/") {
				if path.Clean(rpath) != rpath {
					return nil, errors.New("noncanonical Mach-O runpath")
				}
				if err := p.verifyInertMachBuildRunpath(ctx, root, name, rpath); err != nil {
					return nil, err
				}
				image.foreignRunpath = true
				continue
			}
			dir, e := openRGBMachRelativePath(root, name, executable, rpath)
			if e != nil {
				// macdeployqt retains a plugin build runpath pointing at the
				// adjacent stage/lib. That exact sibling must remain absent in
				// the immutable stage; it is never an external library provider.
				if rpath != "@loader_path/../../../../lib" || !strings.HasPrefix(name, filepath.Join(root, "Contents", "PlugIns")+string(filepath.Separator)) {
					return nil, e
				}
				dir = filepath.Clean(filepath.Join(filepath.Dir(name), "..", "..", "..", "..", "lib"))
				if dir != filepath.Join(filepath.Dir(root), "lib") {
					return nil, e
				}
				if _, statErr := os.Lstat(dir); !os.IsNotExist(statErr) {
					return nil, errors.Join(errors.New("Mach-O plugin sibling runpath is not immutably absent"), statErr)
				}
			}
			image.runpaths = append(image.runpaths, dir)
			allRunpaths = append(allRunpaths, dir)
		}
	}
	systems := map[string]bool{}
	privateTarget := func(name string) (string, error) {
		resolved, e := filepath.EvalSymlinks(name)
		if e != nil {
			return "", e
		}
		if !openRGBMachWithin(root, resolved) || images[resolved] == nil {
			return "", fmt.Errorf("Mach-O dependency is not a native image inside the pinned bundle: %s", name)
		}
		return resolved, nil
	}
	for name, image := range images {
		for _, load := range image.metadata.Dylibs {
			library := load.Path
			if openRGBMachSystemPath(library) {
				systems[library] = true
				continue
			}
			if strings.HasPrefix(library, "/") {
				return nil, fmt.Errorf("foreign absolute Mach-O dependency %s in %s", library, name)
			}
			var target string
			if strings.HasPrefix(library, "@rpath/") {
				image.runpathLoad = true
				rest := strings.TrimPrefix(library, "@rpath/")
				if rest == "" || path.Clean(rest) != rest || strings.HasPrefix(rest, "../") {
					return nil, errors.New("unsafe Mach-O runpath dependency")
				}
				resolve := func(dirs []string) (string, error) {
					var found string
					for _, dir := range dirs {
						candidate, e := privateTarget(filepath.Join(dir, filepath.FromSlash(rest)))
						if os.IsNotExist(e) {
							continue
						}
						if e != nil {
							return "", e
						}
						if found != "" && found != candidate {
							return "", errors.New("ambiguous Mach-O runpath dependency")
						}
						found = candidate
					}
					return found, nil
				}
				// Self and the fixed executable establish a finite usable path.
				// Every other private runpath must agree, regardless of chain order.
				var e error
				target, e = resolve(append(append([]string{}, image.runpaths...), main.runpaths...))
				if e != nil {
					return nil, e
				}
				if target == "" {
					return nil, fmt.Errorf("unresolved Mach-O runpath dependency %s in %s", library, name)
				}
				all, e := resolve(allRunpaths)
				if e != nil || all != target {
					return nil, errors.Join(errors.New("Mach-O runpath scope is ambiguous"), e)
				}
			} else {
				candidate, e := openRGBMachRelativePath(root, name, executable, library)
				if e != nil {
					return nil, e
				}
				target, e = privateTarget(candidate)
				if e != nil {
					return nil, e
				}
			}
			image.children = append(image.children, target)
		}
	}
	if err := verifyOpenRGBMachRunpathChains(images); err != nil {
		return nil, err
	}
	result := make([]string, 0, len(systems))
	for name := range systems {
		result = append(result, name)
	}
	sort.Strings(result)
	return result, nil
}

func (p openRGBReleaseProfile) verifyInertMachBuildRunpath(ctx context.Context, root, name, rpath string) error {
	// These exact C libraries have no dynamic-loader/Qt/ObjC imports, dynamic
	// loader-name bytes or initializer sections in the pinned release. Their
	// entire declared descendant graph must additionally contain no @rpath load.
	// This is a bounded release exception, never a generic dlopen guarantee.
	intel := map[string]string{
		"libdbus-1.3.dylib": "b81e2f25be3d3fd24447570fc596156e37e78bf61003420233759849435e6097",
		"libjasper.7.dylib": "ca5c5a93576de7f84471cc97d7f7b86525ccabe6c312c9901adf295702dd42f1",
		"libjpeg.8.dylib":   "145a752e0b09c51d6e93690b2591232d5facd8be06729052732033f5efc81b9d",
	}
	arm := map[string]string{
		"libdbus-1.3.dylib": "2ece2fe19cb41612ffcdb8c0c30fa5266b0766d3ea3a511db40799cbf9f283d9",
		"libjasper.7.dylib": "b653b7922da5f1dd2c3d2e84e3f2bb91d9e271a9187a811028ae2ab0d12b2f0d",
		"libjpeg.8.dylib":   "1531f01d2382deefcf58b81720ce7d52a17f2ecf48e14f9ba298cee0806f71b1",
	}
	paths := map[string]string{"libdbus-1.3.dylib": "dbus/1.16.2_1/lib", "libjasper.7.dylib": "jasper/4.2.9/lib", "libjpeg.8.dylib": "jpeg-turbo/3.1.4.1/lib"}
	prefix, hashes := "/usr/local/Cellar/", intel
	if p.Arch == "arm64" {
		prefix, hashes = "/opt/homebrew/Cellar/", arm
	}
	base := filepath.Base(name)
	hash := hashes[base]
	if hash == "" || name != filepath.Join(root, "Contents", "Frameworks", base) || rpath != prefix+paths[base] {
		return fmt.Errorf("unqualified external Mach-O runpath %q", rpath)
	}
	return binaryfetch.VerifyArtifactContext(ctx, name, "file", hash)
}

func (p openRGBReleaseProfile) verifyDarwinLoaderCache(ctx context.Context, bundle string, contains func(string) (bool, error)) error {
	libraries, err := p.darwinLoaderSystemImages(ctx, bundle)
	if err != nil {
		return err
	}
	for _, library := range libraries {
		if err := ctx.Err(); err != nil {
			return err
		}
		present, err := contains(library)
		if err != nil {
			return err
		}
		if !present {
			return fmt.Errorf("OpenRGB native system image %s is absent from active dyld cache; native ABI is unqualified", library)
		}
	}
	return nil
}

func verifyOpenRGBMachRunpathChains(images map[string]*openRGBMachLoaderImage) error {
	// An absolute build-time runpath is inert only when it cannot participate
	// in any declared @rpath dependency in that image's entire descendant graph.
	for name, image := range images {
		if !image.foreignRunpath {
			continue
		}
		seen := map[string]bool{}
		var visit func(string) bool
		visit = func(name string) bool {
			if seen[name] {
				return false
			}
			seen[name] = true
			current := images[name]
			if current.runpathLoad {
				return true
			}
			for _, child := range current.children {
				if visit(child) {
					return true
				}
			}
			return false
		}
		if visit(name) {
			return fmt.Errorf("foreign Mach-O runpath can affect dependency loading: %s", name)
		}
	}
	return nil
}
