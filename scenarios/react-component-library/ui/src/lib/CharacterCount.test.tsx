import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { CharacterCount } from "../../../library/components/CharacterCount/versions/0.1.0/CharacterCount";
import { renderWithProviders } from "../test-utils/renderWithProviders";

describe("CharacterCount feedback", () => {
  it("exposes live help and a limit message on one associated output", () => {
    renderWithProviders(<CharacterCount id="entry-count" count="642 / 500 characters" overLimit message="Shorten this entry." />);
    const output = screen.getByRole("status");
    expect(output.tagName).toBe("OUTPUT");
    expect(output).toHaveAttribute("id", "entry-count");
    expect(output).toHaveAttribute("aria-live", "polite");
    expect(output).toHaveAttribute("data-exceeded");
    expect(output).toHaveTextContent("642 / 500 characters");
    expect(output).toHaveTextContent("Shorten this entry.");
  });

  it("omits the limit warning while the user is within the limit", () => {
    renderWithProviders(<CharacterCount count="214 / 500 characters" />);
    const output = screen.getByRole("status");
    expect(output).not.toHaveAttribute("data-exceeded");
    expect(output.querySelector("[data-rcl-character-count-message]")).toBeNull();
  });
});
