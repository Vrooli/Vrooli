import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { create, type MessageInitShape } from "@bufbuild/protobuf";
import {
  ItemSchema,
  ItemKind,
  Retention,
  type Item,
} from "@vrooli/proto-types/device-sync-hub/v1/transfer/transfer_pb";

import { renderWithProviders, seedSession } from "../../test-utils";

const { listItems, deleteItem, downloadItem } = vi.hoisted(() => ({
  listItems: vi.fn(),
  deleteItem: vi.fn(),
  downloadItem: vi.fn(),
}));

vi.mock("../../api/transfer", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/transfer")>();
  return {
    ...actual,
    transferClient: { listItems, deleteItem },
    downloadItem,
    // ItemThumbnail calls fetchItemBlob; stub it so the image branch is inert.
    fetchItemBlob: vi.fn().mockResolvedValue(new Blob()),
  };
});

import { ReceivePanel } from "./ReceivePanel";
import { selectors } from "../../consts/selectors";

type ItemInit = MessageInitShape<typeof ItemSchema>;

const fileItem = (over: ItemInit = {}): Item =>
  create(ItemSchema, {
    id: "item-file",
    kind: ItemKind.FILE,
    name: "report.pdf",
    sizeBytes: 2048n,
    retention: Retention.HELD,
    originDeviceId: "other-device",
    ...over,
  });

const textItem = (over: ItemInit = {}): Item =>
  create(ItemSchema, {
    id: "item-text",
    kind: ItemKind.TEXT,
    name: "",
    text: "hello clipboard",
    retention: Retention.LIVE,
    originDeviceId: "other-device",
    ...over,
  });

describe("ReceivePanel", () => {
  beforeEach(() => {
    seedSession({ device: undefined });
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("shows the empty state when there are no items", async () => {
    listItems.mockResolvedValue({ items: [] });
    renderWithProviders(<ReceivePanel />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.receive.empty)).toBeInTheDocument();
    });
  });

  it("renders file + text items and exposes the right per-kind actions", async () => {
    listItems.mockResolvedValue({ items: [fileItem(), textItem()] });
    renderWithProviders(<ReceivePanel />);

    await waitFor(() => {
      expect(screen.getByTestId(selectors.receive.item({ id: "item-file" }))).toBeInTheDocument();
    });
    // File item → download button; text item → copy button.
    expect(screen.getByTestId(selectors.receive.download({ id: "item-file" }))).toBeInTheDocument();
    expect(screen.getByTestId(selectors.receive.copy({ id: "item-text" }))).toBeInTheDocument();
  });

  it("triggers the device-token download when Download is clicked", async () => {
    const user = userEvent.setup();
    listItems.mockResolvedValue({ items: [fileItem()] });
    renderWithProviders(<ReceivePanel />);

    await waitFor(() =>
      expect(screen.getByTestId(selectors.receive.download({ id: "item-file" }))).toBeInTheDocument(),
    );
    await user.click(screen.getByTestId(selectors.receive.download({ id: "item-file" })));
    expect(downloadItem).toHaveBeenCalledWith("item-file", "report.pdf");
  });

  it("filters to text items only via the kind filter", async () => {
    const user = userEvent.setup();
    listItems.mockResolvedValue({ items: [fileItem(), textItem()] });
    renderWithProviders(<ReceivePanel />);

    await waitFor(() =>
      expect(screen.getByTestId(selectors.receive.item({ id: "item-file" }))).toBeInTheDocument(),
    );
    await user.selectOptions(screen.getByTestId(selectors.receive.filter), "text");

    expect(screen.queryByTestId(selectors.receive.item({ id: "item-file" }))).not.toBeInTheDocument();
    expect(screen.getByTestId(selectors.receive.item({ id: "item-text" }))).toBeInTheDocument();
  });

  it("searches items by name", async () => {
    const user = userEvent.setup();
    listItems.mockResolvedValue({
      items: [fileItem({ id: "a", name: "alpha.txt" }), fileItem({ id: "b", name: "beta.txt" })],
    });
    renderWithProviders(<ReceivePanel />);

    await waitFor(() => expect(screen.getByTestId(selectors.receive.item({ id: "a" }))).toBeInTheDocument());
    await user.type(screen.getByTestId(selectors.receive.search), "alpha");

    expect(screen.getByTestId(selectors.receive.item({ id: "a" }))).toBeInTheDocument();
    expect(screen.queryByTestId(selectors.receive.item({ id: "b" }))).not.toBeInTheDocument();
  });

  it("exposes remove only on items this device originated", async () => {
    listItems.mockResolvedValue({
      items: [fileItem({ id: "mine", originDeviceId: "dev-1" }), fileItem({ id: "theirs", originDeviceId: "other" })],
    });
    renderWithProviders(<ReceivePanel />);

    await waitFor(() => expect(screen.getByTestId(selectors.receive.item({ id: "mine" }))).toBeInTheDocument());
    const mine = screen.getByTestId(selectors.receive.item({ id: "mine" }));
    const theirs = screen.getByTestId(selectors.receive.item({ id: "theirs" }));
    expect(within(mine).queryByTestId(selectors.receive.remove({ id: "mine" }))).toBeInTheDocument();
    expect(within(theirs).queryByTestId(selectors.receive.remove({ id: "theirs" }))).not.toBeInTheDocument();
  });
});
