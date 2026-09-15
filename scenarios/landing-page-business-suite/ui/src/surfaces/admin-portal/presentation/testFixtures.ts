import { create } from '@bufbuild/protobuf';
import { vi } from 'vitest';
import { PresentationEditorResponseSchema, PreviewPresentationResponseSchema } from '@vrooli/proto-types/landing-page-business-suite/v1/product_presentation_pb';
import { ProductPresentationDocumentSchema, ResolvedProductPresentationSchema } from '@vrooli/proto-types/landing-page-business-suite/v1/shared/product_presentation_pb';
import type { ProductPresentationClient } from '../../../shared/api/productPresentation';
import type { AdminAuthContextValue } from '../../../app/providers/AdminAuthContext';

export const admin: AdminAuthContextValue = {
  isAuthenticated: true, isSessionLoading: false, user: { email: 'editor@example.test' },
  login: () => Promise.resolve(), logout: () => undefined, canResetDemoData: false,
};
export function documentFixture() {
  return create(ProductPresentationDocumentSchema, {
    schemaVersion: 1,
    bundle: { key: 'example', name: 'Configured bundle', appOrder: ['second-app', 'first-app'], defaultLocale: 'fr', locales: ['fr', 'en'] },
    apps: [{ key: 'first-app', name: 'Configured private app', visibility: 'private', preservationRef: 'PRIVATE-REFERENCE' }],
    pages: [
      { id: 'z-page', locale: 'fr', title: 'Configured first page', description: 'Configured description', blocks: [
        { id: 'z-block', kind: 'faq', version: 1, variant: 'accordion', content: { value: { case: 'faq', value: { heading: 'Configured questions', items: [{ question: 'First question?', answer: 'Configured answer', accessibleLabel: 'First question?' }] } } } },
        { id: 'a-block', kind: 'footer', version: 1, variant: 'defined', content: { value: { case: 'footer', value: { label: 'Configured footer', links: [] } } } },
      ] },
      { id: 'a-page', locale: 'en', title: 'Configured second page', description: 'Other description', blocks: [] },
    ],
    strings: { fr: { values: { 'custom.copy': 'Configured localized copy' } } },
  });
}
export function snapshot(generation = 9007199254740993n) {
  return create(PresentationEditorResponseSchema, {
    variantSlug: 'control', revision: 'draft-revision', document: documentFixture(),
    state: { generation, draftRevision: 'draft-revision', activeRevision: 'active-revision', publishedRevisions: ['older-revision', 'active-revision'] },
  });
}
export function previewFixture() {
  return create(ResolvedProductPresentationSchema, {
    schemaVersion: 1, mode: 'empty', scope: 'bundle', page: {
      id: 'empty', locale: 'fr', title: 'Configured preview',
      theme: { variant: 'signal', primary: '#202924', background: '#faf9f6', accent: '#b94721' },
      navigation: { label: 'Configured navigation' }, footer: { label: 'Configured footer' },
      blocks: [{ id: 'closing', kind: 'closing-action', version: 1, variant: 'plain', content: { value: { case: 'closingAction', value: { heading: 'Configured preview heading', description: 'Configured closing copy', actions: [{ kind: 'purchase', label: 'Configured action', accessibleLabel: 'Configured action', planRef: 'configured-plan' }] } } } }],
      display: { shell: {
        brandName: 'Configured brand', brandMark: 'suite', brandTarget: '/', skipLabel: 'Configured skip', menuLabel: 'Configured menu',
        footerBrandName: 'Configured owner', footerBrandMark: 'suite', footerBrandTarget: '/', footerTagline: 'Configured tagline',
        unavailableReason: 'Configured unavailable reason', previewLabel: 'Configured private banner',
      } },
    },
    diagnostics: { preview: true, noindex: true, noStore: true, requestedRevision: 'draft-revision', resolvedRevision: 'draft-revision', blockDigest: 'example-digest' },
  });
}
export function clientFixture() {
  return {
    getPresentation: vi.fn<ProductPresentationClient['getPresentation']>().mockResolvedValue(snapshot()),
    saveDraft: vi.fn<ProductPresentationClient['saveDraft']>().mockResolvedValue(snapshot(9007199254740994n)),
    publish: vi.fn<ProductPresentationClient['publish']>().mockResolvedValue(snapshot(9007199254740994n)),
    rollback: vi.fn<ProductPresentationClient['rollback']>().mockResolvedValue(snapshot(9007199254740994n)),
    preview: vi.fn<ProductPresentationClient['preview']>().mockResolvedValue(create(PreviewPresentationResponseSchema, { presentation: previewFixture() })),
  };
}
