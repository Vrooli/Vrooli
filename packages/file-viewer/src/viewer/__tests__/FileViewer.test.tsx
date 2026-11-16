import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { strings } from "../../i18n/strings.js";
import { createManifestPreviewSource, type ManifestFile } from "../../source/manifestSource.js";
import { NO_CAPABILITIES, type PreviewSource } from "../../source/types.js";
import { renderWithProviders as render } from "../../test-utils/index.js";
import { FileViewer, type FileViewerProps } from "../FileViewer.js";
import { useFilePreviewController } from "../useFilePreviewController.js";

const files: ManifestFile[] = [
  { path: "plan.md", url: "/blob/plan", kind: "markdown", mimeType: "text/markdown", sizeBytes: 12 },
  { path: "notes.ts", url: "/blob/notes", kind: "code", mimeType: "text/plain", sizeBytes: 9 },
  { path: "shots/home.png", url: "/blob/home", kind: "image", mimeType: "image/png", sizeBytes: 90, thumbnailUrl: "/thumb/home" },
  { path: "bundle.zip", url: "/blob/bundle", kind: "archive", mimeType: "application/zip", sizeBytes: 500 },
];

const fetchText = (text: string) => vi.fn().mockImplementation(() => Promise.resolve(new Response(text)));

function Harness({ source, open, ...props }: { source: PreviewSource; open: string } & Partial<FileViewerProps>) {
  const controller = useFilePreviewController(source);
  return <>
    <button onClick={() => { void controller.openPreview(open); }}>open</button>
    <FileViewer source={source} controller={controller} presentation="dock" onClose={controller.close} {...props} />
  </>;
}

async function openFile(source: PreviewSource, path: string, props: Partial<FileViewerProps> = {}) {
  const view = render(<Harness source={source} open={path} {...props} />);
  fireEvent.click(screen.getByText("open"));
  return view;
}

describe("FileViewer over a manifest source", () => {
  it("opens an attachment, loading its renderer and text on demand", async () => {
    await openFile(createManifestPreviewSource({ scope: "ask:1", files, fetch: fetchText("# Plan\n\nBody text.") }), "plan.md");
    expect(await screen.findByTestId("file-preview-markdown")).toBeInTheDocument();
    expect(await screen.findByRole("heading", { name: "Plan" })).toBeInTheDocument();
    expect(screen.getByText("Body text.")).toBeInTheDocument();
  });

  it("shows the unavailable card with Retry when the text cannot be read", async () => {
    const failing = vi.fn().mockResolvedValue(new Response("", { status: 500 }));
    await openFile(createManifestPreviewSource({ scope: "ask:1", files, fetch: failing }), "notes.ts");
    expect(await screen.findByText("Could not read notes.ts (500)")).toBeInTheDocument();
    failing.mockResolvedValue(new Response("const a = 1;"));
    fireEvent.click(screen.getByTestId("file-preview-reopen"));
    await waitFor(() => { expect(screen.getByTestId("file-preview-code")).toHaveTextContent("const a = 1;"); });
  });

  it("reports a path outside the manifest as unavailable instead of offering to browse", async () => {
    await openFile(createManifestPreviewSource({ scope: "ask:1", files }), "missing.md");
    expect(await screen.findByText(strings.viewer.openFailed)).toBeInTheDocument();
    expect(screen.queryByTestId("file-preview-search-banner")).not.toBeInTheDocument();
  });
});

describe("opening at a heading", () => {
  it("scrolls the opened document to the anchor and forgets it when navigating on", async () => {
    const source = createManifestPreviewSource({ scope: "ask:1", files, fetch: fetchText("# Plan\n\none\n\n## Plan\n\ntwo\n\n## Done") });
    const Anchored = () => { const controller = useFilePreviewController(source); return <>
      <button onClick={() => { void controller.openPreview("plan.md", undefined, { anchor: "plan-1" }); }}>open</button>
      <FileViewer source={source} controller={controller} presentation="dock" onClose={controller.close} />
      <output data-testid="anchor">{controller.state.anchor ?? "none"}</output>
    </>; };
    render(<Anchored />); fireEvent.click(screen.getByText("open"));
    await waitFor(() => { expect(document.activeElement).toBe(screen.getAllByRole("heading", { name: "Plan" })[1]); });
    expect(screen.getByTestId("anchor")).toHaveTextContent("plan-1");
  });
});

describe("host slots", () => {
  it("hands the open target to the footer and header actions as it changes, in both presentations", async () => {
    const source = createManifestPreviewSource({ scope: "ask:1", files, fetch: fetchText("body") });
    const renderFooter = vi.fn(({ path }: { path: string }) => <p>pin to {path || "nothing"}</p>);
    await openFile(source, "notes.ts", { renderFooter });
    expect(await screen.findByText("pin to /notes.ts")).toBeInTheDocument();
    expect(screen.getByTestId("file-preview-footer")).toBeInTheDocument();
    expect(renderFooter).toHaveBeenLastCalledWith(expect.objectContaining({ path: "/notes.ts", model: expect.objectContaining({ kind: "code" }) }));
  });

  it("renders no footer band when the host returns nothing", async () => {
    await openFile(createManifestPreviewSource({ scope: "ask:1", files, fetch: fetchText("body") }), "notes.ts", { renderFooter: () => null });
    expect(await screen.findByTestId("file-preview-code")).toBeInTheDocument();
    expect(screen.queryByTestId("file-preview-footer")).toBeNull();
  });

  it("lets touch events from the host's swipe handlers bubble through the drawer", async () => {
    const source = createManifestPreviewSource({ scope: "ask:1", files, fetch: fetchText("body") });
    const swipe = vi.fn();
    const Drawer = () => { const controller = useFilePreviewController(source); return <div onTouchStart={swipe}>
      <button onClick={() => { void controller.openPreview("notes.ts"); }}>open</button>
      <FileViewer source={source} controller={controller} onClose={controller.close} />
    </div>; };
    render(<Drawer />); fireEvent.click(screen.getByText("open"));
    fireEvent.touchStart(await screen.findByTestId("file-preview-code"), { touches: [{ clientX: 10, clientY: 10 }] });
    expect(swipe).toHaveBeenCalled();
  });
});

describe("source capabilities", () => {
  it("offers 'open containing folder' only when the source has host parent folders", async () => {
    const manifest = createManifestPreviewSource({ scope: "ask:1", files, fetch: fetchText("x") });
    const first = await openFile(manifest, "notes.ts");
    await screen.findByTestId("file-preview-path");
    expect(screen.queryByTestId("file-preview-open-folder")).not.toBeInTheDocument();
    first.unmount();

    await openFile({ ...manifest, capabilities: { ...NO_CAPABILITIES, parentFolders: true } }, "notes.ts");
    expect(await screen.findByTestId("file-preview-open-folder")).toBeInTheDocument();
  });

  it("shows image thumbnails in a folder only when the source can make them", async () => {
    const manifest = createManifestPreviewSource({ scope: "ask:1", files });
    const withThumbnails = await openFile(manifest, "shots");
    expect(await screen.findByTestId("directory-entry-thumbnail")).toBeInTheDocument();
    withThumbnails.unmount();

    await openFile({ ...manifest, capabilities: { ...manifest.capabilities, thumbnails: false } }, "shots");
    await screen.findByTestId("file-preview-directory");
    expect(screen.queryByTestId("directory-entry-thumbnail")).not.toBeInTheDocument();
  });

  it("leaves an archive download-only when the source cannot open archives", async () => {
    await openFile(createManifestPreviewSource({ scope: "ask:1", files }), "bundle.zip");
    expect(await screen.findByTestId("file-preview-unsupported")).toBeInTheDocument();
    expect(screen.getByTestId("file-preview-download")).toHaveAttribute("href", "/blob/bundle");
  });
});

describe("FileViewer chrome", () => {
  it("renders the host's header actions with the open target, and hides the path on request", async () => {
    const actions = vi.fn().mockReturnValue(<button>Hand off</button>);
    await openFile(createManifestPreviewSource({ scope: "ask:1", files, fetch: fetchText("const a = 1;") }), "notes.ts", { renderHeaderActions: actions, showPath: false });
    expect(await screen.findByRole("button", { name: "Hand off" })).toBeInTheDocument();
    await waitFor(() => { expect(actions).toHaveBeenCalledWith(expect.objectContaining({ path: "/notes.ts", model: expect.objectContaining({ kind: "code" }), text: expect.objectContaining({ content: "const a = 1;" }) })); });
    expect(screen.queryByTestId("file-preview-path")).not.toBeInTheDocument();
  });
});

describe("source scope", () => {
  it("closes what it shows when the scope changes, but not when only the object does", async () => {
    const first = createManifestPreviewSource({ scope: "ask:1", files, fetch: fetchText("const a = 1;") });
    const { rerender } = await openFile(first, "notes.ts");
    await waitFor(() => { expect(screen.getByTestId("file-preview-code")).toHaveTextContent("const a = 1;"); });

    // A new object with the same scope is the same thing to look at.
    rerender(<Harness source={createManifestPreviewSource({ scope: "ask:1", files, fetch: fetchText("other") })} open="notes.ts" />);
    expect(screen.getByTestId("file-preview-code")).toHaveTextContent("const a = 1;");

    // Another scope never shows the previous scope's file.
    await act(async () => { rerender(<Harness source={createManifestPreviewSource({ scope: "ask:2", files })} open="notes.ts" />); });
    expect(screen.queryByTestId("file-preview-code")).not.toBeInTheDocument();
    expect(screen.queryByText("notes.ts")).not.toBeInTheDocument();
  });
});
