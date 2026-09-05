import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { strings } from "../../consts/strings";

// Capture the most-recently constructed provider so tests can poke
// onResult / onError directly.
const constructed: Array<{
  start: ReturnType<typeof vi.fn>;
  stop: ReturnType<typeof vi.fn>;
  dispose: ReturnType<typeof vi.fn>;
  onResult: ((s: string) => void) | null;
  onError: ((s: string) => void) | null;
  onPartial: ((s: string) => void) | null;
}> = [];

vi.mock("../../audio-integration", () => ({
  PcmVoiceStreamProvider: class {
    onResult: ((s: string) => void) | null = null;
    onError: ((s: string) => void) | null = null;
    onPartial: ((s: string) => void) | null = null;
    start = vi.fn().mockResolvedValue(undefined);
    stop = vi.fn();
    dispose = vi.fn();
    constructor() {
      constructed.push(this);
    }
  },
  MicReadinessIndicator: ({ state }: { state: string }) => (
    <span data-testid="mic-readiness" data-state={state} />
  ),
}));

vi.mock("./useMicPermission", () => ({ useMicPermission: () => "ready" }));

import { LiveTry } from "./LiveTry";

beforeEach(() => {
  constructed.length = 0;
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("LiveTry", () => {
  it("renders the Start button by default", () => {
    renderWithProviders(<LiveTry onTrace={() => {}} />);
    expect(
      screen.getByRole("button", { name: new RegExp(strings.diagnostics.liveStart, "i") }),
    ).toBeInTheDocument();
  });

  it("starting a live run constructs the provider and flips the button to Stop", async () => {
    const user = userEvent.setup();
    renderWithProviders(<LiveTry onTrace={() => {}} />);
    await user.click(
      screen.getByRole("button", { name: new RegExp(strings.diagnostics.liveStart, "i") }),
    );
    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: new RegExp(strings.diagnostics.liveStop, "i") }),
      ).toBeInTheDocument();
    });
    expect(constructed[0]?.start).toHaveBeenCalledTimes(1);
  });

  it("on result: renders the final transcript and emits a trace", async () => {
    const onTrace = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(<LiveTry onTrace={onTrace} />);
    await user.click(
      screen.getByRole("button", { name: new RegExp(strings.diagnostics.liveStart, "i") }),
    );
    await waitFor(() => expect(constructed[0]).toBeTruthy());
    constructed[0]?.onResult?.("hello world");
    expect(await screen.findByText(/^hello world$/)).toBeInTheDocument();
    expect(onTrace).toHaveBeenCalledWith(expect.objectContaining({ providerId: "voice-stream" }));
  });

  it("on error: surfaces the error message in the UI", async () => {
    const user = userEvent.setup();
    renderWithProviders(<LiveTry onTrace={() => {}} />);
    await user.click(
      screen.getByRole("button", { name: new RegExp(strings.diagnostics.liveStart, "i") }),
    );
    await waitFor(() => expect(constructed[0]).toBeTruthy());
    constructed[0]?.onError?.("mic-failed");
    expect(await screen.findByText(/^mic-failed$/)).toBeInTheDocument();
  });
});
