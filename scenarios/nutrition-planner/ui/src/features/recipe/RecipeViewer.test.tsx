import { describe, expect, it } from "vitest";
import { create } from "@bufbuild/protobuf";
import userEvent from "@testing-library/user-event";
import { screen } from "@testing-library/react";
import { RecipeSchema, RecipeMethodSchema, RecipeStepSchema } from "@vrooli/proto-types/nutrition-planner/v1/recipe/recipe_pb";
import { renderWithProviders } from "../../test-utils";
import { RecipeViewer } from "./RecipeViewer";

describe("RecipeViewer", () => {
  it("shares one revision across map, read, and cook tabs", async () => {
    const user = userEvent.setup();
    const recipe = create(RecipeSchema, { name: "Bowl", revision: 2n, originalText: "Mix everything." , methods: [create(RecipeMethodSchema, { id: "m1", steps: [create(RecipeStepSchema, { id: "s1", instruction: "Mix." }), create(RecipeStepSchema, { id: "s2", instruction: "Serve." })] })] });
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
});
