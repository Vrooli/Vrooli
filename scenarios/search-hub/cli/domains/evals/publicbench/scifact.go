// Package publicbench adapts reproducible, public benchmark data to Search
// Hub's existing evaluation contracts. It never downloads a dataset: callers
// stage a reviewed upstream release and pass its directory explicitly.
package publicbench

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
)

const (
	SciFactDatasetID       = "scifact"
	SciFactDevSplit        = "dev"
	SciFactSuiteID         = "public.scifact.dev.v1"
	SciFactSourceURL       = "https://github.com/allenai/scifact"
	SciFactReleaseURL      = "https://scifact.s3-us-west-2.amazonaws.com/release/latest/data.tar.gz"
	SciFactDataSpecURL     = "https://github.com/allenai/scifact/blob/master/doc/data.md"
	SciFactLicenseURL      = "https://github.com/allenai/scifact/blob/master/LICENSE.md"
	SciFactClaimsLicense   = "CC-BY-4.0"
	SciFactAbstractLicense = "ODC-By-1.0"
	SciFactPaperURL        = "https://arxiv.org/abs/2004.14974"

	maxSciFactFileBytes = 64 << 20
	maxSciFactLineBytes = 2 << 20
	maxSciFactCases     = 300
)

// Manifest names the independently authored dataset, its license boundary,
// and the exact official split used. The official SciFact test labels are not
// public; this adapter therefore accepts only claims_dev.jsonl.
type Manifest struct {
	DatasetID               string `json:"dataset_id"`
	SourceURL               string `json:"source_url"`
	DatasetArtifactURL      string `json:"dataset_artifact_url"`
	DataSpecificationURL    string `json:"data_specification_url"`
	LicenseURL              string `json:"license_url"`
	ClaimsLicense           string `json:"claims_license"`
	AbstractsLicense        string `json:"abstracts_license"`
	CitationURL             string `json:"citation_url"`
	Split                   string `json:"split"`
	SplitFile               string `json:"split_file"`
	LabelSource             string `json:"label_source"`
	SubsetPolicy            string `json:"subset_policy"`
	ExpectedLabelProvenance string `json:"expected_label_provenance"`
}

// SciFactManifest is the versioned source contract for the public subset.
var SciFactManifest = Manifest{
	DatasetID:               SciFactDatasetID,
	SourceURL:               SciFactSourceURL,
	DatasetArtifactURL:      SciFactReleaseURL,
	DataSpecificationURL:    SciFactDataSpecURL,
	LicenseURL:              SciFactLicenseURL,
	ClaimsLicense:           SciFactClaimsLicense,
	AbstractsLicense:        SciFactAbstractLicense,
	CitationURL:             SciFactPaperURL,
	Split:                   SciFactDevSplit,
	SplitFile:               "claims_dev.jsonl",
	LabelSource:             "claims_dev.jsonl evidence annotations; do not substitute the unlabeled official test split or BEIR's renamed split",
	SubsetPolicy:            "sort eligible dev records by numeric claim id, then choose the first maxCases records with at least one evidence document; maxCases is explicit and capped at 300",
	ExpectedLabelProvenance: "claim record evidence[doc_id][].label; relevance IDs are exactly the evidence map keys",
}

// EvidenceLabel retains the upstream per-document annotation. A claim can
// have several evidence documents, and labels remain attached to each one.
type EvidenceLabel struct {
	DocumentID string   `json:"document_id"`
	Labels     []string `json:"labels"`
}

// Case is one source-linked benchmark query with the source's gold evidence
// document IDs and labels. It is built only from official dev annotations.
type Case struct {
	CaseID         string          `json:"case_id"`
	ClaimID        int64           `json:"claim_id"`
	Query          string          `json:"query"`
	Split          string          `json:"split"`
	ExpectedIDs    []string        `json:"expected_ids"`
	EvidenceLabels []EvidenceLabel `json:"evidence_labels"`
	LabelSource    string          `json:"label_source"`
	SourceURL      string          `json:"source_url"`
	LicenseID      string          `json:"license_id"`
}

// Document is a source corpus item needed by a retrieval evaluator. The
// adapter returns the full official corpus rather than only gold abstracts,
// preserving the benchmark's distractor distribution.
type Document struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Text  string `json:"text"`
}

// Dataset is a bounded, content-addressed subset ready for existing Search
// Hub graders. CorpusSHA256 and LabelSHA256 bind their exact source files;
// DatasetSHA256 binds both together. No source revision is guessed from a
// moving branch name.
type Dataset struct {
	Manifest        Manifest   `json:"manifest"`
	CorpusSHA256    string     `json:"corpus_sha256"`
	LabelSHA256     string     `json:"label_sha256"`
	DatasetSHA256   string     `json:"dataset_sha256"`
	SelectedCaseIDs []string   `json:"selected_case_ids"`
	Cases           []Case     `json:"cases"`
	Corpus          []Document `json:"corpus"`
	SkippedNoGold   int        `json:"skipped_no_gold_evidence"`
}

type rawClaim struct {
	ID       int64                    `json:"id"`
	Claim    string                   `json:"claim"`
	Evidence map[string][]rawEvidence `json:"evidence"`
}

type rawEvidence struct {
	Label     string `json:"label"`
	Sentences []int  `json:"sentences"`
}

type rawDocument struct {
	DocumentID json.Number `json:"doc_id"`
	Title      string      `json:"title"`
	Abstract   []string    `json:"abstract"`
}

// LoadSciFactDev reads an already staged official SciFact release. maxCases
// must be 1..300, making source size and evaluation workload explicit. The
// split and file names are fixed here to prevent accidentally evaluating
// training data or relying on labels withheld for the official test split.
func LoadSciFactDev(root string, maxCases int) (Dataset, error) {
	if strings.TrimSpace(root) == "" {
		return Dataset{}, fmt.Errorf("SciFact data directory is required")
	}
	if maxCases < 1 || maxCases > maxSciFactCases {
		return Dataset{}, fmt.Errorf("maxCases must be between 1 and %d", maxSciFactCases)
	}
	claimsPath := filepath.Join(root, "claims_dev.jsonl")
	corpusPath := filepath.Join(root, "corpus.jsonl")
	claimsBytes, err := readBoundedFile(claimsPath)
	if err != nil {
		return Dataset{}, fmt.Errorf("read official SciFact dev claims: %w", err)
	}
	corpusBytes, err := readBoundedFile(corpusPath)
	if err != nil {
		return Dataset{}, fmt.Errorf("read official SciFact corpus: %w", err)
	}

	corpus, err := parseCorpus(corpusBytes)
	if err != nil {
		return Dataset{}, fmt.Errorf("parse official SciFact corpus: %w", err)
	}
	corpusIDs := make(map[string]struct{}, len(corpus))
	for _, doc := range corpus {
		corpusIDs[doc.ID] = struct{}{}
	}
	claims, err := parseClaims(claimsBytes)
	if err != nil {
		return Dataset{}, fmt.Errorf("parse official SciFact dev split: %w", err)
	}
	sort.Slice(claims, func(i, j int) bool { return claims[i].ID < claims[j].ID })

	out := Dataset{
		Manifest: SciFactManifest,
		Corpus:   corpus,
		Cases:    make([]Case, 0, maxCases),
	}
	for _, claim := range claims {
		if len(claim.Evidence) == 0 {
			out.SkippedNoGold++
		}
	}
	for _, claim := range claims {
		if len(out.Cases) >= maxCases {
			break
		}
		if claim.ID < 0 || strings.TrimSpace(claim.Claim) == "" {
			return Dataset{}, fmt.Errorf("claim record has invalid id or empty text")
		}
		if len(claim.Evidence) == 0 {
			continue
		}
		ids := make([]string, 0, len(claim.Evidence))
		labels := make([]EvidenceLabel, 0, len(claim.Evidence))
		for docID, annotations := range claim.Evidence {
			docID = strings.TrimSpace(docID)
			if docID == "" || len(annotations) == 0 {
				return Dataset{}, fmt.Errorf("claim %d has incomplete evidence annotation", claim.ID)
			}
			if _, found := corpusIDs[docID]; !found {
				return Dataset{}, fmt.Errorf("claim %d labels missing corpus document %q", claim.ID, docID)
			}
			perDoc := make([]string, 0, len(annotations))
			for _, annotation := range annotations {
				label := strings.ToUpper(strings.TrimSpace(annotation.Label))
				if label != "SUPPORT" && label != "CONTRADICT" {
					return Dataset{}, fmt.Errorf("claim %d document %s has invalid evidence label %q", claim.ID, docID, annotation.Label)
				}
				perDoc = append(perDoc, label)
			}
			sort.Strings(perDoc)
			ids = append(ids, docID)
			labels = append(labels, EvidenceLabel{DocumentID: docID, Labels: perDoc})
		}
		sort.Strings(ids)
		sort.Slice(labels, func(i, j int) bool { return labels[i].DocumentID < labels[j].DocumentID })
		out.Cases = append(out.Cases, Case{
			CaseID:         fmt.Sprintf("scifact/dev/%d", claim.ID),
			ClaimID:        claim.ID,
			Query:          strings.TrimSpace(claim.Claim),
			Split:          SciFactDevSplit,
			ExpectedIDs:    ids,
			EvidenceLabels: labels,
			LabelSource:    formatEvidenceLabelSource(claim.ID, labels),
			SourceURL:      SciFactReleaseURL,
			LicenseID:      SciFactClaimsLicense,
		})
	}
	if len(out.Cases) == 0 {
		return Dataset{}, fmt.Errorf("official SciFact dev split yielded no evidence-labeled cases")
	}
	out.SelectedCaseIDs = make([]string, len(out.Cases))
	for i := range out.Cases {
		out.SelectedCaseIDs[i] = out.Cases[i].CaseID
	}
	out.CorpusSHA256 = contentDigest(corpusBytes)
	out.LabelSHA256 = contentDigest(claimsBytes)
	out.DatasetSHA256 = datasetDigest(claimsBytes, corpusBytes)
	return out, nil
}

// SearchSuite converts only the public evidence retrieval cases to the shared
// Search Hub suite shape. Provider registration and the independent fixed
// denominator receipt remain with their existing owners.
func (d Dataset) SearchSuite(providerID string) (*evalv1.EvalSuite, error) {
	providerID = strings.TrimSpace(providerID)
	if providerID == "" {
		return nil, fmt.Errorf("public benchmark provider id is required")
	}
	if len(d.Cases) == 0 || !isSHA256(d.CorpusSHA256) || !isSHA256(d.LabelSHA256) || !isSHA256(d.DatasetSHA256) {
		return nil, fmt.Errorf("public benchmark dataset identity is incomplete")
	}
	suiteID := fmt.Sprintf("%s.n%d.%s", SciFactSuiteID, len(d.Cases), d.DatasetSHA256[:12])
	suite := &evalv1.EvalSuite{
		SuiteId:     suiteID,
		ProviderId:  providerID,
		Name:        fmt.Sprintf("SciFact official dev retrieval subset (%d cases)", len(d.Cases)),
		Description: fmt.Sprintf("Source-linked SciFact %s subset (%d evidence-labeled claims; %d dev records without evidence excluded from retrieval), corpus_sha256=%s, label_sha256=%s, dataset_sha256=%s; claims/evidence CC BY 4.0 and abstracts ODC-By 1.0. %s", SciFactDevSplit, len(d.Cases), d.SkippedNoGold, d.CorpusSHA256, d.LabelSHA256, d.DatasetSHA256, SciFactLicenseURL),
		State:       "active",
		Cases:       make([]*evalv1.EvalCase, 0, len(d.Cases)),
	}
	for _, c := range d.Cases {
		note := fmt.Sprintf("source=%s; schema=%s; split=%s; claim_id=%d; expected_ids_and_labels=%s; claims_license=%s; corpus_license=%s; license_source=%s; corpus_sha256=%s; label_sha256=%s; dataset_sha256=%s", c.SourceURL, SciFactDataSpecURL, c.Split, c.ClaimID, c.LabelSource, SciFactClaimsLicense, SciFactAbstractLicense, SciFactLicenseURL, d.CorpusSHA256, d.LabelSHA256, d.DatasetSHA256)
		suite.Cases = append(suite.Cases, &evalv1.EvalCase{
			CaseId:           c.CaseID,
			Query:            c.Query,
			Tags:             []string{"public", "scifact", "scifact-dev"},
			ExpectIds:        append([]string(nil), c.ExpectedIDs...),
			ExpectWithinTopK: 10,
			Status:           "reviewed",
			Note:             note,
		})
	}
	return suite, nil
}

// SearchProviderDescriptor binds the suite to Search Hub's staged-corpus
// lexical BM25 provider. Its API module reads corpus.jsonl from
// SEARCH_HUB_SCIFACT_DATA_DIR; provider and suite therefore share one source.
func SearchProviderDescriptor(providerID, corpusSHA256 string) (*registryv1.ProviderDescriptor, error) {
	providerID = strings.TrimSpace(providerID)
	if providerID == "" {
		return nil, fmt.Errorf("provider ID is required")
	}
	corpusSHA256 = strings.TrimSpace(corpusSHA256)
	if len(corpusSHA256) != 64 {
		return nil, fmt.Errorf("expected corpus SHA-256 is required")
	}
	if _, err := hex.DecodeString(corpusSHA256); err != nil {
		return nil, fmt.Errorf("expected corpus SHA-256 is invalid: %w", err)
	}
	return &registryv1.ProviderDescriptor{
		ProviderId:    providerID,
		ProviderGroup: "search-hub-publicbench",
		Bucket:        registryv1.Bucket_BUCKET_KNOW,
		Type:          "doc",
		Description:   "Official SciFact full abstract corpus staged at SEARCH_HUB_SCIFACT_DATA_DIR; lexical BM25 retrieval, no model inference; expected corpus_sha256=" + corpusSHA256,
		Endpoint: &registryv1.Endpoint{Kind: &registryv1.Endpoint_HttpJson{HttpJson: &registryv1.HttpJsonEndpoint{
			ScenarioId: "search-hub", Path: "/api/v1/publicbench/scifact/search", Method: registryv1.HttpMethod_HTTP_METHOD_POST,
			BodyTemplate: `{"query":"{{query}}","limit":{{limit}},"expected_corpus_sha256":"` + corpusSHA256 + `"}`, Headers: map[string]string{"Content-Type": "application/json"},
		}}},
		ResultMapping: &registryv1.ResultMapping{ResultsPath: "results", IdField: "id", TitleField: "title", ScoreField: "score", SnippetField: "snippet", PathField: "path", ScoreScale: registryv1.ScoreScale_SCORE_SCALE_RAW},
		Scope:         registryv1.Scope_SCOPE_PROJECT,
		State:         registryv1.ProviderState_PROVIDER_STATE_ACTIVE,
		Lifecycle:     registryv1.Lifecycle_LIFECYCLE_EXPERIMENTAL,
	}, nil
}

func formatEvidenceLabelSource(claimID int64, labels []EvidenceLabel) string {
	parts := make([]string, 0, len(labels))
	for _, label := range labels {
		parts = append(parts, label.DocumentID+":"+strings.Join(label.Labels, "+"))
	}
	return fmt.Sprintf("claims_dev.jsonl record id=%d; field evidence[doc_id][].label; %s", claimID, strings.Join(parts, ","))
}

func parseClaims(data []byte) ([]rawClaim, error) {
	var claims []rawClaim
	seen := map[int64]struct{}{}
	err := scanJSONL(data, func(line int, raw []byte) error {
		var claim rawClaim
		if err := json.Unmarshal(raw, &claim); err != nil {
			return fmt.Errorf("line %d: %w", line, err)
		}
		if _, duplicate := seen[claim.ID]; duplicate {
			return fmt.Errorf("line %d: duplicate claim id %d", line, claim.ID)
		}
		seen[claim.ID] = struct{}{}
		claims = append(claims, claim)
		return nil
	})
	return claims, err
}

func parseCorpus(data []byte) ([]Document, error) {
	var corpus []Document
	seen := map[string]struct{}{}
	err := scanJSONL(data, func(line int, raw []byte) error {
		var source rawDocument
		dec := json.NewDecoder(strings.NewReader(string(raw)))
		dec.UseNumber()
		if err := dec.Decode(&source); err != nil {
			return fmt.Errorf("line %d: %w", line, err)
		}
		docID := strings.TrimSpace(source.DocumentID.String())
		if docID == "" || docID == "0" {
			return fmt.Errorf("line %d: document id is required", line)
		}
		if _, duplicate := seen[docID]; duplicate {
			return fmt.Errorf("line %d: duplicate document id %s", line, docID)
		}
		seen[docID] = struct{}{}
		text := strings.TrimSpace(strings.Join(source.Abstract, " "))
		if text == "" {
			return fmt.Errorf("line %d: document %s has empty abstract", line, docID)
		}
		corpus = append(corpus, Document{ID: docID, Title: strings.TrimSpace(source.Title), Text: text})
		return nil
	})
	sort.Slice(corpus, func(i, j int) bool { return numericThenLexical(corpus[i].ID, corpus[j].ID) })
	return corpus, err
}

func scanJSONL(data []byte, visit func(int, []byte) error) error {
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	scanner.Buffer(make([]byte, 64<<10), maxSciFactLineBytes)
	line := 0
	for scanner.Scan() {
		line++
		raw := strings.TrimSpace(scanner.Text())
		if raw == "" {
			continue
		}
		if err := visit(line, []byte(raw)); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan JSONL: %w", err)
	}
	return nil
}

func readBoundedFile(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxSciFactFileBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxSciFactFileBytes {
		return nil, fmt.Errorf("file exceeds %d bytes", maxSciFactFileBytes)
	}
	return data, nil
}

func datasetDigest(claims, corpus []byte) string {
	h := sha256.New()
	for _, part := range []struct {
		name string
		data []byte
	}{{"claims_dev.jsonl", claims}, {"corpus.jsonl", corpus}} {
		_, _ = io.WriteString(h, part.name+"\x00")
		_, _ = h.Write(part.data)
		_, _ = io.WriteString(h, "\x00")
	}
	return hex.EncodeToString(h.Sum(nil))
}

func contentDigest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func numericThenLexical(a, b string) bool {
	ai, aErr := strconv.ParseInt(a, 10, 64)
	bi, bErr := strconv.ParseInt(b, 10, 64)
	if aErr == nil && bErr == nil && ai != bi {
		return ai < bi
	}
	return a < b
}

func isSHA256(value string) bool {
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == sha256.Size
}
