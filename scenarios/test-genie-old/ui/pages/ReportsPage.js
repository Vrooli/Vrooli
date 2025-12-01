/**
 * ReportsPage - Analytics and Reports View
 * Handles loading and displaying analytics, trends, and insights
 */

import { eventBus, EVENT_TYPES } from '../core/EventBus.js';
import { stateManager } from '../core/StateManager.js';
import { apiClient } from '../core/ApiClient.js';
import { notificationManager } from '../managers/NotificationManager.js';
import {
    formatPercent,
    formatLabel,
    formatDateRange,
    escapeHtml,
    getStatusDescriptor,
    getStatusClass
} from '../utils/index.js';
import { refreshIcons } from '../utils/domHelpers.js';

/**
 * ReportsPage class - Manages analytics and reports
 */
export class ReportsPage {
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

        // State
        this.reportsWindowDays = 30;
        this.reportsIsLoading = false;
        this.reportsTrendCanvas = null;

        // Debug mode
        this.debug = false;

        this.initialize();
    }

    /**
     * Initialize reports page
     */
    initialize() {
        this.setupEventListeners();

        if (this.debug) {
            console.log('[ReportsPage] Reports page initialized');
        }
    }

    /**
     * Setup event listeners
     * @private
     */
    setupEventListeners() {
        // Listen for page load requests
        this.eventBus.on(EVENT_TYPES.PAGE_LOAD_REQUESTED, (event) => {
            if (event.data.page === 'reports') {
                this.load();
            }
        });

        // Listen for window resize to redraw charts
        window.addEventListener('resize', () => {
            if (this.stateManager.get('activePage') === 'reports') {
                this.handleWindowResize();
            }
        });

        // Listen for time window changes
        const windowSelect = document.getElementById('reports-window-select');
        if (windowSelect) {
            windowSelect.addEventListener('change', (event) => {
                const days = Number(event.target.value);
                if (Number.isFinite(days) && days > 0) {
                    this.updateReportsTimeWindow(days);
                }
            });
        }
    }

    /**
     * Load reports data
     */
    async load() {
        if (this.debug) {
            console.log('[ReportsPage] Loading reports');
        }

        const scenariosContainer = document.getElementById('reports-scenarios');
        if (!scenariosContainer) {
            return;
        }

        const windowSelect = document.getElementById('reports-window-select');
        if (windowSelect && Number(windowSelect.value) !== this.reportsWindowDays) {
            windowSelect.value = String(this.reportsWindowDays);
        }

        try {
            // Set loading state
            this.stateManager.setLoading('reports', true);
            this.reportsIsLoading = true;
            this.showReportsLoading();

            const query = `?window_days=${encodeURIComponent(this.reportsWindowDays)}`;

            // Load reports data in parallel
            const [overview, trends, insights] = await Promise.all([
                this.apiClient.getReportsOverview(query),
                this.apiClient.getReportsTrends(query),
                this.apiClient.getReportsInsights(query)
            ]);

            if (!overview || !trends || !insights) {
                throw new Error('reports endpoints returned empty responses');
            }

            // Update state
            this.stateManager.setData('reports', { overview, trends, insights });

            // Render all sections
            this.updateReportsWindowMeta(overview);
            this.renderReportsOverview(overview);
            this.renderReportsTrends(trends);
            this.renderReportsScenarioTable(overview);
            this.renderReportsInsights(insights);
            refreshIcons();

            // Emit success event
            this.eventBus.emit(EVENT_TYPES.PAGE_LOADED, { page: 'reports' });

        } catch (error) {
            console.error('[ReportsPage] Failed to load reports:', error);
            this.renderReportsError('Failed to load analytics. Please try again.');
        } finally {
            this.reportsIsLoading = false;
            this.stateManager.setLoading('reports', false);
        }
    }

    /**
     * Show loading state for all report sections
     * @private
     */
    showReportsLoading() {
        const metricValueIds = ['report-quality-index', 'report-regressions', 'report-coverage', 'report-vault'];
        const metricSubtitleIds = ['report-quality-subtitle', 'report-regressions-subtitle', 'report-coverage-subtitle', 'report-vault-subtitle'];

        metricValueIds.forEach((id) => {
            const el = document.getElementById(id);
            if (el) {
                el.textContent = '--';
            }
        });

        metricSubtitleIds.forEach((id) => {
            const el = document.getElementById(id);
            if (el) {
                el.textContent = 'Loading…';
            }
        });

        const meta = document.getElementById('reports-window-meta');
        if (meta) {
            meta.textContent = '';
        }

        const containers = [
            document.getElementById('reports-trend-body'),
            document.getElementById('reports-scenarios'),
            document.getElementById('reports-insights')
        ];

        containers.forEach((container) => {
            if (!container) {
                return;
            }
            container.innerHTML = '<div class="loading"><div class="spinner"></div>Loading analytics…</div>';
        });
    }

    /**
     * Update reports window metadata display
     * @param {Object} overview - Overview data
     * @private
     */
    updateReportsWindowMeta(overview) {
        const meta = document.getElementById('reports-window-meta');
        if (!meta || !overview) {
            return;
        }

        const rangeLabel = formatDateRange(overview.window_start, overview.window_end);
        if (rangeLabel) {
            meta.textContent = `Window: ${rangeLabel}`;
        } else {
            meta.textContent = '';
        }
    }

    /**
     * Render reports overview metrics
     * @param {Object} overview - Overview data
     * @private
     */
    renderReportsOverview(overview) {
        if (!overview) {
            return;
        }

        const scenarios = Array.isArray(overview.scenarios) ? overview.scenarios : [];
        const global = overview.global || {};
        const vaults = overview.vaults || {};
        const scenarioCount = scenarios.length;
        const averageHealth = scenarioCount
            ? scenarios.reduce((sum, item) => sum + (Number(item.health_score) || 0), 0) / scenarioCount
            : 0;

        // Quality Index
        const qualityValue = document.getElementById('report-quality-index');
        if (qualityValue) {
            qualityValue.textContent = Math.round(averageHealth);
        }

        const qualitySubtitle = document.getElementById('report-quality-subtitle');
        if (qualitySubtitle) {
            qualitySubtitle.textContent = `${scenarioCount} scenario${scenarioCount === 1 ? '' : 's'} tracked`;
        }

        // Regressions
        const regressionsValue = document.getElementById('report-regressions');
        if (regressionsValue) {
            regressionsValue.textContent = Number(global.active_regressions || 0);
        }

        const regressionsSubtitle = document.getElementById('report-regressions-subtitle');
        if (regressionsSubtitle) {
            const running = Number(global.active_executions || 0);
            const runningLabel = running > 0 ? `${running} running execution${running === 1 ? '' : 's'}` : 'No active runs';
            regressionsSubtitle.textContent = runningLabel;
        }

        // Coverage
        const avgCoverage = Number(global.average_coverage || 0);
        const coverageTarget = 95;
        const coverageValue = document.getElementById('report-coverage');
        if (coverageValue) {
            coverageValue.textContent = formatPercent(avgCoverage, 1);
        }

        const coverageSubtitle = document.getElementById('report-coverage-subtitle');
        if (coverageSubtitle) {
            const delta = avgCoverage - coverageTarget;
            const deltaLabel = `${delta >= 0 ? '+' : '-'}${Math.abs(delta).toFixed(1)}% vs target`;
            coverageSubtitle.textContent = `Target ${coverageTarget}% (${deltaLabel})`;
        }

        // Vault
        const vaultValue = document.getElementById('report-vault');
        if (vaultValue) {
            const successRate = Number(vaults.success_rate || 0);
            vaultValue.textContent = formatPercent(successRate, 1);
        }

        const vaultSubtitle = document.getElementById('report-vault-subtitle');
        if (vaultSubtitle) {
            const totalExec = Number(vaults.total_executions || 0);
            const failedExec = Number(vaults.failed_executions || 0);
            vaultSubtitle.textContent = totalExec
                ? `${totalExec} execution${totalExec === 1 ? '' : 's'} • ${failedExec} failure${failedExec === 1 ? '' : 's'}`
                : 'No vault executions in window';
        }
    }

    /**
     * Render reports trends chart
     * @param {Object} trends - Trends data
     * @private
     */
    renderReportsTrends(trends) {
        const container = document.getElementById('reports-trend-body');
        if (!container) {
            return;
        }

        const series = trends && Array.isArray(trends.series) ? trends.series : [];
        if (series.length === 0) {
            container.innerHTML = '<div class="reports-empty">No trend data available yet. Execute test suites to build history.</div>';
            this.reportsTrendCanvas = null;
            return;
        }

        container.innerHTML = `
            <canvas id="reports-trend-canvas" height="220"></canvas>
            <div class="chart-legend">
                <span class="legend-item"><span class="legend-swatch pass"></span>Pass rate</span>
                <span class="legend-item"><span class="legend-swatch fail"></span>Failed executions</span>
            </div>
        `;

        this.reportsTrendCanvas = document.getElementById('reports-trend-canvas');
        if (!this.reportsTrendCanvas) {
            return;
        }

        this.drawReportsTrendChart(series);
    }

    /**
     * Draw reports trend chart on canvas
     * @param {Array} series - Time series data
     * @private
     */
    drawReportsTrendChart(series) {
        if (!this.reportsTrendCanvas) {
            return;
        }

        const devicePixelRatio = window.devicePixelRatio || 1;
        const displayWidth = this.reportsTrendCanvas.clientWidth || 600;
        const displayHeight = this.reportsTrendCanvas.getAttribute('height') ? Number(this.reportsTrendCanvas.getAttribute('height')) : 220;

        this.reportsTrendCanvas.width = Math.max(displayWidth * devicePixelRatio, 1);
        this.reportsTrendCanvas.height = Math.max(displayHeight * devicePixelRatio, 1);

        const ctx = this.reportsTrendCanvas.getContext('2d');
        if (!ctx) {
            return;
        }

        ctx.save();
        ctx.scale(devicePixelRatio, devicePixelRatio);

        const width = displayWidth;
        const height = displayHeight;
        ctx.clearRect(0, 0, width, height);

        const margin = { top: 20, right: 24, bottom: 32, left: 40 };
        const chartWidth = width - margin.left - margin.right;
        const chartHeight = height - margin.top - margin.bottom;

        if (chartWidth <= 0 || chartHeight <= 0) {
            ctx.restore();
            return;
        }

        const passRates = series.map((point) => {
            const passed = Number(point.passed_tests || 0);
            const failed = Number(point.failed_tests || 0);
            const total = passed + failed;
            if (total === 0) {
                return null;
            }
            return (passed / total) * 100;
        });

        const failureCounts = series.map((point) => Number(point.failed_executions || 0));
        const maxFailure = failureCounts.reduce((max, value) => Math.max(max, value), 0) || 1;

        const stepCount = series.length > 1 ? series.length - 1 : 1;
        const stepX = chartWidth / stepCount;
        const baseY = margin.top + chartHeight;
        const barMaxHeight = chartHeight * 0.35;
        const barWidth = Math.max(6, chartWidth / (series.length * 2));

        // Grid lines
        ctx.strokeStyle = 'rgba(255, 255, 255, 0.08)';
        ctx.lineWidth = 1;
        [0, 25, 50, 75, 100].forEach((pct) => {
            const y = margin.top + chartHeight - (pct / 100) * chartHeight;
            ctx.beginPath();
            ctx.moveTo(margin.left, y);
            ctx.lineTo(margin.left + chartWidth, y);
            ctx.stroke();

            ctx.fillStyle = 'rgba(255, 255, 255, 0.35)';
            ctx.font = '10px Inter, sans-serif';
            ctx.fillText(`${pct}%`, 4, y + 3);
        });

        // Failure bars
        ctx.fillStyle = 'rgba(255, 0, 64, 0.35)';
        series.forEach((point, index) => {
            const fail = failureCounts[index];
            if (!Number.isFinite(fail)) {
                return;
            }
            const x = margin.left + stepX * index;
            const barHeight = Math.min(barMaxHeight, (fail / maxFailure) * barMaxHeight);
            ctx.fillRect(x - barWidth / 2, baseY - barHeight, barWidth, barHeight);
        });

        // Pass rate line
        ctx.strokeStyle = '#00ff41';
        ctx.lineWidth = 2;
        ctx.beginPath();
        passRates.forEach((value, index) => {
            const x = margin.left + stepX * index;
            const y = value === null ? baseY : margin.top + chartHeight - (value / 100) * chartHeight;
            if (index === 0) {
                ctx.moveTo(x, y);
            } else {
                ctx.lineTo(x, y);
            }
        });
        ctx.stroke();

        // Pass rate markers
        ctx.fillStyle = '#00ff41';
        passRates.forEach((value, index) => {
            if (value === null) {
                return;
            }
            const x = margin.left + stepX * index;
            const y = margin.top + chartHeight - (value / 100) * chartHeight;
            ctx.beginPath();
            ctx.arc(x, y, 3, 0, Math.PI * 2, false);
            ctx.fill();
        });

        // X-axis labels
        ctx.fillStyle = 'rgba(255, 255, 255, 0.6)';
        ctx.font = '10px Inter, sans-serif';
        ctx.textAlign = 'center';

        // Calculate how many labels we can fit without overlap
        const labelWidth = 50; // Approximate width needed per label
        const availableWidth = chartWidth;
        const maxLabels = Math.floor(availableWidth / labelWidth);
        const labelInterval = Math.max(1, Math.ceil(series.length / maxLabels));

        series.forEach((point, index) => {
            // Only show every Nth label to prevent overlap
            if (index % labelInterval !== 0 && index !== series.length - 1) {
                return;
            }

            const bucket = point.bucket ? new Date(point.bucket) : null;
            if (!bucket || Number.isNaN(bucket.getTime())) {
                return;
            }
            const label = bucket.toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
            const x = margin.left + stepX * index;
            ctx.fillText(label, x, height - 8);
        });

        ctx.restore();
    }

    /**
     * Render reports scenario table
     * @param {Object} overview - Overview data
     * @private
     */
    renderReportsScenarioTable(overview) {
        const container = document.getElementById('reports-scenarios');
        if (!container) {
            return;
        }

        const scenarios = overview && Array.isArray(overview.scenarios) ? [...overview.scenarios] : [];
        if (scenarios.length === 0) {
            container.innerHTML = '<p class="reports-empty">No scenario analytics yet. Generate and execute suites to populate this view.</p>';
            return;
        }

        scenarios.sort((a, b) => {
            const scoreA = Number(a.health_score || 0);
            const scoreB = Number(b.health_score || 0);
            return scoreA - scoreB;
        });

        const rows = scenarios.map((scenario) => {
            const name = escapeHtml(scenario.scenario_name || 'Unknown');
            const healthScore = Math.round(Number(scenario.health_score || 0));
            const passRate = Number(scenario.pass_rate || 0);
            const coverage = Number(scenario.coverage || 0);
            const lastStatus = scenario.last_execution_status || 'unknown';

            const descriptor = getStatusDescriptor(lastStatus);
            const statusClass = getStatusClass(lastStatus);
            const statusLabel = descriptor ? descriptor.label : formatLabel(lastStatus || 'unknown');
            const statusIcon = descriptor?.icon ? `<i data-lucide="${descriptor.icon}"></i>` : '';

            return `
                <tr>
                    <td><strong>${name}</strong></td>
                    <td>${healthScore}</td>
                    <td>${formatPercent(passRate, 1)}</td>
                    <td>${formatPercent(coverage, 1)}</td>
                    <td><span class="status ${statusClass}">${statusIcon}${statusLabel}</span></td>
                </tr>
            `;
        }).join('');

        container.innerHTML = `
            <table class="table reports-scenario-table">
                <thead>
                    <tr>
                        <th>Scenario</th>
                        <th>Health</th>
                        <th>Pass Rate</th>
                        <th>Coverage</th>
                        <th>Last Status</th>
                    </tr>
                </thead>
                <tbody>
                    ${rows}
                </tbody>
            </table>
        `;

        refreshIcons();
    }

    /**
     * Render reports insights
     * @param {Object} insightsResponse - Insights data
     * @private
     */
    renderReportsInsights(insightsResponse) {
        const container = document.getElementById('reports-insights');
        if (!container) {
            return;
        }

        const insights = insightsResponse && Array.isArray(insightsResponse.insights)
            ? insightsResponse.insights
            : [];

        if (insights.length === 0) {
            container.innerHTML = '<p class="reports-empty">No insights generated for this window.</p>';
            return;
        }

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

        const items = insights.map((insight) => {
            const severityKey = (insight.severity || 'info').toLowerCase();
            const severityClass = severityClassMap[severityKey] || severityClassMap.info;
            const severityIcon = severityIconMap[severityKey] || severityIconMap.info;
            const severityLabel = formatLabel(insight.severity || 'info');
            const actions = Array.isArray(insight.actions) ? insight.actions : [];
            const scenarioBadge = insight.scenario_name
                ? `<span class="insight-scenario"><i data-lucide="box"></i>${escapeHtml(insight.scenario_name)}</span>`
                : '';

            const actionsHtml = actions.length
                ? `<ul class="insight-actions">${actions.map((action) => `<li>${escapeHtml(action)}</li>`).join('')}</ul>`
                : '';

            return `
                <article class="insight-card">
                    <header class="insight-header">
                        <div>
                            <h3 class="insight-title">${escapeHtml(insight.title || 'Insight')}</h3>
                            <div class="insight-meta">${scenarioBadge}</div>
                        </div>
                        <span class="insight-severity ${severityClass}"><i data-lucide="${severityIcon}"></i>${escapeHtml(severityLabel)}</span>
                    </header>
                    <p class="insight-detail">${escapeHtml(insight.detail || '')}</p>
                    ${actionsHtml}
                </article>
            `;
        }).join('');

        container.innerHTML = items;
    }

    /**
     * Render reports error state
     * @param {string} message - Error message
     * @private
     */
    renderReportsError(message) {
        const errorHtml = `<div class="reports-error">${escapeHtml(message)}</div>`;
        const trend = document.getElementById('reports-trend-body');
        const scenarios = document.getElementById('reports-scenarios');
        const insights = document.getElementById('reports-insights');

        if (trend) {
            trend.innerHTML = errorHtml;
        }
        if (scenarios) {
            scenarios.innerHTML = errorHtml;
        }
        if (insights) {
            insights.innerHTML = errorHtml;
        }

        const metricSubtitleIds = ['report-quality-subtitle', 'report-regressions-subtitle', 'report-coverage-subtitle', 'report-vault-subtitle'];
        metricSubtitleIds.forEach((id) => {
            const el = document.getElementById(id);
            if (el) {
                el.textContent = 'Error loading metrics';
            }
        });
    }

    /**
     * Handle window resize (redraw charts)
     * @private
     */
    handleWindowResize() {
        const reports = this.stateManager.get('data.reports');
        const trends = reports?.trends;
        if (!trends || !Array.isArray(trends.series) || trends.series.length === 0) {
            return;
        }

        this.renderReportsTrends(trends);
    }

    /**
     * Update reports time window
     * @param {number} days - Number of days
     */
    async updateReportsTimeWindow(days) {
        if (this.debug) {
            console.log('[ReportsPage] Updating time window to:', days);
        }

        this.reportsWindowDays = days;
        await this.load();
    }

    /**
     * Refresh reports data
     */
    async refresh() {
        await this.load();
        this.notificationManager.showSuccess('Reports refreshed!');
    }

    /**
     * Set debug mode
     * @param {boolean} enabled
     */
    setDebug(enabled) {
        this.debug = enabled;
        console.log(`[ReportsPage] Debug mode ${enabled ? 'enabled' : 'disabled'}`);
    }

    /**
     * Get debug information
     * @returns {Object}
     */
    getDebugInfo() {
        return {
            reportsWindowDays: this.reportsWindowDays,
            reportsIsLoading: this.reportsIsLoading,
            hasReportsTrendCanvas: this.reportsTrendCanvas !== null,
            hasReportsData: Boolean(this.stateManager.get('data.reports'))
        };
    }
}

// Export singleton instance
export const reportsPage = new ReportsPage();

// Export default for convenience
export default reportsPage;
