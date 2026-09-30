/**
 * DesignCard tests — focused on the design-card surface only. Renders
 * <DesignCard /> directly so failures point at design-feature behaviour, not
 * shell composition. Follows the canonical mock-builder pattern.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { makeDesignResponse } from "./mocks/factories";
import { makeDesignMocks } from "./mocks/design";

vi.mock("../../api/design", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/design")>();
  return { ...actual, ...makeDesignMocks() };
});

import { DesignCard } from "./DesignCard";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";

describe("DesignCard", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("keeps the generate button disabled until a brand id is entered", async () => {
    renderWithProviders(<DesignCard />);
    const button = screen.getByTestId(selectors.design.generateButton);
    expect(button).toBeDisabled();

    const user = userEvent.setup();
    await user.type(screen.getByTestId(selectors.design.brandInput), "brand-1");
    expect(button).toBeEnabled();
  });

  it("renders the DESIGN.md markdown returned by the server", async () => {
    const { generateDesignLanguage } = await import("../../api/design");
    vi.mocked(generateDesignLanguage).mockResolvedValueOnce(
      makeDesignResponse({ brandId: "brand-1", markdown: "# Acme DESIGN.md\n\n## Color System" }),
    );

    const user = userEvent.setup();
    renderWithProviders(<DesignCard />);
    await user.type(screen.getByTestId(selectors.design.brandInput), "brand-1");
    await user.click(screen.getByTestId(selectors.design.generateButton));

    await waitFor(() => {
      expect(screen.getByTestId(selectors.design.result)).toBeInTheDocument();
    });
    expect(screen.getByTestId(selectors.design.markdown).textContent).toContain("# Acme DESIGN.md");
    expect(screen.getByTestId(selectors.design.markdown).textContent).toContain("## Color System");
  });
});
