import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import ApprovalLinkedText from "./ApprovalLinkedText";
import { renderWithProviders } from "../../test-utils";

describe("ApprovalLinkedText", () => {
  it("renders hub markdown links as usable, safe links", () => {
    renderWithProviders(<ApprovalLinkedText content={'Review [the run](https://example.com/run) or javascript:[unsafe](javascript:alert(1)).'} />);

    const link = screen.getByRole("link", { name: "the run" });
    expect(link).toHaveAttribute("href", "https://example.com/run");
    expect(link).toHaveAttribute("target", "_blank");
    expect(link).toHaveAttribute("rel", "noopener noreferrer");
    expect(screen.queryByRole("link", { name: "unsafe" })).not.toBeInTheDocument();
    expect(screen.getByTestId("approval-linked-text")).toHaveTextContent("unsafe");
  });
});
