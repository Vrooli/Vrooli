import { afterEach, describe, expect, it, vi } from "vitest";
import { act, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import daily from "../../../../../skills/notification-hub/kit/samples/daily-briefing.json?raw";

import { fetchAttachment } from "../../../api/attachments";
import { conversationsClient } from "../../../api/notifications";
import { selectors } from "../../../consts/selectors";
import { strings } from "../../../consts/strings";
import { useTheme } from "../../../theme/themeContext";
import { renderWithProviders } from "../../../test-utils";
import { makeAttachment } from "../mocks/askFactory";
import { BRIDGE_PROTOCOL } from "./bridge";
import { ExplainerFrame } from "./ExplainerFrame";

vi.mock("../../../api/attachments", () => ({ attachmentUrl: vi.fn(), fetchAttachment: vi.fn() }));
vi.mock("../../../api/notifications", () => ({ conversationsClient: { getAsk: vi.fn(), putAttachmentBlob: vi.fn() } }));

const explainer = makeAttachment({ name: "explainer.html", explainer: true, mediaType: "text/html", previewKind: "code" });
const attachments = [explainer, makeAttachment({ name: "set-a.png", previewKind: "image", mediaType: "image/png", sizeBytes: 100 })];
const PAGE = '<!doctype html><title>x</title><h1>Pick</h1><img src="set-a.png"><button onclick="hub.prefill(\'set\',\'a\')">A</button>';

const response = (body: string, type: string) => ({ ok: true, text: () => Promise.resolve(body), blob: () => Promise.resolve(new Blob([body], { type })) }) as unknown as Response;

function mockAttachments() {
  vi.mocked(fetchAttachment).mockImplementation((_askId, name) =>
    Promise.resolve(name === "explainer.html" ? response(PAGE, "text/html") : response("PNGDATA", "image/png")),
  );
}

async function renderFrame() {
  const onOpen = vi.fn();
  const onOpenAsk = vi.fn();
  const onPrefill = vi.fn();
  renderWithProviders(<ExplainerFrame askId="ask-1" explainer={explainer} attachments={attachments} onOpen={onOpen} onOpenAsk={onOpenAsk} onPrefill={onPrefill} />);
  const frame = await screen.findByTitle<HTMLIFrameElement>(strings.packet.explainer.frameTitle);
  return { frame, onOpen, onOpenAsk, onPrefill };
}

const post = (frame: HTMLIFrameElement, data: unknown, source: Window | null = frame.contentWindow) => {
  act(() => {
    window.dispatchEvent(new MessageEvent("message", { data, source }));
  });
};

afterEach(() => vi.clearAllMocks());

describe("ExplainerFrame", () => {
  it("seals the agent's page: scripts only, no same-origin access, a policy first, and the attachment inlined", async () => {
    mockAttachments();
    const { frame } = await renderFrame();

    expect(frame.getAttribute("sandbox")).toBe("allow-scripts");
    const doc = new DOMParser().parseFromString(frame.getAttribute("srcdoc") ?? "", "text/html");
    expect(doc.head.firstElementChild?.getAttribute("http-equiv")).toBe("Content-Security-Policy");
    expect(doc.querySelector("img")?.getAttribute("src")).toMatch(/^data:image\/png;base64,/);
    expect(doc.querySelector('script[data-hub="bridge"]')?.textContent).toContain("window.hub");
    expect(screen.getByTestId(selectors.packet.explainer)).toHaveAttribute("aria-label", strings.packet.explainer.label);
  });

  it("lets the page pre-fill the form and open an attachment, and ignores everything else", async () => {
    mockAttachments();
    const { frame, onOpen, onOpenAsk, onPrefill } = await renderFrame();

    post(frame, { hub: BRIDGE_PROTOCOL, type: "prefill", questionId: "set", value: "a" });
    expect(onPrefill).toHaveBeenCalledWith("set", "a");
    post(frame, { hub: BRIDGE_PROTOCOL, type: "open", name: "set-a.png" });
    expect(onOpen).toHaveBeenCalledWith("set-a.png", "");
    post(frame, { hub: BRIDGE_PROTOCOL, type: "open-ask", askId: "ask-next" });
    expect(onOpenAsk).toHaveBeenCalledWith("ask-next");

    // A name the ask does not have, a foreign window, and an unknown message type.
    post(frame, { hub: BRIDGE_PROTOCOL, type: "open", name: "secret.png" });
    post(frame, { hub: BRIDGE_PROTOCOL, type: "prefill", questionId: "set", value: "b" }, window);
    post(frame, { hub: BRIDGE_PROTOCOL, type: "submit", answer: "close" });
    expect(onOpen).toHaveBeenCalledTimes(1);
    expect(onPrefill).toHaveBeenCalledTimes(1);
  });

  it("saves an item note through the Hub sheet into the answer changes", async () => {
    const user = userEvent.setup();
    const onAnswerChange = vi.fn();
    renderWithProviders(<ExplainerFrame askId="ask-1" templateName="daily-briefing" templateData={JSON.parse(daily)} attachments={[]} onOpen={vi.fn()} onPrefill={vi.fn()} onAnswerChange={onAnswerChange} />);
    const frame = await screen.findByTitle<HTMLIFrameElement>(strings.packet.explainer.frameTitle);
    post(frame, { hub: BRIDGE_PROTOCOL, type: "note", key: "measure.latency", label: "Latency" });
    await user.type(await screen.findByRole("textbox", { name: strings.packet.noteSheet.yourNote }), "The chart needs context.");
    await user.click(screen.getByRole("button", { name: strings.packet.noteSheet.save }));
    expect(onAnswerChange).toHaveBeenCalledWith({ type: "answer", operation: "set", key: "notes", label: "Notes · 1", value: { "measure.latency": { label: "Latency", text: "The chart needs context.", attachments: [] } } });
  });

  it("resolves only an attached folder entry to a data URL for dynamic viewers", async () => {
    mockAttachments();
    const evidence = makeAttachment({ name: "evidence", folder: true, entries: [{ path: "image/a.png", mediaType: "image/png" }] });
    renderWithProviders(<ExplainerFrame askId="ask-1" explainer={explainer} attachments={[...attachments, evidence]} onOpen={vi.fn()} onPrefill={vi.fn()} />);
    const frame = await screen.findByTitle<HTMLIFrameElement>(strings.packet.explainer.frameTitle);
    const posted = vi.spyOn(frame.contentWindow as Window, "postMessage");
    post(frame, { hub: BRIDGE_PROTOCOL, type: "asset", requestId: "a3", name: "evidence", path: "image/a.png" });
    await waitFor(() => expect(posted).toHaveBeenCalledWith(expect.objectContaining({ hub: BRIDGE_PROTOCOL, type: "asset-resolve", requestId: "a3", dataUrl: expect.stringMatching(/^data:image\/png;base64,/) }), "*"));
    expect(fetchAttachment).toHaveBeenCalledWith("ask-1", "evidence", "image/a.png");
  });

  it("renders a Hub kit template without fetching producer HTML and supplies real Hub context", async () => {
    const onAnswerChange = vi.fn();
    const templateData = JSON.parse(daily);
    renderWithProviders(<ExplainerFrame
      askId="ask-1"
      kind="status"
      recurrence={{ count: 3, dates: ["2026-11-14", "2026-11-15", "2026-11-16"], previousAnswers: [] }}
      templateName="daily-briefing"
      templateData={templateData}
      attachments={[]}
      onOpen={vi.fn()}
      onPrefill={vi.fn()}
      onAnswerChange={onAnswerChange}
    />);
    const frame = await screen.findByTitle<HTMLIFrameElement>(strings.packet.explainer.frameTitle);
    const doc = new DOMParser().parseFromString(frame.getAttribute("srcdoc") ?? "", "text/html");
    expect(fetchAttachment).not.toHaveBeenCalled();
    expect(JSON.parse(doc.getElementById("hub-data")?.textContent ?? "null")).toEqual(templateData);
    expect(JSON.parse(doc.getElementById("hub-context")?.textContent ?? "null")).toEqual({
      kind: "status",
      recurrence: { count: 3, dates: ["2026-11-14", "2026-11-15", "2026-11-16"], previousAnswers: [] },
      asks: {},
      asksTruncated: false,
    });

    post(frame, { hub: BRIDGE_PROTOCOL, type: "answer", operation: "set", key: "pace", label: "Use Wednesday", value: { tier: "wednesday" } });
    expect(onAnswerChange).toHaveBeenCalledWith({ type: "answer", operation: "set", key: "pace", label: "Use Wednesday", value: { tier: "wednesday" } });
  });

  it("loads live Hub context only for ask ids named in template data", async () => {
    vi.mocked(conversationsClient.getAsk).mockResolvedValue({ ask: { id: "ask-linked", state: "pending", createdAt: "2026-11-17T12:00:00Z", answeredAt: "", question: "Approve the test?" } } as never);
    const templateData = { ...JSON.parse(daily), asks: [{ id: "ask-linked" }] };
    mockAttachments();
    renderWithProviders(<ExplainerFrame askId="ask-1" kind="status" templateData={templateData} explainer={explainer} attachments={attachments} onOpen={vi.fn()} onPrefill={vi.fn()} />);
    const frame = await screen.findByTitle<HTMLIFrameElement>(strings.packet.explainer.frameTitle);
    await waitFor(() => {
      const context = JSON.parse(new DOMParser().parseFromString(frame.getAttribute("srcdoc") ?? "", "text/html").getElementById("hub-context")?.textContent ?? "null");
      expect(context.asks["ask-linked"]).toEqual({ state: "pending", askedAt: "2026-11-17T12:00:00Z", answeredAt: "", question: "Approve the test?" });
    });
    expect(conversationsClient.getAsk).toHaveBeenCalledWith({ askId: "ask-linked" });
  });

  it("bounds named ask reads and signals omitted template references", async () => {
    const ids = Array.from({ length: 25 }, (_, index) => `ask-${index}`);
    vi.mocked(conversationsClient.getAsk).mockImplementation(({ askId }) => Promise.resolve({ ask: { id: askId, state: "pending", createdAt: "", answeredAt: "", question: "" } } as never));
    const templateData = {
      ...JSON.parse(daily),
      asks: [
        ...ids.slice(0, 20).map((id) => ({ id })),
        { id: "ask-0" }, // duplicate does not add another query
        false,
        { id: "   " },
        { id: 42 },
        ...ids.slice(20).map((id) => ({ id })),
      ],
    };
    mockAttachments();
    renderWithProviders(<ExplainerFrame askId="ask-1" kind="status" templateData={templateData} explainer={explainer} attachments={attachments} onOpen={vi.fn()} onPrefill={vi.fn()} />);
    const frame = await screen.findByTitle<HTMLIFrameElement>(strings.packet.explainer.frameTitle);

    await waitFor(() => {
      const context = JSON.parse(new DOMParser().parseFromString(frame.getAttribute("srcdoc") ?? "", "text/html").getElementById("hub-context")?.textContent ?? "null");
      expect(Object.keys(context.asks)).toHaveLength(20);
      expect(context.asksTruncated).toBe(true);
    });
    expect(conversationsClient.getAsk).toHaveBeenCalledTimes(20);
    expect(conversationsClient.getAsk).not.toHaveBeenCalledWith({ askId: "ask-20" });
  });

  it("sizes the frame to the height the page reports", async () => {
    mockAttachments();
    const { frame } = await renderFrame();

    post(frame, { hub: BRIDGE_PROTOCOL, type: "height", height: 640 });
    expect(frame.style.height).toBe("640px");
  });

  it("tells the running page when the theme changes instead of reloading it", async () => {
    mockAttachments();
    const user = userEvent.setup();
    function DarkButton() {
      const { setTheme } = useTheme();
      return <button type="button" onClick={() => setTheme("dark")}>dark</button>;
    }
    renderWithProviders(
      <>
        <DarkButton />
        <ExplainerFrame askId="ask-1" explainer={explainer} attachments={attachments} onOpen={vi.fn()} onPrefill={vi.fn()} />
      </>,
    );
    const frame = await screen.findByTitle<HTMLIFrameElement>(strings.packet.explainer.frameTitle);
    const posted = vi.spyOn(frame.contentWindow as Window, "postMessage");
    const srcdoc = frame.getAttribute("srcdoc");

    await user.click(screen.getByRole("button", { name: "dark" }));

    expect(posted).toHaveBeenCalledWith({ hub: BRIDGE_PROTOCOL, type: "theme", theme: "dark" }, "*");
    expect(frame.getAttribute("srcdoc")).toBe(srcdoc);
  });

  it("falls back to opening the page as a file when it cannot be loaded, and never blocks the decision", async () => {
    vi.mocked(fetchAttachment).mockRejectedValue(new Error("gone"));
    const user = userEvent.setup();
    const onOpen = vi.fn();
    renderWithProviders(<ExplainerFrame askId="ask-1" explainer={explainer} attachments={attachments} onOpen={onOpen} onPrefill={vi.fn()} />);

    expect(await screen.findByRole("alert", {}, { timeout: 3000 })).toHaveTextContent("packet.explainer.failed");
    await waitFor(() => expect(screen.queryByTitle(strings.packet.explainer.frameTitle)).not.toBeInTheDocument());
    await user.click(screen.getByRole("button", { name: "packet.explainer.openFile" }));
    expect(onOpen).toHaveBeenCalledWith("explainer.html", "");
  });

  it("renders the page without the old header strip and bleeds to the screen edge on a phone", async () => {
    mockAttachments();
    const { frame } = await renderFrame();

    const section = screen.getByTestId(selectors.packet.explainer);
    expect(section).toContainElement(frame);
    expect(section).toHaveClass("-mx-[var(--page-gutter)]", "border-y", "md:mx-0", "md:border");
    expect(section.querySelector("header")).toBeNull();
    expect(section).not.toHaveTextContent(strings.packet.explainer.sealed);
    expect(frame).not.toHaveClass("p-3");
  });
});
