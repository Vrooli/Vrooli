import type { ChartDatum, ChartType } from './chart-engine';

export type SampleDataKey = 'sales' | 'revenue' | 'performance' | 'growth' | 'correlation';

export const sampleDataMap: Record<SampleDataKey, ChartDatum[]> = {
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
  ],
  growth: [
    { x: '2019', y: 18 },
    { x: '2020', y: 25 },
    { x: '2021', y: 32 },
    { x: '2022', y: 41 },
    { x: '2023', y: 57 }
  ],
  correlation: [
    { x: 12, y: 90 },
    { x: 18, y: 82 },
    { x: 22, y: 74 },
    { x: 30, y: 63 },
    { x: 38, y: 54 },
    { x: 45, y: 41 }
  ]
};

export const chartTypeSamples: Record<ChartType, SampleDataKey> = {
  bar: 'sales',
  line: 'growth',
  area: 'growth',
  pie: 'performance',
  scatter: 'correlation'
};
