import { describe, expect, it } from "vitest";
import { i18n } from "./index";

describe("i18n", () => {
  it("resolves an authored board string from the en catalog", () => {
    expect(i18n.t("app.title")).toBe("Operational truth at a glance");
  });

  it("returns the provided defaultValue for an unknown key, which is how the library translator falls back", () => {
    expect(i18n.t("library.unknown.key", { defaultValue: "Fallback" })).toBe("Fallback");
  });
});
