import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child';

const BRIDGE_FLAG = '__chartGeneratorBridgeInitialized';

function bootstrapIframeBridge() {
    if (typeof window === 'undefined' || window[BRIDGE_FLAG]) {
        return;
    }

    if (window.parent !== window) {
        let parentOrigin;
        try {
            if (document.referrer) {
                parentOrigin = new URL(document.referrer).origin;
            }
        } catch (error) {
            console.warn('[ChartGenerator] Unable to parse parent origin for iframe bridge', error);
        }

        initIframeBridgeChild({ parentOrigin, appId: 'chart-generator' });
        window[BRIDGE_FLAG] = true;
    }
}

bootstrapIframeBridge();

/**
 * Chart Generator Application
 * Handles UI interactions and chart generation
 */

class ChartGeneratorApp {
    constructor() {
        this.chartEngine = new ChartEngine();
        this.currentChartType = 'bar';
        this.currentStyle = 'professional';
        this.currentData = null;
        this.previewContainer = document.getElementById('chart-preview');
        
        this.init();
    }

    init() {
        this.setupEventListeners();
        this.setupHealthEndpoint();
        this.loadSampleData('sales');
        this.updateUI();
    }

    /**
     * Set up all event listeners
     */
    setupEventListeners() {
        // Navigation tabs
        document.querySelectorAll('.nav-tab').forEach(tab => {
            tab.addEventListener('click', (e) => {
                this.switchTab(e.target.dataset.tab);
            });
        });

        // Chart type selection
        document.querySelectorAll('.chart-type-card').forEach(card => {
            card.addEventListener('click', (e) => {
                this.selectChartType(card.dataset.type);
            });
        });

        // Style selection
        document.querySelectorAll('.style-card').forEach(card => {
            card.addEventListener('click', (e) => {
                this.selectStyle(card.dataset.style);
            });
        });

        // Sample data buttons
        document.querySelectorAll('[data-sample]').forEach(btn => {
            btn.addEventListener('click', (e) => {
                this.loadSampleData(btn.dataset.sample);
            });
        });

        // Data editor
        const dataInput = document.getElementById('data-input');
        dataInput.addEventListener('input', (e) => {
            this.validateAndUpdateData(e.target.value);
        });

        // Action buttons
        document.getElementById('validate-data-btn').addEventListener('click', () => {
            this.validateCurrentData();
        });

        document.getElementById('clear-data-btn').addEventListener('click', () => {
            this.clearData();
        });

        document.getElementById('refresh-preview-btn').addEventListener('click', () => {
            this.refreshPreview();
        });

        document.getElementById('generate-chart-btn').addEventListener('click', () => {
            this.generateChart();
        });

        document.getElementById('import-data-btn').addEventListener('click', () => {
            this.importData();
        });

        // Export functionality
        document.querySelectorAll('[data-format]').forEach(btn => {
            btn.addEventListener('click', (e) => {
                this.exportChart(btn.dataset.format);
            });
        });

        // Style creator modal
        document.getElementById('create-style-btn').addEventListener('click', () => {
            this.showStyleCreator();
        });

        document.getElementById('close-style-modal').addEventListener('click', () => {
            this.hideStyleCreator();
        });

        document.getElementById('cancel-style-btn').addEventListener('click', () => {
            this.hideStyleCreator();
        });

        document.getElementById('save-style-btn').addEventListener('click', () => {
            this.saveCustomStyle();
        });

        // Color picker updates for style preview
        document.querySelectorAll('#style-creator-modal input[type="color"]').forEach(input => {
            input.addEventListener('change', () => {
                this.updateStylePreview();
            });
        });

        // Modal overlay click to close
        document.getElementById('style-creator-modal').addEventListener('click', (e) => {
            if (e.target.classList.contains('modal-overlay')) {
                this.hideStyleCreator();
            }
        });
    }

    /**
     * Set up health check endpoint for service monitoring
     */
    setupHealthEndpoint() {
        // Simple health check - if this page loads, the UI service is healthy
        const healthCheck = document.getElementById('health-check');
        if (healthCheck) {
            // Add timestamp to health check for monitoring
            const timestamp = new Date().toISOString();
            healthCheck.setAttribute('data-timestamp', timestamp);
        }

        // Expose health endpoint at /health for programmatic checking
        if (window.location.pathname.endsWith('/health')) {
            document.body.innerHTML = 'OK';
            document.body.style.fontFamily = 'monospace';
            document.body.style.padding = '20px';
        }
    }

    /**
     * Switch between navigation tabs
     */
    switchTab(tabName) {
        // Update tab buttons
        document.querySelectorAll('.nav-tab').forEach(tab => {
            tab.classList.toggle('active', tab.dataset.tab === tabName);
        });

        // Update tab content
        document.querySelectorAll('.tab-content').forEach(content => {
            content.classList.toggle('active', content.id === tabName);
        });
    }

    /**
     * Select chart type
     */
    selectChartType(type) {
        this.currentChartType = type;
        
        // Update UI
        document.querySelectorAll('.chart-type-card').forEach(card => {
            card.classList.toggle('active', card.dataset.type === type);
        });

        document.getElementById('current-type').textContent = this.capitalizeFirst(type) + ' Chart';
        
        this.refreshPreview();
    }

    /**
     * Select style
     */
    selectStyle(style) {
        this.currentStyle = style;
        
        // Update UI
        document.querySelectorAll('.style-card').forEach(card => {
            card.classList.toggle('active', card.dataset.style === style);
        });

        document.getElementById('current-style').textContent = this.capitalizeFirst(style);
        
        this.refreshPreview();
    }

    /**
     * Load sample data
     */
    loadSampleData(sampleType) {
        const sampleData = this.chartEngine.getSampleData(sampleType);
        this.currentData = sampleData;
        
        // Update data editor
        document.getElementById('data-input').value = JSON.stringify(sampleData, null, 2);
        
        this.updateDataInfo();
        this.refreshPreview();
    }

    /**
     * Validate and update data from text input
     */
    validateAndUpdateData(dataString) {
        try {
            if (!dataString.trim()) {
                this.currentData = null;
                this.updateDataInfo();
                return;
            }

            const data = JSON.parse(dataString);
            if (Array.isArray(data)) {
                this.currentData = data;
                this.updateDataInfo();
                this.refreshPreview();
                this.showDataStatus('valid');
            } else {
                this.showDataStatus('invalid', 'Data must be an array');
            }
        } catch (error) {
            this.showDataStatus('invalid', 'Invalid JSON format');
        }
    }

    /**
     * Validate current data
     */
    validateCurrentData() {
        const dataInput = document.getElementById('data-input');
        this.validateAndUpdateData(dataInput.value);
    }

    /**
     * Clear data
     */
    clearData() {
        document.getElementById('data-input').value = '';
        this.currentData = null;
        this.updateDataInfo();
        this.showPlaceholder();
    }

    /**
     * Refresh chart preview
     */
    refreshPreview() {
        if (!this.currentData || !Array.isArray(this.currentData) || this.currentData.length === 0) {
            this.showPlaceholder();
            return;
        }

        try {
            this.chartEngine.generateChart(
                this.previewContainer,
                this.currentChartType,
                this.currentData,
                this.currentStyle
            );
            
            this.updateDataInfo();
        } catch (error) {
            console.error('Chart generation error:', error);
            this.showError('Failed to generate chart: ' + error.message);
        }
    }

    /**
     * Generate chart (main action)
     */
    generateChart() {
        if (!this.currentData) {
            alert('Please provide data before generating a chart.');
            return;
        }

        this.refreshPreview();
        
        // Show success message
        this.showNotification('Chart generated successfully!', 'success');
    }

    /**
     * Import data from file
     */
    importData() {
        const input = document.createElement('input');
        input.type = 'file';
        input.accept = '.json,.csv,.txt';
        
        input.onchange = (e) => {
            const file = e.target.files[0];
            if (!file) return;

            const reader = new FileReader();
            reader.onload = (e) => {
                try {
                    let data;
                    const content = e.target.result;
                    
                    if (file.name.endsWith('.csv')) {
                        data = this.parseCSV(content);
                    } else {
                        data = JSON.parse(content);
                    }
                    
                    if (Array.isArray(data)) {
                        this.currentData = data;
                        document.getElementById('data-input').value = JSON.stringify(data, null, 2);
                        this.updateDataInfo();
                        this.refreshPreview();
                        this.showNotification('Data imported successfully!', 'success');
                    } else {
                        throw new Error('Data must be an array');
                    }
                } catch (error) {
                    this.showNotification('Failed to import data: ' + error.message, 'error');
                }
            };
            reader.readAsText(file);
        };
        
        input.click();
    }

    /**
     * Parse CSV data
     */
    parseCSV(csvString) {
        const lines = csvString.trim().split('\n');
        if (lines.length < 2) return [];
        
        const headers = lines[0].split(',').map(h => h.trim());
        const data = [];
        
        for (let i = 1; i < lines.length; i++) {
            const values = lines[i].split(',');
            const row = {};
            headers.forEach((header, index) => {
                const value = values[index]?.trim();
                // Try to parse as number, otherwise keep as string
                row[header] = !isNaN(value) ? parseFloat(value) : value;
            });
            data.push(row);
        }
        
        return data;
    }

    /**
     * Export chart in specified format
     */
    exportChart(format) {
        if (!this.previewContainer.querySelector('svg')) {
            alert('No chart to export. Please generate a chart first.');
            return;
        }

        const result = this.chartEngine.exportChart(this.previewContainer, format);
        
        if (format === 'svg') {
            this.downloadFile(result, `chart.${format}`, 'image/svg+xml');
        } else if (format === 'png' && result instanceof Promise) {
            result.then(dataUrl => {
                this.downloadDataURL(dataUrl, `chart.${format}`);
            });
        } else {
            this.showNotification('Export format not yet implemented', 'warning');
        }
    }

    /**
     * Show style creator modal
     */
    showStyleCreator() {
        document.getElementById('style-creator-modal').classList.add('active');
        this.updateStylePreview();
    }

    /**
     * Hide style creator modal
     */
    hideStyleCreator() {
        document.getElementById('style-creator-modal').classList.remove('active');
    }

    /**
     * Update style preview in creator
     */
    updateStylePreview() {
        const primary = document.getElementById('color-primary').value;
        const secondary = document.getElementById('color-secondary').value;
        const accent = document.getElementById('color-accent').value;
        
        // Create a mini bar chart preview
        const svg = d3.select('#style-preview-chart');
        svg.selectAll('*').remove();
        
        const sampleData = [
            { x: 'A', y: 30 },
            { x: 'B', y: 50 },
            { x: 'C', y: 40 }
        ];
        
        const width = 180;
        const height = 100;
        const margin = { top: 10, right: 10, bottom: 20, left: 20 };
        
        const xScale = d3.scaleBand()
            .domain(sampleData.map(d => d.x))
            .range([margin.left, width - margin.right])
            .padding(0.2);
            
        const yScale = d3.scaleLinear()
            .domain([0, d3.max(sampleData, d => d.y)])
            .range([height - margin.bottom, margin.top]);
        
        svg.selectAll('.bar')
            .data(sampleData)
            .enter().append('rect')
            .attr('class', 'bar')
            .attr('x', d => xScale(d.x))
            .attr('width', xScale.bandwidth())
            .attr('y', d => yScale(d.y))
            .attr('height', d => height - margin.bottom - yScale(d.y))
            .attr('fill', (d, i) => [primary, secondary, accent][i]);
    }

    /**
     * Save custom style
     */
    saveCustomStyle() {
        const name = document.getElementById('style-name').value.trim();
        const category = document.getElementById('style-category').value;
        
        if (!name) {
            alert('Please enter a style name.');
            return;
        }
        
        const customStyle = {
            colors: [
                document.getElementById('color-primary').value,
                document.getElementById('color-secondary').value,
                document.getElementById('color-accent').value
            ],
            backgroundColor: '#ffffff',
            textColor: '#1f2937',
            gridColor: '#e5e7eb',
            fontSize: 12,
            fontFamily: 'Inter, sans-serif'
        };
        
        // Add to chart engine styles
        this.chartEngine.defaultStyles[name.toLowerCase().replace(/\s+/g, '-')] = customStyle;
        
        // Add to UI (in a real app, this would be saved to database)
        this.addStyleToUI(name, category, customStyle);
        
        this.hideStyleCreator();
        this.showNotification(`Style "${name}" created successfully!`, 'success');
        
        // Reset form
        document.getElementById('style-name').value = '';
        document.getElementById('style-category').value = 'professional';
    }

    /**
     * Add new style to UI
     */
    addStyleToUI(name, category, style) {
        const stylesList = document.querySelector('.styles-list');
        const styleCard = document.createElement('div');
        styleCard.className = 'style-card';
        styleCard.dataset.style = name.toLowerCase().replace(/\s+/g, '-');
        
        styleCard.innerHTML = `
            <div class="style-preview">
                <div class="style-swatch" style="background: linear-gradient(45deg, ${style.colors[0]}, ${style.colors[1]})"></div>
            </div>
            <div class="style-info">
                <span class="style-name">${name}</span>
                <span class="style-desc">${category} styling</span>
            </div>
        `;
        
        styleCard.addEventListener('click', () => {
            this.selectStyle(styleCard.dataset.style);
        });
        
        stylesList.appendChild(styleCard);
    }

    /**
     * Show placeholder when no chart is displayed
     */
    showPlaceholder() {
        this.previewContainer.innerHTML = `
            <div class="chart-placeholder">
                <div class="placeholder-icon">📊</div>
                <h3>Select chart type and data to preview</h3>
                <p>Choose a chart type from the sidebar and provide data to see your visualization</p>
            </div>
        `;
    }

    /**
     * Show error message
     */
    showError(message) {
        this.previewContainer.innerHTML = `
            <div class="chart-placeholder">
                <div class="placeholder-icon">⚠️</div>
                <h3>Chart Generation Error</h3>
                <p>${message}</p>
            </div>
        `;
    }

    /**
     * Update data information display
     */
    updateDataInfo() {
        const count = this.currentData ? this.currentData.length : 0;
        document.getElementById('data-count').textContent = count;
        document.getElementById('last-updated').textContent = new Date().toLocaleTimeString();
    }

    /**
     * Show data validation status
     */
    showDataStatus(status, message = '') {
        const dataInput = document.getElementById('data-input');
        dataInput.classList.remove('valid', 'invalid');
        
        if (status === 'valid') {
            dataInput.classList.add('valid');
        } else if (status === 'invalid') {
            dataInput.classList.add('invalid');
            if (message) {
                console.warn('Data validation error:', message);
            }
        }
    }

    /**
     * Show notification message
     */
    showNotification(message, type = 'info') {
        // Simple notification system (in a real app, you'd use a toast library)
        const notification = document.createElement('div');
        notification.className = `notification ${type}`;
        notification.textContent = message;
        notification.style.cssText = `
            position: fixed;
            top: 20px;
            right: 20px;
            padding: 12px 24px;
            background: ${type === 'success' ? '#10b981' : type === 'error' ? '#ef4444' : type === 'warning' ? '#f59e0b' : '#2563eb'};
            color: white;
            border-radius: 6px;
            z-index: 2000;
            font-size: 14px;
            font-weight: 500;
        `;
        
        document.body.appendChild(notification);
        
        setTimeout(() => {
            notification.remove();
        }, 3000);
    }

    /**
     * Download file
     */
    downloadFile(content, filename, mimeType) {
        const blob = new Blob([content], { type: mimeType });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = filename;
        a.click();
        URL.revokeObjectURL(url);
    }

    /**
     * Download data URL as file
     */
    downloadDataURL(dataUrl, filename) {
        const a = document.createElement('a');
        a.href = dataUrl;
        a.download = filename;
        a.click();
    }

    /**
     * Capitalize first letter of string
     */
    capitalizeFirst(str) {
        return str.charAt(0).toUpperCase() + str.slice(1);
    }
}

// Initialize app when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    new ChartGeneratorApp();
});

// Health check endpoint - if accessing /health, just show OK
if (window.location.pathname.endsWith('/health')) {
    document.addEventListener('DOMContentLoaded', () => {
        document.body.innerHTML = '<div style="font-family: monospace; padding: 20px; text-align: center; font-size: 18px;">OK</div>';
    });
}
