package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	rules "scenario-auditor/rules"
)

/*
Rule: Helmet Frame Ancestors
Description: Ensure Helmet configuration disables frameguard and sets CSP frameAncestors to allow trusted loopback iframe parents.
Reason: Default Helmet X-Frame-Options blocks the Vrooli shell, while unrestricted framing invites clickjacking; we need a CSP that balances both.
Category: ui
Severity: high
Standard: ui-security-v1
Targets: ui

<test-case id="helmet-missing-configuration-v2" should-fail="true" path="ui/server.js">
  <description>Helmet enabled without frameguard override or frameAncestors directive</description>
  <input language="javascript">
const express = require('express');
const helmet = require('helmet');

const app = express();
app.use(helmet());
  </input>
  <expected-violations>2</expected-violations>
  <expected-message>frameguard</expected-message>
</test-case>

<test-case id="helmet-missing-frame-ancestors-v2" should-fail="true" path="ui/server.js">
  <description>frameguard disabled but frameAncestors directive missing</description>
  <input language="javascript">
const express = require('express');
const helmet = require('helmet');

const app = express();
app.use(helmet({
  frameguard: false,
  contentSecurityPolicy: { directives: { defaultSrc: ["'self'"] } }
}));
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>frameAncestors</expected-message>
</test-case>

<test-case id="helmet-missing-loopback-hosts-v2" should-fail="true" path="ui/server.js">
  <description>frameAncestors defined but missing mandatory loopback origins</description>
  <input language="javascript">
const express = require('express');
const helmet = require('helmet');

const app = express();
app.use(helmet({
  frameguard: false,
  contentSecurityPolicy: {
    useDefaults: true,
    directives: {
      defaultSrc: ["'self'"],
      frameAncestors: ["'self'", 'https://trusted.example']
    }
  }
}));
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>loopback</expected-message>
</test-case>

<test-case id="helmet-configured-correctly" should-fail="false" path="ui/server.js">
  <description>frameguard disabled and frameAncestors allows loopback plus env overrides</description>
  <input language="javascript">
const express = require('express');
const helmet = require('helmet');

const app = express();
const localFrameAncestors = [
  "'self'",
  'http://localhost:*',
  'http://127.0.0.1:*',
  'http://[::1]:*'
];
const extraFrameAncestors = (process.env.FRAME_ANCESTORS || '').split(/\s+/).filter(Boolean);

app.use(helmet({
  frameguard: false,
  contentSecurityPolicy: {
    useDefaults: true,
    directives: {
      defaultSrc: ["'self'"],
      connectSrc: ["'self'", 'http://localhost:1234', 'ws:', 'wss:'],
      frameAncestors: [...localFrameAncestors, ...extraFrameAncestors]
    }
  }
}));
  </input>
</test-case>

<test-case id="helmet-configured-like-test-genie" should-fail="false" path="ui/server.js">
  <description>Matches Test Genie UI configuration with loopback + env-driven frame ancestors</description>
  <input language="javascript">
const express = require('express');
const cors = require('cors');
const helmet = require('helmet');
const compression = require('compression');
const morgan = require('morgan');

const app = express();
const apiBaseUrl = process.env.TEST_GENIE_API_URL || `http://localhost:${process.env.API_PORT}`;

const localFrameAncestors = [
  "'self'",
  'http://localhost:*',
  'http://127.0.0.1:*',
  'http://[::1]:*'
];

const extraFrameAncestors = (process.env.FRAME_ANCESTORS || '')
  .split(/\s+/)
  .filter(Boolean);

app.use(helmet({
  frameguard: false,
  contentSecurityPolicy: {
    useDefaults: true,
    directives: {
      defaultSrc: ["'self'"],
      styleSrc: ["'self'", "'unsafe-inline'", "https://fonts.googleapis.com"],
      fontSrc: ["'self'", "https://fonts.gstatic.com"],
      scriptSrc: ["'self'", "'unsafe-inline'"],
      imgSrc: ["'self'", "data:", "https:"],
      connectSrc: ["'self'", apiBaseUrl, "ws:", "wss:"],
      frameAncestors: [...localFrameAncestors, ...extraFrameAncestors]
    }
  }
}));

app.use(cors());
app.use(compression());
app.use(morgan('combined'));
  </input>
</test-case>
*/

// CheckHelmetFrameAncestors enforces iframe-safe Helmet configuration on UI servers.
func CheckHelmetFrameAncestors(content []byte, filePath string) []rules.Violation {
	if !isHelmetJSFile(filePath) || !shouldEvaluateHelmetFile(filePath) {
		return nil
	}

	source := string(content)
	if !strings.Contains(source, "helmet") {
		return nil
	}

	var violations []rules.Violation

	if !strings.Contains(source, "frameguard") || !strings.Contains(source, "frameguard: false") {
		line := findHelmetLineNumber(source, "helmet")
		violations = append(violations, newHelmetViolation(filePath, line, "Helmet configuration must disable frameguard and rely on CSP frameAncestors for iframe control"))
	}

	if !strings.Contains(source, "frameAncestors") {
		line := findHelmetLineNumber(source, "contentSecurityPolicy")
		violations = append(violations, newHelmetViolation(filePath, line, "Helmet contentSecurityPolicy directives must define frameAncestors to permit trusted parents"))
		return violations
	}

	missing := missingLoopbackAncestors(source)
	if len(missing) > 0 {
		line := findHelmetLineNumber(source, "frameAncestors")
		violations = append(violations, newHelmetViolation(filePath, line, fmt.Sprintf("frameAncestors must include loopback origins for local iframe embedding: %s", strings.Join(missing, ", "))))
	}

	return violations
}

func missingLoopbackAncestors(source string) []string {
	var missing []string
	for _, host := range requiredLoopbackAncestors() {
		if !strings.Contains(source, host) {
			missing = append(missing, host)
		}
	}
	return missing
}

func requiredLoopbackAncestors() []string {
	return []string{
		"http://localhost:*",
		"http://127.0.0.1:*",
		"http://[::1]:*",
	}
}

func isHelmetJSFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".js", ".ts", ".mjs", ".cjs", ".jsx", ".tsx":
		return true
	default:
		return false
	}
}

func shouldEvaluateHelmetFile(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	base := strings.ToLower(filepath.Base(path))
	if strings.HasPrefix(base, "test_") {
		return false
	}
	name := strings.TrimSuffix(base, filepath.Ext(base))
	if strings.Contains(name, "server") || strings.Contains(name, "ui") {
		return true
	}
	if name == "app" || name == "index" || name == "main" || name == "dev" || name == "proxy" {
		return true
	}
	if strings.Contains(path, "/server") || strings.Contains(path, "/ui/") || strings.Contains(path, "/proxy") {
		return true
	}
	return false
}

func findHelmetLineNumber(source, needle string) int {
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

func newHelmetViolation(filePath string, line int, message string) rules.Violation {
	if line <= 0 {
		line = 1
	}
	return rules.Violation{
		RuleID:         "ui-helmet-frame-ancestors",
		Type:           "ui_helmet_frame_ancestors",
		Severity:       "high",
		Title:          "Helmet iframe protection misconfigured",
		Message:        message,
		Description:    message,
		File:           filePath,
		FilePath:       filePath,
		Line:           line,
		LineNumber:     line,
		Recommendation: "Disable helmet frameguard and configure CSP frameAncestors to include loopback origins plus optional trusted hosts via FRAME_ANCESTORS",
		Standard:       "ui-security-v1",
	}
}
