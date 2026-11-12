package home

import (
	"os"
	"path/filepath"
	"regexp"
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
	// Topics are the existing topics REVIEW.md has a `## <topic>` heading for.
	Topics   []string
	Verdicts *OperatorLog
	// Proposals are the files under proposals/.
	Proposals []FileFact
	// Unreadable lists REVIEW.md, VERDICTS.md or proposals/ when they exist
	// but cannot be read (rulings E1).
	Unreadable []Evidence
}

var reviewTopicHeading = regexp.MustCompile(`(?m)^## +(` + NamePattern + `)\s*$`)

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
		r := &Review{ID: entry.Name(), Dir: dir, Title: Text{Evidence: Unknown("no " + ReviewFile + " yet")}}
		path := filepath.Join(dir, ReviewFile)
		if fact, ok := stat(path); ok {
			r.Review = &fact
			if title, ok := firstHeading(path); ok {
				r.Title = title
			}
			data, err := os.ReadFile(path)
			if err != nil {
				r.Unreadable = append(r.Unreadable, Unreadable(path, err))
			}
			for _, m := range reviewTopicHeading.FindAllStringSubmatch(string(data), -1) {
				if slug, ok := topicOf(m[1]); ok {
					r.Topics = append(r.Topics, slug)
				}
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
