package home

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vrooli/api-core/researchhome"
)

const (
	maxFeedbackEntries = 500
	maxFeedbackFiles   = 40
	maxFeedbackDepth   = 5
)

// TopicOptionalFile distinguishes an absent optional entry point from an
// unreadable one. The Desk links to the source; it never interprets its prose.
type TopicOptionalFile struct {
	File *FileFact
	Evidence
}

// TopicDocuments is bounded navigation metadata for the topic's optional
// working-memory views, feedback records, skills and programs.
type TopicDocuments struct {
	CurrentView       TopicOptionalFile
	Tools             TopicOptionalFile
	FeedbackFiles     []FileFact
	FeedbackEvidence  Evidence
	FeedbackTruncated bool
	Capabilities      Capabilities
}

// Capabilities are the research skills and programs one scope holds — a
// topic's skills/ and programs/, or the research home's own (supervisor)
// ones — by prefix id, each with the problems that stop it resolving.
// Prompt Manager and Program Runtime read the same files by the same ids;
// the desk only reports, it is never on their path.
type Capabilities struct {
	Items []researchhome.Item
	Evidence
}

// readCapabilities lists a scope's skills and programs; topic "" is the
// research home itself.
func readCapabilities(researchDir, topic string) Capabilities {
	scope := researchhome.Ref{Topic: topic}.ScopeDir(researchDir)
	items, err := researchhome.List(researchDir, topic)
	if err != nil {
		return Capabilities{Items: items, Evidence: Unknown("cannot list research skills and programs under " + scope + ": " + err.Error())}
	}
	return Capabilities{Items: items, Evidence: Evidence{Known: true, Source: scope}}
}

func readTopicDocuments(topicDir, researchDir string) TopicDocuments {
	root, rootErr := filepath.EvalSymlinks(researchDir)
	resolvedTopic, topicErr := filepath.EvalSymlinks(topicDir)
	if rootErr != nil || topicErr != nil || !pathWithin(root, resolvedTopic) {
		reason := "topic directory cannot be safely resolved within the research home"
		return TopicDocuments{
			CurrentView:      TopicOptionalFile{Evidence: Unknown(reason)},
			Tools:            TopicOptionalFile{Evidence: Unknown(reason)},
			FeedbackEvidence: Unknown(reason),
			Capabilities:     Capabilities{Evidence: Unknown(reason)},
		}
	}
	docs := TopicDocuments{
		CurrentView: topicOptionalFile(resolvedTopic, "CURRENT.md"),
		Tools:       topicOptionalFile(resolvedTopic, "TOOLS.md"),
	}
	docs.FeedbackFiles, docs.FeedbackEvidence, docs.FeedbackTruncated = feedbackFiles(resolvedTopic)
	docs.Capabilities = readCapabilities(researchDir, filepath.Base(topicDir))
	return docs
}

func topicOptionalFile(topicDir, name string) TopicOptionalFile {
	path := filepath.Join(topicDir, name)
	info, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return TopicOptionalFile{Evidence: Evidence{Known: true, Source: path, Reason: "optional file is absent"}}
	}
	if err != nil {
		return TopicOptionalFile{Evidence: Unknown("cannot inspect optional file " + path + ": " + err.Error())}
	}
	if !info.Mode().IsRegular() && info.Mode()&os.ModeSymlink == 0 {
		return TopicOptionalFile{Evidence: Unknown("optional path is not a regular file: " + path)}
	}
	resolved, err := resolveTopicPath(topicDir, path)
	if err != nil {
		return TopicOptionalFile{Evidence: Unknown("cannot safely resolve optional file " + path + ": " + err.Error())}
	}
	info, err = os.Stat(resolved)
	if err != nil || !info.Mode().IsRegular() {
		if err == nil {
			err = errors.New("not a regular file")
		}
		return TopicOptionalFile{Evidence: Unknown("cannot read optional file " + path + ": " + err.Error())}
	}
	return TopicOptionalFile{
		File:     &FileFact{Name: name, Path: resolved, Bytes: info.Size(), Modified: info.ModTime()},
		Evidence: From(resolved, info.ModTime()),
	}
}

func feedbackFiles(topicDir string) ([]FileFact, Evidence, bool) {
	dir := filepath.Join(topicDir, "feedback")
	info, err := os.Lstat(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, Evidence{Known: true, Source: dir, Reason: "optional feedback directory is absent"}, false
	}
	if err != nil {
		return nil, Unknown("cannot inspect feedback directory " + dir + ": " + err.Error()), false
	}
	if !info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
		return nil, Unknown("feedback path is not a directory: " + dir), false
	}
	resolved, err := resolveTopicPath(topicDir, dir)
	if err != nil {
		return nil, Unknown("cannot safely resolve feedback directory " + dir + ": " + err.Error()), false
	}
	info, err = os.Stat(resolved)
	if err != nil || !info.IsDir() {
		if err == nil {
			err = errors.New("not a directory")
		}
		return nil, Unknown("cannot read feedback directory " + dir + ": " + err.Error()), false
	}
	files := make([]FileFact, 0, 8)
	visited := 0
	truncated := false
	err = filepath.WalkDir(resolved, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		visited++
		if visited > maxFeedbackEntries {
			truncated = true
			return fs.SkipAll
		}
		rel, relErr := filepath.Rel(resolved, path)
		if relErr != nil {
			return relErr
		}
		depth := 0
		if rel != "." {
			depth = strings.Count(rel, string(filepath.Separator)) + 1
		}
		if depth > maxFeedbackDepth {
			truncated = true
			if entry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if path != resolved && strings.HasPrefix(entry.Name(), ".") {
			if entry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			truncated = true // Do not follow links; their contents are not in the read set.
			return nil
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		if len(files) >= maxFeedbackFiles {
			truncated = true
			return fs.SkipAll
		}
		fileInfo, statErr := entry.Info()
		if statErr != nil {
			return statErr
		}
		files = append(files, FileFact{Name: filepath.ToSlash(rel), Path: path, Bytes: fileInfo.Size(), Modified: fileInfo.ModTime()})
		return nil
	})
	if err != nil {
		return files, Unknown("feedback directory walk is incomplete: " + err.Error()), true
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	evidence := From(resolved, info.ModTime())
	if truncated {
		evidence.Reason = "feedback listing is bounded; some entries were not followed or returned"
	}
	return files, evidence, truncated
}

func resolveTopicPath(topicDir, path string) (string, error) {
	root, err := filepath.EvalSymlinks(topicDir)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	if !pathWithin(root, resolved) {
		return "", errors.New("path resolves outside topic")
	}
	return resolved, nil
}

func pathWithin(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}
