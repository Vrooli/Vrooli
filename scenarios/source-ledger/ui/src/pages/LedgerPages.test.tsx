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

  beforeEach(() => {
    fetchSpy = vi.spyOn(globalThis, "fetch").mockImplementation(async (input, init) => {
      const url = String(input);
      const requestBody = typeof init?.body === "string" ? JSON.parse(init.body) as { scope?: string } : {};
      if (url.includes("ScopesService/ListScopes")) return response({ scopes: [scope] });
      if (url.includes("JournalService/ListEntries")) {
        return response({ entries: [{ id: "entry-1", body: "A durable source-ledger fact", facetId: "decision", kind: "decision" }] });
      }
      if (url.includes("ForestService/GetFrontier")) return response({ eligibleCount: 1, target: 32, nodes: [{ id: "node-1", entryId: "entry-1", facetId: "decision", depth: 1 }] });
      if (url.includes("FacetsService/ListFacets")) return response({ facets: [{ id: "decision", label: "Decision", retentionPolicy: "retain" }] });
      if (url.includes("RecallService/Recall")) return response({ hits: [{ entryId: "entry-1", facetId: "decision", text: `Result from ${requestBody.scope}`, score: 0.91 }] });
      return response({});
    });
  });

  afterEach(() => {
    cleanup();
    fetchSpy?.mockRestore();
    fetchSpy = undefined;
  });

  it("renders the scope inventory and bounded health metrics", async () => {
    renderWithProviders(<TestAppRouter initialEntries={["/"]} />, { withoutRouter: true });
    await waitFor(() => expect(screen.getByText("1/32")).toBeInTheDocument());
    expect(screen.getByText("256")).toBeInTheDocument();
  });

  it("renders source timeline, frontier, and facet review for a valid scope", async () => {
    renderWithProviders(<TestAppRouter initialEntries={["/scopes/agent-memory"]} />, { withoutRouter: true });
    await waitFor(() => expect(screen.getByText("A durable source-ledger fact")).toBeInTheDocument());
    expect(screen.getByText("Frontier explorer")).toBeInTheDocument();
    expect(screen.getByText("Facet review queue")).toBeInTheDocument();
  });

  it("renders a bounded state for an invalid placeholder scope", async () => {
    renderWithProviders(<TestAppRouter initialEntries={["/scopes/:scope"]} />, { withoutRouter: true });
    await waitFor(() => expect(screen.getByText("Scope unavailable")).toBeInTheDocument());
    expect(screen.getByRole("link", { name: "Back to ledger" })).toHaveAttribute("href", "/");
  });

  it("renders vocabulary controls for a valid scope", async () => {
    renderWithProviders(<TestAppRouter initialEntries={["/scopes/agent-memory/vocabulary"]} />, { withoutRouter: true });
    await waitFor(() => expect(screen.getByLabelText("decision retention policy")).toBeInTheDocument());
    expect(screen.getByRole("button", { name: "Save vocabulary draft" })).toBeInTheDocument();
  });

  it("keeps vocabulary placeholders bounded and exposes the saved draft state", async () => {
    renderWithProviders(<TestAppRouter initialEntries={["/scopes/:scope/vocabulary"]} />, { withoutRouter: true });
    await waitFor(() => expect(screen.getByText("Scope unavailable")).toBeInTheDocument());

    cleanup();
    renderWithProviders(<TestAppRouter initialEntries={["/scopes/agent-memory/vocabulary"]} />, { withoutRouter: true });
    await waitFor(() => expect(screen.getByRole("button", { name: "Save vocabulary draft" })).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: "Save vocabulary draft" }));
    expect(screen.getByRole("button", { name: "Draft saved" })).toBeInTheDocument();
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
