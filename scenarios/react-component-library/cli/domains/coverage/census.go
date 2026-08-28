package coverage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type censusReport struct {
	GeneratedAt       string           `json:"generatedAt"`
	StoryContracts    storyCensus      `json:"storyContracts"`
	StoryCorruption   storyCensus      `json:"storyCorruption"`
	Composition       map[string]int   `json:"composition"`
	Versions          int              `json:"versions"`
	StoryFiles        int              `json:"storyFiles"`
	Stories           int              `json:"stories"`
	Adoptions         adoptionCensus   `json:"adoptions"`
	AdoptionRecords   int              `json:"adoptionRecords"`
	AdoptionFiles     int              `json:"adoptionFiles"`
	Obligations       obligationCensus `json:"obligations"`
	Styling           stylingCensus    `json:"styling"`
	EvidenceFreshness freshnessCensus  `json:"evidenceFreshness"`
}

type storyCensus struct {
	CorruptCount int      `json:"corruptCount"`
	CorruptFiles int      `json:"corruptFiles"`
	Examples     []string `json:"examples,omitempty"`
}
type adoptionCensus struct {
	Records    int            `json:"records"`
	Files      int            `json:"files"`
	ByMode     map[string]int `json:"byMode"`
	ByLocal    map[string]int `json:"byLocalStatus"`
	ByLibrary  map[string]int `json:"byLibraryVersionStatus"`
	ByFork     map[string]int `json:"byForkStatus"`
	ByScenario map[string]int `json:"byScenario"`
}
type obligationCensus struct {
	TranslateAssignments int `json:"translateAssignments"`
	SelectorFiles        int `json:"selectorFiles"`
	SelectorAdopters     int `json:"selectorAdopters"`
}
type stylingCensus struct {
	InlineStyleFiles int `json:"inlineStyleFiles"`
	MergeFiles       int `json:"mergeFiles"`
}
type freshnessCensus struct {
	Fresh int `json:"fresh"`
	Stale int `json:"stale"`
	Never int `json:"never"`
	Total int `json:"total"`
}

func runCensus(args []string) error {
	for _, arg := range args {
		if arg != "--json" {
			return fmt.Errorf("unknown census option: %s", arg)
		}
	}
	report, err := buildCensus(findRepoRoot())
	if err != nil {
		return err
	}
	return printJSON(report)
}

func buildCensus(root string) (censusReport, error) {
	libraryRoot := filepath.Join(root, "scenarios", "react-component-library", "library")
	report := censusReport{
		Composition: map[string]int{"specimen": 0, "harness": 0, "fixture": 0, "frame": 0, "none": 0},
		Adoptions:   adoptionCensus{ByMode: map[string]int{}, ByLocal: map[string]int{}, ByLibrary: map[string]int{}, ByFork: map[string]int{}, ByScenario: map[string]int{}},
	}
	err := filepath.WalkDir(libraryRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if filepath.Base(path) == "story.json" {
			report.StoryFiles++
			var contract struct {
				Args struct {
					Fields []struct {
						Default any `json:"default"`
					} `json:"fields"`
				} `json:"args"`
				Stories []struct {
					Args        any            `json:"args"`
					Composition map[string]any `json:"composition"`
				} `json:"stories"`
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if err := json.Unmarshal(raw, &contract); err != nil {
				return fmt.Errorf("decode %s: %w", path, err)
			}
			report.Stories += len(contract.Stories)
			for _, story := range contract.Stories {
				shape := "none"
				for _, key := range []string{"specimen", "harness", "fixture", "frame"} {
					if _, ok := story.Composition[key]; ok {
						shape = key
						break
					}
				}
				report.Composition[shape]++
			}
			corrupt := 0
			var scan func(any)
			scan = func(value any) {
				switch typed := value.(type) {
				case map[string]any:
					if value, ok := typed["$text"].(string); ok && isHTMLTag(value) {
						corrupt++
						if len(report.StoryContracts.Examples) < 12 {
							report.StoryContracts.Examples = append(report.StoryContracts.Examples, filepath.ToSlash(path))
						}
					}
					for _, child := range typed {
						scan(child)
					}
				case []any:
					for _, child := range typed {
						scan(child)
					}
				}
			}
			scan(contract.Args)
			for _, field := range contract.Args.Fields {
				scan(field.Default)
			}
			for _, story := range contract.Stories {
				scan(story.Args)
			}
			report.StoryContracts.CorruptCount += corrupt
			if corrupt > 0 {
				report.StoryContracts.CorruptFiles++
			}
		}
		// Story previews are authored JSX, but the styling baseline is about
		// importable component sources. Exclude the story harnesses so the
		// result matches the plan's 55-file source population.
		if strings.HasSuffix(path, ".tsx") && filepath.Base(path) != "story.tsx" && strings.Contains(filepath.ToSlash(path), "/library/components/") {
			body, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			text := string(body)
			if strings.Contains(text, "style={{") {
				report.Styling.InlineStyleFiles++
			}
			if strings.Contains(text, "twMerge") || strings.Contains(text, "clsx") {
				report.Styling.MergeFiles++
			}
		}
		return nil
	})
	if err != nil {
		return report, err
	}
	if err := filepath.WalkDir(libraryRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() && filepath.Base(filepath.Dir(path)) == "versions" {
			report.Versions++
			return filepath.SkipDir
		}
		return nil
	}); err != nil {
		return report, err
	}
	dbPath := filepath.Join(root, "scenarios", "react-component-library", "data", "react-component-library.db")
	if !hasAdoptionTables(dbPath) {
		dbPath = filepath.Join(os.Getenv("HOME"), ".vrooli", "data", "vrooli", "react-component-library", "react-component-library.db")
	}
	if err := censusAdoptions(dbPath, &report.Adoptions); err != nil {
		return report, err
	}
	if err := censusEvidenceFreshness(dbPath, libraryRoot, &report.EvidenceFreshness); err != nil {
		return report, err
	}
	report.StoryCorruption = report.StoryContracts
	report.AdoptionRecords = report.Adoptions.Records
	report.AdoptionFiles = report.Adoptions.Files
	if err := censusObligations(root, &report.Obligations); err != nil {
		return report, err
	}
	report.GeneratedAt = "filesystem"
	return report, nil
}

func censusEvidenceFreshness(dbPath, libraryRoot string, out *freshnessCensus) error {
	if _, err := os.Stat(dbPath); err != nil {
		return nil
	}
	rows, err := sqliteJSON(dbPath, "SELECT root_library_id AS library_id, root_version AS version, MAX(created_at) AS created_at FROM component_test_reports GROUP BY root_library_id, root_version;")
	if err != nil {
		return nil
	}
	latest := map[string]time.Time{}
	for _, row := range rows {
		libraryID, _ := row["library_id"].(string)
		version, _ := row["version"].(string)
		created, _ := row["created_at"].(string)
		parsed, parseErr := time.Parse(time.RFC3339Nano, created)
		if parseErr == nil {
			latest[libraryID+"@"+version] = parsed
		}
	}
	return filepath.WalkDir(libraryRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Base(path) != "story.json" {
			return nil
		}
		versionDir := filepath.Dir(path)
		version := filepath.Base(versionDir)
		componentDir := filepath.Dir(filepath.Dir(versionDir))
		manifest, readErr := os.ReadFile(filepath.Join(componentDir, "component.json"))
		if readErr != nil {
			return nil
		}
		var metadata struct {
			LibraryID string `json:"libraryId"`
		}
		if err := json.Unmarshal(manifest, &metadata); err != nil {
			return err
		}
		out.Total++
		created, ok := latest[metadata.LibraryID+"@"+version]
		if !ok {
			out.Never++
			return nil
		}
		info, statErr := entry.Info()
		if statErr != nil {
			return statErr
		}
		if created.After(info.ModTime()) {
			out.Fresh++
		} else {
			out.Stale++
		}
		return nil
	})
}

func hasAdoptionTables(path string) bool {
	if _, err := os.Stat(path); err != nil {
		return false
	}
	rows, err := sqliteJSON(path, "SELECT name FROM sqlite_master WHERE type='table' AND name='adoption_records';")
	return err == nil && len(rows) > 0
}

func censusAdoptions(dbPath string, out *adoptionCensus) error {
	if _, err := os.Stat(dbPath); err != nil {
		return nil
	}
	query := `SELECT 'records' AS k, COUNT(*) AS n, '' AS v FROM adoption_records UNION ALL SELECT 'files', COUNT(*), '' FROM adoption_files UNION ALL SELECT 'local', COUNT(*), local_status FROM adoption_records GROUP BY local_status UNION ALL SELECT 'library', COUNT(*), library_version_status FROM adoption_records GROUP BY library_version_status UNION ALL SELECT 'fork', COUNT(*), fork_status FROM adoption_records GROUP BY fork_status UNION ALL SELECT 'scenario', COUNT(*), scenario FROM adoption_records GROUP BY scenario;`
	rows, err := sqliteJSON(dbPath, query)
	if err != nil {
		return err
	}
	for _, row := range rows {
		k, _ := row["k"].(string)
		n := intNumber(row["n"])
		v, _ := row["v"].(string)
		switch k {
		case "records":
			out.Records = n
		case "files":
			out.Files = n
		case "mode":
			out.ByMode[v] = n
		case "local":
			out.ByLocal[v] = n
		case "library":
			out.ByLibrary[v] = n
		case "fork":
			out.ByFork[v] = n
		case "scenario":
			out.ByScenario[v] = n
		}
	}
	if hasColumn(dbPath, "adoption_records", "mode") {
		modeRows, err := sqliteJSON(dbPath, "SELECT COALESCE(mode,'copied') AS v, COUNT(*) AS n FROM adoption_records GROUP BY COALESCE(mode,'copied');")
		if err != nil {
			return err
		}
		for _, row := range modeRows {
			value, _ := row["v"].(string)
			out.ByMode[value] = intNumber(row["n"])
		}
	} else {
		out.ByMode["copied"] = out.Records
	}
	return nil
}

func hasColumn(path, table, column string) bool {
	rows, err := sqliteJSON(path, "PRAGMA table_info("+table+");")
	if err != nil {
		return false
	}
	for _, row := range rows {
		if value, _ := row["name"].(string); value == column {
			return true
		}
	}
	return false
}

func censusObligations(root string, out *obligationCensus) error {
	translateAssignment := regexp.MustCompile(`(?:globalThis|globalThis\[['"]__vrooliTranslate['"]\])\s*\.?(?:__vrooliTranslate)?\s*[:=]`)
	return filepath.WalkDir(filepath.Join(root, "scenarios"), func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == "node_modules" || entry.Name() == "dist" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".ts") && !strings.HasSuffix(path, ".tsx") {
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		text := string(body)
		// The library's own bridge is the capability being measured, not an
		// adopter installer. Count only assignments outside that scenario.
		if !strings.Contains(filepath.ToSlash(path), "/scenarios/react-component-library/") && translateAssignment.MatchString(text) {
			out.TranslateAssignments++
		}
		// Selector obligations are satisfied by the generated library registry and
		// its composition into the adopter registry. The generated file deliberately
		// contains derived values rather than literal source test IDs, so looking for
		// test-id strings in selectors.ts undercounts every composed adopter.
		if filepath.Base(path) == "selectors.library.ts" &&
			!strings.Contains(filepath.ToSlash(path), "/scenarios/react-component-library/") {
			out.SelectorFiles++
			selectorsPath := filepath.Join(filepath.Dir(path), "selectors.ts")
			selectorsBody, readErr := os.ReadFile(selectorsPath)
			if readErr == nil && strings.Contains(string(selectorsBody), "librarySelectors") {
				out.SelectorAdopters++
			}
		}
		return nil
	})
}

// containsAnyTestID is retained for the focused census unit test and for
// callers that need to compare a selector body against a derived ID set.
func containsAnyTestID(text string, ids map[string]struct{}) bool {
	for id := range ids {
		if strings.Contains(text, id) {
			return true
		}
	}
	return false
}

func sqliteJSON(path, query string) ([]map[string]any, error) {
	cleanPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve sqlite census database: %w", err)
	}
	if strings.ContainsRune(cleanPath, '\x00') || strings.ContainsRune(query, '\x00') {
		return nil, fmt.Errorf("sqlite census input contains a NUL byte")
	}
	if info, statErr := os.Stat(cleanPath); statErr != nil || info.IsDir() {
		if statErr != nil {
			return nil, fmt.Errorf("stat sqlite census database: %w", statErr)
		}
		return nil, fmt.Errorf("sqlite census database is a directory: %s", cleanPath)
	}
	// The query strings are package-owned constants and the path is resolved to
	// an existing regular file before it reaches exec. Keep argv boundaries
	// explicit; no shell is involved in this diagnostic-only census.
	cmd := exec.Command("sqlite3", "-json", cleanPath, query) // #nosec G702 -- validated argv, no shell, package-owned SQL
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("sqlite census: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	if strings.TrimSpace(stdout.String()) == "" {
		return nil, nil
	}
	var rows []map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &rows); err != nil {
		return nil, err
	}
	return rows, nil
}

func intNumber(value any) int {
	if number, ok := value.(float64); ok {
		return int(number)
	}
	return 0
}

func isHTMLTag(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "div", "p", "span", "button", "nav", "section", "article", "ul", "ol", "li":
		return true
	}
	return false
}

func findRepoRoot() string {
	wd, _ := os.Getwd()
	for current := wd; current != filepath.Dir(current); current = filepath.Dir(current) {
		if _, err := os.Stat(filepath.Join(current, "scenarios", "react-component-library")); err == nil {
			return current
		}
	}
	return "."
}

func printJSON(value any) error {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(append(encoded, '\n'))
	return err
}
