import { describe, expect, it } from "vitest";
import { i18n } from "./index";

describe("i18n fallback", () => {
  it("uses a supplied default value and otherwise returns the key", () => {
    expect(i18n.t("desktop.ready", { defaultValue: "Desktop ready" })).toBe(
      "Desktop ready",
    );
    expect(i18n.t("desktop.missing")).toBe("desktop.missing");
  });
});
