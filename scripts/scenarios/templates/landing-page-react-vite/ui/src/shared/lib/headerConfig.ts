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

  const base = buildDefaultHeaderConfig(name ?? config.branding?.label ?? 'Landing');
  return {
    branding: {
      mode: config.branding?.mode ?? base.branding.mode,
      label: config.branding?.label ?? base.branding.label,
      subtitle: config.branding?.subtitle,
      mobile_preference: config.branding?.mobile_preference ?? base.branding.mobile_preference,
      logo_url: config.branding?.logo_url ?? base.branding.logo_url,
      logo_icon_url: config.branding?.logo_icon_url ?? base.branding.logo_icon_url,
    },
    nav: {
      links: (config.nav?.links ?? []).map((link, index) => normalizeNavLink(link, index)),
    },
    ctas: {
      primary: normalizeHeaderCTA(config.ctas?.primary, base.ctas.primary),
      secondary: normalizeHeaderCTA(config.ctas?.secondary, base.ctas.secondary),
    },
    behavior: {
      sticky: config.behavior?.sticky ?? base.behavior.sticky,
      hide_on_scroll: config.behavior?.hide_on_scroll ?? base.behavior.hide_on_scroll,
    },
  };
}

function normalizeNavLink(link: LandingHeaderNavLink, index: number): LandingHeaderNavLink {
  const visibility = ensureVisibility(link.visible_on);
  const children = Array.isArray(link.children)
    ? link.children.map((child, childIdx) => normalizeNavLink(child, childIdx))
    : undefined;
  return {
    id: link.id || `nav-${link.type}-${index}`,
    type: link.type ?? 'section',
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
  if (!visibility?.desktop && !visibility?.mobile) {
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
  return JSON.parse(JSON.stringify(config)) as LandingHeaderConfig;
}
