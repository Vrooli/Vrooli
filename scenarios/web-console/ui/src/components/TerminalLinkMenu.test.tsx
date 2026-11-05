import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders as render } from "../test-utils";
import type { TerminalLinkActivation } from "../lib/terminalFileLinks";
import TerminalLinkMenu from "./TerminalLinkMenu";

const position = { x: 40, y: 80 };
const activation = (link: Omit<TerminalLinkActivation, "position">): TerminalLinkActivation => ({ ...link, position });

describe("TerminalLinkMenu", () => {
  it("opens a file link in the preview and closes when dismissed", () => {
    const onOpenFile = vi.fn();
    const onClose = vi.fn();
    render(
      <TerminalLinkMenu
        activation={activation({ kind: "file", value: "/srv/app/main.go", text: "main.go:12" })}
        onOpenFile={onOpenFile}
        onClose={onClose}
      />,
    );
    expect(screen.getByTestId("link-copy")).toHaveTextContent("Copy path");
    fireEvent.click(screen.getByTestId("link-open-file"));
    expect(onOpenFile).toHaveBeenCalledWith("/srv/app/main.go");
  });

  it("offers to open a URL in a new tab and keeps both ends of a long link readable", () => {
    const open = vi.spyOn(window, "open").mockImplementation(() => null);
    const url = `https://example.com/${"segment/".repeat(10)}report.html`;
    render(<TerminalLinkMenu activation={activation({ kind: "url", value: url, text: url })} onOpenFile={vi.fn()} onClose={vi.fn()} />);
    expect(screen.getByTestId("link-copy")).toHaveTextContent("Copy link");
    const title = screen.getByText(/…/);
    expect(title.textContent?.startsWith("https://")).toBe(true);
    expect(title.textContent?.endsWith("report.html")).toBe(true);
    expect(title.textContent?.length).toBeLessThanOrEqual(44);
    fireEvent.click(screen.getByTestId("link-open-url"));
    expect(open).toHaveBeenCalledWith(url, "_blank", "noopener,noreferrer");
    open.mockRestore();
  });
});
