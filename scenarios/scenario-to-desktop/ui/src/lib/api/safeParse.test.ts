import { describe, expect, it, vi } from "vitest";
import {
  isValidationError,
  parseOrThrow,
  safeParse,
  safeParseWithDefault,
  ValidationError,
  z,
} from "./safeParse";

const desktopTargetSchema = z.object({
  scenario: z.string().min(1),
  platform: z.enum(["linux", "win", "mac"]),
});

describe("safe parsing boundary", () => {
  it("returns typed data for valid generated-desktop inputs", () => {
    expect(
      safeParse(desktopTargetSchema, {
        scenario: "calculator",
        platform: "linux",
      }),
    ).toEqual({
      success: true,
      data: { scenario: "calculator", platform: "linux" },
    });
  });

  it("returns human-readable field diagnostics for malformed inputs", () => {
    const result = safeParse(desktopTargetSchema, {
      scenario: "",
      platform: "android",
    });

    expect(result.success).toBe(false);
    if (!result.success) {
      expect(result.error).toContain("scenario:");
      expect(result.error).toContain("platform:");
      expect(result.details?.issues).toHaveLength(2);
    }
  });

  it("uses a fallback only for malformed non-critical data and warns in development", () => {
    const warn = vi.spyOn(console, "warn").mockImplementation(() => undefined);
    const fallback = { scenario: "fallback", platform: "linux" as const };

    expect(
      safeParseWithDefault(
        desktopTargetSchema,
        { scenario: "", platform: "linux" },
        fallback,
      ),
    ).toBe(fallback);
    expect(warn).toHaveBeenCalledWith(
      "[safeParse] Using default value due to validation failure:",
      expect.objectContaining({ data: { scenario: "", platform: "linux" } }),
    );
    expect(
      safeParseWithDefault(
        desktopTargetSchema,
        { scenario: "calculator", platform: "linux" },
        fallback,
      ),
    ).toEqual({ scenario: "calculator", platform: "linux" });
  });

  it("throws and identifies ValidationError at strict boundaries", () => {
    expect(() =>
      parseOrThrow(desktopTargetSchema, { scenario: "", platform: "win" }),
    ).toThrow(ValidationError);
    try {
      parseOrThrow(desktopTargetSchema, { scenario: "", platform: "win" });
    } catch (error) {
      expect(isValidationError(error)).toBe(true);
      if (isValidationError(error)) {
        expect(error.zodError.issues[0]?.path).toEqual(["scenario"]);
      }
    }
    expect(isValidationError(new Error("ordinary"))).toBe(false);
  });
});
