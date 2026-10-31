import { describe, expect, it } from "vitest";

import { encodeQr } from "./qr";

describe("encodeQr", () => {
  it("produces a square module matrix sized to the chosen version", () => {
    const matrix = encodeQr("PAIR-1234");
    expect(matrix.length).toBeGreaterThan(0);
    // Square: every row matches the column count.
    for (const row of matrix) {
      expect(row.length).toBe(matrix.length);
    }
    // QR sizes are 17 + 4*version; version 1 = 21.
    expect((matrix.length - 17) % 4).toBe(0);
  });

  it("places the three finder patterns (dark module at each finder centre)", () => {
    const m = encodeQr("HELLO");
    const n = m.length;
    // Finder centre is at (2..4) of each 7x7 finder; centre core is dark.
    expect(m[3]?.[3]).toBe(true); // top-left
    expect(m[3]?.[n - 4]).toBe(true); // top-right
    expect(m[n - 4]?.[3]).toBe(true); // bottom-left
  });

  it("grows the matrix for a longer payload", () => {
    const small = encodeQr("A");
    const large = encodeQr("X".repeat(60));
    expect(large.length).toBeGreaterThanOrEqual(small.length);
  });

  it("throws for a payload beyond the supported versions", () => {
    expect(() => encodeQr("Z".repeat(200))).toThrow(/too large/i);
  });
});
