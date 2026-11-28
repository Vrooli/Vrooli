import { describe, expect, it, vi } from "vitest";
import { screen } from "@testing-library/react";

import { useComponentBundle } from "../data";
import { renderWithProviders } from "../../test-utils";
import { WorkspaceComponentSlot } from "./WorkspaceComponentSlot";

vi.mock("../data", () => ({ useComponentBundle: vi.fn() }));
vi.mock("../component/FixtureFrame", () => ({
  FixtureFrame: ({ height, version }: { height: number; version: string }) => (
    <div data-testid="fixture" data-height={height} data-version={version} />
  ),
}));

const mockedBundle = vi.mocked(useComponentBundle);

describe("WorkspaceComponentSlot", () => {
  it("uses the authored slot height for the real component fixture", () => {
    mockedBundle.mockReturnValue({
      isLoading: false,
      data: {
        asset: { latestVersion: "2.0.0", version: "2.0.0" },
        stories: [{ id: "default", name: "Default", version: "1.2.0" }],
      },
    } as ReturnType<typeof useComponentBundle>);

    renderWithProviders(
      <WorkspaceComponentSlot
        componentId="controls.button"
        storyId="default"
        componentVersion="1.2.0"
        height={176}
      />,
    );

    expect(screen.getByTestId("fixture")).toHaveAttribute("data-height", "176");
    expect(screen.getByTestId("fixture")).toHaveAttribute("data-version", "1.2.0");
    expect(mockedBundle).toHaveBeenCalledWith("controls.button", "1.2.0");
  });

  it("keeps loading feedback at the authored slot height", () => {
    mockedBundle.mockReturnValue({ isLoading: true } as ReturnType<typeof useComponentBundle>);

    renderWithProviders(
      <WorkspaceComponentSlot componentId="controls.button" storyId="default" height={176} />,
    );

    expect(document.querySelector("[style*='height: 176px']")).not.toBeNull();
  });
});
