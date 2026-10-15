import { describe, expect, it } from "vitest";
import { resolveDeepLinkSession } from "./useSessionManager";

const IDS = [
  "3fd85877-037b-4c8f-a067-f93f3061722a",
  "16d2eba3-2969-5d47-b42c-b94e2b5bdced",
];

describe("resolveDeepLinkSession", () => {
  it("returns the requested session when the operator has it", () => {
    expect(resolveDeepLinkSession("?session=3fd85877-037b-4c8f-a067-f93f3061722a", IDS)).toBe(
      "3fd85877-037b-4c8f-a067-f93f3061722a",
    );
  });

  it("ignores a session the operator does not have", () => {
    expect(resolveDeepLinkSession("?session=does-not-exist", IDS)).toBe("");
  });

  it("returns empty when there is no session param", () => {
    expect(resolveDeepLinkSession("?foo=bar", IDS)).toBe("");
    expect(resolveDeepLinkSession("", IDS)).toBe("");
  });

  it("trims surrounding whitespace before matching", () => {
    expect(resolveDeepLinkSession("?session=%20%203fd85877-037b-4c8f-a067-f93f3061722a%20", IDS)).toBe(
      "3fd85877-037b-4c8f-a067-f93f3061722a",
    );
  });

  it("ignores an empty session value", () => {
    expect(resolveDeepLinkSession("?session=", IDS)).toBe("");
  });

  it("preserves other query params' irrelevance (only session matters)", () => {
    expect(resolveDeepLinkSession("?tab=2&session=16d2eba3-2969-5d47-b42c-b94e2b5bdced&x=1", IDS)).toBe(
      "16d2eba3-2969-5d47-b42c-b94e2b5bdced",
    );
  });
});
