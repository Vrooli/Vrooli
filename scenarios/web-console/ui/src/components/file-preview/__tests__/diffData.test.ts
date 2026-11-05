import { describe, expect, it } from 'vitest';
import { parseDiff, splitHunk } from '../diffData';
describe('unified patch model', () => {
  it('keeps old/new coordinates and pairs adjacent additions/deletions', () => {
    const result = parseDiff('diff --git a/a.ts b/a.ts\n--- a/a.ts\n+++ b/a.ts\n@@ -4,3 +4,4 @@ function\n context\n-old\n+new\n+extra\n tail\n');
    expect(result.valid).toBe(true); const file = result.files[0]; expect(file).toMatchObject({ name: 'a.ts', oldName: 'a.ts', added: 2, removed: 1 });
    expect(file?.hunks[0]?.lines.map((line) => [line.kind, line.old, line.next])).toEqual([['context', 4, 4], ['remove', 5, null], ['add', null, 5], ['add', null, 6], ['context', 6, 7]]);
    const pairs = splitHunk(file?.hunks[0]?.lines ?? []); expect(pairs[1]?.left?.text).toBe('-old'); expect(pairs[1]?.right?.text).toBe('+new'); expect(pairs[2]?.left).toBeNull();
  });
  it('handles files, empty ranges, no-newline notes and metadata-only changes', () => {
    const result = parseDiff('--- /dev/null\n+++ b/new.txt\n@@ -0,0 +1 @@\n+created\n\\ No newline at end of file\n--- a/old.txt\n+++ /dev/null\n@@ -1 +0,0 @@\n-deleted\n');
    expect(result.valid).toBe(true); expect(result.files).toHaveLength(2); expect(result.files[0]?.hunks[0]?.lines[1]?.kind).toBe('note');
    expect(result.files[1]).toMatchObject({ name: '/dev/null', oldName: 'old.txt', removed: 1 });
    expect(parseDiff('diff --git a/a b/a\nold mode 100644\nnew mode 100755\n').valid).toBe(true);
    expect(parseDiff('diff --git a/a b/a\nBinary files a/a and b/a differ\n').valid).toBe(true);
  });
  it('falls back to source for malformed, combined and oversized patches', () => {
    for (const input of ['not a patch', '@@@ -1,1 -1,1 +1,1 @@@\n+combined', '@@ -1,2 +1 @@\n-old\n+new', '@@ -1 +1 @@\n-old\n+new\n+unaccounted', 'x\n'.repeat(12001)]) expect(parseDiff(input).valid).toBe(false);
  });
});
