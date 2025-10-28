package runtime

import (
	"encoding/json"
	"strconv"
	"strings"
)

func buildWorkflowScript(instructions []Instruction) (string, error) {
	payload, err := json.Marshal(instructions)
	if err != nil {
		return "", err
	}

	script := strings.Replace(workflowScriptTemplate, "__INSTRUCTIONS__", string(payload), 1)
	script = strings.ReplaceAll(script, "__VIEWPORT_WIDTH__", strconv.Itoa(defaultViewportWidth))
	script = strings.ReplaceAll(script, "__VIEWPORT_HEIGHT__", strconv.Itoa(defaultViewportHeight))
	script = strings.ReplaceAll(script, "__DEFAULT_TIMEOUT__", strconv.Itoa(defaultTimeoutMillis))
	return script, nil
}

const workflowScriptTemplate = `export default async ({ page, context }) => {
  const steps = __INSTRUCTIONS__;
  const results = [];
  const runStart = Date.now();

  await page.setViewport({ width: __VIEWPORT_WIDTH__, height: __VIEWPORT_HEIGHT__ });

  for (const step of steps) {
    const stepStart = Date.now();
    const params = step.params || {};
    const stepName = params.name || '';
    const baseResult = { index: step.index, nodeId: step.nodeId, type: step.type, stepName };

    try {
      switch (step.type) {
        case 'navigate': {
          if (!params.url) {
            throw new Error('navigate step missing url');
          }
          const waitUntil = params.waitUntil || 'networkidle2';
          const timeout = Number.isFinite(params.timeoutMs) ? params.timeoutMs : __DEFAULT_TIMEOUT__;
          await page.goto(params.url, { waitUntil, timeout });
          if (params.waitForMs) {
            await page.waitForTimeout(params.waitForMs);
          }
          results.push({ ...baseResult, success: true, durationMs: Date.now() - stepStart });
          break;
        }
        case 'wait': {
          const waitType = params.waitType || 'time';
          if (waitType === 'element') {
            if (!params.selector) {
              throw new Error('wait element step missing selector');
            }
            const timeout = Number.isFinite(params.timeoutMs) ? params.timeoutMs : 10000;
            await page.waitForSelector(params.selector, { timeout });
          } else {
            const duration = Number.isFinite(params.durationMs) ? Math.max(params.durationMs, 0) : 1000;
            await page.waitForTimeout(duration);
          }
          results.push({ ...baseResult, success: true, durationMs: Date.now() - stepStart });
          break;
        }
        case 'screenshot': {
          if (params.viewportWidth && params.viewportHeight) {
            await page.setViewport({ width: params.viewportWidth, height: params.viewportHeight });
          }
          if (params.waitForMs) {
            await page.waitForTimeout(params.waitForMs);
          }
          const fullPage = (typeof params.fullPage === 'boolean') ? params.fullPage : true;
          const capture = await page.screenshot({ encoding: 'base64', fullPage });
          results.push({ ...baseResult, success: true, durationMs: Date.now() - stepStart, screenshotBase64: capture });
          break;
        }
        default: {
          const errorMessage = 'Unsupported step type: ' + step.type;
          results.push({ ...baseResult, success: false, error: errorMessage, durationMs: Date.now() - stepStart });
          return { success: false, error: errorMessage, steps: results };
        }
      }
    } catch (error) {
      const message = (error && error.message) ? error.message : String(error);
      const stack = (error && error.stack) ? error.stack : undefined;
      results.push({ ...baseResult, success: false, error: message, stack, durationMs: Date.now() - stepStart });
      return { success: false, error: message, steps: results };
    }
  }

  return { success: true, steps: results, totalDurationMs: Date.now() - runStart };
};
`

// BuildWorkflowScript exposes the script generation for unit tests.
func BuildWorkflowScript(instructions []Instruction) (string, error) {
	return buildWorkflowScript(instructions)
}
