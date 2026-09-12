/**
 * Tests for AISearchResults. Verifies that a non-empty query triggers a
 * searchAI request with the expected body, renders scored results, and shows
 * a fallback hint when the API reports fallback='unavailable'.
 */

import { describe, expect, it, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { AISearchResults } from "./AISearchResults";
import * as aiSearch from "../../../../lib/ai-search";

beforeEach(() => {
  vi.restoreAllMocks();
});

describe("AISearchResults", () => {
  it("shows an empty-query hint when query is blank", () => {
    render(<AISearchResults query="" onItemClick={() => {}} />);
    expect(screen.getByText(/type to search/i)).toBeInTheDocument();
  });

  it("calls searchAI with the expected request shape and renders results", async () => {
    const spy = vi.spyOn(aiSearch, "searchAI").mockResolvedValue({
      results: [
        {
          entity: "backlog",
          id: "alpha",
          score: 0.9,
          scorePercent: 90,
          payload: { title: "Alpha Item", status: "ready", kind: "execute" },
        },
      ],
      total: 1,
      query: "retry",
      entity: "both",
      fallback: "none",
      latencyMs: 12,
    });

    render(<AISearchResults query="retry" onItemClick={() => {}} />);

    await waitFor(() => expect(spy).toHaveBeenCalledTimes(1));
    expect(spy).toHaveBeenCalledWith({ query: "retry", entity: "both", limit: 20 });

    await waitFor(() => screen.getByTestId("ai-search-results"));
    expect(screen.getByText("Alpha Item")).toBeInTheDocument();
    expect(screen.getByText("90%")).toBeInTheDocument();
  });

  it("renders a fallback-unavailable hint when the API reports it", async () => {
    vi.spyOn(aiSearch, "searchAI").mockResolvedValue({
      results: [],
      total: 0,
      query: "x",
      entity: "both",
      fallback: "unavailable",
      latencyMs: 1,
    });

    render(<AISearchResults query="x" onItemClick={() => {}} />);
    await waitFor(() => screen.getByTestId("ai-search-results"));
    expect(screen.getByText(/AI search unavailable/i)).toBeInTheDocument();
  });

  it("renders an error banner when searchAI rejects", async () => {
    vi.spyOn(aiSearch, "searchAI").mockRejectedValue(new Error("boom"));

    render(<AISearchResults query="x" onItemClick={() => {}} />);
    await waitFor(() => screen.getByTestId("ai-search-error"));
  });
});
