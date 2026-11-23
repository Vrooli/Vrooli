package courier

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/api-core/decisionpage"

	"research-desk/internal/home"
)

// A decision whose raiser left its page out goes with a page the shared
// decision-page writer (packages/api-core/decisionpage) makes from the item,
// its brief and the file it is in: a Codex run that follows the
// notification-hub skill and passes the Hub's own packet check. The writer
// presents what the brief says; it never fills in what the brief lacks, so
// only a decision whose folder is otherwise ready is handed to it. The raiser's
// own page always wins. The item stays packet_not_ready until the page is
// ready (the operator answers in hours, a page takes minutes), and stays so
// when the writer gives up or is not there: no page restating the item ever
// reaches the operator. A writer that gave up is a page miss, recorded
// (PageMissesFile) and reported (the plan row, Health).

// PageWriter is the decision-page writer as the courier uses it: Status reads
// a job without changing anything (the dry run), Advance starts or moves it
// on without blocking (a pass). *decisionpage.Writer is one.
type PageWriter interface {
	Status(decisionpage.Request) (decisionpage.Result, error)
	Advance(context.Context, decisionpage.Request) (decisionpage.Result, error)
}

// Sender is who the writer's pages are for.
const Sender = "Research Desk"

// PageMissesFile records, in the desk state folder, every decision the
// writer gave up on, so it was held unsent: a map from the item's key to its
// latest PageMiss.
const PageMissesFile = "page-misses.json"

// PageMiss is one decision held unsent because the writer gave up on its page.
type PageMiss struct {
	At    time.Time `json:"at"`
	Ask   string    `json:"ask"`
	Class string    `json:"class"`
	// Detail is the writer's own account of the failure.
	Detail string `json:"detail"`
	// Job is the writer's job folder: its attempts, pages and findings.
	Job string `json:"job"`
}

// pageJob is one item's page as the writer holds it this pass.
type pageJob struct {
	request decisionpage.Request
	result  decisionpage.Result
	// err is why the job could not be read; the item waits.
	err error
}

// written is the job's page, nil until the writer has one.
func (j *pageJob) page() *Page {
	if j == nil || j.err != nil || j.result.State != decisionpage.StateReady || j.result.HTML == "" {
		return nil
	}
	return &Page{HTML: j.result.HTML, SHA256: j.result.SHA256, Written: j.result.Page}
}

// missed is the writer's failure when it has given up on the item: every
// failure except an outage, which it retries.
func (j *pageJob) missed() *decisionpage.Failure {
	if j == nil || j.err != nil || j.result.State != decisionpage.StateFailed || !j.result.Terminal() || j.result.Failure == nil {
		return nil
	}
	return j.result.Failure
}

// waiting says what the ask waits for while the writer works.
func (j *pageJob) waiting() string {
	const lead = "waiting for its page from the decision-page writer"
	switch {
	case j.err != nil:
		return lead + ": the job cannot be read (" + j.err.Error() + "); the next pass tries again"
	case j.result.State == "":
		return lead + ": not started; this pass starts it"
	case j.result.State == decisionpage.StateFailed && j.result.Failure != nil:
		return lead + ": the writer is not available (" + j.result.Failure.Error() + "); it tries again after its cool-down"
	}
	why := lead + ": " + string(j.result.State)
	if n := len(j.result.Attempts); n > 0 {
		a := j.result.Attempts[n-1]
		why += fmt.Sprintf(", attempt %d", a.N)
		if a.RunID != "" {
			why += " (agent run " + a.RunID
			if a.Model != "" {
				why += ", " + a.Model
			}
			why += ")"
		}
		if !a.StartedAt.IsZero() {
			why += " since " + stamp(a.StartedAt)
		}
	}
	return why + "; job " + j.result.Dir
}

// pageRequest is what the writer needs for an item: the operator's text, the
// answers with what executes on each, and the files that hold the rest (the
// brief first). The brief's hash is in the technical details, so a revised
// brief is a new job and a fresh page.
func pageRequest(f *home.FeedItem, effects map[string]string, brief []byte, now time.Time) decisionpage.Request {
	r := decisionpage.Request{
		Sender:     Sender,
		SubjectKey: AskPrefix + f.Key(),
		Question:   f.Item.Title,
		BigPicture: f.Item.Prose(),
		IfNoAnswer: ifNoAnswer(f.Item, now),
	}
	for _, o := range f.Item.Choices() {
		r.Options = append(r.Options, decisionpage.Option{Key: o.Key, Label: o.Label, Effect: effects[o.Key]})
	}
	if rec := f.Item.Recommended; rec != nil {
		r.Recommended = rec.Key
		r.RecommendationReason = f.Item.Reason
		if r.RecommendationReason == "" {
			r.RecommendationReason = rec.Label
		}
	}
	for _, path := range []string{f.Item.PacketPath(), f.Item.Source} {
		if filepath.IsAbs(path) {
			r.Evidence = append(r.Evidence, path)
		}
	}
	sum := sha256.Sum256(brief)
	r.TechnicalDetails = technicalDetails(f, AskPrefix+f.Key(), nil, effects) + "\nBrief sha256: " + hex.EncodeToString(sum[:8])
	r.Facts = "The decision's brief (the first evidence file) is what an agent executes on the answer: its commands, " +
		"what each answer settles, and the history and cause behind the ask. Commands the operator must run are " +
		"under its `## Commands`, marked operator-only."
	return r
}

// readJobs reads the writer's job for every decision whose page was left
// out; a pass's write-page steps move them on. Without a writer such a decision is not
// ready: nothing could make its page.
func (c *Courier) readJobs(pages *Pages) {
	for key, req := range pages.wanted {
		if c.PageWriter == nil {
			pages.unready[key] = joinWhy(pages.unready[key], "its page does not exist and this desk has no decision-page writer to make it")
			continue
		}
		res, err := c.PageWriter.Status(req)
		job := &pageJob{request: req, result: res, err: err}
		pages.jobs[key] = job
		if p := job.page(); p != nil {
			pages.byKey[key] = p
		}
	}
}

// misses reads the recorded page misses; a missing file is none.
func (c *Courier) misses() (map[string]PageMiss, error) {
	m := map[string]PageMiss{}
	if _, err := c.Writer.ReadState(PageMissesFile, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// held are the pass's page misses: the decisions it holds unsent because the
// writer gave up on their page, by item key.
func held(plan Plan, at time.Time) map[string]PageMiss {
	out := map[string]PageMiss{}
	for _, s := range plan.Steps {
		if s.Item == nil || s.Reason != home.ReasonPacketNotReady {
			continue
		}
		if job := plan.pages.job(s.Item); job != nil && job.missed() != nil {
			f := job.missed()
			out[s.Item.Key()] = PageMiss{At: at.UTC(), Ask: s.Key, Class: f.Class, Detail: f.Detail, Job: job.result.Dir}
		}
	}
	return out
}

// recordMisses adds the pass's page misses to PageMissesFile; a miss already
// recorded for the same job and failure keeps its first time.
func (c *Courier) recordMisses(plan Plan, at time.Time) error {
	changed := false
	for key, m := range held(plan, at) {
		if old, ok := plan.misses[key]; ok && old.Job == m.Job && old.Class == m.Class {
			continue
		}
		plan.misses[key] = m
		changed = true
	}
	if !changed {
		return nil
	}
	_, err := c.Writer.WriteState(PageMissesFile, plan.misses)
	return err
}

// sayMisses is the decisions held for a page miss now, in one sentence; ""
// for none.
func sayMisses(m map[string]PageMiss) string {
	if len(m) == 0 {
		return ""
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	first := m[keys[0]]
	return fmt.Sprintf("%d decision(s) held unsent because the decision-page writer could not make their page; first %s (%s: %s; job %s)",
		len(m), keys[0], first.Class, strings.TrimSpace(first.Detail), first.Job)
}
