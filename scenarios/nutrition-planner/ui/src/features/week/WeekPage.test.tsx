import { beforeEach, describe, expect, it, vi } from "vitest";
import { create } from "@bufbuild/protobuf";
import { CatalogRevisionSchema, NutrientValueSchema } from "@vrooli/proto-types/nutrition-planner/v1/catalog/catalog_pb";
import { useLocation } from "react-router-dom";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "../../test-utils";
import { localDateString } from "../../lib/dates";
import type { PlanDraft } from "../../api/planning";
const generatePlan = vi.hoisted(() => vi.fn());
const getPlan = vi.hoisted(() => vi.fn());
const listRecipes = vi.hoisted(() => vi.fn());
const listCatalogRevisions = vi.hoisted(() => vi.fn().mockResolvedValue([]));
const applyPlan = vi.hoisted(() => vi.fn());
const previewSwap = vi.hoisted(() => vi.fn());
const exportWeeklyPDF = vi.hoisted(() => vi.fn());
const listIntakes = vi.hoisted(() => vi.fn().mockResolvedValue([]));
const listSupplementSchedules = vi.hoisted(() => vi.fn().mockResolvedValue([]));
const scheduleAppliesOnLocalDate = vi.hoisted(() => vi.fn((_schedule: unknown, _date: string) => false));
vi.mock("../../api/planning", () => ({ generatePlan, getPlan, applyPlan, previewSwap }));
vi.mock("../../api/recipes", () => ({ listRecipes }));
vi.mock("../../api/catalog", () => ({ listCatalogRevisions }));
vi.mock("../../api/portability", () => ({ exportWeeklyPDF }));
vi.mock("../../api/nutrition", () => ({ listIntakes }));
vi.mock("../../api/supplement", () => ({ listSupplementSchedules, scheduleAppliesOnLocalDate }));
vi.mock("../../api/workspace", () => ({ ensureWorkspace: vi.fn().mockResolvedValue({ id: "w1" }) }));
beforeEach(() => { generatePlan.mockReset(); getPlan.mockReset(); listRecipes.mockReset(); listCatalogRevisions.mockReset(); applyPlan.mockReset(); previewSwap.mockReset(); exportWeeklyPDF.mockReset(); listIntakes.mockReset(); listSupplementSchedules.mockReset(); scheduleAppliesOnLocalDate.mockReset(); listRecipes.mockResolvedValue([]); listCatalogRevisions.mockResolvedValue([]); listIntakes.mockResolvedValue([]); listSupplementSchedules.mockResolvedValue([]); scheduleAppliesOnLocalDate.mockReturnValue(false); generatePlan.mockResolvedValue({ occurrences: [], unresolved: [], currentRevision: 0n }); getPlan.mockImplementation(async (input) => ({ draft: await generatePlan(input), hasPlan: true })); });
import { WeekPage } from "./WeekPage";
function CurrentSearch() { const { search } = useLocation(); return <output data-testid="current-search">{search}</output>; }
describe("WeekPage", () => { beforeEach(() => { vi.clearAllMocks(); listRecipes.mockResolvedValue([]); getPlan.mockImplementation(async (input) => ({ draft: await generatePlan(input), hasPlan: true })); }); it("renders seven explicit dinner rows from the persisted date range", async () => { const user = userEvent.setup(); getPlan.mockResolvedValue({ draft: { occurrences: [], unresolved: [], currentRevision: 0n }, hasPlan: false }); renderWithProviders(<WeekPage />, { routerEntries: ["/week?range=all"] }); await waitFor(() => expect(screen.getAllByText("No meal has been saved for this date.")).toHaveLength(7)); await user.click(screen.getByRole("button", { name: "Next week" })); await waitFor(() => expect(getPlan).toHaveBeenCalledTimes(2)); expect(generatePlan).not.toHaveBeenCalled(); }); it("drafts only after explicit user action when the selected week is empty", async () => { const user = userEvent.setup(); getPlan.mockResolvedValue({ draft: { occurrences: [], unresolved: [], currentRevision: 0n }, hasPlan: false }); generatePlan.mockResolvedValue({ occurrences: [], unresolved: [{ date: "2026-10-03", code: "no_candidate", message: "No eligible meal" }], currentRevision: 0n }); renderWithProviders(<WeekPage />); await user.click(await screen.findByRole("button", { name: "Plan my week" })); await waitFor(() => expect(generatePlan).toHaveBeenCalledTimes(1)); }); it("supports explicit lock and skip controls", async () => { const user = userEvent.setup(); const date = new Date().toISOString().slice(0, 10); generatePlan.mockResolvedValue({ occurrences: [{ date, slotName: "dinner", recipeId: "a", recipeName: "A", locked: false, reason: "Known fit" }], unresolved: [], currentRevision: 1n }); applyPlan.mockResolvedValue({ revision: 2n }); renderWithProviders(<WeekPage />); await waitFor(() => expect(screen.getByRole("button", { name: "Lock" })).toBeInTheDocument()); await user.click(screen.getByRole("button", { name: "Lock" })); expect(screen.getByRole("button", { name: "Unlock" })).toBeInTheDocument(); await user.click(screen.getByRole("button", { name: "Skip" })); expect(screen.getByText("Open slot")).toBeInTheDocument(); await user.click(screen.getByRole("button", { name: "Save week" })); expect(applyPlan).toHaveBeenCalled(); }); it("previews and applies a scoped swap", async () => { const user = userEvent.setup(); const date = new Date().toISOString().slice(0, 10); generatePlan.mockResolvedValue({ occurrences: [{ date, slotName: "dinner", recipeId: "a", recipeName: "A", locked: false, reason: "Known fit" }], unresolved: [], currentRevision: 1n }); listRecipes.mockResolvedValue([{ id: "b", name: "B" }]); previewSwap.mockResolvedValue({ revision: 1n, preview: { draft: { occurrences: [{ date, slotName: "dinner", recipeId: "b", recipeName: "B", locked: false, reason: "swapped" }], unresolved: [], inputReferences: [], runId: "seed-0", seed: 0, currentRevision: 1n }, changes: [{ date, slotName: "dinner", beforeName: "A", afterName: "B" }], shoppingChanges: [{ key: "ingredient:rice", after: { key: "ingredient:rice", label: "rice", need: "unknown", stock: "unknown", missing: "unknown", packageCount: "unknown", price: "unknown", sourceRecipeIds: ["b"], checked: false } }] }, affectedDates: [date] }); applyPlan.mockResolvedValue({ revision: 2n }); renderWithProviders(<WeekPage />); await waitFor(() => expect(screen.getByRole("heading", { name: "Your week" })).toBeInTheDocument()); await user.click(screen.getByRole("button", { name: "Swap meal" })); await user.selectOptions(screen.getByLabelText("Replacement meal"), "b"); await user.click(screen.getByRole("button", { name: "Preview swap" })); await waitFor(() => expect(screen.getByText(/A → B/)).toBeInTheDocument()); expect(screen.getByRole("region", { name: "Shopping impact" })).toHaveTextContent("Added: rice"); expect(screen.getByRole("region", { name: "Shopping impact" })).toHaveTextContent("Price: unknown"); await user.click(screen.getByRole("button", { name: "Apply reviewed changes" })); await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument()); expect(previewSwap).toHaveBeenCalledWith(expect.objectContaining({ slotName: "dinner" })); expect(applyPlan).toHaveBeenCalled(); }); });

it("requires reviewing the full-week impact before an explicit replan save", async () => {
  vi.clearAllMocks();
  listRecipes.mockResolvedValue([]);
  const user = userEvent.setup();
  const previousDate = "2026-10-03";
  getPlan.mockResolvedValue({ draft: { occurrences: [
    { date: previousDate, slotName: "dinner", recipeId: "old", recipeName: "Saved soup", locked: false, reason: "saved" },
    { date: "2026-10-04", slotName: "lunch", recipeId: "old-2", recipeName: "Saved toast", locked: false, reason: "saved" },
  ], unresolved: [], currentRevision: 3n }, hasPlan: true });
  generatePlan.mockResolvedValue({ occurrences: [
    { date: previousDate, slotName: "dinner", recipeId: "new", recipeName: "New rice", locked: false, reason: "generated" },
    { date: "2026-10-04", slotName: "lunch", recipeId: "new-2", recipeName: "New beans", locked: false, reason: "generated" },
  ], unresolved: [
    { date: "2026-10-05", code: "no_candidate", message: "No eligible meal" },
    { date: "2026-10-06", code: "no_candidate", message: "Another date needs review" },
  ], currentRevision: 3n });
  renderWithProviders(<WeekPage />);
  await user.click(await screen.findByRole("button", { name: "Replan week" }));
  expect(await screen.findByText(/2 meal or lock changes to review/)).toBeInTheDocument();
  await user.click(screen.getByText("Show full-week changes and unresolved dates"));
  expect(screen.getByText(/Saved soup → New rice/)).toBeInTheDocument();
  expect(screen.getAllByRole("listitem").some((item) => item.textContent?.includes("2026-10-05") && item.textContent.includes("No eligible meal"))).toBe(true);
  expect(applyPlan).not.toHaveBeenCalled();
});

it("states when a generated week has no changes to the saved plan", async () => {
  vi.clearAllMocks();
  listRecipes.mockResolvedValue([]);
  getPlan.mockResolvedValue({ draft: { occurrences: [], unresolved: [], currentRevision: 0n }, hasPlan: true });
  renderWithProviders(<WeekPage />);
  expect(await screen.findByRole("heading", { name: "Your week" })).toBeInTheDocument();
  expect(screen.queryByRole("region", { name: "Plan changes" })).not.toBeInTheDocument();
  expect(screen.queryByRole("button", { name: "Save week" })).not.toBeInTheDocument();
  await userEvent.click(screen.getByRole("button", { name: "Replan week" }));
  expect(await screen.findByText("No meal or lock changes from the saved week.")).toBeInTheDocument();
  expect(screen.queryByRole("button", { name: "Save week" })).not.toBeInTheDocument();
  await userEvent.click(screen.getByText("Show full-week changes and unresolved dates"));
  expect(screen.getByText("No changes.")).toBeInTheDocument();
});

it("summarizes explicit lock and unlock changes against the saved week", async () => {
  vi.clearAllMocks();
  listRecipes.mockResolvedValue([]);
  const date = new Date();
  date.setHours(0, 0, 0, 0);
  const selectedDate = `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, "0")}-${String(date.getDate()).padStart(2, "0")}`;
  getPlan.mockResolvedValue({ draft: { occurrences: [{ date: selectedDate, slotName: "dinner", recipeId: "soup", recipeName: "Soup", locked: false, reason: "saved" }], unresolved: [], currentRevision: 1n }, hasPlan: true });
  renderWithProviders(<WeekPage />);
  await userEvent.click(await screen.findByRole("button", { name: "Lock" }));
  await userEvent.click(screen.getByText("Show full-week changes and unresolved dates"));
  expect(screen.getByText(/Soup → Soup \(locked\)/)).toBeInTheDocument();
});

it("labels an explicitly unlocked saved occurrence in the review", async () => {
  vi.clearAllMocks();
  listRecipes.mockResolvedValue([]);
  const date = new Date();
  date.setHours(0, 0, 0, 0);
  const selectedDate = `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, "0")}-${String(date.getDate()).padStart(2, "0")}`;
  getPlan.mockResolvedValue({ draft: { occurrences: [{ date: selectedDate, slotName: "dinner", recipeId: "soup", recipeName: "Soup", locked: true, reason: "saved" }], unresolved: [], currentRevision: 1n }, hasPlan: true });
  renderWithProviders(<WeekPage />);
  await userEvent.click(await screen.findByRole("button", { name: "Unlock" }));
  await userEvent.click(screen.getByText("Show full-week changes and unresolved dates"));
  expect(screen.getByText(/Soup → Soup \(unlocked\)/)).toBeInTheDocument();
});

it("reviews newly added open slots without pretending they contain a recipe", async () => {
  vi.clearAllMocks();
  listRecipes.mockResolvedValue([]);
  getPlan.mockResolvedValue({ draft: { occurrences: [], unresolved: [], currentRevision: 1n }, hasPlan: true });
  generatePlan.mockResolvedValue({ occurrences: [{ date: "2026-10-03", slotName: "snack", recipeId: "", recipeName: "", locked: false, reason: "open" }], unresolved: [], currentRevision: 1n });
  renderWithProviders(<WeekPage />);
  await userEvent.click(await screen.findByRole("button", { name: "Replan week" }));
  await userEvent.click(screen.getByText("Show full-week changes and unresolved dates"));
  expect(screen.getByText(/2026-10-03 · snack: added open slot/)).toBeInTheDocument();
});

it("describes an open slot changing modes without inventing a recipe name", async () => {
  vi.clearAllMocks();
  listRecipes.mockResolvedValue([]);
  getPlan.mockResolvedValue({ draft: { occurrences: [{ date: "2026-10-03", slotName: "snack", mode: "open", recipeId: "", recipeName: "", locked: false, reason: "saved open slot" }], unresolved: [], currentRevision: 1n }, hasPlan: true });
  generatePlan.mockResolvedValue({ occurrences: [{ date: "2026-10-03", slotName: "snack", mode: "social", recipeId: "", recipeName: "", locked: false, reason: "social" }], unresolved: [], currentRevision: 1n });
  renderWithProviders(<WeekPage />);
  await userEvent.click(await screen.findByRole("button", { name: "Replan week" }));
  await userEvent.click(screen.getByText("Show full-week changes and unresolved dates"));
  expect(screen.getByText(/2026-10-03 · snack: open slot → open slot/)).toBeInTheDocument();
});

it("replans all persisted slots while carrying open and locked identity forward", async () => {
  vi.clearAllMocks();
  listRecipes.mockResolvedValue([]);
  const date = new Date();
  date.setHours(0, 0, 0, 0);
  const selectedDate = `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, "0")}-${String(date.getDate()).padStart(2, "0")}`;
  getPlan.mockResolvedValue({ draft: { occurrences: [
    { date: selectedDate, slotName: "breakfast", mode: "fixed", quantity: "1", recipeId: "locked-id", recipeName: "Fixed breakfast", locked: true, reason: "owner lock" },
    { date: selectedDate, slotName: "dinner", mode: "open", quantity: "2", recipeId: "", recipeName: "", locked: false, reason: "open by owner" },
  ], unresolved: [], currentRevision: 5n }, hasPlan: true });
  generatePlan.mockResolvedValue({ occurrences: [], unresolved: [], currentRevision: 5n });
  renderWithProviders(<WeekPage />);
  await userEvent.click(await screen.findByRole("button", { name: "Replan week" }));
  await waitFor(() => expect(generatePlan).toHaveBeenCalledTimes(1));
  const request = generatePlan.mock.calls[0]?.[0];
  expect(request.mealSlots).toContainEqual({ date: selectedDate, slotName: "breakfast", mode: "fixed", quantity: "1", lockedRecipeId: "locked-id" });
  expect(request.mealSlots).toContainEqual({ date: selectedDate, slotName: "dinner", mode: "open", quantity: "2", lockedRecipeId: "" });
});

it("renders every saved slot for a date instead of collapsing the week to dinner", async () => {
  vi.clearAllMocks();
  listRecipes.mockResolvedValue([]);
  const date = new Date();
  date.setHours(0, 0, 0, 0);
  const selectedDate = `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, "0")}-${String(date.getDate()).padStart(2, "0")}`;
  getPlan.mockResolvedValue({ draft: { occurrences: [
    { date: selectedDate, slotName: "breakfast", recipeId: "b", recipeName: "Oats", locked: true, reason: "locked" },
    { date: selectedDate, slotName: "dinner", recipeId: "d", recipeName: "Soup", locked: false, reason: "planned" },
  ], unresolved: [], currentRevision: 2n }, hasPlan: true });
  renderWithProviders(<WeekPage />);
  expect(await screen.findByText("Oats")).toBeInTheDocument();
  expect(screen.getByText("Soup")).toBeInTheDocument();
  expect(screen.getByText("breakfast")).toBeInTheDocument();
});

it("separates planned and recorded nutrition and supports a one-day agenda", async () => {
  vi.clearAllMocks();
  listRecipes.mockResolvedValue([]);
  let today = "";
  let weekDates: string[] = [];
  getPlan.mockImplementation(async (input) => {
    today = input.fromDate;
    const occurrences = Array.from({ length: 7 }, (_, index) => {
      const day = new Date(`${today}T12:00:00`);
      day.setDate(day.getDate() + index);
      const dayString = localDateString(day);
      return { date: dayString, slotName: index === 1 ? "lunch" : "dinner", recipeId: index === 1 ? "r2" : "r1", recipeName: index === 1 ? "Bean soup" : "Lentil bowl", locked: false, reason: "planned" };
    });
    weekDates = occurrences.map((item) => item.date);
    return { draft: { occurrences, unresolved: [], currentRevision: 1n }, hasPlan: true };
  });
  listIntakes.mockImplementation(async () => weekDates.map((date) => ({ date, nutrientId: "protein", amount: "21", unit: "g" })));
  renderWithProviders(<WeekPage />);
  const user = userEvent.setup();
  await user.click(await screen.findByRole("button", { name: "Nutrition" }));
  expect(screen.getAllByText(/Lentil bowl/)).toHaveLength(6);
  expect(screen.getAllByText(/protein 21 g/)).toHaveLength(7);
  expect(screen.getAllByText(/no confirmed supplement schedule applies/)).toHaveLength(7);
  await user.click(screen.getByRole("button", { name: "Day" }));
  const daySelect = screen.getByLabelText("Selected day");
  await user.selectOptions(daySelect, daySelect.querySelectorAll("option")[1]?.getAttribute("value") ?? "");
  expect(screen.getByText(/Bean soup/)).toBeInTheDocument();
  expect(screen.getAllByText(/protein 21 g/)).toHaveLength(1);
});

it("shows a confirmed supplement schedule as expected without recording or changing its dose", async () => {
  vi.clearAllMocks();
  listRecipes.mockResolvedValue([]);
  getPlan.mockResolvedValue({ draft: { occurrences: [], unresolved: [], currentRevision: 1n }, hasPlan: true });
  listSupplementSchedules.mockResolvedValue([{ id: "schedule-1", revision: 4n, productRevisionId: "catalog:vitamin-d:3", dose: "1000", doseUnit: "IU", weekdays: [1], startDate: "2026-10-01", endDate: "2026-12-31", paused: false, confirmed: true, createdAt: "2026-10-01T00:00:00Z" }]);
  listCatalogRevisions.mockResolvedValue([create(CatalogRevisionSchema, { id: "vitamin-d", revision: 3n, name: "Vitamin D", productName: "Vitamin D3", servingQuantity: "1000", servingUnit: "IU", nutrients: [create(NutrientValueSchema, { nutrientId: "vitamin_d", amount: "25", unit: "mcg", basis: "1000", basisUnit: "IU", evidence: "label", sourceRef: "fixture label" })] })]);
  scheduleAppliesOnLocalDate.mockReturnValue(true);
  renderWithProviders(<WeekPage />);
  await userEvent.click(await screen.findByRole("button", { name: "Nutrition" }));
  expect(await screen.findAllByText(/vitamin_d: 25 mcg expected \(label\)/)).not.toHaveLength(0);
  expect(screen.getAllByText(/no recorded intake/).length).toBeGreaterThan(0);
  expect(listSupplementSchedules).toHaveBeenCalledWith("w1");
});

describe("Week swap dialog keyboard behavior", () => { beforeEach(() => { vi.clearAllMocks(); listRecipes.mockResolvedValue([{ id: "b", name: "B" }]); }); it("focuses the dialog and restores the trigger on Escape", async () => { const user = userEvent.setup(); const date = new Date().toISOString().slice(0, 10); generatePlan.mockResolvedValue({ occurrences: [{ date, slotName: "dinner", recipeId: "a", recipeName: "A", locked: false, reason: "Known fit" }], unresolved: [], currentRevision: 1n }); renderWithProviders(<WeekPage />); await waitFor(() => expect(screen.getByRole("button", { name: "Swap meal" })).toBeInTheDocument()); const trigger = screen.getByRole("button", { name: "Swap meal" }); await user.click(trigger); expect(screen.getByLabelText("Replacement meal")).toHaveFocus(); await user.keyboard("{Escape}"); expect(screen.queryByRole("dialog")).not.toBeInTheDocument(); expect(trigger).toHaveFocus(); }); });

it("shows a drafting failure while the selected week is empty", async () => { getPlan.mockResolvedValue({ draft: { occurrences: [], unresolved: [], currentRevision: 0n }, hasPlan: false }); generatePlan.mockRejectedValue(new Error("no eligible meals")); renderWithProviders(<WeekPage />); await userEvent.click(await screen.findByRole("button", { name: "Plan my week" })); expect(await screen.findByRole("alert")).toHaveTextContent("no eligible meals"); });
it("states when a swap does not add or remove shopping lines", async () => {
  const date = new Date().toISOString().slice(0, 10);
  getPlan.mockResolvedValue({ draft: { occurrences: [{ date, slotName: "dinner", recipeId: "a", recipeName: "A", locked: false, reason: "saved" }], unresolved: [], currentRevision: 1n }, hasPlan: true });
  listRecipes.mockResolvedValue([{ id: "b", name: "B" }]);
  previewSwap.mockResolvedValue({ revision: 1n, preview: { draft: { occurrences: [{ date, slotName: "dinner", recipeId: "b", recipeName: "B", locked: false, reason: "swap" }], unresolved: [], inputReferences: [], runId: "seed-0", seed: 0, currentRevision: 1n }, changes: [{ date, slotName: "dinner", beforeName: "A", afterName: "B" }], shoppingChanges: [] }, affectedDates: [date] });
  renderWithProviders(<WeekPage />);
  const user = userEvent.setup();
  await user.click(await screen.findByRole("button", { name: "Swap meal" }));
  await user.selectOptions(screen.getByLabelText("Replacement meal"), "b");
  await user.click(screen.getByRole("button", { name: "Preview swap" }));
  expect(await screen.findByText("This preview changes one meal slot and no shopping lines.")).toBeInTheDocument();
  expect(screen.queryByRole("region", { name: "Shopping impact" })).not.toBeInTheDocument();
  expect(screen.getByRole("button", { name: "Confirm swap" })).toBeInTheDocument();
});
it("requires review of unchanged later occurrences of the same recipe", async () => {
  const date = new Date().toISOString().slice(0, 10);
  const later = new Date(Date.now() + 2 * 86400000).toISOString().slice(0, 10);
  getPlan.mockResolvedValue({ draft: { occurrences: [{ date, slotName: "dinner", recipeId: "a", recipeName: "A", locked: false, reason: "saved" }, { date: later, slotName: "lunch", recipeId: "a", recipeName: "A", locked: false, reason: "saved" }], unresolved: [], currentRevision: 1n }, hasPlan: true });
  listRecipes.mockResolvedValue([{ id: "b", name: "B" }]);
  previewSwap.mockResolvedValue({ revision: 1n, preview: { draft: { occurrences: [{ date, slotName: "dinner", recipeId: "b", recipeName: "B", locked: false, reason: "swap" }, { date: later, slotName: "lunch", recipeId: "a", recipeName: "A", locked: false, reason: "saved" }], unresolved: [], inputReferences: [], runId: "seed-0", seed: 0, currentRevision: 1n }, changes: [{ date, slotName: "dinner", beforeName: "A", afterName: "B" }], relatedOccurrences: [{ date: later, slotName: "lunch", recipeRevision: 3, recipeName: "A", locked: false }], preparedBatchImpacts: [{ batchId: "batch-a", recipeId: "a", recipeName: "A", recipeRevision: 3, available: "2", unit: "servings" }], shoppingChanges: [] }, affectedDates: [date] });
  renderWithProviders(<WeekPage />);
  const user = userEvent.setup();
  await user.click(await screen.findAllByRole("button", { name: "Swap meal" }).then((buttons) => buttons[0]!));
  await user.selectOptions(screen.getByLabelText("Replacement meal"), "b");
  await user.click(screen.getByRole("button", { name: "Preview swap" }));
  const impact = await screen.findByRole("region", { name: "Shopping impact" });
  expect(impact).toHaveTextContent("Later uses to review");
  expect(impact).toHaveTextContent(`${later} · lunch: A`);
  expect(impact).toHaveTextContent("revision 3");
  expect(impact).toHaveTextContent("no batch or leftover relationship is inferred");
  expect(impact).toHaveTextContent("Prepared portions to review");
  expect(impact).toHaveTextContent("A · revision 3 · batch batch-a: 2 servings available");
  expect(impact).toHaveTextContent("does not allocate or consume a batch");
  expect(screen.getByRole("button", { name: "Apply reviewed changes" })).toBeInTheDocument();
  expect(applyPlan).not.toHaveBeenCalled();
});
it("requires a wide review when only prepared batch portions add an impact", async () => {
  const date = new Date().toISOString().slice(0, 10);
  applyPlan.mockClear();
  getPlan.mockResolvedValue({ draft: { occurrences: [{ date, slotName: "dinner", recipeId: "a", recipeName: "A", locked: false, reason: "saved" }], unresolved: [], currentRevision: 1n }, hasPlan: true });
  listRecipes.mockResolvedValue([{ id: "b", name: "B" }]);
  previewSwap.mockResolvedValue({ revision: 1n, preview: { draft: { occurrences: [{ date, slotName: "dinner", recipeId: "b", recipeName: "B", locked: false, reason: "swap" }], unresolved: [], inputReferences: [], runId: "seed-0", seed: 0, currentRevision: 1n }, changes: [{ date, slotName: "dinner", beforeName: "A", afterName: "B" }], preparedBatchImpacts: [{ batchId: "batch-a", recipeId: "a", recipeName: "A", recipeRevision: 2, available: "3", unit: "servings" }], shoppingChanges: [] }, affectedDates: [date] });
  renderWithProviders(<WeekPage />);
  const user = userEvent.setup();
  await user.click(await screen.findByRole("button", { name: "Swap meal" }));
  await user.selectOptions(screen.getByLabelText("Replacement meal"), "b");
  await user.click(screen.getByRole("button", { name: "Preview swap" }));
  const impact = await screen.findByRole("region", { name: "Shopping impact" });
  expect(impact).toHaveTextContent("A · revision 2 · batch batch-a: 3 servings available");
  expect(screen.getByRole("button", { name: "Apply reviewed changes" })).toBeInTheDocument();
  expect(screen.queryByRole("button", { name: "Confirm swap" })).not.toBeInTheDocument();
  expect(applyPlan).not.toHaveBeenCalled();
});
it("reports a stale save instead of claiming the week was saved", async () => { const date = new Date().toISOString().slice(0, 10); getPlan.mockResolvedValue({ draft: { occurrences: [{ date, slotName: "dinner", recipeId: "r1", recipeName: "Bowl", locked: false, reason: "Known fit" }], unresolved: [], currentRevision: 4n }, hasPlan: true }); applyPlan.mockRejectedValue(new Error("plan changed elsewhere")); renderWithProviders(<WeekPage />); await userEvent.click(await screen.findByRole("button", { name: "Lock" })); await userEvent.click(await screen.findByRole("button", { name: "Save week" })); expect(await screen.findByRole("alert")).toHaveTextContent("plan changed elsewhere"); expect(screen.queryByText(/Week saved/)).not.toBeInTheDocument(); });
it("keeps an ordinary save failure separate from stale recovery", async () => {
  const date = localDateString(new Date());
  getPlan.mockResolvedValue({ draft: { occurrences: [{ date, slotName: "dinner", recipeId: "dinner", recipeName: "Dinner", locked: false, reason: "Saved" }], unresolved: [], currentRevision: 2n }, hasPlan: true });
  applyPlan.mockRejectedValue(new Error("storage unavailable"));
  renderWithProviders(<WeekPage />);
  await userEvent.click(await screen.findByRole("button", { name: "Lock" }));
  await userEvent.click(await screen.findByRole("button", { name: "Save week" }));
  expect(await screen.findByRole("alert")).toHaveTextContent("storage unavailable");
  expect(screen.queryByRole("region", { name: "Stale plan review" })).not.toBeInTheDocument();
  expect(screen.getByRole("button", { name: "Save week" })).toBeEnabled();
});
it("shows a safe page error when the saved week cannot be loaded", async () => { getPlan.mockRejectedValue("offline"); renderWithProviders(<WeekPage />); expect(await screen.findByRole("alert")).toHaveTextContent("Unable to load this week."); expect(screen.queryByRole("list", { name: "Meals by day" })).not.toBeInTheDocument(); });
it("keeps the draft and reports failure when stale review cannot load the latest week", async () => {
  const date = localDateString(new Date());
  getPlan.mockResolvedValueOnce({ draft: { occurrences: [{ date, slotName: "dinner", recipeId: "bowl", recipeName: "Bowl", locked: false, reason: "Saved" }], unresolved: [], currentRevision: 4n }, hasPlan: true }).mockRejectedValueOnce(new Error("latest revision unavailable"));
  applyPlan.mockRejectedValueOnce(new Error("plan changed elsewhere"));
  const user = userEvent.setup();
  renderWithProviders(<WeekPage />);
  await user.click(await screen.findByRole("button", { name: "Lock" }));
  await user.click(await screen.findByRole("button", { name: "Save week" }));
  await user.click(await screen.findByRole("button", { name: "Review latest week" }));
  expect(await screen.findByRole("alert")).toHaveTextContent("latest revision unavailable");
  expect(screen.getByText("Bowl")).toBeInTheDocument();
});
it("reviews and rebases non-overlapping local and current-week changes after a stale save", async () => {
  const date = localDateString(new Date());
  const baseline: PlanDraft = { occurrences: [
    { date, slotName: "dinner", recipeId: "d1", recipeName: "Dinner", locked: false, reason: "Saved" },
    { date, slotName: "lunch", recipeId: "l1", recipeName: "Old lunch", locked: false, reason: "Saved" },
  ], unresolved: [], inputReferences: [], runId: "baseline", seed: 1, currentRevision: 4n };
  const latest: PlanDraft = { ...baseline, occurrences: [baseline.occurrences[0]!, { ...baseline.occurrences[1]!, recipeId: "l2", recipeName: "New lunch" }], currentRevision: 5n };
  getPlan.mockResolvedValueOnce({ draft: baseline, hasPlan: true }).mockResolvedValueOnce({ draft: latest, hasPlan: true });
  applyPlan.mockRejectedValueOnce(new Error("plan changed elsewhere")).mockResolvedValueOnce({ revision: 6n });
  renderWithProviders(<WeekPage />);
  const user = userEvent.setup();
  await user.click((await screen.findAllByRole("button", { name: "Lock" }))[0]!);
  await user.click(screen.getByRole("button", { name: "Save week" }));
  expect(await screen.findByRole("alert")).toHaveTextContent("plan changed elsewhere");
  await user.click(screen.getByRole("button", { name: "Review latest week" }));
  expect(await screen.findByText(new RegExp(`${date} · lunch: Old lunch → New lunch`))).toBeInTheDocument();
  await user.click(screen.getByRole("button", { name: "Rebase my changes for review" }));
  expect(screen.getByText("New lunch")).toBeInTheDocument();
  expect(await screen.findAllByText(/Dinner → Dinner \(locked\)/)).not.toHaveLength(0);
  await user.click(screen.getByRole("button", { name: "Save week" }));
  await waitFor(() => expect(applyPlan).toHaveBeenCalled());
  expect(applyPlan).toHaveBeenLastCalledWith(expect.objectContaining({ expectedRevision: 5n, draft: expect.objectContaining({ currentRevision: 5n }) }));
});
it("accepts a local edit that independently converged with the latest saved slot", async () => {
  const date = localDateString(new Date());
  const baseline: PlanDraft = { occurrences: [{ date, slotName: "dinner", recipeId: "d1", recipeName: "Dinner", locked: false, reason: "Saved" }], unresolved: [], inputReferences: [], runId: "base", seed: 1, currentRevision: 4n };
  const latest: PlanDraft = { ...baseline, occurrences: [{ ...baseline.occurrences[0]!, locked: true }], currentRevision: 5n };
  getPlan.mockResolvedValueOnce({ draft: baseline, hasPlan: true }).mockResolvedValueOnce({ draft: latest, hasPlan: true });
  applyPlan.mockRejectedValueOnce(new Error("plan changed elsewhere")).mockResolvedValueOnce({ revision: 6n });
  const user = userEvent.setup();
  renderWithProviders(<WeekPage />);
  await user.click(await screen.findByRole("button", { name: "Lock" }));
  await user.click(screen.getByRole("button", { name: "Save week" }));
  await user.click(await screen.findByRole("button", { name: "Review latest week" }));
  expect(await screen.findAllByText(/Dinner → Dinner \(locked\)/)).not.toHaveLength(0);
  expect(screen.getByText("The current saved week and your draft can be combined without replacing the same changed slot. Review the updated summary, then explicitly save again.")).toBeInTheDocument();
  await user.click(screen.getByRole("button", { name: "Rebase my changes for review" }));
  expect(screen.queryByRole("button", { name: "Save week" })).not.toBeInTheDocument();
  expect(screen.getByText("No meal or lock changes from the saved week.")).toBeInTheDocument();
  expect(applyPlan).toHaveBeenCalledTimes(1);
});
it("does not merge overlapping stale edits and lets the user explicitly load the current week", async () => {
  const date = localDateString(new Date());
  const base: PlanDraft = { occurrences: [{ date, slotName: "dinner", recipeId: "old", recipeName: "Old dinner", locked: false, reason: "Saved" }], unresolved: [], inputReferences: [], runId: "base", seed: 1, currentRevision: 2n };
  const latest: PlanDraft = { occurrences: [{ date, slotName: "dinner", recipeId: "new", recipeName: "New dinner", locked: false, reason: "Changed elsewhere" }], unresolved: [], inputReferences: [], runId: "latest", seed: 2, currentRevision: 3n };
  getPlan.mockResolvedValueOnce({ draft: base, hasPlan: true }).mockResolvedValueOnce({ draft: latest, hasPlan: true });
  applyPlan.mockRejectedValueOnce(new Error("plan changed elsewhere"));
  renderWithProviders(<WeekPage />);
  const user = userEvent.setup();
  await user.click(await screen.findByRole("button", { name: "Lock" }));
  await user.click(screen.getByRole("button", { name: "Save week" }));
  await user.click(await screen.findByRole("button", { name: "Review latest week" }));
  expect(await screen.findByText(/Both drafts changed these same slots/)).toBeInTheDocument();
  expect(screen.queryByRole("button", { name: "Rebase my changes for review" })).not.toBeInTheDocument();
  await user.click(screen.getByRole("button", { name: "Use latest saved week" }));
  expect(screen.getByText("New dinner")).toBeInTheDocument();
  expect(screen.queryByText(/Plan changed while you were reviewing/)).not.toBeInTheDocument();
});
it("requires a fresh shopping preview after a stale swap is rebased", async () => {
  const date = localDateString(new Date());
  const saved: PlanDraft = { occurrences: [{ date, slotName: "dinner", recipeId: "old", recipeName: "Old bowl", locked: false, reason: "Saved" }], unresolved: [], inputReferences: [], runId: "saved", seed: 1, currentRevision: 4n };
  getPlan.mockResolvedValueOnce({ draft: saved, hasPlan: true }).mockResolvedValueOnce({ draft: saved, hasPlan: true });
  listRecipes.mockResolvedValue([{ id: "new", name: "New soup" }]);
  const swap = (revision: bigint) => ({ revision, preview: { draft: { ...saved, occurrences: [{ ...saved.occurrences[0]!, recipeId: "new", recipeName: "New soup" }] }, changes: [{ date, slotName: "dinner", beforeName: "Old bowl", afterName: "New soup" }], shoppingChanges: [{ key: "ingredient:rice", after: { key: "ingredient:rice", label: "rice", need: "1 bag", stock: "unknown", missing: "unknown", packageCount: "unknown", price: "unknown", sourceRecipeIds: ["new"], checked: false } }] }, affectedDates: [date] });
  previewSwap.mockResolvedValueOnce(swap(4n)).mockResolvedValueOnce(swap(4n));
  applyPlan.mockRejectedValueOnce(new Error("aborted: shopping impact input changed; refresh the review")).mockResolvedValueOnce({ revision: 5n });
  const user = userEvent.setup();
  renderWithProviders(<WeekPage />);
  await user.click(await screen.findByRole("button", { name: "Swap meal" }));
  await user.selectOptions(screen.getByLabelText("Replacement meal"), "new");
  await user.click(screen.getByRole("button", { name: "Preview swap" }));
  await user.click(await screen.findByRole("button", { name: "Apply reviewed changes" }));
  await user.click(await screen.findByRole("button", { name: "Review latest week" }));
  await user.click(await screen.findByRole("button", { name: "Rebase my changes for review" }));
  await user.click(await screen.findByRole("button", { name: "Refresh swap and shopping review" }));
  expect(await screen.findByRole("region", { name: "Shopping impact" })).toHaveTextContent("Price: unknown");
  expect(screen.getByRole("button", { name: "Apply reviewed changes" })).toBeInTheDocument();
  await user.click(screen.getByRole("button", { name: "Apply reviewed changes" }));
  await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
  expect(applyPlan).toHaveBeenLastCalledWith(expect.objectContaining({ expectedRevision: 4n }));
});
it("keeps stale swap review available when refreshing impact fails", async () => {
  const date = localDateString(new Date());
  const saved: PlanDraft = { occurrences: [{ date, slotName: "dinner", recipeId: "old", recipeName: "Old bowl", locked: false, reason: "Saved" }], unresolved: [], inputReferences: [], runId: "saved", seed: 1, currentRevision: 4n };
  getPlan.mockResolvedValueOnce({ draft: saved, hasPlan: true }).mockResolvedValueOnce({ draft: saved, hasPlan: true });
  listRecipes.mockResolvedValue([{ id: "new", name: "New soup" }]);
  previewSwap.mockResolvedValueOnce({ revision: 4n, preview: { draft: { ...saved, occurrences: [{ ...saved.occurrences[0]!, recipeId: "new", recipeName: "New soup" }] }, changes: [{ date, slotName: "dinner", beforeName: "Old bowl", afterName: "New soup" }], shoppingChanges: [{ key: "ingredient:rice", after: { key: "ingredient:rice", label: "rice", need: "1 bag", stock: "unknown", missing: "unknown", packageCount: "unknown", price: "unknown", sourceRecipeIds: ["new"], checked: false } }] }, affectedDates: [date] }).mockRejectedValueOnce("offline");
  applyPlan.mockRejectedValueOnce(new Error("revision changed elsewhere"));
  const user = userEvent.setup();
  renderWithProviders(<WeekPage />);
  await user.click(await screen.findByRole("button", { name: "Swap meal" }));
  await user.selectOptions(screen.getByLabelText("Replacement meal"), "new");
  await user.click(screen.getByRole("button", { name: "Preview swap" }));
  await user.click(await screen.findByRole("button", { name: "Apply reviewed changes" }));
  await user.click(await screen.findByRole("button", { name: "Review latest week" }));
  await user.click(await screen.findByRole("button", { name: "Rebase my changes for review" }));
  await user.click(await screen.findByRole("button", { name: "Refresh swap and shopping review" }));
  expect(await screen.findByRole("alert")).toHaveTextContent("Unable to refresh the swap impact.");
  expect(screen.getByRole("region", { name: "Swap needs fresh review" })).toBeInTheDocument();
});

it("requires a full impact review when a swap affects another date without changing groceries", async () => {
  const date = localDateString(new Date());
  const nextDate = localDateString(new Date(new Date(`${date}T12:00:00`).setDate(new Date(`${date}T12:00:00`).getDate() + 1)));
  generatePlan.mockResolvedValue({ occurrences: [{ date, slotName: "dinner", recipeId: "old", recipeName: "Old bowl", locked: false, reason: "Saved" }], unresolved: [], currentRevision: 4n });
  listRecipes.mockResolvedValue([{ id: "new", name: "New soup" }]);
  previewSwap.mockResolvedValue({ revision: 4n, preview: { draft: { occurrences: [{ date, slotName: "dinner", recipeId: "new", recipeName: "New soup", locked: false, reason: "Swap" }], unresolved: [], inputReferences: [], runId: "swap", seed: 1, currentRevision: 4n }, changes: [{ date, slotName: "dinner", beforeName: "Old bowl", afterName: "New soup" }], shoppingChanges: [] }, affectedDates: [date, nextDate] });
  const user = userEvent.setup();
  renderWithProviders(<WeekPage />);
  await user.click(await screen.findByRole("button", { name: "Swap meal" }));
  await user.selectOptions(screen.getByLabelText("Replacement meal"), "new");
  await user.click(screen.getByRole("button", { name: "Preview swap" }));
  expect(await screen.findByRole("region", { name: "Shopping impact" })).toHaveTextContent(`dates affected: ${date}, ${nextDate}`);
  expect(screen.getByRole("button", { name: "Apply reviewed changes" })).toBeInTheDocument();
});
it("switches between persisted daily nutrition evidence and unknown weekly time and cost", async () => {
  const date = localDateString(new Date());
  vi.stubGlobal("innerWidth", 1280);
  listIntakes.mockResolvedValue([{ date, nutrientId: "protein", amount: "18", unit: "g" }]);
  scheduleAppliesOnLocalDate.mockImplementation((_schedule, checkDate) => checkDate === date);
  listSupplementSchedules.mockResolvedValue([{ id: "schedule", revision: 1n, productRevisionId: "confirmed:1", dose: "1", doseUnit: "capsule", confirmed: true, paused: false, startDate: date, endDate: "", weekdays: [] }]);
  getPlan.mockResolvedValue({ draft: { occurrences: [{ date, slotName: "dinner", recipeId: "meal", recipeName: "Saved dinner", locked: false, reason: "Persisted" }], unresolved: [{ date: "2099-12-31", code: "none", message: "No eligible dinner" }], currentRevision: 3n }, hasPlan: true });
  const user = userEvent.setup();
  renderWithProviders(<WeekPage />, { routerEntries: ["/week?view=nutrition&range=day"] });
  expect(await screen.findByRole("list", { name: "Daily nutrition scopes" })).toHaveTextContent("Planned: Saved dinner");
  expect(screen.getByRole("list", { name: "Daily nutrition scopes" })).toHaveTextContent("Recorded: protein 18 g");
  expect(screen.getByRole("list", { name: "Daily nutrition scopes" })).toHaveTextContent("catalog revision is not linked; nutrient amount remains unknown");
  await user.click(screen.getByRole("button", { name: "All week" }));
  expect(screen.getByRole("list", { name: "Daily nutrition scopes" }).querySelectorAll("li")).toHaveLength(7);
  await user.click(screen.getByRole("button", { name: "Time & cost" }));
  expect(screen.getByRole("list", { name: "Daily time and cost coverage" })).toHaveTextContent("Portion cost, checkout total, and actual spend: unknown");
  expect(screen.getByRole("list", { name: "Daily time and cost coverage" })).toHaveTextContent("Prep and cook time: unknown");
  expect(screen.getByRole("button", { name: "All week" })).toHaveAttribute("aria-pressed", "true");
  vi.unstubAllGlobals();
});
it("reviews additions, removals, slot states, lock changes, and unresolved dates after replan", async () => {
  const date = localDateString(new Date());
  const saved: PlanDraft = { occurrences: [
    { date, slotName: "dinner", recipeId: "before", recipeName: "Before", locked: false, mode: "flexible", quantity: "1", reason: "Saved" },
    { date, slotName: "snack", recipeId: "removed", recipeName: "Removed snack", locked: false, reason: "Saved" },
    { date, slotName: "lunch", recipeId: "gone", recipeName: "Gone lunch", locked: false, reason: "Saved" },
  ], unresolved: [], inputReferences: [], runId: "saved", seed: 1, currentRevision: 2n };
  const replanned: PlanDraft = { occurrences: [
    { date, slotName: "dinner", recipeId: "after", recipeName: "After", locked: true, mode: "fixed", quantity: "2", reason: "Replanned" },
    { date, slotName: "snack", recipeId: "", recipeName: "", locked: false, mode: "open", reason: "Kept open" },
    { date, slotName: "brunch", recipeId: "", recipeName: "", locked: false, mode: "open", reason: "New open slot" },
  ], unresolved: [{ date: "2099-12-31", code: "no_candidate", message: "No eligible meal" }], inputReferences: [], runId: "new", seed: 2, currentRevision: 2n };
  getPlan.mockResolvedValue({ draft: saved, hasPlan: true });
  generatePlan.mockResolvedValue(replanned);
  const user = userEvent.setup();
  renderWithProviders(<WeekPage />);
  await user.click(await screen.findByRole("button", { name: "Replan week" }));
  expect(await screen.findByText(/dinner: Before → After \(locked\)/)).toBeInTheDocument();
  await user.click(screen.getByText("Show full-week changes and unresolved dates"));
  expect(screen.getByText(/lunch: removed Gone lunch/)).toBeInTheDocument();
  expect(screen.getByText(/snack: Removed snack → open slot/)).toBeInTheDocument();
  expect(screen.getByText(/brunch: added open slot/)).toBeInTheDocument();
  expect(screen.getByText(/No eligible meal/)).toBeInTheDocument();
});
it("reports a failed weekly export and restores the export action", async () => { getPlan.mockResolvedValue({ draft: { occurrences: [], unresolved: [], currentRevision: 2n }, hasPlan: true }); exportWeeklyPDF.mockRejectedValue("offline"); renderWithProviders(<WeekPage />); await userEvent.click(await screen.findByRole("button", { name: "Download weekly PDF" })); expect(await screen.findByRole("alert")).toHaveTextContent("Unable to export this week."); expect(screen.getByRole("button", { name: "Download weekly PDF" })).toBeEnabled(); });

it("restores the linked Week view and range, and keeps them in the URL when changed", async () => {
  getPlan.mockResolvedValue({ draft: { occurrences: [], unresolved: [], currentRevision: 0n }, hasPlan: false });
  renderWithProviders(<><WeekPage /><CurrentSearch /></>, { routerEntries: ["/week?view=nutrition&range=day"] });
  const nutrition = await screen.findByRole("button", { name: "Nutrition" });
  expect(nutrition).toHaveAttribute("aria-pressed", "true");
  expect(screen.getByRole("button", { name: "Day" })).toHaveAttribute("aria-pressed", "true");
  expect(screen.getByRole("list", { name: "Daily nutrition scopes" })).toBeInTheDocument();
  await userEvent.click(screen.getByRole("button", { name: "Time & cost" }));
  expect(screen.getByRole("button", { name: "Time & cost" })).toHaveAttribute("aria-pressed", "true");
  expect(screen.getByTestId("current-search")).toHaveTextContent("view=time-cost");
  expect(screen.getByTestId("current-search")).toHaveTextContent("range=day");
});

it("renders the seven-date meal board on desktop and keeps phone agenda markup out of that layout", async () => {
  const previousWidth = window.innerWidth;
  Object.defineProperty(window, "innerWidth", { configurable: true, value: 1440 });
  const date = localDateString(new Date());
  getPlan.mockResolvedValue({ draft: { occurrences: [{ date, slotName: "dinner", recipeId: "r1", recipeName: "Dinner bowl", locked: true, reason: "Kept" }], unresolved: [], currentRevision: 2n }, hasPlan: true });
  renderWithProviders(<WeekPage />);
  expect(await screen.findByRole("table", { name: "Meals planned across the week" })).toBeInTheDocument();
  expect(screen.getByRole("button", { name: "Swap meal" })).toBeInTheDocument();
  expect(screen.queryByRole("list", { name: "Meals by day" })).not.toBeInTheDocument();
  Object.defineProperty(window, "innerWidth", { configurable: true, value: previousWidth });
});
