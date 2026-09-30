import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ConceptExplainerDialog, type ConceptExplainerSection } from "./concept-explainer-dialog";
import { selectors } from "../../consts/selectors";

const SECTIONS: ConceptExplainerSection[] = [
  { heading: "Group A", label: "alpha", body: "Alpha body" },
  { heading: "Group A", label: "beta", body: "Beta body" },
  { heading: "Group B", label: "gamma", body: "Gamma body" },
];

describe("ConceptExplainerDialog", () => {
  it("renders nothing when closed", () => {
    render(
      <ConceptExplainerDialog
        isOpen={false}
        onClose={() => {}}
        title="Scope"
        sections={SECTIONS}
      />,
    );
    expect(screen.queryByTestId(selectors.goalDetails.conceptExplainerDialog)).not.toBeInTheDocument();
  });

  it("renders title, intro, and sections grouped by heading when open", () => {
    render(
      <ConceptExplainerDialog
        isOpen
        onClose={() => {}}
        title="Scope"
        intro="Scope determines the unit of work."
        sections={SECTIONS}
      />,
    );
    expect(screen.getByTestId(selectors.goalDetails.conceptExplainerDialog)).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Scope" })).toBeInTheDocument();
    expect(screen.getByText("Scope determines the unit of work.")).toBeInTheDocument();
    expect(screen.getByText("Group A")).toBeInTheDocument();
    expect(screen.getByText("Group B")).toBeInTheDocument();
    expect(screen.getByText("alpha")).toBeInTheDocument();
    expect(screen.getByText("Alpha body")).toBeInTheDocument();
    expect(screen.getByText("beta")).toBeInTheDocument();
    expect(screen.getByText("gamma")).toBeInTheDocument();
  });

  it("renders an optional canonical documentation link", () => {
    render(
      <ConceptExplainerDialog
        isOpen
        onClose={() => {}}
        title="Scope"
        intro="Scope determines the unit of work."
        docLink={{ href: "/docs/example.md", label: "Read the canonical doc" }}
        sections={SECTIONS}
      />,
    );
    expect(screen.getByRole("link", { name: "Read the canonical doc" })).toHaveAttribute(
      "href",
      "/docs/example.md",
    );
  });

  it("calls onClose when the close button is clicked", async () => {
    const onClose = vi.fn();
    render(
      <ConceptExplainerDialog
        isOpen
        onClose={onClose}
        title="Scope"
        sections={SECTIONS}
      />,
    );
    await userEvent.click(screen.getByLabelText("Close dialog"));
    expect(onClose).toHaveBeenCalled();
  });

  it("honors a custom testId so wrappers can preserve legacy selectors", () => {
    render(
      <ConceptExplainerDialog
        isOpen
        onClose={() => {}}
        title="Phase graph"
        sections={SECTIONS}
        testId="phase-graph-glossary-dialog"
      />,
    );
    expect(screen.getByTestId("phase-graph-glossary-dialog")).toBeInTheDocument();
  });
});
