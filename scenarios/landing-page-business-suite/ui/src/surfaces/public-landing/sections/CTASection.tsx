import { ArrowRight, BellOff } from 'lucide-react';
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
    <section className="relative overflow-hidden bg-[#0B1020] py-20 text-white">
      <div className="pointer-events-none absolute inset-0 opacity-70">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_20%_20%,rgba(124,58,237,0.12),transparent_35%),radial-gradient(circle_at_80%_10%,rgba(14,165,233,0.12),transparent_32%),radial-gradient(circle_at_50%_80%,rgba(249,115,22,0.12),transparent_32%)]" />
      </div>
      <div className="container relative mx-auto px-6">
        <div className="relative overflow-hidden rounded-[32px] border border-white/10 bg-gradient-to-br from-[#0E1428] via-[#0C1224] to-[#0B1120] p-10 shadow-[0_40px_120px_-50px_rgba(0,0,0,0.7)] lg:p-14">
          <div className="absolute inset-x-0 top-0 h-1 bg-gradient-to-r from-[#F97316] via-[#7C3AED] to-[#0EA5E9]" />
          <div className="grid gap-10 lg:grid-cols-[1.2fr_0.8fr] lg:items-center">
            <div className="space-y-5">
              <div className="inline-flex items-center gap-2 rounded-full border border-white/10 bg-white/5 px-3 py-2 text-xs font-semibold uppercase tracking-[0.25em] text-slate-300">
                <BellOff className="h-4 w-4 text-slate-200" />
                Silent founder mode
              </div>
              <div className="space-y-3">
                <h2 className="text-4xl font-semibold leading-tight">
                  {content.title || 'See your ops and marketing run themselves'}
                </h2>
                <p className="text-lg text-slate-200">
                  {content.subtitle ||
                    'Launch the Silent Founder OS without meetings. Automate a dreaded tab and ship a production-ready asset quietly.'}
                </p>
              </div>
              <ul className="grid gap-2 text-sm text-slate-300 sm:grid-cols-3">
                <li className="flex items-center gap-2 rounded-xl border border-white/5 bg-white/5 px-3 py-2">
                  <span className="h-2 w-2 rounded-full bg-emerald-400" />
                  Setup in silent mode
                </li>
                <li className="flex items-center gap-2 rounded-xl border border-white/5 bg-white/5 px-3 py-2">
                  <span className="h-2 w-2 rounded-full bg-amber-400" />
                  Automate a dreaded tab
                </li>
                <li className="flex items-center gap-2 rounded-xl border border-white/5 bg-white/5 px-3 py-2">
                  <span className="h-2 w-2 rounded-full bg-sky-400" />
                  Ship your first asset quietly
                </li>
              </ul>
            </div>

            <div className="space-y-4 rounded-2xl border border-white/10 bg-white/5 p-6">
              <Button
                size="default"
                onClick={handleCTAClick}
                className="h-auto w-full justify-center gap-2 px-6 py-4 text-base font-semibold shadow-lg shadow-orange-500/25 transition hover:-translate-y-0.5 hover:shadow-orange-500/35"
                data-testid="cta-button"
              >
                {content.cta_text || 'Start with Vrooli Ascension'}
                <ArrowRight className="h-5 w-5" />
              </Button>
              <p className="text-sm text-slate-300">
                Bring a browser tab you dread opening. We’ll automate it and ship the first asset without calls or demos.
              </p>
              <div className="flex flex-wrap items-center gap-3 text-xs uppercase tracking-[0.25em] text-slate-400">
                <span className="flex items-center gap-2">
                  <span className="h-1.5 w-1.5 rounded-full bg-emerald-400" />
                  No meetings
                </span>
                <span className="h-1 w-1 rounded-full bg-slate-600" />
                <span className="flex items-center gap-2">
                  <span className="h-1.5 w-1.5 rounded-full bg-amber-400" />
                  ~30 minutes setup
                </span>
                <span className="h-1 w-1 rounded-full bg-slate-600" />
                <span className="flex items-center gap-2">
                  <span className="h-1.5 w-1.5 rounded-full bg-sky-400" />
                  Identity optional
                </span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
