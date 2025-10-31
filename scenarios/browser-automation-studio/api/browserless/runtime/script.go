package runtime

import (
	"encoding/json"
	"strconv"
	"strings"
)

func buildStepScript(instruction Instruction) (string, error) {
	payload, err := json.Marshal(instruction)
	if err != nil {
		return "", err
	}

	script := strings.Replace(stepScriptTemplate, "__INSTRUCTION__", string(payload), 1)
	script = strings.ReplaceAll(script, "__VIEWPORT_WIDTH__", strconv.Itoa(defaultViewportWidth))
	script = strings.ReplaceAll(script, "__VIEWPORT_HEIGHT__", strconv.Itoa(defaultViewportHeight))
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

  const rehydrateFromPreload = () => {
    if (!preloadHTML) {
      return;
    }
    try {
      const parser = typeof DOMParser === 'function' ? new DOMParser() : null;
      if (!parser) {
        if (document && document.body && (!document.body.children || document.body.children.length === 0)) {
          document.body.innerHTML = preloadHTML;
        }
        return;
      }
      const parsed = parser.parseFromString(preloadHTML, 'text/html');
      if (!parsed || !parsed.documentElement) {
        if (document && document.body && (!document.body.children || document.body.children.length === 0)) {
          document.body.innerHTML = preloadHTML;
        }
        return;
      }
      if (!document || !document.documentElement) {
        return;
      }
      const newDoc = parsed.documentElement;
      const current = document.documentElement;
      if (current.innerHTML && current.innerHTML.trim().length > 0) {
        return;
      }
      const attrs = current.getAttributeNames();
      for (const name of attrs) {
        if (!newDoc.hasAttribute(name)) {
          current.removeAttribute(name);
        }
      }
      for (const attr of Array.from(newDoc.attributes || [])) {
        current.setAttribute(attr.name, attr.value);
      }
      current.innerHTML = newDoc.innerHTML;
    } catch {}
  };

  rehydrateFromPreload();

  try {
    const currentUrl = typeof page.url === 'function' ? page.url() : 'unknown';
    console.log('[BAS][debug] step start url:', currentUrl);
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

  const ensureReplayScreenshot = async () => {
    if (!shouldCaptureReplay()) {
      return null;
    }
    if (extras && typeof extras === 'object' && Object.prototype.hasOwnProperty.call(extras, 'screenshotBase64') && extras.screenshotBase64) {
      return extras.screenshotBase64;
    }
    try {
      const capture = await page.screenshot({ encoding: 'base64', fullPage: true });
      if (capture && capture.length) {
        extras = { ...extras, screenshotBase64: capture };
        return capture;
      }
    } catch (captureError) {
      try {
        console.warn('[BAS][replay] failed to capture replay screenshot:', captureError);
      } catch (_) {}
    }
    return null;
  };

  try {
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

        if (mode === 'exists' || mode === 'not_exists') {
          requireSelector();
          let lastExists = false;
          let satisfied = false;
          while (Date.now() <= deadline) {
            let handle = null;
            try {
              handle = await page.$(selector);
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
            failureDetail = mode === 'exists'
              ? 'Expected selector "' + selector + '" to exist'
              : 'Expected selector "' + selector + '" to be absent';
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
	return buildStepScript(instruction)
}
