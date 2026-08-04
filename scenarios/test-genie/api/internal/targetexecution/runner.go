// Package targetexecution owns language-to-runner selection for executable
// targets. It returns a typed unsupported result so missing language support is
// visible as a capability gap rather than a silent pass.
package targetexecution

import "strings"

type Language string

const (
	LanguageGo         Language = "go"
	LanguageTypeScript Language = "typescript"
	LanguageJavaScript Language = "javascript"
	LanguageUnknown    Language = "unknown"
)

type Runner struct {
	Language Language
	Command  []string
}

type Unsupported struct {
	Language Language
	Reason   string
}

func (e Unsupported) Error() string { return "unsupported execution language " + string(e.Language) + ": " + e.Reason }

// ForLanguage selects only runners with an established deterministic command.
func ForLanguage(language string) (Runner, error) {
	switch Language(strings.ToLower(strings.TrimSpace(language))) {
	case LanguageGo:
		return Runner{Language: LanguageGo, Command: []string{"go", "test", "./..."}}, nil
	case LanguageTypeScript, LanguageJavaScript:
		return Runner{Language: LanguageTypeScript, Command: []string{"pnpm", "test"}}, nil
	default:
		kind := Language(strings.ToLower(strings.TrimSpace(language)))
		if kind == "" {
			kind = LanguageUnknown
		}
		return Runner{}, Unsupported{Language: kind, Reason: "no Go or TypeScript runner is registered"}
	}
}
