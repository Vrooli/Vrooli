import { Check, Zap, Shield, Sparkles } from 'lucide-react';

const iconMap = {
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
      title: 'Lightning Fast',
      description: 'Built for speed with optimized performance and minimal bundle size.',
      icon: 'zap' as const,
    },
    {
      title: 'Secure by Default',
      description: 'Industry-standard security practices built into every feature.',
      icon: 'shield' as const,
    },
    {
      title: 'AI-Powered',
      description: 'Smart customization and optimization powered by advanced AI.',
      icon: 'sparkles' as const,
    },
  ];

  return (
    <section className="py-24 bg-slate-950/50 relative overflow-hidden">
      {/* Background decoration */}
      <div className="absolute top-0 left-1/4 w-96 h-96 bg-purple-500/5 rounded-full blur-3xl"></div>
      <div className="absolute bottom-0 right-1/4 w-96 h-96 bg-blue-500/5 rounded-full blur-3xl"></div>

      <div className="container relative z-10 mx-auto px-6">
        <div className="max-w-6xl mx-auto">
          {/* Header */}
          <div className="text-center mb-16 space-y-4">
            <h2 className="text-4xl lg:text-5xl font-bold bg-gradient-to-r from-white to-slate-300 bg-clip-text text-transparent">
              {content.title || 'Powerful Features'}
            </h2>
            <p className="text-xl text-slate-400 max-w-2xl mx-auto">
              {content.subtitle || 'Everything you need to build, launch, and optimize your landing pages.'}
            </p>
          </div>

          {/* Features Grid */}
          <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-8">
            {features.map((feature, index) => {
              const IconComponent = feature.icon ? iconMap[feature.icon] : Check;
              return (
                <div
                  key={index}
                  className="group relative rounded-2xl border border-white/10 bg-gradient-to-br from-white/5 to-white/0 p-8 hover:border-purple-500/50 hover:bg-white/10 transition-all duration-300"
                  data-testid={`feature-${index}`}
                >
                  <div className="absolute inset-0 bg-gradient-to-br from-purple-500/0 to-blue-500/0 group-hover:from-purple-500/10 group-hover:to-blue-500/10 rounded-2xl transition-all duration-300"></div>
                  <div className="relative">
                    <div className="w-14 h-14 rounded-xl bg-gradient-to-br from-purple-500/20 to-blue-500/20 flex items-center justify-center mb-6 group-hover:scale-110 transition-transform duration-300">
                      <IconComponent className="w-7 h-7 text-purple-400" />
                    </div>
                    <h3 className="text-xl font-semibold mb-3 text-white">
                      {feature.title}
                    </h3>
                    <p className="text-slate-400 leading-relaxed">
                      {feature.description}
                    </p>
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      </div>
    </section>
  );
}
