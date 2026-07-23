import { describe, expect, it, vi } from "vitest";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { createTestQueryClient, renderWithProviders } from "../../test-utils";

const mockQueue = vi.fn();
const mockGet = vi.fn();

vi.mock("../../services", () => ({
  backlogService: { queue: (...args: unknown[]) => mockQueue(...args) },
}));

vi.mock("../../lib/api-client", () => ({
  defaultApiClient: { get: (...args: unknown[]) => mockGet(...args) },
  isApiError: () => false,
}));

import { RunSheet } from "./run-sheet";

const readyPreview = {
  taskId: "", runId: "", baseUrl: "", created: "", dryRun: true,
  queued: false, message: "ready", blockingReasons: [], pendingDecisions: 0, pendingSuggestions: 0,
};

describe("RunSheet", () => {
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
});
