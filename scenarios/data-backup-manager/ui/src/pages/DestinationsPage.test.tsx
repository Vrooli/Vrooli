import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";

vi.mock("../api/destinations", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../api/destinations")>()),
  listDestinations: vi.fn(),
  createDestination: vi.fn(),
  updateDestination: vi.fn(),
  deleteDestination: vi.fn(),
}));

import * as api from "../api/destinations";
import { BackendKind, CapPolicy, UsageState } from "../api/destinations";
import { DestinationsPage } from "./DestinationsPage";

const dest = {
  id: "d1",
  name: "local",
  backendKind: BackendKind.FILESYSTEM,
  location: "/var/backups",
  capBytes: 100n * 1024n ** 3n,
  capPolicy: CapPolicy.ALERT_BLOCK,
  encryptionAlgorithm: "AES256-GCM",
  secretRef: "vrooli/kopia/d1:repository-passphrase",
  usageBytes: 10n * 1024n ** 3n,
  usageState: UsageState.WITHIN,
};

beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(api.listDestinations).mockResolvedValue([dest] as never);
  vi.mocked(api.createDestination).mockResolvedValue({ id: "d2" } as never);
  vi.mocked(api.deleteDestination).mockResolvedValue(undefined);
});

afterEach(() => cleanup());

describe("DestinationsPage", () => {
  it("lists destinations with their secret reference shown read-only", async () => {
    renderWithProviders(<DestinationsPage />);
    const row = await screen.findByTestId(selectors.destinations.row({ id: "d1" }));
    expect(row).toHaveTextContent("vrooli/kopia/d1:repository-passphrase");
  });

  it("requires name and location before creating", async () => {
    const user = userEvent.setup();
    renderWithProviders(<DestinationsPage />);
    await user.click(screen.getByTestId(selectors.destinations.createButton));
    await user.click(screen.getByTestId(selectors.destinations.formSubmit));
    expect(screen.getByText(strings.destinations.nameRequired)).toBeInTheDocument();
    expect(api.createDestination).not.toHaveBeenCalled();
  });

  it("creates a destination, converting the GB cap to bytes", async () => {
    const user = userEvent.setup();
    renderWithProviders(<DestinationsPage />);
    await user.click(screen.getByTestId(selectors.destinations.createButton));
    await user.type(screen.getByTestId(selectors.destinations.formName), "offsite");
    await user.type(screen.getByTestId(selectors.destinations.formLocation), "s3://bucket/dbm");
    const cap = screen.getByTestId(selectors.destinations.formCap);
    await user.clear(cap);
    await user.type(cap, "2");
    await user.click(screen.getByTestId(selectors.destinations.formSubmit));
    expect(api.createDestination).toHaveBeenCalledWith(
      expect.objectContaining({ name: "offsite", location: "s3://bucket/dbm", capBytes: 2n * 1024n ** 3n }),
    );
  });

  it("previews the derived kopia repository path for a filesystem destination", async () => {
    const user = userEvent.setup();
    renderWithProviders(<DestinationsPage />);
    await user.click(screen.getByTestId(selectors.destinations.createButton));
    await user.type(screen.getByTestId(selectors.destinations.formName), "elements-local");
    await user.type(screen.getByTestId(selectors.destinations.formLocation), "/media/usb/vrooli-backups");
    const preview = await screen.findByTestId(selectors.destinations.formRepoPreview);
    expect(preview).toHaveTextContent("/media/usb/vrooli-backups/repositories/elements-local.kopia");
  });

  it("rejects a non-slug destination name", async () => {
    const user = userEvent.setup();
    renderWithProviders(<DestinationsPage />);
    await user.click(screen.getByTestId(selectors.destinations.createButton));
    await user.type(screen.getByTestId(selectors.destinations.formName), "Elements Local");
    await user.type(screen.getByTestId(selectors.destinations.formLocation), "/media/usb");
    await user.click(screen.getByTestId(selectors.destinations.formSubmit));
    expect(screen.getByText(strings.destinations.nameInvalid)).toBeInTheDocument();
    expect(api.createDestination).not.toHaveBeenCalled();
  });

  it("shows the nested repository path on a destination row", async () => {
    vi.mocked(api.listDestinations).mockResolvedValue([
      { ...dest, repositoryLocation: "/var/backups/repositories/local.kopia" },
    ] as never);
    renderWithProviders(<DestinationsPage />);
    const row = await screen.findByTestId(selectors.destinations.row({ id: "d1" }));
    expect(row).toHaveTextContent("/var/backups/repositories/local.kopia");
  });

  it("passes the delete-repository choice through delete", async () => {
    const user = userEvent.setup();
    renderWithProviders(<DestinationsPage />);
    await screen.findByTestId(selectors.destinations.row({ id: "d1" }));
    await user.click(screen.getByTestId(selectors.destinations.deleteButton));
    await user.click(screen.getByTestId(selectors.destinations.deleteRepoToggle));
    await user.click(screen.getByTestId(selectors.destinations.deleteConfirm));
    expect(api.deleteDestination).toHaveBeenCalledWith("d1", true);
  });
});
