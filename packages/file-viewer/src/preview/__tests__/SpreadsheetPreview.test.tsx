import { act, fireEvent, screen, waitFor, within } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { renderWithProviders as render } from '../../test-utils/index.js';
import { previewModel, previewProps } from '../../test-utils/previewFixtures.js';
import { strings } from '../../i18n/strings.js';
import { copyText } from '../../lib/clipboard.js';
import type { WorkbookResult } from '../workbookData.js';
import { SpreadsheetPreview } from '../renderers/SpreadsheetPreview.js';
vi.mock('@vrooli/react-component-library/useResizeObserver', () => ({ useResizeObserver: () => ({ ref: () => {}, rect: { width: state.width } }) }));
vi.mock('../useWorkbook.js', () => ({ useWorkbook: (_href: string, _size: number, attempt: number) => { state.attempt = attempt; return state.result; } }));
vi.mock('../../lib/clipboard.js', () => ({ copyText: vi.fn().mockResolvedValue(true) }));
const state = vi.hoisted(() => ({ width: 390, attempt: 0, result: null as WorkbookResult | null }));
const c = strings.inspectionPreview;
const props = previewProps({ model: previewModel({ kind: 'spreadsheet', resolvedPath: '/book.xlsx', basename: 'book.xlsx' }) });
function sample(): WorkbookResult { return { data: { limited: false, totalSheets: 2, sheets: [
  { name: 'Revenue', visibility: 0, range: 'A1:C2', rows: 2, columns: 3, startRow: 0, startColumn: 0, limited: false, cells: [
    { address: 'A1', row: 0, column: 0, display: '<img src=x onerror=alert(1)>', raw: '<img src=x onerror=alert(1)>', type: 's', missingValue: false },
    { address: 'B2', row: 1, column: 1, display: '$1,234.50', raw: '1234.5', formula: '=SUM(B3,1234.5)', type: 'n', missingValue: false },
    { address: 'C2', row: 1, column: 2, display: '', raw: '', formula: '=NOW()', type: 'n', missingValue: true },
  ] }, { name: 'Hidden notes', visibility: 1, range: '', rows: 0, columns: 0, startRow: 0, startColumn: 0, cells: [], limited: false },
] } }; }
beforeEach(() => { vi.clearAllMocks(); state.width = 390; state.result = sample(); });
describe('workbook preview', () => {
  it('composes phone row/cell drill-in, keeps formulas separate and copies exact values', async () => {
    render(<SpreadsheetPreview {...props} />); expect(screen.getByTestId('inspection-workbook-rows')).toBeInTheDocument();
    const row = document.querySelector<HTMLButtonElement>('[data-workbook-row="1"]'); if (!row) throw Error('Missing row'); fireEvent.click(row);
    expect(screen.getByRole('button', { name: c.backToSheet })).toHaveFocus(); const cell = document.querySelector<HTMLButtonElement>('[data-workbook-row-cell="B2"]'); if (!cell) throw Error('Missing cell'); fireEvent.click(cell);
    const detail = screen.getByTestId('inspection-workbook-detail'); expect(detail).toHaveTextContent('$1,234.50'); expect(detail).toHaveTextContent('=SUM(B3,1234.5)');
    await act(async () => { fireEvent.click(within(detail).getByRole('button', { name: c.copyValue })); await Promise.resolve(); }); expect(copyText).toHaveBeenLastCalledWith('1234.5');
    await act(async () => { fireEvent.click(within(detail).getByRole('button', { name: c.copyFormula })); await Promise.resolve(); }); expect(copyText).toHaveBeenLastCalledWith('=SUM(B3,1234.5)');
    fireEvent.click(screen.getByRole('button', { name: c.rowDetails })); expect(document.querySelector('[data-workbook-row-cell="B2"]')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: c.backToSheet })); expect(screen.getByTestId('inspection-workbook-rows')).toBeInTheDocument();
  });
  it('shows a sticky desktop grid with keyboard cell navigation and inert HTML-like content', () => {
    state.width = 1100; render(<SpreadsheetPreview {...props} />); const grid = screen.getByTestId('inspection-workbook-grid'); expect(grid).toBeInTheDocument(); expect(grid.querySelector('img')).toBeNull();
    const first = grid.querySelector<HTMLButtonElement>('[data-workbook-cell="A1"]'); if (!first) throw Error('Missing A1'); act(() => { first.focus(); }); fireEvent.keyDown(first, { key: 'ArrowDown' });
    expect(document.activeElement).toHaveAttribute('data-workbook-cell', 'A2'); fireEvent.keyDown(document.activeElement ?? first, { key: 'End' }); expect(document.activeElement).toHaveAttribute('data-workbook-cell', 'C2'); expect(screen.getByText(c.noSavedValue)).toBeInTheDocument(); const sheets = screen.getByRole('navigation', { name: c.sheet }); fireEvent.click(within(sheets).getByRole('button', { name: /^Hidden notes/ })); expect(screen.getByText(c.emptySheet)).toBeInTheDocument(); fireEvent.click(within(sheets).getByRole('button', { name: 'Revenue' })); expect(screen.getByTestId('inspection-workbook-grid')).toBeInTheDocument();
  });
  it('filters loaded rows, switches hidden sheets, validates addresses and retains download', async () => {
    render(<SpreadsheetPreview {...props} />); fireEvent.click(screen.getByRole('button', { name: c.findCells })); fireEvent.change(screen.getByRole('searchbox', { name: c.findCells }), { target: { value: 'SUM(' } }); expect(document.querySelectorAll('[data-workbook-row]')).toHaveLength(1);
    fireEvent.change(screen.getByRole('combobox', { name: c.sheet }), { target: { value: '1' } }); expect(screen.getByText(c.emptySheet)).toBeInTheDocument();
    fireEvent.change(screen.getByRole('combobox', { name: c.sheet }), { target: { value: '0' } }); fireEvent.click(screen.getByRole('button', { name: c.workbookOptions }));
    const address = screen.getByRole('textbox', { name: c.goCell }); fireEvent.change(address, { target: { value: 'XFD999' } }); fireEvent.click(screen.getByRole('button', { name: c.go })); expect(screen.getByRole('alert')).toHaveTextContent(c.cellInvalid);
    fireEvent.change(address, { target: { value: 'B2' } }); fireEvent.click(screen.getByRole('button', { name: c.go })); await waitFor(() => { expect(screen.queryByRole('dialog')).not.toBeInTheDocument(); }); expect(screen.getByTestId('inspection-workbook-detail')).toHaveTextContent('$1,234.50'); expect(screen.getByTestId('file-preview-download')).toHaveAttribute('href', '/preview/blob');
  });
  it('shows distinct loading, damaged, encrypted and bounded error recovery with retry', () => {
    state.result = null; const { rerender } = render(<SpreadsheetPreview {...props} />); expect(screen.getByRole('status')).toHaveTextContent(c.loadingWorkbook);
    for (const [error, label] of [['invalid', c.workbookError], ['encrypted', c.workbookEncrypted], ['limit', c.workbookTooLarge], ['unavailable', c.workbookUnavailable], ['fetch', c.workbookFetch]] as const) {
      state.result = { error }; rerender(<SpreadsheetPreview {...props} />); expect(screen.getByRole('status')).toHaveTextContent(label);
    }
    fireEvent.click(screen.getByRole('button', { name: c.retry })); expect(state.attempt).toBe(1);
  });
  it('explains that spreadsheet preview is not available here, offering download and no retry', () => {
    state.result = { error: 'unsupported' }; render(<SpreadsheetPreview {...props} />);
    expect(screen.getByRole('status')).toHaveTextContent(c.workbookNotSupported); expect(screen.queryByRole('button', { name: c.retry })).toBeNull();
    expect(screen.getByTestId('file-preview-download')).toHaveAttribute('href', '/preview/blob');
  });
  it('pages loaded rows, switches grid/records views, copies desktop values and dismisses search', async () => {
    state.width = 1100; const data = sample().data; if (!data) throw Error('Missing data'); const firstSheet = data.sheets[0]; if (!firstSheet) throw Error('Missing sheet');
    firstSheet.rows = 85; firstSheet.range = 'A1:C85'; data.limited = true; firstSheet.limited = true; state.result = { data };
    render(<SpreadsheetPreview {...props} />);
    fireEvent.click(screen.getByRole('button', { name: strings.artifactPreview.nextPage })); expect(document.querySelector('[data-workbook-cell="A41"]')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: strings.artifactPreview.previousPage })); const value = document.querySelector<HTMLButtonElement>('[data-workbook-cell="B2"]'); if (!value) throw Error('Missing value'); fireEvent.click(value);
    await act(async () => { fireEvent.click(screen.getByRole('button', { name: c.copyAddress })); await Promise.resolve(); }); expect(copyText).toHaveBeenLastCalledWith('B2');
    await act(async () => { fireEvent.click(screen.getByRole('button', { name: c.copyValue })); await Promise.resolve(); }); expect(copyText).toHaveBeenLastCalledWith('1234.5');
    fireEvent.click(screen.getByRole('button', { name: c.workbookOptions })); fireEvent.click(screen.getByRole('radio', { name: strings.artifactPreview.recordsView })); expect(screen.getByTestId('inspection-workbook-rows')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('radio', { name: strings.artifactPreview.tableViewLabel }));
    fireEvent.keyDown(screen.getByRole('dialog'), { key: 'Escape' }); await waitFor(() => { expect(screen.queryByRole('dialog')).not.toBeInTheDocument(); });
    fireEvent.click(screen.getByRole('button', { name: c.findCells })); fireEvent.change(screen.getByRole('searchbox'), { target: { value: 'missing' } }); expect(screen.getByText(strings.artifactPreview.noMatchingRows)).toBeInTheDocument();
    fireEvent.keyDown(screen.getByRole('searchbox'), { key: 'Escape' }); expect(screen.queryByRole('searchbox')).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: c.findCells })); fireEvent.click(screen.getByRole('button', { name: c.findCells })); expect(screen.queryByRole('searchbox')).not.toBeInTheDocument();
  });

  it('keeps the phone table unobstructed until a cell is opened and returns focus when details close', async () => {
    render(<SpreadsheetPreview {...props} />);
    fireEvent.click(screen.getByRole('button', { name: c.workbookOptions }));
    fireEvent.click(screen.getByRole('radio', { name: strings.artifactPreview.tableViewLabel }));
    fireEvent.click(screen.getByTestId('inspection-options.close'));
    await waitFor(() => { expect(screen.queryByRole('dialog')).not.toBeInTheDocument(); });
    expect(screen.getByTestId('inspection-workbook-grid')).toBeInTheDocument();
    expect(screen.queryByRole('complementary', { name: c.cellDetails })).not.toBeInTheDocument();
    const cell = document.querySelector<HTMLButtonElement>('[data-workbook-cell="A1"]'); if (!cell) throw Error('Missing A1');
    fireEvent.click(cell);
    expect(screen.getByRole('complementary', { name: c.cellDetails })).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: c.backToSheet }));
    await act(async () => { await new Promise((resolve) => requestAnimationFrame(resolve)); });
    expect(screen.queryByRole('complementary', { name: c.cellDetails })).not.toBeInTheDocument();
    expect(cell).toHaveFocus();
  });

  it('restores desktop pagination when a phone row inspector is resized into a grid', () => {
    const data = sample().data; if (!data) throw Error('Missing data'); const sheet = data.sheets[0]; if (!sheet) throw Error('Missing sheet'); sheet.rows = 85; state.result = { data };
    const { rerender } = render(<SpreadsheetPreview {...props} />); const row = document.querySelector<HTMLButtonElement>('[data-workbook-row="1"]'); if (!row) throw Error('Missing row'); fireEvent.click(row);
    expect(screen.queryByRole('button', { name: strings.artifactPreview.nextPage })).not.toBeInTheDocument();
    state.width = 1100; rerender(<SpreadsheetPreview {...props} />);
    expect(screen.getByTestId('inspection-workbook-grid')).toBeInTheDocument(); expect(screen.getByRole('button', { name: strings.artifactPreview.nextPage })).toBeEnabled();
  });

  describe('header row and chrome', () => {
    const text = (address: string, row: number, column: number, value: string) => ({ address, row, column, display: value, raw: value, type: 's', missingValue: false });
    const num = (address: string, row: number, column: number, value: number) => ({ address, row, column, display: String(value), raw: String(value), type: 'n', missingValue: false });
    function shots(): WorkbookResult { return { data: { limited: false, totalSheets: 1, sheets: [{ name: 'Shots', visibility: 0, range: 'A1:C3', rows: 3, columns: 3, startRow: 0, startColumn: 0, limited: false, cells: [
      text('A1', 0, 0, 'Shot'), text('B1', 0, 1, 'Length (s)'), text('C1', 0, 2, 'Status'),
      text('A2', 1, 0, 'Intro walk-on'), num('B2', 1, 1, 15), text('C2', 1, 2, 'Rendered'),
      text('A3', 2, 0, 'Product demo'), num('B3', 2, 1, 40), text('C3', 2, 2, 'Waiting on set'),
    ] }] } }; }
    const chooseHeader = (name: string) => { fireEvent.click(screen.getByRole('button', { name: c.workbookOptions })); fireEvent.click(screen.getByRole('radio', { name })); };

    it('labels each value by its column header and leaves the header out of the row cards', () => {
      state.result = shots(); render(<SpreadsheetPreview {...props} />);
      const cards = [...document.querySelectorAll('[data-workbook-row]')];
      expect(cards.map((card) => card.getAttribute('data-workbook-row'))).toEqual(['1', '2']);
      expect(cards[0]).toHaveTextContent('Intro walk-on'); expect(cards[0]).toHaveTextContent('Length (s): 15 · Status: Rendered'); expect(cards[0]).not.toHaveTextContent('B:');
      fireEvent.click(cards[0] as HTMLElement); expect(screen.getByTestId('inspection-workbook-detail')).toHaveTextContent('Length (s)');
    });
    it('lets the reader say the first row is data, and bring the header back', () => {
      state.result = shots(); render(<SpreadsheetPreview {...props} />);
      chooseHeader(c.headerNone);
      expect([...document.querySelectorAll('[data-workbook-row]')].map((card) => card.getAttribute('data-workbook-row'))).toEqual(['0', '1', '2']);
      expect(document.querySelector('[data-workbook-row="1"]')).toHaveTextContent('B: 15');
      fireEvent.click(screen.getByRole('radio', { name: c.headerAuto }));
      expect(document.querySelector('[data-workbook-row="0"]')).toBeNull();
    });
    it('names grid columns by header on a wide screen and still opens the header cells by address', () => {
      state.width = 1100; state.result = shots(); render(<SpreadsheetPreview {...props} />);
      const grid = screen.getByTestId('inspection-workbook-grid');
      expect(within(grid).getByRole('columnheader', { name: 'Length (s)' })).toHaveAttribute('title', 'B');
      expect(grid.querySelector('[data-workbook-cell="B1"]')).toBeNull(); expect(grid.querySelector('[data-workbook-cell="B2"]')).toBeInTheDocument();
      fireEvent.click(screen.getByRole('button', { name: c.workbookOptions })); fireEvent.change(screen.getByRole('textbox', { name: c.goCell }), { target: { value: 'B1' } }); fireEvent.click(screen.getByRole('button', { name: c.go }));
      expect(screen.getByTestId('inspection-workbook-detail')).toHaveTextContent('Length (s)');
    });
    it('keeps the chrome calm for one sheet on one page: no sheet picker or paging, one summary line, no row-count bar', () => {
      state.result = shots(); render(<SpreadsheetPreview {...props} />);
      expect(screen.queryByRole('combobox', { name: c.sheet })).toBeNull(); expect(screen.queryByRole('button', { name: strings.artifactPreview.nextPage })).toBeNull();
      expect(screen.getByText(`Shots · A1:C3 · ${c.workbookRows}`)).toBeInTheDocument(); expect(screen.queryByText(strings.artifactPreview.matchingRows)).toBeNull();
      expect(screen.getByText(c.readOnly)).toBeInTheDocument();
    });
    it('counts matching rows in the same line while searching, and shows paging only past one page', () => {
      state.result = shots(); render(<SpreadsheetPreview {...props} />);
      fireEvent.click(screen.getByRole('button', { name: c.findCells })); fireEvent.change(screen.getByRole('searchbox', { name: c.findCells }), { target: { value: 'demo' } });
      expect(screen.getByText(new RegExp(c.workbookMatches))).toBeInTheDocument();
    });
    it('leaves file size and download to a host that shows them itself', () => {
      state.result = shots(); const { unmount } = render(<SpreadsheetPreview {...props} />); expect(screen.getByTestId('file-preview-download')).toBeInTheDocument(); unmount();
      render(<SpreadsheetPreview {...props} />, { host: { fileInfo: 'host' } }); expect(screen.queryByTestId('file-preview-download')).toBeNull(); expect(screen.queryByText(strings.artifactPreview.actions)).toBeNull();
    });
  });
});
