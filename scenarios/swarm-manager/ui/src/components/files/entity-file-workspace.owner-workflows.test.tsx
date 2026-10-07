import { useState } from "react";
import { act, cleanup, fireEvent, screen, waitFor, within } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";
import { getSpatialNav } from "@vrooli/iframe-bridge/spatial";
import { EntityFileWorkspace } from "./entity-file-workspace";
import type { FileActionType } from "./entity-file-browser";
import { FileServiceProvider, useFileService } from "../../contexts/FileServiceContext";
import { createGoalFileServiceAdapter } from "../../services/goals-file-service-adapter";
import { createBacklogFileServiceAdapter } from "../../services/backlog/backlog-file-service-adapter";
import { useActionMutation } from "../../hooks/useActionMutation";
import { createTestQueryClient, renderWithProviders } from "../../test-utils";
import { ApiError, defaultApiClient } from "../../lib/api-client";
import { DEFAULT_SETTINGS } from "../../services/settings-service";
import type { BacklogFile } from "../../types";

// Actual workspace/context/preview/upload/browser/adapters. This parent fixture
// owns list selection and file-operation callbacks; page-specific handlers are
// not replaced or qualified by these tests. Only network methods are substituted.
const notes: BacklogFile = { name: "notes.md", path: "docs/notes.md", type: "file", size: 42 };
let base: string;
let files: BacklogFile[];
let content: string;
let locked = false;
const clients: QueryClient[] = [];
const unexpected: string[] = [];
const allowed = new Set<string>();
function proto(file: BacklogFile) { return { ...file, size: String(file.size) }; }
function fileText(file: File): Promise<string> { return new Promise((resolve, reject) => { const reader = new FileReader(); reader.onload = () => resolve(String(reader.result)); reader.onerror = () => reject(reader.error); reader.readAsText(file); }); }
function Parent() {
  const service = useFileService(); const client = useQueryClient();
  const [selected, select] = useState<BacklogFile | null>(null);
  const query = useQuery({ queryKey: [...service.queryKeyPrefix, "files"], queryFn: service.getFiles });
  const operation = useActionMutation({
    mutationFn: ({ action, target, destination }: { action: FileActionType; target: BacklogFile; destination?: string }) => {
      if (action === "delete") return service.deleteFile(target.path);
      if (action === "rename") return service.renameFile(target.path, destination!);
      if (action === "move") return service.moveFile(target.path, destination!);
      return service.copyFile(target.path, destination!);
    },
    errorMessage: "Owner file operation failed", allowRetry: false,
    onSuccess: (result) => { if (result.deletedPath === selected?.path) select(null); else if (result.file) select(result.file); void client.invalidateQueries({ queryKey: [...service.queryKeyPrefix, "files"] }); },
  });
  return <EntityFileWorkspace files={query.data} isLoadingFiles={query.isLoading} filesError={query.error} selectedFile={selected} isLocked={locked} onFileSelect={select} onRefetchFiles={() => { void query.refetch(); }} onUploadComplete={() => { void query.refetch(); }} fileActionPending={operation.isPending} onFileAction={(action, target, destination) => operation.mutate({ action, target, destination })} />;
}
function mount(owner: "goal" | "backlog") {
  const service = owner === "goal" ? createGoalFileServiceAdapter("fixture-goal") : createBacklogFileServiceAdapter("fix", "fixture-work");
  base = owner === "goal" ? "/goals/fixture-goal/files" : "/backlog/fix/fixture-work/files";
  const client = createTestQueryClient(); clients.push(client);
  client.setQueryDefaults(["unrelated-owner", "files"], { gcTime: Infinity }); client.setQueryData(["unrelated-owner", "files"], [{ sentinel: "unchanged" }]);
  return { service, client, ...renderWithProviders(<FileServiceProvider value={service}><Parent /></FileServiceProvider>, { queryClient: client }) };
}
async function selectNotes() { fireEvent.click(await screen.findByTestId("file-tree-button-docs/notes.md")); const editor = await screen.findByTestId("file-preview-editor"); await waitFor(() => expect(editor).toHaveValue(content)); return editor; }
async function menu(label: string) { if (label === "Delete") await waitFor(() => expect(clients.at(-1)?.getQueryData<{ deleteConfirmation: { backlogFile: string } }>(["settings"])?.deleteConfirmation.backlogFile).toBe("simple")); fireEvent.click(screen.getByTestId("file-header-actions-trigger")); fireEvent.click(await screen.findByRole("button", { name: label })); }
function listReads() { return vi.mocked(defaultApiClient.get).mock.calls.filter(([path]) => path === base).length; }
beforeEach(() => {
  base = ""; files = [{ ...notes }]; content = "# Original owner content"; locked = false; unexpected.length = 0; allowed.clear();
  vi.spyOn(defaultApiClient, "get").mockImplementation(async path => {
    if (path === "/settings") { const { deleteConfirmation: _, ...settings } = DEFAULT_SETTINGS; return { settings: { ...settings, deleteConfirmationLevels: { backlogFile: "DELETE_CONFIRM_LEVEL_SIMPLE" } } }; }
    if (path === base) return { files: files.map(proto) };
    if (path === base + "/docs/notes.md" || (path.startsWith(base + "/") && files.some(file => path === base + "/" + file.path))) return content;
    unexpected.push(path); throw new Error(`Unexpected owner file read ${path}`);
  });
  for (const method of ["post", "put", "patch", "delete"] as const) vi.spyOn(defaultApiClient, method).mockRejectedValue(new Error(`Unexpected file ${method}`));
  vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new Error("Unexpected workspace fetch")));
});
afterEach(() => {
  cleanup(); getSpatialNav()?.dispose();
  try { expect(unexpected).toEqual([]); for (const method of ["post", "put", "patch", "delete"] as const) if (!allowed.has(method)) expect(defaultApiClient[method]).not.toHaveBeenCalled(); expect(fetch).not.toHaveBeenCalled(); for (const client of clients) expect(client.getQueryData(["unrelated-owner", "files"])).toEqual([{ sentinel: "unchanged" }]); }
  finally { clients.splice(0).forEach(client => client.clear()); vi.restoreAllMocks(); vi.unstubAllGlobals(); }
});

for (const owner of ["goal", "backlog"] as const) describe(`${owner} file owner through actual shared workspace`, () => {
  it("saves edited bytes to the exact file directory and waits for owner completion before cache refresh", async () => {
    allowed.add("post"); let finish!: (value: unknown) => void; vi.mocked(defaultApiClient.post).mockImplementation(() => new Promise(resolve => { finish = resolve; }));
    const { client, service } = mount(owner); const editor = await selectNotes(); const before = listReads();
    fireEvent.change(editor, { target: { value: "# Accepted revised bytes" } }); fireEvent.click(screen.getByTestId("file-preview-save")); await waitFor(() => expect(defaultApiClient.post).toHaveBeenCalledOnce());
    const [path, payload, options] = vi.mocked(defaultApiClient.post).mock.calls[0]!; expect(path).toBe(base); expect(options).toEqual({ headers: {} }); expect(payload).toBeInstanceOf(FormData); const form = payload as FormData; const file = form.get("file") as File;
    expect(file.name).toBe("notes.md"); expect(file.type).toBe("text/markdown"); expect(form.get("path")).toBe("docs"); expect(await fileText(file)).toBe("# Accepted revised bytes");
    await waitFor(() => expect(screen.getByTestId("file-preview-save")).toBeDisabled()); expect(screen.getByTestId("file-preview-discard")).toBeDisabled(); fireEvent.click(screen.getByTestId("file-preview-save")); expect(defaultApiClient.post).toHaveBeenCalledOnce(); expect(listReads()).toBe(before);
    expect(client.getQueryData([...service.queryKeyPrefix, "files", notes.path, "content"])).toBe("# Original owner content"); content = "# Accepted revised bytes";
    await act(async () => finish({ file: proto(notes) })); await waitFor(() => expect(client.getQueryData([...service.queryKeyPrefix, "files", notes.path, "content"])).toBe(content)); await waitFor(() => expect(listReads()).toBeGreaterThan(before)); expect(screen.getByTestId("file-preview-save")).toBeDisabled();
  });
  it("retains refused edited bytes and cached owner content without a fallback write or refresh", async () => {
    allowed.add("post"); vi.mocked(defaultApiClient.post).mockRejectedValue(new ApiError("http", "Owner denied file save", { status: 403 })); const { client, service } = mount(owner); const editor = await selectNotes(); const before = listReads();
    fireEvent.change(editor, { target: { value: "# Unaccepted bytes" } }); fireEvent.click(screen.getByTestId("file-preview-save")); expect(await screen.findByText("You don't have permission to access this resource.")).toBeVisible(); expect(editor).toHaveValue("# Unaccepted bytes"); expect(defaultApiClient.post).toHaveBeenCalledOnce(); expect(listReads()).toBe(before); expect(client.getQueryData([...service.queryKeyPrefix, "files", notes.path, "content"])).toBe("# Original owner content");
  });
  it("discards a local draft without any owner mutation", async () => { mount(owner); const editor = await selectNotes(); fireEvent.change(editor, { target: { value: "Unsubmitted bytes" } }); fireEvent.click(screen.getByTestId("file-preview-discard")); expect(editor).toHaveValue(content); expect(defaultApiClient.post).not.toHaveBeenCalled(); });
  it("uploads the selected actual File through the owner adapter and refreshes only on completion", async () => {
    allowed.add("post"); let finish!: (value: unknown) => void; vi.mocked(defaultApiClient.post).mockImplementation(() => new Promise(resolve => { finish = resolve; })); mount(owner); await screen.findByTestId("file-tree-button-docs/notes.md"); const before = listReads();
    fireEvent.click(screen.getByRole("button", { name: "Upload files" })); const file = new File(["Owner upload"], "uploaded.txt", { type: "text/plain" }); fireEvent.change(screen.getByTestId("file-upload-input"), { target: { files: [file] } }); await waitFor(() => expect(defaultApiClient.post).toHaveBeenCalledOnce()); const [path, payload, options] = vi.mocked(defaultApiClient.post).mock.calls[0]!; expect(path).toBe(base); expect((payload as FormData).get("file")).toBe(file); expect((payload as FormData).get("path")).toBeNull(); expect(options).toEqual({ headers: {} }); expect(listReads()).toBe(before);
    const uploaded: BacklogFile = { name: "uploaded.txt", path: "uploaded.txt", type: "file", size: 12 }; files = [...files, uploaded]; await act(async () => finish({ file: proto(uploaded) })); expect(await screen.findByTestId("file-tree-button-uploaded.txt")).toBeVisible(); expect(defaultApiClient.post).toHaveBeenCalledOnce();
  });
  it("shows upload refusal without automatically retrying or changing the owner tree", async () => {
    allowed.add("post"); vi.mocked(defaultApiClient.post).mockRejectedValue(new ApiError("http", "Owner denied upload", { status: 403 })); mount(owner); await screen.findByTestId("file-tree-button-docs/notes.md"); const before = listReads(); fireEvent.click(screen.getByRole("button", { name: "Upload files" })); fireEvent.change(screen.getByTestId("file-upload-input"), { target: { files: [new File(["refused"], "refused.txt")] } }); expect(await screen.findByText("You don't have permission to access this resource.")).toBeVisible(); expect(screen.getByTestId("file-upload-retry-0")).toBeVisible(); expect(defaultApiClient.post).toHaveBeenCalledOnce(); expect(listReads()).toBe(before); expect(screen.queryByTestId("file-tree-button-refused.txt")).toBeNull();
  });
  it.each([["Rename", "renamed.md", "rename", "docs/renamed.md"], ["Move", "archive/notes.md", "move", "archive/notes.md"], ["Copy", "copies/notes.md", "copy", "copies/notes.md"]] as const)("%s offers only the exact current path through its real adapter", async (label, input, operation, destination) => {
    allowed.add("patch"); let finish!: (value: unknown) => void; vi.mocked(defaultApiClient.patch).mockImplementation(() => new Promise(resolve => { finish = resolve; })); mount(owner); await selectNotes(); const before = listReads(); await menu(label); const dialog = screen.getByRole("dialog", { name: `${label} file` }); fireEvent.change(within(dialog).getByRole("textbox"), { target: { value: input } }); fireEvent.click(within(dialog).getByRole("button", { name: "Apply" })); await waitFor(() => expect(vi.mocked(defaultApiClient.patch).mock.calls).toEqual([[base, { operation, source_path: "docs/notes.md", destination_path: destination }]])); expect(listReads()).toBe(before);
    const result: BacklogFile = { ...notes, path: destination, name: destination.split("/").pop()! }; files = operation === "copy" ? [...files, result] : [result]; await act(async () => finish({ file: proto(result) })); await waitFor(() => expect(listReads()).toBeGreaterThan(before)); expect(screen.getByTestId("file-preview-name")).toHaveTextContent(result.name); expect(defaultApiClient.patch).toHaveBeenCalledOnce();
  });
  it("reports an operation refusal without changing the list or selecting a fabricated destination", async () => {
    allowed.add("patch"); vi.mocked(defaultApiClient.patch).mockRejectedValue(new ApiError("http", "Owner denied rename", { status: 403 })); mount(owner); await selectNotes(); const before = listReads(); await menu("Rename"); const dialog = screen.getByRole("dialog", { name: "Rename file" }); fireEvent.change(within(dialog).getByRole("textbox"), { target: { value: "refused.md" } }); fireEvent.click(within(dialog).getByRole("button", { name: "Apply" }));
    expect(await screen.findByText("You don't have permission to access this resource.")).toBeVisible(); expect(vi.mocked(defaultApiClient.patch).mock.calls).toEqual([[base, { operation: "rename", source_path: "docs/notes.md", destination_path: "docs/refused.md" }]]); expect(listReads()).toBe(before); expect(screen.getByTestId("file-preview-name")).toHaveTextContent("notes.md"); expect(screen.queryByTestId("file-tree-button-docs/refused.md")).toBeNull();
  });
  it("rejects invalid rename input and cancels without any adapter write", async () => {
    mount(owner); await selectNotes(); const before = listReads(); await menu("Rename"); const dialog = screen.getByRole("dialog", { name: "Rename file" }); fireEvent.change(within(dialog).getByRole("textbox"), { target: { value: "other/path.md" } }); fireEvent.click(within(dialog).getByRole("button", { name: "Apply" }));
    expect(await screen.findByText("Rename requires a file or folder name without slashes.")).toBeVisible(); expect(defaultApiClient.patch).not.toHaveBeenCalled(); fireEvent.click(within(dialog).getByRole("button", { name: "Cancel" })); expect(screen.queryByRole("dialog", { name: "Rename file" })).toBeNull(); expect(listReads()).toBe(before);
  });
  it("cancels deletion of the selected file without offering a write", async () => { mount(owner); await selectNotes(); await menu("Delete"); const dialog = screen.getByRole("alertdialog", { name: "Delete file" }); expect(dialog).toHaveTextContent("docs/notes.md"); fireEvent.click(within(dialog).getByRole("button", { name: "Cancel" })); expect(screen.queryByRole("alertdialog")).toBeNull(); expect(defaultApiClient.patch).not.toHaveBeenCalled(); expect(screen.getByTestId("file-preview-name")).toHaveTextContent("notes.md"); });
  it("deletes only the confirmed owner path and clears selection after authoritative completion", async () => {
    allowed.add("patch"); let finish!: (value: unknown) => void; vi.mocked(defaultApiClient.patch).mockImplementation(() => new Promise(resolve => { finish = resolve; })); mount(owner); await selectNotes(); const before = listReads(); await menu("Delete"); fireEvent.click(within(screen.getByRole("alertdialog", { name: "Delete file" })).getByRole("button", { name: "Delete" })); await waitFor(() => expect(vi.mocked(defaultApiClient.patch).mock.calls).toEqual([[base, { operation: "delete", source_path: "docs/notes.md" }]])); expect(screen.getByTestId("file-preview-name")).toHaveTextContent("notes.md"); expect(listReads()).toBe(before); files = []; await act(async () => finish({ deleted_path: "docs/notes.md" })); expect(await screen.findByText("No file selected")).toBeVisible(); await waitFor(() => expect(listReads()).toBeGreaterThan(before));
  });
  it("reads the canonical specification while disabling edit and every file action", async () => {
    const canonical = owner === "goal" ? "goal.json" : "spec.json"; files = [{ name: canonical, path: canonical, type: "file", size: 12 }]; content = '{"owner":"canonical"}'; mount(owner); fireEvent.click(await screen.findByTestId(`file-tree-button-${canonical}`)); const editor = await screen.findByTestId("file-preview-editor"); await waitFor(() => expect(editor).toHaveValue(content)); expect(editor).toHaveAttribute("data-read-only", "true"); expect(screen.getByTestId("file-read-only-badge")).toHaveTextContent("Read-only"); expect(screen.queryByTestId("file-preview-save")).toBeNull(); fireEvent.click(screen.getByTestId("file-header-actions-trigger")); for (const label of ["Rename", "Move", "Copy", "Delete"]) { const button = screen.getByRole("button", { name: new RegExp(label) }); expect(button).toBeDisabled(); fireEvent.click(button); } expect(defaultApiClient.post).not.toHaveBeenCalled(); expect(defaultApiClient.patch).not.toHaveBeenCalled();
  });
  it("keeps upload disabled when the parent reports the owner workspace locked", async () => { locked = true; mount(owner); await screen.findByTestId("file-tree-button-docs/notes.md"); const upload = screen.getByRole("button", { name: "Upload files" }); expect(upload).toBeDisabled(); fireEvent.click(upload); expect(screen.queryByTestId("file-upload-input")).toBeNull(); expect(defaultApiClient.post).not.toHaveBeenCalled(); });
});
