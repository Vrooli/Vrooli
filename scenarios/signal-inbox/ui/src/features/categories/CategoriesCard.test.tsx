import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { strings } from "../../consts/strings";

const api = vi.hoisted(() => ({
  createCategory: vi.fn(),
  listCategories: vi.fn(),
  renameCategory: vi.fn(),
  retireCategory: vi.fn(),
}));

vi.mock("../../api/categories", () => ({
  categoriesClient: api,
}));

import { CategoriesCard } from "./CategoriesCard";

describe("CategoriesCard [REQ:SIG-P0-004] [REQ:SIG-P0-005]", () => {
  beforeEach(() => {
    api.listCategories.mockResolvedValue({ categories: [] });
    api.createCategory.mockResolvedValue({});
    api.renameCategory.mockResolvedValue({});
    api.retireCategory.mockResolvedValue({});
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("creates categories as runtime operator data", async () => {
    const user = userEvent.setup();
    renderWithProviders(<CategoriesCard />);

    await user.type(screen.getByLabelText(strings.categories.newName), "Research");
    await user.type(screen.getByLabelText(strings.categories.newDescription), "Things to study");
    await user.click(screen.getByRole("button", { name: strings.categories.create }));

    await waitFor(() => expect(api.createCategory).toHaveBeenCalledWith({ name: "Research", description: "Things to study" }));
  });

  it("allows an operator to rename or retire a non-reserved category", async () => {
    const user = userEvent.setup();
    api.listCategories.mockResolvedValue({ categories: [{ id: "category-1", name: "Old", description: "", reserved: false }] });
    renderWithProviders(<CategoriesCard />);

    const name = await screen.findByLabelText(strings.categories.nameFor);
    await user.clear(name);
    await user.type(name, "New");
    await user.click(screen.getByRole("button", { name: strings.categories.save }));
    await waitFor(() => expect(api.renameCategory).toHaveBeenCalledWith({ id: "category-1", name: "New", description: "" }));

    await user.click(screen.getByRole("button", { name: strings.categories.retire }));
    await waitFor(() => expect(api.retireCategory).toHaveBeenCalledWith({ id: "category-1" }));
  });

  it("shows the reserved fallback without allowing it to be mutated", async () => {
    api.listCategories.mockResolvedValue({ categories: [{ id: "fallback", name: "uncategorized", description: "", reserved: true }] });
    renderWithProviders(<CategoriesCard />);

    expect(await screen.findByText(strings.categories.fallback)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: strings.categories.retire })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: strings.categories.save })).not.toBeInTheDocument();
  });
});
