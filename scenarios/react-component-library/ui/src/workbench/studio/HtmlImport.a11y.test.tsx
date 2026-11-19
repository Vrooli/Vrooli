import { cleanup, fireEvent, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { setLocale } from "../../i18n";
import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { HtmlImport } from "./HtmlImport";

const fixture = `<!doctype html>
<html>
  <head><title>Landing page</title></head>
  <body>
    <main>
      <section>
        <h1>Welcome</h1>
        <p>Reference copy</p>
        <img src="https://example.test/hero.png" alt="Hero">
        <button>Start</button>
      </section>
      <footer><small>Fine print</small></footer>
    </main>
  </body>
</html>`;

function renderDialog(onImport = vi.fn(), onClose = vi.fn()) {
  const result = renderWithProviders(<HtmlImport onImport={onImport} onClose={onClose} />);
  return { ...result, onImport, onClose };
}

async function chooseFixture(name = "landing.html", source = fixture): Promise<void> {
  const file = new File([source], name, { type: "text/html" });
  fireEvent.change(screen.getByTestId("html-import-file"), { target: { files: [file] } });
  await screen.findByTestId("html-import-counts");
}

beforeEach(async () => {
  await setLocale("en");
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("HtmlImport accessibility", () => {
  it("exposes a modal dialog with a labelled file input in the empty state", async () => {
    const { container } = renderDialog();

    const dialog = screen.getByTestId("html-import");
    expect(dialog).toHaveAttribute("role", "dialog");
    expect(dialog).toHaveAttribute("aria-modal", "true");
    expect(dialog).toHaveAccessibleName("Import an HTML page");

    const input = screen.getByTestId("html-import-file");
    expect(input).toHaveAttribute("type", "file");
    expect(input).toHaveAccessibleName("HTML file");
    expect(screen.getByTestId("html-import-import")).toBeDisabled();
    expect(dialog.contains(document.activeElement)).toBe(true);

    await expectNoA11yViolations(container);
  });

  it("closes on Escape", () => {
    const { onClose } = renderDialog();
    fireEvent.keyDown(document, { key: "Escape" });
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("renders a parsed document, warnings and the region chooser without axe violations", async () => {
    const { container, onImport } = renderDialog();
    await chooseFixture();

    expect(screen.getByTestId("html-import-title")).toHaveTextContent("Landing page");
    expect(screen.getByTestId("html-import-counts")).toHaveTextContent(/layers/);
    expect(screen.getByTestId("html-import-warnings")).toHaveTextContent(/image was skipped/);
    expect(screen.getByTestId("html-import-import")).toBeEnabled();
    expect(screen.getByTestId("html-import-region-whole")).toHaveAttribute("aria-pressed", "true");

    const section = screen
      .getAllByTestId(/^html-import-region-/)
      .find((button) => button.textContent.includes("section"));
    if (!section) throw new Error("Expected a section region");
    fireEvent.click(section);
    expect(section).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByTestId("html-import-region-whole")).toHaveAttribute("aria-pressed", "false");

    fireEvent.click(screen.getByTestId("html-import-import"));
    expect(onImport).toHaveBeenCalledTimes(1);
    const [template, name] = onImport.mock.calls[0] ?? [];
    expect(template?.spec.type).toBe("page");
    expect(name).toBe("Landing page");

    await expectNoA11yViolations(container);
  });

  it("reports an oversized file as an alert and keeps import disabled", async () => {
    const { container } = renderDialog();
    const file = new File(["<p>too big</p>"], "huge.html", { type: "text/html" });
    Object.defineProperty(file, "size", { configurable: true, value: 3 * 1024 * 1024 });
    fireEvent.change(screen.getByTestId("html-import-file"), { target: { files: [file] } });

    expect(await screen.findByTestId("html-import-error")).toHaveAttribute("role", "alert");
    expect(screen.getByTestId("html-import-import")).toBeDisabled();
    expect(screen.getByTestId("html-import-error")).toHaveTextContent(/2 MiB/);

    await expectNoA11yViolations(container);
  });
});
