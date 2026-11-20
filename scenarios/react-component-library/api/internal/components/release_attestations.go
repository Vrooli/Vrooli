package components

import (
	"encoding/json"
	"errors"
	"io/fs"
	"path"
	"strings"
)

// Why released bytes have two records, and which one wins.
//
// A released version's bytes are attested twice: by `content_sha256` in the
// index, and by `library/released-version-hashes.json` in the working tree. The
// index is a projection that a reindex rebuilds; the registry is a tracked
// artifact that changes only in a commit, beside the source it describes.
//
// When a migration rewrites released source it updates the registry in the same
// commit — that is what makes the rewrite reviewable. It cannot update the
// index of every machine that has one. So the two disagree for exactly as long
// as those machines have not reindexed, and the index is the stale side by
// construction.
//
// Treating the index as authoritative in that window is what wedged this
// scenario: 95 of 236 assets failed to index with "recorded hash differs from
// incoming hash", each failure skipped the asset, and the resulting partial
// index made retention blind to the dependencies those assets declared.
//
// The registry therefore outranks the index for released bytes. That is not a
// weakening of the immutability rule: bytes matching neither record are still
// refused, and the registry is the record a reviewer actually reads.

// releaseAttestationRegistry is the tracked record of released version bytes.
const releaseAttestationRegistry = "released-version-hashes.json"

// releaseRepairRegistry records a reviewed correction when a later commit
// edited released bytes without republishing them. It never replaces the
// published digest: a repair is accepted only when it names the digest the main
// registry published, so a repair for one publication cannot vouch for another.
const releaseRepairRegistry = "release-repairs.json"

type releaseAttestationFile struct {
	SchemaVersion int `json:"schemaVersion"`
	Entries       []struct {
		Path   string `json:"path"`
		SHA256 string `json:"sha256"`
	} `json:"entries"`
}

type releaseRepairFile struct {
	SchemaVersion int `json:"schemaVersion"`
	Repairs       []struct {
		Path            string `json:"path"`
		PublishedSHA256 string `json:"publishedSha256"`
		RepairedSHA256  string `json:"repairedSha256"`
		Reason          string `json:"reason"`
	} `json:"repairs"`
}

// loadReleaseAttestations reads the committed hash registry from the library
// tree, keyed by library-root-relative slash path.
//
// A missing registry is a legitimate state — lightweight fixtures carry no
// registry — and yields an empty set, which reconciles nothing and leaves the
// index the sole authority exactly as before.
func loadReleaseAttestations(fsys fs.FS) (map[string]string, error) {
	raw, err := fs.ReadFile(fsys, releaseAttestationRegistry)
	if errors.Is(err, fs.ErrNotExist) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, err
	}
	var registry releaseAttestationFile
	if err := json.Unmarshal(raw, &registry); err != nil {
		return nil, ErrInvalidHeader{SourcePath: releaseAttestationRegistry, Field: "entries", Reason: err.Error()}
	}
	out := make(map[string]string, len(registry.Entries))
	for _, entry := range registry.Entries {
		cleaned := strings.TrimSpace(entry.Path)
		digest := strings.TrimSpace(entry.SHA256)
		if cleaned == "" || digest == "" {
			continue
		}
		out[path.Clean(cleaned)] = digest
	}
	repairs, err := loadReleaseRepairs(fsys, out)
	if err != nil {
		return nil, err
	}
	for repairedPath, digest := range repairs {
		out[repairedPath] = digest
	}
	return out, nil
}

func loadReleaseRepairs(fsys fs.FS, published map[string]string) (map[string]string, error) {
	raw, err := fs.ReadFile(fsys, releaseRepairRegistry)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var registry releaseRepairFile
	if err := json.Unmarshal(raw, &registry); err != nil {
		return nil, ErrInvalidHeader{SourcePath: releaseRepairRegistry, Field: "repairs", Reason: err.Error()}
	}
	out := make(map[string]string, len(registry.Repairs))
	for _, repair := range registry.Repairs {
		cleaned := path.Clean(strings.TrimSpace(repair.Path))
		publishedDigest := strings.TrimSpace(repair.PublishedSHA256)
		repairedDigest := strings.TrimSpace(repair.RepairedSHA256)
		reason := strings.TrimSpace(repair.Reason)
		switch {
		case cleaned == "" || publishedDigest == "" || repairedDigest == "":
			return nil, ErrInvalidHeader{SourcePath: releaseRepairRegistry, Field: "path", Reason: "path, publishedSha256 and repairedSha256 are required"}
		case reason == "":
			return nil, ErrInvalidHeader{SourcePath: releaseRepairRegistry, Field: "reason", Reason: "a repair must state why the published bytes changed"}
		case published[cleaned] != publishedDigest:
			return nil, ErrInvalidHeader{SourcePath: releaseRepairRegistry, Field: "publishedSha256", Reason: "repair does not match the published digest for " + cleaned}
		}
		out[cleaned] = repairedDigest
	}
	return out, nil
}

// attestsRelease reports whether the committed registry vouches for the bytes
// the indexer just read at this path.
func attestsRelease(attestations map[string]string, sourcePath, incoming string) bool {
	if len(attestations) == 0 || sourcePath == "" || incoming == "" {
		return false
	}
	return attestations[path.Clean(sourcePath)] == incoming
}
