package spatialnav

import (
	"path/filepath"
	"strings"

	auditrules "structure-health/internal/packs/auditrules"
)

/*
Rule: Spatial Nav Provider
Description: Ensure scenario UI entry point initialises spatial navigation for gamepad/controller support, or explicitly opts out.
Reason: Scenario UIs must be navigable with game controllers (Xbox, PlayStation) via spatial navigation. Without it, console browser users cannot reliably interact with the UI.
Category: ui
Severity: medium
Standard: ui-interop-v1
Targets: ui

<test-case id="spatial-nav-missing" should-fail="true" path="ui/src/main.tsx">
  <description>main.tsx with no spatial nav initialisation and no opt-out comment</description>
  <input language="typescript">
import React from "react";
import ReactDOM from "react-dom/client";
import { initIframeBridgeChild } from "@vrooli/iframe-bridge";
import App from "./App";

if (window.top !== window.self) {
  initIframeBridgeChild();
}

ReactDOM.createRoot(document.getElementById("root")!).render(
  &lt;React.StrictMode&gt;&lt;App /&gt;&lt;/React.StrictMode&gt;
);
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>initSpatialNav</expected-message>
</test-case>

<test-case id="spatial-nav-present" should-fail="false" path="ui/src/main.tsx">
  <description>main.tsx with spatial nav initialisation</description>
  <input language="typescript">
import React from "react";
import ReactDOM from "react-dom/client";
import { initIframeBridgeChild } from "@vrooli/iframe-bridge";
import { initSpatialNav } from "@vrooli/iframe-bridge/spatial";
import App from "./App";

if (window.top !== window.self) {
  initIframeBridgeChild();
}

initSpatialNav();

ReactDOM.createRoot(document.getElementById("root")!).render(
  &lt;React.StrictMode&gt;&lt;App /&gt;&lt;/React.StrictMode&gt;
);
  </input>
</test-case>

<test-case id="spatial-nav-opt-out" should-fail="false" path="ui/src/main.tsx">
  <description>main.tsx with explicit opt-out comment</description>
  <input language="typescript">
import React from "react";
import ReactDOM from "react-dom/client";
import App from "./App";

// spatial-nav: disabled

ReactDOM.createRoot(document.getElementById("root")!).render(
  &lt;React.StrictMode&gt;&lt;App /&gt;&lt;/React.StrictMode&gt;
);
  </input>
</test-case>
*/

// CheckSpatialNavProvider ensures the UI entry point initialises spatial
// navigation or explicitly opts out.
func CheckSpatialNavProvider(content []byte, filePath string) []auditrules.Violation {
	if !isSpatialNavTarget(filePath) {
		return nil
	}

	source := string(content)

	// Pass if spatial nav is initialised.
	if strings.Contains(source, "initSpatialNav") {
		return nil
	}

	// Pass if explicitly opted out.
	if strings.Contains(source, "spatial-nav: disabled") {
		return nil
	}

	line := findLineNumber(source, "ReactDOM")
	if line <= 0 {
		line = 1
	}

	return []auditrules.Violation{
		{
			RuleID:         "ui-spatial-nav-provider",
			Type:           "ui_spatial_nav_provider",
			Severity:       "medium",
			Title:          "Missing spatial navigation initialisation",
			Message:        "UI entry point must call initSpatialNav() from @vrooli/iframe-bridge/spatial for gamepad support, or include a '// spatial-nav: disabled' comment to opt out",
			Description:    "Scenario UIs should initialise spatial navigation to support gamepad/controller input on console browsers",
			File:           filePath,
			FilePath:       filePath,
			Line:           line,
			LineNumber:     line,
			Recommendation: "Add `import { initSpatialNav } from '@vrooli/iframe-bridge/spatial';` and call `initSpatialNav();` in main.tsx",
			Standard:       "ui-interop-v1",
		},
	}
}

func isSpatialNavTarget(path string) bool {
	base := filepath.Base(path)
	// Only check main entry points.
	return base == "main.tsx" || base == "main.ts" || base == "main.jsx"
}

func findLineNumber(source, needle string) int {
	if needle == "" {
		return 1
	}
	lines := strings.Split(source, "\n")
	for idx, line := range lines {
		if strings.Contains(line, needle) {
			return idx + 1
		}
	}
	return 1
}
