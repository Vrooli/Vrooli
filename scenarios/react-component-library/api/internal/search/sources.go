package search

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	searchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/search"
	"react-component-library/internal/catalogcoverage"
)

// catalogDocuments projects the authored catalog. Page templates are indexed as
// page types; every other catalog kind is an asset. The catalog remains the
// owner of names, descriptions, capabilities and regions — this function only
// chooses the searchable shape.
func (s *Service) catalogDocuments(ctx context.Context, now time.Time) ([]Document, []string, error) {
	assets, err := catalogcoverage.LoadCatalogContext(ctx, filepath.Join(s.scenarioRoot, "catalog"))
	if err != nil {
		return nil, nil, fmt.Errorf("load catalog for search: %w", err)
	}
	docs := make([]Document, 0, len(assets))
	warnings := make([]string, 0)
	for _, asset := range assets {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}
		kind := searchv1.SearchKind_SEARCH_KIND_ASSET
		if asset.Kind == "page-template" {
			kind = searchv1.SearchKind_SEARCH_KIND_PAGE_TYPE
		}
		availability := searchv1.SearchAvailability_SEARCH_AVAILABILITY_PLANNED
		var badges []string
		if s.availability != nil {
			built, label, resolveErr := s.availability(ctx, asset.ID)
			if resolveErr != nil {
				// One unavailable resolver must not take the whole index down.
				// The row is still indexed, but it is honestly marked unavailable
				// and the degradation is recorded in search_state warnings.
				warnings = append(warnings, fmt.Sprintf("availability %s: %v", asset.ID, resolveErr))
				availability = searchv1.SearchAvailability_SEARCH_AVAILABILITY_UNAVAILABLE
				if label == "" {
					label = "availability unknown"
				}
			} else if built {
				availability = searchv1.SearchAvailability_SEARCH_AVAILABILITY_AVAILABLE
			}
			if label != "" {
				badges = append(badges, label)
			}
		}
		docs = append(docs, Document{
			ID:           DocumentID(kind, asset.ID),
			SubjectID:    asset.ID,
			Kind:         kind,
			Title:        asset.Name,
			Subtitle:     joinNonEmpty("/", asset.Domain, asset.Kind),
			Body:         truncateBody(catalogBody(asset), 8000),
			Href:         catalogHref(kind, asset.Name),
			Availability: availability,
			Badges:       badges,
			UpdatedAt:    now,
		})
	}
	return docs, warnings, nil
}

func catalogBody(asset catalogcoverage.Asset) string {
	parts := []string{asset.Description}
	if len(asset.Capabilities) > 0 {
		parts = append(parts, "Capabilities: "+strings.Join(asset.Capabilities, ", "))
	}
	if len(asset.Regions) > 0 {
		parts = append(parts, "Regions: "+strings.Join(asset.Regions, ", "))
	}
	if len(asset.States) > 0 {
		parts = append(parts, "States: "+strings.Join(asset.States, ", "))
	}
	parts = append(parts, "Catalog id: "+asset.ID)
	return joinNonEmpty("\n", parts...)
}

func catalogHref(kind searchv1.SearchKind, name string) string {
	if kind == searchv1.SearchKind_SEARCH_KIND_PAGE_TYPE {
		return "/library?kind=page-template&q=" + url.QueryEscape(name)
	}
	return "/library?q=" + url.QueryEscape(name)
}

type workspaceRecord struct {
	id, name, provenance, tags, bindingScenario, bindingPage, updatedAt string
}

type pageRecord struct {
	id, workspaceID, name, route, pageTypeID, spec, updatedAt string
}

type displayRecord struct {
	id, pageID, name, device string
	width, height            int
}

type stateRecord struct {
	id, displayID, name, status, args, scenarioData, updatedAt string
}

// workspaceDocuments projects the workbench DB tables when they exist. The
// workspace package owns these tables; search must remain buildable against a
// database from before that domain shipped, so existence is discovered at
// runtime instead of assumed. Missing tables mean "source not present", not an
// error; a present table with a missing identity column is a real contract
// mismatch and is reported.
func (s *Service) workspaceDocuments(ctx context.Context, tx *sql.Tx, now time.Time) ([]Document, error) {
	var docs []Document

	workspaces, err := readWorkspaces(ctx, tx)
	if err != nil {
		return nil, err
	}
	for _, workspace := range workspaces {
		title := textOr(workspace.name, workspace.id)
		body := joinNonEmpty("\n",
			workspace.name,
			workspace.tags,
			workspace.bindingScenario,
			workspace.bindingPage,
			"Workspace id: "+workspace.id,
		)
		var badges []string
		if workspace.provenance != "" {
			badges = append(badges, workspace.provenance)
		}
		docs = append(docs, Document{
			ID:           DocumentID(searchv1.SearchKind_SEARCH_KIND_WORKSPACE, workspace.id),
			SubjectID:    workspace.id,
			Kind:         searchv1.SearchKind_SEARCH_KIND_WORKSPACE,
			Title:        title,
			Subtitle:     "Workspace",
			Body:         truncateBody(body, 8000),
			Href:         workspaceHref(workspace.id),
			Availability: searchv1.SearchAvailability_SEARCH_AVAILABILITY_AVAILABLE,
			Badges:       badges,
			Relations:    []string{"workspace:" + workspace.id},
			UpdatedAt:    parseTimeOr(workspace.updatedAt, now),
		})
	}

	pages, err := readPages(ctx, tx)
	if err != nil {
		return nil, err
	}
	workspaceByID := make(map[string]workspaceRecord, len(workspaces))
	for _, workspace := range workspaces {
		workspaceByID[workspace.id] = workspace
	}
	pageByID := make(map[string]pageRecord, len(pages))
	for _, page := range pages {
		pageByID[page.id] = page
		workspace := workspaceByID[page.workspaceID]
		title := textOr(page.name, textOr(page.route, page.id))
		var badges []string
		if page.pageTypeID != "" {
			badges = append(badges, "page type: "+page.pageTypeID)
		}
		docs = append(docs, Document{
			ID:        DocumentID(searchv1.SearchKind_SEARCH_KIND_PAGE, page.id),
			SubjectID: page.id,
			Kind:      searchv1.SearchKind_SEARCH_KIND_PAGE,
			Title:     title,
			Subtitle:  joinNonEmpty(" · ", "Page", workspace.name),
			Body: truncateBody(joinNonEmpty("\n",
				page.name,
				page.route,
				page.pageTypeID,
				page.spec,
				"Page id: "+page.id,
			), 8000),
			Href:         pageHref(page.workspaceID, page.id),
			Availability: searchv1.SearchAvailability_SEARCH_AVAILABILITY_AVAILABLE,
			Badges:       badges,
			Relations:    []string{"workspace:" + page.workspaceID, "page:" + page.id},
			UpdatedAt:    parseTimeOr(page.updatedAt, now),
		})
	}

	displays, err := readDisplays(ctx, tx)
	if err != nil {
		return nil, err
	}
	displayByID := make(map[string]displayRecord, len(displays))
	for _, display := range displays {
		displayByID[display.id] = display
		page := pageByID[display.pageID]
		workspace := workspaceByID[page.workspaceID]
		title := textOr(display.name, textOr(displayTitle(display.device), display.id))
		body := joinNonEmpty("\n",
			display.name,
			display.device,
			displayDimensions(display.width, display.height),
			page.name,
			workspace.name,
			"Display id: "+display.id,
		)
		docs = append(docs, Document{
			ID:           DocumentID(searchv1.SearchKind_SEARCH_KIND_DISPLAY, display.id),
			SubjectID:    display.id,
			Kind:         searchv1.SearchKind_SEARCH_KIND_DISPLAY,
			Title:        title,
			Subtitle:     joinNonEmpty(" · ", "Display", workspace.name, page.name),
			Body:         truncateBody(body, 8000),
			Href:         displayHref(page.workspaceID, page.id, display.id),
			Availability: searchv1.SearchAvailability_SEARCH_AVAILABILITY_AVAILABLE,
			Relations:    []string{"workspace:" + page.workspaceID, "page:" + page.id, "display:" + display.id},
			UpdatedAt:    now,
		})
	}

	states, err := readStates(ctx, tx)
	if err != nil {
		return nil, err
	}
	for _, state := range states {
		display := displayByID[state.displayID]
		page := pageByID[display.pageID]
		workspace := workspaceByID[page.workspaceID]
		// A state's address is its identity: two displays both have a "Default"
		// state, and a bare name would make them indistinguishable in results.
		title := joinNonEmpty(" · ", page.name, displayTitle(display.device), state.name)
		if title == "" {
			title = "State " + state.id
		}
		var badges []string
		if state.status != "" {
			badges = append(badges, state.status)
		}
		docs = append(docs, Document{
			ID:        DocumentID(searchv1.SearchKind_SEARCH_KIND_STATE, state.id),
			SubjectID: state.id,
			Kind:      searchv1.SearchKind_SEARCH_KIND_STATE,
			Title:     title,
			Subtitle:  joinNonEmpty(" · ", "State", workspace.name),
			Body: truncateBody(joinNonEmpty("\n",
				state.name,
				state.status,
				state.args,
				state.scenarioData,
				workspace.name,
				page.name,
				display.device,
				"State id: "+state.id,
			), 8000),
			Href:         stateHref(page.workspaceID, page.id, display.id, state.id),
			Availability: searchv1.SearchAvailability_SEARCH_AVAILABILITY_AVAILABLE,
			Badges:       badges,
			Relations:    []string{"workspace:" + page.workspaceID, "page:" + page.id, "display:" + display.id, "state:" + state.id},
			UpdatedAt:    parseTimeOr(state.updatedAt, now),
		})
	}
	return docs, nil
}

func readWorkspaces(ctx context.Context, tx *sql.Tx) ([]workspaceRecord, error) {
	exists, cols, err := tableShape(ctx, tx, "workspaces")
	if err != nil || !exists {
		return nil, err
	}
	if !cols["id"] {
		return nil, errors.New("workspaces table is missing its id column")
	}
	query := `SELECT ` + strings.Join([]string{
		stringColumn(cols, "id"),
		stringColumn(cols, "name"),
		stringColumn(cols, "provenance"),
		stringColumn(cols, "tags_json"),
		stringColumn(cols, "binding_scenario"),
		stringColumn(cols, "binding_page"),
		stringColumn(cols, "updated_at"),
	}, ", ") + ` FROM workspaces ORDER BY id`
	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("read workspaces for search: %w", err)
	}
	defer rows.Close()
	out := make([]workspaceRecord, 0)
	for rows.Next() {
		var record workspaceRecord
		if err := rows.Scan(&record.id, &record.name, &record.provenance, &record.tags,
			&record.bindingScenario, &record.bindingPage, &record.updatedAt); err != nil {
			return nil, fmt.Errorf("scan workspace for search: %w", err)
		}
		if strings.TrimSpace(record.id) == "" {
			continue
		}
		out = append(out, record)
	}
	return out, rows.Err()
}

func readPages(ctx context.Context, tx *sql.Tx) ([]pageRecord, error) {
	exists, cols, err := tableShape(ctx, tx, "workspace_pages")
	if err != nil || !exists {
		return nil, err
	}
	if !cols["id"] || !cols["workspace_id"] {
		return nil, errors.New("workspace_pages table is missing its id or workspace_id column")
	}
	query := `SELECT ` + strings.Join([]string{
		stringColumn(cols, "id"),
		stringColumn(cols, "workspace_id"),
		stringColumn(cols, "name"),
		stringColumn(cols, "route"),
		stringColumn(cols, "page_type_id"),
		stringColumn(cols, "spec_json"),
		stringColumn(cols, "updated_at"),
	}, ", ") + ` FROM workspace_pages ORDER BY id`
	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("read workspace pages for search: %w", err)
	}
	defer rows.Close()
	out := make([]pageRecord, 0)
	for rows.Next() {
		var record pageRecord
		if err := rows.Scan(&record.id, &record.workspaceID, &record.name, &record.route,
			&record.pageTypeID, &record.spec, &record.updatedAt); err != nil {
			return nil, fmt.Errorf("scan workspace page for search: %w", err)
		}
		if strings.TrimSpace(record.id) == "" {
			continue
		}
		out = append(out, record)
	}
	return out, rows.Err()
}

func readDisplays(ctx context.Context, tx *sql.Tx) ([]displayRecord, error) {
	exists, cols, err := tableShape(ctx, tx, "workspace_displays")
	if err != nil || !exists {
		return nil, err
	}
	if !cols["id"] || !cols["page_id"] {
		return nil, errors.New("workspace_displays table is missing its id or page_id column")
	}
	query := `SELECT ` + strings.Join([]string{
		stringColumn(cols, "id"),
		stringColumn(cols, "page_id"),
		stringColumn(cols, "name"),
		stringColumn(cols, "device"),
		intColumn(cols, "width"),
		intColumn(cols, "height"),
	}, ", ") + ` FROM workspace_displays ORDER BY id`
	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("read workspace displays for search: %w", err)
	}
	defer rows.Close()
	out := make([]displayRecord, 0)
	for rows.Next() {
		var record displayRecord
		if err := rows.Scan(&record.id, &record.pageID, &record.name, &record.device,
			&record.width, &record.height); err != nil {
			return nil, fmt.Errorf("scan workspace display for search: %w", err)
		}
		if strings.TrimSpace(record.id) == "" {
			continue
		}
		out = append(out, record)
	}
	return out, rows.Err()
}

func readStates(ctx context.Context, tx *sql.Tx) ([]stateRecord, error) {
	exists, cols, err := tableShape(ctx, tx, "workspace_states")
	if err != nil || !exists {
		return nil, err
	}
	if !cols["id"] || !cols["display_id"] {
		return nil, errors.New("workspace_states table is missing its id or display_id column")
	}
	query := `SELECT ` + strings.Join([]string{
		stringColumn(cols, "id"),
		stringColumn(cols, "display_id"),
		stringColumn(cols, "name"),
		stringColumn(cols, "status"),
		stringColumn(cols, "args_json"),
		stringColumn(cols, "scenario_data_json"),
		stringColumn(cols, "updated_at"),
	}, ", ") + ` FROM workspace_states ORDER BY id`
	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("read workspace states for search: %w", err)
	}
	defer rows.Close()
	out := make([]stateRecord, 0)
	for rows.Next() {
		var record stateRecord
		if err := rows.Scan(&record.id, &record.displayID, &record.name, &record.status,
			&record.args, &record.scenarioData, &record.updatedAt); err != nil {
			return nil, fmt.Errorf("scan workspace state for search: %w", err)
		}
		if strings.TrimSpace(record.id) == "" {
			continue
		}
		out = append(out, record)
	}
	return out, rows.Err()
}

// findingDocuments indexes the server-owned findings projection when it exists.
// Findings are evidence, not catalog assets: the href points at the Review queue
// so a reader can act on the result rather than land in Library.
func (s *Service) findingDocuments(ctx context.Context, tx *sql.Tx, now time.Time) ([]Document, error) {
	exists, cols, err := tableShape(ctx, tx, "findings")
	if err != nil || !exists {
		return nil, err
	}
	if !cols["id"] {
		return nil, errors.New("findings table is missing its id column")
	}
	query := `SELECT ` + strings.Join([]string{
		stringColumn(cols, "id"),
		stringColumn(cols, "subject_kind"),
		stringColumn(cols, "subject_id"),
		stringColumn(cols, "gate"),
		stringColumn(cols, "severity"),
		stringColumn(cols, "evidence_json"),
		stringColumn(cols, "status"),
		stringColumn(cols, "owner"),
		stringColumn(cols, "created_at"),
	}, ", ") + ` FROM findings
		WHERE status = 'open' AND severity IN ('blocker', 'high')
		ORDER BY CASE severity WHEN 'blocker' THEN 0 ELSE 1 END, created_at DESC, id
		LIMIT 200`
	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("read findings for search: %w", err)
	}
	defer rows.Close()
	docs := make([]Document, 0)
	for rows.Next() {
		var (
			id, subjectKind, subjectID, gate, severity string
			evidenceJSON, status, owner, createdAt     string
		)
		if err := rows.Scan(&id, &subjectKind, &subjectID, &gate, &severity,
			&evidenceJSON, &status, &owner, &createdAt); err != nil {
			return nil, fmt.Errorf("scan finding for search: %w", err)
		}
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		title := joinNonEmpty(" · ", subjectID, gate)
		if title == "" {
			title = "Finding " + id
		}
		var badges []string
		if severity != "" {
			badges = append(badges, severity)
		}
		if status != "" {
			badges = append(badges, status)
		}
		var relations []string
		if subjectKind != "" || subjectID != "" {
			relations = append(relations, joinNonEmpty(":", subjectKind, subjectID))
		}
		docs = append(docs, Document{
			ID:        DocumentID(searchv1.SearchKind_SEARCH_KIND_EVIDENCE, id),
			SubjectID: id,
			Kind:      searchv1.SearchKind_SEARCH_KIND_EVIDENCE,
			Title:     title,
			Subtitle:  joinNonEmpty(" · ", severity, status),
			Body: truncateBody(joinNonEmpty("\n",
				gate, severity, status, subjectKind, subjectID, owner, evidenceJSON,
				"Finding id: "+id,
			), 8000),
			Href:         "/review?finding=" + strings.ReplaceAll(id, " ", "%20"),
			Availability: searchv1.SearchAvailability_SEARCH_AVAILABILITY_AVAILABLE,
			Badges:       badges,
			Relations:    relations,
			UpdatedAt:    parseTimeOr(createdAt, now),
		})
	}
	return docs, rows.Err()
}

// commandDocuments is a static manifest owned here on purpose. Commands are
// generated in code today; when a command registry lands, this function becomes
// its projection and the ids stay stable.
func commandDocuments(now time.Time) []Document {
	type command struct {
		id, label, group, shortcut, href, description string
	}
	commands := []command{
		{"command.start", "Start", "Navigate", "", "/", "Search and browse the workbench front door."},
		{"command.library", "Library", "Navigate", "", "/library", "Browse assets and page types."},
		{"command.design", "Design", "Navigate", "", "/design", "Open workspaces, pages, displays and states."},
		{"command.review", "Review", "Navigate", "", "/review", "Triage findings and evidence."},
		{"command.stats", "Stats", "Navigate", "", "/stats", "Read quality, activity and service health."},
		{"command.new", "New asset", "Create", "", "/new", "Create, extract or adopt an asset."},
		{"command.settings", "Settings", "Settings", "", "/settings", "Open workbench settings."},
		{"command.search", "Search", "Search", "⌘K", "/", "Focus the global search surface."},
		{"command.reindex-search", "Reindex search", "Search", "", "/settings", "Rebuild the unified search index from its owned sources."},
	}
	docs := make([]Document, 0, len(commands))
	for _, item := range commands {
		var badges []string
		if item.shortcut != "" {
			badges = append(badges, item.shortcut)
		}
		docs = append(docs, Document{
			ID:           DocumentID(searchv1.SearchKind_SEARCH_KIND_COMMAND, item.id),
			SubjectID:    item.id,
			Kind:         searchv1.SearchKind_SEARCH_KIND_COMMAND,
			Title:        item.label,
			Subtitle:     joinNonEmpty(" · ", "Command", item.group),
			Body:         joinNonEmpty("\n", item.label, item.group, item.description, item.href),
			Href:         item.href,
			Availability: searchv1.SearchAvailability_SEARCH_AVAILABILITY_AVAILABLE,
			Badges:       badges,
			Relations:    []string{"command:" + item.id},
			UpdatedAt:    now,
		})
	}
	sort.Slice(docs, func(i, j int) bool { return docs[i].SubjectID < docs[j].SubjectID })
	return docs
}

// tableShape returns whether a table exists and which columns it declares.
// Column discovery is defensive because the workspace domain is landing beside
// this projection: a table can exist with an older subset of the documented
// columns during the migration, and search should index what is present rather
// than fail on the absent optional ones.
func tableShape(ctx context.Context, tx *sql.Tx, table string) (bool, map[string]bool, error) {
	var name string
	err := tx.QueryRowContext(ctx, `SELECT name FROM sqlite_master WHERE type IN ('table','view') AND name = ?`, table).Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil, nil
	}
	if err != nil {
		return false, nil, fmt.Errorf("look up table %s: %w", table, err)
	}
	rows, err := tx.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%q)", table))
	if err != nil {
		return false, nil, fmt.Errorf("read columns for %s: %w", table, err)
	}
	defer rows.Close()
	cols := map[string]bool{}
	for rows.Next() {
		var (
			cid          int
			columnName   string
			columnType   string
			notNull      int
			defaultValue sql.NullString
			primaryKey   int
		)
		if err := rows.Scan(&cid, &columnName, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			return false, nil, fmt.Errorf("scan columns for %s: %w", table, err)
		}
		cols[columnName] = true
	}
	if err := rows.Err(); err != nil {
		return false, nil, fmt.Errorf("read columns for %s: %w", table, err)
	}
	return true, cols, nil
}

func stringColumn(cols map[string]bool, name string) string {
	if cols[name] {
		return "COALESCE(" + name + ", '')"
	}
	return "''"
}

func intColumn(cols map[string]bool, name string) string {
	if cols[name] {
		return "COALESCE(" + name + ", 0)"
	}
	return "0"
}

// Workspace URLs address the workspace studio, not the legacy
// /design/:scenario/:page route, which belongs to a different store.
func workspaceHref(workspaceID string) string {
	return "/design?workspace=" + url.QueryEscape(workspaceID)
}

func pageHref(workspaceID, pageID string) string {
	return workspaceHref(workspaceID) + "&page=" + url.QueryEscape(pageID)
}

func displayHref(workspaceID, pageID, displayID string) string {
	return pageHref(workspaceID, pageID) + "&display=" + url.QueryEscape(displayID)
}

func stateHref(workspaceID, pageID, displayID, stateID string) string {
	return pageHref(workspaceID, pageID) + "&display=" + url.QueryEscape(displayID) + "&state=" + url.QueryEscape(stateID)
}

func displayTitle(device string) string {
	device = strings.TrimSpace(device)
	if device == "" {
		return "Display"
	}
	runes := []rune(device)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes) + " display"
}

func displayDimensions(width, height int) string {
	if width <= 0 && height <= 0 {
		return ""
	}
	return fmt.Sprintf("%d×%d", width, height)
}

func joinNonEmpty(separator string, parts ...string) string {
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return strings.Join(out, separator)
}

func textOr(value, fallback string) string {
	if value = strings.TrimSpace(value); value != "" {
		return value
	}
	return fallback
}

func parseTimeOr(value string, fallback time.Time) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.UTC()
		}
	}
	return fallback
}
