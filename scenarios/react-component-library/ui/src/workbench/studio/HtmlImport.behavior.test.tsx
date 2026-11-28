import { cleanup, fireEvent, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { setLocale } from "../../i18n";
import { renderWithProviders } from "../../test-utils";
import { HtmlImport } from "./HtmlImport";

const renderDialog = (onImport = vi.fn(), onClose = vi.fn()) => {
  renderWithProviders(<HtmlImport onImport={onImport} onClose={onClose} />);
  return { onImport, onClose };
};

beforeEach(async () => setLocale("en"));
afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

describe("HtmlImport source and recovery behavior", () => {
  it("rejects a selection without HTML, then recovers when a valid file is selected", async () => {
    renderDialog();
    const input = screen.getByTestId("html-import-file");

    fireEvent.change(input, {
      target: { files: [new File(["body {}"], "styles.css", { type: "text/css" })] },
    });
    expect(await screen.findByTestId("html-import-error")).toHaveTextContent(
      "includes an HTML file",
    );
    expect(screen.getByTestId("html-import-import")).toBeDisabled();

    fireEvent.change(input, {
      target: {
        files: [new File(["<main><h1>Recovered</h1></main>"], "recovered.html")],
      },
    });
    expect(await screen.findByTestId("html-import-title")).toHaveTextContent("recovered");
    expect(screen.getByTestId("html-import-import")).toBeEnabled();
  });

  it("uses index.html from a bundle, applies the selected region and trimmed name, and closes after import", async () => {
    const { onImport, onClose } = renderDialog();
    const html = (name: string, contents: string) => new File([contents], name);
    const files = [
      html("secondary.html", "<main><h1>Secondary page</h1></main>"),
      html("theme.css", "h1 { color: #123456; }"),
      html("index.html", "<main><section><h1>Primary page</h1><p>Chosen copy</p></section></main>"),
    ];

    fireEvent.change(screen.getByTestId("html-import-bundle"), { target: { files } });
    expect(await screen.findByTestId("html-import-title")).toHaveTextContent("index");
    expect(screen.getByTestId("html-import-file-meta")).toHaveTextContent("index.html");
    expect(screen.getByTestId("html-import-file-meta")).toHaveTextContent(/\d+ B$/);

    const section = screen
      .getAllByTestId(/^html-import-region-/)
      .find((button) => button.textContent.includes("section"));
    if (!section) throw new Error("Expected the section region to be available.");
    fireEvent.click(section);
    fireEvent.change(screen.getByTestId("html-import-name"), {
      target: { value: "  Selected design  " },
    });
    fireEvent.click(screen.getByTestId("html-import-import"));

    expect(onImport).toHaveBeenCalledTimes(1);
    const [template, name] = onImport.mock.calls[0] ?? [];
    expect(name).toBe("Selected design");
    expect(template?.name).toBe("Selected design");
    expect(template?.spec.props.sourceHtml).toContain("Chosen copy");
    expect(template?.spec.props.sourceHtml).toContain("color: #123456");
    expect(template?.spec.props.sourceHtml).not.toContain("Secondary page");
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("surfaces a file read failure and retries the same file successfully", async () => {
    const { onImport } = renderDialog();
    const file = new File(["<main><h1>Retry works</h1></main>"], "retry.html");
    const read = vi
      .fn<() => Promise<string>>()
      .mockRejectedValueOnce(new Error("temporary read failure"))
      .mockResolvedValueOnce("<main><h1>Retry works</h1></main>");
    Object.defineProperty(file, "text", { configurable: true, value: read });
    const input = screen.getByTestId("html-import-file");

    fireEvent.change(input, { target: { files: [file] } });
    expect(await screen.findByTestId("html-import-error")).toHaveTextContent(
      "retry.html could not be read as HTML",
    );
    expect(input).toHaveValue("");
    expect(screen.getByTestId("html-import-import")).toBeDisabled();

    fireEvent.change(input, { target: { files: [file] } });
    expect(await screen.findByTestId("html-import-title")).toHaveTextContent("retry");
    fireEvent.click(screen.getByTestId("html-import-import"));
    expect(read).toHaveBeenCalledTimes(2);
    expect(onImport).toHaveBeenCalledTimes(1);
  });

  it("keeps the import disabled for media-only markup and calls close from the backdrop", async () => {
    const { onClose, onImport } = renderDialog();
    fireEvent.change(screen.getByTestId("html-import-file"), {
      target: { files: [new File(["<main><img src='local.png' alt=''></main>"], "media.html")] },
    });

    expect(await screen.findByTestId("html-import-no-content")).toBeInTheDocument();
    expect(screen.getByTestId("html-import-import")).toBeDisabled();
    expect(onImport).not.toHaveBeenCalled();
    fireEvent.pointerDown(document.querySelector(".hi-backdrop")!);
    expect(onClose).toHaveBeenCalledTimes(1);
  });
});
