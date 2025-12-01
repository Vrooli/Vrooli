/**
 * DOM Manipulation Utilities for Test Genie
 * Helpers for working with the DOM, events, and UI interactions
 */

import { UI, MOBILE_BREAKPOINT } from './constants.js';

/**
 * Safely focus an element if it has a focus method
 * @param {HTMLElement} element
 */
export function focusElement(element) {
    if (!element || typeof element.focus !== 'function') {
        return;
    }
    try {
        element.focus();
    } catch (error) {
        console.warn('Failed to focus element:', error);
    }
}

/**
 * Toggle element visibility (show/hide)
 * @param {HTMLElement} element
 * @param {boolean} shouldShow
 */
export function toggleElementVisibility(element, shouldShow) {
    if (!element) {
        return;
    }
    if (shouldShow) {
        element.removeAttribute('hidden');
        element.style.display = '';
    } else {
        element.setAttribute('hidden', 'true');
        element.style.display = 'none';
    }
}

/**
 * Check if an overlay/dialog is currently active
 * @param {HTMLElement} overlay
 * @returns {boolean}
 */
export function isOverlayActive(overlay) {
    return overlay ? overlay.classList.contains('active') : false;
}

/**
 * Lock body scroll (for dialogs/modals)
 */
export function lockDialogScroll() {
    document.body.classList.add('dialog-open');
}

/**
 * Unlock body scroll if no dialogs are open
 * @param {Array<HTMLElement>} overlays - Array of overlay elements to check
 */
export function unlockDialogScrollIfIdle(overlays = []) {
    const dialogsOpen = overlays.some(overlay => isOverlayActive(overlay));
    if (!dialogsOpen) {
        document.body.classList.remove('dialog-open');
    }
}

/**
 * Refresh Lucide icons (re-render SVG icons)
 */
export function refreshIcons() {
    if (typeof lucide !== 'undefined' && typeof lucide.createIcons === 'function') {
        lucide.createIcons();
    }
}

/**
 * Enable drag-to-scroll functionality on a container
 * @param {HTMLElement} container
 */
export function enableDragScroll(container) {
    if (!container) {
        return;
    }

    container.classList.add('table-scroll');

    if (container.dataset.dragScrollBound === 'true') {
        return;
    }

    container.dataset.dragScrollBound = 'true';
    container.scrollLeft = 0;

    let isDragging = false;
    let startX = 0;
    let initialScrollLeft = 0;
    const skipSelector = UI.DRAG_SCROLL_SKIP_SELECTOR;

    const onPointerDown = (event) => {
        if (event.pointerType !== 'mouse' && event.pointerType !== 'pen') {
            return;
        }
        if (event.button !== 0) {
            return;
        }
        if (event.target.closest(skipSelector)) {
            return;
        }

        isDragging = true;
        startX = event.clientX;
        initialScrollLeft = container.scrollLeft;
        container.classList.add('is-dragging');
        container.setPointerCapture?.(event.pointerId);
    };

    const onPointerMove = (event) => {
        if (!isDragging) {
            return;
        }
        const delta = event.clientX - startX;
        container.scrollLeft = initialScrollLeft - delta;
        event.preventDefault();
    };

    const onPointerUp = (event) => {
        if (!isDragging) {
            return;
        }
        isDragging = false;
        container.classList.remove('is-dragging');
        container.releasePointerCapture?.(event.pointerId);
    };

    container.addEventListener('pointerdown', onPointerDown);
    container.addEventListener('pointermove', onPointerMove);
    container.addEventListener('pointerup', onPointerUp);
    container.addEventListener('pointerleave', onPointerUp);
    container.addEventListener('pointercancel', onPointerUp);
}

/**
 * Initialize drag-scroll on common scrollable containers
 * @param {Array<string>} selectors - CSS selectors for containers
 */
export function initializeScrollableContainers(selectors = []) {
    const defaultSelectors = [
        '#suites-table',
        '#executions-table',
        '#coverage-table',
        '#recent-executions',
        '.vault-activity-panel .panel-body'
    ];

    const allSelectors = selectors.length > 0 ? selectors : defaultSelectors;

    allSelectors.forEach((selector) => {
        document.querySelectorAll(selector).forEach((container) => {
            enableDragScroll(container);
        });
    });
}

/**
 * Update sidebar accessibility attributes
 * @param {HTMLElement} sidebarToggleButton
 * @param {HTMLElement} sidebarElement
 * @param {boolean} isExpanded
 */
export function updateSidebarAccessibility(sidebarToggleButton, sidebarElement, isExpanded) {
    if (sidebarToggleButton) {
        sidebarToggleButton.setAttribute('aria-expanded', isExpanded ? 'true' : 'false');
        sidebarToggleButton.setAttribute('aria-label', isExpanded ? 'Hide navigation' : 'Show navigation');
    }

    if (sidebarElement) {
        sidebarElement.setAttribute('aria-hidden', isExpanded ? 'false' : 'true');
    }
}

/**
 * Handle responsive sidebar behavior on window resize
 * @param {number} mobileBreakpoint - Breakpoint width in pixels
 * @returns {{isMobile: boolean, shouldCollapse: boolean}}
 */
export function handleSidebarResize(mobileBreakpoint = MOBILE_BREAKPOINT) {
    const isMobile = window.innerWidth <= mobileBreakpoint;
    return {
        isMobile,
        shouldCollapse: !isMobile
    };
}

/**
 * Toggle sidebar open/closed state
 * @param {HTMLElement} sidebarToggleButton
 * @param {HTMLElement} sidebarElement
 * @param {HTMLElement} sidebarCloseButton
 * @param {number} mobileBreakpoint
 */
export function toggleSidebar(sidebarToggleButton, sidebarElement, sidebarCloseButton, mobileBreakpoint = MOBILE_BREAKPOINT) {
    const isMobile = window.innerWidth <= mobileBreakpoint;

    if (isMobile) {
        if (document.body.classList.contains('sidebar-open')) {
            closeMobileSidebar(sidebarToggleButton);
        } else {
            document.body.classList.add('sidebar-open');
            document.body.classList.remove('sidebar-collapsed');
            updateSidebarAccessibility(sidebarToggleButton, sidebarElement, true);
            focusElement(sidebarCloseButton);
        }
        return;
    }

    const isCollapsed = document.body.classList.toggle('sidebar-collapsed');
    updateSidebarAccessibility(sidebarToggleButton, sidebarElement, !isCollapsed);
}

/**
 * Close mobile sidebar
 * @param {HTMLElement} sidebarToggleButton
 */
export function closeMobileSidebar(sidebarToggleButton) {
    if (!document.body.classList.contains('sidebar-open')) {
        return;
    }

    document.body.classList.remove('sidebar-open');
    focusElement(sidebarToggleButton);
}

/**
 * Query selector with error handling
 * @param {string} selector
 * @param {HTMLElement} context - Root element to query from
 * @returns {HTMLElement|null}
 */
export function safeQuerySelector(selector, context = document) {
    try {
        return context.querySelector(selector);
    } catch (error) {
        console.warn(`Invalid selector: ${selector}`, error);
        return null;
    }
}

/**
 * Query selector all with error handling
 * @param {string} selector
 * @param {HTMLElement} context - Root element to query from
 * @returns {Array<HTMLElement>}
 */
export function safeQuerySelectorAll(selector, context = document) {
    try {
        return Array.from(context.querySelectorAll(selector));
    } catch (error) {
        console.warn(`Invalid selector: ${selector}`, error);
        return [];
    }
}

/**
 * Add event listener with cleanup tracking
 * @param {HTMLElement} element
 * @param {string} event
 * @param {Function} handler
 * @param {Object} options
 * @returns {Function} Cleanup function
 */
export function addManagedEventListener(element, event, handler, options = {}) {
    if (!element || !event || typeof handler !== 'function') {
        return () => {};
    }

    element.addEventListener(event, handler, options);

    return () => {
        element.removeEventListener(event, handler, options);
    };
}

/**
 * Debounce function calls
 * @param {Function} func
 * @param {number} wait - Milliseconds to wait
 * @returns {Function}
 */
export function debounce(func, wait = 300) {
    let timeout;
    return function executedFunction(...args) {
        const later = () => {
            clearTimeout(timeout);
            func(...args);
        };
        clearTimeout(timeout);
        timeout = setTimeout(later, wait);
    };
}

/**
 * Throttle function calls
 * @param {Function} func
 * @param {number} limit - Milliseconds between calls
 * @returns {Function}
 */
export function throttle(func, limit = 100) {
    let inThrottle;
    return function executedFunction(...args) {
        if (!inThrottle) {
            func(...args);
            inThrottle = true;
            setTimeout(() => inThrottle = false, limit);
        }
    };
}

/**
 * Scroll element into view with smooth behavior
 * @param {HTMLElement} element
 * @param {Object} options
 */
export function smoothScrollIntoView(element, options = {}) {
    if (!element || typeof element.scrollIntoView !== 'function') {
        return;
    }

    const defaultOptions = {
        behavior: 'smooth',
        block: 'nearest',
        inline: 'nearest'
    };

    element.scrollIntoView({ ...defaultOptions, ...options });
}

/**
 * Check if element is in viewport
 * @param {HTMLElement} element
 * @returns {boolean}
 */
export function isInViewport(element) {
    if (!element) return false;

    const rect = element.getBoundingClientRect();
    return (
        rect.top >= 0 &&
        rect.left >= 0 &&
        rect.bottom <= (window.innerHeight || document.documentElement.clientHeight) &&
        rect.right <= (window.innerWidth || document.documentElement.clientWidth)
    );
}

/**
 * Get element's offset from document top
 * @param {HTMLElement} element
 * @returns {{top: number, left: number}}
 */
export function getElementOffset(element) {
    if (!element) return { top: 0, left: 0 };

    const rect = element.getBoundingClientRect();
    const scrollLeft = window.pageXOffset || document.documentElement.scrollLeft;
    const scrollTop = window.pageYOffset || document.documentElement.scrollTop;

    return {
        top: rect.top + scrollTop,
        left: rect.left + scrollLeft
    };
}

/**
 * Create HTML element from string
 * @param {string} htmlString
 * @returns {HTMLElement}
 */
export function createElementFromHTML(htmlString) {
    const template = document.createElement('template');
    template.innerHTML = htmlString.trim();
    return template.content.firstChild;
}

/**
 * Check if user prefers reduced motion
 * @returns {boolean}
 */
export function prefersReducedMotion() {
    return window.matchMedia('(prefers-reduced-motion: reduce)').matches;
}

/**
 * Get computed style property value
 * @param {HTMLElement} element
 * @param {string} property
 * @returns {string}
 */
export function getComputedStyleValue(element, property) {
    if (!element) return '';
    return window.getComputedStyle(element).getPropertyValue(property);
}

/**
 * Copy text to clipboard
 * @param {string} text
 * @returns {Promise<boolean>}
 */
export async function copyToClipboard(text) {
    try {
        if (navigator.clipboard && navigator.clipboard.writeText) {
            await navigator.clipboard.writeText(text);
            return true;
        }

        // Fallback for older browsers
        const textArea = document.createElement('textarea');
        textArea.value = text;
        textArea.style.position = 'fixed';
        textArea.style.opacity = '0';
        document.body.appendChild(textArea);
        textArea.select();
        const success = document.execCommand('copy');
        document.body.removeChild(textArea);
        return success;
    } catch (error) {
        console.error('Failed to copy to clipboard:', error);
        return false;
    }
}
