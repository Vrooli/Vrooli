/**
 * Trailing-partial promotion contract (STT tail-durability).
 *
 * Encodes the desired behavior: a turn that ends on an uncommitted partial must
 * still deliver that text to the durable transcript, and it must NOT duplicate
 * text the server already delivered as a final or a segment-final.
 */
import { describe, expect, it } from "vitest";

import { trailingPartialDelta } from "./trailingPartial";

describe("trailingPartialDelta", () => {
  it("promotes an uncommitted trailing partial when the server never flushed it", () => {
    // kyutai: final is empty (segments committed), the tail rode a partial the
    // teardown race dropped. It must be delivered, not wiped.
    expect(
      trailingPartialDelta({
        trailingPartial: "and one last thing",
        finalDelivered: false,
        hasSegments: true,
      }),
    ).toBe(" and one last thing");
  });

  it("promotes the partial with no leading space when it is the only text in the turn", () => {
    expect(
      trailingPartialDelta({
        trailingPartial: "hello world",
        finalDelivered: false,
        hasSegments: false,
      }),
    ).toBe("hello world");
  });

  it("does NOT double-append when the server final already delivered the tail", () => {
    expect(
      trailingPartialDelta({
        trailingPartial: "already delivered",
        finalDelivered: true,
        hasSegments: false,
      }),
    ).toBeNull();
  });

  it("promotes nothing for an empty or whitespace-only partial", () => {
    expect(
      trailingPartialDelta({ trailingPartial: "", finalDelivered: false, hasSegments: false }),
    ).toBeNull();
    expect(
      trailingPartialDelta({ trailingPartial: "   ", finalDelivered: false, hasSegments: true }),
    ).toBeNull();
  });

  it("does not inject a leading space when the tail opens with closing punctuation", () => {
    expect(
      trailingPartialDelta({
        trailingPartial: ", continued",
        finalDelivered: false,
        hasSegments: true,
      }),
    ).toBe(", continued");
  });
});
