import { describe, expect, it } from "vitest";

import { isRoutedTestMode, livePath, preserveRoutedTestMode } from "./routedTestMode";

describe("preserveRoutedTestMode", () => {
  it("recognizes and removes stale disposable-test flags while preserving normal navigation state", () => {
    expect(isRoutedTestMode("?vrooli_test_mode=1&e4_fixture=old")).toBe(true);
    expect(livePath("/settings/reach?vrooli_test_mode=1&e4_fixture=old&tab=email#channels"))
      .toBe("/settings/reach?tab=email#channels");
    expect(livePath("/settings/reach?tab=email#channels"))
      .toBe("/settings/reach?tab=email#channels");
  });

  it("keeps only an explicitly enabled routed-test flag while preserving the destination query", () => {
    expect(preserveRoutedTestMode("/asks/ask-1", "?vrooli_test_mode=1&q=delivery"))
      .toBe("/asks/ask-1?vrooli_test_mode=1");
    expect(preserveRoutedTestMode("/history?state=expired", "?vrooli_test_mode=1"))
      .toBe("/history?state=expired&vrooli_test_mode=1");
  });

  it("does not add a routed-test flag to ordinary navigation", () => {
    expect(preserveRoutedTestMode("/asks/ask-1", "?q=delivery")).toBe("/asks/ask-1");
  });
});
