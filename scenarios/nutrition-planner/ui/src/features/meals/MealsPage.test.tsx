import { beforeEach, describe, expect, it, vi } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { create } from "@bufbuild/protobuf";
import { RecipeSchema } from "@vrooli/proto-types/nutrition-planner/v1/recipe/recipe_pb";
import { renderWithProviders } from "../../test-utils";

const listRecipes = vi.hoisted(() => vi.fn());
const createRecipe = vi.hoisted(() => vi.fn());
const updateRecipe = vi.hoisted(() => vi.fn());
const exportRecipePDF = vi.hoisted(() => vi.fn());
vi.mock("../../api/recipes", () => ({ listRecipes, createRecipe, updateRecipe }));
vi.mock("../../api/portability", () => ({ exportRecipePDF }));
vi.mock("../../api/workspace", () => ({ ensureWorkspace: vi.fn().mockResolvedValue({ id: "w1" }) }));
import { MealsPage } from "./MealsPage";

const renderMeals = () => renderWithProviders(<MealsPage />);

describe("MealsPage", () => {
  beforeEach(() => { vi.clearAllMocks(); listRecipes.mockResolvedValue([]); });
  it("links the saved-meals collection to Explore", async () => { renderMeals(); expect(await screen.findByRole("link", { name: "Explore meals" })).toHaveAttribute("href", "/meals/explore"); });
  it("keeps an empty collection honest", async () => { renderMeals(); await waitFor(() => expect(screen.getByText(/Your collection starts here/)).toBeInTheDocument()); expect(screen.queryByText("0 kcal")).not.toBeInTheDocument(); });
  it("persists a name-only draft and links to its exact revision", async () => { const user = userEvent.setup(); const recipe = create(RecipeSchema, { id: "r1", name: "Lentil salad", revision: 1n, status: "draft" }); createRecipe.mockResolvedValue(recipe); renderMeals(); await user.type(screen.getByLabelText("Meal name"), "Lentil salad"); await user.click(screen.getByRole("button", { name: "Save draft" })); await waitFor(() => expect(screen.getByText("Lentil salad")).toBeInTheDocument()); expect(screen.getByText(/Ingredients not added/)).toBeInTheDocument(); expect(screen.getByRole("link", { name: /Open recipe/ })).toHaveAttribute("href", "/recipes/r1/revisions/1"); });
  it("preserves structured recipe evidence through a quick edit and returned revision", async () => {
    const user = userEvent.setup();
    const structured = {
      methods: [{ id: "method-1", name: "Roast", steps: [{ id: "step-1", instruction: "Roast until tender", dependsOn: [], inputs: ["carrots"], outputs: ["roasted carrots"] }] }],
      groups: ["Dinner", "Make ahead"],
      requiredAppliances: ["Oven"],
      allergenEvidence: { peanuts: "Verified absent" },
      canonicalYield: "4", servingUnit: "bowls",
      ingredients: [{ id: "ingredient-1", name: "Carrots", amount: "500", unit: "g", preparation: "chopped" }],
    };
    const recipe = create(RecipeSchema, { id: "r1", name: "Lentil salad", notes: "Old note", originalText: "Original source excerpt", sourceUrl: "https://origin.example/recipe", sourceType: "url", revision: 1n, status: "draft", ...structured });
    const updated = create(RecipeSchema, { ...recipe, name: "Warm lentil salad", notes: "With herbs", revision: 2n });
    listRecipes.mockResolvedValue([recipe]);
    updateRecipe.mockResolvedValue(updated);
    renderMeals();
    await user.click(await screen.findByRole("button", { name: "Edit Lentil salad" }));
    await user.clear(screen.getByLabelText("Meal name"));
    await user.type(screen.getByLabelText("Meal name"), "Warm lentil salad");
    await user.clear(screen.getByLabelText("Notes or pasted text"));
    await user.type(screen.getByLabelText("Notes or pasted text"), "With herbs");
    await user.click(screen.getByRole("button", { name: "Save changes" }));
    await waitFor(() => expect(screen.getByRole("heading", { name: "Warm lentil salad" })).toBeInTheDocument());
    expect(updateRecipe).toHaveBeenCalledWith(expect.objectContaining({
      workspaceId: "w1", id: "r1", expectedRevision: 1n, name: "Warm lentil salad", notes: "With herbs",
      originalText: "Original source excerpt", sourceUrl: "https://origin.example/recipe", sourceType: "url",
      methods: recipe.methods, groups: recipe.groups, requiredAppliances: recipe.requiredAppliances,
      allergenEvidence: recipe.allergenEvidence, canonicalYield: recipe.canonicalYield,
      servingUnit: recipe.servingUnit, ingredients: recipe.ingredients,
    }));
    expect(screen.getByText(/Revision 2/)).toBeInTheDocument();
    expect(screen.getByText(/Yield · 4 bowls/)).toBeInTheDocument();
    expect(screen.getByText("Dinner")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Edit Warm lentil salad" }));
    expect(screen.getByLabelText("Meal name")).toHaveValue("Warm lentil salad");
    expect(screen.getByLabelText("Notes or pasted text")).toHaveValue("With herbs");
    expect(screen.getByLabelText("Source URL (optional)")).toHaveValue("https://origin.example/recipe");
  });
  it("searches saved meals across names, notes, and source URLs", async () => { const user = userEvent.setup(); listRecipes.mockResolvedValue([create(RecipeSchema, { id: "r1", name: "Lentil salad", notes: "Fresh mint", revision: 1n }), create(RecipeSchema, { id: "r2", name: "Bean soup", notes: "Smoky paprika", sourceUrl: "https://meals.example/soup", revision: 1n })]); renderMeals(); await screen.findAllByRole("link", { name: /Open recipe/ }); const search = screen.getByRole("searchbox"); await user.type(search, "Lentil"); expect(screen.getByRole("heading", { name: "Lentil salad" })).toBeInTheDocument(); expect(screen.queryByRole("heading", { name: "Bean soup" })).not.toBeInTheDocument(); await user.clear(search); await user.type(search, "MINT"); expect(screen.getByRole("heading", { name: "Lentil salad" })).toBeInTheDocument(); await user.clear(search); await user.type(search, "meals.example"); expect(screen.getByRole("heading", { name: "Bean soup" })).toBeInTheDocument(); await user.clear(search); await user.type(search, "nothing here"); expect(screen.getByRole("status")).toHaveTextContent("No saved meals match"); });
  it("retains the draft when the save fails", async () => { const user = userEvent.setup(); createRecipe.mockRejectedValue(new Error("offline")); renderMeals(); await user.type(screen.getByLabelText("Meal name"), "Keep me"); await user.click(screen.getByRole("button", { name: "Save draft" })); await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("offline")); expect(screen.getByDisplayValue("Keep me")).toBeInTheDocument(); });
});

describe("Meals collection organization", () => {
  beforeEach(() => { vi.clearAllMocks(); });
  it("searches truthful ingredient and saved group fields and filters drafts", async () => {
    const user = userEvent.setup();
    listRecipes.mockResolvedValue([
      create(RecipeSchema, { id: "r1", name: "Weeknight bowl", ingredients: [{ name: "Chickpeas" }], groups: ["Dinner"], status: "draft", revision: 1n }),
      create(RecipeSchema, { id: "r2", name: "Saved soup", status: "ready", revision: 1n }),
    ]);
    renderMeals();
    await screen.findByRole("heading", { name: "Weeknight bowl" });
    expect(screen.getByRole("list")).toHaveClass("meals-grid");
    await user.type(screen.getByRole("searchbox"), "chickpeas");
    expect(screen.getByRole("heading", { name: "Weeknight bowl" })).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Saved soup" })).not.toBeInTheDocument();
    await user.clear(screen.getByRole("searchbox"));
    await user.click(screen.getByRole("button", { name: "Drafts" }));
    expect(screen.getByRole("heading", { name: "Weeknight bowl" })).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Saved soup" })).not.toBeInTheDocument();
    expect(screen.queryByText(/minutes|kcal|favorite/i)).not.toBeInTheDocument();
  });
});


describe("Meals export", () => {
  beforeEach(() => { vi.clearAllMocks(); });
  it("requests and downloads the selected saved recipe PDF", async () => {
    const user = userEvent.setup();
    listRecipes.mockResolvedValue([create(RecipeSchema, { id: "r-export", name: "Saved soup", revision: 3n })]);
    exportRecipePDF.mockResolvedValue({ content: [37, 80, 68, 70], filename: "saved-soup.pdf" });
    const createObjectURL = vi.fn(() => "blob:recipe-pdf");
    const revokeObjectURL = vi.fn();
    Object.defineProperty(URL, "createObjectURL", { configurable: true, value: createObjectURL });
    Object.defineProperty(URL, "revokeObjectURL", { configurable: true, value: revokeObjectURL });
    const click = vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(() => undefined);
    renderMeals();
    await user.click(await screen.findByRole("button", { name: "Download PDF for Saved soup" }));
    await waitFor(() => expect(exportRecipePDF).toHaveBeenCalledWith({ workspaceId: "w1", recipeId: "r-export" }));
    expect(createObjectURL).toHaveBeenCalled();
    expect(click).toHaveBeenCalled();
    expect(revokeObjectURL).toHaveBeenCalledWith("blob:recipe-pdf");
    delete (URL as unknown as { createObjectURL?: unknown }).createObjectURL;
    delete (URL as unknown as { revokeObjectURL?: unknown }).revokeObjectURL;
    click.mockRestore();
  });
});
