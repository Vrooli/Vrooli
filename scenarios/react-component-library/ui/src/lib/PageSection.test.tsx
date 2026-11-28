import { describe, expect, it } from "vitest";
import { screen } from "@testing-library/react";
import { PageSection } from "../../../library/components/page-section/versions/0.1.0/PageSectionDraftSource";
import { renderWithProviders } from "../test-utils/renderWithProviders";

describe("PageSection semantics", () => {
  it("associates an optional id with a semantic h2 and leaves children unwrapped", () => {
    renderWithProviders(
      <PageSection id="overview" title="Overview" note="Current state" actions={<button>Refresh</button>}>
        <article data-testid="body">Page content</article>
      </PageSection>,
    );

    const section = document.querySelector("[data-rcl-page-section]");
    const heading = section?.querySelector("h2");
    expect(section).toHaveAttribute("id", "overview");
    expect(section).toHaveAttribute("aria-labelledby", "overview-title");
    expect(heading).toHaveAttribute("id", "overview-title");
    expect(heading).toHaveTextContent("Overview");
    expect(section).toContainElement(screen.getByTestId("body"));
    expect(section).toHaveTextContent("Current state");
    expect(section).toHaveTextContent("Refresh");
  });

  it("leaves the section unlabelled when there is no id and omits an empty note", () => {
    renderWithProviders(<PageSection title={<span>Heading node</span>}>Plain content</PageSection>);

    const section = document.querySelector("[data-rcl-page-section]");
    expect(section).not.toHaveAttribute("aria-labelledby");
    expect(section?.querySelector("h2")).toHaveTextContent("Heading node");
    expect(section?.querySelector("[data-rcl-page-section-note]")).toBeNull();
    expect(section).toHaveTextContent("Plain content");
  });
});
