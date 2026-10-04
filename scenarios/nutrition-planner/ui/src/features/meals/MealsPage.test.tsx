import { beforeEach, describe, expect, it, vi } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { create } from "@bufbuild/protobuf";
import { RecipeSchema } from "@vrooli/proto-types/nutrition-planner/v1/recipe/recipe_pb";
import { renderWithProviders } from "../../test-utils";

const listRecipes = vi.hoisted(() => vi.fn());
const createRecipe = vi.hoisted(() => vi.fn());
const updateRecipe = vi.hoisted(() => vi.fn());
vi.mock("../../api/recipes", () => ({ listRecipes, createRecipe, updateRecipe }));
vi.mock("../../api/workspace", () => ({ ensureWorkspace: vi.fn().mockResolvedValue({ id: "w1" }) }));
import { MealsPage } from "./MealsPage";

const renderMeals = () => renderWithProviders(<MealsPage />);

describe("MealsPage", () => {
  beforeEach(() => { vi.clearAllMocks(); listRecipes.mockResolvedValue([]); });
  it("keeps an empty collection honest", async () => { renderMeals(); await waitFor(() => expect(screen.getByText(/No saved meals yet/)).toBeInTheDocument()); expect(screen.queryByText("0 kcal")).not.toBeInTheDocument(); });
  it("persists a name-only draft and links to its exact revision", async () => { const user = userEvent.setup(); const recipe = create(RecipeSchema, { id: "r1", name: "Lentil salad", revision: 1n, status: "draft" }); createRecipe.mockResolvedValue(recipe); renderMeals(); await user.type(screen.getByLabelText("Meal name"), "Lentil salad"); await user.click(screen.getByRole("button", { name: "Save draft" })); await waitFor(() => expect(screen.getByText("Lentil salad")).toBeInTheDocument()); expect(screen.getByText(/Nutrition · cost · yield: Unknown/)).toBeInTheDocument(); expect(screen.getByRole("link", { name: "Open recipe" })).toHaveAttribute("href", "/recipes/r1/revisions/1"); });
  it("edits a captured meal as a new optimistic revision", async () => { const user = userEvent.setup(); const recipe = create(RecipeSchema, { id: "r1", name: "Lentil salad", notes: "Old note", revision: 1n, status: "draft" }); const updated = create(RecipeSchema, { id: "r1", name: "Warm lentil salad", notes: "With herbs", revision: 2n, status: "draft" }); listRecipes.mockResolvedValue([recipe]); updateRecipe.mockResolvedValue(updated); renderMeals(); await user.click(await screen.findByRole("button", { name: "Edit meal" })); const name = screen.getByLabelText("Meal name"); await user.clear(name); await user.type(name, "Warm lentil salad"); const notes = screen.getByLabelText("Notes or pasted text"); await user.clear(notes); await user.type(notes, "With herbs"); await user.click(screen.getByRole("button", { name: "Save changes" })); await waitFor(() => expect(screen.getByText("Warm lentil salad")).toBeInTheDocument()); expect(updateRecipe).toHaveBeenCalledWith(expect.objectContaining({ workspaceId: "w1", id: "r1", expectedRevision: 1n, name: "Warm lentil salad", notes: "With herbs", originalText: "With herbs" })); expect(screen.getByText(/Revision 2/)).toBeInTheDocument(); });
  it("searches saved meals across names, notes, and source URLs", async () => { const user = userEvent.setup(); listRecipes.mockResolvedValue([create(RecipeSchema, { id: "r1", name: "Lentil salad", notes: "Fresh mint", revision: 1n }), create(RecipeSchema, { id: "r2", name: "Bean soup", notes: "Smoky paprika", sourceUrl: "https://meals.example/soup", revision: 1n })]); renderMeals(); await screen.findAllByRole("link", { name: "Open recipe" }); const search = screen.getByLabelText("Search saved meals"); await user.type(search, "Lentil"); expect(screen.getByRole("heading", { name: "Lentil salad" })).toBeInTheDocument(); expect(screen.queryByRole("heading", { name: "Bean soup" })).not.toBeInTheDocument(); await user.clear(search); await user.type(search, "MINT"); expect(screen.getByRole("heading", { name: "Lentil salad" })).toBeInTheDocument(); await user.clear(search); await user.type(search, "meals.example"); expect(screen.getByRole("heading", { name: "Bean soup" })).toBeInTheDocument(); await user.clear(search); await user.type(search, "nothing here"); expect(screen.getByRole("status")).toHaveTextContent("No saved meals match"); });
  it("retains the draft when the save fails", async () => { const user = userEvent.setup(); createRecipe.mockRejectedValue(new Error("offline")); renderMeals(); await user.type(screen.getByLabelText("Meal name"), "Keep me"); await user.click(screen.getByRole("button", { name: "Save draft" })); await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("offline")); expect(screen.getByDisplayValue("Keep me")).toBeInTheDocument(); });
});
