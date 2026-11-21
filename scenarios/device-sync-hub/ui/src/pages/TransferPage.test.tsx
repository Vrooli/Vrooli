import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, cleanup, screen, waitFor } from "@testing-library/react";
import { create, toJson } from "@bufbuild/protobuf";
import {
  EventSchema,
  EventType,
} from "@vrooli/proto-types/device-sync-hub/v1/realtime/realtime_pb";

import { renderWithProviders, seedSession } from "../test-utils";

const pairingLine = JSON.stringify(
  toJson(
    EventSchema,
    create(EventSchema, {
      type: EventType.PAIRING_REQUESTED,
      pairing: { deviceId: "p1", name: "Phone", kind: "phone" },
    }),
    { useProtoFieldName: true },
  ),
);

function emitPairing() {
  const instances = (globalThis.EventSource as unknown as { instances: { emit: (d: string) => void }[] }).instances;
  act(() => instances[instances.length - 1]!.emit(pairingLine));
}

const { listItems, listDevices } = vi.hoisted(() => ({
  listItems: vi.fn(),
  listDevices: vi.fn(),
}));

vi.mock("../api/transfer", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/transfer")>();
  return { ...actual, transferClient: { listItems } };
});

vi.mock("../api/devices", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/devices")>();
  return { ...actual, devicesClient: { listDevices } };
});

import { TransferPage } from "./TransferPage";
import { selectors } from "../consts/selectors";

describe("TransferPage", () => {
  beforeEach(() => {
    listItems.mockResolvedValue({ items: [] });
    listDevices.mockResolvedValue({ devices: [] });
  });
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the receive-over-send split", () => {
    renderWithProviders(<TransferPage />);
    expect(screen.getByTestId(selectors.pages.transfer)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.send.sendButton)).toBeInTheDocument();
  });

  it("overlays the pairing banner when a pairing request arrives", async () => {
    // A paired session opens the SSE stream the banner state rides on.
    seedSession();
    renderWithProviders(<TransferPage />);

    emitPairing();
    await waitFor(() =>
      expect(screen.getByTestId(selectors.devices.pendingBanner)).toBeInTheDocument(),
    );
  });
});
