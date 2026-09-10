import { beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { createTestQueryClient, renderWithProviders } from "../../test-utils";

const mockQueue = vi.fn();
const mockGet = vi.fn();

vi.mock("../../services", () => ({
  backlogService: { queue: (...args: unknown[]) => mockQueue(...args) },
}));

vi.mock("../../lib/api-client", () => ({
  API_BASE: "http://localhost:3000/api/v1",
  defaultApiClient: { get: (...args: unknown[]) => mockGet(...args) },
  isApiError: () => false,
}));

import { RunSheet } from "./run-sheet";

const readyPreview = {
  taskId: "", runId: "", baseUrl: "", created: "", dryRun: true,
  queued: false, message: "ready", blockingReasons: [], pendingDecisions: 0, pendingSuggestions: 0,
};

describe("RunSheet", () => {
  beforeEach(() => {
    mockGet.mockReset();
    mockQueue.mockReset();
  });

  it("previews on open and only queues after the explicit Run action", async () => {
    mockGet.mockResolvedValue({ items: [{
      id: "phased-plan-drain", workflow_key: "swarm-manager/phased-plan-drain",
      display_name: "Phased plan drain", description: "Bounded slices", when_to_use: "For ready plans.", cost_band: "medium", cost_estimate: 3.25,
    }] });
    mockQueue.mockResolvedValue(readyPreview);
    const onClose = vi.fn();

    renderWithProviders(<RunSheet isOpen onClose={onClose} target={{ kind: "execute", name: "item-a", title: "Item A" }} />, { queryClient: createTestQueryClient() });

    await waitFor(() => expect(mockQueue).toHaveBeenCalledWith("execute", "item-a", { mode: "yolo", confirm: false }));
    expect(mockQueue).toHaveBeenCalledTimes(1);
    expect(screen.getByRole("button", { name: "Run" })).toBeEnabled();

    fireEvent.click(screen.getByRole("button", { name: "Run" }));

    await waitFor(() => expect(mockQueue).toHaveBeenLastCalledWith("execute", "item-a", {
      mode: "yolo", startedBy: "swarm-manager-ui", confirm: true, force: false,
      strategy: "phased-plan-drain", maxSlices: 6,
    }));
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("uses the item's saved adaptive strategy when the operator runs it", async () => {
    mockGet.mockResolvedValue({ items: [
      { id: "phased-plan-drain", display_name: "Phased plan drain", cost_estimate: 3.25 },
      { id: "adaptive-improvement", display_name: "Adaptive improvement", cost_estimate: 3.25 },
    ] });
    mockQueue.mockResolvedValue({ ...readyPreview, item: { executionStrategy: "adaptive-improvement" } });

    renderWithProviders(<RunSheet isOpen onClose={vi.fn()} target={{ kind: "execute", name: "develop-audio" }} />, { queryClient: createTestQueryClient() });

    await waitFor(() => expect(screen.getByRole("radio", { name: /Adaptive improvement/ })).toBeChecked());
    fireEvent.click(screen.getByRole("button", { name: "Run" }));
    await waitFor(() => expect(mockQueue).toHaveBeenLastCalledWith("execute", "develop-audio", expect.objectContaining({ confirm: true, strategy: "adaptive-improvement" })));
  });

  it("replaces the previous item's selection with the newly reviewed item's strategy", async () => {
    mockGet.mockResolvedValue({ items: [
      { id: "phased-plan-drain", display_name: "Phased plan drain", cost_estimate: 3.25 },
      { id: "adaptive-improvement", display_name: "Adaptive improvement", cost_estimate: 3.25 },
    ] });
    mockQueue.mockImplementation(async (_kind, name) => ({
      ...readyPreview,
      item: { executionStrategy: name === "develop-audio" ? "adaptive-improvement" : "phased-plan-drain" },
    }));
    const onClose = vi.fn();
    const rendered = renderWithProviders(<RunSheet isOpen onClose={onClose} target={{ kind: "execute", name: "develop-audio" }} />, { queryClient: createTestQueryClient() });
    await waitFor(() => expect(screen.getByRole("radio", { name: /Adaptive improvement/ })).toBeChecked());

    rendered.rerender(<RunSheet isOpen onClose={onClose} target={{ kind: "execute", name: "ordered-repair" }} />);
    await waitFor(() => expect(screen.getByRole("radio", { name: /Phased plan drain/ })).toBeChecked());
    fireEvent.click(screen.getByRole("button", { name: "Run" }));
    await waitFor(() => expect(mockQueue).toHaveBeenLastCalledWith("execute", "ordered-repair", expect.objectContaining({ confirm: true, strategy: "phased-plan-drain" })));
  });

  it("displays reviewed limits and allows narrowing the saved slice ceiling", async () => {
    mockGet.mockResolvedValue({ items: [{ id: "adaptive-improvement", display_name: "Adaptive improvement", cost_estimate: 3.25 }] });
    mockQueue.mockResolvedValue({ ...readyPreview, item: { executionStrategy: "adaptive-improvement", executionLimits: {
      maxSlices: 128, maxTokens: 2000000, maxWallSeconds: 604800, maxTurns: 2400, maxChargeMicroUsd: 250000000, maxChildren: 512, maxNodeAttempts: 512, maxRetries: 128,
    } } });
    renderWithProviders(<RunSheet isOpen onClose={vi.fn()} target={{ kind: "execute", name: "develop-audio" }} />, { queryClient: createTestQueryClient() });
    const slider = await screen.findByRole("slider", { name: /Maximum slices/ });
    await waitFor(() => expect(slider).toHaveValue("128"));
    expect(slider).toHaveAttribute("max", "128");
    expect(screen.getByText("2,000,000")).toBeVisible();
    expect(screen.getByText("$250.00")).toBeVisible();
    fireEvent.change(slider, { target: { value: "64" } });
    fireEvent.click(screen.getByRole("button", { name: "Run" }));
    await waitFor(() => expect(mockQueue).toHaveBeenLastCalledWith("execute", "develop-audio", expect.objectContaining({ confirm: true, maxSlices: 64 })));
  });

  it("previews every bulk item and keeps each item's saved execution settings", async () => {
    mockGet.mockResolvedValue({ items: [{ id: "adaptive-improvement", display_name: "Adaptive improvement", cost_estimate: 3.25 }] });
    mockQueue.mockResolvedValue(readyPreview);
    renderWithProviders(<RunSheet isOpen onClose={vi.fn()} targets={[{ kind: "execute", name: "audio" }, { kind: "execute", name: "ttd" }]} />, { queryClient: createTestQueryClient() });
    await waitFor(() => expect(mockQueue).toHaveBeenCalledWith("execute", "ttd", { mode: "yolo", confirm: false }));
    fireEvent.click(screen.getByRole("button", { name: "Run" }));
    await waitFor(() => expect(mockQueue).toHaveBeenCalledTimes(4));
    for (const name of ["audio", "ttd"]) expect(mockQueue).toHaveBeenCalledWith("execute", name, {
      mode: "yolo", confirm: true, force: false, startedBy: "swarm-manager-ui",
    });
  });
});
