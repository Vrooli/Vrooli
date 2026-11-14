import { useCallback, useId, useLayoutEffect, useRef, useState } from 'react';
import { Button } from '../../../shared/ui/button';
import { Input } from '../../../shared/ui/input';
import { Textarea } from '../../../shared/ui/textarea';
import { formatPresentationDocument, parsePresentationDocument, type ProductPresentationDocument } from '../../../shared/api/productPresentation';
import { importLegacyPresentationSnapshot } from '../../../shared/lib/presentationLegacyImport';

const MAX_SOURCE_BYTES = 1024 * 1024;
const sizeError = 'The legacy snapshot exceeds the 1 MiB UTF-8 import limit.';

export interface LegacyImportPanelProps {
  document?: ProductPresentationDocument;
  documentText: string;
  /** Changes when the authenticated editor's route or preview scope changes. */
  contextKey: string;
  disabled: boolean;
  onChange: (text: string) => void;
}

/** Local conversion only. Persistence and publication remain explicit editor actions. */
export function LegacyImportPanel({ document, documentText, contextKey, disabled, onChange }: LegacyImportPanelProps) {
  const id = useId();
  const [appKey, setAppKey] = useState('');
  const [locale, setLocale] = useState('');
  const [source, setSource] = useState('');
  const [pending, setPending] = useState<'file' | 'import'>();
  const [error, setError] = useState('');
  const [receipt, setReceipt] = useState<string>();
  const operation = useRef(0);
  const working = useRef(false);
  const mounted = useRef(false);
  const unavailable = disabled || !document;
  const apps = document?.apps.filter(app => !app.enabled && app.visibility === 'private' && app.publication === 'draft') ?? [];
  const matchingApps = apps.filter(app => app.key === appKey);
  const selected = matchingApps.length === 1 ? matchingApps[0] : undefined;
  const pages = selected ? document?.pages.filter(page => page.id === selected.pageId) ?? [] : [];
  const locales = [...new Set(pages.map(page => page.locale).filter(Boolean))];
  const targetCount = pages.filter(page => page.locale === locale).length;
  const targetValid = !!selected && !!locale && targetCount === 1;
  const sourceOversized = new TextEncoder().encode(source).byteLength > MAX_SOURCE_BYTES;

  const cancelOperation = useCallback(() => {
    operation.current++;
    working.current = false;
  }, []);
  const invalidate = useCallback(() => { cancelOperation(); setPending(undefined); }, [cancelOperation]);
  useLayoutEffect(() => {
    mounted.current = true;
    return () => { mounted.current = false; cancelOperation(); };
  }, [cancelOperation]);
  useLayoutEffect(() => {
    const wasPending = working.current;
    invalidate();
    if (wasPending) setError('The editor document or context changed. The import was discarded; review the target and try again.');
  }, [contextKey, disabled, document, documentText, invalidate]);

  function changeInput(change: () => void) {
    invalidate(); setError(''); setReceipt(undefined); change();
  }
  function current(token: number) { return mounted.current && operation.current === token; }
  function begin(kind: 'file' | 'import') {
    invalidate(); working.current = true; setPending(kind); setError(''); setReceipt(undefined);
    return operation.current;
  }
  function finish(token: number) {
    if (current(token)) { working.current = false; setPending(undefined); }
  }
  async function readFile(file: File | undefined) {
    if (!file || unavailable || working.current) return;
    const token = begin('file');
    setSource('');
    if (file.size > MAX_SOURCE_BYTES) { setError(sizeError); finish(token); return; }
    let bytes: ArrayBuffer;
    try { bytes = await file.arrayBuffer(); }
    catch {
      if (current(token)) setError('The selected file could not be read. Choose a readable UTF-8 JSON file.');
      finish(token); return;
    }
    // Native Blob reads cannot be aborted; cancelled/superseded completions
    // must never replace more recent pasted input or another editor context.
    if (!current(token)) return;
    try {
      // Preserve a BOM if present; the adapter owns envelope acceptance and
      // exact-source preservation. Invalid UTF-8 must not become replacement text.
      const value = new TextDecoder('utf-8', { fatal: true, ignoreBOM: true }).decode(bytes);
      if (new TextEncoder().encode(value).byteLength > MAX_SOURCE_BYTES) throw new Error(sizeError);
      setSource(value);
    } catch { setError('The selected file must contain valid UTF-8 JSON within the 1 MiB import limit.'); }
    finally { finish(token); }
  }
  async function importSnapshot() {
    if (unavailable || working.current || !targetValid || !source.trim()) return;
    if (sourceOversized) { setError(sizeError); return; }
    const token = begin('import');
    try {
      // Parse a new generated document from the exact dirty source, never use
      // the saved revision or hand the adapter a mutable editor-state object.
      const base = parsePresentationDocument(documentText);
      const result = await importLegacyPresentationSnapshot(base, source, { adapterVersion: 1, appKey, locale });
      if (!current(token)) return;
      const next = formatPresentationDocument(result.document);
      const receiptJson = JSON.stringify(result.receipt, null, 2);
      finish(token);
      onChange(next);
      setReceipt(receiptJson);
    } catch (cause) {
      if (current(token)) setError(cause instanceof Error ? cause.message : 'The legacy snapshot could not be imported. The editor document was not changed.');
    } finally { finish(token); }
  }

  return <section aria-label="Legacy snapshot import" className="min-w-0 space-y-4 rounded-xl border border-white/10 bg-slate-900/60 p-5">
    <h2 className="text-lg font-semibold">Import a legacy snapshot</h2>
    <p className="text-sm text-slate-300">Private, local conversion only. Import appends recovery content to your current unsaved document. It does not save, publish, enable an app, or qualify product claims and assets. Use Save draft separately after review.</p>
    <fieldset disabled={unavailable} className="min-w-0 space-y-4">
      <legend className="sr-only">Legacy import source and target</legend>
      <label htmlFor={`${id}-app`} className="block text-sm">Import target app
        <select id={`${id}-app`} className="mt-1 block w-full min-w-0 rounded-lg border border-white/20 bg-slate-900 px-3 py-3" value={selected ? appKey : ''} onChange={event => { changeInput(() => { setAppKey(event.target.value); setLocale(''); }); }}>
          <option value="">Choose a disabled, private, draft app</option>
          {apps.map((app, index) => <option key={`${app.key}:${String(index)}`} value={app.key}>{app.name || app.key} · {app.key}</option>)}
        </select>
      </label>
      <label htmlFor={`${id}-locale`} className="block text-sm">Import target locale
        <select id={`${id}-locale`} className="mt-1 block w-full min-w-0 rounded-lg border border-white/20 bg-slate-900 px-3 py-3" disabled={!selected} value={locales.includes(locale) ? locale : ''} onChange={event => { changeInput(() => { setLocale(event.target.value); }); }}>
          <option value="">Choose an exact configured page locale</option>
          {locales.map(value => <option key={value} value={value}>{value}</option>)}
        </select>
      </label>
      {!apps.length && <p className="text-sm text-slate-300">No disabled, private, draft app is configured. Import does not create or change app eligibility.</p>}
      {selected && !locales.length && <p className="text-sm text-slate-300">This app has no configured target page locale.</p>}
      {selected && locale && targetCount !== 1 && <p role="alert" className="text-sm text-red-200">The selected locale must identify exactly one configured target page.</p>}
      <label htmlFor={`${id}-file`} className="block text-sm">Load legacy JSON file (maximum 1 MiB)
        <Input id={`${id}-file`} type="file" accept=".json,application/json" disabled={!!pending} onChange={event => { const file = event.target.files?.[0]; event.target.value = ''; void readFile(file); }} />
      </label>
      <label htmlFor={`${id}-source`} className="block text-sm">Legacy snapshot JSON
        <Textarea id={`${id}-source`} value={source} rows={8} spellCheck={false} autoComplete="off" autoCapitalize="off" className="mt-1 font-mono text-sm" aria-invalid={sourceOversized || !!error} aria-describedby={`${id}-source-help${sourceOversized || error ? ` ${id}-error` : ''}`} onChange={event => { changeInput(() => { setSource(event.target.value); }); }} />
      </label>
      <p id={`${id}-source-help`} className="text-sm text-slate-400">Paste UTF-8 JSON or load a local file, up to 1 MiB. No source URL is fetched. Changing the source or target cancels a pending import.</p>
      <Button className="h-auto whitespace-normal" disabled={!!pending || !targetValid || !source.trim() || sourceOversized} onClick={() => { void importSnapshot(); }}>Import into local document</Button>
    </fieldset>
    {pending && <div className="flex flex-wrap items-center gap-3"><p role="status" className="text-sm text-slate-300">{pending === 'file' ? 'Reading local snapshot…' : 'Converting legacy snapshot locally…'}</p><Button variant="outline" onClick={() => { invalidate(); setError('Import cancelled. The editor document was not changed.'); }}>Cancel import</Button></div>}
    {(sourceOversized || error) && <p id={`${id}-error`} role="alert" className="whitespace-pre-wrap break-words text-sm text-red-200">{sourceOversized ? sizeError : error}</p>}
    {receipt && <section aria-label="Local import receipt" className="min-w-0 space-y-2">
      <h3 className="font-semibold">Local import receipt</h3>
      <p role="status" className="text-sm text-slate-300">Imported into the local document only. Nothing was saved or published. This private receipt describes the last import, not publication approval.</p>
      <pre className="max-h-80 overflow-auto whitespace-pre-wrap break-all rounded-lg bg-black/20 p-3 text-xs" tabIndex={0} aria-label="Import receipt JSON">{receipt}</pre>
    </section>}
  </section>;
}
