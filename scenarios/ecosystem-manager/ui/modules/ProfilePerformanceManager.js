// Profile Performance Manager Module
import { logger } from '../utils/logger.js';

/**
 * Manages Auto Steer profile performance history and analytics
 */
export class ProfilePerformanceManager {
    constructor(apiBase, showToast) {
        this.apiBase = apiBase;
        this.showToast = showToast;
        this.allExecutions = [];
        this.currentExecution = null;
        this.currentRating = 3;
        this.profiles = [];
    }

    /**
     * Initialize the performance manager
     */
    async initialize() {
        try {
            await this.loadProfilePerformanceHistory();
            await this.loadProfiles();
            this.populateFilters();
            logger.debug('Profile Performance Manager initialized');
        } catch (error) {
            logger.error('Failed to initialize Profile Performance Manager:', error);
            this.showToast('Failed to load profile performance data', 'error');
        }
    }

    /**
     * Load all profile performance history
     */
    async loadProfilePerformanceHistory() {
        try {
            const response = await fetch(`${this.apiBase}/auto-steer/history`);
            if (!response.ok) {
                throw new Error(`Failed to load history: ${response.statusText}`);
            }
            this.allExecutions = await response.json();
            logger.debug(`Loaded ${this.allExecutions.length} profile executions`);
            this.renderPerformanceList();
        } catch (error) {
            logger.error('Failed to load profile performance history:', error);
            const container = document.getElementById('profile-performance-list');
            if (container) {
                container.innerHTML = `
                    <div class="performance-empty">
                        <i class="fas fa-exclamation-triangle"></i>
                        <p>Failed to load performance data</p>
                        <p class="hint">${this.escapeHtml(error.message)}</p>
                    </div>
                `;
            }
        }
    }

    /**
     * Load profiles for filter dropdown
     */
    async loadProfiles() {
        try {
            const response = await fetch(`${this.apiBase}/auto-steer/profiles`);
            if (!response.ok) {
                throw new Error(`Failed to load profiles: ${response.statusText}`);
            }
            this.profiles = await response.json();
        } catch (error) {
            logger.error('Failed to load profiles:', error);
        }
    }

    /**
     * Populate filter dropdowns
     */
    populateFilters() {
        // Populate profile filter
        const profileFilter = document.getElementById('performance-profile-filter');
        if (profileFilter && this.profiles.length > 0) {
            const currentValue = profileFilter.value;
            profileFilter.innerHTML = '<option value="">All Profiles</option>' +
                this.profiles.map(p => `<option value="${this.escapeHtml(p.id)}">${this.escapeHtml(p.name)}</option>`).join('');
            profileFilter.value = currentValue;
        }

        // Populate scenario filter
        const scenarioFilter = document.getElementById('performance-scenario-filter');
        if (scenarioFilter && this.allExecutions.length > 0) {
            const scenarios = [...new Set(this.allExecutions.map(e => e.scenario_name))].sort();
            const currentValue = scenarioFilter.value;
            scenarioFilter.innerHTML = '<option value="">All Scenarios</option>' +
                scenarios.map(s => `<option value="${this.escapeHtml(s)}">${this.escapeHtml(s)}</option>`).join('');
            scenarioFilter.value = currentValue;
        }
    }

    /**
     * Filter performance history based on current filter values
     */
    filterPerformanceHistory() {
        const profileId = document.getElementById('performance-profile-filter')?.value || '';
        const scenario = document.getElementById('performance-scenario-filter')?.value || '';
        const startDate = document.getElementById('performance-start-date')?.value || '';
        const endDate = document.getElementById('performance-end-date')?.value || '';

        const filtered = this.allExecutions.filter(exec => {
            if (profileId && exec.profile_id !== profileId) return false;
            if (scenario && exec.scenario_name !== scenario) return false;
            if (startDate && new Date(exec.executed_at) < new Date(startDate)) return false;
            if (endDate && new Date(exec.executed_at) > new Date(endDate + 'T23:59:59')) return false;
            return true;
        });

        this.renderPerformanceList(filtered);
    }

    /**
     * Render the performance list table
     */
    renderPerformanceList(executions = this.allExecutions) {
        const container = document.getElementById('profile-performance-list');
        if (!container) return;

        if (executions.length === 0) {
            container.innerHTML = `
                <div class="performance-empty">
                    <i class="fas fa-chart-line"></i>
                    <p>No executions found</p>
                    <p class="hint">Try adjusting your filters</p>
                </div>
            `;
            return;
        }

        container.innerHTML = `
            <table class="performance-table">
                <thead>
                    <tr>
                        <th>Date</th>
                        <th>Profile</th>
                        <th>Scenario</th>
                        <th>Iterations</th>
                        <th>Duration</th>
                        <th>Rating</th>
                        <th>Improvement</th>
                    </tr>
                </thead>
                <tbody>
                    ${executions.map(exec => this.renderExecutionRow(exec)).join('')}
                </tbody>
            </table>
        `;
    }

    /**
     * Render a single execution row
     */
    renderExecutionRow(execution) {
        const profileName = this.profiles.find(p => p.id === execution.profile_id)?.name || 'Unknown';
        const date = new Date(execution.executed_at).toLocaleDateString();
        const duration = this.formatDuration(execution.total_duration);
        const improvement = (execution.end_metrics.operational_targets_percentage -
            execution.start_metrics.operational_targets_percentage).toFixed(1);
        const improvementClass = improvement > 0 ? 'positive' : improvement < 0 ? 'negative' : 'neutral';

        const rating = execution.user_feedback?.rating || null;
        const ratingHtml = rating
            ? `<span class="rating-display">${'★'.repeat(rating)}${'☆'.repeat(5 - rating)}</span>`
            : `<em class="text-muted">Not rated</em>`;

        return `
            <tr class="performance-row" onclick="ecosystemManager.profilePerformanceManager.showExecutionDetail('${this.escapeHtml(execution.execution_id)}')">
                <td>${this.escapeHtml(date)}</td>
                <td>${this.escapeHtml(profileName)}</td>
                <td>${this.escapeHtml(execution.scenario_name)}</td>
                <td>${execution.total_iterations}</td>
                <td>${this.escapeHtml(duration)}</td>
                <td>${ratingHtml}</td>
                <td class="improvement-${improvementClass}">
                    ${improvement > 0 ? '+' : ''}${improvement}%
                </td>
            </tr>
        `;
    }

    /**
     * Show execution detail view
     */
    async showExecutionDetail(executionId) {
        try {
            const response = await fetch(`${this.apiBase}/auto-steer/history/${executionId}`);
            if (!response.ok) {
                throw new Error(`Failed to load execution: ${response.statusText}`);
            }
            this.currentExecution = await response.json();

            // Hide list, show detail
            document.getElementById('profile-performance-list').style.display = 'none';
            document.getElementById('profile-execution-detail').style.display = 'block';

            // Render detail view
            this.renderExecutionDetail();
        } catch (error) {
            logger.error('Failed to load execution detail:', error);
            this.showToast('Failed to load execution details', 'error');
        }
    }

    /**
     * Close execution detail view
     */
    closeExecutionDetail() {
        document.getElementById('profile-execution-detail').style.display = 'none';
        document.getElementById('profile-performance-list').style.display = 'block';
        this.currentExecution = null;
    }

    /**
     * Render execution detail view
     */
    renderExecutionDetail() {
        if (!this.currentExecution) return;

        const exec = this.currentExecution;
        const profileName = this.profiles.find(p => p.id === exec.profile_id)?.name || 'Unknown';

        // Update title
        document.getElementById('profile-execution-title').textContent =
            `${exec.scenario_name} - ${profileName}`;

        // Show/hide rating button
        const ratingBtn = document.getElementById('rate-execution-btn');
        if (ratingBtn) {
            ratingBtn.style.display = exec.user_feedback ? 'none' : 'inline-flex';
        }

        // Render summary
        this.renderExecutionSummary(exec);

        // Render metrics comparison
        this.renderMetricsComparison(exec);

        // Render phase breakdown
        this.renderPhaseBreakdown(exec);

        // Render feedback if exists
        if (exec.user_feedback) {
            this.renderUserFeedback(exec.user_feedback);
        } else {
            document.getElementById('profile-execution-feedback').style.display = 'none';
        }
    }

    /**
     * Render execution summary stats
     */
    renderExecutionSummary(execution) {
        const container = document.getElementById('profile-execution-summary');
        if (!container) return;

        const date = new Date(execution.executed_at).toLocaleString();
        const duration = this.formatDuration(execution.total_duration);

        container.innerHTML = `
            <div class="summary-grid">
                <div class="summary-stat">
                    <div class="stat-label">Date</div>
                    <div class="stat-value">${this.escapeHtml(date)}</div>
                </div>
                <div class="summary-stat">
                    <div class="stat-label">Total Iterations</div>
                    <div class="stat-value">${execution.total_iterations}</div>
                </div>
                <div class="summary-stat">
                    <div class="stat-label">Duration</div>
                    <div class="stat-value">${this.escapeHtml(duration)}</div>
                </div>
                <div class="summary-stat">
                    <div class="stat-label">Phases Completed</div>
                    <div class="stat-value">${execution.phase_breakdown.length}</div>
                </div>
            </div>
        `;
    }

    /**
     * Render metrics comparison table
     */
    renderMetricsComparison(execution) {
        const container = document.getElementById('profile-metrics-comparison');
        if (!container) return;

        const { start_metrics, end_metrics } = execution;
        const metrics = [
            { name: 'Operational Targets %', key: 'operational_targets_percentage', format: v => v.toFixed(1) + '%' },
            { name: 'Operational Targets Passing', key: 'operational_targets_passing' },
            { name: 'Build Status', key: 'build_status', format: v => v === 1 ? 'Pass' : 'Fail' },
        ];

        // Add UX metrics if available
        if (end_metrics.ux) {
            metrics.push({ name: 'Accessibility Score', key: 'ux.accessibility_score', format: v => v.toFixed(1) });
            metrics.push({ name: 'UI Test Coverage', key: 'ux.ui_test_coverage', format: v => v.toFixed(1) + '%' });
        }

        // Add Refactor metrics if available
        if (end_metrics.refactor) {
            metrics.push({ name: 'Tidiness Score', key: 'refactor.tidiness_score', format: v => v.toFixed(1) });
            metrics.push({ name: 'Complexity Avg', key: 'refactor.cyclomatic_complexity_avg', format: v => v.toFixed(1) });
        }

        container.innerHTML = `
            <table class="metrics-table">
                <thead>
                    <tr>
                        <th>Metric</th>
                        <th>Start</th>
                        <th>End</th>
                        <th>Change</th>
                    </tr>
                </thead>
                <tbody>
                    ${metrics.map(m => {
                        const startVal = this.getNestedValue(start_metrics, m.key);
                        const endVal = this.getNestedValue(end_metrics, m.key);
                        if (startVal === undefined || endVal === undefined) return '';

                        const delta = endVal - startVal;
                        const deltaClass = delta > 0 ? 'positive' : delta < 0 ? 'negative' : 'neutral';
                        const format = m.format || (v => v);

                        return `
                            <tr>
                                <td>${this.escapeHtml(m.name)}</td>
                                <td>${this.escapeHtml(format(startVal))}</td>
                                <td>${this.escapeHtml(format(endVal))}</td>
                                <td class="metric-delta-${deltaClass}">
                                    ${delta > 0 ? '+' : ''}${this.escapeHtml(format(delta))}
                                </td>
                            </tr>
                        `;
                    }).join('')}
                </tbody>
            </table>
        `;
    }

    /**
     * Render phase breakdown
     */
    renderPhaseBreakdown(execution) {
        const container = document.getElementById('profile-phase-breakdown');
        if (!container) return;

        const phases = execution.phase_breakdown;

        container.innerHTML = `
            <table class="phase-breakdown-table">
                <thead>
                    <tr>
                        <th>Phase</th>
                        <th>Mode</th>
                        <th>Iterations</th>
                        <th>Duration</th>
                        <th>Key Improvements</th>
                        <th>Effectiveness</th>
                    </tr>
                </thead>
                <tbody>
                    ${phases.map((phase, idx) => {
                        const duration = this.formatDuration(phase.duration);
                        const effectiveness = (phase.effectiveness * 100).toFixed(1);
                        const improvementsList = Object.entries(phase.metric_deltas || {})
                            .filter(([_, delta]) => Math.abs(delta) > 0.1)
                            .map(([metric, delta]) => {
                                const sign = delta > 0 ? '+' : '';
                                return `${metric}: ${sign}${delta.toFixed(1)}`;
                            })
                            .join(', ');

                        return `
                            <tr>
                                <td>${idx + 1}</td>
                                <td>
                                    <span class="phase-mode-badge phase-mode-${phase.mode}">${this.escapeHtml(phase.mode)}</span>
                                </td>
                                <td>${phase.iterations}</td>
                                <td>${this.escapeHtml(duration)}</td>
                                <td class="improvements-cell">${improvementsList || '-'}</td>
                                <td>
                                    <div class="effectiveness-bar">
                                        <div class="effectiveness-fill" style="width: ${effectiveness}%"></div>
                                        <span class="effectiveness-value">${effectiveness}%</span>
                                    </div>
                                </td>
                            </tr>
                        `;
                    }).join('')}
                </tbody>
            </table>
        `;
    }

    /**
     * Render user feedback section
     */
    renderUserFeedback(feedback) {
        const container = document.getElementById('profile-feedback-content');
        if (!container) return;

        const date = new Date(feedback.submitted_at).toLocaleString();
        const stars = '★'.repeat(feedback.rating) + '☆'.repeat(5 - feedback.rating);

        container.innerHTML = `
            <div class="feedback-display">
                <div class="feedback-rating">${stars}</div>
                <div class="feedback-comments">${this.escapeHtml(feedback.comments) || '<em>No comments</em>'}</div>
                <div class="feedback-date"><small>Submitted ${this.escapeHtml(date)}</small></div>
            </div>
        `;

        document.getElementById('profile-execution-feedback').style.display = 'block';
    }

    /**
     * Open feedback dialog
     */
    openFeedbackDialog() {
        this.currentRating = 3;
        this.updateRatingStars(3);
        document.getElementById('feedback-comments').value = '';
        document.getElementById('feedback-dialog-modal').style.display = 'flex';
    }

    /**
     * Close feedback dialog
     */
    closeFeedbackDialog() {
        document.getElementById('feedback-dialog-modal').style.display = 'none';
    }

    /**
     * Set rating value
     */
    setRating(rating) {
        this.currentRating = rating;
        this.updateRatingStars(rating);
    }

    /**
     * Update rating stars display
     */
    updateRatingStars(rating) {
        const stars = document.querySelectorAll('#feedback-rating i');
        stars.forEach((star, idx) => {
            if (idx < rating) {
                star.classList.remove('far');
                star.classList.add('fas');
            } else {
                star.classList.remove('fas');
                star.classList.add('far');
            }
        });
    }

    /**
     * Submit feedback
     */
    async submitFeedback() {
        if (!this.currentExecution) return;

        const comments = document.getElementById('feedback-comments').value.trim();

        try {
            const response = await fetch(
                `${this.apiBase}/auto-steer/history/${this.currentExecution.execution_id}/feedback`,
                {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        rating: this.currentRating,
                        comments: comments
                    })
                }
            );

            if (!response.ok) {
                throw new Error(`Failed to submit feedback: ${response.statusText}`);
            }

            this.showToast('Feedback submitted successfully', 'success');
            this.closeFeedbackDialog();

            // Update current execution with feedback
            this.currentExecution.user_feedback = {
                rating: this.currentRating,
                comments: comments,
                submitted_at: new Date().toISOString()
            };

            // Re-render detail view
            this.renderExecutionDetail();

            // Refresh list
            await this.loadProfilePerformanceHistory();
        } catch (error) {
            logger.error('Failed to submit feedback:', error);
            this.showToast('Failed to submit feedback: ' + error.message, 'error');
        }
    }

    /**
     * Refresh performance data
     */
    async refreshPerformance() {
        await this.initialize();
    }

    /**
     * Format duration in milliseconds to human-readable string
     */
    formatDuration(ms) {
        const seconds = Math.floor(ms / 1000);
        const minutes = Math.floor(seconds / 60);
        const hours = Math.floor(minutes / 60);

        if (hours > 0) {
            return `${hours}h ${minutes % 60}m`;
        } else if (minutes > 0) {
            return `${minutes}m ${seconds % 60}s`;
        } else {
            return `${seconds}s`;
        }
    }

    /**
     * Get nested value from object using dot notation
     */
    getNestedValue(obj, path) {
        return path.split('.').reduce((current, key) => current?.[key], obj);
    }

    /**
     * Escape HTML to prevent XSS
     */
    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
}
