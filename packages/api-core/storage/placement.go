package storage

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

// Rung describes how an entry may be managed by a deployment.
type Rung string

const (
	RungOwned       Rung = "owned"
	RungRelocatable Rung = "relocatable"
	RungPinned      Rung = "pinned"
)

type PlacementRequest struct {
	Entry    string
	Path     PortablePath
	Platform Platform
	Profile  Profile
	Root     string
}
type Placement struct {
	Entry        string   `json:"entry"`
	AbsolutePath string   `json:"absolute_path"`
	Platform     Platform `json:"platform"`
	Profile      Profile  `json:"profile"`
	Applicable   bool     `json:"applicable"`
}

// ResolvePlacement computes a concrete location and never changes the host.
func ResolvePlacement(req PlacementRequest, seams PlatformSeams) (Placement, error) {
	path, err := ResolvePortablePath(req.Entry, req.Path, req.Platform, seams)
	if err != nil {
		return Placement{Entry: req.Entry, Platform: req.Platform, Profile: req.Profile}, err
	}
	if req.Root != "" {
		if !filepath.IsAbs(req.Root) {
			return Placement{}, fmt.Errorf("placement root must be absolute")
		}
		if !filepath.IsAbs(path) {
			path = filepath.Join(req.Root, path)
		}
	}
	return Placement{Entry: req.Entry, AbsolutePath: filepath.Clean(path), Platform: req.Platform, Profile: req.Profile, Applicable: true}, nil
}

type MigrationResult struct {
	PlanID          string `json:"plan_id"`
	Source          string `json:"source"`
	Destination     string `json:"destination"`
	Bytes           int64  `json:"bytes"`
	Verified        bool   `json:"verified"`
	SourcePreserved bool   `json:"source_preserved"`
}

// Migrate moves one declared tree only after the caller has approved its plan.
// It copies, verifies, and only then removes the source. A failed copy or
// verification leaves the source intact and removes the incomplete destination.
func Migrate(planID, source, destination string, approved bool) error {
	_, err := MigrateVerified(planID, source, destination, approved)
	return err
}

func MigrateVerified(planID, source, destination string, approved bool) (MigrationResult, error) {
	result := MigrationResult{PlanID: planID, Source: source, Destination: destination, SourcePreserved: true}
	if planID == "" || !approved {
		return result, fmt.Errorf("placement migration requires an approved plan id")
	}
	if source == "" || destination == "" || !filepath.IsAbs(source) || !filepath.IsAbs(destination) {
		return result, fmt.Errorf("placement migration requires absolute source and destination")
	}
	if source == destination {
		result.Verified = true
		result.SourcePreserved = true
		return result, nil
	}
	if _, err := os.Stat(source); err != nil {
		return result, fmt.Errorf("stat migration source: %w", err)
	}
	if _, err := os.Stat(destination); err == nil {
		return result, fmt.Errorf("refusing to overwrite existing destination %q", destination)
	} else if !os.IsNotExist(err) {
		return result, fmt.Errorf("stat migration destination: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return result, fmt.Errorf("prepare migration destination: %w", err)
	}
	if err := copyTree(source, destination); err != nil {
		_ = os.RemoveAll(destination)
		return result, fmt.Errorf("copy migration: %w", err)
	}
	if err := verifyTree(source, destination); err != nil {
		_ = os.RemoveAll(destination)
		return result, fmt.Errorf("verify migration: %w", err)
	}
	bytes, err := treeBytes(destination)
	if err != nil {
		_ = os.RemoveAll(destination)
		return result, err
	}
	if err := os.RemoveAll(source); err != nil {
		_ = os.RemoveAll(destination)
		return result, fmt.Errorf("remove migration source: %w", err)
	}
	result.Bytes, result.Verified, result.SourcePreserved = bytes, true, false
	return result, nil
}

func copyTree(source, destination string) error {
	info, err := os.Lstat(source)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("symlink source is not supported")
	}
	if info.IsDir() {
		if err := os.Mkdir(destination, info.Mode().Perm()); err != nil {
			return err
		}
		entries, err := os.ReadDir(source)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if err := copyTree(filepath.Join(source, entry.Name()), filepath.Join(destination, entry.Name())); err != nil {
				return err
			}
		}
		return nil
	}
	return copyFile(source, destination, info.Mode().Perm())
}

func copyFile(source, destination string, mode os.FileMode) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	if err := out.Sync(); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

func verifyTree(source, destination string) error {
	left, err := treeDigest(source)
	if err != nil {
		return err
	}
	right, err := treeDigest(destination)
	if err != nil {
		return err
	}
	if left != right {
		return fmt.Errorf("source and destination digests differ")
	}
	return nil
}

func treeBytes(root string) (int64, error) {
	var total int64
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			total += info.Size()
		}
		return nil
	})
	return total, err
}

func treeDigest(root string) ([32]byte, error) {
	h := sha256.New()
	paths := make([]string, 0)
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		paths = append(paths, path)
		return nil
	})
	if err != nil {
		return [32]byte{}, err
	}
	sort.Strings(paths)
	for _, path := range paths {
		info, err := os.Lstat(path)
		if err != nil {
			return [32]byte{}, err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return [32]byte{}, err
		}
		_, _ = io.WriteString(h, rel+"\x00")
		if info.IsDir() {
			continue
		}
		file, err := os.Open(path)
		if err != nil {
			return [32]byte{}, err
		}
		if _, err := io.Copy(h, file); err != nil {
			_ = file.Close()
			return [32]byte{}, err
		}
		if err := file.Close(); err != nil {
			return [32]byte{}, err
		}
	}
	var digest [32]byte
	copy(digest[:], h.Sum(nil))
	return digest, nil
}
