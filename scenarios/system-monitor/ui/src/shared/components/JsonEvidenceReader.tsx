import { useEffect, useId, useMemo, useRef, useState } from 'react';
import Editor, { loader, type OnMount } from '@monaco-editor/react';
import { Copy, MoreHorizontal, Search, WrapText } from 'lucide-react';
import * as monaco from 'monaco-editor';
import EditorWorker from 'monaco-editor/editor/editor.worker.js?worker';
import JsonWorker from 'monaco-editor/language/json/json.worker.js?worker';
import './JsonEvidenceReader.css';

// Bundle editor and workers with the application; never load editor code from a CDN.
globalThis.MonacoEnvironment = {
  getWorker: (_workerId: string, label: string) => label === 'json' ? new JsonWorker() : new EditorWorker(),
};
loader.config({ monaco });

export interface JsonEvidenceReaderProps {
  title: string;
  value: string;
  description?: string;
  onClose: () => void;
}

/** Insert whitespace only outside strings; preserve numbers, keys and duplicate fields. */
export function formatJsonEvidence(value: string): string {
  try { JSON.parse(value); } catch { return value; }
  const chunks: string[] = [];
  let depth = 0, inString = false, escaped = false, stringStart = 0;
  const indent = () => '  '.repeat(depth);
  for (let index = 0; index < value.length; index += 1) {
    const character = value.charAt(index);
    if (inString) {
      if (escaped) escaped = false;
      else if (character === '\\') escaped = true;
      else if (character === '"') { inString = false; chunks.push(value.slice(stringStart, index + 1)); }
    } else if (character === '"') { inString = true; stringStart = index; }
    else if (character === '{' || character === '[') { depth += 1; chunks.push(`${character}\n${indent()}`); }
    else if (character === '}' || character === ']') { depth = Math.max(0, depth - 1); chunks.push(`\n${indent()}${character}`); }
    else if (character === ',') chunks.push(`,\n${indent()}`);
    else if (character === ':') chunks.push(': ');
    else if (!/\s/.test(character)) chunks.push(character);
  }
  return chunks.join('');
}

/** A read-only, full-content JSON workspace usable by any evidence surface. */
export function JsonEvidenceReader({ title, value, description, onClose }: JsonEvidenceReaderProps) {
  const formatted = useMemo(() => formatJsonEvidence(value), [value]);
  const largeDocument = value.length > 128 * 1024;
  const partOffsets = useMemo(() => {
    const offsets = [0];
    if (!value.length) return [0, 0];
    while ((offsets[offsets.length - 1] ?? 0) < value.length) {
      const start = offsets[offsets.length - 1] ?? 0;
      let end = Math.min(start + 4096, value.length);
      if (end < value.length && value.charCodeAt(end) >= 0xdc00 && value.charCodeAt(end) <= 0xdfff && value.charCodeAt(end - 1) >= 0xd800 && value.charCodeAt(end - 1) <= 0xdbff) end -= 1;
      offsets.push(end);
    }
    return offsets;
  }, [value]);
  const editor = useRef<monaco.editor.IStandaloneCodeEditor | null>(null);
  const heading = useRef<HTMLHeadingElement>(null);
  const partText = useRef<HTMLTextAreaElement>(null);
  const [wrap, setWrap] = useState(true);
  const [copyStatus, setCopyStatus] = useState('');
  const [moreOpen, setMoreOpen] = useState(false);
  const moreId = useId();
  const [fontSize, setFontSize] = useState(14);
  const [textMode, setTextMode] = useState(false);
  const [part, setPart] = useState(0);
  const [search, setSearch] = useState('');
  const [searchFrom, setSearchFrom] = useState(0);
  const [searchStatus, setSearchStatus] = useState('');
  const [matchSelection, setMatchSelection] = useState<{ start: number; end: number } | null>(null);
  const showTextParts = textMode;
  const visiblePart = Math.max(0, Math.min(part, partOffsets.length - 2));
  const moreButton = useRef<HTMLButtonElement>(null);
  const [theme, setTheme] = useState(() => document.documentElement.dataset.theme === 'dark' ? 'vs-dark' : 'vs');

  useEffect(() => {
    heading.current?.focus();
    const observer = new MutationObserver(() => { setTheme(document.documentElement.dataset.theme === 'dark' ? 'vs-dark' : 'vs'); });
    observer.observe(document.documentElement, { attributes: true, attributeFilter: ['data-theme'] });
    return () => { observer.disconnect(); };
  }, []);

  const onMount: OnMount = instance => { editor.current = instance; };
  const copy = async () => {
    try { await navigator.clipboard.writeText(value); setCopyStatus('Copied metadata'); }
    catch { setCopyStatus('Could not copy metadata'); }
  };
  useEffect(() => { setTextMode(false); setPart(0); setSearch(''); setSearchFrom(0); setSearchStatus(''); setMatchSelection(null); }, [value]);

  useEffect(() => {
    if (!matchSelection || !showTextParts || !partText.current) return;
    partText.current.focus();
    partText.current.setSelectionRange(matchSelection.start, matchSelection.end);
  }, [matchSelection, showTextParts]);

  const findInText = () => {
    if (!search) { setSearchStatus('Enter text to find.'); return; }
    let index = value.indexOf(search, searchFrom);
    const wrapped = index < 0 && searchFrom > 0;
    if (wrapped) index = value.indexOf(search);
    if (index < 0) { setSearchStatus('No match in this document.'); return; }
    const nextPart = Math.max(0, partOffsets.findIndex((offset, i) => i < partOffsets.length - 1 && index >= offset && index < (partOffsets[i + 1] ?? value.length)));
    setPart(nextPart);
    setSearchFrom(index + Math.max(1, search.length));
    const partStart = partOffsets[nextPart] ?? 0;
    const partEnd = partOffsets[nextPart + 1] ?? value.length;
    setMatchSelection({ start: index - partStart, end: Math.min(index + search.length, partEnd) - partStart });
    setSearchStatus(`Match begins in part ${nextPart + 1}${index + search.length > partEnd ? ' and continues in later parts' : ''}${wrapped ? ' · Search wrapped' : ''}.`);
  };

  return <section className="json-evidence-reader" aria-label={title}>
    <header className="json-evidence-reader__header">
      <button type="button" className="header-button" onClick={onClose}>← Back</button>
      <h2 ref={heading} tabIndex={-1}>{title}</h2>
      <span>JSON · Read only</span>
    </header>
    <p className="json-evidence-reader__context">{description}<span className="json-evidence-reader__mobile-status">{description && ' · '}JSON · Read only</span></p>
    <div className="json-evidence-reader__tools" onKeyDown={event => { if (event.key === 'Escape' && moreOpen) { setMoreOpen(false); moreButton.current?.focus(); } }}>
      <button type="button" className="header-button" onClick={() => { setTextMode(!showTextParts); setMatchSelection(null); setMoreOpen(false); }}>{showTextParts ? 'Return to editor' : 'Read text in parts'}</button>
      {!showTextParts && <>
      <button type="button" className="json-evidence-reader__tool" aria-label="Find" title="Find" onClick={() => { void editor.current?.getAction('actions.find')?.run(); }}><Search aria-hidden="true" size={18} /></button>
      <button type="button" className="json-evidence-reader__tool" aria-label="Wrap" title="Wrap" aria-pressed={wrap} onClick={() => { setWrap(!wrap); }}><WrapText aria-hidden="true" size={18} /></button>
      </>}
      <button type="button" className="json-evidence-reader__tool" aria-label="Copy metadata" title="Copy metadata" onClick={() => void copy()}><Copy aria-hidden="true" size={18} /></button>
      {!showTextParts && <>
      <button ref={moreButton} type="button" className="json-evidence-reader__tool" aria-label="More document tools" title="More document tools" aria-expanded={moreOpen} aria-controls={moreId} onClick={() => { setMoreOpen(!moreOpen); }}><MoreHorizontal aria-hidden="true" size={18} /></button>
      {moreOpen && <div id={moreId} className="json-evidence-reader__more" aria-label="Document tools">
        <strong>Document tools</strong>
        <div className="json-evidence-reader__more-actions">
          <button type="button" aria-label="Find in document" onClick={() => { setMoreOpen(false); void editor.current?.getAction('actions.find')?.run(); }}>Find</button>
          <button type="button" aria-label="Wrap lines" aria-pressed={wrap} onClick={() => { setWrap(!wrap); }}>Wrap</button>
          <button type="button" aria-label="Copy all metadata" onClick={() => { setMoreOpen(false); moreButton.current?.focus(); void copy(); }}>Copy all</button>
        </div>
        <div className="json-evidence-reader__size"><span>Text size</span><button type="button" onClick={() => { setFontSize(size => Math.max(10, size - 1)); }} aria-label="Decrease text size">A−</button><button type="button" onClick={() => { setFontSize(size => Math.min(24, size + 1)); }} aria-label="Increase text size">A+</button></div>
        <p>Read only · Complete document</p>
      </div>}
      </>}
      <span role="status">{copyStatus}</span>
    </div>
    {showTextParts ? <div className="json-evidence-reader__parts">
      <p>Read the complete original document in parts. Copy metadata copies the entire document.</p>
      <form onSubmit={event => { event.preventDefault(); findInText(); }}>
        <label>Find in complete document <input value={search} onChange={event => { setSearch(event.target.value); setSearchFrom(0); setSearchStatus(''); }} /></label>
        <button type="submit" className="header-button">Find next</button>
      </form>
      <p role="status">{searchStatus}</p>
      <div className="json-evidence-reader__part-controls">
        <button type="button" className="header-button" disabled={visiblePart === 0} onClick={() => setPart(visiblePart - 1)}>Previous part</button>
        <span>Part {visiblePart + 1} of {partOffsets.length - 1}</span>
        <button type="button" className="header-button" disabled={visiblePart >= partOffsets.length - 2} onClick={() => setPart(visiblePart + 1)}>Next part</button>
      </div>
      <textarea ref={partText} readOnly aria-label={`Document text, part ${visiblePart + 1} of ${partOffsets.length - 1}`} value={value.slice(partOffsets[visiblePart], partOffsets[visiblePart + 1])} />
    </div> : <div className="json-evidence-reader__editor">
      <Editor height="100%" language="json" theme={theme} value={formatted} onMount={onMount}
        options={{ readOnly: true, domReadOnly: true, automaticLayout: true, fontSize, wordWrap: wrap ? 'on' : 'off', minimap: { enabled: false }, scrollBeyondLastLine: false, accessibilitySupport: largeDocument ? 'auto' : 'on', cursorBlinking: 'solid', smoothScrolling: false }} />
    </div>}
  </section>;
}
