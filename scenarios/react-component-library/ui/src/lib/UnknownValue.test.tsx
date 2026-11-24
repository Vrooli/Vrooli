import { describe, expect, it } from "vitest";
import { UnknownValue } from "../../../library/components/UnknownValue/versions/0.1.0/UnknownValue";
import { renderWithProviders } from "../test-utils/renderWithProviders";

describe("UnknownValue inline fallback", () => {
  it("keeps a visible label, inline reason and tooltip on a phrasing span", () => {
    renderWithProviders(<p>Mode: <UnknownValue label="Unknown" reason="No current setting is available." /></p>);
    const value = document.querySelector("[data-rcl-unknown-value]");
    expect(value).toHaveTextContent("Unknown · No current setting is available.");
    expect(value).not.toBeNull();
    if (!value) throw new Error("UnknownValue should render its data marker");
    expect(value.tagName).toBe("SPAN");
    expect(value).toHaveAttribute("title", "No current setting is available.");
    expect(value).toHaveAttribute("data-rcl-unknown-value", "");
  });

  it("omits reason punctuation and tooltip when the cause is not known", () => {
    renderWithProviders(<UnknownValue label="Unknown" />);
    const value = document.querySelector("[data-rcl-unknown-value]");
    expect(value).toHaveTextContent("Unknown");
    expect(value).not.toBeNull();
    if (!value) throw new Error("UnknownValue should render its data marker");
    expect(value).not.toHaveAttribute("title");
    expect(value.querySelector("[data-rcl-unknown-value-reason]")).toBeNull();
  });
});
