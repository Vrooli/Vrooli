import { act, fireEvent, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { SidebarNavigation } from "./SidebarNavigation";
import { renderWithProviders } from "../../test-utils/renderWithProviders";
import { createPipelineStatus } from "../../test-utils/mocks";
import { usePipelineStore, useSidebarStore } from "../../store";
import {
  StageName,
  StageStatus,
} from "@vrooli/proto-types/scenario-to-desktop/v1/shared/common_pb";

function setState() {
  act(() => {
    usePipelineStore.getState().reset();
    useSidebarStore.setState({ collapsed: false, activeSection: "configuration" });
  });
}

describe("SidebarNavigation", () => {
  beforeEach(setState);

  it("shows every section, identifies the active running stage, and navigates", () => {
    const onSectionClick = vi.fn();
    act(() => {
      useSidebarStore.setState({ activeSection: "build" });
      usePipelineStore.setState({
        pipelineStatus: createPipelineStatus({
          currentStage: StageName.BUILD,
          status: StageStatus.RUNNING,
          stages: {
            build: { stage: StageName.BUILD, status: StageStatus.RUNNING },
          },
        }),
      });
    });
    renderWithProviders(<SidebarNavigation onSectionClick={onSectionClick} />);

    expect(screen.getByText("Configuration")).toBeInTheDocument();
    expect(screen.getByText("Build")).toHaveClass("text-blue-400");
    expect(screen.getByText("Running...")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /Deploy/ }));
    expect(onSectionClick).toHaveBeenCalledWith("deploy");
  });

  it("uses accessible icon controls in its collapsed presentation", () => {
    const onSectionClick = vi.fn();
    renderWithProviders(
      <SidebarNavigation onSectionClick={onSectionClick} collapsed />,
    );

    expect(screen.queryByText("Set up your desktop app")).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Smoke Test" }));
    expect(onSectionClick).toHaveBeenCalledWith("smoketest");
  });

  it("marks configuration complete only after an actual pipeline stage begins", () => {
    const { rerender } = renderWithProviders(
      <SidebarNavigation onSectionClick={vi.fn()} />,
    );
    expect(screen.getByText("Configuration").closest("button")).not.toHaveClass(
      "border-l-2",
    );

    act(() => {
      usePipelineStore.setState({
        pipelineStatus: createPipelineStatus({
          status: StageStatus.RUNNING,
          stages: {
            bundle: { stage: StageName.BUNDLE, status: StageStatus.RUNNING },
          },
        }),
      });
    });
    rerender(<SidebarNavigation onSectionClick={vi.fn()} />);
    expect(screen.getByText("Configuration").closest("button")).toHaveClass(
      "bg-green-950/30",
    );
  });
});
