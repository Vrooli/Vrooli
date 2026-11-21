import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Code, ConnectError } from "@connectrpc/connect";

import { renderWithProviders } from "../../test-utils";
import { strings } from "../../consts/strings";

const { issuePairingCode } = vi.hoisted(() => ({ issuePairingCode: vi.fn() }));

vi.mock("../../api/devices", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/devices")>();
  return { ...actual, devicesClient: { issuePairingCode } };
});

import { IssuePairingCode } from "./IssuePairingCode";
import { selectors } from "../../consts/selectors";

describe("IssuePairingCode", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("issues a code with the trimmed device name and renders it with a QR", async () => {
    const user = userEvent.setup();
    issuePairingCode.mockResolvedValue({ pairingCode: { code: "ABCD-1234" } });
    renderWithProviders(<IssuePairingCode />);

    await user.type(screen.getByTestId(selectors.devices.issueNameInput), "  Laptop  ");
    await user.click(screen.getByTestId(selectors.devices.issueButton));

    await waitFor(() => expect(issuePairingCode).toHaveBeenCalledWith({ deviceName: "Laptop" }));
    await waitFor(() =>
      expect(screen.getByTestId(selectors.devices.issuedCode)).toHaveTextContent("ABCD-1234"),
    );
    expect(screen.getByTestId(selectors.devices.issuedQr)).toBeInTheDocument();
  });

  it("surfaces an error message when issuing fails", async () => {
    const user = userEvent.setup();
    issuePairingCode.mockRejectedValue(new ConnectError("nope", Code.PermissionDenied));
    renderWithProviders(<IssuePairingCode />);

    await user.click(screen.getByTestId(selectors.devices.issueButton));

    // cimode renders the i18n key, so the mapped error key is what surfaces.
    await waitFor(() =>
      expect(screen.getByText(strings.errors.permission_denied)).toBeInTheDocument(),
    );
  });
});
