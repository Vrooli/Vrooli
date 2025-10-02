/**
 * SuitesPage - Test Suites Management View
 * Handles loading, rendering, filtering, and executing test suites
 */

import { eventBus, EVENT_TYPES } from '../core/EventBus.js';
import { stateManager } from '../core/StateManager.js';
import { apiClient } from '../core/ApiClient.js';
import { notificationManager } from '../managers/NotificationManager.js';
import { selectionManager } from '../managers/SelectionManager.js';
import { dialogManager } from '../managers/DialogManager.js';
import {
    normalizeCollection,
    normalizeId,
    formatTimestamp,
    getStatusClass,
    formatLabel,
    escapeHtml
} from '../utils/index.js';
import { enableDragScroll, refreshIcons } from '../utils/domHelpers.js';

/**
 * SuitesPage class - Manages test suites page functionality
 */
export class SuitesPage {
    constructor(
        eventBusInstance = eventBus,
        stateManagerInstance = stateManager,
        apiClientInstance = apiClient,
        notificationManagerInstance = notificationManager,
        selectionManagerInstance = selectionManager,
        dialogManagerInstance = dialogManager
    ) {
        this.eventBus = eventBusInstance;
        this.stateManager = stateManagerInstance;
        this.apiClient = apiClientInstance;
        this.notificationManager = notificationManagerInstance;
        this.selectionManager = selectionManagerInstance;
        this.dialogManager = dialogManagerInstance;

        // DOM references
        this.suitesTableContainer = null;

        // Current data
        this.currentSuiteRows = [];
        this.renderedSuiteIds = [];
        this.availableSuiteIds = [];

        // Filters
        this.suiteFilters = {
            search: '',
            status: 'all'
        };

        // Debug mode
        this.debug = true; // Temporarily enabled for troubleshooting

        this.initialize();
    }

    /**
     * Initialize suites page
     */
    initialize() {
        this.setupDOMReferences();
        this.setupEventListeners();

        if (this.debug) {
            console.log('[SuitesPage] Suites page initialized');
        }
    }

    /**
     * Setup DOM element references
     * @private
     */
    setupDOMReferences() {
        this.suitesTableContainer = document.getElementById('suites-table');
    }

    /**
     * Setup event listeners
     * @private
     */
    setupEventListeners() {
        // Listen for page load requests
        this.eventBus.on(EVENT_TYPES.PAGE_LOAD_REQUESTED, (event) => {
            if (event.data.page === 'suites') {
                this.load();
            }
        });

        // Listen for data updates
        this.eventBus.on(EVENT_TYPES.DATA_LOADED, (event) => {
            const { collection } = event.data;
            if (collection === 'suites' || collection === 'scenarios') {
                const activePage = this.stateManager.get('activePage');
                if (activePage === 'suites') {
                    this.renderSuitesTableWithCurrentData();
                }
            }
        });

        // Listen for suite execution completion
        this.eventBus.on(EVENT_TYPES.SUITE_EXECUTED, () => {
            const activePage = this.stateManager.get('activePage');
            if (activePage === 'suites') {
                this.load(); // Refresh to show updated status
            }
        });

        // Listen for filter changes
        this.eventBus.on(EVENT_TYPES.FILTER_CHANGED, (event) => {
            if (event.data.collection === 'suites') {
                this.suiteFilters = {
                    ...this.suiteFilters,
                    ...event.data.filters
                };
                this.renderSuitesTableWithCurrentData();
            }
        });
    }

    /**
     * Load test suites data
     */
    async load() {
        if (!this.suitesTableContainer) {
            return;
        }

        if (this.debug) {
            console.log('[SuitesPage] Loading test suites');
        }

        this.suitesTableContainer.innerHTML = '<div class="loading"><div class="spinner"></div>Loading test suites...</div>';

        try {
            // Set loading state
            this.stateManager.setLoading('suites', true);

            // Load scenarios and suites in parallel
            const [scenariosResponse, suitesResponse] = await Promise.all([
                this.apiClient.getScenarios(),
                this.apiClient.getTestSuites()
            ]);

            // Normalize collections
            const scenarios = normalizeCollection(scenariosResponse, 'scenarios');
            const suites = normalizeCollection(suitesResponse, 'test_suites');

            if (this.debug) {
                console.log('[SuitesPage] Scenarios:', scenarios);
                console.log('[SuitesPage] Suites:', suites);
            }

            // Update state
            this.stateManager.setData('scenarios', scenarios);
            this.stateManager.setData('suites', suites);

            // Group suites by scenario
            const suitesByScenario = this.groupSuitesByScenario(suites);

            if (this.debug) {
                console.log('[SuitesPage] Suites by scenario:', suitesByScenario);
            }

            // Build scenario rows
            const scenarioRows = this.buildScenarioRows(scenarios, suitesByScenario);

            if (this.debug) {
                console.log('[SuitesPage] Built scenario rows:', scenarioRows);
            }

            // Store rows and render
            this.currentSuiteRows = scenarioRows;
            this.availableSuiteIds = scenarioRows
                .map(row => row.latestSuiteId)
                .filter(Boolean);

            // Store scenario rows in state for dialog access
            this.stateManager.setData('scenarioRows', scenarioRows);

            // Prune invalid selections
            this.selectionManager.pruneSuiteSelection(this.availableSuiteIds);

            // Render table
            this.renderSuitesTableWithCurrentData();

            // Emit success event
            this.eventBus.emit(EVENT_TYPES.PAGE_LOADED, { page: 'suites' });

        } catch (error) {
            console.error('[SuitesPage] Failed to load test suites:', error);
            this.suitesTableContainer.innerHTML = '<p style="color: var(--accent-error); text-align: center; padding: 2rem;">Failed to load test suites</p>';
            this.currentSuiteRows = [];
            this.renderedSuiteIds = [];
            this.availableSuiteIds = [];
            this.selectionManager.updateSuiteSelectionUI();
        } finally {
            this.stateManager.setLoading('suites', false);
        }
    }

    /**
     * Group suites by scenario name
     * @param {Array} suites - Array of test suites
     * @returns {Map} Map of scenario name to array of suites
     * @private
     */
    groupSuitesByScenario(suites) {
        const suitesByScenario = new Map();

        if (!Array.isArray(suites)) {
            return suitesByScenario;
        }

        suites.forEach((suite) => {
            const scenarioName = suite?.scenario_name || suite?.scenarioName;
            if (!scenarioName) {
                return;
            }

            const normalized = scenarioName.trim();
            if (!suitesByScenario.has(normalized)) {
                suitesByScenario.set(normalized, []);
            }
            suitesByScenario.get(normalized).push(suite);
        });

        return suitesByScenario;
    }

    /**
     * Build scenario rows from scenarios and suites
     * @param {Array} scenarios - Array of scenarios
     * @param {Map} suitesByScenario - Map of scenario name to suites
     * @returns {Array} Array of scenario row objects
     * @private
     */
    buildScenarioRows(scenarios, suitesByScenario) {
        const scenarioRows = [];
        const seenScenarios = new Set();

        // Helper function to build a row from a scenario
        const buildRow = (scenario, attachedSuites = []) => {
            const rawName = scenario?.name || scenario?.scenario_name;
            if (!rawName) {
                return null;
            }

            const scenarioName = String(rawName).trim();
            if (!scenarioName) {
                return null;
            }

            const suiteCount = Number(scenario?.suite_count ?? attachedSuites.length ?? 0) || 0;
            const suiteArray = attachedSuites.length ? attachedSuites : (suitesByScenario.get(scenarioName) || []);

            // Calculate test case count
            const testCaseCount = Number(scenario?.test_case_count ?? 0) || suiteArray.reduce((sum, suite) => {
                if (Array.isArray(suite?.test_cases) && suite.test_cases.length) {
                    return sum + suite.test_cases.length;
                }
                const total = Number(suite?.total_tests);
                if (Number.isFinite(total) && total > 0) {
                    return sum + total;
                }
                return sum;
            }, 0);

            // Collect test types/phases
            const typeSet = new Set(Array.isArray(scenario?.suite_types) ? scenario.suite_types : []);
            if (!typeSet.size && suiteArray.length) {
                suiteArray.forEach((suite) => {
                    const rawType = Array.isArray(suite?.test_types)
                        ? suite.test_types
                        : (suite?.suite_type ? String(suite.suite_type).split(',') : []);
                    rawType.forEach((type) => {
                        const trimmed = String(type || '').trim().toLowerCase();
                        if (trimmed) {
                            typeSet.add(trimmed);
                        }
                    });
                });
            }

            // Find latest suite ID
            const latestSuiteId = this.findLatestSuiteId(scenario, suiteArray);

            // Find latest generated timestamp
            const latestGeneratedAt = scenario?.latest_suite_generated_at
                || (latestSuiteId
                    ? (suiteArray.find((suite) => normalizeId(suite?.id) === latestSuiteId)?.generated_at
                        || suiteArray.find((suite) => normalizeId(suite?.id) === latestSuiteId)?.created_at)
                    : null);

            // Find latest status
            const latestStatus = scenario?.latest_suite_status
                || (suiteArray.find((suite) => normalizeId(suite?.id) === latestSuiteId)?.status)
                || (suiteCount > 0 ? 'unknown' : 'missing');

            // Find latest coverage
            const latestCoverage = this.findLatestCoverage(scenario, suiteArray, latestSuiteId);

            // Build row object
            const phases = Array.from(typeSet).map((type) => formatLabel(type));
            const statusClass = getStatusClass(latestStatus);

            return {
                scenarioName,
                phases,
                testsCount: testCaseCount,
                coverage: latestCoverage,
                status: latestStatus,
                statusClass,
                createdAt: latestGeneratedAt || null,
                latestSuiteId: normalizeId(latestSuiteId),
                suiteCount,
                hasTests: Boolean(scenario?.has_tests || suiteCount > 0),
                hasTestDirectory: Boolean(scenario?.has_test_directory),
                isMissing: !(scenario?.has_tests || suiteCount > 0)
            };
        };

        // Process explicit scenarios
        if (Array.isArray(scenarios) && scenarios.length) {
            scenarios.forEach((scenario) => {
                const row = buildRow(scenario);
                if (row) {
                    scenarioRows.push(row);
                    seenScenarios.add(row.scenarioName);
                }
            });
        }

        // Process suites without explicit scenarios
        suitesByScenario.forEach((suiteArray, scenarioName) => {
            if (!seenScenarios.has(scenarioName)) {
                const fallback = {
                    name: scenarioName,
                    has_tests: suiteArray.length > 0,
                    suite_count: suiteArray.length,
                    test_case_count: suiteArray.reduce((sum, suite) => {
                        if (Array.isArray(suite?.test_cases)) {
                            return sum + suite.test_cases.length;
                        }
                        const total = Number(suite?.total_tests);
                        if (Number.isFinite(total) && total > 0) {
                            return sum + total;
                        }
                        return sum;
                    }, 0)
                };
                const row = buildRow(fallback, suiteArray);
                if (row) {
                    scenarioRows.push(row);
                }
            }
        });

        // Sort by scenario name
        scenarioRows.sort((a, b) => a.scenarioName.localeCompare(b.scenarioName));

        return scenarioRows;
    }

    /**
     * Find latest suite ID from scenario or suite array
     * @param {Object} scenario - Scenario object
     * @param {Array} suiteArray - Array of suites
     * @returns {string} Latest suite ID
     * @private
     */
    findLatestSuiteId(scenario, suiteArray) {
        return scenario?.latest_suite_id || suiteArray.reduce((latest, suite) => {
            const id = normalizeId(suite?.id);
            if (!id) {
                return latest;
            }
            if (!latest) {
                return id;
            }

            const currentTime = suite?.generated_at || suite?.created_at;
            const latestSuite = suiteArray.find((item) => normalizeId(item?.id) === latest);
            const latestTime = latestSuite?.generated_at || latestSuite?.created_at;

            if (!latestTime) {
                return id;
            }
            if (!currentTime) {
                return latest;
            }

            return new Date(currentTime).getTime() > new Date(latestTime).getTime() ? id : latest;
        }, '');
    }

    /**
     * Find latest coverage from scenario or suite array
     * @param {Object} scenario - Scenario object
     * @param {Array} suiteArray - Array of suites
     * @param {string} latestSuiteId - Latest suite ID
     * @returns {number} Coverage percentage
     * @private
     */
    findLatestCoverage(scenario, suiteArray, latestSuiteId) {
        const scenarioCoverage = Number(scenario?.latest_suite_coverage ?? 0);
        if (scenarioCoverage > 0) {
            return scenarioCoverage;
        }

        const suite = suiteArray.find((item) => normalizeId(item?.id) === latestSuiteId) || suiteArray[0];
        if (!suite) {
            return 0;
        }

        const coverage = Number(suite?.coverage_metrics?.code_coverage ?? suite?.coverage ?? 0);
        return Number.isFinite(coverage) ? coverage : 0;
    }

    /**
     * Apply filters to suite rows
     * @param {Array} rows - Array of suite rows
     * @returns {Array} Filtered rows
     * @private
     */
    applySuiteFilters(rows) {
        if (!Array.isArray(rows)) {
            return [];
        }

        let filtered = rows;

        // Apply search filter
        const search = (this.suiteFilters.search || '').trim().toLowerCase();
        if (search) {
            filtered = filtered.filter((row) => {
                const nameMatch = (row.scenarioName || '').toLowerCase().includes(search);
                const phaseMatch = Array.isArray(row.phases) && row.phases.some((phase) => phase.toLowerCase().includes(search));
                const statusMatch = (row.status || '').toLowerCase().includes(search);
                const idMatch = row.latestSuiteId && normalizeId(row.latestSuiteId).includes(search);
                return nameMatch || phaseMatch || statusMatch || idMatch;
            });
        }

        // Apply status filter
        const statusFilter = this.suiteFilters.status;
        if (statusFilter && statusFilter !== 'all') {
            filtered = filtered.filter((row) => (row.statusClass || getStatusClass(row.status)) === statusFilter);
        }

        return filtered;
    }

    /**
     * Render suites table with current data
     * @param {HTMLElement} container - Container element (optional)
     */
    renderSuitesTableWithCurrentData(container = this.suitesTableContainer) {
        if (!container) {
            return;
        }

        // Handle empty data
        if (!Array.isArray(this.currentSuiteRows) || this.currentSuiteRows.length === 0) {
            container.innerHTML = '<p style="color: var(--text-muted); text-align: center; padding: 2rem;">No scenarios found</p>';
            this.renderedSuiteIds = [];
            this.selectionManager.updateSuiteSelectionUI();
            return;
        }

        // Apply filters
        const filtered = this.applySuiteFilters(this.currentSuiteRows);

        // Handle no results after filtering
        if (!filtered.length) {
            container.innerHTML = '<p style="color: var(--text-muted); text-align: center; padding: 2rem;">No suites match the current filters.</p>';
            this.renderedSuiteIds = [];
            this.selectionManager.updateSuiteSelectionUI();
            return;
        }

        // Render table
        container.innerHTML = this.renderSuitesTable(filtered);
        container.scrollLeft = 0;

        // Update rendered IDs
        this.renderedSuiteIds = filtered
            .map((row) => normalizeId(row.latestSuiteId))
            .filter(Boolean);

        // Bind actions
        this.bindSuitesTableActions(container);

        // Refresh icons
        refreshIcons();

        // Update selection UI
        this.selectionManager.updateSuiteSelectionUI();
    }

    /**
     * Render suites table HTML
     * @param {Array} scenarios - Array of scenario rows
     * @returns {string} HTML string
     * @private
     */
    renderSuitesTable(scenarios) {
        if (!scenarios || scenarios.length === 0) {
            return '<p style="color: var(--text-muted); text-align: center; padding: 2rem;">No scenarios found</p>';
        }

        const cardsHtml = this.renderSuitesCards(scenarios);
        const tableHtml = `
            <table class="table suites-table" data-selectable="true">
                <thead>
                    <tr>
                        <th class="cell-select">
                            <input type="checkbox" data-suite-select-all aria-label="Select all suites">
                        </th>
                        <th data-sortable="scenario">Scenario</th>
                        <th data-sortable="coverage">Coverage</th>
                        <th data-sortable="status">Status</th>
                        <th>Actions</th>
                    </tr>
                </thead>
                <tbody>
                    ${scenarios.map((scenario) => this.renderSuiteRow(scenario)).join('')}
                </tbody>
            </table>
        `;

        return `
            <div class="suites-cards-view">${cardsHtml}</div>
            <div class="suites-table-view">${tableHtml}</div>
        `;
    }

    /**
     * Render single suite row
     * @param {Object} scenario - Scenario row object
     * @returns {string} HTML string
     * @private
     */
    renderSuiteRow(scenario) {
        const suiteId = scenario.latestSuiteId ? normalizeId(scenario.latestSuiteId) : '';
        const isChecked = suiteId && this.selectionManager.isSuiteSelected(suiteId);
        const rawCoverage = Number(scenario.coverage ?? 0);
        const coverageValue = Math.max(0, Math.min(100, Number.isFinite(rawCoverage) ? Math.round(rawCoverage) : 0));
        const hasSuite = Boolean(suiteId);
        const isMissing = Boolean(scenario.isMissing);
        const phaseCount = Array.isArray(scenario.phases) ? scenario.phases.length : 0;
        const phasesLabel = phaseCount > 0
            ? `${phaseCount} phase${phaseCount === 1 ? '' : 's'}`
            : (hasSuite ? '—' : 'None yet');
        const statusRaw = isMissing ? 'missing' : (scenario.status || 'unknown');
        const statusLabel = formatLabel(statusRaw);
        const statusClass = scenario.statusClass || getStatusClass(statusRaw);
        const rowClass = isMissing ? 'scenario-row missing-scenario' : 'scenario-row has-suite';

        const coverageCell = hasSuite
            ? `
                <div class="coverage-meter">
                    <div class="progress">
                        <div class="progress-bar" style="width: ${coverageValue}%"></div>
                    </div>
                    <div class="coverage-details">
                        <span class="coverage-percentage">${coverageValue}%</span>
                        <span class="coverage-phases">${phasesLabel}</span>
                    </div>
                </div>
              `
            : '<span class="coverage-empty">—</span>';

        return `
            <tr class="${rowClass}" data-suite-id="${suiteId}" style="cursor: ${hasSuite ? 'pointer' : 'default'}">
                <td class="cell-select">
                    ${hasSuite ? `<input type="checkbox" data-suite-id="${suiteId}" ${isChecked ? 'checked' : ''} aria-label="Select suite for ${escapeHtml(scenario.scenarioName)}">` : ''}
                </td>
                <td class="cell-scenario" data-value="${escapeHtml(scenario.scenarioName)}">
                    <strong>${escapeHtml(scenario.scenarioName)}</strong>
                </td>
                <td class="cell-coverage" data-value="${coverageValue}">${coverageCell}</td>
                <td class="cell-status" data-value="${statusRaw}"><span class="status ${statusClass}">${statusLabel}</span></td>
                <td class="cell-actions">
                    ${hasSuite ? `
                        <button class="btn icon-btn" type="button" data-action="execute" data-suite-id="${suiteId}" aria-label="Execute latest suite for ${escapeHtml(scenario.scenarioName)}">
                            <i data-lucide="play"></i>
                        </button>
                    ` : ''}
                    ${!hasSuite ? `
                        <button class="btn icon-btn highlighted" type="button" data-action="generate" data-scenario="${escapeHtml(scenario.scenarioName)}" aria-label="Generate tests for ${escapeHtml(scenario.scenarioName)}">
                            <i data-lucide="sparkles"></i>
                        </button>
                    ` : ''}
                </td>
            </tr>
        `;
    }

    /**
     * Render suites cards view
     * @param {Array} scenarios - Array of scenario rows
     * @returns {string} HTML string
     * @private
     */
    renderSuitesCards(scenarios) {
        return scenarios.map((scenario) => {
            const suiteId = scenario.latestSuiteId ? normalizeId(scenario.latestSuiteId) : '';
            const isChecked = suiteId && this.selectionManager.isSuiteSelected(suiteId);
            const rawCoverage = Number(scenario.coverage ?? 0);
            const coverageValue = Math.max(0, Math.min(100, Number.isFinite(rawCoverage) ? Math.round(rawCoverage) : 0));
            const hasSuite = Boolean(suiteId);
            const isMissing = Boolean(scenario.isMissing);
            const phaseCount = Array.isArray(scenario.phases) ? scenario.phases.length : 0;
            const phasesLabel = phaseCount > 0
                ? `${phaseCount} phase${phaseCount === 1 ? '' : 's'}`
                : (hasSuite ? '—' : 'None yet');
            const statusRaw = isMissing ? 'missing' : (scenario.status || 'unknown');
            const statusLabel = formatLabel(statusRaw);
            const statusClass = scenario.statusClass || getStatusClass(statusRaw);

            return `
                <div class="suite-card ${isMissing ? 'missing' : ''}" data-suite-id="${suiteId}" style="cursor: ${hasSuite ? 'pointer' : 'default'}">
                    <div class="suite-card-header">
                        <div class="suite-card-title">
                            ${hasSuite ? `<input type="checkbox" class="suite-card-checkbox" data-suite-id="${suiteId}" ${isChecked ? 'checked' : ''} aria-label="Select suite for ${escapeHtml(scenario.scenarioName)}">` : ''}
                            <strong>${escapeHtml(scenario.scenarioName)}</strong>
                        </div>
                        <span class="status ${statusClass}">${statusLabel}</span>
                    </div>
                    ${hasSuite ? `
                        <div class="suite-card-body">
                            <div class="suite-card-metric">
                                <div class="metric-label">Coverage</div>
                                <div class="coverage-meter">
                                    <div class="progress">
                                        <div class="progress-bar" style="width: ${coverageValue}%"></div>
                                    </div>
                                    <div class="coverage-details">
                                        <span class="coverage-percentage">${coverageValue}%</span>
                                        <span class="coverage-phases">${phasesLabel}</span>
                                    </div>
                                </div>
                            </div>
                        </div>
                    ` : ''}
                    <div class="suite-card-actions">
                        ${hasSuite ? `
                            <button class="btn icon-btn" type="button" data-action="execute" data-suite-id="${suiteId}" aria-label="Execute suite">
                                <i data-lucide="play"></i> Execute
                            </button>
                        ` : ''}
                        ${!hasSuite ? `
                            <button class="btn icon-btn highlighted" type="button" data-action="generate" data-scenario="${escapeHtml(scenario.scenarioName)}" aria-label="Generate tests">
                                <i data-lucide="sparkles"></i> Generate Tests
                            </button>
                        ` : ''}
                    </div>
                </div>
            `;
        }).join('');
    }

    /**
     * Bind actions to suites table
     * @param {HTMLElement} container - Container element
     * @private
     */
    bindSuitesTableActions(container) {
        if (!container) {
            return;
        }

        // Prevent duplicate binding
        if (container.dataset.bound === 'true') {
            return;
        }

        // Checkbox selection handler
        container.addEventListener('change', (event) => {
            // Select all handler - check this first
            if (event.target.hasAttribute('data-suite-select-all')) {
                const selectAllCheckbox = event.target;
                if (selectAllCheckbox.checked) {
                    this.selectionManager.selectAllSuites(this.renderedSuiteIds);
                } else {
                    this.selectionManager.clearSuiteSelection();
                }
                return;
            }

            // Individual checkbox handler
            if (event.target.type === 'checkbox' && event.target.hasAttribute('data-suite-id')) {
                const suiteId = event.target.dataset.suiteId;
                if (suiteId) {
                    this.selectionManager.toggleSuiteSelection(suiteId);
                }
            }
        });

        // Action buttons handler
        container.addEventListener('click', (event) => {
            const button = event.target.closest('[data-action]');
            if (!button) {
                return;
            }

            const action = button.dataset.action;

            if (action === 'execute' && button.dataset.suiteId) {
                this.executeSuite(button.dataset.suiteId, button);
            } else if (action === 'generate' && button.dataset.scenario) {
                this.openGenerateDialog(button, button.dataset.scenario);
            }
        });

        // Row click handler (view suite details)
        container.addEventListener('click', (event) => {
            // Ignore clicks on checkboxes and buttons
            if (event.target.closest('input[type="checkbox"], button')) {
                return;
            }

            // Find the closest row element (tr or div.suite-card)
            const row = event.target.closest('tr[data-suite-id], div.suite-card[data-suite-id]');
            if (row && row.dataset.suiteId) {
                const suiteId = row.dataset.suiteId;
                // Only open dialog if there's actually a suite ID (not empty string)
                if (suiteId.trim()) {
                    this.viewSuite(suiteId);
                }
            }
        });

        container.dataset.bound = 'true';
    }

    /**
     * View suite details
     * @param {string} suiteId - Suite ID
     */
    viewSuite(suiteId) {
        if (this.debug) {
            console.log('[SuitesPage] Viewing suite:', suiteId);
        }

        // Open suite detail dialog
        this.dialogManager.openSuiteDetailDialog(suiteId);

        // Load suite details into the dialog
        this.loadSuiteDetails(suiteId);
    }

    /**
     * Load suite details into dialog
     * @param {string} suiteId - Suite ID
     * @private
     */
    async loadSuiteDetails(suiteId) {
        const contentContainer = document.getElementById('suite-detail-content');
        if (!contentContainer) {
            return;
        }

        contentContainer.innerHTML = '<div class="loading"><div class="spinner"></div>Loading suite details...</div>';

        try {
            const suite = await this.apiClient.getTestSuite(suiteId);
            if (!suite) {
                contentContainer.innerHTML = '<div class="suite-detail-empty">Suite not found.</div>';
                return;
            }

            // Update dialog title
            const titleEl = document.getElementById('suite-detail-title');
            const idEl = document.getElementById('suite-detail-id');
            const subtitleEl = document.getElementById('suite-detail-subtitle');

            if (titleEl) {
                titleEl.textContent = suite.scenario_name || 'Test Suite';
            }
            if (idEl) {
                idEl.textContent = `Suite ID: ${suiteId}`;
            }
            if (subtitleEl) {
                const testCount = Array.isArray(suite.test_cases) ? suite.test_cases.length : 0;
                subtitleEl.textContent = `${testCount} test case${testCount === 1 ? '' : 's'}`;
            }

            // Render suite content
            contentContainer.innerHTML = this.renderSuiteDetailContent(suite);
            refreshIcons();

        } catch (error) {
            console.error('[SuitesPage] Failed to load suite details:', error);
            contentContainer.innerHTML = '<div class="suite-detail-empty" style="color: var(--accent-error);">Failed to load suite details.</div>';
        }
    }

    /**
     * Render suite detail content
     * @param {Object} suite - Suite data
     * @returns {string} HTML string
     * @private
     */
    renderSuiteDetailContent(suite) {
        const testCases = Array.isArray(suite.test_cases) ? suite.test_cases : [];

        if (testCases.length === 0) {
            return '<div class="suite-detail-empty">No test cases found in this suite.</div>';
        }

        const testRows = testCases.map((test, index) => {
            const testType = test.type || test.test_type || 'unknown';
            const testName = test.name || test.test_name || `Test ${index + 1}`;
            const description = test.description || '';

            return `
                <tr>
                    <td><strong>${escapeHtml(testName)}</strong></td>
                    <td><span class="badge">${formatLabel(testType)}</span></td>
                    <td>${escapeHtml(description)}</td>
                </tr>
            `;
        }).join('');

        return `
            <div class="suite-detail-section">
                <h3>Test Cases</h3>
                <div class="suite-detail-tests">
                    <table>
                        <thead>
                            <tr>
                                <th>Test Name</th>
                                <th>Type</th>
                                <th>Description</th>
                            </tr>
                        </thead>
                        <tbody>
                            ${testRows}
                        </tbody>
                    </table>
                </div>
            </div>
        `;
    }

    /**
     * Execute test suite
     * @param {string} suiteId - Suite ID
     * @param {HTMLElement} triggerElement - Trigger element (optional)
     */
    async executeSuite(suiteId, triggerElement = null) {
        if (this.debug) {
            console.log('[SuitesPage] Executing suite:', suiteId);
        }

        try {
            // Disable button during execution
            if (triggerElement) {
                triggerElement.disabled = true;
                triggerElement.innerHTML = '<div class="spinner"></div>';
            }

            const result = await this.apiClient.executeTestSuite(suiteId);

            this.notificationManager.showSuccess('Test suite execution started!');

            // Reload data to show updated status
            await this.load();

        } catch (error) {
            console.error('[SuitesPage] Failed to execute suite:', error);
            this.notificationManager.showError(`Failed to execute suite: ${error.message}`);
        } finally {
            if (triggerElement) {
                triggerElement.disabled = false;
                triggerElement.innerHTML = '<i data-lucide="play"></i>';
                refreshIcons();
            }
        }
    }

    /**
     * Open generate dialog for scenario
     * @param {HTMLElement} triggerElement - Trigger element
     * @param {string} scenarioName - Scenario name
     */
    openGenerateDialog(triggerElement, scenarioName) {
        if (this.debug) {
            console.log('[SuitesPage] Opening generate dialog for:', scenarioName);
        }

        this.dialogManager.openGenerateDialog(triggerElement, scenarioName, false);
    }

    /**
     * Execute selected suites
     */
    async runSelectedSuites() {
        const selectedIds = this.selectionManager.getSelectedSuiteIds();

        if (selectedIds.length === 0) {
            this.notificationManager.showWarning('No suites selected');
            return;
        }

        if (this.debug) {
            console.log('[SuitesPage] Running selected suites:', selectedIds);
        }

        try {
            this.notificationManager.showInfo(`Executing ${selectedIds.length} test suite${selectedIds.length === 1 ? '' : 's'}...`);

            // Execute all selected suites in parallel
            await Promise.all(
                selectedIds.map(id => this.apiClient.executeTestSuite(id))
            );

            this.notificationManager.showSuccess(`Executed ${selectedIds.length} test suite${selectedIds.length === 1 ? '' : 's'}!`);

            // Clear selection
            this.selectionManager.clearSuiteSelection();

            // Reload data
            await this.load();

        } catch (error) {
            console.error('[SuitesPage] Failed to execute selected suites:', error);
            this.notificationManager.showError(`Failed to execute suites: ${error.message}`);
        }
    }

    /**
     * Refresh suites data
     */
    async refresh() {
        await this.load();
        this.notificationManager.showSuccess('Suites refreshed!');
    }

    /**
     * Set debug mode
     * @param {boolean} enabled
     */
    setDebug(enabled) {
        this.debug = enabled;
        console.log(`[SuitesPage] Debug mode ${enabled ? 'enabled' : 'disabled'}`);
    }

    /**
     * Get debug information
     * @returns {Object}
     */
    getDebugInfo() {
        return {
            hasSuitesTableContainer: this.suitesTableContainer !== null,
            currentSuiteRowsCount: this.currentSuiteRows.length,
            renderedSuiteIdsCount: this.renderedSuiteIds.length,
            availableSuiteIdsCount: this.availableSuiteIds.length,
            filters: { ...this.suiteFilters }
        };
    }
}

// Export singleton instance
export const suitesPage = new SuitesPage();

// Export default for convenience
export default suitesPage;
