package hub

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
	"unicode"
)

// RelatedAsk is one suggestion from the relatedness read. Basis is the
// strongest evidence the Hub has. A suggestion never merges or authorises
// anything; decision_family remains the producer's claim.
type RelatedAsk struct {
	AskID    string
	Title    string
	Question string
	State    string
	Sender   string
	Score    float64
	Basis    string
}

// relatedLimit is how many suggestions a decision detail shows.
const relatedLimit = 3

// relatedFloor keeps noise out of the detail. Below it, two questions share
// words by accident.
const relatedFloor = 0.30

// Related returns the open decisions worth reading with this one, strongest
// first, limited to asks the audience may already see. The read is advisory:
// it never changes state and never widens what the reader can open.
func (s *Service) Related(ctx context.Context, ask AskRecord, audience AskAudience) []RelatedAsk {
	if strings.TrimSpace(ask.Question) == "" {
		return nil
	}
	visible, args, ok := askVisibility(audience)
	if !ok {
		return nil
	}
	// Suggestions never cross the recipient boundary, even for the owner: an
	// ask addressed to someone else is not this operator's to read.
	where := ` WHERE a.state IN ('pending','escalated') AND a.kind <> 'status' AND a.id <> ? AND ` + forRecipientSQL
	queryArgs := append([]any{ask.ID, ask.Recipient}, args...)
	where += visible
	rows, err := s.db.QueryContext(ctx, `SELECT a.id, a.question, a.subject_key, a.decision_family, a.decision_cause,
		a.sender, a.state, COALESCE((SELECT n.title FROM notifications n WHERE n.id = a.notification_id), '')
		FROM asks a`+where+` LIMIT 200`, queryArgs...)
	if err != nil {
		s.log.Printf("related: read candidates: %v", err)
		return nil
	}
	defer rows.Close()

	askWords := relatedWords(ask.Question)
	var matches []RelatedAsk
	for rows.Next() {
		var candidate RelatedAsk
		var subject, family, cause string
		if err := rows.Scan(&candidate.AskID, &candidate.Question, &subject, &family, &cause, &candidate.Sender, &candidate.State, &candidate.Title); err != nil {
			s.log.Printf("related: scan candidate: %v", err)
			return nil
		}
		switch {
		case ask.DecisionFamily != "" && family == ask.DecisionFamily:
			candidate.Score, candidate.Basis = 1, "family"
		case ask.SubjectKey != "" && subject == ask.SubjectKey:
			candidate.Score, candidate.Basis = 0.95, "subject"
		default:
			candidate.Score = lexicalScore(askWords, relatedWords(candidate.Question))
			candidate.Basis = "lexical"
		}
		if candidate.Title == "" {
			candidate.Title = candidate.Question
		}
		matches = append(matches, candidate)
	}
	if rows.Err() != nil {
		return nil
	}
	if s.currentEmbedder() != nil && len(matches) > 0 {
		// The optional semantic leg can find a paraphrase the words miss. It
		// only raises a score and changes the basis; the lexical read is the
		// floor it must beat, and a failure leaves it untouched.
		ids := make([]string, 0, len(matches))
		for _, match := range matches {
			ids = append(ids, match.AskID)
		}
		if vectors := s.relatedVectors(ctx, ids); len(vectors) > 0 {
			if scores := s.semanticScores(ctx, ask.Question, vectors); len(scores) > 0 {
				for i := range matches {
					if score, ok := scores[matches[i].AskID]; ok && score > matches[i].Score {
						matches[i].Score, matches[i].Basis = score, "semantic"
					}
				}
			}
		}
	}
	kept := matches[:0]
	for _, match := range matches {
		if match.Score >= relatedFloor {
			kept = append(kept, match)
		}
	}
	matches = kept
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].Score != matches[j].Score {
			return matches[i].Score > matches[j].Score
		}
		return matches[i].AskID < matches[j].AskID
	})
	if len(matches) > relatedLimit {
		matches = matches[:relatedLimit]
	}
	return matches
}

// relatedWords is the token set the lexical score compares. Numbers collapse
// so "v4.2" and "4.2" do not read as different subjects.
func relatedWords(text string) map[string]bool {
	words := map[string]bool{}
	var token strings.Builder
	flush := func() {
		word := token.String()
		token.Reset()
		if len(word) < 3 || relatedStop[word] {
			return
		}
		words[word] = true
	}
	for _, r := range strings.ToLower(text) {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r):
			token.WriteRune(r)
		default:
			flush()
		}
	}
	flush()
	return words
}

var relatedStop = map[string]bool{
	"the": true, "and": true, "for": true, "with": true, "this": true, "that": true,
	"from": true, "into": true, "should": true, "would": true, "could": true, "does": true,
	"what": true, "when": true, "which": true, "have": true, "keep": true, "change": true,
	"ask": true, "decision": true, "use": true, "yes": true, "not": true, "are": true,
}

// lexicalScore is a bounded similarity in [0,1]: shared words plus shared
// adjacent pairs, so word order counts a little without needing a model.
func lexicalScore(a, b map[string]bool) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	shared := 0
	for word := range a {
		if b[word] {
			shared++
		}
	}
	union := len(a) + len(b) - shared
	if union == 0 {
		return 0
	}
	score := float64(shared) / float64(union)
	return math.Round(score*100) / 100
}

// decorateRead fills the projections a decision detail shows and a list does
// not: suggestions and the owner's follow-through report. Both are audience
// limited and advisory.
func (s *Service) decorateRead(ctx context.Context, record *AskRecord, audience AskAudience) {
	record.Related = s.Related(ctx, *record, audience)
	report, err := s.ReadFollowthrough(ctx, record.ID)
	if err == nil {
		record.Followthrough = report
	} else {
		s.log.Printf("read: follow-through for %s: %v", record.ID, err)
	}
}

// Followthrough is what the owner reported about the run an answer started.
// Its absence is the honest "no owner has reported" state.
type Followthrough struct {
	State        string
	Owner        string
	RunRef       string
	StartedAt    string
	FinishedAt   string
	BoundSeconds int
	ResultURL    string
	Summary      string
	ReportedAt   string
}

var followthroughStates = map[string]bool{"": true, "queued": true, "running": true, "done": true, "stalled": true, "failed": true}

// RecordFollowthrough stores the owner's report, replacing any earlier one.
func (s *Service) RecordFollowthrough(ctx context.Context, askID string, report Followthrough) (Followthrough, error) {
	askID = strings.TrimSpace(askID)
	if askID == "" {
		return Followthrough{}, ErrNotFound
	}
	report.State = strings.ToLower(strings.TrimSpace(report.State))
	if !followthroughStates[report.State] {
		return Followthrough{}, fmt.Errorf("%w: followthrough state must be queued, running, done, stalled or failed", ErrInvalidArgument)
	}
	if report.State == "" {
		return Followthrough{}, fmt.Errorf("%w: followthrough state is required", ErrInvalidArgument)
	}
	if report.BoundSeconds < 0 {
		return Followthrough{}, fmt.Errorf("%w: followthrough bound cannot be negative", ErrInvalidArgument)
	}
	for name, value := range map[string]string{"started_at": report.StartedAt, "finished_at": report.FinishedAt} {
		if value == "" {
			continue
		}
		if _, err := time.Parse(time.RFC3339Nano, value); err != nil {
			return Followthrough{}, fmt.Errorf("%w: followthrough %s must be RFC 3339", ErrInvalidArgument, name)
		}
	}
	for target, value := range map[*string]string{
		&report.Owner:     report.Owner,
		&report.RunRef:    report.RunRef,
		&report.ResultURL: report.ResultURL,
		&report.Summary:   report.Summary,
	} {
		*target = strings.TrimSpace(value)
	}
	if len(report.Summary) > 500 {
		return Followthrough{}, fmt.Errorf("%w: followthrough summary is at most 500 characters", ErrInvalidArgument)
	}
	state, err := s.getAskState(ctx, askID)
	if err != nil {
		return Followthrough{}, err
	}
	if state != AskAnswered && state != AskDefaulted {
		return Followthrough{}, fmt.Errorf("%w: follow-through starts after the decision is answered", ErrInvalidArgument)
	}
	now := s.clock.Now().UTC().Format(time.RFC3339Nano)
	report.ReportedAt = now
	if _, err := s.db.ExecContext(ctx, `INSERT INTO ask_followthrough
		(ask_id, state, owner, run_ref, started_at, finished_at, bound_seconds, result_url, summary, reported_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(ask_id) DO UPDATE SET state=excluded.state, owner=excluded.owner, run_ref=excluded.run_ref,
		started_at=excluded.started_at, finished_at=excluded.finished_at, bound_seconds=excluded.bound_seconds,
		result_url=excluded.result_url, summary=excluded.summary, reported_at=excluded.reported_at, updated_at=excluded.updated_at`,
		askID, report.State, report.Owner, report.RunRef, report.StartedAt, report.FinishedAt,
		report.BoundSeconds, report.ResultURL, report.Summary, now, now); err != nil {
		return Followthrough{}, err
	}
	return report, nil
}

// Followthrough reads the owner's report for one ask; nil when none exists.
func (s *Service) ReadFollowthrough(ctx context.Context, askID string) (*Followthrough, error) {
	var report Followthrough
	err := s.db.QueryRowContext(ctx, `SELECT state, owner, run_ref, started_at, finished_at, bound_seconds, result_url, summary, reported_at
		FROM ask_followthrough WHERE ask_id = ?`, askID).Scan(&report.State, &report.Owner, &report.RunRef,
		&report.StartedAt, &report.FinishedAt, &report.BoundSeconds, &report.ResultURL, &report.Summary, &report.ReportedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &report, nil
}

func (s *Service) getAskState(ctx context.Context, askID string) (string, error) {
	var state string
	err := s.db.QueryRowContext(ctx, `SELECT state FROM asks WHERE id = ?`, askID).Scan(&state)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	return state, err
}
