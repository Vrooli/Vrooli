/*
Rule: Shortcut Relay
ID: interop_shortcut_relay
Description: Checks that keyboard hook files use emitShortcutIntent from
  the iframe-bridge to relay shortcut keys to the parent frame instead
  of handling them locally.
Why: When a scenario UI runs inside an iframe, keyboard shortcuts
  (Ctrl+S, Ctrl+K, etc.) are captured by the iframe and never reach
  the host application. emitShortcutIntent sends a structured message
  to the parent so both layers can coordinate shortcut handling.
Category: interop
Severity: medium
Slot: [G]
SlotFile: ui/src/hooks/
TechStack: iframe-bridge
Recommendation: Import emitShortcutIntent from @vrooli/iframe-bridge and
  call it in your keyboard hook before or after local handling.
Standard: vrooli-ui-interop-v1

GoodExample:
    // ui/src/hooks/useKeyboard.ts
    import { emitShortcutIntent } from "@vrooli/iframe-bridge";

    export function useKeyboard() {
      useEffect(() => {
        const handler = (e: KeyboardEvent) => {
          emitShortcutIntent({ key: e.key, ctrl: e.ctrlKey });
          // local handling...
        };
        window.addEventListener("keydown", handler);
        return () => window.removeEventListener("keydown", handler);
      }, []);
    }

BadExample:
    // ui/src/hooks/useKeyboard.ts — no emitShortcutIntent
    export function useKeyboard() {
      useEffect(() => {
        const handler = (e: KeyboardEvent) => {
          if (e.ctrlKey && e.key === "s") {
            e.preventDefault();
            save();
          }
        };
        window.addEventListener("keydown", handler);
        return () => window.removeEventListener("keydown", handler);
      }, []);
    }

<test-case id="shortcut-relay-present-in-hooks" should-fail="false">
  <description>Keyboard hook uses emitShortcutIntent</description>
  <input>
    [ui/src/hooks/useKeyboard.ts]
    import { emitShortcutIntent } from "@vrooli/iframe-bridge";
    export function useKeyboard() {
      window.addEventListener("keydown", (e) => {
        emitShortcutIntent({ key: e.key, ctrl: e.ctrlKey });
      });
    }
  </input>
</test-case>

<test-case id="shortcut-relay-no-hooks-dir" should-fail="false">
  <description>No hooks directory exists — rule is skipped</description>
  <input>
    [ui/src/App.tsx]
    import React from "react";
    export default function App() { return <div />; }
  </input>
</test-case>

<test-case id="shortcut-relay-no-keyboard-hooks" should-fail="false">
  <description>Hooks directory exists but no keyboard-related hooks</description>
  <input>
    [ui/src/hooks/useTheme.ts]
    export function useTheme() { return "dark"; }
  </input>
</test-case>

<test-case id="shortcut-relay-missing" should-fail="true">
  <description>Keyboard hook exists without emitShortcutIntent</description>
  <input>
    [ui/src/hooks/useKeyboard.ts]
    export function useKeyboard() {
      window.addEventListener("keydown", (e) => {
        if (e.ctrlKey && e.key === "s") {
          e.preventDefault();
          save();
        }
      });
    }
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>emitShortcutIntent not found</expected-message>
</test-case>
*/

package checks

import (
	"os"
	"path/filepath"
	"strings"

	"ui-health/internal/uiinterop"
)

func init() {
	uiinterop.Register("interop_shortcut_relay", checkShortcutRelay)
}

// keyboardIndicators are substrings that suggest a file deals with keyboard input.
var keyboardIndicators = []string{"keydown", "keyboard", "shortcut", "hotkey"}

func checkShortcutRelay(ctx uiinterop.CheckContext) uiinterop.RuleResult {
	const ruleID = "interop_shortcut_relay"

	hooksDir := filepath.Join(ctx.ScenarioRoot, "ui", "src", "hooks")
	if _, err := os.Stat(hooksDir); os.IsNotExist(err) {
		return uiinterop.RuleResult{
			RuleID:     ruleID,
			Skipped:    true,
			SkipReason: "ui/src/hooks/ directory not found",
			Message:    "ui/src/hooks/ directory not found; skipping",
		}
	}

	// Find keyboard-related hook files.
	hasKeyboardHooks := false
	_ = filepath.Walk(hooksDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		ext := filepath.Ext(info.Name())
		if _, ok := scanExtensions[ext]; !ok {
			return nil
		}

		// Check file name first.
		lower := strings.ToLower(info.Name())
		for _, kw := range keyboardIndicators {
			if strings.Contains(lower, kw) {
				hasKeyboardHooks = true
				return nil
			}
		}

		// Check file content.
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		content := strings.ToLower(string(data))
		for _, kw := range keyboardIndicators {
			if strings.Contains(content, kw) {
				hasKeyboardHooks = true
				return nil
			}
		}
		return nil
	})

	if !hasKeyboardHooks {
		return uiinterop.RuleResult{
			RuleID:     ruleID,
			Skipped:    true,
			SkipReason: "no keyboard-related hooks found",
			Message:    "no keyboard-related hooks found in ui/src/hooks/; skipping",
		}
	}

	// Check for emitShortcutIntent in hooks/ first.
	if containsInDir(hooksDir, "emitShortcutIntent") {
		return uiinterop.RuleResult{
			RuleID:  ruleID,
			Passed:  true,
			Message: "emitShortcutIntent found in ui/src/hooks/",
		}
	}

	// Broaden search to ui/src/.
	srcDir := filepath.Join(ctx.ScenarioRoot, "ui", "src")
	if containsInDir(srcDir, "emitShortcutIntent") {
		return uiinterop.RuleResult{
			RuleID:  ruleID,
			Passed:  true,
			Message: "emitShortcutIntent found in ui/src/",
		}
	}

	return uiinterop.RuleResult{
		RuleID:  ruleID,
		Passed:  false,
		Message: "emitShortcutIntent not found but keyboard hooks exist",
		Violations: []uiinterop.Violation{{
			RuleID:         ruleID,
			Severity:       "medium",
			Title:          "Missing emitShortcutIntent",
			Description:    "Keyboard hooks exist in ui/src/hooks/ but emitShortcutIntent is not called anywhere under ui/src/",
			FilePath:       "ui/src/hooks/",
			Recommendation: "Import emitShortcutIntent from @vrooli/iframe-bridge and call it in your keyboard hook",
		}},
	}
}

// containsInDir walks a directory and returns true if any scannable file
// contains the given needle string.
func containsInDir(dir, needle string) bool {
	found := false
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || found {
			return nil
		}
		if info.IsDir() {
			if _, skip := skipDirectories[info.Name()]; skip {
				return filepath.SkipDir
			}
			return nil
		}
		ext := filepath.Ext(info.Name())
		if _, ok := scanExtensions[ext]; !ok {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		if strings.Contains(string(data), needle) {
			found = true
		}
		return nil
	})
	return found
}
