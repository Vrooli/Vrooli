/**
 * Professional Chart Generation Engine
 * Built on D3.js for maximum quality and customization
 */

class ChartEngine {
    constructor() {
        this.defaultStyles = {
            professional: {
                colors: ['#2563eb', '#64748b', '#f59e0b', '#10b981', '#ef4444', '#8b5cf6'],
                backgroundColor: '#ffffff',
                textColor: '#1f2937',
                gridColor: '#e5e7eb',
                fontSize: 12,
                fontFamily: 'Inter, sans-serif'
            },
            minimal: {
                colors: ['#6b7280', '#9ca3af', '#d1d5db', '#374151', '#111827', '#f9fafb'],
                backgroundColor: '#ffffff',
                textColor: '#374151',
                gridColor: '#f3f4f6',
                fontSize: 11,
                fontFamily: 'Inter, sans-serif'
            },
            vibrant: {
                colors: ['#f59e0b', '#ef4444', '#8b5cf6', '#10b981', '#3b82f6', '#f97316'],
                backgroundColor: '#ffffff',
                textColor: '#1f2937',
                gridColor: '#f3f4f6',
                fontSize: 12,
                fontFamily: 'Inter, sans-serif'
            }
        };
    }

    /**
     * Generate a chart based on type, data, and style
     */
    generateChart(container, type, data, style = 'professional', config = {}) {
        // Clear existing chart
        d3.select(container).selectAll('*').remove();
        
        const styleConfig = this.defaultStyles[style] || this.defaultStyles.professional;
        
        // Set up dimensions
        const margin = { top: 20, right: 30, bottom: 40, left: 50 };
        const containerRect = container.getBoundingClientRect();
        const width = (containerRect.width || 600) - margin.left - margin.right;
        const height = (containerRect.height || 400) - margin.top - margin.bottom;

        // Create SVG
        const svg = d3.select(container)
            .append('svg')
            .attr('width', width + margin.left + margin.right)
            .attr('height', height + margin.top + margin.bottom);

        const g = svg.append('g')
            .attr('transform', `translate(${margin.left},${margin.top})`);

        // Apply background
        svg.insert('rect', ':first-child')
            .attr('width', '100%')
            .attr('height', '100%')
            .attr('fill', styleConfig.backgroundColor);

        // Route to specific chart type
        switch (type) {
            case 'bar':
                this.createBarChart(g, data, width, height, styleConfig, config);
                break;
            case 'line':
                this.createLineChart(g, data, width, height, styleConfig, config);
                break;
            case 'pie':
                this.createPieChart(g, data, width, height, styleConfig, config);
                break;
            case 'scatter':
                this.createScatterPlot(g, data, width, height, styleConfig, config);
                break;
            case 'area':
                this.createAreaChart(g, data, width, height, styleConfig, config);
                break;
            default:
                this.createBarChart(g, data, width, height, styleConfig, config);
        }

        return svg.node();
    }

    /**
     * Create a professional bar chart
     */
    createBarChart(g, data, width, height, style, config) {
        if (!data || !Array.isArray(data) || data.length === 0) {
            this.showNoDataMessage(g, width, height);
            return;
        }

        // Ensure data has x and y properties
        const processedData = data.map(d => ({
            x: d.x || d.label || d.category || 'Unknown',
            y: +d.y || +d.value || +d.count || 0
        }));

        // Set up scales
        const xScale = d3.scaleBand()
            .domain(processedData.map(d => d.x))
            .range([0, width])
            .padding(0.2);

        const yScale = d3.scaleLinear()
            .domain([0, d3.max(processedData, d => d.y)])
            .nice()
            .range([height, 0]);

        // Create bars
        g.selectAll('.bar')
            .data(processedData)
            .enter().append('rect')
            .attr('class', 'bar')
            .attr('x', d => xScale(d.x))
            .attr('width', xScale.bandwidth())
            .attr('y', d => yScale(d.y))
            .attr('height', d => height - yScale(d.y))
            .attr('fill', (d, i) => style.colors[i % style.colors.length])
            .attr('opacity', 0.8)
            .on('mouseover', function(event, d) {
                d3.select(this).attr('opacity', 1);
            })
            .on('mouseout', function(event, d) {
                d3.select(this).attr('opacity', 0.8);
            });

        // Add axes
        this.addXAxis(g, xScale, height, style);
        this.addYAxis(g, yScale, style);

        // Add grid lines
        this.addGridLines(g, xScale, yScale, width, height, style);
    }

    /**
     * Create a professional line chart
     */
    createLineChart(g, data, width, height, style, config) {
        if (!data || !Array.isArray(data) || data.length === 0) {
            this.showNoDataMessage(g, width, height);
            return;
        }

        // Process data
        const processedData = data.map((d, i) => ({
            x: d.x || d.date || i,
            y: +d.y || +d.value || 0
        }));

        // Set up scales
        const xScale = d3.scaleLinear()
            .domain(d3.extent(processedData, (d, i) => typeof d.x === 'number' ? d.x : i))
            .range([0, width]);

        const yScale = d3.scaleLinear()
            .domain(d3.extent(processedData, d => d.y))
            .nice()
            .range([height, 0]);

        // Create line generator
        const line = d3.line()
            .x((d, i) => xScale(typeof d.x === 'number' ? d.x : i))
            .y(d => yScale(d.y))
            .curve(d3.curveMonotoneX);

        // Add line
        g.append('path')
            .datum(processedData)
            .attr('fill', 'none')
            .attr('stroke', style.colors[0])
            .attr('stroke-width', 2)
            .attr('d', line);

        // Add dots
        g.selectAll('.dot')
            .data(processedData)
            .enter().append('circle')
            .attr('class', 'dot')
            .attr('cx', (d, i) => xScale(typeof d.x === 'number' ? d.x : i))
            .attr('cy', d => yScale(d.y))
            .attr('r', 4)
            .attr('fill', style.colors[0]);

        // Add axes
        this.addXAxis(g, xScale, height, style);
        this.addYAxis(g, yScale, style);
        this.addGridLines(g, xScale, yScale, width, height, style);
    }

    /**
     * Create a professional pie chart
     */
    createPieChart(g, data, width, height, style, config) {
        if (!data || !Array.isArray(data) || data.length === 0) {
            this.showNoDataMessage(g, width, height);
            return;
        }

        // Process data
        const processedData = data.map(d => ({
            label: d.x || d.label || d.category || 'Unknown',
            value: +d.y || +d.value || 0
        }));

        const radius = Math.min(width, height) / 2;

        // Center the pie chart
        g.attr('transform', `translate(${width / 2}, ${height / 2})`);

        const pie = d3.pie()
            .value(d => d.value)
            .sort(null);

        const arc = d3.arc()
            .innerRadius(0)
            .outerRadius(radius - 10);

        const labelArc = d3.arc()
            .innerRadius(radius - 40)
            .outerRadius(radius - 40);

        // Create pie slices
        const arcs = g.selectAll('.arc')
            .data(pie(processedData))
            .enter().append('g')
            .attr('class', 'arc');

        arcs.append('path')
            .attr('d', arc)
            .attr('fill', (d, i) => style.colors[i % style.colors.length])
            .attr('opacity', 0.8)
            .on('mouseover', function(event, d) {
                d3.select(this).attr('opacity', 1);
            })
            .on('mouseout', function(event, d) {
                d3.select(this).attr('opacity', 0.8);
            });

        // Add labels
        arcs.append('text')
            .attr('transform', d => `translate(${labelArc.centroid(d)})`)
            .attr('dy', '.35em')
            .style('text-anchor', 'middle')
            .style('font-family', style.fontFamily)
            .style('font-size', `${style.fontSize}px`)
            .style('fill', style.textColor)
            .text(d => d.data.label);
    }

    /**
     * Create a scatter plot
     */
    createScatterPlot(g, data, width, height, style, config) {
        if (!data || !Array.isArray(data) || data.length === 0) {
            this.showNoDataMessage(g, width, height);
            return;
        }

        // Process data
        const processedData = data.map(d => ({
            x: +d.x || 0,
            y: +d.y || 0
        }));

        // Set up scales
        const xScale = d3.scaleLinear()
            .domain(d3.extent(processedData, d => d.x))
            .nice()
            .range([0, width]);

        const yScale = d3.scaleLinear()
            .domain(d3.extent(processedData, d => d.y))
            .nice()
            .range([height, 0]);

        // Add dots
        g.selectAll('.dot')
            .data(processedData)
            .enter().append('circle')
            .attr('class', 'dot')
            .attr('cx', d => xScale(d.x))
            .attr('cy', d => yScale(d.y))
            .attr('r', 5)
            .attr('fill', style.colors[0])
            .attr('opacity', 0.7)
            .on('mouseover', function(event, d) {
                d3.select(this).attr('r', 7).attr('opacity', 1);
            })
            .on('mouseout', function(event, d) {
                d3.select(this).attr('r', 5).attr('opacity', 0.7);
            });

        // Add axes
        this.addXAxis(g, xScale, height, style);
        this.addYAxis(g, yScale, style);
        this.addGridLines(g, xScale, yScale, width, height, style);
    }

    /**
     * Create an area chart
     */
    createAreaChart(g, data, width, height, style, config) {
        if (!data || !Array.isArray(data) || data.length === 0) {
            this.showNoDataMessage(g, width, height);
            return;
        }

        // Process data
        const processedData = data.map((d, i) => ({
            x: d.x || i,
            y: +d.y || +d.value || 0
        }));

        // Set up scales
        const xScale = d3.scaleLinear()
            .domain(d3.extent(processedData, (d, i) => typeof d.x === 'number' ? d.x : i))
            .range([0, width]);

        const yScale = d3.scaleLinear()
            .domain([0, d3.max(processedData, d => d.y)])
            .nice()
            .range([height, 0]);

        // Create area generator
        const area = d3.area()
            .x((d, i) => xScale(typeof d.x === 'number' ? d.x : i))
            .y0(height)
            .y1(d => yScale(d.y))
            .curve(d3.curveMonotoneX);

        // Add area
        g.append('path')
            .datum(processedData)
            .attr('fill', style.colors[0])
            .attr('opacity', 0.6)
            .attr('d', area);

        // Add line on top
        const line = d3.line()
            .x((d, i) => xScale(typeof d.x === 'number' ? d.x : i))
            .y(d => yScale(d.y))
            .curve(d3.curveMonotoneX);

        g.append('path')
            .datum(processedData)
            .attr('fill', 'none')
            .attr('stroke', style.colors[0])
            .attr('stroke-width', 2)
            .attr('d', line);

        // Add axes
        this.addXAxis(g, xScale, height, style);
        this.addYAxis(g, yScale, style);
        this.addGridLines(g, xScale, yScale, width, height, style);
    }

    /**
     * Add X-axis with professional styling
     */
    addXAxis(g, xScale, height, style) {
        g.append('g')
            .attr('class', 'x-axis')
            .attr('transform', `translate(0,${height})`)
            .call(d3.axisBottom(xScale))
            .selectAll('text')
            .style('font-family', style.fontFamily)
            .style('font-size', `${style.fontSize}px`)
            .style('fill', style.textColor);

        // Style axis line
        g.select('.x-axis .domain')
            .style('stroke', style.gridColor);

        // Style tick lines
        g.selectAll('.x-axis .tick line')
            .style('stroke', style.gridColor);
    }

    /**
     * Add Y-axis with professional styling
     */
    addYAxis(g, yScale, style) {
        g.append('g')
            .attr('class', 'y-axis')
            .call(d3.axisLeft(yScale))
            .selectAll('text')
            .style('font-family', style.fontFamily)
            .style('font-size', `${style.fontSize}px`)
            .style('fill', style.textColor);

        // Style axis line
        g.select('.y-axis .domain')
            .style('stroke', style.gridColor);

        // Style tick lines
        g.selectAll('.y-axis .tick line')
            .style('stroke', style.gridColor);
    }

    /**
     * Add subtle grid lines
     */
    addGridLines(g, xScale, yScale, width, height, style) {
        // Vertical grid lines
        if (xScale.ticks) {
            g.selectAll('.grid-line-vertical')
                .data(xScale.ticks())
                .enter().append('line')
                .attr('class', 'grid-line-vertical')
                .attr('x1', d => xScale(d))
                .attr('x2', d => xScale(d))
                .attr('y1', 0)
                .attr('y2', height)
                .style('stroke', style.gridColor)
                .style('stroke-opacity', 0.3)
                .style('stroke-dasharray', '2,2');
        }

        // Horizontal grid lines
        g.selectAll('.grid-line-horizontal')
            .data(yScale.ticks())
            .enter().append('line')
            .attr('class', 'grid-line-horizontal')
            .attr('x1', 0)
            .attr('x2', width)
            .attr('y1', d => yScale(d))
            .attr('y2', d => yScale(d))
            .style('stroke', style.gridColor)
            .style('stroke-opacity', 0.3)
            .style('stroke-dasharray', '2,2');
    }

    /**
     * Show a professional "no data" message
     */
    showNoDataMessage(g, width, height) {
        const messageGroup = g.append('g')
            .attr('class', 'no-data-message')
            .attr('transform', `translate(${width / 2}, ${height / 2})`);

        messageGroup.append('text')
            .attr('text-anchor', 'middle')
            .attr('dy', '-10px')
            .style('font-family', 'Inter, sans-serif')
            .style('font-size', '18px')
            .style('fill', '#9ca3af')
            .text('📊');

        messageGroup.append('text')
            .attr('text-anchor', 'middle')
            .attr('dy', '20px')
            .style('font-family', 'Inter, sans-serif')
            .style('font-size', '14px')
            .style('fill', '#6b7280')
            .text('No data available');
    }

    /**
     * Get sample data for testing
     */
    getSampleData(type) {
        const sampleData = {
            sales: [
                { x: 'Jan', y: 1200 },
                { x: 'Feb', y: 1900 },
                { x: 'Mar', y: 3000 },
                { x: 'Apr', y: 5000 },
                { x: 'May', y: 4200 },
                { x: 'Jun', y: 5500 }
            ],
            revenue: [
                { x: 'Q1', y: 45000 },
                { x: 'Q2', y: 52000 },
                { x: 'Q3', y: 61000 },
                { x: 'Q4', y: 58000 }
            ],
            performance: [
                { x: 'Speed', y: 85 },
                { x: 'Quality', y: 92 },
                { x: 'Efficiency', y: 78 },
                { x: 'Satisfaction', y: 88 }
            ]
        };

        return sampleData[type] || sampleData.sales;
    }

    /**
     * Export chart as different formats
     */
    exportChart(container, format = 'png') {
        const svg = container.querySelector('svg');
        if (!svg) return null;

        switch (format.toLowerCase()) {
            case 'svg':
                return this.exportAsSVG(svg);
            case 'png':
                return this.exportAsPNG(svg);
            case 'pdf':
                // PDF export would require additional library like jsPDF
                console.log('PDF export requires additional implementation');
                return null;
            default:
                return this.exportAsPNG(svg);
        }
    }

    /**
     * Export as SVG string
     */
    exportAsSVG(svg) {
        const serializer = new XMLSerializer();
        return serializer.serializeToString(svg);
    }

    /**
     * Export as PNG data URL
     */
    exportAsPNG(svg) {
        const canvas = document.createElement('canvas');
        const ctx = canvas.getContext('2d');
        const data = new XMLSerializer().serializeToString(svg);
        
        const img = new Image();
        const svgBlob = new Blob([data], { type: 'image/svg+xml;charset=utf-8' });
        const url = URL.createObjectURL(svgBlob);
        
        return new Promise((resolve) => {
            img.onload = function() {
                canvas.width = img.width;
                canvas.height = img.height;
                ctx.drawImage(img, 0, 0);
                URL.revokeObjectURL(url);
                resolve(canvas.toDataURL('image/png'));
            };
            img.src = url;
        });
    }
}

// Make ChartEngine globally available
window.ChartEngine = ChartEngine;