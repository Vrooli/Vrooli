// [REQ:REQ-P1-002-UI-OPERATIONS-PARITY]
import { describe, expect, it } from "vitest";
import {
  MEMBER_ITEM_STRATEGY_LABEL,
  MEMBER_ITEM_STRATEGY_WIRE_VALUE,
  isMemberItemStrategy,
  normalizeModeWireValue,
  presentModeLabel,
  resolveModePresentation,
} from "./member-item-strategy";

describe("member-item-strategy mapping module", () => {
  describe("resolveModePresentation", () => {
    it("maps the legacy 'item-level' wire value to the strategy presentation", () => {
      expect(resolveModePresentation("item-level")).toEqual({
        kind: "member-item-strategy",
        wireValue: MEMBER_ITEM_STRATEGY_WIRE_VALUE,
        label: MEMBER_ITEM_STRATEGY_LABEL,
      });
    });

    it("maps blank / null / undefined (legacy records) to the strategy presentation", () => {
      for (const value of ["", "   ", null, undefined]) {
        expect(resolveModePresentation(value).kind).toBe("member-item-strategy");
      }
    });

    it("passes genuine modes through untouched", () => {
      expect(resolveModePresentation("holistic-loop")).toEqual({
        kind: "mode",
        mode: "holistic-loop",
      });
      expect(resolveModePresentation("phased-plan-drain")).toEqual({
        kind: "mode",
        mode: "phased-plan-drain",
      });
    });
  });

  describe("isMemberItemStrategy", () => {
    it("is true only for the legacy wire value and blank", () => {
      expect(isMemberItemStrategy("item-level")).toBe(true);
      expect(isMemberItemStrategy("")).toBe(true);
      expect(isMemberItemStrategy(undefined)).toBe(true);
      expect(isMemberItemStrategy("holistic-loop")).toBe(false);
    });
  });

  describe("normalizeModeWireValue (data-side default)", () => {
    it("collapses blank to the strategy wire value — the permanent persisted-wire policy", () => {
      expect(normalizeModeWireValue("")).toBe("item-level");
      expect(normalizeModeWireValue(null)).toBe("item-level");
      expect(normalizeModeWireValue(undefined)).toBe("item-level");
      expect(normalizeModeWireValue("item-level")).toBe("item-level");
    });

    it("passes genuine modes through", () => {
      expect(normalizeModeWireValue("holistic-loop")).toBe("holistic-loop");
    });
  });

  describe("presentModeLabel (display-side mapping)", () => {
    it("relabels the strategy — statistics keep the bucket, renamed not dropped", () => {
      expect(presentModeLabel("item-level")).toBe("Member-item workflow");
      expect(presentModeLabel("")).toBe("Member-item workflow");
    });

    it("ignores the server catalog label for the strategy (explicit relabel)", () => {
      expect(presentModeLabel("item-level", "Item Level")).toBe("Member-item workflow");
    });

    it("prefers the server label for genuine modes", () => {
      expect(presentModeLabel("holistic-loop", "Holistic Loop!")).toBe("Holistic Loop!");
    });

    it("humanizes genuine mode ids without a server label", () => {
      expect(presentModeLabel("phased-plan-drain")).toBe("Phased Plan Drain");
      expect(presentModeLabel("holistic_loop")).toBe("Holistic Loop");
    });
  });

  describe("deep-link normalization contract", () => {
    it("the legacy route param resolves to the strategy presentation, keeping the wire value addressable", () => {
      // /operating-modes/item-level stays a permanently valid URL; the page
      // presents it as the strategy via this mapping.
      const presentation = resolveModePresentation("item-level");
      expect(presentation.kind).toBe("member-item-strategy");
      if (presentation.kind === "member-item-strategy") {
        expect(presentation.wireValue).toBe("item-level");
        expect(presentation.label).toBe(MEMBER_ITEM_STRATEGY_LABEL);
      }
    });
  });
});
