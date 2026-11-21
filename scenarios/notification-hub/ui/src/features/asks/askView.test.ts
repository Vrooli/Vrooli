import { describe, expect, it } from "vitest";

import { askPreview, askTone, isAwaiting } from "./askView";

const HOUR = 3_600_000;
const NOW = Date.parse("2026-11-14T12:00:00Z");
const inHours = (hours: number) => new Date(NOW + hours * HOUR).toISOString();

describe("askTone", () => {
  it("is loud only for decisions that still wait on the owner", () => {
    expect(askTone({ state: "answered", urgency: "critical", deadline: inHours(1) }, NOW)).toBe("normal");
    expect(askTone({ state: "expired", urgency: "high", deadline: "" }, NOW)).toBe("normal");
  });

  it("treats escalation and critical urgency as critical", () => {
    expect(askTone({ state: "escalated", urgency: "", deadline: "" }, NOW)).toBe("critical");
    expect(askTone({ state: "pending", urgency: "critical", deadline: "" }, NOW)).toBe("critical");
  });

  it("asks for attention for high urgency or a deadline inside a day, and no more", () => {
    expect(askTone({ state: "pending", urgency: "high", deadline: "" }, NOW)).toBe("attention");
    expect(askTone({ state: "pending", urgency: "normal", deadline: inHours(23) }, NOW)).toBe("attention");
    expect(askTone({ state: "pending", urgency: "normal", deadline: inHours(30) }, NOW)).toBe("normal");
    expect(askTone({ state: "pending", urgency: "", deadline: "" }, NOW)).toBe("normal");
  });
});

describe("askPreview", () => {
  it("flattens Markdown into one plain line", () => {
    expect(askPreview({ bigPicture: "## Why\n\nShip **now** or [wait](https://example.test) until `Monday`.\n\n```sh\nrm -rf /\n```" })).toBe("Why Ship now or wait until Monday.");
    expect(askPreview({ bigPicture: "" })).toBe("");
  });
});

describe("isAwaiting", () => {
  it("counts pending and escalated, nothing else", () => {
    expect(["pending", "escalated", "answered", "defaulted", "expired", "withdrawn", "dismissed"].filter((state) => isAwaiting({ state }))).toEqual(["pending", "escalated"]);
  });
});
