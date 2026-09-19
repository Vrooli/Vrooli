import { beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";

const ensureWorkspace = vi.hoisted(() => vi.fn());
const listRecipes = vi.hoisted(() => vi.fn());
const exportRecipes = vi.hoisted(() => vi.fn());
const exportWorkspace = vi.hoisted(() => vi.fn());
const exportGroceriesCSV = vi.hoisted(() => vi.fn());
const exportWeeklyPDF = vi.hoisted(() => vi.fn());
const exportRecipePDF = vi.hoisted(() => vi.fn());
const previewWorkspaceImport = vi.hoisted(() => vi.fn());
const applyWorkspaceImport = vi.hoisted(() => vi.fn());
const previewRecipesImport = vi.hoisted(() => vi.fn());
const applyRecipesImport = vi.hoisted(() => vi.fn());

vi.mock("../../api/workspace", () => ({ ensureWorkspace }));
vi.mock("../../api/recipes", () => ({ listRecipes }));
vi.mock("../../api/portability", () => ({ exportRecipes, exportWorkspace, exportGroceriesCSV, exportWeeklyPDF, exportRecipePDF, previewWorkspaceImport, applyWorkspaceImport, previewRecipesImport, applyRecipesImport }));

import { DataTransferPage } from "./DataTransferPage";

describe("DataTransferPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    ensureWorkspace.mockResolvedValue({ id: "w1", revision: 2n });
    listRecipes.mockResolvedValue([{ id: "r1", name: "Soup" }]);
    exportRecipes.mockResolvedValue({ filename: "daily-recipes.json", content: "{}", omissions: [] });
    exportWorkspace.mockResolvedValue({ filename: "daily-workspace.json", content: "{}", omissions: [] });
    exportGroceriesCSV.mockResolvedValue({ filename: "daily-groceries.csv", content: "key,label\n", revision: 2n });
    exportWeeklyPDF.mockResolvedValue({ filename: "weekly.pdf", content: new Uint8Array([37, 80, 68, 70]), revision: 2n });
    exportRecipePDF.mockResolvedValue({ filename: "recipe-r1.pdf", content: new Uint8Array([37, 80, 68, 70]) });
    previewWorkspaceImport.mockResolvedValue({ valid: true, format: "daily.workspace", schemaVersion: 2, recordCount: 3, recordKinds: ["workspace"], omissions: [], errors: [] });
    applyWorkspaceImport.mockResolvedValue({ workspaceRevision: 3n, recipesApplied: 1, checkpointId: "cp1" });
    previewRecipesImport.mockResolvedValue({ valid: true, format: "daily.recipes", schemaVersion: 2, recipeCount: 1, duplicateCount: 0, conflictCount: 0, errors: [] });
    applyRecipesImport.mockResolvedValue({ workspaceRevision: 3n, recipesApplied: 1, recipesSkipped: 0, remappedIds: [] });
    Object.defineProperty(URL, "createObjectURL", { configurable: true, value: vi.fn(() => "blob:test") });
    Object.defineProperty(URL, "revokeObjectURL", { configurable: true, value: vi.fn() });
    vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(() => undefined);
  });

  afterEach(() => { vi.restoreAllMocks(); delete (URL as unknown as { createObjectURL?: unknown }).createObjectURL; delete (URL as unknown as { revokeObjectURL?: unknown }).revokeObjectURL; });

  it("exposes distinct native, PDF, and CSV export cards", async () => {
    renderWithProviders(<DataTransferPage />);
    await waitFor(() => expect(screen.getByRole("option", { name: "Soup" })).toBeInTheDocument());
    expect(screen.getByRole("button", { name: "Export recipes" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Download workspace backup" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Export weekly PDF" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Export grocery CSV" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Export recipe PDF" })).toBeInTheDocument();
    expect(screen.getByRole("combobox", { name: "PDF paper size" })).toHaveValue("A4");
  });

  it("keeps the transfer center free of automated accessibility violations", async () => {
    const { container } = renderWithProviders(<DataTransferPage />);
    await waitFor(() => expect(screen.getByRole("option", { name: "Soup" })).toBeInTheDocument());
    await expectNoA11yViolations(container);
  });

  it("runs every export and staged restore action", async () => {
    renderWithProviders(<DataTransferPage />);
    await waitFor(() => expect(screen.getByRole("option", { name: "Soup" })).toBeInTheDocument());
    await userEvent.click(screen.getByRole("button", { name: "Export recipes" }));
    await userEvent.click(screen.getByRole("button", { name: "Download workspace backup" }));
    await userEvent.click(screen.getByRole("button", { name: "Export weekly PDF" }));
    await userEvent.click(screen.getByRole("button", { name: "Export grocery CSV" }));
    await userEvent.selectOptions(screen.getByRole("combobox", { name: "PDF paper size" }), "LETTER");
    await userEvent.selectOptions(screen.getByRole("combobox", { name: "Recipe for PDF" }), "r1");
    await userEvent.click(screen.getByRole("button", { name: "Export recipe PDF" }));
    const file = new File(["{}"], "backup.json", { type: "application/json" });
    Object.defineProperty(file, "text", { value: async () => "{}" });
    fireEvent.change(screen.getByLabelText("Choose a native daily JSON file"), { target: { files: [file] } });
    await waitFor(() => expect(screen.getByRole("button", { name: "Restore this validated backup" })).toBeInTheDocument());
    await userEvent.click(screen.getByRole("button", { name: "Restore this validated backup" }));
    expect(exportRecipes).toHaveBeenCalledWith("w1");
    expect(exportWorkspace).toHaveBeenCalledWith("w1");
    expect(exportWeeklyPDF).toHaveBeenCalledWith({ workspaceId: "w1", expectedRevision: 2n, pageSize: "A4" });
    expect(exportGroceriesCSV).toHaveBeenCalledWith({ workspaceId: "w1", expectedRevision: 2n });
    expect(exportRecipePDF).toHaveBeenCalledWith({ workspaceId: "w1", recipeId: "r1", pageSize: "LETTER" });
    expect(applyWorkspaceImport).toHaveBeenCalledWith(expect.objectContaining({ workspaceId: "w1", expectedWorkspaceRevision: 2n, contentJson: "{}" }));
  });

  it("routes a native recipe file through staged conflict review and apply", async () => {
    const user = userEvent.setup();
    renderWithProviders(<DataTransferPage />);
    await waitFor(() => expect(screen.getByRole("option", { name: "Soup" })).toBeInTheDocument());
    const file = new File(['{"format":"daily.recipes"}'], "recipes.json", { type: "application/json" });
    Object.defineProperty(file, "text", { value: async () => '{"format":"daily.recipes"}' });
    fireEvent.change(screen.getByLabelText("Choose a native daily JSON file"), { target: { files: [file] } });
    await waitFor(() => expect(screen.getByText(/1 recipes · 0 duplicates · 0 conflicts/)).toBeInTheDocument());
    await user.selectOptions(screen.getByLabelText("Conflict policy"), "replace");
    await user.click(screen.getByRole("button", { name: "Apply staged recipe import" }));
    expect(previewRecipesImport).toHaveBeenCalledWith({ workspaceId: "w1", contentJson: '{"format":"daily.recipes"}' });
    expect(applyRecipesImport).toHaveBeenCalledWith(expect.objectContaining({ workspaceId: "w1", expectedWorkspaceRevision: 2n, conflictPolicy: "replace" }));
  });

  it("reports export and restore failures without pretending a file was created", async () => {
    const user = userEvent.setup();
    renderWithProviders(<DataTransferPage />);
    await waitFor(() => expect(screen.getByRole("option", { name: "Soup" })).toBeInTheDocument());
    exportRecipes.mockRejectedValueOnce("recipes unavailable");
    await user.click(screen.getByRole("button", { name: "Export recipes" }));
    await waitFor(() => expect(screen.getByRole("status")).toHaveTextContent("Unable to export recipes."));
    exportWorkspace.mockRejectedValueOnce("workspace unavailable");
    await user.click(screen.getByRole("button", { name: "Download workspace backup" }));
    await waitFor(() => expect(screen.getByRole("status")).toHaveTextContent("Unable to export workspace."));
    exportWeeklyPDF.mockRejectedValueOnce("week unavailable");
    await user.click(screen.getByRole("button", { name: "Export weekly PDF" }));
    await waitFor(() => expect(screen.getByRole("status")).toHaveTextContent("Unable to export the weekly PDF."));
    exportGroceriesCSV.mockRejectedValueOnce("groceries unavailable");
    await user.click(screen.getByRole("button", { name: "Export grocery CSV" }));
    await waitFor(() => expect(screen.getByRole("status")).toHaveTextContent("Unable to export groceries."));
    await user.click(screen.getByRole("button", { name: "Export recipe PDF" }));
    expect(screen.getByRole("status")).toHaveTextContent("Choose a recipe before exporting its PDF.");
    await user.selectOptions(screen.getByRole("combobox", { name: "Recipe for PDF" }), "r1");
    exportRecipePDF.mockRejectedValueOnce("recipe unavailable");
    await user.click(screen.getByRole("button", { name: "Export recipe PDF" }));
    await waitFor(() => expect(screen.getByRole("status")).toHaveTextContent("Unable to export the recipe PDF."));
    previewWorkspaceImport.mockResolvedValueOnce({ valid: false, format: "", schemaVersion: 0, recordCount: 0, recordKinds: [], omissions: [], errors: ["bad backup"] });
    const file = new File(["bad"], "bad.json", { type: "application/json" });
    Object.defineProperty(file, "text", { value: async () => "bad" });
    fireEvent.change(screen.getByLabelText("Choose a native daily JSON file"), { target: { files: [file] } });
    await waitFor(() => expect(screen.getByText("Import cannot be applied")).toBeInTheDocument());
    previewWorkspaceImport.mockRejectedValueOnce("preview unavailable");
    fireEvent.change(screen.getByLabelText("Choose a native daily JSON file"), { target: { files: [file] } });
    await waitFor(() => expect(screen.getByRole("status")).toHaveTextContent("Unable to inspect workspace import."));
    previewWorkspaceImport.mockResolvedValueOnce({ valid: true, format: "daily.workspace", schemaVersion: 2, recordCount: 1, recordKinds: ["workspace"], omissions: [], errors: [] });
    fireEvent.change(screen.getByLabelText("Choose a native daily JSON file"), { target: { files: [file] } });
    await waitFor(() => expect(screen.getByRole("button", { name: "Restore this validated backup" })).toBeInTheDocument());
    applyWorkspaceImport.mockRejectedValueOnce("restore unavailable");
    await user.click(screen.getByRole("button", { name: "Restore this validated backup" }));
    await waitFor(() => expect(screen.getByText("Unable to apply workspace restore.")).toBeInTheDocument());
  });

  it("reports initial transfer-option failures and visible omissions", async () => {
    ensureWorkspace.mockRejectedValueOnce("workspace unavailable");
    const first = renderWithProviders(<DataTransferPage />);
    await waitFor(() => expect(screen.getByRole("status")).toHaveTextContent("Unable to load transfer options."));
    first.unmount();
    vi.clearAllMocks();
    ensureWorkspace.mockResolvedValue({ id: "w1", revision: 2n });
    listRecipes.mockResolvedValue([]);
    exportWorkspace.mockResolvedValue({ filename: "daily-workspace.json", content: "{}", omissions: ["inventory"] });
    renderWithProviders(<DataTransferPage />);
    await waitFor(() => expect(screen.getByRole("button", { name: "Download workspace backup" })).toBeInTheDocument());
    await userEvent.click(screen.getAllByRole("button", { name: "Download workspace backup" }).at(-1)!);
    await waitFor(() => expect(screen.getByText("Backup ready. Omitted domains: 1.")).toBeInTheDocument());
    previewWorkspaceImport.mockResolvedValueOnce({ valid: true, format: "daily.workspace", schemaVersion: 2, recordCount: 1, recordKinds: ["workspace"], omissions: ["inventory"], errors: [] });
    const file = new File(["{}"], "backup.json", { type: "application/json" });
    Object.defineProperty(file, "text", { value: async () => "{}" });
    fireEvent.change(screen.getByLabelText("Choose a native daily JSON file"), { target: { files: [file] } });
    await waitFor(() => expect(screen.getByText("Omissions: inventory")).toBeInTheDocument());
  });
});
