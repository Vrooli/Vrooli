import { beforeEach, describe, expect, it, vi } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { create } from "@bufbuild/protobuf";
import { RecipeSchema } from "@vrooli/proto-types/nutrition-planner/v1/recipe/recipe_pb";
import { renderWithProviders } from "../../test-utils";

const listRecipes = vi.hoisted(() => vi.fn());
const createRecipe = vi.hoisted(() => vi.fn());
vi.mock("../../api/recipes", () => ({ listRecipes, createRecipe }));
vi.mock("../../api/workspace", () => ({ ensureWorkspace: vi.fn().mockResolvedValue({ id: "w1" }) }));
import { MealsPage } from "./MealsPage";

describe("MealsPage", () => {
  beforeEach(() => { vi.clearAllMocks(); listRecipes.mockResolvedValue([]); });
  it("keeps an empty collection honest", async () => { renderWithProviders(<MealsPage />); await waitFor(() => expect(screen.getByText(/No saved meals yet/)).toBeInTheDocument()); expect(screen.queryByText("0 kcal")).not.toBeInTheDocument(); });
  it("persists a name-only draft and renders unknown fields honestly", async () => { const user = userEvent.setup(); const recipe = create(RecipeSchema, { id: "r1", name: "Lentil salad", revision: 1n, status: "draft" }); createRecipe.mockResolvedValue(recipe); renderWithProviders(<MealsPage />); await user.type(screen.getByLabelText("Meal name"), "Lentil salad"); await user.click(screen.getByRole("button", { name: "Save draft" })); await waitFor(() => expect(screen.getByText("Lentil salad")).toBeInTheDocument()); expect(screen.getByText(/Nutrition · cost · yield: Unknown/)).toBeInTheDocument(); await user.click(screen.getByRole("button", { name: "Open recipe" })); expect(screen.getByRole("heading", { name: "Lentil salad", level: 2 })).toBeInTheDocument(); await user.click(screen.getByRole("button", { name: "Close" })); });
  it("retains the draft when the save fails", async () => { const user = userEvent.setup(); createRecipe.mockRejectedValue(new Error("offline")); renderWithProviders(<MealsPage />); await user.type(screen.getByLabelText("Meal name"), "Keep me"); await user.click(screen.getByRole("button", { name: "Save draft" })); await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("offline")); expect(screen.getByDisplayValue("Keep me")).toBeInTheDocument(); });
});
