import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { Alert } from "../../../library/components/Alert/versions/1.0.7/Alert";
import { renderWithProviders } from "../test-utils/renderWithProviders";

describe("Alert description content", () => {
  it("keeps span descriptions by default and supports a block container when requested", () => {
    const description = "Details unavailable.";
    const item = "one item";
    const { rerender } = renderWithProviders(<Alert tone="warning" title="Cannot read files" description={description} />);
    expect(screen.getByText(description).tagName).toBe("SPAN");

    rerender(
      <Alert
        tone="warning"
        title="Cannot read files"
        descriptionAs="div"
        description={<><span>{description}</span><ul><li>{item}</li></ul></>}
      />,
    );
    expect(screen.getByText(description).closest("[data-rcl-alert-description]")?.tagName).toBe("DIV");
    expect(screen.getByText(item)).toBeInTheDocument();
  });
});
