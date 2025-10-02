/**
 * CardRenderer - Card and metric rendering for Test Genie
 *
 * Provides consistent card rendering for:
 * - Dashboard metrics cards
 * - Report insights cards
 * - Scenario cards (suites page)
 * - Overview metric displays
 * - Summary cards
 */

import {
    escapeHtml,
    formatLabel,
    formatPercent,
    getStatusClass,
    normalizeId
} from '../utils/index.js';

/**
 * CardRenderer class - Handles all card and metric rendering
 */
export class CardRenderer {
    constructor() {
        this.debug = false;
    }

    /**
     * Render metric card
     * @param {Object} metric - Metric data
     * @param {string} metric.title - Metric title
     * @param {string|number} metric.value - Metric value
     * @param {string} metric.subtitle - Metric subtitle
     * @param {string} metric.icon - Lucide icon name
     * @param {string} metric.trend - Trend indicator (up, down, neutral)
     * @returns {string} HTML string
     */
    renderMetricCard(metric) {
        const {
            title = 'Metric',
            value = '0',
            subtitle = '',
            icon = 'bar-chart',
            trend = null
        } = metric;

        const trendIndicator = trend
            ? `<span class="metric-trend trend-${trend}">
                <i data-lucide="trending-${trend === 'up' ? 'up' : 'down'}"></i>
               </span>`
            : '';

        return `
            <div class="metric-card">
                <div class="metric-header">
                    <i data-lucide="${escapeHtml(icon)}"></i>
                    <span class="metric-title">${escapeHtml(title)}</span>
                    ${trendIndicator}
                </div>
                <div class="metric-value">${escapeHtml(String(value))}</div>
                ${subtitle ? `<div class="metric-subtitle">${escapeHtml(subtitle)}</div>` : ''}
            </div>
        `;
    }

    /**
     * Render insight card
     * @param {Object} insight - Insight data
     * @param {string} insight.title - Insight title
     * @param {string} insight.detail - Insight detail/description
     * @param {string} insight.severity - Severity level (high, medium, low, info)
     * @param {Array} insight.actions - Suggested actions
     * @param {string} insight.scenario_name - Related scenario name
     * @returns {string} HTML string
     */
    renderInsightCard(insight) {
        const {
            title = 'Insight',
            detail = '',
            severity = 'info',
            actions = [],
            scenario_name = null
        } = insight;

        const severityClassMap = {
            high: 'insight-severity-high',
            medium: 'insight-severity-medium',
            info: 'insight-severity-info',
            low: 'insight-severity-low'
        };

        const severityIconMap = {
            high: 'alert-triangle',
            medium: 'alert-circle',
            info: 'info',
            low: 'sparkles'
        };

        const severityKey = (severity || 'info').toLowerCase();
        const severityClass = severityClassMap[severityKey] || severityClassMap.info;
        const severityIcon = severityIconMap[severityKey] || severityIconMap.info;
        const severityLabel = formatLabel(severity || 'info');

        const scenarioBadge = scenario_name
            ? `<span class="insight-scenario"><i data-lucide="box"></i>${escapeHtml(scenario_name)}</span>`
            : '';

        const actionsHtml = actions.length
            ? `<ul class="insight-actions">${actions.map((action) => `<li>${escapeHtml(action)}</li>`).join('')}</ul>`
            : '';

        return `
            <article class="insight-card">
                <header class="insight-header">
                    <div>
                        <h3 class="insight-title">${escapeHtml(title)}</h3>
                        <div class="insight-meta">${scenarioBadge}</div>
                    </div>
                    <span class="insight-severity ${severityClass}">
                        <i data-lucide="${severityIcon}"></i>${escapeHtml(severityLabel)}
                    </span>
                </header>
                <p class="insight-detail">${escapeHtml(detail)}</p>
                ${actionsHtml}
            </article>
        `;
    }

    /**
     * Render suite card (for suites page cards view)
     * @param {Object} scenario - Scenario data
     * @param {Function} isSelected - Function to check if suite is selected
     * @returns {string} HTML string
     */
    renderSuiteCard(scenario, isSelected = () => false) {
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
        const cardClass = isMissing ? 'suite-card missing-scenario' : 'suite-card has-suite';

        const checkbox = hasSuite
            ? `<input type="checkbox" data-suite-id="${suiteId}" ${isChecked ? 'checked' : ''} class="suite-card-checkbox" aria-label="Select suite for ${escapeHtml(scenario.scenarioName)}">`
            : '';

        const actions = hasSuite
            ? `<button class="btn icon-btn" type="button" data-action="execute" data-suite-id="${suiteId}" aria-label="Execute latest suite">
                <i data-lucide="play"></i>
               </button>`
            : `<button class="btn icon-btn highlighted" type="button" data-action="generate" data-scenario="${escapeHtml(scenario.scenarioName)}" aria-label="Generate tests">
                <i data-lucide="sparkles"></i>
               </button>`;

        return `
            <div class="${cardClass}" data-suite-id="${suiteId}">
                <div class="suite-card-header">
                    ${checkbox}
                    <strong class="suite-card-scenario">${escapeHtml(scenario.scenarioName)}</strong>
                    <span class="suite-card-status status ${statusClass}">${statusLabel}</span>
                </div>
                <div class="suite-card-body">
                    <div class="suite-card-coverage">
                        <div class="progress">
                            <div class="progress-bar" style="width: ${coverageValue}%"></div>
                        </div>
                        <div class="coverage-details">
                            <span class="coverage-percentage">${coverageValue}%</span>
                            <span class="coverage-phases">${phasesLabel}</span>
                        </div>
                    </div>
                </div>
                <div class="suite-card-actions">
                    ${actions}
                </div>
            </div>
        `;
    }

    /**
     * Render summary card (for dashboard overview)
     * @param {Object} summary - Summary data
     * @param {string} summary.title - Summary title
     * @param {Array} summary.items - Summary items [{label, value, highlight}]
     * @param {string} summary.icon - Lucide icon name
     * @returns {string} HTML string
     */
    renderSummaryCard(summary) {
        const {
            title = 'Summary',
            items = [],
            icon = 'file-text'
        } = summary;

        const itemsHtml = items.map(item => {
            const highlightClass = item.highlight ? 'summary-item-highlight' : '';
            return `
                <div class="summary-item ${highlightClass}">
                    <span class="summary-label">${escapeHtml(item.label)}</span>
                    <span class="summary-value">${escapeHtml(String(item.value))}</span>
                </div>
            `;
        }).join('');

        return `
            <div class="summary-card">
                <div class="summary-header">
                    <i data-lucide="${escapeHtml(icon)}"></i>
                    <h3 class="summary-title">${escapeHtml(title)}</h3>
                </div>
                <div class="summary-body">
                    ${itemsHtml}
                </div>
            </div>
        `;
    }

    /**
     * Render status badge
     * @param {string} status - Status value
     * @param {Object} options - Rendering options
     * @param {boolean} options.showIcon - Show status icon
     * @returns {string} HTML string
     */
    renderStatusBadge(status, options = {}) {
        const { showIcon = false } = options;

        const statusClass = getStatusClass(status);
        const statusLabel = formatLabel(status);

        const iconHtml = showIcon
            ? `<i data-lucide="${this.getStatusIcon(status)}"></i>`
            : '';

        return `<span class="status ${statusClass}">${iconHtml}${escapeHtml(statusLabel)}</span>`;
    }

    /**
     * Get icon for status
     * @param {string} status - Status value
     * @returns {string} Lucide icon name
     * @private
     */
    getStatusIcon(status) {
        const statusLower = (status || '').toLowerCase();

        const iconMap = {
            running: 'loader',
            pending: 'clock',
            completed: 'check-circle',
            success: 'check-circle',
            failed: 'x-circle',
            error: 'alert-circle',
            cancelled: 'x-circle',
            timeout: 'clock'
        };

        return iconMap[statusLower] || 'circle';
    }

    /**
     * Render progress bar
     * @param {number} percentage - Progress percentage (0-100)
     * @param {Object} options - Rendering options
     * @param {string} options.label - Label text
     * @param {string} options.color - Progress bar color class
     * @returns {string} HTML string
     */
    renderProgressBar(percentage, options = {}) {
        const {
            label = null,
            color = 'primary'
        } = options;

        const clampedPercentage = Math.max(0, Math.min(100, percentage));

        const labelHtml = label
            ? `<span class="progress-label">${escapeHtml(label)}</span>`
            : '';

        return `
            <div class="progress-container">
                ${labelHtml}
                <div class="progress">
                    <div class="progress-bar progress-${color}" style="width: ${clampedPercentage}%"></div>
                </div>
                <span class="progress-percentage">${clampedPercentage}%</span>
            </div>
        `;
    }

    /**
     * Render empty state card
     * @param {string} message - Empty state message
     * @param {string} icon - Lucide icon name
     * @returns {string} HTML string
     */
    renderEmptyCard(message = 'No data available', icon = 'inbox') {
        return `
            <div class="empty-card">
                <i data-lucide="${escapeHtml(icon)}" class="empty-icon"></i>
                <p class="empty-message">${escapeHtml(message)}</p>
            </div>
        `;
    }

    /**
     * Render loading card
     * @param {string} message - Loading message
     * @returns {string} HTML string
     */
    renderLoadingCard(message = 'Loading...') {
        return `
            <div class="loading-card">
                <div class="spinner"></div>
                <p class="loading-message">${escapeHtml(message)}</p>
            </div>
        `;
    }

    /**
     * Render error card
     * @param {string} message - Error message
     * @returns {string} HTML string
     */
    renderErrorCard(message = 'An error occurred') {
        return `
            <div class="error-card">
                <i data-lucide="alert-circle" class="error-icon"></i>
                <p class="error-message">${escapeHtml(message)}</p>
            </div>
        `;
    }

    /**
     * Set debug mode
     * @param {boolean} enabled
     */
    setDebug(enabled) {
        this.debug = enabled;
        console.log(`[CardRenderer] Debug mode ${enabled ? 'enabled' : 'disabled'}`);
    }
}

// Export singleton instance
export const cardRenderer = new CardRenderer();

// Export default for convenience
export default cardRenderer;
