import { describe, it, expect } from "vitest";
import { pluralize } from "../lib/pluralize";
// [REQ:P0-001a] Responsive Pane Grid Layout - count display
// [REQ:P0-008b] Session Status and Controls - session count display
describe("pluralize", () => {
    it("returns singular for count 1", () => {
        expect(pluralize(1, "terminal")).toBe("terminal");
    });
    it("returns plural for count 0", () => {
        expect(pluralize(0, "terminal")).toBe("terminals");
    });
    it("returns plural for count > 1", () => {
        expect(pluralize(3, "session")).toBe("sessions");
    });
    it("works with different words", () => {
        expect(pluralize(2, "pane")).toBe("panes");
        expect(pluralize(1, "pane")).toBe("pane");
    });
});
