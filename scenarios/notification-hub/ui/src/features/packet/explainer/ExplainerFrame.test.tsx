import { afterEach, describe, expect, it, vi } from "vitest";
import { act, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { fetchAttachment } from "../../../api/attachments";
import { selectors } from "../../../consts/selectors";
import { strings } from "../../../consts/strings";
import { useTheme } from "../../../theme/themeContext";
import { renderWithProviders } from "../../../test-utils";
import { makeAttachment } from "../mocks/askFactory";
import { BRIDGE_PROTOCOL } from "./bridge";
import { ExplainerFrame } from "./ExplainerFrame";

vi.mock("../../../api/attachments", () => ({ attachmentUrl: vi.fn(), fetchAttachment: vi.fn() }));

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
  const onPrefill = vi.fn();
  renderWithProviders(<ExplainerFrame askId="ask-1" explainer={explainer} attachments={attachments} onOpen={onOpen} onPrefill={onPrefill} />);
  const frame = await screen.findByTitle<HTMLIFrameElement>(strings.packet.explainer.frameTitle);
  return { frame, onOpen, onPrefill };
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
    expect(screen.getByTestId(selectors.packet.explainer)).toHaveTextContent("packet.explainer.label");
  });

  it("lets the page pre-fill the form and open an attachment, and ignores everything else", async () => {
    mockAttachments();
    const { frame, onOpen, onPrefill } = await renderFrame();

    post(frame, { hub: BRIDGE_PROTOCOL, type: "prefill", questionId: "set", value: "a" });
    expect(onPrefill).toHaveBeenCalledWith("set", "a");
    post(frame, { hub: BRIDGE_PROTOCOL, type: "open", name: "set-a.png" });
    expect(onOpen).toHaveBeenCalledWith("set-a.png", "");

    // A name the ask does not have, a foreign window, and an unknown message type.
    post(frame, { hub: BRIDGE_PROTOCOL, type: "open", name: "secret.png" });
    post(frame, { hub: BRIDGE_PROTOCOL, type: "prefill", questionId: "set", value: "b" }, window);
    post(frame, { hub: BRIDGE_PROTOCOL, type: "submit", answer: "close" });
    expect(onOpen).toHaveBeenCalledTimes(1);
    expect(onPrefill).toHaveBeenCalledTimes(1);
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
});
