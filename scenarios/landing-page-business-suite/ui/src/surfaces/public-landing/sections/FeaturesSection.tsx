import { Check, Layers, Target, Database, Zap, Shield, Sparkles } from 'lucide-react';

const iconMap = {
  layers: Layers,
  target: Target,
  database: Database,
  zap: Zap,
  shield: Shield,
  sparkles: Sparkles,
  check: Check,
};

interface Feature {
  title: string;
  description: string;
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
  const features = content.features || [
    {
      title: 'Automation-first OS',
      description: 'Vrooli Ascension builds, retries, and heals workflows so founders ship without engineers.',
      icon: 'zap' as const,
    },
    {
      title: 'Stunning screen recordings',
      description: 'Generate HD walkthroughs and reels directly from your automations—no webcam, no editing.',
      icon: 'sparkles' as const,
    },
    {
      title: 'Future-proof bundle',
      description: 'Silent Founder OS grows into a suite: customer ops, outreach, and analytics plug in as you scale.',
      icon: 'layers' as const,
    },
  ];

  return (
    <section className="border-y border-white/5 bg-[#0F172A] py-24 text-white">
      <div className="container mx-auto px-6">
        <div className="max-w-4xl space-y-4">
          <p className="text-xs uppercase tracking-[0.35em] text-slate-500">System features</p>
          <h2 className="text-4xl font-semibold">
            {content.title || 'The Silent Founder OS: automate ops, export proof, stay in flow'}
          </h2>
          <p className="text-lg text-slate-300">
            {content.subtitle ||
              'Start with Vrooli Ascension, then snap in upcoming business apps. Your assets, automations, and analytics live together.'}
          </p>
        </div>

        <div className="mt-12 grid gap-8 md:grid-cols-2 lg:grid-cols-3">
          {features.map((feature, index) => {
            const IconComponent = feature.icon ? iconMap[feature.icon] ?? Check : Check;
            return (
              <article
                key={feature.title + index}
                className="rounded-3xl border border-white/5 bg-[#1E2433] p-6 text-left transition-transform duration-200 hover:-translate-y-1"
                data-testid={`feature-${index}`}
              >
                <div className="flex h-12 w-12 items-center justify-center rounded-2xl bg-white/5 text-[#F97316]">
                  <IconComponent className="h-5 w-5" />
                </div>
                <h3 className="mt-6 text-xl font-semibold text-white">{feature.title}</h3>
                <p className="mt-3 text-sm text-slate-300">{feature.description}</p>
              </article>
            );
          })}
        </div>

        <div className="mt-12 rounded-3xl border border-white/5 bg-white/5 p-6 text-sm text-slate-300">
          <p>
            Early adopters lock in bundle pricing as we add new modules. Vrooli Ascension is the first drop—customer ops,
            outreach, and analytics ship next without breaking your workflow.
          </p>
        </div>
      </div>
    </section>
  );
}
