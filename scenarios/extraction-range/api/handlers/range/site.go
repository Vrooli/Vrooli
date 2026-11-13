package rangehandler

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"extraction-range/internal/rangeengine"
)

// PracticeSite serves the owned target pages crawlers exercise. It is kept
// separate from the Connect management API: these HTML routes intentionally
// behave like the varied sites the range is designed to model.
func (h *Handler) PracticeSite() http.Handler { return http.HandlerFunc(h.servePractice) }

// PracticeAPI is the small REST contract used by extraction tools and the
// range's own browser UI. The stable external routes are /levels,
// /levels/{id}, /score and /scoreboard; Connect remains available for native
// Vrooli clients.
func (h *Handler) PracticeAPI() http.Handler { return http.HandlerFunc(h.servePracticeAPI) }

func (h *Handler) servePracticeAPI(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(r.URL.Path, "/")
	writeJSON := func(value any) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(value)
	}
	switch {
	case path == "levels" && r.Method == http.MethodGet:
		levels := make([]any, 0)
		for _, level := range h.engine.List() {
			if r.URL.Query().Get("family") == "" || r.URL.Query().Get("family") == level.Family {
				levels = append(levels, map[string]any{"id": level.ID, "family": level.Family, "family_name": level.FamilyName, "difficulty": level.Difficulty, "title": level.Title, "task": level.Task, "tags": level.Tags, "target_url": "/practice/" + level.ID})
			}
		}
		writeJSON(map[string]any{"levels": levels})
	case strings.HasPrefix(path, "levels/") && r.Method == http.MethodGet:
		id := strings.TrimPrefix(path, "levels/")
		level, ok := h.engine.Get(id)
		if !ok {
			http.Error(w, "unknown level", http.StatusNotFound)
			return
		}
		writeJSON(map[string]any{"id": level.ID, "family": level.Family, "family_name": level.FamilyName, "difficulty": level.Difficulty, "title": level.Title, "task": level.Task, "tags": level.Tags, "target_url": "/practice/" + level.ID})
	case path == "score" && r.Method == http.MethodPost:
		var req struct {
			Tool    string               `json:"tool"`
			LevelID string               `json:"level_id"`
			Records []rangeengine.Record `json:"records"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20)).Decode(&req); err != nil {
			http.Error(w, "invalid score payload", http.StatusBadRequest)
			return
		}
		result, err := h.engine.Score(req.Tool, req.LevelID, req.Records)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if h.scores != nil {
			if err := h.scores.SaveBest(r.Context(), result); err != nil {
				http.Error(w, "score persistence failed", http.StatusInternalServerError)
				return
			}
		}
		writeJSON(result)
	case path == "scoreboard" && r.Method == http.MethodGet:
		results := h.engine.Scoreboard()
		if h.scores != nil {
			var err error
			results, err = h.scores.List(r.Context())
			if err != nil {
				http.Error(w, "scoreboard unavailable", http.StatusInternalServerError)
				return
			}
		}
		filtered := make([]rangeengine.Result, 0)
		for _, result := range results {
			if r.URL.Query().Get("tool") != "" && r.URL.Query().Get("tool") != result.Tool {
				continue
			}
			if r.URL.Query().Get("family") != "" && r.URL.Query().Get("family") != result.Family {
				continue
			}
			filtered = append(filtered, result)
		}
		writeJSON(map[string]any{"scores": filtered})
	default:
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "not found", http.StatusNotFound)
	}
}

func (h *Handler) servePractice(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/practice/"), "/")
	if path == "" {
		http.NotFound(w, r)
		return
	}
	parts := strings.Split(path, "/")
	level, ok := h.engine.Get(parts[0])
	if !ok {
		http.NotFound(w, r)
		return
	}
	if len(parts) > 1 && parts[1] == "data" {
		h.serveData(w, r, level)
		return
	}
	if len(parts) > 1 && parts[1] == "session" && r.Method == http.MethodPost {
		h.serveSession(w, r, level)
		return
	}
	if level.Mechanism == "resilience" {
		if status, ok := number(level.Page.Behavior["status_code"]); ok && status >= 400 {
			h.mu.Lock()
			h.resilienceRequests[level.ID]++
			attempt := h.resilienceRequests[level.ID]
			h.mu.Unlock()
			if attempt == 1 {
				if delay, ok := number(level.Page.Behavior["delay_ms"]); ok && delay > 0 && h.sleep != nil {
					h.sleep(time.Duration(delay) * time.Millisecond)
				}
				if status == http.StatusTooManyRequests {
					w.Header().Set("Retry-After", "2")
				}
				http.Error(w, "synthetic transient practice response; respect Retry-After and retry", status)
				return
			}
		}
		if delay, ok := number(level.Page.Behavior["delay_ms"]); ok && delay > 0 && h.sleep != nil {
			h.sleep(time.Duration(delay) * time.Millisecond)
		}
	}
	if level.Page.RequiresSession && cookieValue(r, "range-session") != level.Page.SessionCode {
		h.serveLogin(w, level)
		return
	}
	if level.Page.BotGate != "" && r.URL.Query().Get("token") != level.Page.BotGate {
		h.serveBotGate(w, level)
		return
	}
	if level.Mechanism == "download" && r.URL.Query().Get("format") == "csv" {
		h.serveCSV(w, level)
		return
	}
	rows := append([]rangeengine.Record(nil), level.Page.Records...)
	if level.Mechanism == "search" {
		q := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
		if q != "" {
			filtered := rows[:0]
			for _, row := range rows {
				for _, field := range level.Page.SearchFields {
					if strings.Contains(strings.ToLower(fmt.Sprint(row[field])), q) {
						filtered = append(filtered, row)
						break
					}
				}
			}
			rows = filtered
		}
	}
	pageSize := level.Page.PageSize
	if pageSize < 1 {
		pageSize = len(rows)
	}
	offset := 0
	if level.Mechanism == "pagination" || level.Mechanism == "script_loaded" {
		offset, _ = strconv.Atoi(r.URL.Query().Get("offset"))
		if offset < 0 {
			offset = 0
		}
	}
	start := offset
	if start > len(rows) {
		start = len(rows)
	}
	end := start + pageSize
	if end > len(rows) {
		end = len(rows)
	}
	visible := rows[start:end]
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Practice-Family", level.Family)
	var b strings.Builder
	fmt.Fprintf(&b, "<!doctype html><html><head><meta charset=\"utf-8\"><title>%s</title></head><body><main><h1>%s</h1>", html.EscapeString(level.Page.Title), html.EscapeString(level.Page.Title))
	if level.Mechanism == "search" {
		fmt.Fprintf(&b, "<form><label>Search <input name=\"q\" value=\"%s\"></label><button>Search</button></form>", html.EscapeString(r.URL.Query().Get("q")))
	}
	if level.Mechanism == "download" {
		fmt.Fprintf(&b, "<a href=\"/practice/%s?format=csv\">Download records CSV</a>", url.PathEscape(level.ID))
	}
	if level.Mechanism == "changing_layout" {
		layout := r.URL.Query().Get("layout")
		if layout == "" {
			layout = level.Page.LayoutVariant
		}
		fmt.Fprintf(&b, "<p>Layout: %s</p><a href=\"?visit=next&layout=%s\">Switch layout</a>", html.EscapeString(layout), map[string]string{"cards": "table", "table": "cards"}[layout])
		if layout == "cards" {
			b.WriteString("<section class=\"cards\">")
			for _, row := range visible {
				writeCard(&b, row)
			}
			b.WriteString("</section>")
		} else {
			writeTable(&b, visible, level.Mechanism)
		}
	} else if level.Mechanism == "script_loaded" {
		fmt.Fprintf(&b, "<div id=\"results\" data-source=\"/practice/%s/data\" data-offset=\"%d\">Records load after interaction.</div><button id=\"load\">Load records</button><script>document.querySelector('#load').onclick=async()=>{const box=document.querySelector('#results');const r=await fetch(box.dataset.source+'?offset='+box.dataset.offset);const d=await r.json();box.insertAdjacentHTML('beforeend',d.records.map(x=>`<article data-id='${x.id}' data-active='${x.active}'><span data-field='id'>${x.id}</span><b data-field='name'>${x.name}</b><span data-field='category'>${x.category}</span><span data-field='value'>${x.value}</span><span>active=${x.active}</span></article>`).join(''));if(d.next_offset===null){document.querySelector('#load').disabled=true}else{box.dataset.offset=d.next_offset;document.querySelector('#load').textContent='Load more'}}</script>", url.PathEscape(level.ID), start)
	} else if level.Mechanism == "messy_markup" {
		b.WriteString("<div class=\"record-list legacy-v3\">")
		for i, row := range visible {
			fmt.Fprintf(&b, "<div class=\"row row-%d\" data-record=\"%s\" data-active=\"%v\"><span class=\"field id\">%s</span><div class=\"content\"><strong>%s</strong><i>%s</i><em>%v</em><small>active=%v</small></div></div>", i, html.EscapeString(fmt.Sprint(row["id"])), row["active"], html.EscapeString(fmt.Sprint(row["id"])), html.EscapeString(fmt.Sprint(row["name"])), html.EscapeString(fmt.Sprint(row["category"])), row["value"], row["active"])
		}
		b.WriteString("</div>")
	} else {
		writeTable(&b, visible, level.Mechanism)
	}
	if level.Mechanism == "pagination" {
		for i := 0; i < len(rows); i += pageSize {
			fmt.Fprintf(&b, " <a href=\"?offset=%d\">Page %d</a>", i, i/pageSize+1)
		}
	}
	if level.Mechanism == "script_loaded" && end < len(rows) {
		fmt.Fprintf(&b, "<a href=\"?offset=%d\">Load more</a>", end)
	}
	b.WriteString("</main></body></html>")
	_, _ = w.Write([]byte(b.String()))
}

func (h *Handler) serveData(w http.ResponseWriter, r *http.Request, level rangeengine.Level) {
	if level.Page.RequiresSession && cookieValue(r, "range-session") != level.Page.SessionCode {
		http.Error(w, "sign in required", http.StatusUnauthorized)
		return
	}
	if level.Page.BotGate != "" && r.URL.Query().Get("token") != level.Page.BotGate {
		http.Error(w, "practice challenge required", http.StatusForbidden)
		return
	}
	rows := level.Page.Records
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}
	if offset > len(rows) {
		offset = len(rows)
	}
	size := level.Page.PageSize
	if size < 1 {
		size = len(rows)
	}
	end := offset + size
	if end > len(rows) {
		end = len(rows)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"records": rows[offset:end], "next_offset": func() any {
		if end < len(rows) {
			return end
		}
		return nil
	}()})
}

func (h *Handler) serveLogin(w http.ResponseWriter, level rangeengine.Level) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, "<!doctype html><title>Practice sign in</title><h1>Practice sign in</h1><form method=post action=\"/practice/%s/session\"><label>Access code <input name=code></label><button>Sign in</button></form>", html.EscapeString(level.ID))
}

func (h *Handler) serveSession(w http.ResponseWriter, r *http.Request, level rangeengine.Level) {
	if err := r.ParseForm(); err != nil || r.Form.Get("code") != level.Page.SessionCode {
		http.Error(w, "invalid practice access code", http.StatusUnauthorized)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "range-session", Value: level.Page.SessionCode, Path: "/practice/", HttpOnly: true, SameSite: http.SameSiteLaxMode})
	http.Redirect(w, r, "/practice/"+url.PathEscape(level.ID), http.StatusSeeOther)
}

func (h *Handler) serveBotGate(w http.ResponseWriter, level rangeengine.Level) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, "<!doctype html><title>Owned practice check</title><h1>Owned practice check</h1><p>This synthetic owned-ground level uses the practice token %s.</p><a href=\"?token=%s\">Continue to records</a>", html.EscapeString(level.Page.BotGate), url.QueryEscape(level.Page.BotGate))
}

func (h *Handler) serveCSV(w http.ResponseWriter, level rangeengine.Level) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=\"records.csv\"")
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"id", "name", "category", "value", "active"})
	for _, row := range level.Page.Records {
		_ = cw.Write([]string{fmt.Sprint(row["id"]), fmt.Sprint(row["name"]), fmt.Sprint(row["category"]), fmt.Sprint(row["value"]), fmt.Sprint(row["active"])})
	}
	cw.Flush()
}

func writeTable(b *strings.Builder, rows []rangeengine.Record, kind string) {
	b.WriteString("<table><thead><tr><th>ID</th><th>Name</th><th>Category</th><th>Value</th><th>Active</th></tr></thead><tbody>")
	for _, row := range rows {
		fmt.Fprintf(b, "<tr data-id=\"%s\"><td>%s</td><td>%s</td><td>%s</td><td>%v</td><td>%v</td></tr>", html.EscapeString(fmt.Sprint(row["id"])), html.EscapeString(fmt.Sprint(row["id"])), html.EscapeString(fmt.Sprint(row["name"])), html.EscapeString(fmt.Sprint(row["category"])), row["value"], row["active"])
	}
	b.WriteString("</tbody></table>")
}

func writeCard(b *strings.Builder, row rangeengine.Record) {
	fmt.Fprintf(b, "<article data-id=\"%s\" data-active=\"%v\"><h2>%s</h2><p>%s</p><span>%v</span><small>active=%v</small></article>", html.EscapeString(fmt.Sprint(row["id"])), row["active"], html.EscapeString(fmt.Sprint(row["name"])), html.EscapeString(fmt.Sprint(row["category"])), row["value"], row["active"])
}

func number(v any) (int, bool) {
	switch n := v.(type) {
	case float64:
		return int(n), true
	case int:
		return n, true
	}
	return 0, false
}

func cookieValue(r *http.Request, name string) string {
	c, err := r.Cookie(name)
	if err != nil {
		return ""
	}
	return c.Value
}
