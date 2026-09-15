import { renderWithProviders } from "../test-utils/renderWithProviders";
import { create } from "@bufbuild/protobuf";
import { PushSafetyReportSchema, PushRecoveryArtifactSchema } from "@vrooli/proto-types/git-control-tower/v1/repo/repo_pb";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, expect, test, vi } from "vitest";
import { PushSafetyDialog } from "./PushSafetyDialog";
import { getPushRecovery, inspectPushSafety, preparePushRecovery } from "../lib/api-push-safety";

vi.mock("../lib/api-push-safety", () => ({ getPushRecovery: vi.fn(), inspectPushSafety: vi.fn(), preparePushRecovery: vi.fn() }));
const blocked = () => create(PushSafetyReportSchema, {
  complete: true, state: "blocked", reason: "Push blocked by oversized files.",
  head: "old-head", base: "remote-base", remote: "origin", branch: "agi", fingerprint: "preview-1",
  limit: 104857600n, commits: ["first", "second", "third", "fourth"], canPrepare: true,
  files: [{ oid: "blob", bytes: 440820652n, paths: ["bundle/vault"], commits: ["first"], blocked: true }],
});
beforeEach(() => {
  vi.resetAllMocks();
  vi.mocked(window.matchMedia).mockImplementation((query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    addListener: vi.fn(),
    removeListener: vi.fn(),
    dispatchEvent: vi.fn(),
  }));
  vi.mocked(inspectPushSafety).mockResolvedValue(blocked());
});

test("blocked push explains historical scope and requires explicit preparation consent", async () => {
  const push = vi.fn(); renderWithProviders(<PushSafetyDialog repoId="one" onClose={vi.fn()} onPush={push} />);
  expect(await screen.findByText("bundle/vault")).toBeVisible();
  expect(screen.queryByRole("button", { name: "Continue to push" })).not.toBeInTheDocument();
  expect(screen.getByText(/Deleting the file in a new commit/)).toBeVisible();
  const prepare = screen.getByRole("button", { name: "Prepare isolated recovery" });
  expect(prepare).toBeDisabled(); expect(preparePushRecovery).not.toHaveBeenCalled();
  vi.mocked(preparePushRecovery).mockResolvedValue(create(PushRecoveryArtifactSchema, {
    state: "prepared", fingerprint: "preview-1", message: "Active branch unchanged. Committed history only.", originalBundle: "/backup/original.bundle",
    repairedBundle: "/backup/repaired.bundle", mappings: [{ original: "first", replacement: "replacement" }],
  }));
  fireEvent.click(screen.getByRole("checkbox")); fireEvent.click(prepare);
  expect(await screen.findByText("Prepared artifacts checked — not applied")).toBeVisible();
  expect(preparePushRecovery).toHaveBeenCalledWith(expect.objectContaining({ fingerprint: "preview-1" }), "one");
  expect(push).not.toHaveBeenCalled();
  expect(screen.getByText(/these artifacts do not back up uncommitted work/i)).toBeInTheDocument();
});

test("incomplete scan offers refresh without a push bypass", async () => {
  vi.mocked(inspectPushSafety).mockResolvedValue(create(PushSafetyReportSchema, { state: "unknown", reason: "Remote authentication unavailable." }));
  renderWithProviders(<PushSafetyDialog onClose={vi.fn()} onPush={vi.fn()} />);
  expect(await screen.findByText("Remote authentication unavailable.")).toBeVisible();
  expect(screen.queryByRole("button", { name: "Continue to push" })).not.toBeInTheDocument();
  expect(screen.queryByRole("button", { name: "Prepare isolated recovery" })).not.toBeInTheDocument();
  expect(screen.getByRole("button", { name: "Refresh inspection" })).toBeEnabled();
});

test("stale preview failure clears consent and never claims recovery succeeded", async () => {
  vi.mocked(preparePushRecovery).mockRejectedValue(new Error("Recovery preview is stale. Refresh."));
  renderWithProviders(<PushSafetyDialog onClose={vi.fn()} onPush={vi.fn()} />);
  await screen.findByText("bundle/vault"); fireEvent.click(screen.getByRole("checkbox"));
  fireEvent.click(screen.getByRole("button", { name: "Prepare isolated recovery" }));
  expect(await screen.findByRole("alert")).toHaveTextContent("stale");
  expect(screen.getByRole("checkbox")).not.toBeChecked();
  expect(screen.queryByText("Prepared artifacts checked — not applied")).not.toBeInTheDocument();
});

test("clear inspection still requires a separate push action", async () => {
  vi.mocked(inspectPushSafety).mockResolvedValue(create(PushSafetyReportSchema, { complete: true, state: "clear", head: "head" }));
  const push = vi.fn(); renderWithProviders(<PushSafetyDialog onClose={vi.fn()} onPush={push} />);
  const button = await screen.findByRole("button", { name: "Continue to push" });
  expect(push).not.toHaveBeenCalled(); fireEvent.click(button); expect(push).toHaveBeenCalledTimes(1);
});

test("renders push review through the shared narrow-viewport sheet", async () => {
  vi.mocked(inspectPushSafety).mockResolvedValue(create(PushSafetyReportSchema, { complete: true, state: "clear", reason: "Ready to push." }));
  renderWithProviders(<PushSafetyDialog onClose={vi.fn()} onPush={vi.fn()} />);
  const dialog = await screen.findByTestId("push-safety-dialog");
  expect(dialog).toHaveAttribute("role", "dialog");
  expect(dialog.parentElement).toHaveAttribute("data-presentation", "sheet");
});

test("repository change discards old consent and late responses", async () => {
  let resolveOld: ((value: ReturnType<typeof blocked>) => void) | undefined;
  vi.mocked(inspectPushSafety).mockImplementationOnce(() => new Promise((resolve) => { resolveOld = resolve; }));
  const props = { onClose: vi.fn(), onPush: vi.fn() };
  const view = renderWithProviders(<PushSafetyDialog {...props} repoId="one" />);
  vi.mocked(inspectPushSafety).mockResolvedValue(create(PushSafetyReportSchema, { state: "unknown", reason: "Second repo" }));
  view.rerender(<PushSafetyDialog {...props} repoId="two" />);
  expect(await screen.findByText("Second repo")).toBeVisible();
  resolveOld?.(blocked());
  await waitFor(() => expect(screen.queryByText("bundle/vault")).not.toBeInTheDocument());
});

 test("preparation status can be read after reopening without issuing a writer", async () => {
  vi.mocked(getPushRecovery).mockResolvedValue(create(PushRecoveryArtifactSchema, {state:"preparing", message:"Preparation is in progress or was interrupted."}));
  renderWithProviders(<PushSafetyDialog repoId="one" onClose={vi.fn()} onPush={vi.fn()} />);
  await screen.findByText("bundle/vault"); fireEvent.click(screen.getByRole("button",{name:"Check preparation status"}));
  expect(await screen.findByText("Recovery state: preparing")).toBeVisible();
  expect(preparePushRecovery).not.toHaveBeenCalled();
  expect(screen.queryByText("Prepared artifacts checked — not applied")).not.toBeInTheDocument();
 });

 test("finds an older operation even when inspection is offline", async () => {
  vi.mocked(inspectPushSafety).mockRejectedValue(new Error("Remote offline"));
  vi.mocked(getPushRecovery).mockResolvedValue(create(PushRecoveryArtifactSchema,{state:"stale",fingerprint:"older-operation",message:"Source changed; retained bundles remain available."}));
  renderWithProviders(<PushSafetyDialog repoId="one" onClose={vi.fn()} onPush={vi.fn()} />);
  await screen.findByText("Remote offline");
  fireEvent.click(screen.getByRole("button",{name:"Check preparation status"}));
  expect(await screen.findByText("Recovery state: stale")).toBeVisible();
  expect(getPushRecovery).toHaveBeenCalledWith("", "one");
  expect(screen.getByText(/older-operation/)).toBeVisible();
  expect(preparePushRecovery).not.toHaveBeenCalled();
 });
 test("can attach an exact operation independently of the current preview", async () => {
  vi.mocked(getPushRecovery).mockResolvedValue(create(PushRecoveryArtifactSchema,{state:"damaged",fingerprint:"saved-id",message:"Bundle integrity failed."}));
  renderWithProviders(<PushSafetyDialog repoId="one" onClose={vi.fn()} onPush={vi.fn()} />);
  await screen.findByText("bundle/vault");
  fireEvent.change(screen.getByRole("textbox",{name:/Operation ID/}),{target:{value:"saved-id"}});
  fireEvent.click(screen.getByRole("button",{name:"Check preparation status"}));
  expect(await screen.findByText("Recovery state: damaged")).toBeVisible();
  expect(getPushRecovery).toHaveBeenCalledWith("saved-id","one");
  expect(screen.queryByText("Prepared artifacts checked — not applied")).not.toBeInTheDocument();
 });

 test("closing preparation and reopening after a new commit reattaches without another writer", async () => {
  vi.mocked(preparePushRecovery).mockImplementation(()=>new Promise(()=>{}));
  const view=renderWithProviders(<PushSafetyDialog repoId="one" onClose={vi.fn()} onPush={vi.fn()} />);
  await screen.findByText("bundle/vault");fireEvent.click(screen.getByRole("checkbox"));
  fireEvent.click(screen.getByRole("button",{name:"Prepare isolated recovery"}));
  expect(await screen.findByText(/Closing this screen does not cancel/)).toBeVisible();
  view.unmount();
  vi.mocked(inspectPushSafety).mockResolvedValue(create(PushSafetyReportSchema,{...blocked(),head:"new-head",fingerprint:"preview-2"}));
  vi.mocked(getPushRecovery).mockResolvedValue(create(PushRecoveryArtifactSchema,{state:"stale",fingerprint:"preview-1",message:"Source changed."}));
  renderWithProviders(<PushSafetyDialog repoId="one" onClose={vi.fn()} onPush={vi.fn()} />);
  await screen.findByText("bundle/vault");fireEvent.click(screen.getByRole("button",{name:"Check preparation status"}));
  expect(await screen.findByText("Operation ID: preview-1")).toBeVisible();
  expect(getPushRecovery).toHaveBeenCalledWith("","one");
  expect(preparePushRecovery).toHaveBeenCalledTimes(1);
  expect(screen.getByRole("checkbox")).not.toBeChecked();
 });
 test("a failed status refresh does not retain a verified result card", async()=>{
  vi.mocked(getPushRecovery).mockResolvedValueOnce(create(PushRecoveryArtifactSchema,{state:"prepared",fingerprint:"preview-1"})).mockRejectedValueOnce(new Error("Status unavailable"));
  renderWithProviders(<PushSafetyDialog repoId="one" onClose={vi.fn()} onPush={vi.fn()} />);
  await screen.findByText("bundle/vault");fireEvent.click(screen.getByRole("button",{name:"Check preparation status"}));
  await screen.findByText("Prepared artifacts checked — not applied");
  fireEvent.click(screen.getByRole("button",{name:"Check preparation status"}));
  expect(await screen.findByRole("alert")).toHaveTextContent("Status unavailable");
  expect(screen.queryByText("Prepared artifacts checked — not applied")).not.toBeInTheDocument();
 });
