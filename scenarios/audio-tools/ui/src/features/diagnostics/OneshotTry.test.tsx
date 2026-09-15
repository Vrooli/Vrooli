import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { renderWithProviders } from "../../test-utils";
import { strings } from "../../consts/strings";

// MicReadinessIndicator pulls in audio-integration with MediaRecorder side
// effects; stub it for unit isolation.
vi.mock("../../audio-integration", () => ({
  MicReadinessIndicator: ({ state }: { state: string }) => (
    <span data-testid="mic-readiness" data-state={state} />
  ),
}));

vi.mock("./useMicPermission", () => ({
  useMicPermission: () => "ready",
}));

vi.mock("../../services/diagnostics", () => ({
  transcribe: vi.fn(),
}));

import { OneshotTry } from "./OneshotTry";

beforeEach(() => {
  // jsdom doesn't implement MediaRecorder or navigator.mediaDevices.
  // OneshotTry only touches them on user-initiated start(), so leaving
  // them absent is fine for the tests we care about.
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("OneshotTry", () => {
  it("renders the Record button and the mic readiness indicator", () => {
    renderWithProviders(<OneshotTry onTrace={() => {}} />);
    expect(
      screen.getByRole("button", { name: new RegExp(strings.diagnostics.oneshotRecord, "i") }),
    ).toBeInTheDocument();
    expect(screen.getByTestId("mic-readiness")).toBeInTheDocument();
  });

  it("renders the hint copy key", () => {
    renderWithProviders(<OneshotTry onTrace={() => {}} />);
    expect(screen.getByText(strings.diagnostics.oneshotHint)).toBeInTheDocument();
  });

  it("Record button is not in the recording-pressed state initially", () => {
    renderWithProviders(<OneshotTry onTrace={() => {}} />);
    const btn = screen.getByRole("button", { name: new RegExp(strings.diagnostics.oneshotRecord, "i") });
    expect(btn).toHaveAttribute("aria-pressed", "false");
  });
});
