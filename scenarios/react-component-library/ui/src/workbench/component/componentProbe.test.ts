import { describe, expect, it } from "vitest";

import { componentProbeFromMessage } from "./componentProbe";

describe("componentProbeFromMessage", () => {
  it("accepts a measured probe with a type sample", () => {
    expect(
      componentProbeFromMessage({
        measured: true,
        targets: { total: 3, small: 1 },
        typography: {
          fontFamily: "Inter",
          fontSize: "16px",
          color: "rgb(244, 246, 251)",
          backgroundStack: ["rgba(0, 0, 0, 0)", "rgb(12, 14, 18)"],
          label: "Retention",
        },
      }),
    ).toEqual({
      measured: true,
      targets: { total: 3, small: 1 },
      typography: {
        fontFamily: "Inter",
        fontSize: "16px",
        color: "rgb(244, 246, 251)",
        backgroundStack: ["rgba(0, 0, 0, 0)", "rgb(12, 14, 18)"],
        label: "Retention",
      },
    });
  });

  it("rejects a message without the measured flag or target counts", () => {
    expect(componentProbeFromMessage(null)).toBeNull();
    expect(componentProbeFromMessage({ measured: true })).toBeNull();
    expect(componentProbeFromMessage({ measured: true, targets: { total: 1 } })).toBeNull();
    expect(componentProbeFromMessage({ targets: { total: 1, small: 0 } })).toBeNull();
  });

  it("drops an incomplete type sample instead of reporting half of one", () => {
    expect(
      componentProbeFromMessage({
        measured: true,
        targets: { total: 1, small: 0 },
        typography: { fontFamily: "Inter" },
      })?.typography,
    ).toBeNull();
  });

  it("never returns negative counts", () => {
    expect(
      componentProbeFromMessage({ measured: true, targets: { total: -2, small: -1 } })?.targets,
    ).toEqual({ total: 0, small: 0 });
  });
});
