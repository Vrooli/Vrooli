import { describe, expect, it } from 'vitest';
import { priorityLabel, priorityTone } from './formatPriority';

describe('priorityLabel', () => {
  it('maps known priorities to RFC 5424 short labels', () => {
    expect(priorityLabel(0)).toBe('EMERG');
    expect(priorityLabel(3)).toBe('ERR');
    expect(priorityLabel(7)).toBe('DEBUG');
  });

  it('returns UNK for unknown priorities', () => {
    expect(priorityLabel(-1)).toBe('UNK');
    expect(priorityLabel(99)).toBe('UNK');
  });
});

describe('priorityTone', () => {
  it('groups 0-2 as critical', () => {
    expect(priorityTone(0)).toBe('critical');
    expect(priorityTone(1)).toBe('critical');
    expect(priorityTone(2)).toBe('critical');
  });
  it('maps 3 to error, 4 to warning, 5 to notice, 6 to info, 7 to debug', () => {
    expect(priorityTone(3)).toBe('error');
    expect(priorityTone(4)).toBe('warning');
    expect(priorityTone(5)).toBe('notice');
    expect(priorityTone(6)).toBe('info');
    expect(priorityTone(7)).toBe('debug');
  });
  it('returns unknown for negative or out-of-range', () => {
    expect(priorityTone(-1)).toBe('unknown');
    expect(priorityTone(42)).toBe('unknown');
  });
});
