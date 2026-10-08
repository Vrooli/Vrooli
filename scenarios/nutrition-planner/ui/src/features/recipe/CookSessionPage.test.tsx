import { beforeEach, describe, expect, it, vi } from "vitest";
import { create } from "@bufbuild/protobuf";
import { act, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { CookingSessionSchema, TimerSchema } from "@vrooli/proto-types/nutrition-planner/v1/cooking/cooking_pb";
import { RecipeIngredientSchema, RecipeMethodSchema, RecipeSchema, RecipeStepSchema } from "@vrooli/proto-types/nutrition-planner/v1/recipe/recipe_pb";
import { renderWithProviders } from "../../test-utils";
import { TestAppRouter } from "../../app/routes";

const getCookingSession = vi.hoisted(() => vi.fn());
const saveCookingSession = vi.hoisted(() => vi.fn());
const getRecipeRevision = vi.hoisted(() => vi.fn());
const listInventoryBatches = vi.hoisted(() => vi.fn());
const prepareInventoryBatch = vi.hoisted(() => vi.fn());
vi.mock("../../api/cooking", () => ({ getCookingSession, saveCookingSession }));
vi.mock("../../api/inventory", () => ({ listInventoryBatches, prepareInventoryBatch }));
vi.mock("../../api/recipes", () => ({ getRecipeRevision }));
vi.mock("../../api/workspace", () => ({ ensureWorkspace: vi.fn().mockResolvedValue({ id: "w1" }) }));

const recipe = create(RecipeSchema, { id: "r1", revision: 3n, name: "Bean soup", canonicalYield: "4", servingUnit: "bowls", ingredients: [{ id: "beans", name: "Beans", amount: "2", unit: "cup" }], methods: [{ id: "stove", name: "Stovetop", steps: [{ id: "chop", instruction: "Chop vegetables" }, { id: "simmer", instruction: "Simmer for ten minutes" }] }] });
const session = create(CookingSessionSchema, { id: "cook-1", workspaceId: "w1", recipeId: "r1", recipeRevision: 3n, methodId: "stove", scale: "4", currentStepIndex: 0, status: "active", version: 1n });

describe("CookSessionPage", () => {
  beforeEach(() => { vi.clearAllMocks(); getCookingSession.mockResolvedValue(session); getRecipeRevision.mockResolvedValue(recipe); listInventoryBatches.mockResolvedValue([]); prepareInventoryBatch.mockResolvedValue({ id: "cook-session:cook-1", recipeId: "r1", recipeRevision: 3n, yieldAmount: "5", availableAmount: "5", unit: "bowls" }); saveCookingSession.mockImplementation(async (input) => create(CookingSessionSchema, { ...session, currentStepIndex: input.currentStepIndex, completedSteps: input.completedSteps, timers: input.timers, status: input.finish ? "finished" : session.status, actualYield: input.actualYield, yieldUnit: input.yieldUnit, version: input.expectedVersion + 1n })); });

  it("keeps navigation separate from explicit step completion", async () => {
    const user = userEvent.setup();
    renderWithProviders(<TestAppRouter initialEntries={["/cook/cook-1"]} />, { withoutRouter: true });
    await screen.findByRole("heading", { name: "Chop vegetables" });
    await user.click(screen.getByRole("button", { name: "Next step" }));
    await waitFor(() => expect(saveCookingSession).toHaveBeenLastCalledWith(expect.objectContaining({ currentStepIndex: 1, completedSteps: [] })));
    await user.click(screen.getByRole("button", { name: "Mark step done" }));
    await waitFor(() => expect(saveCookingSession).toHaveBeenLastCalledWith(expect.objectContaining({ currentStepIndex: 1, completedSteps: ["simmer"] })));
  });

  it("shows only current-step linked ingredients scaled to the pinned session yield", async () => {
    const scopedRecipe = create(RecipeSchema, { ...recipe, canonicalYield: "4", ingredients: [
      create(RecipeIngredientSchema, { id: "beans", name: "Beans", amount: "2", unit: "cup", preparation: "rinsed" }),
      create(RecipeIngredientSchema, { id: "carrots", name: "Carrots", amount: "to taste" }),
      create(RecipeIngredientSchema, { id: "salt", name: "Salt", amount: "   " }),
      create(RecipeIngredientSchema, { id: "onions", name: "Onions", amount: "1", unit: "whole" }),
    ], methods: [create(RecipeMethodSchema, { id: "stove", name: "Stovetop", steps: [
      create(RecipeStepSchema, { id: "chop", instruction: "Chop vegetables", inputs: ["beans", "Carrots", "salt", "prepared-mixture"] }),
      create(RecipeStepSchema, { id: "simmer", instruction: "Simmer for ten minutes", inputs: ["onions"] }),
    ] })] });
    getRecipeRevision.mockResolvedValue(scopedRecipe);
    getCookingSession.mockResolvedValue(create(CookingSessionSchema, { ...session, scale: "6" }));
    renderWithProviders(<TestAppRouter initialEntries={["/cook/cook-1"]} />, { withoutRouter: true });
    const linked = within(await screen.findByRole("region", { name: "Ingredients for this step" }));
    expect(within(linked.getByRole("list")).getAllByRole("listitem")[0]).toHaveTextContent("Beans: 3 cup · rinsed");
    expect(within(linked.getByRole("list")).getAllByRole("listitem")[1]).toHaveTextContent("Carrots: to taste");
    expect(within(linked.getByRole("list")).getAllByRole("listitem")[2]).toHaveTextContent("Salt: Unknown");
    expect(linked.getByText(/Unresolved linked inputs \(quantity unknown\): prepared-mixture/)).toBeInTheDocument();
    expect(linked.queryByText(/Onions/)).not.toBeInTheDocument();
  });

  it("does not scale or guess quantities when recipe yield or session target is unknown", async () => {
    getRecipeRevision.mockResolvedValue(create(RecipeSchema, { ...recipe, canonicalYield: "", ingredients: [create(RecipeIngredientSchema, { id: "beans", name: "Beans", amount: "2", unit: "cup" })], methods: [create(RecipeMethodSchema, { id: "stove", name: "Stovetop", steps: [create(RecipeStepSchema, { id: "chop", instruction: "Chop vegetables", inputs: ["beans"] })] })] }));
    getCookingSession.mockResolvedValue(create(CookingSessionSchema, { ...session, scale: "unknown" }));
    renderWithProviders(<TestAppRouter initialEntries={["/cook/cook-1"]} />, { withoutRouter: true });
    const linked = within(await screen.findByRole("region", { name: "Ingredients for this step" }));
    expect(within(linked.getByRole("list")).getByRole("listitem")).toHaveTextContent("Beans: Unknown cup");
  });

  it("opens an accessible step list with completion from the persisted session", async () => {
    const user = userEvent.setup();
    getCookingSession.mockResolvedValue(create(CookingSessionSchema, { ...session, completedSteps: ["chop"] }));
    renderWithProviders(<TestAppRouter initialEntries={["/cook/cook-1"]} />, { withoutRouter: true });
    const opener = await screen.findByRole("button", { name: "View all steps" });
    await user.click(opener);
    const dialog = screen.getByRole("dialog", { name: "Cooking steps" });
    expect(dialog).toBeInTheDocument();
    expect(within(dialog).getByRole("button", { name: /Step 1 · Current Chop vegetables Completed/ })).toBeInTheDocument();
    expect(within(dialog).getByRole("button", { name: /Step 2 Simmer for ten minutes Not completed/ })).toBeInTheDocument();
    expect(document.activeElement).toBe(within(dialog).getByRole("button", { name: "Close step list" }));
    await user.tab({ shift: true });
    expect(document.activeElement).toBe(within(dialog).getByRole("button", { name: /Step 2 Simmer for ten minutes Not completed/ }));
    await user.tab();
    expect(document.activeElement).toBe(within(dialog).getByRole("button", { name: "Close step list" }));
  });

  it("persists direct step navigation without changing completed steps and restores focus", async () => {
    const user = userEvent.setup();
    getCookingSession.mockResolvedValue(create(CookingSessionSchema, { ...session, completedSteps: ["chop"] }));
    renderWithProviders(<TestAppRouter initialEntries={["/cook/cook-1"]} />, { withoutRouter: true });
    const opener = await screen.findByRole("button", { name: "View all steps" });
    await user.click(opener);
    await user.click(screen.getByRole("button", { name: /Step 2 Simmer for ten minutes Not completed/ }));
    await waitFor(() => expect(saveCookingSession).toHaveBeenLastCalledWith(expect.objectContaining({ currentStepIndex: 1, completedSteps: ["chop"] })));
    await waitFor(() => expect(document.activeElement).toBe(opener));
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("closes the step list with Escape and returns focus to its opener", async () => {
    const user = userEvent.setup();
    renderWithProviders(<TestAppRouter initialEntries={["/cook/cook-1"]} />, { withoutRouter: true });
    const opener = await screen.findByRole("button", { name: "View all steps" });
    await user.click(opener);
    const close = screen.getByRole("button", { name: "Close step list" });
    await user.keyboard("{Escape}");
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    expect(document.activeElement).toBe(opener);
    await user.click(opener);
    await user.click(screen.getByRole("button", { name: "Close step list" }));
    expect(document.activeElement).toBe(opener);
    expect(close).not.toBeInTheDocument();
    expect(saveCookingSession).not.toHaveBeenCalled();
  });

  it("saves running timers and restores them from the session after reload", async () => {
    const user = userEvent.setup();
    let persisted = session;
    getCookingSession.mockImplementation(async () => persisted);
    saveCookingSession.mockImplementation(async (input) => { persisted = create(CookingSessionSchema, { ...persisted, timers: input.timers, version: persisted.version + 1n }); return persisted; });
    const view = renderWithProviders(<TestAppRouter initialEntries={["/cook/cook-1"]} />, { withoutRouter: true });
    await screen.findByRole("heading", { name: "Chop vegetables" });
    await user.click(screen.getByRole("button", { name: "Start timer for this step" }));
    await waitFor(() => expect(persisted.timers).toHaveLength(1));
    expect(persisted.timers[0]?.durationSeconds).toBe(600n);
    view.unmount();
    renderWithProviders(<TestAppRouter initialEntries={["/cook/cook-1"]} />, { withoutRouter: true });
    expect(await screen.findByText(/chop: 10:00/i)).toBeInTheDocument();
  });

  it("keeps finishing without actual yield separate from Kitchen stock", async () => {
    const user = userEvent.setup();
    renderWithProviders(<TestAppRouter initialEntries={["/cook/cook-1"]} />, { withoutRouter: true });
    await user.click(await screen.findByRole("button", { name: "Finish cooking" }));
    expect(await screen.findByText(/no stock change recorded/i)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Add measured yield to Kitchen stock" })).not.toBeInTheDocument();
    expect(prepareInventoryBatch).not.toHaveBeenCalled();
  });

  it("adds only the explicitly confirmed actual yield to Kitchen stock", async () => {
    const user = userEvent.setup();
    renderWithProviders(<TestAppRouter initialEntries={["/cook/cook-1"]} />, { withoutRouter: true });
    await screen.findByRole("heading", { name: "Chop vegetables" });
    await user.type(screen.getByRole("textbox", { name: "Actual yield" }), "5");
    await user.click(screen.getByRole("button", { name: "Finish cooking" }));
    await user.click(await screen.findByRole("button", { name: "Add measured yield to Kitchen stock" }));
    await waitFor(() => expect(prepareInventoryBatch).toHaveBeenCalledWith(expect.objectContaining({ workspaceId: "w1", eventId: "cook-yield:cook-1", batchId: "cook-session:cook-1", recipeId: "r1", recipeRevision: 3n, yieldAmount: "5", unit: "bowls", requirements: [{ itemId: "beans", amount: "2", unit: "cup" }] })));
    expect(await screen.findByText(/Kitchen stock updated: 5 bowls available/)).toBeInTheDocument();
  });

  it("offers a resume action for a paused timer", async () => {
    const user = userEvent.setup();
    const paused = create(CookingSessionSchema, { ...session, timers: [create(TimerSchema, { id: "timer-1", stepId: "chop", durationSeconds: 600n, startedAt: "2026-10-03T10:00:00Z", pausedAt: "2026-10-03T10:00:30Z", elapsedSeconds: 30n })] });
    getCookingSession.mockResolvedValue(paused);
    saveCookingSession.mockImplementation(async (input) => create(CookingSessionSchema, { ...paused, timers: input.timers, version: paused.version + 1n }));
    renderWithProviders(<TestAppRouter initialEntries={["/cook/cook-1"]} />, { withoutRouter: true });
    await user.click(await screen.findByRole("button", { name: "Resume" }));
    await waitFor(() => expect(saveCookingSession).toHaveBeenCalledWith(expect.objectContaining({ timers: [expect.objectContaining({ pausedAt: "", elapsedSeconds: 30n })] })));
  });

  it("adds one minute to a running timer deadline", async () => {
    const user = userEvent.setup();
    const running = create(CookingSessionSchema, { ...session, timers: [create(TimerSchema, { id: "running", stepId: "chop", durationSeconds: 600n, startedAt: new Date().toISOString(), elapsedSeconds: 0n })] });
    getCookingSession.mockResolvedValue(running);
    renderWithProviders(<TestAppRouter initialEntries={["/cook/cook-1"]} />, { withoutRouter: true });
    await user.click(await screen.findByRole("button", { name: "Add minute" }));
    await waitFor(() => expect(saveCookingSession).toHaveBeenCalledWith(expect.objectContaining({ timers: [expect.objectContaining({ id: "running", durationSeconds: 660n, startedAt: running.timers[0]?.startedAt })] })));
  });

  it("adds one minute to paused remaining time without resuming", async () => {
    const user = userEvent.setup();
    const paused = create(CookingSessionSchema, { ...session, timers: [create(TimerSchema, { id: "paused", stepId: "chop", durationSeconds: 600n, startedAt: "2026-10-03T10:00:00Z", pausedAt: "2026-10-03T10:00:30Z", elapsedSeconds: 30n })] });
    getCookingSession.mockResolvedValue(paused);
    renderWithProviders(<TestAppRouter initialEntries={["/cook/cook-1"]} />, { withoutRouter: true });
    await user.click(await screen.findByRole("button", { name: "Add minute" }));
    await waitFor(() => expect(saveCookingSession).toHaveBeenCalledWith(expect.objectContaining({ timers: [expect.objectContaining({ id: "paused", durationSeconds: 660n, startedAt: "2026-10-03T10:00:00Z", pausedAt: "2026-10-03T10:00:30Z", elapsedSeconds: 30n })] })));
  });

  it("dismisses only the selected timer", async () => {
    const user = userEvent.setup();
    const timers = ["one", "two"].map((id) => create(TimerSchema, { id, stepId: "chop", durationSeconds: 600n, startedAt: new Date().toISOString(), elapsedSeconds: 0n }));
    getCookingSession.mockResolvedValue(create(CookingSessionSchema, { ...session, timers }));
    renderWithProviders(<TestAppRouter initialEntries={["/cook/cook-1"]} />, { withoutRouter: true });
    const dismissButtons = await screen.findAllByRole("button", { name: "Dismiss timer" });
    await user.click(dismissButtons[0]!);
    await waitFor(() => expect(saveCookingSession).toHaveBeenCalledWith(expect.objectContaining({ timers: [expect.objectContaining({ id: "two" })] })));
  });

  it("announces each expired timer once without saving or completing cooking", async () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-10-07T12:00:00Z"));
    const expired = create(CookingSessionSchema, { ...session, timers: [create(TimerSchema, { id: "expired", stepId: "chop", durationSeconds: 1n, startedAt: "2026-10-07T11:59:58Z", elapsedSeconds: 0n })] });
    getCookingSession.mockResolvedValue(expired);
    const view = renderWithProviders(<TestAppRouter initialEntries={["/cook/cook-1"]} />, { withoutRouter: true });
    await act(async () => { await Promise.resolve(); await Promise.resolve(); });
    const announcement = screen.getByRole("status");
    expect(announcement).toHaveTextContent("Timer expired has ended.");
    await act(async () => { vi.advanceTimersByTime(3000); });
    expect(announcement).toHaveTextContent("Timer expired has ended.");
    expect(saveCookingSession).not.toHaveBeenCalled();
    expect(prepareInventoryBatch).not.toHaveBeenCalled();
    expect(screen.getByText(/Step 1 of 2/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Finish cooking" })).toBeInTheDocument();
    view.unmount();
    vi.useRealTimers();
  });

  it("reports a failed session load without inventing a recipe method", async () => {
    getCookingSession.mockRejectedValue("offline");
    renderWithProviders(<TestAppRouter initialEntries={["/cook/cook-1"]} />, { withoutRouter: true });
    expect(await screen.findByRole("alert")).toHaveTextContent("Unable to resume this cooking session.");
    expect(getRecipeRevision).not.toHaveBeenCalled();
  });

  it("explains when the pinned method has no recorded steps", async () => {
    const noSteps = create(RecipeSchema, { ...recipe, methods: [create(RecipeMethodSchema, { id: "stove", name: "Stovetop", steps: [] })] });
    getRecipeRevision.mockResolvedValue(noSteps);
    renderWithProviders(<TestAppRouter initialEntries={["/cook/cook-1"]} />, { withoutRouter: true });
    expect(await screen.findByText("No steps recorded")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Start timer for this step" })).toBeDisabled();
  });
});
