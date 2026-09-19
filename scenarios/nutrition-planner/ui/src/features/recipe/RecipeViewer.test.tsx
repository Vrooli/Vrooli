import { describe, expect, it } from "vitest";
import { create } from "@bufbuild/protobuf";
import userEvent from "@testing-library/user-event";
import { screen } from "@testing-library/react";
import { RecipeSchema, RecipeMethodSchema, RecipeStepSchema, RecipeIngredientSchema } from "@vrooli/proto-types/nutrition-planner/v1/recipe/recipe_pb";
import { renderWithProviders } from "../../test-utils";
import { RecipeViewer } from "./RecipeViewer";

describe("RecipeViewer", () => {
  it("shares one revision across map, read, and cook tabs", async () => {
    const user = userEvent.setup();
    const recipe = create(RecipeSchema, { name: "Bowl", revision: 2n, originalText: "Mix everything.", canonicalYield: "2", servingUnit: "servings", ingredients: [create(RecipeIngredientSchema, { id: "rice", name: "Rice", amount: "40", unit: "g" })], methods: [create(RecipeMethodSchema, { id: "m1", steps: [create(RecipeStepSchema, { id: "s1", instruction: "Mix." }), create(RecipeStepSchema, { id: "s2", instruction: "Serve." })] })] });
    renderWithProviders(<RecipeViewer recipe={recipe} onClose={() => undefined} />);
    expect(screen.getByText("Recipe revision 2")).toBeInTheDocument();
    await user.click(screen.getByRole("tab", { name: "Read" }));
    expect(screen.getByText("Mix everything.")).toBeInTheDocument();
    await user.click(screen.getByRole("tab", { name: "Cook" }));
    expect(screen.getByText("Mix.")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Next" }));
    expect(screen.getByText("Serve.")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Previous" }));
    await user.click(screen.getByRole("button", { name: "Start 1-minute timer" }));
    expect(screen.getByRole("status")).toHaveTextContent("Timer: 60 seconds remaining");
    await user.click(screen.getByRole("button", { name: "Close" }));
  });

  it("scales displayed quantities without mutating the stored revision", async () => {
    const user = userEvent.setup();
    const recipe = create(RecipeSchema, { name: "Egg bowl", revision: 3n, canonicalYield: "2", servingUnit: "servings", ingredients: [create(RecipeIngredientSchema, { id: "rice", name: "Rice", amount: "40", unit: "g" }), create(RecipeIngredientSchema, { id: "egg", name: "Egg", amount: "1", unit: "count", discrete: true })] });
    renderWithProviders(<RecipeViewer recipe={recipe} onClose={() => undefined} />);
    await user.clear(screen.getByLabelText("Displayed servings"));
    await user.type(screen.getByLabelText("Displayed servings"), "3");
    const ingredients = screen.getAllByRole("listitem");
    expect(ingredients[0]).toHaveTextContent("Rice: 60 g");
    expect(ingredients[1]).toHaveTextContent("Egg: 1.5 count");
    expect(screen.getByText(/whole units required/)).toBeInTheDocument();
    expect(recipe.canonicalYield).toBe("2");
  });
});
