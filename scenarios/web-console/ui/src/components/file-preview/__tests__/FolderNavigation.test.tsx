import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { renderWithProviders as render } from '../../../test-utils';
import { previewModel } from '../../../test-utils/filePreview';
import { resolveFilePreview, listDirectory, getFilePreviewText } from '../../../api/filePreview';
import { useFilePreviewController } from '../useFilePreviewController';
import { strings } from '../../../consts/strings';
import MessagesFileViewer from '../../MessagesFileViewer';
vi.mock('../../../api/filePreview', async original => ({ ...await original<typeof import('../../../api/filePreview')>(), resolveFilePreview: vi.fn(), listDirectory: vi.fn(), getFilePreviewText: vi.fn() }));
function Harness({ close }: { close: () => void }) {
  const c = useFilePreviewController('session');
  return <><button onClick={() => { void c.openPreview('/root'); }}>Open folder</button><button onClick={() => { void c.openPreview('/root/child.txt'); }}>Open file</button><MessagesFileViewer state={c.state} onClose={close} onReopen={c.reopen} onRendererError={c.reportError} onNavigate={c.navigateTo} onNavigateBack={c.navigateBack} onLoadMore={c.loadMore} onListOptionsChange={c.setListOptions} /></>;
}
function setup() {
  vi.mocked(resolveFilePreview).mockImplementation((_session, path) => Promise.resolve(previewModel({ previewId: path, kind: path.endsWith('.txt') ? 'text' : 'directory', resolvedPath: path, basename: path.split('/').at(-1) ?? '', listingAvailable: !path.endsWith('.txt'), textContentAvailable: path.endsWith('.txt') })));
  vi.mocked(listDirectory).mockImplementation((_session, path) => Promise.resolve({ resolvedPath: path, parentPath: '/', entries: [{ name: 'child.txt', kind: 'text', entryType: 'file', canPreview: true, sizeBytes: 10, mtimeMs: 0, symlinkTarget: '', symlinkBroken: false, mode: '', childCount: null }], totalEntries: 1, truncated: false, nextPageToken: '', effectiveSort: 'dirs_first_name', sort: 'dirs_first_name', showHidden: false, warnings: [] }));
  vi.mocked(getFilePreviewText).mockResolvedValue({ resolvedPath: '/root/child.txt', kind: 'text', mimeType: 'text/plain', content: 'Hello', truncated: false });
  const close = vi.fn(); render(<Harness close={close} />); return close;
}
beforeEach(() => { Object.defineProperty(HTMLElement.prototype, 'scrollIntoView', { configurable: true, value: vi.fn() }); });
describe('folder navigation within the drawer', () => {
  it('keeps one stable Back target when returning from a file to the root, without dismissing', async () => {
    const close = setup(); fireEvent.click(screen.getByText('Open folder'));
    await screen.findByTestId('file-preview-directory');
    const back = screen.getByTestId('file-preview-back'); expect(back).toBeDisabled();
    fireEvent.click(screen.getByTestId('directory-entry')); await screen.findByTestId('file-preview-code');
    expect(screen.getByTestId('file-preview-back')).toBe(back); expect(back).toBeEnabled();
    fireEvent.click(back); await screen.findByTestId('file-preview-directory');
    expect(screen.getByTestId('file-preview-back')).toBe(back); expect(back).toBeDisabled();
    await waitFor(() => { expect(screen.getByTestId('file-preview-navigation')).toHaveFocus(); });
    fireEvent.click(back); expect(close).not.toHaveBeenCalled();
    expect(screen.getByTestId('messages-file-viewer-panel')).toHaveAttribute('data-state', 'open');
  });
  it('opens the containing folder from a file opened directly, and Back returns to the file', async () => {
    const close = setup(); fireEvent.click(screen.getByText('Open file'));
    await screen.findByTestId('file-preview-code');
    // No navigation history yet, but the file still offers a way into its folder.
    expect(screen.queryByTestId('file-preview-navigation')).not.toBeInTheDocument();
    fireEvent.click(screen.getByTestId('file-preview-open-folder'));
    await screen.findByTestId('file-preview-directory');
    expect(vi.mocked(listDirectory)).toHaveBeenCalledWith('session', '/root', expect.anything());
    const back = screen.getByTestId('file-preview-back'); expect(back).toBeEnabled();
    fireEvent.click(back); await screen.findByTestId('file-preview-code');
    expect(close).not.toHaveBeenCalled();
  });
  it('returns from a failed child load and ignores the dismissed request response', async () => {
    const close = setup(); fireEvent.click(screen.getByText('Open folder')); await screen.findByTestId('file-preview-directory');
    let complete: ((value: ReturnType<typeof previewModel>) => void) | undefined;
    vi.mocked(resolveFilePreview).mockImplementationOnce(() => new Promise(resolve => { complete = resolve; }));
    fireEvent.click(screen.getByTestId('directory-entry')); fireEvent.click(screen.getByTestId('file-preview-back'));
    await screen.findByTestId('file-preview-directory');
    await act(async () => { complete?.(previewModel({ kind: 'image' })); await Promise.resolve(); });
    expect(screen.getByTestId('file-preview-directory')).toBeInTheDocument(); expect(close).not.toHaveBeenCalled();
    vi.mocked(resolveFilePreview).mockRejectedValueOnce(new Error('Missing file'));
    fireEvent.click(screen.getByTestId('directory-entry')); await screen.findByText('Missing file');
    fireEvent.click(screen.getByTestId('file-preview-back')); await screen.findByTestId('file-preview-directory'); expect(close).not.toHaveBeenCalled();
  });
  it('keeps folder options mounted during sort refresh and preserves the drawer on Escape', async () => {
    const close = setup(); fireEvent.click(screen.getByText('Open folder')); await screen.findByTestId('file-preview-directory');
    fireEvent.click(screen.getByRole('button', { name: strings.documentPreview.folderOptions }));
    const dialog = screen.getByTestId('inspection-options');
    const original = await vi.mocked(listDirectory)('session', '/root', { sort: 'dirs_first_name', showHidden: false });
    let finish: ((value: typeof original) => void) | undefined;
    vi.mocked(listDirectory).mockImplementationOnce(() => new Promise(resolve => { finish = resolve; }));
    const sort = screen.getByTestId('directory-sort'); sort.focus();
    fireEvent.change(sort, { target: { value: 'name' } });
    expect(sort).toHaveFocus();
    expect(dialog).toBeInTheDocument(); expect(screen.getByTestId('directory-toggle-hidden')).toHaveAttribute('aria-disabled', 'true');
    await act(async () => { finish?.({ ...original, sort: 'name', effectiveSort: 'name' }); await Promise.resolve(); });
    expect(screen.getByTestId('inspection-options')).toBe(dialog); expect(screen.getByTestId('directory-sort')).toHaveValue('name');
    fireEvent.keyDown(dialog, { key: 'Escape' }); await waitFor(() => { expect(dialog).not.toBeInTheDocument(); });
    expect(close).not.toHaveBeenCalled(); expect(screen.getByTestId('file-preview-directory')).toBeInTheDocument();
  });

});
