import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { CapabilityGate } from "./CapabilityGate";
import { renderWithProviders } from "../../test-utils";

describe("CapabilityGate", () => {
  it("shows the owner-only one-turn action and routes the answer", () => {
    const onAnswer = vi.fn();
    renderWithProviders(<CapabilityGate scope="filesystem.write" withheld="a workspace write" unblock="owner approval" expiresAt="in 10 minutes" viewerIsOwner onAnswer={onAnswer} />);
    expect(screen.getByTestId("capability-gate-withheld")).toHaveTextContent("filesystem.write");
    fireEvent.click(screen.getByTestId("capability-gate-grant"));
    expect(onAnswer).toHaveBeenCalledWith(true);
  });

  it("does not render grant controls for a non-owner", () => {
    renderWithProviders(<CapabilityGate scope="owner" withheld="owner work" unblock="owner approval" expiresAt="in 10 minutes" viewerIsOwner={false} />);
    expect(screen.queryByTestId("capability-gate-grant")).not.toBeInTheDocument();
    expect(screen.getByTestId("capability-gate-permission-denied")).toBeInTheDocument();
  });
});
