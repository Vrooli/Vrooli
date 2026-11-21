package installgateway

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// pinPnpmTarballIntegrity gives every tarball URL dependency in the staged lock
// an integrity. pnpm records one when it adds the URL, but `pnpm remove` and
// older pnpm versions leave an entry with only `tarball:`, which pnpm then
// refuses to install. The URL is downloaded once (bounded, in memory), its
// sha512 is written into the staged lock, and the reviewer is told which bytes
// the lock now identifies. The same download is the disk-admission sizing.
func pinPnpmTarballIntegrity(ctx context.Context, s *JSStage, r Resolution) error {
	lock := s.next["pnpm-lock.yaml"]
	if lock == nil {
		lock = s.base["pnpm-lock.yaml"]
	}
	packages, err := readPnpmLockPackages(lock)
	if err != nil {
		return err
	}
	var keys []string
	for key, pkg := range packages {
		if _, spec, ok := splitPnpmPackageKey(key); ok && strings.HasPrefix(spec, "https://") && pkg.Resolution.Integrity == "" && pkg.Resolution.Tarball != "" {
			keys = append(keys, key)
		}
	}
	if len(keys) == 0 {
		return nil
	}
	sort.Strings(keys)
	sizer, err := newRegistrySizer()
	if err != nil {
		return err
	}
	for _, key := range keys {
		url := packages[key].Resolution.Tarball
		size, err := sizer.Tarball(ctx, url, "")
		if err != nil {
			return fmt.Errorf("pin the integrity of %s: %w", key, err)
		}
		if size.Integrity == "" {
			return fmt.Errorf("pin the integrity of %s: the download produced no digest", key)
		}
		pinned, err := setPnpmLockIntegrity(lock, key, size.Integrity)
		if err != nil {
			return err
		}
		lock = pinned
		if s.tarballSizes == nil {
			s.tarballSizes = map[string]artifactSize{}
		}
		s.tarballSizes[url] = size
		s.Notes = append(s.Notes, fmt.Sprintf("tarball dependency %s: the lock recorded no integrity, so %s was downloaded once and pinned to %s", key, url, size.Integrity))
	}
	s.next["pnpm-lock.yaml"] = lock
	return nil
}

// setPnpmLockIntegrity adds `integrity` to the flow-style resolution of one lock
// package by editing its text, so the rest of the lockfile keeps pnpm's exact
// formatting and a reviewer sees a one-line diff.
func setPnpmLockIntegrity(lock []byte, key, integrity string) ([]byte, error) {
	lines := strings.Split(string(lock), "\n")
	heads := []string{"  " + key + ":", "  '" + key + "':", "  \"" + key + "\":"}
	for i, line := range lines {
		matched := false
		for _, head := range heads {
			if line == head {
				matched = true
			}
		}
		if !matched {
			continue
		}
		for j := i + 1; j < len(lines) && strings.HasPrefix(lines[j], "    "); j++ {
			if resolution, ok := strings.CutPrefix(lines[j], "    resolution: {"); ok && !strings.Contains(resolution, "integrity:") {
				lines[j] = "    resolution: {integrity: " + integrity + ", " + resolution
				pinned := []byte(strings.Join(lines, "\n"))
				if packages, err := readPnpmLockPackages(pinned); err != nil || packages[key].Resolution.Integrity != integrity {
					return nil, fmt.Errorf("pinning %s produced a lockfile pnpm would read differently", key)
				}
				return pinned, nil
			}
		}
	}
	return nil, fmt.Errorf("pnpm lock package %s has no flow-style resolution to pin", key)
}

func encodePnpmLock(doc *yaml.Node) ([]byte, error) {
	var out bytes.Buffer
	encoder := yaml.NewEncoder(&out)
	encoder.SetIndent(2)
	if err := encoder.Encode(doc); err != nil {
		return nil, err
	}
	return out.Bytes(), encoder.Close()
}
