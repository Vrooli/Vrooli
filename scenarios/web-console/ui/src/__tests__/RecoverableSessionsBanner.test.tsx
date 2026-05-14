import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "@testing-library/react";

import RecoverableSessionsBanner from "../components/RecoverableSessionsBanner";
import type { RecoverableSession, RecoverResult } from "../api/sessions";

const { listMock, recoverMock, dismissMock } = vi.hoisted(() => ({
  listMock: vi.fn<() => Promise<RecoverableSession[]>>(),
  recoverMock: vi.fn<(id: string) => Promise<RecoverResult>>(),
  dismissMock: vi.fn<(id: string) => Promise<void>>(),
}));

vi.mock("../api/sessions", async () => {
  const actual = await vi.importActual<typeof import("../api/sessions")>("../api/sessions");
  return {
    ...actual,
    listRecoverableSessions: listMock,
    recoverSession: recoverMock,
    dismissRecoverableSession: dismissMock,
  };
});

beforeEach(() => {
  listMock.mockReset();
  recoverMock.mockReset();
  dismissMock.mockReset();
});

describe("RecoverableSessionsBanner", () => {
  it("renders nothing when the list is empty", async () => {
    listMock.mockResolvedValue([]);
    const { container } = render(<RecoverableSessionsBanner />);
    // Wait for the effect to settle.
    await waitFor(() => expect(listMock).toHaveBeenCalled());
    expect(container.querySelector("[data-testid='recoverable-sessions-banner']")).toBeNull();
  });

  it("shows a row per orphan and disables Reattach when not recoverable", async () => {
    listMock.mockResolvedValue([
      {
        id: "aaaaaaaa-1111-2222-3333-444444444444",
        agent_type: "codex",
        agent_session_id: "codex-uuid-1",
        recoverable: true,
      },
      {
        id: "bbbbbbbb-1111-2222-3333-444444444444",
        agent_type: "claude",
        recoverable: false,
        not_recoverable_reason: "claude session id is required",
      },
    ]);
    render(<RecoverableSessionsBanner />);
    await waitFor(() => expect(screen.getByTestId("recoverable-sessions-banner")).toBeInTheDocument());

    const reattachA = screen.getByTestId("recoverable-row-aaaaaaaa-1111-2222-3333-444444444444-recover");
    expect(reattachA).not.toBeDisabled();
    const reattachB = screen.getByTestId("recoverable-row-bbbbbbbb-1111-2222-3333-444444444444-recover");
    expect(reattachB).toBeDisabled();
  });

  it("invokes recoverSession on Reattach click and notifies via onRecovered", async () => {
    listMock
      .mockResolvedValueOnce([
        {
          id: "id1",
          agent_type: "codex",
          recoverable: true,
        },
      ])
      .mockResolvedValueOnce([]); // refresh after recover

    recoverMock.mockResolvedValue({
      old_session_id: "id1",
      new_session_id: "newid",
      agent_type: "codex",
      command_sent: "codex --yolo resume x\n",
    });

    const onRecovered = vi.fn();
    render(<RecoverableSessionsBanner onRecovered={onRecovered} />);
    await waitFor(() => expect(screen.getByTestId("recoverable-sessions-banner")).toBeInTheDocument());

    fireEvent.click(screen.getByTestId("recoverable-row-id1-recover"));

    await waitFor(() => expect(recoverMock).toHaveBeenCalledWith("id1"));
    await waitFor(() =>
      expect(onRecovered).toHaveBeenCalledWith(
        expect.objectContaining({ old_session_id: "id1", new_session_id: "newid" }),
      ),
    );
  });

  it("invokes dismissRecoverableSession on Dismiss click", async () => {
    listMock
      .mockResolvedValueOnce([
        { id: "id2", agent_type: "codex", recoverable: true },
      ])
      .mockResolvedValueOnce([]);
    dismissMock.mockResolvedValue(undefined);

    render(<RecoverableSessionsBanner />);
    await waitFor(() => expect(screen.getByTestId("recoverable-sessions-banner")).toBeInTheDocument());

    fireEvent.click(screen.getByTestId("recoverable-row-id2-dismiss"));
    await waitFor(() => expect(dismissMock).toHaveBeenCalledWith("id2"));
  });
});
