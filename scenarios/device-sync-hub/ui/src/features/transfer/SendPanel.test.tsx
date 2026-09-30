import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders, seedSession } from "../../test-utils";

const { createTextItem, uploadItem, listDevices } = vi.hoisted(() => ({
  createTextItem: vi.fn(),
  uploadItem: vi.fn(),
  listDevices: vi.fn(),
}));

vi.mock("../../api/transfer", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/transfer")>();
  return { ...actual, transferClient: { createTextItem }, uploadItem };
});

vi.mock("../../api/devices", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/devices")>();
  return { ...actual, devicesClient: { listDevices } };
});

import { SendPanel } from "./SendPanel";
import { selectors } from "../../consts/selectors";

describe("SendPanel", () => {
  beforeEach(() => {
    seedSession();
    listDevices.mockResolvedValue({ devices: [] });
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("disables Send until something is staged", () => {
    renderWithProviders(<SendPanel />);
    expect(screen.getByTestId(selectors.send.sendButton)).toBeDisabled();
  });

  it("stages a text snippet and sends it via createTextItem", async () => {
    const user = userEvent.setup();
    createTextItem.mockResolvedValue({});
    renderWithProviders(<SendPanel />);

    await user.type(screen.getByTestId(selectors.send.textInput), "ship it");
    await user.click(screen.getByTestId(selectors.send.addText));
    expect(screen.getByTestId(selectors.send.staged)).toBeInTheDocument();

    await user.click(screen.getByTestId(selectors.send.sendButton));
    await waitFor(() => {
      expect(createTextItem).toHaveBeenCalledWith(
        expect.objectContaining({ text: "ship it" }),
      );
    });
  });

  it("stages a picked file and uploads it via the device-token upload", async () => {
    const user = userEvent.setup();
    uploadItem.mockResolvedValue({});
    renderWithProviders(<SendPanel />);

    const file = new File(["data"], "doc.txt", { type: "text/plain" });
    await user.upload(screen.getByTestId(selectors.send.fileInput), file);
    expect(screen.getByTestId(selectors.send.staged)).toBeInTheDocument();

    await user.click(screen.getByTestId(selectors.send.sendButton));
    await waitFor(() => {
      expect(uploadItem).toHaveBeenCalledTimes(1);
    });
    expect(uploadItem.mock.calls[0]?.[0]).toBe(file);
  });

  it("removes a staged item before sending", async () => {
    const user = userEvent.setup();
    renderWithProviders(<SendPanel />);

    await user.type(screen.getByTestId(selectors.send.textInput), "discard me");
    await user.click(screen.getByTestId(selectors.send.addText));
    const staged = screen.getByTestId(selectors.send.staged);
    const removeBtn = staged.querySelector('[data-testid^="send-remove-staged-"]');
    expect(removeBtn).not.toBeNull();
    await user.click(removeBtn as HTMLElement);

    expect(screen.getByTestId(selectors.send.sendButton)).toBeDisabled();
  });
});
