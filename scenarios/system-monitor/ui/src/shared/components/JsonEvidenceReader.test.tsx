import { fireEvent, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { renderWithProviders as render } from '../../test-utils/renderWithProviders';
import { formatJsonEvidence, JsonEvidenceReader } from './JsonEvidenceReader';

const editorMock = vi.hoisted(() => ({ props: {}, find: vi.fn(), configure: vi.fn() }));
vi.mock('monaco-editor', () => ({}));
vi.mock('monaco-editor/editor/editor.worker.js?worker', () => ({ default: vi.fn() }));
vi.mock('monaco-editor/language/json/json.worker.js?worker', () => ({ default: vi.fn() }));
vi.mock('@monaco-editor/react', () => ({
  loader: { config: editorMock.configure },
  default: (props: Record<string, unknown>) => {
    editorMock.props = props;
    (props.onMount as (editor: unknown) => void)({ getAction: () => ({ run: editorMock.find }) });
    return <div data-testid="monaco" />;
  },
}));

describe('JsonEvidenceReader', () => {
  it('passes the complete record to a read-only editor and exposes find and wrap', () => {
    const close = vi.fn();
    const value = '{"MESSAGE":"complete","unmapped":"retained"}';
    render(<JsonEvidenceReader title="Journal metadata" value={value} onClose={close} />);
    expect(editorMock.props.value).toBe(formatJsonEvidence(value));
    expect(editorMock.props.options).toMatchObject({ readOnly: true, domReadOnly: true, wordWrap: 'on' });
    fireEvent.click(screen.getByRole('button', { name: 'Find' }));
    expect(editorMock.find).toHaveBeenCalled();
    fireEvent.click(screen.getByRole('button', { name: 'Wrap' }));
    expect(editorMock.props.options).toMatchObject({ wordWrap: 'off' });
    fireEvent.click(screen.getByRole('button', { name: /Back/ }));
    expect(close).toHaveBeenCalled();
  });

  it('changes reader text size and restores More focus on Escape', () => {
    render(<JsonEvidenceReader title="Journal metadata" value="{}" onClose={vi.fn()} />);
    const more = screen.getByRole('button', { name: 'More document tools' });
    fireEvent.click(more);
    expect(more).toHaveAttribute('aria-expanded', 'true');
    more.focus();
    fireEvent.keyDown(more, { key: 'Escape' });
    expect(more).toHaveAttribute('aria-expanded', 'false');
    expect(more).toHaveFocus();
    fireEvent.click(more);
    expect(screen.getByRole('button', { name: 'Find in document' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Copy all metadata' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Wrap lines' })).toHaveAttribute('aria-pressed', 'true');
    const increase = screen.getByRole('button', { name: 'Increase text size' });
    fireEvent.click(increase);
    expect(editorMock.props.options).toMatchObject({ fontSize: 15 });
    increase.focus();
    fireEvent.keyDown(increase, { key: 'Escape' });
    expect(more).toHaveAttribute('aria-expanded', 'false');
    expect(more).toHaveFocus();
  });

  it('formats whitespace without changing strings, large numbers or duplicate keys', () => {
    const raw = String.raw`{"x":"a, {\"b\"}","n":9007199254740993,"x":"again"}`;
    const formatted = formatJsonEvidence(raw);
    expect(formatted).toContain(String.raw`"a, {\"b\"}"`);
    expect(formatted).toContain('9007199254740993');
    expect(formatted.match(/"x"/g)).toHaveLength(2);
    expect(formatJsonEvidence('not json')).toBe('not json');
  });

  it('preserves a large message and metadata while formatting the complete document', () => {
    const message = 'm'.repeat(4 * 1024 * 1024 + 1) + ' 🛰️ "quoted" \\ end';
    const metadata = 'e'.repeat(5000);
    const raw = JSON.stringify({ MESSAGE: message, CUSTOM_FIELD: metadata });
    expect(JSON.parse(formatJsonEvidence(raw))).toEqual({ MESSAGE: message, CUSTOM_FIELD: metadata });
  });

  it('provides complete, searchable text parts for a large document', () => {
    const raw = JSON.stringify({ MESSAGE: 'x'.repeat(4083) + '🛰️marker' + 'y'.repeat(130000) });
    render(<JsonEvidenceReader title="Journal metadata" value={raw} onClose={vi.fn()} />);
    expect(editorMock.props.options).toMatchObject({ accessibilitySupport: 'auto' });
    fireEvent.click(screen.getByRole('button', { name: 'Read text in parts' }));
    let reconstructed = '';
    for (;;) {
      const text = screen.getByLabelText(/^Document text, part/);
      if (!(text instanceof HTMLTextAreaElement)) throw new Error('Expected a text area');
      reconstructed += text.value;
      const next = screen.getByRole('button', { name: 'Next part' });
      if (next.hasAttribute('disabled')) break;
      fireEvent.click(next);
    }
    expect(reconstructed).toBe(raw);
    fireEvent.change(screen.getByLabelText('Find in complete document'), { target: { value: '🛰️marker' } });
    fireEvent.click(screen.getByRole('button', { name: 'Find next' }));
    expect(screen.getByText(/Match begins in part/)).toBeInTheDocument();
    const found = screen.getByLabelText(/^Document text, part/);
    if (!(found instanceof HTMLTextAreaElement)) throw new Error('Expected a text area');
    expect(found.value).toContain('🛰️marker');
    fireEvent.click(screen.getByRole('button', { name: 'Return to editor' }));
    expect(screen.getByTestId('monaco')).toBeInTheDocument();
  });

  it('offers complete text reading and search for a small document', () => {
    const raw = '{"MESSAGE":"short marker","CUSTOM":"metadata"}';
    render(<JsonEvidenceReader title="Journal metadata" value={raw} onClose={vi.fn()} />);
    fireEvent.click(screen.getByRole('button', { name: 'Read text in parts' }));
    const text = screen.getByLabelText('Document text, part 1 of 1');
    expect(text).toHaveValue(raw);
    fireEvent.change(screen.getByLabelText('Find in complete document'), { target: { value: 'short marker' } });
    fireEvent.click(screen.getByRole('button', { name: 'Find next' }));
    if (!(text instanceof HTMLTextAreaElement)) throw new Error('Expected a text area');
    expect(text.value.slice(text.selectionStart, text.selectionEnd)).toBe('short marker');
    fireEvent.click(screen.getByRole('button', { name: 'Return to editor' }));
    expect(screen.getByTestId('monaco')).toBeInTheDocument();
  });

  it('locates a search match that spans text parts', () => {
    const raw = 'x'.repeat(4093) + 'CROSSBOUND' + 'y'.repeat(132000);
    render(<JsonEvidenceReader title="Journal metadata" value={raw} onClose={vi.fn()} />);
    fireEvent.click(screen.getByRole('button', { name: 'Read text in parts' }));
    fireEvent.change(screen.getByLabelText('Find in complete document'), { target: { value: 'CROSSBOUND' } });
    fireEvent.click(screen.getByRole('button', { name: 'Find next' }));
    expect(screen.getByText('Match begins in part 1 and continues in later parts.')).toBeInTheDocument();
    const text = screen.getByLabelText(/^Document text, part 1 of/);
    if (!(text instanceof HTMLTextAreaElement)) throw new Error('Expected a text area');
    expect(text).toHaveFocus();
    expect(text.value.slice(text.selectionStart, text.selectionEnd)).toBe('CRO');
    fireEvent.click(screen.getByRole('button', { name: 'Next part' }));
    const next = screen.getByLabelText(/^Document text, part 2 of/);
    if (!(next instanceof HTMLTextAreaElement)) throw new Error('Expected a text area');
    expect(next.value.startsWith('SSBOUND')).toBe(true);
  });

  it('resets text navigation when the document changes', () => {
    const first = JSON.stringify({ message: 'a'.repeat(150000) });
    const second = JSON.stringify({ message: 'b'.repeat(132000) });
    const view = render(<JsonEvidenceReader title="Journal metadata" value={first} onClose={vi.fn()} />);
    fireEvent.click(screen.getByRole('button', { name: 'Read text in parts' }));
    fireEvent.click(screen.getByRole('button', { name: 'Next part' }));
    view.rerender(<JsonEvidenceReader title="Journal metadata" value={second} onClose={vi.fn()} />);
    expect(screen.getByRole('button', { name: 'Read text in parts' })).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Read text in parts' }));
    const text = screen.getByLabelText(/^Document text, part 1 of/);
    if (!(text instanceof HTMLTextAreaElement)) throw new Error('Expected a text area');
    expect(text.value).toContain('b');
  });

  it('copies the original record rather than the formatted display', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, 'clipboard', { configurable: true, value: { writeText } });
    const raw = '{"n":9007199254740993,"n":1}';
    render(<JsonEvidenceReader title="Journal metadata" value={raw} onClose={vi.fn()} />);
    fireEvent.click(screen.getByRole('button', { name: 'Copy metadata' }));
    expect(await screen.findByText('Copied metadata')).toBeInTheDocument();
    expect(writeText).toHaveBeenCalledWith(raw);
  });
});
