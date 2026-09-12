/**
 * useDocumentTitle tests — sets a distinct document title while mounted and
 * restores the previous one on unmount.
 */
import { afterEach, describe, expect, it } from "vitest";
import { renderHook } from "@testing-library/react";

import { useDocumentTitle } from "./useDocumentTitle";

describe("useDocumentTitle", () => {
  afterEach(() => {
    document.title = "";
  });

  it("sets '<title> · <app>' while mounted", () => {
    renderHook(() => useDocumentTitle("Overview"));
    expect(document.title).toBe("Overview · Tunnel Manager");
  });

  it("honours a custom app name", () => {
    renderHook(() => useDocumentTitle("Exposure", "Custom App"));
    expect(document.title).toBe("Exposure · Custom App");
  });

  it("restores the previous title on unmount", () => {
    document.title = "Previous";
    const { unmount } = renderHook(() => useDocumentTitle("Metrics"));
    expect(document.title).toBe("Metrics · Tunnel Manager");
    unmount();
    expect(document.title).toBe("Previous");
  });
});
