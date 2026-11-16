// Package searcheval provides canonical identity helpers for Search Hub's
// immutable evaluation corpus and denominator contracts.
package searcheval

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
	"google.golang.org/protobuf/proto"
)

// SuiteCorpusDigest identifies the evaluation-relevant bytes of a Search Hub
// suite. Display metadata and store timestamps are excluded so a UI rename
// does not invalidate reviewed labels, while any query, expectation, scope,
// status, tag, or provider change does.
func SuiteCorpusDigest(suite *evalv1.EvalSuite) string {
	if suite == nil {
		return ""
	}
	canonical := proto.Clone(suite).(*evalv1.EvalSuite)
	canonical.Name = ""
	canonical.Description = ""
	canonical.CreatedAt = ""
	canonical.UpdatedAt = ""
	encoded, err := (proto.MarshalOptions{Deterministic: true}).Marshal(canonical)
	if err != nil {
		return ""
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:])
}

// CaseIDsDigest is the canonical order-independent digest for an immutable
// evaluation denominator. Duplicate IDs remain represented in the digest;
// callers must validate uniqueness before treating it as a denominator.
func CaseIDsDigest(ids []string) string {
	canonical := append([]string(nil), ids...)
	sort.Strings(canonical)
	digest := sha256.Sum256([]byte(strings.Join(canonical, "\n")))
	return hex.EncodeToString(digest[:])
}
