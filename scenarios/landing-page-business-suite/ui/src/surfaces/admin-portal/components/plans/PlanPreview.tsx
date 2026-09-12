import { MetricsModeProvider } from '../../../../shared/hooks/useMetrics';
import { PricingSection } from '../../../public-landing/sections/PricingSection';
import type { PricingPreviewData } from '../../services/pricing.service';

const PRICING_PREVIEW_CONTENT = {
  title: 'Landing pricing preview',
  subtitle: 'Updates instantly with unsaved copy changes so you can validate the three-card layout visitors will see.',
};

interface PlanPreviewProps {
  data: PricingPreviewData;
}

export function PlanPreview({ data }: PlanPreviewProps) {
  let statusMessage = 'Showing live preview of enabled monthly plans.';
  if (data.monthlyCount === 0 && data.placeholderCount > 0) {
    statusMessage = 'No saved monthly plans yet - displaying demo placeholders so the layout stays complete.';
  } else if (data.placeholderCount > 0) {
    statusMessage = `Showing ${String(data.monthlyCount)} saved plan${data.monthlyCount === 1 ? '' : 's'} plus ${String(data.placeholderCount)} demo placeholder${data.placeholderCount === 1 ? '' : 's'} to fill the preview.`;
  } else if (data.monthlyCount > 0) {
    statusMessage = `Showing ${String(data.monthlyCount)} saved monthly plan${data.monthlyCount === 1 ? '' : 's'}.`;
  }

  return (
    <div className="bg-slate-950/60">
      <div className="flex items-center justify-between gap-3">
        <div>
          <p className="text-sm font-semibold text-white">Live Pricing Preview</p>
          <p className="text-xs text-slate-400">{statusMessage}</p>
        </div>
      </div>
      <MetricsModeProvider mode="preview">
        <div className="relative mt-4">
          <div
            className="max-h-[640px] overflow-y-auto rounded-[28px] border border-white/10 bg-surface-darker p-1"
            onClickCapture={(event) => {
              event.preventDefault();
              event.stopPropagation();
            }}
            onKeyDownCapture={(event) => {
              if (event.key === 'Enter' || event.key === ' ') {
                event.preventDefault();
                event.stopPropagation();
              }
            }}
          >
            <div className="pointer-events-none">
              <PricingSection content={PRICING_PREVIEW_CONTENT} pricingOverview={data.overview} />
            </div>
          </div>
        </div>
      </MetricsModeProvider>
    </div>
  );
}
