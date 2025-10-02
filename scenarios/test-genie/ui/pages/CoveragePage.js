/**
 * CoveragePage - Code Coverage Analysis View
 * Handles loading, rendering, and generating coverage analyses
 */

import { eventBus, EVENT_TYPES } from '../core/EventBus.js';
import { stateManager } from '../core/StateManager.js';
import { apiClient } from '../core/ApiClient.js';
import { notificationManager } from '../managers/NotificationManager.js';
import { dialogManager } from '../managers/DialogManager.js';
import {
    normalizeCollection,
    formatTimestamp,
    formatLabel,
    escapeHtml
} from '../utils/index.js';
import { enableDragScroll, refreshIcons } from '../utils/domHelpers.js';

/**
 * CoveragePage class - Manages coverage analysis page functionality
 */
export class CoveragePage {
    constructor(
        eventBusInstance = eventBus,
        stateManagerInstance = stateManager,
        apiClientInstance = apiClient,
        notificationManagerInstance = notificationManager,
        dialogManagerInstance = dialogManager
    ) {
        this.eventBus = eventBusInstance;
        this.stateManager = stateManagerInstance;
        this.apiClient = apiClientInstance;
        this.notificationManager = notificationManagerInstance;
        this.dialogManager = dialogManagerInstance;

        // DOM references
        this.coverageTableContainer = null;

        // Debug mode
        this.debug = false;

        this.initialize();
    }

    /**
     * Initialize coverage page
     */
    initialize() {
        this.setupDOMReferences();
        this.setupEventListeners();

        if (this.debug) {
            console.log('[CoveragePage] Coverage page initialized');
        }
    }

    /**
     * Setup DOM element references
     * @private
     */
    setupDOMReferences() {
        this.coverageTableContainer = document.getElementById('coverage-table');
    }

    /**
     * Setup event listeners
     * @private
     */
    setupEventListeners() {
        // Listen for page load requests
        this.eventBus.on(EVENT_TYPES.PAGE_LOAD_REQUESTED, (event) => {
            if (event.data.page === 'coverage') {
                this.load();
            }
        });

        // Listen for data updates
        this.eventBus.on(EVENT_TYPES.DATA_LOADED, (event) => {
            const { collection } = event.data;
            if (collection === 'coverage') {
                const activePage = this.stateManager.get('activePage');
                if (activePage === 'coverage') {
                    this.render();
                }
            }
        });
    }

    /**
     * Load coverage summaries
     */
    async load() {
        if (!this.coverageTableContainer) {
            return;
        }

        if (this.debug) {
            console.log('[CoveragePage] Loading coverage summaries');
        }

        this.coverageTableContainer.innerHTML = '<div class="loading"><div class="spinner"></div>Loading coverage summaries...</div>';

        try {
            // Set loading state
            this.stateManager.setLoading('coverage', true);

            // Load coverage summaries
            const response = await this.apiClient.getCoverageSummaries();
            const coverages = normalizeCollection(response, 'coverages');

            // Update state
            this.stateManager.setData('coverage', coverages || []);

            // Render
            this.render();

            // Emit success event
            this.eventBus.emit(EVENT_TYPES.PAGE_LOADED, { page: 'coverage' });

        } catch (error) {
            console.error('[CoveragePage] Failed to load coverage summaries:', error);
            this.coverageTableContainer.innerHTML = '<p style="color: var(--accent-error); text-align: center; padding: 2rem;">Failed to load coverage summaries.</p>';
            this.coverageTableContainer.scrollLeft = 0;
            enableDragScroll(this.coverageTableContainer);
        } finally {
            this.stateManager.setLoading('coverage', false);
        }
    }

    /**
     * Render coverage table
     */
    render() {
        if (!this.coverageTableContainer) {
            return;
        }

        const coverages = this.stateManager.get('data.coverage') || [];

        if (!coverages || coverages.length === 0) {
            this.coverageTableContainer.innerHTML = '<p style="color: var(--text-muted); text-align: center; padding: 2rem;">No coverage analyses found. Run a scenario\'s unit tests to generate coverage data.</p>';
            this.coverageTableContainer.scrollLeft = 0;
            enableDragScroll(this.coverageTableContainer);
            return;
        }

        this.coverageTableContainer.innerHTML = this.renderCoverageTable(coverages);
        this.coverageTableContainer.scrollLeft = 0;
        enableDragScroll(this.coverageTableContainer);
        this.bindCoverageTableActions();
    }

    /**
     * Render coverage table HTML
     * @param {Array} coverages - Array of coverage summaries
     * @returns {string} HTML string
     * @private
     */
    renderCoverageTable(coverages) {
        const rows = coverages.map((coverage) => {
            const overall = Number(coverage.overall_coverage || 0).toFixed(1);
            const generatedAt = coverage.generated_at ? formatTimestamp(coverage.generated_at) : '—';
            const languages = Array.isArray(coverage.languages)
                ? coverage.languages.map((lang) => {
                    const statements = Number(lang.metrics?.statements ?? lang.metrics?.lines ?? 0).toFixed(1);
                    return `${escapeHtml(formatLabel(lang.language))} (${statements}%)`;
                  }).join(', ')
                : '—';

            const warnings = Array.isArray(coverage.warnings) && coverage.warnings.length > 0
                ? coverage.warnings.map((w) => `<div class="coverage-warning">⚠ ${escapeHtml(w)}</div>`).join('')
                : '';

            return `
                <tr>
                    <td class="cell-scenario"><strong>${escapeHtml(coverage.scenario_name || 'unknown')}</strong>${warnings}</td>
                    <td class="cell-coverage">${overall}%</td>
                    <td>${languages || '—'}</td>
                    <td>${generatedAt}</td>
                    <td class="cell-actions">
                        <button class="btn icon-btn" type="button" data-action="coverage-view" data-scenario="${escapeHtml(coverage.scenario_name)}">
                            <i data-lucide="eye"></i>
                        </button>
                        <button class="btn icon-btn secondary" type="button" data-action="coverage-generate" data-scenario="${escapeHtml(coverage.scenario_name)}">
                            <i data-lucide="zap"></i>
                        </button>
                    </td>
                </tr>
            `;
        }).join('');

        return `
            <table class="table coverage-table">
                <thead>
                    <tr>
                        <th>Scenario</th>
                        <th>Overall Coverage</th>
                        <th>Languages</th>
                        <th>Last Generated</th>
                        <th>Actions</th>
                    </tr>
                </thead>
                <tbody>
                    ${rows}
                </tbody>
            </table>
        `;
    }

    /**
     * Bind actions to coverage table
     * @private
     */
    bindCoverageTableActions() {
        if (!this.coverageTableContainer) {
            return;
        }

        this.coverageTableContainer.querySelectorAll('button[data-action]').forEach((button) => {
            button.addEventListener('click', async (event) => {
                const action = button.dataset.action;
                const scenarioName = button.dataset.scenario;
                if (!scenarioName) {
                    return;
                }

                switch (action) {
                    case 'coverage-view':
                        await this.viewCoverageAnalysis(scenarioName, button);
                        break;
                    case 'coverage-generate':
                        await this.generateCoverageAnalysis(scenarioName, button);
                        break;
                    default:
                        break;
                }
            });
        });

        refreshIcons();
    }

    /**
     * View coverage analysis for a scenario
     * @param {string} scenarioName - Scenario name
     * @param {HTMLElement} triggerButton - Trigger button (optional)
     */
    async viewCoverageAnalysis(scenarioName, triggerButton) {
        if (this.debug) {
            console.log('[CoveragePage] Viewing coverage analysis for:', scenarioName);
        }

        // Open coverage detail dialog
        this.dialogManager.openCoverageDetail(triggerButton, scenarioName);

        // Get coverage detail content container
        const coverageDetailContent = document.getElementById('coverage-detail-content');
        if (!coverageDetailContent) {
            return;
        }

        coverageDetailContent.innerHTML = '<div class="loading"><div class="spinner"></div>Loading coverage analysis...</div>';

        try {
            const analysis = await this.apiClient.getCoverageAnalysis(scenarioName);
            if (!analysis) {
                coverageDetailContent.innerHTML = '<p style="color: var(--accent-error);">Failed to load coverage analysis.</p>';
                return;
            }

            coverageDetailContent.innerHTML = this.renderCoverageDetail(scenarioName, analysis);
            refreshIcons();
        } catch (error) {
            console.error('[CoveragePage] Failed to load coverage analysis:', error);
            coverageDetailContent.innerHTML = '<p style="color: var(--accent-error);">Failed to load coverage analysis.</p>';
        }
    }

    /**
     * Render coverage detail HTML
     * @param {string} scenarioName - Scenario name
     * @param {Object} analysis - Coverage analysis data
     * @returns {string} HTML string
     * @private
     */
    renderCoverageDetail(scenarioName, analysis) {
        const overall = Number(analysis.overall_coverage ?? 0).toFixed(1);
        const coverageRows = Object.entries(analysis.coverage_by_file || {})
            .sort((a, b) => Number(b[1]) - Number(a[1]))
            .slice(0, 15)
            .map(([file, pct]) => `<tr><td>${escapeHtml(file)}</td><td>${Number(pct).toFixed(1)}%</td></tr>`)
            .join('');

        const gaps = analysis.coverage_gaps || {};
        const suggestions = Array.isArray(analysis.improvement_suggestions) && analysis.improvement_suggestions.length
            ? analysis.improvement_suggestions.map((item) => `<li>${escapeHtml(item)}</li>`).join('')
            : '<li>No immediate improvements suggested.</li>';
        const priorities = Array.isArray(analysis.priority_areas) && analysis.priority_areas.length
            ? analysis.priority_areas.map((item) => `<li>${escapeHtml(item)}</li>`).join('')
            : '<li>Monitor coverage trends over time.</li>';

        const gapSection = [
            { label: 'Untested Functions', items: gaps.untested_functions },
            { label: 'Untested Branches', items: gaps.untested_branches },
            { label: 'Untested Edge Cases', items: gaps.untested_edge_cases }
        ].map(({ label, items }) => {
            if (!Array.isArray(items) || items.length === 0) {
                return '';
            }
            return `
                <div class="coverage-gap-section">
                    <strong>${escapeHtml(label)}</strong>
                    <ul>${items.map((item) => `<li>${escapeHtml(item)}</li>`).join('')}</ul>
                </div>
            `;
        }).join('');

        const coverageTable = coverageRows
            ? `<table class="table compact"><thead><tr><th>File</th><th>Coverage</th></tr></thead><tbody>${coverageRows}</tbody></table>`
            : '<p style="color: var(--text-muted);">No file-level coverage data available.</p>';

        return `
            <div class="coverage-detail">
                <div class="coverage-detail-header">
                    <h3 style="margin-bottom: 0.25rem;">${escapeHtml(scenarioName)}</h3>
                    <p style="color: var(--text-muted);">Overall coverage: <strong>${overall}%</strong></p>
                </div>
                <div class="coverage-detail-grid">
                    <div class="coverage-detail-column">
                        <h4>Coverage by File</h4>
                        ${coverageTable}
                    </div>
                    <div class="coverage-detail-column">
                        <h4>Priority Areas</h4>
                        <ul>${priorities}</ul>
                        <h4>Improvement Suggestions</h4>
                        <ul>${suggestions}</ul>
                        ${gapSection}
                    </div>
                </div>
            </div>
        `;
    }

    /**
     * Generate coverage analysis for a scenario
     * @param {string} scenarioName - Scenario name
     * @param {HTMLElement} triggerButton - Trigger button (optional)
     */
    async generateCoverageAnalysis(scenarioName, triggerButton) {
        if (!scenarioName) {
            this.notificationManager.showError('Scenario name missing');
            return;
        }

        if (this.debug) {
            console.log('[CoveragePage] Generating coverage analysis for:', scenarioName);
        }

        const previousDisabled = triggerButton?.disabled || false;
        if (triggerButton) {
            triggerButton.disabled = true;
        }

        try {
            await this.apiClient.generateCoverageAnalysis(scenarioName);

            this.notificationManager.showSuccess(`Coverage analysis generated for ${scenarioName}`);

            // Reload coverage summaries
            await this.load();

            // View the new analysis
            await this.viewCoverageAnalysis(scenarioName, null);

        } catch (error) {
            console.error('[CoveragePage] Coverage analysis generation failed:', error);
            this.notificationManager.showError(`Coverage analysis failed: ${error.message}`);
        } finally {
            if (triggerButton) {
                triggerButton.disabled = previousDisabled;
            }
        }
    }

    /**
     * Refresh coverage data
     */
    async refresh() {
        await this.load();
        this.notificationManager.showSuccess('Coverage refreshed!');
    }

    /**
     * Set debug mode
     * @param {boolean} enabled
     */
    setDebug(enabled) {
        this.debug = enabled;
        console.log(`[CoveragePage] Debug mode ${enabled ? 'enabled' : 'disabled'}`);
    }

    /**
     * Get debug information
     * @returns {Object}
     */
    getDebugInfo() {
        return {
            hasCoverageTableContainer: this.coverageTableContainer !== null,
            coverageCount: (this.stateManager.get('data.coverage') || []).length
        };
    }
}

// Export singleton instance
export const coveragePage = new CoveragePage();

// Export default for convenience
export default coveragePage;
