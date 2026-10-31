import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { renderWithProviders as render } from '../../../test-utils';
import { previewModel, previewProps } from '../../../test-utils/filePreview';
import { strings } from '../../../consts/strings';
import { copyText } from '../../../lib/clipboard';
import { DiffPreview } from '../renderers/DiffPreview';
import postcss from 'postcss';
import tailwindcss from 'tailwindcss';
import tailwindConfig from '../../../../tailwind.config';
vi.mock('@vrooli/react-component-library/useResizeObserver', () => ({ useResizeObserver: () => ({ ref: () => {}, rect: { width: 1000 } }) }));
vi.mock('../../../lib/clipboard', () => ({ copyText: vi.fn().mockResolvedValue(true) }));
const c = strings.inspectionPreview, content = '--- a/one.ts\n+++ b/one.ts\n@@ -1 +1 @@\n-old\n+new\n@@ -8 +8 @@\n-before\n+after\n--- /dev/null\n+++ b/two.ts\n@@ -0,0 +1 @@\n+created\n';
function props(value = content) { return previewProps({ model: previewModel({ kind: 'diff', resolvedPath: '/review.patch' }), text: { resolvedPath: '/review.patch', kind: 'diff', mimeType: 'text/x-diff', content: value, truncated: false } }); }
describe('change reader', () => {
  it('generates actual foreground and background rules for both removed and added content', async () => {
    render(<DiffPreview {...props()} />);
    const result = await postcss([tailwindcss({ ...tailwindConfig, content: [{ raw: document.body.innerHTML, extension: 'html' }] })]).process('@tailwind utilities;', { from: undefined });
    const painted = (text: string) => {
      const line = screen.getByText(text).parentElement;
      if (!line) throw Error('Missing diff row');
      const declarations: Record<string, string> = {};
      result.root.walkRules((rule) => {
        if (!line.matches(rule.selector)) return;
        rule.walkDecls((decl) => { declarations[decl.prop] = decl.value; });
      });
      return declarations;
    };
    const removed = painted('-old'), added = painted('+new');
    expect(removed['background-color']).toBeTruthy();
    expect(removed.color).toBeTruthy();
    expect(added['background-color']).toBeTruthy();
    expect(added.color).toBeTruthy();
    expect(removed['background-color']).not.toBe(added['background-color']);
    expect(removed.color).not.toBe(added.color);
  });

  it('navigates hunks/files with old/new versions and unified/source modes', async () => {
    render(<DiffPreview {...props()} />); expect(screen.getAllByText(c.oldVersion)[0]).toBeInTheDocument(); expect(screen.getAllByText(c.newVersion)[0]).toBeInTheDocument(); expect(screen.getByText('-old')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: c.nextChange })); expect(screen.getByText('+after')).toBeInTheDocument(); fireEvent.click(screen.getByRole('button', { name: c.nextChange })); expect(screen.getByText('+created')).toBeInTheDocument(); expect(screen.getByRole('button', { name: c.nextChange })).toBeDisabled();
    fireEvent.click(screen.getByRole('button', { name: c.previousChange })); expect(screen.getByText('+after')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: c.options })); fireEvent.click(screen.getByRole('radio', { name: c.unified })); fireEvent.click(screen.getByTestId('inspection-options.close')); await waitFor(() => { expect(screen.queryByRole('dialog')).not.toBeInTheDocument(); }); expect(screen.queryByText(c.oldVersion, { selector: "p" })).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: strings.readerPreview.showSource })); expect(screen.getByTestId('file-preview-code')).toBeInTheDocument();
  });
  it('retains exact patch copy and a reliable source fallback for malformed input', async () => {
    render(<DiffPreview {...props('@@ invalid\n+original')} />); expect(screen.getByText(c.diffFallback)).toBeInTheDocument(); expect(screen.getByTestId('file-preview-code')).toBeInTheDocument();
    fireEvent.click(screen.getByText(strings.artifactPreview.actions)); await act(async () => { fireEvent.click(screen.getByRole('button', { name: c.copySource })); await Promise.resolve(); }); expect(copyText).toHaveBeenCalledWith('@@ invalid\n+original');
  });
  it('switches files directly, changes wrapping, dismisses options and returns from source', async () => {
    render(<DiffPreview {...props()} />);
    fireEvent.change(screen.getByRole('combobox', { name: c.changedFile }), { target: { value: '1' } }); expect(screen.getByText('+created')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: c.options })); fireEvent.click(screen.getByRole('checkbox', { name: strings.messagesFileViewer.wrap })); expect(screen.getByRole('checkbox')).not.toBeChecked();
    fireEvent.keyDown(screen.getByRole('dialog'), { key: 'Escape' }); await waitFor(() => { expect(screen.queryByRole('dialog')).not.toBeInTheDocument(); });
    fireEvent.click(screen.getByRole('button', { name: strings.readerPreview.showSource }));
    fireEvent.click(screen.getByRole('button', { name: c.showChanges })); expect(screen.getByText('+created')).toBeInTheDocument();
  });

  it('bounds a large hunk while keeping every loaded change reachable by paging', () => {
    const changes = Array.from({ length: 900 }, (_, index) => `+added ${String(index + 1)}`).join('\n');
    render(<DiffPreview {...props(`@@ -0,0 +1,900 @@\n${changes}\n`)} />);
    expect(screen.getByText('+added 1')).toBeInTheDocument(); expect(screen.queryByText('+added 401')).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: strings.artifactPreview.nextPage })); expect(screen.getByText('+added 401')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: strings.artifactPreview.previousPage })); expect(screen.getByText('+added 1')).toBeInTheDocument();
  });

});
