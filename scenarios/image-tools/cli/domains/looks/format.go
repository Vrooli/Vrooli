package looks

import (
	"fmt"
	"sort"
	"strings"

	looksv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/looks"
)

// formatLook is the one-line list rendering of a Look.
func formatLook(l *looksv1.Look) string {
	tag := "custom"
	if l.GetBuiltin() {
		tag = "built-in"
	}
	return fmt.Sprintf("%-16s [%s/%s] %s", l.GetId(), kindName(l.GetKind()), tag, l.GetName())
}

// formatLookDetail is the multi-line detail rendering of a Look.
func formatLookDetail(l *looksv1.Look) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s — %s\n", l.GetName(), l.GetDescription())
	fmt.Fprintf(&b, "  id=%s kind=%s builtin=%t\n", l.GetId(), kindName(l.GetKind()), l.GetBuiltin())
	if t := l.GetPromptTemplate(); t != "" {
		fmt.Fprintf(&b, "  prompt: %s\n", t)
	}
	for i, s := range l.GetSteps() {
		fmt.Fprintf(&b, "  step %d: %s [%s] %s\n", i+1, s.GetOperation(), stepKindName(s.GetKind()), formatParams(s.GetParams()))
	}
	if ref := l.GetThumbnailRef(); ref != "" {
		fmt.Fprintf(&b, "  thumbnail: %s\n", ref)
	}
	return strings.TrimRight(b.String(), "\n")
}

// formatParams renders a step's params map in deterministic (sorted) order.
func formatParams(p map[string]string) string {
	if len(p) == 0 {
		return ""
	}
	keys := make([]string, 0, len(p))
	for k := range p {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", k, p[k]))
	}
	return strings.Join(parts, " ")
}

func kindName(k looksv1.LookKind) string {
	switch k {
	case looksv1.LookKind_LOOK_KIND_STYLE:
		return "style"
	case looksv1.LookKind_LOOK_KIND_FILM:
		return "film"
	case looksv1.LookKind_LOOK_KIND_CAMERA:
		return "camera"
	case looksv1.LookKind_LOOK_KIND_ENHANCE:
		return "enhance"
	case looksv1.LookKind_LOOK_KIND_CUSTOM:
		return "custom"
	default:
		return "unspecified"
	}
}

func stepKindName(k looksv1.StepKind) string {
	switch k {
	case looksv1.StepKind_STEP_KIND_DETERMINISTIC:
		return "deterministic"
	case looksv1.StepKind_STEP_KIND_AI:
		return "ai"
	default:
		return "?"
	}
}
