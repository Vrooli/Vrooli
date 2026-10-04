import { beforeEach, describe, expect, it, vi } from "vitest";
import { create } from "@bufbuild/protobuf";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { CookingSessionSchema, TimerSchema } from "@vrooli/proto-types/nutrition-planner/v1/cooking/cooking_pb";
import { RecipeMethodSchema, RecipeSchema } from "@vrooli/proto-types/nutrition-planner/v1/recipe/recipe_pb";
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
