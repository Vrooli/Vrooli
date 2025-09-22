package rules

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

/*
Rule: Secure Tunnel Setup
Description: Ensure UI servers proxy API traffic through proxyToApi to remain tunnel-compatible
Reason: Direct API calls bypass the HTTPS tunnel and break scenarios when hosted behind secure proxies
Category: ui
Severity: high
Standard: tunnel-security-v1
Targets: ui

<test-case id="missing-proxy-function" should-fail="true">
  <description>Express server talks to API without defining proxyToApi</description>
  <input language="javascript">
const express = require('express');
const fetch = require('node-fetch');
const app = express();
const API_PORT = process.env.API_PORT;

app.use('/api/metrics', async (req, res) => {
  const response = await fetch(`http://localhost:${API_PORT}/api/metrics`);
  const data = await response.json();
  res.json(data);
});
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>proxyToApi</expected-message>
</test-case>

<test-case id="proxy-not-used" should-fail="true">
  <description>proxyToApi defined but API routes still call fetch directly</description>
  <input language="javascript">
const express = require('express');
const app = express();
const http = require('http');
const API_PORT = process.env.API_PORT;

function proxyToApi(req, res, apiPath) {
  const options = { hostname: 'localhost', port: API_PORT, path: apiPath };
  const proxyReq = http.request(options, proxyRes => proxyRes.pipe(res));
  req.pipe(proxyReq);
}

app.get('/api/health', async (req, res) => {
  const response = await fetch(`http://localhost:${API_PORT}/api/health`);
  res.json(await response.json());
});
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>proxyToApi</expected-message>
</test-case>

<test-case id="direct-http-request-outside-proxy" should-fail="true">
  <description>http.request using API_PORT outside proxyToApi</description>
  <input language="javascript">
const express = require('express');
const http = require('http');
const app = express();
const API_PORT = process.env.API_PORT;

function proxyToApi(req, res, apiPath) {
  const options = { hostname: 'localhost', port: API_PORT, path: apiPath };
  const proxyReq = http.request(options, proxyRes => proxyRes.pipe(res));
  req.pipe(proxyReq);
}

app.post('/api/logs', (req, res) => {
  const request = http.request({
    hostname: 'localhost',
    port: API_PORT,
    path: '/api/logs',
    method: 'POST'
  }, proxyRes => proxyRes.pipe(res));
  req.pipe(request);
});
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>proxyToApi</expected-message>
</test-case>

<test-case id="secure-proxy-implementation" should-fail="false">
  <description>proxyToApi defined and all routes use it</description>
  <input language="javascript">
const express = require('express');
const http = require('http');
const path = require('path');

const app = express();
const API_PORT = process.env.API_PORT;
const UI_PORT = process.env.UI_PORT;

function proxyToApi(req, res, apiPath) {
  const options = {
    hostname: 'localhost',
    port: API_PORT,
    path: apiPath || req.url,
    method: req.method,
    headers: { ...req.headers, host: `localhost:${API_PORT}` }
  };

  const proxyReq = http.request(options, proxyRes => {
    res.status(proxyRes.statusCode);
    Object.entries(proxyRes.headers).forEach(([key, value]) => res.setHeader(key, value));
    proxyRes.pipe(res);
  });

  proxyReq.on('error', (err) => {
    res.status(502).json({ error: 'API server unavailable', details: err.message });
  });

  if (req.method === 'GET' || req.method === 'HEAD') {
    proxyReq.end();
  } else {
    req.pipe(proxyReq);
  }
}

app.use('/api', (req, res) => {
  const fullPath = '/api' + (req.url.startsWith('/') ? req.url : '/' + req.url);
  proxyToApi(req, res, fullPath);
});

app.get('/health', (req, res) => proxyToApi(req, res, '/health'));
app.use(express.static(path.join(__dirname, 'public')));

app.listen(UI_PORT, () => {
  console.log(`UI running on ${UI_PORT}`);
});
  </input>
</test-case>
*/

var proxyFunctionRegex = regexp.MustCompile(`(?m)^(?:\s*(?:async\s+)?function\s+proxyToApi\s*\(|\s*(?:const|let|var)\s+proxyToApi\s*=)`)

// CheckSecureTunnelSetup ensures Express UI servers proxy API calls through proxyToApi.
func CheckSecureTunnelSetup(content []byte, filePath string) []Violation {
	if !isJSFile(filePath) || !shouldEvaluateFile(filePath) {
		return nil
	}

	source := string(content)
	if !strings.Contains(source, "API_PORT") {
		return nil
	}
	if !strings.Contains(source, "express") && !strings.Contains(source, "proxyToApi") {
		return nil
	}

	hasProxyDefinition := proxyFunctionRegex.Match(content)
	hasProxyCall := strings.Contains(source, "proxyToApi(")

	if !hasProxyDefinition {
		line := findLineNumber(source, "API_PORT")
		return []Violation{newTunnelViolation(filePath, line, "UI servers must define proxyToApi to tunnel API traffic")}
	}

	var violations []Violation

	if !hasProxyCall {
		line := findLineNumber(source, "proxyToApi")
		if line == 0 {
			line = findLineNumber(source, "app.")
		}
		violations = append(violations, newTunnelViolation(filePath, line, "API routes must call proxyToApi instead of direct HTTP requests"))
	}

	directCallLines := detectDirectApiCalls(source)
	for _, line := range directCallLines {
		violations = append(violations, newTunnelViolation(filePath, line, "Direct API calls detected outside proxyToApi; route them through proxyToApi"))
	}

	return dedupeTunnelViolations(violations)
}

func isJSFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".js", ".ts", ".mjs", ".cjs", ".javascript", ".typescript":
		return true
	default:
		return false
	}
}

func shouldEvaluateFile(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	base := strings.ToLower(filepath.Base(path))
	if strings.HasPrefix(base, "test_") {
		return true
	}
	name := strings.TrimSuffix(base, filepath.Ext(base))
	if strings.Contains(name, "server") {
		return true
	}
	if name == "app" || name == "index" || name == "main" || name == "dev" || name == "proxy" {
		return true
	}
	if strings.Contains(path, "/server") || strings.Contains(path, "/proxy") {
		return true
	}
	return false
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

func detectDirectApiCalls(source string) []int {
	lines := strings.Split(source, "\n")
	inProxy := false
	pendingProxyStart := false
	braceDepth := 0
	var direct []int

	for idx, line := range lines {
		trimmed := strings.TrimSpace(line)

		if !inProxy {
			if proxyFunctionStart(trimmed) {
				inProxy = true
				braceDelta := strings.Count(line, "{") - strings.Count(line, "}")
				braceDepth = braceDelta
				if braceDepth <= 0 {
					pendingProxyStart = true
				}
				continue
			}
			if pendingProxyStart {
				braceDelta := strings.Count(line, "{") - strings.Count(line, "}")
				braceDepth += braceDelta
				if braceDepth > 0 {
					pendingProxyStart = false
					inProxy = true
					continue
				}
			}
		} else {
			braceDepth += strings.Count(line, "{") - strings.Count(line, "}")
			if braceDepth <= 0 {
				inProxy = false
				pendingProxyStart = false
			}
			continue
		}

		lower := strings.ToLower(line)
		if !(strings.Contains(lower, "fetch(") || strings.Contains(lower, "axios") || strings.Contains(lower, "http.request") || strings.Contains(lower, "http.get")) {
			continue
		}

		if strings.Contains(lower, "proxytoapi(") {
			continue
		}

		if !referencesAPITarget(lines, idx) {
			continue
		}

		if strings.Contains(lower, "console.log") {
			continue
		}

		direct = append(direct, idx+1)
	}

	return direct
}

func proxyFunctionStart(line string) bool {
	if line == "" {
		return false
	}
	if strings.HasPrefix(line, "function proxyToApi") || strings.HasPrefix(line, "async function proxyToApi") {
		return true
	}
	if strings.HasPrefix(line, "const proxyToApi") || strings.HasPrefix(line, "let proxyToApi") || strings.HasPrefix(line, "var proxyToApi") {
		return true
	}
	return false
}

func newTunnelViolation(filePath string, line int, message string) Violation {
	if line <= 0 {
		line = 1
	}
	return Violation{
		Type:           "ui_secure_tunnel",
		Severity:       "high",
		Title:          "Secure tunnel proxy missing",
		Description:    message,
		FilePath:       filePath,
		LineNumber:     line,
		Recommendation: "Proxy API routes through proxyToApi as outlined in TUNNEL_CONVERSION_GUIDE.md",
		Standard:       "tunnel-security-v1",
	}
}

func dedupeTunnelViolations(list []Violation) []Violation {
	if len(list) == 0 {
		return list
	}
	seen := make(map[string]bool)
	var deduped []Violation
	for _, v := range list {
		key := strings.Join([]string{v.Description, v.FilePath, strconv.Itoa(v.LineNumber)}, "|")
		if seen[key] {
			continue
		}
		seen[key] = true
		deduped = append(deduped, v)
	}
	return deduped
}

func referencesAPITarget(lines []string, idx int) bool {
	limit := len(lines)
	for offset := 0; offset <= 3; offset++ {
		pos := idx + offset
		if pos >= limit {
			break
		}
		lower := strings.ToLower(lines[pos])
		if strings.Contains(lower, "api_port") || strings.Contains(lower, "localhost") || strings.Contains(lower, "http://") || strings.Contains(lower, "https://") {
			return true
		}
	}
	return false
}
