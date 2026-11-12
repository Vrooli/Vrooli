package resources

import (
	"bytes"
	"context"
	"debug/macho"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/binaryfetch"
)

func openRGBMachTestCommand(command uint32, name string, minimum int) []byte {
	size := (minimum + len(name) + 1 + 7) &^ 7
	raw := make([]byte, size)
	binary.LittleEndian.PutUint32(raw, command)
	binary.LittleEndian.PutUint32(raw[4:], uint32(size))
	binary.LittleEndian.PutUint32(raw[8:], uint32(minimum))
	copy(raw[minimum:], name)
	return raw
}

func openRGBMachTestImage(commands ...[]byte) []byte {
	size := 0
	for _, command := range commands {
		size += len(command)
	}
	image := make([]byte, 32+size)
	for i, n := range []uint32{0xfeedfacf, 0x100000c, 0, 2, uint32(len(commands)), uint32(size), 0, 0} {
		binary.LittleEndian.PutUint32(image[i*4:], n)
	}
	at := 32
	for _, command := range commands {
		copy(image[at:], command)
		at += len(command)
	}
	return image
}

func TestOpenRGBMachEveryLoaderFormRequiresQualifiedDependency(t *testing.T) {
	p, _ := openRGBReleaseProfileFor("darwin", "arm64")
	for _, command := range []uint32{0xc, 0x80000018, 0x8000001f, 0x80000023, 0x20} {
		t.Run(fmt.Sprintf("%x", command), func(t *testing.T) {
			root := t.TempDir()
			target := filepath.Join(root, filepath.FromSlash(p.Payload))
			if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
				t.Fatal(err)
			}
			write := func(name string) {
				t.Helper()
				b := openRGBMachTestImage(openRGBMachTestCommand(0xe, "/usr/lib/dyld", 12), openRGBMachTestCommand(command, name, 24))
				if err := os.WriteFile(target, b, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			write("/usr/lib/required.dylib")
			seen := false
			if err := p.verifyDarwinLoaderCache(t.Context(), root, func(name string) (bool, error) { seen = name == "/usr/lib/required.dylib"; return false, nil }); err == nil || !seen {
				t.Fatal("missing dependency escaped full-form cache check")
			}
			if err := p.verifyDarwinLoaderCache(t.Context(), root, func(string) (bool, error) { return true, nil }); err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"/Users/operator/evil.dylib", "@loader_path/../../../evil.dylib", "@rpath/missing.dylib", "/usr/lib/../local/evil.dylib"} {
				write(name)
				if _, err := p.darwinLoaderSystemImages(t.Context(), root); err == nil {
					t.Fatalf("foreign/unresolved loader %#x accepted %s", command, name)
				}
			}
		})
	}
}

func TestOpenRGBMachRejectsMalformedLoaderAndEnvironment(t *testing.T) {
	p, _ := openRGBReleaseProfileFor("darwin", "arm64")
	for _, kind := range []string{"custom-dyld", "embedded-environment", "foreign-runpath", "unknown-required", "truncated", "bad-name-offset", "unterminated"} {
		t.Run(kind, func(t *testing.T) {
			raw := openRGBMachTestImage(openRGBMachTestCommand(0xe, "/usr/lib/dyld", 12))
			switch kind {
			case "custom-dyld":
				raw = openRGBMachTestImage(openRGBMachTestCommand(0xe, "/Users/operator/dyld", 12))
			case "embedded-environment":
				raw = openRGBMachTestImage(openRGBMachTestCommand(0xe, "/usr/lib/dyld", 12), openRGBMachTestCommand(0x27, "DYLD_LIBRARY_PATH=/Users/operator", 12))
			case "foreign-runpath":
				raw = openRGBMachTestImage(openRGBMachTestCommand(0xe, "/usr/lib/dyld", 12), openRGBMachTestCommand(0x8000001c, "/Users/operator", 12))
			case "unknown-required":
				binary.LittleEndian.PutUint32(raw[32:], 0x80007777)
			case "truncated":
				raw = raw[:len(raw)-1]
			case "bad-name-offset":
				binary.LittleEndian.PutUint32(raw[40:], 8)
			case "unterminated":
				for i := 44; i < len(raw); i++ {
					raw[i] = 'x'
				}
			}
			root := t.TempDir()
			target := filepath.Join(root, filepath.FromSlash(p.Payload))
			if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(target, raw, 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := p.darwinLoaderSystemImages(t.Context(), root); err == nil {
				t.Fatal("unqualified loader metadata accepted")
			}
		})
	}
}

func TestOpenRGBMacRecipeMatchesCanonicalDeclaration(t *testing.T) {
	for _, arch := range []string{"amd64", "arm64"} {
		p, _ := openRGBReleaseProfileFor("darwin", arch)
		raw, err := os.ReadFile(filepath.Join(projectRootForResourcesTest(t), "resources", "openrgb", "config", "darwin-"+arch+"-headless-serving-recipe.json"))
		if err != nil {
			t.Fatal(err)
		}
		var d any
		if err = json.Unmarshal(raw, &d); err != nil {
			t.Fatal(err)
		}
		canonical, err := json.Marshal(d)
		if err != nil {
			t.Fatal(err)
		}
		expected, err := p.servingRecipeBytes()
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(canonical, expected) || p.validate() != nil {
			t.Fatal("Mac recipe differs from compiled profile")
		}
	}
}

func TestOpenRGBMachForeignRunpathCannotFlowThroughDependencyChain(t *testing.T) {
	images := map[string]*openRGBMachLoaderImage{
		"parent": {foreignRunpath: true, children: []string{"middle"}},
		"middle": {children: []string{"leaf"}},
		"leaf":   {},
	}
	if err := verifyOpenRGBMachRunpathChains(images); err != nil {
		t.Fatal(err)
	}
	images["leaf"].runpathLoad = true
	if err := verifyOpenRGBMachRunpathChains(images); err == nil {
		t.Fatal("inherited foreign runpath reached a descendant load")
	}
	images["leaf"].children = []string{"parent"}
	if err := verifyOpenRGBMachRunpathChains(images); err == nil {
		t.Fatal("dependency cycle concealed inherited foreign runpath")
	}
	images["leaf"].runpathLoad = false
	if err := verifyOpenRGBMachRunpathChains(images); err != nil {
		t.Fatal("inert closed cycle refused", err)
	}
}

func TestOpenRGBMachPrivateRunpathCandidatesMustAgree(t *testing.T) {
	p, _ := openRGBReleaseProfileFor("darwin", "arm64")
	root := t.TempDir()
	main := filepath.Join(root, filepath.FromSlash(p.Payload))
	frameworks, shadow := filepath.Join(root, "Contents", "Frameworks"), filepath.Join(root, "Contents", "Shadow")
	for _, dir := range []string{filepath.Dir(main), frameworks, shadow} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	image := openRGBMachTestImage(openRGBMachTestCommand(0xe, "/usr/lib/dyld", 12), openRGBMachTestCommand(0x8000001c, "@executable_path/../Frameworks", 12), openRGBMachTestCommand(0x8000001c, "@executable_path/../Shadow", 12), openRGBMachTestCommand(0xc, "@rpath/choice.dylib", 24))
	if err := os.WriteFile(main, image, 0o600); err != nil {
		t.Fatal(err)
	}
	dylib := openRGBMachTestImage(openRGBMachTestCommand(0xd, "@rpath/choice.dylib", 24))
	binary.LittleEndian.PutUint32(dylib[12:], 6)
	first, second := filepath.Join(frameworks, "choice.dylib"), filepath.Join(shadow, "choice.dylib")
	if err := os.WriteFile(first, dylib, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := p.darwinLoaderSystemImages(t.Context(), root); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(second, dylib, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := p.darwinLoaderSystemImages(t.Context(), root); err == nil {
		t.Fatal("different private images accepted under ambiguous runpath order")
	}
	if err := os.Remove(second); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join("..", "Frameworks", "choice.dylib"), second); err != nil {
		t.Fatal(err)
	}
	if _, err := p.darwinLoaderSystemImages(t.Context(), root); err != nil {
		t.Fatal("same in-tree canonical target refused", err)
	}
}

func TestOpenRGBCompleteMacLoaderOfflineEvidence(t *testing.T) {
	root := os.Getenv("VROOLI_OPENRGB_PROFILE_MEASUREMENT_ROOT")
	if root == "" {
		t.Skip("explicit official Mac artifact metadata qualification only")
	}
	for _, tc := range []struct{ arch, name string }{{"amd64", "OpenRGB_1.1_MacOS_Intel_8e16de2.zip"}, {"arm64", "OpenRGB_1.1_MacOS_ARM64_8e16de2.zip"}} {
		t.Run(tc.arch, func(t *testing.T) {
			p, err := openRGBReleaseProfileFor("darwin", tc.arch)
			if err != nil {
				t.Fatal(err)
			}
			archive := filepath.Join(root, tc.name)
			if err = binaryfetch.VerifyArtifactContext(t.Context(), archive, "file", p.EnvelopeSHA256); err != nil {
				t.Fatal(err)
			}
			file, err := os.Open(archive)
			if err != nil {
				t.Fatal(err)
			}
			defer file.Close()
			bundle := filepath.Join(t.TempDir(), "OpenRGB.app")
			if err = os.Mkdir(bundle, 0o700); err != nil {
				t.Fatal(err)
			}
			if err = extractOpenRGBZip(t.Context(), file, bundle, p); err != nil {
				t.Fatal(err)
			}
			if err = binaryfetch.VerifyArtifactContext(t.Context(), bundle, "dir", p.TreeSHA256); err != nil {
				t.Fatal(err)
			}
			if _, err = p.darwinLoaderSystemImages(context.Background(), bundle); err == nil {
				t.Fatal("original foreign Homebrew dependency was silently admitted")
			}
			frameworkTree, err := binaryfetch.TreeDigest(filepath.Join(bundle, "Contents", "Frameworks", "QtCore5Compat.framework"))
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("omitted framework tree=%s", frameworkTree)
			if err = p.deriveServingTree(t.Context(), bundle); err != nil {
				t.Fatal(err)
			}
			if err = p.verifyBundleABI(t.Context(), bundle); err != nil {
				t.Fatal(err)
			}
			for _, leaf := range []string{"libdbus-1.3.dylib", "libjasper.7.dylib", "libjpeg.8.dylib"} {
				data, err := os.ReadFile(filepath.Join(bundle, "Contents", "Frameworks", leaf))
				if err != nil {
					t.Fatal(err)
				}
				image, err := macho.NewFile(bytes.NewReader(data))
				if err != nil {
					t.Fatal(err)
				}
				symbols, err := image.ImportedSymbols()
				if err != nil {
					t.Fatal(err)
				}
				for _, symbol := range symbols {
					for _, loader := range []string{"dlopen", "dlsym", "nscreateobjectfileimage", "nslinkmodule", "nslookup", "nsaddimage", "_dyld_", "objc", "qlibrary", "qplugin"} {
						if strings.Contains(strings.ToLower(symbol), loader) {
							t.Fatalf("inert runpath image %s imports loader entry %s", leaf, symbol)
						}
					}
				}
				for _, section := range image.Sections {
					if section.Flags&0xff == 9 || section.Flags&0xff == 0x16 {
						t.Fatalf("inert runpath image %s has constructor section %s", leaf, section.Name)
					}
				}
				for _, load := range image.Loads {
					if image.ByteOrder.Uint32(load.Raw()) == 0x1a {
						t.Fatal("inert runpath image contains constructor load command")
					}
				}
				for _, name := range []string{"dlopen", "dlsym", "NSCreateObjectFileImage", "NSLinkModule", "@rpath/"} {
					if bytes.Contains(data, []byte(name)) {
						t.Fatalf("inert runpath image contains dynamic loader name %s", name)
					}
				}
			}
			systems, err := p.darwinLoaderSystemImages(t.Context(), bundle)
			if err != nil {
				t.Fatal(err)
			}
			tree, err := binaryfetch.TreeDigest(bundle)
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("candidate derived tree=%s systemimages=%d", tree, len(systems))
			if tree != p.servingTreeSHA256() {
				t.Fatal("derived Mac closure identity mismatch")
			}
			if err = os.Mkdir(filepath.Join(filepath.Dir(bundle), "lib"), 0o700); err != nil {
				t.Fatal(err)
			}
			if _, err = p.darwinLoaderSystemImages(t.Context(), bundle); err == nil {
				t.Fatal("adjacent plugin runpath provider accepted")
			}
		})
	}
}
