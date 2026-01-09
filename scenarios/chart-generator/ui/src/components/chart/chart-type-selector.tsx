import { ChartArea, ChartBar, ChartLine, ChartPie, ChartScatter } from 'lucide-react';
import * as React from 'react';

import type { ChartType } from '../../lib/chart-engine';
import { cn } from '../../lib/utils';
import { Card } from '../ui/card';

const chartTypeIconMap: Record<ChartType, React.ReactNode> = {
  bar: <ChartBar className="h-5 w-5" />,
  line: <ChartLine className="h-5 w-5" />,
  pie: <ChartPie className="h-5 w-5" />,
  scatter: <ChartScatter className="h-5 w-5" />,
  area: <ChartArea className="h-5 w-5" />,
};

const chartTypeCopy: Record<ChartType, { title: string; description: string }> = {
  bar: { title: 'Bar chart', description: 'Compare categories with crisp columns.' },
  line: { title: 'Line chart', description: 'Track performance trends over time.' },
  pie: { title: 'Pie chart', description: 'Show proportional composition.' },
  scatter: { title: 'Scatter plot', description: 'Highlight correlations and clusters.' },
  area: { title: 'Area chart', description: 'Emphasise magnitude and accumulation.' },
};

interface ChartTypeSelectorProps {
  value: ChartType;
  onChange: (value: ChartType) => void;
}

export const ChartTypeSelector: React.FC<ChartTypeSelectorProps> = ({ value, onChange }) => {
  return (
    <div className="grid gap-3">
      {Object.keys(chartTypeIconMap).map((key) => {
        const type = key as ChartType;
        const copy = chartTypeCopy[type];

        return (
          <Card
            key={type}
            role="button"
            tabIndex={0}
            onClick={() => onChange(type)}
            onKeyDown={(event) => {
              if (event.key === 'Enter' || event.key === ' ') {
                event.preventDefault();
                onChange(type);
              }
            }}
            data-testid={`chart-type-${type}`}
            className={cn(
              'flex items-start gap-4 rounded-xl border border-transparent bg-white/80 p-4 shadow-sm transition hover:-translate-y-0.5 hover:shadow-md focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-300',
              value === type && 'border-brand-200 bg-brand-50 shadow-brand',
            )}
          >
            <div
              className={cn(
                'flex h-10 min-h-[40px] w-10 min-w-[40px] items-center justify-center rounded-lg bg-brand-100 text-brand-700',
                value === type && 'bg-brand-500 text-white shadow-md',
              )}
            >
              {chartTypeIconMap[type]}
            </div>
            <div className="space-y-1">
              <p className="text-sm font-semibold text-foreground">{copy.title}</p>
              <p className="text-xs text-muted-foreground leading-snug">{copy.description}</p>
            </div>
          </Card>
        );
      })}
    </div>
  );
};
