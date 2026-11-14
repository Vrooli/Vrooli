package jobs

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
)

var ErrInvalid = errors.New("invalid job pipeline record")

type Service struct {
	db        *sql.DB
	sources   []Source
	providers []DiscoveryProvider
}

// DiscoveryProvider is an optional, isolated integration. A provider failure is
// recorded beside successful results and never prevents other sources running.
type DiscoveryProvider interface {
	Name() string
	Search(context.Context, string) ([]Opening, error)
}

// ConfiguredSourcesFromEnv reads optional link-out sources. Invalid or absent
// configuration falls back to useful public search links instead of disabling discovery.
func ConfiguredSourcesFromEnv() []Source {
	raw := strings.TrimSpace(os.Getenv("PROSE_STUDIO_JOB_SOURCES"))
	if raw == "" {
		return nil
	}
	var sources []Source
	if json.Unmarshal([]byte(raw), &sources) != nil {
		return nil
	}
	valid := make([]Source, 0, len(sources))
	for _, v := range sources {
		if validText(v.ID) && validText(v.Name) && strings.Contains(v.URLTemplate, "{query}") {
			v.Configured = true
			valid = append(valid, v)
		}
	}
	if len(valid) == 0 {
		return nil
	}
	return valid
}

func New(db *sql.DB, sources []Source, providers ...DiscoveryProvider) *Service {
	if len(sources) == 0 {
		sources = []Source{
			{ID: "linkedin", Name: "LinkedIn", URLTemplate: "https://www.linkedin.com/jobs/search/?keywords={query}"},
			{ID: "indeed", Name: "Indeed", URLTemplate: "https://www.indeed.com/jobs?q={query}"},
			{ID: "google", Name: "Google Jobs", URLTemplate: "https://www.google.com/search?ibp=htl;jobs&q={query}"},
		}
	}
	return &Service{db: db, sources: sources, providers: providers}
}
func id() string {
	var b [16]byte
	if _, e := rand.Read(b[:]); e != nil {
		return fmt.Sprintf("job-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}
func validText(s string) bool { return strings.TrimSpace(s) != "" }

// saveRecord handles different table shapes without exposing dynamic SQL inputs.
func (s *Service) saveRecord(ctx context.Context, table string, v any, key, related string) error {
	b, e := json.Marshal(v)
	if e != nil {
		return e
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if related == "" {
		_, e = s.db.ExecContext(ctx, "INSERT INTO "+table+"(id,payload,created_at,updated_at) VALUES(?,?,?,?)", key, string(b), now, now)
	} else {
		_, e = s.db.ExecContext(ctx, "INSERT INTO "+table+"(id,opening_id,payload,created_at,updated_at) VALUES(?,?,?,?,?)", key, related, string(b), now, now)
	}
	return e
}
func (s *Service) list(ctx context.Context, table string) ([]json.RawMessage, error) {
	rows, e := s.db.QueryContext(ctx, "SELECT payload FROM "+table+" ORDER BY created_at DESC")
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []json.RawMessage{}
	for rows.Next() {
		var b []byte
		if e = rows.Scan(&b); e != nil {
			return nil, e
		}
		out = append(out, json.RawMessage(b))
	}
	return out, rows.Err()
}
func decodeList[T any](raw []json.RawMessage) []T {
	out := make([]T, 0, len(raw))
	for _, b := range raw {
		var v T
		if json.Unmarshal(b, &v) == nil {
			out = append(out, v)
		}
	}
	return out
}
func (s *Service) Openings(ctx context.Context) ([]Opening, error) {
	r, e := s.list(ctx, "job_openings")
	return decodeList[Opening](r), e
}
func (s *Service) Facts(ctx context.Context) ([]Fact, error) {
	r, e := s.list(ctx, "job_facts")
	return decodeList[Fact](r), e
}
func (s *Service) Resumes(ctx context.Context) ([]Resume, error) {
	r, e := s.list(ctx, "job_resumes")
	return decodeList[Resume](r), e
}
func (s *Service) Letters(ctx context.Context) ([]CoverLetter, error) {
	r, e := s.list(ctx, "job_cover_letters")
	return decodeList[CoverLetter](r), e
}
func (s *Service) CreateOpening(ctx context.Context, v Opening) (Opening, error) {
	if !validText(v.Company) || !validText(v.Title) {
		return v, ErrInvalid
	}
	if v.Status == "" {
		v.Status = "saved"
	}
	if v.ID == "" {
		v.ID = id()
	}
	e := s.saveRecord(ctx, "job_openings", v, v.ID, "")
	return v, e
}
func (s *Service) CreateFact(ctx context.Context, v Fact) (Fact, error) {
	if !validText(v.Label) || !validText(v.Content) {
		return v, ErrInvalid
	}
	if v.ID == "" {
		v.ID = id()
	}
	e := s.saveRecord(ctx, "job_facts", v, v.ID, "")
	return v, e
}
func (s *Service) CreateResume(ctx context.Context, v Resume) (Resume, error) {
	if !validText(v.Title) || !validText(v.Content) {
		return v, ErrInvalid
	}
	if v.ID == "" {
		v.ID = id()
	}
	e := s.saveRecord(ctx, "job_resumes", v, v.ID, v.OpeningID)
	return v, e
}
func (s *Service) CreateLetter(ctx context.Context, v CoverLetter) (CoverLetter, error) {
	if !validText(v.Title) || !validText(v.Content) {
		return v, ErrInvalid
	}
	if v.ID == "" {
		v.ID = id()
	}
	e := s.saveRecord(ctx, "job_cover_letters", v, v.ID, v.OpeningID)
	return v, e
}
func (s *Service) UpdateOpening(ctx context.Context, id string, patch map[string]any) (Opening, error) {
	var v Opening
	var b []byte
	e := s.db.QueryRowContext(ctx, "SELECT payload FROM job_openings WHERE id=?", id).Scan(&b)
	if e != nil {
		return v, e
	}
	_ = json.Unmarshal(b, &v)
	for k, x := range patch {
		str, ok := x.(string)
		if !ok {
			continue
		}
		switch k {
		case "category":
			v.Category = str
		case "status":
			v.Status = str
		case "source":
			v.Source = str
		case "source_id":
			v.SourceID = str
		case "location":
			v.Location = str
		case "url":
			v.URL = str
		case "description":
			v.Description = str
		case "company":
			v.Company = str
		case "title":
			v.Title = str
		}
	}
	b, e = json.Marshal(v)
	if e != nil {
		return v, e
	}
	_, e = s.db.ExecContext(ctx, "UPDATE job_openings SET payload=?,updated_at=? WHERE id=?", string(b), time.Now().UTC().Format(time.RFC3339Nano), id)
	return v, e
}

var reviewWords = regexp.MustCompile(`[a-zA-Z0-9%]+`)
var claimSignal = regexp.MustCompile(`(?i)\b(led|managed|built|created|increased|reduced|improved|delivered|achieved|generated|grew|saved|owned|launched|drove|\d+%?)\b`)

func (s *Service) ReviewResume(ctx context.Context, id string) (Resume, error) {
	var out Resume
	var payload []byte
	if err := s.db.QueryRowContext(ctx, "SELECT payload FROM job_resumes WHERE id=?", id).Scan(&payload); err != nil {
		return out, err
	}
	if err := json.Unmarshal(payload, &out); err != nil {
		return out, err
	}
	factByID := map[string]Fact{}
	for _, factID := range out.FactIDs {
		var factJSON []byte
		if err := s.db.QueryRowContext(ctx, "SELECT payload FROM job_facts WHERE id=?", factID).Scan(&factJSON); err == nil {
			var fact Fact
			if json.Unmarshal(factJSON, &fact) == nil {
				factByID[factID] = fact
			}
		}
	}
	factCorpus := ""
	for _, f := range factByID {
		factCorpus += " " + f.Content
	}
	findings := []string{"Automated review is a deterministic text comparison; it cannot verify truth, context, or implied claims."}
	if len(out.FactIDs) == 0 {
		findings = append(findings, "No reusable facts are linked, so factual resume claims lack linked support.")
	}
	if out.OpeningID != "" {
		var openingJSON []byte
		if err := s.db.QueryRowContext(ctx, "SELECT payload FROM job_openings WHERE id=?", out.OpeningID).Scan(&openingJSON); err == nil {
			var opening Opening
			if json.Unmarshal(openingJSON, &opening) == nil {
				roleTerms := significantTerms(opening.Title)
				resumeTerms := termSet(out.Content)
				missing := []string{}
				for _, term := range roleTerms {
					if !resumeTerms[term] {
						missing = append(missing, term)
					}
				}
				if len(missing) > 0 {
					findings = append(findings, "Opening role keywords missing from resume: "+strings.Join(missing, ", ")+".")
				}
			}
		}
	}
	for _, sentence := range regexp.MustCompile(`[.!?\n]+`).Split(out.Content, -1) {
		sentence = strings.TrimSpace(sentence)
		if sentence == "" || !claimSignal.MatchString(sentence) {
			continue
		}
		if len(factByID) == 0 || !supportedByFacts(sentence, factCorpus) {
			findings = append(findings, fmt.Sprintf("Unsupported claim: %q has no close match in linked facts (%s).", sentence, strings.Join(out.FactIDs, ", ")))
		}
	}
	out.ReviewFindings = findings
	updated, err := json.Marshal(out)
	if err != nil {
		return out, err
	}
	_, err = s.db.ExecContext(ctx, "UPDATE job_resumes SET payload=?,updated_at=? WHERE id=?", string(updated), time.Now().UTC().Format(time.RFC3339Nano), id)
	return out, err
}

func termSet(text string) map[string]bool {
	set := map[string]bool{}
	for _, term := range reviewWords.FindAllString(strings.ToLower(text), -1) {
		if len(term) > 2 {
			set[term] = true
		}
	}
	return set
}
func significantTerms(text string) []string {
	set := termSet(text)
	out := []string{}
	for term := range set {
		switch term {
		case "senior", "junior", "lead", "staff", "principal", "engineer", "writer", "designer", "manager", "director", "analyst", "associate", "the", "and", "for", "with":
			continue
		}
		out = append(out, term)
	}
	sort.Strings(out)
	return out
}
func supportedByFacts(sentence, corpus string) bool {
	claim := termSet(sentence)
	facts := termSet(corpus)
	for _, term := range reviewWords.FindAllString(strings.ToLower(sentence), -1) {
		if len(term) > 0 && term[0] >= '0' && term[0] <= '9' && !facts[term] {
			return false
		}
	}
	matched := 0
	for term := range claim {
		if facts[term] {
			matched++
		}
	}
	return matched >= 2
}
func (s *Service) Sources() []Source {
	out := append([]Source(nil), s.sources...)
	for _, p := range s.providers {
		out = append(out, Source{ID: p.Name(), Name: p.Name(), Configured: true})
	}
	return out
}
func (s *Service) Discover(ctx context.Context, query string) (DiscoveryResult, error) {
	if !validText(query) {
		return DiscoveryResult{}, ErrInvalid
	}
	q := url.QueryEscape(query)
	out := DiscoveryResult{Results: []SearchResult{}}
	for _, source := range s.sources {
		link := strings.ReplaceAll(source.URLTemplate, "{query}", q)
		out.Results = append(out.Results, SearchResult{Source: source.ID, Query: query, URL: link, Status: "link_out"})
	}
	type providerResult struct {
		index int
		items []Opening
		err   error
	}
	results := make(chan providerResult, len(s.providers))
	for i, p := range s.providers {
		go func(i int, p DiscoveryProvider) {
			providerCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()
			items, err := p.Search(providerCtx, query)
			results <- providerResult{index: i, items: items, err: err}
		}(i, p)
	}
	providerOut := make([]SearchResult, len(s.providers))
	for i, p := range s.providers {
		providerOut[i] = SearchResult{Source: p.Name(), Query: query, Status: "unavailable", Error: "provider timed out"}
	}
	deadline := time.NewTimer(3 * time.Second)
	defer deadline.Stop()
	for remaining := len(s.providers); remaining > 0; {
		select {
		case result := <-results:
			remaining--
			r := providerOut[result.index]
			if result.err == nil {
				r.Status = "ok"
				r.Error = ""
				r.Openings = result.items
			} else {
				r.Error = result.err.Error()
				out.Partial = true
			}
			providerOut[result.index] = r
		case <-deadline.C:
			out.Partial = true
			remaining = 0
		case <-ctx.Done():
			out.Partial = true
			remaining = 0
		}
	}
	for _, r := range providerOut {
		if r.Status == "unavailable" {
			out.Partial = true
		}
		out.Results = append(out.Results, r)
	}
	return out, nil
}
