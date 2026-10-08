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

  it("maps a Connect error whose code has no rawMessage interpolation", () => {
    const err = new ConnectError("missing", Code.NotFound);
    // Assert against the same i18n key path the implementation uses, never
    // against brittle copy.
    expect(errorMessage(err, i18n.t)).toBe(i18n.t("errors.not_found", { message: "missing" }));
  });

  it("maps a known REST ApiError code directly", () => {
    const err = makeApiError("permission_denied", "nope", 403);
    expect(errorMessage(err, i18n.t)).toBe(
      i18n.t("errors.permission_denied", { message: "permission_denied: nope" }),
    );
  });

  it("maps an unrecognised REST ApiError code to the generic unknown message", () => {
    const err = makeApiError("teapot", "I'm a teapot", 418);
    // normalizeApiErrorCode's `?? strings.errors.unknown` branch.
    expect(errorMessage(err, i18n.t)).toBe(
      i18n.t("errors.unknown", { message: "teapot: I'm a teapot" }),
    );
  });

  it("stringifies a non-Error, non-typed throwable", () => {
    expect(errorMessage("plain string failure", i18n.t)).toBe("plain string failure");
    expect(errorMessage(42, i18n.t)).toBe("42");
    expect(errorMessage(null, i18n.t)).toBe("null");
    expect(errorMessage({ toString: () => "obj-err" }, i18n.t)).toBe("obj-err");
  });
});
