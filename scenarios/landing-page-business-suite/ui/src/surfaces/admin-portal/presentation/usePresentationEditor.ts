import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { create } from '@bufbuild/protobuf';
import { Code, ConnectError } from '@connectrpc/connect';
import { ProductPresentationDocumentSchema } from '@vrooli/proto-types/landing-page-business-suite/v1/shared/product_presentation_pb';
import {
  formatPresentationDocument, parsePresentationDocument,
  type ProductPresentationClient, type PresentationEditorResponse, type ResolvedProductPresentation,
} from '../../../shared/api/productPresentation';

type Failure = { kind: 'conflict' | 'authorization' | 'validation' | 'unavailable'; message: string };
export type Activation = { kind: 'publish' | 'rollback'; revision: string; generation: bigint };

function failure(error: unknown): Failure {
  if (error instanceof ConnectError) {
    if (error.code === Code.Aborted || error.code === Code.AlreadyExists) return {
      kind: 'conflict', message: 'The presentation changed on the server. Your local edits are retained. Reload the draft before another save, publish, or rollback.',
    };
    if (error.code === Code.Unauthenticated || error.code === Code.PermissionDenied) return {
      kind: 'authorization', message: 'Administrator access is required. Private editor data has been hidden. Sign in again to continue.',
    };
    if (error.code === Code.InvalidArgument || error.code === Code.FailedPrecondition) return {
      kind: 'validation', message: error.rawMessage,
    };
  }
  return { kind: 'unavailable', message: 'The request could not be completed. Its outcome may be unknown. Reload the draft before another write; your local edits are retained.' };
}

export function usePresentationEditor(variantSlug: string, client: ProductPresentationClient, onDirtyChange?: (dirty: boolean) => void) {
  const [snapshot, setSnapshot] = useState<PresentationEditorResponse>();
  const [text, setText] = useState('');
  const [baseline, setBaseline] = useState('');
  const [busy, setBusy] = useState<string>();
  const [error, setError] = useState<Failure>();
  const [notice, setNotice] = useState('');
  const [preview, setPreview] = useState<ResolvedProductPresentation>();
  const alive = useRef(false);
  const pending = useRef<AbortController>();
  const dirty = text !== baseline;
  const parsed = useMemo(() => {
    if (!text) return { document: undefined, error: '' };
    try { return { document: parsePresentationDocument(text), error: '' }; }
    catch (cause) { return { document: undefined, error: cause instanceof Error ? cause.message : 'Invalid document JSON.' }; }
  }, [text]);
  const writeLocked = !!busy || !snapshot?.state || error?.kind === 'conflict' || error?.kind === 'unavailable' || error?.kind === 'authorization';

  const accept = useCallback((response: PresentationEditorResponse) => {
    if (!response.state || response.variantSlug !== variantSlug || (response.revision && !response.document)) {
      throw new Error('Incomplete editor response');
    }
    const source = formatPresentationDocument(response.document ?? create(ProductPresentationDocumentSchema, { schemaVersion: 1 }));
    setSnapshot(response); setText(source); setBaseline(source); setPreview(undefined);
  }, [variantSlug]);

  const perform = useCallback(async <T,>(label: string, operation: (signal: AbortSignal) => Promise<T>, success: (result: T) => void) => {
    // Synchronous exclusion covers double-clicks before React commits busy state.
    if (pending.current || !alive.current) return;
    const controller = new AbortController();
    pending.current = controller; setBusy(label); setError(undefined); setNotice('');
    try {
      const result = await operation(controller.signal);
      if (!controller.signal.aborted) success(result);
    } catch (cause) {
      if (!controller.signal.aborted) {
        const next = failure(cause);
        setError(next); setPreview(undefined);
        if (next.kind === 'authorization') { setSnapshot(undefined); setText(''); setBaseline(''); }
      }
    } finally {
      if (pending.current === controller) {
        pending.current = undefined;
        if (!controller.signal.aborted) setBusy(undefined);
      }
    }
  }, []);

  useEffect(() => {
    alive.current = true;
    void perform('Loading draft', signal => client.getPresentation({ variantSlug }, { signal, timeoutMs: 30000 }), accept);
    return () => { alive.current = false; pending.current?.abort(); pending.current = undefined; };
  }, [accept, client, perform, variantSlug]);

  useEffect(() => {
    onDirtyChange?.(dirty);
    const guard = (event: BeforeUnloadEvent) => { event.preventDefault(); };
    if (dirty) window.addEventListener('beforeunload', guard);
    return () => { window.removeEventListener('beforeunload', guard); };
  }, [dirty, onDirtyChange]);

  useEffect(() => () => { onDirtyChange?.(false); }, [onDirtyChange]);

  function edit(value: string) {
    if (pending.current) return;
    setText(value); setNotice(''); setPreview(undefined);
    if (error?.kind === 'validation') setError(undefined);
  }
  function reload() {
    if (error?.kind === 'authorization') return;
    void perform('Loading draft', signal => client.getPresentation({ variantSlug }, { signal, timeoutMs: 30000 }), response => {
      accept(response); setNotice('Draft reloaded.');
    });
  }
  function save() {
    if (writeLocked || !dirty || !parsed.document || !snapshot.state) return;
    const expectedGeneration = snapshot.state.generation;
    const document = parsed.document;
    void perform('Saving draft', signal => client.saveDraft({ variantSlug, expectedGeneration, document }, { signal, timeoutMs: 30000 }), response => {
      accept(response); setNotice('Draft saved. The active publication was not changed.');
    });
  }
  function activate(action: Activation) {
    const state = snapshot?.state;
    if (writeLocked || dirty || !state || state.generation !== action.generation) return;
    if (action.kind === 'publish' && (snapshot.revision !== state.draftRevision || action.revision !== state.draftRevision)) return;
    if (action.kind === 'rollback' && (!state.publishedRevisions.includes(action.revision) || action.revision === state.activeRevision)) return;
    const request = { variantSlug, revision: action.revision, expectedGeneration: action.generation };
    void perform(action.kind === 'publish' ? 'Publishing revision' : 'Rolling back',
      signal => action.kind === 'publish' ? client.publish(request, { signal, timeoutMs: 30000 }) : client.rollback(request, { signal, timeoutMs: 30000 }),
      response => { accept(response); setNotice(action.kind === 'publish' ? 'Publication activated.' : 'Rollback activated. Reload the draft to resume draft editing.'); });
  }
  function requestPreview(route: string, locale: string, selectedRevision?: string) {
    const revision = selectedRevision || snapshot?.revision;
    if (writeLocked || !revision || (revision !== snapshot.revision && !snapshot.state?.publishedRevisions.includes(revision)) || !route.startsWith('/') || route.startsWith('//')) return;
    setPreview(undefined);
    void perform('Resolving private preview', signal => client.preview({ variantSlug, revision, route, locale }, { signal, timeoutMs: 30000 }), response => {
      const resolved = response.presentation;
      const diagnostics = resolved?.diagnostics;
      if (!resolved || !diagnostics?.preview || !diagnostics.noindex || !diagnostics.noStore || diagnostics.resolvedRevision !== revision) {
        throw new Error('Preview privacy or revision mismatch');
      }
      setPreview(resolved); setNotice('Authorized revision preview loaded.');
    });
  }
  return { snapshot, text, dirty, parsed, busy, error, notice, preview, writeLocked,
    edit, reload, save, activate, requestPreview, clearPreview: () => { setPreview(undefined); } };
}
