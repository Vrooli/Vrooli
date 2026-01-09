package main

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	_ "embed"

	"github.com/lib/pq"
	"golang.org/x/text/unicode/runenames"
)

//go:embed data/Blocks.txt
var blocksFile string

//go:embed data/DerivedAge.txt
var derivedAgeFile string

//go:embed data/UnicodeData.txt
var unicodeDataFile string

// ensureUnicodeData bootstraps the unicode tables when the database is empty
func (api *API) ensureUnicodeData(ctx context.Context) error {
	const minimumCharacters = 5000

	if ctx == nil {
		ctx = context.Background()
	}

	var count int
	if err := api.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM characters").Scan(&count); err != nil {
		return fmt.Errorf("failed counting characters: %w", err)
	}

	if count >= minimumCharacters {
		return nil
	}

	log.Printf("Populating unicode dataset (current count %d)", count)

	parsedBlocks, err := parseBlockDefinitions(blocksFile)
	if err != nil {
		return fmt.Errorf("parse blocks: %w", err)
	}

	unicodeMeta := parseUnicodeMetadata(unicodeDataFile)

	ageRanges, err := parseDerivedAge(derivedAgeFile)
	if err != nil {
		return fmt.Errorf("parse derived age: %w", err)
	}

	inserted, err := api.populateUnicodeCharacters(ctx, parsedBlocks, unicodeMeta, ageRanges)
	if err != nil {
		return err
	}

	log.Printf("Unicode dataset populated with %d characters", inserted)
	return nil
}

type blockRange struct {
	Start rune
	End   rune
	Name  string
}

type ageRange struct {
	Start   rune
	End     rune
	Version string
}

type unicodeRecord struct {
	Name     string
	Category string
}

func parseBlockDefinitions(data string) ([]blockRange, error) {
	scanner := bufio.NewScanner(strings.NewReader(data))
	blocks := make([]blockRange, 0, 350)

	for scanner.Scan() {
		line := scanner.Text()
		if idx := strings.Index(line, "#"); idx >= 0 {
			line = line[:idx]
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, ";")
		if len(parts) != 2 {
			continue
		}

		rangePart := strings.TrimSpace(parts[0])
		name := strings.TrimSpace(parts[1])

		rangePieces := strings.Split(rangePart, "..")
		if len(rangePieces) != 2 {
			continue
		}

		start, err := strconv.ParseUint(rangePieces[0], 16, 32)
		if err != nil {
			return nil, fmt.Errorf("parse block start %q: %w", rangePieces[0], err)
		}
		end, err := strconv.ParseUint(rangePieces[1], 16, 32)
		if err != nil {
			return nil, fmt.Errorf("parse block end %q: %w", rangePieces[1], err)
		}

		blocks = append(blocks, blockRange{Start: rune(start), End: rune(end), Name: name})
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	sort.Slice(blocks, func(i, j int) bool {
		return blocks[i].Start < blocks[j].Start
	})

	return blocks, nil
}

func parseDerivedAge(data string) ([]ageRange, error) {
	scanner := bufio.NewScanner(strings.NewReader(data))
	ranges := make([]ageRange, 0, 1024)

	for scanner.Scan() {
		line := scanner.Text()
		if idx := strings.Index(line, "#"); idx >= 0 {
			line = line[:idx]
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, ";")
		if len(parts) != 2 {
			continue
		}

		rangePart := strings.TrimSpace(parts[0])
		version := strings.TrimSpace(parts[1])

		var startVal, endVal uint64
		var err error

		if strings.Contains(rangePart, "..") {
			rangePieces := strings.Split(rangePart, "..")
			if len(rangePieces) != 2 {
				continue
			}
			startVal, err = strconv.ParseUint(rangePieces[0], 16, 32)
			if err != nil {
				return nil, fmt.Errorf("parse age start %q: %w", rangePieces[0], err)
			}
			endVal, err = strconv.ParseUint(rangePieces[1], 16, 32)
			if err != nil {
				return nil, fmt.Errorf("parse age end %q: %w", rangePieces[1], err)
			}
		} else {
			startVal, err = strconv.ParseUint(rangePart, 16, 32)
			if err != nil {
				return nil, fmt.Errorf("parse age value %q: %w", rangePart, err)
			}
			endVal = startVal
		}

		ranges = append(ranges, ageRange{Start: rune(startVal), End: rune(endVal), Version: version})
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	sort.Slice(ranges, func(i, j int) bool {
		return ranges[i].Start < ranges[j].Start
	})

	return ranges, nil
}

func parseUnicodeMetadata(data string) map[rune]unicodeRecord {
	scanner := bufio.NewScanner(strings.NewReader(data))
	result := make(map[rune]unicodeRecord, 150000)

	for scanner.Scan() {
		line := scanner.Text()
		if idx := strings.Index(line, "#"); idx >= 0 {
			line = line[:idx]
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, ";")
		if len(parts) < 3 {
			continue
		}

		codeHex := strings.TrimSpace(parts[0])
		name := strings.TrimSpace(parts[1])
		category := strings.TrimSpace(parts[2])

		if strings.Contains(name, "First>") || strings.Contains(name, "Last>") {
			continue
		}

		codeVal, err := strconv.ParseUint(codeHex, 16, 32)
		if err != nil {
			continue
		}

		result[rune(codeVal)] = unicodeRecord{Name: name, Category: category}
	}

	return result
}

var categoryOrder []string
var scriptOrder []string

func init() {
	for key := range unicode.Categories {
		if len(key) == 2 {
			categoryOrder = append(categoryOrder, key)
		}
	}
	sort.Strings(categoryOrder)

	for key := range unicode.Scripts {
		scriptOrder = append(scriptOrder, key)
	}
	sort.Strings(scriptOrder)
}

func determineCategory(r rune, meta map[rune]unicodeRecord) string {
	if rec, ok := meta[r]; ok && rec.Category != "" {
		return rec.Category
	}

	for _, cat := range categoryOrder {
		if table := unicode.Categories[cat]; table != nil && unicode.In(r, table) {
			return cat
		}
	}

	return "Cn"
}

func determineScript(r rune) string {
	for _, script := range scriptOrder {
		table := unicode.Scripts[script]
		if table != nil && unicode.In(r, table) {
			return script
		}
	}
	return "Unknown"
}

func findBlockName(r rune, blocks []blockRange) string {
	idx := sort.Search(len(blocks), func(i int) bool {
		return blocks[i].End >= r
	})
	if idx < len(blocks) {
		block := blocks[idx]
		if r >= block.Start && r <= block.End {
			return block.Name
		}
	}
	if len(blocks) > 0 && r < blocks[0].Start {
		return blocks[0].Name
	}
	return "No_Block"
}

func formatCodepoint(r rune) string {
	if r <= 0xFFFF {
		return fmt.Sprintf("U+%04X", r)
	}
	return fmt.Sprintf("U+%06X", r)
}

func formatHTML(r rune) string {
	return fmt.Sprintf("&#%d;", r)
}

func formatCSSContent(r rune) string {
	hex := fmt.Sprintf("%X", r)
	if len(hex) < 4 {
		hex = strings.Repeat("0", 4-len(hex)) + hex
	}
	return "\\" + hex
}

func buildPropertiesJSON(category, script, version, block string) string {
	props := map[string]string{
		"general_category": category,
		"script":           script,
		"unicode_version":  version,
		"block":            block,
	}
	payload, err := json.Marshal(props)
	if err != nil {
		log.Printf("failed to marshal properties for %s: %v", category, err)
		return "{}"
	}
	return string(payload)
}

func (api *API) populateUnicodeCharacters(ctx context.Context, blocks []blockRange, meta map[rune]unicodeRecord, ages []ageRange) (int, error) {
	tx, err := api.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}

	if _, err := tx.ExecContext(ctx, "DELETE FROM characters"); err != nil {
		_ = tx.Rollback()
		return 0, fmt.Errorf("truncate characters: %w", err)
	}

	if err := api.upsertBlocks(ctx, tx, blocks); err != nil {
		_ = tx.Rollback()
		return 0, err
	}

	stmt, err := tx.PrepareContext(ctx, pq.CopyIn(
		"characters",
		"codepoint", "decimal", "name", "category", "block", "unicode_version",
		"description", "html_entity", "css_content", "properties",
	))
	if err != nil {
		_ = tx.Rollback()
		return 0, fmt.Errorf("prepare copy: %w", err)
	}

	inserted := 0
	for _, age := range ages {
		for code := age.Start; code <= age.End; code++ {
			category := determineCategory(code, meta)
			if category == "Cn" {
				continue
			}
			if category == "Cs" {
				continue
			}
			name := runenames.Name(code)
			if name == "" {
				if rec, ok := meta[code]; ok && rec.Name != "" {
					name = strings.ToUpper(rec.Name)
				} else {
					// Fallback placeholder name
					name = fmt.Sprintf("UNICODE CHARACTER U+%04X", code)
				}
			}

			blockName := findBlockName(code, blocks)
			script := determineScript(code)

			propsJSON := buildPropertiesJSON(category, script, age.Version, blockName)
			codepointID := formatCodepoint(code)
			htmlEntity := formatHTML(code)
			cssContent := formatCSSContent(code)

			if _, err := stmt.ExecContext(ctx,
				codepointID,
				int(code),
				name,
				category,
				blockName,
				age.Version,
				nil,
				htmlEntity,
				cssContent,
				propsJSON,
			); err != nil {
				_ = stmt.Close()
				_ = tx.Rollback()
				return 0, fmt.Errorf("copy exec for %s: %w", codepointID, err)
			}

			inserted++
		}
	}

	if _, err := stmt.ExecContext(ctx); err != nil {
		_ = stmt.Close()
		_ = tx.Rollback()
		return 0, fmt.Errorf("finalize copy: %w", err)
	}

	if err := stmt.Close(); err != nil {
		_ = tx.Rollback()
		return 0, fmt.Errorf("close copy: %w", err)
	}

	if _, err := tx.ExecContext(ctx, "ANALYZE characters"); err != nil {
		_ = tx.Rollback()
		return 0, fmt.Errorf("analyze characters: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "ANALYZE character_blocks"); err != nil {
		_ = tx.Rollback()
		return 0, fmt.Errorf("analyze blocks: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit unicode population: %w", err)
	}

	// Give Postgres a moment to make data visible for subsequent queries.
	time.Sleep(100 * time.Millisecond)

	return inserted, nil
}

func (api *API) upsertBlocks(ctx context.Context, tx *sql.Tx, blocks []blockRange) error {
	if _, err := tx.ExecContext(ctx, "DELETE FROM character_blocks"); err != nil {
		return fmt.Errorf("clear blocks: %w", err)
	}

	stmt, err := tx.PrepareContext(ctx, pq.CopyIn(
		"character_blocks",
		"name", "start_codepoint", "end_codepoint", "description",
	))
	if err != nil {
		return fmt.Errorf("prepare block copy: %w", err)
	}

	for _, block := range blocks {
		description := fmt.Sprintf("Unicode block %s", block.Name)
		if _, err := stmt.ExecContext(ctx,
			block.Name,
			int(block.Start),
			int(block.End),
			description,
		); err != nil {
			_ = stmt.Close()
			return fmt.Errorf("insert block %s: %w", block.Name, err)
		}
	}
	if _, err := stmt.ExecContext(ctx, "No_Block", 0, 0, "Code points without an assigned block"); err != nil {
		_ = stmt.Close()
		return fmt.Errorf("insert No_Block entry: %w", err)
	}

	if _, err := stmt.ExecContext(ctx); err != nil {
		_ = stmt.Close()
		return fmt.Errorf("finalize block copy: %w", err)
	}

	if err := stmt.Close(); err != nil {
		return fmt.Errorf("close block copy: %w", err)
	}

	return nil
}
