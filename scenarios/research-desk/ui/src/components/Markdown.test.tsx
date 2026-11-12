import { cleanup, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

import { renderWithProviders, stubWire } from "../test-utils";
import { Markdown } from "./Markdown";

const HOME = "/home/u/.vrooli/research";
const UNSAFE_LABEL = "secret";

afterEach(() => cleanup());

function show(content: string, baseDir?: string) {
  stubWire((path) => (path.endsWith("/GetOverview") ? { researchHome: HOME } : {}));
  return renderWithProviders(<Markdown content={content} baseDir={baseDir} />);
}

describe("Markdown", () => {
  it("renders the block structure research files use", () => {
    const { container } = show(
      [
        "# Mission",
        "Plain **bold** and *soft* with `code`.",
        "",
        "- one",
        "  - nested",
        "- two",
        "",
        "1. first",
        "2. second",
        "",
        "> quoted",
        "",
        "---",
        "",
        "```",
        "| not a table |",
        "```",
        "",
        "| Gate | Value |",
        "| --- | --- |",
        "| charter | 9 KB |",
      ].join("\r\n"),
    );
    // A file's own top heading sits under the page heading.
    expect(screen.getByRole("heading", { level: 2, name: "Mission" })).toBeInTheDocument();
    expect(container.querySelector("strong")?.textContent).toBe("bold");
    expect(container.querySelector("em")?.textContent).toBe("soft");
    expect(container.querySelector("ul ul li")?.textContent).toBe("nested");
    expect(container.querySelectorAll("ol > li")).toHaveLength(2);
    expect(container.querySelector("blockquote")?.textContent).toBe("quoted");
    expect(container.querySelector("hr")).not.toBeNull();
    expect(container.querySelector("pre code")?.textContent).toBe("| not a table |");
    // Wide tables scroll inside their own box instead of widening the page.
    expect(container.querySelector(".rd-table-scroll table td")?.textContent).toBe("charter");
  });

  it("links home paths to the file page, opens the web in a new tab, and leaves other targets as text", async () => {
    show(`See [the state](STATE.md), [mission](${HOME}/topics/t/MISSION.md), [docs](https://example.org/a) and [${UNSAFE_LABEL}](/etc/passwd).`, `${HOME}/cohorts/c1`);
    await waitFor(() => expect(screen.getByRole("link", { name: "the state" })).toHaveAttribute("href", "/files/cohorts/c1/STATE.md"));
    expect(screen.getByRole("link", { name: "mission" })).toHaveAttribute("href", "/files/topics/t/MISSION.md");
    const web = screen.getByRole("link", { name: "docs" });
    expect(web).toHaveAttribute("target", "_blank");
    expect(web).toHaveAttribute("rel", expect.stringContaining("noopener"));
    expect(screen.queryByRole("link", { name: UNSAFE_LABEL })).not.toBeInTheDocument();
    expect(screen.getByText(UNSAFE_LABEL)).toHaveClass("rd-dead-link");
  });

  it("resolves a relative path out of its folder and links bare web addresses", async () => {
    show("Back to [the topic](../../topics/t/MISSION.md) or https://example.org/b", `${HOME}/cohorts/c1`);
    await waitFor(() => expect(screen.getByRole("link", { name: "the topic" })).toHaveAttribute("href", "/files/topics/t/MISSION.md"));
    expect(screen.getByRole("link", { name: "https://example.org/b" })).toHaveAttribute("href", "https://example.org/b");
  });
});
