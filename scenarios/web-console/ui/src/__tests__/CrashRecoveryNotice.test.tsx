import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import CrashRecoveryNotice from "../components/CrashRecoveryNotice";

const listMock = vi.hoisted(() => vi.fn());
vi.mock("../api/sessions", () => ({ listRecoverableSessions: listMock }));

describe("CrashRecoveryNotice", () => {
  beforeEach(() => {
    listMock.mockReset();
  });

  it("renders nothing when no crash orphans exist", async () => {
    listMock.mockResolvedValue([]);
    const { container } = render(<CrashRecoveryNotice onOpenArchive={vi.fn()} />);
    await waitFor(() => expect(listMock).toHaveBeenCalled());
    expect(container.querySelector("[data-testid='crash-recovery-notice']")).toBeNull();
  });

  it("reports the count compactly and opens the archive", async () => {
    listMock.mockResolvedValue([
      { id: "one", agent_type: "codex", recoverable: true },
      { id: "two", agent_type: "grok", recoverable: false },
    ]);
    const onOpenArchive = vi.fn();
    render(<CrashRecoveryNotice onOpenArchive={onOpenArchive} topSafe />);
    const notice = await screen.findByTestId("crash-recovery-notice");
    expect(notice).toHaveTextContent("recoverableSessions.heading");
    expect(notice.className).toContain("--wc-safe-top");
    fireEvent.click(screen.getByRole("button", { name: "recoverableSessions.viewArchive" }));
    expect(onOpenArchive).toHaveBeenCalledOnce();
  });
});
