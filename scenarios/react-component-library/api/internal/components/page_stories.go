package components

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"react-component-library/internal/librarywalk"
)

const PageStoryPrefix = "page:"
const PageStoryVersion = "workspace"

// PageStorySubject is a working application subject, not a published asset.
// It shares the declarative contract and runner with released components.
type PageStorySubject struct {
	Projection ComponentStory
	Contract   *StoryContract
	Revision   string
}

func LoadPageStories(ctx context.Context, scenarioRoot string) ([]PageStorySubject, error) {
	uiRoot := filepath.Join(scenarioRoot, "ui", "src")
	var paths []string
	digest := sha256.New()
	err := librarywalk.WalkContext(ctx, uiRoot, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("page story source cannot follow symlink %s", path)
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(uiRoot, path)
		fmt.Fprintf(digest, "%s\x00%d\x00", rel, len(raw))
		_, _ = digest.Write(raw)
		if strings.HasSuffix(path, ".story.json") {
			paths = append(paths, path)
		}
		return nil
	})
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	revision := hex.EncodeToString(digest.Sum(nil))
	var subjects []PageStorySubject
	seen := map[string]bool{}
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		contract, diagnostics := ParseStoryContract(raw)
		if problems := StoryContractErrors(diagnostics); len(problems) > 0 {
			return nil, fmt.Errorf("%s: %v", path, problems)
		}
		if contract.Kind != StoryKindPage {
			return nil, fmt.Errorf("%s: application story must have kind page", path)
		}
		name := strings.TrimSuffix(filepath.Base(path), ".story.json")
		id := PageStoryPrefix + name
		if seen[id] {
			return nil, fmt.Errorf("duplicate page story subject %s", id)
		}
		seen[id] = true
		source, err := os.Stat(strings.TrimSuffix(path, ".story.json") + ".tsx")
		if err != nil {
			return nil, fmt.Errorf("%s: read page source: %w", id, err)
		}
		if !source.Mode().IsRegular() || source.Size() == 0 {
			return nil, fmt.Errorf("%s: page source must be a nonempty regular file", id)
		}
		stories, err := json.Marshal(contract.Stories)
		if err != nil {
			return nil, err
		}
		subjects = append(subjects, PageStorySubject{
			Projection: ComponentStory{ID: id, ComponentID: id, LibraryID: id, Version: PageStoryVersion, SchemaVersion: contract.SchemaVersion, Kind: StoryKindPage, Title: contract.Title, ArgsJSON: `{"fields":[]}`, EnvironmentJSON: `{"fixtures":[]}`, StoriesJSON: string(stories), ContractJSON: string(raw), SourcePath: path},
			Contract:   contract, Revision: revision,
		})
	}
	return subjects, nil
}

func LoadPageStory(ctx context.Context, scenarioRoot, id string) (PageStorySubject, error) {
	subjects, err := LoadPageStories(ctx, scenarioRoot)
	if err != nil {
		return PageStorySubject{}, err
	}
	for _, subject := range subjects {
		if subject.Projection.ID == id {
			return subject, nil
		}
	}
	return PageStorySubject{}, fmt.Errorf("page story %q not found", id)
}
