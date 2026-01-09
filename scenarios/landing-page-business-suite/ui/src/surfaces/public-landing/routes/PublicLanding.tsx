import { useMemo, useState, useEffect } from 'react';
import { ArrowRight } from 'lucide-react';
import { Button } from '../../../shared/ui/button';
import { SEOHead } from '../../../shared/ui/SEOHead';
import { useLandingVariant, type VariantResolution } from '../../../app/providers/LandingVariantProvider';
import type {
  DownloadApp,
  LandingBranding,
  LandingConfigResponse,
  LandingHeaderConfig,
  LandingSection,
} from '../../../shared/api';
import { HeroSection } from '../sections/HeroSection';
import { FeaturesSection } from '../sections/FeaturesSection';
import { PricingSection } from '../sections/PricingSection';
import { CTASection } from '../sections/CTASection';
import { TestimonialsSection } from '../sections/TestimonialsSection';
import { FAQSection } from '../sections/FAQSection';
import { FooterSection } from '../sections/FooterSection';
import { VideoSection } from '../sections/VideoSection';
import { DownloadSection } from '../sections/DownloadSection';
import { DOWNLOAD_ANCHOR_ID, getSectionAnchorId, getSectionKey } from '../../../shared/lib/sections';
import { normalizeHeaderConfig } from '../../../shared/lib/headerConfig';

interface SectionRendererContext {
  section: LandingSection;
  config: LandingConfigResponse | null;
}

type SectionRenderer = (context: SectionRendererContext) => JSX.Element | null;

interface NavItem {
  id: string;
  label: string;
}

interface HeaderNavRenderItem {
  id: string;
  label: string;
  href?: string;
  desktop: boolean;
  mobile: boolean;
  type: 'link' | 'menu';
  children?: HeaderNavRenderChild[];
}

interface HeaderNavRenderChild {
  id: string;
  label: string;
  href: string;
  desktop?: boolean;
  mobile?: boolean;
}

interface HeaderCTAResolved {
  label: string;
  href: string;
  variant: 'solid' | 'ghost';
  testId: string;
}

interface HeaderCTASet {
  primary: HeaderCTAResolved | null;
  secondary: HeaderCTAResolved | null;
}

interface HeaderRuntimeMeta {
  fallbackActive: boolean;
  variantLabel: string;
  resolutionLabel: string;
  statusNote: string | null;
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
const RESOLUTION_LABELS: Record<VariantResolution, string> = {
  url_param: 'URL parameter',
  local_storage: 'Stored assignment',
  api_select: 'Weighted API',
  fallback: 'Offline fallback',
  unknown: 'Unknown source',
};
const DOWNLOAD_PLATFORM_LABELS: Record<string, string> = {
  windows: 'Windows',
  mac: 'macOS',
  linux: 'Linux',
};

function hasDownloadTargets(app: DownloadApp) {
  return (app.platforms?.length ?? 0) > 0 || (app.storefronts?.length ?? 0) > 0;
}

function formatDownloadPlatform(platform?: string) {
  if (!platform) {
    return 'Download';
  }
  return DOWNLOAD_PLATFORM_LABELS[platform.toLowerCase()] ?? platform;
}

// Map section types declared in variant schemas to their React implementations.
// Add new entries here when introducing a new section so renderers stay centralized.
const SECTION_COMPONENTS: Record<string, SectionRenderer> = {
  hero: ({ section }) => <HeroSection content={section.content} />, 
  features: ({ section }) => <FeaturesSection content={section.content} />,
  pricing: ({ section, config }) => {
    // Ensure we always have pricing data; fall back to section content tiers if needed
    return <PricingSection content={section.content} pricingOverview={config?.pricing} />;
  },
  cta: ({ section }) => <CTASection content={section.content} />, 
  testimonials: ({ section }) => <TestimonialsSection content={section.content} />,
  faq: ({ section, config }) => <FAQSection content={section.content} supportChatUrl={config?.branding?.support_chat_url} />,
  footer: ({ section }) => <FooterSection content={section.content} />,
  video: ({ section }) => <VideoSection content={section.content} />,
  downloads: ({ section, config }) => (
    <DownloadSection content={section.content as any} downloads={config?.downloads} />
  ),
};

function buildNavItems(sections: LandingSection[], includeDownloads: boolean, downloadAnchorId: string): NavItem[] {
  const items: NavItem[] = [];
  for (const type of SECTION_NAV_ORDER) {
    const match = sections.find((section) => section.section_type === type);
    if (!match) {
      continue;
    }
    items.push({ id: getSectionAnchorId(match), label: SECTION_NAV_LABELS[type] ?? type });
  }
  if (includeDownloads) {
    items.push({ id: downloadAnchorId, label: 'Downloads' });
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
  const isDebugMode = useMemo(
    () => new URLSearchParams(window.location.search).get('debug') === '1',
    [],
  );

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
  const downloadsSection = sections.find((section) => section.section_type === 'downloads');
  const downloadApps = useMemo<DownloadApp[]>(() => {
    const raw = config?.downloads ?? [];
    return raw.filter((app) => hasDownloadTargets(app));
  }, [config]);
  const hasDownloads = downloadApps.length > 0;
  const downloadAnchorId = downloadsSection ? getSectionAnchorId(downloadsSection) : DOWNLOAD_ANCHOR_ID;
  const heroSection = sections.find((section) => section.section_type === 'hero');
  const heroCTAContent = heroSection?.content as { cta_text?: string; cta_url?: string } | undefined;
  const heroCtaText = typeof heroCTAContent?.cta_text === 'string' ? heroCTAContent.cta_text : undefined;
  const heroCtaUrl = typeof heroCTAContent?.cta_url === 'string' ? heroCTAContent.cta_url : undefined;
  const variantLabel = variant?.name ?? variant?.slug ?? 'Variant not resolved';
  const resolutionLabel = RESOLUTION_LABELS[resolution] ?? RESOLUTION_LABELS.unknown;
  const downloadButtonLabel = useMemo(() => {
    if (!hasDownloads) {
      return 'Downloads';
    }
    if (downloadApps.length === 1) {
      const single = downloadApps[0];
      const singleInstaller = single.platforms?.[0];
      if ((single.platforms?.length ?? 0) === 1 && singleInstaller) {
        return `Download ${formatDownloadPlatform(singleInstaller.platform)}`;
      }
      if ((single.storefronts?.length ?? 0) === 1 && (single.platforms?.length ?? 0) === 0) {
        return `Open ${single.storefronts?.[0]?.label ?? 'store'}`;
      }
      return `View ${single.name}`;
    }
    return 'View downloads';
  }, [downloadApps, hasDownloads]);
  const headerConfig = useMemo(
    () => normalizeHeaderConfig(config?.header, config?.branding?.site_name ?? variantLabel),
    [config?.branding?.site_name, config?.header, variantLabel],
  );
  const navLinks = useMemo(
    () => resolveNavLinks(headerConfig, sections, hasDownloads, downloadAnchorId),
    [headerConfig, sections, hasDownloads, downloadAnchorId],
  );
  const ctas = useMemo(
    () =>
      resolveHeaderCTAs(
        headerConfig,
        { label: heroCtaText, href: heroCtaUrl },
        hasDownloads ? { label: downloadButtonLabel, href: `#${downloadAnchorId}` } : null,
      ),
    [headerConfig, heroCtaText, heroCtaUrl, hasDownloads, downloadButtonLabel, downloadAnchorId],
  );
  const runtimeMeta = useMemo<HeaderRuntimeMeta>(
    () => ({
      fallbackActive,
      variantLabel,
      resolutionLabel,
      statusNote,
    }),
    [fallbackActive, variantLabel, resolutionLabel, statusNote],
  );

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

    const rendered = renderer({
      section,
      config,
    });

    if (!rendered) {
      return null;
    }

    return (
      <div key={getSectionKey(section)} id={getSectionAnchorId(section)} className="scroll-mt-28">
        {rendered}
      </div>
    );
  };

  return (
    <div className="min-h-screen bg-[#07090F] text-slate-50">
      {/* Client-side SEO meta tag updates based on branding */}
      <SEOHead branding={config?.branding} />

      <LandingExperienceHeader
        headerConfig={headerConfig}
        branding={config?.branding}
        navLinks={navLinks}
        runtimeMeta={runtimeMeta}
        ctas={ctas}
        showMeta={isDebugMode}
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
      {isDebugMode && (
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
      {hasDownloads && !downloadsSection && (
        <div id={downloadAnchorId} className="scroll-mt-28">
          <DownloadSection
            downloads={downloadApps}
            content={{
              title: 'Download Vrooli Ascension',
              subtitle: 'Install on Windows, macOS, Linux, or via the stores while we verify entitlements.',
            }}
          />
        </div>
      )}
    </div>
  );
}

interface LandingExperienceHeaderProps {
  headerConfig: LandingHeaderConfig;
  branding?: LandingBranding | null;
  navLinks: HeaderNavRenderItem[];
  runtimeMeta: HeaderRuntimeMeta;
  ctas: HeaderCTASet;
  showMeta: boolean;
}

function LandingExperienceHeader({
  headerConfig,
  branding,
  navLinks,
  runtimeMeta,
  ctas,
  showMeta,
}: LandingExperienceHeaderProps) {
  const sticky = headerConfig.behavior?.sticky ?? true;
  const hideOnScroll = sticky && headerConfig.behavior?.hide_on_scroll;
  const isHidden = useHideOnScroll(hideOnScroll);
  const hasDesktopNav = navLinks.some((link) => link.desktop);
  const hasMobileNav = navLinks.some((link) => link.mobile);
  const containerClasses = [
    sticky ? 'sticky top-0' : 'relative',
    'z-30 border-b border-white/10 bg-[#07090F]/95 backdrop-blur transition-transform duration-300',
    isHidden ? '-translate-y-full opacity-0' : 'translate-y-0 opacity-100',
  ].join(' ');

  return (
    <div className={containerClasses} data-testid="landing-experience-header">
      <div className="mx-auto flex max-w-6xl flex-col gap-3 px-6 py-4 md:flex-row md:items-center md:justify-between">
        <BrandingBlock
          header={headerConfig}
          branding={branding}
          runtime={runtimeMeta}
          showMeta={showMeta}
        />
        {hasDesktopNav && (
          <nav className="hidden items-center gap-4 text-sm text-slate-300 md:flex">
            {navLinks
              .filter((link) => link.desktop)
              .map((link) =>
                link.type === 'menu' && link.children && link.children.length > 0 ? (
                  <div key={link.id} className="relative group">
                    <button
                      type="button"
                      className="inline-flex items-center gap-1 rounded-md px-2 py-1 transition-colors hover:text-white"
                    >
                      {link.label}
                      <svg
                        className="h-3 w-3 opacity-70"
                        viewBox="0 0 10 6"
                        fill="none"
                        xmlns="http://www.w3.org/2000/svg"
                      >
                        <path d="M1 1l4 4 4-4" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
                      </svg>
                    </button>
                    <div className="invisible absolute left-0 top-full z-40 mt-2 min-w-[180px] rounded-lg border border-white/10 bg-[#0B1020] py-2 opacity-0 shadow-xl transition-all duration-150 group-hover:visible group-hover:opacity-100">
                      {link.children
                        .filter((child) => child.desktop ?? true)
                        .map((child) => (
                          <a
                            key={child.id}
                            href={child.href}
                            className="block px-4 py-2 text-sm text-slate-200 hover:bg-white/5"
                          >
                            {child.label}
                          </a>
                        ))}
                    </div>
                  </div>
                ) : (
                  <a key={link.id} href={link.href} className="transition-colors hover:text-white">
                    {link.label}
                  </a>
                ),
              )}
          </nav>
        )}
        {(ctas.primary || ctas.secondary) && (
          <div className="flex flex-col gap-2 sm:flex-row sm:flex-nowrap sm:items-center">
            {ctas.primary && (
              <Button asChild size="sm" className="gap-1 whitespace-nowrap" data-testid={ctas.primary.testId}>
                <a href={ctas.primary.href}>
                  {ctas.primary.label}
                  <ArrowRight className="ml-1 h-4 w-4" />
                </a>
              </Button>
            )}
            {ctas.secondary && (
              <Button
                asChild
                size="sm"
                variant={ctas.secondary.variant === 'ghost' ? 'ghost' : undefined}
                className={ctas.secondary.variant === 'ghost' ? 'gap-1 whitespace-nowrap bg-white/5 text-white hover:bg-white/10' : 'gap-1 whitespace-nowrap'}
                data-testid={ctas.secondary.testId}
              >
                <a href={ctas.secondary.href}>
                  {ctas.secondary.label}
                  <ArrowRight className="ml-1 h-4 w-4" />
                </a>
              </Button>
            )}
          </div>
        )}
      </div>
      {hasMobileNav && (
        <div className="flex gap-3 overflow-x-auto px-6 pb-3 text-xs text-slate-400 md:hidden" data-testid="landing-nav-mobile">
          {navLinks
            .filter((link) => link.mobile)
            .map((link) => {
              if (link.type === 'menu' && link.children?.length) {
                return link.children
                  .filter((child) => child.mobile ?? true)
                  .map((child) => (
                    <a
                      key={`${link.id}-${child.id}`}
                      href={child.href}
                      className="whitespace-nowrap rounded-full border border-white/10 px-3 py-1"
                    >
                      {link.label} · {child.label}
                    </a>
                  ));
              }
              return (
                <a key={link.id} href={link.href} className="whitespace-nowrap rounded-full border border-white/10 px-3 py-1">
                  {link.label}
                </a>
              );
            })}
        </div>
      )}
    </div>
  );
}

function BrandingBlock({
  header,
  branding,
  runtime,
  showMeta,
}: {
  header: LandingHeaderConfig;
  branding?: LandingBranding | null;
  runtime: HeaderRuntimeMeta;
  showMeta: boolean;
}) {
  const mode = header.branding?.mode ?? 'logo_and_name';
  const label = header.branding?.label ?? branding?.site_name ?? runtime.variantLabel;
  const subtitle = header.branding?.subtitle ?? branding?.tagline ?? null;
  const defaultLogo = '/logo-mask-512x512.webp';
  const logoUrl =
    header.branding?.logo_icon_url ??
    header.branding?.logo_url ??
    branding?.logo_icon_url ??
    branding?.logo_url ??
    defaultLogo;
  const mobilePref = header.branding?.mobile_preference ?? 'auto';
  const showLogo = mode === 'logo' || mode === 'logo_and_name';
  const showName = mode === 'name' || mode === 'logo_and_name';

  if (mode === 'none') {
    return (
      <div className="flex flex-col gap-1">
        <p className="text-sm font-semibold text-white">{runtime.variantLabel}</p>
        {showMeta && <RuntimeMeta runtime={runtime} />}
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-1">
      <div className="flex items-center gap-3">
        {showLogo && (
          logoUrl ? (
            <img
              src={logoUrl}
              alt={`${label} logo`}
              className={`${mobilePref === 'name' ? 'hidden sm:flex' : ''} h-10 w-10 rounded-xl object-contain bg-white/5`}
              data-testid="branding-logo"
            />
          ) : (
            <BrandPlaceholder label={label} mobilePreference={mobilePref} hideOnMobile={mobilePref === 'name'} />
          )
        )}
        {showName && (
          <div className={mobilePref === 'logo' ? 'hidden sm:block' : ''}>
            <p className="text-lg font-semibold text-white">{label}</p>
            {subtitle && <p className="text-xs text-slate-400">{subtitle}</p>}
          </div>
        )}
      </div>
      {showMeta && <RuntimeMeta runtime={runtime} />}
    </div>
  );
}

function BrandPlaceholder({
  label,
  mobilePreference,
  hideOnMobile,
}: {
  label: string;
  mobilePreference: string;
  hideOnMobile: boolean;
}) {
  const initials = label
    .split(' ')
    .map((part) => part.charAt(0))
    .join('')
    .slice(0, 2)
    .toUpperCase() || 'VR';
  const visibilityClass =
    hideOnMobile || mobilePreference === 'name' ? 'hidden sm:flex' : 'flex';

  return (
    <div
      className={`${visibilityClass} h-10 w-10 items-center justify-center rounded-xl bg-gradient-to-br from-sky-500/80 to-indigo-500/80 text-sm font-bold text-white`}
    >
      {initials}
    </div>
  );
}

function RuntimeMeta({ runtime }: { runtime: HeaderRuntimeMeta }) {
  return (
    <div className="flex flex-wrap gap-2 text-xs text-slate-500">
      <span
        className={`rounded-full border px-2 py-0.5 ${runtime.fallbackActive ? 'border-[#F97316]/40 bg-[#F97316]/10 text-[#ffd3b5]' : 'border-[#10B981]/40 bg-[#10B981]/10 text-[#c5f4df]'}`}
      >
        {runtime.fallbackActive ? 'Fallback copy active' : runtime.resolutionLabel}
      </span>
      <span className="text-slate-300">{runtime.variantLabel}</span>
      {runtime.statusNote && <span className="text-slate-500">{runtime.statusNote}</span>}
    </div>
  );
}

function useHideOnScroll(enabled: boolean) {
  const [hidden, setHidden] = useState(false);

  useEffect(() => {
    if (!enabled || typeof window === 'undefined') {
      setHidden(false);
      return;
    }
    let lastScroll = window.scrollY;
    const handleScroll = () => {
      const current = window.scrollY;
      if (current > lastScroll + 20 && current > 120) {
        setHidden(true);
      } else if (current < lastScroll - 20) {
        setHidden(false);
      }
      lastScroll = current;
    };
    window.addEventListener('scroll', handleScroll);
    return () => window.removeEventListener('scroll', handleScroll);
  }, [enabled]);

  return hidden;
}

function resolveNavLinks(
  headerConfig: LandingHeaderConfig,
  sections: LandingSection[],
  downloadsAvailable: boolean,
  downloadAnchorId: string,
): HeaderNavRenderItem[] {
  const defaultLinks = buildNavItems(sections, downloadsAvailable, downloadAnchorId).map<HeaderNavRenderItem>((item) => ({
    id: item.id,
    label: item.label,
    href: `#${item.id}`,
    desktop: true,
    mobile: true,
    type: 'link',
  }));

  const configuredLinks = headerConfig.nav?.links ?? [];
  if (configuredLinks.length === 0) {
    return defaultLinks;
  }

  const anchors = sections.map((section) => ({
    id: section.id,
    type: section.section_type,
    anchor: getSectionAnchorId(section),
  }));

  return configuredLinks.flatMap<HeaderNavRenderItem>((link, index) => {
    const visibility = ensureVisibilityFlags(link.visible_on);
    const label = link.label || SECTION_NAV_LABELS[link.section_type ?? ''] || 'Section';

    if (link.type === 'downloads') {
      if (!downloadsAvailable) {
        return [];
      }
      return [
        {
          id: link.id || `nav-downloads-${index}`,
          label,
          href: `#${downloadAnchorId}`,
          desktop: visibility.desktop,
          mobile: visibility.mobile,
          type: 'link',
        },
      ];
    }

    if (link.type === 'custom' && link.href) {
      return [
        {
          id: link.id || `nav-custom-${index}`,
          label,
          href: link.href,
          desktop: visibility.desktop,
          mobile: visibility.mobile,
          type: 'link',
        },
      ];
    }

    if (link.type === 'menu' && Array.isArray(link.children) && link.children.length > 0) {
      const children: HeaderNavRenderChild[] = link.children.map((child, childIdx) => {
        const childVisibility = ensureVisibilityFlags(child.visible_on);
        // child visibility influences only child; parent visibility still used for desktop render
        const childLabel = child.label || SECTION_NAV_LABELS[child.section_type ?? ''] || 'Item';
        const childAnchor =
          child.anchor ||
          (typeof child.section_id === 'number'
            ? anchors.find((entry) => entry.id === child.section_id)?.anchor
            : undefined) ||
          (child.section_type ? anchors.find((entry) => entry.type === child.section_type)?.anchor : undefined);
        const childHref =
          child.type === 'downloads'
            ? `#${downloadAnchorId}`
            : childAnchor
              ? childAnchor.startsWith('#')
                ? childAnchor
                : `#${childAnchor}`
              : child.href || '#';

        return {
          id: child.id || `${link.id || 'menu'}-child-${childIdx}`,
          label: childLabel,
          href: childHref,
          desktop: childVisibility.desktop,
          mobile: childVisibility.mobile,
        };
      });
      return [
        {
          id: link.id || `nav-menu-${index}`,
          label,
          type: 'menu',
          desktop: visibility.desktop,
          mobile: visibility.mobile,
          children,
        },
      ];
    }

    const anchor =
      link.anchor ||
      (typeof link.section_id === 'number'
        ? anchors.find((entry) => entry.id === link.section_id)?.anchor
        : undefined) ||
      (link.section_type ? anchors.find((entry) => entry.type === link.section_type)?.anchor : undefined);

    if (!anchor) {
      return [];
    }

    return [
      {
        id: link.id || `nav-section-${index}`,
        label,
        href: `#${anchor}`,
        desktop: visibility.desktop,
        mobile: visibility.mobile,
        type: 'link',
      },
    ];
  });
}

function ensureVisibilityFlags(visibility?: { desktop?: boolean; mobile?: boolean }) {
  const desktop = visibility?.desktop ?? true;
  const mobile = visibility?.mobile ?? true;
  if (!visibility) {
    return { desktop: true, mobile: true };
  }
  if (!desktop && !mobile) {
    return { desktop: true, mobile: true };
  }
  return { desktop, mobile };
}

function resolveHeaderCTAs(
  headerConfig: LandingHeaderConfig,
  heroCTA: { label?: string; href?: string },
  downloadCTA: { label: string; href: string } | null,
): HeaderCTASet {
  return {
    primary: resolveCTA(headerConfig.ctas?.primary, heroCTA, downloadCTA, 'landing-nav-cta'),
    secondary: resolveCTA(
      headerConfig.ctas?.secondary,
      heroCTA,
      downloadCTA,
      headerConfig.ctas?.secondary?.mode === 'downloads' ? 'landing-nav-download' : 'landing-nav-cta-secondary',
    ),
  };
}

function resolveCTA(
  config: { mode?: string; label?: string; href?: string; variant?: 'solid' | 'ghost' } | undefined,
  heroCTA: { label?: string; href?: string },
  downloadCTA: { label: string; href: string } | null,
  testId: string,
): HeaderCTAResolved | null {
  const mode = config?.mode ?? 'inherit_hero';
  const variant = config?.variant ?? (mode === 'downloads' ? 'ghost' : 'solid');

  if (mode === 'hidden') {
    return null;
  }

  if (mode === 'downloads') {
    if (!downloadCTA) {
      return null;
    }
    return {
      label: config?.label ?? downloadCTA.label,
      href: downloadCTA.href,
      variant,
      testId: 'landing-nav-download',
    };
  }

  if (mode === 'inherit_hero') {
    if (!heroCTA.label || !heroCTA.href) {
      return null;
    }
    return {
      label: config?.label ?? heroCTA.label,
      href: heroCTA.href,
      variant,
      testId,
    };
  }

  if (mode === 'custom' && config?.label && config?.href) {
    return {
      label: config.label,
      href: config.href,
      variant,
      testId,
    };
  }

  return null;
}
