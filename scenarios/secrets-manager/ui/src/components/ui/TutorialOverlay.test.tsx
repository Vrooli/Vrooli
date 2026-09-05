import { cleanup, fireEvent, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils";
import { TutorialOverlay } from "./TutorialOverlay";

describe("TutorialOverlay", () => {
  afterEach(cleanup);

  it("focuses its anchor, supports navigation, tutorial selection, and dragging", () => {
    const onClose = vi.fn();
    const onNext = vi.fn();
    const onBack = vi.fn();
    const onSelectTutorial = vi.fn();
    const anchor = document.createElement("div");
    anchor.id = "deployment-anchor";
    document.body.append(anchor);

    renderWithProviders(
      <TutorialOverlay
        title="Deployment basics"
        subtitle="Learn the handoff"
        stepLabel="Step 1 of 2"
        content={<p>Choose a campaign.</p>}
        anchorId="deployment-anchor"
        onClose={onClose}
        onNext={onNext}
        onBack={onBack}
        tutorials={[{ id: "deployment", label: "Deployment" }, { id: "vault", label: "Vault" }]}
        activeTutorialId="deployment"
        onSelectTutorial={onSelectTutorial}
      />
    );

    expect(anchor).toHaveAttribute("tabindex", "-1");
    expect(anchor).toHaveClass("tutorial-highlight");
    fireEvent.click(screen.getByRole("button", { name: /Deployment basics/ }));
    fireEvent.click(screen.getByRole("button", { name: "Vault" }));
    fireEvent.click(screen.getByRole("button", { name: "Back" }));
    fireEvent.click(screen.getByRole("button", { name: "Next" }));
    fireEvent.click(screen.getByRole("button", { name: "Close tutorial" }));
    fireEvent.mouseDown(screen.getByText("Tutorial"), { clientX: 80, clientY: 140 });
    fireEvent.mouseMove(window, { clientX: 120, clientY: 180 });
    fireEvent.mouseUp(window);

    expect(onSelectTutorial).toHaveBeenCalledWith("vault");
    expect(onBack).toHaveBeenCalledOnce();
    expect(onNext).toHaveBeenCalledOnce();
    expect(onClose).toHaveBeenCalledOnce();
    anchor.remove();
  });

  it("keeps absent optional controls disabled or hidden", () => {
    renderWithProviders(
      <TutorialOverlay title="Readiness" stepLabel="Step 1" onClose={() => {}} disableNext />
    );
    expect(screen.getByRole("button", { name: /Readiness/ })).toBeDisabled();
    expect(screen.queryByRole("button", { name: "Back" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Next" })).not.toBeInTheDocument();
  });
});
