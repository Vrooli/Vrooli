import { create } from '@bufbuild/protobuf';
import { ResolvedProductPresentationSchema } from '@vrooli/proto-types/landing-page-business-suite/v1/shared/product_presentation_pb';
import type { LandingConfigResponse } from '../../../shared/api/types';

/** Synthetic integration input, never imported by a production module. */
export function publicConfig(route = '/', locale = 'en', variant = 'control'): LandingConfigResponse {
  return {
    downloads: [], fallback: false,
    presentation: create(ResolvedProductPresentationSchema, {
      schemaVersion: 1, mode: route === '/' ? 'empty' : 'app_detail', scope: route === '/' ? 'bundle' : 'app',
      page: {
        id: 'configured', locale, title: `Configured ${route}`, description: `Description ${locale}`,
        theme: { variant: 'signal', primary: '#263e38', background: '#faf9f6', accent: '#a84422' },
        navigation: { label: 'Configured navigation', items: [{ label: 'Configured detail', accessibleLabel: 'Configured detail', target: '/apps/example' }] },
        footer: { label: 'Configured footer', links: [] },
        display: { shell: {
          brandName: 'Configured product', brandMark: 'suite', brandTarget: '/', skipLabel: 'Skip content', menuLabel: 'Open menu',
          footerBrandName: 'Configured owner', footerBrandMark: 'suite', footerBrandTarget: '/', unavailableReason: 'Owner unavailable', previewLabel: 'Private preview',
        } },
        blocks: [{ id: 'closing', kind: 'closing-action', version: 1, variant: 'plain', content: { value: { case: 'closingAction', value: {
          heading: `Published ${route}`, description: 'Configured content', actions: [{ kind: 'purchase', label: 'Configured purchase', accessibleLabel: 'Configured purchase', planRef: 'plan' }],
        } } } }],
      },
      diagnostics: { requestedRoute: route, resolvedRoute: route, requestedVariant: '', resolvedVariant: variant, locale, resolvedRevision: 'revision-1', blockDigest: 'digest-1' },
    }),
  };
}
