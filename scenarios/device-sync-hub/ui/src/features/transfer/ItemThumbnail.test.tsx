import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, render, waitFor } from "@testing-library/react";

const { fetchItemBlob } = vi.hoisted(() => ({ fetchItemBlob: vi.fn() }));

vi.mock("../../api/transfer", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/transfer")>();
  return { ...actual, fetchItemBlob };
});

import { ItemThumbnail } from "./ItemThumbnail";

describe("ItemThumbnail", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
    vi.unstubAllGlobals();
  });

  it("renders the decorative placeholder until the blob resolves", () => {
    fetchItemBlob.mockReturnValue(new Promise(() => {}));
    const { container } = render(<ItemThumbnail itemId="i1" alt="thumb" />);
    expect(container.querySelector('span[aria-hidden="true"]')).toBeInTheDocument();
    expect(container.querySelector("img")).toBeNull();
  });

  it("fetches the thumbnail variant and renders an img from the object url", async () => {
    fetchItemBlob.mockResolvedValue(new Blob(["img"]));
    const createObjectURL = vi.fn().mockReturnValue("blob:thumb");
    vi.stubGlobal("URL", { ...URL, createObjectURL, revokeObjectURL: vi.fn() });

    const { container } = render(<ItemThumbnail itemId="i1" alt="my photo" />);

    await waitFor(() => expect(container.querySelector("img")).not.toBeNull());
    expect(fetchItemBlob).toHaveBeenCalledWith("i1", { thumb: true });
    const img = container.querySelector("img")!;
    expect(img).toHaveAttribute("src", "blob:thumb");
    expect(img).toHaveAttribute("alt", "my photo");
  });

  it("falls back to the placeholder when the fetch rejects", async () => {
    fetchItemBlob.mockRejectedValue(new Error("no thumb"));
    const { container } = render(<ItemThumbnail itemId="i1" alt="thumb" />);
    await waitFor(() => expect(fetchItemBlob).toHaveBeenCalled());
    expect(container.querySelector("img")).toBeNull();
  });

  it("revokes the object url on unmount", async () => {
    fetchItemBlob.mockResolvedValue(new Blob(["img"]));
    const revokeObjectURL = vi.fn();
    vi.stubGlobal("URL", { ...URL, createObjectURL: () => "blob:thumb", revokeObjectURL });

    const { container, unmount } = render(<ItemThumbnail itemId="i1" alt="thumb" />);
    await waitFor(() => expect(container.querySelector("img")).not.toBeNull());
    unmount();
    expect(revokeObjectURL).toHaveBeenCalledWith("blob:thumb");
  });
});
