import { fireEvent, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders as render } from "../../../test-utils";
import { previewProps, previewModel } from "../../../test-utils/filePreview";
import { FontPreview } from "../renderers/FontPreview";
import { useFontPreview } from "../useFontPreview";
import { strings } from "../../../consts/strings";
vi.mock("../useFontPreview");
function glyphButton() { const button = screen.getByText("U+0041").closest("button"); if (!button) throw Error("Missing glyph button"); return button; }
const props = () => previewProps({ model: previewModel({ kind: "font", basename: "face.ttf", resolvedPath: "/face.ttf" }) });
const state = () => ({ error: null, loaded: { family: "wc-preview-font-1", info: { family: "Specimen", style: "Regular", format: "ttf" as const, glyphs: 200, unitsPerEm: 1000, axes: [{ tag: "wght", name: "Weight", min: 100, max: 900, default: 400 }], characters: Array.from({ length: 150 }, (_, i) => 65 + i) } } });
beforeEach(() => { vi.clearAllMocks(); vi.mocked(useFontPreview).mockReturnValue(state()); });
describe("font specimen inspector", () => {
  it("keeps the editable specimen unframed and scopes the loaded font to content", () => {
    render(<FontPreview {...props()} />); const c = strings.fontPreview;
    const sample = screen.getByRole("textbox", { name: c.sample }); expect(sample).toHaveStyle({ fontFamily: '"wc-preview-font-1", sans-serif', fontSynthesis: "none", fontSize: "48px" }); expect(screen.getByTestId("font-preview-specimen").className).not.toMatch(/rounded|border/);
    fireEvent.change(sample, { target: { value: "My sample" } }); expect(sample).toHaveValue("My sample"); fireEvent.click(screen.getByRole("button", { name: c.larger })); expect(sample).toHaveStyle({ fontSize: "52px" });
    expect(screen.getByText(c.title)).not.toHaveStyle({ fontFamily: '"wc-preview-font-1", sans-serif' });
  });
  it("browses mapped characters, resets pagination on search and inserts a selected character", () => {
    render(<FontPreview {...props()} />); const c = strings.fontPreview;
    fireEvent.change(screen.getByRole("combobox", { name: c.mode }), { target: { value: "characters" } }); expect(glyphButton()).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: c.next })); expect(screen.queryByText("U+0041")).not.toBeInTheDocument();
    fireEvent.change(screen.getByRole("searchbox", { name: c.search }), { target: { value: "U+0041" } }); fireEvent.click(glyphButton()); fireEvent.click(screen.getByRole("button", { name: c.useCharacter })); expect(screen.getByRole<HTMLTextAreaElement>("textbox", { name: c.sample }).value.endsWith("A")).toBe(true);
  });
  it("applies axes/appearance and resets all specimen settings", () => {
    render(<FontPreview {...props()} />); const c = strings.fontPreview; fireEvent.click(screen.getByRole("button", { name: c.options }));
    fireEvent.change(screen.getByRole("slider", { name: "Weight" }), { target: { value: "700" } }); fireEvent.click(screen.getByRole("radio", { name: c.light })); expect(screen.getByTestId("font-preview-sample")).toHaveStyle({ fontVariationSettings: '"wght" 700' });
    fireEvent.click(screen.getByRole("button", { name: c.reset })); expect(screen.getByTestId("font-preview-sample")).toHaveStyle({ fontVariationSettings: '"wght" 400' });
  });
  it("shows mapped private-use characters for an icon font instead of a fallback pangram", () => {
    const value = state(); value.loaded.info.characters = [0xe000, 0xe001]; vi.mocked(useFontPreview).mockReturnValue(value);
    render(<FontPreview {...props()} />); expect(screen.getByRole("textbox", { name: strings.fontPreview.sample })).toHaveValue("\ue000 \ue001");
  });
  it("preserves whole Unicode characters at the sample length limit", () => {
    const value = state(); value.loaded.info.characters = [0x1f600]; vi.mocked(useFontPreview).mockReturnValue(value);
    render(<FontPreview {...props()} />); const c = strings.fontPreview;
    fireEvent.change(screen.getByRole("textbox", { name: c.sample }), { target: { value: "X".repeat(999) } });
    fireEvent.change(screen.getByRole("combobox", { name: c.mode }), { target: { value: "characters" } });
    const button = screen.getByText("U+1F600").closest("button"); if (!button) throw Error("Missing astral glyph"); fireEvent.click(button);
    expect(screen.getByRole("button", { name: c.useCharacter })).toBeDisabled();
  });
  it("resets user text and mode on new preview identity", () => {
    const { rerender } = render(<FontPreview {...props()} />); const c = strings.fontPreview; fireEvent.change(screen.getByRole("textbox", { name: c.sample }), { target: { value: "Custom" } }); rerender(<FontPreview {...props()} model={{ ...props().model, previewId: "next" }} />); expect(screen.getByRole("textbox", { name: c.sample })).toHaveValue(c.sampleDefault);
  });
  it("preserves source renewal and original download on failure", () => {
    vi.mocked(useFontPreview).mockReturnValue({ loaded: null, error: "fetch" }); const onReopen = vi.fn(); render(<FontPreview {...props()} onReopen={onReopen} />); expect(screen.getByText(strings.fontPreview.fetch)).toBeInTheDocument(); fireEvent.click(screen.getByRole("button", { name: strings.messagesFileViewer.reopen })); expect(onReopen).toHaveBeenCalledOnce(); expect(screen.getByTestId("file-preview-download")).toHaveAttribute("download", "face.ttf");
  });
});
