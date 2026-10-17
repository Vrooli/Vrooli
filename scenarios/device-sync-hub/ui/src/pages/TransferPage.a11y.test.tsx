import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";

import { expectNoA11yViolations, renderWithProviders, seedSession } from "../test-utils";
import { setLocale } from "../i18n";

const { listItems, listDevices } = vi.hoisted(() => ({
  listItems: vi.fn(),
  listDevices: vi.fn(),
}));

vi.mock("../api/transfer", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/transfer")>();
  return { ...actual, transferClient: { listItems }, uploadItem: vi.fn(), downloadItem: vi.fn() };
});
vi.mock("../api/devices", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/devices")>();
  return { ...actual, devicesClient: { listDevices } };
});

import { TransferPage } from "./TransferPage";
import { selectors } from "../consts/selectors";

describe("TransferPage accessibility", () => {
  beforeEach(async () => {
    await setLocale("en");
    seedSession();
    listItems.mockResolvedValue({ items: [] });
    listDevices.mockResolvedValue({ devices: [] });
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the split-screen transfer surface without axe violations", async () => {
    const { container } = renderWithProviders(<TransferPage />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.receive.empty)).toBeInTheDocument();
    });
    await expectNoA11yViolations(container);
  });
});
