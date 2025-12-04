package browser

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"test-genie/internal/structure/smoke/orchestrator"
)

// PayloadGenerator generates JavaScript payloads for Browserless execution.
type PayloadGenerator struct{}

// NewPayloadGenerator creates a new PayloadGenerator.
func NewPayloadGenerator() *PayloadGenerator {
	return &PayloadGenerator{}
}

// Ensure PayloadGenerator implements orchestrator.PayloadGenerator.
var _ orchestrator.PayloadGenerator = (*PayloadGenerator)(nil)

// DefaultHandshakeSignals are the built-in handshake signals checked when no custom signals are provided.
var DefaultHandshakeSignals = []string{
	"__vrooliBridgeChildInstalled",
	"IFRAME_BRIDGE_READY",
	"IframeBridge.ready",
	"iframeBridge.ready",
	"IframeBridge.getState().ready",
}

// Generate creates a JavaScript payload for the UI smoke test.
// timeout and handshakeTimeout can be time.Duration or int64 (milliseconds).
// viewport specifies the browser viewport dimensions.
// customSignals is an optional list of window property paths to check.
// If empty, DefaultHandshakeSignals are used.
func (g *PayloadGenerator) Generate(uiURL string, timeout, handshakeTimeout interface{}, viewport orchestrator.Viewport, customSignals []string) string {
	urlJSON, _ := json.Marshal(uiURL)
	timeoutMs := toMilliseconds(timeout)
	handshakeTimeoutMs := toMilliseconds(handshakeTimeout)

	signals := customSignals
	if len(signals) == 0 {
		signals = DefaultHandshakeSignals
	}
	handshakeCheckJS := generateHandshakeCheck(signals)

	return fmt.Sprintf(payloadTemplate, handshakeCheckJS, viewport.Width, viewport.Height, string(urlJSON), timeoutMs, handshakeTimeoutMs)
}

// generateHandshakeCheck creates the JavaScript function body for checking handshake signals.
func generateHandshakeCheck(signals []string) string {
	if len(signals) == 0 {
		return "false"
	}

	var checks []string
	for _, signal := range signals {
		check := generateSignalCheck(signal)
		if check != "" {
			checks = append(checks, check)
		}
	}

	if len(checks) == 0 {
		return "false"
	}

	return strings.Join(checks, " ||\n            ")
}

// generateSignalCheck creates a JavaScript expression to check a single signal.
func generateSignalCheck(signal string) string {
	// Handle different signal patterns:
	// - Simple property: "IFRAME_BRIDGE_READY" -> window.IFRAME_BRIDGE_READY === true
	// - Nested property: "IframeBridge.ready" -> window.IframeBridge && window.IframeBridge.ready === true
	// - Method call: "IframeBridge.getState().ready" -> window.IframeBridge && typeof window.IframeBridge.getState === 'function' && window.IframeBridge.getState().ready === true

	if strings.Contains(signal, "()") {
		// Method call pattern (e.g., "IframeBridge.getState().ready")
		parts := strings.Split(signal, ".")
		if len(parts) < 2 {
			return ""
		}
		// Find the method part
		methodIdx := -1
		for i, p := range parts {
			if strings.HasSuffix(p, "()") {
				methodIdx = i
				break
			}
		}
		if methodIdx < 0 {
			return ""
		}

		// Build the check: window.Obj && typeof window.Obj.method === 'function' && window.Obj.method().prop === true
		objPath := "window." + strings.Join(parts[:methodIdx], ".")
		methodName := strings.TrimSuffix(parts[methodIdx], "()")
		var result string
		if methodIdx == 0 {
			result = fmt.Sprintf("(typeof window.%s === 'function' && window.%s().%s === true)",
				methodName, methodName, strings.Join(parts[methodIdx+1:], "."))
		} else {
			result = fmt.Sprintf("(%s && typeof %s.%s === 'function' && %s.%s().%s === true)",
				objPath, objPath, methodName, objPath, methodName, strings.Join(parts[methodIdx+1:], "."))
		}
		return result
	} else if strings.Contains(signal, ".") {
		// Nested property pattern (e.g., "IframeBridge.ready")
		parts := strings.Split(signal, ".")
		if len(parts) < 2 {
			return fmt.Sprintf("window.%s === true", signal)
		}
		// Build guard: window.Obj && window.Obj.prop === true
		objPath := "window." + parts[0]
		fullPath := "window." + signal
		return fmt.Sprintf("(%s && %s === true)", objPath, fullPath)
	} else {
		// Simple property pattern (e.g., "IFRAME_BRIDGE_READY")
		return fmt.Sprintf("window.%s === true", signal)
	}
}

// toMilliseconds converts a timeout value to milliseconds.
func toMilliseconds(v interface{}) int64 {
	switch t := v.(type) {
	case time.Duration:
		return t.Milliseconds()
	case int64:
		return t
	case int:
		return int64(t)
	default:
		return 0
	}
}

// payloadTemplate is the JavaScript code template for UI smoke testing.
// It loads the UI in an iframe and checks for iframe-bridge handshake.
const payloadTemplate = `module.exports = async ({ page }) => {
    const result = {
        success: false,
        console: [],
        network: [],
        pageErrors: [],
        handshake: { signaled: false, timedOut: false, durationMs: 0 },
        storageShim: [],
        screenshot: null,
        html: '',
        title: '',
        url: '',
        error: null,
        timings: {}
    };

    const handshakeCheck = () => (
        (typeof window !== 'undefined') && (
            %s
        )
    );

    try {
        page.on('console', msg => {
            result.console.push({
                level: msg.type(),
                message: msg.text(),
                timestamp: new Date().toISOString()
            });
        });
        page.on('pageerror', error => {
            result.pageErrors.push({
                message: error.message,
                stack: error.stack || null,
                timestamp: new Date().toISOString()
            });
        });
        page.on('requestfailed', request => {
            result.network.push({
                url: request.url(),
                method: request.method(),
                resourceType: request.resourceType(),
                status: request.response() ? request.response().status() : null,
                errorText: request.failure() ? request.failure().errorText : null,
                timestamp: new Date().toISOString()
            });
        });
        page.on('response', response => {
            if (response.status() >= 400) {
                result.network.push({
                    url: response.url(),
                    method: response.request().method(),
                    resourceType: response.request().resourceType(),
                    status: response.status(),
                    errorText: null,
                    timestamp: new Date().toISOString()
                });
            }
        });

        await page.setViewport({ width: %d, height: %d });

        const targetUrl = %s;
        const hostMarkup = ` + "`" + `
<!DOCTYPE html>
<html>
  <head>
    <meta charset="utf-8" />
    <style>
      html, body { margin: 0; padding: 0; background: #050505; height: 100%%; }
      #ui-smoke-frame { border: 0; width: 100%%; height: 100vh; }
    </style>
  </head>
  <body>
    <iframe id="ui-smoke-frame" src=${targetUrl} allow="clipboard-read; clipboard-write"></iframe>
  </body>
</html>` + "`" + `;

        const startTime = Date.now();
        await page.setContent(hostMarkup, { waitUntil: 'load' });
        const frameElement = await page.waitForSelector('#ui-smoke-frame');
        const frame = await frameElement.contentFrame();
        if (!frame) {
            throw new Error('Failed to attach to scenario frame');
        }

        await frame.waitForSelector('body', { timeout: %d });
        const afterGoto = Date.now();

        try {
            const handshakeStart = Date.now();
            await frame.waitForFunction(handshakeCheck, { timeout: %d });
            result.handshake.signaled = true;
            result.handshake.durationMs = Date.now() - handshakeStart;
        } catch (handshakeError) {
            result.handshake.timedOut = true;
            result.handshake.error = handshakeError.message;
        }

        await new Promise(resolve => setTimeout(resolve, 1000));
        try {
            result.storageShim = await frame.evaluate(() => window.__VROOLI_UI_SMOKE_STORAGE_PATCH__ || []);
        } catch (shimError) {
            result.storageShim = [{ prop: 'localStorage', patched: false, reason: shimError?.message || 'shim-eval-failed' }];
        }
        const screenshotBuffer = await frameElement.screenshot({ type: 'png' });

        result.screenshot = screenshotBuffer.toString('base64');
        result.html = await frame.content();
        result.title = await frame.title();
        result.url = frame.url();
        result.timings = {
            gotoMs: afterGoto - startTime,
            totalMs: Date.now() - startTime
        };
        result.success = true;
        return { data: result, type: 'application/json' };
    } catch (error) {
        result.success = false;
        result.error = error.message;
        result.stack = error.stack || null;
        return { data: result, type: 'application/json' };
    }
};`
