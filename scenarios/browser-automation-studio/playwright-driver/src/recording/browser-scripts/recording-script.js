/**
 * Recording Script - Browser-Side Event Capture
 *
 * This script is injected into browser pages to capture user actions during recording.
 * It runs in the MAIN context via context.addInitScript() and communicates with the
 * Playwright driver via context.exposeBinding().
 *
 * IMPORTANT: This file runs in the BROWSER context, not Node.js.
 * - No imports/requires - must be self-contained
 * - No TypeScript - plain JavaScript only
 * - Placeholders are replaced at injection time:
 *   - INJECTED_CONFIG: Configuration from selector-config.ts
 *   - INJECTED_BINDING_NAME: The binding name for event communication
 *   - RECORDING_CONTROL_MESSAGE_TYPE: Message type for start/stop control
 *
 * ARCHITECTURE:
 * 1. Handler Registry - All event handlers registered in one place
 * 2. ActionType Mapping - Clear mapping from browser events to action types
 * 3. Modular Handlers - Each event type has its own handler function
 * 4. Configurable Categories - Events grouped into enable/disable categories
 * 5. Unified captureAction - Single function to send events to backend
 *
 * @see ../init-script-generator.ts - The Node.js module that generates this script
 * @see ../context-initializer.ts - Sets up the init script and binding
 * @see ../selector-config.ts - Configuration source of truth
 */

(function () {
  'use strict';

  // Wrap everything in try-catch to detect script errors
  try {

  // ============================================================================
  // SECTION 0: Idempotency & Cleanup
  // ============================================================================

  // IDEMPOTENCY: Clean up previous instance if exists
  // This ensures re-injection doesn't cause duplicate event handlers
  if (typeof window.__vrooli_recording_cleanup === 'function') {
    try {
      window.__vrooli_recording_cleanup();
      console.log('[Recording] Previous instance cleaned up');
    } catch (e) {
      console.warn('[Recording] Cleanup failed:', e.message);
    }
  }

  // Generate unique page load ID for duplicate detection
  // This survives soft navigations but resets on hard page loads
  if (!document.__vrooli_page_load_id) {
    document.__vrooli_page_load_id = Date.now().toString(36) + Math.random().toString(36).slice(2);
  }

  // IDEMPOTENCY: Check for existing initialization on THIS page load
  // Skip if already initialized with the same page load ID
  if (window.__recordingInitialized &&
      window.__vrooli_recording_page_load_id === document.__vrooli_page_load_id) {
    console.log('[Recording] Already initialized for this page load, skipping');
    return;
  }

  // Track that we're initializing for this page load
  window.__vrooli_recording_page_load_id = document.__vrooli_page_load_id;

  // ============================================================================
  // SECTION 1: Initialization & State
  // ============================================================================

  var BINDING_NAME = '__INJECTED_BINDING_NAME__';
  var MESSAGE_TYPE = '__RECORDING_CONTROL_MESSAGE_TYPE__';
  var EVENT_MESSAGE_TYPE = '__VROOLI_RECORDING_EVENT__';

  // VERIFICATION MARKERS - Set immediately on execution
  // These allow Node.js to verify script injection worked
  window.__vrooli_recording_script_loaded = true;
  window.__vrooli_recording_script_load_time = Date.now();
  window.__vrooli_recording_script_version = '2.1.0';
  window.__vrooli_recording_script_context = 'MAIN'; // Proves we're in MAIN context

  // Track all registered event listeners for cleanup
  // This enables safe re-initialization without listener accumulation
  var registeredListeners = [];

  /**
   * Add an event listener with tracking for cleanup.
   * Use this instead of addEventListener to enable cleanup on re-injection.
   */
  function addTrackedListener(target, event, handler, options) {
    target.addEventListener(event, handler, options);
    registeredListeners.push({ target: target, event: event, handler: handler, options: options });
  }

  /**
   * Send a recording event via fetch to the intercepted route.
   * This bypasses the isolated context issue entirely.
   *
   * WHY fetch instead of binding:
   * - This init script runs in MAIN context (via context.addInitScript)
   * - The exposed binding is only available in ISOLATED context (rebrowser-playwright)
   * - Fetch works from any context and is intercepted by Playwright's route handler
   */
  var RECORDING_EVENT_URL = '/__vrooli_recording_event__';

  function sendEvent(eventData) {
    // Use fetch with keepalive to ensure the request completes even if page navigates
    fetch(RECORDING_EVENT_URL, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(eventData),
      keepalive: true,
    }).catch(function(e) {
      // Silently ignore fetch errors (page might be navigating)
      console.warn('[Recording] Failed to send event:', e.message);
    });
  }

  window.__recordingInitialized = true;
  console.log('[Recording] Init script loaded');

  // Recording state - starts ACTIVE by default
  // The Node.js side controls whether events are actually processed
  // This avoids complex cross-context activation issues with rebrowser-playwright
  var isActive = true;
  var sessionId = 'auto';

  console.log('[Recording] Starting in active mode');

  // ============================================================================
  // SECTION 2: Configuration (injected from selector-config.ts)
  // ============================================================================

  var SHARED_CONFIG = __INJECTED_CONFIG__;

  // Flatten config for easier access
  var CONFIG = {
    // Debounce timings
    INPUT_DEBOUNCE_MS: SHARED_CONFIG.RECORDING_DEBOUNCE.input,
    SCROLL_DEBOUNCE_MS: SHARED_CONFIG.RECORDING_DEBOUNCE.scroll,
    RESIZE_DEBOUNCE_MS: SHARED_CONFIG.RECORDING_DEBOUNCE.resize,
    HOVER_DEBOUNCE_MS: SHARED_CONFIG.RECORDING_DEBOUNCE.hover || 200,

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

    // Event categories
    EVENT_CATEGORIES: SHARED_CONFIG.RECORDING_EVENT_CATEGORIES,
  };

  // ============================================================================
  // SECTION 3: Event Category Helpers
  // ============================================================================

  /**
   * Check if an event category is enabled.
   * @param {string} category - Category name (core, focus, hover, dragDrop, gesture)
   * @returns {boolean}
   */
  function isCategoryEnabled(category) {
    var cat = CONFIG.EVENT_CATEGORIES[category];
    return cat ? cat.enabled : false;
  }

  // ============================================================================
  // SECTION 4: Handler State
  // ============================================================================

  // Input debouncing state
  var inputBuffer = '';
  var inputTarget = null;
  var inputTimeout = null;

  // Scroll debouncing state
  var scrollTimeout = null;
  var lastScrollPos = { x: 0, y: 0 };

  // Hover debouncing state
  var hoverTimeout = null;
  var lastHoverTarget = null;

  // Drag/drop state
  var dragState = null;

  // Touch gesture state
  var touchState = null;

  // ============================================================================
  // SECTION 5: Selector Generation
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
   */
  function escapeXPathString(str) {
    var sq = "'";
    var dq = '"';
    if (str.indexOf(sq) !== -1 && str.indexOf(dq) !== -1) {
      var parts = str.split(sq);
      var result = 'concat(';
      for (var i = 0; i < parts.length; i++) {
        if (i > 0) result += ', ' + dq + sq + dq + ', ';
        result += sq + parts[i] + sq;
      }
      result += ')';
      return result;
    }
    if (str.indexOf(sq) !== -1) {
      return dq + str + dq;
    }
    return sq + str + sq;
  }

  /**
   * Generate a positional XPath selector.
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
  // SECTION 6: Helper Functions
  // ============================================================================

  function isUniqueSelector(selector) {
    try {
      return document.querySelectorAll(selector).length === 1;
    } catch (e) {
      return false;
    }
  }

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

  function isDynamicId(id) {
    for (var i = 0; i < CONFIG.DYNAMIC_ID_PATTERNS.length; i++) {
      if (CONFIG.DYNAMIC_ID_PATTERNS[i].test(id)) {
        return true;
      }
    }
    return false;
  }

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

  function getTextTag(element) {
    var tag = element.tagName.toLowerCase();
    return CONFIG.TEXT_CONTENT_TAGS.indexOf(tag) !== -1 ? tag : null;
  }

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

  function getVisibleText(element) {
    if (element.tagName === 'INPUT' || element.tagName === 'TEXTAREA') {
      return (element.value || element.placeholder || '').slice(0, CONFIG.MAX_TEXT_LENGTH);
    }
    return (element.textContent || '').trim().slice(0, CONFIG.MAX_TEXT_LENGTH);
  }

  function escapeCssSelector(str) {
    return str.replace(/([!"#$%&'()*+,./:;<=>?@[\\\]^`{|}~])/g, '\\$1');
  }

  function escapeCssAttr(str) {
    return str.replace(/"/g, '\\"').replace(/\n/g, '\\n');
  }

  // ============================================================================
  // SECTION 7: Element Metadata Extraction
  // ============================================================================

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

  function getBoundingBox(element) {
    var rect = element.getBoundingClientRect();
    return {
      x: rect.x,
      y: rect.y,
      width: rect.width,
      height: rect.height,
    };
  }

  function getModifiers(event) {
    var mods = [];
    if (event.ctrlKey) mods.push('ctrl');
    if (event.shiftKey) mods.push('shift');
    if (event.altKey) mods.push('alt');
    if (event.metaKey) mods.push('meta');
    return mods;
  }

  // ============================================================================
  // SECTION 8: Core Capture Function
  // ============================================================================

  /**
   * Capture an action and send it to Playwright.
   * @param {string} type - Action type
   * @param {Element} target - Target element
   * @param {Event} event - Original event
   * @param {object} payload - Additional payload data
   */
  function captureAction(type, target, event, payload) {
    // Early exit if not active
    if (!isActive) return;

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

    console.log('[Recording] Event captured:', type, 'active:', isActive);

    // Send to Playwright via postMessage bridge
    // The bridge in ISOLATED context receives this and forwards to the exposed binding
    sendEvent(action);
  }

  // ============================================================================
  // SECTION 9: Handler Functions - Core (Always Enabled)
  // ============================================================================

  /**
   * Handle click events - captures ALL clicks (no filtering).
   */
  function handleClick(e) {
    console.error('[Recording] DIAGNOSTIC: Click detected, isActive=' + isActive);
    captureAction('click', e.target, e, {
      button: e.button === 0 ? 'left' : e.button === 2 ? 'right' : 'middle',
      modifiers: getModifiers(e),
      clickCount: e.detail || 1,
    });
  }

  /**
   * Handle double-click events.
   */
  function handleDoubleClick(e) {
    captureAction('click', e.target, e, {
      button: 'left',
      modifiers: getModifiers(e),
      clickCount: 2,
    });
  }

  /**
   * Handle context menu (right-click) events.
   */
  function handleContextMenu(e) {
    captureAction('click', e.target, e, {
      button: 'right',
      modifiers: getModifiers(e),
      clickCount: 1,
    });
  }

  /**
   * Handle input events (debounced).
   */
  function handleInput(e) {
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
  }

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

  /**
   * Handle change events (for select elements).
   */
  function handleChange(e) {
    var target = e.target;

    if (target.tagName === 'SELECT') {
      var option = target.options[target.selectedIndex];
      captureAction('select', target, e, {
        value: target.value,
        selectedText: option ? option.text : '',
        selectedIndex: target.selectedIndex,
      });
    }
  }

  /**
   * Handle scroll events (debounced).
   */
  function handleScroll(e) {
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
  }

  /**
   * Handle keydown events (for special keys).
   */
  function handleKeydown(e) {
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
      captureAction('keyboard', e.target, e, {
        key: e.key,
        code: e.code,
        modifiers: getModifiers(e),
      });
    }
  }

  // ============================================================================
  // SECTION 10: Handler Functions - Focus Category
  // ============================================================================

  /**
   * Handle focus events.
   */
  function handleFocus(e) {
    var target = e.target;

    // Only capture focus on form elements
    if (
      target.tagName === 'INPUT' ||
      target.tagName === 'TEXTAREA' ||
      target.tagName === 'SELECT'
    ) {
      captureAction('focus', target, e, {});
    }
  }

  /**
   * Handle blur events.
   */
  function handleBlur(e) {
    var target = e.target;

    // Only capture blur on form elements
    if (
      target.tagName === 'INPUT' ||
      target.tagName === 'TEXTAREA' ||
      target.tagName === 'SELECT'
    ) {
      captureAction('blur', target, e, {});
    }
  }

  // ============================================================================
  // SECTION 11: Handler Functions - Hover Category
  // ============================================================================

  /**
   * Handle mouseenter events (debounced).
   */
  function handleMouseenter(e) {
    var target = e.target;

    // Skip if same target (prevent spam)
    if (target === lastHoverTarget) return;

    // Debounce hover events
    clearTimeout(hoverTimeout);
    hoverTimeout = setTimeout(function () {
      lastHoverTarget = target;
      captureAction('hover', target, e, {});
    }, CONFIG.HOVER_DEBOUNCE_MS);
  }

  // ============================================================================
  // SECTION 12: Handler Functions - Drag/Drop Category
  // ============================================================================

  /**
   * Handle dragstart events.
   */
  function handleDragstart(e) {
    var target = e.target;
    dragState = {
      source: target,
      startX: e.clientX,
      startY: e.clientY,
      startTime: Date.now(),
    };

    captureAction('drag-drop', target, e, {
      phase: 'start',
      dataTransfer: e.dataTransfer ? Array.from(e.dataTransfer.types) : [],
    });
  }

  /**
   * Handle drop events.
   */
  function handleDrop(e) {
    if (!dragState) return;

    var target = e.target;
    captureAction('drag-drop', target, e, {
      phase: 'drop',
      sourceSelector: generateSelectors(dragState.source).primary,
      targetSelector: generateSelectors(target).primary,
      duration: Date.now() - dragState.startTime,
    });

    dragState = null;
  }

  /**
   * Handle dragend events (cleanup).
   */
  function handleDragend() {
    dragState = null;
  }

  // ============================================================================
  // SECTION 13: Handler Functions - Gesture Category
  // ============================================================================

  /**
   * Handle touch events for gesture detection.
   */
  function handleTouch(e) {
    var type = e.type;

    if (type === 'touchstart') {
      if (e.touches.length === 1) {
        touchState = {
          startX: e.touches[0].clientX,
          startY: e.touches[0].clientY,
          startTime: Date.now(),
          target: e.target,
        };
      }
    } else if (type === 'touchend' && touchState) {
      if (e.changedTouches.length === 1) {
        var endX = e.changedTouches[0].clientX;
        var endY = e.changedTouches[0].clientY;
        var deltaX = endX - touchState.startX;
        var deltaY = endY - touchState.startY;
        var duration = Date.now() - touchState.startTime;

        // Determine gesture type
        var gestureType = 'tap';
        if (Math.abs(deltaX) > 50 || Math.abs(deltaY) > 50) {
          if (Math.abs(deltaX) > Math.abs(deltaY)) {
            gestureType = deltaX > 0 ? 'swipe-right' : 'swipe-left';
          } else {
            gestureType = deltaY > 0 ? 'swipe-down' : 'swipe-up';
          }
        } else if (duration > 500) {
          gestureType = 'long-press';
        }

        captureAction('gesture', touchState.target, e, {
          gestureType: gestureType,
          deltaX: deltaX,
          deltaY: deltaY,
          duration: duration,
        });
      }

      touchState = null;
    }
  }

  // ============================================================================
  // SECTION 14: Handler Registration
  // ============================================================================

  /**
   * Register core handlers (always enabled).
   * Uses addTrackedListener for cleanup support.
   */
  function registerCoreHandlers() {
    addTrackedListener(document, 'click', handleClick, true);
    addTrackedListener(document, 'dblclick', handleDoubleClick, true);
    addTrackedListener(document, 'contextmenu', handleContextMenu, true);
    addTrackedListener(document, 'input', handleInput, true);
    addTrackedListener(document, 'change', handleChange, true);
    addTrackedListener(document, 'scroll', handleScroll, true);
    addTrackedListener(document, 'keydown', handleKeydown, true);

    // Flush input on unload
    addTrackedListener(window, 'beforeunload', function () {
      flushInput();
    });
  }

  /**
   * Register focus handlers (if enabled).
   * Uses addTrackedListener for cleanup support.
   */
  function registerFocusHandlers() {
    addTrackedListener(document, 'focus', handleFocus, true);
    addTrackedListener(document, 'blur', handleBlur, true);
  }

  /**
   * Register hover handlers (if enabled).
   * Uses addTrackedListener for cleanup support.
   */
  function registerHoverHandlers() {
    addTrackedListener(document, 'mouseenter', handleMouseenter, true);
  }

  /**
   * Register drag/drop handlers (if enabled).
   * Uses addTrackedListener for cleanup support.
   */
  function registerDragDropHandlers() {
    addTrackedListener(document, 'dragstart', handleDragstart, true);
    addTrackedListener(document, 'drop', handleDrop, true);
    addTrackedListener(document, 'dragend', handleDragend, true);
  }

  /**
   * Register gesture handlers (if enabled).
   * Uses addTrackedListener for cleanup support.
   */
  function registerGestureHandlers() {
    addTrackedListener(document, 'touchstart', handleTouch, true);
    addTrackedListener(document, 'touchend', handleTouch, true);
  }

  /**
   * Register all handlers based on category configuration.
   */
  function registerAllHandlers() {
    // Core handlers - always registered
    registerCoreHandlers();

    // Focus handlers - register if enabled
    if (isCategoryEnabled('focus')) {
      registerFocusHandlers();
      console.log('[Recording] Focus handlers registered');
    }

    // Hover handlers - register if enabled
    if (isCategoryEnabled('hover')) {
      registerHoverHandlers();
      console.log('[Recording] Hover handlers registered');
    }

    // Drag/drop handlers - register if enabled
    if (isCategoryEnabled('dragDrop')) {
      registerDragDropHandlers();
      console.log('[Recording] Drag/drop handlers registered');
    }

    // Gesture handlers - register if enabled
    if (isCategoryEnabled('gesture')) {
      registerGestureHandlers();
      console.log('[Recording] Gesture handlers registered');
    }
  }

  // ============================================================================
  // SECTION 15: Navigation Event Handlers
  // ============================================================================

  var lastNavigationUrl = window.location.href;

  /**
   * Capture a navigation action.
   */
  function captureNavigation(targetUrl, cause) {
    if (!isActive) return;

    if (!targetUrl || targetUrl === 'about:blank' || targetUrl === lastNavigationUrl) {
      return;
    }

    lastNavigationUrl = targetUrl;

    var target = document.body || document.documentElement;

    var action = {
      actionType: 'navigate',
      timestamp: Date.now(),
      selector: null,
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
        cause: cause,
      },
    };

    console.log('[Recording] Navigation captured:', cause, targetUrl.slice(0, 50));

    // Send to Playwright via postMessage bridge
    sendEvent(action);
  }

  // Store original History API methods for restoration during cleanup
  var originalPushState = window.history.pushState;
  var originalReplaceState = window.history.replaceState;

  /**
   * Register navigation handlers.
   * Uses addTrackedListener for cleanup support.
   */
  function registerNavigationHandlers() {
    // Intercept link clicks that will navigate
    addTrackedListener(
      document,
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
    window.history.pushState = function () {
      var result = originalPushState.apply(this, arguments);
      setTimeout(function () {
        captureNavigation(window.location.href, 'history');
      }, 0);
      return result;
    };

    window.history.replaceState = function () {
      var result = originalReplaceState.apply(this, arguments);
      setTimeout(function () {
        captureNavigation(window.location.href, 'history');
      }, 0);
      return result;
    };

    // Capture browser back/forward navigation
    addTrackedListener(window, 'popstate', function () {
      captureNavigation(window.location.href, 'popstate');
    });

    // Capture hash changes (SPA routing)
    addTrackedListener(window, 'hashchange', function () {
      captureNavigation(window.location.href, 'hash');
    });
  }

  // ============================================================================
  // SECTION 16: Activation/Deactivation
  // ============================================================================

  addTrackedListener(window, 'message', function (event) {
    var data = event.data;
    if (!data || data.type !== MESSAGE_TYPE) return;

    console.log('[Recording] Received control message:', data.action, 'source:', event.source === window ? 'window' : 'other');

    if (data.action === 'start' && data.sessionId) {
      isActive = true;
      sessionId = data.sessionId;
      console.log('[Recording] Activated for session:', sessionId);
      console.error('[Recording] DIAGNOSTIC: Activated for session:', sessionId);
    } else if (data.action === 'stop') {
      isActive = false;
      sessionId = null;
      console.log('[Recording] Deactivated');
    }
  });

  // ============================================================================
  // SECTION 17: Initialize
  // ============================================================================

  // Register all handlers
  registerAllHandlers();
  registerNavigationHandlers();

  // Expose helpers for testing/debugging
  window.__generateSelectors = generateSelectors;
  window.__isRecordingActive = function () {
    return isActive;
  };

  // ============================================================================
  // SECTION 18: Cleanup Function (for idempotent re-injection)
  // ============================================================================

  /**
   * Cleanup function - called before re-initialization.
   * Removes all event listeners and resets state.
   * This enables safe re-injection without listener accumulation.
   */
  window.__vrooli_recording_cleanup = function() {
    // Remove all tracked event listeners
    for (var i = 0; i < registeredListeners.length; i++) {
      var l = registeredListeners[i];
      try {
        l.target.removeEventListener(l.event, l.handler, l.options);
      } catch (e) {
        // Target may have been removed from DOM, ignore
      }
    }
    registeredListeners = [];

    // Clear pending debounce timeouts
    if (typeof inputTimeout !== 'undefined' && inputTimeout) {
      clearTimeout(inputTimeout);
    }
    if (typeof scrollTimeout !== 'undefined' && scrollTimeout) {
      clearTimeout(scrollTimeout);
    }
    if (typeof hoverTimeout !== 'undefined' && hoverTimeout) {
      clearTimeout(hoverTimeout);
    }

    // Restore original History API methods
    if (originalPushState) {
      window.history.pushState = originalPushState;
    }
    if (originalReplaceState) {
      window.history.replaceState = originalReplaceState;
    }

    // Reset state
    window.__recordingInitialized = false;
    window.__vrooli_recording_ready = false;
    isActive = false;

    console.log('[Recording] Cleanup complete');
  };

  // ============================================================================
  // SECTION 19: Verification Markers (set at end of initialization)
  // ============================================================================

  // VERIFICATION: Mark script as fully initialized and ready
  window.__vrooli_recording_ready = true;
  window.__vrooli_recording_handlers_count = registeredListeners.length;

  console.log('[Recording] Event listeners attached (' + registeredListeners.length + ' handlers)');
  console.error('[Recording] DIAGNOSTIC: Script loaded and initialized');

  } catch (e) {
    console.error('[Recording] FATAL ERROR during initialization:', e.message, e.stack);
    // Mark as not ready on error
    window.__vrooli_recording_ready = false;
    window.__vrooli_recording_init_error = e.message;
  }
})();
