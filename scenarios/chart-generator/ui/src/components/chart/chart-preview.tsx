import { Lightbulb } from 'lucide-react';
import * as React from 'react';

import type { ChartDatum, ChartEngine, ChartStyleId, ChartType } from '../../lib/chart-engine';
import { cn } from '../../lib/utils';

interface ChartPreviewProps {
  engine: ChartEngine;
  chartType: ChartType;
  styleId: ChartStyleId;
  data: ChartDatum[];
  className?: string;
}

export const ChartPreview = React.forwardRef<HTMLDivElement, ChartPreviewProps>(
  ({ engine, chartType, styleId, data, className }, ref) => {
    const containerRef = React.useRef<HTMLDivElement | null>(null);

    React.useEffect(() => {
      if (!containerRef.current) {
        return;
      }

      if (!data || data.length === 0) {
        containerRef.current.innerHTML = '';
        return;
      }

      engine.generateChart(containerRef.current, chartType, data, styleId);
    }, [data, chartType, styleId, engine]);

    React.useImperativeHandle(ref, () => containerRef.current as HTMLDivElement);

    const hasData = data && data.length > 0;

    return (
      <div
        data-testid="chart-preview"
        className={cn(
          'relative flex h-[420px] min-h-[320px] w-full items-center justify-center overflow-hidden rounded-2xl border border-border bg-white/90 shadow-lg backdrop-blur',
          className,
        )}
      >
        <div ref={containerRef} className="h-full w-full" data-testid="chart-svg" />
        {!hasData && (
          <div className="absolute inset-0 flex flex-col items-center justify-center gap-2 text-center text-muted-foreground">
            <Lightbulb className="h-9 w-9 text-brand-500" />
            <p className="text-base font-semibold text-foreground">Ready to bring your data to life</p>
            <p className="max-w-sm text-sm text-muted-foreground">
              Choose a chart type, paste JSON data, or start with a sample dataset to preview professional-grade
              visualisations instantly.
            </p>
          </div>
        )}
      </div>
    );
  },
);

ChartPreview.displayName = 'ChartPreview';
