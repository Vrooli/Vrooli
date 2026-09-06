import { act, cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ApprovalPrompt } from "../../../library/components/ApprovalPrompt/versions/1.2.0/ApprovalPrompt";
const request = { action: "publish", target: "production", scope: "one release" };
afterEach(() => { cleanup(); vi.restoreAllMocks(); });
describe("approval decisions", () => {
  it("preserves readable consent text in compact mode", () => {
    render(<ApprovalPrompt {...request} density="compact" />);
    expect(document.querySelector("[data-rcl-approval-consent-line]")?.textContent).toBe("Approve publish for production within one release.");
  });
  it("requires an actual decision handler", () => {
    render(<ApprovalPrompt {...request} />);
    expect(screen.getByRole("button", { name: "Approve publish" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Not now" })).toBeDisabled();
  });
  it("records a declined decision once and closes decision controls", async () => {
    let resolve!: () => void;
    const onDeny = vi.fn(() => new Promise<void>(done => { resolve = done; }));
    render(<ApprovalPrompt {...request} onDeny={onDeny} />);
    const deny = screen.getByRole("button", { name: "Not now" });
    fireEvent.click(deny); fireEvent.click(deny);
    expect(onDeny).toHaveBeenCalledTimes(1);
    await act(async () => resolve());
    expect(screen.getByText("Request declined")).toBeVisible();
    expect(screen.queryByRole("button")).toBeNull();
  });
  it("retries the failed decline without granting permission", async () => {
    const onApprove = vi.fn();
    const onDeny = vi.fn().mockRejectedValueOnce(new Error("offline")).mockResolvedValueOnce(undefined);
    render(<ApprovalPrompt {...request} onApprove={onApprove} onDeny={onDeny} />);
    fireEvent.click(screen.getByRole("button", { name: "Not now" }));
    fireEvent.click(await screen.findByRole("button", { name: "Retry approval" }));
    await screen.findByText("Request declined");
    expect(onDeny).toHaveBeenCalledTimes(2);
    expect(onApprove).not.toHaveBeenCalled();
  });
  it("does not turn an access request into approval", async () => {
    const onRetry = vi.fn().mockResolvedValue(undefined);
    render(<ApprovalPrompt {...request} defaultStatus="permission-denied" onRetry={onRetry} />);
    fireEvent.click(screen.getByRole("button", { name: "Request access" }));
    await waitFor(() => expect(screen.getByRole("button", { name: "Request access" })).toBeEnabled());
    expect(onRetry).toHaveBeenCalledTimes(1);
    expect(screen.queryByText("Approval recorded")).toBeNull();
    expect(screen.queryByRole("button", { name: "Not now" })).toBeNull();
  });
  it("rejects an expired request on first render", () => {
    render(<ApprovalPrompt {...request} expiresAt={Date.now() - 1} onApprove={vi.fn()} />);
    expect(screen.getByText("Request expired")).toBeVisible();
    expect(screen.queryByRole("button")).toBeNull();
  });
  it("checks the deadline at activation between refresh ticks", () => {
    const clock = vi.spyOn(Date, "now").mockReturnValue(1000);
    const onApprove = vi.fn();
    render(<ApprovalPrompt {...request} expiresAt={2000} onApprove={onApprove} />);
    clock.mockReturnValue(2001);
    fireEvent.click(screen.getByRole("button", { name: "Approve publish" }));
    expect(onApprove).not.toHaveBeenCalled();
    expect(screen.getByText("Request expired")).toBeVisible();
  });
  it("keeps recorded decisions after the request deadline", () => {
    render(<ApprovalPrompt {...request} status="success" expiresAt={Date.now() - 1} />);
    expect(screen.getByText("Approval recorded")).toBeVisible();
    expect(screen.queryByText("Request expired")).toBeNull();
  });
});
