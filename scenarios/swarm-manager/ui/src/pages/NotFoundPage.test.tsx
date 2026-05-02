import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, fireEvent } from "@testing-library/react";
import { NotFoundPage } from "./NotFoundPage";
import { selectors } from "../consts/selectors";
import { renderWithProviders } from "../test-utils";

// Mock react-router-dom navigation
const mockNavigate = vi.fn();
vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom");
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  };
});

/**
 * NotFoundPage Tests
 *
 * Tests verify:
 * - Page renders with correct structure and content
 * - User-friendly messaging (no technical 404 jargon)
 * - Navigation to home (Ideas page) works correctly
 * - Test selectors are properly applied
 *
 * [REQ:PHASE6] Error recovery path tests
 */

describe("NotFoundPage", () => {
  beforeEach(() => {
    mockNavigate.mockClear();
  });

  function renderPage() {
    return renderWithProviders(<NotFoundPage />);
  }

  describe("page structure", () => {
    it("renders the page container with correct test ID", () => {
      renderPage();

      expect(screen.getByTestId(selectors.notFound.page)).toBeInTheDocument();
    });

    it("renders user-friendly title", () => {
      renderPage();

      const title = screen.getByTestId(selectors.notFound.title);
      expect(title).toHaveTextContent("Page not found");
      // Should NOT contain technical 404 code
      expect(title).not.toHaveTextContent("404");
    });

    it("renders helpful message", () => {
      renderPage();

      const message = screen.getByTestId(selectors.notFound.message);
      expect(message).toHaveTextContent("doesn't exist");
      expect(message).toHaveTextContent("may have been moved");
    });

    it("renders home button with correct test ID", () => {
      renderPage();

      expect(screen.getByTestId(selectors.notFound.homeButton)).toBeInTheDocument();
    });

    it("home button displays correct text", () => {
      renderPage();

      const button = screen.getByTestId(selectors.notFound.homeButton);
      expect(button).toHaveTextContent("Go to Backlog");
    });
  });

  describe("navigation", () => {
    it("navigates to /backlog when home button clicked", () => {
      renderPage();

      const button = screen.getByTestId(selectors.notFound.homeButton);
      fireEvent.click(button);

      expect(mockNavigate).toHaveBeenCalledTimes(1);
      expect(mockNavigate).toHaveBeenCalledWith("/backlog", { replace: true });
    });

    it("uses replace navigation to prevent back-to-404 loop", () => {
      renderPage();

      const button = screen.getByTestId(selectors.notFound.homeButton);
      fireEvent.click(button);

      // The key assertion: replace: true prevents user from navigating back to 404
      expect(mockNavigate).toHaveBeenCalledWith("/backlog", expect.objectContaining({ replace: true }));
    });
  });

  describe("accessibility", () => {
    it("has proper heading hierarchy", () => {
      renderPage();

      const heading = screen.getByRole("heading", { level: 1 });
      expect(heading).toHaveTextContent("Page not found");
    });

    it("button is focusable", () => {
      renderPage();

      const button = screen.getByTestId(selectors.notFound.homeButton);
      expect(button.tagName.toLowerCase()).toBe("button");
    });
  });
});
