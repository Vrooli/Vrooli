package sketch

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vrooli/api-core/storage"
	"github.com/vrooli/platform-go"
)

// CandidateRepository owns immutable alternatives independently of whether the
// current page has selected them. Candidate pages use the same authored shape.
type CandidateRepository interface {
	SaveCandidate(scenario, page, designID, expectedHash string, doc Document) (Candidate, error)
	ReadCandidate(scenario, designID, hash string) (Candidate, error)
}
type Candidate struct {
	Refinement    *CandidateRefinement `json:"refinement,omitempty"`
	ParentHash    string               `json:"parentHash,omitempty"`
	SchemaVersion int                  `json:"schemaVersion"`
	DesignID      string               `json:"designId"`
	Page          string               `json:"page"`
	BaseHash      string               `json:"baseHash"`
	PageBytes     []byte               `json:"pageBytes"`
	Hash          string               `json:"hash"`
}

func candidateHash(c Candidate) (string, error) {
	c.Hash = ""
	// Canonical page bytes make formatting irrelevant to candidate identity.
	var value any
	decoder := json.NewDecoder(bytes.NewReader(c.PageBytes))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return "", err
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	c.PageBytes = raw
	raw, err = json.Marshal(c)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:]), nil
}
func candidatePath(designID, hash string) (string, error) {
	if !safeSegment.MatchString(designID) || !validHash.MatchString(hash) {
		return "", fmt.Errorf("invalid candidate identity")
	}
	return filepath.Join("designs", designID, "candidates", hash+".json"), nil
}
func (c Candidate) Snapshot() (Snapshot, error) {
	s, err := decodeSnapshot(c.PageBytes)
	if err == nil {
		s.ContentHash = c.Hash
	}
	return s, err
}
func readCandidate(root *os.Root, designID, hash string) (Candidate, error) {
	path, err := candidatePath(designID, hash)
	if err != nil {
		return Candidate{}, err
	}
	raw, err := root.ReadFile(path)
	if err != nil {
		return Candidate{}, err
	}
	if err := rejectDuplicateKeys(raw); err != nil {
		return Candidate{}, err
	}
	var c Candidate
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&c); err != nil {
		return c, err
	}
	if (c.ParentHash != "" && !validHash.MatchString(c.ParentHash)) || c.SchemaVersion != 1 || c.DesignID != designID || c.Hash != hash || !safeSegment.MatchString(c.Page) || !validHash.MatchString(c.BaseHash) {
		return Candidate{}, fmt.Errorf("invalid candidate envelope")
	}
	if _, err := decodeSnapshot(c.PageBytes); err != nil {
		return Candidate{}, err
	}
	if c.Refinement != nil {
		if err := validateRefinement(*c.Refinement); err != nil {
			return c, err
		}
	}
	actual, err := candidateHash(c)
	if err != nil {
		return Candidate{}, err
	}
	if actual != hash {
		return Candidate{}, fmt.Errorf("immutable candidate bytes do not match their identity")
	}
	return c, nil
}
func (s *Store) ReadCandidate(scenario, designID, hash string) (Candidate, error) {
	root, err := s.openExperience(scenario)
	if err != nil {
		return Candidate{}, err
	}
	defer root.Close()
	return readCandidate(root, designID, hash)
}
func (s *Store) SaveCandidate(scenario, page, designID, expectedHash string, doc Document) (Candidate, error) {
	if !safeSegment.MatchString(designID) {
		return Candidate{}, fmt.Errorf("invalid design identity")
	}
	if expectedHash == "" {
		return Candidate{}, ErrExpectedRevision
	}
	root, name, err := s.openPages(scenario, page)
	if err != nil {
		return Candidate{}, err
	}
	defer root.Close()
	lock, err := root.OpenFile(filepath.Join("pages", "."+page+".lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return Candidate{}, err
	}
	defer lock.Close()
	release, err := platform.LockFile(lock, false)
	if err != nil {
		return Candidate{}, err
	}
	defer release()
	raw, err := root.ReadFile(name)
	if err != nil {
		return Candidate{}, err
	}
	snapshot, err := decodeSnapshot(raw)
	if err != nil {
		return Candidate{}, err
	}
	if snapshot.ContentHash != expectedHash {
		return Candidate{}, &ConflictError{Expected: expectedHash, Current: snapshot.ContentHash}
	}
	updated, err := pageWithSketch(raw, doc)
	if err != nil {
		return Candidate{}, err
	}
	c := Candidate{SchemaVersion: 1, DesignID: designID, Page: page, BaseHash: expectedHash, PageBytes: updated}
	return publishCandidate(root, c)
}
func publishCandidate(root *os.Root, c Candidate) (Candidate, error) {
	var err error
	c.Hash, err = candidateHash(c)
	if err != nil {
		return Candidate{}, err
	}
	if existing, err := readCandidate(root, c.DesignID, c.Hash); err == nil {
		return existing, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return Candidate{}, err
	}
	path, err := candidatePath(c.DesignID, c.Hash)
	if err != nil {
		return Candidate{}, err
	}
	encoded, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return Candidate{}, err
	}
	if err := storage.WriteFileAtomicInRoot(root, path, encoded, 0600); err != nil {
		return Candidate{}, err
	}
	return c, nil
}

// DeriveCandidate branches immutable authored bytes, independent of current
// page edits. Selection still checks BaseHash separately before publication.
func (s *Store) DeriveCandidate(scenario, designID, parentHash string, doc Document) (Candidate, error) {
	return s.deriveCandidate(scenario, designID, parentHash, doc, nil)
}
func (s *Store) deriveCandidate(scenario, designID, parentHash string, doc Document, refinement *CandidateRefinement) (Candidate, error) {
	root, err := s.openExperience(scenario)
	if err != nil {
		return Candidate{}, err
	}
	defer root.Close()
	parent, err := readCandidate(root, designID, parentHash)
	if err != nil {
		return Candidate{}, err
	}
	lock, err := root.OpenFile(filepath.Join("pages", "."+parent.Page+".lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return Candidate{}, err
	}
	defer lock.Close()
	release, err := platform.LockFile(lock, false)
	if err != nil {
		return Candidate{}, err
	}
	defer release()
	updated, err := pageWithSketch(parent.PageBytes, doc)
	if err != nil {
		return Candidate{}, err
	}
	if refinement == nil {
		refinement = parent.Refinement
	}
	c := Candidate{Refinement: refinement, SchemaVersion: 1, DesignID: designID, Page: parent.Page, BaseHash: parent.BaseHash, ParentHash: parentHash, PageBytes: updated}
	return publishCandidate(root, c)
}

// ListCandidates verifies archived identities before returning page alternatives.
// The bounded scan fails explicitly rather than silently hiding saved designs.
func (s *Store) ListCandidates(scenario, page string) ([]Candidate, error) {
	if !safeSegment.MatchString(page) {
		return nil, fmt.Errorf("invalid page identity")
	}
	root, err := s.openExperience(scenario)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	dir, err := root.Open("designs")
	if errors.Is(err, os.ErrNotExist) {
		return []Candidate{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer dir.Close()
	designs, err := dir.ReadDir(4097)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	if len(designs) > 4096 {
		return nil, fmt.Errorf("candidate inventory exceeds 4096 designs")
	}
	result := []Candidate{}
	scanned := 0
	for _, design := range designs {
		if !design.IsDir() || !safeSegment.MatchString(design.Name()) {
			continue
		}
		files, err := root.Open(filepath.Join("designs", design.Name(), "candidates"))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		entries, readErr := files.ReadDir(4097 - scanned)
		files.Close()
		if readErr != nil && !errors.Is(readErr, io.EOF) {
			return nil, readErr
		}
		scanned += len(entries)
		if scanned > 4096 {
			return nil, fmt.Errorf("candidate inventory exceeds 4096 revisions")
		}
		for _, entry := range entries {
			if !strings.HasSuffix(entry.Name(), ".json") {
				continue
			}
			hash := strings.TrimSuffix(entry.Name(), ".json")
			c, err := readCandidate(root, design.Name(), hash)
			if err != nil {
				return nil, fmt.Errorf("candidate %s/%s: %w", design.Name(), hash, err)
			}
			if c.Page == page {
				result = append(result, c)
			}
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].DesignID != result[j].DesignID {
			return result[i].DesignID < result[j].DesignID
		}
		return result[i].Hash < result[j].Hash
	})
	return result, nil
}
