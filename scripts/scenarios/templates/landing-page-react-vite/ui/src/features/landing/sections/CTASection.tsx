import { ArrowRight } from 'lucide-react';
import { Button } from '../../../components/ui/button';
import { useMetrics } from '../../../hooks/useMetrics';

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
    <section className="py-24 bg-gradient-to-br from-purple-900/30 via-slate-950 to-blue-900/30 relative overflow-hidden">
      {/* Animated background elements */}
      <div className="absolute inset-0">
        <div className="absolute top-0 left-1/4 w-96 h-96 bg-purple-500/20 rounded-full blur-3xl animate-pulse"></div>
        <div className="absolute bottom-0 right-1/4 w-96 h-96 bg-blue-500/20 rounded-full blur-3xl animate-pulse" style={{ animationDelay: '1s' }}></div>
      </div>

      <div className="container relative z-10 mx-auto px-6">
        <div className="max-w-4xl mx-auto">
          <div className="rounded-3xl border border-white/20 bg-gradient-to-br from-white/10 to-white/5 p-12 lg:p-16 backdrop-blur shadow-2xl">
            <div className="text-center space-y-8">
              <h2 className="text-4xl lg:text-5xl xl:text-6xl font-bold leading-tight bg-gradient-to-r from-white via-purple-200 to-blue-200 bg-clip-text text-transparent">
                {content.title || 'Ready to Get Started?'}
              </h2>
              <p className="text-xl lg:text-2xl text-slate-300 max-w-2xl mx-auto leading-relaxed">
                {content.subtitle || 'Join thousands of teams building better landing pages with our platform.'}
              </p>
              {content.cta_text && (
                <div className="pt-4">
                  <Button
                    size="lg"
                    onClick={handleCTAClick}
                    className="text-lg px-10 py-7 bg-gradient-to-r from-purple-600 to-blue-600 hover:from-purple-500 hover:to-blue-500 shadow-xl shadow-purple-500/25 transform hover:scale-105 transition-all duration-300"
                    data-testid="cta-button"
                  >
                    {content.cta_text}
                    <ArrowRight className="ml-2 h-6 w-6" />
                  </Button>
                </div>
              )}
              <div className="flex flex-wrap items-center justify-center gap-8 text-sm text-slate-400 pt-4">
                <div className="flex items-center gap-2">
                  <div className="w-2 h-2 bg-green-400 rounded-full"></div>
                  <span>14-day free trial</span>
                </div>
                <div className="flex items-center gap-2">
                  <div className="w-2 h-2 bg-blue-400 rounded-full"></div>
                  <span>No credit card required</span>
                </div>
                <div className="flex items-center gap-2">
                  <div className="w-2 h-2 bg-purple-400 rounded-full"></div>
                  <span>Cancel anytime</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
