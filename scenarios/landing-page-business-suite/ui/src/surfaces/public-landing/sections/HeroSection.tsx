import { ArrowRight, Layers, LineChart, ShieldCheck, Sparkles, Palette } from 'lucide-react';
import { Button } from '../../../shared/ui/button';
import { useMetrics } from '../../../shared/hooks/useMetrics';
import { getStylingConfig } from '../../../shared/lib/stylingConfig';

interface HeroSectionProps {
  content: {
    title?: string;
    subtitle?: string;
    cta_text?: string;
    cta_url?: string;
    image_url?: string;
    background_style?: 'gradient' | 'solid' | 'image';
    secondary_cta_text?: string;
    secondary_cta_url?: string;
  };
}

const STYLING = getStylingConfig();

export function HeroSection({ content }: HeroSectionProps) {
  const { trackCTAClick } = useMetrics();

  const handleCTAClick = () => {
    if (content.cta_url) {
      trackCTAClick('hero-cta', {
        cta_text: content.cta_text,
        cta_url: content.cta_url,
      });
      window.location.href = content.cta_url;
    }
  };

  const brandPalette = [
    STYLING.palette?.accent_primary ?? '#F97316',
    STYLING.palette?.accent_secondary ?? '#38BDF8',
    STYLING.palette?.surface_alt ?? '#F6F5F2',
    STYLING.palette?.surface_muted ?? '#1E2433',
  ];

  const timelineSteps = [
    {
      label: 'Discover signals',
      copy: 'Variant analytics + download telemetry expose where the funnel leaks.',
    },
    {
      label: 'Customize + stage',
      copy: 'Apply the Clause-grade styling pack, update sections, and preview in generated/.',
    },
    {
      label: 'Launch with proof',
      copy: 'Promote to scenarios/ with metrics + brand assets held in one repo.',
    },
  ];

  return (
    <section className="relative overflow-hidden bg-case-study text-white">
      <div className="absolute inset-0 opacity-20 mix-blend-plus-lighter">
        <div className="noise-overlay absolute inset-0" />
      </div>
      <div className="container relative mx-auto grid gap-12 px-6 py-24 lg:grid-cols-[1.05fr,0.95fr] lg:items-center">
        <div className="space-y-8">
          <span className="inline-flex items-center gap-2 rounded-full border border-white/10 bg-white/5 px-4 py-1 text-xs uppercase tracking-[0.4em] text-slate-400">
            Case Study
            <Sparkles className="h-3.5 w-3.5 text-[#F97316]" />
          </span>
          <div className="space-y-5">
            <h1 className="text-5xl leading-tight text-white md:text-6xl">
              {content.title || 'Landing Manager · Automation Studio handoff'}
            </h1>
            <p className="text-lg text-slate-300">
              {content.subtitle ||
                'A Clause-inspired, enterprise-ready landing page system with layered dashboards, brand guidelines, and download governance baked in.'}
            </p>
          </div>
          {content.cta_text && (
            <div className="flex flex-wrap gap-3">
              <Button
                size="default"
                onClick={handleCTAClick}
                className="gap-2"
                data-testid="hero-cta"
              >
                {content.cta_text}
                <ArrowRight className="h-5 w-5" />
              </Button>
              <Button
                variant="outline"
                size="default"
                asChild
              >
                <a href={content.secondary_cta_url || '#features'}>
                  {content.secondary_cta_text || 'Preview the system'}
                </a>
              </Button>
            </div>
          )}
          <dl className="grid gap-6 rounded-3xl border border-white/5 bg-white/5 p-6 md:grid-cols-3">
            <MetricCard
              icon={<LineChart className="h-5 w-5 text-[#F97316]" />}
              label="Runtime decisions"
              value="12 signals"
              caption="Variant analytics + download gating tracked per visitor."
            />
            <MetricCard
              icon={<Layers className="h-5 w-5 text-[#38BDF8]" />}
              label="UI composition"
              value="3 layered panels"
              caption="Hero showcases stacked dashboards + brand strip."
            />
            <MetricCard
              icon={<ShieldCheck className="h-5 w-5 text-[#10B981]" />}
              label="Entitlement control"
              value="Gated assets"
              caption="Download rail respects plan tier + credit balances."
            />
          </dl>
          <div className="rounded-3xl border border-white/5 bg-white/5 p-6">
            <p className="text-xs uppercase tracking-[0.3em] text-slate-400">Process timeline</p>
            <ol className="mt-4 grid gap-5 md:grid-cols-3">
              {timelineSteps.map((step, index) => (
                <li key={step.label} className="space-y-2">
                  <div className="flex items-center gap-2 text-sm font-semibold text-white">
                    <span className="flex h-8 w-8 items-center justify-center rounded-full bg-white/10 text-sm">
                      {index + 1}
                    </span>
                    {step.label}
                  </div>
                  <p className="text-sm text-slate-300">{step.copy}</p>
                </li>
              ))}
            </ol>
          </div>
        </div>

        <div className="relative">
          <div className="absolute -top-8 -right-6 h-64 w-64 rounded-full bg-[#F97316]/10 blur-3xl" />
          <div className="relative space-y-6">
            <LayeredPanels imageUrl={content.image_url} />
            <div className="grid gap-4 md:grid-cols-[1.2fr,0.8fr]">
              <BrandStrip palette={brandPalette} />
              <PersonaCard />
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}

function MetricCard({ icon, label, value, caption }: { icon: React.ReactNode; label: string; value: string; caption: string }) {
  return (
    <div className="space-y-2">
      <div className="flex items-center gap-2 text-xs uppercase tracking-[0.3em] text-slate-400">
        {icon}
        {label}
      </div>
      <p className="text-3xl font-semibold text-white">{value}</p>
      <p className="text-sm text-slate-300">{caption}</p>
    </div>
  );
}

function LayeredPanels({ imageUrl }: { imageUrl?: string }) {
  return (
    <div className="relative">
      <div className="absolute inset-0 rounded-[28px] bg-[#1E2433] opacity-70 blur-2xl" />
      <div className="relative space-y-5 rounded-[28px] border border-white/10 bg-[#0F172A] p-6 panel-glow">
        <div className="flex items-center justify-between text-sm text-slate-400">
          <span className="inline-flex items-center gap-2">
            <Sparkles className="h-4 w-4 text-[#F97316]" />
            Variant overview
          </span>
          <span className="rounded-full border border-white/10 px-3 py-1 text-xs text-slate-500">Live</span>
        </div>
        <div className="grid gap-4 md:grid-cols-2">
          <div className="rounded-2xl border border-white/5 bg-[#1E2433] p-4">
            <p className="text-xs uppercase tracking-[0.3em] text-slate-500">Traffic split</p>
            <div className="mt-3 space-y-2">
              <div className="flex items-center justify-between text-sm">
                <span>Control</span>
                <span className="text-white">50%</span>
              </div>
              <div className="flex items-center justify-between text-sm">
                <span>Variant A</span>
                <span className="text-white">50%</span>
              </div>
            </div>
            <div className="mt-4 h-2 rounded-full bg-white/5">
              <div className="h-full rounded-full bg-[#F97316]" style={{ width: '50%' }} />
            </div>
          </div>
          <div className="rounded-2xl border border-white/5 bg-[#1E2433] p-4">
            <p className="text-xs uppercase tracking-[0.3em] text-slate-500">Download unlocks</p>
            <div className="mt-4 flex items-baseline gap-2">
              <p className="text-4xl font-semibold text-white">324</p>
              <span className="text-sm text-[#10B981]">+18% WoW</span>
            </div>
            <p className="mt-2 text-sm text-slate-400">Entitlement-backed installers delivered post-conversion.</p>
          </div>
        </div>
        {imageUrl && (
          <div className="rounded-2xl border border-white/5 bg-black/40 p-3">
            <img src={imageUrl} alt="Product preview" className="h-full w-full rounded-xl object-cover" />
          </div>
        )}
      </div>
    </div>
  );
}

function BrandStrip({ palette }: { palette: string[] }) {
  return (
    <div className="rounded-[24px] border border-white/10 bg-white/5 p-5">
      <div className="flex items-center gap-2 text-xs uppercase tracking-[0.3em] text-slate-400">
        <Palette className="h-4 w-4 text-[#38BDF8]" />
        Brand system
      </div>
      <div className="mt-5 flex gap-2">
        {palette.map((color) => (
          <div key={color} className="flex flex-1 flex-col items-center gap-2">
            <span className="h-12 w-full rounded-full" style={{ background: color }} />
            <span className="text-[11px] text-slate-400">{color}</span>
          </div>
        ))}
      </div>
      <p className="mt-4 text-xs text-slate-400">Space Grotesk headlines · Inter body copy</p>
    </div>
  );
}

function PersonaCard() {
  return (
    <div className="rounded-[24px] border border-white/10 bg-white/5 p-5 text-left">
      <p className="text-xs uppercase tracking-[0.3em] text-slate-400">Persona insight</p>
      <h3 className="mt-3 text-lg font-semibold text-white">Ops leader · Demo-led</h3>
      <p className="mt-2 text-sm text-slate-300">
        Highlights download gating, analytics trust, and the generated/ → scenarios/ promotion workflow.
      </p>
      <div className="mt-5 flex flex-wrap gap-2">
        {['Analytics-ready', 'Download governance', 'Clause aesthetic'].map((tag) => (
          <span key={tag} className="rounded-full border border-white/10 px-3 py-1 text-xs text-slate-300">
            {tag}
          </span>
        ))}
      </div>
    </div>
  );
}
