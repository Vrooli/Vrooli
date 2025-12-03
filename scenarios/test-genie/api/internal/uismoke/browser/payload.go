package browser

import (
	"encoding/json"
	"fmt"
	"time"

	"test-genie/internal/uismoke/orchestrator"
)

// PayloadGenerator generates JavaScript payloads for Browserless execution.
type PayloadGenerator struct{}

// NewPayloadGenerator creates a new PayloadGenerator.
func NewPayloadGenerator() *PayloadGenerator {
	return &PayloadGenerator{}
}

// Ensure PayloadGenerator implements orchestrator.PayloadGenerator.
var _ orchestrator.PayloadGenerator = (*PayloadGenerator)(nil)

// Generate creates a JavaScript payload for the UI smoke test.
// timeout and handshakeTimeout can be time.Duration or int64 (milliseconds).
func (g *PayloadGenerator) Generate(uiURL string, timeout, handshakeTimeout interface{}) string {
	urlJSON, _ := json.Marshal(uiURL)
	timeoutMs := toMilliseconds(timeout)
	handshakeTimeoutMs := toMilliseconds(handshakeTimeout)

	return fmt.Sprintf(payloadTemplate, string(urlJSON), timeoutMs, handshakeTimeoutMs)
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

    await page.evaluateOnNewDocument(() => {
        // Simplified storage patch for browserless v1 (vm2) compatibility
        // Avoid complex error object access that triggers VM2 internal state issues
        window.__VROOLI_UI_SMOKE_STORAGE_PATCH__ = [];
    });

    const handshakeCheck = () => (
        (typeof window !== 'undefined') && (
            window.__vrooliBridgeChildInstalled === true ||
            window.IFRAME_BRIDGE_READY === true ||
            (window.IframeBridge && window.IframeBridge.ready === true) ||
            (window.iframeBridge && window.iframeBridge.ready === true) ||
            (window.IframeBridge && typeof window.IframeBridge.getState === 'function' && window.IframeBridge.getState().ready === true)
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

        await page.setViewport({ width: 1280, height: 720 });

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
