import { useCallback, useEffect, useId, useState } from 'react';
import { useAdminAuth } from '../../../app/providers/useAdminAuth';
import { Button } from '../../../shared/ui/button';
import { Input } from '../../../shared/ui/input';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '../../../shared/ui/dialog';
import { previewPresentationDocument, productPresentationClient, type ProductPresentationClient } from '../../../shared/api/productPresentation';
import { DocumentEditor } from './DocumentEditor';
import { LegacyImportPanel } from './LegacyImportPanel';
import { RevisionPreview } from './RevisionPreview';
import { usePresentationEditor, type Activation } from './usePresentationEditor';
import { useDocumentPreview, type DocumentPreviewRequest } from './useDocumentPreview';
import './presentationEditor.css';

export interface PresentationEditorRouteProps {
  /** Parent supplies the route's explicit variant. No public-assignment lookup. */
  variantSlug: string;
  linkBase?: string;
  client?: ProductPresentationClient;
  /** Parent can connect its router's dirty-navigation guard without a new router. */
  onDirtyChange?: (dirty: boolean) => void;
}

/** Mount under the existing AdminAuthProvider and admin layout; no route changes. */
export function PresentationEditorRoute(props: PresentationEditorRouteProps) {
  const auth = useAdminAuth();
  useEffect(() => {
    const robots = document.createElement('meta');
    robots.name = 'robots'; robots.content = 'noindex, nofollow';
    document.head.append(robots);
    return () => { robots.remove(); };
  }, []);
  if (auth.isSessionLoading) return <p role="status">Checking administrator session…</p>;
  if (!auth.isAuthenticated) return <p role="alert">Sign in as an administrator to edit presentations.</p>;
  if (!props.variantSlug.trim()) return <p role="alert">Select an explicit presentation variant before opening the editor.</p>;
  return <AuthenticatedEditor key={`${props.variantSlug}:${auth.user?.email ?? ''}`} {...props} />;
}

function AuthenticatedEditor({ variantSlug, client = productPresentationClient, onDirtyChange, linkBase }: PresentationEditorRouteProps) {
  const editor = usePresentationEditor(variantSlug, client, onDirtyChange);
  const [confirmation, setConfirmation] = useState<Activation | 'reload'>();
  const [rollbackRevision, setRollbackRevision] = useState('');
  const [route, setRoute] = useState('/');
  const [locale, setLocale] = useState('');
  const [previewMode, setPreviewMode] = useState<'document' | 'revision'>('document');
  const [previewRevision, setPreviewRevision] = useState('');
  const requestDocumentPreview = useCallback<DocumentPreviewRequest>((input, signal) => previewPresentationDocument(client, input, signal), [client]);
  const localPreview = useDocumentPreview({ variantSlug, document: editor.parsed.document, route, locale,
    enabled: previewMode === 'document' && !!editor.snapshot && !editor.busy && editor.error?.kind !== 'authorization',
  }, requestDocumentPreview);
  const id = useId();
  const state = editor.snapshot?.state;
  const confirmActivation = confirmation && confirmation !== 'reload' ? confirmation : undefined;
  const canPublish = !editor.writeLocked && !editor.dirty && !!editor.snapshot?.revision && editor.snapshot.revision === state?.draftRevision && state.activeRevision !== state.draftRevision;
  const canRollback = !editor.writeLocked && !editor.dirty && !!rollbackRevision && state?.publishedRevisions.includes(rollbackRevision) && rollbackRevision !== state.activeRevision;

  if (editor.error?.kind === 'authorization' || localPreview.unauthorized) return <p role="alert" className="rounded-xl border border-amber-400/40 bg-slate-900 p-6 text-amber-100">{editor.error?.message ?? localPreview.error}</p>;
  return <div className="space-y-6 text-slate-100" data-testid="presentation-editor">
    <header className="flex flex-wrap items-start justify-between gap-4">
      <div><p className="text-xs uppercase tracking-widest text-slate-400">Presentation configuration</p><h1 className="mt-2 text-3xl font-semibold">Presentation editor</h1><p className="mt-2 text-sm text-slate-300">Variant: <span className="break-all font-mono">{variantSlug}</span> · Administrator-only workspace</p></div>
      <Button variant="outline" disabled={!!editor.busy} onClick={() => { if (editor.dirty) setConfirmation('reload'); else editor.reload(); }}>Reload draft</Button>
    </header>
    <div className="grid gap-3 rounded-xl border border-white/10 bg-slate-900/70 p-5 sm:grid-cols-3">
      <div><p className="text-xs uppercase text-slate-400">Loaded generation</p><p className="mt-1 font-mono">{state?.generation.toString() ?? 'Not loaded'}</p></div>
      <div><p className="text-xs uppercase text-slate-400">Draft revision</p><p className="mt-1 break-all font-mono text-xs">{state?.draftRevision || 'No saved draft'}</p></div>
      <div><p className="text-xs uppercase text-slate-400">Active revision</p><p className="mt-1 break-all font-mono text-xs">{state?.activeRevision || 'Not published'}</p></div>
    </div>
    <p role="status" aria-live="polite" className="text-sm text-slate-300">{editor.busy ? `${editor.busy}…` : editor.notice || (editor.dirty ? 'Unsaved document changes.' : 'No unsaved changes.')}</p>
    {editor.error && <div role="alert" className="rounded-xl border border-red-400/40 bg-red-950/30 p-4 text-red-200"><h2 className="font-semibold">{editor.error.kind === 'validation' ? 'Server validation rejected the request' : editor.error.kind === 'conflict' ? 'Generation conflict' : 'Request not confirmed'}</h2><p className="mt-1 whitespace-pre-wrap break-words text-sm">{editor.error.message}</p></div>}
    {editor.snapshot && <>
      <div className="presentation-editing-grid">
      <div className="min-w-0 space-y-6">
      <DocumentEditor text={editor.text} document={editor.parsed.document} parseError={editor.parsed.error} disabled={!!editor.busy || !!confirmation} onChange={editor.edit} />
      <LegacyImportPanel document={editor.parsed.document} documentText={editor.text} contextKey={JSON.stringify([variantSlug, route, locale, previewMode])} disabled={editor.writeLocked || !!confirmation} onChange={editor.edit} />
      <section className="flex flex-wrap items-center justify-between gap-4 rounded-xl border border-white/10 bg-slate-900/60 p-5" aria-label="Save draft">
        <div><h2 className="font-semibold">Save draft only</h2><p className="mt-1 text-sm text-slate-300">Save validates the complete document and updates the draft. It never activates publication.</p></div>
        <Button disabled={editor.writeLocked || !editor.dirty || !editor.parsed.document} onClick={editor.save}>Save draft</Button>
      </section>
      <section className="space-y-4 rounded-xl border border-white/10 bg-slate-900/60 p-5" aria-label="Publication controls">
        <h2 className="text-lg font-semibold">Publication</h2>
        <p className="text-sm text-slate-300">Publish or restore one saved revision. The server must qualify all references and claims. Neither action changes commerce or installer configuration.</p>
        {editor.dirty && <p className="text-sm text-amber-200">Save or discard local edits before publishing or rolling back. Local preview does not save them.</p>}
        <div className="flex flex-wrap items-end gap-3">
          <Button disabled={!canPublish} onClick={() => { if (state) setConfirmation({ kind: 'publish', revision: state.draftRevision, generation: state.generation }); }}>Publish draft…</Button>
          <label className="min-w-0 flex-1 text-sm" htmlFor={`${id}-rollback`}>Retained published revision
            <select id={`${id}-rollback`} className="mt-1 block w-full min-w-0 rounded-lg border border-white/20 bg-slate-900 px-3 py-3 text-sm" disabled={!!editor.busy || !!confirmation} value={rollbackRevision} onChange={event => { setRollbackRevision(event.target.value); }}>
              <option value="">Choose a revision</option>
              {state?.publishedRevisions.map(revision => <option key={revision} value={revision} disabled={revision === state.activeRevision}>{revision}{revision === state.activeRevision ? ' (active)' : ''}</option>)}
            </select>
          </label>
          <Button variant="outline" disabled={!canRollback} onClick={() => { if (state) setConfirmation({ kind: 'rollback', revision: rollbackRevision, generation: state.generation }); }}>Roll back…</Button>
        </div>
      </section>
      </div>
      <div className="presentation-preview-column min-w-0 space-y-4">
      <section id="presentation-preview" className="space-y-4 rounded-xl border border-white/10 bg-slate-900/60 p-5" aria-label="Preview controls">
        <h2 className="text-lg font-semibold">Private preview</h2>
        <fieldset className="flex flex-wrap gap-4"><legend className="mb-2 text-sm">Preview source</legend>
          <label className="flex items-center gap-2 text-sm"><input type="radio" name={`${id}-preview-mode`} checked={previewMode === 'document'} onChange={() => { setPreviewMode('document'); editor.clearPreview(); }} />Local document (read-only)</label>
          <label className="flex items-center gap-2 text-sm"><input type="radio" name={`${id}-preview-mode`} checked={previewMode === 'revision'} onChange={() => { setPreviewMode('revision'); editor.clearPreview(); }} />Saved revision</label>
        </fieldset>
        {previewMode === 'document' && <p className="text-sm text-slate-300">Valid edits resolve here after 300ms. Nothing is saved or published automatically.</p>}
        {previewMode === 'revision' && <label className="block text-sm" htmlFor={`${id}-preview-revision`}>Revision to inspect
          <select id={`${id}-preview-revision`} className="mt-1 block w-full min-w-0 rounded-lg border border-white/20 bg-slate-900 px-3 py-3" disabled={!!editor.busy} value={previewRevision} onChange={event => { setPreviewRevision(event.target.value); editor.clearPreview(); }}>
            <option value="">Loaded draft/revision</option>
            {state?.publishedRevisions.map(revision => <option key={revision} value={revision}>{revision}</option>)}
          </select>
        </label>}
        <p className="break-all font-mono text-xs text-slate-400">Loaded revision: {editor.snapshot.revision || 'Save a draft first'}</p>
        <div className="flex flex-wrap items-end gap-3">
          <label className="min-w-48 flex-1 text-sm" htmlFor={`${id}-route`}>Page route<Input id={`${id}-route`} value={route} disabled={!!editor.busy} onChange={event => { setRoute(event.target.value); editor.clearPreview(); }} /></label>
          <label className="text-sm" htmlFor={`${id}-locale`}>Locale<Input id={`${id}-locale`} value={locale} disabled={!!editor.busy} placeholder={editor.parsed.document?.bundle?.defaultLocale} onChange={event => { setLocale(event.target.value); editor.clearPreview(); }} /></label>
          {previewMode === 'revision' && <Button variant="outline" disabled={editor.writeLocked || !editor.snapshot.revision || !route.startsWith('/') || route.startsWith('//')} onClick={() => { editor.requestPreview(route, locale, previewRevision); }}>Preview saved revision</Button>}
        </div>
        <p className="text-sm text-slate-300">A blank locale uses the configured default. Preview stays in this authenticated editor; no public revision link is created.</p>
      </section>
      {previewMode === 'document' ? <>
        {editor.parsed.error && <p role="alert" className="text-sm text-red-200">Preview hidden until the document syntax is valid.</p>}
        {localPreview.error && <p role="alert" className="whitespace-pre-wrap break-words text-sm text-red-200">{localPreview.error}</p>}
        {(localPreview.status === 'loading' || localPreview.status === 'debouncing') && <p aria-live="polite" className="text-sm text-slate-300">{localPreview.status === 'loading' ? 'Resolving unsaved document preview…' : 'Waiting for local edits…'}</p>}
        {localPreview.presentation && <RevisionPreview value={localPreview.presentation} linkBase={linkBase} ephemeral />}
      </> : editor.preview && <RevisionPreview value={editor.preview} linkBase={linkBase} />}
      </div>
      </div>
    </>}
    <Dialog open={!!confirmation} onOpenChange={open => { if (!open && !editor.busy) setConfirmation(undefined); }}>
      <DialogContent>
        <DialogHeader><DialogTitle>{confirmation === 'reload' ? 'Discard local edits and reload?' : confirmActivation?.kind === 'publish' ? 'Publish this draft?' : 'Restore this published revision?'}</DialogTitle>
          <DialogDescription>{confirmation === 'reload' ? 'Reload replaces all unsaved text with the current server draft. Cancel to keep your edits.' : 'This explicitly changes the active public revision, subject to server validation and the loaded generation. Saving alone does not perform this action.'}</DialogDescription></DialogHeader>
        {confirmActivation && <dl className="space-y-2 break-all text-sm text-slate-200"><dt>Revision</dt><dd className="font-mono">{confirmActivation.revision}</dd><dt>Expected generation</dt><dd>{confirmActivation.generation.toString()}</dd></dl>}
        <DialogFooter><Button variant="ghost" onClick={() => { setConfirmation(undefined); }}>Cancel</Button>
          <Button variant={confirmation === 'reload' ? 'destructive' : 'default'} onClick={() => { if (confirmation === 'reload') editor.reload(); else if (confirmActivation) editor.activate(confirmActivation); setConfirmation(undefined); }}>{confirmation === 'reload' ? 'Discard and reload' : confirmActivation?.kind === 'publish' ? 'Confirm publish' : 'Confirm rollback'}</Button></DialogFooter>
      </DialogContent>
    </Dialog>
  </div>;
}
