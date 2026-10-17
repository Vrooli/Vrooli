import { afterEach, describe, expect, it } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { renderWithProviders } from "../test-utils";
import { NotFoundPage } from "./NotFoundPage";

describe("NotFoundPage", () => {
  afterEach(() => cleanup());

  it("renders title, description, and a back-to-dashboard link", () => {
    renderWithProviders(<NotFoundPage />);
    expect(screen.getByTestId("not-found-page")).toBeInTheDocument();
    expect(screen.getByTestId("not-found-home")).toHaveAttribute("href", "/");
  });
});
