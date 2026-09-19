import { beforeEach, describe, expect, it, vi } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "../../test-utils";

const listTargets = vi.hoisted(() => vi.fn());
const createTarget = vi.hoisted(() => vi.fn());
const listIntakes = vi.hoisted(() => vi.fn());
const recordIntake = vi.hoisted(() => vi.fn());
const evaluateScope = vi.hoisted(() => vi.fn());
const listSupplementSchedules = vi.hoisted(() => vi.fn());
const createSupplementSchedule = vi.hoisted(() => vi.fn());
const updateSupplementSchedule = vi.hoisted(() => vi.fn());
const listRoutineTemplates = vi.hoisted(() => vi.fn());
const createRoutineTemplate = vi.hoisted(() => vi.fn());
const generateRoutineOccurrences = vi.hoisted(() => vi.fn());
vi.mock("../../api/nutrition", () => ({ listTargets, createTarget, listIntakes, recordIntake, evaluateScope }));
vi.mock("../../api/supplement", () => ({ listSupplementSchedules, createSupplementSchedule, updateSupplementSchedule }));
vi.mock("../../api/routine", () => ({ listRoutineTemplates, createRoutineTemplate, generateRoutineOccurrences }));
vi.mock("../../api/workspace", () => ({ ensureWorkspace: vi.fn().mockResolvedValue({ id: "w1" }) }));
import { NutritionPage } from "./NutritionPage";

describe("NutritionPage", () => {
  beforeEach(() => { vi.clearAllMocks(); listTargets.mockResolvedValue([]); listIntakes.mockResolvedValue([]); listSupplementSchedules.mockResolvedValue([]); listRoutineTemplates.mockResolvedValue([]); evaluateScope.mockResolvedValue({ nutrientId: "protein", known: "0", complete: true, unresolved: [], status: "pass", reason: "complete" }); });
  it("keeps an empty account honest and saves only an entered target", async () => {
    renderWithProviders(<NutritionPage />);
    await waitFor(() => expect(screen.getByText("No targets configured")).toBeInTheDocument());
    expect(screen.getByText(/does not claim full-day coverage/)).toBeInTheDocument();
    await userEvent.type(screen.getByLabelText("Protein minimum (g/day)"), "80");
    createTarget.mockResolvedValue({ id: "t1", revision: 1n, nutrientId: "protein", lower: "80", upper: "unknown", period: "local_day", scope: "planned_day", enforcement: "preferred", provenance: "user_assertion", effectiveFrom: "2026-09-18T00:00:00Z", effectiveTo: "", active: true });
    await userEvent.click(screen.getByRole("button", { name: "Save protein target" }));
    expect(createTarget).toHaveBeenCalledWith(expect.objectContaining({ workspaceId: "w1", nutrientId: "protein", lower: "80", upper: "unknown" }));
    await waitFor(() => expect(screen.getByRole("listitem")).toHaveTextContent("protein"));
  });
  it("records actual intake separately from targets", async () => {
    recordIntake.mockResolvedValue({ id: "e1", date: "2026-09-18", recipeId: "", recipeRevision: 0n, nutrientId: "protein", amount: "41.5", unit: "g", reason: "user-recorded intake", correctionOf: "", recordedAt: "" });
    renderWithProviders(<NutritionPage />);
    await waitFor(() => expect(screen.getByRole("button", { name: "Record intake" })).toBeInTheDocument());
    await userEvent.type(screen.getByLabelText("Protein amount (g)"), "41.5");
    await userEvent.click(screen.getByRole("button", { name: "Record intake" }));
    expect(recordIntake).toHaveBeenCalledWith(expect.objectContaining({ workspaceId: "w1", nutrientId: "protein", amount: "41.5", unit: "g" }));
    await waitFor(() => expect(screen.getByRole("list", { name: "Recorded intake" })).toHaveTextContent("41.5 g"));
  });
  it("shows deterministic evaluation status and completeness for a configured target", async () => {
    listTargets.mockResolvedValue([{ id: "t1", revision: 1n, nutrientId: "protein", lower: "80", upper: "unknown", period: "local_day", scope: "planned_day", enforcement: "preferred", provenance: "user_assertion", effectiveFrom: "2026-09-18T00:00:00Z", effectiveTo: "", active: true }]);
    evaluateScope.mockResolvedValue({ nutrientId: "protein", known: "41.5", complete: false, unresolved: ["unresolved"], status: "unknown", reason: "partial" });
    renderWithProviders(<NutritionPage />);
    await waitFor(() => expect(screen.getByText(/Recorded so far: 41.5 g · unknown/)).toBeInTheDocument());
    expect(screen.getByText(/partial evidence/)).toBeInTheDocument();
  });
  it("refreshes a target after recording intake and keeps evaluation failures explicit", async () => {
    listTargets.mockResolvedValue([{ id: "t1", revision: 1n, nutrientId: "protein", lower: "80", upper: "unknown", period: "local_day", scope: "planned_day", enforcement: "preferred", provenance: "user_assertion", effectiveFrom: "2026-09-18T00:00:00Z", effectiveTo: "", active: true }]);
    evaluateScope.mockResolvedValueOnce({ nutrientId: "protein", known: "0", complete: true, unresolved: [], status: "fail", reason: "below" }).mockRejectedValueOnce(new Error("offline"));
    recordIntake.mockResolvedValue({ id: "e2", date: "2026-09-18", recipeId: "", recipeRevision: 0n, nutrientId: "protein", amount: "41.5", unit: "g", reason: "user-recorded intake", correctionOf: "", recordedAt: "" });
    renderWithProviders(<NutritionPage />);
    await waitFor(() => expect(screen.getByText(/Recorded so far: 0 g · fail/)).toBeInTheDocument());
    await userEvent.type(screen.getByLabelText("Protein amount (g)"), "41.5");
    await userEvent.click(screen.getByRole("button", { name: "Record intake" }));
    await waitFor(() => expect(screen.getByRole("list", { name: "Recorded intake" })).toHaveTextContent("41.5 g"));
    expect(evaluateScope).toHaveBeenCalledTimes(2);
  });
  it("surfaces a load failure instead of rendering an empty nutrition state", async () => {
    listTargets.mockRejectedValue(new Error("offline"));
    renderWithProviders(<NutritionPage />);
    await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("offline"));
  });
  it("creates and pauses a confirmed supplement schedule without rewriting intake", async () => {
    createSupplementSchedule.mockResolvedValue({ id: "s1", revision: 1n, productRevisionId: "omega-1", dose: "1", doseUnit: "capsule", weekdays: [0, 1, 2, 3, 4, 5, 6], startDate: "2026-09-18", endDate: "", paused: false, confirmed: true, createdAt: "" });
    updateSupplementSchedule.mockResolvedValue({ id: "s1", revision: 2n, productRevisionId: "omega-1", dose: "1", doseUnit: "capsule", weekdays: [0, 1, 2, 3, 4, 5, 6], startDate: "2026-09-18", endDate: "", paused: true, confirmed: true, createdAt: "" });
    renderWithProviders(<NutritionPage />);
    await waitFor(() => expect(screen.getByRole("button", { name: "Save confirmed schedule" })).toBeInTheDocument());
    await userEvent.type(screen.getByLabelText("Product revision ID"), "omega-1");
    await userEvent.type(screen.getByLabelText("Dose"), "1");
    await userEvent.click(screen.getByRole("button", { name: "Save confirmed schedule" }));
    await waitFor(() => expect(screen.getByRole("button", { name: "Pause" })).toBeInTheDocument());
    await userEvent.click(screen.getByRole("button", { name: "Pause" }));
    expect(updateSupplementSchedule).toHaveBeenCalledWith(expect.objectContaining({ id: "s1", expectedRevision: 1n, paused: true }));
  });
  it("creates an open routine slot and previews generated occurrences", async () => {
    createRoutineTemplate.mockResolvedValue({ id: "r1", revision: 1n, slotName: "breakfast", recipeId: "", quantity: "1", weekdays: [0, 1, 2, 3, 4, 5, 6], startDate: "2026-09-18", endDate: "", mode: "open", active: true });
    generateRoutineOccurrences.mockResolvedValue([{ date: "2026-09-18", slotName: "breakfast", recipeId: "", quantity: "1", templateId: "r1", templateRevision: 1n, mode: "open" }]);
    renderWithProviders(<NutritionPage />);
    await waitFor(() => expect(screen.getByRole("button", { name: "Save routine slot" })).toBeInTheDocument());
    await userEvent.click(screen.getByRole("button", { name: "Save routine slot" }));
    await waitFor(() => expect(screen.getByRole("list", { name: "Routine templates" })).toHaveTextContent("open slot"));
    await userEvent.click(screen.getByRole("button", { name: "Preview today" }));
    await waitFor(() => expect(screen.getByRole("list", { name: "Routine occurrences" })).toHaveTextContent("breakfast"));
  });
  it("surfaces routine preview failures without claiming a result", async () => {
    generateRoutineOccurrences.mockRejectedValue(new Error("preview unavailable"));
    renderWithProviders(<NutritionPage />);
    await waitFor(() => expect(screen.getByRole("button", { name: "Preview today" })).toBeInTheDocument());
    await userEvent.click(screen.getByRole("button", { name: "Preview today" }));
    await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("preview unavailable"));
    expect(screen.queryByRole("list", { name: "Routine occurrences" })).not.toBeInTheDocument();
  });
  it("surfaces routine save failures without adding a phantom slot", async () => {
    createRoutineTemplate.mockRejectedValue(new Error("save unavailable"));
    renderWithProviders(<NutritionPage />);
    await waitFor(() => expect(screen.getByRole("button", { name: "Save routine slot" })).toBeInTheDocument());
    await userEvent.click(screen.getByRole("button", { name: "Save routine slot" }));
    await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("save unavailable"));
    expect(screen.queryByRole("list", { name: "Routine templates" })).not.toBeInTheDocument();
  });
  it("saves a fixed routine slot when a recipe is supplied", async () => {
    createRoutineTemplate.mockResolvedValue({ id: "r2", revision: 1n, slotName: "dinner", recipeId: "recipe-1", quantity: "2", weekdays: [0, 1, 2, 3, 4, 5, 6], startDate: "2026-09-18", endDate: "", mode: "fixed", active: true });
    renderWithProviders(<NutritionPage />);
    await waitFor(() => expect(screen.getByRole("button", { name: "Save routine slot" })).toBeInTheDocument());
    await userEvent.clear(screen.getByLabelText("Meal slot"));
    await userEvent.type(screen.getByLabelText("Meal slot"), "dinner");
    await userEvent.type(screen.getByLabelText("Recipe ID (optional for an open slot)"), "recipe-1");
    await userEvent.clear(screen.getByLabelText("Quantity"));
    await userEvent.type(screen.getByLabelText("Quantity"), "2");
    await userEvent.click(screen.getByRole("button", { name: "Save routine slot" }));
    await waitFor(() => expect(screen.getByRole("list", { name: "Routine templates" })).toHaveTextContent("recipe-1"));
    expect(createRoutineTemplate).toHaveBeenCalledWith(expect.objectContaining({ recipeId: "recipe-1", mode: "fixed", quantity: "2" }));
  });
});
