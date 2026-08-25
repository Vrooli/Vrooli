import { cleanup, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

import { EmptyState } from "@vrooli/react-component-library/EmptyState/1.1.0";
import { renderWithProviders } from "../../test-utils";

const declarationGuidance = "Declare the first household token.";
const tokenIconLabel = "token icon";

describe("EmptyState", () => {
  afterEach(cleanup);

  it("renders optional guidance, icon, action, and caller styling", () => {
    renderWithProviders(
      <EmptyState
        title="No token types"
        description={declarationGuidance}
        icon={<span aria-label={tokenIconLabel}>◎</span>}
        action={<button type="button">Declare token</button>}
        className="custom-empty"
      />,
    );

    expect(screen.getByRole("heading", { name: "No token types" })).toBeInTheDocument();
    expect(screen.getByText(declarationGuidance)).toBeInTheDocument();
    expect(screen.getByLabelText(tokenIconLabel)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Declare token" })).toBeInTheDocument();
    expect(screen.getByRole("heading").parentElement?.parentElement).toHaveClass("custom-empty");
  });

  it("omits optional regions when no supporting content is supplied", () => {
    renderWithProviders(<EmptyState title="Nothing here" />);
    expect(screen.getByRole("heading", { name: "Nothing here" })).toBeInTheDocument();
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
  });
});
