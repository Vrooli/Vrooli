import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { renderWithProviders as render } from '../../../test-utils';
import copy from '../../../i18n/locales/en.json';
import { i18n } from '../../../i18n';
import { copyText } from '../../../lib/clipboard';
import { highlightCode } from '../../../lib/codeHighlighter';
import { CodeLinePreview } from '../renderers/CodeLinePreview';
vi.mock('../../../lib/codeHighlighter', () => ({ getLanguageFromPath: () => 'typescript', highlightCode: vi.fn().mockResolvedValue([{ lineNumber: 1, tokens: [{ content: 'const ' }, { content: 'sum = 1', color: '#c9d1d9' }] }]) }));
vi.mock('../../../lib/clipboard', () => ({ copyText: vi.fn().mockResolvedValue(true) }));
const c = copy.inspectionPreview;
beforeEach(async () => { vi.clearAllMocks(); await i18n.changeLanguage('en'); });
describe('source reader', () => {
  it('finds literal text across token boundaries and navigates repeated results', async () => {
    render(<CodeLinePreview content={'const sum = 1\nconst sum = 2'} path='/code.ts' highlightLine={null} />);
    await act(async () => { await Promise.resolve(); });
    fireEvent.click(screen.getByRole('button', { name: c.search })); const search = screen.getByRole('searchbox', { name: c.search }); fireEvent.change(search, { target: { value: 'const sum' } });
    expect(screen.getAllByRole('status')[0]).toHaveTextContent('1 of 2 matches'); expect(document.querySelectorAll('mark')).toHaveLength(3);
    fireEvent.click(screen.getByRole('button', { name: c.nextMatch })); expect(screen.getAllByRole('status')[0]).toHaveTextContent('2 of 2 matches');
    fireEvent.keyDown(search, { key: 'Enter' }); expect(screen.getAllByRole('status')[0]).toHaveTextContent('1 of 2 matches');
    fireEvent.keyDown(search, { key: 'Escape' }); expect(screen.queryByRole('searchbox')).not.toBeInTheDocument(); expect(screen.getByRole('button', { name: c.search })).toHaveFocus();
  });
  it('copies an exact Shift-selected range without copying line numbers', async () => {
    render(<CodeLinePreview content={'first\r\nsecond\r\nthird'} path='/file.log' highlightLine={null} />);
    fireEvent.click(screen.getByRole('button', { name: /Select line 1;/ })); fireEvent.click(screen.getByRole('button', { name: /Select line 2;/ }), { shiftKey: true });
    await act(async () => { fireEvent.click(screen.getByRole('button', { name: c.copySelection })); await Promise.resolve(); }); expect(copyText).toHaveBeenCalledWith('first\r\nsecond');
  });
  it('filters log labels while retaining original numbers and allows navigation back to all lines', async () => {
    render(<CodeLinePreview content={'[INFO] ready\n[ERROR] failed\n[WARN] check'} path='/server.log' highlightLine={null} />);
    fireEvent.click(screen.getByRole('button', { name: c.options })); fireEvent.change(screen.getByRole('combobox', { name: c.severity }), { target: { value: 'error' } });
    expect(screen.queryByText('[INFO] ready')).not.toBeInTheDocument(); expect(screen.getByText('[ERROR] failed')).toBeInTheDocument(); expect(document.querySelector('[data-source-line="2"]')).toBeInTheDocument();
    fireEvent.change(screen.getByRole('textbox', { name: c.goLine }), { target: { value: '1' } }); fireEvent.click(screen.getByRole('button', { name: c.go })); await waitFor(() => { expect(screen.queryByRole('dialog')).not.toBeInTheDocument(); }); expect(screen.getByText('[INFO] ready')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: c.options })); fireEvent.change(screen.getByRole('textbox', { name: c.goLine }), { target: { value: '999' } }); fireEvent.click(screen.getByRole('button', { name: c.go })); expect(screen.getByRole('alert')).toHaveTextContent(c.lineInvalid);
  });
  it('bounds large source rendering/highlighting and opens linked lines beyond the first page', () => {
    render(<CodeLinePreview content={Array.from({ length: 1500 }, (_, index) => `line ${String(index + 1)}`).join('\n')} path='/file.log' highlightLine={1000} truncated />);
    expect(document.querySelectorAll('[data-source-line]').length).toBeLessThanOrEqual(400); expect(screen.getByText('line 1000')).toBeInTheDocument(); expect(highlightCode).not.toHaveBeenCalled(); expect(screen.getByText(copy.messagesFileViewer.truncatedNotice)).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: copy.artifactPreview.nextPage })); expect(screen.getByText('line 1201')).toBeInTheDocument();
  });
  it('supports font bounds, wrapping, backward search and dismissible reading options', async () => {
    render(<CodeLinePreview content={'const sum = 1\nconst sum = 2'} path='/code.ts' highlightLine={null} />);
    await act(async () => { await Promise.resolve(); });
    fireEvent.click(screen.getByRole('button', { name: c.options }));
    const increase = screen.getByRole('button', { name: copy.messagesFileViewer.increaseFontSize });
    for (let i = 0; i < 9; i++) fireEvent.click(increase);
    expect(increase).toBeDisabled(); expect(screen.getByText('18px')).toBeInTheDocument();
    const decrease = screen.getByRole('button', { name: copy.messagesFileViewer.decreaseFontSize });
    for (let i = 0; i < 9; i++) fireEvent.click(decrease);
    expect(decrease).toBeDisabled(); expect(screen.getByText('11px')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: copy.messagesFileViewer.enableWordWrap }));
    expect(screen.getByRole('button', { name: copy.messagesFileViewer.disableWordWrap })).toHaveAttribute('aria-pressed', 'true');
    fireEvent.keyDown(screen.getByRole('dialog'), { key: 'Escape' }); await waitFor(() => { expect(screen.queryByRole('dialog')).not.toBeInTheDocument(); }); expect(screen.getByRole('button', { name: c.options })).toHaveFocus();
    fireEvent.click(screen.getByRole('button', { name: c.search })); fireEvent.change(screen.getByRole('searchbox'), { target: { value: 'sum' } }); fireEvent.click(screen.getByRole('button', { name: c.previousMatch })); expect(screen.getAllByRole('status')[0]).toHaveTextContent('2 of 2 matches');
  });

  it('uses one gutter tab stop and supports keyboard range selection across lines', async () => {
    render(<CodeLinePreview content={'first\nsecond\nthird'} path='/file.log' highlightLine={null} />);
    const lines = Array.from(document.querySelectorAll<HTMLButtonElement>('[data-source-line] > button')); const first = lines[0]; if (!first) throw Error('Missing line');
    expect(lines.filter((line) => line.tabIndex === 0)).toHaveLength(1); act(() => { first.focus(); }); fireEvent.keyDown(first, { key: 'ArrowDown', shiftKey: true });
    await act(async () => { await new Promise((resolve) => requestAnimationFrame(resolve)); });
    expect(lines[1]).toHaveFocus(); await act(async () => { fireEvent.click(screen.getByRole('button', { name: c.copySelection })); await Promise.resolve(); }); expect(copyText).toHaveBeenLastCalledWith('first\nsecond');
    fireEvent.keyDown(lines[1] ?? first, { key: 'End' }); await act(async () => { await new Promise((resolve) => requestAnimationFrame(resolve)); }); expect(lines[2]).toHaveFocus();
    fireEvent.keyDown(lines[2] ?? first, { key: 'Home' }); await act(async () => { await new Promise((resolve) => requestAnimationFrame(resolve)); }); expect(first).toHaveFocus();
  });

  it('opens text files wrapped and preserves exact text when selecting and copying', async () => {
    const content = 'Release package\n\nBrowse  Documents and Source without losing whitespace.';
    render(<CodeLinePreview content={content} path='/README.txt' highlightLine={null} plainText />);
    expect(highlightCode).not.toHaveBeenCalled();
    expect(screen.queryByRole('status')).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: c.options }));
    expect(screen.getByRole('button', { name: copy.messagesFileViewer.disableWordWrap })).toHaveAttribute('aria-pressed', 'true');
    expect(screen.getByText('14px')).toBeInTheDocument();
    fireEvent.keyDown(screen.getByRole('dialog'), { key: 'Escape' });
    await waitFor(() => { expect(screen.queryByRole('dialog')).not.toBeInTheDocument(); });
    fireEvent.click(screen.getByRole('button', { name: /Select line 1;/ }));
    fireEvent.click(screen.getByRole('button', { name: /Select line 3;/ }), { shiftKey: true });
    await act(async () => { fireEvent.click(screen.getByRole('button', { name: c.copySelection })); await Promise.resolve(); });
    expect(copyText).toHaveBeenLastCalledWith(content);
  });

});
