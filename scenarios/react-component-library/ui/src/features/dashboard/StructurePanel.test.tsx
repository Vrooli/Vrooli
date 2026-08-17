import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { getCatalogStructure } from "../../api/catalogGraph";
import type { CatalogStructure } from "../../api/catalogGraph";
import { StructurePanel } from "./StructurePanel";

// provider-free-exception: the structure panel is a standalone typed read-model surface and its state tests intentionally avoid the unrelated application-shell/i18n harness.
vi.mock("../../api/catalogGraph", () => ({ getCatalogStructure: vi.fn() }));
const mocked = vi.mocked(getCatalogStructure);

function renderPanel() { return render(<QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}><StructurePanel /></QueryClientProvider>); }

describe("StructurePanel", () => {
  beforeEach(() => { mocked.mockReset(); });
  it("renders loading and success states", async () => {
    mocked.mockResolvedValue({ population: [{ rung: 3, rungName: "component", count: 2 }], invariants: [{ id: "rank", label: "Rank ordering", status: "holds", detail: "" }], blastRadius: [] } as unknown as CatalogStructure);
    renderPanel();
    await waitFor(() => expect(screen.getByTestId("dashboard-structure")).toHaveAttribute("data-state", "success"));
    expect(screen.getByText("Rank ordering")).toBeInTheDocument();
  });
  it("renders error and empty states", async () => {
    mocked.mockResolvedValueOnce(null);
    renderPanel();
    await waitFor(() => expect(screen.getByTestId("dashboard-structure")).toHaveAttribute("data-state", "empty"));
    mocked.mockRejectedValueOnce(new Error("offline"));
    renderPanel();
    await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("unavailable"));
  });
});
