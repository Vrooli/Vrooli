/**
 * DetailPanelRenderer - Detail panel and complex component rendering
 *
 * Provides rendering for:
 * - Vault phase timelines
 * - Execution detail panels
 * - Test results displays
 * - Coverage detail views
 * - Phase configuration UI
 */

import {
    escapeHtml,
    formatPhaseLabel,
    formatDurationSeconds,
    formatLabel,
    formatTimestamp,
    getStatusDescriptor,
    formatPercent
} from '../utils/index.js';

/**
 * DetailPanelRenderer class - Handles complex detail panel rendering
 */
export class DetailPanelRenderer {
    constructor() {
        this.debug = false;
    }

    /**
     * Render vault phase timeline
     * @param {Array} phases - Ordered phase names
     * @param {Object} phaseResults - Phase results object
     * @param {Set} completedSet - Set of completed phase names (lowercase)
     * @param {Set} failedSet - Set of failed phase names (lowercase)
     * @param {string} currentPhase - Current running phase (lowercase)
     * @returns {string} HTML string
     */
    renderVaultTimeline(phases, phaseResults = {}, completedSet = new Set(), failedSet = new Set(), currentPhase = '') {
        if (!phases || phases.length === 0) {
            return '<div class="history-empty">No phase data available yet. Run the vault to populate results.</div>';
        }

        const timelineHtml = phases.map((phaseName) => {
            const lowerPhase = phaseName.toLowerCase();
            let descriptor = getStatusDescriptor(phaseResults[lowerPhase]?.status);

            // Override status based on phase state
            if (failedSet.has(lowerPhase)) {
                descriptor = { key: 'failed', label: 'Failed', icon: 'x' };
            } else if (completedSet.has(lowerPhase)) {
                descriptor = { key: 'completed', label: 'Completed', icon: 'check' };
            } else if (currentPhase === lowerPhase) {
                descriptor = { key: 'running', label: 'In Progress', icon: 'zap' };
            }

            const result = phaseResults[lowerPhase];
            const detailParts = [];

            // Add duration
            const duration = result?.duration;
            if (Number.isFinite(duration)) {
                detailParts.push(`Duration: ${formatDurationSeconds(duration)}`);
            }

            // Add test count
            let testCount = result?.test_count;
            if (!Number.isFinite(testCount) && Array.isArray(result?.test_results)) {
                testCount = result.test_results.length;
            }
            if (Number.isFinite(testCount)) {
                detailParts.push(`${testCount} tests`);
            }

            // Default detail if no specific info
            if (!detailParts.length) {
                if (descriptor.key === 'pending') {
                    detailParts.push('Waiting to start');
                } else {
                    detailParts.push(descriptor.label);
                }
            }

            const iconHtml = descriptor.icon ? `<i data-lucide="${descriptor.icon}"></i>` : '';
            const label = formatPhaseLabel(phaseName);

            return `
                <div class="timeline-phase">
                    <div class="timeline-indicator ${descriptor.key}">${iconHtml}</div>
                    <div class="timeline-content">
                        <div class="timeline-phase-name">${escapeHtml(label)}</div>
                        <div class="timeline-phase-status">${detailParts.map((part) => escapeHtml(part)).join(' • ')}</div>
                    </div>
                </div>
            `;
        }).join('');

        return timelineHtml;
    }

    /**
     * Render test results detail
     * @param {Object} execution - Execution object
     * @param {Array} testResults - Array of test results
     * @returns {string} HTML string
     */
    renderTestResultsDetail(execution, testResults = []) {
        if (!testResults || testResults.length === 0) {
            return '<div class="detail-empty">No test results available for this execution.</div>';
        }

        const passedCount = testResults.filter(t => t.status === 'passed').length;
        const failedCount = testResults.filter(t => t.status === 'failed').length;
        const skippedCount = testResults.filter(t => t.status === 'skipped').length;

        const summaryHtml = `
            <div class="results-summary">
                <div class="summary-stat">
                    <span class="stat-value text-success">${passedCount}</span>
                    <span class="stat-label">Passed</span>
                </div>
                <div class="summary-stat">
                    <span class="stat-value text-error">${failedCount}</span>
                    <span class="stat-label">Failed</span>
                </div>
                <div class="summary-stat">
                    <span class="stat-value text-muted">${skippedCount}</span>
                    <span class="stat-label">Skipped</span>
                </div>
            </div>
        `;

        const resultsHtml = testResults.map(test => {
            const descriptor = getStatusDescriptor(test.status);
            const icon = descriptor.icon ? `<i data-lucide="${descriptor.icon}"></i>` : '';
            const durationLabel = Number.isFinite(test.duration)
                ? formatDurationSeconds(test.duration)
                : '—';

            const errorHtml = test.error_message
                ? `<div class="test-error"><pre>${escapeHtml(test.error_message)}</pre></div>`
                : '';

            return `
                <div class="test-result ${descriptor.key}">
                    <div class="test-header">
                        <span class="test-status">${icon} ${descriptor.label}</span>
                        <span class="test-name">${escapeHtml(test.name || 'Unnamed Test')}</span>
                        <span class="test-duration">${durationLabel}</span>
                    </div>
                    ${errorHtml}
                </div>
            `;
        }).join('');

        return `${summaryHtml}<div class="test-results-list">${resultsHtml}</div>`;
    }

    /**
     * Render coverage detail panel
     * @param {Object} coverage - Coverage data
     * @returns {string} HTML string
     */
    renderCoverageDetail(coverage) {
        if (!coverage) {
            return '<div class="detail-empty">No coverage data available.</div>';
        }

        const overall = Number(coverage.overall_coverage || 0).toFixed(1);
        const files = Array.isArray(coverage.files) ? coverage.files : [];

        const overviewHtml = `
            <div class="coverage-overview">
                <div class="coverage-stat-large">
                    <div class="stat-value">${overall}%</div>
                    <div class="stat-label">Overall Coverage</div>
                </div>
            </div>
        `;

        const filesHtml = files.length > 0
            ? files.map(file => {
                const filePercent = Number(file.coverage || 0).toFixed(1);
                const lines = file.lines_covered && file.lines_total
                    ? `${file.lines_covered}/${file.lines_total} lines`
                    : '';

                return `
                    <div class="coverage-file">
                        <div class="file-header">
                            <span class="file-path">${escapeHtml(file.path || 'Unknown file')}</span>
                            <span class="file-coverage">${filePercent}%</span>
                        </div>
                        ${lines ? `<div class="file-meta">${escapeHtml(lines)}</div>` : ''}
                        <div class="progress">
                            <div class="progress-bar" style="width: ${filePercent}%"></div>
                        </div>
                    </div>
                `;
              }).join('')
            : '<div class="detail-empty">No file-level coverage data available.</div>';

        return `${overviewHtml}<div class="coverage-files">${filesHtml}</div>`;
    }

    /**
     * Render execution summary panel
     * @param {Object} execution - Execution object
     * @returns {string} HTML string
     */
    renderExecutionSummary(execution) {
        if (!execution) {
            return '<div class="detail-empty">No execution data available.</div>';
        }

        const descriptor = getStatusDescriptor(execution.status);
        const icon = descriptor.icon ? `<i data-lucide="${descriptor.icon}"></i>` : '';

        const items = [
            { label: 'Status', value: `${icon} ${descriptor.label}`, class: descriptor.key },
            { label: 'Suite', value: execution.suite_name || 'Unknown' },
            { label: 'Started', value: formatTimestamp(execution.start_time) },
            { label: 'Duration', value: formatDurationSeconds(execution.duration || 0) },
            { label: 'Passed Tests', value: execution.passed_tests || 0, class: 'text-success' },
            { label: 'Failed Tests', value: execution.failed_tests || 0, class: execution.failed_tests > 0 ? 'text-error' : '' },
            { label: 'Coverage', value: formatPercent(execution.coverage || 0, 1) }
        ];

        const itemsHtml = items.map(item => `
            <div class="summary-item">
                <span class="item-label">${escapeHtml(item.label)}</span>
                <span class="item-value ${item.class || ''}">${escapeHtml(String(item.value))}</span>
            </div>
        `).join('');

        return `
            <div class="execution-summary">
                <h3 class="summary-title">Execution Summary</h3>
                <div class="summary-items">
                    ${itemsHtml}
                </div>
            </div>
        `;
    }

    /**
     * Render phase configuration form
     * @param {string} phase - Phase name
     * @param {Object} config - Phase configuration
     * @param {number} config.timeout - Timeout in seconds
     * @param {string} config.description - Phase description/intent
     * @returns {string} HTML string
     */
    renderPhaseConfigForm(phase, config = {}) {
        const {
            timeout = 600,
            description = ''
        } = config;

        const label = formatPhaseLabel(phase);

        return `
            <div class="phase-config" data-phase="${escapeHtml(phase)}">
                <h4 class="phase-config-title">${escapeHtml(label)}</h4>
                <div class="form-field">
                    <label for="timeout-${escapeHtml(phase)}">Timeout (seconds)</label>
                    <input
                        type="number"
                        id="timeout-${escapeHtml(phase)}"
                        data-vault-timeout="${escapeHtml(phase)}"
                        value="${timeout}"
                        min="60"
                        max="3600"
                        step="60"
                    >
                </div>
                <div class="form-field">
                    <label for="desc-${escapeHtml(phase)}">Phase Intent / Description</label>
                    <textarea
                        id="desc-${escapeHtml(phase)}"
                        data-vault-desc="${escapeHtml(phase)}"
                        rows="3"
                        placeholder="Describe what this phase should accomplish..."
                    >${escapeHtml(description)}</textarea>
                </div>
            </div>
        `;
    }

    /**
     * Render scenario analytics row (for reports page)
     * @param {Object} scenario - Scenario analytics data
     * @returns {string} HTML string
     */
    renderScenarioAnalyticsRow(scenario) {
        const health = Number(scenario.health_score || 0);
        const healthClass = health >= 80 ? 'health-good' : health >= 50 ? 'health-warning' : 'health-poor';
        const passRate = Number(scenario.pass_rate || 0);
        const avgDuration = Number(scenario.avg_duration || 0);

        return `
            <tr>
                <td class="cell-scenario"><strong>${escapeHtml(scenario.scenario_name || 'Unknown')}</strong></td>
                <td class="cell-health ${healthClass}">${health.toFixed(0)}</td>
                <td>${passRate.toFixed(1)}%</td>
                <td>${Number(scenario.total_executions || 0)}</td>
                <td>${formatDurationSeconds(avgDuration)}</td>
            </tr>
        `;
    }

    /**
     * Render empty state
     * @param {string} message - Empty state message
     * @param {string} icon - Lucide icon name
     * @returns {string} HTML string
     */
    renderEmptyState(message = 'No data available', icon = 'inbox') {
        return `
            <div class="detail-empty">
                <i data-lucide="${escapeHtml(icon)}"></i>
                <p>${escapeHtml(message)}</p>
            </div>
        `;
    }

    /**
     * Render loading state
     * @param {string} message - Loading message
     * @returns {string} HTML string
     */
    renderLoadingState(message = 'Loading details...') {
        return `<div class="loading-message">${escapeHtml(message)}</div>`;
    }

    /**
     * Render error state
     * @param {string} message - Error message
     * @returns {string} HTML string
     */
    renderErrorState(message = 'Failed to load details') {
        return `<div class="error-message">${escapeHtml(message)}</div>`;
    }

    /**
     * Set debug mode
     * @param {boolean} enabled
     */
    setDebug(enabled) {
        this.debug = enabled;
        console.log(`[DetailPanelRenderer] Debug mode ${enabled ? 'enabled' : 'disabled'}`);
    }
}

// Export singleton instance
export const detailPanelRenderer = new DetailPanelRenderer();

// Export default for convenience
export default detailPanelRenderer;
