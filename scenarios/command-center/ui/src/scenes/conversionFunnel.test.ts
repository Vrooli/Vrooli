import { describe, expect, it } from "vitest";
import { funnelLayout } from "./conversionFunnel";

describe("conversion funnel placement", () => {
  it("uses the open band beside the hero and keeps a restrained width", () => {
    const layout = funnelLayout({ w: 1440, h: 900, quiet: [{ x: 70, y: 200, w: 650, h: 480 }] }, 4);
    expect(layout.centerX).toBeGreaterThan(720);
    expect(layout.maxWidth).toBeLessThan(300);
  });
});
