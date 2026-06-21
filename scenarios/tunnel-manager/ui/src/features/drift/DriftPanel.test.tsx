/**
 * DriftPanel tests — the classified ingress table with per-row Adopt / Ignore /
 * Prune actions. Covers loading / error / empty states, state-badge copy, and
 * that each row action calls the right mutation (Prune behind a confirm).
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { OwnershipState, IngressSource } from "@vrooli/proto-types/tunnel-manager/v1/config/config_pb";

import { renderWithProviders } from "../../test-utils";
import { makeConfigMocks, makeDriftResponse, makeIngressEntry } from "../../test-utils/mocks/config";

vi.mock("../../api/config", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/config")>();
  return { ...actual, ...makeConfigMocks() };
});

import { DriftPanel } from "./DriftPanel";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";

const HOST = "agent-manager.itsagitime.com";

describe("DriftPanel", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("shows the loading state, then the classified table", async () => {
    renderWithProviders(<DriftPanel />);
    expect(screen.getByTestId(selectors.queryState.loading)).toBeInTheDocument();
    await waitFor(() => {
      expect(screen.getByTestId(selectors.drift.table)).toBeInTheDocument();
    });
    expect(screen.getByTestId(selectors.drift.hostname)).toHaveTextContent(HOST);
    expect(screen.getByTestId(selectors.drift.stateBadge)).toHaveTextContent("Managed");
    expect(screen.getByTestId(selectors.drift.sourceBadge)).toHaveTextContent("Scenario");
  });

  it("renders the error state when getDrift rejects", async () => {
    const { configClient } = await import("../../api/config");
    vi.mocked(configClient.getDrift).mockRejectedValueOnce(new Error("boom"));

    renderWithProviders(<DriftPanel />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.queryState.error)).toBeInTheDocument();
    });
  });

  it("renders the empty state when there is no live ingress", async () => {
    const { configClient } = await import("../../api/config");
    vi.mocked(configClient.getDrift).mockResolvedValueOnce(makeDriftResponse({ entries: [] }));

    renderWithProviders(<DriftPanel />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.queryState.empty)).toHaveTextContent("No ingress hostnames are live yet");
    });
  });

  it("classifies an unmanaged hostname with a danger badge", async () => {
    const { configClient } = await import("../../api/config");
    vi.mocked(configClient.getDrift).mockResolvedValueOnce(
      makeDriftResponse({
        entries: [
          makeIngressEntry({
            hostname: "stray.itsagitime.com",
            scenario: "",
            state: OwnershipState.UNMANAGED,
            source: IngressSource.EXTERNAL,
          }),
        ],
      }),
    );

    renderWithProviders(<DriftPanel />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.drift.stateBadge)).toHaveTextContent("Unmanaged");
    });
    expect(screen.getByTestId(selectors.drift.sourceBadge)).toHaveTextContent("External");
  });

  it("adopts a hostname when the row Adopt action is used", async () => {
    const { configClient } = await import("../../api/config");
    const user = userEvent.setup();
    renderWithProviders(<DriftPanel />);
    await waitFor(() => expect(screen.getByTestId(selectors.drift.table)).toBeInTheDocument());

    await user.click(screen.getByTestId(selectors.drift.adoptButton({ hostname: HOST })));
    await waitFor(() => {
      expect(configClient.adoptIngress).toHaveBeenCalledWith({ hostname: HOST });
    });
  });

  it("ignores a hostname when the row Ignore action is used", async () => {
    const { configClient } = await import("../../api/config");
    const user = userEvent.setup();
    renderWithProviders(<DriftPanel />);
    await waitFor(() => expect(screen.getByTestId(selectors.drift.table)).toBeInTheDocument());

    await user.click(screen.getByTestId(selectors.drift.ignoreButton({ hostname: HOST })));
    await waitFor(() => {
      expect(configClient.ignoreIngress).toHaveBeenCalledWith({ hostname: HOST });
    });
  });

  it("requires confirmation before pruning, then calls pruneIngress", async () => {
    const { configClient } = await import("../../api/config");
    const user = userEvent.setup();
    renderWithProviders(<DriftPanel />);
    await waitFor(() => expect(screen.getByTestId(selectors.drift.table)).toBeInTheDocument());

    // No dialog and no call until confirmed.
    expect(screen.queryByTestId(selectors.drift.confirmDialog)).not.toBeInTheDocument();
    await user.click(screen.getByTestId(selectors.drift.pruneButton({ hostname: HOST })));
    expect(screen.getByTestId(selectors.drift.confirmDialog)).toHaveTextContent(HOST);
    expect(configClient.pruneIngress).not.toHaveBeenCalled();

    await user.click(screen.getByTestId(selectors.drift.confirmButton));
    await waitFor(() => {
      expect(configClient.pruneIngress).toHaveBeenCalledWith({ hostname: HOST });
    });
  });

  it("cancels the prune confirm without calling pruneIngress", async () => {
    const { configClient } = await import("../../api/config");
    const user = userEvent.setup();
    renderWithProviders(<DriftPanel />);
    await waitFor(() => expect(screen.getByTestId(selectors.drift.table)).toBeInTheDocument());

    await user.click(screen.getByTestId(selectors.drift.pruneButton({ hostname: HOST })));
    await user.click(screen.getByTestId(selectors.drift.cancelButton));

    expect(screen.queryByTestId(selectors.drift.confirmDialog)).not.toBeInTheDocument();
    expect(configClient.pruneIngress).not.toHaveBeenCalled();
  });

  it("surfaces an action error when a mutation rejects", async () => {
    const { configClient } = await import("../../api/config");
    vi.mocked(configClient.adoptIngress).mockRejectedValueOnce(new Error("nope"));
    const user = userEvent.setup();
    renderWithProviders(<DriftPanel />);
    await waitFor(() => expect(screen.getByTestId(selectors.drift.table)).toBeInTheDocument());

    await user.click(screen.getByTestId(selectors.drift.adoptButton({ hostname: HOST })));
    await waitFor(() => {
      expect(screen.getByTestId(selectors.drift.actionError)).toBeInTheDocument();
    });
  });
});
