import { HistoryModeHeader } from "./HistoryModeHeader";
import appSource from "../App.tsx?raw";
import ts from "typescript";
import { HistoryFileList } from "./HistoryFileList";
import { renderWithProviders as render } from "../test-utils/renderWithProviders";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { create } from "@bufbuild/protobuf";
import { PushSafetyReportSchema } from "@vrooli/proto-types/git-control-tower/v1/repo/repo_pb";
import { afterEach, expect, test, vi } from "vitest";
import { historySafetyLabel, PushSafetyProvider, PushSafetyNotice, StagedSafetyNotice, StagedSafetyBadge, historyPathBlockers } from "./PushSafetyIndicators";
import { CommitPanel } from "./CommitPanel";
import { inspectPushSafety } from "../lib/api-push-safety";
vi.mock("../lib/api-push-safety", () => ({ inspectPushSafety: vi.fn() }));
afterEach(() => { vi.clearAllMocks(); vi.restoreAllMocks(); vi.useRealTimers(); });
const hashes = ["a", "b", "c", "d"].map(x => x.repeat(40));
const report = () => create(PushSafetyReportSchema, { complete:true, state:"blocked", canPrepare:true, commits:hashes, limit:104857600n, stagedComplete:true, files:[{oid:"blob",paths:["generated.bin"],bytes:440820652n,blocked:true,commits:[(hashes[1] ?? ""),(hashes[2] ?? "")]}], stagedFiles:[{oid:"index",paths:["staged.bin"],bytes:440820652n,blocked:true}] });
test("every application layout offering recovery review mounts its dialog", () => {
 // Guard the actual App wiring: component click tests alone missed the desktop
 // branch setting open state without rendering any dialog.
 const source = ts.createSourceFile("App.tsx", appSource, ts.ScriptTarget.Latest, true, ts.ScriptKind.TSX);
 const layouts: ts.JsxElement[] = [];
 const visit = (node: ts.Node) => {
  if (ts.isJsxElement(node) && node.openingElement.tagName.getText(source) === "PushSafetyProvider") layouts.push(node);
  ts.forEachChild(node, visit);
 };
 visit(source);
 expect(layouts.length).toBeGreaterThan(0);
 for (const layout of layouts) {
  let dialogs = 0;
  const findDialog = (node: ts.Node) => {
   if (ts.isJsxSelfClosingElement(node) && node.tagName.getText(source) === "PushSafetyDialog") dialogs++;
   ts.forEachChild(node, findDialog);
  };
  findDialog(layout);
  expect(dialogs, `Recovery dialog missing from App layout at line ${source.getLineAndCharacterOfPosition(layout.pos).line + 1}`).toBe(1);
 }
});
test("every application layout mounts commit authorization", () => {
 const source = ts.createSourceFile("App.tsx", appSource, ts.ScriptTarget.Latest, true, ts.ScriptKind.TSX);
 const layouts: ts.JsxElement[] = [];
 const visit = (node: ts.Node) => {
  if (ts.isJsxElement(node) && node.openingElement.tagName.getText(source) === "PushSafetyProvider") layouts.push(node);
  ts.forEachChild(node, visit);
 };
 visit(source);
 expect(layouts.length).toBeGreaterThan(0);
 for (const layout of layouts) {
  let dialogs = 0;
  const findDialog = (node: ts.Node) => {
   if ((ts.isJsxElement(node) && node.openingElement.tagName.getText(source) === "CommitAuthorizationDialog") || (ts.isJsxSelfClosingElement(node) && node.tagName.getText(source) === "CommitAuthorizationDialog")) dialogs++;
   ts.forEachChild(node, findDialog);
  };
  findDialog(layout);
  expect(dialogs, `Commit authorization missing from App layout at line ${source.getLineAndCharacterOfPosition(layout.pos).line + 1}`).toBe(1);
 }
});
test("history marks the introducing commit without repeating inherited or successful checks", () => {
 const r = report();
 expect(historySafetyLabel(r, (hashes[0] ?? ""),true)).toBeUndefined();
 expect(historySafetyLabel(r, (hashes[1] ?? "").slice(0,7),true)).toBe("Introduces push blocker");
 expect(historySafetyLabel(r, (hashes[2] ?? ""),true)).toBeUndefined();
 expect(historySafetyLabel(r, (hashes[3] ?? ""),true)).toBeUndefined();
 expect(historySafetyLabel(undefined, (hashes[1] ?? ""),true)).toBeUndefined();
 expect(historySafetyLabel(r, "e".repeat(40),false)).toBeUndefined();
 r.canPrepare=false;
 expect(historySafetyLabel(r,(hashes[3] ?? ""),true)).toBeUndefined();
});
function view(revision="one",repoId="repo-a",client=new QueryClient({defaultOptions:{queries:{retry:false}}})) {
 return {client, tree:<QueryClientProvider client={client}><PushSafetyProvider repoId={repoId} revision={revision} review={() => {}}><PushSafetyNotice/><StagedSafetyNotice/></PushSafetyProvider></QueryClientProvider>};
}
test("one shared report exposes concise staged and outgoing warnings without mutating",async()=>{
 vi.mocked(inspectPushSafety).mockResolvedValue(report());
 render(view().tree);
 await screen.findByText(/1 large staged file would block push/);
 expect(screen.getByText(/Push blocked: 1 oversized file ·/)).toBeTruthy();
 expect(screen.queryByText(/Staged file sizes checked/)).toBeNull();
 expect(inspectPushSafety).toHaveBeenCalledTimes(1);
});
test("repository or observed workspace change immediately removes prior safety claims",async()=>{
 vi.mocked(inspectPushSafety).mockResolvedValue(report());
 const initial=view();const mounted=render(initial.tree);
 await screen.findByText(/1 large staged file would block push/);
 mounted.rerender(view("two","repo-a",initial.client).tree);
 expect(screen.queryByText(/1 large staged file would block push/)).toBeNull();
 expect(screen.queryByText(/Push blocked:/)).toBeNull();
 vi.mocked(inspectPushSafety).mockRejectedValue(new Error("offline"));
 mounted.rerender(view("two","repo-b",initial.client).tree);
 await waitFor(()=>expect(screen.getByText(/Couldn’t verify push file sizes ·/)).toBeTruthy());
 expect(screen.queryByText(/1 large staged file would block push/)).toBeNull();
});
test("expired reports are not displayed as current",async()=>{
 vi.mocked(inspectPushSafety).mockResolvedValue(report());
 render(view().tree);
 await screen.findByText(/1 large staged file would block push/);
 vi.spyOn(Date, "now").mockReturnValue(Date.now()+61000);
 await waitFor(()=>expect(screen.queryByText(/1 large staged file would block push/)).toBeNull(),{timeout:6000});
}, 8000);

test("blocked staging stays committable and its badge opens review without a mutation", async () => {
 vi.mocked(inspectPushSafety).mockResolvedValue(report());
 const review=vi.fn(); const commit=vi.fn();
 const client=new QueryClient({defaultOptions:{queries:{retry:false}}});
 render(<QueryClientProvider client={client}><PushSafetyProvider repoId="repo" revision="one" review={review}>
  <StagedSafetyBadge path="staged.bin" />
  <CommitPanel stagedCount={1} commitMessage="Keep local checkpoint" onCommitMessageChange={()=>{}} onCommit={commit} isCommitting={false}/>
 </PushSafetyProvider></QueryClientProvider>);
 const badge=await screen.findByRole("button",{name:"Push blocker"});
 fireEvent.click(badge);
 expect(review).toHaveBeenCalledOnce(); expect(commit).not.toHaveBeenCalled();
 expect(screen.getByTestId("commit-button")).toBeEnabled();
});

test.each([false, true])("history navbar opens recovery without exiting (compact=%s)", async compact => {
 vi.mocked(inspectPushSafety).mockResolvedValue(report());
 const review=vi.fn(); const exit=vi.fn(); const select=vi.fn();
 render(<PushSafetyProvider repoId="repo" revision="one" review={review}>
  <HistoryModeHeader compact={compact} commit={{hash:hashes[1] ?? "",subject:"binary",files:["generated.bin"]}} onExit={exit}/>
  <HistoryFileList viewingCommit={{hash:hashes[1] ?? "",subject:"binary",files:["generated.bin","safe.ts"]}} onSelectFile={select}/>
 </PushSafetyProvider>);
 fireEvent.click(await screen.findByRole("button",{name:"Review push recovery options"}));
 expect(review).toHaveBeenCalledOnce(); expect(exit).not.toHaveBeenCalled();
 const badge=await screen.findByRole("button",{name:/Push blocker in outgoing history · 420/});
 fireEvent.click(badge);
 expect(review).toHaveBeenCalledTimes(2); expect(select).not.toHaveBeenCalled();
 expect(screen.getAllByText(/Push blocker in outgoing history/)).toHaveLength(1);
});
test("history file attribution handles deletion, unknown snapshots, and published commits",()=>{
 const r=report();
 expect(historyPathBlockers(r,hashes[3] ?? "","generated.bin")).toHaveLength(1);
 expect(historyPathBlockers(r,hashes[1] ?? "","safe.ts")).toHaveLength(0);
 expect(historyPathBlockers(undefined,hashes[1] ?? "","generated.bin")).toHaveLength(0);
 expect(historyPathBlockers(r,"e".repeat(40),"generated.bin")).toHaveLength(0);
});
test("history file lists omit duplicated summaries about other commits",async()=>{
 vi.mocked(inspectPushSafety).mockResolvedValue(report());
 render(<PushSafetyProvider repoId="repo" revision="one" review={()=>{}}>
  <HistoryFileList viewingCommit={{hash:hashes[3] ?? "",subject:"later work",files:["safe.ts"]}} onSelectFile={()=>{}}/>
 </PushSafetyProvider>);
 await waitFor(() => expect(inspectPushSafety).toHaveBeenCalled());
 expect(screen.queryByText(/Other paths blocking this outgoing push/)).toBeNull();
 expect(screen.queryByRole("button",{name:"generated.bin"})).toBeNull();
 expect(screen.queryByText(/Push blocker in outgoing history ·/)).toBeNull();
});


test.each([false, true])("successful checks leave staging and history quiet (compact=%s)", async compact => {
 const clean = create(PushSafetyReportSchema, {complete:true, state:"clear", stagedComplete:true, limit:104857600n, commits:hashes});
 vi.mocked(inspectPushSafety).mockResolvedValue(clean);
 const client = new QueryClient();
 render(<QueryClientProvider client={client}><PushSafetyProvider repoId="clean" revision="one" review={()=>{}}>
  <PushSafetyNotice/><StagedSafetyNotice/>
  <HistoryModeHeader compact={compact} commit={{hash:hashes[0] ?? "",subject:"source",files:["safe.ts"]}} onExit={()=>{}}/>
  <HistoryFileList viewingCommit={{hash:hashes[0] ?? "",subject:"source",files:["safe.ts"]}} onSelectFile={()=>{}}/>
 </PushSafetyProvider></QueryClientProvider>);
 await waitFor(()=>expect(client.getQueryState(["push-safety-indicators", "clean", "one"])?.status).toBe("success"));
 expect(screen.queryByText(/check passed|sizes checked|Review recovery|unverified|Checking push safety/i)).toBeNull();
 expect(screen.queryByRole("button", {name:"Review push recovery options"})).toBeNull();
});

test("warning-sized outgoing files remain actionable without a hard blocker", async () => {
 const warning = report(); warning.state="clear"; warning.files=warning.files.map(file => ({...file, blocked:false})); warning.stagedFiles=[];
 vi.mocked(inspectPushSafety).mockResolvedValue(warning);
 render(view().tree);
 expect(await screen.findByRole("button",{name:"1 large file in outgoing commits · Review"})).toBeTruthy();
 expect(screen.queryByText(/large staged file/)).toBeNull();
});


test("background safety inspection waits for pending writes to finish", async () => {
 vi.mocked(inspectPushSafety).mockResolvedValue(report());
 const mounted=render(<PushSafetyProvider repoId="busy" revision="one" paused review={()=>{}}><PushSafetyNotice/></PushSafetyProvider>);
 expect(inspectPushSafety).not.toHaveBeenCalled();
 expect(screen.queryByRole("button",{name:/Review/})).toBeNull();
 mounted.rerender(<PushSafetyProvider repoId="busy" revision="one" paused={false} review={()=>{}}><PushSafetyNotice/></PushSafetyProvider>);
 await screen.findByRole("button",{name:/Push blocked/});
 expect(inspectPushSafety).toHaveBeenCalledTimes(1);
});
