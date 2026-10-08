import { Code, ConnectError } from "@connectrpc/connect";
import { beforeEach, describe, expect, it } from "vitest";

import { makeApiError } from "../api/client";
import { i18n } from "../i18n";
import { errorMessage } from "./errorMessage";

describe("errorMessage", () => {
  beforeEach(async () => {
    await i18n.changeLanguage("en");
  });

  it("maps Connect errors to Connect code i18n keys", () => {
    const err = new ConnectError("title is required", Code.InvalidArgument);

    expect(errorMessage(err, i18n.t)).toBe("Invalid input: title is required");
  });

  it("maps REST ApiError codes through the same error catalog", () => {
    const err = makeApiError("invalid_request", "title is required", 400);

    expect(errorMessage(err, i18n.t)).toContain("Invalid input:");
  });

  it("falls back to ordinary error messages", () => {
    expect(errorMessage(new Error("boom"), i18n.t)).toBe("boom");
  });

  it("maps unknown API codes and non-error values safely", () => {
    expect(errorMessage(makeApiError("brand_new_code", "new failure", 500), i18n.t)).toBe("An unknown error occurred.");
    expect(errorMessage("plain failure", i18n.t)).toBe("plain failure");
  });
});
