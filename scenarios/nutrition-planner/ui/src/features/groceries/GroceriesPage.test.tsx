import { beforeEach, describe, expect, it, vi } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "../../test-utils";

const getShoppingPreview = vi.hoisted(() => vi.fn());
const setShoppingChecked = vi.hoisted(() => vi.fn());
const setShoppingHaveThis = vi.hoisted(() => vi.fn());
const confirmShoppingPurchases = vi.hoisted(() => vi.fn());
const listInventoryEvents = vi.hoisted(() => vi.fn());
const listInventoryBatches = vi.hoisted(() => vi.fn());
const correctInventoryBatchYield = vi.hoisted(() => vi.fn());
const prepareInventoryBatch = vi.hoisted(() => vi.fn());
const consumeInventoryBatchPortion = vi.hoisted(() => vi.fn());
const undoInventoryBatchPortion = vi.hoisted(() => vi.fn());
const listRecipes = vi.hoisted(() => vi.fn());
const getProfile = vi.hoisted(() => vi.fn());
const exportGroceriesCSV = vi.hoisted(() => vi.fn());
vi.mock("../../api/planning", () => ({ getShoppingPreview, setShoppingChecked, setShoppingHaveThis, confirmShoppingPurchases }));
vi.mock("../../api/inventory", () => ({ listInventoryEvents, listInventoryBatches, correctInventoryBatchYield, prepareInventoryBatch, consumeInventoryBatchPortion, undoInventoryBatchPortion }));
vi.mock("../../api/recipes", () => ({ listRecipes }));
vi.mock("../../api/profile", () => ({ getProfile }));
vi.mock("../../api/portability", () => ({ exportGroceriesCSV }));
vi.mock("../../api/workspace", () => ({ ensureWorkspace: vi.fn().mockResolvedValue({ id: "w1" }) }));
import { GroceriesPage } from "./GroceriesPage";

describe("GroceriesPage", () => {
  beforeEach(() => { vi.clearAllMocks(); setShoppingChecked.mockResolvedValue(true); setShoppingHaveThis.mockResolvedValue(true); confirmShoppingPurchases.mockResolvedValue(undefined); listInventoryEvents.mockResolvedValue([]); listInventoryBatches.mockResolvedValue([]); listRecipes.mockResolvedValue([]); getProfile.mockResolvedValue(undefined); exportGroceriesCSV.mockResolvedValue({ filename: "daily-groceries.csv", content: "key,label\n", revision: 1n }); });
  it("shows unknown facts and persists checklist-only state", async () => {
    getShoppingPreview.mockResolvedValue({ revision: 1n, lines: [{ key: "ingredient:rice", label: "rice", need: "unknown", stock: "unknown", missing: "unknown", packageCount: "unknown", price: "unknown", sourceRecipeIds: ["r1"], checked: false, haveThis: false }] });
    renderWithProviders(<GroceriesPage />);
    await waitFor(() => expect(screen.getByText("rice")).toBeInTheDocument());
    expect(screen.getByText(/Package price: unknown/)).toBeInTheDocument();
    expect(screen.getByText(/Portion cost: unknown · Checkout total: unknown · Actual spend: unknown/)).toBeInTheDocument();
    await userEvent.click(screen.getByRole("checkbox", { name: "Check rice" }));
    expect(setShoppingChecked).toHaveBeenCalledWith({ workspaceId: "w1", lineKey: "ingredient:rice", checked: true });
    await userEvent.click(screen.getByRole("checkbox", { name: "Have this rice" }));
    expect(setShoppingHaveThis).toHaveBeenCalledWith({ workspaceId: "w1", lineKey: "ingredient:rice", haveThis: true });
  });
  it("keeps the line unchanged when checklist persistence fails", async () => {
    getShoppingPreview.mockResolvedValue({ revision: 1n, lines: [{ key: "ingredient:rice", label: "rice", need: "unknown", stock: "unknown", missing: "unknown", packageCount: "unknown", price: "unknown", sourceRecipeIds: [], checked: false }] });
    setShoppingChecked.mockRejectedValue(new Error("offline"));
    renderWithProviders(<GroceriesPage />);
    await waitFor(() => expect(screen.getByRole("checkbox", { name: "Check rice" })).toBeInTheDocument());
    await userEvent.click(screen.getByRole("checkbox", { name: "Check rice" }));
    expect(screen.getByRole("checkbox", { name: "Check rice" })).not.toBeChecked();
  });
  it("confirms reviewed actual quantities and omissions together", async () => {
    getShoppingPreview.mockResolvedValue({ revision: 1n, lines: [
      { key: "ingredient:rice", label: "rice", need: "100 g", stock: "unknown", missing: "unknown", packageCount: "unknown", price: "unknown", sourceRecipeIds: [], checked: true, haveThis: false, actualQuantity: "", actualUnit: "", actualPrice: "", purchaseOmitted: false },
      { key: "ingredient:beans", label: "beans", need: "2 can", stock: "unknown", missing: "unknown", packageCount: "unknown", price: "unknown", sourceRecipeIds: [], checked: false, haveThis: false, actualQuantity: "", actualUnit: "", actualPrice: "", purchaseOmitted: false },
    ] });
    renderWithProviders(<GroceriesPage />);
    await waitFor(() => expect(screen.getByRole("button", { name: "Confirm purchases" })).toBeInTheDocument());
    await userEvent.type(screen.getByRole("textbox", { name: "Actual quantity rice" }), "40");
    await userEvent.clear(screen.getByRole("textbox", { name: "Actual unit rice" }));
    await userEvent.type(screen.getByRole("textbox", { name: "Actual unit rice" }), "g");
    await userEvent.type(screen.getByRole("textbox", { name: "Actual price rice" }), "1.20");
    await userEvent.click(screen.getByRole("checkbox", { name: "Omit beans" }));
    await userEvent.click(screen.getByRole("button", { name: "Confirm purchases" }));
    await waitFor(() => expect(confirmShoppingPurchases).toHaveBeenCalledWith(expect.objectContaining({ workspaceId: "w1", lines: [
      expect.objectContaining({ lineKey: "ingredient:rice", itemId: "rice", amount: "40", unit: "g", price: "1.20", omitted: false }),
      expect.objectContaining({ lineKey: "ingredient:beans", itemId: "beans", omitted: true }),
    ] })));
    expect(screen.queryByRole("button", { name: "Record purchase" })).not.toBeInTheDocument();
  });
  it("requires an explicit new trip after reloading a confirmed review", async () => {
    getShoppingPreview.mockResolvedValue({ revision: 1n, lines: [{ key: "ingredient:rice", label: "rice", need: "100 g", stock: "40 g", missing: "60 g", packageCount: "unknown", price: "unknown", sourceRecipeIds: [], checked: false, haveThis: false, actualQuantity: "40", actualUnit: "g", actualPrice: "1.20", purchaseOmitted: false }] });
    renderWithProviders(<GroceriesPage />);
    await waitFor(() => expect(screen.getByRole("status")).toHaveTextContent("This trip is confirmed"));
    expect(screen.queryByRole("button", { name: "Confirm purchases" })).not.toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Start a new trip" }));
    expect(screen.getByRole("button", { name: "Confirm purchases" })).toBeInTheDocument();
  });
  it("prepares and portions a declared recipe batch", async () => {
    getShoppingPreview.mockResolvedValue({ revision: 1n, lines: [] });
    listRecipes.mockResolvedValue([{ id: "r1", revision: 2n, name: "Soup", canonicalYield: "4", servingUnit: "serving", ingredients: [{ id: "rice", name: "Rice", amount: "500", unit: "g" }] }]);
    prepareInventoryBatch.mockResolvedValue({ id: "b1", recipeId: "r1", recipeRevision: 2n, yieldAmount: "4", availableAmount: "4", unit: "serving" });
    consumeInventoryBatchPortion.mockResolvedValue({ id: "b1", recipeId: "r1", recipeRevision: 2n, yieldAmount: "4", availableAmount: "3", unit: "serving" });
    renderWithProviders(<GroceriesPage />);
    await waitFor(() => expect(screen.getByRole("combobox", { name: "Batch recipe" })).toBeInTheDocument());
    await userEvent.selectOptions(screen.getByRole("combobox", { name: "Batch recipe" }), "r1");
    await userEvent.click(screen.getByRole("button", { name: "Prepare batch" }));
    await waitFor(() => expect(screen.getByText("Measured yield 4 serving · 4 serving available")).toBeInTheDocument());
    await userEvent.click(screen.getByRole("button", { name: "Consume portion from b1" }));
    expect(consumeInventoryBatchPortion).toHaveBeenCalledWith(expect.objectContaining({ batchId: "b1", amount: "1", recipeId: "r1" }));
  });
  it("restores only measured prepared yield as Kitchen stock", async () => {
    getShoppingPreview.mockResolvedValue({ revision: 1n, lines: [] });
    listRecipes.mockResolvedValue([{ id: "r1", revision: 3n, name: "Lentil soup" }]);
    listInventoryBatches.mockResolvedValue([{ id: "cook-session:cook-1", recipeId: "r1", recipeRevision: 3n, yieldAmount: "3.5", availableAmount: "2.5", unit: "bowls" }]);
    renderWithProviders(<GroceriesPage />);
    await waitFor(() => expect(screen.getByRole("heading", { name: "Prepared Kitchen stock" })).toBeInTheDocument());
    expect(screen.getByText("Lentil soup · revision 3")).toBeInTheDocument();
    expect(screen.getByText("Measured yield 3.5 bowls · 2.5 bowls available")).toBeInTheDocument();
  });
  it("reports selected equipment without filling in an unselected capability", async () => {
    getShoppingPreview.mockResolvedValue({ revision: 1n, lines: [] });
    getProfile.mockResolvedValue({ appliances: ["stove", "rice_cooker"] });
    renderWithProviders(<GroceriesPage />);
    await waitFor(() => expect(screen.getByRole("heading", { name: "Cooking equipment" })).toBeInTheDocument());
    expect(screen.getByText("stove")).toBeInTheDocument();
    expect(screen.getByText("rice cooker")).toBeInTheDocument();
    expect(screen.queryByText("oven")).not.toBeInTheDocument();
  });
  it("corrects a measured yield while preserving the saved remaining portion count", async () => {
    getShoppingPreview.mockResolvedValue({ revision: 1n, lines: [] });
    const original = { id: "b1", recipeId: "r1", recipeRevision: 3n, yieldAmount: "3.5", availableAmount: "2.5", unit: "bowls" };
    listInventoryBatches.mockResolvedValue([original]);
    correctInventoryBatchYield.mockResolvedValue({ ...original, yieldAmount: "4", availableAmount: "3" });
    renderWithProviders(<GroceriesPage />);
    const input = await screen.findByRole("textbox", { name: "Correct yield for b1" });
    await userEvent.clear(input); await userEvent.type(input, "4");
    await userEvent.click(screen.getByRole("button", { name: "Save yield correction" }));
    await waitFor(() => expect(correctInventoryBatchYield).toHaveBeenCalledWith(expect.objectContaining({ workspaceId: "w1", batchId: "b1", yieldAmount: "4", unit: "bowls" })));
    expect(await screen.findByText("Measured yield 4 bowls · 3 bowls available")).toBeInTheDocument();
    expect(screen.getByText(/recorded nutrition intake stays separate/i)).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Consume portion from b1" }));
    expect(consumeInventoryBatchPortion).toHaveBeenCalledWith(expect.objectContaining({ workspaceId: "w1", batchId: "b1", amount: "1", unit: "bowls" }));
  });
  it("downloads the current checklist as CSV", async () => {
    getShoppingPreview.mockResolvedValue({ revision: 1n, lines: [] });
    const createObjectURL = vi.fn().mockReturnValue("blob:test");
    const revokeObjectURL = vi.fn();
    Object.defineProperty(URL, "createObjectURL", { value: createObjectURL, configurable: true });
    Object.defineProperty(URL, "revokeObjectURL", { value: revokeObjectURL, configurable: true });
    const click = vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(() => undefined);
    renderWithProviders(<GroceriesPage />);
    await waitFor(() => expect(screen.getByRole("button", { name: "Download grocery CSV" })).toBeInTheDocument());
    await userEvent.click(screen.getByRole("button", { name: "Download grocery CSV" }));
    expect(exportGroceriesCSV).toHaveBeenCalledWith({ workspaceId: "w1", expectedRevision: 1n });
    expect(createObjectURL).toHaveBeenCalled();
    expect(revokeObjectURL).toHaveBeenCalledWith("blob:test");
    click.mockRestore();
  });
});
