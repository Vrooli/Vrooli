import { create } from "@bufbuild/protobuf";
import { CatalogRevisionSchema, NutrientValueSchema } from "@vrooli/proto-types/nutrition-planner/v1/catalog/catalog_pb";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useLocation } from "react-router-dom";
import { renderWithProviders } from "../../test-utils";
const generatePlan = vi.hoisted(() => vi.fn());
const getPlan = vi.hoisted(() => vi.fn());
const applyPlan = vi.hoisted(() => vi.fn());
const recordFeedback = vi.hoisted(() => vi.fn());
const undoFeedback = vi.hoisted(() => vi.fn());
const previewSwap = vi.hoisted(() => vi.fn());
const listRecipes = vi.hoisted(() => vi.fn());
const listCatalogRevisions = vi.hoisted(() => vi.fn().mockResolvedValue([]));
const getRecipe = vi.hoisted(() => vi.fn());
const listSupplementSchedules = vi.hoisted(() => vi.fn().mockResolvedValue([]));
vi.mock("../../api/planning", () => ({ generatePlan, getPlan, applyPlan, previewSwap, recordFeedback, undoFeedback }));
vi.mock("../../api/recipes", () => ({ listRecipes, getRecipe }));
vi.mock("../../api/catalog", () => ({ listCatalogRevisions }));
vi.mock("../../api/supplement", () => ({ listSupplementSchedules, scheduleAppliesOnLocalDate: (schedule: { confirmed: boolean; paused: boolean; startDate: string; endDate: string; weekdays: number[] }, date: string) => schedule.confirmed && !schedule.paused && date >= schedule.startDate && (!schedule.endDate || date <= schedule.endDate) && schedule.weekdays.includes(new Date(`${date}T00:00:00Z`).getUTCDay()) }));
vi.mock("../../api/supplement", () => ({ listSupplementSchedules, scheduleAppliesOnLocalDate: (schedule: { confirmed: boolean; paused: boolean; startDate: string; endDate: string; weekdays: number[] }, date: string) => schedule.confirmed && !schedule.paused && date >= schedule.startDate && (!schedule.endDate || date <= schedule.endDate) && schedule.weekdays.includes(new Date(`${date}T00:00:00Z`).getUTCDay()) }));
vi.mock("../../api/workspace", () => ({ ensureWorkspace: vi.fn().mockResolvedValue({ id: "w1" }) }));
beforeEach(() => { listCatalogRevisions.mockResolvedValue([]); getPlan.mockImplementation(async (input) => ({ draft: await generatePlan(input), hasPlan: true })); });
import { TodayPage } from "./TodayPage";

function CurrentPath() { return <output data-testid="current-path">{useLocation().pathname}</output>; }
const renderToday = () => renderWithProviders(<><TodayPage /><CurrentPath /></>, { routerEntries: ["/today"] });
describe("TodayPage", () => { beforeEach(() => { vi.clearAllMocks(); listCatalogRevisions.mockResolvedValue([]); listSupplementSchedules.mockResolvedValue([]); getPlan.mockImplementation(async (input) => ({ draft: await generatePlan(input), hasPlan: true })); }); it("shows a persisted empty state without generating during mount", async () => { getPlan.mockResolvedValue({ draft: { occurrences: [], unresolved: [], currentRevision: 0n }, hasPlan: false }); renderToday(); await waitFor(() => expect(screen.getByRole("button", { name: "Draft today’s meals" })).toBeInTheDocument()); expect(getPlan).toHaveBeenCalled(); expect(generatePlan).not.toHaveBeenCalled(); expect(screen.queryByText("$0.00")).not.toBeInTheDocument(); }); it("requests full-day slots only after the user starts a draft", async () => { getPlan.mockResolvedValue({ draft: { occurrences: [], unresolved: [], currentRevision: 0n }, hasPlan: false }); generatePlan.mockResolvedValue({ occurrences: [], unresolved: [] }); renderToday(); await userEvent.click(await screen.findByRole("button", { name: "Draft today’s meals" })); await waitFor(() => expect(generatePlan).toHaveBeenCalledTimes(1)); const request = generatePlan.mock.calls[0]?.[0] as { mealSlots: { slotName: string }[] }; expect(request.mealSlots.map((slot) => slot.slotName)).toEqual(["breakfast", "lunch", "dinner", "snack"]); }); it("keeps a persisted open slot user-controlled", async () => { generatePlan.mockResolvedValue({ occurrences: [{ date: "2026-09-21", slotName: "snack", recipeName: "", locked: false, reason: "Open slot remains user-controlled" }], unresolved: [] }); renderToday(); await waitFor(() => expect(screen.getByRole("heading", { name: "Open slot" })).toBeInTheDocument()); expect(screen.getByText(/user-controlled/)).toBeInTheDocument(); }); it("shows persisted meals without fabricated metrics", async () => { generatePlan.mockResolvedValue({ occurrences: [{ date: "2026-09-21", recipeName: "Bowl", locked: false, reason: "Known fit" }], unresolved: [] }); renderToday(); await waitFor(() => expect(screen.getByRole("heading", { name: "Bowl" })).toBeInTheDocument()); expect(screen.getAllByText(/Unknown until entered/)).toHaveLength(4); }); });
it("uses the persisted week snapshot for the seven-day strip and links each occurrence to its date", async () => {
  const today = new Date();
  const dateOf = (value: Date) => `${value.getFullYear()}-${String(value.getMonth() + 1).padStart(2, "0")}-${String(value.getDate()).padStart(2, "0")}`;
  const todayDate = dateOf(today);
  const tomorrow = new Date(today); tomorrow.setDate(tomorrow.getDate() + 1);
  const tomorrowDate = dateOf(tomorrow);
  const weekEnd = new Date(today); weekEnd.setDate(weekEnd.getDate() + 6);
  const weekEndDate = dateOf(weekEnd);
  getPlan.mockImplementation(async ({ fromDate }: { fromDate: string } = { fromDate: todayDate }) => ({
    draft: { occurrences: [
      { date: todayDate, slotName: "dinner", recipeId: "dinner", recipeName: "Today's bowl", locked: false, reason: "Saved today" },
      { date: tomorrowDate, slotName: "lunch", recipeId: "lunch", recipeName: "Tomorrow's soup", locked: false, reason: "Saved ahead" },
    ].filter((item) => item.date >= fromDate), unresolved: [], currentRevision: 4n },
    hasPlan: true,
  }));
  renderToday();
  const strip = await screen.findByRole("list", { name: "Your week" });
  const tomorrowMeal = screen.getByRole("link", { name: /Tomorrow's soup/ });
  expect(strip).toContainElement(tomorrowMeal);
  expect(tomorrowMeal).toHaveAttribute("href", `/week?range=day&date=${tomorrowDate}`);
  expect(getPlan).toHaveBeenCalledWith(expect.objectContaining({ fromDate: todayDate, toDate: todayDate }));
  expect(getPlan).toHaveBeenCalledWith(expect.objectContaining({ fromDate: todayDate, toDate: weekEndDate }));
});
it("keeps Today available when week and supplement reads fail", async () => {
  getPlan.mockImplementationOnce(async (input) => ({ draft: await generatePlan(input), hasPlan: true })).mockRejectedValueOnce(new Error("week snapshot unavailable"));
  listSupplementSchedules.mockRejectedValue(new Error("schedule service unavailable"));
  renderToday();
  expect(await screen.findByRole("heading", { name: "Your day" })).toBeInTheDocument();
  expect(await screen.findByText("The rest of this week is unavailable right now.")).toBeInTheDocument();
  expect(screen.getByText("Schedule status is unavailable; today’s supplement expectations are unknown.")).toBeInTheDocument();
});
it("shows a safe error when drafting fails without an Error object", async () => { getPlan.mockResolvedValue({ draft: { occurrences: [], unresolved: [], currentRevision: 0n }, hasPlan: false }); generatePlan.mockRejectedValue("offline"); renderToday(); await userEvent.click(await screen.findByRole("button", { name: "Draft today’s meals" })); expect(await screen.findByRole("alert")).toHaveTextContent("Unable to draft today’s plan."); });
it("shows the server message when drafting fails with an Error", async () => { getPlan.mockResolvedValue({ draft: { occurrences: [], unresolved: [], currentRevision: 0n }, hasPlan: false }); generatePlan.mockRejectedValue(new Error("planner unavailable")); renderToday(); await userEvent.click(await screen.findByRole("button", { name: "Draft today’s meals" })); expect(await screen.findByRole("alert")).toHaveTextContent("planner unavailable"); });
describe("TodayPage actions", () => { it("applies the visible draft", async () => { generatePlan.mockResolvedValue({ occurrences: [{ date: "2026-09-21", recipeName: "Bowl", locked: false, reason: "Known fit" }], unresolved: [] }); applyPlan.mockResolvedValue({ revision: 1n }); renderToday(); await waitFor(() => expect(screen.getByRole("heading", { name: "Bowl" })).toBeInTheDocument()); await userEvent.click(screen.getByRole("button", { name: "Let’s make it" })); await waitFor(() => expect(screen.getByText("This plan is saved for the workspace.")).toBeInTheDocument()); expect(applyPlan).toHaveBeenCalled(); }); it("surfaces an apply failure", async () => { generatePlan.mockResolvedValue({ occurrences: [{ date: "2026-09-21", recipeName: "Bowl", locked: false, reason: "Known fit" }], unresolved: [] }); applyPlan.mockRejectedValue(new Error("stale draft")); renderToday(); await waitFor(() => expect(screen.getByRole("heading", { name: "Bowl" })).toBeInTheDocument()); await userEvent.click(screen.getByRole("button", { name: "Let’s make it" })); await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("stale draft")); }); });

it("keeps a stale day draft for review and makes loading the latest saved day explicit", async () => {
  const date = new Date().toISOString().slice(0, 10);
  const baseline = { occurrences: [{ date, slotName: "dinner", recipeId: "old", recipeName: "Old bowl", locked: false, reason: "Saved" }], unresolved: [], currentRevision: 4n };
  const latest = { occurrences: [{ date, slotName: "dinner", recipeId: "new", recipeName: "New soup", locked: false, reason: "Saved elsewhere" }], unresolved: [], currentRevision: 5n };
  getPlan.mockResolvedValueOnce({ draft: baseline, hasPlan: true }).mockResolvedValueOnce({ draft: latest, hasPlan: true }).mockResolvedValueOnce({ draft: { ...latest, occurrences: [{ ...latest.occurrences[0], recipeName: "New soup" }] }, hasPlan: true });
  applyPlan.mockRejectedValueOnce(new Error("revision changed elsewhere"));
  renderToday();
  await userEvent.click(await screen.findByRole("button", { name: "Let’s make it" }));
  expect(await screen.findByRole("alert")).toHaveTextContent("revision changed elsewhere");
  expect(screen.getByRole("button", { name: "Let’s make it" })).toBeDisabled();
  await userEvent.click(screen.getByRole("button", { name: "Review latest day" }));
  expect(await screen.findByText(/Old bowl → New soup/)).toBeInTheDocument();
  await userEvent.click(screen.getByRole("button", { name: "Use latest saved day" }));
  expect(screen.getByRole("heading", { name: "New soup" })).toBeInTheDocument();
  expect(screen.queryByRole("region", { name: "Stale day review" })).not.toBeInTheDocument();
});
it("does not show stale recovery for an ordinary save error", async () => {
  const date = new Date().toISOString().slice(0, 10);
  getPlan.mockResolvedValue({ draft: { occurrences: [{ date, slotName: "dinner", recipeId: "dinner", recipeName: "Dinner", locked: false, reason: "Saved" }], unresolved: [], currentRevision: 2n }, hasPlan: true });
  applyPlan.mockRejectedValue(new Error("storage unavailable"));
  renderToday();
  await userEvent.click(await screen.findByRole("button", { name: "Let’s make it" }));
  expect(await screen.findByRole("alert")).toHaveTextContent("storage unavailable");
  expect(screen.queryByRole("region", { name: "Stale day review" })).not.toBeInTheDocument();
  expect(screen.getByRole("button", { name: "Let’s make it" })).toBeEnabled();
});
it("lists added and removed occurrences from a stale saved day", async () => {
  const date = new Date().toISOString().slice(0, 10);
  const baseline = { occurrences: [
    { date, slotName: "dinner", recipeId: "dinner", recipeName: "Dinner", locked: false, reason: "Saved" },
    { date, slotName: "snack", recipeId: "snack", recipeName: "Snack", locked: false, reason: "Saved" },
  ], unresolved: [], currentRevision: 4n };
  const latest = { occurrences: [
    baseline.occurrences[0],
    { date, slotName: "brunch", recipeId: "brunch", recipeName: "Brunch", locked: false, reason: "Added elsewhere" },
  ], unresolved: [], currentRevision: 5n };
  getPlan.mockResolvedValueOnce({ draft: baseline, hasPlan: true }).mockResolvedValueOnce({ draft: latest, hasPlan: true }).mockResolvedValueOnce({ draft: latest, hasPlan: true });
  applyPlan.mockRejectedValueOnce(new Error("revision changed elsewhere"));
  renderToday();
  await userEvent.click(await screen.findByRole("button", { name: "Let’s make it" }));
  await userEvent.click(await screen.findByRole("button", { name: "Review latest day" }));
  expect(await screen.findByText(new RegExp(`${date} · brunch: added Brunch`))).toBeInTheDocument();
  expect(screen.getByText(new RegExp(`${date} · snack: removed Snack`))).toBeInTheDocument();
  expect(screen.getByText("No local occurrence changes.")).toBeInTheDocument();
});
it("replans from the current day while preserving its custom slot and lock", async () => {
  const date = new Date().toISOString().slice(0, 10);
  const current = { occurrences: [{ date, slotName: "brunch", mode: "fixed", quantity: "2", recipeId: "locked-recipe", recipeName: "Oats", locked: true, reason: "Saved" }], unresolved: [], currentRevision: 5n };
  getPlan.mockResolvedValueOnce({ draft: current, hasPlan: true }).mockResolvedValueOnce({ draft: current, hasPlan: true }).mockResolvedValueOnce({ draft: current, hasPlan: true });
  applyPlan.mockRejectedValueOnce(new Error("revision changed elsewhere"));
  generatePlan.mockResolvedValue({ ...current, occurrences: [{ ...current.occurrences[0], recipeName: "New oats" }] });
  renderToday();
  await userEvent.click(await screen.findByRole("button", { name: "Let’s make it" }));
  await userEvent.click(await screen.findByRole("button", { name: "Review latest day" }));
  await userEvent.click(await screen.findByRole("button", { name: "Replan from latest day" }));
  const input = generatePlan.mock.calls.at(-1)?.[0] as { mealSlots: { date: string; slotName: string; mode: string; quantity: string; lockedRecipeId: string }[] };
  expect(input.mealSlots).toEqual([{ date, slotName: "brunch", mode: "fixed", quantity: "2", lockedRecipeId: "locked-recipe" }]);
});
it("refreshes a stale swap against the latest saved day before showing its shopping review again", async () => {
  const date = new Date().toISOString().slice(0, 10);
  const baseline = { occurrences: [{ date, slotName: "dinner", recipeId: "old", recipeName: "Old bowl", locked: false, reason: "Saved" }], unresolved: [], currentRevision: 4n };
  const latest = { occurrences: [{ date, slotName: "dinner", recipeId: "current", recipeName: "Current soup", locked: false, reason: "Saved elsewhere" }], unresolved: [], currentRevision: 5n };
  getPlan.mockResolvedValueOnce({ draft: baseline, hasPlan: true }).mockResolvedValueOnce({ draft: latest, hasPlan: true }).mockResolvedValueOnce({ draft: latest, hasPlan: true });
  listRecipes.mockResolvedValue([{ id: "replacement", name: "Replacement" }]);
  const preview = (revision: bigint, recipeId: string, recipeName: string) => ({ revision, preview: { draft: { ...baseline, currentRevision: revision, occurrences: [{ ...baseline.occurrences[0], recipeId, recipeName }] }, changes: [{ date, slotName: "dinner", beforeName: "Old bowl", afterName: recipeName }], shoppingChanges: [{ key: "ingredient:rice", after: { key: "ingredient:rice", label: "rice", need: "1 bag", stock: "unknown", missing: "unknown", packageCount: "unknown", price: "unknown", sourceRecipeIds: [recipeId], checked: false } }] }, affectedDates: [date] });
  previewSwap.mockResolvedValueOnce(preview(4n, "replacement", "Replacement")).mockResolvedValueOnce(preview(5n, "replacement", "Replacement"));
  applyPlan.mockRejectedValueOnce(new Error("aborted: shopping impact input changed; refresh the review")).mockResolvedValueOnce({ revision: 6n });
  renderToday();
  await userEvent.click(await screen.findByRole("button", { name: "Swap meal" }));
  await userEvent.selectOptions(screen.getByLabelText("Replacement meal"), "replacement");
  await userEvent.click(screen.getByRole("button", { name: "Preview swap" }));
  await userEvent.click(await screen.findByRole("button", { name: "Apply reviewed changes" }));
  await userEvent.click(await screen.findByRole("button", { name: "Review latest day" }));
  await userEvent.click(await screen.findByRole("button", { name: "Use latest saved day" }));
  await userEvent.click(await screen.findByRole("button", { name: "Refresh swap review" }));
  expect(previewSwap).toHaveBeenLastCalledWith(expect.objectContaining({ expectedRevision: 5n, replacementRecipeId: "replacement" }));
  expect(await screen.findByRole("region", { name: "Shopping impact" })).toBeInTheDocument();
  expect(screen.getByRole("button", { name: "Apply reviewed changes" })).toBeInTheDocument();
  await userEvent.click(screen.getByRole("button", { name: "Apply reviewed changes" }));
  await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
  expect(applyPlan).toHaveBeenLastCalledWith(expect.objectContaining({ expectedRevision: 5n }));
});

it("records and undoes explicit eating feedback", async () => { generatePlan.mockResolvedValue({ occurrences: [{ date: "2026-09-21", slotName: "dinner", recipeId: "r1", recipeName: "Bowl", locked: false, reason: "Known fit" }], unresolved: [] }); recordFeedback.mockResolvedValue({}); undoFeedback.mockResolvedValue({}); renderToday(); await waitFor(() => expect(screen.getByRole("button", { name: "I ate this" })).toBeInTheDocument()); await userEvent.click(screen.getByRole("button", { name: "I ate this" })); await waitFor(() => expect(screen.getByText(/Recorded as eaten/)).toBeInTheDocument()); await userEvent.click(screen.getByRole("button", { name: "Undo eaten" })); await waitFor(() => expect(screen.getByRole("button", { name: "I ate this" })).toBeInTheDocument()); });
it("surfaces a feedback failure", async () => { generatePlan.mockResolvedValue({ occurrences: [{ date: "2026-09-21", slotName: "dinner", recipeId: "r1", recipeName: "Bowl", locked: false, reason: "Known fit" }], unresolved: [] }); recordFeedback.mockRejectedValue(new Error("stale plan")); renderToday(); await waitFor(() => expect(screen.getByRole("button", { name: "I ate this" })).toBeInTheDocument()); await userEvent.click(screen.getByRole("button", { name: "I ate this" })); await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("stale plan")); });
it("previews and applies a scoped swap", async () => { generatePlan.mockResolvedValue({ currentRevision: 1n, occurrences: [{ date: "2026-09-21", slotName: "dinner", recipeId: "r1", recipeName: "Bowl", locked: false, reason: "Known fit" }], unresolved: [] }); listRecipes.mockResolvedValue([{ id: "r2", name: "Soup" }]); previewSwap.mockResolvedValue({ revision: 1n, preview: { draft: { currentRevision: 1n, occurrences: [{ date: "2026-09-21", recipeId: "r2", recipeName: "Soup", locked: false, reason: "Compatible" }], unresolved: [] }, changes: [{ date: "2026-09-21", slotName: "dinner", beforeName: "Bowl", afterName: "Soup" }], shoppingChanges: [{ key: "ingredient:rice", after: { key: "ingredient:rice", label: "rice", need: "300 g", stock: "50 g", missing: "250 g", packageCount: "2", price: "4.00 USD / 200 g", sourceRecipeIds: ["r2"], checked: false } }] }, affectedDates: ["2026-09-21"] }); applyPlan.mockResolvedValue({ revision: 2n }); renderToday(); await waitFor(() => expect(screen.getByRole("heading", { name: "Bowl" })).toBeInTheDocument()); await userEvent.click(screen.getByRole("button", { name: "Swap meal" })); await userEvent.selectOptions(screen.getByLabelText("Replacement meal"), "r2"); await userEvent.click(screen.getByRole("button", { name: "Preview swap" })); await waitFor(() => expect(screen.getByText(/dinner: Bowl.*Soup/))); expect(screen.getByRole("region", { name: "Shopping impact" })).toHaveTextContent("Added: rice"); expect(screen.getByRole("region", { name: "Shopping impact" })).toHaveTextContent("Price: 4.00 USD / 200 g"); await userEvent.click(screen.getByRole("button", { name: "Apply reviewed changes" })); await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument()); expect(previewSwap).toHaveBeenCalledWith(expect.objectContaining({ slotName: "dinner" })); expect(applyPlan).toHaveBeenCalled(); });
it("uses a short confirmation when a scoped swap has no shopping or cross-date changes", async () => { generatePlan.mockResolvedValue({ currentRevision: 1n, occurrences: [{ date: "2026-09-21", slotName: "dinner", recipeId: "r1", recipeName: "Bowl", locked: false, reason: "Known fit" }], unresolved: [] }); listRecipes.mockResolvedValue([{ id: "r2", name: "Soup" }]); previewSwap.mockResolvedValue({ revision: 1n, preview: { draft: { currentRevision: 1n, occurrences: [{ date: "2026-09-21", slotName: "dinner", recipeId: "r2", recipeName: "Soup", locked: false, reason: "Compatible" }], unresolved: [] }, changes: [{ date: "2026-09-21", slotName: "dinner", beforeName: "Bowl", afterName: "Soup" }], shoppingChanges: [] }, affectedDates: ["2026-09-21"] }); applyPlan.mockResolvedValue({ revision: 2n }); renderToday(); await waitFor(() => expect(screen.getByRole("heading", { name: "Bowl" })).toBeInTheDocument()); await userEvent.click(screen.getByRole("button", { name: "Swap meal" })); await userEvent.selectOptions(screen.getByLabelText("Replacement meal"), "r2"); await userEvent.click(screen.getByRole("button", { name: "Preview swap" })); expect(await screen.findByText("This preview changes one meal slot and no shopping lines.")).toBeInTheDocument(); expect(screen.queryByRole("region", { name: "Shopping impact" })).not.toBeInTheDocument(); expect(screen.getByRole("button", { name: "Confirm swap" })).toBeInTheDocument(); });
it("requires a full impact review when a scoped swap leaves prepared portions available", async () => {
  const date = new Date().toISOString().slice(0, 10);
  applyPlan.mockClear();
  generatePlan.mockResolvedValue({ currentRevision: 1n, occurrences: [{ date, slotName: "dinner", recipeId: "r1", recipeName: "Beans", locked: false, reason: "Saved" }], unresolved: [] });
  listRecipes.mockResolvedValue([{ id: "r2", name: "Soup" }]);
  previewSwap.mockResolvedValue({ revision: 1n, preview: { draft: { currentRevision: 1n, occurrences: [{ date, slotName: "dinner", recipeId: "r2", recipeName: "Soup", locked: false, reason: "Swap" }], unresolved: [] }, changes: [{ date, slotName: "dinner", beforeName: "Beans", afterName: "Soup" }], relatedOccurrences: [], preparedBatchImpacts: [{ batchId: "beans-batch", recipeId: "r1", recipeName: "Beans", recipeRevision: 2, available: "1", unit: "serving" }], shoppingChanges: [] }, affectedDates: [date] });
  renderToday();
  await userEvent.click(await screen.findByRole("button", { name: "Swap meal" }));
  await userEvent.selectOptions(screen.getByLabelText("Replacement meal"), "r2");
  await userEvent.click(screen.getByRole("button", { name: "Preview swap" }));
  const impact = await screen.findByRole("region", { name: "Shopping impact" });
  expect(impact).toHaveTextContent("Prepared portions to review");
  expect(impact).toHaveTextContent("Beans · revision 2 · batch beans-batch: 1 serving available");
  expect(screen.getByRole("button", { name: "Apply reviewed changes" })).toBeInTheDocument();
  expect(applyPlan).not.toHaveBeenCalled();
});
it("requires a full impact review for a cross-date swap even when shopping lines do not change", async () => {
  const today = new Date().toISOString().slice(0, 10);
  const later = new Date(new Date(`${today}T12:00:00`).setDate(new Date(`${today}T12:00:00`).getDate() + 1)).toISOString().slice(0, 10);
  generatePlan.mockResolvedValue({ currentRevision: 1n, occurrences: [{ date: today, slotName: "dinner", recipeId: "r1", recipeName: "Bowl", locked: false, reason: "Known fit" }], unresolved: [] });
  listRecipes.mockResolvedValue([{ id: "r2", name: "Soup" }]);
  previewSwap.mockResolvedValue({ revision: 1n, preview: { draft: { currentRevision: 1n, occurrences: [{ date: today, slotName: "dinner", recipeId: "r2", recipeName: "Soup", locked: false, reason: "Compatible" }], unresolved: [] }, changes: [{ date: today, slotName: "dinner", beforeName: "Bowl", afterName: "Soup" }], shoppingChanges: [] }, affectedDates: [today, later] });
  renderToday();
  await userEvent.click(await screen.findByRole("button", { name: "Swap meal" }));
  await userEvent.selectOptions(screen.getByLabelText("Replacement meal"), "r2");
  await userEvent.click(screen.getByRole("button", { name: "Preview swap" }));
  expect(await screen.findByRole("region", { name: "Shopping impact" })).toHaveTextContent(`dates affected: ${today}, ${later}`);
  expect(screen.getByRole("button", { name: "Apply reviewed changes" })).toBeInTheDocument();
  expect(screen.queryByRole("button", { name: "Confirm swap" })).not.toBeInTheDocument();
});
it("reports replacement-list failures", async () => { generatePlan.mockResolvedValue({ occurrences: [{ date: "2026-09-21", slotName: "dinner", recipeId: "r1", recipeName: "Bowl", locked: false, reason: "Known fit" }], unresolved: [] }); listRecipes.mockRejectedValue(new Error("catalog unavailable")); renderToday(); await waitFor(() => expect(screen.getByRole("heading", { name: "Bowl" })).toBeInTheDocument()); await userEvent.click(screen.getByRole("button", { name: "Swap meal" })); await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("catalog unavailable")); });
it("reports swap preview and apply failures", async () => { generatePlan.mockResolvedValue({ currentRevision: 1n, occurrences: [{ date: "2026-09-21", slotName: "dinner", recipeId: "r1", recipeName: "Bowl", locked: false, reason: "Known fit" }], unresolved: [] }); listRecipes.mockResolvedValue([{ id: "r2", name: "Soup" }]); previewSwap.mockRejectedValueOnce(new Error("ineligible replacement")); renderToday(); await waitFor(() => expect(screen.getByRole("heading", { name: "Bowl" })).toBeInTheDocument()); await userEvent.click(screen.getByRole("button", { name: "Swap meal" })); await userEvent.selectOptions(screen.getByLabelText("Replacement meal"), "r2"); await userEvent.click(screen.getByRole("button", { name: "Preview swap" })); await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("ineligible replacement")); previewSwap.mockResolvedValueOnce({ revision: 1n, preview: { draft: { currentRevision: 1n, occurrences: [], unresolved: [] }, changes: [] } }); applyPlan.mockRejectedValueOnce(new Error("stale swap")); await userEvent.click(screen.getByRole("button", { name: "Preview swap" })); await waitFor(() => expect(screen.getByRole("button", { name: "Confirm swap" })).toBeInTheDocument()); await userEvent.click(screen.getByRole("button", { name: "Confirm swap" })); await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("stale swap")); });
it("navigates from a planned meal to its exact recipe revision", async () => { generatePlan.mockResolvedValue({ occurrences: [{ date: "2026-09-21", slotName: "dinner", recipeId: "r1", recipeName: "Bowl", locked: false, reason: "Known fit" }], unresolved: [] }); getRecipe.mockResolvedValue({ id: "r1", revision: 2n, name: "Bowl", notes: "A careful bowl", originalText: "Mix it", methods: [], requiredAppliances: [], allergenEvidence: {} }); renderToday(); await waitFor(() => expect(screen.getByRole("heading", { name: "Bowl" })).toBeInTheDocument()); await userEvent.click(screen.getByRole("button", { name: "See the recipe map" })); await waitFor(() => expect(screen.getByTestId("current-path")).toHaveTextContent("/recipes/r1/revisions/2")); expect(getRecipe).toHaveBeenCalledWith("w1", "r1"); });
it("surfaces recipe loading failures", async () => { generatePlan.mockResolvedValue({ occurrences: [{ date: "2026-09-21", slotName: "dinner", recipeId: "r1", recipeName: "Bowl", locked: false, reason: "Known fit" }], unresolved: [] }); getRecipe.mockRejectedValue(new Error("recipe unavailable")); renderToday(); await waitFor(() => expect(screen.getByRole("heading", { name: "Bowl" })).toBeInTheDocument()); await userEvent.click(screen.getByRole("button", { name: "See the recipe map" })); await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("recipe unavailable")); });
it("uses a safe fallback for unknown recipe failures", async () => { generatePlan.mockResolvedValue({ occurrences: [{ date: "2026-09-21", slotName: "dinner", recipeId: "r1", recipeName: "Bowl", locked: false, reason: "Known fit" }], unresolved: [] }); getRecipe.mockRejectedValue("unavailable"); renderToday(); await waitFor(() => expect(screen.getByRole("heading", { name: "Bowl" })).toBeInTheDocument()); await userEvent.click(screen.getByRole("button", { name: "See the recipe map" })); await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("Unable to load this recipe.")); });

it("routes Today actions to the selected persisted meal slot", async () => {
  vi.clearAllMocks();
  listSupplementSchedules.mockResolvedValue([]);
  const meals = [
    { date: "2026-09-21", slotName: "breakfast", recipeId: "oats", recipeName: "Oats", locked: false, reason: "Breakfast" },
    { date: "2026-09-21", slotName: "dinner", recipeId: "soup", recipeName: "Soup", locked: false, reason: "Dinner" },
  ];
  getPlan.mockResolvedValue({ draft: { occurrences: meals, unresolved: [], currentRevision: 4n }, hasPlan: true });
  recordFeedback.mockResolvedValue({});
  renderToday();
  await userEvent.click(await screen.findByRole("button", { name: "Select dinner actions" }));
  expect(screen.getByText(/2026-09-21 · dinner plan actions/)).toBeInTheDocument();
  await userEvent.click(screen.getByRole("button", { name: "I ate this" }));
  await waitFor(() => expect(recordFeedback).toHaveBeenCalledWith(expect.objectContaining({ date: "2026-09-21", recipeId: "soup" })));
});

describe("Today fixed supplement expectations", () => { beforeEach(() => { vi.clearAllMocks(); }); it("shows confirmed schedules due on the local date without implying consumption", async () => { const now = new Date(); const localDate = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, "0")}-${String(now.getDate()).padStart(2, "0")}`; listSupplementSchedules.mockResolvedValue([{ id: "due", revision: 1n, productRevisionId: "vitamin:p:2", dose: "1", doseUnit: "capsule", weekdays: [now.getDay()], startDate: localDate, endDate: localDate, paused: false, confirmed: true }, { id: "paused", revision: 1n, productRevisionId: "paused:p:1", dose: "2", doseUnit: "tablet", weekdays: [now.getDay()], startDate: localDate, endDate: "", paused: true, confirmed: true }]); renderToday(); expect(await screen.findByRole("list", { name: "Supplements scheduled today" })).toHaveTextContent("vitamin:p:2 · 1 capsule scheduled"); expect(screen.queryByText(/paused:p:1/)).not.toBeInTheDocument(); expect(screen.getByText(/does not record that a dose was taken/)).toBeInTheDocument(); }); });

describe("Today mapped supplement expectations", () => { beforeEach(() => { vi.clearAllMocks(); listCatalogRevisions.mockResolvedValue([]); }); it("includes pinned catalog nutrient amounts in Expected without recording a dose", async () => { const now = new Date(); const date = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, "0")}-${String(now.getDate()).padStart(2, "0")}`; listSupplementSchedules.mockResolvedValue([{ id: "s1", revision: 3n, productRevisionId: "catalog:vitamin-c:2", dose: "2", doseUnit: "capsule", weekdays: [now.getDay()], startDate: date, endDate: date, paused: false, confirmed: true, createdAt: "" }]); listCatalogRevisions.mockResolvedValue([create(CatalogRevisionSchema, { id: "vitamin-c", revision: 2n, name: "Vitamin C", productName: "Citrus tablet", servingQuantity: "1", servingUnit: "capsule", nutrients: [create(NutrientValueSchema, { nutrientId: "vitamin_c", amount: "250", unit: "mg", basis: "1", basisUnit: "capsule", evidence: "label", sourceRef: "label:2" })] })]); renderToday(); expect(await screen.findByText(/vitamin_c: 500 mg expected \(label\)/)).toBeInTheDocument(); expect(screen.getByText(/does not record that a dose was taken/)).toBeInTheDocument(); }); });

describe("Today swap dialog keyboard behavior", () => { beforeEach(() => { vi.clearAllMocks(); }); it("focuses the dialog, traps both Tab directions, ignores other keys, and restores the trigger on Escape", async () => { const user = userEvent.setup(); generatePlan.mockResolvedValue({ occurrences: [{ date: "2026-09-21", slotName: "dinner", recipeId: "r1", recipeName: "Bowl", locked: false, reason: "Known fit" }], unresolved: [] }); listRecipes.mockResolvedValue([{ id: "r2", name: "Soup" }]); renderToday(); await waitFor(() => expect(screen.getByRole("button", { name: "Swap meal" })).toBeInTheDocument()); const trigger = screen.getByRole("button", { name: "Swap meal" }); await user.click(trigger); const select = screen.getByLabelText("Replacement meal"); expect(select).toHaveFocus(); await user.keyboard("x"); expect(select).toHaveFocus(); await user.keyboard("{Shift>}{Tab}{/Shift}"); const cancel = screen.getByRole("button", { name: "Cancel" }); expect(cancel).toHaveFocus(); await user.tab(); expect(select).toHaveFocus(); await user.keyboard("{Escape}"); expect(screen.queryByRole("dialog")).not.toBeInTheDocument(); expect(trigger).toHaveFocus(); }); });

it("shows the pinned revision for unchanged later uses in Today swap review", async () => {
  const todayDate = new Date();
  const format = (value: Date) => `${value.getFullYear()}-${String(value.getMonth() + 1).padStart(2, "0")}-${String(value.getDate()).padStart(2, "0")}`;
  const today = format(todayDate);
  todayDate.setDate(todayDate.getDate() + 1);
  const later = format(todayDate);
  generatePlan.mockResolvedValue({ currentRevision: 1n, occurrences: [{ date: today, slotName: "dinner", recipeId: "r1", recipeRevision: 4, recipeName: "Beans", locked: false, reason: "Saved" }], unresolved: [] });
  listRecipes.mockResolvedValue([{ id: "r2", name: "Soup" }]);
  previewSwap.mockResolvedValue({ revision: 1n, preview: { draft: { currentRevision: 1n, occurrences: [], unresolved: [] }, changes: [{ date: today, slotName: "dinner", beforeName: "Beans", afterName: "Soup" }], relatedOccurrences: [{ date: later, slotName: "dinner", recipeRevision: 4, recipeName: "Beans", locked: false }], shoppingChanges: [] }, affectedDates: [today] });
  renderToday();
  await userEvent.click(await screen.findByRole("button", { name: "Swap meal" }));
  await userEvent.selectOptions(screen.getByLabelText("Replacement meal"), "r2");
  await userEvent.click(screen.getByRole("button", { name: "Preview swap" }));
  expect(await screen.findByText(`${later} · dinner: Beans · revision 4`)).toBeInTheDocument();
  expect(applyPlan).not.toHaveBeenCalled();
});
