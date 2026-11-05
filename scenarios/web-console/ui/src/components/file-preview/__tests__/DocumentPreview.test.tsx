import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { renderWithProviders as render } from '../../../test-utils';
import { previewModel, previewProps } from '../../../test-utils/filePreview';
import { strings } from '../../../consts/strings';
import { selectors } from '../../../consts/selectors';
import { useDocument } from '../useDocument';
import { copyText } from '../../../lib/clipboard';
import { DocumentPreview } from '../renderers/DocumentPreview';
vi.mock('../useDocument', () => ({ useDocument: vi.fn() }));
vi.mock('../../../lib/clipboard', () => ({ copyText: vi.fn().mockResolvedValue(true) }));
const geometry = vi.hoisted(() => ({ width: 390 }));
vi.mock('@vrooli/react-component-library/useResizeObserver', () => ({ useResizeObserver: () => ({ ref: () => {}, rect: { width: geometry.width } }) }));
const c = strings.documentPreview, reading = strings.readerPreview;
const html = '<h1>Overview</h1><p>Read <strong>this phrase</strong> and this phrase.</p><h2>Milestones</h2><table><tr><td>Complete</td></tr></table><p id="docx-note">A note</p><a href="#docx-note">See note</a>';
function props() { return previewProps({ model: previewModel({ kind: 'document', basename: 'brief.docx', resolvedPath: '/brief.docx' }) }); }
beforeEach(() => { geometry.width = 390; vi.mocked(useDocument).mockReturnValue({ data: { html, warnings: false, omittedImages: 0 } }); Object.defineProperty(HTMLElement.prototype, 'scrollIntoView', { configurable: true, value: vi.fn() }); });
describe('Word document reader', () => {
  it('renders semantic content and uses a sequential outline on phones with heading focus on return', async () => {
    render(<DocumentPreview {...props()} />); expect(screen.getByRole('heading', { name: 'Overview' })).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: reading.contents })); const outline = screen.getByRole('navigation', { name: reading.contents });
    await waitFor(() => { expect(within(outline).getAllByRole('button')).toHaveLength(2); });
    expect(screen.getByTestId(selectors.documentPreview.pane)).toHaveClass('hidden');
    fireEvent.click(within(outline).getByRole('button', { name: 'Milestones' }));
    await waitFor(() => { expect(screen.getByRole('heading', { name: 'Milestones' })).toHaveFocus(); }); expect(screen.queryByRole('navigation')).not.toBeInTheDocument();
  });
  it('keeps the outline alongside the document in wide drawers', async () => {
    geometry.width = 1000; render(<DocumentPreview {...props()} />); fireEvent.click(screen.getByRole('button', { name: reading.contents }));
    const outline = screen.getByRole('navigation', { name: reading.contents }); await within(outline).findByRole('button', { name: 'Milestones' });
    fireEvent.click(within(outline).getByRole('button', { name: 'Milestones' })); expect(screen.getByTestId(selectors.documentPreview.pane)).not.toHaveClass('hidden'); expect(outline).toBeInTheDocument();
  });
  it('searches across inline text, treats patterns literally and consumes Escape before drawer dismissal', async () => {
    render(<DocumentPreview {...props()} />); fireEvent.click(screen.getByRole('button', { name: reading.findDocument }));
    const input = screen.getByRole('searchbox', { name: reading.findDocument }); expect(input).toHaveFocus(); fireEvent.change(input, { target: { value: 'Read this phrase' } });
    await waitFor(() => { expect(screen.getByRole('button', { name: strings.artifactPreview.nextMatch })).toBeEnabled(); });
    fireEvent.keyDown(input, { key: 'Enter' }); fireEvent.keyDown(input, { key: 'Enter', shiftKey: true });
    fireEvent.change(input, { target: { value: '.*' } }); await waitFor(() => { expect(screen.getByRole('button', { name: strings.artifactPreview.nextMatch })).toBeDisabled(); });
    fireEvent.keyDown(input, { key: 'Escape' }); expect(input).not.toBeInTheDocument(); expect(screen.getByRole('button', { name: reading.findDocument })).toHaveFocus();
  });
  it('changes text size in shared settings, copies only readable text and resolves notes inside the reader', async () => {
    render(<DocumentPreview {...props()} />); fireEvent.click(screen.getByRole('button', { name: c.options })); fireEvent.click(screen.getByRole('radio', { name: '20 px' }));
    expect(screen.getByTestId(selectors.documentPreview.article).parentElement).toHaveStyle({ fontSize: '20px' }); fireEvent.click(screen.getByTestId('inspection-options.close'));
    await waitFor(() => { expect(screen.queryByRole('dialog')).not.toBeInTheDocument(); });
    fireEvent.click(screen.getByRole('link', { name: 'See note' })); expect(screen.getByText('A note')).toHaveFocus();
    fireEvent.click(screen.getByText(strings.artifactPreview.actions)); fireEvent.click(screen.getByRole('button', { name: c.copyText })); await waitFor(() => { expect(copyText).toHaveBeenCalledWith(expect.stringContaining('Read this phrase')); }); expect(vi.mocked(copyText).mock.calls.at(-1)?.[0]).not.toContain('<strong>');
  });
  it('shows recovery/download for errors and resets reader options between preview identities', () => {
    vi.mocked(useDocument).mockReturnValue({ error: 'encrypted' }); const { rerender } = render(<DocumentPreview {...props()} />); expect(screen.getByText(c.encrypted)).toBeInTheDocument(); expect(screen.getByTestId('file-preview-download')).toHaveAttribute('download', 'brief.docx');
    fireEvent.click(screen.getByRole('button', { name: strings.messagesFileViewer.reopen })); expect(useDocument).toHaveBeenLastCalledWith('/preview/blob', 1024, 1);
    vi.mocked(useDocument).mockReturnValue({ data: { html: '', warnings: true, omittedImages: 0 } }); rerender(<DocumentPreview {...props()} model={{ ...props().model, previewId: 'new' }} />);
    expect(screen.getByText(strings.messagesFileViewer.emptyFile)).toBeInTheDocument(); expect(screen.getByText(c.warnings)).toBeInTheDocument(); expect(useDocument).toHaveBeenLastCalledWith('/preview/blob', 1024, 0);
  });
  it('renews the target capability when retrying a failed or stale blob fetch', () => {
    vi.mocked(useDocument).mockReturnValue({ error: 'fetch' }); const onReopen = vi.fn();
    render(<DocumentPreview {...props()} onReopen={onReopen} />); fireEvent.click(screen.getByRole('button', { name: strings.messagesFileViewer.reopen }));
    expect(onReopen).toHaveBeenCalledOnce();
  });

});
