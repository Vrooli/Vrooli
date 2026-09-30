import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";
import { ReleaseLadderPage } from "./ReleaseLadderPage";
import { renderWithProviders } from "../test-utils";

const api = vi.hoisted(() => ({ fetchReleaseLadder: vi.fn(), setReleaseRank: vi.fn() }));
vi.mock("../api/offers", () => api);

afterEach(() => { cleanup(); vi.clearAllMocks(); });

describe("release ladder states", () => {
  it("renders the ordered ladder and operator rank mutation", async () => {
    api.fetchReleaseLadder.mockResolvedValue({ entries: [{ deliverable: { id: "d1", name: "Console", releaseRank: 1 }, unlockedRamps: [{ name: "desktop" }], unlockedStreams: [{ name: "voice_minutes" }], audiences: [{ name: "developer" }], cumulativeRamps: [{ name: "desktop" }], goalImpacts: [{ goalName: "console-readiness", deliverableName: "Console", projectedPriority: 0 }] }] });
    api.setReleaseRank.mockResolvedValue({});
    renderWithProviders(<ReleaseLadderPage />);
    expect(await screen.findByRole("row", { name: /Console/ })).toBeVisible();
    expect(screen.getByText("desktop")).toBeVisible();
    fireEvent.change(screen.getByLabelText("Deliverable"), { target: { value: "d1" } });
    fireEvent.change(screen.getByLabelText("Release rank"), { target: { value: "2" } });
    fireEvent.click(screen.getByRole("button", { name: "Save rank" }));
    await waitFor(() => expect(api.setReleaseRank).toHaveBeenCalledWith({ nodeId: "d1", releaseRank: 2 }));
  });

  it("shows goals whose projected priority follows the release graph", async () => {
    api.fetchReleaseLadder.mockResolvedValue({ entries: [{ deliverable: { id: "d1", name: "Console", releaseRank: 1 }, goalImpacts: [{ goalName: "console-readiness", goalTitle: "Console readiness", deliverableName: "Console", projectedPriority: 0 }] }] });
    renderWithProviders(<ReleaseLadderPage />);
    await screen.findByRole("row", { name: /Console/ });
    fireEvent.click(screen.getByRole("button", { name: "What moves" }));
    expect(await screen.findByText("console-readiness")).toBeVisible();
    expect(screen.getByText(/projected priority 0/)).toBeVisible();
  });

  it("distinguishes an empty ladder from a request error", async () => {
    api.fetchReleaseLadder.mockResolvedValue({ entries: [] });
    const view = renderWithProviders(<ReleaseLadderPage />);
    expect(await screen.findByText("No deliverables have a release rank yet.")).toBeVisible();
    view.unmount();
    view.queryClient.clear();
    api.fetchReleaseLadder.mockReset();
    api.fetchReleaseLadder.mockRejectedValue(new Error("Offer Desk unavailable"));
    renderWithProviders(<ReleaseLadderPage />);
    await waitFor(() => expect(screen.getByTestId("page-release-ladder")).toHaveAttribute("data-experience-state", "request-error"));
  });

  it("shows goal impacts on each scheduled ladder row", async () => {
    api.fetchReleaseLadder.mockResolvedValue({ entries: [{ deliverable: { id: "d1", name: "Console", releaseRank: 1 }, goalImpacts: [{ goalName: "console-readiness" }] }] });
    renderWithProviders(<ReleaseLadderPage />);
    const row = await screen.findByRole("row", { name: /Console/ });
    expect(row).toHaveTextContent("console-readiness");
  });

  it("shows the deployment-manager readiness projection per deliverable", async () => {
    api.fetchReleaseLadder.mockResolvedValue({ entries: [{ deliverable: { id: "d1", name: "Console", releaseRank: 1 }, readinessGoalExists: true, readinessGoalClosed: true, readinessApprovedCommit: "abc123" }] });
    renderWithProviders(<ReleaseLadderPage />);
    expect(await screen.findByTestId("readiness-d1")).toHaveTextContent("closed (abc123)");
  });

  it("sorts enabling work by urgency and labels zero urgency", async () => {
    api.fetchReleaseLadder.mockResolvedValue({ entries: [{ deliverable: { id: "d1", name: "Console", releaseRank: 1 } }], enabling: [
      { node: { id: "zero", name: "Nothing", finishBar: "OPERATOR_FACING" }, derivedUrgency: 0 },
      { node: { id: "two", name: "Two", finishBar: "OPERATOR_FACING" }, derivedUrgency: 2 },
      { node: { id: "one", name: "One", finishBar: "OPERATOR_FACING" }, derivedUrgency: 1 },
    ] });
    renderWithProviders(<ReleaseLadderPage />);
    const panel = await screen.findByTestId("release-ladder-enabling");
    const text = panel.textContent ?? "";
    expect(text.indexOf("One")).toBeLessThan(text.indexOf("Two"));
    expect(text.indexOf("Two")).toBeLessThan(text.indexOf("Nothing"));
    expect(screen.getByText("Enables nothing scheduled")).toBeVisible();
  });

  it("warns when marketed deliverables are unscheduled", async () => {
    api.fetchReleaseLadder.mockResolvedValue({ entries: [], unscheduled: [{ id: "d2", name: "Unscheduled app" }] });
    renderWithProviders(<ReleaseLadderPage />);
    expect(await screen.findByTestId("release-ladder-unscheduled")).toHaveTextContent("Unscheduled app");
  });
});
