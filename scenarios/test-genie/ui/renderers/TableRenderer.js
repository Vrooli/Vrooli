/**
 * TableRenderer - Unified table rendering for Test Genie
 *
 * Provides consistent table rendering across all pages with:
 * - Executions tables (dashboard, executions page)
 * - Suites tables (suites page)
 * - Coverage tables (coverage page)
 * - Vault history tables (vault page)
 * - Consistent styling and behavior
 * - Selection support
 * - Action buttons
 */

import {
    escapeHtml,
    formatTimestamp,
    formatDateTime,
    formatDurationSeconds,
    formatLabel,
    getStatusClass,
    normalizeId,
    calculateDuration
} from '../utils/index.js';

/**
 * TableRenderer class - Handles all table rendering logic
 */
export class TableRenderer {
    constructor() {
        this.debug = false;
    }

    /**
     * Render executions table
     * @param {Array<Object>} executions - Array of execution objects
     * @param {Object} options - Rendering options
     * @param {boolean} options.selectable - Whether table supports selection (default: false)
     * @param {Function} options.isSelected - Function to check if row is selected
     * @param {boolean} options.showDeleteButton - Show delete button (default: false)
     * @returns {string} HTML string
     */
    renderExecutionsTable(executions, options = {}) {
        const {
            selectable = false,
            isSelected = () => false,
            showDeleteButton = false
        } = options;

        if (!executions || executions.length === 0) {
            return '<p style="color: var(--text-muted); text-align: center; padding: 2rem;">No test executions found</p>';
        }

        const selectHeader = selectable
            ? '<th class="cell-select"><input type="checkbox" data-execution-select-all aria-label="Select all executions"></th>'
            : '';

        const rows = executions.map(exec =>
            this.renderExecutionRow(exec, { selectable, isSelected, showDeleteButton })
        ).join('');

        return `
            <table class="table executions-table" data-selectable="${selectable}">
                <thead>
                    <tr>
                        ${selectHeader}
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
                    ${rows}
                </tbody>
            </table>
        `;
    }

    /**
     * Render single execution row
     * @param {Object} exec - Execution object
     * @param {Object} options - Rendering options
     * @returns {string} HTML string
     * @private
     */
    renderExecutionRow(exec, options = {}) {
        const {
            selectable = false,
            isSelected = () => false,
            showDeleteButton = false
        } = options;

        const executionId = normalizeId(exec.id);
        const isChecked = selectable && isSelected(executionId);
        const failedClass = exec.failed > 0 ? 'text-error' : 'text-muted';
        const durationLabel = Number.isFinite(exec.duration) ? `${exec.duration}s` : '—';
        const statusClass = exec.statusClass || getStatusClass(exec.status);

        const selectCell = selectable
            ? `<td class="cell-select">
                <input type="checkbox" data-execution-id="${executionId}" ${isChecked ? 'checked' : ''} aria-label="Select execution">
               </td>`
            : '';

        const deleteButton = showDeleteButton
            ? `<button class="btn icon-btn" type="button" data-action="delete-execution" data-execution-id="${executionId}" aria-label="Delete execution">
                <i data-lucide="trash-2"></i>
               </button>`
            : '';

        return `
            <tr data-execution-id="${executionId}">
                ${selectCell}
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
                    ${deleteButton}
                </td>
            </tr>
        `;
    }

    /**
     * Render suites table
     * @param {Array<Object>} scenarios - Array of scenario row objects
     * @param {Object} options - Rendering options
     * @param {Function} options.isSelected - Function to check if suite is selected
     * @returns {string} HTML string
     */
    renderSuitesTable(scenarios, options = {}) {
        const { isSelected = () => false } = options;

        if (!scenarios || scenarios.length === 0) {
            return '<p style="color: var(--text-muted); text-align: center; padding: 2rem;">No scenarios found</p>';
        }

        const rows = scenarios.map(scenario =>
            this.renderSuiteRow(scenario, { isSelected })
        ).join('');

        return `
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
                    ${rows}
                </tbody>
            </table>
        `;
    }

    /**
     * Render single suite row
     * @param {Object} scenario - Scenario row object
     * @param {Object} options - Rendering options
     * @returns {string} HTML string
     * @private
     */
    renderSuiteRow(scenario, options = {}) {
        const { isSelected = () => false } = options;

        const suiteId = scenario.latestSuiteId ? normalizeId(scenario.latestSuiteId) : '';
        const isChecked = suiteId && isSelected(suiteId);
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
     * Render coverage table
     * @param {Array<Object>} coverages - Array of coverage summary objects
     * @returns {string} HTML string
     */
    renderCoverageTable(coverages) {
        if (!coverages || coverages.length === 0) {
            return '<p style="color: var(--text-muted); text-align: center; padding: 2rem;">No coverage data found</p>';
        }

        const rows = coverages.map(coverage =>
            this.renderCoverageRow(coverage)
        ).join('');

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
     * Render single coverage row
     * @param {Object} coverage - Coverage summary object
     * @returns {string} HTML string
     * @private
     */
    renderCoverageRow(coverage) {
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
    }

    /**
     * Render vault history table
     * @param {Array<Object>} executions - Array of vault execution objects
     * @returns {string} HTML string
     */
    renderVaultHistoryTable(executions) {
        if (!executions || executions.length === 0) {
            return '<div class="history-empty">No execution history available.</div>';
        }

        const rows = executions.map((execution, index) =>
            this.renderVaultHistoryRow(execution, index)
        ).join('');

        return `
            <table class="history-table">
                <thead>
                    <tr>
                        <th>Started</th>
                        <th>Status</th>
                        <th>Completed</th>
                        <th>Failed</th>
                        <th>Duration</th>
                    </tr>
                </thead>
                <tbody>
                    ${rows}
                </tbody>
            </table>
        `;
    }

    /**
     * Render single vault history row
     * @param {Object} execution - Vault execution object
     * @param {number} index - Row index
     * @returns {string} HTML string
     * @private
     */
    renderVaultHistoryRow(execution, index) {
        const status = execution.status || 'unknown';
        const statusClass = getStatusClass(status);
        const statusHtml = `<span class="status ${statusClass}">${escapeHtml(formatLabel(status))}</span>`;

        let durationSeconds = Number(execution.duration);
        if (!Number.isFinite(durationSeconds)) {
            durationSeconds = calculateDuration(execution.start_time, execution.end_time);
        }

        const completedCount = Array.isArray(execution.completed_phases) ? execution.completed_phases.length : 0;
        const failedCount = Array.isArray(execution.failed_phases) ? execution.failed_phases.length : 0;

        return `
            <tr ${index === 0 ? 'data-latest="true"' : ''}>
                <td>${escapeHtml(formatDateTime(execution.start_time))}</td>
                <td>${statusHtml}</td>
                <td>${completedCount}</td>
                <td>${failedCount}</td>
                <td>${escapeHtml(formatDurationSeconds(durationSeconds))}</td>
            </tr>
        `;
    }

    /**
     * Render vault list
     * @param {Array<Object>} vaults - Array of vault objects
     * @returns {string} HTML string
     */
    renderVaultList(vaults) {
        if (!vaults || vaults.length === 0) {
            return '<div class="vault-empty">No vaults created yet. Create your first vault to get started!</div>';
        }

        const items = vaults.map(vault =>
            this.renderVaultListItem(vault)
        ).join('');

        return `<div class="vault-list">${items}</div>`;
    }

    /**
     * Render single vault list item
     * @param {Object} vault - Vault object
     * @returns {string} HTML string
     * @private
     */
    renderVaultListItem(vault) {
        const vaultId = normalizeId(vault.id);
        const scenarioName = escapeHtml(vault.scenario_name || 'Unknown');
        const createdAt = vault.created_at ? formatTimestamp(vault.created_at) : '—';
        const phaseCount = Array.isArray(vault.phases) ? vault.phases.length : 0;
        const phasesLabel = phaseCount > 0
            ? `${phaseCount} phase${phaseCount === 1 ? '' : 's'}`
            : 'No phases';

        return `
            <div class="vault-item" data-vault-id="${vaultId}">
                <div class="vault-item-header">
                    <strong class="vault-scenario">${scenarioName}</strong>
                    <span class="vault-id">${vaultId}</span>
                </div>
                <div class="vault-item-meta">
                    <span class="vault-phases">${phasesLabel}</span>
                    <span class="vault-created">Created ${createdAt}</span>
                </div>
            </div>
        `;
    }

    /**
     * Render empty state message
     * @param {string} message - Message to display
     * @returns {string} HTML string
     */
    renderEmptyState(message = 'No data found') {
        return `<p style="color: var(--text-muted); text-align: center; padding: 2rem;">${escapeHtml(message)}</p>`;
    }

    /**
     * Render loading state
     * @param {string} message - Loading message
     * @returns {string} HTML string
     */
    renderLoadingState(message = 'Loading...') {
        return `<div class="loading-message">${escapeHtml(message)}</div>`;
    }

    /**
     * Render error state
     * @param {string} message - Error message
     * @returns {string} HTML string
     */
    renderErrorState(message = 'Failed to load data') {
        return `<div class="error-message">${escapeHtml(message)}</div>`;
    }

    /**
     * Set debug mode
     * @param {boolean} enabled
     */
    setDebug(enabled) {
        this.debug = enabled;
        console.log(`[TableRenderer] Debug mode ${enabled ? 'enabled' : 'disabled'}`);
    }
}

// Export singleton instance
export const tableRenderer = new TableRenderer();

// Export default for convenience
export default tableRenderer;
