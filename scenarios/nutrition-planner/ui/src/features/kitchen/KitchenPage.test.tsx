import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

const listInventoryBatches = vi.hoisted(() => vi.fn());
const listInventoryEvents = vi.hoisted(() => vi.fn());
const recordInventoryEvent = vi.hoisted(() => vi.fn());
const listRecipes = vi.hoisted(() => vi.fn());
const getProfile = vi.hoisted(() => vi.fn());
const applyProfile = vi.hoisted(() => vi.fn());
const saveKitchenDevices = vi.hoisted(() => vi.fn());
vi.mock("../../api/inventory", () => ({ listInventoryBatches, listInventoryEvents, recordInventoryEvent }));
vi.mock("../../api/recipes", () => ({ listRecipes }));
vi.mock("../../api/profile", () => ({ getProfile, applyProfile, saveKitchenDevices }));
vi.mock("../../api/workspace", () => ({ ensureWorkspace: vi.fn().mockResolvedValue({ id: "w1" }) }));

import { KitchenPage } from "./KitchenPage";

describe("KitchenPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    listInventoryBatches.mockResolvedValue([]);
    listInventoryEvents.mockResolvedValue([]);
    recordInventoryEvent.mockImplementation(async (input) => ({ id: input.id, lotId: input.lotId ?? input.id, workspaceId: input.workspaceId, kind: input.kind, itemId: input.itemId, amount: input.amount, amountMode: input.amountMode, expression: input.expression ?? "", storage: input.storage ?? "pantry", bestBeforeDate: input.bestBeforeDate ?? "", useSoon: input.useSoon ?? false, unit: input.unit, createdAt: new Date().toISOString() }));
    listRecipes.mockResolvedValue([]);
    let persistedProfile = { workspaceId: "w1", revision: 1n, preset: "vegan", presetVersion: 1n, activeRules: ["vegan"], excludedGroups: ["meat"], allergies: ["peanut"], appliances: ["stove"], cookware: ["saucepan"], tools: ["chef_knife"], kitchenDevices: [], costWeight: 0.2, effortWeight: 0.4, varietyWeight: 0.4, draftJson: "" };
    getProfile.mockImplementation(async () => persistedProfile);
    applyProfile.mockImplementation(async (input) => {
      persistedProfile = { ...persistedProfile, ...input, revision: persistedProfile.revision + 1n };
      return { profile: persistedProfile, matchingMeals: 0n, needsReviewMeals: 0n, excludedMeals: 0n };
    });
    saveKitchenDevices.mockImplementation(async (_workspaceId, devices) => { persistedProfile = { ...persistedProfile, kitchenDevices: devices, revision: persistedProfile.revision + 1n }; return persistedProfile; });
  });

  it("shows saved equipment and groups recorded inventory evidence by storage", async () => {
    listInventoryEvents.mockResolvedValue([{ id: "p1", kind: "purchase", itemId: "tofu", amount: "400", unit: "g", storage: "fridge", createdAt: "2026-10-05T12:00:00Z" }]);
    renderWithProviders(<KitchenPage />);
    await waitFor(() => expect(screen.getByRole("heading", { name: "Ingredients" })).toBeInTheDocument());
    expect(screen.getByText("400 g")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Fridge" })).toBeInTheDocument();
    expect(screen.getByText(/Last checked: No check recorded/)).toBeInTheDocument();
    expect(screen.queryByText("400 g available")).not.toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Freezer" }));
    expect(screen.getByText(/No ingredient evidence matches/)).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "All" }));
    expect(screen.getByText("tofu")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Manage shopping and receipts" })).toHaveAttribute("href", "/groceries");
    await userEvent.click(screen.getByRole("tab", { name: "Equipment" }));
    expect(screen.getByRole("button", { name: "Remove Cooktop" })).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByRole("button", { name: "Select Oven" })).toHaveAttribute("aria-pressed", "false");
  });

  it("derives Use soon from a recorded date and shows Last checked from an assertion", async () => {
    const date = new Date();
    date.setDate(date.getDate() + 2);
    const bestBeforeDate = `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, "0")}-${String(date.getDate()).padStart(2, "0")}`;
    listInventoryEvents.mockResolvedValue([{ id: "a1", kind: "assertion", itemId: "greens", amount: "unknown", amountMode: "unknown", storage: "fridge", bestBeforeDate, useSoon: false, unit: "", createdAt: "2026-10-08T14:00:00Z" }]);
    renderWithProviders(<KitchenPage />);
    expect(await screen.findByRole("link", { name: "Find meals" })).toHaveAttribute("href", expect.stringContaining("ingredients=greens"));
    expect(screen.getByRole("link", { name: "Find meals" })).toHaveAttribute("href", expect.stringContaining("rules=vegan"));
    expect(screen.getByText(/Last checked:/)).toHaveTextContent(/10\/8\/2026/);
    expect(screen.getByText(/Use soon · based on recorded date/)).toBeInTheDocument();
    expect(listInventoryEvents).toHaveBeenCalledTimes(1);
    expect(recordInventoryEvent).not.toHaveBeenCalled();
  });

  it("offers row actions that append stock evidence instead of rewriting history", async () => {
    listInventoryEvents.mockResolvedValue([{ id: "a1", kind: "assertion", itemId: "beans", amount: "2", amountMode: "exact", unit: "cans", storage: "pantry", createdAt: "2026-10-08T14:00:00Z" }]);
    renderWithProviders(<KitchenPage />);
    await userEvent.click(await screen.findByText("Actions"));
    await userEvent.click(screen.getByRole("button", { name: "Mark out" }));
    await waitFor(() => expect(recordInventoryEvent).toHaveBeenCalledWith(expect.objectContaining({ kind: "assertion", itemId: "beans", amount: "unknown", amountMode: "out", storage: "pantry", lotId: "a1" })));
    expect(screen.getAllByText("beans")).toHaveLength(1);
    expect(screen.getAllByText("Out").length).toBeGreaterThan(1);
  });

  it("keeps inventory unknown when no evidence exists", async () => {
    getProfile.mockResolvedValue(undefined);
    renderWithProviders(<KitchenPage />);
    await waitFor(() => expect(screen.getByText(/No ingredient evidence matches/)).toBeInTheDocument());
    expect(screen.getByText("No prepared portions yet")).toBeInTheDocument();
  });

  it("records and displays qualitative evidence without turning it into a quantity", async () => {
    renderWithProviders(<KitchenPage />);
    await userEvent.type(await screen.findByLabelText("Ingredient"), "rice");
    await userEvent.selectOptions(screen.getByLabelText("Amount mode"), "qualitative");
    await userEvent.selectOptions(screen.getByLabelText("Storage"), "freezer");
    await userEvent.type(screen.getByLabelText("Original expression"), "Some left");
    await userEvent.type(screen.getByLabelText("Best-before date"), "2099-01-01");
    await userEvent.click(screen.getByLabelText("Use soon (set by you)"));
    await userEvent.click(screen.getByRole("button", { name: "Add ingredient" }));
    await waitFor(() => expect(recordInventoryEvent).toHaveBeenCalledWith(expect.objectContaining({ workspaceId: "w1", kind: "assertion", itemId: "rice", amount: "unknown", amountMode: "qualitative", expression: "Some left", storage: "freezer", bestBeforeDate: "2099-01-01", useSoon: true, lotId: expect.stringContaining("lot-"), unit: "" })));
    expect(screen.getByRole("heading", { name: "Freezer" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Find meals" })).toHaveAttribute("href", expect.stringContaining("ingredients=rice"));
    expect(screen.getByRole("link", { name: "Find meals" })).toHaveAttribute("href", expect.stringContaining("rules=vegan"));
    expect(screen.getAllByText(/Use soon \(set by you\)/).length).toBeGreaterThan(0);
    expect(screen.queryByText(/Some left g/)).not.toBeInTheDocument();
  });

  it("saves an appliance change immediately while preserving the rest of the profile", async () => {
    renderWithProviders(<KitchenPage />);
    await userEvent.click(await screen.findByRole("tab", { name: "Equipment" }));
    await userEvent.click(screen.getByRole("button", { name: "Select Oven" }));
    await waitFor(() => expect(applyProfile).toHaveBeenCalledWith({
      workspaceId: "w1", preset: "vegan", excludedGroups: ["meat"], allergies: ["peanut"], appliances: ["stove", "oven"], cookware: ["saucepan"], tools: ["chef_knife"], costWeight: 0.2, effortWeight: 0.4, varietyWeight: 0.4,
    }));
    expect(screen.getByRole("button", { name: "Remove Oven" })).toHaveAttribute("aria-pressed", "true");
  });

  it("saves range and pressure-only multicooker device details without changing categories, then reloads them", async () => {
    const view = renderWithProviders(<KitchenPage />);
    await userEvent.click(await screen.findByRole("tab", { name: "Equipment" }));
    await userEvent.type(screen.getByLabelText("Capacity"), "5 burners");
    await userEvent.click(screen.getByLabelText("Cooktop"));
    await userEvent.click(screen.getByLabelText("Oven"));
    await userEvent.click(screen.getByRole("button", { name: "Add device" }));
    await waitFor(() => expect(screen.getByText("Enabled: Cooktop, Oven")).toBeInTheDocument());
    const range = saveKitchenDevices.mock.calls.at(-1)?.[1][0];
    expect(range).toMatchObject({ type: "range", count: 1, capacity: "5 burners", capabilities: ["cooktop", "oven"] });
    expect(range.id).toMatch(/^device-/);
    await userEvent.selectOptions(screen.getByLabelText("Device type"), "multicooker");
    await userEvent.clear(screen.getByLabelText("Capacity"));
    await userEvent.type(screen.getByLabelText("Capacity"), "6 qt");
    await userEvent.click(screen.getByLabelText("Pressure cook"));
    await userEvent.click(screen.getByRole("button", { name: "Add device" }));
    await waitFor(() => expect(screen.getByText("Enabled: Pressure cook")).toBeInTheDocument());
    expect(saveKitchenDevices).toHaveBeenLastCalledWith("w1", expect.arrayContaining([
      expect.objectContaining({ id: range.id, type: "range", capabilities: ["cooktop", "oven"] }),
      expect.objectContaining({ type: "multicooker", count: 1, capacity: "6 qt", capabilities: ["pressure_cook"] }),
    ]));
    expect(screen.getByRole("button", { name: "Remove Cooktop" })).toHaveAttribute("aria-pressed", "true");
    view.unmount();
    renderWithProviders(<KitchenPage />);
    await userEvent.click(await screen.findByRole("tab", { name: "Equipment" }));
    expect(screen.getByText("Enabled: Cooktop, Oven")).toBeInTheDocument();
    expect(screen.getByText("Enabled: Pressure cook")).toBeInTheDocument();
    await userEvent.click(screen.getByRole("tab", { name: "Cookware" }));
    expect(screen.getByRole("button", { name: "Remove Saucepan" })).toHaveAttribute("aria-pressed", "true");
  });

  it("shows and persists semantic cookware and tool selections", async () => {
    const view = renderWithProviders(<KitchenPage />);
    await userEvent.click(await screen.findByRole("tab", { name: "Equipment" }));
    await userEvent.click(screen.getByRole("tab", { name: "Cookware" }));
    expect(screen.getByRole("button", { name: "Remove Saucepan" })).toHaveAttribute("aria-pressed", "true");
    await userEvent.click(screen.getByRole("button", { name: "Select Frying pan" }));
    await waitFor(() => expect(applyProfile).toHaveBeenLastCalledWith(expect.objectContaining({ cookware: ["saucepan", "frying_pan"], tools: ["chef_knife"] })));
    await waitFor(() => expect(screen.getByRole("button", { name: "Remove Frying pan" })).toBeInTheDocument());
    await userEvent.click(screen.getByRole("tab", { name: "Tools" }));
    expect(screen.getByRole("button", { name: "Remove Chef’s knife" })).toHaveAttribute("aria-pressed", "true");
    await userEvent.click(screen.getByRole("button", { name: "Remove Chef’s knife" }));
    await waitFor(() => expect(applyProfile).toHaveBeenLastCalledWith(expect.objectContaining({ cookware: ["saucepan", "frying_pan"], tools: [] })));
    await userEvent.click(screen.getByRole("tab", { name: "Cookware" }));
    expect(screen.getByRole("button", { name: "Remove Frying pan" })).toHaveAttribute("aria-pressed", "true");
    view.unmount();
    renderWithProviders(<KitchenPage />);
    await userEvent.click(await screen.findByRole("tab", { name: "Equipment" }));
    await userEvent.click(screen.getByRole("tab", { name: "Cookware" }));
    expect(screen.getByRole("button", { name: "Remove Saucepan" })).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByRole("button", { name: "Remove Frying pan" })).toHaveAttribute("aria-pressed", "true");
    await userEvent.click(screen.getByRole("tab", { name: "Tools" }));
    expect(screen.getByRole("button", { name: "Select Chef’s knife" })).toHaveAttribute("aria-pressed", "false");
  });

  it("rolls an optimistic equipment choice back when profile saving fails", async () => {
    applyProfile.mockRejectedValue(new Error("offline"));
    renderWithProviders(<KitchenPage />);
    await userEvent.click(await screen.findByRole("tab", { name: "Equipment" }));
    await userEvent.click(screen.getByRole("button", { name: "Select Oven" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("offline");
    expect(screen.getByRole("button", { name: "Select Oven" })).toHaveAttribute("aria-pressed", "false");
  });

  it("rolls a cookware choice back and exposes the save failure", async () => {
    applyProfile.mockRejectedValue(new Error("profile unavailable"));
    renderWithProviders(<KitchenPage />);
    await userEvent.click(await screen.findByRole("tab", { name: "Equipment" }));
    await userEvent.click(screen.getByRole("tab", { name: "Cookware" }));
    await userEvent.click(screen.getByRole("button", { name: "Select Frying pan" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("profile unavailable");
    expect(screen.getByRole("button", { name: "Select Frying pan" })).toHaveAttribute("aria-pressed", "false");
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
    expect(screen.getByRole("button", { name: "Select Frying pan" })).toBeDisabled();
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
    expect(screen.queryByText("soup")).not.toBeInTheDocument();
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
