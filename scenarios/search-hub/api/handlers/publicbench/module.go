// Package publicbench serves staged public benchmark corpora through the same
// provider endpoint contract used by Search Hub's shared evaluator.
package publicbench

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/gorilla/mux"
	"search-hub/internal/module"
)

const endpointPath = "/api/v1/publicbench/scifact/search"

type document struct {
	ID       json.Number `json:"doc_id"`
	Title    string      `json:"title"`
	Abstract []string    `json:"abstract"`
}
type result struct {
	ID      string  `json:"id"`
	Title   string  `json:"title"`
	Score   float64 `json:"score"`
	Snippet string  `json:"snippet"`
	Path    string  `json:"path"`
}
type searchRequest struct {
	Query                string `json:"query"`
	Limit                int    `json:"limit"`
	ExpectedCorpusSHA256 string `json:"expected_corpus_sha256"`
}
type response struct {
	Results []result `json:"results"`
}
type indexedDoc struct {
	ID, Title, Text string
	Terms           map[string]int
	Length          int
}
type index struct {
	Docs         []indexedDoc
	DF           map[string]int
	AvgLength    float64
	CorpusSHA256 string
}

var tokens = regexp.MustCompile(`[[:alnum:]]+`)

// Module loads only corpus.jsonl from the operator-staged SciFact directory.
// Empty/missing paths leave a clear unavailable response; API boot remains
// independent of whether the optional benchmark data has been staged.
func Module(dataDir string) module.Module {
	idx, loadErr := loadIndex(dataDir)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if loadErr != nil {
			http.Error(w, "SciFact corpus unavailable: "+loadErr.Error(), http.StatusServiceUnavailable)
			return
		}
		var req searchRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil || strings.TrimSpace(req.Query) == "" {
			http.Error(w, "query is required", http.StatusBadRequest)
			return
		}
		if req.ExpectedCorpusSHA256 != idx.CorpusSHA256 {
			http.Error(w, "staged SciFact corpus SHA-256 does not match registered benchmark source", http.StatusConflict)
			return
		}
		if req.Limit < 1 {
			req.Limit = 10
		}
		if req.Limit > 100 {
			req.Limit = 100
		}
		_ = json.NewEncoder(w).Encode(response{Results: idx.search(req.Query, req.Limit)})
	})
	return module.Module{Name: "publicbench", Mount: func(r *mux.Router) { r.Handle(endpointPath, handler).Methods(http.MethodPost) }}
}

func loadIndex(dataDir string) (*index, error) {
	if strings.TrimSpace(dataDir) == "" {
		return nil, fmt.Errorf("SEARCH_HUB_SCIFACT_DATA_DIR is not configured")
	}
	f, err := os.Open(filepath.Join(dataDir, "corpus.jsonl"))
	if err != nil {
		return nil, fmt.Errorf("open staged corpus.jsonl: %w", err)
	}
	defer f.Close()
	if stat, err := f.Stat(); err != nil {
		return nil, fmt.Errorf("stat staged corpus.jsonl: %w", err)
	} else if stat.Size() > 64<<20 {
		return nil, fmt.Errorf("staged corpus.jsonl exceeds 64 MiB safety bound")
	}
	out := &index{DF: map[string]int{}}
	seen := map[string]struct{}{}
	digest := sha256.New()
	scanner := bufio.NewScanner(io.TeeReader(f, digest))
	scanner.Buffer(make([]byte, 64*1024), 2<<20)
	for scanner.Scan() {
		var src document
		if err := json.Unmarshal(scanner.Bytes(), &src); err != nil {
			return nil, fmt.Errorf("decode corpus record: %w", err)
		}
		id := strings.TrimSpace(src.ID.String())
		text := strings.TrimSpace(strings.Join(src.Abstract, " "))
		if id == "" || id == "0" || text == "" {
			return nil, fmt.Errorf("corpus record requires doc_id and abstract")
		}
		if _, duplicate := seen[id]; duplicate {
			return nil, fmt.Errorf("duplicate document id %s", id)
		}
		seen[id] = struct{}{}
		termCounts := map[string]int{}
		for _, token := range tokens.FindAllString(strings.ToLower(src.Title+" "+text), -1) {
			termCounts[token]++
		}
		for token := range termCounts {
			out.DF[token]++
		}
		length := 0
		for _, n := range termCounts {
			length += n
		}
		out.Docs = append(out.Docs, indexedDoc{ID: id, Title: src.Title, Text: text, Terms: termCounts, Length: length})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read corpus records: %w", err)
	}
	if len(out.Docs) == 0 {
		return nil, fmt.Errorf("staged corpus is empty")
	}
	for _, d := range out.Docs {
		out.AvgLength += float64(d.Length)
	}
	out.AvgLength /= float64(len(out.Docs))
	out.CorpusSHA256 = hex.EncodeToString(digest.Sum(nil))
	return out, nil
}

func (idx *index) search(query string, limit int) []result {
	queryTerms := tokens.FindAllString(strings.ToLower(query), -1)
	if len(queryTerms) == 0 {
		return []result{}
	}
	type scored struct {
		result
		rank float64
	}
	ranked := make([]scored, 0, len(idx.Docs))
	n := float64(len(idx.Docs))
	avg := idx.AvgLength
	if avg < 1 {
		avg = 1
	}
	for _, d := range idx.Docs {
		score := 0.0
		for _, term := range queryTerms {
			tf := float64(d.Terms[term])
			if tf == 0 {
				continue
			}
			df := float64(idx.DF[term])
			idf := 1 + (n-df+0.5)/(df+0.5)
			score += idf * (tf * 2.2) / (tf + 1.2*(0.25+0.75*float64(d.Length)/avg))
		}
		if score <= 0 {
			continue
		}
		snippet := d.Text
		if len(snippet) > 320 {
			snippet = snippet[:320]
		}
		ranked = append(ranked, scored{result: result{ID: d.ID, Title: d.Title, Score: score, Snippet: snippet, Path: d.ID}, rank: score})
	}
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].rank == ranked[j].rank {
			return ranked[i].ID < ranked[j].ID
		}
		return ranked[i].rank > ranked[j].rank
	})
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}
	out := make([]result, len(ranked))
	for i := range ranked {
		out[i] = ranked[i].result
	}
	return out
}
