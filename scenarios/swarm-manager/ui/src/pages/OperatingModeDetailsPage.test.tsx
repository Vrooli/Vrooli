import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, expect, it, vi, beforeEach } from "vitest";
import { OperatingModeDetailsPage } from "./OperatingModeDetailsPage";

const getModeMock = vi.fn();
const updateModeMock = vi.fn();
const navigateMock = vi.fn();

vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual<typeof import("react-router-dom")>("react-router-dom");
  return { ...actual, useNavigate: () => navigateMock };
});

vi.mock("../services", () => ({
  initiativeModeService: {
    getMode: (mode: string) => getModeMock(mode),
    updateMode: (mode: string, args: unknown) => updateModeMock(mode, args),
  },
}));

vi.mock("../app/shell/AppShellContext", () => ({
  useAppShell: () => ({ openSidebar: () => {}, closeSidebar: () => {} }),
}));

function renderPage(mode = "holistic-loop") {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[`/operating-modes/${mode}`]}>
        <Routes>
          <Route path="/operating-modes/:mode" element={<OperatingModeDetailsPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

const SAMPLE_DETAIL = {
  entry: {
    mode: "holistic-loop",
    label: "Holistic Loop",
    description: "Investigate→plan→execute cycles",
    usageCount: 2,
    scopeKind: "initiative",
    runStrategy: "operator_gated_loop",
    workspaceTabId: "operating-mode",
    capabilities: {
      supportsPhases: true,
      canStartPhases: true,
      canCompleteItems: false,
      canApplyBacklogSyncProposals: false,
      requiresAcceptanceCriteria: false,
      supportsArtifacts: true,
      supportsHandoffs: false,
      usesItemExecutionFlow: false,
    },
    default: false,
    switchable: true,
    supportsPhases: true,
    phases: [
      { phase: "investigate", profileKey: "swarm-manager/deep-work", writesRepo: false },
    ],
  },
  linkedInitiatives: [
    { name: "init-a", title: "Initiative A", status: "active", updated: "2026-04-30" },
    { name: "init-b", title: "Initiative B", status: "active", updated: "2026-04-29" },
  ],
};

describe("OperatingModeDetailsPage", () => {
  beforeEach(() => {
    getModeMock.mockReset();
    updateModeMock.mockReset();
    navigateMock.mockReset();
  });

  it("renders mode metadata and linked initiatives", async () => {
    getModeMock.mockResolvedValue(SAMPLE_DETAIL);
    renderPage();

    expect(await screen.findByText("Holistic Loop")).toBeInTheDocument();
    expect(screen.getByText("Investigate→plan→execute cycles")).toBeInTheDocument();
    expect(screen.getByText("Initiative A")).toBeInTheDocument();
    expect(screen.getByText("Initiative B")).toBeInTheDocument();
    // Phase row
    expect(screen.getByText("investigate")).toBeInTheDocument();
  });

  it("navigates to the linked initiative when clicked", async () => {
    getModeMock.mockResolvedValue(SAMPLE_DETAIL);
    renderPage();

    const links = await screen.findAllByTestId("operating-mode-linked-initiative");
    const first = links[0];
    if (!first) throw new Error("expected at least one linked initiative");
    fireEvent.click(first);
    expect(navigateMock).toHaveBeenCalledWith("/initiatives/init-a");
  });

  it("saves edits via updateMode", async () => {
    getModeMock.mockResolvedValue(SAMPLE_DETAIL);
    updateModeMock.mockResolvedValue({
      ...SAMPLE_DETAIL,
      entry: { ...SAMPLE_DETAIL.entry, label: "Renamed Loop", description: "New text" },
    });
    renderPage();

    const editButton = await screen.findByRole("button", { name: /edit/i });
    fireEvent.click(editButton);

    const labelInput = screen.getByTestId("operating-mode-label-input") as HTMLInputElement;
    const descInput = screen.getByTestId("operating-mode-description-input") as HTMLTextAreaElement;
    fireEvent.change(labelInput, { target: { value: "Renamed Loop" } });
    fireEvent.change(descInput, { target: { value: "New text" } });

    fireEvent.click(screen.getByRole("button", { name: /save/i }));

    await waitFor(() => {
      expect(updateModeMock).toHaveBeenCalledWith("holistic-loop", {
        label: "Renamed Loop",
        description: "New text",
      });
    });
  });

  it("disables save when label is blank", async () => {
    getModeMock.mockResolvedValue(SAMPLE_DETAIL);
    renderPage();

    fireEvent.click(await screen.findByRole("button", { name: /edit/i }));
    const labelInput = screen.getByTestId("operating-mode-label-input") as HTMLInputElement;
    fireEvent.change(labelInput, { target: { value: "   " } });

    expect(screen.getByRole("button", { name: /save/i })).toBeDisabled();
  });
});
