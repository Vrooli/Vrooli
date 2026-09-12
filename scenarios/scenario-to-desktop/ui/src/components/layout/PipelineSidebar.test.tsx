import { act, fireEvent, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { PipelineSidebar } from "./PipelineSidebar";
import { MobilePipelineSummary } from "./MobilePipelineSummary";
import { renderWithProviders } from "@vrooli/api-base/testing";
import { useSidebarStore, usePipelineStore, initialPipelineState } from "../../store";
import {
  StageName,
  StageStatus,
} from "@vrooli/proto-types/scenario-to-desktop/v1/shared/common_pb";

const isMobile = vi.hoisted(() => vi.fn(() => false));

vi.mock("../../hooks/useMediaQuery", () => ({ useIsMobile: isMobile }));
vi.mock("./SidebarHeader", () => ({
  SidebarHeader: () => <div>pipeline header</div>,
}));
vi.mock("./SidebarNavigation", () => ({
  SidebarNavigation: ({ collapsed }: { collapsed?: boolean }) => (
    <div>{collapsed ? "collapsed navigation" : "expanded navigation"}</div>
  ),
}));

describe("PipelineSidebar", () => {
  beforeEach(() => {
    isMobile.mockReturnValue(false);
    act(() => {
      useSidebarStore.setState({ collapsed: false });
      usePipelineStore.setState(initialPipelineState);
    });
  });

  it("toggles between expanded and compact desktop navigation", () => {
    renderWithProviders(<PipelineSidebar onSectionClick={vi.fn()} />);

    expect(screen.getByText("pipeline header")).toBeInTheDocument();
    expect(screen.getByText("expanded navigation")).toBeInTheDocument();
    fireEvent.click(
      screen.getByRole("button", { name: "Collapse pipeline sidebar" }),
    );
    expect(screen.getByText("collapsed navigation")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Expand pipeline sidebar" }),
    ).toBeInTheDocument();
  });

  it("keeps the sidebar expanded and removes the desktop collapse control on mobile", () => {
    isMobile.mockReturnValue(true);
    act(() => {
      useSidebarStore.setState({ collapsed: true });
    });
    renderWithProviders(<PipelineSidebar onSectionClick={vi.fn()} />);

    expect(screen.getByText("expanded navigation")).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /pipeline sidebar/ }),
    ).not.toBeInTheDocument();
  });

  it("shows the compact pipeline summary for idle, completed, and failed states", () => {
    const onOpenDrawer = vi.fn();
    const { rerender } = renderWithProviders(
      <MobilePipelineSummary onOpenDrawer={onOpenDrawer} />,
    );

    expect(screen.getByText("No scenario selected")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Open pipeline sidebar" }));

    act(() => {
      usePipelineStore.setState({
        scenarioName: "completed-scenario",
        runStatus: "completed",
      });
    });
    rerender(<MobilePipelineSummary onOpenDrawer={onOpenDrawer} />);
    expect(screen.getByText("completed-scenario")).toBeInTheDocument();

    act(() => {
      usePipelineStore.setState({
        scenarioName: "failed-scenario",
        runStatus: "failed",
      });
    });
    rerender(<MobilePipelineSummary onOpenDrawer={onOpenDrawer} />);
    expect(screen.getByText("failed-scenario")).toBeInTheDocument();
    expect(onOpenDrawer).toHaveBeenCalledTimes(1);
  });

  it("shows running progress with the active stage or percentage fallback", () => {
    const onOpenDrawer = vi.fn();
    const { rerender } = renderWithProviders(
      <MobilePipelineSummary onOpenDrawer={onOpenDrawer} />,
    );

    act(() => {
      usePipelineStore.setState({
        scenarioName: "running-scenario",
        runStatus: "running",
        pipelineStatus: {
          currentStage: StageName.BUILD,
          stageOrder: [StageName.BUNDLE],
          stages: {
            bundle: { stage: StageName.BUNDLE, status: StageStatus.COMPLETED },
          },
        } as never,
      });
    });
    rerender(<MobilePipelineSummary onOpenDrawer={onOpenDrawer} />);
    expect(screen.getByText("Build")).toBeInTheDocument();
    expect(screen.getByText("running-scenario")).toBeInTheDocument();

    act(() => {
      usePipelineStore.setState({
        pipelineStatus: {
          stageOrder: [StageName.BUNDLE],
          stages: {
            bundle: { stage: StageName.BUNDLE, status: StageStatus.PENDING },
          },
        } as never,
      });
    });
    rerender(<MobilePipelineSummary onOpenDrawer={onOpenDrawer} />);
    expect(screen.getByText("0%")).toBeInTheDocument();
  });
});
