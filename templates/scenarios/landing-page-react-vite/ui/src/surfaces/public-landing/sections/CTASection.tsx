import { ArrowRight } from 'lucide-react';
import { Button } from '../../../shared/ui/button';
import { useMetrics } from '../../../shared/hooks/useMetrics';

interface CTASectionProps {
  content: {
    title?: string;
    subtitle?: string;
    cta_text?: string;
    cta_url?: string;
    background_style?: 'gradient' | 'solid';
  };
}

export function CTASection({ content }: CTASectionProps) {
  const { trackCTAClick } = useMetrics();

  const handleCTAClick = () => {
    if (content.cta_url) {
      trackCTAClick('cta-section', {
        cta_text: content.cta_text,
        cta_url: content.cta_url,
      });
      window.location.href = content.cta_url;
    }
  };

  return (
    <section className="bg-[#0F172A] py-20 text-white">
      <div className="container mx-auto px-6">
        <div className="rounded-[32px] border border-white/10 bg-[#07090F] p-10 lg:p-14">
          <div className="flex flex-col gap-10 lg:flex-row lg:items-center lg:justify-between">
            <div className="space-y-4">
              <p className="text-xs uppercase tracking-[0.35em] text-slate-500">Next step</p>
              <h2 className="text-4xl font-semibold leading-tight">
                {content.title || 'Book a live review of your variant stack'}
              </h2>
              <p className="text-lg text-slate-300">
                {content.subtitle ||
                  'Walk through analytics, styling.json guardrails, and staged downloads with an operator who ships Clause-grade experiences every week.'}
              </p>
            </div>
            {content.cta_text && (
              <div className="flex flex-col gap-3">
                <Button
                  size="default"
                  onClick={handleCTAClick}
                  className="gap-2 px-10 py-4"
                  data-testid="cta-button"
                >
                  {content.cta_text}
                  <ArrowRight className="h-5 w-5" />
                </Button>
                <p className="text-sm text-slate-400">
                  Includes a full style-pack export, variant audit, and download entitlement verification.
                </p>
              </div>
            )}
          </div>
        </div>
      </div>
    </section>
  );
}
