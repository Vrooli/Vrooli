import type {
  HeaderVisibilityConfig,
  LandingHeaderConfig,
  LandingHeaderNavLink,
} from '../api';

export function buildDefaultHeaderConfig(name?: string): LandingHeaderConfig {
  return {
    branding: {
      mode: 'logo_and_name',
      label: name ?? 'Landing',
      mobile_preference: 'auto',
      logo_url: undefined,
      logo_icon_url: undefined,
    },
    nav: {
      links: [],
    },
    ctas: {
      primary: {
        mode: 'inherit_hero',
        variant: 'solid',
      },
      secondary: {
        mode: 'downloads',
        variant: 'ghost',
      },
    },
    behavior: {
      sticky: true,
      hide_on_scroll: false,
    },
  };
}

export function normalizeHeaderConfig(config?: LandingHeaderConfig | null, name?: string): LandingHeaderConfig {
  if (!config) {
    return buildDefaultHeaderConfig(name);
  }

  // Header configuration is API data and may be partial during migrations or
  // when older saved variants are loaded. Treat it as partial at this boundary.
  const partialConfig = config as Partial<LandingHeaderConfig>;
  const base = buildDefaultHeaderConfig(name ?? partialConfig.branding?.label ?? 'Landing');
  return {
    branding: {
      mode: partialConfig.branding?.mode ?? base.branding.mode,
      label: partialConfig.branding?.label ?? base.branding.label,
      subtitle: partialConfig.branding?.subtitle,
      mobile_preference: partialConfig.branding?.mobile_preference ?? base.branding.mobile_preference,
      logo_url: partialConfig.branding?.logo_url ?? base.branding.logo_url,
      logo_icon_url: partialConfig.branding?.logo_icon_url ?? base.branding.logo_icon_url,
    },
    nav: {
      links: (partialConfig.nav?.links ?? []).map((link, index) => normalizeNavLink(link, index)),
    },
    ctas: {
      primary: normalizeHeaderCTA(partialConfig.ctas?.primary, base.ctas.primary),
      secondary: normalizeHeaderCTA(partialConfig.ctas?.secondary, base.ctas.secondary),
    },
    behavior: {
      sticky: partialConfig.behavior?.sticky ?? base.behavior.sticky,
      hide_on_scroll: partialConfig.behavior?.hide_on_scroll ?? base.behavior.hide_on_scroll,
    },
  };
}

function normalizeNavLink(link: LandingHeaderNavLink, index: number): LandingHeaderNavLink {
  const visibility = ensureVisibility(link.visible_on);
  const children = Array.isArray(link.children)
    ? link.children.map((child, childIdx) => normalizeNavLink(child, childIdx))
    : undefined;
  return {
    id: link.id || `nav-${link.type}-${String(index)}`,
    type: link.type,
    label: link.label || 'Section',
    section_type: link.section_type,
    section_id: link.section_id,
    anchor: link.anchor,
    href: link.href,
    visible_on: visibility,
    children,
  };
}

function ensureVisibility(visibility?: HeaderVisibilityConfig): Required<HeaderVisibilityConfig> {
  const desktop = visibility?.desktop ?? true;
  const mobile = visibility?.mobile ?? true;
  if (!desktop && !mobile) {
    return { desktop: true, mobile: true };
  }
  return { desktop, mobile };
}

function normalizeHeaderCTA<T extends { mode?: string; label?: string; href?: string; variant?: string }>(
  incoming: T | undefined,
  fallback: T,
): T {
  return {
    ...fallback,
    ...(incoming ?? {}),
  };
}

export function cloneHeaderConfig(config: LandingHeaderConfig): LandingHeaderConfig {
  return structuredClone(config);
}
