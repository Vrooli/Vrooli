import { useEffect, useState } from 'react';
import { useVariant } from '../contexts/VariantContext';
import { getSections, type ContentSection } from '../lib/api';
import { HeroSection } from '../components/sections/HeroSection';
import { FeaturesSection } from '../components/sections/FeaturesSection';
import { PricingSection } from '../components/sections/PricingSection';
import { CTASection } from '../components/sections/CTASection';
import { TestimonialsSection } from '../components/sections/TestimonialsSection';
import { FAQSection } from '../components/sections/FAQSection';
import { FooterSection } from '../components/sections/FooterSection';
import { VideoSection } from '../components/sections/VideoSection';

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
export default function PublicHome() {
  const { variant, loading: variantLoading, error: variantError } = useVariant();
  const [sections, setSections] = useState<ContentSection[]>([]);
  const [sectionsLoading, setSectionsLoading] = useState(true);
  const [sectionsError, setSectionsError] = useState<string | null>(null);

  useEffect(() => {
    async function fetchSections() {
      if (!variant) {
        setSectionsLoading(false);
        return;
      }

      try {
        setSectionsLoading(true);
        const data = await getSections(parseInt(variant.id));
        // Sort by order and filter enabled sections
        const enabledSections = data.sections
          .filter((s) => s.enabled)
          .sort((a, b) => a.order - b.order);
        setSections(enabledSections);
        setSectionsError(null);
      } catch (err) {
        console.error('Failed to fetch sections:', err);
        setSectionsError(err instanceof Error ? err.message : 'Unknown error');
      } finally {
        setSectionsLoading(false);
      }
    }

    fetchSections();
  }, [variant?.id]);

  // Loading state
  if (variantLoading || sectionsLoading) {
    return (
      <div className="min-h-screen bg-slate-950 flex items-center justify-center">
        <div className="text-center space-y-4">
          <div className="w-16 h-16 border-4 border-purple-500 border-t-transparent rounded-full animate-spin mx-auto"></div>
          <p className="text-slate-400">Loading your landing page...</p>
        </div>
      </div>
    );
  }

  // Error state
  if (variantError) {
    return (
      <div className="min-h-screen bg-slate-950 flex items-center justify-center px-6">
        <div className="max-w-md w-full rounded-2xl border border-red-500/20 bg-red-500/10 p-8 text-center space-y-4">
          <div className="w-16 h-16 rounded-full bg-red-500/20 flex items-center justify-center mx-auto">
            <svg className="w-8 h-8 text-red-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </div>
          <h1 className="text-2xl font-bold text-red-400">Failed to Load Variant</h1>
          <p className="text-red-300">{variantError}</p>
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

  // No variant state
  if (!variant) {
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

  // Sections error state
  if (sectionsError) {
    return (
      <div className="min-h-screen bg-slate-950 flex items-center justify-center px-6">
        <div className="max-w-md w-full rounded-2xl border border-orange-500/20 bg-orange-500/10 p-8 text-center space-y-4">
          <h1 className="text-2xl font-bold text-orange-400">Failed to Load Sections</h1>
          <p className="text-orange-300">{sectionsError}</p>
          <p className="text-sm text-orange-200">Using variant: {variant.name}</p>
        </div>
      </div>
    );
  }

  // Render sections
  const renderSection = (section: ContentSection) => {
    const commonProps = {
      key: section.id,
      content: section.content,
    };

    switch (section.section_type) {
      case 'hero':
        return <HeroSection {...commonProps} />;
      case 'features':
        return <FeaturesSection {...commonProps} />;
      case 'pricing':
        return <PricingSection {...commonProps} />;
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
      {/* Debug info in dev mode (only visible with ?debug=1) */}
      {new URLSearchParams(window.location.search).get('debug') === '1' && (
        <div className="fixed top-4 right-4 z-50 max-w-xs rounded-lg border border-yellow-500/20 bg-yellow-500/10 p-4 text-xs backdrop-blur">
          <div className="font-semibold text-yellow-400 mb-2">Debug Info</div>
          <div className="space-y-1 text-yellow-200">
            <div>Variant: {variant.name} ({variant.slug})</div>
            <div>Sections: {sections.length}</div>
            <div>Status: {variant.status}</div>
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
            {/* Admin preview link kept under /preview */}
            <a
              href="/preview/admin"
              className="inline-block px-6 py-2 bg-purple-500/20 hover:bg-purple-500/30 rounded-lg transition-colors text-purple-300 text-sm"
            >
              Go to Admin Panel
            </a>
          </div>
        </div>
      )}
    </div>
  );
}
