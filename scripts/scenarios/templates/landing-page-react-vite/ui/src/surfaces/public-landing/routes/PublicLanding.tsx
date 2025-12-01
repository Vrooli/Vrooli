import { useMemo } from 'react';
import { ArrowRight } from 'lucide-react';
import { Button } from '../../../shared/ui/button';
import { useLandingVariant, type VariantResolution } from '../../../app/providers/LandingVariantProvider';
import type { LandingConfigResponse, LandingSection } from '../../../shared/api';
import { HeroSection } from '../sections/HeroSection';
import { FeaturesSection } from '../sections/FeaturesSection';
import { PricingSection } from '../sections/PricingSection';
import { CTASection } from '../sections/CTASection';
import { TestimonialsSection } from '../sections/TestimonialsSection';
import { FAQSection } from '../sections/FAQSection';
import { FooterSection } from '../sections/FooterSection';
import { VideoSection } from '../sections/VideoSection';
import { DownloadSection } from '../sections/DownloadSection';

interface SectionRendererContext {
  section: LandingSection;
  config: LandingConfigResponse | null;
}

type SectionRenderer = (context: SectionRendererContext) => JSX.Element | null;

interface NavItem {
  id: string;
  label: string;
}

const SECTION_NAV_ORDER = ['hero', 'video', 'features', 'pricing', 'testimonials', 'faq', 'cta', 'footer'] as const;
const SECTION_NAV_LABELS: Record<string, string> = {
  hero: 'Overview',
  video: 'Demo',
  features: 'Features',
  pricing: 'Pricing',
  testimonials: 'Proof',
  faq: 'FAQ',
  cta: 'Call to Action',
  footer: 'More',
};
const DOWNLOAD_ANCHOR_ID = 'downloads-section';
const RESOLUTION_LABELS: Record<VariantResolution, string> = {
  url_param: 'URL parameter',
  local_storage: 'Stored assignment',
  api_select: 'Weighted API',
  fallback: 'Offline fallback',
  unknown: 'Unknown source',
};

// Map section types declared in variant schemas to their React implementations.
// Add new entries here when introducing a new section so renderers stay centralized.
const SECTION_COMPONENTS: Record<string, SectionRenderer> = {
  hero: ({ section }) => <HeroSection content={section.content} />,
  features: ({ section }) => <FeaturesSection content={section.content} />,
  pricing: ({ section, config }) => (
    <PricingSection content={section.content} pricingOverview={config?.pricing} />
  ),
  cta: ({ section }) => <CTASection content={section.content} />,
  testimonials: ({ section }) => <TestimonialsSection content={section.content} />,
  faq: ({ section }) => <FAQSection content={section.content} />,
  footer: ({ section }) => <FooterSection content={section.content} />,
  video: ({ section }) => <VideoSection content={section.content} />,
};

function getSectionKey(section: LandingSection) {
  return section.id ?? `${section.section_type}-${section.order}`;
}

function getSectionAnchorId(section: LandingSection) {
  const base = section.section_type.replace(/_/g, '-');
  const suffix = section.id ?? section.order;
  return `${base}-${suffix}`;
}

function buildNavItems(sections: LandingSection[], includeDownloads: boolean): NavItem[] {
  const items: NavItem[] = [];
  for (const type of SECTION_NAV_ORDER) {
    const match = sections.find((section) => section.section_type === type);
    if (!match) {
      continue;
    }
    items.push({ id: getSectionAnchorId(match), label: SECTION_NAV_LABELS[type] ?? type });
  }
  if (includeDownloads) {
    items.push({ id: DOWNLOAD_ANCHOR_ID, label: 'Downloads' });
  }
  return items;
}

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
  const {
    variant,
    config,
    loading: configLoading,
    error: configError,
    resolution,
    statusNote,
    lastUpdated,
  } = useLandingVariant();
  const fallbackActive = Boolean(config?.fallback);
  const variantPinnedViaParam = resolution === 'url_param';

  const sections = useMemo(() => {
    const incoming = config?.sections ?? [];
    return incoming
      .filter((section) => section.enabled !== false)
      .sort((a, b) => a.order - b.order);
  }, [config]);
  const debugSectionTypes: string[] | null = null;
  const sectionsToRender = debugSectionTypes
    ? sections.filter((section) => debugSectionTypes.includes(section.section_type))
    : sections;
  const downloads = config?.downloads ?? [];
  const hasDownloads = downloads.length > 0;
  const navItems = useMemo(() => buildNavItems(sections, hasDownloads), [sections, hasDownloads]);
  const heroSection = sections.find((section) => section.section_type === 'hero');
  const heroCTAContent = heroSection?.content as { cta_text?: string; cta_url?: string } | undefined;
  const heroCtaText = typeof heroCTAContent?.cta_text === 'string' ? heroCTAContent.cta_text : undefined;
  const heroCtaUrl = typeof heroCTAContent?.cta_url === 'string' ? heroCTAContent.cta_url : undefined;
  const variantLabel = variant?.name ?? variant?.slug ?? 'Variant not resolved';
  const resolutionLabel = RESOLUTION_LABELS[resolution] ?? RESOLUTION_LABELS.unknown;
  const downloadButtonLabel = useMemo(() => {
    if (downloads.length === 0) {
      return 'Downloads';
    }
    if (downloads.length === 1) {
      const single = downloads[0];
      if (single?.label) {
        return `Download ${single.label}`;
      }
      if (single?.platform) {
        return `Download ${single.platform}`;
      }
      return 'Download';
    }
    return 'View downloads';
  }, [downloads]);

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
    const renderer = SECTION_COMPONENTS[section.section_type];
    if (!renderer) {
      console.warn(`Unknown section type: ${section.section_type}`);
      return null;
    }

    return (
      <div key={getSectionKey(section)} id={getSectionAnchorId(section)} className="scroll-mt-28">
        {renderer({
          section,
          config,
        })}
      </div>
    );
  };

  return (
    <div className="min-h-screen bg-[#07090F] text-slate-50">
      <LandingExperienceHeader
        navItems={navItems}
        ctaText={heroCtaText}
        ctaUrl={heroCtaUrl}
        variantLabel={variantLabel}
        resolutionLabel={resolutionLabel}
        fallbackActive={fallbackActive}
        statusNote={statusNote}
        downloadsAvailable={hasDownloads}
        downloadAnchorId={DOWNLOAD_ANCHOR_ID}
        downloadButtonLabel={downloadButtonLabel}
      />
      {variantPinnedViaParam && (
        <div className="border border-[#38BDF8]/30 bg-[#38BDF8]/10 py-3 px-4 text-center text-sm text-[#d3f2ff]" data-testid="variant-source-banner">
          Variant <strong>{variant?.name ?? variant?.slug}</strong> is pinned via URL parameter. Remove the <code>?variant=</code> query to resume weighted traffic allocation.
        </div>
      )}

      {fallbackActive && (
        <div className="border border-[#F97316]/30 bg-[#F97316]/10 py-3 px-4 text-center text-sm text-[#ffd3b5]" data-testid="fallback-signal-banner">
          Offline-safe fallback variant is active. {statusNote && <span>{statusNote}. </span>}
          Live analytics, pricing, and downloads may be outdated until the API recovers.
          {lastUpdated && (
            <span className="ml-1 text-amber-200/80">Last sync: {new Date(lastUpdated).toLocaleTimeString()}.</span>
          )}
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
            <div>Resolution: {resolution}</div>
            {statusNote && <div>Note: {statusNote}</div>}
            {lastUpdated && <div>Last updated: {new Date(lastUpdated).toLocaleTimeString()}</div>}
            {fallbackActive && <div>Fallback Variant Active</div>}
          </div>
        </div>
      )}

      {/* Render all sections in order */}
      {sectionsToRender.length > 0 ? (
        sectionsToRender.map(renderSection)
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
      {hasDownloads && config?.downloads && (
        <div id={DOWNLOAD_ANCHOR_ID} className="scroll-mt-28">
          <DownloadSection
            downloads={config.downloads}
            content={{
              title: 'Download Browser Automation Studio',
              subtitle: 'Install on Windows, macOS, or Linux while we verify your subscription entitlements.',
            }}
          />
        </div>
      )}
    </div>
  );
}

interface LandingExperienceHeaderProps {
  navItems: NavItem[];
  ctaText?: string;
  ctaUrl?: string;
  variantLabel: string;
  resolutionLabel: string;
  fallbackActive: boolean;
  statusNote: string | null;
  downloadsAvailable: boolean;
  downloadAnchorId?: string;
  downloadButtonLabel?: string;
}

function LandingExperienceHeader({
  navItems,
  ctaText,
  ctaUrl,
  variantLabel,
  resolutionLabel,
  fallbackActive,
  statusNote,
  downloadsAvailable,
  downloadAnchorId,
  downloadButtonLabel,
}: LandingExperienceHeaderProps) {
  const hasNav = navItems.length > 0;
  const configClass = fallbackActive
    ? 'border-[#F97316]/40 bg-[#F97316]/10 text-[#ffd3b5]'
    : 'border-[#10B981]/40 bg-[#10B981]/10 text-[#c5f4df]';

  return (
    <div className="sticky top-0 z-30 border-b border-white/10 bg-[#07090F]/95 backdrop-blur" data-testid="landing-experience-header">
      <div className="mx-auto flex max-w-6xl flex-col gap-3 px-6 py-4 md:flex-row md:items-center md:justify-between">
        <div>
          <p className="text-xs uppercase tracking-[0.3em] text-slate-500">Landing runtime</p>
          <div className="mt-2 flex flex-wrap gap-2 text-xs font-medium">
            <span className={`rounded-full border px-3 py-1 ${configClass}`}>
              {fallbackActive ? 'Fallback copy active' : 'Live API config'}
            </span>
            <span className="rounded-full border border-white/10 px-3 py-1 text-slate-100">{variantLabel}</span>
            <span className="rounded-full border border-white/10 px-3 py-1 text-slate-400">Source: {resolutionLabel}</span>
          </div>
          {statusNote && <p className="mt-1 text-xs text-slate-500">{statusNote}</p>}
        </div>
        {hasNav && (
          <nav className="hidden items-center gap-4 text-sm text-slate-300 md:flex">
            {navItems.map((item) => (
              <a key={item.id} href={`#${item.id}`} className="transition-colors hover:text-white">
                {item.label}
              </a>
            ))}
          </nav>
        )}
        {(ctaText && ctaUrl) || (downloadsAvailable && downloadAnchorId) ? (
          <div className="flex flex-wrap gap-2">
            {ctaText && ctaUrl && (
              <Button asChild size="sm" className="gap-1 whitespace-nowrap" data-testid="landing-nav-cta">
                <a href={ctaUrl}>
                  {ctaText}
                  <ArrowRight className="ml-1 h-4 w-4" />
                </a>
              </Button>
            )}
            {downloadsAvailable && downloadAnchorId && (
              <Button
                asChild
                size="sm"
                variant="ghost"
                className="gap-1 whitespace-nowrap bg-white/5 text-white hover:bg-white/10"
                data-testid="landing-nav-download"
              >
                <a href={`#${downloadAnchorId}`}>
                  {downloadButtonLabel ?? 'Downloads'}
                  <ArrowRight className="ml-1 h-4 w-4" />
                </a>
              </Button>
            )}
          </div>
        ) : null}
      </div>
      {hasNav && (
        <div className="flex gap-3 overflow-x-auto px-6 pb-3 text-xs text-slate-400 md:hidden" data-testid="landing-nav-mobile">
          {navItems.map((item) => (
            <a key={item.id} href={`#${item.id}`} className="whitespace-nowrap rounded-full border border-white/10 px-3 py-1">
              {item.label}
            </a>
          ))}
        </div>
      )}
    </div>
  );
}
