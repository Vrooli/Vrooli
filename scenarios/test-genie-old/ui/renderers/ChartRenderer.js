/**
 * ChartRenderer - Custom chart rendering for Test Genie
 *
 * Provides custom Canvas 2D chart rendering for:
 * - Trends charts (pass rate line + failure bars)
 * - Time series visualization
 * - Performance metrics charts
 * - High DPI display support
 * - Responsive resizing
 */

/**
 * ChartRenderer class - Handles all chart rendering logic
 */
export class ChartRenderer {
    constructor() {
        this.debug = false;
        this.devicePixelRatio = window.devicePixelRatio || 1;
    }

    /**
     * Render trends chart with pass rate line and failure bars
     * @param {HTMLCanvasElement} canvas - Canvas element
     * @param {Array} series - Time series data
     * @param {Object} options - Rendering options
     * @param {number} options.height - Chart height (default: 220)
     * @param {Object} options.margin - Chart margins (default: {top: 20, right: 24, bottom: 32, left: 40})
     * @param {string} options.passLineColor - Pass rate line color (default: '#00ff41')
     * @param {string} options.failBarColor - Failure bar color (default: 'rgba(255, 0, 64, 0.35)')
     * @param {string} options.gridColor - Grid line color (default: 'rgba(255, 255, 255, 0.08)')
     */
    renderTrendsChart(canvas, series, options = {}) {
        if (!canvas || !series || series.length === 0) {
            return;
        }

        const {
            height = 220,
            margin = { top: 20, right: 24, bottom: 32, left: 40 },
            passLineColor = '#00ff41',
            failBarColor = 'rgba(255, 0, 64, 0.35)',
            gridColor = 'rgba(255, 255, 255, 0.08)'
        } = options;

        // Setup canvas for high DPI displays
        const displayWidth = canvas.clientWidth || 600;
        const displayHeight = height;

        canvas.width = Math.max(displayWidth * this.devicePixelRatio, 1);
        canvas.height = Math.max(displayHeight * this.devicePixelRatio, 1);

        const ctx = canvas.getContext('2d');
        if (!ctx) {
            return;
        }

        ctx.save();
        ctx.scale(this.devicePixelRatio, this.devicePixelRatio);

        const width = displayWidth;
        ctx.clearRect(0, 0, width, displayHeight);

        const chartWidth = width - margin.left - margin.right;
        const chartHeight = displayHeight - margin.top - margin.bottom;

        if (chartWidth <= 0 || chartHeight <= 0) {
            ctx.restore();
            return;
        }

        // Calculate pass rates and failure counts
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

        // Draw grid lines
        this.drawGrid(ctx, margin, chartWidth, chartHeight, gridColor);

        // Draw failure bars
        this.drawFailureBars(ctx, series, failureCounts, maxFailure, {
            margin,
            stepX,
            baseY,
            barMaxHeight,
            barWidth,
            color: failBarColor
        });

        // Draw pass rate line
        this.drawPassRateLine(ctx, passRates, {
            margin,
            chartHeight,
            stepX,
            color: passLineColor
        });

        // Draw pass rate markers
        this.drawPassRateMarkers(ctx, passRates, {
            margin,
            chartHeight,
            stepX,
            color: passLineColor
        });

        // Draw X-axis labels
        this.drawXAxisLabels(ctx, series, {
            margin,
            stepX,
            height: displayHeight
        });

        ctx.restore();
    }

    /**
     * Draw grid lines
     * @private
     */
    drawGrid(ctx, margin, chartWidth, chartHeight, gridColor) {
        ctx.strokeStyle = gridColor;
        ctx.lineWidth = 1;

        [0, 25, 50, 75, 100].forEach((pct) => {
            const y = margin.top + chartHeight - (pct / 100) * chartHeight;

            // Grid line
            ctx.beginPath();
            ctx.moveTo(margin.left, y);
            ctx.lineTo(margin.left + chartWidth, y);
            ctx.stroke();

            // Y-axis label
            ctx.fillStyle = 'rgba(255, 255, 255, 0.35)';
            ctx.font = '10px Inter, sans-serif';
            ctx.fillText(`${pct}%`, 4, y + 3);
        });
    }

    /**
     * Draw failure bars
     * @private
     */
    drawFailureBars(ctx, series, failureCounts, maxFailure, options) {
        const { margin, stepX, baseY, barMaxHeight, barWidth, color } = options;

        ctx.fillStyle = color;

        series.forEach((point, index) => {
            const fail = failureCounts[index];
            if (!Number.isFinite(fail)) {
                return;
            }

            const x = margin.left + stepX * index;
            const barHeight = Math.min(barMaxHeight, (fail / maxFailure) * barMaxHeight);
            ctx.fillRect(x - barWidth / 2, baseY - barHeight, barWidth, barHeight);
        });
    }

    /**
     * Draw pass rate line
     * @private
     */
    drawPassRateLine(ctx, passRates, options) {
        const { margin, chartHeight, stepX, color } = options;

        ctx.strokeStyle = color;
        ctx.lineWidth = 2;
        ctx.beginPath();

        passRates.forEach((value, index) => {
            const x = margin.left + stepX * index;
            const baseY = margin.top + chartHeight;
            const y = value === null ? baseY : margin.top + chartHeight - (value / 100) * chartHeight;

            if (index === 0) {
                ctx.moveTo(x, y);
            } else {
                ctx.lineTo(x, y);
            }
        });

        ctx.stroke();
    }

    /**
     * Draw pass rate markers
     * @private
     */
    drawPassRateMarkers(ctx, passRates, options) {
        const { margin, chartHeight, stepX, color } = options;

        ctx.fillStyle = color;

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
    }

    /**
     * Draw X-axis labels
     * @private
     */
    drawXAxisLabels(ctx, series, options) {
        const { margin, stepX, height } = options;

        ctx.fillStyle = 'rgba(255, 255, 255, 0.6)';
        ctx.font = '10px Inter, sans-serif';

        series.forEach((point, index) => {
            const bucket = point.bucket ? new Date(point.bucket) : null;
            if (!bucket || Number.isNaN(bucket.getTime())) {
                return;
            }

            const label = bucket.toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
            const x = margin.left + stepX * index;
            ctx.fillText(label, x - 18, height - 8);
        });
    }

    /**
     * Render simple line chart
     * @param {HTMLCanvasElement} canvas - Canvas element
     * @param {Array} data - Data points [{x, y}]
     * @param {Object} options - Rendering options
     */
    renderLineChart(canvas, data, options = {}) {
        if (!canvas || !data || data.length === 0) {
            return;
        }

        const {
            height = 200,
            margin = { top: 20, right: 20, bottom: 30, left: 40 },
            lineColor = '#00ff41',
            gridColor = 'rgba(255, 255, 255, 0.08)',
            showMarkers = true
        } = options;

        const displayWidth = canvas.clientWidth || 600;
        const displayHeight = height;

        canvas.width = Math.max(displayWidth * this.devicePixelRatio, 1);
        canvas.height = Math.max(displayHeight * this.devicePixelRatio, 1);

        const ctx = canvas.getContext('2d');
        if (!ctx) {
            return;
        }

        ctx.save();
        ctx.scale(this.devicePixelRatio, this.devicePixelRatio);
        ctx.clearRect(0, 0, displayWidth, displayHeight);

        const chartWidth = displayWidth - margin.left - margin.right;
        const chartHeight = displayHeight - margin.top - margin.bottom;

        if (chartWidth <= 0 || chartHeight <= 0) {
            ctx.restore();
            return;
        }

        // Find data range
        const yValues = data.map(d => d.y);
        const minY = Math.min(...yValues);
        const maxY = Math.max(...yValues);
        const yRange = maxY - minY || 1;

        const stepX = chartWidth / (data.length - 1 || 1);

        // Draw grid
        ctx.strokeStyle = gridColor;
        ctx.lineWidth = 1;
        for (let i = 0; i <= 4; i++) {
            const y = margin.top + (chartHeight / 4) * i;
            ctx.beginPath();
            ctx.moveTo(margin.left, y);
            ctx.lineTo(margin.left + chartWidth, y);
            ctx.stroke();
        }

        // Draw line
        ctx.strokeStyle = lineColor;
        ctx.lineWidth = 2;
        ctx.beginPath();

        data.forEach((point, index) => {
            const x = margin.left + stepX * index;
            const y = margin.top + chartHeight - ((point.y - minY) / yRange) * chartHeight;

            if (index === 0) {
                ctx.moveTo(x, y);
            } else {
                ctx.lineTo(x, y);
            }
        });

        ctx.stroke();

        // Draw markers
        if (showMarkers) {
            ctx.fillStyle = lineColor;
            data.forEach((point, index) => {
                const x = margin.left + stepX * index;
                const y = margin.top + chartHeight - ((point.y - minY) / yRange) * chartHeight;

                ctx.beginPath();
                ctx.arc(x, y, 3, 0, Math.PI * 2);
                ctx.fill();
            });
        }

        ctx.restore();
    }

    /**
     * Render bar chart
     * @param {HTMLCanvasElement} canvas - Canvas element
     * @param {Array} data - Data points [{label, value}]
     * @param {Object} options - Rendering options
     */
    renderBarChart(canvas, data, options = {}) {
        if (!canvas || !data || data.length === 0) {
            return;
        }

        const {
            height = 200,
            margin = { top: 20, right: 20, bottom: 40, left: 40 },
            barColor = '#00ff41',
            gridColor = 'rgba(255, 255, 255, 0.08)'
        } = options;

        const displayWidth = canvas.clientWidth || 600;
        const displayHeight = height;

        canvas.width = Math.max(displayWidth * this.devicePixelRatio, 1);
        canvas.height = Math.max(displayHeight * this.devicePixelRatio, 1);

        const ctx = canvas.getContext('2d');
        if (!ctx) {
            return;
        }

        ctx.save();
        ctx.scale(this.devicePixelRatio, this.devicePixelRatio);
        ctx.clearRect(0, 0, displayWidth, displayHeight);

        const chartWidth = displayWidth - margin.left - margin.right;
        const chartHeight = displayHeight - margin.top - margin.bottom;

        if (chartWidth <= 0 || chartHeight <= 0) {
            ctx.restore();
            return;
        }

        const maxValue = Math.max(...data.map(d => d.value)) || 1;
        const barWidth = Math.max(10, chartWidth / data.length - 10);
        const stepX = chartWidth / data.length;

        // Draw grid
        ctx.strokeStyle = gridColor;
        ctx.lineWidth = 1;
        for (let i = 0; i <= 4; i++) {
            const y = margin.top + (chartHeight / 4) * i;
            ctx.beginPath();
            ctx.moveTo(margin.left, y);
            ctx.lineTo(margin.left + chartWidth, y);
            ctx.stroke();
        }

        // Draw bars
        ctx.fillStyle = barColor;
        data.forEach((point, index) => {
            const x = margin.left + stepX * index + stepX / 2 - barWidth / 2;
            const barHeight = (point.value / maxValue) * chartHeight;
            const y = margin.top + chartHeight - barHeight;

            ctx.fillRect(x, y, barWidth, barHeight);

            // Draw label
            if (point.label) {
                ctx.fillStyle = 'rgba(255, 255, 255, 0.6)';
                ctx.font = '10px Inter, sans-serif';
                ctx.save();
                ctx.translate(x + barWidth / 2, displayHeight - 10);
                ctx.rotate(-Math.PI / 4);
                ctx.fillText(point.label, 0, 0);
                ctx.restore();
                ctx.fillStyle = barColor;
            }
        });

        ctx.restore();
    }

    /**
     * Clear canvas
     * @param {HTMLCanvasElement} canvas - Canvas element
     */
    clearCanvas(canvas) {
        if (!canvas) {
            return;
        }

        const ctx = canvas.getContext('2d');
        if (!ctx) {
            return;
        }

        ctx.clearRect(0, 0, canvas.width, canvas.height);
    }

    /**
     * Set debug mode
     * @param {boolean} enabled
     */
    setDebug(enabled) {
        this.debug = enabled;
        console.log(`[ChartRenderer] Debug mode ${enabled ? 'enabled' : 'disabled'}`);
    }
}

// Export singleton instance
export const chartRenderer = new ChartRenderer();

// Export default for convenience
export default chartRenderer;
