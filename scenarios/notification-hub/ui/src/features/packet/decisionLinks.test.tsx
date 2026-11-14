import { describe, expect, it, vi } from "vitest";
import { create } from "@bufbuild/protobuf";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { AskAttachmentSchema } from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";

import { renderWithProviders } from "../../test-utils";
import { fileReferenceHref, hasLocalFileReference, renderDecisionText, type DecisionLinks } from "./decisionLinks";

const CONSOLE = "https://console.example.test/files";
// Words the sender wrote, supplied by each test and read back from the page.
const GUIDE_LABEL = "the guide";
const PLAN_PATH = "/work/plan.md";
const OTHER_PATH = "/work/other.md";
const plan = create(AskAttachmentSchema, { name: "plan.md", sourcePath: "/work/plan.md" });

const show = (text: string, links?: DecisionLinks) => renderWithProviders(<p>{renderDecisionText(text, links)}</p>);

describe("fileReferenceHref", () => {
  it("opens a local path in Web Console's preview with the path as ?file=", () => {
    expect(fileReferenceHref("/work/plan.md", CONSOLE)).toBe(`${CONSOLE}?file=%2Fwork%2Fplan.md`);
    expect(fileReferenceHref("src/app/main.ts:42,", CONSOLE)).toContain("file=src%2Fapp%2Fmain.ts%3A42");
  });

  it("makes no link without Web Console, or for something that is not a file path", () => {
    expect(fileReferenceHref("/work/plan.md", "")).toBeNull();
    expect(fileReferenceHref("https://example.test/a.md", CONSOLE)).toBeNull();
    expect(fileReferenceHref("two words.md", CONSOLE)).toBeNull();
    expect(fileReferenceHref("example.com", CONSOLE)).toBeNull();
    expect(fileReferenceHref("plain", CONSOLE)).toBeNull();
  });

  it("refuses a Web Console address that is not http(s)", () => {
    expect(fileReferenceHref("/work/plan.md", "javascript:alert(1)")).toBeNull();
    expect(fileReferenceHref("/work/plan.md", "not a url")).toBeNull();
  });
});

describe("hasLocalFileReference", () => {
  it("is true for a local path the Hub cannot open itself", () => {
    expect(hasLocalFileReference("See ~/notes/todo.md first")).toBe(true);
  });

  it("is false for prose, web links, and paths that are one of the ask's own attachments", () => {
    expect(hasLocalFileReference("Nothing to open here")).toBe(false);
    expect(hasLocalFileReference("Read https://example.test/doc.md")).toBe(false);
    expect(hasLocalFileReference("Read /work/plan.md", [plan])).toBe(false);
  });
});

describe("renderDecisionText", () => {
  it("links web addresses and Markdown links to the web, in a new tab without opener access", () => {
    show(`Docs at https://example.test/a and [${GUIDE_LABEL}](https://example.test/guide).`);
    const links = screen.getAllByRole("link");
    expect(links.map((link) => link.getAttribute("href"))).toEqual(["https://example.test/a", "https://example.test/guide"]);
    for (const link of links) {
      expect(link).toHaveAttribute("target", "_blank");
      expect(link).toHaveAttribute("rel", "noopener noreferrer");
    }
    expect(screen.getByRole("link", { name: GUIDE_LABEL })).toBeInTheDocument();
  });

  it("never turns a script or data address into a link", () => {
    show("[click](javascript:alert(1)) and [file](data:text/html,hi)");
    expect(screen.queryByRole("link")).not.toBeInTheDocument();
  });

  it("shows a local path as a literal, linking it only when Web Console is available", () => {
    const { unmount } = show(`Open ${OTHER_PATH} now`);
    expect(screen.queryByRole("link")).not.toBeInTheDocument();
    expect(screen.getByText(OTHER_PATH).tagName).toBe("CODE");
    unmount();
    show(`Open ${OTHER_PATH} now`, { webConsoleUrl: CONSOLE, webConsoleTitle: "Open in Web Console" });
    const link = screen.getByRole("link");
    expect(link).toHaveAttribute("href", `${CONSOLE}?file=%2Fwork%2Fother.md`);
    expect(link).toHaveAttribute("title", "Open in Web Console");
  });

  it("opens a path that names one of the ask's attachments in the Hub's own viewer", async () => {
    const onOpenAttachment = vi.fn();
    show(`Review \`${PLAN_PATH}\` please`, { attachments: [plan], onOpenAttachment, webConsoleUrl: CONSOLE, openAttachmentTitle: "Open here" });
    expect(screen.queryByRole("link")).not.toBeInTheDocument();
    await userEvent.setup().click(screen.getByRole("button", { name: PLAN_PATH }));
    expect(onOpenAttachment).toHaveBeenCalledWith({ name: "plan.md", path: "" });
    expect(screen.getByRole("button")).toHaveAttribute("title", "Open here");
  });

  it("shows an embedded PNG, JPEG or WEBP image inline with its description", () => {
    show("Result: ![chart](data:image/png;base64,iVBORw0KGgo=) done");
    const image = screen.getByRole("img", { name: "chart" });
    expect(image).toHaveAttribute("src", "data:image/png;base64,iVBORw0KGgo=");
    expect(screen.queryByRole("link")).not.toBeInTheDocument();
  });

  it("does not embed other image types, so a vector or script payload stays text", () => {
    show("![x](data:image/svg+xml;base64,PHN2Zz4=)");
    expect(screen.queryByRole("img")).not.toBeInTheDocument();
  });

  it("lays links out at the surrounding line height in dense mode, and as touch targets otherwise", () => {
    const { unmount } = show("https://example.test/a");
    expect(screen.getByRole("link")).toHaveClass("min-h-11");
    unmount();
    show("https://example.test/a", { dense: true });
    expect(screen.getByRole("link")).not.toHaveClass("min-h-11");
  });

  it("returns plain text, and only text, when there is nothing to link", () => {
    expect(renderDecisionText("Just words, no links.")).toEqual(["Just words, no links."]);
  });
});
