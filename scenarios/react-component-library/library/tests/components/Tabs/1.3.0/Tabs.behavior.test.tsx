import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { Tabs } from "@vrooli/react-component-library/Tabs/1.3.0";

describe("Tabs current release contract", () => {
  it("names the strip root with the catalog id by default", () => {
    render(<Tabs items={["one", "two"]} ariaLabel="Sections" />);
    expect(screen.getByTestId("navigation.tabs")).toBeInTheDocument();
  });

  // The root id was hardcoded, so two strips mounted at once — a dialog's over
  // a sidebar's — resolved to the same selector and every root-level query for
  // either became ambiguous.
  it("lets two strips on one surface be told apart", () => {
    render(
      <>
        <Tabs items={["one", "two"]} testId="sidebar-origin-tabs" ariaLabel="Origins" />
        <Tabs items={["a", "b"]} testId="launcher-mode-tabs" ariaLabel="Modes" />
      </>,
    );
    expect(screen.getByTestId("sidebar-origin-tabs")).toBeInTheDocument();
    expect(screen.getByTestId("launcher-mode-tabs")).toBeInTheDocument();
    expect(screen.queryByTestId("navigation.tabs")).toBeNull();
  });
});
