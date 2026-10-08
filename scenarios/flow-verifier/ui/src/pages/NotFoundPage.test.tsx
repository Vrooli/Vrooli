import { afterEach, describe, expect, it } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { renderWithProviders } from "../test-utils";
import { NotFoundPage } from "./NotFoundPage";

describe("NotFoundPage", () => {
  afterEach(() => cleanup());

  it("renders heading and a home link", () => {
    renderWithProviders(<NotFoundPage />);
    expect(screen.getByTestId("not-found-page")).toBeInTheDocument();
    expect(screen.getByTestId("not-found-home")).toHaveAttribute("href", "/");
  });
});
