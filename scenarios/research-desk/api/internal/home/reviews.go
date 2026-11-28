package home

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
)

// Review files written by the research-review skill (FILE-CONTRACT.md §1).
const (
	ReviewFile   = "REVIEW.md"
	ReviewHTML   = "REVIEW.html"
	VerdictsFile = "VERDICTS.md"
	ProposalsDir = "proposals"
	// ReviewsDir is the reviews folder under the research home.
	ReviewsDir = "reviews"
)

// Review is one `reviews/<id>/` folder.
type Review struct {
	ID    string
	Dir   string
	Title Text
	// Review is REVIEW.md, nil while the agent has not written it.
	Review *FileFact
	// HTML is REVIEW.html with embedded media, when the review wrote one.
	HTML *FileFact
	// Topics are the existing topics REVIEW.md has a `## <topic>` heading
	// for, else the ones it links as topics/<topic>. TopicsEvidence says
	// which, or why they are unknown: never an empty list that reads as none.
	Topics         []string
	TopicsEvidence Evidence
	Verdicts       *OperatorLog
	// Proposals are the files under proposals/.
	Proposals []FileFact
	// Unreadable lists REVIEW.md, VERDICTS.md or proposals/ when they exist
	// but cannot be read.
	Unreadable []Evidence
}

var (
	reviewTopicHeading = regexp.MustCompile(`(?m)^## +(` + NamePattern + `)\s*$`)
	reviewTopicLink    = regexp.MustCompile(`\btopics/(` + NamePattern + `)\b`)
)

// reviewTopics are the existing topics a REVIEW.md names: its per-topic
// headings, else its links into topics/. Neither leaves them unknown.
func reviewTopics(path string, data []byte, topicOf func(string) (string, bool)) ([]string, Evidence) {
	for _, way := range []struct {
		re   *regexp.Regexp
		what string
	}{{reviewTopicHeading, "its ## <topic> headings"}, {reviewTopicLink, "its topics/<topic> links"}} {
		var out []string
		for _, m := range way.re.FindAllStringSubmatch(string(data), -1) {
			if slug, ok := topicOf(m[1]); ok && !slices.Contains(out, slug) {
				out = append(out, slug)
			}
		}
		if len(out) > 0 {
			return out, Evidence{Known: true, Source: path, Reason: "from " + way.what}
		}
	}
	return nil, Unknown(path + " names no topic by a ## <topic> heading or a topics/<topic> link")
}

// readReviews lists reviews/<id>/ folders, newest REVIEW.md first. A missing
// reviews folder has none; an unreadable one is an error.
func readReviews(researchDir string, topicOf func(string) (string, bool)) ([]*Review, error) {
	root := filepath.Join(researchDir, ReviewsDir)
	entries, err := os.ReadDir(root)
	if absent(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []*Review
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		dir := filepath.Join(root, entry.Name())
		r := &Review{ID: entry.Name(), Dir: dir, Title: Text{Evidence: Unknown("no " + ReviewFile + " yet")}, TopicsEvidence: Unknown("no " + ReviewFile + " yet")}
		path := filepath.Join(dir, ReviewFile)
		if fact, ok := stat(path); ok {
			r.Review = &fact
			if title, ok := firstHeading(path); ok {
				r.Title = title
			}
			if data, err := os.ReadFile(path); err != nil {
				r.Unreadable = append(r.Unreadable, Unreadable(path, err))
				r.TopicsEvidence = Unknown("cannot read " + path + ": " + err.Error())
			} else {
				r.Topics, r.TopicsEvidence = reviewTopics(path, data, topicOf)
			}
		}
		if fact, ok := stat(filepath.Join(dir, ReviewHTML)); ok {
			r.HTML = &fact
		}
		if r.Verdicts, err = readOperatorFile(filepath.Join(dir, VerdictsFile)); err != nil {
			r.Unreadable = append(r.Unreadable, Unreadable(filepath.Join(dir, VerdictsFile), err))
		}
		if r.Proposals, err = topFiles(filepath.Join(dir, ProposalsDir)); err != nil {
			r.Unreadable = append(r.Unreadable, Unreadable(filepath.Join(dir, ProposalsDir), err))
		}
		out = append(out, r)
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i].Review, out[j].Review
		switch {
		case a == nil || b == nil:
			return a != nil
		default:
			return a.Modified.After(b.Modified)
		}
	})
	return out, nil
}
