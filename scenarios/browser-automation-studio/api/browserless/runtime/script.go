package runtime

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

func buildStepScript(instruction Instruction, viewportWidth, viewportHeight int) (string, error) {
	payload, err := json.Marshal(instruction)
	if err != nil {
		return "", err
	}

	// Debug: Log if preloadHTML is present in the instruction struct
	if instruction.PreloadHTML != "" {
		// Also check if it's actually in the marshalled JSON
		var check map[string]interface{}
		json.Unmarshal(payload, &check)
		if preloadHtml, ok := check["preloadHtml"].(string); ok {
			fmt.Printf("[BAS-SCRIPT-DEBUG] PreloadHTML in JSON: %d bytes\n", len(preloadHtml))
		} else {
			fmt.Printf("[BAS-SCRIPT-DEBUG] PreloadHTML field exists (%d bytes) but NOT in JSON!\n", len(instruction.PreloadHTML))
		}
	}

	if viewportWidth <= 0 {
		viewportWidth = defaultViewportWidth
	}
	if viewportHeight <= 0 {
		viewportHeight = defaultViewportHeight
	}
	viewportWidth = clampViewport(viewportWidth)
	viewportHeight = clampViewport(viewportHeight)

	script := strings.Replace(stepScriptTemplate, "__INSTRUCTION__", string(payload), 1)
	script = strings.ReplaceAll(script, "__VIEWPORT_WIDTH__", strconv.Itoa(viewportWidth))
	script = strings.ReplaceAll(script, "__VIEWPORT_HEIGHT__", strconv.Itoa(viewportHeight))
	script = strings.ReplaceAll(script, "__DEFAULT_TIMEOUT__", strconv.Itoa(defaultTimeoutMillis))
	return script, nil
}

const stepScriptTemplate = `export default async ({ page, context }) => {
  const step = __INSTRUCTION__;
  const params = step.params || {};
  const stepName = params.name || '';
  const runStart = Date.now();
  const preloadHTML = typeof step.preloadHtml === 'string' ? step.preloadHtml : '';

  if (!page.__basSetup) {
    await page.setViewport({ width: __VIEWPORT_WIDTH__, height: __VIEWPORT_HEIGHT__ });

    page.__basSetup = {
      consoleLogs: [],
      networkEvents: [],
      consoleCursor: 0,
      networkCursor: 0,
    };

    const telemetry = page.__basSetup;

    page.on('console', (msg) => {
      try {
        telemetry.consoleLogs.push({
          type: msg.type(),
          text: msg.text(),
          timestamp: Date.now(),
        });
      } catch (err) {
        telemetry.consoleLogs.push({
          type: 'error',
          text: 'console serialization failed: ' + err,
          timestamp: Date.now(),
        });
      }
    });

    page.on('request', (request) => {
      try {
        telemetry.networkEvents.push({
          type: 'request',
          url: request.url(),
          method: request.method(),
          resourceType: typeof request.resourceType === 'function' ? request.resourceType() : undefined,
          timestamp: Date.now(),
        });
      } catch (err) {}
    });

    page.on('response', (response) => {
      try {
        telemetry.networkEvents.push({
          type: 'response',
          url: response.url(),
          status: response.status(),
          ok: response.ok(),
          timestamp: Date.now(),
        });
      } catch (err) {}
    });

    page.on('requestfailed', (request) => {
      try {
        const failure = typeof request.failure === 'function' ? request.failure() : null;
        telemetry.networkEvents.push({
          type: 'requestfailed',
          url: request.url(),
          method: request.method(),
          failure: failure && (failure.errorText || String(failure)),
          timestamp: Date.now(),
        });
      } catch (err) {}
    });
  }

  const telemetry = page.__basSetup;

  const waitForTime = async (durationMs) => {
    const duration = Number.isFinite(durationMs) ? Math.max(durationMs, 0) : 0;
    if (typeof page.waitForTimeout === 'function') {
      await page.waitForTimeout(duration);
      return;
    }
    await new Promise((resolve) => setTimeout(resolve, duration));
  };

  const resolveTimeout = (candidate, fallback) => {
    if (Number.isFinite(candidate) && candidate > 0) {
      return candidate;
    }
    if (Number.isFinite(fallback) && fallback > 0) {
      return fallback;
    }
    return __DEFAULT_TIMEOUT__;
  };

  const hasPrecondition = typeof params.preconditionSelector === 'string' && params.preconditionSelector.trim().length > 0;
  const hasSuccessCheck = typeof params.successSelector === 'string' && params.successSelector.trim().length > 0;

  async function ensurePreconditionReady() {
    if (!hasPrecondition) {
      return;
    }
    const timeout = resolveTimeout(params.preconditionTimeoutMs, params.timeoutMs);
    const element = await page
      .waitForSelector(params.preconditionSelector, { timeout, state: 'visible' })
      .catch(() => null);
    if (!element) {
      throw new Error('Precondition selector ' + params.preconditionSelector + ' not found');
    }
    if (typeof element.dispose === 'function') {
      await element.dispose();
    }
    if (Number.isFinite(params.preconditionWaitMs) && params.preconditionWaitMs > 0) {
      await waitForTime(params.preconditionWaitMs);
    }
  }

  async function verifySuccessState() {
    if (!hasSuccessCheck) {
      return;
    }
    const timeout = resolveTimeout(params.successTimeoutMs, params.timeoutMs);
    await page.waitForSelector(params.successSelector, { timeout });
    if (Number.isFinite(params.successWaitMs) && params.successWaitMs > 0) {
      await waitForTime(params.successWaitMs);
    }
  }

  // Rehydration must run in the page context to access document/window
  if (preloadHTML) {
    try {
      await page.evaluate((htmlContent) => {
        try {
          // Check if we're at about:blank - if so, always inject regardless of existing content
          const currentUrl = window.location.href;
          const isBlankPage = currentUrl === 'about:blank' || currentUrl === '';

          console.log('[BAS][rehydrate] URL:', currentUrl, 'isBlank:', isBlankPage, 'contentLength:', document.documentElement.innerHTML.length);

          const parser = new DOMParser();
          const parsed = parser.parseFromString(htmlContent, 'text/html');
          if (!parsed || !parsed.documentElement) {
            console.log('[BAS][rehydrate] Parse failed');
            return;
          }

          const newDoc = parsed.documentElement;
          const current = document.documentElement;

          // Skip injection only if we have content AND we're not at about:blank
          // This allows us to override browserless loader HTML when starting fresh
          if (!isBlankPage && current.innerHTML && current.innerHTML.trim().length > 0) {
            console.log('[BAS][rehydrate] Skipping - has content and not blank');
            return;
          }

          console.log('[BAS][rehydrate] Injecting', htmlContent.length, 'bytes');

          // Copy attributes
          const attrs = current.getAttributeNames();
          for (const name of attrs) {
            if (!newDoc.hasAttribute(name)) {
              current.removeAttribute(name);
            }
          }
          for (const attr of Array.from(newDoc.attributes || [])) {
            current.setAttribute(attr.name, attr.value);
          }

          // Inject content
          current.innerHTML = newDoc.innerHTML;
          console.log('[BAS][rehydrate] Complete - new length:', current.innerHTML.length);
        } catch (err) {
          console.log('[BAS][rehydrate] Error:', err.message);
        }
      }, preloadHTML);
    } catch (err) {
      // Rehydration is best-effort, continue even if it fails
    }
  }

  try {
    const currentUrl = typeof page.url === 'function' ? page.url() : 'unknown';
    console.log('[BAS][debug] step start url:', currentUrl);
    console.log('[BAS][debug] preloadHTML length:', preloadHTML ? preloadHTML.length : 0);
  } catch {}

  const flushTelemetry = () => {
    const consoleLogs = telemetry.consoleLogs.slice(telemetry.consoleCursor);
    const networkEvents = telemetry.networkEvents.slice(telemetry.networkCursor);
    telemetry.consoleCursor = telemetry.consoleLogs.length;
    telemetry.networkCursor = telemetry.networkEvents.length;
    return { consoleLogs, networkEvents };
  };

  const stepStart = Date.now();
  const baseResult = { index: step.index, nodeId: step.nodeId, type: step.type, stepName };
  let extras = {};

  async function ensureDocumentHydrated() {
    const maxAttempts = 4;
    let hydrationSucceeded = false;
    for (let attempt = 0; attempt < maxAttempts; attempt += 1) {
      let result = null;
      try {
        result = await page.evaluate((state) => {
          const serializeError = (error) => {
            if (!error) {
              return undefined;
            }
            if (typeof error === 'string') {
              return error;
            }
            if (error && typeof error.message === 'string') {
              return error.message;
            }
            try {
              return String(error);
            } catch (_) {
              return 'Unknown error';
            }
          };

          try {
            const doc = document;
            const body = doc && doc.body ? doc.body : null;
            const parseColor = (value) => {
              if (!value || typeof value !== 'string') {
                return null;
              }
              if (value.toLowerCase() === 'transparent') {
                return { r: 0, g: 0, b: 0, a: 0 };
              }
              const rgbaMatch = value.match(/rgba?\(([^)]+)\)/i);
              if (rgbaMatch) {
                const parts = rgbaMatch[1].split(',').map((part) => part.trim());
                const r = Number.parseFloat(parts[0]);
                const g = Number.parseFloat(parts[1]);
                const b = Number.parseFloat(parts[2]);
                const a = parts[3] != null ? Number.parseFloat(parts[3]) : 1;
                if ([r, g, b].some((component) => Number.isNaN(component))) {
                  return null;
                }
                return { r, g, b, a: Number.isNaN(a) ? 1 : a };
              }
              if (value.startsWith('#')) {
                let hex = value.slice(1);
                if (hex.length === 3 || hex.length === 4) {
                  hex = hex.split('').map((ch) => ch + ch).join('');
                }
                if (hex.length === 6 || hex.length === 8) {
                  const r = Number.parseInt(hex.slice(0, 2), 16);
                  const g = Number.parseInt(hex.slice(2, 4), 16);
                  const b = Number.parseInt(hex.slice(4, 6), 16);
                  const a = hex.length === 8 ? Number.parseInt(hex.slice(6, 8), 16) / 255 : 1;
                  if ([r, g, b].some((component) => Number.isNaN(component))) {
                    return null;
                  }
                  return { r, g, b, a };
                }
              }
              return null;
            };

            const hasMeaningfulContent = () => {
              if (!body) {
                return false;
              }

              const globalStyle = (typeof window !== 'undefined' && typeof window.getComputedStyle === 'function')
                ? window.getComputedStyle(body)
                : null;
              if (globalStyle) {
                const bgColor = parseColor(globalStyle.backgroundColor);
                if (bgColor && bgColor.a > 0.05 && (bgColor.r < 245 || bgColor.g < 245 || bgColor.b < 245)) {
                  return true;
                }
              }

              const text = typeof body.innerText === 'string' ? body.innerText.trim() : '';
              if (text.length > 0) {
                return true;
              }

              const elements = Array.from(body.querySelectorAll('*'));
              for (const element of elements) {
                if (!element || typeof element.tagName !== 'string') {
                  continue;
                }
                const tag = element.tagName.toLowerCase();
                if (['script', 'style', 'meta', 'link', 'head', 'title', 'noscript', 'template'].includes(tag)) {
                  continue;
                }

                if (['img', 'svg', 'video', 'canvas', 'iframe', 'picture'].includes(tag)) {
                  const mediaRect = typeof element.getBoundingClientRect === 'function'
                    ? element.getBoundingClientRect()
                    : null;
                  if (mediaRect && mediaRect.width >= 3 && mediaRect.height >= 3) {
                    return true;
                  }
                }

                const rect = typeof element.getBoundingClientRect === 'function'
                  ? element.getBoundingClientRect()
                  : null;
                if (!rect || rect.width < 6 || rect.height < 6) {
                  continue;
                }

                if (typeof element.innerText === 'string' && element.innerText.trim().length > 0) {
                  return true;
                }

                const style = (typeof window !== 'undefined' && typeof window.getComputedStyle === 'function')
                  ? window.getComputedStyle(element)
                  : null;
                if (!style) {
                  continue;
                }

                if (style.backgroundImage && style.backgroundImage !== 'none') {
                  return true;
                }
                if (style.boxShadow && style.boxShadow !== 'none') {
                  return true;
                }

                const candidateColors = [
                  parseColor(style.backgroundColor),
                  parseColor(style.borderTopColor),
                  parseColor(style.borderRightColor),
                  parseColor(style.borderBottomColor),
                  parseColor(style.borderLeftColor),
                  parseColor(style.color),
                ].filter(Boolean);

                if (candidateColors.some((color) => color.a > 0.05 && (color.r < 245 || color.g < 245 || color.b < 245))) {
                  return true;
                }
              }

              return false;
            };

            if (hasMeaningfulContent()) {
              return { hasVisualContent: true, rehydrated: false };
            }

            const htmlCandidates = [];
            const pushCandidate = (value) => {
              if (!value || typeof value !== 'string' || value.length === 0) {
                return;
              }
              if (!htmlCandidates.includes(value)) {
                htmlCandidates.push(value);
              }
            };

            if (state && typeof state.lastHtml === 'string' && state.lastHtml.length > 0) {
              pushCandidate(state.lastHtml);
            }
            if (state && typeof state.lastNavHtml === 'string' && state.lastNavHtml.length > 0) {
              pushCandidate(state.lastNavHtml);
            }

            if (typeof window !== 'undefined') {
              if (typeof window.__BAS_LAST_REPLAY_HTML__ === 'string' && window.__BAS_LAST_REPLAY_HTML__.length > 0) {
                pushCandidate(window.__BAS_LAST_REPLAY_HTML__);
              }
              if (typeof window.__BAS_LAST_NAV_HTML__ === 'string' && window.__BAS_LAST_NAV_HTML__.length > 0) {
                pushCandidate(window.__BAS_LAST_NAV_HTML__);
              }
            }

            for (const html of htmlCandidates) {
              try {
                if (!html || html.length === 0) {
                  continue;
                }
                const parser = typeof DOMParser === 'function' ? new DOMParser() : null;
                if (parser) {
                  const parsed = parser.parseFromString(html, 'text/html');
                  if (parsed && parsed.documentElement) {
                    doc.documentElement.innerHTML = parsed.documentElement.innerHTML;
                  } else {
                    doc.documentElement.innerHTML = html;
                  }
                } else {
                  doc.documentElement.innerHTML = html;
                }

                if (hasMeaningfulContent()) {
                  return { hasVisualContent: true, rehydrated: true };
                }
              } catch (rehydrateErr) {
                return { hasVisualContent: false, rehydrated: false, error: serializeError(rehydrateErr) };
              }
            }

            return { hasVisualContent: false, rehydrated: false };
          } catch (err) {
            return { hasVisualContent: false, rehydrated: false, error: serializeError(err) };
          }
        }, {
          lastHtml: page.__basLastReplayHTML || '',
          lastNavHtml: page.__basLastNavHTML || '',
        });
      } catch (executionError) {
        result = {
          hasVisualContent: false,
          rehydrated: false,
          error: (executionError && executionError.message) ? executionError.message : String(executionError),
        };
      }

      if (result && result.error) {
        try {
          console.warn('[BAS][replay] hydration probe warning:', result.error);
        } catch (_) {}
      }

      if (result && result.hasVisualContent) {
        hydrationSucceeded = true;
        break;
      }

      const delayBase = result && result.rehydrated ? 160 : 110;
      const attemptDelay = delayBase + (attempt * 70);
      await waitForTime(attemptDelay);
    }

    try {
      await page.evaluate(() => {
        if (typeof requestAnimationFrame !== 'function') {
          return null;
        }
        return new Promise((resolve) => {
          requestAnimationFrame(() => {
            requestAnimationFrame(() => resolve(null));
          });
        });
      });
    } catch (_) {}

    if (hydrationSucceeded) {
      return { hasVisualContent: true };
    }

    const fallbackScreenshot = typeof page.__basLastReplayScreenshotBase64 === 'string'
      && page.__basLastReplayScreenshotBase64.length > 0
      ? page.__basLastReplayScreenshotBase64
      : null;

    return { hasVisualContent: false, fallbackScreenshot };
  }

  async function storeReplaySnapshot(sourceTag = '') {
    try {
      const captureHtml = await page.content();
      const captureUrl = typeof page.url === 'function' ? page.url() : '';

      if (!captureHtml || typeof captureHtml !== 'string' || captureHtml.trim().length === 0) {
        return;
      }

      page.__basLastReplayHTML = captureHtml;
      page.__basLastReplayURL = captureUrl;
      if (sourceTag === 'navigate') {
        page.__basLastNavHTML = captureHtml;
        page.__basLastNavURL = captureUrl;
      }

      try {
        await page.evaluate((state) => {
          try {
            if (typeof window !== 'undefined') {
              window.__BAS_LAST_REPLAY_HTML__ = state.html;
              window.__BAS_LAST_REPLAY_URL__ = state.url;
              if (state.source === 'navigate') {
                window.__BAS_LAST_NAV_HTML__ = state.html;
                window.__BAS_LAST_NAV_URL__ = state.url;
              }
            }
          } catch (_) {}
        }, { html: captureHtml, url: captureUrl, source: sourceTag });
      } catch (_) {}
    } catch (_) {}
  }

  const shouldCaptureReplay = () => {
    const type = (step && typeof step.type === 'string') ? step.type.toLowerCase() : '';
    if (!type) {
      return false;
    }
    if (!page || typeof page.screenshot !== 'function') {
      return false;
    }
    if (type === 'screenshot') {
      return false;
    }
    return true;
  };

  if (!Array.isArray(page.__basReplayHistory)) {
    page.__basReplayHistory = [];
  }

  const calculateByteEntropy = (buffer) => {
    if (!buffer || typeof buffer.length !== 'number' || buffer.length === 0) {
      return 0;
    }
    const counts = new Array(256).fill(0);
    for (let i = 0; i < buffer.length; i += 1) {
      counts[buffer[i]] += 1;
    }
    const total = buffer.length;
    let entropy = 0;
    for (let i = 0; i < counts.length; i += 1) {
      const count = counts[i];
      if (count === 0) {
        continue;
      }
      const probability = count / total;
      entropy -= probability * Math.log2(probability);
    }
    return entropy;
  };

  const LOW_ENTROPY_THRESHOLD = 3.2;

  const ensureReplayScreenshot = async () => {
    if (!shouldCaptureReplay()) {
      return null;
    }
    if (extras && typeof extras === 'object' && Object.prototype.hasOwnProperty.call(extras, 'screenshotBase64') && extras.screenshotBase64) {
      page.__basLastReplayScreenshotBase64 = extras.screenshotBase64;
      try {
        await storeReplaySnapshot();
      } catch (_) {}
      return extras.screenshotBase64;
    }

    let usingFallback = false;
    const hydration = await ensureDocumentHydrated();
    const screenshotHistory = Array.isArray(page.__basReplayHistory) ? page.__basReplayHistory : [];
    const lastScreenshotBase64 = screenshotHistory.length > 0
      ? screenshotHistory[screenshotHistory.length - 1]
      : null;
    if (hydration && hydration.hasVisualContent === false && hydration.fallbackScreenshot) {
      extras = { ...extras, screenshotBase64: hydration.fallbackScreenshot };
      return hydration.fallbackScreenshot;
    }

    let capture = null;
    try {
      capture = await page.screenshot({ encoding: 'base64', fullPage: true });
    } catch (captureError) {
      try {
        console.warn('[BAS][replay] failed to capture replay screenshot:', captureError);
      } catch (_) {}
    }

    if (capture && capture.length) {
      try {
        const entropy = calculateByteEntropy(Buffer.from(capture, 'base64'));
        if (!Number.isNaN(entropy) && entropy > 0 && entropy < LOW_ENTROPY_THRESHOLD && lastScreenshotBase64) {
          capture = lastScreenshotBase64;
          usingFallback = true;
        }
      } catch (_) {}
    }

    if ((!capture || !capture.length) && lastScreenshotBase64) {
      capture = lastScreenshotBase64;
      usingFallback = true;
    }

    if (capture && capture.length) {
      extras = { ...extras, screenshotBase64: capture };
      if (usingFallback) {
        if (lastScreenshotBase64) {
          screenshotHistory.push(lastScreenshotBase64);
        }
      } else {
        page.__basLastReplayScreenshotBase64 = capture;
        screenshotHistory.push(capture);
        try {
          await storeReplaySnapshot();
        } catch (_) {}
      }
      return capture;
    }

    return null;
  };

  try {
    if (hasPrecondition) {
      await ensurePreconditionReady();
    }

    switch (step.type) {
      case 'navigate': {
        if (!params.url) {
          throw new Error('navigate step missing url');
        }
        const waitUntil = params.waitUntil || 'networkidle2';
        const timeout = Number.isFinite(params.timeoutMs) ? params.timeoutMs : __DEFAULT_TIMEOUT__;
        const rawUrl = params.url;
        const lowerUrl = typeof rawUrl === 'string' ? rawUrl.toLowerCase() : '';
        let navigated = false;
        let finalUrl = '';

        if (lowerUrl.startsWith('data:text/html')) {
          const extractDataUrlPayload = (value) => {
            try {
              const commaIndex = value.indexOf(',');
              if (commaIndex === -1) {
                return null;
              }
              const meta = value.slice(0, commaIndex);
              const dataPart = value.slice(commaIndex + 1);
              const isBase64 = /;base64/i.test(meta);
              if (isBase64) {
                if (typeof Buffer !== 'undefined' && typeof Buffer.from === 'function') {
                  return Buffer.from(dataPart, 'base64').toString('utf8');
                }
                if (typeof atob === 'function') {
                  const binary = atob(dataPart);
                  let decoded = '';
                  for (let i = 0; i < binary.length; i += 1) {
                    decoded += String.fromCharCode(binary.charCodeAt(i));
                  }
                  return decoded;
                }
                return null;
              }
              try {
                return decodeURIComponent(dataPart);
              } catch {
                return dataPart;
              }
            } catch {
              return null;
    }
  };

          const htmlPayload = extractDataUrlPayload(rawUrl);
          if (htmlPayload && htmlPayload.trim().length > 0) {
            try {
              await page.goto('about:blank', { waitUntil: 'domcontentloaded', timeout });
              await page.evaluate((html) => {
                const parser = typeof DOMParser === 'function' ? new DOMParser() : null;
                if (!parser) {
                  document.body.innerHTML = html;
                  return;
                }
                const parsed = parser.parseFromString(html, 'text/html');
                if (!parsed || !parsed.documentElement) {
                  document.body.innerHTML = html;
                  return;
                }
                const newDocumentElement = parsed.documentElement;
                const currentDocumentElement = document.documentElement;
                const currentAttributes = currentDocumentElement.getAttributeNames();
                for (const name of currentAttributes) {
                  if (!newDocumentElement.hasAttribute(name)) {
                    currentDocumentElement.removeAttribute(name);
                  }
                }
                for (const attr of Array.from(newDocumentElement.attributes || [])) {
                  currentDocumentElement.setAttribute(attr.name, attr.value);
                }
                currentDocumentElement.innerHTML = newDocumentElement.innerHTML;
              }, htmlPayload);
              await page.evaluate((html, url) => {
                try {
                  window.__BAS_LAST_NAV_HTML__ = html;
                  window.__BAS_LAST_NAV_URL__ = url;
                } catch {}
              }, htmlPayload, rawUrl);
              navigated = true;
              finalUrl = rawUrl;
            } catch (setContentError) {
              navigated = false;
            }
          }
        }

        if (!navigated) {
          await page.goto(rawUrl, { waitUntil, timeout });
          navigated = true;
          finalUrl = page.url();
          await page.evaluate((url) => {
            try {
              window.__BAS_LAST_NAV_HTML__ = null;
              window.__BAS_LAST_NAV_URL__ = url;
            } catch {}
          }, rawUrl);
        }

        if (params.waitForMs) {
          await waitForTime(params.waitForMs);
        }
        if (navigated) {
          try {
            const bodyPreview = await page.evaluate(() => {
              if (!document || !document.body) {
                return null;
              }
              const html = document.body.innerHTML || '';
              return html.length > 200 ? html.slice(0, 200) : html;
            });
            extras = { finalUrl: finalUrl || page.url() || rawUrl, bodyPreview };
          } catch {
            extras = { finalUrl: finalUrl || page.url() || rawUrl };
          }
        } else {
          extras = { finalUrl: finalUrl || page.url() || rawUrl };
        }
        // Capture DOM snapshot for next step's preload
        try {
          const domSnapshot = await page.content();
          if (typeof domSnapshot === 'string' && domSnapshot.length > 0) {
            extras.domSnapshot = domSnapshot;
          }
        } catch (snapshotError) {
          // DOM snapshot is optional, continue without it
        }
        await storeReplaySnapshot('navigate');
        break;
      }
      case 'wait': {
        const waitType = params.waitType || 'time';
        if (waitType === 'element') {
          if (!params.selector) {
            throw new Error('wait element step missing selector');
          }
          const timeout = Number.isFinite(params.timeoutMs) ? params.timeoutMs : __DEFAULT_TIMEOUT__;
          await page.evaluate((selector) => {
            try {
              const exists = document && typeof document.querySelector === 'function' && document.querySelector(selector);
              if (!exists && typeof window !== 'undefined') {
                const html = typeof window.__BAS_LAST_NAV_HTML__ === 'string' ? window.__BAS_LAST_NAV_HTML__ : null;
                if (html && html.length > 0) {
                  const parser = typeof DOMParser === 'function' ? new DOMParser() : null;
                  if (parser) {
                    const parsed = parser.parseFromString(html, 'text/html');
                    if (parsed && parsed.documentElement) {
                      document.documentElement.innerHTML = parsed.documentElement.innerHTML;
                    }
                  } else if (document && document.body) {
                    document.body.innerHTML = html;
                  }
                }
              }
            } catch {}
          }, params.selector);
          let elementHandle = null;
          let primaryError = null;
          try {
            elementHandle = await page.waitForSelector(params.selector, { timeout });
          } catch (error) {
            primaryError = error;
            const startedAt = Date.now();
            const pollDelay = Math.min(Math.max(Math.floor(timeout / 20), 50), 500);
            while ((Date.now() - startedAt) < timeout) {
              try {
                const exists = await page.evaluate((selector) => !!document.querySelector(selector), params.selector);
                if (exists) {
                  primaryError = null;
                  break;
                }
              } catch (evalError) {
                primaryError = evalError;
              }
              await waitForTime(pollDelay);
            }
            if (primaryError) {
              throw primaryError;
            }
          }
          if (elementHandle && typeof elementHandle.dispose === 'function') {
            await elementHandle.dispose();
          }
        } else if (waitType === 'navigation') {
          const timeout = Number.isFinite(params.timeoutMs) ? params.timeoutMs : __DEFAULT_TIMEOUT__;
          await page.waitForNavigation({ timeout });
        } else {
          const duration = Number.isFinite(params.durationMs) ? Math.max(params.durationMs, 0) : 1000;
          await waitForTime(duration);
        }
        break;
      }
      case 'click': {
        if (!params.selector) {
          throw new Error('click step missing selector');
        }
        const timeout = Number.isFinite(params.timeoutMs) ? params.timeoutMs : __DEFAULT_TIMEOUT__;
        const element = await page.waitForSelector(params.selector, { timeout, state: 'visible' });
        if (!element) {
          throw new Error('click step element not found');
        }
        const boundingBox = await element.boundingBox();
        const clickOptions = {};
        if (typeof params.button === 'string' && params.button !== 'left') {
          clickOptions.button = params.button;
        }
        if (Number.isFinite(params.clickCount) && params.clickCount > 1) {
          clickOptions.clickCount = params.clickCount;
        }
        await element.click(clickOptions);
        if (params.waitForMs) {
          await waitForTime(params.waitForMs);
        }
        if (params.waitForSelector) {
          await page.waitForSelector(params.waitForSelector, { timeout });
        }
        if (boundingBox) {
          extras = {
            elementBoundingBox: {
              x: boundingBox.x,
              y: boundingBox.y,
              width: boundingBox.width,
              height: boundingBox.height,
            },
            clickPosition: {
              x: boundingBox.x + (boundingBox.width / 2),
              y: boundingBox.y + (boundingBox.height / 2),
            },
          };
        } else {
          extras = {};
        }
        if (typeof element.dispose === 'function') {
          await element.dispose();
        }
        break;
      }
      case 'type': {
        if (!params.selector) {
          throw new Error('type step missing selector');
        }
        const timeout = Number.isFinite(params.timeoutMs) ? params.timeoutMs : __DEFAULT_TIMEOUT__;
        const element = await page.waitForSelector(params.selector, { timeout, state: 'visible' });
        if (!element) {
          throw new Error('type step element not found');
        }
        const boundingBox = await element.boundingBox();
        if (params.clear === true && typeof element.fill === 'function') {
          await element.fill('');
        }
        if (typeof params.text === 'string') {
          if (Number.isFinite(params.delayMs) && params.delayMs > 0 && typeof element.type === 'function') {
            await element.type(params.text, { delay: params.delayMs });
          } else if (typeof element.fill === 'function') {
            await element.fill(params.text);
          } else {
            await page.type(params.selector, params.text);
          }
        } else if (typeof element.focus === 'function') {
          await element.focus();
        }
        if (params.submit) {
          await element.press('Enter');
        }
        extras = {};
        if (boundingBox) {
          extras = {
            elementBoundingBox: {
              x: boundingBox.x,
              y: boundingBox.y,
              width: boundingBox.width,
              height: boundingBox.height,
            },
          };
        }
        if (typeof element.dispose === 'function') {
          await element.dispose();
        }
        break;
      }
      case 'shortcut': {
        const rawKeys = Array.isArray(params.shortcutKeys)
          ? params.shortcutKeys.filter((value) => typeof value === 'string').map((value) => value.trim()).filter(Boolean)
          : [];
        if (!rawKeys.length) {
          throw new Error('shortcut step missing shortcuts');
        }

        const normalizeShortcut = (value) => {
          if (typeof value !== 'string') {
            return '';
          }
          const segments = value.split('+').map((segment) => segment.trim()).filter(Boolean);
          return segments.length ? segments.join('+') : '';
        };

        const normalizedKeys = rawKeys
          .map((combo) => normalizeShortcut(combo))
          .filter((combo, index, array) => combo && array.indexOf(combo) === index);

        if (!normalizedKeys.length) {
          throw new Error('shortcut step has no valid shortcuts');
        }

        const timeout = Number.isFinite(params.timeoutMs) ? params.timeoutMs : __DEFAULT_TIMEOUT__;
        let elementHandle = null;
        if (typeof params.focusSelector === 'string' && params.focusSelector.trim().length > 0) {
          elementHandle = await page.waitForSelector(params.focusSelector, { timeout, state: 'attached' }).catch(() => null);
          if (!elementHandle) {
            throw new Error('shortcut focus selector not found');
          }
          try {
            if (typeof elementHandle.scrollIntoViewIfNeeded === 'function') {
              await elementHandle.scrollIntoViewIfNeeded();
            } else {
              await elementHandle.evaluate((element) => element.scrollIntoView({ block: 'center', inline: 'center', behavior: 'instant' }));
            }
            if (typeof elementHandle.focus === 'function') {
              await elementHandle.focus();
            } else if (typeof params.focusSelector === 'string' && params.focusSelector.trim().length > 0) {
              await page.focus(params.focusSelector).catch(() => {});
            }
          } catch (_) {}
        }

        const delay = Number.isFinite(params.shortcutDelayMs) ? Math.max(params.shortcutDelayMs, 0) : 0;
        for (const combo of normalizedKeys) {
          if (!combo) {
            continue;
          }
          if (delay > 0) {
            await page.keyboard.press(combo, { delay });
          } else {
            await page.keyboard.press(combo);
          }
        }

        if (params.waitForMs) {
          await waitForTime(params.waitForMs);
        }

        extras = { shortcuts: normalizedKeys };
        if (elementHandle) {
          const boundingBox = await elementHandle.boundingBox().catch(() => null);
          if (boundingBox) {
            extras.elementBoundingBox = {
              x: boundingBox.x,
              y: boundingBox.y,
              width: boundingBox.width,
              height: boundingBox.height,
            };
          }
          if (typeof elementHandle.dispose === 'function') {
            await elementHandle.dispose();
          }
        }
        break;
      }
      case 'probeElements': {
        const rawX = Number.isFinite(params.probeX) ? params.probeX : 0;
        const rawY = Number.isFinite(params.probeY) ? params.probeY : 0;
        const probeRadius = Number.isFinite(params.probeRadius) ? Math.max(1, Math.min(params.probeRadius, 48)) : 6;
        const probeSamples = Number.isFinite(params.probeSamples) ? Math.max(4, Math.min(params.probeSamples, 64)) : 24;

        await ensureDocumentHydrated();

        const probeResult = await page.evaluate(async (payload) => {
          const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms));
          const clamp = (value, min, max) => Math.min(Math.max(value, min), max);

          const waitForReady = async () => {
            const deadline = Date.now() + 5000;
            while (Date.now() < deadline) {
              const ready = document.readyState;
              const hasBody = document.body && document.body.children && document.body.children.length > 0;
              if ((ready === 'interactive' || ready === 'complete') && hasBody) {
                return true;
              }
              await sleep(120);
            }
            return false;
          };

          await waitForReady();

          const devicePixelRatio = Number.isFinite(window.devicePixelRatio) && window.devicePixelRatio > 0 ? window.devicePixelRatio : 1;
          const baseX = payload.rawX / devicePixelRatio;
          const baseY = payload.rawY / devicePixelRatio;

          const getViewport = () => ({
            width: window.innerWidth || document.documentElement.clientWidth || document.body.clientWidth || 0,
            height: window.innerHeight || document.documentElement.clientHeight || document.body.clientHeight || 0,
            scrollWidth: document.documentElement.scrollWidth || document.body.scrollWidth || 0,
            scrollHeight: document.documentElement.scrollHeight || document.body.scrollHeight || 0,
          });

          const getScroll = () => ({
            x: window.scrollX || window.pageXOffset || document.documentElement.scrollLeft || document.body.scrollLeft || 0,
            y: window.scrollY || window.pageYOffset || document.documentElement.scrollTop || document.body.scrollTop || 0,
          });

          const ensurePointVisible = () => {
            const viewport = getViewport();
            const scroll = getScroll();
            if (viewport.width <= 0 || viewport.height <= 0) {
              return;
            }

            let nextX = scroll.x;
            let nextY = scroll.y;

            if (baseX < scroll.x || baseX > scroll.x + viewport.width) {
              const maxX = Math.max(viewport.scrollWidth - viewport.width, 0);
              nextX = clamp(baseX - viewport.width / 2, 0, maxX);
            }

            if (baseY < scroll.y || baseY > scroll.y + viewport.height) {
              const maxY = Math.max(viewport.scrollHeight - viewport.height, 0);
              nextY = clamp(baseY - viewport.height / 2, 0, maxY);
            }

            if (nextX !== scroll.x || nextY !== scroll.y) {
              window.scrollTo(nextX, nextY);
            }
          };

          ensurePointVisible();
          await sleep(50);

          const viewportMetrics = getViewport();
          const scrollMetrics = getScroll();
          const viewportX = viewportMetrics.width > 0
            ? clamp(baseX - scrollMetrics.x, 0, Math.max(viewportMetrics.width - 1, 0))
            : baseX - scrollMetrics.x;
          const viewportY = viewportMetrics.height > 0
            ? clamp(baseY - scrollMetrics.y, 0, Math.max(viewportMetrics.height - 1, 0))
            : baseY - scrollMetrics.y;

          const ordered = [];
          const seen = new Set();

          const push = (element) => {
            if (!element || element.nodeType !== Node.ELEMENT_NODE || seen.has(element)) {
              return;
            }
            seen.add(element);
            ordered.push(element);
          };

          const addParents = (element) => {
            let current = element ? element.parentElement : null;
            let guard = 0;
            while (current && guard < 25) {
              push(current);
              current = current.parentElement;
              guard += 1;
            }
          };

          const addShadowContents = (element, pointX, pointY) => {
            if (!element || !element.shadowRoot) {
              return;
            }
            const shadowElements = Array.from(element.shadowRoot.elementsFromPoint(pointX, pointY) || []);
            for (const nested of shadowElements) {
              push(nested);
              addShadowContents(nested, pointX, pointY);
              addParents(nested);
            }
          };

          const gatherFromPoint = (pointX, pointY) => {
            const elements = Array.from(document.elementsFromPoint(pointX, pointY) || []);
            for (const element of elements) {
              push(element);
              addShadowContents(element, pointX, pointY);
              addParents(element);
            }
          };

          gatherFromPoint(viewportX, viewportY);

          const isMeaningful = (element) => {
            if (!element || element.nodeType !== Node.ELEMENT_NODE) {
              return false;
            }
            const tag = element.tagName ? element.tagName.toLowerCase() : '';
            return tag && tag !== 'html' && tag !== 'body';
          };

          const buildOffsets = () => {
            const offsets = [{ dx: 0, dy: 0 }];
            const sampleLimit = Math.max(1, Math.min(64, payload.samples || 24));
            const maxDistance = Math.max(1, Math.min(16, payload.radius || 6));
            for (let dist = 1; dist <= maxDistance && offsets.length < sampleLimit; dist += 1) {
              const combos = [
                { dx: dist, dy: 0 },
                { dx: -dist, dy: 0 },
                { dx: 0, dy: dist },
                { dx: 0, dy: -dist },
                { dx: dist, dy: dist },
                { dx: dist, dy: -dist },
                { dx: -dist, dy: dist },
                { dx: -dist, dy: -dist },
              ];
              for (const combo of combos) {
                offsets.push(combo);
                if (offsets.length >= sampleLimit) {
                  break;
                }
              }
            }
            return offsets;
          };

          if (!ordered.some(isMeaningful)) {
            const offsets = buildOffsets();
            const clampX = (value) => viewportMetrics.width > 0 ? clamp(value, 0, Math.max(viewportMetrics.width - 1, 0)) : value;
            const clampY = (value) => viewportMetrics.height > 0 ? clamp(value, 0, Math.max(viewportMetrics.height - 1, 0)) : value;
            for (const offset of offsets) {
              const sampleX = clampX(viewportX + offset.dx);
              const sampleY = clampY(viewportY + offset.dy);
              gatherFromPoint(sampleX, sampleY);
              if (ordered.some(isMeaningful)) {
                break;
              }
            }
          }

          const getScrollAwareBox = (element) => {
            const rect = element.getBoundingClientRect();
            if (!rect || rect.width <= 0 || rect.height <= 0) {
              return null;
            }
            const scroll = getScroll();
            return {
              x: rect.x + scroll.x,
              y: rect.y + scroll.y,
              width: rect.width,
              height: rect.height,
            };
          };

          const collectBoundingMatches = () => {
            const results = [];
            const stack = [{ element: document.documentElement, depth: 0 }];
            const visited = new Set();
            let scanned = 0;
            const maxScanned = 6000;
            while (stack.length > 0 && scanned < maxScanned) {
              const current = stack.pop();
              scanned += 1;
              if (!current || !current.element || current.element.nodeType !== Node.ELEMENT_NODE) {
                continue;
              }
              const element = current.element;
              if (visited.has(element)) {
                continue;
              }
              visited.add(element);

              const rect = getScrollAwareBox(element);
              if (rect) {
                if (baseX >= rect.x && baseX <= rect.x + rect.width && baseY >= rect.y && baseY <= rect.y + rect.height) {
                  results.push({ element, rect, depth: current.depth });
                }
              }

              const children = element.children;
              if (children && children.length > 0) {
                for (let i = children.length - 1; i >= 0; i -= 1) {
                  stack.push({ element: children[i], depth: current.depth + 1 });
                }
              }

              if (element.shadowRoot && element.shadowRoot.children) {
                const shadowChildren = element.shadowRoot.children;
                for (let i = shadowChildren.length - 1; i >= 0; i -= 1) {
                  stack.push({ element: shadowChildren[i], depth: current.depth + 1 });
                }
              }
            }

            results.sort((a, b) => {
              if (b.depth !== a.depth) {
                return b.depth - a.depth;
              }
              const areaA = a.rect.width * a.rect.height;
              const areaB = b.rect.width * b.rect.height;
              return areaA - areaB;
            });

            return results.slice(0, 80);
          };

          if (!ordered.length || ordered.every((element) => !isMeaningful(element))) {
            const boundingHits = collectBoundingMatches();
            for (const hit of boundingHits) {
              if (!hit || !hit.element) {
                continue;
              }
              push(hit.element);
              addParents(hit.element);
            }
          }

          if (!ordered.length) {
            const fallback = document.body || document.documentElement;
            if (fallback) {
              push(fallback);
              addParents(fallback);
            }
          }

          const calculateConfidence = (element) => {
            let confidence = 0.1;
            const rect = element.getBoundingClientRect();
            if (rect.width > 0 && rect.height > 0) confidence += 0.3;
            if (element.textContent && element.textContent.trim()) confidence += 0.2;
            const tagName = element.tagName.toLowerCase();
            if ([ 'button', 'input', 'select', 'textarea' ].includes(tagName)) confidence += 0.3;
            if (tagName === 'a' && typeof element.href === 'string' && element.href) confidence += 0.2;
            if (element.getAttribute('role') === 'button') confidence += 0.2;
            return Math.min(confidence, 1.0);
          };

          const generateSelectors = (element) => {
            const selectors = [];
            if (element.id) {
              selectors.push({ selector: '#' + element.id, type: 'id', robustness: 0.9, fallback: false });
            }
            for (const attr of element.attributes || []) {
              if (attr.name && attr.name.startsWith('data-')) {
                selectors.push({ selector: '[' + attr.name + '="' + attr.value + '"]', type: 'data-attr', robustness: 0.8, fallback: false });
              }
            }
            const className = typeof element.className === 'string' ? element.className : '';
            if (className) {
              const classes = className.split(/\s+/).filter(Boolean);
              const semanticClasses = classes.filter((cls) => /^(btn|button|link|nav|menu|form|input|submit|login|search)/.test(cls));
              if (semanticClasses.length > 0) {
                selectors.push({ selector: '.' + semanticClasses[0], type: 'class', robustness: 0.6, fallback: false });
              }
            }
            let cssSelector = element.tagName.toLowerCase();
            if (element.type) cssSelector += '[type="' + element.type + '"]';
            if (element.name) cssSelector += '[name="' + element.name + '"]';
            selectors.push({ selector: cssSelector, type: 'css', robustness: 0.4, fallback: true });
            selectors.sort((a, b) => b.robustness - a.robustness);
            return selectors;
          };

          const categorizeElement = (element) => {
            const text = element.textContent?.toLowerCase() || '';
            const type = element.type?.toLowerCase() || '';
            const tagName = element.tagName.toLowerCase();
            if (type === 'password' || text.includes('password') || text.includes('login') || text.includes('sign in')) {
              return 'authentication';
            }
            if (type === 'search' || text.includes('search') || (typeof element.name === 'string' && element.name.includes('search'))) {
              return 'data-entry';
            }
            if (tagName === 'a' || text.includes('menu') || text.includes('nav')) {
              return 'navigation';
            }
            if (type === 'submit' || text.includes('submit') || text.includes('save') || text.includes('send')) {
              return 'actions';
            }
            if (tagName === 'input' || tagName === 'textarea' || tagName === 'select') {
              return 'data-entry';
            }
            return 'general';
          };

          const describePathSegment = (element) => {
            if (!element || element.nodeType !== Node.ELEMENT_NODE) {
              return '';
            }
            const tag = element.tagName.toLowerCase();
            if (element.id) {
              return tag + '#' + element.id;
            }
            const classList = typeof element.className === 'string'
              ? element.className.split(/\s+/).filter(Boolean).slice(0, 2)
              : [];
            let descriptor = tag;
            if (classList.length > 0) {
              descriptor += '.' + classList.join('.');
            }
            const parent = element.parentElement;
            if (parent) {
              const siblings = Array.from(parent.children).filter((sibling) => sibling.tagName === element.tagName);
              if (siblings.length > 1) {
                const index = siblings.indexOf(element);
                if (index >= 0) {
                  descriptor += ':nth-of-type(' + (index + 1) + ')';
                }
              }
            }
            return descriptor;
          };

          const buildDomPath = (element) => {
            const segments = [];
            let current = element;
            let safety = 0;
            while (current && current.nodeType === Node.ELEMENT_NODE && safety < 25) {
              segments.push(describePathSegment(current));
              current = current.parentElement;
              safety += 1;
            }
            return segments;
          };

          const buildFallbackSelector = (element) => {
            if (!element || element.nodeType !== Node.ELEMENT_NODE) {
              return '';
            }
            if (element.id) {
              return '#' + element.id;
            }
            const segments = [];
            let current = element;
            let guard = 0;
            while (current && current.nodeType === Node.ELEMENT_NODE && guard < 25) {
              if (current.id) {
                segments.unshift('#' + current.id);
                break;
              }
              const tag = current.tagName.toLowerCase();
              const parent = current.parentElement;
              if (!parent) {
                segments.unshift(tag);
                break;
              }
              const siblings = Array.from(parent.children).filter((sibling) => sibling.tagName === current.tagName);
              if (siblings.length > 1) {
                const index = siblings.indexOf(current);
                segments.unshift(tag + ':nth-of-type(' + (index + 1) + ')');
              } else {
                segments.unshift(tag);
              }
              current = parent;
              guard += 1;
            }
            return segments.join(' > ');
          };

          const buildElementInfo = (element) => {
            if (!element || element.nodeType !== Node.ELEMENT_NODE) {
              return null;
            }
            const rect = element.getBoundingClientRect();
            if (!rect || rect.width <= 0 || rect.height <= 0) {
              return null;
            }
            const textContent = element.textContent?.trim() || element.value || element.placeholder || '';
            return {
              text: textContent.substring(0, 100),
              tagName: element.tagName,
              type: element.type || element.tagName.toLowerCase(),
              selectors: generateSelectors(element),
              boundingBox: {
                x: rect.x,
                y: rect.y,
                width: rect.width,
                height: rect.height,
              },
              confidence: calculateConfidence(element),
              category: categorizeElement(element),
              attributes: {
                id: element.id || '',
                className: typeof element.className === 'string' ? element.className : '',
                name: element.name || '',
                placeholder: element.placeholder || '',
                'aria-label': element.getAttribute('aria-label') || '',
                title: element.title || '',
              },
            };
          };

          const rawCandidates = ordered.map((element) => {
            const info = buildElementInfo(element);
            if (!info) {
              return null;
            }
            const selectors = Array.isArray(info.selectors) ? info.selectors : [];
            const fallbackSelector = buildFallbackSelector(element);
            const selector = selectors.length > 0 && typeof selectors[0].selector === 'string'
              ? selectors[0].selector
              : fallbackSelector;
            const path = buildDomPath(element);
            const pathSummary = path.length > 0 ? path.slice().reverse().join(' > ') : fallbackSelector;
            return {
              element: info,
              selector: selector || fallbackSelector || '',
              depth: path.length,
              path,
              pathSummary,
            };
          }).filter(Boolean);

          if (!rawCandidates.length && ordered.length) {
            const fallbackElement = ordered[0];
            const info = buildElementInfo(fallbackElement);
            if (info) {
              const selectors = Array.isArray(info.selectors) ? info.selectors : [];
              const fallbackSelector = buildFallbackSelector(fallbackElement);
              const selector = selectors.length > 0 && typeof selectors[0].selector === 'string'
                ? selectors[0].selector
                : fallbackSelector;
              const path = buildDomPath(fallbackElement);
              rawCandidates.push({
                element: info,
                selector: selector || fallbackSelector || '',
                depth: path.length,
                path,
                pathSummary: path.length > 0 ? path.slice().reverse().join(' > ') : selector,
              });
            }
          }

          const scoreCandidate = (candidate) => {
            if (!candidate || !candidate.element) {
              return -1000;
            }
            const element = candidate.element;
            const tag = (element.tagName || '').toLowerCase();
            const isRoot = tag === 'html' || tag === 'body';
            const interactiveTags = ['button', 'a', 'input', 'select', 'textarea', 'label', 'option'];
            const interactiveScore = interactiveTags.includes(tag) ? 30 : 0;
            const hasId = !!(element.attributes && typeof element.attributes.id === 'string' && element.attributes.id.trim().length > 0);
            const hasReliableSelector = Array.isArray(element.selectors)
              ? element.selectors.some((entry) => entry && entry.fallback === false && typeof entry.selector === 'string' && entry.selector.trim().length > 0)
              : false;
            const textScore = element.text && element.text.trim().length > 0 ? 2 : 0;
            const box = element.boundingBox;
            const area = box && box.width > 0 && box.height > 0 ? box.width * box.height : 0;
            let score = 0;
            score += Number.isFinite(candidate.depth) ? candidate.depth * 10 : 0;
            score += interactiveScore;
            if (hasId) {
              score += 8;
            }
            if (hasReliableSelector) {
              score += 5;
            }
            score += textScore;
            if (area > 0) {
              score -= Math.min(Math.log(area + 1), 8);
            }
            if (isRoot) {
              score -= 50;
            }
            return score;
          };

          const candidates = rawCandidates.length > 1
            ? rawCandidates.slice().sort((a, b) => scoreCandidate(b) - scoreCandidate(a))
            : rawCandidates.slice();

          let selectedIndex = candidates.findIndex((candidate) => {
            if (!candidate || !candidate.element) {
              return false;
            }
            const tag = (candidate.element.tagName || '').toLowerCase();
            if (!tag || tag === 'html' || tag === 'body') {
              return false;
            }
            const box = candidate.element.boundingBox;
            return box && box.width > 0 && box.height > 0;
          });

          if (selectedIndex === -1) {
            selectedIndex = candidates.findIndex((candidate) => {
              if (!candidate || !candidate.element) {
                return false;
              }
              const tag = (candidate.element.tagName || '').toLowerCase();
              return tag && tag !== 'html' && tag !== 'body';
            });
          }

          if (selectedIndex === -1 && candidates.length > 0) {
            selectedIndex = 0;
          }

          const chosen = selectedIndex >= 0 ? candidates[selectedIndex] : null;
          const clickPosition = chosen && chosen.element && chosen.element.boundingBox
            ? {
                x: chosen.element.boundingBox.x + (chosen.element.boundingBox.width / 2),
                y: chosen.element.boundingBox.y + (chosen.element.boundingBox.height / 2),
              }
            : null;

          return {
            element: chosen ? chosen.element : null,
            candidates,
            selectedIndex: selectedIndex >= 0 ? selectedIndex : 0,
            clickPosition,
          };
        }, {
          rawX,
          rawY,
          radius: probeRadius,
          samples: probeSamples,
        });

        if (!probeResult || (probeResult.error && probeResult.error.length)) {
          const message = probeResult && probeResult.error ? probeResult.error : 'Element probe failed';
          throw new Error(message);
        }

        extras = { probeResult };
        if (probeResult && probeResult.clickPosition) {
          extras.clickPosition = probeResult.clickPosition;
        }
        break;
      }
      case 'extract': {
        if (!params.selector) {
          throw new Error('extract step missing selector');
        }
        const extraction = await page.$$eval(
          params.selector,
          (elements, opts) => {
            const extractValue = (element) => {
              if (!element) {
                return null;
              }
              switch (opts.extractType) {
                case 'html':
                  return element.innerHTML;
                case 'value':
                  return element.value ?? element.getAttribute('value');
                case 'attribute':
                  return opts.attribute ? element.getAttribute(opts.attribute) : null;
                case 'text':
                default:
                  return element.textContent;
              }
            };

            if (!elements || elements.length === 0) {
              return opts.allMatches ? [] : null;
            }

            if (opts.allMatches) {
              return Array.from(elements).map((element) => extractValue(element));
            }

            return extractValue(elements[0]);
          },
          {
            extractType: params.extractType || 'text',
            attribute: params.attribute || null,
            allMatches: Boolean(params.allMatches),
          },
        );
        extras = { extractedData: extraction };
        const elementHandle = await page.$(params.selector);
        if (elementHandle) {
          const boundingBox = await elementHandle.boundingBox();
          if (boundingBox) {
            extras.elementBoundingBox = {
              x: boundingBox.x,
              y: boundingBox.y,
              width: boundingBox.width,
              height: boundingBox.height,
            };
          }
          if (typeof elementHandle.dispose === 'function') {
            await elementHandle.dispose();
          }
        }
        break;
      }
      case 'screenshot': {
        if (params.viewportWidth && params.viewportHeight) {
          await page.setViewport({ width: params.viewportWidth, height: params.viewportHeight });
        }
        if (params.waitForMs) {
          await waitForTime(params.waitForMs);
        }
        const fullPage = (typeof params.fullPage === 'boolean') ? params.fullPage : true;

        const highlightSelectors = Array.isArray(params.highlightSelectors) && params.highlightSelectors.length
          ? params.highlightSelectors
          : (params.focusSelector ? [params.focusSelector] : []);
        const maskSelectors = Array.isArray(params.maskSelectors) ? params.maskSelectors : [];
        const highlightPadding = Number.isFinite(params.highlightPadding) ? params.highlightPadding : 8;
        const highlightColor = (typeof params.highlightColor === 'string' && params.highlightColor.trim().length)
          ? params.highlightColor
          : '#00E5FF';
        const highlightBorderRadius = Number.isFinite(params.highlightBorderRadius) ? params.highlightBorderRadius : 12;
        const maskOpacityValue = Number.isFinite(params.maskOpacity)
          ? Math.min(Math.max(params.maskOpacity, 0), 1)
          : 0.45;

        let focusMetadata = null;
        const highlightMetadata = { highlightRegions: [], maskRegions: [] };
        const captureDomSnapshot = params.captureDomSnapshot === true || params.capture_dom_snapshot === true;
        let domSnapshotContent = null;

        const cleanupVisuals = async () => {
          try {
            await page.evaluate(() => {
              const root = document.querySelector('[data-bas-overlay-root="true"]');
              if (root && root.parentNode) {
                root.parentNode.removeChild(root);
              }
              document.querySelectorAll('[data-bas-overlay="true"]').forEach((node) => {
                if (node && node.parentNode) {
                  node.parentNode.removeChild(node);
                }
              });
              const body = document.body;
              if (body && typeof body.dataset.basOriginalBackground !== 'undefined') {
                const original = body.dataset.basOriginalBackground;
                if (original === '__bas-empty__') {
                  body.style.removeProperty('background');
                } else if (typeof original === 'string' && original.length) {
                  body.style.background = original;
                }
                delete body.dataset.basOriginalBackground;
              }
            });
          } catch (cleanupError) {
            // best-effort cleanup
          }
        };

        try {
          if (params.background) {
            await page.evaluate((background) => {
              const body = document.body;
              if (!body) {
                return;
              }
              if (typeof body.dataset.basOriginalBackground === 'undefined') {
                const computed = window.getComputedStyle(body);
                body.dataset.basOriginalBackground = computed && computed.background ? computed.background : '__bas-empty__';
              }
              body.style.background = background;
            }, params.background);
          }

          if (params.focusSelector) {
            const timeout = Number.isFinite(params.timeoutMs) ? params.timeoutMs : __DEFAULT_TIMEOUT__;
            const focusHandle = await page.waitForSelector(params.focusSelector, { timeout, state: 'visible' }).catch(() => null);
            if (focusHandle) {
              if (typeof focusHandle.scrollIntoViewIfNeeded === 'function') {
                await focusHandle.scrollIntoViewIfNeeded();
              } else {
                await focusHandle.evaluate((element) => element.scrollIntoView({ behavior: 'instant', block: 'center', inline: 'center' }));
              }
              const focusBox = await focusHandle.boundingBox();
              const focusPayload = { selector: params.focusSelector };
              if (focusBox) {
                focusPayload.boundingBox = {
                  x: focusBox.x,
                  y: focusBox.y,
                  width: focusBox.width,
                  height: focusBox.height,
                };
              }
              focusMetadata = focusPayload;
              if (typeof focusHandle.dispose === 'function') {
                await focusHandle.dispose();
              }
            }
          }

          if ((highlightSelectors && highlightSelectors.length) || (maskSelectors && maskSelectors.length)) {
            const overlayResult = await page.evaluate((config) => {
              const { highlightSelectors, padding, color, borderRadius, maskSelectors, maskOpacity } = config;
              const highlightRegions = [];
              const maskRegions = [];
              const root = document.createElement('div');
              root.setAttribute('data-bas-overlay-root', 'true');
              root.style.position = 'fixed';
              root.style.left = '0';
              root.style.top = '0';
              root.style.width = '100vw';
              root.style.height = '100vh';
              root.style.pointerEvents = 'none';
              root.style.zIndex = '2147483646';
              document.body.appendChild(root);

              const appendOverlay = (overlay) => {
                overlay.setAttribute('data-bas-overlay', 'true');
                root.appendChild(overlay);
              };

              if (Array.isArray(highlightSelectors)) {
                highlightSelectors.forEach((selector) => {
                  const elements = Array.from(document.querySelectorAll(selector));
                  elements.forEach((element) => {
                    const rect = element.getBoundingClientRect();
                    if (!rect || !rect.width || !rect.height) {
                      return;
                    }
                    const overlay = document.createElement('div');
                    overlay.style.position = 'absolute';
                    overlay.style.left = (rect.x - padding) + 'px';
                    overlay.style.top = (rect.y - padding) + 'px';
                    overlay.style.width = (rect.width + (padding * 2)) + 'px';
                    overlay.style.height = (rect.height + (padding * 2)) + 'px';
                    overlay.style.border = '3px solid ' + color;
                    overlay.style.borderRadius = String(borderRadius) + 'px';
                    overlay.style.boxShadow = '0 14px 40px rgba(0, 0, 0, 0.35)';
                    overlay.style.background = 'rgba(255, 255, 255, 0.06)';
                    overlay.style.pointerEvents = 'none';
                    overlay.style.backdropFilter = 'blur(1px)';
                    appendOverlay(overlay);
                    highlightRegions.push({
                      selector,
                      boundingBox: {
                        x: rect.x,
                        y: rect.y,
                        width: rect.width,
                        height: rect.height,
                      },
                      padding,
                      color,
                    });
                  });
                });
              }

              if (Array.isArray(maskSelectors)) {
                maskSelectors.forEach((selector) => {
                  const elements = Array.from(document.querySelectorAll(selector));
                  elements.forEach((element) => {
                    const rect = element.getBoundingClientRect();
                    if (!rect || !rect.width || !rect.height) {
                      return;
                    }
                    const overlay = document.createElement('div');
                    overlay.style.position = 'absolute';
                    overlay.style.left = rect.x + 'px';
                    overlay.style.top = rect.y + 'px';
                    overlay.style.width = rect.width + 'px';
                    overlay.style.height = rect.height + 'px';
                    overlay.style.background = 'rgba(0, 0, 0, ' + maskOpacity + ')';
                    overlay.style.borderRadius = String(borderRadius) + 'px';
                    overlay.style.pointerEvents = 'none';
                    appendOverlay(overlay);
                    maskRegions.push({
                      selector,
                      boundingBox: {
                        x: rect.x,
                        y: rect.y,
                        width: rect.width,
                        height: rect.height,
                      },
                      opacity: maskOpacity,
                    });
                  });
                });
              }

              return { highlightRegions, maskRegions };
            }, {
              highlightSelectors,
              padding: highlightPadding,
              color: highlightColor,
              borderRadius: highlightBorderRadius,
              maskSelectors,
              maskOpacity: maskOpacityValue,
            });

            if (overlayResult && Array.isArray(overlayResult.highlightRegions)) {
              highlightMetadata.highlightRegions = overlayResult.highlightRegions;
            }
            if (overlayResult && Array.isArray(overlayResult.maskRegions)) {
              highlightMetadata.maskRegions = overlayResult.maskRegions;
            }
          }

          const capture = await page.screenshot({ encoding: 'base64', fullPage });
          extras = { screenshotBase64: capture };
          if (focusMetadata) {
            extras.focusedElement = focusMetadata;
          }
          if (highlightMetadata.highlightRegions.length) {
            extras.highlightRegions = highlightMetadata.highlightRegions;
          }
          if (highlightMetadata.maskRegions.length) {
            extras.maskRegions = highlightMetadata.maskRegions;
          }
          if (Number.isFinite(params.zoomFactor) && params.zoomFactor > 0) {
            extras.zoomFactor = params.zoomFactor;
          }
        } finally {
          try {
            await cleanupVisuals();
          } catch (cleanupError) {
            // best-effort cleanup
          }
          if (captureDomSnapshot && domSnapshotContent == null) {
            try {
              domSnapshotContent = await page.content();
            } catch (snapshotError) {
              domSnapshotContent = null;
            }
          }
        }

        if (captureDomSnapshot && typeof domSnapshotContent === 'string' && domSnapshotContent.length > 0) {
          extras.domSnapshot = domSnapshotContent;
        }

        await storeReplaySnapshot();
        break;
      }
      case 'assert': {
        const rawMode = typeof params.assertMode === 'string' ? params.assertMode : params.mode;
        const mode = (rawMode || 'exists').toLowerCase();
        const selector = typeof params.selector === 'string' ? params.selector.trim() : '';
        const timeout = Number.isFinite(params.timeoutMs) ? Math.max(params.timeoutMs, 0) : __DEFAULT_TIMEOUT__;
        const failureMessage = typeof params.failureMessage === 'string' && params.failureMessage.trim().length
          ? params.failureMessage.trim()
          : 'Assertion failed (' + mode + ')';
        const negate = params.negate === true;
        const caseSensitive = params.caseSensitive === true;
        const continueOnFailure = params.continueOnFailure === true;
        const expectedValue = Object.prototype.hasOwnProperty.call(params, 'expectedValue') ? params.expectedValue : null;

        const assertion = {
          mode,
          selector,
          expected: expectedValue,
          negated: negate,
          caseSensitive,
          success: false,
        };

        const requireSelector = () => {
          if (!selector) {
            throw new Error('assert step missing selector');
          }
        };

        const deadline = Date.now() + timeout;
        const pollDelay = Math.max(100, Math.min(500, Math.floor(timeout / 10) || 150));
        let elementBoundingBox = null;

        const recordBoundingBox = async (handle) => {
          if (!handle) {
            return;
          }
          try {
            const box = await handle.boundingBox();
            if (box) {
              elementBoundingBox = {
                x: box.x,
                y: box.y,
                width: box.width,
                height: box.height,
              };
            }
          } catch (err) {
            // ignore bounding box failures
          }
        };

        const normalizeString = (value) => {
          if (value == null) {
            return '';
          }
          const stringValue = String(value);
          return caseSensitive ? stringValue : stringValue.toLowerCase();
        };

        let passed = false;
        let actualValue = null;
        let failureDetail = '';

        if (mode === 'exists_or_not') {
          // exists_or_not mode always passes - used for optional elements that may or may not appear
          requireSelector();
          let handle = null;
          try {
            // Try to find the element with a short timeout (just poll once)
            handle = await page.$(selector);
            if (handle) {
              await recordBoundingBox(handle);
            }
          } finally {
            if (handle && typeof handle.dispose === 'function') {
              await handle.dispose();
            }
          }
          actualValue = Boolean(handle);
          passed = true; // Always pass regardless of element existence
        } else if (mode === 'exists' || mode === 'not_exists') {
          requireSelector();
          let lastExists = false;
          let satisfied = false;
          let debugInfo = '';

          while (Date.now() <= deadline) {
            let handle = null;
            try {
              // Try page.waitForSelector first (same as wait node uses)
              const remainingTime = Math.max(100, deadline - Date.now());
              try {
                handle = await page.waitForSelector(selector, {
                  timeout: Math.min(remainingTime, pollDelay),
                  visible: false // Don't require visibility, just existence in DOM
                });
              } catch (waitErr) {
                // waitForSelector timed out, check if element exists via evaluate
                const pageUrl = await page.url();
                const elemInfo = await page.evaluate((sel) => {
                  const el = document.querySelector(sel);
                  if (!el) {
                    return {
                      exists: false,
                      url: window.location.href,
                      readyState: document.readyState,
                      allTestIds: Array.from(document.querySelectorAll('[data-testid]')).map(e => e.getAttribute('data-testid')).slice(0, 10)
                    };
                  }
                  return {
                    exists: true,
                    url: window.location.href,
                    tagName: el.tagName,
                    visible: el.offsetParent !== null,
                    display: window.getComputedStyle(el).display
                  };
                }, selector);

                debugInfo = JSON.stringify({...elemInfo, cdpUrl: pageUrl});

                if (elemInfo.exists) {
                  // Element exists in DOM but waitForSelector failed - try page.$()
                  handle = await page.$(selector);
                }
              }

              lastExists = Boolean(handle);
              if (handle) {
                await recordBoundingBox(handle);
              }
            } finally {
              if (handle && typeof handle.dispose === 'function') {
                await handle.dispose();
              }
            }

            if ((mode === 'exists' && lastExists) || (mode === 'not_exists' && !lastExists)) {
              satisfied = true;
              break;
            }
            await waitForTime(pollDelay);
          }
          actualValue = mode === 'exists' ? Boolean(elementBoundingBox) : lastExists;
          passed = satisfied;
          if (!passed) {
            const baseMsg = mode === 'exists'
              ? 'Expected selector "' + selector + '" to exist'
              : 'Expected selector "' + selector + '" to be absent';
            failureDetail = debugInfo ? baseMsg + ' [DEBUG: ' + debugInfo + ']' : baseMsg;
          }
        } else if (mode === 'text_equals' || mode === 'text_contains') {
          requireSelector();
          const expectedNormalized = normalizeString(expectedValue);
          let textValue = '';
          while (Date.now() <= deadline) {
            try {
              textValue = await page.$eval(selector, (element) => (element && element.textContent) || '');
              const normalized = normalizeString(textValue);
              if ((mode === 'text_equals' && normalized === expectedNormalized) ||
                  (mode === 'text_contains' && normalized.includes(expectedNormalized))) {
                const handle = await page.$(selector);
                await recordBoundingBox(handle);
                if (handle && typeof handle.dispose === 'function') {
                  await handle.dispose();
                }
                passed = true;
                break;
              }
            } catch (err) {
              failureDetail = String(err);
              break;
            }
            await waitForTime(pollDelay);
          }
          actualValue = textValue;
          if (!passed && !failureDetail) {
            const expectedStr = String(expectedValue != null ? expectedValue : '');
            failureDetail = mode === 'text_equals'
              ? 'Expected text ' + (caseSensitive ? '' : '(case-insensitive) ') + 'to equal "' + expectedStr + '"'
              : 'Expected text ' + (caseSensitive ? '' : '(case-insensitive) ') + 'to contain "' + expectedStr + '"';
          }
        } else if (mode === 'attribute_equals' || mode === 'attribute_contains') {
          requireSelector();
          const attribute = typeof params.attribute === 'string' && params.attribute.trim().length
            ? params.attribute.trim()
            : 'value';
          const expectedNormalized = normalizeString(expectedValue);
          let attributeValue = '';
          while (Date.now() <= deadline) {
            try {
              attributeValue = await page.$eval(
                selector,
                (element, attr) => {
                  if (!element) {
                    return '';
                  }
                  const value = element.getAttribute(attr);
                  return value == null ? '' : value;
                },
                attribute,
              );
              const normalized = normalizeString(attributeValue);
              if ((mode === 'attribute_equals' && normalized === expectedNormalized) ||
                  (mode === 'attribute_contains' && normalized.includes(expectedNormalized))) {
                const handle = await page.$(selector);
                await recordBoundingBox(handle);
                if (handle && typeof handle.dispose === 'function') {
                  await handle.dispose();
                }
                passed = true;
                break;
              }
            } catch (err) {
              failureDetail = String(err);
              break;
            }
            await waitForTime(pollDelay);
          }
          actualValue = attributeValue;
          if (!passed && !failureDetail) {
            const expectedStr = String(expectedValue != null ? expectedValue : '');
            failureDetail = mode === 'attribute_equals'
              ? 'Expected attribute "' + attribute + '" ' + (caseSensitive ? '' : '(case-insensitive) ') + 'to equal "' + expectedStr + '"'
              : 'Expected attribute "' + attribute + '" ' + (caseSensitive ? '' : '(case-insensitive) ') + 'to contain "' + expectedStr + '"';
          }
        } else if (mode === 'expression') {
          const expression = typeof params.expression === 'string' ? params.expression.trim() : '';
          if (!expression) {
            throw new Error('assert expression missing');
          }
          const evaluation = await page.evaluate((expr) => {
            try {
              const fn = new Function('return (' + expr + ');');
              const value = fn();
              return { value, success: Boolean(value) };
            } catch (err) {
              return { error: String(err) };
            }
          }, expression);
          if (evaluation && Object.prototype.hasOwnProperty.call(evaluation, 'error')) {
            failureDetail = evaluation.error;
            passed = false;
          } else {
            actualValue = evaluation ? evaluation.value : null;
            passed = Boolean(evaluation && evaluation.success);
          }
        } else {
          failureDetail = 'Unsupported assertion mode ' + mode;
        }

        assertion.actual = actualValue;
        const effectiveSuccess = negate ? !passed : passed;
        assertion.success = effectiveSuccess;

        if (!assertion.success && !failureDetail) {
          failureDetail = failureMessage;
        }

        if (!assertion.success) {
          assertion.message = failureDetail || failureMessage;
        }

        extras = { assertion, __forcedSuccess: assertion.success };
        if (elementBoundingBox) {
          extras.elementBoundingBox = elementBoundingBox;
        }

        if (!assertion.success && !continueOnFailure) {
          throw new Error(assertion.message || failureMessage);
        }

        break;
      }
      default: {
        throw new Error('Unsupported step type: ' + step.type);
      }
    }

    if (hasSuccessCheck) {
      await verifySuccessState();
    }

    await ensureReplayScreenshot();
    const telemetryResult = flushTelemetry();
    const successPayload = {
      ...baseResult,
      success: true,
      durationMs: Date.now() - stepStart,
      ...extras,
    };
    if (Object.prototype.hasOwnProperty.call(successPayload, '__forcedSuccess')) {
      successPayload.success = Boolean(successPayload.__forcedSuccess);
      delete successPayload.__forcedSuccess;
    }
    if (telemetryResult.consoleLogs.length) {
      successPayload.consoleLogs = telemetryResult.consoleLogs;
    }
    if (telemetryResult.networkEvents.length) {
      successPayload.networkEvents = telemetryResult.networkEvents;
    }
    return { success: true, steps: [successPayload], totalDurationMs: Date.now() - runStart };
  } catch (error) {
    try {
      await ensureReplayScreenshot();
    } catch (_) {}
    const message = (error && error.message) ? error.message : String(error);
    const stack = (error && error.stack) ? error.stack : undefined;
    const telemetryResult = flushTelemetry();
    const failurePayload = {
      ...baseResult,
      success: false,
      error: message,
      stack,
      durationMs: Date.now() - stepStart,
      ...extras,
    };
    if (telemetryResult.consoleLogs.length) {
      failurePayload.consoleLogs = telemetryResult.consoleLogs;
    }
    if (telemetryResult.networkEvents.length) {
      failurePayload.networkEvents = telemetryResult.networkEvents;
    }
    return { success: false, error: message, steps: [failurePayload], totalDurationMs: Date.now() - runStart };
  }
};
`

// BuildStepScript exposes the step script generation for unit tests.
func BuildStepScript(instruction Instruction) (string, error) {
	return buildStepScript(instruction, defaultViewportWidth, defaultViewportHeight)
}
