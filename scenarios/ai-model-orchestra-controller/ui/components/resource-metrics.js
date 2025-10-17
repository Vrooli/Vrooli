/**
 * Resource Metrics Component
 * Displays system resource usage, memory pressure, and performance metrics
 */

import apiClient from '../services/api-client.js';
import formatters from '../utils/formatters.js';

export class ResourceMetricsComponent {
    constructor(containerId) {
        this.container = document.getElementById(containerId);
        this.refreshInterval = 5000; // 5 seconds for real-time feel
        this.intervalId = null;
        this.metrics = null;
        this.history = [];
        this.maxHistoryPoints = 50;
    }

    /**
     * Initialize and start the component
     */
    async init() {
        this.render();
        await this.loadMetrics();
        this.startAutoRefresh();
    }

    /**
     * Load metrics from API
     */
    async loadMetrics() {
        try {
            const response = await apiClient.getResourceMetrics(1, true);
            if (response.success) {
                this.metrics = response.data.current;
                this.updateHistory(response.data.history || []);
                this.updateDisplay();
            } else {
                this.showError('Failed to load resource metrics');
            }
        } catch (error) {
            console.error('Error loading metrics:', error);
            this.showError('Error connecting to API');
        }
    }

    /**
     * Render the component structure
     */
    render() {
        this.container.innerHTML = `
            <div class="resource-metrics-component">
                <div class="component-header">
                    <h2 class="text-xl font-semibold">System Resources</h2>
                    <div class="header-actions">
                        <button id="refresh-metrics" class="btn-icon" title="Refresh">
                            🔄
                        </button>
                        <span class="update-time" id="metrics-update-time">
                            Updated: Never
                        </span>
                    </div>
                </div>
                
                <div class="metrics-grid">
                    <div class="metric-card" id="memory-pressure-card">
                        <h3>Memory Pressure</h3>
                        <div class="metric-value" id="memory-pressure-value">--</div>
                        <div class="metric-chart" id="memory-pressure-chart"></div>
                    </div>
                    
                    <div class="metric-card" id="memory-usage-card">
                        <h3>Memory Usage</h3>
                        <div class="metric-details">
                            <div class="detail-row">
                                <span>Available:</span>
                                <span id="memory-available">--</span>
                            </div>
                            <div class="detail-row">
                                <span>Free:</span>
                                <span id="memory-free">--</span>
                            </div>
                            <div class="detail-row">
                                <span>Total:</span>
                                <span id="memory-total">--</span>
                            </div>
                        </div>
                        <div class="progress-bar">
                            <div class="progress-fill" id="memory-progress"></div>
                        </div>
                    </div>
                    
                    <div class="metric-card" id="cpu-usage-card">
                        <h3>CPU Usage</h3>
                        <div class="metric-value" id="cpu-usage-value">--</div>
                        <div class="metric-chart" id="cpu-usage-chart"></div>
                    </div>
                    
                    <div class="metric-card" id="request-metrics-card">
                        <h3>Request Metrics</h3>
                        <div class="metric-details">
                            <div class="detail-row">
                                <span>Total Requests:</span>
                                <span id="total-requests">--</span>
                            </div>
                            <div class="detail-row">
                                <span>Success Rate:</span>
                                <span id="success-rate">--</span>
                            </div>
                            <div class="detail-row">
                                <span>Avg Response:</span>
                                <span id="avg-response">--</span>
                            </div>
                        </div>
                    </div>
                </div>
                
                <div id="metrics-error" class="error-message hidden"></div>
                
                <div class="metrics-footer">
                    <div class="legend">
                        <span class="legend-item">
                            <span class="dot bg-green-500"></span> Normal
                        </span>
                        <span class="legend-item">
                            <span class="dot bg-yellow-500"></span> Warning
                        </span>
                        <span class="legend-item">
                            <span class="dot bg-red-500"></span> Critical
                        </span>
                    </div>
                </div>
            </div>
        `;

        this.attachEventListeners();
    }

    /**
     * Update the display with metrics data
     */
    updateDisplay() {
        if (!this.metrics) return;

        // Update memory pressure
        const pressureLevel = formatters.getMemoryPressureLevel(this.metrics.memory_pressure || 0);
        const pressureElement = document.getElementById('memory-pressure-value');
        pressureElement.textContent = formatters.formatPercent((this.metrics.memory_pressure || 0) * 100);
        pressureElement.className = `metric-value ${pressureLevel.color}`;
        
        // Update memory card background based on pressure
        const memoryCard = document.getElementById('memory-pressure-card');
        memoryCard.className = `metric-card ${this.getPressureCardClass(this.metrics.memory_pressure)}`;

        // Update memory usage
        document.getElementById('memory-available').textContent = 
            formatters.formatMemoryGB(this.metrics.memory_available_gb || 0);
        document.getElementById('memory-free').textContent = 
            formatters.formatMemoryGB(this.metrics.memory_free_gb || 0);
        document.getElementById('memory-total').textContent = 
            formatters.formatMemoryGB(this.metrics.memory_total_gb || 0);

        // Update memory progress bar
        const memoryUsedPercent = ((this.metrics.memory_total_gb - this.metrics.memory_available_gb) / 
                                   this.metrics.memory_total_gb) * 100;
        const memoryProgress = document.getElementById('memory-progress');
        memoryProgress.style.width = `${memoryUsedPercent}%`;
        memoryProgress.className = `progress-fill ${this.getProgressBarClass(memoryUsedPercent)}`;

        // Update CPU usage
        const cpuValue = document.getElementById('cpu-usage-value');
        cpuValue.textContent = formatters.formatPercent(this.metrics.cpu_usage_percent || 0);
        cpuValue.className = `metric-value ${this.getCPUValueClass(this.metrics.cpu_usage_percent)}`;

        // Update request metrics if available
        if (this.metrics.requests) {
            document.getElementById('total-requests').textContent = 
                this.metrics.requests.total || '0';
            document.getElementById('success-rate').textContent = 
                formatters.formatPercent(this.metrics.requests.success_rate || 0);
            document.getElementById('avg-response').textContent = 
                formatters.formatResponseTime(this.metrics.requests.avg_response_ms || 0);
        }

        // Update timestamp
        document.getElementById('metrics-update-time').textContent = 
            `Updated: ${new Date().toLocaleTimeString()}`;

        // Update charts
        this.updateCharts();
    }

    /**
     * Update history data
     */
    updateHistory(newHistory) {
        this.history = newHistory.slice(-this.maxHistoryPoints);
        if (this.metrics) {
            this.history.push({
                timestamp: new Date(),
                memory_pressure: this.metrics.memory_pressure,
                cpu_usage_percent: this.metrics.cpu_usage_percent
            });
        }
        if (this.history.length > this.maxHistoryPoints) {
            this.history = this.history.slice(-this.maxHistoryPoints);
        }
    }

    /**
     * Update mini charts (simplified visualization)
     */
    updateCharts() {
        // Memory pressure mini chart
        this.updateMiniChart('memory-pressure-chart', 
            this.history.map(h => h.memory_pressure * 100));
        
        // CPU usage mini chart
        this.updateMiniChart('cpu-usage-chart', 
            this.history.map(h => h.cpu_usage_percent));
    }

    /**
     * Update a mini chart (simplified sparkline)
     */
    updateMiniChart(elementId, data) {
        const element = document.getElementById(elementId);
        if (!element || data.length === 0) return;

        const max = Math.max(...data, 100);
        const width = 150;
        const height = 40;
        
        const points = data.map((value, index) => {
            const x = (index / (data.length - 1)) * width;
            const y = height - ((value / max) * height);
            return `${x},${y}`;
        }).join(' ');

        element.innerHTML = `
            <svg width="${width}" height="${height}" class="mini-chart">
                <polyline points="${points}" 
                    fill="none" 
                    stroke="currentColor" 
                    stroke-width="2"/>
            </svg>
        `;
    }

    /**
     * Get card class based on pressure
     */
    getPressureCardClass(pressure) {
        if (pressure < 0.5) return 'metric-card bg-green-50';
        if (pressure < 0.7) return 'metric-card bg-yellow-50';
        if (pressure < 0.9) return 'metric-card bg-orange-50';
        return 'metric-card bg-red-50';
    }

    /**
     * Get progress bar class based on percentage
     */
    getProgressBarClass(percent) {
        if (percent < 50) return 'bg-green-500';
        if (percent < 70) return 'bg-yellow-500';
        if (percent < 90) return 'bg-orange-500';
        return 'bg-red-500';
    }

    /**
     * Get CPU value class
     */
    getCPUValueClass(cpu) {
        if (cpu < 50) return 'text-green-500';
        if (cpu < 70) return 'text-yellow-500';
        if (cpu < 90) return 'text-orange-500';
        return 'text-red-500';
    }

    /**
     * Show error message
     */
    showError(message) {
        const errorDiv = document.getElementById('metrics-error');
        errorDiv.textContent = message;
        errorDiv.classList.remove('hidden');
        setTimeout(() => errorDiv.classList.add('hidden'), 5000);
    }

    /**
     * Attach event listeners
     */
    attachEventListeners() {
        const refreshBtn = document.getElementById('refresh-metrics');
        if (refreshBtn) {
            refreshBtn.addEventListener('click', () => this.loadMetrics());
        }
    }

    /**
     * Start auto-refresh
     */
    startAutoRefresh() {
        this.intervalId = setInterval(() => this.loadMetrics(), this.refreshInterval);
    }

    /**
     * Stop auto-refresh
     */
    stopAutoRefresh() {
        if (this.intervalId) {
            clearInterval(this.intervalId);
            this.intervalId = null;
        }
    }

    /**
     * Cleanup on destroy
     */
    destroy() {
        this.stopAutoRefresh();
        this.container.innerHTML = '';
    }
}

export default ResourceMetricsComponent;