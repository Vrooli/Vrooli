// provider-free-exception: this local prop-fed panel has no auth, router or API context; the authenticated editor mount is tested separately.
import { afterEach, describe, expect, it, vi } from 'vitest';
import { act, cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { create } from '@bufbuild/protobuf';
import { PresentationAppSchema } from '@vrooli/proto-types/landing-page-business-suite/v1/shared/product_presentation_pb';
import * as adapter from '../../../shared/lib/presentationLegacyImport';
import { formatPresentationDocument, parsePresentationDocument } from '../../../shared/api/productPresentation';
import { documentFixture } from './testFixtures';
import { LegacyImportPanel, type LegacyImportPanelProps } from './LegacyImportPanel';

afterEach(() => { cleanup(); vi.restoreAllMocks(); });
function baseDocument() {
  const document = documentFixture();
  document.apps = [
    create(PresentationAppSchema, { key: 'live', name: 'Published app', enabled: true, visibility: 'public', publication: 'published', pageId: 'live-page' }),
    create(PresentationAppSchema, { key: 'eligible', name: 'Configured recovery app', enabled: false, visibility: 'private', publication: 'draft', pageId: 'z-page' }),
    create(PresentationAppSchema, { key: 'other', name: 'Other recovery app', enabled: false, visibility: 'private', publication: 'draft', pageId: 'a-page' }),
    create(PresentationAppSchema, { key: 'enabled', enabled: true, visibility: 'private', publication: 'draft', pageId: 'enabled-page' }),
    create(PresentationAppSchema, { key: 'public', enabled: false, visibility: 'public', publication: 'draft', pageId: 'public-page' }),
    create(PresentationAppSchema, { key: 'published', enabled: false, visibility: 'private', publication: 'published', pageId: 'published-page' }),
  ];
  return document;
}
function mount(overrides: Partial<LegacyImportPanelProps> = {}) {
  const document = baseDocument();
  const props = { document, documentText: formatPresentationDocument(document), contextKey: 'control:/', disabled: false, onChange: vi.fn(), ...overrides };
  return { props, ...render(<LegacyImportPanel {...props} />) };
}
function selectTarget() {
  fireEvent.change(screen.getByRole('combobox', { name: 'Import target app' }), { target: { value: 'eligible' } });
  fireEvent.change(screen.getByRole('combobox', { name: 'Import target locale' }), { target: { value: 'fr' } });
}
function paste(value = '{"sections":[]}') { fireEvent.change(screen.getByRole('textbox', { name: 'Legacy snapshot JSON' }), { target: { value } }); }
const importButton = () => screen.getByRole('button', { name: 'Import into local document' });
const legacySource = JSON.stringify({ variant: { slug: 'recovered' }, sections: [{ key: 'faq', section_type: 'faq', order: 1, content: { title: 'Imported questions', items: [{ question: 'Imported question?', answer: 'Imported answer.' }] } }], '<script>unsafe()</script>': 'preserved source field' });
// jsdom File lacks Blob.arrayBuffer. Supply the browser API on each test file,
// with exact bytes and controllable read completion; no production polyfill.
function localFile(source: string | Uint8Array<ArrayBuffer>, name = 'snapshot.json') {
  const bytes = typeof source === 'string' ? new TextEncoder().encode(source) : source;
  return Object.assign(new File([bytes], name, { type: 'application/json' }), {
    arrayBuffer: vi.fn<() => Promise<ArrayBuffer>>().mockResolvedValue(bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength)),
  });
}

describe('local legacy snapshot import', () => {
  it('appends through the real adapter using the current dirty document and displays the private receipt as text', async () => {
    const document = baseDocument(); const first = document.pages[0]; if (!first) throw new Error('Fixture missing');
    first.title = 'Keep my unsaved title';
    const before = formatPresentationDocument(document);
    const expected = await adapter.importLegacyPresentationSnapshot(document, legacySource, { adapterVersion: 1, appKey: 'eligible', locale: 'fr' });
    const fetch = vi.spyOn(globalThis, 'fetch'); const storage = vi.spyOn(Storage.prototype, 'setItem');
    const { props, container, rerender } = mount({ document, documentText: before });
    selectTarget(); paste(legacySource); fireEvent.click(importButton());
    await screen.findByRole('region', { name: 'Local import receipt' });
    expect(props.onChange).toHaveBeenCalledOnce();
    expect(props.onChange).toHaveBeenCalledWith(formatPresentationDocument(expected.document));
    expect(formatPresentationDocument(document)).toBe(before);
    expect(expected.document.pages[0]?.blocks.slice(0, 2)).toEqual(document.pages[0]?.blocks);
    expect(expected.document.pages[0]?.blocks).toHaveLength(3);
    expect(screen.getByLabelText('Import receipt JSON')).toHaveTextContent(expected.receipt.sha256);
    expect(screen.getByLabelText('Import receipt JSON')).toHaveTextContent('/<script>unsafe()<~1script>');
    expect(container.querySelector('script,iframe')).toBeNull();
    expect(fetch).not.toHaveBeenCalled(); expect(storage).not.toHaveBeenCalled();
    // Applying editor.edit changes document identity; the last receipt remains
    // an explicitly local event record, not a claim about the saved revision.
    rerender(<LegacyImportPanel {...props} document={expected.document} documentText={formatPresentationDocument(expected.document)} />);
    expect(screen.getByRole('region', { name: 'Local import receipt' })).toHaveTextContent('Nothing was saved or published');
  });
  it.each(['document', 'invalid-document', 'route', 'busy', 'source', 'app', 'locale', 'unmount'] as const)('discards a successful async import after %s changes without overwriting newer edits', async changed => {
    const document = baseDocument();
    const result = await adapter.importLegacyPresentationSnapshot(document, legacySource, { adapterVersion: 1, appKey: 'eligible', locale: 'fr' });
    let finish: ((result: adapter.LegacyImportResult) => void) | undefined;
    const convert = vi.spyOn(adapter, 'importLegacyPresentationSnapshot').mockImplementation(() => new Promise(resolve => { finish = resolve; }));
    const view = mount({ document, documentText: formatPresentationDocument(document) });
    selectTarget(); paste(legacySource); fireEvent.click(importButton()); fireEvent.click(importButton());
    expect(convert).toHaveBeenCalledOnce(); expect(importButton()).toBeDisabled();
    if (changed === 'document') {
      const next = parsePresentationDocument(view.props.documentText); if (!next.pages[0]) throw new Error('Fixture missing'); next.pages[0].title = 'Concurrent dirty edit';
      view.rerender(<LegacyImportPanel {...view.props} document={next} documentText={formatPresentationDocument(next)} />);
    }
    if (changed === 'invalid-document') view.rerender(<LegacyImportPanel {...view.props} document={undefined} documentText="{new invalid edit" />);
    if (changed === 'route') view.rerender(<LegacyImportPanel {...view.props} contextKey="another-variant:/apps/other" />);
    if (changed === 'busy') view.rerender(<LegacyImportPanel {...view.props} disabled />);
    if (changed === 'source') paste('New source');
    if (changed === 'app') fireEvent.change(screen.getByRole('combobox', { name: 'Import target app' }), { target: { value: 'other' } });
    if (changed === 'locale') fireEvent.change(screen.getByRole('combobox', { name: 'Import target locale' }), { target: { value: '' } });
    if (changed === 'unmount') view.unmount();
    await act(async () => { finish?.(result); await Promise.resolve(); });
    expect(view.props.onChange).not.toHaveBeenCalled();
    expect(screen.queryByRole('region', { name: 'Local import receipt' })).toBeNull();
  });
  it('discards a pending file read when pasted input supersedes it', async () => {
    let finish: ((bytes: ArrayBuffer) => void) | undefined;
    const file = localFile('{}', 'old.json');
    file.arrayBuffer.mockImplementation(() => new Promise(resolve => { finish = resolve; }));
    const { props } = mount(); selectTarget();
    fireEvent.change(screen.getByLabelText('Load legacy JSON file (maximum 1 MiB)'), { target: { files: [file] } });
    paste('Newer pasted source');
    await act(async () => { finish?.(new TextEncoder().encode('Obsolete file').buffer); await Promise.resolve(); });
    expect(screen.getByRole('textbox')).toHaveValue('Newer pasted source'); expect(props.onChange).not.toHaveBeenCalled();
  });
  it('requires explicit eligible app and exact target-page locale rather than choosing the first app or bundle default', () => {
    const convert = vi.spyOn(adapter, 'importLegacyPresentationSnapshot'); mount(); paste();
    const app = screen.getByRole('combobox', { name: 'Import target app' });
    const locale = screen.getByRole('combobox', { name: 'Import target locale' });
    expect(app).toHaveValue(''); expect(locale).toBeDisabled(); expect(importButton()).toBeDisabled();
    expect([...app.querySelectorAll('option')].map(option => option.value)).toEqual(['', 'eligible', 'other']);
    fireEvent.change(app, { target: { value: 'eligible' } });
    expect(locale).toHaveValue(''); expect([...locale.querySelectorAll('option')].map(option => option.value)).toEqual(['', 'fr']);
    expect(importButton()).toBeDisabled();
    fireEvent.change(locale, { target: { value: 'fr' } }); expect(importButton()).toBeEnabled();
    fireEvent.change(app, { target: { value: 'other' } });
    expect(locale).toHaveValue(''); expect([...locale.querySelectorAll('option')].map(option => option.value)).toEqual(['', 'en']);
    expect(importButton()).toBeDisabled(); expect(convert).not.toHaveBeenCalled();
  });
  it('reports malformed adapter input inline without changing the current document', async () => {
    const convert = vi.spyOn(adapter, 'importLegacyPresentationSnapshot').mockRejectedValue(new Error('Malformed legacy section envelope'));
    const { props } = mount(); selectTarget(); paste('{bad json'); fireEvent.click(importButton());
    expect(await screen.findByRole('alert')).toHaveTextContent('Malformed legacy section envelope');
    expect(props.onChange).not.toHaveBeenCalled(); expect(props.documentText).toContain('Configured first page');
    expect(convert).toHaveBeenCalledWith(expect.anything(), '{bad json', { adapterVersion: 1, appKey: 'eligible', locale: 'fr' });
    expect(convert.mock.calls[0]?.[0]).not.toBe(props.document);
    expect(convert.mock.calls[0]?.[0]).toEqual(props.document);
  });
  it('surfaces real malformed JSON rejection without replacing the dirty document', async () => {
    const { props } = mount(); selectTarget(); paste('{"variant":{},"sections":'); fireEvent.click(importButton());
    expect(await screen.findByRole('alert')).toHaveTextContent('Legacy presentation import:');
    expect(props.onChange).not.toHaveBeenCalled(); expect(screen.queryByRole('region', { name: 'Local import receipt' })).toBeNull();
  });
  it('reports an ambiguous target-page locale and does not call the adapter', () => {
    const document = baseDocument(); const page = document.pages[0]; if (!page) throw new Error('Fixture missing'); document.pages.push(page);
    const convert = vi.spyOn(adapter, 'importLegacyPresentationSnapshot');
    mount({ document, documentText: formatPresentationDocument(document) }); selectTarget(); paste();
    expect(screen.getByRole('alert')).toHaveTextContent('exactly one configured target page');
    expect(importButton()).toBeDisabled(); expect(convert).not.toHaveBeenCalled();
  });
  it('keeps missing target pages explicit without guessing the bundle locale', () => {
    const document = baseDocument(); const app = document.apps.find(app => app.key === 'eligible'); if (!app) throw new Error('Fixture missing'); app.pageId = 'missing-page';
    mount({ document, documentText: formatPresentationDocument(document) });
    fireEvent.change(screen.getByRole('combobox', { name: 'Import target app' }), { target: { value: 'eligible' } }); paste();
    expect(screen.getByText('This app has no configured target page locale.')).toBeVisible(); expect(importButton()).toBeDisabled();
  });
  it('bounds pasted UTF-8 bytes, not JavaScript character count, before calling the adapter', () => {
    const convert = vi.spyOn(adapter, 'importLegacyPresentationSnapshot'); mount(); selectTarget(); paste('é'.repeat(524289));
    expect(screen.getByRole('alert')).toHaveTextContent('1 MiB UTF-8 import limit');
    expect(importButton()).toBeDisabled(); expect(convert).not.toHaveBeenCalled();
  });
  it('rejects an oversized file before reading its bytes and does not retain an older pasted source', () => {
    const file = localFile(new Uint8Array(1024 * 1024 + 1), 'too-large.json'); mount(); selectTarget(); paste('old source');
    fireEvent.change(screen.getByLabelText('Load legacy JSON file (maximum 1 MiB)'), { target: { files: [file] } });
    expect(screen.getByRole('alert')).toHaveTextContent('1 MiB'); expect(file.arrayBuffer.mock.calls).toHaveLength(0);
    expect(screen.getByRole('textbox')).toHaveValue(''); expect(importButton()).toBeDisabled();
  });
  it('loads a local UTF-8 file into the source editor without converting or mutating the document automatically', async () => {
    const convert = vi.spyOn(adapter, 'importLegacyPresentationSnapshot'); const { props } = mount();
    const source = '{"sections":[],"note":"Résumé"}';
    fireEvent.change(screen.getByLabelText('Load legacy JSON file (maximum 1 MiB)'), { target: { files: [localFile(source)] } });
    await waitFor(() => { expect(screen.getByRole('textbox')).toHaveValue(source); });
    expect(convert).not.toHaveBeenCalled(); expect(props.onChange).not.toHaveBeenCalled();
  });
  it('preserves a UTF-8 BOM and original line endings for adapter validation rather than silently normalizing file bytes', async () => {
    const convert = vi.spyOn(adapter, 'importLegacyPresentationSnapshot').mockRejectedValue(new Error('Envelope not accepted'));
    mount(); selectTarget();
    const source = '\uFEFF{\r\n"variant":{},"sections":[]\r\n}';
    fireEvent.change(screen.getByLabelText('Load legacy JSON file (maximum 1 MiB)'), { target: { files: [localFile(source)] } });
    await waitFor(() => { expect(importButton()).toBeEnabled(); });
    fireEvent.click(importButton());
    await screen.findByRole('alert');
    expect(convert).toHaveBeenCalledWith(expect.anything(), source, { adapterVersion: 1, appKey: 'eligible', locale: 'fr' });
  });
  it('ignores a read failure after cancellation instead of replacing the cancellation feedback', async () => {
    let reject: ((error: Error) => void) | undefined;
    const file = localFile('{}'); file.arrayBuffer.mockImplementation(() => new Promise((_resolve, fail) => { reject = fail; }));
    const { props } = mount();
    fireEvent.change(screen.getByLabelText('Load legacy JSON file (maximum 1 MiB)'), { target: { files: [file] } });
    fireEvent.click(screen.getByRole('button', { name: 'Cancel import' }));
    await act(async () => { reject?.(new Error('Late read failure')); await Promise.resolve(); });
    expect(screen.getByRole('alert')).toHaveTextContent('Import cancelled'); expect(props.onChange).not.toHaveBeenCalled();
  });
  it('reports unreadable and invalid UTF-8 files without applying replacement text', async () => {
    const { props } = mount();
    fireEvent.change(screen.getByLabelText('Load legacy JSON file (maximum 1 MiB)'), { target: { files: [localFile(new Uint8Array([0xff, 0xfe]), 'invalid.json')] } });
    expect(await screen.findByRole('alert')).toHaveTextContent('valid UTF-8 JSON');
    expect(screen.getByRole('textbox')).toHaveValue(''); expect(props.onChange).not.toHaveBeenCalled();
    const unreadable = localFile('{}', 'unreadable.json'); unreadable.arrayBuffer.mockRejectedValue(new Error('Read failed'));
    fireEvent.change(screen.getByLabelText('Load legacy JSON file (maximum 1 MiB)'), { target: { files: [unreadable] } });
    expect(await screen.findByRole('alert')).toHaveTextContent('could not be read'); expect(props.onChange).not.toHaveBeenCalled();
  });
  it.each(['disabled', 'invalid-document'] as const)('does not allow an import in %s state', state => {
    const convert = vi.spyOn(adapter, 'importLegacyPresentationSnapshot');
    mount(state === 'disabled' ? { disabled: true } : { document: undefined, documentText: '{invalid' });
    expect(screen.getByRole('combobox', { name: 'Import target app' })).toBeDisabled();
    expect(screen.getByRole('textbox')).toBeDisabled(); expect(importButton()).toBeDisabled(); expect(convert).not.toHaveBeenCalled();
  });
  it('keeps cancelled adapter failures from replacing current input feedback', async () => {
    let reject: ((error: Error) => void) | undefined;
    vi.spyOn(adapter, 'importLegacyPresentationSnapshot').mockImplementation(() => new Promise((_resolve, fail) => { reject = fail; }));
    const { props } = mount(); selectTarget(); paste(); fireEvent.click(importButton());
    expect(importButton()).toBeDisabled(); fireEvent.click(screen.getByRole('button', { name: 'Cancel import' }));
    await act(async () => { reject?.(new Error('Stale failure')); await Promise.resolve(); });
    expect(screen.getByRole('alert')).toHaveTextContent('Import cancelled'); expect(screen.queryByText('Stale failure')).toBeNull();
    expect(props.onChange).not.toHaveBeenCalled();
  });
});
