import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { renderWithProviders } from "../../test-utils";
import { ComponentPageLoadState } from "./ComponentPageLoadState";

describe("ComponentPageLoadState", () => {
  it("keeps retry inside the accessible RCL error alert", () => {
    const onRetry = vi.fn();
    renderWithProviders(
      <ComponentPageLoadState
        loading={false}
        error={new Error("The catalog service is unavailable.")}
        onRetry={onRetry}
      />,
      { withoutRouter: true },
    );

    const alert = screen.getByRole("alert");
    expect(alert).toHaveAttribute("data-rcl-alert");
    expect(alert).toHaveTextContent("workbench.component.loadError");
    expect(alert).toHaveTextContent("The catalog service is unavailable.");
    fireEvent.click(screen.getByRole("button"));
    expect(onRetry).toHaveBeenCalledOnce();
  });
});
