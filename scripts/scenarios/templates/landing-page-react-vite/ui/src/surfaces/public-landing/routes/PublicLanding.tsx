import { useMemo } from 'react';
import { useLandingVariant } from '../../../app/providers/LandingVariantProvider';
import type { LandingSection } from '../../../shared/api';
import { HeroSection } from '../sections/HeroSection';
import { FeaturesSection } from '../sections/FeaturesSection';
import { PricingSection } from '../sections/PricingSection';
import { CTASection } from '../sections/CTASection';
import { TestimonialsSection } from '../sections/TestimonialsSection';
import { FAQSection } from '../sections/FAQSection';
import { FooterSection } from '../sections/FooterSection';
import { VideoSection } from '../sections/VideoSection';
import { DownloadSection } from '../sections/DownloadSection';

/**
 * Public Landing Page
 * Renders content sections based on selected variant
 *
 * Implements:
 * - OT-P0-014: URL variant selection
 * - OT-P0-015: localStorage persistence
 * - OT-P0-016: API weight-based selection
 * - OT-P0-019: Event variant tagging (via useMetrics in sections)
 * - OT-P0-021: Minimum event coverage (page_view, scroll, clicks)
 */
export function PublicLanding() {
  const { variant, config, loading: configLoading, error: configError } = useLandingVariant();
  const fallbackActive = Boolean(config?.fallback);

  const sections = useMemo(() => {
    const incoming = config?.sections ?? [];
    return incoming
      .filter((section) => section.enabled !== false)
      .sort((a, b) => a.order - b.order);
  }, [config]);

  if (configLoading) {
    return (
      <div className="min-h-screen bg-slate-950 flex items-center justify-center">
        <div className="text-center space-y-4">
          <div className="w-16 h-16 border-4 border-purple-500 border-t-transparent rounded-full animate-spin mx-auto"></div>
          <p className="text-slate-400">Loading your landing page...</p>
        </div>
      </div>
    );
  }

  if (configError && !fallbackActive) {
    return (
      <div className="min-h-screen bg-slate-950 flex items-center justify-center px-6">
        <div className="max-w-md w-full rounded-2xl border border-red-500/20 bg-red-500/10 p-8 text-center space-y-4">
          <div className="w-16 h-16 rounded-full bg-red-500/20 flex items-center justify-center mx-auto">
            <svg className="w-8 h-8 text-red-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </div>
          <h1 className="text-2xl font-bold text-red-400">Failed to Load Variant</h1>
          <p className="text-red-300">{configError}</p>
          <button
            onClick={() => window.location.reload()}
            className="px-6 py-2 bg-red-500/20 hover:bg-red-500/30 rounded-lg transition-colors text-red-300"
          >
            Retry
          </button>
        </div>
      </div>
    );
  }

  if (!variant || !config) {
    return (
      <div className="min-h-screen bg-slate-950 flex items-center justify-center px-6">
        <div className="max-w-md w-full rounded-2xl border border-yellow-500/20 bg-yellow-500/10 p-8 text-center space-y-4">
          <div className="w-16 h-16 rounded-full bg-yellow-500/20 flex items-center justify-center mx-auto">
            <svg className="w-8 h-8 text-yellow-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
            </svg>
          </div>
          <h1 className="text-2xl font-bold text-yellow-400">No Variant Available</h1>
          <p className="text-yellow-300">Please create a variant in the admin panel to view this landing page.</p>
        </div>
      </div>
    );
  }

  const renderSection = (section: LandingSection) => {
    const key = section.id ?? `${section.section_type}-${section.order}`;
    const commonProps = {
      key,
      content: section.content,
    };

    switch (section.section_type) {
      case 'hero':
        return <HeroSection {...commonProps} />;
      case 'features':
        return <FeaturesSection {...commonProps} />;
      case 'pricing':
        return <PricingSection {...commonProps} pricingOverview={config?.pricing} />;
      case 'cta':
        return <CTASection {...commonProps} />;
      case 'testimonials':
        return <TestimonialsSection {...commonProps} />;
      case 'faq':
        return <FAQSection {...commonProps} />;
      case 'footer':
        return <FooterSection {...commonProps} />;
      case 'video':
        return <VideoSection {...commonProps} />;
      default:
        console.warn(`Unknown section type: ${section.section_type}`);
        return null;
    }
  };

  return (
    <div className="min-h-screen bg-slate-950 text-slate-50">
      {fallbackActive && (
        <div className="bg-amber-500/10 border border-amber-500/30 text-amber-100 text-sm py-3 px-4 text-center">
          Offline-safe fallback variant is active. Live analytics, pricing, and downloads may be outdated until the API recovers.
        </div>
      )}

      {/* Debug info in dev mode (only visible with ?debug=1) */}
      {new URLSearchParams(window.location.search).get('debug') === '1' && (
        <div className="fixed top-4 right-4 z-50 max-w-xs rounded-lg border border-yellow-500/20 bg-yellow-500/10 p-4 text-xs backdrop-blur">
          <div className="font-semibold text-yellow-400 mb-2">Debug Info</div>
          <div className="space-y-1 text-yellow-200">
            <div>Variant: {variant.name} ({variant.slug})</div>
            <div>Sections: {sections.length}</div>
            <div>Status: {variant.status}</div>
            {fallbackActive && <div>Fallback Variant Active</div>}
          </div>
        </div>
      )}

      {/* Render all sections in order */}
      {sections.length > 0 ? (
        sections.map(renderSection)
      ) : (
        <div className="min-h-screen flex items-center justify-center px-6">
          <div className="max-w-md w-full rounded-2xl border border-white/10 bg-white/5 p-8 text-center space-y-4">
            <h1 className="text-2xl font-bold">No Content Yet</h1>
            <p className="text-slate-300">
              This variant ({variant.name}) doesn't have any sections yet.
              Visit the admin panel to add content sections.
            </p>
          </div>
        </div>
      )}
      {config?.downloads && config.downloads.length > 0 && (
        <DownloadSection
          downloads={config.downloads}
          content={{
            title: 'Download Browser Automation Studio',
            subtitle: 'Install on Windows, macOS, or Linux while we verify your subscription entitlements.',
          }}
        />
      )}
    </div>
  );
}
