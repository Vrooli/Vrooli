import { ArrowRight } from 'lucide-react';
import { Button } from '../../../components/ui/button';
import { useMetrics } from '../../../hooks/useMetrics';

interface HeroSectionProps {
  content: {
    title?: string;
    subtitle?: string;
    cta_text?: string;
    cta_url?: string;
    image_url?: string;
    background_style?: 'gradient' | 'solid' | 'image';
  };
}

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

  return (
    <section className="relative min-h-[90vh] flex items-center justify-center overflow-hidden bg-gradient-to-br from-slate-950 via-purple-950/20 to-slate-950">
      {/* Background decorative elements */}
      <div className="absolute inset-0 bg-mesh-gradient opacity-50"></div>
      <div className="absolute top-20 left-10 w-72 h-72 bg-purple-500/10 rounded-full blur-3xl"></div>
      <div className="absolute bottom-20 right-10 w-96 h-96 bg-blue-500/10 rounded-full blur-3xl"></div>

      <div className="container relative z-10 mx-auto px-6 py-20">
        <div className="max-w-6xl mx-auto">
          <div className="grid lg:grid-cols-2 gap-12 items-center">
            {/* Left column: Content */}
            <div className="space-y-8">
              <h1 className="text-5xl lg:text-6xl xl:text-7xl font-bold leading-tight bg-gradient-to-r from-white via-purple-200 to-blue-200 bg-clip-text text-transparent">
                {content.title || 'Build Your Perfect Landing Page'}
              </h1>
              <p className="text-xl lg:text-2xl text-slate-300 leading-relaxed">
                {content.subtitle || 'Launch faster, convert better, and scale with confidence using our powerful landing page platform.'}
              </p>
              {content.cta_text && (
                <div className="flex flex-wrap gap-4">
                  <Button
                    size="lg"
                    onClick={handleCTAClick}
                    className="text-lg px-8 py-6 bg-gradient-to-r from-purple-600 to-blue-600 hover:from-purple-500 hover:to-blue-500 shadow-xl shadow-purple-500/25"
                    data-testid="hero-cta"
                  >
                    {content.cta_text}
                    <ArrowRight className="ml-2 h-5 w-5" />
                  </Button>
                </div>
              )}
              <div className="flex items-center gap-8 text-sm text-slate-400">
                <div className="flex items-center gap-2">
                  <div className="w-2 h-2 bg-green-400 rounded-full animate-pulse"></div>
                  <span>Live in minutes</span>
                </div>
                <div className="flex items-center gap-2">
                  <div className="w-2 h-2 bg-blue-400 rounded-full animate-pulse"></div>
                  <span>No code required</span>
                </div>
              </div>
            </div>

            {/* Right column: Image/Visual */}
            <div className="relative">
              {content.image_url ? (
                <div className="relative rounded-2xl overflow-hidden shadow-2xl">
                  <img
                    src={content.image_url}
                    alt={content.title || 'Hero'}
                    className="w-full h-auto"
                  />
                  <div className="absolute inset-0 bg-gradient-to-t from-slate-950/50 to-transparent"></div>
                </div>
              ) : (
                <div className="relative aspect-square rounded-2xl bg-gradient-to-br from-purple-500/20 to-blue-500/20 border border-white/10 backdrop-blur p-8 flex items-center justify-center">
                  <div className="text-center space-y-4">
                    <div className="w-32 h-32 mx-auto bg-gradient-to-br from-purple-500 to-blue-500 rounded-2xl opacity-50 blur-xl"></div>
                    <p className="text-slate-400">Add your hero image in customization</p>
                  </div>
                </div>
              )}
            </div>
          </div>
        </div>
      </div>

      {/* Bottom wave decoration */}
      <div className="absolute bottom-0 left-0 right-0 h-32 bg-gradient-to-t from-slate-950 to-transparent"></div>
    </section>
  );
}
