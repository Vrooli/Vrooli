import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";

const getVoiceStreamConfig = vi.fn();
const updateVoiceStreamConfig = vi.fn();
vi.mock("../../audio-integration", () => ({
  getVoiceStreamConfig: () => getVoiceStreamConfig(),
  updateVoiceStreamConfig: (patch: { overlapMaxStallRejects?: number }) => updateVoiceStreamConfig(patch),
}));

const pushToast = vi.fn();
vi.mock("../../components/ui/toast", () => ({
  pushToast: (...args: unknown[]) => pushToast(...args),
}));

import { OverlapStallGuard } from "./OverlapStallGuard";

beforeEach(() => {
  vi.clearAllMocks();
  getVoiceStreamConfig.mockResolvedValue({ overlapMaxStallRejects: 3 });
  updateVoiceStreamConfig.mockResolvedValue({ overlapMaxStallRejects: 3 });
});
afterEach(cleanup);

describe("OverlapStallGuard", () => {
  it("loads the persisted value into the input", async () => {
    renderWithProviders(<OverlapStallGuard />);
    await waitFor(() =>
      expect(screen.getByTestId(selectors.streamConfig.stallRejectsInput)).toHaveValue(3),
    );
  });

  it("saves the current value through the update-mask path", async () => {
    const user = userEvent.setup();
    renderWithProviders(<OverlapStallGuard />);
    await waitFor(() =>
      expect(screen.getByTestId(selectors.streamConfig.stallRejectsInput)).toHaveValue(3),
    );
    await user.click(screen.getByTestId(selectors.streamConfig.saveOverlap));
    await waitFor(() =>
      expect(updateVoiceStreamConfig).toHaveBeenCalledWith({ overlapMaxStallRejects: 3 }),
    );
    await waitFor(() => expect(pushToast).toHaveBeenCalled());
  });

  it("warns when the guard is disabled (0) and saves 0", async () => {
    const user = userEvent.setup();
    renderWithProviders(<OverlapStallGuard />);
    const input = await screen.findByTestId(selectors.streamConfig.stallRejectsInput);
    await user.clear(input);
    await user.type(input, "0");
    expect(screen.getByText(strings.streamConfigAdmin.stallRejectsDisabled)).toBeInTheDocument();
    await user.click(screen.getByTestId(selectors.streamConfig.saveOverlap));
    await waitFor(() =>
      expect(updateVoiceStreamConfig).toHaveBeenCalledWith({ overlapMaxStallRejects: 0 }),
    );
  });
});
