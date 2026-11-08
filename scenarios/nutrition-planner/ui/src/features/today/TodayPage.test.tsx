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
const exploreRecipes = vi.hoisted(() => vi.fn().mockResolvedValue({ candidates: [], blockingReasons: [] }));
const listCatalogRevisions = vi.hoisted(() => vi.fn().mockResolvedValue([]));
const getRecipe = vi.hoisted(() => vi.fn());
const startCookingSession = vi.hoisted(() => vi.fn());
const listSupplementSchedules = vi.hoisted(() => vi.fn().mockResolvedValue([]));
vi.mock("../../api/planning", () => ({ generatePlan, getPlan, applyPlan, previewSwap, exploreRecipes, recordFeedback, undoFeedback }));
vi.mock("../../api/recipes", () => ({ getRecipe }));
vi.mock("../../api/catalog", () => ({ listCatalogRevisions }));
vi.mock("../../api/cooking", () => ({ startCookingSession }));
vi.mock("../../api/supplement", () => ({ listSupplementSchedules, scheduleAppliesOnLocalDate: (schedule: { confirmed: boolean; paused: boolean; startDate: string; endDate: string; weekdays: number[] }, date: string) => schedule.confirmed && !schedule.paused && date >= schedule.startDate && (!schedule.endDate || date <= schedule.endDate) && schedule.weekdays.includes(new Date(`${date}T00:00:00Z`).getUTCDay()) }));
vi.mock("../../api/supplement", () => ({ listSupplementSchedules, scheduleAppliesOnLocalDate: (schedule: { confirmed: boolean; paused: boolean; startDate: string; endDate: string; weekdays: number[] }, date: string) => schedule.confirmed && !schedule.paused && date >= schedule.startDate && (!schedule.endDate || date <= schedule.endDate) && schedule.weekdays.includes(new Date(`${date}T00:00:00Z`).getUTCDay()) }));
vi.mock("../../api/workspace", () => ({ ensureWorkspace: vi.fn().mockResolvedValue({ id: "w1" }) }));
beforeEach(() => { listCatalogRevisions.mockResolvedValue([]); getPlan.mockImplementation(async (input) => ({ draft: await generatePlan(input), hasPlan: true })); });
import { TodayPage } from "./TodayPage";

function CurrentPath() { return <output data-testid="current-path">{useLocation().pathname}</output>; }
const renderToday = () => renderWithProviders(<><TodayPage /><CurrentPath /></>, { routerEntries: ["/today"] });
describe("TodayPage", () => { beforeEach(() => { vi.clearAllMocks(); listCatalogRevisions.mockResolvedValue([]); listSupplementSchedules.mockResolvedValue([]); getPlan.mockImplementation(async (input) => ({ draft: await generatePlan(input), hasPlan: true })); }); it("shows a persisted empty state without generating during mount", async () => { getPlan.mockResolvedValue({ draft: { occurrences: [], unresolved: [], currentRevision: 0n }, hasPlan: false }); renderToday(); await waitFor(() => expect(screen.getByRole("button", { name: "Draft today’s meals" })).toBeInTheDocument()); expect(getPlan).toHaveBeenCalled(); expect(generatePlan).not.toHaveBeenCalled(); expect(screen.queryByText("$0.00")).not.toBeInTheDocument(); }); it("requests full-day slots only after the user starts a draft", async () => { getPlan.mockResolvedValue({ draft: { occurrences: [], unresolved: [], currentRevision: 0n }, hasPlan: false }); generatePlan.mockResolvedValue({ occurrences: [], unresolved: [] }); renderToday(); await userEvent.click(await screen.findByRole("button", { name: "Draft today’s meals" })); await waitFor(() => expect(generatePlan).toHaveBeenCalledTimes(1)); const request = generatePlan.mock.calls[0]?.[0] as { mealSlots: { slotName: string }[] }; expect(request.mealSlots.map((slot) => slot.slotName)).toEqual(["breakfast", "lunch", "dinner", "snack"]); }); it("keeps a persisted open slot user-controlled", async () => { generatePlan.mockResolvedValue({ occurrences: [{ date: "2026-09-21", slotName: "snack", recipeName: "", locked: false, reason: "Open slot remains user-controlled" }], unresolved: [] }); renderToday(); await waitFor(() => expect(screen.getByRole("heading", { name: "Open slot" })).toBeInTheDocument()); expect(screen.getByText(/user-controlled/)).toBeInTheDocument(); }); it("shows persisted meals without fabricated metrics", async () => { generatePlan.mockResolvedValue({ occurrences: [{ date: "2026-09-21", recipeName: "Bowl", locked: false, reason: "Known fit" }], unresolved: [] }); renderToday(); await waitFor(() => expect(screen.getByRole("heading", { name: "Bowl" })).toBeInTheDocument()); expect(screen.getAllByText(/Unknown until entered/)).toHaveLength(4); }); });
it("keeps Today’s meal, whole-day overview, week, then grocery action in visual reading order", async () => {
  const date = new Date().toISOString().slice(0, 10);
  getPlan.mockResolvedValue({ draft: { occurrences: [{ date, slotName: "dinner", recipeId: "dinner", recipeName: "Dinner", locked: false, reason: "Saved" }], unresolved: [], currentRevision: 2n }, hasPlan: true });
  const { container } = renderToday();
  await screen.findByRole("heading", { name: "Dinner" });
  const sections = [...container.querySelectorAll(".today-hero, .today-overview, .today-week-section, .today-shopping-footer")];
  expect(sections.map((section) => section.classList[0])).toEqual(["today-hero", "today-overview", "today-week-section", "today-shopping-footer"]);
});
it("places the truthful empty-plan action before week context", async () => {
  getPlan.mockResolvedValue({ draft: { occurrences: [], unresolved: [], currentRevision: 0n }, hasPlan: false });
  const { container } = renderToday();
  await screen.findByRole("button", { name: "Draft today’s meals" });
  const empty = container.querySelector(".today-empty-state");
  const week = container.querySelector(".today-week-section");
  expect(empty).toBeInTheDocument();
  expect(week).toBeInTheDocument();
  expect(empty!.compareDocumentPosition(week!) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
});
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
describe("TodayPage actions", () => { it("applies the visible draft without starting cooking", async () => { const date = new Date().toISOString().slice(0, 10); generatePlan.mockResolvedValue({ currentRevision: 1n, occurrences: [{ date, recipeId: "r1", recipeRevision: 2, recipeName: "Bowl", locked: false, reason: "Known fit" }], unresolved: [] }); applyPlan.mockResolvedValue({ revision: 2n }); renderToday(); await waitFor(() => expect(screen.getByRole("heading", { name: "Bowl" })).toBeInTheDocument()); expect(screen.getByRole("img", { name: "Meal photo unavailable" })).toBeInTheDocument(); expect(screen.getByRole("button", { name: "Start cooking" })).toBeInTheDocument(); await userEvent.click(screen.getByRole("button", { name: "Save/Apply plan" })); await waitFor(() => expect(screen.getByText("This plan is saved for the workspace.")).toBeInTheDocument()); expect(applyPlan).toHaveBeenCalled(); expect(startCookingSession).not.toHaveBeenCalled(); expect(screen.getByTestId("current-path")).toHaveTextContent("/today"); }); it("surfaces an apply failure", async () => { generatePlan.mockResolvedValue({ occurrences: [{ date: "2026-09-21", recipeName: "Bowl", locked: false, reason: "Known fit" }], unresolved: [] }); applyPlan.mockRejectedValue(new Error("stale draft")); renderToday(); await waitFor(() => expect(screen.getByRole("heading", { name: "Bowl" })).toBeInTheDocument()); await userEvent.click(screen.getByRole("button", { name: "Save/Apply plan" })); await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("stale draft")); }); });

it("keeps a stale day draft for review and makes loading the latest saved day explicit", async () => {
  const date = new Date().toISOString().slice(0, 10);
  const baseline = { occurrences: [{ date, slotName: "dinner", recipeId: "old", recipeName: "Old bowl", locked: false, reason: "Saved" }], unresolved: [], currentRevision: 4n };
  const latest = { occurrences: [{ date, slotName: "dinner", recipeId: "new", recipeName: "New soup", locked: false, reason: "Saved elsewhere" }], unresolved: [], currentRevision: 5n };
  getPlan.mockResolvedValueOnce({ draft: baseline, hasPlan: true }).mockResolvedValueOnce({ draft: latest, hasPlan: true }).mockResolvedValueOnce({ draft: { ...latest, occurrences: [{ ...latest.occurrences[0], recipeName: "New soup" }] }, hasPlan: true });
  applyPlan.mockRejectedValueOnce(new Error("revision changed elsewhere"));
  renderToday();
  await userEvent.click(await screen.findByRole("button", { name: "Save/Apply plan" }));
  expect(await screen.findByRole("alert")).toHaveTextContent("revision changed elsewhere");
  expect(screen.getByRole("button", { name: "Save/Apply plan" })).toBeDisabled();
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
  await userEvent.click(await screen.findByRole("button", { name: "Save/Apply plan" }));
  expect(await screen.findByRole("alert")).toHaveTextContent("storage unavailable");
  expect(screen.queryByRole("region", { name: "Stale day review" })).not.toBeInTheDocument();
  expect(screen.getByRole("button", { name: "Save/Apply plan" })).toBeEnabled();
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
  await userEvent.click(await screen.findByRole("button", { name: "Save/Apply plan" }));
  await userEvent.click(await screen.findByRole("button", { name: "Review latest day" }));
  expect(await screen.findByText(new RegExp(`${date} · brunch: added Brunch`))).toBeInTheDocument();
  expect(screen.getByText(new RegExp(`${date} · snack: removed Snack`))).toBeInTheDocument();
  expect(screen.getByText("No local occurrence changes.")).toBeInTheDocument();
});
it("replans from the current day while preserving its custom slot and lock", async () => {
  const previousTimezone = process.env.TZ;
  process.env.TZ = "America/New_York";
  const NativeDate = Date;
  const instant = NativeDate.parse("2026-10-07T02:30:00Z");
  vi.stubGlobal("Date", class extends NativeDate {
    constructor(...args: [value?: string | number | Date]) {
      if (args.length === 0) super(instant);
      else super(args[0]!);
    }
    static now() { return instant; }
  });
  try {
  const today = new Date();
  const date = `${today.getFullYear()}-${String(today.getMonth() + 1).padStart(2, "0")}-${String(today.getDate()).padStart(2, "0")}`;
  expect(date).toBe("2026-10-06");
  const current = { occurrences: [{ date, slotName: "brunch", mode: "fixed", quantity: "2", recipeId: "locked-recipe", recipeName: "Oats", locked: true, reason: "Saved" }], unresolved: [], currentRevision: 5n };
  getPlan.mockResolvedValueOnce({ draft: current, hasPlan: true }).mockResolvedValueOnce({ draft: current, hasPlan: true }).mockResolvedValueOnce({ draft: current, hasPlan: true });
  applyPlan.mockRejectedValueOnce(new Error("revision changed elsewhere"));
  generatePlan.mockResolvedValue({ ...current, occurrences: [{ ...current.occurrences[0], recipeName: "New oats" }] });
  renderToday();
  await userEvent.click(await screen.findByRole("button", { name: "Save/Apply plan" }));
  await userEvent.click(await screen.findByRole("button", { name: "Review latest day" }));
  await userEvent.click(await screen.findByRole("button", { name: "Replan from latest day" }));
  const input = generatePlan.mock.calls.at(-1)?.[0] as { mealSlots: { date: string; slotName: string; mode: string; quantity: string; lockedRecipeId: string }[] };
  expect(input.mealSlots).toEqual([{ date, slotName: "brunch", mode: "fixed", quantity: "2", lockedRecipeId: "locked-recipe" }]);
  } finally {
  vi.unstubAllGlobals();
  if (previousTimezone === undefined) delete process.env.TZ;
  else process.env.TZ = previousTimezone;
  }
});
it("refreshes a stale swap against the latest saved day before showing its shopping review again", async () => {
  const date = new Date().toISOString().slice(0, 10);
  const baseline = { occurrences: [{ date, slotName: "dinner", recipeId: "old", recipeName: "Old bowl", locked: false, reason: "Saved" }], unresolved: [], currentRevision: 4n };
  const latest = { occurrences: [{ date, slotName: "dinner", recipeId: "current", recipeName: "Current soup", locked: false, reason: "Saved elsewhere" }], unresolved: [], currentRevision: 5n };
  getPlan.mockResolvedValueOnce({ draft: baseline, hasPlan: true }).mockResolvedValueOnce({ draft: latest, hasPlan: true }).mockResolvedValueOnce({ draft: latest, hasPlan: true });
  exploreRecipes.mockResolvedValue({ candidates: [{ recipeId: "replacement", name: "Replacement", recipeRevision: 1n, fitReasons: [{ code: "eligible", rule: "current rules", reference: "profile", message: "Fits current food rules.", methodId: "" }], methodAssessments: [], summary: "" }] });
  const preview = (revision: bigint, recipeId: string, recipeName: string) => ({ revision, preview: { draft: { ...baseline, currentRevision: revision, occurrences: [{ ...baseline.occurrences[0], recipeId, recipeName }] }, changes: [{ date, slotName: "dinner", beforeName: "Old bowl", afterName: recipeName }], shoppingChanges: [{ key: "ingredient:rice", after: { key: "ingredient:rice", label: "rice", need: "1 bag", stock: "unknown", missing: "unknown", packageCount: "unknown", price: "unknown", sourceRecipeIds: [recipeId], checked: false } }] }, affectedDates: [date] });
  previewSwap.mockResolvedValueOnce(preview(4n, "replacement", "Replacement")).mockResolvedValueOnce(preview(5n, "replacement", "Replacement"));
  applyPlan.mockRejectedValueOnce(new Error("aborted: shopping impact input changed; refresh the review")).mockResolvedValueOnce({ revision: 6n });
  renderToday();
  await userEvent.click(await screen.findByRole("button", { name: "Swap meal" }));
  await userEvent.click(screen.getByRole("button", { name: "Review Replacement swap" }));
  await userEvent.click(await screen.findByRole("button", { name: "Apply reviewed changes" }));
  await userEvent.click(await screen.findByRole("button", { name: "Review latest day" }));
  await userEvent.click(await screen.findByRole("button", { name: "Use latest saved day" }));
  await userEvent.click(await screen.findByRole("button", { name: "Refresh swap review" }));
  expect(previewSwap).toHaveBeenLastCalledWith(expect.objectContaining({ expectedRevision: 5n, replacementRecipeId: "replacement" }));
  expect(await screen.findByRole("region", { name: "Swap impact" })).toBeInTheDocument();
  expect(screen.getByRole("button", { name: "Apply reviewed changes" })).toBeInTheDocument();
  await userEvent.click(screen.getByRole("button", { name: "Apply reviewed changes" }));
  await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
  expect(applyPlan).toHaveBeenLastCalledWith(expect.objectContaining({ expectedRevision: 5n }));
});

it("records and undoes explicit eating feedback", async () => { generatePlan.mockResolvedValue({ occurrences: [{ date: "2026-09-21", slotName: "dinner", recipeId: "r1", recipeName: "Bowl", locked: false, reason: "Known fit" }], unresolved: [] }); recordFeedback.mockResolvedValue({}); undoFeedback.mockResolvedValue({}); renderToday(); await waitFor(() => expect(screen.getByRole("button", { name: "I ate this" })).toBeInTheDocument()); await userEvent.click(screen.getByRole("button", { name: "I ate this" })); await waitFor(() => expect(screen.getByText(/Recorded as eaten/)).toBeInTheDocument()); await userEvent.click(screen.getByRole("button", { name: "Undo eaten" })); await waitFor(() => expect(screen.getByRole("button", { name: "I ate this" })).toBeInTheDocument()); });
it("surfaces a feedback failure", async () => { generatePlan.mockResolvedValue({ occurrences: [{ date: "2026-09-21", slotName: "dinner", recipeId: "r1", recipeName: "Bowl", locked: false, reason: "Known fit" }], unresolved: [] }); recordFeedback.mockRejectedValue(new Error("stale plan")); renderToday(); await waitFor(() => expect(screen.getByRole("button", { name: "I ate this" })).toBeInTheDocument()); await userEvent.click(screen.getByRole("button", { name: "I ate this" })); await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("stale plan")); });
it("previews and applies a scoped swap", async () => { generatePlan.mockResolvedValue({ currentRevision: 1n, occurrences: [{ date: "2026-09-21", slotName: "dinner", recipeId: "r1", recipeName: "Bowl", locked: false, reason: "Known fit" }], unresolved: [] }); exploreRecipes.mockResolvedValue({ candidates: [{ recipeId: "r2", name: "Soup", recipeRevision: 1n, fitReasons: [{ code: "eligible", rule: "current rules", reference: "profile", message: "Fits current food rules.", methodId: "" }], methodAssessments: [], summary: "" }] }); previewSwap.mockResolvedValue({ revision: 1n, preview: { draft: { currentRevision: 1n, occurrences: [{ date: "2026-09-21", recipeId: "r2", recipeName: "Soup", locked: false, reason: "Compatible" }], unresolved: [] }, changes: [{ date: "2026-09-21", slotName: "dinner", beforeName: "Bowl", afterName: "Soup" }], shoppingChanges: [{ key: "ingredient:rice", after: { key: "ingredient:rice", label: "rice", need: "300 g", stock: "50 g", missing: "250 g", packageCount: "2", price: "4.00 USD / 200 g", sourceRecipeIds: ["r2"], checked: false } }] }, affectedDates: ["2026-09-21"] }); applyPlan.mockResolvedValue({ revision: 2n }); renderToday(); await waitFor(() => expect(screen.getByRole("heading", { name: "Bowl" })).toBeInTheDocument()); await userEvent.click(screen.getByRole("button", { name: "Swap meal" })); await userEvent.click(screen.getByRole("button", { name: "Review Soup swap" })); await waitFor(() => expect(screen.getByText(/dinner: Bowl.*Soup/))); expect(screen.getByRole("region", { name: "Swap impact" })).toHaveTextContent("Added: rice"); expect(screen.getByRole("region", { name: "Swap impact" })).toHaveTextContent("Price: 4.00 USD / 200 g"); await userEvent.click(screen.getByRole("button", { name: "Apply reviewed changes" })); await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument()); expect(previewSwap).toHaveBeenCalledWith(expect.objectContaining({ slotName: "dinner" })); expect(applyPlan).toHaveBeenCalled(); });
it("shows rule-eligible swap options immediately and names unavailable measures as unknown", async () => {
  previewSwap.mockClear();
  generatePlan.mockResolvedValue({ currentRevision: 1n, occurrences: [{ date: "2026-09-21", slotName: "dinner", recipeId: "r1", recipeName: "Bowl", locked: false, reason: "Saved" }], unresolved: [] });
  exploreRecipes.mockResolvedValue({ candidates: [{ recipeId: "r2", name: "Soup", recipeRevision: 3n, fitReasons: [{ code: "method_viable", rule: "Stovetop", reference: "recipe.methods.stove", message: "This complete method meets your saved kitchen requirements.", methodId: "stove" }], methodAssessments: [], summary: "A saved dinner idea." }], blockingReasons: [{ message: "Does not meet the vegan rule." }] });
  renderToday();
  await userEvent.click(await screen.findByRole("button", { name: "Swap meal" }));
  const alternatives = await screen.findByRole("list", { name: "Suitable alternatives" });
  expect(alternatives).toHaveTextContent("Soup");
  expect(alternatives).toHaveTextContent("Active time: Time not set");
  expect(alternatives).toHaveTextContent("Protein: unknown");
  expect(alternatives).not.toHaveTextContent("Does not meet the vegan rule.");
  expect(screen.queryByLabelText("Replacement meal")).not.toBeInTheDocument();
  expect(previewSwap).not.toHaveBeenCalled();
});
it("names server-reported blockers when there are no alternative meals", async () => {
  previewSwap.mockClear();
  generatePlan.mockResolvedValue({ currentRevision: 1n, occurrences: [{ date: "2026-09-21", slotName: "dinner", recipeId: "r1", recipeName: "Bowl", locked: false, reason: "Saved" }], unresolved: [] });
  exploreRecipes.mockResolvedValue({ candidates: [], blockingReasons: [{ code: "required_group", rule: "vegan", reference: "profile", message: "No saved alternative meets the vegan rule.", methodId: "" }] });
  renderToday();
  await userEvent.click(await screen.findByRole("button", { name: "Swap meal" }));
  expect(await screen.findByRole("list", { name: "Why other saved meals were excluded" })).toHaveTextContent("vegan rule");
  expect(screen.getByRole("link", { name: "Add or review meals" })).toHaveAttribute("href", "/meals");
  expect(screen.getByRole("link", { name: "Review restrictions" })).toHaveAttribute("href", "/kitchen?tab=preferences");
  expect(previewSwap).not.toHaveBeenCalled();
});
it("lets an optional reason reorder alternatives using saved note evidence", async () => {
  generatePlan.mockResolvedValue({ currentRevision: 1n, occurrences: [{ date: "2026-09-21", slotName: "dinner", recipeId: "r1", recipeName: "Bowl", locked: false, reason: "Saved" }], unresolved: [] });
  exploreRecipes.mockResolvedValue({ candidates: [
    { recipeId: "r2", name: "Stew", recipeRevision: 1n, fitReasons: [], methodAssessments: [], summary: "A family favorite." },
    { recipeId: "r3", name: "Quick soup", recipeRevision: 1n, fitReasons: [], methodAssessments: [], summary: "Quick preparation." },
  ] });
  renderToday();
  await userEvent.click(await screen.findByRole("button", { name: "Swap meal" }));
  const alternatives = await screen.findByRole("list", { name: "Suitable alternatives" });
  const names = () => [...alternatives.querySelectorAll("h3")].map((heading) => heading.textContent);
  expect(names()).toEqual(["Stew", "Quick soup"]);
  await userEvent.click(screen.getByRole("button", { name: "Less effort (saved-text match)" }));
  expect(names()).toEqual(["Quick soup", "Stew"]);
});
it("uses the existing swap preview cost and stock deltas for supported reason chips", async () => {
  generatePlan.mockResolvedValue({ currentRevision: 1n, occurrences: [{ date: "2026-09-21", slotName: "dinner", recipeId: "r1", recipeName: "Bowl", locked: false, reason: "Saved" }], unresolved: [] });
  exploreRecipes.mockResolvedValue({ candidates: [
    { recipeId: "r2", name: "Stew", recipeRevision: 1n, fitReasons: [], methodAssessments: [], summary: "" },
    { recipeId: "r3", name: "Affordable soup", recipeRevision: 1n, fitReasons: [], methodAssessments: [], summary: "" },
  ] });
  previewSwap.mockImplementation(async ({ replacementRecipeId }: { replacementRecipeId: string }) => {
    const cheap = replacementRecipeId === "r3";
    const line = (amount: number, missing: string) => ({ key: "ingredient:beans", label: "Beans", need: "1 bag", stock: "0 g", missing, packageCount: "1", price: "unknown", portionCost: `${amount} USD`, checkoutTotal: "unknown", actualSpend: "unknown", sourceRecipeIds: [replacementRecipeId], checked: false });
    return { revision: 1n, preview: { draft: { currentRevision: 1n, occurrences: [], unresolved: [] }, changes: [], relatedOccurrences: [], preparedBatchImpacts: [], shoppingChanges: [{ key: "ingredient:beans", before: line(8, "500 g"), after: line(cheap ? 2 : 12, cheap ? "0 g" : "300 g") }] }, affectedDates: ["2026-09-21"] };
  });
  renderToday();
  await userEvent.click(await screen.findByRole("button", { name: "Swap meal" }));
  const alternatives = await screen.findByRole("list", { name: "Suitable alternatives" });
  const names = () => [...alternatives.querySelectorAll("h3")].map((heading) => heading.textContent);
  await userEvent.click(screen.getByRole("button", { name: "Lower cost" }));
  await waitFor(() => expect(names()).toEqual(["Affordable soup", "Stew"]));
  await userEvent.click(screen.getByRole("button", { name: "Missing ingredients" }));
  await waitFor(() => expect(names()).toEqual(["Affordable soup", "Stew"]));
  expect(previewSwap).toHaveBeenCalledTimes(2);
});
it("keeps the server candidate order when cost or ingredient preview evidence is unavailable", async () => {
  previewSwap.mockClear();
  generatePlan.mockResolvedValue({ currentRevision: 1n, occurrences: [{ date: "2026-09-21", slotName: "dinner", recipeId: "r1", recipeName: "Bowl", locked: false, reason: "Saved" }], unresolved: [] });
  exploreRecipes.mockResolvedValue({ candidates: [
    { recipeId: "r2", name: "Pantry stew", recipeRevision: 1n, fitReasons: [], methodAssessments: [], summary: "Affordable pantry ingredients." },
    { recipeId: "r3", name: "Soup", recipeRevision: 1n, fitReasons: [], methodAssessments: [], summary: "A saved meal." },
  ] });
  previewSwap.mockRejectedValue(new Error("preview unavailable"));
  renderToday();
  await userEvent.click(await screen.findByRole("button", { name: "Swap meal" }));
  const alternatives = await screen.findByRole("list", { name: "Suitable alternatives" });
  const names = () => [...alternatives.querySelectorAll("h3")].map((heading) => heading.textContent);
  await userEvent.click(screen.getByRole("button", { name: "Lower cost" }));
  await waitFor(() => expect(previewSwap).toHaveBeenCalledTimes(2));
  await waitFor(() => expect(names()).toEqual(["Pantry stew", "Soup"]));
  await userEvent.click(screen.getByRole("button", { name: "Missing ingredients" }));
  expect(previewSwap).toHaveBeenCalledTimes(2);
  await waitFor(() => expect(names()).toEqual(["Pantry stew", "Soup"]));
});
it("keeps the server candidate order when preview line metrics are unknown", async () => {
  previewSwap.mockClear();
  generatePlan.mockResolvedValue({ currentRevision: 1n, occurrences: [{ date: "2026-09-21", slotName: "dinner", recipeId: "r1", recipeName: "Bowl", locked: false, reason: "Saved" }], unresolved: [] });
  exploreRecipes.mockResolvedValue({ candidates: [
    { recipeId: "r2", name: "Cheap pantry stew", recipeRevision: 1n, fitReasons: [], methodAssessments: [], summary: "Affordable pantry ingredients." },
    { recipeId: "r3", name: "Soup", recipeRevision: 1n, fitReasons: [], methodAssessments: [], summary: "A saved meal." },
  ] });
  const line = (recipeId: string) => ({ key: "ingredient:beans", label: "Beans", need: "unknown", stock: "unknown", missing: "unknown", packageCount: "unknown", price: "unknown", portionCost: "unknown", checkoutTotal: "unknown", actualSpend: "unknown", sourceRecipeIds: [recipeId], checked: false });
  previewSwap.mockImplementation(async ({ replacementRecipeId }: { replacementRecipeId: string }) => ({ revision: 1n, preview: { draft: { currentRevision: 1n, occurrences: [], unresolved: [] }, changes: [], relatedOccurrences: [], preparedBatchImpacts: [], shoppingChanges: [{ key: "ingredient:beans", before: line(replacementRecipeId), after: line(replacementRecipeId) }] }, affectedDates: ["2026-09-21"] }));
  renderToday();
  await userEvent.click(await screen.findByRole("button", { name: "Swap meal" }));
  const alternatives = await screen.findByRole("list", { name: "Suitable alternatives" });
  const names = () => [...alternatives.querySelectorAll("h3")].map((heading) => heading.textContent);
  await userEvent.click(screen.getByRole("button", { name: "Lower cost" }));
  await waitFor(() => expect(previewSwap).toHaveBeenCalledTimes(2));
  expect(names()).toEqual(["Cheap pantry stew", "Soup"]);
  await userEvent.click(screen.getByRole("button", { name: "Missing ingredients" }));
  expect(previewSwap).toHaveBeenCalledTimes(2);
  expect(names()).toEqual(["Cheap pantry stew", "Soup"]);
});
it("requires impact review when target effects are absent from a scoped swap preview", async () => { generatePlan.mockResolvedValue({ currentRevision: 1n, occurrences: [{ date: "2026-09-21", slotName: "dinner", recipeId: "r1", recipeName: "Bowl", locked: false, reason: "Known fit" }], unresolved: [] }); exploreRecipes.mockResolvedValue({ candidates: [{ recipeId: "r2", name: "Soup", recipeRevision: 1n, fitReasons: [{ code: "eligible", rule: "current rules", reference: "profile", message: "Fits current food rules.", methodId: "" }], methodAssessments: [], summary: "" }] }); previewSwap.mockResolvedValue({ revision: 1n, preview: { draft: { currentRevision: 1n, occurrences: [{ date: "2026-09-21", slotName: "dinner", recipeId: "r2", recipeName: "Soup", locked: false, reason: "Compatible" }], unresolved: [] }, changes: [{ date: "2026-09-21", slotName: "dinner", beforeName: "Bowl", afterName: "Soup" }], shoppingChanges: [] }, affectedDates: ["2026-09-21"] }); renderToday(); await waitFor(() => expect(screen.getByRole("heading", { name: "Bowl" })).toBeInTheDocument()); await userEvent.click(screen.getByRole("button", { name: "Swap meal" })); await userEvent.click(screen.getByRole("button", { name: "Review Soup swap" })); expect(await screen.findByRole("region", { name: "Swap impact" })).toHaveTextContent("Nutrition target effects are not included in this preview"); expect(screen.getByRole("button", { name: "Apply reviewed changes" })).toBeInTheDocument(); expect(screen.queryByRole("button", { name: "Confirm swap" })).not.toBeInTheDocument(); });
it("requires a full impact review when a scoped swap leaves prepared portions available", async () => {
  const date = new Date().toISOString().slice(0, 10);
  applyPlan.mockClear();
  generatePlan.mockResolvedValue({ currentRevision: 1n, occurrences: [{ date, slotName: "dinner", recipeId: "r1", recipeName: "Beans", locked: false, reason: "Saved" }], unresolved: [] });
  exploreRecipes.mockResolvedValue({ candidates: [{ recipeId: "r2", name: "Soup", recipeRevision: 1n, fitReasons: [{ code: "eligible", rule: "current rules", reference: "profile", message: "Fits current food rules.", methodId: "" }], methodAssessments: [], summary: "" }] });
  previewSwap.mockResolvedValue({ revision: 1n, preview: { draft: { currentRevision: 1n, occurrences: [{ date, slotName: "dinner", recipeId: "r2", recipeName: "Soup", locked: false, reason: "Swap" }], unresolved: [] }, changes: [{ date, slotName: "dinner", beforeName: "Beans", afterName: "Soup" }], relatedOccurrences: [], preparedBatchImpacts: [{ batchId: "beans-batch", recipeId: "r1", recipeName: "Beans", recipeRevision: 2, available: "1", unit: "serving" }], shoppingChanges: [] }, affectedDates: [date] });
  renderToday();
  await userEvent.click(await screen.findByRole("button", { name: "Swap meal" }));
  await userEvent.click(screen.getByRole("button", { name: "Review Soup swap" }));
  const impact = await screen.findByRole("region", { name: "Swap impact" });
  expect(impact).toHaveTextContent("Prepared portions to review");
  expect(impact).toHaveTextContent("Beans · revision 2 · batch beans-batch: 1 serving available");
  expect(screen.getByRole("button", { name: "Apply reviewed changes" })).toBeInTheDocument();
  expect(applyPlan).not.toHaveBeenCalled();
});
it("requires a full impact review for a cross-date swap even when shopping lines do not change", async () => {
  const today = new Date().toISOString().slice(0, 10);
  const later = new Date(new Date(`${today}T12:00:00`).setDate(new Date(`${today}T12:00:00`).getDate() + 1)).toISOString().slice(0, 10);
  generatePlan.mockResolvedValue({ currentRevision: 1n, occurrences: [{ date: today, slotName: "dinner", recipeId: "r1", recipeName: "Bowl", locked: false, reason: "Known fit" }], unresolved: [] });
  exploreRecipes.mockResolvedValue({ candidates: [{ recipeId: "r2", name: "Soup", recipeRevision: 1n, fitReasons: [{ code: "eligible", rule: "current rules", reference: "profile", message: "Fits current food rules.", methodId: "" }], methodAssessments: [], summary: "" }] });
  previewSwap.mockResolvedValue({ revision: 1n, preview: { draft: { currentRevision: 1n, occurrences: [{ date: today, slotName: "dinner", recipeId: "r2", recipeName: "Soup", locked: false, reason: "Compatible" }], unresolved: [] }, changes: [{ date: today, slotName: "dinner", beforeName: "Bowl", afterName: "Soup" }], shoppingChanges: [] }, affectedDates: [today, later] });
  renderToday();
  await userEvent.click(await screen.findByRole("button", { name: "Swap meal" }));
  await userEvent.click(screen.getByRole("button", { name: "Review Soup swap" }));
  expect(await screen.findByRole("region", { name: "Swap impact" })).toHaveTextContent(`dates affected: ${today}, ${later}`);
  expect(screen.getByRole("button", { name: "Apply reviewed changes" })).toBeInTheDocument();
  expect(screen.queryByRole("button", { name: "Confirm swap" })).not.toBeInTheDocument();
});
it("reports replacement-list failures", async () => { generatePlan.mockResolvedValue({ occurrences: [{ date: "2026-09-21", slotName: "dinner", recipeId: "r1", recipeName: "Bowl", locked: false, reason: "Known fit" }], unresolved: [] }); exploreRecipes.mockRejectedValue(new Error("eligibility unavailable")); renderToday(); await waitFor(() => expect(screen.getByRole("heading", { name: "Bowl" })).toBeInTheDocument()); await userEvent.click(screen.getByRole("button", { name: "Swap meal" })); await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("eligibility unavailable")); });
it("reports swap preview and apply failures", async () => { generatePlan.mockResolvedValue({ currentRevision: 1n, occurrences: [{ date: "2026-09-21", slotName: "dinner", recipeId: "r1", recipeName: "Bowl", locked: false, reason: "Known fit" }], unresolved: [] }); exploreRecipes.mockResolvedValue({ candidates: [{ recipeId: "r2", name: "Soup", recipeRevision: 1n, fitReasons: [{ code: "eligible", rule: "current rules", reference: "profile", message: "Fits current food rules.", methodId: "" }], methodAssessments: [], summary: "" }] }); previewSwap.mockRejectedValueOnce(new Error("ineligible replacement")); renderToday(); await waitFor(() => expect(screen.getByRole("heading", { name: "Bowl" })).toBeInTheDocument()); await userEvent.click(screen.getByRole("button", { name: "Swap meal" })); await userEvent.click(screen.getByRole("button", { name: "Review Soup swap" })); await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("ineligible replacement")); previewSwap.mockResolvedValueOnce({ revision: 1n, preview: { draft: { currentRevision: 1n, occurrences: [], unresolved: [] }, changes: [] } }); applyPlan.mockRejectedValueOnce(new Error("stale swap")); await userEvent.click(screen.getByRole("button", { name: "Review Soup swap" })); await waitFor(() => expect(screen.getByRole("button", { name: "Apply reviewed changes" })).toBeInTheDocument()); await userEvent.click(screen.getByRole("button", { name: "Apply reviewed changes" })); await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("stale swap")); });
it("starts cooking by navigating to the occurrence’s pinned revision without applying the plan", async () => {
  vi.clearAllMocks();
  const date = new Date().toISOString().slice(0, 10);
  generatePlan.mockResolvedValue({ currentRevision: 8n, occurrences: [{ date, slotName: "dinner", recipeId: "r1", recipeRevision: 7, recipeName: "Bowl", locked: false, reason: "Planned" }], unresolved: [] });
  renderToday();
  await userEvent.click(await screen.findByRole("button", { name: "Start cooking" }));
  await waitFor(() => expect(screen.getByTestId("current-path")).toHaveTextContent("/recipes/r1/revisions/7"));
  expect(applyPlan).not.toHaveBeenCalled();
  expect(startCookingSession).not.toHaveBeenCalled();
});

it("keeps cooking unavailable when an occurrence has no pinned recipe revision", async () => {
  vi.clearAllMocks();
  generatePlan.mockResolvedValue({ currentRevision: 8n, occurrences: [{ date: new Date().toISOString().slice(0, 10), slotName: "dinner", recipeId: "r1", recipeName: "Bowl", locked: false, reason: "Planned" }], unresolved: [] });
  renderToday();
  const start = await screen.findByRole("button", { name: "Start cooking" });
  expect(start).toBeDisabled();
  expect(screen.getByText("This planned meal has no pinned recipe revision, so cooking cannot be started safely.")).toBeInTheDocument();
  expect(applyPlan).not.toHaveBeenCalled();
});

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

describe("Today swap dialog keyboard behavior", () => { beforeEach(() => { vi.clearAllMocks(); }); it("focuses the dialog, traps both Tab directions, ignores other keys, and restores the trigger on Escape", async () => { const user = userEvent.setup(); generatePlan.mockResolvedValue({ occurrences: [{ date: "2026-09-21", slotName: "dinner", recipeId: "r1", recipeName: "Bowl", locked: false, reason: "Known fit" }], unresolved: [] }); exploreRecipes.mockResolvedValue({ candidates: [{ recipeId: "r2", name: "Soup", recipeRevision: 1n, fitReasons: [{ code: "eligible", rule: "current rules", reference: "profile", message: "Fits current food rules.", methodId: "" }], methodAssessments: [], summary: "" }] }); renderToday(); await waitFor(() => expect(screen.getByRole("button", { name: "Swap meal" })).toBeInTheDocument()); const trigger = screen.getByRole("button", { name: "Swap meal" }); await user.click(trigger); const firstCandidate = screen.getByRole("button", { name: "Review Soup swap" }); expect(firstCandidate).toHaveFocus(); await user.keyboard("x"); expect(firstCandidate).toHaveFocus(); await user.keyboard("{Shift>}{Tab}{/Shift}"); const cancel = screen.getByRole("button", { name: "Cancel" }); expect(cancel).toHaveFocus(); await user.tab(); expect(firstCandidate).toHaveFocus(); await user.keyboard("{Escape}"); expect(screen.queryByRole("dialog")).not.toBeInTheDocument(); expect(trigger).toHaveFocus(); }); });

it("places the dimming backdrop outside the swap dialog content", async () => {
  vi.clearAllMocks();
  generatePlan.mockResolvedValue({ occurrences: [{ date: "2026-09-21", slotName: "dinner", recipeId: "r1", recipeName: "Bowl", locked: false, reason: "Known fit" }], unresolved: [] });
  exploreRecipes.mockResolvedValue({ candidates: [{ recipeId: "r2", name: "Soup", recipeRevision: 1n, fitReasons: [], methodAssessments: [], summary: "" }] });
  renderToday();
  await userEvent.click(await screen.findByRole("button", { name: "Swap meal" }));
  const dialog = await screen.findByRole("dialog");
  const backdrop = document.querySelector(".today-swap-backdrop");
  expect(backdrop).toHaveAttribute("aria-hidden", "true");
  expect(backdrop?.nextElementSibling).toBe(dialog);
  await userEvent.click(screen.getByRole("button", { name: "Cancel" }));
  expect(document.querySelector(".today-swap-backdrop")).not.toBeInTheDocument();
});

it("shows the pinned revision for unchanged later uses in Today swap review", async () => {
  const todayDate = new Date();
  const format = (value: Date) => `${value.getFullYear()}-${String(value.getMonth() + 1).padStart(2, "0")}-${String(value.getDate()).padStart(2, "0")}`;
  const today = format(todayDate);
  todayDate.setDate(todayDate.getDate() + 1);
  const later = format(todayDate);
  generatePlan.mockResolvedValue({ currentRevision: 1n, occurrences: [{ date: today, slotName: "dinner", recipeId: "r1", recipeRevision: 4, recipeName: "Beans", locked: false, reason: "Saved" }], unresolved: [] });
  exploreRecipes.mockResolvedValue({ candidates: [{ recipeId: "r2", name: "Soup", recipeRevision: 1n, fitReasons: [{ code: "eligible", rule: "current rules", reference: "profile", message: "Fits current food rules.", methodId: "" }], methodAssessments: [], summary: "" }] });
  previewSwap.mockResolvedValue({ revision: 1n, preview: { draft: { currentRevision: 1n, occurrences: [], unresolved: [] }, changes: [{ date: today, slotName: "dinner", beforeName: "Beans", afterName: "Soup" }], relatedOccurrences: [{ date: later, slotName: "dinner", recipeRevision: 4, recipeName: "Beans", locked: false }], shoppingChanges: [] }, affectedDates: [today] });
  renderToday();
  await userEvent.click(await screen.findByRole("button", { name: "Swap meal" }));
  await userEvent.click(screen.getByRole("button", { name: "Review Soup swap" }));
  expect(await screen.findByText(`${later} · dinner: Beans · revision 4`)).toBeInTheDocument();
  expect(applyPlan).not.toHaveBeenCalled();
});
