/**
 * Recording Script - Browser-Side Event Capture
 *
 * This script is injected into browser pages to capture user actions during recording.
 * It runs in the page context and communicates with the Playwright driver via exposed functions.
 *
 * IMPORTANT: This file runs in the BROWSER context, not Node.js.
 * - No imports/requires - must be self-contained
 * - No TypeScript - plain JavaScript only
 * - The SHARED_CONFIG placeholder is replaced at injection time with actual configuration
 *
 * @see ../injector.ts - The Node.js module that injects this script
 * @see ../selector-config.ts - Configuration source of truth
 */

(function () {
  'use strict';

  // Prevent double-injection
  if (window.__recordingActive) {
    console.log('[Recording] Already active, skipping injection');
    return;
  }
  window.__recordingActive = true;
  console.log('[Recording] Injected recording script');

  // ============================================================================
  // Configuration (injected from selector-config.ts at runtime)
  // The placeholder below is replaced by injector.ts with actual config values
  // ============================================================================

  var SHARED_CONFIG = __INJECTED_CONFIG__;

  // Flatten config for easier access
  var CONFIG = {
    // Debounce timings
    INPUT_DEBOUNCE_MS: SHARED_CONFIG.RECORDING_DEBOUNCE.input,
    SCROLL_DEBOUNCE_MS: SHARED_CONFIG.RECORDING_DEBOUNCE.scroll,
    RESIZE_DEBOUNCE_MS: SHARED_CONFIG.RECORDING_DEBOUNCE.resize,

    // Limits
    MAX_TEXT_LENGTH: SHARED_CONFIG.SELECTOR_DEFAULTS.maxTextLength,
    MAX_SELECTOR_DEPTH: SHARED_CONFIG.SELECTOR_DEFAULTS.maxCssDepth,

    // Patterns from shared config
    UNSTABLE_CLASS_PATTERNS: SHARED_CONFIG.UNSTABLE_CLASS_PATTERNS,
    DYNAMIC_ID_PATTERNS: SHARED_CONFIG.DYNAMIC_ID_PATTERNS,
    SEMANTIC_CLASS_PATTERNS: SHARED_CONFIG.SEMANTIC_CLASS_PATTERNS,
    TEST_ID_ATTRIBUTES: SHARED_CONFIG.TEST_ID_ATTRIBUTES,
    TEXT_CONTENT_TAGS: SHARED_CONFIG.TEXT_CONTENT_TAGS,

    // Confidence scores
    CONFIDENCE: SHARED_CONFIG.CONFIDENCE_SCORES,
    SPECIFICITY: SHARED_CONFIG.SPECIFICITY_SCORES,
  };

  // ============================================================================
  // State
  // ============================================================================

  var inputBuffer = '';
  var inputTarget = null;
  var inputTimeout = null;
  var scrollTimeout = null;
  var lastScrollPos = { x: 0, y: 0 };

  // ============================================================================
  // Selector Generation
  // ============================================================================

  /**
   * Generate multiple selector candidates for an element.
   * @param {Element} element - The DOM element to generate selectors for
   * @returns {{ primary: string, candidates: Array<{type: string, value: string, confidence: number, specificity: number}> }}
   */
  function generateSelectors(element) {
    var candidates = [];

    // Strategy 1: data-testid
    var testIdSelector = generateTestIdSelector(element);
    if (testIdSelector) candidates.push(testIdSelector);

    // Strategy 2: Unique ID
    var idSelector = generateIdSelector(element);
    if (idSelector) candidates.push(idSelector);

    // Strategy 3: ARIA attributes
    var ariaSelector = generateAriaSelector(element);
    if (ariaSelector) candidates.push(ariaSelector);

    // Strategy 4: Tag + text
    var textSelector = generateTextSelector(element);
    if (textSelector) candidates.push(textSelector);

    // Strategy 5: Data attributes
    var dataAttrSelector = generateDataAttrSelector(element);
    if (dataAttrSelector) candidates.push(dataAttrSelector);

    // Strategy 6: CSS path
    var cssSelector = generateCssPath(element);
    if (cssSelector) candidates.push(cssSelector);

    // Strategy 7: XPath
    var xpathSelector = generateXPathSelector(element);
    if (xpathSelector) candidates.push(xpathSelector);

    // Sort by confidence
    candidates.sort(function (a, b) {
      return b.confidence - a.confidence;
    });

    return {
      primary: candidates[0] ? candidates[0].value : generateFallbackSelector(element),
      candidates: candidates,
    };
  }

  /**
   * Generate a selector using data-testid attribute.
   * @param {Element} element
   * @returns {{ type: string, value: string, confidence: number, specificity: number } | null}
   */
  function generateTestIdSelector(element) {
    for (var i = 0; i < CONFIG.TEST_ID_ATTRIBUTES.length; i++) {
      var attr = CONFIG.TEST_ID_ATTRIBUTES[i];
      var value = element.getAttribute(attr);
      if (value) {
        var selector = '[' + attr + '="' + escapeCssAttr(value) + '"]';
        if (isUniqueSelector(selector)) {
          return {
            type: 'data-testid',
            value: selector,
            confidence: CONFIG.CONFIDENCE.dataTestId,
            specificity: CONFIG.SPECIFICITY.dataTestId,
          };
        }
      }
    }
    return null;
  }

  /**
   * Generate a selector using element ID.
   * @param {Element} element
   * @returns {{ type: string, value: string, confidence: number, specificity: number } | null}
   */
  function generateIdSelector(element) {
    var id = element.id;
    if (!id) return null;

    var isDynamic = isDynamicId(id);
    var selector = '#' + escapeCssSelector(id);

    if (isUniqueSelector(selector)) {
      return {
        type: 'id',
        value: selector,
        confidence: isDynamic ? CONFIG.CONFIDENCE.idDynamic : CONFIG.CONFIDENCE.id,
        specificity: CONFIG.SPECIFICITY.id,
      };
    }
    return null;
  }

  /**
   * Generate a selector using ARIA attributes.
   * @param {Element} element
   * @returns {{ type: string, value: string, confidence: number, specificity: number } | null}
   */
  function generateAriaSelector(element) {
    var ariaLabel = element.getAttribute('aria-label');
    if (ariaLabel) {
      var selector = '[aria-label="' + escapeCssAttr(ariaLabel) + '"]';
      if (isUniqueSelector(selector)) {
        return {
          type: 'aria',
          value: selector,
          confidence: CONFIG.CONFIDENCE.ariaLabel,
          specificity: CONFIG.SPECIFICITY.ariaLabel,
        };
      }
    }
    return null;
  }

  /**
   * Generate a selector using visible text content.
   * @param {Element} element
   * @returns {{ type: string, value: string, confidence: number, specificity: number } | null}
   */
  function generateTextSelector(element) {
    var tag = getTextTag(element);
    if (!tag) return null;

    var text = getVisibleText(element);
    var minLen = SHARED_CONFIG.SELECTOR_DEFAULTS.minTextLength;
    var maxLen = SHARED_CONFIG.SELECTOR_DEFAULTS.maxTextLengthForSelector;
    var truncateLen = SHARED_CONFIG.SELECTOR_DEFAULTS.selectorTextMaxLength;

    if (!text || text.length < minLen || text.length > maxLen) return null;

    // Truncate text for selector stability
    var selectorText = text.length > truncateLen ? text.slice(0, truncateLen) : text;
    var selector = tag + ':has-text("' + selectorText.replace(/"/g, '\\"') + '")';
    return {
      type: 'text',
      value: selector,
      confidence: text.length > 15 ? CONFIG.CONFIDENCE.textLong : CONFIG.CONFIDENCE.textShort,
      specificity: CONFIG.SPECIFICITY.text,
    };
  }

  /**
   * Generate a selector using data attributes (excluding test IDs).
   * @param {Element} element
   * @returns {{ type: string, value: string, confidence: number, specificity: number } | null}
   */
  function generateDataAttrSelector(element) {
    var skipAttrs = ['testid', 'test-id', 'test', 'cy', 'qa'];

    for (var i = 0; i < element.attributes.length; i++) {
      var attr = element.attributes[i];
      if (!attr.name.startsWith('data-')) continue;
      var suffix = attr.name.slice(5);
      if (skipAttrs.indexOf(suffix) !== -1) continue;

      var selector = '[' + attr.name + '="' + escapeCssAttr(attr.value) + '"]';
      if (isUniqueSelector(selector)) {
        return {
          type: 'data-attr',
          value: selector,
          confidence: CONFIG.CONFIDENCE.dataAttr,
          specificity: CONFIG.SPECIFICITY.dataAttr,
        };
      }
    }
    return null;
  }

  /**
   * Generate a CSS path selector.
   * @param {Element} element
   * @returns {{ type: string, value: string, confidence: number, specificity: number } | null}
   */
  function generateCssPath(element) {
    var parts = [];
    var current = element;
    var depth = 0;

    while (current && current !== document.body && depth < CONFIG.MAX_SELECTOR_DEPTH) {
      var tag = current.tagName.toLowerCase();
      var stableClasses = getStableClasses(current);

      var part = tag;
      if (stableClasses.length > 0) {
        part = tag + '.' + stableClasses.slice(0, 2).join('.');
      }

      parts.unshift(part);

      var selector = parts.join(' > ');
      if (isUniqueSelector(selector)) {
        return {
          type: 'css',
          value: selector,
          confidence: assessCssStability(selector, stableClasses),
          specificity: CONFIG.SPECIFICITY.cssPath - depth * 5,
        };
      }

      current = current.parentElement;
      depth++;
    }

    // Try with nth-child
    return generateCssPathWithNthChild(element);
  }

  /**
   * Generate a CSS path selector using nth-of-type.
   * @param {Element} element
   * @returns {{ type: string, value: string, confidence: number, specificity: number } | null}
   */
  function generateCssPathWithNthChild(element) {
    var parts = [];
    var current = element;
    var depth = 0;

    while (current && current !== document.body && depth < CONFIG.MAX_SELECTOR_DEPTH) {
      var tag = current.tagName.toLowerCase();
      var parent = current.parentElement;

      var part = tag;
      if (parent) {
        var siblings = Array.from(parent.children).filter(function (s) {
          return s.tagName.toLowerCase() === tag;
        });
        if (siblings.length > 1) {
          var index = siblings.indexOf(current) + 1;
          part = tag + ':nth-of-type(' + index + ')';
        }
      }

      parts.unshift(part);

      var selector = parts.join(' > ');
      if (isUniqueSelector(selector)) {
        return {
          type: 'css',
          value: selector,
          confidence: CONFIG.CONFIDENCE.cssNthChild,
          specificity: CONFIG.SPECIFICITY.cssNthChild - depth * 5,
        };
      }

      current = parent;
      depth++;
    }
    return null;
  }

  /**
   * Generate an XPath selector.
   * @param {Element} element
   * @returns {{ type: string, value: string, confidence: number, specificity: number }}
   */
  function generateXPathSelector(element) {
    var tag = element.tagName.toLowerCase();
    var text = getVisibleText(element);
    var maxLen = SHARED_CONFIG.SELECTOR_DEFAULTS.maxTextLengthForSelector;

    if (text && text.length > 0 && text.length <= maxLen) {
      var escapedText = escapeXPathString(text);
      var xpath = '//' + tag + '[contains(text(), ' + escapedText + ')]';
      if (isUniqueXPath(xpath)) {
        return {
          type: 'xpath',
          value: xpath,
          confidence: CONFIG.CONFIDENCE.xpathText,
          specificity: CONFIG.SPECIFICITY.xpathText,
        };
      }
    }

    // Positional fallback
    return {
      type: 'xpath',
      value: generatePositionalXPath(element),
      confidence: CONFIG.CONFIDENCE.xpathPositional,
      specificity: CONFIG.SPECIFICITY.xpathPositional,
    };
  }

  /**
   * Escape string for use in XPath expressions.
   * Handles strings containing single quotes, double quotes, or both.
   * @param {string} str
   * @returns {string}
   */
  function escapeXPathString(str) {
    var sq = "'";
    var dq = '"';
    // If string contains both quotes, use concat()
    if (str.indexOf(sq) !== -1 && str.indexOf(dq) !== -1) {
      // Split on single quotes and rejoin with concat
      var parts = str.split(sq);
      var result = 'concat(';
      for (var i = 0; i < parts.length; i++) {
        if (i > 0) result += ', ' + dq + sq + dq + ', ';
        result += sq + parts[i] + sq;
      }
      result += ')';
      return result;
    }
    // If string contains single quotes, use double quotes
    if (str.indexOf(sq) !== -1) {
      return dq + str + dq;
    }
    // Default to single quotes
    return sq + str + sq;
  }

  /**
   * Generate a positional XPath selector.
   * @param {Element} element
   * @returns {string}
   */
  function generatePositionalXPath(element) {
    var parts = [];
    var current = element;

    while (current && current !== document.documentElement) {
      var tag = current.tagName.toLowerCase();
      var parent = current.parentElement;

      if (parent) {
        var siblings = Array.from(parent.children).filter(function (s) {
          return s.tagName.toLowerCase() === tag;
        });
        if (siblings.length > 1) {
          var index = siblings.indexOf(current) + 1;
          parts.unshift(tag + '[' + index + ']');
        } else {
          parts.unshift(tag);
        }
      } else {
        parts.unshift(tag);
      }

      current = parent;
    }

    return '/' + parts.join('/');
  }

  /**
   * Generate a fallback selector when all strategies fail.
   * @param {Element} element
   * @returns {string}
   */
  function generateFallbackSelector(element) {
    var tag = element.tagName.toLowerCase();

    for (var i = 0; i < element.attributes.length; i++) {
      var attr = element.attributes[i];
      if (attr.name === 'class' || attr.name === 'style') continue;
      var selector = tag + '[' + attr.name + '="' + escapeCssAttr(attr.value) + '"]';
      if (isUniqueSelector(selector)) return selector;
    }

    return generatePositionalXPath(element);
  }

  // ============================================================================
  // Helper Functions
  // ============================================================================

  /**
   * Check if a CSS selector matches exactly one element.
   * @param {string} selector
   * @returns {boolean}
   */
  function isUniqueSelector(selector) {
    try {
      return document.querySelectorAll(selector).length === 1;
    } catch (e) {
      return false;
    }
  }

  /**
   * Check if an XPath expression matches exactly one element.
   * @param {string} xpath
   * @returns {boolean}
   */
  function isUniqueXPath(xpath) {
    try {
      var result = document.evaluate(
        xpath,
        document,
        null,
        XPathResult.ORDERED_NODE_SNAPSHOT_TYPE,
        null
      );
      return result.snapshotLength === 1;
    } catch (e) {
      return false;
    }
  }

  /**
   * Check if an ID appears to be dynamically generated.
   * @param {string} id
   * @returns {boolean}
   */
  function isDynamicId(id) {
    for (var i = 0; i < CONFIG.DYNAMIC_ID_PATTERNS.length; i++) {
      if (CONFIG.DYNAMIC_ID_PATTERNS[i].test(id)) {
        return true;
      }
    }
    return false;
  }

  /**
   * Get stable (non-generated) CSS classes from an element.
   * @param {Element} element
   * @returns {string[]}
   */
  function getStableClasses(element) {
    return Array.from(element.classList).filter(function (cls) {
      if (!cls.trim()) return false;
      for (var i = 0; i < CONFIG.UNSTABLE_CLASS_PATTERNS.length; i++) {
        if (CONFIG.UNSTABLE_CLASS_PATTERNS[i].test(cls)) {
          return false;
        }
      }
      return true;
    });
  }

  /**
   * Assess the stability of a CSS selector based on its structure.
   * @param {string} selector
   * @param {string[]} stableClasses
   * @returns {number}
   */
  function assessCssStability(selector, stableClasses) {
    var score = CONFIG.CONFIDENCE.cssPath;

    for (var i = 0; i < stableClasses.length; i++) {
      var cls = stableClasses[i];
      for (var j = 0; j < CONFIG.SEMANTIC_CLASS_PATTERNS.length; j++) {
        if (CONFIG.SEMANTIC_CLASS_PATTERNS[j].test(cls)) {
          score += 0.1;
          break;
        }
      }
    }

    if (selector.indexOf('nth-child') !== -1 || selector.indexOf('nth-of-type') !== -1) {
      score -= 0.15;
    }

    var matches = selector.match(/>/g);
    var depth = matches ? matches.length : 0;
    if (depth > 3) score -= 0.05 * (depth - 3);

    return Math.max(0.3, Math.min(0.9, score));
  }

  /**
   * Get the tag name if the element commonly has meaningful text.
   * @param {Element} element
   * @returns {string | null}
   */
  function getTextTag(element) {
    var tag = element.tagName.toLowerCase();
    return CONFIG.TEXT_CONTENT_TAGS.indexOf(tag) !== -1 ? tag : null;
  }

  /**
   * Infer ARIA role for metadata (not for selectors).
   * @param {Element} element
   * @returns {string | null}
   */
  function inferRole(element) {
    var tag = element.tagName.toLowerCase();
    var roles = {
      button: 'button',
      a: 'link',
      input:
        element.type === 'checkbox'
          ? 'checkbox'
          : element.type === 'radio'
            ? 'radio'
            : element.type === 'submit'
              ? 'button'
              : 'textbox',
      select: 'combobox',
      textarea: 'textbox',
      img: 'img',
      nav: 'navigation',
      form: 'form',
    };
    return roles[tag] || null;
  }

  /**
   * Get visible text content from an element.
   * @param {Element} element
   * @returns {string}
   */
  function getVisibleText(element) {
    if (element.tagName === 'INPUT' || element.tagName === 'TEXTAREA') {
      return (element.value || element.placeholder || '').slice(0, CONFIG.MAX_TEXT_LENGTH);
    }
    return (element.textContent || '').trim().slice(0, CONFIG.MAX_TEXT_LENGTH);
  }

  /**
   * Escape special characters for CSS selector ID.
   * @param {string} str
   * @returns {string}
   */
  function escapeCssSelector(str) {
    return str.replace(/([!"#$%&'()*+,./:;<=>?@[\\\]^`{|}~])/g, '\\$1');
  }

  /**
   * Escape special characters for CSS attribute value.
   * @param {string} str
   * @returns {string}
   */
  function escapeCssAttr(str) {
    return str.replace(/"/g, '\\"').replace(/\n/g, '\\n');
  }

  // ============================================================================
  // Element Metadata Extraction
  // ============================================================================

  /**
   * Extract metadata about an element.
   * @param {Element} element
   * @returns {object}
   */
  function extractElementMeta(element) {
    var style = window.getComputedStyle(element);
    var tag = element.tagName.toLowerCase();

    return {
      tagName: tag,
      id: element.id || undefined,
      className: element.className || undefined,
      innerText: getVisibleText(element) || undefined,
      attributes: getRelevantAttributes(element),
      isVisible:
        style.display !== 'none' &&
        style.visibility !== 'hidden' &&
        parseFloat(style.opacity) > 0,
      isEnabled: !element.disabled,
      role: element.getAttribute('role') || inferRole(element) || undefined,
      ariaLabel: element.getAttribute('aria-label') || undefined,
    };
  }

  /**
   * Get relevant attributes from an element.
   * @param {Element} element
   * @returns {object}
   */
  function getRelevantAttributes(element) {
    var attrs = {};
    var interesting = [
      'type',
      'name',
      'placeholder',
      'title',
      'alt',
      'href',
      'value',
      'role',
    ];

    for (var i = 0; i < interesting.length; i++) {
      var attr = interesting[i];
      var val = element.getAttribute(attr);
      if (val) attrs[attr] = val.slice(0, 100);
    }

    for (var j = 0; j < element.attributes.length; j++) {
      var a = element.attributes[j];
      if (a.name.startsWith('data-')) {
        attrs[a.name] = a.value.slice(0, 100);
      }
    }

    return attrs;
  }

  /**
   * Get bounding box of an element.
   * @param {Element} element
   * @returns {{ x: number, y: number, width: number, height: number }}
   */
  function getBoundingBox(element) {
    var rect = element.getBoundingClientRect();
    return {
      x: rect.x,
      y: rect.y,
      width: rect.width,
      height: rect.height,
    };
  }

  /**
   * Get modifier keys from an event.
   * @param {Event} event
   * @returns {string[]}
   */
  function getModifiers(event) {
    var mods = [];
    if (event.ctrlKey) mods.push('ctrl');
    if (event.shiftKey) mods.push('shift');
    if (event.altKey) mods.push('alt');
    if (event.metaKey) mods.push('meta');
    return mods;
  }

  // ============================================================================
  // Action Capture
  // ============================================================================

  /**
   * Capture an action and send it to Playwright.
   * @param {string} type - Action type
   * @param {Element} target - Target element
   * @param {Event} event - Original event
   * @param {object} payload - Additional payload data
   */
  function captureAction(type, target, event, payload) {
    payload = payload || {};
    if (!target || target === document || target === window) return;

    var action = {
      actionType: type,
      timestamp: Date.now(),
      selector: generateSelectors(target),
      elementMeta: extractElementMeta(target),
      boundingBox: getBoundingBox(target),
      cursorPos: event ? { x: event.clientX || 0, y: event.clientY || 0 } : null,
      url: window.location.href,
      frameId: window.frameElement ? window.frameElement.id || 'frame' : null,
      payload: payload,
    };

    // Send to Playwright via exposed function
    if (typeof window.__recordAction === 'function') {
      window.__recordAction(action);
    } else {
      console.warn('[Recording] __recordAction not available');
    }
  }

  // ============================================================================
  // Event Handlers
  // ============================================================================

  // Click handler
  document.addEventListener(
    'click',
    function (e) {
      var target = e.target;

      // Ignore clicks on non-interactive elements unless they have handlers
      var tag = target.tagName.toLowerCase();
      var interactiveTags = ['a', 'button', 'input', 'select', 'textarea', 'label'];
      var hasRole = target.getAttribute('role');
      var hasClickAttr = target.hasAttribute('onclick') || target.hasAttribute('data-click');

      // Always capture if interactive or has role/handlers
      if (
        interactiveTags.indexOf(tag) !== -1 ||
        hasRole ||
        hasClickAttr ||
        target.closest('button, a')
      ) {
        captureAction('click', target, e, {
          button: e.button === 0 ? 'left' : e.button === 2 ? 'right' : 'middle',
          modifiers: getModifiers(e),
          clickCount: e.detail || 1,
        });
      }
    },
    true
  );

  // Input handler (debounced)
  document.addEventListener(
    'input',
    function (e) {
      var target = e.target;

      // Only capture for form elements
      if (target.tagName !== 'INPUT' && target.tagName !== 'TEXTAREA') return;

      // Debounce: reset timer and update buffer
      if (inputTarget !== target) {
        flushInput();
        inputTarget = target;
      }

      inputBuffer = target.value;
      clearTimeout(inputTimeout);
      inputTimeout = setTimeout(flushInput, CONFIG.INPUT_DEBOUNCE_MS);
    },
    true
  );

  /**
   * Flush buffered input to capture action.
   */
  function flushInput() {
    if (inputBuffer && inputTarget) {
      captureAction('type', inputTarget, null, {
        text: inputBuffer,
      });
    }
    inputBuffer = '';
    inputTarget = null;
  }

  // Change handler (for select elements)
  document.addEventListener(
    'change',
    function (e) {
      var target = e.target;

      if (target.tagName === 'SELECT') {
        var option = target.options[target.selectedIndex];
        captureAction('select', target, e, {
          value: target.value,
          selectedText: option ? option.text : '',
          selectedIndex: target.selectedIndex,
        });
      }
    },
    true
  );

  // Scroll handler (debounced)
  document.addEventListener(
    'scroll',
    function (e) {
      clearTimeout(scrollTimeout);
      scrollTimeout = setTimeout(function () {
        var target = e.target === document ? document.documentElement : e.target;
        var scrollX = window.scrollX;
        var scrollY = window.scrollY;

        // Only capture if scroll position changed significantly
        var deltaX = Math.abs(scrollX - lastScrollPos.x);
        var deltaY = Math.abs(scrollY - lastScrollPos.y);

        if (deltaX > 50 || deltaY > 50) {
          captureAction('scroll', target, e, {
            scrollX: scrollX,
            scrollY: scrollY,
            deltaX: scrollX - lastScrollPos.x,
            deltaY: scrollY - lastScrollPos.y,
          });
          lastScrollPos = { x: scrollX, y: scrollY };
        }
      }, CONFIG.SCROLL_DEBOUNCE_MS);
    },
    true
  );

  // Focus handler
  document.addEventListener(
    'focus',
    function (e) {
      var target = e.target;

      // Only capture focus on form elements
      if (
        target.tagName === 'INPUT' ||
        target.tagName === 'TEXTAREA' ||
        target.tagName === 'SELECT'
      ) {
        captureAction('focus', target, e, {});
      }
    },
    true
  );

  // Keydown handler (for special keys)
  document.addEventListener(
    'keydown',
    function (e) {
      // Capture special key combinations
      var specialKeys = [
        'Enter',
        'Escape',
        'Tab',
        'Backspace',
        'Delete',
        'ArrowUp',
        'ArrowDown',
        'ArrowLeft',
        'ArrowRight',
      ];

      if (specialKeys.indexOf(e.key) !== -1 || e.ctrlKey || e.metaKey) {
        var target = e.target;
        captureAction('keypress', target, e, {
          key: e.key,
          code: e.code,
          modifiers: getModifiers(e),
        });
      }
    },
    true
  );

  // Flush any pending input on unload
  window.addEventListener('beforeunload', function () {
    flushInput();
  });

  // ============================================================================
  // Navigation Event Handlers
  // ============================================================================

  // Track the current URL for navigation detection
  var lastNavigationUrl = window.location.href;

  /**
   * Capture a navigation action.
   * @param {string} targetUrl
   * @param {string} cause
   */
  function captureNavigation(targetUrl, cause) {
    // Skip if URL hasn't changed or is about:blank
    if (!targetUrl || targetUrl === 'about:blank' || targetUrl === lastNavigationUrl) {
      return;
    }

    lastNavigationUrl = targetUrl;

    // Create a minimal element for the action (document body as fallback)
    var target = document.body || document.documentElement;

    var action = {
      actionType: 'navigate',
      timestamp: Date.now(),
      selector: null, // Navigation doesn't need a selector
      elementMeta: {
        tagName: 'document',
        isVisible: true,
        isEnabled: true,
      },
      boundingBox: null,
      cursorPos: null,
      url: targetUrl,
      frameId: null,
      payload: {
        targetUrl: targetUrl,
        cause: cause, // 'link', 'history', 'popstate', 'hash'
      },
    };

    // Send to Playwright via exposed function
    if (typeof window.__recordAction === 'function') {
      window.__recordAction(action);
    }
  }

  // Intercept link clicks that will navigate
  // Note: We capture the intent - the actual navigation will be captured by the controller
  // This provides a more complete picture when the link click causes navigation
  document.addEventListener(
    'click',
    function (e) {
      var link = e.target.closest('a[href]');
      if (!link) return;

      var href = link.getAttribute('href');
      if (!href) return;

      // Skip anchors, javascript:, and other non-navigating links
      if (
        href.startsWith('#') ||
        href.startsWith('javascript:') ||
        href.startsWith('mailto:') ||
        href.startsWith('tel:')
      ) {
        return;
      }

      // Skip links that open in new tabs/windows
      var target = link.getAttribute('target');
      if (target === '_blank') return;

      // Skip if default prevented (handled by JS)
      if (e.defaultPrevented) return;

      // Skip if modifier keys held (user wants new tab)
      if (e.ctrlKey || e.metaKey || e.shiftKey) return;

      // Build full URL
      try {
        var fullUrl = new URL(href, window.location.href).href;
        captureNavigation(fullUrl, 'link');
      } catch (err) {
        // Invalid URL, skip
      }
    },
    true
  );

  // Intercept History API calls
  var originalPushState = window.history.pushState;
  var originalReplaceState = window.history.replaceState;

  window.history.pushState = function () {
    var result = originalPushState.apply(this, arguments);
    // After pushState, the URL has changed
    setTimeout(function () {
      captureNavigation(window.location.href, 'history');
    }, 0);
    return result;
  };

  window.history.replaceState = function () {
    var result = originalReplaceState.apply(this, arguments);
    // After replaceState, the URL might have changed
    setTimeout(function () {
      captureNavigation(window.location.href, 'history');
    }, 0);
    return result;
  };

  // Capture browser back/forward navigation
  window.addEventListener('popstate', function () {
    captureNavigation(window.location.href, 'popstate');
  });

  // Capture hash changes (SPA routing)
  window.addEventListener('hashchange', function () {
    captureNavigation(window.location.href, 'hash');
  });

  // Expose generateSelectors for testing/debugging
  window.__generateSelectors = generateSelectors;

  console.log('[Recording] Event listeners attached');
})();
