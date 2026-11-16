package operatordecision

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// maxBriefBytes bounds the packet brief a page reads.
const maxBriefBytes = 256 << 10

// BriefSection is one `## ` section of a packet brief.
type BriefSection struct{ Heading, Body string }

// ReadBrief reads a packet's BRIEF.md (path is the brief or its folder; a
// leading ~/ is the home folder) and returns its `## ` sections in order.
// Producers show the operator-facing ones on the decision page, so what the
// operator approves is exactly what the packet will run.
func ReadBrief(path string) ([]BriefSection, error) {
	if rest, ok := strings.CutPrefix(path, "~/"); ok {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		path = filepath.Join(home, rest)
	}
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		path = filepath.Join(path, "BRIEF.md")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(data) > maxBriefBytes {
		data = data[:maxBriefBytes]
	}
	var sections []BriefSection
	for _, part := range strings.Split("\n"+string(data), "\n## ")[1:] {
		heading, body, _ := strings.Cut(part, "\n")
		sections = append(sections, BriefSection{Heading: strings.TrimSpace(heading), Body: strings.TrimSpace(body)})
	}
	return sections, nil
}

// PacketLine finds a `Packet: <path>` line in free text (a requester's own
// words), and returns the path and the text without that line.
func PacketLine(text string) (path, rest string) {
	var kept []string
	for _, line := range strings.Split(text, "\n") {
		if p, ok := strings.CutPrefix(strings.TrimSpace(line), "Packet:"); ok && path == "" && strings.TrimSpace(p) != "" {
			path = strings.TrimSpace(p)
			continue
		}
		kept = append(kept, line)
	}
	return path, strings.TrimSpace(strings.Join(kept, "\n"))
}

var inlineMark = regexp.MustCompile("`([^`]+)`|\\*\\*([^*]+)\\*\\*")

// inline escapes text and keeps `code` and **bold**.
func inline(text string) string {
	var b strings.Builder
	last := 0
	for _, m := range inlineMark.FindAllStringSubmatchIndex(text, -1) {
		b.WriteString(esc(text[last:m[0]]))
		if m[2] >= 0 {
			b.WriteString("<code>" + esc(text[m[2]:m[3]]) + "</code>")
		} else {
			b.WriteString("<strong>" + esc(text[m[4]:m[5]]) + "</strong>")
		}
		last = m[1]
	}
	b.WriteString(esc(text[last:]))
	return b.String()
}

func tableCells(line string) []string {
	cells := strings.Split(strings.Trim(strings.TrimSpace(line), "|"), "|")
	for i := range cells {
		cells[i] = strings.TrimSpace(cells[i])
	}
	return cells
}

// Markdown adds plain Markdown as page blocks: paragraphs, `- ` lists, pipe
// tables (cards on a phone) and `###` headings, with `code` and **bold**.
// Everything is escaped, so a packet's own words can be shown as written.
func (p *Page) Markdown(md string) *Page {
	lines := strings.Split(strings.ReplaceAll(md, "\r\n", "\n"), "\n")
	var para []string
	flush := func() {
		if len(para) > 0 {
			p.body.WriteString("<p>" + inline(strings.Join(para, " ")) + "</p>")
			para = nil
		}
	}
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		switch {
		case line == "":
			flush()
		case strings.HasPrefix(line, "#"):
			flush()
			p.body.WriteString("<h3>" + inline(strings.TrimSpace(strings.TrimLeft(line, "#"))) + "</h3>")
		case strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* "):
			flush()
			p.body.WriteString("<ul>")
			for ; i < len(lines); i++ {
				item := strings.TrimSpace(lines[i])
				if !strings.HasPrefix(item, "- ") && !strings.HasPrefix(item, "* ") {
					if item != "" && strings.HasPrefix(lines[i], "  ") {
						p.body.WriteString(" " + inline(item)) // a wrapped item line
						continue
					}
					i--
					break
				}
				if !strings.HasSuffix(p.body.String(), "<ul>") {
					p.body.WriteString("</li>")
				}
				p.body.WriteString("<li>" + inline(item[2:]))
			}
			p.body.WriteString("</li></ul>")
		case strings.HasPrefix(line, "|") && i+1 < len(lines) && strings.Contains(lines[i+1], "---"):
			flush()
			headers := tableCells(line)
			var rows [][]string
			for i += 2; i < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[i]), "|"); i++ {
				rows = append(rows, tableCells(lines[i]))
			}
			i--
			p.tableMarkup(headers, rows)
		default:
			para = append(para, line)
		}
	}
	flush()
	return p
}

// tableMarkup is Table with inline Markdown in the cells.
func (p *Page) tableMarkup(headers []string, rows [][]string) {
	p.body.WriteString(`<table class="hub-tradeoffs"><thead><tr>`)
	for _, h := range headers {
		p.body.WriteString("<th>" + inline(h) + "</th>")
	}
	p.body.WriteString("</tr></thead><tbody>")
	for _, row := range rows {
		p.body.WriteString("<tr>")
		for i, cell := range row {
			label := ""
			if i < len(headers) {
				label = headers[i]
			}
			p.body.WriteString(`<td data-label="` + esc(label) + `">` + inline(cell) + "</td>")
		}
		p.body.WriteString("</tr>")
	}
	p.body.WriteString("</tbody></table>")
}
