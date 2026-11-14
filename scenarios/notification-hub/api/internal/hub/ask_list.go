package hub

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

// AskOrder is how a page of asks is ordered.
type AskOrder string

const (
	// AskOrderNewest is created_at DESC, id DESC, and what "" means.
	AskOrderNewest AskOrder = "newest"
	// AskOrderOldest is created_at ASC, id ASC.
	AskOrderOldest AskOrder = "oldest"
	// AskOrderAttention puts what needs the operator first: escalated asks, then
	// urgency (critical, high, normal, low), then the earliest deadline (an ask
	// with none last), then the newest.
	AskOrderAttention AskOrder = "attention"
	// AskOrderDue puts the earliest deadline first (an ask with none last), then
	// urgency, then the newest.
	AskOrderDue AskOrder = "due"
)

// sortColumn is one column of an order. sql is the expression the query orders
// by, key reads the same value off a record for the cursor, so the two must
// stay together. numeric columns are integers; the rest compare as text.
//
// Deadlines and created_at compare as the text they are stored in. storedTime
// makes that text the same width for every instant, so text order is time order,
// within a second as well. '~' sorts after every digit, so an empty deadline
// sorts last.
type sortColumn struct {
	sql     string
	desc    bool
	numeric bool
	key     func(AskRecord) string
}

var (
	rankColumn = sortColumn{
		sql: `CASE WHEN a.state = 'escalated' THEN 0 ELSE 1 END`, numeric: true,
		key: func(r AskRecord) string {
			if r.State == AskEscalated {
				return "0"
			}
			return "1"
		},
	}
	urgencyColumn = sortColumn{
		sql: `CASE ` + urgencySQL + ` WHEN 'critical' THEN 0 WHEN 'high' THEN 1 WHEN 'low' THEN 3 ELSE 2 END`, numeric: true,
		key: func(r AskRecord) string { return strconv.Itoa(urgencyRank(r.Urgency)) },
	}
	deadlineColumn = sortColumn{
		sql: `CASE WHEN a.deadline = '' THEN '~' ELSE a.deadline END`,
		key: func(r AskRecord) string {
			if r.Deadline == "" {
				return "~"
			}
			return r.Deadline
		},
	}
	createdNewest = sortColumn{sql: `a.created_at`, desc: true, key: func(r AskRecord) string { return r.CreatedAt }}
	idNewest      = sortColumn{sql: `a.id`, desc: true, key: func(r AskRecord) string { return r.ID }}
	createdOldest = sortColumn{sql: `a.created_at`, key: func(r AskRecord) string { return r.CreatedAt }}
	idOldest      = sortColumn{sql: `a.id`, key: func(r AskRecord) string { return r.ID }}
)

// orderColumns is each order as its columns. The last is always id, so no two
// asks tie and a keyset boundary is exact.
var orderColumns = map[AskOrder][]sortColumn{
	AskOrderNewest:    {createdNewest, idNewest},
	AskOrderOldest:    {createdOldest, idOldest},
	AskOrderAttention: {rankColumn, urgencyColumn, deadlineColumn, createdNewest, idNewest},
	AskOrderDue:       {deadlineColumn, urgencyColumn, createdNewest, idNewest},
}

// The urgency of an ask as the query reads it: an ask stored with none is normal.
const urgencySQL = `COALESCE(NULLIF(a.urgency, ''), 'normal')`

// senderSQL is the sender of an ask as the query reads it. Asks are stored with
// the sender DeriveSender names; the fallback covers a row written without one.
const senderSQL = `COALESCE(NULLIF(a.sender, ''), CASE WHEN a.created_by LIKE 'run:%' THEN '` + SenderAgentRun + `' ELSE '` + SenderUnnamed + `' END)`

func urgencyRank(urgency string) int {
	switch urgency {
	case "critical":
		return 0
	case "high":
		return 1
	case "low":
		return 3
	}
	return 2
}

// urgencyOrder lists the urgencies from the most to the least urgent.
var urgencyOrder = []string{"critical", "high", "normal", "low"}

// ListAsks is the first-page convenience used by existing internal callers.
// Browser and API callers use ListAsksPage to continue through every result.
func (s *Service) ListAsks(ctx context.Context, recipient string, openOnly bool, limit int) ([]AskRecord, error) {
	page, err := s.ListAsksPage(ctx, recipient, AskListOptions{OpenOnly: openOnly, Limit: limit})
	return page.Asks, err
}

type AskListOptions struct {
	OpenOnly    bool
	Limit       int
	Cursor      string
	Query       string
	StateFilter string
	Order       AskOrder
	// Urgencies and Senders keep only asks of those urgencies and from those
	// senders (exact text); empty keeps all. Values of one filter combine with
	// OR, the two filters with AND.
	Urgencies []string
	Senders   []string
}

type AskListPage struct {
	Asks       []AskRecord
	NextCursor string
	// TotalAwaiting is how many pending and escalated asks the reader may see,
	// whatever the query, state filter or cursor of the request.
	TotalAwaiting int
	// Facets are the filter choices with the number of asks each would show.
	Facets AskFacets
}

// AskFacetCount is one filter value and how many asks it holds.
type AskFacetCount struct {
	Value string
	Count int
}

// AskFacets count the asks of a list by urgency and by sender. Each facet
// applies every filter but its own.
type AskFacets struct {
	Urgencies []AskFacetCount
	Senders   []AskFacetCount
}

// maxSenderFacets bounds the sender facet; the senders a request selected are
// listed on top of it.
const maxSenderFacets = 20

// askListCursor is the last row of a page: the value of each column of the
// order, in the order's column order.
type askListCursor struct {
	Version int      `json:"v"`
	Scope   string   `json:"scope"`
	Keys    []string `json:"keys"`
}

// askVisibility is the SQL condition that limits asks to the ones the audience
// may see, and its arguments. The owner sees every ask of the recipient (an empty
// condition); a run or a service only its own. ok is false for an audience that
// can see nothing.
func askVisibility(audience AskAudience) (condition string, args []any, ok bool) {
	if audience.Owner {
		return "", nil, true
	}
	if audience.empty() {
		return "", nil, false
	}
	// Each identity the reader holds adds the asks it may see.
	var visible []string
	if audience.CreatedBy != "" {
		visible = append(visible, `a.created_by = ?`)
		args = append(args, audience.CreatedBy)
	}
	if audience.Conversation != "" {
		visible = append(visible, `a.created_conversation = ?`)
		args = append(args, audience.Conversation)
	}
	if audience.Source != "" {
		visible = append(visible, `(a.source_event_type <> '' AND a.source = ?)`)
		args = append(args, audience.Source)
	}
	return ` AND (` + strings.Join(visible, ` OR `) + `)`, args, true
}

// CountAwaiting is how many pending and escalated asks of recipient the
// audience may see. The condition on state is the one idx_asks_open_attention
// is partial on, so the count reads only open asks.
func (s *Service) CountAwaiting(ctx context.Context, recipient string, audience AskAudience) (int, error) {
	if strings.TrimSpace(recipient) == "" {
		return 0, ErrNotFound
	}
	visible, args, ok := askVisibility(audience)
	if !ok {
		return 0, nil
	}
	var total int
	err := s.db.QueryRowContext(ctx, awaitingCountSQL+visible, append([]any{recipient}, args...)...).Scan(&total)
	return total, err
}

// forRecipientSQL limits asks to one recipient's. It is a lookup of each
// candidate ask's notification by its primary key, so the cost follows the asks
// the rest of the query selects (a few open ones), not the notifications table.
const forRecipientSQL = `EXISTS (SELECT 1 FROM notifications rn WHERE rn.id = a.notification_id AND rn.requested_by = ?)`

const awaitingCountSQL = `SELECT COUNT(*) FROM asks a WHERE a.state IN ('pending', 'escalated') AND ` + forRecipientSQL

// ListAsksPage searches the recipient's full retained inventory before applying
// a stable keyset boundary. Reads never resolve, answer or otherwise change asks.
// Pages are live: newly created rows appear on refresh, not midway through older pages.
func (s *Service) ListAsksPage(ctx context.Context, recipient string, opts AskListOptions) (AskListPage, error) {
	return s.ListAsksPageFor(ctx, recipient, OwnerAudience(), opts)
}

// ListAsksPageFor is ListAsksPage limited to the asks the audience may see. The
// owner sees every ask of the recipient; a run or a service only its own, and
// an audience that can see nothing gets an empty page.
func (s *Service) ListAsksPageFor(ctx context.Context, recipient string, audience AskAudience, opts AskListOptions) (AskListPage, error) {
	if strings.TrimSpace(recipient) == "" {
		return AskListPage{}, ErrNotFound
	}
	if opts.Limit <= 0 {
		opts.Limit = 50
	}
	if opts.Limit > 200 {
		opts.Limit = 200
	}
	opts.Query = strings.TrimSpace(opts.Query)
	if !utf8.ValidString(opts.Query) || utf8.RuneCountInString(opts.Query) > 500 {
		return AskListPage{}, fmt.Errorf("%w: search must be valid UTF-8 and at most 500 characters", ErrInvalidArgument)
	}
	state := strings.TrimSpace(opts.StateFilter)
	if state == "" {
		state = "all"
	}
	if opts.OpenOnly {
		if state != "all" && state != "awaiting" && state != AskPending && state != AskEscalated {
			return AskListPage{}, fmt.Errorf("%w: open_only conflicts with state_filter", ErrInvalidArgument)
		}
		if state == "all" {
			state = "awaiting"
		}
	}
	switch state {
	case "all", "awaiting", AskPending, AskEscalated, AskAnswered, AskDefaulted, AskExpired, AskWithdrawn, AskDismissed:
	default:
		return AskListPage{}, fmt.Errorf("%w: unknown state_filter", ErrInvalidArgument)
	}
	order := opts.Order
	if order == "" {
		order = AskOrderNewest
	}
	columns, ok := orderColumns[order]
	if !ok {
		return AskListPage{}, fmt.Errorf("%w: unknown order", ErrInvalidArgument)
	}
	urgencies, err := normalizeUrgencyFilter(opts.Urgencies)
	if err != nil {
		return AskListPage{}, err
	}
	senders, err := normalizeSenderFilter(opts.Senders)
	if err != nil {
		return AskListPage{}, err
	}
	visible, visibleArgs, ok := askVisibility(audience)
	if !ok {
		return AskListPage{}, nil
	}
	total, err := s.CountAwaiting(ctx, recipient, audience)
	if err != nil {
		return AskListPage{}, err
	}
	scopeBytes, _ := json.Marshal([]any{recipient, opts.Query, state, audience.scopeKey(), string(order), urgencies, senders})
	scope := fmt.Sprintf("%x", sha256.Sum256(scopeBytes))
	var after *askListCursor
	if opts.Cursor != "" {
		after, err = decodeAskListCursor(opts.Cursor, scope, len(columns))
		if err != nil {
			return AskListPage{}, err
		}
	}
	base := askBaseWhere(recipient, visible, visibleArgs, state, opts.Query)
	query, args := askListSQL(base, urgencies, senders, columns, after, opts.Limit+1)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return AskListPage{}, err
	}
	defer rows.Close()
	out := make([]AskRecord, 0, opts.Limit+1)
	for rows.Next() {
		record, scanErr := scanAsk(rows)
		if scanErr != nil {
			return AskListPage{}, scanErr
		}
		out = append(out, record)
	}
	if err := rows.Err(); err != nil {
		return AskListPage{}, err
	}
	facets, err := s.askFacets(ctx, base, urgencies, senders)
	if err != nil {
		return AskListPage{}, err
	}
	page := AskListPage{Asks: out, TotalAwaiting: total, Facets: facets}
	if len(out) > opts.Limit {
		page.Asks = out[:opts.Limit]
		last := page.Asks[len(page.Asks)-1]
		next := askListCursor{Version: 2, Scope: scope, Keys: make([]string, len(columns))}
		for i, column := range columns {
			next.Keys[i] = column.key(last)
		}
		encoded, _ := json.Marshal(next)
		page.NextCursor = base64.RawURLEncoding.EncodeToString(encoded)
	}
	return page, nil
}

// normalizeUrgencyFilter lowercases and de-duplicates the urgencies of a
// filter, in a fixed order so equal filters make equal cursors.
func normalizeUrgencyFilter(values []string) ([]string, error) {
	picked := map[string]bool{}
	for _, value := range values {
		urgency := strings.ToLower(strings.TrimSpace(value))
		if urgency == "" {
			continue
		}
		if !validUrgencies[urgency] {
			return nil, fmt.Errorf("%w: urgencies must be low, normal, high, or critical", ErrInvalidArgument)
		}
		picked[urgency] = true
	}
	var out []string
	for _, urgency := range urgencyOrder {
		if picked[urgency] {
			out = append(out, urgency)
		}
	}
	return out, nil
}

// normalizeSenderFilter trims and de-duplicates the senders of a filter.
func normalizeSenderFilter(values []string) ([]string, error) {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		sender, err := normalizeSender(value)
		if err != nil {
			return nil, err
		}
		if sender == "" || seen[sender] {
			continue
		}
		seen[sender] = true
		out = append(out, sender)
	}
	sort.Strings(out)
	return out, nil
}

// decodeAskListCursor reads a cursor and refuses one made for another search,
// reader or order.
func decodeAskListCursor(raw, scope string, columns int) (*askListCursor, error) {
	if len(raw) > 2048 {
		return nil, fmt.Errorf("%w: cursor is too long", ErrInvalidArgument)
	}
	var cursor askListCursor
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil || json.Unmarshal(decoded, &cursor) != nil || cursor.Version != 2 || cursor.Scope != scope || len(cursor.Keys) != columns {
		return nil, fmt.Errorf("%w: cursor does not match this search", ErrInvalidArgument)
	}
	return &cursor, nil
}

// askWhere is the part of a list query every read of the list shares: the
// recipient's asks the reader may see, in state and matching text. The facets
// count over it with one filter left out.
type askWhere struct {
	sql  string
	args []any
}

func askBaseWhere(recipient, visible string, visibleArgs []any, state, text string) askWhere {
	where := ` WHERE ` + forRecipientSQL + visible
	args := append([]any{recipient}, visibleArgs...)
	if state == "awaiting" {
		where += ` AND a.state IN ('pending', 'escalated')`
	} else if state != "all" {
		where += ` AND a.state = ?`
		args = append(args, state)
	}
	if text != "" {
		// User wildcards are literal. Include option labels so readable answers,
		// not only their machine keys, are searchable without loading all records.
		pattern := "%" + strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(strings.ToLower(text)) + "%"
		where += ` AND (LOWER(a.question) LIKE ? ESCAPE '\' OR LOWER(a.source) LIKE ? ESCAPE '\' OR LOWER(a.options_json) LIKE ? ESCAPE '\' OR LOWER(a.default_answer) LIKE ? ESCAPE '\' OR LOWER(a.requester_ref) LIKE ? ESCAPE '\' OR LOWER(COALESCE(ans.answer, '')) LIKE ? ESCAPE '\' OR LOWER(COALESCE(ans.note, '')) LIKE ? ESCAPE '\' OR LOWER((SELECT n.title FROM notifications n WHERE n.id = a.notification_id)) LIKE ? ESCAPE '\' OR LOWER(` + senderSQL + `) LIKE ? ESCAPE '\')`
		for range 9 {
			args = append(args, pattern)
		}
	}
	return askWhere{sql: where, args: args}
}

// inClause is `expr IN (?, ...)` for the values, or nothing for none.
func inClause(expr string, values []string) (string, []any) {
	if len(values) == 0 {
		return "", nil
	}
	args := make([]any, len(values))
	for i, value := range values {
		args[i] = value
	}
	return ` AND ` + expr + ` IN (?` + strings.Repeat(`, ?`, len(values)-1) + `)`, args
}

// askListSQL is the page query: the base asks of the list in the urgencies and
// from the senders chosen, after the cursor, in order, limit rows.
func askListSQL(base askWhere, urgencies, senders []string, columns []sortColumn, after *askListCursor, limit int) (string, []any) {
	query := askSelect + base.sql
	args := append([]any(nil), base.args...)
	for _, filter := range []struct {
		expr   string
		values []string
	}{{urgencySQL, urgencies}, {senderSQL, senders}} {
		clause, clauseArgs := inClause(filter.expr, filter.values)
		query += clause
		args = append(args, clauseArgs...)
	}
	if after != nil {
		clause, clauseArgs := keysetClause(columns, after.Keys)
		query += clause
		args = append(args, clauseArgs...)
	}
	order := make([]string, len(columns))
	for i, column := range columns {
		order[i] = column.sql + sortDirection(column)
	}
	return query + ` ORDER BY ` + strings.Join(order, `, `) + ` LIMIT ?`, append(args, limit)
}

func sortDirection(column sortColumn) string {
	if column.desc {
		return ` DESC`
	}
	return ` ASC`
}

// keysetClause keeps the rows after the keys in the order: one alternative per
// column, in which every column before it ties and it comes after.
func keysetClause(columns []sortColumn, keys []string) (string, []any) {
	var alternatives []string
	var args []any
	for i := range columns {
		var terms []string
		for j := 0; j < i; j++ {
			terms = append(terms, columns[j].sql+` = ?`)
			args = append(args, cursorValue(columns[j], keys[j]))
		}
		comparison := ` > ?`
		if columns[i].desc {
			comparison = ` < ?`
		}
		terms = append(terms, columns[i].sql+comparison)
		args = append(args, cursorValue(columns[i], keys[i]))
		alternatives = append(alternatives, `(`+strings.Join(terms, ` AND `)+`)`)
	}
	return ` AND (` + strings.Join(alternatives, ` OR `) + `)`, args
}

// cursorValue is a cursor key as the query binds it: an integer for a numeric
// column, so SQLite compares numbers and not text.
func cursorValue(column sortColumn, key string) any {
	if column.numeric {
		if n, err := strconv.Atoi(key); err == nil {
			return n
		}
	}
	return key
}

// askFacets counts the asks of the base list by urgency and by sender. Each
// count applies the other facet's filter and not its own, so choosing a value
// never empties the choices beside it. A value the request selected is listed
// even when it holds no ask.
func (s *Service) askFacets(ctx context.Context, base askWhere, urgencies, senders []string) (AskFacets, error) {
	count := func(expr string, other string, otherArgs []any) (map[string]int, error) {
		query := `SELECT ` + expr + `, COUNT(*) FROM asks a LEFT JOIN answers ans ON ans.ask_id = a.id` + base.sql + other + ` GROUP BY 1`
		rows, err := s.db.QueryContext(ctx, query, append(append([]any(nil), base.args...), otherArgs...)...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		counts := map[string]int{}
		for rows.Next() {
			var value string
			var n int
			if err := rows.Scan(&value, &n); err != nil {
				return nil, err
			}
			counts[value] = n
		}
		return counts, rows.Err()
	}
	senderFilter, senderArgs := inClause(senderSQL, senders)
	urgencyCounts, err := count(urgencySQL, senderFilter, senderArgs)
	if err != nil {
		return AskFacets{}, err
	}
	urgencyFilter, urgencyArgs := inClause(urgencySQL, urgencies)
	senderCounts, err := count(senderSQL, urgencyFilter, urgencyArgs)
	if err != nil {
		return AskFacets{}, err
	}
	var facets AskFacets
	selectedUrgency := map[string]bool{}
	for _, urgency := range urgencies {
		selectedUrgency[urgency] = true
	}
	for _, urgency := range urgencyOrder {
		if n := urgencyCounts[urgency]; n > 0 || selectedUrgency[urgency] {
			facets.Urgencies = append(facets.Urgencies, AskFacetCount{Value: urgency, Count: n})
		}
	}
	names := make([]string, 0, len(senderCounts))
	for name := range senderCounts {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool {
		if senderCounts[names[i]] != senderCounts[names[j]] {
			return senderCounts[names[i]] > senderCounts[names[j]]
		}
		return names[i] < names[j]
	})
	if len(names) > maxSenderFacets {
		names = names[:maxSenderFacets]
	}
	listed := map[string]bool{}
	for _, name := range names {
		listed[name] = true
		facets.Senders = append(facets.Senders, AskFacetCount{Value: name, Count: senderCounts[name]})
	}
	for _, name := range senders {
		if !listed[name] {
			facets.Senders = append(facets.Senders, AskFacetCount{Value: name})
		}
	}
	return facets, nil
}
