import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { AppCard } from "./AppCard";
import { renderWithProviders } from "../../test-utils/renderWithProviders";
import type { DesktopRecordItemView } from "./recordPresentation";

vi.mock("./SigningBadge", () => ({ SigningBadge: () => <span>Signing ready</span> }));

function item(overrides: Partial<DesktopRecordItemView> = {}): DesktopRecordItemView {
  return {
    record: { id: "rec-1", build_id: "build-1", scenario_name: "canvas-lab", app_display_name: "Canvas Lab", output_path: "/tmp/canvas" },
    has_build: true,
    build_state: "ready",
    ...overrides,
  };
}

describe("AppCard", () => {
  it("presents the generated app and invokes the detail action by mouse and keyboard", () => {
    const onClick = vi.fn();
    renderWithProviders(<AppCard item={item()} onClick={onClick} />);
    expect(screen.getAllByText("Canvas Lab")).toHaveLength(2);
    expect(screen.getAllByText("Ready")).toHaveLength(2);
    const cards = screen.getAllByRole("button");
    const [desktopCard, mobileCard] = cards;
    if (!desktopCard || !mobileCard) throw new Error("expected desktop and mobile app cards");
    fireEvent.click(desktopCard);
    fireEvent.keyDown(mobileCard, { key: "Enter" });
    fireEvent.keyDown(mobileCard, { key: " " });
    expect(onClick).toHaveBeenCalledTimes(3);
  });

  it.each([
    ["building", "Building"],
    ["failed", "Failed"],
    ["queued", "queued"],
  ])("presents %s build state", (build_state, label) => {
    renderWithProviders(<AppCard item={item({ build_state })} onClick={vi.fn()} />);
    expect(screen.getAllByText(label)).not.toHaveLength(0);
  });

  it("falls back to the scenario name when no display name exists", () => {
    renderWithProviders(<AppCard item={item({ record: { ...item().record, app_display_name: undefined } })} onClick={vi.fn()} />);
    expect(screen.getAllByText("canvas-lab")).toHaveLength(4);
  });
});
