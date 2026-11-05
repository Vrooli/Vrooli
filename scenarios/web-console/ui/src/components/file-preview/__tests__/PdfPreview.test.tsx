import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders as render } from "../../../test-utils";
import { previewModel, previewProps } from "../../../test-utils/filePreview";
import { strings } from "../../../consts/strings";
import type { PdfCallbacks, PdfSession } from "../pdfRuntime";
import { PdfPreview } from "../renderers/PdfPreview";

const runtime = vi.hoisted(() => ({ create: vi.fn(), session: { page: vi.fn(), zoom: vi.fn(), find: vi.fn(), password: vi.fn(), destroy: vi.fn() } }));
vi.mock("../pdfRuntime", () => ({ createPdfSession: runtime.create }));
const c = strings.artifactPreview;
const props = previewProps({ model: previewModel({ kind: "pdf", mimeType: "application/pdf", basename: "report.pdf" }) });
let callbacks: PdfCallbacks | null = null;
beforeEach(() => {
  callbacks = null;
  vi.clearAllMocks();
  runtime.create.mockImplementation((_container: HTMLDivElement, _pages: HTMLDivElement, _href: string, events: PdfCallbacks) => { callbacks = events; return Promise.resolve(runtime.session); });
});
async function loaded() {
  const result = render(<PdfPreview {...props} />);
  await waitFor(() => { expect(runtime.create).toHaveBeenCalled(); });
  const events = callbacks;
  if (!events) throw new Error("Missing reader callbacks");
  act(() => { events.ready(27); events.scale(0.5); });
  return { ...result, events };
}

describe("document reader", () => {
  it("loads lazily, navigates pages, zooms and exposes download independently", async () => {
    const { events } = await loaded();
    expect(runtime.create.mock.calls[0]?.[2]).toBe("/preview/blob");
    fireEvent.click(screen.getByRole("button", { name: c.nextPage }));
    expect(runtime.session.page).toHaveBeenCalledWith(2);
    act(() => { events.page(2); });
    expect(screen.getByRole("textbox", { name: c.page })).toHaveValue("2");
    fireEvent.change(screen.getByRole("textbox", { name: c.page }), { target: { value: "12" } });
    const form = screen.getByRole("textbox", { name: c.page }).closest("form");
    if (!form) throw new Error("Missing page navigation form");
    fireEvent.submit(form);
    expect(runtime.session.page).toHaveBeenCalledWith(12);
    fireEvent.change(screen.getByRole("combobox", { name: c.pdfZoom }), { target: { value: "page-fit" } });
    expect(runtime.session.zoom).toHaveBeenCalledWith("page-fit");
    fireEvent.click(screen.getByRole("button", { name: c.zoomIn }));
    expect(runtime.session.zoom).toHaveBeenLastCalledWith("0.625");
    expect(screen.getByTestId("file-preview-download")).toHaveAttribute("href", "/preview/blob");
  });
  it("searches text, moves between results and clears highlights when search closes", async () => {
    const { events } = await loaded();
    fireEvent.click(screen.getByRole("button", { name: c.searchDocument }));
    const search = screen.getByRole("searchbox", { name: c.searchDocument });
    fireEvent.change(search, { target: { value: "release" } });
    await waitFor(() => { expect(runtime.session.find).toHaveBeenCalledWith("release", false, false); });
    act(() => { events.matches(1, 4, false); });
    fireEvent.click(screen.getByRole("button", { name: c.nextMatch }));
    expect(runtime.session.find).toHaveBeenLastCalledWith("release", false, true);
    fireEvent.keyDown(search, { key: "Escape" });
    expect(screen.queryByRole("searchbox")).not.toBeInTheDocument();
    expect(runtime.session.find).toHaveBeenLastCalledWith("", false, false);
  });
  it("supports password entry and keeps recovery actions available after a load error", async () => {
    const { events } = await loaded();
    act(() => { events.password(true); });
    expect(screen.getByText(c.incorrectPassword)).toBeInTheDocument();
    const input = screen.getByLabelText(c.password);
    fireEvent.change(input, { target: { value: "password" } });
    const form = input.closest("form");
    if (!form) throw new Error("Missing password form");
    fireEvent.submit(form);
    expect(runtime.session.password).toHaveBeenCalledWith("password");
    expect(input).toHaveValue("");
    act(() => { events.error(); });
    expect(screen.getByRole("alert")).toHaveTextContent(c.pdfLoadError);
    expect(screen.getByTestId("file-preview-download")).toBeInTheDocument();
  });
  it("destroys the session on close and resets document state for another source", async () => {
    const { rerender, unmount } = await loaded();
    rerender(<PdfPreview {...props} model={{ ...props.model, previewId: "next", blobHref: "/next" }} />);
    expect(runtime.session.destroy).toHaveBeenCalledTimes(1);
    expect(screen.getByRole("button", { name: c.nextPage })).toBeDisabled();
    await waitFor(() => { expect(runtime.create).toHaveBeenCalledTimes(2); });
    unmount();
    expect(runtime.session.destroy).toHaveBeenCalledTimes(2);
  });
  it("disposes a late reader without replacing or updating the next document", async () => {
    let resolve: ((session: PdfSession) => void) | undefined;
    const stale = { ...runtime.session, destroy: vi.fn() };
    runtime.create.mockImplementationOnce((_container: HTMLDivElement, _pages: HTMLDivElement, _href: string, events: PdfCallbacks) => {
      callbacks = events;
      return new Promise<PdfSession>((complete) => { resolve = complete; });
    });
    const { rerender, unmount } = render(<PdfPreview {...props} />);
    await waitFor(() => { expect(runtime.create).toHaveBeenCalledTimes(1); });
    const previousCallbacks = callbacks;
    if (!previousCallbacks) throw new Error("Missing previous reader callbacks");
    rerender(<PdfPreview {...props} model={{ ...props.model, previewId: "next", blobHref: "/next" }} />);
    await waitFor(() => { expect(runtime.create).toHaveBeenCalledTimes(2); });
    act(() => { previousCallbacks.ready(99); previousCallbacks.error(); });
    expect(screen.getByRole("button", { name: c.nextPage })).toBeDisabled();
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
    await act(async () => { resolve?.(stale); await Promise.resolve(); });
    expect(stale.destroy).toHaveBeenCalledTimes(1);
    expect(runtime.session.destroy).not.toHaveBeenCalled();
    unmount();
    expect(runtime.session.destroy).toHaveBeenCalledTimes(1);
  });
});
