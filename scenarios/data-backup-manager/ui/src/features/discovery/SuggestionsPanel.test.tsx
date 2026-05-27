/**
 * SuggestionsPanel tests. The load-bearing behaviors: Enable goes through the
 * EXISTING register/create RPCs (never a separate accept path) with correctly
 * mapped args; Dismiss calls dismissSuggestion; and a separate-root-unsafe
 * destination disables Enable and shows the reason (CreateDestination would
 * reject it anyway).
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";

vi.mock("../../api/discovery", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../../api/discovery")>()),
  listTargetSuggestions: vi.fn(),
  listDestinationSuggestions: vi.fn(),
  dismissSuggestion: vi.fn(),
}));
vi.mock("../../api/targets", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../../api/targets")>()),
  registerTarget: vi.fn(),
}));
vi.mock("../../api/destinations", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../../api/destinations")>()),
  createDestination: vi.fn(),
}));

import * as discoveryApi from "../../api/discovery";
import * as targetsApi from "../../api/targets";
import * as destinationsApi from "../../api/destinations";
import { SourceKind } from "../../api/targets";
import { BackendKind, CapPolicy } from "../../api/destinations";
import { DriveClass } from "../../api/discovery";
import { SuggestionsPanel } from "./SuggestionsPanel";

const targetSuggestion = {
  id: "ts1",
  owner: "vrooli",
  name: "plans",
  sourceKind: SourceKind.FILESYSTEM,
  locator: "/home/u/.vrooli/plans",
  rationale: "Your Vrooli plans.",
  approxBytes: 4096n,
};

const sensitiveSuggestion = {
  id: "ts-creds",
  owner: "claude-code",
  name: "credentials",
  sourceKind: SourceKind.FILESYSTEM,
  locator: "/home/u/.claude/.credentials.json",
  rationale: "Claude Code OAuth credentials.",
  approxBytes: 471n,
  sensitive: true,
  warning: "Includes credentials/tokens — review before backing up.",
};

const usbSuggestion = {
  id: "ds-usb",
  label: "Removable drive — USB",
  backendKind: BackendKind.FILESYSTEM,
  location: "/media/u/USB",
  driveClass: DriveClass.REMOVABLE,
  freeBytes: 50n,
  totalBytes: 64n,
  removable: true,
  separateRootOk: true,
  rationale: "Plugged-in removable drive.",
};

const rootSuggestion = {
  id: "ds-root",
  label: "Volume — system",
  backendKind: BackendKind.FILESYSTEM,
  location: "/",
  driveClass: DriveClass.FIXED,
  freeBytes: 100n,
  totalBytes: 500n,
  removable: false,
  separateRootOk: false,
  rationale: "Overlaps protected data.",
};

beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(discoveryApi.listTargetSuggestions).mockResolvedValue([targetSuggestion] as never);
  vi.mocked(discoveryApi.listDestinationSuggestions).mockResolvedValue([
    usbSuggestion,
    rootSuggestion,
  ] as never);
  vi.mocked(discoveryApi.dismissSuggestion).mockResolvedValue(true);
  vi.mocked(targetsApi.registerTarget).mockResolvedValue({ id: "t1" } as never);
  vi.mocked(destinationsApi.createDestination).mockResolvedValue({ id: "d1" } as never);
});

afterEach(() => cleanup());

describe("SuggestionsPanel", () => {
  it("enables a target suggestion via registerTarget with mapped args", async () => {
    const user = userEvent.setup();
    renderWithProviders(<SuggestionsPanel />);
    await screen.findByTestId(selectors.discovery.suggestionRow({ id: "ts1" }));

    await user.click(screen.getByTestId(selectors.discovery.enableButton({ id: "ts1" })));

    await waitFor(() =>
      expect(targetsApi.registerTarget).toHaveBeenCalledWith({
        owner: "vrooli",
        name: "plans",
        sourceKind: SourceKind.FILESYSTEM,
        locator: "/home/u/.vrooli/plans",
      }),
    );
  });

  it("enables a destination suggestion via createDestination with a filesystem backend", async () => {
    const user = userEvent.setup();
    renderWithProviders(<SuggestionsPanel />);
    await screen.findByTestId(selectors.discovery.suggestionRow({ id: "ds-usb" }));

    await user.click(screen.getByTestId(selectors.discovery.enableButton({ id: "ds-usb" })));

    await waitFor(() =>
      expect(destinationsApi.createDestination).toHaveBeenCalledWith({
        name: "Removable drive — USB",
        backendKind: BackendKind.FILESYSTEM,
        location: "/media/u/USB",
        capBytes: 0n,
        capPolicy: CapPolicy.UNSPECIFIED,
      }),
    );
  });

  it("dismisses a suggestion via dismissSuggestion", async () => {
    const user = userEvent.setup();
    renderWithProviders(<SuggestionsPanel />);
    await screen.findByTestId(selectors.discovery.suggestionRow({ id: "ts1" }));

    await user.click(screen.getByTestId(selectors.discovery.dismissButton({ id: "ts1" })));

    await waitFor(() => expect(discoveryApi.dismissSuggestion).toHaveBeenCalledWith("ts1"));
  });

  it("disables Enable and shows the reason for a separate-root-unsafe destination", async () => {
    renderWithProviders(<SuggestionsPanel />);
    await screen.findByTestId(selectors.discovery.suggestionRow({ id: "ds-root" }));

    expect(screen.getByTestId(selectors.discovery.enableButton({ id: "ds-root" }))).toBeDisabled();
    expect(screen.getByText(strings.discovery.unsafe)).toBeInTheDocument();
    // The safe USB row's Enable stays clickable.
    expect(screen.getByTestId(selectors.discovery.enableButton({ id: "ds-usb" }))).not.toBeDisabled();
  });

  it("flags a sensitive target suggestion with a badge and warning", async () => {
    vi.mocked(discoveryApi.listTargetSuggestions).mockResolvedValue([
      targetSuggestion,
      sensitiveSuggestion,
    ] as never);
    renderWithProviders(<SuggestionsPanel />);
    await screen.findByTestId(selectors.discovery.suggestionRow({ id: "ts-creds" }));

    // Exactly the sensitive row carries the badge + the server-provided warning.
    expect(screen.getAllByText(strings.discovery.sensitive)).toHaveLength(1);
    expect(screen.getByText(sensitiveSuggestion.warning)).toBeInTheDocument();
  });

  it("shows per-group empty copy when a group has no suggestions", async () => {
    vi.mocked(discoveryApi.listTargetSuggestions).mockResolvedValue([] as never);
    renderWithProviders(<SuggestionsPanel />);
    expect(await screen.findByText(strings.discovery.targetsEmpty)).toBeInTheDocument();
  });
});
