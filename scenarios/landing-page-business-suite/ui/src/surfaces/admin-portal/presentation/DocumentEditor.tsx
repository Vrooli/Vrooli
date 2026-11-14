import { useId, useState } from 'react';
import { Button } from '../../../shared/ui/button';
import { Input } from '../../../shared/ui/input';
import { Textarea } from '../../../shared/ui/textarea';
import { formatPresentationDocument, parsePresentationDocument, type ProductPresentationDocument } from '../../../shared/api/productPresentation';

function move<T>(items: T[], from: number, direction: -1 | 1): T[] {
  const next = [...items];
  const to = from + direction;
  if (to < 0 || to >= next.length) return next;
  const item = next.splice(from, 1)[0];
  if (item !== undefined) next.splice(to, 0, item);
  return next;
}

export function DocumentEditor({ text, document, parseError, disabled, onChange }: {
  text: string; document?: ProductPresentationDocument; parseError: string; disabled: boolean; onChange: (text: string) => void;
}) {
  const id = useId();
  const [pageIndex, setPageIndex] = useState(0);
  const page = document?.pages[pageIndex];
  function update(change: (draft: ProductPresentationDocument) => void) {
    if (disabled || !document) return;
    const draft = parsePresentationDocument(text);
    change(draft); onChange(formatPresentationDocument(draft));
  }
  return <fieldset disabled={disabled} className="min-w-0 space-y-6">
    <legend className="mb-3 text-lg font-semibold">Document configuration</legend>
    <div className="grid min-w-0 gap-6 xl:grid-cols-[minmax(240px,1fr)_minmax(0,2fr)]">
      <section className="min-w-0 rounded-xl border border-white/10 bg-slate-900/60 p-5" aria-label="Page and block order">
        <h2 className="text-base font-semibold">Page and block order</h2>
        <p className="mb-4 mt-1 text-sm text-slate-300">Arrays are authoritative. Move items explicitly; nothing is automatically sorted.</p>
        <ol className="space-y-3">
          {document?.pages.map((item, index) => <li key={`${item.id}:${item.locale}:${String(index)}`} className="rounded-lg border border-white/10 p-3">
            <Button variant={index === pageIndex ? 'secondary' : 'ghost'} size="sm" className="max-w-full whitespace-normal break-all" onClick={() => { setPageIndex(index); }} aria-pressed={index === pageIndex}>{item.id || `Page ${String(index + 1)}`} · {item.locale}</Button>
            <div className="mt-2 flex flex-wrap gap-2">
              <Button variant="outline" size="sm" disabled={index === 0} aria-label={`Move page ${item.id} ${item.locale} up`} onClick={() => { update(draft => { draft.pages = move(draft.pages, index, -1); }); setPageIndex(index - 1); }}>↑</Button>
              <Button variant="outline" size="sm" disabled={index === document.pages.length - 1} aria-label={`Move page ${item.id} ${item.locale} down`} onClick={() => { update(draft => { draft.pages = move(draft.pages, index, 1); }); setPageIndex(index + 1); }}>↓</Button>
            </div>
          </li>)}
        </ol>
        {!document?.pages.length && <p className="text-sm text-slate-400">Add configured pages in the document editor.</p>}
        {page && <div className="mt-6 space-y-4">
          <label className="block text-sm" htmlFor={`${id}-title`}>Page title<Input id={`${id}-title`} value={page.title} onChange={event => { update(draft => { const current = draft.pages[pageIndex]; if (current) current.title = event.target.value; }); }} /></label>
          <label className="block text-sm" htmlFor={`${id}-description`}>Page description<Textarea id={`${id}-description`} value={page.description} onChange={event => { update(draft => { const current = draft.pages[pageIndex]; if (current) current.description = event.target.value; }); }} /></label>
          <h3 className="font-semibold">Blocks in {page.id}</h3>
          <ol className="space-y-3">{page.blocks.map((block, index) => <li key={`${block.id}:${String(index)}`} className="rounded-lg bg-white/5 p-3">
            <p className="break-all text-sm">{block.id} <span className="text-slate-400">· {block.kind} / {block.variant}</span></p>
            <div className="mt-2 flex gap-2">
              {([-1, 1] as const).map(direction => <Button key={direction} variant="outline" size="sm" aria-label={`Move block ${block.id} ${direction === -1 ? 'up' : 'down'}`} disabled={index + direction < 0 || index + direction >= page.blocks.length} onClick={() => { update(draft => { const current = draft.pages[pageIndex]; if (current) current.blocks = move(current.blocks, index, direction); }); }}>{direction === -1 ? '↑' : '↓'}</Button>)}
            </div>
          </li>)}</ol>
        </div>}
      </section>
      <section className="min-w-0 rounded-xl border border-white/10 bg-slate-900/60 p-5">
        <label htmlFor={`${id}-source`} className="text-base font-semibold">Complete document JSON</label>
        <p id={`${id}-help`} className="mb-4 mt-1 text-sm text-slate-300">Edit all bundle, app, page, block, fixture, asset, capability, and localized string fields here. Add or remove items in their arrays. Content uses generated typed objects such as content.product_hero. Unknown fields are rejected, not silently removed. Server validation remains authoritative.</p>
        <Textarea id={`${id}-source`} value={text} onChange={event => { onChange(event.target.value); }} rows={30} spellCheck={false} autoComplete="off" autoCapitalize="off" aria-invalid={!!parseError} aria-describedby={`${id}-help${parseError ? ` ${id}-error` : ''}`} className="min-h-[32rem] resize-y font-mono text-sm leading-6" />
        {parseError && <p id={`${id}-error`} role="alert" className="mt-3 whitespace-pre-wrap break-words text-sm text-red-300">{parseError}</p>}
      </section>
    </div>
  </fieldset>;
}
