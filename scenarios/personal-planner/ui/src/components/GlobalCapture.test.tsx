import { afterEach, describe, expect, it, vi } from "vitest";
import userEvent from "@testing-library/user-event";
import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";
import { renderWithProviders } from "../test-utils";
import { localDateForSource, parseNaturalCapture, startMinutesForTime } from "./GlobalCapture";

const { createWorkItem } = vi.hoisted(() => ({ createWorkItem: vi.fn().mockResolvedValue({ id: "captured" }) }));
vi.mock("../api/work", () => ({ createWorkItem }));
const { previewAllocation, applyAllocationProposal } = vi.hoisted(() => ({ previewAllocation: vi.fn().mockResolvedValue({ id: "proposal", state: "feasible", reason: "", baseRevision: 1n, startMinutes: 780, durationMinutes: 60 }), applyAllocationProposal: vi.fn().mockResolvedValue({ id: "allocation" }) }));
vi.mock("../api/calendar", () => ({ previewAllocation, applyAllocationProposal }));
import { GlobalCapture } from "./GlobalCapture";

describe("parseNaturalCapture", () => {
  afterEach(() => {
    cleanup();
    createWorkItem.mockReset().mockResolvedValue({ id: "captured" });
    previewAllocation.mockReset().mockResolvedValue({ id: "proposal", state: "feasible", reason: "", baseRevision: 1n, startMinutes: 780, durationMinutes: 60 });
    applyAllocationProposal.mockReset().mockResolvedValue({ id: "allocation" });
    vi.unstubAllGlobals();
  });

  it("turns a natural-language capture into a useful work item", () => {
    expect(parseNaturalCapture("lunch with Sam tue 1pm 1h")).toMatchObject({ title: "lunch with Sam", minutes: 60, source: "tue", time: "1pm" });
  });

  it("keeps free-form captures intact when no schedule metadata is supplied", () => {
    expect(parseNaturalCapture("think about the launch")).toMatchObject({ title: "think about the launch", minutes: 0, source: "quick capture", time: "" });
  });

  it("preserves minute estimates without treating them as hours", () => {
    expect(parseNaturalCapture("review the brief 30 mins")).toMatchObject({ title: "review the brief", minutes: 30 });
  });

  it("normalizes clock phrases and keeps unscheduled captures on today", () => {
    expect(startMinutesForTime("12:30am")).toBe(30);
    expect(startMinutesForTime("12:30pm")).toBe(750);
    expect(startMinutesForTime("09:15")).toBe(555);
    expect(startMinutesForTime("soon")).toBe(540);
    expect(localDateForSource("quick capture")).toMatch(/^\d{4}-\d{2}-\d{2}$/);
  });

  it("opens from the global shortcut and saves parsed capture metadata", async () => {
    const user = userEvent.setup();
    renderWithProviders(<GlobalCapture />);
    await user.keyboard("{Meta>}k{/Meta}");
    const input = screen.getByPlaceholderText("lunch with Sam tue 1pm 1h");
    await user.type(input, "lunch with Sam tue 1pm 1h");
    expect(screen.getByText(/lunch with Sam · 60 min/)).toBeInTheDocument();
    const captureButtons = screen.getAllByRole("button", { name: "Capture task" });
    const captureButton = captureButtons.at(-1);
    expect(captureButton).toBeDefined();
    if (!captureButton) throw new Error("capture button should be present");
    await user.click(captureButton);
    await waitFor(() => expect(createWorkItem).toHaveBeenCalledWith(expect.objectContaining({ title: "lunch with Sam", remainingMinutes: 60 })));
  });

  it("opens from the visible trigger and closes without saving", async () => {
    const user = userEvent.setup();
    renderWithProviders(<GlobalCapture />);
    await user.click(screen.getByRole("button", { name: /Capture ⌘K/ }));
    expect(screen.getByPlaceholderText("lunch with Sam tue 1pm 1h")).toBeInTheDocument();
    const closeButtons = screen.getAllByRole("button", { name: "Close capture" });
    const closeButton = closeButtons.at(-1);
    expect(closeButton).toBeDefined();
    if (!closeButton) throw new Error("close button should be present");
    await user.click(closeButton);
    await waitFor(() => expect(screen.queryByPlaceholderText("lunch with Sam tue 1pm 1h")).not.toBeInTheDocument());
  });

  it("opens from the mobile floating action button", async () => {
    const user = userEvent.setup();
    vi.stubGlobal("matchMedia", () => ({ matches: true, addEventListener: vi.fn(), removeEventListener: vi.fn() }));
    renderWithProviders(<GlobalCapture />);
    await user.click(screen.getByRole("button", { name: "Capture task" }));
    expect(screen.getByPlaceholderText("lunch with Sam tue 1pm 1h")).toBeInTheDocument();
  });

  it("saves a free-form capture without inventing scheduling metadata", async () => {
    const user = userEvent.setup();
    renderWithProviders(<GlobalCapture />);
    await user.keyboard("{Meta>}k{/Meta}");
    fireEvent.change(screen.getByPlaceholderText("lunch with Sam tue 1pm 1h"), { target: { value: "think about the launch" } });
    const submitButton = screen.getAllByRole("button", { name: "Capture task" }).find((button) => button.getAttribute("type") === "submit");
    expect(submitButton).toBeDefined();
    if (!submitButton) throw new Error("capture submit button should be present");
    await user.click(submitButton);
    await waitFor(() => expect(createWorkItem).toHaveBeenCalledWith(expect.objectContaining({ sourceLabel: "quick capture", remainingMinutes: 0 })));
  });
});
