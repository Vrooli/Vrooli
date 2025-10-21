import * as d3 from 'd3';

export type ChartType = 'bar' | 'line' | 'pie' | 'scatter' | 'area';
export type ChartStyleId = 'professional' | 'minimal' | 'vibrant';

export interface ChartDatum {
  x?: string | number;
  y?: number;
  label?: string;
  value?: number;
  category?: string;
  count?: number;
  date?: string | number;
}

export interface ChartStyle {
  colors: string[];
  backgroundColor: string;
  textColor: string;
  gridColor: string;
  fontSize: number;
  fontFamily: string;
}

const defaultStyles: Record<ChartStyleId, ChartStyle> = {
  professional: {
    colors: ['#2563eb', '#64748b', '#f59e0b', '#10b981', '#ef4444', '#8b5cf6'],
    backgroundColor: '#ffffff',
    textColor: '#1f2937',
    gridColor: '#e5e7eb',
    fontSize: 12,
    fontFamily: 'Inter, sans-serif',
  },
  minimal: {
    colors: ['#6b7280', '#9ca3af', '#d1d5db', '#374151', '#111827', '#f9fafb'],
    backgroundColor: '#ffffff',
    textColor: '#374151',
    gridColor: '#f3f4f6',
    fontSize: 11,
    fontFamily: 'Inter, sans-serif',
  },
  vibrant: {
    colors: ['#f59e0b', '#ef4444', '#8b5cf6', '#10b981', '#3b82f6', '#f97316'],
    backgroundColor: '#ffffff',
    textColor: '#1f2937',
    gridColor: '#f3f4f6',
    fontSize: 12,
    fontFamily: 'Inter, sans-serif',
  },
};

export class ChartEngine {
  readonly defaultStyles = defaultStyles;

  generateChart(
    container: HTMLElement,
    type: ChartType,
    data: ChartDatum[],
    style: ChartStyleId = 'professional',
    _config: Record<string, unknown> = {},
  ) {
    const styleConfig = this.defaultStyles[style] ?? this.defaultStyles.professional;

    const root = d3.select(container);
    root.selectAll('*').remove();

    const margin = { top: 20, right: 30, bottom: 40, left: 56 };
    const rect = container.getBoundingClientRect();
    const width = Math.max(rect.width || 600, 360) - margin.left - margin.right;
    const height = Math.max(rect.height || 420, 240) - margin.top - margin.bottom;

    const svg = root
      .append('svg')
      .attr('width', width + margin.left + margin.right)
      .attr('height', height + margin.top + margin.bottom);

    svg
      .insert('rect', ':first-child')
      .attr('width', '100%')
      .attr('height', '100%')
      .attr('rx', 16)
      .attr('fill', styleConfig.backgroundColor);

    const g = svg
      .append('g')
      .attr('transform', `translate(${margin.left},${margin.top})`);

    switch (type) {
      case 'bar':
        this.createBarChart(g, data, width, height, styleConfig);
        break;
      case 'line':
        this.createLineChart(g, data, width, height, styleConfig);
        break;
      case 'pie':
        this.createPieChart(g, data, width, height, styleConfig);
        break;
      case 'scatter':
        this.createScatterPlot(g, data, width, height, styleConfig);
        break;
      case 'area':
        this.createAreaChart(g, data, width, height, styleConfig);
        break;
      default:
        this.createBarChart(g, data, width, height, styleConfig);
    }

    return svg.node();
  }

  private createBarChart(
    g: d3.Selection<SVGGElement, unknown, null, undefined>,
    data: ChartDatum[],
    width: number,
    height: number,
    style: ChartStyle,
    _config: Record<string, unknown> = {},
  ) {
    if (!Array.isArray(data) || data.length === 0) {
      this.showNoDataMessage(g, width, height);
      return;
    }

    const processed = data.map((d) => ({
      x: String(d.x ?? d.label ?? d.category ?? 'Unknown'),
      y: Number(d.y ?? d.value ?? d.count ?? 0),
    }));

    const xScale = d3
      .scaleBand<string>()
      .domain(processed.map((d) => d.x))
      .range([0, width])
      .padding(0.2);

    const yMax = d3.max(processed, (d) => d.y) ?? 0;
    const yScale = d3
      .scaleLinear()
      .domain([0, yMax <= 0 ? 1 : yMax])
      .nice()
      .range([height, 0]);

    g.selectAll('.bar')
      .data(processed)
      .enter()
      .append('rect')
      .attr('class', 'bar')
      .attr('x', (d) => xScale(d.x) ?? 0)
      .attr('width', xScale.bandwidth())
      .attr('y', (d) => yScale(d.y))
      .attr('height', (d) => Math.max(0, height - yScale(d.y)))
      .attr('rx', 6)
      .attr('fill', (_d, i) => style.colors[i % style.colors.length])
      .attr('opacity', 0.85)
      .on('mouseover', function mouseover() {
        d3.select(this).attr('opacity', 1);
      })
      .on('mouseout', function mouseout() {
        d3.select(this).attr('opacity', 0.85);
      });

    this.addXAxis(g, xScale, height, style);
    this.addYAxis(g, yScale, style);
    this.addGridLines(g, xScale, yScale, width, height, style);
  }

  private createLineChart(
    g: d3.Selection<SVGGElement, unknown, null, undefined>,
    data: ChartDatum[],
    width: number,
    height: number,
    style: ChartStyle,
  ) {
    if (!Array.isArray(data) || data.length === 0) {
      this.showNoDataMessage(g, width, height);
      return;
    }

    const processed = data.map((d, index) => ({
      x: typeof d.x === 'number' ? d.x : typeof d.date === 'number' ? d.date : index,
      y: Number(d.y ?? d.value ?? 0),
    }));

    const xValues = processed.map((d) => d.x);
    const xExtent = d3.extent(xValues);
    const yExtent = d3.extent(processed, (d) => d.y);

    const xScale = d3
      .scaleLinear()
      .domain([xExtent[0] ?? 0, xExtent[1] ?? processed.length - 1])
      .range([0, width]);

    const yScale = d3
      .scaleLinear()
      .domain([yExtent[0] ?? 0, yExtent[1] ?? 1])
      .nice()
      .range([height, 0]);

    const line = d3
      .line<{ x: number; y: number }>()
      .x((d) => xScale(d.x))
      .y((d) => yScale(d.y))
      .curve(d3.curveMonotoneX);

    g.append('path')
      .datum(processed)
      .attr('fill', 'none')
      .attr('stroke', style.colors[0])
      .attr('stroke-width', 2.25)
      .attr('d', line);

    g.selectAll('.dot')
      .data(processed)
      .enter()
      .append('circle')
      .attr('class', 'dot')
      .attr('cx', (d) => xScale(d.x))
      .attr('cy', (d) => yScale(d.y))
      .attr('r', 4.5)
      .attr('fill', style.colors[0])
      .attr('stroke', style.backgroundColor)
      .attr('stroke-width', 1.5);

    this.addXAxis(g, xScale, height, style);
    this.addYAxis(g, yScale, style);
    this.addGridLines(g, xScale, yScale, width, height, style);
  }

  private createPieChart(
    g: d3.Selection<SVGGElement, unknown, null, undefined>,
    data: ChartDatum[],
    width: number,
    height: number,
    style: ChartStyle,
  ) {
    if (!Array.isArray(data) || data.length === 0) {
      this.showNoDataMessage(g, width, height);
      return;
    }

    const processed = data.map((d) => ({
      label: String(d.x ?? d.label ?? d.category ?? 'Unknown'),
      value: Number(d.y ?? d.value ?? 0),
    }));

    const radius = Math.min(width, height) / 2;

    g.attr('transform', `translate(${width / 2}, ${height / 2})`);

    const pie = d3
      .pie<{ label: string; value: number }>()
      .value((d) => d.value)
      .sort(null);

    const arc = d3
      .arc<d3.PieArcDatum<{ label: string; value: number }>>()
      .innerRadius(0)
      .outerRadius(radius - 10);

    const labelArc = d3
      .arc<d3.PieArcDatum<{ label: string; value: number }>>()
      .innerRadius(radius - 42)
      .outerRadius(radius - 42);

    const arcs = g
      .selectAll<SVGGElement, d3.PieArcDatum<{ label: string; value: number }>>('.arc')
      .data(pie(processed))
      .enter()
      .append('g')
      .attr('class', 'arc');

    arcs
      .append('path')
      .attr('d', arc)
      .attr('fill', (_d, i) => style.colors[i % style.colors.length])
      .attr('opacity', 0.9)
      .on('mouseover', function mouseover() {
        d3.select(this).attr('opacity', 1);
      })
      .on('mouseout', function mouseout() {
        d3.select(this).attr('opacity', 0.9);
      });

    arcs
      .append('text')
      .attr('transform', (d) => `translate(${labelArc.centroid(d)})`)
      .attr('dy', '.35em')
      .style('text-anchor', 'middle')
      .style('font-family', style.fontFamily)
      .style('font-size', `${style.fontSize}px`)
      .style('fill', style.textColor)
      .text((d) => d.data.label);
  }

  private createScatterPlot(
    g: d3.Selection<SVGGElement, unknown, null, undefined>,
    data: ChartDatum[],
    width: number,
    height: number,
    style: ChartStyle,
  ) {
    if (!Array.isArray(data) || data.length === 0) {
      this.showNoDataMessage(g, width, height);
      return;
    }

    const processed = data.map((d) => ({
      x: Number(d.x ?? 0),
      y: Number(d.y ?? 0),
    }));

    const xExtent = d3.extent(processed, (d) => d.x);
    const yExtent = d3.extent(processed, (d) => d.y);

    const xScale = d3
      .scaleLinear()
      .domain([xExtent[0] ?? 0, xExtent[1] ?? 1])
      .nice()
      .range([0, width]);

    const yScale = d3
      .scaleLinear()
      .domain([yExtent[0] ?? 0, yExtent[1] ?? 1])
      .nice()
      .range([height, 0]);

    g.selectAll('.dot')
      .data(processed)
      .enter()
      .append('circle')
      .attr('class', 'dot')
      .attr('cx', (d) => xScale(d.x))
      .attr('cy', (d) => yScale(d.y))
      .attr('r', 5)
      .attr('fill', style.colors[0])
      .attr('opacity', 0.75)
      .on('mouseover', function mouseover() {
        d3.select(this).attr('r', 7).attr('opacity', 1);
      })
      .on('mouseout', function mouseout() {
        d3.select(this).attr('r', 5).attr('opacity', 0.75);
      });

    this.addXAxis(g, xScale, height, style);
    this.addYAxis(g, yScale, style);
    this.addGridLines(g, xScale, yScale, width, height, style);
  }

  private createAreaChart(
    g: d3.Selection<SVGGElement, unknown, null, undefined>,
    data: ChartDatum[],
    width: number,
    height: number,
    style: ChartStyle,
  ) {
    if (!Array.isArray(data) || data.length === 0) {
      this.showNoDataMessage(g, width, height);
      return;
    }

    const processed = data.map((d, index) => ({
      x: typeof d.x === 'number' ? d.x : index,
      y: Number(d.y ?? d.value ?? 0),
    }));

    const xValues = processed.map((d) => d.x);
    const xExtent = d3.extent(xValues);
    const yMax = d3.max(processed, (d) => d.y) ?? 0;

    const xScale = d3
      .scaleLinear()
      .domain([xExtent[0] ?? 0, xExtent[1] ?? processed.length - 1])
      .range([0, width]);

    const yScale = d3
      .scaleLinear()
      .domain([0, yMax <= 0 ? 1 : yMax])
      .nice()
      .range([height, 0]);

    const area = d3
      .area<{ x: number; y: number }>()
      .x((d) => xScale(d.x))
      .y0(height)
      .y1((d) => yScale(d.y))
      .curve(d3.curveMonotoneX);

    g.append('path')
      .datum(processed)
      .attr('fill', style.colors[0])
      .attr('opacity', 0.55)
      .attr('d', area);

    const line = d3
      .line<{ x: number; y: number }>()
      .x((d) => xScale(d.x))
      .y((d) => yScale(d.y))
      .curve(d3.curveMonotoneX);

    g.append('path')
      .datum(processed)
      .attr('fill', 'none')
      .attr('stroke', style.colors[0])
      .attr('stroke-width', 2.25)
      .attr('d', line);

    this.addXAxis(g, xScale, height, style);
    this.addYAxis(g, yScale, style);
    this.addGridLines(g, xScale, yScale, width, height, style);
  }

  private addXAxis(
    g: d3.Selection<SVGGElement, unknown, null, undefined>,
    xScale: d3.AxisScale<d3.AxisDomain>,
    height: number,
    style: ChartStyle,
  ) {
    const axis = d3.axisBottom(xScale as d3.AxisScale<d3.AxisDomain>);

    g.append('g')
      .attr('class', 'x-axis')
      .attr('transform', `translate(0,${height})`)
      .call(axis)
      .selectAll('text')
      .style('font-family', style.fontFamily)
      .style('font-size', `${style.fontSize}px`)
      .style('fill', style.textColor);

    g.select('.x-axis .domain').style('stroke', style.gridColor);
    g.selectAll('.x-axis .tick line').style('stroke', style.gridColor);
  }

  private addYAxis(
    g: d3.Selection<SVGGElement, unknown, null, undefined>,
    yScale: d3.AxisScale<d3.AxisDomain>,
    style: ChartStyle,
  ) {
    const axis = d3.axisLeft(yScale as d3.AxisScale<d3.AxisDomain>);

    g.append('g')
      .attr('class', 'y-axis')
      .call(axis)
      .selectAll('text')
      .style('font-family', style.fontFamily)
      .style('font-size', `${style.fontSize}px`)
      .style('fill', style.textColor);

    g.select('.y-axis .domain').style('stroke', style.gridColor);
    g.selectAll('.y-axis .tick line').style('stroke', style.gridColor);
  }

  private addGridLines(
    g: d3.Selection<SVGGElement, unknown, null, undefined>,
    xScale: d3.AxisScale<d3.AxisDomain>,
    yScale: d3.AxisScale<d3.AxisDomain>,
    width: number,
    height: number,
    style: ChartStyle,
  ) {
    const hasTicks = typeof (xScale as d3.ScaleLinear<number, number>).ticks === 'function';

    if (hasTicks) {
      const ticks = (xScale as d3.ScaleLinear<number, number>).ticks();
      g.selectAll('.grid-line-vertical')
        .data(ticks)
        .enter()
        .append('line')
        .attr('class', 'grid-line-vertical')
        .attr('x1', (d) => Number((xScale as d3.ScaleLinear<number, number>)(d)))
        .attr('x2', (d) => Number((xScale as d3.ScaleLinear<number, number>)(d)))
        .attr('y1', 0)
        .attr('y2', height)
        .style('stroke', style.gridColor)
        .style('stroke-opacity', 0.3)
        .style('stroke-dasharray', '2,2');
    }

    const yTicks = (yScale as d3.ScaleLinear<number, number>).ticks?.() ?? [];
    g.selectAll('.grid-line-horizontal')
      .data(yTicks)
      .enter()
      .append('line')
      .attr('class', 'grid-line-horizontal')
      .attr('x1', 0)
      .attr('x2', width)
      .attr('y1', (d) => Number((yScale as d3.ScaleLinear<number, number>)(d)))
      .attr('y2', (d) => Number((yScale as d3.ScaleLinear<number, number>)(d)))
      .style('stroke', style.gridColor)
      .style('stroke-opacity', 0.3)
      .style('stroke-dasharray', '2,2');
  }

  private showNoDataMessage(
    g: d3.Selection<SVGGElement, unknown, null, undefined>,
    width: number,
    height: number,
  ) {
    const messageGroup = g
      .append('g')
      .attr('class', 'no-data-message')
      .attr('transform', `translate(${width / 2}, ${height / 2})`);

    messageGroup
      .append('text')
      .attr('text-anchor', 'middle')
      .attr('dy', '-10px')
      .style('font-family', 'Inter, sans-serif')
      .style('font-size', '18px')
      .style('fill', '#9ca3af')
      .text('📊');

    messageGroup
      .append('text')
      .attr('text-anchor', 'middle')
      .attr('dy', '20px')
      .style('font-family', 'Inter, sans-serif')
      .style('font-size', '14px')
      .style('fill', '#6b7280')
      .text('No data available');
  }

  exportChart(container: HTMLElement, format: 'png' | 'svg' = 'png'): string | Promise<string> | null {
    const svg = container.querySelector('svg');
    if (!svg) {
      return null;
    }

    if (format === 'svg') {
      return this.exportAsSVG(svg);
    }

    return this.exportAsPNG(svg);
  }

  private exportAsSVG(svg: SVGSVGElement) {
    const serializer = new XMLSerializer();
    return serializer.serializeToString(svg);
  }

  private exportAsPNG(svg: SVGSVGElement) {
    const canvas = document.createElement('canvas');
    const ctx = canvas.getContext('2d');
    if (!ctx) {
      return null;
    }

    const data = new XMLSerializer().serializeToString(svg);
    const img = new Image();
    const svgBlob = new Blob([data], { type: 'image/svg+xml;charset=utf-8' });
    const url = URL.createObjectURL(svgBlob);

    return new Promise<string>((resolve) => {
      img.onload = () => {
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
