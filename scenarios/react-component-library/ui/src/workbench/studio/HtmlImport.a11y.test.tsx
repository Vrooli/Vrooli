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

async function chooseFixture(name = "landing.html", source = fixture, css?: string): Promise<void> {
  const files = [new File([source], name, { type: "text/html" })];
  if (css) files.push(new File([css], "screens.css", { type: "text/css" }));
  fireEvent.change(screen.getByTestId("html-import-file"), { target: { files } });
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
    await chooseFixture(
      "landing.html",
      fixture,
      ".device { color: red; background-image: url(https://remote.test/bg.png); }",
    );

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
    expect(template?.spec.props.sourceHtml).toContain("color: red");
    expect(template?.spec.props.sourceHtml).toContain("Content-Security-Policy");
    expect(template?.spec.props.sourceHtml).not.toContain("remote.test");
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

  it("filters a dense bundle to a named screen and recovers from no matches", async () => {
    const screens = Array.from(
      { length: 8 },
      (_, index) =>
        `<article data-name="${index === 5 ? "money dark" : `screen ${index + 1}`}"><h1>Screen ${index + 1}</h1><p>Unique copy ${index + 1}</p><button>Continue ${index + 1}</button></article>`,
    ).join("");
    renderDialog();
    await chooseFixture(
      "screens.html",
      `<!doctype html><html><head><title>Bundle</title></head><body>${screens}</body></html>`,
    );

    const search = screen.getByTestId("html-import-region-search");
    const totalRegions = screen.getAllByTestId(/^html-import-region-region-/).length;
    expect(search).toHaveAccessibleName("Search regions");
    expect(screen.getByTestId("html-import-region-results")).toHaveTextContent(
      /Showing \d+ of \d+ regions/,
    );
    fireEvent.change(search, { target: { value: "money dark" } });
    const namedScreen = screen
      .getAllByTestId(/^html-import-region-/)
      .find((button) => button.textContent.toLowerCase().includes("money dark"));
    if (!namedScreen) throw new Error("Expected the named screen region to match the search.");
    expect(screen.getByTestId("html-import-region-results")).toHaveTextContent("Showing 1 of");
    fireEvent.click(namedScreen);
    expect(namedScreen).toHaveAttribute("aria-pressed", "true");

    fireEvent.change(search, { target: { value: "nothing matches" } });
    expect(screen.getByTestId("html-import-region-empty")).toHaveTextContent(
      /No regions match this search/,
    );
    expect(screen.getByTestId("html-import-region-results")).toHaveTextContent("Showing 0 of");
    fireEvent.change(search, { target: { value: "" } });
    expect(screen.queryByTestId("html-import-region-empty")).not.toBeInTheDocument();
    expect(screen.getByTestId("html-import-region-results")).toHaveTextContent(
      `Showing ${totalRegions} of ${totalRegions} regions`,
    );
    expect(screen.getAllByTestId(/^html-import-region-region-/)).toHaveLength(totalRegions);
  });
});
