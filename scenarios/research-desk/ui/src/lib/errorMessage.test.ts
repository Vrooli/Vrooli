import { Code, ConnectError } from "@connectrpc/connect";
import { describe, expect, it } from "vitest";

import { strings } from "../consts/strings";
import { i18n } from "../i18n";
import { errorMessage, failureKind } from "./errorMessage";

describe("errorMessage", () => {
  it("shows the API's own reason verbatim for a refused request", () => {
    const err = new ConnectError("discard needs a reason", Code.InvalidArgument);
    expect(failureKind(err)).toBe("refused");
    expect(errorMessage(err, i18n.t)).toBe("discard needs a reason");
  });

  it("names an authorization failure as such, not as a generic error", () => {
    expect(failureKind(new ConnectError("no", Code.PermissionDenied))).toBe("unauthorized");
    expect(errorMessage(new ConnectError("no", Code.Unauthenticated), i18n.t)).toBe(strings.errors.unauthorized);
  });

  it("shows the API's words when it refuses an action as not the operator's", () => {
    const words = "this is the operator's action; write a needs-you item instead (caller: agent run r1)";
    expect(failureKind(new ConnectError(words, Code.PermissionDenied))).toBe("unauthorized");
    expect(errorMessage(new ConnectError(words, Code.PermissionDenied), i18n.t)).toBe(words);
    expect(errorMessage(new ConnectError("HTTP 403", Code.PermissionDenied), i18n.t)).toBe(strings.errors.unauthorized);
  });

  it("shows a dependency the API names as down in its words, not as an unreachable desk", () => {
    const words = "Web Console unavailable: connection refused";
    expect(failureKind(new ConnectError(words, Code.Unavailable))).toBe("refused");
    expect(errorMessage(new ConnectError(words, Code.Unavailable), i18n.t)).toBe(words);
  });

  it("names an unreachable desk API, including a network failure", () => {
    expect(failureKind(new ConnectError("HTTP 503", Code.Unavailable))).toBe("unreachable");
    expect(failureKind(new ConnectError("Failed to fetch", Code.Unknown))).toBe("unreachable");
    expect(failureKind(new TypeError("Failed to fetch"))).toBe("unreachable");
    expect(errorMessage(new TypeError("Failed to fetch"), i18n.t)).toBe(strings.errors.unreachable);
  });

  it("shows any other failure's own words", () => {
    expect(errorMessage(new Error("build failed: gate overflow"), i18n.t)).toBe("build failed: gate overflow");
    expect(errorMessage("plain text", i18n.t)).toBe("plain text");
  });
});
