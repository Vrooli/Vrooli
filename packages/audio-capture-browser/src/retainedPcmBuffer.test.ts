import { describe, expect, it } from "vitest";
import { RetainedPcmBuffer } from "./retainedPcmBuffer";

describe("RetainedPcmBuffer", () => {
  it("retains and snapshots the full copy up to its byte limit", () => {
    const buffer = new RetainedPcmBuffer(8);
    buffer.append(new Int16Array([1, 2]));
    buffer.append(new Int16Array([3, 4]));
    expect(buffer.byteLength).toBe(8);
    expect(buffer.hasCompleteAudio).toBe(true);
    expect(buffer.snapshot()).toEqual(new Int16Array([1, 2, 3, 4]));
  });

  it("discards the partial copy once the limit is exceeded", () => {
    const buffer = new RetainedPcmBuffer(6);
    buffer.append(new Int16Array([1, 2]));
    buffer.append(new Int16Array([3, 4]));
    expect(buffer.isOverflowed).toBe(true);
    expect(buffer.byteLength).toBe(0);
    expect(buffer.hasCompleteAudio).toBe(false);
    expect(buffer.snapshot()).toBeNull();
    buffer.append(new Int16Array([5]));
    expect(buffer.byteLength).toBe(0);
  });

  it("resets the overflow state for the next turn", () => {
    const buffer = new RetainedPcmBuffer(2);
    buffer.append(new Int16Array([1, 2]));
    expect(buffer.isOverflowed).toBe(true);
    buffer.clear();
    buffer.append(new Int16Array([7]));
    expect(buffer.snapshot()).toEqual(new Int16Array([7]));
  });
});
