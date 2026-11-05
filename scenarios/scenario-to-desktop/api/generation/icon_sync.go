package generation

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	"image/png"
	"os"
	"path/filepath"
)

// placeholderHashes are the known placeholder bytes that must be replaced, not
// kept. Registered in brand-manager's declared-icon-targets placeholder registry.
var placeholderHashes = map[string]bool{
	"80c4d8d2859afab47cd0a5b85f67d02d9fe0247e521cb980a9e462604d006e24": true, // react-vite apple-icon-180
	"d6fc31f168b5e32c1e7117faa208dc451e82a21d3b6fe2f30c45a47bffe8cb2a": true, // react-vite favicon-196
	"3ab1dfe4d938bc2a2c1f3d2b300828ed97c65d1a113128866b743b97aff2f654": true, // react-vite manifest-icon-192.maskable
	"7e41bbd6f20456270710e878b88cedae4175465dd28adad0905dc2d8c5f24796": true, // react-vite manifest-icon-512.maskable
	"75ccd7d14e470b80a07986ee693ba755aff663d9cb346aa7c492d04d3e8c8f78": true, // desktop 256x256 placeholder (663 B)
	"93d376406bbc46cab650351a81b16d8b4a36adc9979b299c27413d5527c0dc7d": true, // desktop 512x512 placeholder (1817 B)
	"8afc9cb0e46be8347433188dcacedbc95b8424a0f3e21a837d120dc47f2a4f59": true, // desktop 16x16 placeholder
	"c8734c930135bfe23762c4f2c211d55c40de38ae7e7880cb041f969e8bf3aa6b": true, // desktop 32x32 placeholder
	"05c38a2deadb0a6fd54a3dcfd747898aac4da476288aae1d2da06c4573537465": true, // desktop 48x48 placeholder
	"ab53df3db0985319918519cb37e3ceea5a86ba969a08442cbfa4f0f6d6eca038": true, // desktop 128x128 placeholder
	"d5980c80fb6855432d115ade65585585ece71982608d1b77351b11fbe749052b": true, // desktop 1024x1024 placeholder
	"6ec56d91b27232305c9271800b7f36af478b08a3bdcccbb11b0fbc0ed85f1fde": true, // react-vite apple-touch-icon.png
	"182f80d2cc53eda7d4eb1afa6ca43da802a127793140656b59163aabca42f081": true, // react-vite favicon-16.png
	"bea3831ffad435516291003e907a6160e5834b75f7ac625eadd99f091b839e09": true, // react-vite favicon-196.png
	"4233184e0cad164596484b95c2d199ce3ce17e589d48a66a8b8a2d64b072305f": true, // react-vite favicon-32.png
	"8a251d975c3484cfc7d6eebb35b2e28360f6f4cdcf18502d2f642b73aad7b602": true, // react-vite icon-192.png
	"2970439bc43d56cd8ded8d2375dbe03c25d7d57d76fc0d4f5b72de9e3806fbaa": true, // react-vite icon-512.png
	"a3cdc03013621d05ec5e42aba24235ae452fd785f0adae153523386dee396255": true, // react-vite og-image.png
	"9e45b711f2317d1930df7595c9d49d437c81ac185aa9fff20738804eb9b8417b": true, // react-vite logo.svg
}

// iconTargets maps a desktop asset filename to its exact pixel size.
var iconTargets = []struct {
	Name string
	Size int
}{
	{"icon.png", 512},
	{"icon-1024x1024.png", 1024},
	{"icon-512x512.png", 512},
	{"icon-256x256.png", 256},
	{"icon-128x128.png", 128},
	{"icon-64x64.png", 64},
	{"icon-48x48.png", 48},
	{"icon-32x32.png", 32},
	{"icon-16x16.png", 16},
}

// DefaultIconSyncer is the default implementation of IconSyncer.
type DefaultIconSyncer struct {
	vrooliRoot string
}

// NewIconSyncer creates a new icon syncer.
func NewIconSyncer(vrooliRoot string) *DefaultIconSyncer {
	return &DefaultIconSyncer{vrooliRoot: vrooliRoot}
}

// SyncIcons fills the Electron assets directory with correctly sized icons,
// preferring brand-manager-applied assets and never overwriting a correct one.
// It is idempotent: a second sync over correct assets writes nothing.
func (s *DefaultIconSyncer) SyncIcons(scenario, outputDir string, logf func(string, map[string]interface{})) error {
	assetsDir := filepath.Join(outputDir, "assets")

	if s.allTargetsCorrect(assetsDir) {
		return nil // brand-manager's correctly sized set is already in place
	}

	srcPath := s.bestSource(scenario, outputDir)
	if srcPath == "" {
		return nil // nothing to derive from
	}
	src, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("read icon: %w", err)
	}
	decoded, _, err := image.Decode(bytes.NewReader(src))
	if err != nil {
		return fmt.Errorf("decode icon %s: %w", srcPath, err)
	}

	if err := os.MkdirAll(assetsDir, 0o755); err != nil {
		return fmt.Errorf("create assets dir: %w", err)
	}
	for _, target := range iconTargets {
		dst := filepath.Join(assetsDir, target.Name)
		if correctSize(dst, target.Size) && !isPlaceholder(dst) {
			continue // keep a correct, non-placeholder asset
		}
		scaled := scaleNearest(decoded, target.Size, target.Size)
		var buf bytes.Buffer
		if err := png.Encode(&buf, scaled); err != nil {
			return fmt.Errorf("encode %s: %w", target.Name, err)
		}
		if err := os.WriteFile(dst, buf.Bytes(), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", target.Name, err)
		}
	}

	if logf != nil {
		logf("info", map[string]interface{}{
			"msg":       "synced scenario icon into desktop assets",
			"scenario":  scenario,
			"source":    srcPath,
			"assetsDir": assetsDir,
		})
	}
	return nil
}

// bestSource returns the first available source in preference order:
// brand-manager's electron-v1 assets, then the web-public-v1 icon-512, then the
// legacy renderer/dist names.
func (s *DefaultIconSyncer) bestSource(scenario, outputDir string) string {
	candidates := []string{
		filepath.Join(s.vrooliRoot, "scenarios", scenario, "platforms", "electron", "assets", "icon.png"),
		filepath.Join(s.vrooliRoot, "scenarios", scenario, "ui", "public", "public", "icon-512.png"),
		filepath.Join(outputDir, "renderer", "manifest-icon-512.maskable.png"),
		filepath.Join(outputDir, "renderer", "apple-icon-180.png"),
		filepath.Join(s.vrooliRoot, "scenarios", scenario, "ui", "dist", "manifest-icon-512.maskable.png"),
	}
	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && info.Mode().IsRegular() {
			return c
		}
	}
	return ""
}

func (s *DefaultIconSyncer) allTargetsCorrect(assetsDir string) bool {
	for _, target := range iconTargets {
		dst := filepath.Join(assetsDir, target.Name)
		if !correctSize(dst, target.Size) || isPlaceholder(dst) {
			return false
		}
	}
	return true
}

// correctSize reports whether path is a PNG of exactly w×h.
func correctSize(path string, w int) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return false
	}
	return cfg.Width == w && cfg.Height == w
}

func isPlaceholder(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	sum := sha256.Sum256(data)
	return placeholderHashes[hex.EncodeToString(sum[:])]
}

// scaleNearest scales src to w×h with nearest-neighbour sampling. It needs no
// external resize tool, so desktop packaging works on any host.
func scaleNearest(src image.Image, w, h int) *image.NRGBA {
	dst := image.NewNRGBA(image.Rect(0, 0, w, h))
	b := src.Bounds()
	sw, sh := b.Dx(), b.Dy()
	if sw == 0 || sh == 0 {
		return dst
	}
	for y := 0; y < h; y++ {
		sy := b.Min.Y + y*sh/h
		for x := 0; x < w; x++ {
			sx := b.Min.X + x*sw/w
			r, g, bl, a := src.At(sx, sy).RGBA()
			dst.SetNRGBA(x, y, color.NRGBA{uint8(r >> 8), uint8(g >> 8), uint8(bl >> 8), uint8(a >> 8)})
		}
	}
	return dst
}
