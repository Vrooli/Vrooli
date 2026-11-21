/** Isolated browser fixture: real editor/renderer, synthetic read-only API responses. */
import { create, fromJsonString } from '@bufbuild/protobuf';
import { PresentationEditorResponseSchema, PreviewPresentationResponseSchema } from '@vrooli/proto-types/landing-page-business-suite/v1/product_presentation_pb';
import { ProductPresentationDocumentSchema, ResolvedProductPresentationSchema } from '@vrooli/proto-types/landing-page-business-suite/v1/shared/product_presentation_pb';
import type { ProductPresentationClient } from '../../../shared/api/productPresentation';
import { AdminAuthContext } from '../../../app/providers/AdminAuthContext';
import { PresentationEditorRoute } from './PresentationEditorRoute';
import signal from '../../public-landing/presentation/fixtures/signal.json';
import '../../../styles.css';

function presentation() { return fromJsonString(ResolvedProductPresentationSchema, JSON.stringify(signal.presentation)); }
const original = presentation();
const document = create(ProductPresentationDocumentSchema, { schemaVersion: 1, bundle: { name: 'Synthetic editor review', defaultLocale: 'en', locales: ['en'] }, pages: original.page ? [original.page] : [] });
const client: ProductPresentationClient = {
  getPresentation: () => Promise.resolve(create(PresentationEditorResponseSchema, { variantSlug: 'control', revision: 'review-draft', document,
    state: { generation: 1n, draftRevision: 'review-draft' } })),
  preview: request => {
    const p = presentation(); const d = p.diagnostics;
    if (!p.page || !d) return Promise.reject(new Error('Missing review fixture'));
    Object.assign(d, { preview: true, noindex: true, noStore: true, fallback: false, requestedRoute: request.route, resolvedRoute: request.route,
      requestedVariant: request.variantSlug, resolvedVariant: request.variantSlug, requestedRevision: request.document ? 'a'.repeat(64) : request.revision, resolvedRevision: request.document ? 'a'.repeat(64) : request.revision, blockDigest: 'sha256:' + 'b'.repeat(64), locale: 'en' });
    if (request.document?.pages?.[0]) p.page.title = request.document.pages[0].title ?? '';
    return Promise.resolve(create(PreviewPresentationResponseSchema, { presentation: p }));
  },
  saveDraft: () => Promise.reject(new Error('Writes forbidden in isolated review')),
  publish: () => Promise.reject(new Error('Writes forbidden in isolated review')),
  rollback: () => Promise.reject(new Error('Writes forbidden in isolated review')),
};
export function EditorReview() {
  return <AdminAuthContext.Provider value={{ isAuthenticated: true, isSessionLoading: false, user: { email: 'review@example.test' }, login: () => Promise.resolve(), logout: () => undefined, canResetDemoData: false }}>
    <div className="min-h-screen bg-slate-950 p-4 text-slate-100"><PresentationEditorRoute variantSlug="control" client={client} /></div>
  </AdminAuthContext.Provider>;
}
