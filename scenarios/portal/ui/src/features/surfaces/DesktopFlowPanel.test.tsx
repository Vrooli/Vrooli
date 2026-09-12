// provider-free-exception: The test uses a provider-free or feature-specific harness to isolate its boundary.
import { create } from "@bufbuild/protobuf";
import { act, cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { SessionRefSchema } from "@vrooli/proto-types/common/v1/surface_pb";
import { FlowRecordSchema, SavedDesktopFlowSchema } from "@vrooli/proto-types/device-control/v1/desktop/desktop_pb";
import { desktopClient } from "../../api/desktop";
import { strings } from "../../consts/strings";
import { DesktopFlowPanel } from "./DesktopFlowPanel";
vi.mock("../../api/desktop", () => ({ desktopClient: { runFlow: vi.fn(), runSavedFlow: vi.fn(), promoteFlow: vi.fn(), getSavedFlow: vi.fn() }, operatorHeaders: (token: string) => ({ Authorization: `Bearer ${token}` }) }));
const session = create(SessionRefSchema, { sessionId: "lease", desktopSessionId: "2", surface: { ownerScenario: "device-control", surfaceId: "desktop", target: { ownerScenario: "vrooli-bridge", resourceId: "host", hostNodeId: "host" } } });
const props = () => ({ token: "account-token", session, control: true, applicationId: "app", applicationRevision: "catalog", applicationFresh: true, windowName: "Editor", fieldName: "Entry", busy: false, blocked: false, onBusy: vi.fn(), onUncertain: vi.fn() });
beforeEach(() => { vi.clearAllMocks(); });
afterEach(cleanup);
function assertionDraft() {
 fireEvent.click(screen.getByText(strings.desktopFlows.addAssert));
 fireEvent.change(screen.getByLabelText(strings.desktopFlows.expected), { target: { value: "日本語" } });
 fireEvent.change(screen.getByLabelText(strings.desktopFlows.context), { target: { value: "editor:v1" } });
}
it("saves only the completed asserted draft with its source session", async () => {
 vi.mocked(desktopClient.runFlow).mockImplementation(request => Promise.resolve(create(FlowRecordSchema, { runId: request.runId, disposition: "passed", confirmed: 1, steps: 1 })));
 vi.mocked(desktopClient.promoteFlow).mockResolvedValue(create(SavedDesktopFlowSchema, { id: "saved", version: 1, contextKey: "editor:v1" }));
 const value = props(); render(<DesktopFlowPanel {...value} />);
 assertionDraft();
 expect(screen.getByText(strings.desktopFlows.save)).toBeDisabled();
 fireEvent.click(screen.getByText(strings.desktopFlows.run));
 await waitFor(() => expect(screen.getByText(strings.desktopFlows.save)).toBeEnabled());
 const request = vi.mocked(desktopClient.runFlow).mock.calls[0]![0];
 expect(request.flow?.steps?.[0]).toMatchObject({ kind: "desktop-text-assert", target: "Entry", arguments: { window: "Editor", text: "日本語" } });
 expect(request.session).toEqual(session);
 fireEvent.click(screen.getByText(strings.desktopFlows.save));
 await waitFor(() => expect(desktopClient.promoteFlow).toHaveBeenCalledWith(expect.objectContaining({ sourceSession: session, sourceRunId: request.runId, contextKey: "editor:v1" }), expect.objectContaining({ headers: { Authorization: "Bearer account-token" } })));
 fireEvent.change(screen.getByLabelText(strings.desktopFlows.expected), { target: { value: "changed" } });
 expect(screen.getByText(strings.desktopFlows.save)).toBeDisabled();
});
it("checks an uncertain run with the identical request instead of creating new input", async () => {
 vi.mocked(desktopClient.runFlow).mockRejectedValueOnce(new Error("lost response")).mockImplementation(request => Promise.resolve(create(FlowRecordSchema, { runId: request.runId, disposition: "incomplete" })));
 const value = props(); render(<DesktopFlowPanel {...value} />); assertionDraft();
 fireEvent.click(screen.getByText(strings.desktopFlows.run));
 await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent(strings.desktopFlows.failed));
 expect(value.onUncertain).toHaveBeenCalledWith(true);
 expect(screen.getByText(strings.desktopFlows.run)).toBeDisabled();
 fireEvent.click(screen.getByText(strings.desktopFlows.check));
 await waitFor(() => expect(desktopClient.runFlow).toHaveBeenCalledTimes(2));
 expect(vi.mocked(desktopClient.runFlow).mock.calls[1]![0]).toEqual(vi.mocked(desktopClient.runFlow).mock.calls[0]![0]);
 expect(screen.getByText(strings.desktopFlows.save)).toBeDisabled();
});
it("aborts pending execution when Stop removes the session and ignores late success", async () => {
 let finish!: (value: ReturnType<typeof create<typeof FlowRecordSchema>>) => void;
 vi.mocked(desktopClient.runFlow).mockImplementation(() => new Promise(resolve => { finish = resolve; }));
 const value = props(); const view = render(<DesktopFlowPanel {...value} />); assertionDraft();
 fireEvent.click(screen.getByText(strings.desktopFlows.run));
 await waitFor(() => expect(desktopClient.runFlow).toHaveBeenCalled());
 const options = vi.mocked(desktopClient.runFlow).mock.calls[0]![1];
 view.rerender(<DesktopFlowPanel {...value} session={undefined} control={false} />);
 expect(options?.signal?.aborted).toBe(true);
 await act(async () => { finish(create(FlowRecordSchema, { disposition: "passed", confirmed: 1 })); await Promise.resolve(); });
 expect(screen.getByText(strings.desktopFlows.save)).toBeDisabled();
 expect(desktopClient.promoteFlow).not.toHaveBeenCalled();
});

it("loads and replays the exact saved revision with current application references", async () => { // FLOW-01
 vi.mocked(desktopClient.getSavedFlow).mockResolvedValue(create(SavedDesktopFlowSchema, { id: "shared", version: 2, contextKey: "editor:v1" }));
 vi.mocked(desktopClient.runSavedFlow).mockImplementation(request => Promise.resolve(create(FlowRecordSchema, { runId: request.runId, disposition: "passed", confirmed: 1 })));
 const value = props(); render(<DesktopFlowPanel {...value} />);
 fireEvent.change(screen.getByLabelText(strings.desktopFlows.savedId), { target: { value: "shared" } });
 fireEvent.change(screen.getByLabelText(strings.desktopFlows.version), { target: { value: "2" } });
 fireEvent.change(screen.getByLabelText(strings.desktopFlows.context), { target: { value: "editor:v1" } });
 fireEvent.click(screen.getByText(strings.desktopFlows.load));
 await waitFor(() => expect(screen.getByText(strings.desktopFlows.replay)).toBeEnabled());
 fireEvent.click(screen.getByText(strings.desktopFlows.replay));
 await waitFor(() => expect(desktopClient.runSavedFlow).toHaveBeenCalledWith(expect.objectContaining({ id: "shared", version: 2, contextKey: "editor:v1", session, applicationId: "app", applicationRevision: "catalog" }), expect.objectContaining({ timeoutMs: 35000 })));
 expect(desktopClient.runFlow).not.toHaveBeenCalled();
});
