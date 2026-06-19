/**
 * RecoveryPanel tests — state machine snapshot, event timeline, and the
 * confirm-guarded manual recover action (with force toggle).
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { makeRecoveryMocks, makeRecoveryState, makeRecoveryEvent } from "../../test-utils/mocks/recovery";

vi.mock("../../api/recovery", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/recovery")>();
  return { ...actual, ...makeRecoveryMocks() };
});

import { RecoveryPanel } from "./RecoveryPanel";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";

describe("RecoveryPanel", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the state snapshot with status and circuit badges", async () => {
    renderWithProviders(<RecoveryPanel />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.recovery.statusValue)).toHaveTextContent("Monitoring");
    });
    expect(screen.getByTestId(selectors.recovery.circuitValue)).toHaveTextContent("Closed");
    expect(screen.getByTestId(selectors.recovery.summary)).toHaveTextContent("No restart is queued");
    expect(screen.getByTestId(selectors.recovery.policyNote)).toHaveTextContent("Automatic recovery follows server policy");
  });

  it("shows the open circuit breaker when the state reports it", async () => {
    const { recoveryClient } = await import("../../api/recovery");
    vi.mocked(recoveryClient.getState).mockResolvedValueOnce({
      state: makeRecoveryState({ status: 4, circuitOpen: true }),
    } as never);

    renderWithProviders(<RecoveryPanel />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.recovery.circuitValue)).toHaveTextContent("Open");
    });
    expect(screen.getByTestId(selectors.recovery.summary)).toHaveTextContent("circuit breaker is open");
    expect(screen.getByTestId(selectors.recovery.nextAction)).toHaveTextContent("Wait for the retry window");
  });

  it("guides operators when failures are accumulating", async () => {
    const { recoveryClient } = await import("../../api/recovery");
    vi.mocked(recoveryClient.getState).mockResolvedValueOnce({
      state: makeRecoveryState({ consecFailures: 2, backoffLevel: 1 }),
    } as never);

    renderWithProviders(<RecoveryPanel />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.recovery.summary)).toHaveTextContent("2 consecutive failure");
    });
    expect(screen.getByTestId(selectors.recovery.nextAction)).toHaveTextContent("Inspect probe and tunnel metrics");
  });

  it("renders the event timeline when events exist", async () => {
    const { recoveryClient } = await import("../../api/recovery");
    vi.mocked(recoveryClient.listEvents).mockResolvedValueOnce({
      events: [makeRecoveryEvent(), makeRecoveryEvent({ id: "event-2", outcome: 2, trigger: "manual" })],
    } as never);

    renderWithProviders(<RecoveryPanel />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.recovery.timeline)).toBeInTheDocument();
    });
    expect(screen.getAllByTestId(selectors.recovery.timelineRow)).toHaveLength(2);
    expect(screen.getAllByTestId(selectors.recovery.eventDetails)[0]).toHaveTextContent("restarted cloudflared");
  });

  it("requires confirmation before recovering, then calls recover", async () => {
    const { recoveryClient } = await import("../../api/recovery");
    const user = userEvent.setup();
    renderWithProviders(<RecoveryPanel />);
    await waitFor(() => expect(screen.getByTestId(selectors.recovery.recoverButton)).toBeInTheDocument());

    // No dialog until the button is pressed.
    expect(screen.queryByTestId(selectors.recovery.confirmDialog)).not.toBeInTheDocument();
    await user.click(screen.getByTestId(selectors.recovery.recoverButton));
    expect(screen.getByTestId(selectors.recovery.confirmDialog)).toBeInTheDocument();

    await user.click(screen.getByTestId(selectors.recovery.confirmButton));
    await waitFor(() => expect(recoveryClient.recover).toHaveBeenCalledTimes(1));
    expect(vi.mocked(recoveryClient.recover).mock.calls[0]?.[0]).toMatchObject({ force: false });
  });

  it("passes force=true when the force toggle is checked", async () => {
    const { recoveryClient } = await import("../../api/recovery");
    const user = userEvent.setup();
    renderWithProviders(<RecoveryPanel />);
    await waitFor(() => expect(screen.getByTestId(selectors.recovery.forceToggle)).toBeInTheDocument());

    await user.click(screen.getByTestId(selectors.recovery.forceToggle));
    expect(screen.getByTestId(selectors.recovery.forceWarning)).toHaveTextContent("bypasses the breaker");
    await user.click(screen.getByTestId(selectors.recovery.recoverButton));
    await user.click(screen.getByTestId(selectors.recovery.confirmButton));

    await waitFor(() => expect(recoveryClient.recover).toHaveBeenCalledWith({ force: true }));
  });

  it("cancels the confirm dialog without recovering", async () => {
    const { recoveryClient } = await import("../../api/recovery");
    const user = userEvent.setup();
    renderWithProviders(<RecoveryPanel />);
    await waitFor(() => expect(screen.getByTestId(selectors.recovery.recoverButton)).toBeInTheDocument());

    await user.click(screen.getByTestId(selectors.recovery.recoverButton));
    await user.click(screen.getByTestId(selectors.recovery.cancelButton));

    expect(screen.queryByTestId(selectors.recovery.confirmDialog)).not.toBeInTheDocument();
    expect(recoveryClient.recover).not.toHaveBeenCalled();
  });
});
