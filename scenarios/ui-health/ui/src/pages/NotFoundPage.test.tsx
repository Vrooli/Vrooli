import { describe, it, expect } from "vitest";
import { screen } from "@testing-library/react";

import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { NotFoundPage } from "./NotFoundPage";

describe("NotFoundPage", () => {
  it("renders the not-found heading + back-to-dashboard link", () => {
    renderWithProviders(<NotFoundPage />);
    expect(screen.getByTestId(selectors.pages.notFound)).toBeInTheDocument();
    expect(screen.getByRole("link")).toHaveAttribute("href", "/");
  });
});
