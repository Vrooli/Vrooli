import { describe, expect, it, vi } from "vitest";
import { create } from "@bufbuild/protobuf";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { AskAttachmentSchema } from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";

import { renderWithProviders } from "../../test-utils";
import { isLocalFileReference, renderDecisionText, type DecisionLinks } from "./decisionLinks";

// Words the sender wrote, supplied by each test and read back from the page.
const GUIDE_LABEL = "the guide";
const PLAN_PATH = "/work/plan.md";
const OTHER_PATH = "/work/other.md";
const plan = create(AskAttachmentSchema, { name: "plan.md", sourcePath: "/work/plan.md" });

const show = (text: string, links?: DecisionLinks) => renderWithProviders(<p>{renderDecisionText(text, links)}</p>);

describe("isLocalFileReference", () => {
  it("recognizes a local file path and rejects prose or web URLs", () => {
    expect(isLocalFileReference("~/notes/todo.md")).toBe(true);
    expect(isLocalFileReference("https://example.test/doc.md")).toBe(false);
    expect(isLocalFileReference("two words.md")).toBe(false);
    expect(isLocalFileReference("plain")).toBe(false);
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

  it("opens an unattached local path in the Hub viewer", async () => {
    const onOpenLinkedArtifact = vi.fn();
    show(`Open ${OTHER_PATH} now`, { onOpenLinkedArtifact, openLinkedArtifactTitle: "Open here" });
    await userEvent.setup().click(screen.getByRole("button", { name: OTHER_PATH }));
    expect(onOpenLinkedArtifact).toHaveBeenCalledWith(OTHER_PATH);
  });

  it("keeps a local path readable as a literal when no Hub opener is provided", () => {
    const { unmount } = show(`Open ${OTHER_PATH} now`);
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
    expect(screen.getByText(OTHER_PATH).tagName).toBe("CODE");
    unmount();
  });

  it("opens a path that names one of the ask's attachments in the Hub's own viewer", async () => {
    const onOpenAttachment = vi.fn();
    show(`Review \`${PLAN_PATH}\` please`, { attachments: [plan], onOpenAttachment, openAttachmentTitle: "Open here" });
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
