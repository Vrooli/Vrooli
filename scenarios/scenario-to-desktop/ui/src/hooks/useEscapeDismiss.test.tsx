import { fireEvent } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../test-utils/renderWithProviders";
import { useEscapeDismiss } from "./useEscapeDismiss";

function Fixture({
  active,
  onDismiss,
}: {
  active: boolean;
  onDismiss: () => void;
}) {
  useEscapeDismiss(active, onDismiss);
  return null;
}

describe("useEscapeDismiss", () => {
  it("dismisses only an active surface when Escape is pressed", () => {
    const onDismiss = vi.fn();
    const view = renderWithProviders(
      <Fixture active={false} onDismiss={onDismiss} />,
    );
    fireEvent.keyDown(document, { key: "Escape" });
    expect(onDismiss).not.toHaveBeenCalled();

    view.rerender(<Fixture active onDismiss={onDismiss} />);
    fireEvent.keyDown(document, { key: "Escape" });
    expect(onDismiss).toHaveBeenCalledOnce();
  });
});
