import { beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, screen } from "@testing-library/react";
import type { ButtonHTMLAttributes } from "react";
import {
  Platform,
  StageStatus,
} from "@vrooli/proto-types/scenario-to-desktop/v1/shared/common_pb";

const hooks = vi.hoisted(() => ({
  selectedPlatforms: ["linux"] as string[],
  togglePlatform: vi.fn(),
  needsWineForPlatforms: vi.fn(() => false),
  setShowWineDialog: vi.fn(),
  setPendingPlatforms: vi.fn(),
  showWineDialog: false,
  pendingPlatforms: [] as string[],
  baseWineComplete: vi.fn(),
  mutationError: null as string | null,
  runPipelineWithConfig: vi.fn(),
  reset: vi.fn(),
  pipelineStatus: null as {
    status: StageStatus;
    stages: {
      build?: { details?: { kind: { case: string; value: unknown } } };
    };
  } | null,
  statusIsBuilding: false,
}));

vi.mock("../../hooks", () => ({
  usePlatformSelection: () => ({
    selectedPlatforms: hooks.selectedPlatforms,
    togglePlatform: hooks.togglePlatform,
  }),
  useWineCheck: () => ({
    showWineDialog: hooks.showWineDialog,
    setShowWineDialog: hooks.setShowWineDialog,
    pendingPlatforms: hooks.pendingPlatforms,
    setPendingPlatforms: hooks.setPendingPlatforms,
    needsWineForPlatforms: hooks.needsWineForPlatforms,
    handleWineInstallComplete: hooks.baseWineComplete,
  }),
  usePipelineMutation: () => ({
    state: { buildId: "build-1", error: hooks.mutationError },
    mutation: { isPending: false },
    runPipelineWithConfig: hooks.runPipelineWithConfig,
    reset: hooks.reset,
  }),
  usePipelineStatus: () => ({
    pipelineStatus: hooks.pipelineStatus,
    isBuilding: hooks.statusIsBuilding,
  }),
}));

vi.mock("../ui/button", () => ({
  Button: ({ children, ...props }: ButtonHTMLAttributes<HTMLButtonElement>) => (
    <button {...props}>{children}</button>
  ),
}));
vi.mock("./PlatformChip", () => ({
  PlatformChip: () => <div>platform result</div>,
}));
vi.mock("../wine", () => ({
  WineInstallDialog: ({
    onClose,
    onInstallComplete,
  }: {
    onClose: () => void;
    onInstallComplete: () => void;
  }) => (
    <div>
      <button onClick={onInstallComplete}>Finish Wine setup</button>
      <button onClick={onClose}>Cancel Wine setup</button>
    </div>
  ),
}));
vi.mock("../pipeline", () => ({
  PipelineErrorDisplay: ({
    onRetry,
    errorMessage,
  }: {
    onRetry: () => void;
    errorMessage: string;
  }) => (
    <div>
      <p>{errorMessage}</p>
      <button onClick={onRetry}>Retry build</button>
    </div>
  ),
}));

import { BuildDesktopButton } from "./BuildDesktopButton";
import { renderWithProviders } from "@vrooli/api-base/testing";

function renderButton() {
  return renderWithProviders(<BuildDesktopButton scenarioName="calculator" />);
}

beforeEach(() => {
  hooks.selectedPlatforms = ["linux"];
  hooks.showWineDialog = false;
  hooks.pendingPlatforms = [];
  hooks.mutationError = null;
  hooks.pipelineStatus = null;
  hooks.statusIsBuilding = false;
  hooks.needsWineForPlatforms.mockReturnValue(false);
  vi.clearAllMocks();
});

describe("BuildDesktopButton", () => {
  it("starts a typed Linux installer build from the operator's selection", () => {
    renderButton();
    fireEvent.click(
      screen.getByRole("button", { name: /Build selected installers/ }),
    );

    expect(hooks.runPipelineWithConfig).toHaveBeenCalledWith({
      scenarioName: "calculator",
      platforms: [Platform.LINUX],
    });
  });

  it("routes Windows-required builds through Wine preparation before invoking the pipeline", () => {
    hooks.selectedPlatforms = ["win"];
    hooks.needsWineForPlatforms.mockReturnValue(true);
    renderButton();

    fireEvent.click(
      screen.getByRole("button", { name: /Build selected installers/ }),
    );
    expect(hooks.setPendingPlatforms).toHaveBeenCalledWith(["win"]);
    expect(hooks.setShowWineDialog).toHaveBeenCalledWith(true);
    expect(hooks.runPipelineWithConfig).not.toHaveBeenCalled();
  });

  it("resets and retries a failed build with the current selection", () => {
    hooks.mutationError = "Desktop compiler unavailable";
    renderButton();

    fireEvent.click(screen.getByRole("button", { name: "Retry build" }));
    expect(hooks.reset).toHaveBeenCalledOnce();
    expect(hooks.runPipelineWithConfig).toHaveBeenCalledWith({
      scenarioName: "calculator",
      platforms: [Platform.LINUX],
    });
  });
});
