import { useState } from "react";
import { createTranslator, strings as viewerStrings } from "@vrooli/file-viewer";
import { afterEach, describe, expect, it, vi } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { selectors } from "../../../consts/selectors";
import { strings } from "../../../consts/strings";
import { renderWithProviders } from "../../../test-utils";
import { makeAttachment } from "../mocks/askFactory";
import { attachmentNoteFor, emptyDraft, type PacketDraft } from "../model/draft";
import { AttachmentViewer } from "./AttachmentViewer";
import type { AttachmentTarget } from "./attachments";

vi.mock("../../../api/attachments", () => ({
  attachmentUrl: (askId: string, name: string, path = "") => `/files/${askId}/${name}${path ? `?path=${encodeURIComponent(path)}` : ""}`,
  fetchAttachment: vi.fn(),
}));

const attachments = [
  makeAttachment({ name: "brief.md" }),
  makeAttachment({ name: "set-a.png", previewKind: "image", mediaType: "image/png", sourceStatus: "changed" }),
  makeAttachment({ name: "gone.md", sourceStatus: "missing" }),
  makeAttachment({ name: "renders", folder: true, entries: [{ path: "walk/01.png", previewKind: "image", mediaType: "image/png" }, { path: "notes.txt" }] }),
];

const BRIEF = "# Goal\nthree videos\n## Plan\nfirst\n### Plan\nsecond";

/** What the Hub's content route returns for a text file. */
function serve(files: Record<string, string>) {
  const fetchMock = vi.fn((input: RequestInfo | URL) => {
    const url = String(input instanceof Request ? input.url : input);
    const body = files[url];
    return Promise.resolve(body === undefined ? new Response("unavailable", { status: 404 }) : new Response(body, { status: 200 }));
  });
  vi.stubGlobal("fetch", fetchMock);
  return fetchMock;
}

function Harness({ start, canNote = true, onClose = vi.fn(), initial = emptyDraft() }: { start: AttachmentTarget; canNote?: boolean; onClose?: () => void; initial?: PacketDraft }) {
  const [draft, setDraft] = useState(initial);
  return (
    <>
      <AttachmentViewer askId="ask-1" attachments={attachments} target={start} onClose={onClose} draft={draft} onDraftChange={setDraft} canNote={canNote} />
      <output data-testid="draft-notes">{JSON.stringify(draft.attachmentNotes)}</output>
      <output data-testid="draft-note-brief">{attachmentNoteFor(draft, "brief.md", "")}</output>
    </>
  );
}

/** The shared viewer's own English copy, read from its catalog rather than retyped. */
const viewerText = createTranslator("en");
const footer = () => screen.getByTestId(selectors.packet.viewerComment).closest("div");
const position = () => screen.getByTestId(selectors.packet.viewerPrevious).parentElement;
afterEach(() => {
  vi.unstubAllGlobals();
  vi.clearAllMocks();
});

describe("AttachmentViewer", () => {
  it("shows the Hub's own copy of an image and says when the original changed since", async () => {
    renderWithProviders(<Harness start={{ name: "set-a.png", path: "" }} />);

    expect(await screen.findByRole("img", { name: /set-a\.png/ })).toHaveAttribute("src", "/files/ask-1/set-a.png");
    expect(screen.getByText(strings.packet.viewer.changed)).toBeInTheDocument();
  });

  it("reads a text file from the stored copy through the session", async () => {
    const fetchMock = serve({ "/files/ask-1/brief.md": BRIEF });
    renderWithProviders(<Harness start={{ name: "brief.md", path: "" }} />);

    expect(await screen.findByText(/three videos/)).toBeInTheDocument();
    expect(fetchMock).toHaveBeenCalledWith("/files/ask-1/brief.md", expect.objectContaining({ credentials: "same-origin" }));
  });

  it("sends an operator whose session ended to sign in instead of showing an empty file", async () => {
    vi.stubGlobal("fetch", vi.fn(() => Promise.resolve(new Response("", { status: 401 }))));
    const expired = vi.fn();
    window.addEventListener("notification-hub:session-expired", expired);
    renderWithProviders(<Harness start={{ name: "brief.md", path: "" }} />);

    await waitFor(() => expect(expired).toHaveBeenCalled());
    window.removeEventListener("notification-hub:session-expired", expired);
  });

  it("opens a Markdown file at the heading a section names", async () => {
    serve({ "/files/ask-1/brief.md": BRIEF });
    renderWithProviders(<Harness start={{ name: "brief.md", path: "", anchor: "plan-1" }} />);

    const headings = await screen.findAllByRole("heading", { name: "Plan" });
    await waitFor(() => expect(headings[1]).toHaveFocus());
  });

  it("moves between attachments with the buttons and cannot go past either end", async () => {
    serve({ "/files/ask-1/brief.md": BRIEF });
    const user = userEvent.setup();
    renderWithProviders(<Harness start={{ name: "brief.md", path: "" }} />);

    await screen.findByText(/three videos/);
    expect(screen.getByTestId(selectors.packet.viewerPrevious)).toBeDisabled();
    expect(position()).toHaveTextContent("packet.viewer.position");
    await user.click(screen.getByTestId(selectors.packet.viewerNext));
    expect(await screen.findByRole("img", { name: /set-a\.png/ })).toBeInTheDocument();
    await user.click(screen.getByTestId(selectors.packet.viewerNext));
    await user.click(screen.getByTestId(selectors.packet.viewerNext));
    expect(await screen.findByRole("button", { name: /walk/ })).toBeInTheDocument();
    expect(screen.getByTestId(selectors.packet.viewerNext)).toBeDisabled();
  });

  it("names the file once in the footer with its format and size, with a download link, and shows no path line", async () => {
    serve({ "/files/ask-1/brief.md": BRIEF });
    renderWithProviders(<Harness start={{ name: "brief.md", path: "" }} />);

    await screen.findByText(/three videos/);
    expect(footer()).toHaveTextContent(/MD · 1\.0 KB/);
    const download = screen.getByRole("link", { name: strings.packet.viewer.download });
    expect(download).toHaveAttribute("download");
    expect(download.getAttribute("href")).toContain("/files/ask-1/brief.md");
    // The header names the attachment; the storage path line and its copy control are hidden.
    expect(screen.queryByTestId("file-preview-path")).not.toBeInTheDocument();
  });

  it("offers a file in its own tab, but not a folder listing", async () => {
    serve({ "/files/ask-1/brief.md": BRIEF });
    const user = userEvent.setup();
    renderWithProviders(<Harness start={{ name: "brief.md", path: "" }} />);

    await screen.findByText(/three videos/);
    const tab = screen.getByTestId(selectors.packet.viewerOpenTab);
    expect(tab).toHaveAttribute("target", "_blank");
    expect(tab).toHaveAttribute("rel", expect.stringContaining("noreferrer"));
    expect(tab).toHaveAccessibleName(strings.packet.viewer.openInNewTab);
    expect(tab.getAttribute("href")).toContain("/files/ask-1/brief.md");

    await user.click(screen.getByTestId(selectors.packet.viewerNext));
    await user.click(screen.getByTestId(selectors.packet.viewerNext));
    await user.click(screen.getByTestId(selectors.packet.viewerNext));
    expect(await screen.findByRole("button", { name: /walk/ })).toBeInTheDocument();
    expect(screen.queryByTestId(selectors.packet.viewerOpenTab)).not.toBeInTheDocument();
  });

  it("warns when the original file no longer exists", async () => {
    serve({ "/files/ask-1/gone.md": "old copy" });
    renderWithProviders(<Harness start={{ name: "gone.md", path: "" }} />);

    await waitFor(() => expect(screen.getByText(strings.packet.viewer.missing)).toBeInTheDocument());
    expect(await screen.findByText(/old copy/)).toBeInTheDocument();
  });

  it("lists a folder's files, offers no note on the listing, and notes a file inside it separately", async () => {
    serve({ "/files/ask-1/renders?path=notes.txt": "folder note text" });
    const user = userEvent.setup();
    renderWithProviders(<Harness start={{ name: "renders", path: "" }} />);

    await user.click(await screen.findByRole("button", { name: /notes\.txt/ }));
    expect(await screen.findByText(/folder note text/)).toBeInTheDocument();
    await user.click(screen.getByTestId(selectors.packet.viewerComment));
    await user.type(screen.getByTestId(selectors.packet.viewerNote), "Cut this");
    await user.click(screen.getByTestId(selectors.packet.viewerSaveNote));

    expect(screen.getByTestId("draft-notes")).toHaveTextContent('"attachment":"renders"');
    expect(screen.getByTestId("draft-notes")).toHaveTextContent('"path":"notes.txt"');
    expect(screen.getByTestId("draft-note-brief")).toBeEmptyDOMElement();
  });

  it("pins a note to the file on screen, keeps it when the viewer moves on, and removes it on request", async () => {
    serve({ "/files/ask-1/brief.md": BRIEF });
    const user = userEvent.setup();
    renderWithProviders(<Harness start={{ name: "brief.md", path: "" }} />);

    await user.click(await screen.findByTestId(selectors.packet.viewerComment));
    await user.type(screen.getByTestId(selectors.packet.viewerNote), "Too warm");
    await user.click(screen.getByTestId(selectors.packet.viewerSaveNote));
    expect(screen.getByTestId("draft-note-brief")).toHaveTextContent("Too warm");

    await user.click(screen.getByTestId(selectors.packet.viewerNext));
    expect(await screen.findByRole("img", { name: /set-a\.png/ })).toBeInTheDocument();
    expect(screen.getByTestId(selectors.packet.viewerComment)).toHaveTextContent(strings.packet.viewer.comment);
    await user.click(screen.getByTestId(selectors.packet.viewerPrevious));
    await user.click(await screen.findByTestId(selectors.packet.viewerComment));
    expect(screen.getByTestId(selectors.packet.viewerNote)).toHaveValue("Too warm");
    await user.click(screen.getByRole("button", { name: strings.packet.viewer.commentRemove }));
    expect(screen.getByTestId("draft-notes")).toHaveTextContent("[]");
  });

  it("offers no note once the decision can no longer be answered", async () => {
    renderWithProviders(<Harness start={{ name: "set-a.png", path: "" }} canNote={false} />);

    await screen.findByRole("img", { name: /set-a\.png/ });
    expect(screen.queryByTestId(selectors.packet.viewerComment)).not.toBeInTheDocument();
  });

  it("closes from the viewer's own header", async () => {
    const onClose = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(<Harness start={{ name: "set-a.png", path: "" }} onClose={onClose} />);

    await screen.findByRole("img", { name: /set-a\.png/ });
    // The drawer's backdrop and its header button share one accessible name; the header button comes last.
    const closers = screen.getAllByRole("button", { name: viewerText(viewerStrings.viewer.closeAriaLabel) });
    await user.click(closers[closers.length - 1] as HTMLElement);
    expect(onClose).toHaveBeenCalledTimes(1);
  });
});
