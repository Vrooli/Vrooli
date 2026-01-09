import { useEffect, useMemo, useState } from 'react';
import { Check, Layers, Target, Database, Zap, Shield, Sparkles, Rocket, Video, BarChart3 } from 'lucide-react';
import { Button } from '../../../shared/ui/button';

const iconMap = {
  layers: Layers,
  target: Target,
  database: Database,
  zap: Zap,
  shield: Shield,
  sparkles: Sparkles,
  check: Check,
  rocket: Rocket,
  video: Video,
  growth: BarChart3,
};

interface Feature {
  title: string;
  description: string;
  bullets?: string[];
  icon?: keyof typeof iconMap;
}

interface FeaturesSectionProps {
  content: {
    title?: string;
    subtitle?: string;
    features?: Feature[];
  };
}

export function FeaturesSection({ content }: FeaturesSectionProps) {
  const defaultCategories = useMemo(
    () => [
      {
        id: 'automation',
        label: 'Automation',
        features: [
          {
            title: 'Automation-first OS',
            description: 'Vrooli Ascension builds, retries, and heals workflows so founders ship without engineers.',
            bullets: ['Autopilot retries', 'Observability baked-in', 'No-code overrides'],
            icon: 'zap' as const,
          },
          {
            title: 'Deterministic playbooks',
            description: 'Structured runs that keep human-in-the-loop when needed.',
            bullets: ['Step gating', 'Audit-ready logs', 'Restart from checkpoints'],
            icon: 'shield' as const,
          },
          {
            title: 'Resource-aware orchestration',
            description: 'Makes the most of local models, storage, and browser automation.',
            bullets: ['Local-first', 'Smart throttling', 'Resume after failures'],
            icon: 'database' as const,
          },
        ],
      },
      {
        id: 'recording',
        label: 'Recording',
        features: [
          {
            title: 'Stunning screen recordings',
            description: 'Generate HD walkthroughs and reels directly from your automations—no webcam required.',
            bullets: ['Studio-grade exports', 'Auto-caption & trim', 'Sharable reels'],
            icon: 'video' as const,
          },
          {
            title: 'Narrate as you build',
            description: 'Inline notes and highlights become ready-to-share proof of work.',
            bullets: ['Inline voice notes', 'Branded lower-thirds', 'AI polish pass'],
            icon: 'sparkles' as const,
          },
          {
            title: 'Secure delivery',
            description: 'Share links that inherit your access rules and expirations.',
            bullets: ['Watermarking', 'Access windows', 'Analytics on opens'],
            icon: 'shield' as const,
          },
        ],
      },
      {
        id: 'growth',
        label: 'Growth Suite',
        features: [
          {
            title: 'Customer ops, ready soon',
            description: 'Next drops snap in without rebuilding your stack.',
            bullets: ['Inbox triage', 'Lead outreach', 'Human-quality summaries'],
            icon: 'target' as const,
          },
          {
            title: 'Data that compounds',
            description: 'Usage, wins, and failures feed back into better defaults.',
            bullets: ['Closed-loop telemetry', 'Variant testing', 'Auto-tuned prompts'],
            icon: 'growth' as const,
          },
          {
            title: 'Founders keep control',
            description: 'Local-first footprint, deployable anywhere, never locked in.',
            bullets: ['Own the stack', 'Offline-safe', 'Deploy to your infra'],
            icon: 'rocket' as const,
          },
        ],
      },
    ],
    [],
  );

  const categories = useMemo(() => {
    if (content.features && content.features.length > 0) {
      return [
        {
          id: 'highlights',
          label: 'Highlights',
          features: content.features,
        },
        ...defaultCategories,
      ];
    }
    return defaultCategories;
  }, [content.features, defaultCategories]);

  const [activeTab, setActiveTab] = useState(categories[0]?.id ?? 'automation');
  const [animateCards, setAnimateCards] = useState(false);

  useEffect(() => {
    setActiveTab(categories[0]?.id ?? 'automation');
  }, [categories]);

  useEffect(() => {
    setAnimateCards(true);
  }, []);

  const activeFeatures = categories.find((category) => category.id === activeTab)?.features ?? [];

  return (
    <section className="border-y border-white/5 bg-[#0F172A] py-24 text-white">
      <div className="container mx-auto px-6">
        <div className="max-w-5xl space-y-4">
          <p className="text-xs uppercase tracking-[0.35em] text-slate-500">System features</p>
          <div className="space-y-2">
            <h2 className="text-4xl font-semibold">
              {content.title || 'The Silent Founder OS: automate ops, export proof, stay in flow'}
            </h2>
            <p className="text-lg text-slate-300">
              {content.subtitle ||
                'Start with Vrooli Ascension, then snap in upcoming business apps. Your assets, automations, and analytics live together.'}
            </p>
          </div>
        </div>

        <div className="mt-12 grid gap-8 lg:grid-cols-[1.8fr_0.9fr]">
          <div className="space-y-6">
            <div className="flex flex-wrap gap-2 rounded-2xl border border-white/10 bg-white/5 p-2">
              {categories.map((category) => (
                <button
                  key={category.id}
                  type="button"
                  onClick={() => setActiveTab(category.id)}
                  className={`rounded-xl px-4 py-2 text-sm font-semibold transition focus:outline-none focus:ring-2 focus:ring-[#F97316] ${
                    activeTab === category.id
                      ? 'bg-[#F97316] text-white shadow-lg'
                      : 'bg-transparent text-slate-200 hover:bg-white/10'
                  }`}
                >
                  {category.label}
                </button>
              ))}
            </div>

            <div className="grid gap-6 md:grid-cols-2 xl:grid-cols-3">
              {activeFeatures.map((feature, index) => {
                const IconComponent = feature.icon ? iconMap[feature.icon] ?? Check : Check;
                const bullets = feature.bullets && feature.bullets.length > 0 ? feature.bullets : [feature.description];
                return (
                  <article
                    key={feature.title + index}
                    className={`group relative overflow-hidden rounded-3xl border border-white/10 bg-gradient-to-br from-white/5 via-[#161b2b] to-[#0e1324] p-6 text-left transition duration-300 hover:-translate-y-1 hover:border-[#7C3AED] hover:shadow-[0_0_0_1px_rgba(124,58,237,0.4),0_25px_60px_-30px_rgba(0,0,0,0.75)] ${
                      animateCards ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-3'
                    }`}
                    style={{ transitionDelay: `${index * 60}ms` }}
                    data-testid={`feature-${index}`}
                  >
                    <div className="pointer-events-none absolute inset-0 opacity-0 transition-opacity duration-300 group-hover:opacity-100">
                      <div className="absolute inset-0 bg-[radial-gradient(circle_at_20%_20%,rgba(124,58,237,0.15),transparent_35%),radial-gradient(circle_at_80%_0%,rgba(14,165,233,0.12),transparent_35%)]" />
                    </div>
                    <div className="relative">
                      <div className="flex h-12 w-12 items-center justify-center rounded-2xl bg-white/5 text-[#F97316] shadow-inner shadow-black/30">
                        <IconComponent className="h-5 w-5" />
                      </div>
                      <h3 className="mt-6 text-xl font-semibold text-white">{feature.title}</h3>
                      <ul className="mt-3 space-y-2 text-sm text-slate-300">
                        {bullets.map((item, bulletIndex) => (
                          <li key={`${feature.title}-bullet-${bulletIndex}`} className="flex items-start gap-2">
                            <span className="mt-1 h-1.5 w-1.5 rounded-full bg-[#F97316] opacity-80" />
                            <span>{item}</span>
                          </li>
                        ))}
                      </ul>
                    </div>
                  </article>
                );
              })}
            </div>
          </div>

          <aside className="space-y-4 rounded-3xl border border-white/10 bg-white/5 p-6">
            <p className="text-xs uppercase tracking-[0.3em] text-slate-400">Proof & momentum</p>
            <div className="space-y-3">
              {[
                { label: '87% tasks auto-resolved', desc: 'Retries, guardrails, and human-in-loop when needed.' },
                { label: '<90s to ship new flow', desc: 'Record, edit, deploy—no engineers required.' },
                { label: 'Zero cloud dependency', desc: 'Local-first footprint; deploy where you run.' },
              ].map((stat) => (
                <div
                  key={stat.label}
                  className="rounded-2xl border border-white/10 bg-[#0C1224] px-4 py-3 text-sm text-slate-200 shadow-[0_20px_40px_-28px_rgba(0,0,0,0.7)]"
                >
                  <p className="text-base font-semibold text-white">{stat.label}</p>
                  <p className="text-xs text-slate-400">{stat.desc}</p>
                </div>
              ))}
            </div>
            <div className="flex flex-col gap-3 rounded-2xl border border-white/10 bg-white/5 p-4">
              <p className="text-sm text-slate-200">See how it works, from first automation to downloads.</p>
              <div className="flex flex-wrap gap-3">
                <Button asChild variant="default" size="sm" className="px-4">
                  <a href="#downloads-section">Preview Ascension</a>
                </Button>
                <Button asChild variant="muted" size="sm" className="px-4">
                  <a href="#pricing">See roadmap</a>
                </Button>
              </div>
            </div>
          </aside>
        </div>

        <div className="mt-12 rounded-3xl border border-white/10 bg-white/5 p-6 text-sm text-slate-300">
          <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
            <p>Consistent pricing, no matter how many features and apps we ship. That's the Vrooli guarantee.</p>
            <div className="flex gap-3">
              <Button variant="default" size="sm">Preview Ascension</Button>
              <Button asChild variant="outline" size="sm" className="border-white/20 text-white hover:border-white/40">
                <a href="#downloads-section">Jump to downloads</a>
              </Button>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
