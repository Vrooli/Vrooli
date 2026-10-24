import { describe, expect, it } from 'vitest';
import { utils, write, type WorkBook } from 'xlsx';
import { parseWorkbook } from '../parseWorkbook';
import { cellCoordinates, checkWorkbookContainer, columnName, MAX_WORKBOOK_BYTES, workbookRows } from '../workbookData';
function bytes(workbook: WorkBook, bookType: 'xlsx' | 'xls' | 'xlsb' = 'xlsx'): ArrayBuffer { return write(workbook, { type: 'array', bookType }) as ArrayBuffer; }
function sample() {
  const workbook = utils.book_new(), sheet = utils.aoa_to_sheet([['Account', 'Value', 'Enabled'], ['North', 1234.5, false], ['', 0, true]]);
  sheet.B2 = { t: 'n', v: 1234.5, z: '$#,##0.00', f: 'SUM(B3,1234.5)' }; sheet.D2 = { t: 'n', f: 'NOW()' }; sheet.E2 = { t: 'e', v: 7 }; sheet.F2 = { t: 'n', v: 45948, z: 'yyyy-mm-dd' };
  sheet['!ref'] = 'A1:F3'; sheet['!merges'] = [utils.decode_range('A1:B1')];
  utils.book_append_sheet(workbook, sheet, 'Revenue'); utils.book_append_sheet(workbook, utils.aoa_to_sheet([['<img onerror=alert(1)>']]), 'Notes'); workbook.Workbook = { Sheets: [{ name: 'Revenue' }, { name: 'Notes', Hidden: 2 }] }; return workbook;
}
describe('bounded workbook reader', () => {
  it('retains formatted numbers/dates, formulas, blank/false/zero values, merges and visibility', () => {
    const result = parseWorkbook(bytes(sample())); expect(result.error).toBeUndefined(); const sheet = result.data?.sheets[0];
    expect(sheet?.cells.find((cell) => cell.address === 'B2')).toMatchObject({ display: '$1,234.50', raw: '1234.5', formula: '=SUM(B3,1234.5)', missingValue: false });
    expect(sheet?.cells.find((cell) => cell.address === 'C2')).toMatchObject({ raw: 'false', type: 'b' }); expect(sheet?.cells.find((cell) => cell.address === 'B3')?.raw).toBe('0');
    expect(sheet?.cells.find((cell) => cell.address === 'D2')).toMatchObject({ formula: '=NOW()', missingValue: true }); expect(sheet?.cells.find((cell) => cell.address === 'F2')?.display).toMatch(/^2025-\d{2}-\d{2}$/);
    expect(sheet?.cells.find((cell) => cell.address === 'A1')?.merge).toBe('A1:B1'); expect(result.data?.sheets[1]).toMatchObject({ name: 'Notes', visibility: 2 });
  });
  it('reads legacy XLS and binary XLSB with inert cell values', () => {
    for (const type of ['xls', 'xlsb'] as const) { const result = parseWorkbook(bytes(sample(), type)); expect(result.error).toBeUndefined(); expect(result.data?.sheets[0]?.cells.find((cell) => cell.address === 'B3')?.raw).toBe('0'); }
  });
  it('preserves offset addresses and empty sheets, limits sheets and physical ranges', () => {
    const workbook = utils.book_new(); const offset = { C5: { t: 's', v: 'offset' }, '!ref': 'C5' } as const; utils.book_append_sheet(workbook, offset, 'Offset'); utils.book_append_sheet(workbook, {}, 'Empty');
    const result = parseWorkbook(bytes(workbook)); expect(result.data?.sheets[0]).toMatchObject({ startRow: 4, startColumn: 2, rows: 1, columns: 1 }); expect(result.data?.sheets[1]?.cells).toHaveLength(0);
    const large = utils.aoa_to_sheet([['first']]); large.A2002 = { t: 's', v: 'last' }; large.CW1 = { t: 's', v: 'wide' }; large['!ref'] = 'A1:CW2002';
    const wide = utils.book_new(); utils.book_append_sheet(wide, large, 'Large'); for (let i = 0; i < 32; i++) utils.book_append_sheet(wide, {}, `Empty${String(i)}`);
    const limited = parseWorkbook(bytes(wide)); expect(limited.data?.limited).toBe(true); expect(limited.data?.sheets).toHaveLength(30); expect(limited.data?.sheets[0]?.columns).toBe(100); expect(limited.data?.sheets[0]?.rows).toBeLessThanOrEqual(500);
  });
  it('filters loaded values/formulas and validates addresses without moving coordinates', () => {
    const sheet = parseWorkbook(bytes(sample())).data?.sheets[0]; if (!sheet) throw Error('Missing sheet');
    expect(workbookRows(sheet, 'north')).toEqual([1]); expect(workbookRows(sheet, 'sum(')).toEqual([1]); expect(workbookRows(sheet, 'B3')).toEqual([1, 2]); expect(workbookRows(sheet, 'missing')).toEqual([]); expect(workbookRows(sheet, '')).toEqual([0, 1, 2]);
    expect(columnName(0)).toBe('A'); expect(columnName(26)).toBe('AA'); expect(columnName(16383)).toBe('XFD'); expect(cellCoordinates('$AA$2000')).toEqual({ row: 1999, column: 26 }); expect(cellCoordinates(' xfd1048576 ')).toEqual({ row: 1048575, column: 16383 });
    for (const address of ['A0', 'A-1', 'XFE1', 'A1048577', 'A1:B2', '=A1', 'A1 bad']) expect(cellCoordinates(address)).toBeNull();
  });
  it('rejects disguised HTML, damaged ZIP and oversized expansion before parsing', () => {
    expect(parseWorkbook(new TextEncoder().encode('<html>not Excel</html>').buffer)).toEqual({ error: 'invalid' }); expect(parseWorkbook(new Uint8Array([0x50, 0x4b, 3, 4]).buffer)).toEqual({ error: 'invalid' }); expect(() => { checkWorkbookContainer(new ArrayBuffer(MAX_WORKBOOK_BYTES + 1)); }).toThrow(RangeError);
    const archive = bytes(sample()), view = new DataView(archive); let directory = 0; for (let i = 0; i < archive.byteLength - 46; i++) if (view.getUint32(i, true) === 0x02014b50) { directory = i; break; }
    view.setUint32(directory + 24, 129 * 1024 * 1024, true); expect(parseWorkbook(archive)).toEqual({ error: 'limit' });
  });
  it('marks values omitted by the text budget instead of presenting them as blank cells', () => {
    const workbook = utils.book_new(); utils.book_append_sheet(workbook, utils.aoa_to_sheet(Array.from({ length: 150 }, () => ['x'.repeat(30000)])), 'Long values');
    const result = parseWorkbook(bytes(workbook)); expect(result.data?.limited).toBe(true);
    const cells = result.data?.sheets[0]?.cells ?? []; expect(cells[0]?.raw).toHaveLength(30000); expect(cells.at(-1)).toMatchObject({ address: 'A150', omitted: true, missingValue: true, raw: '' });
  });

});
