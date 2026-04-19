import { describe, expect, it } from "vitest";
import { render } from "@testing-library/react";
import { DashboardLayout } from "./DashboardLayout";

describe("DashboardLayout", () => {
  it("applies different data-theme attributes per themeKey", () => {
    const { container: groundControl } = render(
      <DashboardLayout themeKey="ground-control" title="Mission Control">
        <div>content</div>
      </DashboardLayout>,
    );
    const { container: cosmos } = render(
      <DashboardLayout themeKey="cosmos" title="Panorama">
        <div>content</div>
      </DashboardLayout>,
    );

    const groundLayout = groundControl.querySelector<HTMLElement>(".cc-layout");
    const cosmosLayout = cosmos.querySelector<HTMLElement>(".cc-layout");

    expect(groundLayout?.getAttribute("data-theme")).toBe("ground-control");
    expect(cosmosLayout?.getAttribute("data-theme")).toBe("cosmos");
    expect(groundLayout?.getAttribute("data-theme")).not.toBe(
      cosmosLayout?.getAttribute("data-theme"),
    );
  });

  it("renders the title inside the header", () => {
    const { getByText } = render(
      <DashboardLayout themeKey="foundry" title="The Forge">
        <p>body</p>
      </DashboardLayout>,
    );
    expect(getByText("The Forge")).toBeInTheDocument();
  });

  it("renders the optional aside region when provided", () => {
    const { container } = render(
      <DashboardLayout themeKey="vault" title="Ledger" aside={<span>aside-node</span>}>
        <p>body</p>
      </DashboardLayout>,
    );
    expect(container.querySelector("aside")).not.toBeNull();
  });

  it("wraps the aside in an open metrics drawer for mobile collapse", () => {
    const { getByTestId, getByText } = render(
      <DashboardLayout themeKey="vault" title="Ledger" aside={<span>aside-node</span>}>
        <p>body</p>
      </DashboardLayout>,
    );
    const drawer = getByTestId("metrics-drawer") as HTMLDetailsElement;
    expect(drawer.tagName).toBe("DETAILS");
    expect(drawer.open).toBe(true);
    expect(getByText("Metrics")).toBeInTheDocument();
    expect(getByText("aside-node")).toBeInTheDocument();
  });

  it("omits the metrics drawer when no aside is provided", () => {
    const { queryByTestId, container } = render(
      <DashboardLayout themeKey="cosmos" title="Panorama">
        <p>body</p>
      </DashboardLayout>,
    );
    expect(queryByTestId("metrics-drawer")).toBeNull();
    expect(container.querySelector("aside")).toBeNull();
  });
});
