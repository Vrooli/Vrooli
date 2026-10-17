import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";

vi.mock("../../api/providerLifecycle", () => ({
  streamProviderLogs: vi.fn(),
}));

import { LogsDrawer } from "./LogsDrawer";
import { streamProviderLogs } from "../../api/providerLifecycle";

beforeEach(() => {
  vi.mocked(streamProviderLogs).mockReturnValue((async function* () {
    await Promise.resolve();
    yield { line: "line-1", tsUnixMs: BigInt(0), stream: 1 };
    yield { line: "line-2", tsUnixMs: BigInt(0), stream: 1 };
  })() as never);
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("LogsDrawer", () => {
  it("returns null when closed (no dialog in the DOM)", () => {
    renderWithProviders(
      <LogsDrawer open={false} providerId="whisper" onClose={() => {}} />,
    );
    expect(screen.queryByRole("dialog")).toBeNull();
  });

  it("renders streamed log lines after opening", async () => {
    renderWithProviders(
      <LogsDrawer open providerId="whisper" onClose={() => {}} />,
    );
    await waitFor(() => {
      const stream = screen.getByTestId("logs-stream");
      expect(stream.textContent).toMatch(/line-1/);
      expect(stream.textContent).toMatch(/line-2/);
    });
  });

  it("Close invokes onClose and aborts the stream", async () => {
    const onClose = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(
      <LogsDrawer open providerId="whisper" onClose={onClose} />,
    );
    await user.click(screen.getByRole("button", { name: /Close/i }));
    expect(onClose).toHaveBeenCalled();
  });
});
