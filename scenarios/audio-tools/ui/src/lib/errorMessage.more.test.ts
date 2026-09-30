/**
 * Additional errorMessage coverage — lines 62-63 (`return String(err)`).
 *
 * The existing test already covers ConnectError, ApiError, and plain Error.
 * This file adds the final else branch: non-Error thrown values (numbers,
 * objects, strings, null).
 */
import { describe, it, expect, beforeEach } from "vitest";

import { i18n } from "../i18n";
import { errorMessage } from "./errorMessage";

beforeEach(async () => {
  await i18n.changeLanguage("en");
});

describe("errorMessage — String(err) fallback (line 62-63)", () => {
  it("coerces a thrown number to its string representation", () => {
    expect(errorMessage(42, i18n.t)).toBe("42");
  });

  it("coerces a thrown null to 'null'", () => {
    expect(errorMessage(null, i18n.t)).toBe("null");
  });

  it("coerces a plain object to '[object Object]'", () => {
    expect(errorMessage({}, i18n.t)).toBe("[object Object]");
  });

  it("coerces a thrown string directly", () => {
    expect(errorMessage("raw string error", i18n.t)).toBe("raw string error");
  });
});
