import { jsx as _jsx } from "react/jsx-runtime";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import MessageExportDrawer from "../components/MessageExportDrawer";
function makeEvent(overrides) {
    return {
        sessionId: "sess-1",
        source: "claude_hook",
        role: "assistant",
        text: `Message ${overrides.sequence}`,
        speechParagraphs: [],
        summarized: false,
        createdAt: new Date().toISOString(),
        deliveryState: "received",
        ttsState: "idle",
        consumptionState: "seen",
        ...overrides,
    };
}
const twoEvents = () => [
    makeEvent({ id: "a", sequence: 1, role: "user", text: "how do I deploy?" }),
    makeEvent({ id: "b", sequence: 2, role: "assistant", text: "run make deploy" }),
];
describe("MessageExportDrawer", () => {
    const onClose = vi.fn();
    let writeText;
    beforeEach(() => {
        vi.clearAllMocks();
        writeText = vi.fn().mockResolvedValue(undefined);
        Object.defineProperty(navigator, "clipboard", {
            value: { writeText },
            configurable: true,
        });
    });
    it("renders nothing while closed", () => {
        render(_jsx(MessageExportDrawer, { open: false, events: twoEvents(), onClose: onClose }));
        expect(screen.queryByTestId("msg-export-drawer")).toBeNull();
    });
    it("defaults to Agent XML and previews the selected messages in order", () => {
        render(_jsx(MessageExportDrawer, { open: true, events: twoEvents(), onClose: onClose }));
        expect(screen.getByTestId("msg-export-format-agentXml").getAttribute("aria-checked")).toBe("true");
        const preview = screen.getByTestId("msg-export-preview").textContent ?? "";
        expect(preview).toContain("<conversation>");
        expect(preview).toContain('role="user"');
        expect(preview.indexOf("how do I deploy?")).toBeLessThan(preview.indexOf("run make deploy"));
    });
    it("offers all four formats and switching updates preview and token estimate", () => {
        render(_jsx(MessageExportDrawer, { open: true, events: twoEvents(), onClose: onClose }));
        for (const id of ["agentXml", "markdown", "quote", "plain"]) {
            expect(screen.getByTestId(`msg-export-format-${id}`)).toBeInTheDocument();
        }
        // The header summary is the token/count surface (keys only under cimode).
        expect(screen.getByTestId("msg-export-drawer-summary").textContent).toContain("messageExport.approxTokens");
        fireEvent.click(screen.getByTestId("msg-export-format-plain"));
        expect(screen.getByTestId("msg-export-format-plain").getAttribute("aria-checked")).toBe("true");
        const preview = screen.getByTestId("msg-export-preview").textContent ?? "";
        expect(preview).toContain("[#1] user:");
        expect(preview).not.toContain("<conversation>");
    });
    it("copies the rendered text and shows transient success feedback", async () => {
        render(_jsx(MessageExportDrawer, { open: true, events: twoEvents(), onClose: onClose }));
        fireEvent.click(screen.getByTestId("msg-export-copy"));
        await waitFor(() => {
            expect(screen.getByTestId("msg-export-copy").textContent).toContain("messageExport.copiedFeedback");
        });
        expect(writeText).toHaveBeenCalledTimes(1);
        expect(writeText.mock.calls[0]?.[0]).toContain("<conversation>");
    });
    it("shows an actionable, non-destructive error when the clipboard fails", async () => {
        writeText.mockRejectedValue(new Error("denied"));
        render(_jsx(MessageExportDrawer, { open: true, events: twoEvents(), onClose: onClose }));
        fireEvent.click(screen.getByTestId("msg-export-copy"));
        const error = await screen.findByTestId("msg-export-copy-error");
        expect(error.textContent).toContain("messageExport.copyFailed");
        // Nothing was lost: the preview still shows the full selection and copy is retryable.
        expect(screen.getByTestId("msg-export-preview").textContent).toContain("how do I deploy?");
        expect(screen.getByTestId("msg-export-copy")).not.toBeDisabled();
        expect(onClose).not.toHaveBeenCalled();
    });
    it("disables copy only for empty output", () => {
        render(_jsx(MessageExportDrawer, { open: true, events: [], onClose: onClose }));
        expect(screen.getByTestId("msg-export-copy")).toBeDisabled();
    });
    it("closes via Escape and the close button without side effects", () => {
        render(_jsx(MessageExportDrawer, { open: true, events: twoEvents(), onClose: onClose }));
        fireEvent.keyDown(document, { key: "Escape" });
        expect(onClose).toHaveBeenCalledTimes(1);
        fireEvent.click(screen.getByLabelText("messageExport.closeAriaLabel"));
        expect(onClose).toHaveBeenCalledTimes(2);
    });
    it("labels the preview and format picker for assistive tech", () => {
        render(_jsx(MessageExportDrawer, { open: true, events: twoEvents(), onClose: onClose }));
        expect(screen.getByLabelText("messageExport.previewAriaLabel")).toBeInTheDocument();
        expect(screen.getByRole("radiogroup")).toBeInTheDocument();
    });
    it("bounds very large previews while keeping the full text for copy", async () => {
        const big = makeEvent({ id: "big", sequence: 1, text: "x".repeat(30000) });
        render(_jsx(MessageExportDrawer, { open: true, events: [big], onClose: onClose }));
        expect(screen.getByTestId("msg-export-preview-truncated")).toBeInTheDocument();
        const shown = screen.getByTestId("msg-export-preview").textContent ?? "";
        expect(shown.length).toBeLessThan(30000);
        fireEvent.click(screen.getByTestId("msg-export-copy"));
        await waitFor(() => expect(writeText).toHaveBeenCalled());
        expect((writeText.mock.calls[0]?.[0]).length).toBeGreaterThan(30000);
    });
});
