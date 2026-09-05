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
  it("retains the message for an unrecognized API error code", () => {
    expect(errorMessage(makeApiError("future_server_code", "service changed", 500), i18n.t)).toContain("service changed");
  });

  it("renders non-Error rejections without throwing again", () => {
    expect(errorMessage("connection closed", i18n.t)).toBe("connection closed");
    expect(errorMessage(null, i18n.t)).toBe("null");
  });

});
