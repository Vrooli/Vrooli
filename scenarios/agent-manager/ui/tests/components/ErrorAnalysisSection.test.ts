import assert from "node:assert/strict";
import { screen, within } from "@testing-library/react";
import { createElement } from "react";
import { test, vi } from "vitest";
import type { useQuery } from "@tanstack/react-query";
import { ErrorAnalysisSection } from "../../src/features/stats/components/errors/ErrorAnalysisSection.js";
import type { ErrorPatternsResponse } from "../../src/features/stats/api/types.js";
import { useErrorAnalysis } from "../../src/features/stats/hooks/useErrorAnalysis.js";
import { renderWithProviders } from "../../src/test-utils/index.js";
import { makeErrorPatternsResponse } from "../testutil/stats.js";

vi.mock("../../src/features/stats/hooks/useErrorAnalysis.js", () => ({
  useErrorAnalysis: vi.fn(),
}));

type QueryResult = ReturnType<typeof useQuery<ErrorPatternsResponse, Error>>;

function queryResult(overrides: Partial<QueryResult>): QueryResult {
  return {
    data: undefined,
    isLoading: false,
    error: null,
    ...overrides,
  } as QueryResult;
}

test("ErrorAnalysisSection renders error totals, messages, and sample run links", () => {
  vi.mocked(useErrorAnalysis).mockReturnValue(queryResult({
    data: makeErrorPatternsResponse(),
  }));

  renderWithProviders(createElement(ErrorAnalysisSection));

  assert.ok(screen.getByText("Error Analysis"));
  assert.ok(screen.getByText("13 total errors"));
  assert.ok(screen.getByText("12 occurrences"));
  assert.ok(screen.getByText("1 occurrence"));
  assert.ok(screen.getByText("workspace-sandbox unavailable while applying patch"));
  assert.ok(screen.getByText("model exhausted fallback chain"));

  const sampleLinks = screen.getAllByRole("link", { name: /view sample run: run-erro/i });
  assert.deepEqual(
    sampleLinks.map((link) => link.getAttribute("href")),
    ["#/runs/run-error-12345678", "#/runs/run-error-87654321"],
  );
});

test("ErrorAnalysisSection truncates long error codes but keeps the full title", () => {
  const longError = "model ".repeat(25).trim();
  vi.mocked(useErrorAnalysis).mockReturnValue(queryResult({
    data: makeErrorPatternsResponse({
      errors: [
        {
          errorCode: longError,
          count: 3,
          lastSeen: "2026-05-01T16:00:00.000Z",
          sampleRunId: "run-long-error",
        },
      ],
    }),
  }));

  renderWithProviders(createElement(ErrorAnalysisSection));

  const renderedError = screen.getByTitle(longError);
  assert.ok(renderedError.textContent?.endsWith("..."));
  assert.ok((renderedError.textContent?.length ?? 0) < longError.length);
});

test("ErrorAnalysisSection renders empty, loading, and error states", () => {
  vi.mocked(useErrorAnalysis).mockReturnValue(queryResult({
    data: makeErrorPatternsResponse({ errors: [] }),
  }));

  const empty = renderWithProviders(createElement(ErrorAnalysisSection));
  assert.ok(screen.getByText("No errors detected"));
  assert.ok(screen.getByText("All runs completed successfully in this time period"));

  empty.unmount();
  vi.mocked(useErrorAnalysis).mockReturnValue(queryResult({
    isLoading: true,
  }));

  const loading = renderWithProviders(createElement(ErrorAnalysisSection));
  assert.equal(document.querySelectorAll(".animate-pulse").length, 4);

  loading.unmount();
  vi.mocked(useErrorAnalysis).mockReturnValue(queryResult({
    error: new Error("error stats unavailable"),
  }));

  renderWithProviders(createElement(ErrorAnalysisSection));
  const panel = screen.getByText("Error Analysis").closest("div");
  assert.ok(panel);
  assert.ok(within(panel).getByText("Failed to load: error stats unavailable"));
});
