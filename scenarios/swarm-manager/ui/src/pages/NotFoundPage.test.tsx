import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { NotFoundPage } from "./NotFoundPage";
import { selectors } from "../consts/selectors";

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

  describe("page structure", () => {
    it("renders the page container with correct test ID", () => {
      render(
        <MemoryRouter>
          <NotFoundPage />
        </MemoryRouter>
      );

      expect(screen.getByTestId(selectors.notFound.page)).toBeInTheDocument();
    });

    it("renders user-friendly title", () => {
      render(
        <MemoryRouter>
          <NotFoundPage />
        </MemoryRouter>
      );

      const title = screen.getByTestId(selectors.notFound.title);
      expect(title).toHaveTextContent("Page not found");
      // Should NOT contain technical 404 code
      expect(title).not.toHaveTextContent("404");
    });

    it("renders helpful message", () => {
      render(
        <MemoryRouter>
          <NotFoundPage />
        </MemoryRouter>
      );

      const message = screen.getByTestId(selectors.notFound.message);
      expect(message).toHaveTextContent("doesn't exist");
      expect(message).toHaveTextContent("may have been moved");
    });

    it("renders home button with correct test ID", () => {
      render(
        <MemoryRouter>
          <NotFoundPage />
        </MemoryRouter>
      );

      expect(screen.getByTestId(selectors.notFound.homeButton)).toBeInTheDocument();
    });

    it("home button displays correct text", () => {
      render(
        <MemoryRouter>
          <NotFoundPage />
        </MemoryRouter>
      );

      const button = screen.getByTestId(selectors.notFound.homeButton);
      expect(button).toHaveTextContent("Go to Ideas");
    });
  });

  describe("navigation", () => {
    it("navigates to /ideas when home button clicked", () => {
      render(
        <MemoryRouter>
          <NotFoundPage />
        </MemoryRouter>
      );

      const button = screen.getByTestId(selectors.notFound.homeButton);
      fireEvent.click(button);

      expect(mockNavigate).toHaveBeenCalledTimes(1);
      expect(mockNavigate).toHaveBeenCalledWith("/ideas", { replace: true });
    });

    it("uses replace navigation to prevent back-to-404 loop", () => {
      render(
        <MemoryRouter>
          <NotFoundPage />
        </MemoryRouter>
      );

      const button = screen.getByTestId(selectors.notFound.homeButton);
      fireEvent.click(button);

      // The key assertion: replace: true prevents user from navigating back to 404
      expect(mockNavigate).toHaveBeenCalledWith("/ideas", expect.objectContaining({ replace: true }));
    });
  });

  describe("accessibility", () => {
    it("has proper heading hierarchy", () => {
      render(
        <MemoryRouter>
          <NotFoundPage />
        </MemoryRouter>
      );

      const heading = screen.getByRole("heading", { level: 1 });
      expect(heading).toHaveTextContent("Page not found");
    });

    it("button is focusable", () => {
      render(
        <MemoryRouter>
          <NotFoundPage />
        </MemoryRouter>
      );

      const button = screen.getByTestId(selectors.notFound.homeButton);
      expect(button.tagName.toLowerCase()).toBe("button");
    });
  });
});
