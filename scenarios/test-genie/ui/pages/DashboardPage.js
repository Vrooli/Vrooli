/**
 * DashboardPage - Dashboard View Logic
 * Handles dashboard data loading, metric calculations, and recent executions display
 */

import { eventBus, EVENT_TYPES } from '../core/EventBus.js';
import { stateManager } from '../core/StateManager.js';
import { apiClient } from '../core/ApiClient.js';
import { notificationManager } from '../managers/NotificationManager.js';
import {
    normalizeCollection,
    normalizeId,
    formatTimestamp,
    calculateDuration,
    getStatusClass,
    escapeHtml
} from '../utils/index.js';
import { enableDragScroll, refreshIcons } from '../utils/domHelpers.js';

/**
 * DashboardPage class - Manages dashboard functionality
 */
export class DashboardPage {
    constructor(
        eventBusInstance = eventBus,
        stateManagerInstance = stateManager,
        apiClientInstance = apiClient,
        notificationManagerInstance = notificationManager
    ) {
        this.eventBus = eventBusInstance;
        this.stateManager = stateManagerInstance;
        this.apiClient = apiClientInstance;
        this.notificationManager = notificationManagerInstance;

        // DOM references
        this.headerStatsElements = {
            activeSuites: null,
            runningTests: null,
            avgCoverage: null
        };

        this.dashboardMetricsElements = {
            totalSuites: null,
            testsGenerated: null,
            avgCoverage: null,
            failedTests: null
        };

        this.recentExecutionsContainer = null;

        // Debug mode
        this.debug = false;

        this.initialize();
    }

    /**
     * Initialize dashboard page
     */
    initialize() {
        this.setupDOMReferences();
        this.setupEventListeners();

        if (this.debug) {
            console.log('[DashboardPage] Dashboard page initialized');
        }
    }

    /**
     * Setup DOM element references
     * @private
     */
    setupDOMReferences() {
        // Header stats
        this.headerStatsElements.activeSuites = document.getElementById('active-suites');
        this.headerStatsElements.runningTests = document.getElementById('running-tests');
        this.headerStatsElements.avgCoverage = document.getElementById('avg-coverage');

        // Dashboard metrics
        this.dashboardMetricsElements.totalSuites = document.getElementById('metric-total-suites');
        this.dashboardMetricsElements.testsGenerated = document.getElementById('metric-tests-generated');
        this.dashboardMetricsElements.avgCoverage = document.getElementById('metric-avg-coverage');
        this.dashboardMetricsElements.failedTests = document.getElementById('metric-failed-tests');

        // Recent executions container
        this.recentExecutionsContainer = document.getElementById('recent-executions');
    }

    /**
     * Setup event listeners
     * @private
     */
    setupEventListeners() {
        // Listen for page load requests
        this.eventBus.on(EVENT_TYPES.PAGE_LOAD_REQUESTED, (event) => {
            if (event.data.page === 'dashboard') {
                this.load();
            }
        });

        // Listen for data updates
        this.eventBus.on(EVENT_TYPES.DATA_LOADED, (event) => {
            const { collection } = event.data;
            if (collection === 'suites' || collection === 'executions') {
                // Recalculate metrics when data changes
                const activePage = this.stateManager.get('activePage');
                if (activePage === 'dashboard') {
                    this.calculateAndUpdateMetrics();
                }
            }
        });

        // Listen for execution updates (real-time)
        this.eventBus.on(EVENT_TYPES.EXECUTION_UPDATED, () => {
            const activePage = this.stateManager.get('activePage');
            if (activePage === 'dashboard') {
                this.loadRecentExecutions();
            }
        });

        // Listen for execution completion
        this.eventBus.on(EVENT_TYPES.EXECUTION_COMPLETED, () => {
            const activePage = this.stateManager.get('activePage');
            if (activePage === 'dashboard') {
                this.loadRecentExecutions();
            }
        });
    }

    /**
     * Load dashboard data
     */
    async load() {
        if (this.debug) {
            console.log('[DashboardPage] Loading dashboard data');
        }

        try {
            // Set loading state
            this.stateManager.setLoading('dashboard', true);

            // Load system metrics, suites, and executions in parallel
            const [systemMetrics, testSuitesData, executionsData] = await Promise.all([
                this.apiClient.getSystemMetrics().catch(() => null),
                this.apiClient.getTestSuites(),
                this.apiClient.getTestExecutions()
            ]);

            // Normalize collections
            const suites = normalizeCollection(testSuitesData, 'test_suites');
            const executions = normalizeCollection(executionsData, 'executions');

            // Update state
            this.stateManager.setData('suites', suites);
            this.stateManager.setData('executions', executions);

            if (systemMetrics) {
                this.stateManager.set('data.systemMetrics', systemMetrics);
            }

            // Calculate and update metrics
            this.calculateAndUpdateMetrics();

            // Load recent executions
            await this.loadRecentExecutions();

            // Emit success event
            this.eventBus.emit(EVENT_TYPES.PAGE_LOADED, { page: 'dashboard' });

        } catch (error) {
            console.error('[DashboardPage] Failed to load dashboard data:', error);
            this.notificationManager.showError('Failed to load dashboard data');
        } finally {
            this.stateManager.setLoading('dashboard', false);
        }
    }

    /**
     * Calculate and update all dashboard metrics
     * @private
     */
    calculateAndUpdateMetrics() {
        const suites = this.stateManager.get('data.suites') || [];
        const executions = this.stateManager.get('data.executions') || [];

        // Calculate header stats
        const activeSuites = suites.filter(s =>
            (s.status || '').toLowerCase() === 'active'
        ).length;

        const runningExecutions = executions.filter(e =>
            (e.status || '').toLowerCase() === 'running'
        ).length;

        let avgCoverage = 0;
        if (suites.length > 0) {
            const totalCoverage = suites.reduce((sum, suite) => {
                const coverage = suite.coverage_metrics?.code_coverage ?? suite.coverage ?? 0;
                return sum + Number(coverage);
            }, 0);
            avgCoverage = Math.round(totalCoverage / suites.length);
        }

        // Update header stats
        this.updateHeaderStats({
            activeSuites,
            runningTests: runningExecutions,
            avgCoverage
        });

        // Calculate dashboard metrics
        const totalSuites = suites.length;

        const totalTestCases = suites.reduce((sum, suite) => {
            if (typeof suite.total_tests === 'number') {
                return sum + suite.total_tests;
            }
            if (Array.isArray(suite.test_cases)) {
                return sum + suite.test_cases.length;
            }
            return sum;
        }, 0);

        const failedTests = executions.filter(e =>
            (e.status || '').toLowerCase() === 'failed'
        ).length;

        // Update dashboard metrics
        this.updateDashboardMetrics({
            totalSuites,
            testsGenerated: totalTestCases,
            avgCoverage,
            failedTests
        });
    }

    /**
     * Update header stats display
     * @param {Object} stats - Stats object
     * @param {number} stats.activeSuites - Number of active suites
     * @param {number} stats.runningTests - Number of running tests
     * @param {number} stats.avgCoverage - Average coverage percentage
     */
    updateHeaderStats(stats) {
        if (this.headerStatsElements.activeSuites) {
            this.headerStatsElements.activeSuites.textContent = stats.activeSuites;
        }

        if (this.headerStatsElements.runningTests) {
            this.headerStatsElements.runningTests.textContent = stats.runningTests;
        }

        if (this.headerStatsElements.avgCoverage) {
            this.headerStatsElements.avgCoverage.textContent = `${stats.avgCoverage}%`;
        }
    }

    /**
     * Update dashboard metrics display
     * @param {Object} metrics - Metrics object
     * @param {number} metrics.totalSuites - Total number of suites
     * @param {number} metrics.testsGenerated - Total tests generated
     * @param {number} metrics.avgCoverage - Average coverage percentage
     * @param {number} metrics.failedTests - Number of failed tests
     */
    updateDashboardMetrics(metrics) {
        if (this.dashboardMetricsElements.totalSuites) {
            this.dashboardMetricsElements.totalSuites.textContent = metrics.totalSuites;
        }

        if (this.dashboardMetricsElements.testsGenerated) {
            this.dashboardMetricsElements.testsGenerated.textContent = metrics.testsGenerated;
        }

        if (this.dashboardMetricsElements.avgCoverage) {
            this.dashboardMetricsElements.avgCoverage.textContent = `${metrics.avgCoverage}%`;
        }

        if (this.dashboardMetricsElements.failedTests) {
            this.dashboardMetricsElements.failedTests.textContent = metrics.failedTests;
        }
    }

    /**
     * Load recent executions
     */
    async loadRecentExecutions() {
        if (!this.recentExecutionsContainer) {
            return;
        }

        try {
            const executionsResponse = await this.apiClient.getTestExecutions({ limit: 5, sort: 'desc' });
            const executions = normalizeCollection(executionsResponse, 'executions');

            if (executions && executions.length > 0) {
                // Transform API data to display format
                const formattedExecutions = executions.map(exec => {
                    const suiteId = exec.suite_id;
                    const suiteName = this.lookupSuiteName(suiteId) || exec.suite_name || 'Unknown Suite';

                    return {
                        id: exec.id,
                        suiteName,
                        status: exec.status,
                        statusClass: getStatusClass(exec.status),
                        duration: calculateDuration(exec.start_time, exec.end_time),
                        passed: exec.passed_tests || 0,
                        failed: exec.failed_tests || 0,
                        timestamp: exec.start_time
                    };
                });

                // Render table
                this.recentExecutionsContainer.innerHTML = this.renderExecutionsTable(formattedExecutions);
                this.recentExecutionsContainer.scrollLeft = 0;

                // Enable drag scrolling
                enableDragScroll(this.recentExecutionsContainer);

                // Bind actions
                this.bindExecutionsTableActions(this.recentExecutionsContainer);

                // Refresh icons
                refreshIcons();
            } else {
                this.recentExecutionsContainer.innerHTML = '<p style="color: var(--text-muted); text-align: center; padding: 2rem;">No recent executions found</p>';
                this.recentExecutionsContainer.scrollLeft = 0;
                enableDragScroll(this.recentExecutionsContainer);
            }
        } catch (error) {
            console.error('[DashboardPage] Failed to load recent executions:', error);
            this.recentExecutionsContainer.innerHTML = '<p style="color: var(--accent-error); text-align: center; padding: 2rem;">Failed to load recent executions</p>';
            this.recentExecutionsContainer.scrollLeft = 0;
            enableDragScroll(this.recentExecutionsContainer);
        } finally {
            this.recentExecutionsContainer.classList.remove('loading');
        }
    }

    /**
     * Lookup suite name by suite ID
     * @param {string} suiteId - Suite ID
     * @returns {string|null} Suite name or null
     * @private
     */
    lookupSuiteName(suiteId) {
        if (!suiteId) {
            return null;
        }

        const suites = this.stateManager.get('data.suites') || [];
        const match = suites.find(suite => suite.id === suiteId);
        return match ? (match.scenario_name || match.name) : null;
    }

    /**
     * Render executions table HTML
     * @param {Array<Object>} executions - Formatted executions
     * @returns {string} HTML string
     * @private
     */
    renderExecutionsTable(executions) {
        if (!executions || executions.length === 0) {
            return '<p style="color: var(--text-muted); text-align: center; padding: 2rem;">No recent executions found</p>';
        }

        return `
            <table class="table executions-table" data-selectable="false">
                <thead>
                    <tr>
                        <th>Suite Name</th>
                        <th>Status</th>
                        <th>Duration</th>
                        <th>Passed</th>
                        <th>Failed</th>
                        <th>Time</th>
                        <th>Actions</th>
                    </tr>
                </thead>
                <tbody>
                    ${executions.map(exec => {
                        const executionId = normalizeId(exec.id);
                        const failedClass = exec.failed > 0 ? 'text-error' : 'text-muted';
                        const durationLabel = Number.isFinite(exec.duration) ? `${exec.duration}s` : '—';
                        const statusClass = exec.statusClass || getStatusClass(exec.status);

                        return `
                        <tr>
                            <td class="cell-scenario"><strong>${escapeHtml(exec.suiteName)}</strong></td>
                            <td class="cell-status"><span class="status ${statusClass}">${escapeHtml(exec.status)}</span></td>
                            <td>${durationLabel}</td>
                            <td class="text-success">${exec.passed}</td>
                            <td class="${failedClass}">${exec.failed}</td>
                            <td class="cell-created">${formatTimestamp(exec.timestamp)}</td>
                            <td class="cell-actions">
                                <button class="btn icon-btn" type="button" data-action="view-execution" data-execution-id="${executionId}" aria-label="View execution details">
                                    <i data-lucide="bar-chart-3"></i>
                                </button>
                            </td>
                        </tr>
                        `;
                    }).join('')}
                </tbody>
            </table>
        `;
    }

    /**
     * Bind actions to executions table
     * @param {HTMLElement} container - Container element
     * @private
     */
    bindExecutionsTableActions(container) {
        const table = container.querySelector('.executions-table');
        if (!table || table.dataset.bound === 'true') {
            return;
        }

        table.addEventListener('click', (event) => {
            const button = event.target.closest('[data-action]');
            if (!button) {
                return;
            }

            if (button.dataset.action === 'view-execution' && button.dataset.executionId) {
                this.viewExecution(button.dataset.executionId, button);
            }
        });

        table.dataset.bound = 'true';
    }

    /**
     * View execution details
     * @param {string} executionId - Execution ID
     * @param {HTMLElement} triggerElement - Trigger element (optional)
     */
    viewExecution(executionId, triggerElement = null) {
        if (this.debug) {
            console.log('[DashboardPage] Viewing execution:', executionId);
        }

        // Emit event for execution detail view
        this.eventBus.emit(EVENT_TYPES.EXECUTION_VIEW_REQUESTED, {
            executionId,
            trigger: triggerElement
        });
    }

    /**
     * Refresh dashboard data
     */
    async refresh() {
        await this.load();
        this.notificationManager.showSuccess('Dashboard refreshed!');
    }

    /**
     * Set debug mode
     * @param {boolean} enabled
     */
    setDebug(enabled) {
        this.debug = enabled;
        console.log(`[DashboardPage] Debug mode ${enabled ? 'enabled' : 'disabled'}`);
    }

    /**
     * Get debug information
     * @returns {Object}
     */
    getDebugInfo() {
        return {
            hasHeaderStatsElements: Object.values(this.headerStatsElements).every(el => el !== null),
            hasDashboardMetricsElements: Object.values(this.dashboardMetricsElements).every(el => el !== null),
            hasRecentExecutionsContainer: this.recentExecutionsContainer !== null,
            currentMetrics: {
                activeSuites: this.headerStatsElements.activeSuites?.textContent,
                totalSuites: this.dashboardMetricsElements.totalSuites?.textContent
            }
        };
    }
}

// Export singleton instance
export const dashboardPage = new DashboardPage();

// Export default for convenience
export default dashboardPage;
