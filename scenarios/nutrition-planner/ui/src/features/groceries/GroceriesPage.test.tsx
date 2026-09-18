import { beforeEach, describe, expect, it, vi } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "../../test-utils";

const getShoppingPreview = vi.hoisted(() => vi.fn());
const setShoppingChecked = vi.hoisted(() => vi.fn());
const listInventoryEvents = vi.hoisted(() => vi.fn());
const recordInventoryEvent = vi.hoisted(() => vi.fn());
const exportGroceriesCSV = vi.hoisted(() => vi.fn());
vi.mock("../../api/planning", () => ({ getShoppingPreview, setShoppingChecked }));
vi.mock("../../api/inventory", () => ({ listInventoryEvents, recordInventoryEvent }));
vi.mock("../../api/portability", () => ({ exportGroceriesCSV }));
vi.mock("../../api/workspace", () => ({ ensureWorkspace: vi.fn().mockResolvedValue({ id: "w1" }) }));
import { GroceriesPage } from "./GroceriesPage";

describe("GroceriesPage", () => {
  beforeEach(() => { vi.clearAllMocks(); setShoppingChecked.mockResolvedValue(true); listInventoryEvents.mockResolvedValue([]); recordInventoryEvent.mockResolvedValue({ id: "p1", workspaceId: "w1", kind: "purchase", itemId: "rice", batchId: "", amount: "500", unit: "g", recipeId: "", createdAt: "2026-09-18T00:00:00Z" }); exportGroceriesCSV.mockResolvedValue({ filename: "daily-groceries.csv", content: "key,label\n", revision: 1n }); });
  it("shows unknown facts and persists checklist-only state", async () => {
    getShoppingPreview.mockResolvedValue({ revision: 1n, lines: [{ key: "ingredient:rice", label: "rice", need: "unknown", stock: "unknown", missing: "unknown", packageCount: "unknown", price: "unknown", sourceRecipeIds: ["r1"], checked: false }] });
    renderWithProviders(<GroceriesPage />);
    await waitFor(() => expect(screen.getByText("rice")).toBeInTheDocument());
    expect(screen.getByText(/Price: unknown/)).toBeInTheDocument();
    await userEvent.click(screen.getByRole("checkbox", { name: "Check rice" }));
    expect(setShoppingChecked).toHaveBeenCalledWith({ workspaceId: "w1", lineKey: "ingredient:rice", checked: true });
  });
  it("keeps the line unchanged when checklist persistence fails", async () => {
    getShoppingPreview.mockResolvedValue({ revision: 1n, lines: [{ key: "ingredient:rice", label: "rice", need: "unknown", stock: "unknown", missing: "unknown", packageCount: "unknown", price: "unknown", sourceRecipeIds: [], checked: false }] });
    setShoppingChecked.mockRejectedValue(new Error("offline"));
    renderWithProviders(<GroceriesPage />);
    await waitFor(() => expect(screen.getByRole("checkbox", { name: "Check rice" })).toBeInTheDocument());
    await userEvent.click(screen.getByRole("checkbox", { name: "Check rice" }));
    expect(screen.getByRole("checkbox", { name: "Check rice" })).not.toBeChecked();
  });
  it("records purchased stock separately from checklist state", async () => {
    getShoppingPreview.mockResolvedValue({ revision: 1n, lines: [] });
    renderWithProviders(<GroceriesPage />);
    await waitFor(() => expect(screen.getByRole("button", { name: "Record purchase" })).toBeInTheDocument());
    await userEvent.type(screen.getByRole("textbox", { name: "Inventory item" }), "rice");
    await userEvent.type(screen.getByRole("textbox", { name: "Inventory amount" }), "500");
    await userEvent.click(screen.getByRole("button", { name: "Record purchase" }));
    expect(recordInventoryEvent).toHaveBeenCalledWith(expect.objectContaining({ workspaceId: "w1", kind: "purchase", itemId: "rice", amount: "500", unit: "g" }));
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
