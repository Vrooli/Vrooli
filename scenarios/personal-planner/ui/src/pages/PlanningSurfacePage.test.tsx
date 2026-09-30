import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { fetchWorkItems } from "../api/work";
import { renderWithProviders } from "../test-utils";
import { PlanningSurfacePage } from "./PlanningSurfacePage";

vi.mock("../api/work", () => ({ fetchWorkItems: vi.fn() }));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("PlanningSurfacePage", () => {
  it.each([
    ["plan", "Plan", "Add work from the CLI or the Today surface to begin planning."],
    ["goals", "Goals", "Goals are the next domain to connect to this workspace."],
    ["focus", "Focus", "Choose a work item from Today to start a focus session."],
    ["review", "Review", "There is not enough completed history to review yet."],
  ] as const)("renders the honest empty state for %s", async (surface, title, empty) => {
    vi.mocked(fetchWorkItems).mockResolvedValue([]);

    renderWithProviders(<PlanningSurfacePage surface={surface} />);

    expect(await screen.findByRole("heading", { name: title })).toBeInTheDocument();
    expect(screen.getByText(empty)).toBeInTheDocument();
  });

  it("announces loading and errors", async () => {
    let resolve!: (items: never[]) => void;
    vi.mocked(fetchWorkItems).mockReturnValue(new Promise((res) => { resolve = res; }));
    renderWithProviders(<PlanningSurfacePage surface="plan" />);
    expect(screen.getByRole("status")).toHaveTextContent("Loading work items");
    resolve([]);

    vi.mocked(fetchWorkItems).mockRejectedValueOnce(new Error("offline"));
    renderWithProviders(<PlanningSurfacePage surface="focus" />);
    expect(await screen.findByRole("alert")).toHaveTextContent("Work data is unavailable");
  });

  it("renders connected work with its source and remaining time", async () => {
    vi.mocked(fetchWorkItems).mockResolvedValue([
      { id: "work-1", title: "Draft the launch story", description: "Keep it useful.", remainingMinutes: 45, sourceLabel: "Cadence" },
    ] as never);

    renderWithProviders(<PlanningSurfacePage surface="plan" />);

    expect(await screen.findByRole("heading", { name: "Draft the launch story" })).toBeInTheDocument();
    expect(screen.getByText("Cadence")).toBeInTheDocument();
    expect(screen.getByText("45 min")).toBeInTheDocument();
    expect(screen.getByText("Keep it useful.")).toBeInTheDocument();
  });
});
