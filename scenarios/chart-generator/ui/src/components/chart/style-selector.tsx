import { Palette } from 'lucide-react';
import * as React from 'react';

import type { ChartStyleId } from '../../lib/chart-engine';
import { cn } from '../../lib/utils';
import { Card } from '../ui/card';

const stylePresets: Array<{
  id: ChartStyleId;
  title: string;
  description: string;
  palette: string[];
}> = [
  {
    id: 'professional',
    title: 'Professional',
    description: 'Executive-ready styling with clean lines and sharp contrast.',
    palette: ['#2563eb', '#64748b', '#f59e0b'],
  },
  {
    id: 'minimal',
    title: 'Minimal',
    description: 'Muted neutrals designed for research reports and documentation.',
    palette: ['#6b7280', '#d1d5db', '#111827'],
  },
  {
    id: 'vibrant',
    title: 'Vibrant',
    description: 'High-energy palette that shines in marketing and presentations.',
    palette: ['#f59e0b', '#8b5cf6', '#10b981'],
  },
];

interface StyleSelectorProps {
  value: ChartStyleId;
  onChange: (value: ChartStyleId) => void;
}

export const StyleSelector: React.FC<StyleSelectorProps> = ({ value, onChange }) => {
  return (
    <div className="grid gap-3">
      {stylePresets.map((preset) => (
        <Card
          key={preset.id}
          role="button"
          tabIndex={0}
          onClick={() => onChange(preset.id)}
          onKeyDown={(event) => {
            if (event.key === 'Enter' || event.key === ' ') {
              event.preventDefault();
              onChange(preset.id);
            }
          }}
          className={cn(
            'flex items-start gap-4 rounded-xl border border-transparent bg-white/90 p-4 shadow-sm transition hover:-translate-y-0.5 hover:shadow-md focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-300',
            value === preset.id && 'border-brand-200 bg-brand-50 shadow-brand',
          )}
        >
          <div
            className={cn(
              'flex h-10 w-10 items-center justify-center rounded-lg bg-brand-100 text-brand-700',
              value === preset.id && 'bg-brand-500 text-white shadow-md',
            )}
          >
            <Palette className="h-5 w-5" />
          </div>
          <div className="flex-1 space-y-2">
            <div>
              <p className="text-sm font-semibold text-foreground">{preset.title}</p>
              <p className="text-xs text-muted-foreground leading-snug">{preset.description}</p>
            </div>
            <div className="flex items-center gap-2">
              {preset.palette.map((color) => (
                <span
                  key={color}
                  className="h-2 flex-1 rounded-full"
                  style={{ backgroundColor: color }}
                />
              ))}
            </div>
          </div>
        </Card>
      ))}
    </div>
  );
};
