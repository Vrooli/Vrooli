import { beforeEach, describe, expect, it, vi } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { create } from "@bufbuild/protobuf";
import { CookingSessionSchema } from "@vrooli/proto-types/nutrition-planner/v1/cooking/cooking_pb";
import { RecipeSchema } from "@vrooli/proto-types/nutrition-planner/v1/recipe/recipe_pb";
import { renderWithProviders } from "../../test-utils";
import { TestAppRouter } from "../../app/routes";

const getRecipeRevision = vi.hoisted(() => vi.fn());
const listCookingSessions = vi.hoisted(() => vi.fn());
const startCookingSession = vi.hoisted(() => vi.fn());
vi.mock("../../api/recipes", () => ({ getRecipeRevision }));
vi.mock("../../api/cooking", () => ({ listCookingSessions, startCookingSession }));
vi.mock("../../api/workspace", () => ({ ensureWorkspace: vi.fn().mockResolvedValue({ id: "w1" }) }));

describe("RecipeDetailPage", () => {
  beforeEach(() => { vi.clearAllMocks(); listCookingSessions.mockResolvedValue([]); });

  it("loads the requested immutable revision and scales only displayed ingredients", async () => {
    const user = userEvent.setup();
    const recipe = create(RecipeSchema, { id: "r1", revision: 1n, name: "Original soup", canonicalYield: "3", servingUnit: "bowls", originalText: "Original instructions", ingredients: [{ id: "lentils", name: "Lentils", amount: "1", unit: "cup", discrete: true }, { id: "onion", name: "Onion", amount: "1", unit: "each" }, { id: "salt", amount: "as needed", unit: "pinch" }], methods: [{ id: "stovetop", name: "Stovetop", steps: [{ id: "chop", instruction: "Chop onion", inputs: ["onion"], outputs: ["prepared_onion"] }, { id: "simmer", instruction: "Simmer gently", dependsOn: ["chop"], inputs: ["lentils", "prepared_onion"], outputs: ["soup"] }] }, { id: "oven", name: "Oven", steps: [{ id: "bake", instruction: "Bake until browned" }] }] });
    getRecipeRevision.mockResolvedValue(recipe);
    renderWithProviders(<TestAppRouter initialEntries={["/recipes/r1/revisions/1"]} />, { withoutRouter: true });
    await waitFor(() => expect(screen.getByRole("heading", { name: "Original soup" })).toBeInTheDocument());
    expect(getRecipeRevision).toHaveBeenCalledWith("w1", "r1", 1n);
    expect(screen.getByText(/pinned to revision 1/)).toBeInTheDocument();
    await user.clear(screen.getByLabelText("Displayed servings"));
    await user.type(screen.getByLabelText("Displayed servings"), "2");
    expect(screen.getByText("Lentils").parentElement).toHaveTextContent("0.666666 cup");
    expect(screen.getByText("Lentils").parentElement).toHaveTextContent("Whole units are required");
    expect(screen.getByText("salt").parentElement).toHaveTextContent("Unknown pinch");
    expect(screen.getAllByText("Simmer gently")).toHaveLength(2);
    expect(screen.getByText("Lentils, prepared_onion from chop")).toBeInTheDocument();
    const afterLabel = screen.getAllByText("After").at(1);
    const makesLabel = screen.getAllByText("Makes").at(1);
    if (!afterLabel || !makesLabel) throw new Error("Expected both method steps in the recipe map.");
    expect(afterLabel.parentElement).toHaveTextContent("Chop onion");
    expect(makesLabel.parentElement).toHaveTextContent("soup");
    await user.selectOptions(screen.getByLabelText("Preparation method"), "oven");
    expect(screen.getAllByText("Bake until browned")).toHaveLength(2);
    await user.click(screen.getByRole("tab", { name: "Read original" }));
    expect(screen.getByText("Original instructions")).toBeInTheDocument();
  });

  it("keeps a missing method explicit", async () => {
    getRecipeRevision.mockResolvedValue(create(RecipeSchema, { id: "r1", revision: 3n, name: "Name only", notes: "Copied notes" }));
    renderWithProviders(<TestAppRouter initialEntries={["/recipes/r1/revisions/3"]} />, { withoutRouter: true });
    await waitFor(() => expect(screen.getByText("No preparation method has been entered for this recipe revision.")).toBeInTheDocument());
    await userEvent.click(screen.getByRole("tab", { name: "Read original" }));
    expect(screen.getByText("Copied notes")).toBeInTheDocument();
  });

  it("resumes only an active session pinned to the selected revision and method", async () => {
    const recipe = create(RecipeSchema, { id: "r1", revision: 3n, name: "Soup", canonicalYield: "4", methods: [{ id: "stove", name: "Stovetop", steps: [{ id: "cook", instruction: "Cook" }] }, { id: "oven", name: "Oven", steps: [{ id: "bake", instruction: "Bake" }] }] });
    getRecipeRevision.mockResolvedValue(recipe);
    listCookingSessions.mockResolvedValue([
      create(CookingSessionSchema, { id: "active-stove", workspaceId: "w1", recipeId: "r1", recipeRevision: 3n, methodId: "stove", scale: "4", status: "active" }),
      create(CookingSessionSchema, { id: "old-revision", workspaceId: "w1", recipeId: "r1", recipeRevision: 2n, methodId: "stove", scale: "4", status: "active" }),
    ]);
    const user = userEvent.setup();
    renderWithProviders(<TestAppRouter initialEntries={["/recipes/r1/revisions/3"]} />, { withoutRouter: true });
    expect(await screen.findByRole("link", { name: "Resume active cooking session" })).toHaveAttribute("href", "/cook/active-stove");
    await user.selectOptions(screen.getByLabelText("Preparation method"), "oven");
    expect(screen.queryByRole("link", { name: "Resume active cooking session" })).not.toBeInTheDocument();
  });

  it("reports an invalid revision path and server lookup failures", async () => {
    const invalid = renderWithProviders(<TestAppRouter initialEntries={["/recipes/r1/revisions/0"]} />, { withoutRouter: true });
    expect(await screen.findByRole("alert")).toHaveTextContent("revision link is invalid");

    invalid.unmount();
    getRecipeRevision.mockRejectedValueOnce(new Error("revision was removed"));
    renderWithProviders(<TestAppRouter initialEntries={["/recipes/r1/revisions/2"]} />, { withoutRouter: true });
    expect(await screen.findByRole("alert")).toHaveTextContent("revision was removed");
  });

  it("uses a plain fallback when the loader rejects without an Error", async () => {
    getRecipeRevision.mockRejectedValue("offline");
    renderWithProviders(<TestAppRouter initialEntries={["/recipes/r1/revisions/2"]} />, { withoutRouter: true });
    expect(await screen.findByRole("alert")).toHaveTextContent("Unable to load this recipe revision.");
  });

  it("shows unresolved historical graph references without inventing links", async () => {
    getRecipeRevision.mockResolvedValue(create(RecipeSchema, { id: "r1", revision: 1n, name: "Imported method", methods: [{ id: "legacy", name: "Imported", steps: [{ id: "old-step", instruction: "", inputs: ["missing_component"] }] }] }));
    renderWithProviders(<TestAppRouter initialEntries={["/recipes/r1/revisions/1"]} />, { withoutRouter: true });
    expect(await screen.findByText("Unresolved component: missing_component")).toBeInTheDocument();
    expect(screen.getByText("Instruction not entered.")).toBeInTheDocument();
    expect(screen.getByText("No step dependency recorded.")).toBeInTheDocument();
    expect(screen.getByText("No output recorded.")).toBeInTheDocument();
  });
});
