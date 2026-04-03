import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { ReviewLaunchSheet } from "./review-launch-sheet";
import { selectors } from "../../consts/selectors";
import type { ReviewLaunchSheetProps } from "./review-launch-sheet";

vi.mock("../ui/bottom-sheet", () => ({
  BottomSheet: ({ isOpen, children }: { isOpen: boolean; children: React.ReactNode }) =>
    isOpen ? <div data-testid={selectors.review.launchSheet}>{children}</div> : null,
}));

const defaultProps: ReviewLaunchSheetProps = {
  isOpen: true,
  onClose: vi.fn(),
  onFullReview: vi.fn(),
  onGatherEvidence: vi.fn(),
  isTriggering: false,
  isTriggeringEvidence: false,
  hasExistingFinalization: true,
  reviewAgentEnabled: true,
};

describe("ReviewLaunchSheet", () => {
  it("renders both option cards when open", () => {
    render(<ReviewLaunchSheet {...defaultProps} />);
    expect(screen.getByTestId(selectors.review.launchSheetFullReview)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.review.launchSheetGatherEvidence)).toBeInTheDocument();
  });

  it("renders nothing when closed", () => {
    const { container } = render(<ReviewLaunchSheet {...defaultProps} isOpen={false} />);
    expect(container.firstChild).toBeNull();
  });

  it("fires onFullReview when Full Review option is clicked", () => {
    const onFullReview = vi.fn();
    render(<ReviewLaunchSheet {...defaultProps} onFullReview={onFullReview} />);
    fireEvent.click(screen.getByTestId(selectors.review.launchSheetFullReview));
    expect(onFullReview).toHaveBeenCalledOnce();
  });

  it("fires onGatherEvidence when Gather Evidence option is clicked", () => {
    const onGatherEvidence = vi.fn();
    render(<ReviewLaunchSheet {...defaultProps} onGatherEvidence={onGatherEvidence} />);
    fireEvent.click(screen.getByTestId(selectors.review.launchSheetGatherEvidence));
    expect(onGatherEvidence).toHaveBeenCalledOnce();
  });

  it("disables Gather Evidence when hasExistingFinalization is false", () => {
    render(<ReviewLaunchSheet {...defaultProps} hasExistingFinalization={false} />);
    const btn = screen.getByTestId(selectors.review.launchSheetGatherEvidence);
    expect(btn).toBeDisabled();
  });

  it("keeps Full Review enabled when hasExistingFinalization is false", () => {
    render(<ReviewLaunchSheet {...defaultProps} hasExistingFinalization={false} />);
    const btn = screen.getByTestId(selectors.review.launchSheetFullReview);
    expect(btn).not.toBeDisabled();
  });

  it("shows hint text when no existing finalization", () => {
    render(<ReviewLaunchSheet {...defaultProps} hasExistingFinalization={false} />);
    expect(
      screen.getByText("Run a full review first before gathering additional evidence."),
    ).toBeInTheDocument();
  });

  it("does not show hint text when finalization exists", () => {
    render(<ReviewLaunchSheet {...defaultProps} hasExistingFinalization />);
    expect(
      screen.queryByText("Run a full review first before gathering additional evidence."),
    ).not.toBeInTheDocument();
  });

  it("disables both options when isTriggering is true", () => {
    render(<ReviewLaunchSheet {...defaultProps} isTriggering />);
    expect(screen.getByTestId(selectors.review.launchSheetFullReview)).toBeDisabled();
    expect(screen.getByTestId(selectors.review.launchSheetGatherEvidence)).toBeDisabled();
  });

  it("disables both options when isTriggeringEvidence is true", () => {
    render(<ReviewLaunchSheet {...defaultProps} isTriggeringEvidence />);
    expect(screen.getByTestId(selectors.review.launchSheetFullReview)).toBeDisabled();
    expect(screen.getByTestId(selectors.review.launchSheetGatherEvidence)).toBeDisabled();
  });

  it("shows agent enabled status bar when reviewAgentEnabled is true", () => {
    render(<ReviewLaunchSheet {...defaultProps} reviewAgentEnabled />);
    expect(screen.getByTestId("review-agent-status-bar")).toBeInTheDocument();
    expect(screen.getByText("Review agent enabled")).toBeInTheDocument();
    expect(screen.getByText(/Both options will spawn an AI agent/)).toBeInTheDocument();
  });

  it("shows agent disabled status bar when reviewAgentEnabled is false", () => {
    render(<ReviewLaunchSheet {...defaultProps} reviewAgentEnabled={false} />);
    expect(screen.getByTestId("review-agent-status-bar")).toBeInTheDocument();
    expect(screen.getByText("Review agent disabled")).toBeInTheDocument();
    expect(screen.getByText(/gather evidence automatically/)).toBeInTheDocument();
  });

  it("disables Gather Evidence when agent is disabled even with existing finalization", () => {
    render(<ReviewLaunchSheet {...defaultProps} hasExistingFinalization reviewAgentEnabled={false} />);
    expect(screen.getByTestId(selectors.review.launchSheetGatherEvidence)).toBeDisabled();
  });

  it("shows trigger error when present", () => {
    render(<ReviewLaunchSheet {...defaultProps} triggerError="agent activity spec missing from context" />);
    expect(screen.getByTestId("review-trigger-error")).toBeInTheDocument();
    expect(screen.getByText("Review failed")).toBeInTheDocument();
    expect(screen.getByText(/agent activity spec missing/)).toBeInTheDocument();
  });

  it("does not show error when triggerError is null", () => {
    render(<ReviewLaunchSheet {...defaultProps} triggerError={null} />);
    expect(screen.queryByTestId("review-trigger-error")).not.toBeInTheDocument();
  });

  it("shows error alongside option cards", () => {
    render(<ReviewLaunchSheet {...defaultProps} triggerError="Network error" />);
    // Error visible
    expect(screen.getByTestId("review-trigger-error")).toBeInTheDocument();
    // Options still rendered and clickable (for retry)
    expect(screen.getByTestId(selectors.review.launchSheetFullReview)).not.toBeDisabled();
    expect(screen.getByTestId(selectors.review.launchSheetGatherEvidence)).not.toBeDisabled();
  });
});
