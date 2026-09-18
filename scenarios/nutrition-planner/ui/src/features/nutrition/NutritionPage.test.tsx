import { beforeEach, describe, expect, it, vi } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "../../test-utils";

const listTargets = vi.hoisted(() => vi.fn());
const createTarget = vi.hoisted(() => vi.fn());
const listIntakes = vi.hoisted(() => vi.fn());
const recordIntake = vi.hoisted(() => vi.fn());
vi.mock("../../api/nutrition", () => ({ listTargets, createTarget, listIntakes, recordIntake }));
vi.mock("../../api/workspace", () => ({ ensureWorkspace: vi.fn().mockResolvedValue({ id: "w1" }) }));
import { NutritionPage } from "./NutritionPage";

describe("NutritionPage", () => {
  beforeEach(() => { vi.clearAllMocks(); listTargets.mockResolvedValue([]); listIntakes.mockResolvedValue([]); });
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
});
