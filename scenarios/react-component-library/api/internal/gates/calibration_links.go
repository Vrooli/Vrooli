package gates

// Calibration is the anti-fabrication boundary for catalog gates. A fixture is
// a small, real catalog asset with one deliberately planted defect. The gate
// is run against an isolated overlay containing that asset; a clean result is
// therefore a failed calibration, not evidence of quality.

import (
	"os"
	"path/filepath"
)

func pruneLibraryOverlay(root string) error {
	overlayRoot, err := os.OpenRoot(root)
	if err != nil {
		return err
	}
	defer overlayRoot.Close()
	for _, kind := range []string{"foundations", "hooks", "services", "primitives", "components"} {
		dir := filepath.Join(root, "scenarios", "react-component-library", "library", kind)
		entries, err := os.ReadDir(dir)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if err := overlayRemove(overlayRoot, filepath.Join(dir, entry.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}

func linkScenarioOverlay(root, tmp string) error {
	overlayRoot, err := os.OpenRoot(tmp)
	if err != nil {
		return err
	}
	defer overlayRoot.Close()
	realScenario := filepath.Join(root, "scenarios", "react-component-library")
	tmpScenario := filepath.Join(tmp, "scenarios", "react-component-library")
	for _, path := range []string{"ui", "catalog/config.json", "catalog/weights.json", "catalog/version-shape.json", "templates", "harnesses", "data"} {
		source := filepath.Join(realScenario, path)
		if _, err := os.Stat(source); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		dest := filepath.Join(tmpScenario, path)
		if err := overlayMkdirAll(overlayRoot, filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		if path == "ui" {
			if err := copyUIOverlayInRoot(overlayRoot, source, dest); err != nil {
				return err
			}
			continue
		}
		if path == "harnesses" {
			if err := copyDirectoryFiles(source, dest); err != nil {
				return err
			}
			continue
		}
		if path == "data" {
			if err := copyDataOverlay(source, dest); err != nil {
				return err
			}
			continue
		}
		if err := overlaySymlink(overlayRoot, source, dest); err != nil {
			return err
		}
	}
	for _, kind := range []string{"foundations", "hooks", "services", "primitives", "components"} {
		destKind := filepath.Join(tmpScenario, "library", kind)
		if err := overlayMkdirAll(overlayRoot, destKind, 0o755); err != nil {
			return err
		}
		entries, err := os.ReadDir(filepath.Join(realScenario, "library", kind))
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if err := overlaySymlink(overlayRoot, filepath.Join(realScenario, "library", kind, entry.Name()), filepath.Join(destKind, entry.Name())); err != nil {
				return err
			}
		}
	}
	assetsRoot := filepath.Join(tmpScenario, "catalog", "assets")
	if err := overlayMkdirAll(overlayRoot, assetsRoot, 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(filepath.Join(realScenario, "catalog", "assets"))
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if err := overlaySymlink(overlayRoot, filepath.Join(realScenario, "catalog", "assets", entry.Name()), filepath.Join(assetsRoot, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

func copyUIOverlayInRoot(root *os.Root, source, dest string) error {
	if err := overlayMkdirAll(root, dest, 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(source)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		src := filepath.Join(source, entry.Name())
		dst := filepath.Join(dest, entry.Name())
		if !entry.IsDir() {
			if entry.Name() == "package.json" {
				if err := copyFile(src, dst); err != nil {
					return err
				}
			} else if err := overlayWriteFile(root, dst, nil, 0o644); err != nil {
				return err
			}
			continue
		}
		if entry.Name() == "scripts" {
			if err := copyDirectoryFiles(src, dst); err != nil {
				return err
			}
			continue
		}
		if err := overlaySymlink(root, src, dst); err != nil {
			return err
		}
	}
	return nil
}
