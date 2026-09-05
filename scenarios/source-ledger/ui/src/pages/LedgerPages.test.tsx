import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";

import { renderWithProviders } from "../test-utils";
import { TestAppRouter } from "../app/routes";

const scope = {
  id: "agent-memory",
  label: "Agent memory",
  frontierTarget: 32,
  wakeBudget: 256,
  maxEntryLines: 4,
};

function response(body: unknown) {
  return new Response(JSON.stringify(body), { status: 200, headers: { "content-type": "application/json" } });
}

describe("Source Ledger corpus pages", () => {
  let fetchSpy: { mockRestore: () => void } | undefined;
  let emptyCollections = false;

  beforeEach(() => {
    fetchSpy = vi.spyOn(globalThis, "fetch").mockImplementation(async (input, init) => {
      const url = String(input);
      const requestBody = typeof init?.body === "string" ? JSON.parse(init.body) as { scope?: string } : {};
      if (url.includes("ScopesService/ListScopes")) return response({ scopes: [scope] });
      if (url.includes("ScopesService/GetPolicy")) return response({
        effective: { frontierTarget: 32, wakeBudgetLines: 96, wakeBudgetChars: 12000, maxEntryLines: 2, maxEntryChars: 200, frontierTargetOrigin: "file-default", wakeBudgetLinesOrigin: "file-default", wakeBudgetCharsOrigin: "file-default", maxEntryLinesOrigin: "file-default", maxEntryCharsOrigin: "file-default" },
        defaults: { frontierTarget: 16, wakeBudgetLines: 96, wakeBudgetChars: 12000, maxEntryLines: 2, maxEntryChars: 200 },
        liveness: { unsummarizedLeafCount: 2, oldestUnsummarizedLeafAt: "2026-04-26T00:00:00Z", lastSummaryAt: "2026-04-25T00:00:00Z" },
      });
      if (url.includes("JournalService/ListEntries")) {
        if (emptyCollections) return response({});
        return response({ entries: [{ id: "entry-1", body: "A durable source-ledger fact", facetId: "decision", kind: "decision" }] });
      }
      if (url.includes("ForestService/GetFrontier")) return response(emptyCollections ? {} : { eligibleCount: 1, target: 32, nodes: [{ id: "node-1", entryId: "entry-1", facetId: "decision", depth: 1 }] });
      if (url.includes("FacetsService/ListFacets")) return response(emptyCollections ? {} : { facets: [{ id: "decision", label: "Decision", guidance: "A durable decision.", retentionPolicy: "retain", compactionEligible: false, residentBudget: 4 }] });
      if (url.includes("FacetsService/SetFacetPolicy")) return response({ facet: { id: "decision", label: "Decision", retentionPolicy: "retain", compactionEligible: false, residentBudget: 4 } });
      if (url.includes("RecallService/Recall")) return response({ hits: [{ entryId: "entry-1", facetId: "decision", text: `Result from ${requestBody.scope}`, score: 0.91 }] });
      return response({});
    });
  });

  afterEach(() => {
    cleanup();
    fetchSpy?.mockRestore();
    fetchSpy = undefined;
    emptyCollections = false;
  });

  it("renders the scope inventory and bounded health metrics", async () => {
    renderWithProviders(<TestAppRouter initialEntries={["/"]} />, { withoutRouter: true });
    await waitFor(() => expect(screen.getByText("1/32")).toBeInTheDocument());
    expect(screen.getByText("256")).toBeInTheDocument();
  });

  it("does not crash when proto JSON omits empty repeated fields", async () => {
    emptyCollections = true;
    renderWithProviders(<TestAppRouter initialEntries={["/"]} />, { withoutRouter: true });
    await waitFor(() => expect(screen.getByText("Agent memory")).toBeInTheDocument());
    await waitFor(() => expect(screen.getAllByText("…").length).toBeGreaterThan(0));
  });

  it("renders source timeline, frontier, and facet review for a valid scope", async () => {
    renderWithProviders(<TestAppRouter initialEntries={["/scopes/agent-memory"]} />, { withoutRouter: true });
    await waitFor(() => expect(screen.getByText("A durable source-ledger fact")).toBeInTheDocument());
    expect(screen.getByText("Frontier explorer")).toBeInTheDocument();
    expect(screen.getByText("Facet review queue")).toBeInTheDocument();
    expect(screen.getByText(/Unsummarized leaves: 2/)).toBeInTheDocument();
  });

  it("renders a bounded state for an invalid placeholder scope", async () => {
    renderWithProviders(<TestAppRouter initialEntries={["/scopes/:scope"]} />, { withoutRouter: true });
    await waitFor(() => expect(screen.getByText("Scope unavailable")).toBeInTheDocument());
    expect(screen.getByRole("link", { name: "Back to ledger" })).toHaveAttribute("href", "/");
  });

  it("renders vocabulary controls for a valid scope", async () => {
    renderWithProviders(<TestAppRouter initialEntries={["/scopes/agent-memory/vocabulary"]} />, { withoutRouter: true });
    await waitFor(() => expect(screen.getByLabelText("decision retention policy")).toBeInTheDocument());
    expect(screen.getByRole("button", { name: "Save vocabulary policy" })).toBeInTheDocument();
  });

  it("persists vocabulary policy changes through the operator surface", async () => {
    renderWithProviders(<TestAppRouter initialEntries={["/scopes/:scope/vocabulary"]} />, { withoutRouter: true });
    await waitFor(() => expect(screen.getByText("Scope unavailable")).toBeInTheDocument());

    cleanup();
    renderWithProviders(<TestAppRouter initialEntries={["/scopes/agent-memory/vocabulary"]} />, { withoutRouter: true });
    await waitFor(() => expect(screen.getByRole("button", { name: "Save vocabulary policy" })).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: "Save vocabulary policy" }));
    await waitFor(() => expect(screen.getByRole("button", { name: "Policy saved" })).toBeInTheDocument());
  });

  it("renders an empty search without issuing recall work", async () => {
    renderWithProviders(<TestAppRouter initialEntries={["/search"]} />, { withoutRouter: true });
    await waitFor(() => expect(screen.getByRole("heading", { name: "Cross-scope search" })).toBeInTheDocument());
    expect(screen.getByRole("textbox", { name: "Search every scope" })).toHaveValue("");
  });

  it("fans recall out across the registered scopes and labels the result", async () => {
    renderWithProviders(<TestAppRouter initialEntries={["/search?q=memory"]} />, { withoutRouter: true });
    await waitFor(() => expect(screen.getByText("Result from agent-memory")).toBeInTheDocument());
    expect(screen.getByText("agent-memory · decision")).toBeInTheDocument();
  });
});
