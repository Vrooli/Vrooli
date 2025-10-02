/**
 * NavigationManager - Page Navigation & Routing
 * Handles page transitions, URL updates, and browser history
 */

import { eventBus, EVENT_TYPES } from '../core/EventBus.js';
import { stateManager } from '../core/StateManager.js';
import { MOBILE_BREAKPOINT } from '../utils/constants.js';

/**
 * NavigationManager class - Manages application navigation and routing
 */
export class NavigationManager {
    constructor(eventBusInstance = eventBus, stateManagerInstance = stateManager) {
        this.eventBus = eventBusInstance;
        this.stateManager = stateManagerInstance;
        this.mobileBreakpoint = MOBILE_BREAKPOINT;

        // Valid pages
        this.validPages = ['dashboard', 'suites', 'executions', 'coverage', 'vault', 'reports', 'settings'];

        // Debug mode
        this.debug = false;

        this.initialize();
    }

    /**
     * Initialize navigation system
     */
    initialize() {
        this.setupBrowserNavigation();
        this.setupNavigationClicks();
        this.setupKeyboardShortcuts();
        this.navigateToInitialPage();

        if (this.debug) {
            console.log('[NavigationManager] Navigation system initialized');
        }
    }

    /**
     * Setup browser back/forward navigation
     * @private
     */
    setupBrowserNavigation() {
        window.addEventListener('popstate', (e) => {
            const page = e.state?.page || 'dashboard';
            this.navigateTo(page, false);
        });
    }

    /**
     * Setup navigation item clicks
     * @private
     */
    setupNavigationClicks() {
        document.addEventListener('click', (e) => {
            const navItem = e.target.closest('.nav-item');
            if (navItem) {
                const page = navItem.getAttribute('data-page');
                if (page) {
                    this.navigateTo(page);

                    // Close mobile sidebar after navigation
                    if (window.innerWidth <= this.mobileBreakpoint) {
                        this.eventBus.emit(EVENT_TYPES.UI_SIDEBAR_CLOSE);
                    }
                }
            }
        });
    }

    /**
     * Setup keyboard shortcuts for navigation
     * @private
     */
    setupKeyboardShortcuts() {
        document.addEventListener('keydown', (e) => {
            // Only handle shortcuts with Ctrl/Cmd modifier
            if (!e.ctrlKey && !e.metaKey) {
                return;
            }

            let targetPage = null;

            switch (e.key) {
                case '1':
                    targetPage = 'dashboard';
                    break;
                case '2':
                    targetPage = 'suites';
                    break;
                case '3':
                    targetPage = 'executions';
                    break;
                case '4':
                    targetPage = 'coverage';
                    break;
                case '5':
                    targetPage = 'vault';
                    break;
                case '6':
                    targetPage = 'reports';
                    break;
                case '7':
                    targetPage = 'settings';
                    break;
            }

            if (targetPage) {
                e.preventDefault();
                this.navigateTo(targetPage);
            }
        });
    }

    /**
     * Navigate to initial page based on URL hash
     * @private
     */
    navigateToInitialPage() {
        const initialPage = window.location.hash.replace('#', '') || 'dashboard';
        this.navigateTo(initialPage, false);
    }

    /**
     * Navigate to a specific page
     * @param {string} page - Page name
     * @param {boolean} pushState - Whether to push state to browser history
     */
    navigateTo(page, pushState = true) {
        // Validate page
        let resolvedPage = page;
        if (!this.validPages.includes(page) || !document.getElementById(page)) {
            console.warn(`[NavigationManager] Invalid page '${page}', defaulting to dashboard`);
            resolvedPage = 'dashboard';
        }

        if (this.debug) {
            console.log(`[NavigationManager] Navigating to: ${resolvedPage}`);
        }

        // Get current page before navigation
        const previousPage = this.stateManager.get('activePage');

        // Update active nav item
        this.updateActiveNavItem(resolvedPage);

        // Update active page content
        this.updateActivePage(resolvedPage);

        // Update URL
        if (pushState) {
            history.pushState({ page: resolvedPage }, '', `#${resolvedPage}`);
        }

        // Update state
        this.stateManager.setActivePage(resolvedPage);

        // Close detail panels when navigating away from their pages
        if (resolvedPage !== 'suites') {
            this.eventBus.emit(EVENT_TYPES.UI_SUITE_DETAIL_CLOSE);
        }

        if (resolvedPage !== 'executions') {
            this.eventBus.emit(EVENT_TYPES.UI_EXECUTION_DETAIL_CLOSE);
        }

        // Emit page changed event with previous and current page
        this.eventBus.emit(EVENT_TYPES.PAGE_CHANGED, {
            previous: previousPage,
            current: resolvedPage
        });

        // Request page data load
        this.eventBus.emit(EVENT_TYPES.PAGE_LOAD_REQUESTED, { page: resolvedPage });
    }

    /**
     * Update active nav item styling
     * @private
     */
    updateActiveNavItem(page) {
        // Remove active class from all nav items
        document.querySelectorAll('.nav-item').forEach(item => {
            item.classList.remove('active');
        });

        // Add active class to current page nav item
        const activeNavItem = document.querySelector(`.nav-item[data-page="${page}"]`);
        if (activeNavItem) {
            activeNavItem.classList.add('active');
        }
    }

    /**
     * Update active page content visibility
     * @private
     */
    updateActivePage(page) {
        // Hide all pages
        document.querySelectorAll('.page').forEach(p => {
            p.classList.remove('active');
        });

        // Show current page
        const activePage = document.getElementById(page);
        if (activePage) {
            activePage.classList.add('active');
        }
    }

    /**
     * Get current active page
     * @returns {string}
     */
    getCurrentPage() {
        return this.stateManager.get('activePage') || 'dashboard';
    }

    /**
     * Check if page is currently active
     * @param {string} page
     * @returns {boolean}
     */
    isPageActive(page) {
        return this.getCurrentPage() === page;
    }

    /**
     * Reload current page data
     */
    reloadCurrentPage() {
        const currentPage = this.getCurrentPage();
        this.eventBus.emit(EVENT_TYPES.PAGE_LOAD_REQUESTED, { page: currentPage });
    }

    /**
     * Get valid pages list
     * @returns {Array<string>}
     */
    getValidPages() {
        return [...this.validPages];
    }

    /**
     * Set debug mode
     * @param {boolean} enabled
     */
    setDebug(enabled) {
        this.debug = enabled;
        console.log(`[NavigationManager] Debug mode ${enabled ? 'enabled' : 'disabled'}`);
    }

    /**
     * Get debug information
     * @returns {Object}
     */
    getDebugInfo() {
        return {
            currentPage: this.getCurrentPage(),
            validPages: this.validPages,
            urlHash: window.location.hash,
            historyState: window.history.state
        };
    }
}

// Export singleton instance
export const navigationManager = new NavigationManager();

// Export default for convenience
export default navigationManager;
