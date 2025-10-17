// Performance History Module for Agent Dashboard
// Tracks and visualizes agent performance metrics over time (session-only)

class PerformanceHistory {
    constructor() {
        // Store performance data for each agent (session-only)
        this.historyData = new Map();
        this.maxDataPoints = 30; // Keep last 30 data points
        this.updateInterval = 30000; // Update every 30 seconds
    }

    // Add a performance data point for an agent
    addDataPoint(agentId, metrics) {
        if (!this.historyData.has(agentId)) {
            this.historyData.set(agentId, {
                timestamps: [],
                cpu: [],
                memory: [],
                name: metrics.name || agentId,
                type: metrics.type || 'unknown'
            });
        }

        const history = this.historyData.get(agentId);
        const timestamp = new Date();

        // Add new data points
        history.timestamps.push(timestamp);
        history.cpu.push(metrics.cpu_percent || 0);
        history.memory.push(metrics.memory_mb || 0);

        // Keep only the last N data points
        if (history.timestamps.length > this.maxDataPoints) {
            history.timestamps.shift();
            history.cpu.shift();
            history.memory.shift();
        }
    }

    // Update history for all agents
    updateFromAgents(agents) {
        agents.forEach(agent => {
            if (agent.metrics) {
                this.addDataPoint(agent.id, {
                    ...agent.metrics,
                    name: agent.name,
                    type: agent.type
                });
            }
        });
    }

    // Get history data for an agent
    getHistory(agentId) {
        return this.historyData.get(agentId) || null;
    }

    // Create a mini sparkline chart for an agent card
    createSparkline(agentId, metric = 'cpu', width = 100, height = 30) {
        const history = this.getHistory(agentId);
        if (!history || history[metric].length === 0) {
            return '<div class="sparkline-empty">No data</div>';
        }

        const values = history[metric];
        const max = Math.max(...values) || 1;
        const min = Math.min(...values);
        const range = max - min || 1;

        // Create SVG sparkline
        const points = values.map((value, index) => {
            const x = (index / (values.length - 1)) * width;
            const y = height - ((value - min) / range) * height;
            return `${x},${y}`;
        }).join(' ');

        const color = metric === 'cpu' ? '#00ffff' : '#ff00ff';
        
        return `
            <svg class="sparkline" width="${width}" height="${height}" viewBox="0 0 ${width} ${height}">
                <polyline 
                    points="${points}" 
                    fill="none" 
                    stroke="${color}" 
                    stroke-width="1.5"
                    opacity="0.8"
                />
                <text x="${width - 25}" y="${height - 2}" 
                    fill="${color}" font-size="10" opacity="0.6">
                    ${values[values.length - 1].toFixed(1)}
                </text>
            </svg>
        `;
    }

    // Create a full performance chart for detailed view
    createPerformanceChart(agentId) {
        const history = this.getHistory(agentId);
        if (!history) {
            return '<div class="no-history">No performance history available</div>';
        }

        const chartWidth = 400;
        const chartHeight = 200;
        const padding = 40;
        const graphWidth = chartWidth - 2 * padding;
        const graphHeight = chartHeight - 2 * padding;

        // Format timestamps for x-axis labels
        const timeLabels = history.timestamps.map(ts => {
            const date = new Date(ts);
            return `${date.getHours()}:${date.getMinutes().toString().padStart(2, '0')}`;
        });

        // Create CPU line
        const maxCpu = Math.max(...history.cpu) || 1;
        const cpuPoints = history.cpu.map((value, index) => {
            const x = padding + (index / (history.cpu.length - 1)) * graphWidth;
            const y = padding + graphHeight - (value / maxCpu) * graphHeight;
            return `${x},${y}`;
        }).join(' ');

        // Create Memory line
        const maxMemory = Math.max(...history.memory) || 1;
        const memoryPoints = history.memory.map((value, index) => {
            const x = padding + (index / (history.memory.length - 1)) * graphWidth;
            const y = padding + graphHeight - (value / maxMemory) * graphHeight;
            return `${x},${y}`;
        }).join(' ');

        return `
            <div class="performance-chart">
                <div class="chart-header">
                    <h3>${history.name} Performance History</h3>
                    <div class="chart-legend">
                        <span class="legend-item cpu">● CPU %</span>
                        <span class="legend-item memory">● Memory MB</span>
                    </div>
                </div>
                <svg width="${chartWidth}" height="${chartHeight}" viewBox="0 0 ${chartWidth} ${chartHeight}">
                    <!-- Grid lines -->
                    ${this.createGridLines(chartWidth, chartHeight, padding)}
                    
                    <!-- CPU line -->
                    <polyline 
                        points="${cpuPoints}" 
                        fill="none" 
                        stroke="#00ffff" 
                        stroke-width="2"
                        opacity="0.8"
                    />
                    
                    <!-- Memory line -->
                    <polyline 
                        points="${memoryPoints}" 
                        fill="none" 
                        stroke="#ff00ff" 
                        stroke-width="2"
                        opacity="0.8"
                    />
                    
                    <!-- Axis labels -->
                    <text x="${padding}" y="${padding - 5}" fill="#888" font-size="10">
                        CPU: ${maxCpu.toFixed(1)}%
                    </text>
                    <text x="${chartWidth - padding - 50}" y="${padding - 5}" fill="#888" font-size="10">
                        Mem: ${maxMemory.toFixed(1)}MB
                    </text>
                    
                    <!-- Time labels -->
                    ${timeLabels.slice(-5).map((label, index) => {
                        const x = padding + (index / 4) * graphWidth;
                        return `<text x="${x}" y="${chartHeight - 5}" fill="#888" font-size="9" text-anchor="middle">${label}</text>`;
                    }).join('')}
                </svg>
            </div>
        `;
    }

    // Helper to create grid lines for charts
    createGridLines(width, height, padding) {
        const lines = [];
        const gridCount = 4;
        
        for (let i = 0; i <= gridCount; i++) {
            const y = padding + (i / gridCount) * (height - 2 * padding);
            lines.push(`<line x1="${padding}" y1="${y}" x2="${width - padding}" y2="${y}" stroke="#333" stroke-width="0.5" opacity="0.3"/>`);
        }
        
        return lines.join('');
    }

    // Clear history for an agent
    clearHistory(agentId) {
        this.historyData.delete(agentId);
    }

    // Clear all history
    clearAllHistory() {
        this.historyData.clear();
    }
}

// Export for use in main script
if (typeof module !== 'undefined' && module.exports) {
    module.exports = PerformanceHistory;
}