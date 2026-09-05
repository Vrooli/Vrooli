import { create } from '@bufbuild/protobuf';
import {
  LandingHeaderConfigSchema,
  HeaderNavLinkSchema,
} from '@vrooli/proto-types/landing-page-react-vite/v1/variant_pb';
import type {
  HeaderVisibilityConfig,
  LandingHeaderConfig,
  LandingHeaderNavLink,
} from '../api';

export function buildDefaultHeaderConfig(name?: string): LandingHeaderConfig {
  return create(LandingHeaderConfigSchema, {
    branding: {
      mode: 'logo_and_name',
      label: name ?? 'Landing',
      mobilePreference: 'auto',
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
      hideOnScroll: false,
    },
  });
}

export function normalizeHeaderConfig(
  config?: LandingHeaderConfig | null,
  name?: string,
): LandingHeaderConfig {
  if (!config) {
    return buildDefaultHeaderConfig(name);
  }

  const base = buildDefaultHeaderConfig(name ?? config.branding?.label ?? 'Landing');
  return create(LandingHeaderConfigSchema, {
    branding: {
      mode: config.branding?.mode || base.branding?.mode,
      label: config.branding?.label || base.branding?.label,
      subtitle: config.branding?.subtitle,
      mobilePreference: config.branding?.mobilePreference || base.branding?.mobilePreference,
    },
    nav: {
      links: (config.nav?.links ?? []).map((link, index) => normalizeNavLink(link, index)),
    },
    ctas: {
      primary: normalizeHeaderCTA(config.ctas?.primary, base.ctas?.primary),
      secondary: normalizeHeaderCTA(config.ctas?.secondary, base.ctas?.secondary),
    },
    behavior: {
      sticky: config.behavior?.sticky ?? base.behavior?.sticky ?? true,
      hideOnScroll: config.behavior?.hideOnScroll ?? base.behavior?.hideOnScroll ?? false,
    },
  });
}

function normalizeNavLink(link: LandingHeaderNavLink, index: number): LandingHeaderNavLink {
  const visibility = ensureVisibility(link.visibleOn);
  const children = Array.isArray(link.children)
    ? link.children.map((child, childIdx) => normalizeNavLink(child, childIdx))
    : [];
  return create(HeaderNavLinkSchema, {
    id: link.id || `nav-${link.type}-${index}`,
    type: link.type || 'section',
    label: link.label || 'Section',
    sectionType: link.sectionType,
    sectionId: link.sectionId,
    anchor: link.anchor,
    href: link.href,
    visibleOn: visibility,
    children,
  });
}

function ensureVisibility(visibility?: HeaderVisibilityConfig): {
  desktop: boolean;
  mobile: boolean;
} {
  const desktop = visibility?.desktop ?? true;
  const mobile = visibility?.mobile ?? true;
  if (!visibility?.desktop && !visibility?.mobile) {
    return { desktop: true, mobile: true };
  }
  return { desktop, mobile };
}

interface HeaderCTAInit {
  mode?: string;
  label?: string;
  href?: string;
  variant?: string;
}

function normalizeHeaderCTA(incoming?: HeaderCTAInit, fallback?: HeaderCTAInit): HeaderCTAInit {
  return {
    ...(fallback ?? {}),
    ...(incoming ?? {}),
  };
}

export function cloneHeaderConfig(config: LandingHeaderConfig): LandingHeaderConfig {
  return normalizeHeaderConfig(config);
}
