import { describe, expect, it } from "vitest";
import { previewBlobHref } from "../filePreview";
describe("previewBlobHref", () => {
    it("returns empty for empty input", () => {
        expect(previewBlobHref("")).toBe("");
    });
    it("passes absolute URLs through unchanged", () => {
        expect(previewBlobHref("https://example.com/x")).toBe("https://example.com/x");
    });
    it("keeps a same-origin relative path resolvable (ends with the blob path)", () => {
        const href = previewBlobHref("/api/v1/sessions/s/file-previews/pv/blob");
        expect(href.endsWith("/api/v1/sessions/s/file-previews/pv/blob")).toBe(true);
    });
});
