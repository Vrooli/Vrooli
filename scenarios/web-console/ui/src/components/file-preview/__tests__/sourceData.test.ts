import { describe, expect, it } from "vitest";
import { isLogPath, logLevel, searchSource, selectedSource, sourceLines } from "../sourceData";
describe('source inspection', () => {
  it('finds literal matches with original offsets and bounds result count', () => {
    expect(searchSource(sourceLines('a+b A+B\r\nnext'), 'a+b')).toEqual({ matches: [{ line: 1, start: 0, end: 3 }, { line: 1, start: 4, end: 7 }], limited: false });
    expect(searchSource(sourceLines('test'), '').matches).toHaveLength(0);
    const capped = searchSource(sourceLines('x'.repeat(501)), 'x'); expect(capped.limited).toBe(true); expect(capped.matches).toHaveLength(500);
  });
  it('recognizes log labels without matching incidental identifiers', () => {
    expect(['ERROR failed', '[FATAL] stop', 'level=panic: failed', '{"level":"error","message":"failed"}'].map(logLevel)).toEqual(['error', 'error', 'error', 'error']);
    expect(['WARN: check', '[WARNING] check', 'INFO started', 'DEBUG trace', '[TRACE] x', 'errorCount = 2'].map(logLevel)).toEqual(['warn', 'warn', 'info', 'debug', 'debug', 'other']);
    expect(logLevel('an error occurred')).toBe('other'); expect(logLevel('2026-10-17T23:25:00Z ERROR failed')).toBe('error');
    expect(isLogPath('/output.stdout.txt')).toBe(true); expect(isLogPath('/server.LOG')).toBe(true); expect(isLogPath('/code.ts')).toBe(false);
  });
  it('copies exact line content and CRLF selections in either direction', () => {
    expect(selectedSource('first\r\nsecond\r\nthird', 2, 1)).toBe('first\r\nsecond');
    expect(selectedSource('first\nsecond\n', 2, 3)).toBe('second\n'); expect(selectedSource('first\r', 1, 1)).toBe('first\r'); expect(selectedSource('first\r\nsecond', 1, 1)).toBe('first');
  });
});
