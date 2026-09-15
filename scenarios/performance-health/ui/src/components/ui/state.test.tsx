import { describe, expect, it, vi } from "vitest";
import { fireEvent, screen } from "@testing-library/react";
import { Activity } from "lucide-react";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { EmptyState, ErrorState, LoadingState, Skeleton } from "./state";

describe("state primitives (cimode — copy-independent)", () => {
  describe("LoadingState", () => {
    it("renders a labelled status spinner by default", () => {
      renderWithProviders(<LoadingState />);
      const block = screen.getByTestId(selectors.state.loading);
      expect(block).toHaveAttribute("role", "status");
      expect(block).toHaveAttribute("aria-busy", "true");
      // Falls back to the shared loading copy.
      expect(screen.getByText(strings.common.loading)).toBeInTheDocument();
    });

    it("renders a bespoke skeleton tree when provided", () => {
      renderWithProviders(
        <LoadingState
          label="custom-label"
          skeleton={<Skeleton className="h-4" />}
        />,
      );
      const block = screen.getByTestId(selectors.state.loading);
      expect(block).toHaveAttribute("aria-label", "custom-label");
      // The spinner copy is NOT rendered in skeleton mode.
      expect(screen.queryByText(strings.common.loading)).not.toBeInTheDocument();
    });
  });

  describe("ErrorState", () => {
    it("renders the message with an alert role and the default heading", () => {
      renderWithProviders(<ErrorState message="boom" />);
      const block = screen.getByTestId(selectors.state.error);
      expect(block).toHaveAttribute("role", "alert");
      expect(block).toHaveTextContent("boom");
      expect(screen.getByText(strings.common.errorTitle)).toBeInTheDocument();
    });

    it("renders a retry affordance and fires onRetry", () => {
      const onRetry = vi.fn();
      renderWithProviders(<ErrorState message="boom" onRetry={onRetry} />);
      fireEvent.click(screen.getByTestId(selectors.state.errorRetry));
      expect(onRetry).toHaveBeenCalledTimes(1);
    });

    it("disables retry while retrying", () => {
      renderWithProviders(<ErrorState message="boom" onRetry={vi.fn()} retrying />);
      expect(screen.getByTestId(selectors.state.errorRetry)).toBeDisabled();
    });

    it("omits the retry button when no handler is given", () => {
      renderWithProviders(<ErrorState message="boom" />);
      expect(screen.queryByTestId(selectors.state.errorRetry)).not.toBeInTheDocument();
    });
  });

  describe("EmptyState", () => {
    it("renders the message and a custom icon", () => {
      renderWithProviders(<EmptyState message="nothing here" icon={Activity} />);
      expect(screen.getByTestId(selectors.state.empty)).toHaveTextContent("nothing here");
    });

    it("renders a click action and fires onAction", () => {
      const onAction = vi.fn();
      renderWithProviders(
        <EmptyState message="empty" actionLabel="Go" onAction={onAction} />,
      );
      fireEvent.click(screen.getByTestId(selectors.state.emptyAction));
      expect(onAction).toHaveBeenCalledTimes(1);
    });

    it("prefers a custom action slot over the click action", () => {
      renderWithProviders(
        <EmptyState
          message="empty"
          actionLabel="Go"
          onAction={vi.fn()}
          actionSlot={<a href="/x" data-testid="custom-slot">link-action</a>}
        />,
      );
      expect(screen.getByTestId("custom-slot")).toBeInTheDocument();
      expect(screen.queryByTestId(selectors.state.emptyAction)).not.toBeInTheDocument();
    });
  });
});
