import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

const listInventoryBatches = vi.hoisted(() => vi.fn());
const listInventoryEvents = vi.hoisted(() => vi.fn());
const listRecipes = vi.hoisted(() => vi.fn());
const getProfile = vi.hoisted(() => vi.fn());
const applyProfile = vi.hoisted(() => vi.fn());
vi.mock("../../api/inventory", () => ({ listInventoryBatches, listInventoryEvents }));
vi.mock("../../api/recipes", () => ({ listRecipes }));
vi.mock("../../api/profile", () => ({ getProfile, applyProfile }));
vi.mock("../../api/workspace", () => ({ ensureWorkspace: vi.fn().mockResolvedValue({ id: "w1" }) }));

import { KitchenPage } from "./KitchenPage";

describe("KitchenPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    listInventoryBatches.mockResolvedValue([]);
    listInventoryEvents.mockResolvedValue([]);
    listRecipes.mockResolvedValue([]);
    const profile = { workspaceId: "w1", revision: 1n, preset: "vegan", presetVersion: 1n, activeRules: ["vegan"], excludedGroups: ["meat"], allergies: ["peanut"], appliances: ["stove"], costWeight: 0.2, effortWeight: 0.4, varietyWeight: 0.4, draftJson: "" };
    getProfile.mockResolvedValue(profile);
    applyProfile.mockImplementation(async (input) => ({ profile: { ...profile, ...input }, matchingMeals: 0n, needsReviewMeals: 0n, excludedMeals: 0n }));
  });

  it("shows saved equipment and exact inventory evidence without deriving a raw stock total", async () => {
    listInventoryEvents.mockResolvedValue([{ id: "p1", kind: "purchase", itemId: "tofu", amount: "400", unit: "g", createdAt: "2026-10-05T12:00:00Z" }]);
    renderWithProviders(<KitchenPage />);
    await waitFor(() => expect(screen.getByRole("heading", { name: "Recent activity" })).toBeInTheDocument());
    expect(screen.getByText("400 g")).toBeInTheDocument();
    expect(screen.getByRole("list", { name: "Recent inventory activity" })).toHaveTextContent(/purchase/);
    expect(screen.queryByText("400 g available")).not.toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Manage shopping and receipts" })).toHaveAttribute("href", "/groceries");
    await userEvent.click(screen.getByRole("tab", { name: "Equipment" }));
    expect(screen.getByRole("button", { name: "Remove Cooktop" })).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByRole("button", { name: "Select Oven" })).toHaveAttribute("aria-pressed", "false");
  });

  it("keeps inventory unknown when no evidence exists", async () => {
    getProfile.mockResolvedValue(undefined);
    renderWithProviders(<KitchenPage />);
    await waitFor(() => expect(screen.getByText(/Raw ingredient amounts stay unknown until evidence is added/)).toBeInTheDocument());
    expect(screen.getByText("No prepared portions yet")).toBeInTheDocument();
  });

  it("saves an appliance change immediately while preserving the rest of the profile", async () => {
    renderWithProviders(<KitchenPage />);
    await userEvent.click(await screen.findByRole("tab", { name: "Equipment" }));
    await userEvent.click(screen.getByRole("button", { name: "Select Oven" }));
    await waitFor(() => expect(applyProfile).toHaveBeenCalledWith({
      workspaceId: "w1", preset: "vegan", excludedGroups: ["meat"], allergies: ["peanut"], appliances: ["stove", "oven"], costWeight: 0.2, effortWeight: 0.4, varietyWeight: 0.4,
    }));
    expect(screen.getByRole("button", { name: "Remove Oven" })).toHaveAttribute("aria-pressed", "true");
  });

  it("reports categories the saved profile cannot represent without inventing selections", async () => {
    renderWithProviders(<KitchenPage />);
    await userEvent.click(await screen.findByRole("tab", { name: "Equipment" }));
    await userEvent.click(screen.getByRole("tab", { name: "Cookware" }));
    expect(screen.getByText("Cookware are not in the saved profile")).toBeInTheDocument();
    expect(screen.getByText(/No cookware or tool selection is inferred/)).toBeInTheDocument();
    await userEvent.keyboard("{ArrowRight}");
    expect(screen.getByRole("tab", { name: "Tools" })).toHaveAttribute("aria-selected", "true");
  });

  it("rolls an optimistic equipment choice back when profile saving fails", async () => {
    applyProfile.mockRejectedValue(new Error("offline"));
    renderWithProviders(<KitchenPage />);
    await userEvent.click(await screen.findByRole("tab", { name: "Equipment" }));
    await userEvent.click(screen.getByRole("button", { name: "Select Oven" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("offline");
    expect(screen.getByRole("button", { name: "Select Oven" })).toHaveAttribute("aria-pressed", "false");
  });

  it("shows saved preferences without turning missing fields into targets", async () => {
    renderWithProviders(<KitchenPage />);
    await userEvent.click(await screen.findByRole("tab", { name: "Preferences" }));
    expect(screen.getByRole("heading", { name: "Your preferences" })).toBeInTheDocument();
    expect(screen.getAllByText("vegan")).toHaveLength(2);
    expect(screen.getByText("Cost 0.20 · Effort 0.40 · Variety 0.40")).toBeInTheDocument();
    expect(screen.getByText(/Nutrition targets, household servings, schedules and shopping preferences are not represented/)).toBeInTheDocument();
  });

  it("keeps keyboard focus and selection aligned across Kitchen and equipment tabs", async () => {
    renderWithProviders(<KitchenPage />);
    const onHand = await screen.findByRole("tab", { name: "On hand" });
    fireEvent.keyDown(onHand, { key: "End" });
    const preferences = screen.getByRole("tab", { name: "Preferences" });
    expect(preferences).toHaveFocus();
    expect(preferences).toHaveAttribute("aria-selected", "true");
    fireEvent.keyDown(preferences, { key: "ArrowLeft" });
    const equipment = screen.getByRole("tab", { name: "Equipment" });
    expect(equipment).toHaveFocus();
    fireEvent.keyDown(equipment, { key: "Home" });
    expect(onHand).toHaveFocus();
    fireEvent.keyDown(onHand, { key: "x" });
    expect(onHand).toHaveFocus();

    await userEvent.click(equipment);
    const appliances = screen.getByRole("tab", { name: "Appliances" });
    fireEvent.keyDown(appliances, { key: "ArrowDown" });
    const cookware = screen.getByRole("tab", { name: "Cookware" });
    expect(cookware).toHaveFocus();
    fireEvent.keyDown(cookware, { key: "End" });
    const tools = screen.getByRole("tab", { name: "Tools" });
    expect(tools).toHaveFocus();
    fireEvent.keyDown(tools, { key: "ArrowUp" });
    expect(cookware).toHaveFocus();
    fireEvent.keyDown(cookware, { key: "Home" });
    expect(appliances).toHaveFocus();
  });

  it("keeps unconfigured equipment and preference values explicit", async () => {
    getProfile.mockResolvedValue(undefined);
    renderWithProviders(<KitchenPage />);
    await userEvent.click(await screen.findByRole("tab", { name: "Equipment" }));
    expect(screen.getByText(/No saved profile exists yet/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Select Oven" })).toBeDisabled();
    await userEvent.click(screen.getByRole("tab", { name: "Cookware" }));
    expect(screen.getByText("Cookware are not in the saved profile")).toBeInTheDocument();
    await userEvent.click(screen.getByRole("tab", { name: "Preferences" }));
    expect(screen.getByText("No saved profile yet")).toBeInTheDocument();
  });

  it("labels saved preference fields when the profile records no values", async () => {
    const profile = { workspaceId: "w1", revision: 1n, preset: "", presetVersion: 1n, activeRules: [], excludedGroups: [], allergies: [], appliances: [], costWeight: 0.2, effortWeight: 0.4, varietyWeight: 0.4, draftJson: "" };
    getProfile.mockResolvedValue(profile);
    renderWithProviders(<KitchenPage />);
    await userEvent.click(await screen.findByRole("tab", { name: "Preferences" }));
    expect(screen.getByText("Not configured")).toBeInTheDocument();
    expect(screen.getByText("No allergies recorded")).toBeInTheDocument();
    expect(screen.getByText("No excluded groups recorded")).toBeInTheDocument();
    expect(screen.getByText("No active rules recorded")).toBeInTheDocument();
  });

  it("shows one measured batch and falls back to event facts without inferring amounts", async () => {
    listInventoryBatches.mockResolvedValue([{ id: "b1", recipeId: "missing-recipe", recipeRevision: 3n, yieldAmount: "4", availableAmount: "2", unit: "bowls" }]);
    listInventoryEvents.mockResolvedValue([
      { id: "e1", kind: "prepare", recipeId: "soup", itemId: "", amount: "", unit: "", createdAt: "2026-10-05T12:00:00Z" },
      { id: "e2", kind: "purchase", recipeId: "", itemId: "", amount: "", unit: "", createdAt: "2026-10-04T12:00:00Z" },
    ]);
    renderWithProviders(<KitchenPage />);
    await waitFor(() => expect(screen.getByRole("heading", { name: "On hand" })).toBeInTheDocument());
    expect(screen.getByText("1 batch")).toBeInTheDocument();
    expect(screen.getByText("Saved recipe")).toBeInTheDocument();
    const activity = screen.getByRole("list", { name: "Recent inventory activity" });
    expect(activity).toHaveTextContent("soup");
    expect(activity).toHaveTextContent("Prepared food");
    expect(activity).toHaveTextContent("Amount not recorded");
  });

  it("reports Kitchen loading errors and rolls back a non-Error appliance save failure", async () => {
    listRecipes.mockRejectedValueOnce("offline");
    const first = renderWithProviders(<KitchenPage />);
    expect(await screen.findByRole("alert")).toHaveTextContent("Unable to load Kitchen inventory.");
    expect(screen.queryByRole("tablist", { name: "Kitchen sections" })).not.toBeInTheDocument();

    first.unmount();
    listRecipes.mockResolvedValue([]);
    applyProfile.mockRejectedValue("offline");
    renderWithProviders(<KitchenPage />);
    await userEvent.click(await screen.findByRole("tab", { name: "Equipment" }));
    await userEvent.click(screen.getByRole("button", { name: "Select Oven" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("Equipment selection could not be saved.");
    expect(screen.getByRole("button", { name: "Select Oven" })).toHaveAttribute("aria-pressed", "false");
  });
});
