/**
 * Recording Injector
 *
 * JavaScript code that gets injected into browser pages to capture user actions.
 * This script runs in the page context and communicates with the Playwright driver
 * via exposed functions.
 *
 * The script is designed to be stringified and injected via page.evaluate().
 */

import type { RawBrowserEvent } from './types';

/**
 * Generate the recording script that will be injected into pages.
 * This returns a string containing self-contained JavaScript.
 */
export function getRecordingScript(): string {
  return `
(function() {
  'use strict';

  // Prevent double-injection
  if (window.__recordingActive) {
    console.log('[Recording] Already active, skipping injection');
    return;
  }
  window.__recordingActive = true;
  console.log('[Recording] Injected recording script');

  // ============================================================================
  // Configuration
  // ============================================================================

  const CONFIG = {
    // Debounce timings
    INPUT_DEBOUNCE_MS: 500,
    SCROLL_DEBOUNCE_MS: 150,
    RESIZE_DEBOUNCE_MS: 200,

    // Limits
    MAX_TEXT_LENGTH: 100,
    MAX_SELECTOR_DEPTH: 5,

    // Unstable class patterns to filter out
    UNSTABLE_CLASS_PATTERNS: [
      /^css-[a-z0-9]+$/i,      // CSS-in-JS (Emotion, etc.)
      /^sc-[a-zA-Z]+$/,        // styled-components
      /^_[a-zA-Z0-9]+$/,       // CSS modules
      /^[a-zA-Z]+-[0-9]+$/,    // Generic hash patterns
      /^jsx-[a-z0-9]+$/i,      // Next.js styled-jsx
      /^svelte-[a-z0-9]+$/i,   // Svelte scoped styles
      /^v-[a-z0-9]+$/i,        // Vue scoped styles
    ],
  };

  // ============================================================================
  // State
  // ============================================================================

  let inputBuffer = '';
  let inputTarget = null;
  let inputTimeout = null;
  let scrollTimeout = null;
  let lastScrollPos = { x: 0, y: 0 };

  // ============================================================================
  // Selector Generation
  // ============================================================================

  /**
   * Generate multiple selector candidates for an element.
   */
  function generateSelectors(element) {
    const candidates = [];

    // Strategy 1: data-testid
    const testIdSelector = generateTestIdSelector(element);
    if (testIdSelector) candidates.push(testIdSelector);

    // Strategy 2: Unique ID
    const idSelector = generateIdSelector(element);
    if (idSelector) candidates.push(idSelector);

    // Strategy 3: ARIA attributes
    const ariaSelector = generateAriaSelector(element);
    if (ariaSelector) candidates.push(ariaSelector);

    // Strategy 4: Tag + text
    const textSelector = generateTextSelector(element);
    if (textSelector) candidates.push(textSelector);

    // Strategy 5: Data attributes
    const dataAttrSelector = generateDataAttrSelector(element);
    if (dataAttrSelector) candidates.push(dataAttrSelector);

    // Strategy 6: CSS path
    const cssSelector = generateCssPath(element);
    if (cssSelector) candidates.push(cssSelector);

    // Strategy 7: XPath
    const xpathSelector = generateXPathSelector(element);
    if (xpathSelector) candidates.push(xpathSelector);

    // Sort by confidence
    candidates.sort((a, b) => b.confidence - a.confidence);

    return {
      primary: candidates[0]?.value || generateFallbackSelector(element),
      candidates: candidates,
    };
  }

  function generateTestIdSelector(element) {
    const testIdAttrs = ['data-testid', 'data-test-id', 'data-test', 'data-cy', 'data-qa'];

    for (const attr of testIdAttrs) {
      const value = element.getAttribute(attr);
      if (value) {
        const selector = '[' + attr + '="' + escapeCssAttr(value) + '"]';
        if (isUniqueSelector(selector)) {
          return { type: 'data-testid', value: selector, confidence: 0.98, specificity: 100 };
        }
      }
    }
    return null;
  }

  function generateIdSelector(element) {
    const id = element.id;
    if (!id) return null;

    const isDynamic = isDynamicId(id);
    const selector = '#' + escapeCssSelector(id);

    if (isUniqueSelector(selector)) {
      return {
        type: 'id',
        value: selector,
        confidence: isDynamic ? 0.6 : 0.95,
        specificity: 95,
      };
    }
    return null;
  }

  function generateAriaSelector(element) {
    const ariaLabel = element.getAttribute('aria-label');
    if (ariaLabel) {
      const selector = '[aria-label="' + escapeCssAttr(ariaLabel) + '"]';
      if (isUniqueSelector(selector)) {
        return { type: 'aria', value: selector, confidence: 0.85, specificity: 80 };
      }
    }
    return null;
  }

  function generateTextSelector(element) {
    const tag = getTextTag(element);
    if (!tag) return null;

    const text = getVisibleText(element);
    if (!text || text.length < 3 || text.length > 50) return null;

    // Truncate text to first 30 chars for selector stability
    const selectorText = text.length > 30 ? text.slice(0, 30) : text;
    const selector = tag + ':has-text("' + selectorText.replace(/"/g, '\\\\"') + '")';
    return {
      type: 'text',
      value: selector,
      confidence: text.length > 15 ? 0.6 : 0.55,
      specificity: 50,
    };
  }

  function generateDataAttrSelector(element) {
    const skipAttrs = ['testid', 'test-id', 'test', 'cy', 'qa'];

    for (const attr of element.attributes) {
      if (!attr.name.startsWith('data-')) continue;
      const suffix = attr.name.slice(5);
      if (skipAttrs.includes(suffix)) continue;

      const selector = '[' + attr.name + '="' + escapeCssAttr(attr.value) + '"]';
      if (isUniqueSelector(selector)) {
        return { type: 'data-attr', value: selector, confidence: 0.7, specificity: 65 };
      }
    }
    return null;
  }

  function generateCssPath(element) {
    const parts = [];
    let current = element;
    let depth = 0;

    while (current && current !== document.body && depth < CONFIG.MAX_SELECTOR_DEPTH) {
      const tag = current.tagName.toLowerCase();
      const stableClasses = getStableClasses(current);

      let part = tag;
      if (stableClasses.length > 0) {
        part = tag + '.' + stableClasses.slice(0, 2).join('.');
      }

      parts.unshift(part);

      const selector = parts.join(' > ');
      if (isUniqueSelector(selector)) {
        return {
          type: 'css',
          value: selector,
          confidence: assessCssStability(selector, stableClasses),
          specificity: 60 - depth * 5,
        };
      }

      current = current.parentElement;
      depth++;
    }

    // Try with nth-child
    return generateCssPathWithNthChild(element);
  }

  function generateCssPathWithNthChild(element) {
    const parts = [];
    let current = element;
    let depth = 0;

    while (current && current !== document.body && depth < CONFIG.MAX_SELECTOR_DEPTH) {
      const tag = current.tagName.toLowerCase();
      const parent = current.parentElement;

      let part = tag;
      if (parent) {
        const siblings = Array.from(parent.children).filter(
          s => s.tagName.toLowerCase() === tag
        );
        if (siblings.length > 1) {
          const index = siblings.indexOf(current) + 1;
          part = tag + ':nth-of-type(' + index + ')';
        }
      }

      parts.unshift(part);

      const selector = parts.join(' > ');
      if (isUniqueSelector(selector)) {
        return { type: 'css', value: selector, confidence: 0.5, specificity: 40 - depth * 5 };
      }

      current = parent;
      depth++;
    }
    return null;
  }

  function generateXPathSelector(element) {
    const tag = element.tagName.toLowerCase();
    const text = getVisibleText(element);

    if (text && text.length > 0 && text.length <= 50) {
      const escapedText = escapeXPathString(text);
      const xpath = '//' + tag + '[contains(text(), ' + escapedText + ')]';
      if (isUniqueXPath(xpath)) {
        return { type: 'xpath', value: xpath, confidence: 0.55, specificity: 35 };
      }
    }

    // Positional fallback
    return {
      type: 'xpath',
      value: generatePositionalXPath(element),
      confidence: 0.4,
      specificity: 25,
    };
  }

  /**
   * Escape string for use in XPath expressions.
   * Handles strings containing single quotes, double quotes, or both.
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

  function generatePositionalXPath(element) {
    const parts = [];
    let current = element;

    while (current && current !== document.documentElement) {
      const tag = current.tagName.toLowerCase();
      const parent = current.parentElement;

      if (parent) {
        const siblings = Array.from(parent.children).filter(
          s => s.tagName.toLowerCase() === tag
        );
        if (siblings.length > 1) {
          const index = siblings.indexOf(current) + 1;
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

  function generateFallbackSelector(element) {
    const tag = element.tagName.toLowerCase();

    for (const attr of element.attributes) {
      if (attr.name === 'class' || attr.name === 'style') continue;
      const selector = tag + '[' + attr.name + '="' + escapeCssAttr(attr.value) + '"]';
      if (isUniqueSelector(selector)) return selector;
    }

    return generatePositionalXPath(element);
  }

  // ============================================================================
  // Helper Functions
  // ============================================================================

  function isUniqueSelector(selector) {
    try {
      return document.querySelectorAll(selector).length === 1;
    } catch {
      return false;
    }
  }

  function isUniqueXPath(xpath) {
    try {
      const result = document.evaluate(xpath, document, null, XPathResult.ORDERED_NODE_SNAPSHOT_TYPE, null);
      return result.snapshotLength === 1;
    } catch {
      return false;
    }
  }

  function isDynamicId(id) {
    const patterns = [
      /^[a-f0-9]{8,}$/i,
      /^\\d+$/,
      /:r[0-9]+:/,
      /^:r/,
      /^ember\\d+$/,
      /^___gatsby/,
      /_\\d{10,}/,
    ];
    return patterns.some(p => p.test(id));
  }

  function getStableClasses(element) {
    return Array.from(element.classList).filter(cls => {
      if (!cls.trim()) return false;
      return !CONFIG.UNSTABLE_CLASS_PATTERNS.some(p => p.test(cls));
    });
  }

  function assessCssStability(selector, stableClasses) {
    let score = 0.65;

    const semanticPatterns = [/^btn-/, /^button-/, /^nav-/, /^form-/, /^input-/];
    for (const cls of stableClasses) {
      if (semanticPatterns.some(p => p.test(cls))) score += 0.1;
    }

    if (selector.includes('nth-child') || selector.includes('nth-of-type')) score -= 0.15;

    const depth = (selector.match(/>/g) || []).length;
    if (depth > 3) score -= 0.05 * (depth - 3);

    return Math.max(0.3, Math.min(0.9, score));
  }

  function getTextTag(element) {
    // Return CSS tag name for use with :has-text()
    const tag = element.tagName.toLowerCase();
    // Only return for elements that commonly have meaningful text
    const textTags = ['button', 'a', 'span', 'div', 'p', 'h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'label', 'li', 'td', 'th'];
    return textTags.includes(tag) ? tag : null;
  }

  function inferRole(element) {
    // Infer ARIA role for metadata (not for selectors)
    const tag = element.tagName.toLowerCase();
    const roles = {
      button: 'button',
      a: 'link',
      input: element.type === 'checkbox' ? 'checkbox' :
             element.type === 'radio' ? 'radio' :
             element.type === 'submit' ? 'button' : 'textbox',
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
    return str.replace(/([!"#$%&'()*+,./:;<=>?@[\\\\\\]^\\x60{|}~])/g, '\\\\$1');
  }

  function escapeCssAttr(str) {
    return str.replace(/"/g, '\\\\"').replace(/\\n/g, '\\\\n');
  }

  // ============================================================================
  // Element Metadata Extraction
  // ============================================================================

  function extractElementMeta(element) {
    const style = window.getComputedStyle(element);
    const tag = element.tagName.toLowerCase();

    return {
      tagName: tag,
      id: element.id || undefined,
      className: element.className || undefined,
      innerText: getVisibleText(element) || undefined,
      attributes: getRelevantAttributes(element),
      isVisible: style.display !== 'none' && style.visibility !== 'hidden' && parseFloat(style.opacity) > 0,
      isEnabled: !element.disabled,
      role: element.getAttribute('role') || inferRole(element) || undefined,
      ariaLabel: element.getAttribute('aria-label') || undefined,
    };
  }

  function getRelevantAttributes(element) {
    const attrs = {};
    const interesting = ['type', 'name', 'placeholder', 'title', 'alt', 'href', 'value', 'role'];

    for (const attr of interesting) {
      const val = element.getAttribute(attr);
      if (val) attrs[attr] = val.slice(0, 100);
    }

    for (const attr of element.attributes) {
      if (attr.name.startsWith('data-')) {
        attrs[attr.name] = attr.value.slice(0, 100);
      }
    }

    return attrs;
  }

  function getBoundingBox(element) {
    const rect = element.getBoundingClientRect();
    return {
      x: rect.x,
      y: rect.y,
      width: rect.width,
      height: rect.height,
    };
  }

  function getModifiers(event) {
    const mods = [];
    if (event.ctrlKey) mods.push('ctrl');
    if (event.shiftKey) mods.push('shift');
    if (event.altKey) mods.push('alt');
    if (event.metaKey) mods.push('meta');
    return mods;
  }

  // ============================================================================
  // Action Capture
  // ============================================================================

  function captureAction(type, target, event, payload = {}) {
    if (!target || target === document || target === window) return;

    const action = {
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
  document.addEventListener('click', function(e) {
    const target = e.target;

    // Ignore clicks on non-interactive elements unless they have handlers
    const tag = target.tagName.toLowerCase();
    const interactiveTags = ['a', 'button', 'input', 'select', 'textarea', 'label'];
    const hasRole = target.getAttribute('role');
    const hasClickAttr = target.hasAttribute('onclick') || target.hasAttribute('data-click');

    // Always capture if interactive or has role/handlers
    if (interactiveTags.includes(tag) || hasRole || hasClickAttr || target.closest('button, a')) {
      captureAction('click', target, e, {
        button: e.button === 0 ? 'left' : e.button === 2 ? 'right' : 'middle',
        modifiers: getModifiers(e),
        clickCount: e.detail || 1,
      });
    }
  }, true);

  // Input handler (debounced)
  document.addEventListener('input', function(e) {
    const target = e.target;

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
  }, true);

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
  document.addEventListener('change', function(e) {
    const target = e.target;

    if (target.tagName === 'SELECT') {
      const option = target.options[target.selectedIndex];
      captureAction('select', target, e, {
        value: target.value,
        selectedText: option ? option.text : '',
        selectedIndex: target.selectedIndex,
      });
    }
  }, true);

  // Scroll handler (debounced)
  document.addEventListener('scroll', function(e) {
    clearTimeout(scrollTimeout);
    scrollTimeout = setTimeout(function() {
      const target = e.target === document ? document.documentElement : e.target;
      const scrollX = window.scrollX;
      const scrollY = window.scrollY;

      // Only capture if scroll position changed significantly
      const deltaX = Math.abs(scrollX - lastScrollPos.x);
      const deltaY = Math.abs(scrollY - lastScrollPos.y);

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
  }, true);

  // Focus handler
  document.addEventListener('focus', function(e) {
    const target = e.target;

    // Only capture focus on form elements
    if (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.tagName === 'SELECT') {
      captureAction('focus', target, e, {});
    }
  }, true);

  // Keydown handler (for special keys)
  document.addEventListener('keydown', function(e) {
    // Capture special key combinations
    const specialKeys = ['Enter', 'Escape', 'Tab', 'Backspace', 'Delete', 'ArrowUp', 'ArrowDown', 'ArrowLeft', 'ArrowRight'];

    if (specialKeys.includes(e.key) || e.ctrlKey || e.metaKey) {
      const target = e.target;
      captureAction('keypress', target, e, {
        key: e.key,
        code: e.code,
        modifiers: getModifiers(e),
      });
    }
  }, true);

  // Flush any pending input on unload
  window.addEventListener('beforeunload', function() {
    flushInput();
  });

  // Expose generateSelectors for testing/debugging
  window.__generateSelectors = generateSelectors;

  console.log('[Recording] Event listeners attached');
})();
`;
}

/**
 * Get the cleanup script to remove recording event listeners.
 */
export function getCleanupScript(): string {
  return `
(function() {
  window.__recordingActive = false;
  console.log('[Recording] Deactivated');
})();
`;
}

/**
 * Type definition for the raw record action function exposed to browser context.
 * This receives raw browser events before normalization.
 * @internal
 */
export type RawRecordActionCallback = (action: RawBrowserEvent) => void;
