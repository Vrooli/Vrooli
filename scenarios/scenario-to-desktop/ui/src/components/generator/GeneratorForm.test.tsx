import { fireEvent, render, screen } from "@/test-utils";
import { describe, expect, it, vi, beforeEach } from "vitest";
import { GeneratorForm } from "./GeneratorForm";

const { useGeneratorFormStateMock } = vi.hoisted(() => ({
  useGeneratorFormStateMock: vi.fn(),
}));

vi.mock("./useGeneratorFormState", () => ({
  useGeneratorFormState: useGeneratorFormStateMock,
}));

vi.mock(".", () => ({
  ScenarioSelector: ({
    scenarioName,
    onUnlock,
    onOpenScenarioModal,
    onLoadSaved,
  }: {
    scenarioName: string;
    onUnlock: () => void;
    onOpenScenarioModal: () => void;
    onLoadSaved?: () => void;
  }) => (
    <>
      <button type="button" onClick={onUnlock}>
        Scenario: {scenarioName}
      </button>
      <button type="button" onClick={onOpenScenarioModal}>
        Choose scenario
      </button>
      {onLoadSaved && (
        <button type="button" onClick={onLoadSaved}>
          Load saved connection
        </button>
      )}
    </>
  ),
  AppMetadataSection: ({ appDisplayName }: { appDisplayName: string }) => (
    <div>Application: {appDisplayName}</div>
  ),
  FrameworkTemplateSection: ({
    selectedTemplate,
    onOpenFrameworkModal,
    onOpenTemplateModal,
  }: {
    selectedTemplate: string;
    onOpenFrameworkModal: () => void;
    onOpenTemplateModal: () => void;
  }) => (
    <>
      <div>Template: {selectedTemplate}</div>
      <button type="button" onClick={onOpenFrameworkModal}>
        Choose framework
      </button>
      <button type="button" onClick={onOpenTemplateModal}>
        Choose template
      </button>
    </>
  ),
  SigningInlineSection: ({
    onOpenSigning,
    onToggleSigning,
    onRefresh,
  }: {
    onOpenSigning: () => void;
    onToggleSigning: (enabled: boolean) => void;
    onRefresh: () => void;
  }) => (
    <>
      <button type="button" onClick={onOpenSigning}>
        Open signing
      </button>
      <button
        type="button"
        onClick={() => {
          onToggleSigning(true);
        }}
      >
        Enable build signing
      </button>
      <button type="button" onClick={onRefresh}>
        Refresh signing
      </button>
    </>
  ),
  OutputLocationSelector: ({
    onChange,
  }: {
    onChange: (mode: "proper" | "custom") => void;
  }) => (
    <>
      <button
        type="button"
        onClick={() => {
          onChange("custom");
        }}
      >
        Use custom output
      </button>
      <button
        type="button"
        onClick={() => {
          onChange("proper");
        }}
      >
        Use standard output
      </button>
    </>
  ),
  OutputPathField: ({
    outputPath,
    onOutputPathChange,
  }: {
    outputPath: string;
    onOutputPathChange: (value: string) => void;
  }) => (
    <button
      type="button"
      onClick={() => {
        onOutputPathChange("/custom/output");
      }}
    >
      Output: {outputPath}
    </button>
  ),
  PlatformSelector: ({
    onPlatformChange,
  }: {
    onPlatformChange: (platform: string) => void;
  }) => (
    <button
      type="button"
      onClick={() => {
        onPlatformChange("win");
      }}
    >
      Platform selection
    </button>
  ),
  DeploymentSummarySection: ({
    onOpenDeploymentModal,
  }: {
    onOpenDeploymentModal: () => void;
  }) => (
    <button type="button" onClick={onOpenDeploymentModal}>
      Deployment summary
    </button>
  ),
  ConnectionSectionRouter: ({
    onProxyUrlChange,
    onAutoManageTier1Change,
    onVrooliBinaryPathChange,
    onServerPortChange,
    onLocalServerPathChange,
    onLocalApiEndpointChange,
  }: {
    onProxyUrlChange: (value: string) => void;
    onAutoManageTier1Change: (value: boolean) => void;
    onVrooliBinaryPathChange: (value: string) => void;
    onServerPortChange: (value: number) => void;
    onLocalServerPathChange: (value: string) => void;
    onLocalApiEndpointChange: (value: string) => void;
  }) => (
    <>
      <button
        type="button"
        onClick={() => {
          onProxyUrlChange("https://remote.example");
        }}
      >
        Connection selection
      </button>
      <button
        type="button"
        onClick={() => {
          onAutoManageTier1Change(false);
          onVrooliBinaryPathChange("/bin/vrooli");
          onServerPortChange(4000);
          onLocalServerPathChange("api/main.js");
          onLocalApiEndpointChange("http://127.0.0.1:4000");
        }}
      >
        Change runtime
      </button>
    </>
  ),
  GeneratorFormHeader: () => <div>Form status</div>,
  GeneratorFormFooter: () => <button type="submit">Generate</button>,
  GeneratorModalsContainer: ({
    onScenarioSelect,
    onTemplateSelect,
    onFrameworkSelect,
    onDeploymentChange,
    closeModal,
  }: {
    onScenarioSelect: (name: string) => void;
    onTemplateSelect: (name: string) => void;
    onFrameworkSelect: (framework: string) => void;
    onDeploymentChange: (mode: string, serverType?: string) => void;
    closeModal: (modal: string) => void;
  }) => (
    <>
      <button
        type="button"
        onClick={() => {
          onScenarioSelect("other");
          onTemplateSelect("minimal");
          onFrameworkSelect("electron");
          onDeploymentChange("bundled", "node");
          closeModal("scenario");
        }}
      >
        Configuration modals
      </button>
    </>
  ),
}));

function state(overrides: Record<string, unknown> = {}) {
  return {
    validationStatus: null,
    draftCreatedLabel: null,
    draftUpdatedLabel: null,
    stateSaving: false,
    handleReset: vi.fn(),
    isStale: false,
    pendingChanges: [],
    handleSubmit: vi.fn((event: React.FormEvent) => {
      event.preventDefault();
    }),
    loadingScenarios: false,
    selectedScenario: null,
    openModal: vi.fn(),
    applySavedConnection: vi.fn(),
    scenarioLocked: true,
    setScenarioLocked: vi.fn(),
    appDisplayName: "Secrets Manager Desktop",
    setAppDisplayName: vi.fn(),
    iconPath: "",
    setIconPath: vi.fn(),
    iconPreviewUrl: null,
    iconPreviewError: false,
    setIconPreviewError: vi.fn(),
    appDescription: "",
    setAppDescription: vi.fn(),
    framework: "react-vite",
    setFramework: vi.fn(),
    deploymentMode: "bundled",
    serverType: "managed",
    connectionDecision: "bundled",
    proxyUrl: "",
    setProxyUrl: vi.fn(),
    proxyHints: null,
    connectionTester: null,
    connectionResult: null,
    connectionError: null,
    autoManageTier1: true,
    setAutoManageTier1: vi.fn(),
    vrooliBinaryPath: "vrooli",
    setVrooliBinaryPath: vi.fn(),
    serverPort: "",
    setServerPort: vi.fn(),
    localServerPath: "",
    setLocalServerPath: vi.fn(),
    localApiEndpoint: "http://127.0.0.1:3000",
    setLocalApiEndpoint: vi.fn(),
    platforms: ["linux"],
    handlePlatformChange: vi.fn(),
    signingEnabledForBuild: false,
    setSigningEnabledForBuild: vi.fn(),
    signingConfig: null,
    signingReadiness: null,
    signingLoading: false,
    refreshSigning: vi.fn(),
    locationMode: "custom",
    setLocationMode: vi.fn(),
    standardOutputPath: "/desktop-builds",
    stagingPreviewPath: "/tmp/staging",
    isCustomLocation: true,
    outputPath: "/bundles/secrets-manager",
    setOutputPath: vi.fn(),
    isSubmittingGenerate: false,
    isGenerating: false,
    isGenerateError: false,
    generateErrorMessage: null,
    isUpdateMode: false,
    validationErrors: [],
    clearValidationErrors: vi.fn(),
    modals: {},
    closeModal: vi.fn(),
    scenariosData: { scenarios: [] },
    allowedServerTypes: [],
    handleDeploymentChange: vi.fn(),
    setServerType: vi.fn(),
    ...overrides,
  };
}

describe("GeneratorForm", () => {
  beforeEach(() => {
    useGeneratorFormStateMock.mockReset();
  });

  it("renders inventory-loaded bundled configuration and forwards signing navigation", () => {
    const onOpenSigningTab = vi.fn();
    useGeneratorFormStateMock.mockReturnValue(state());

    render(
      <GeneratorForm
        selectedTemplate="react-vite"
        onTemplateChange={vi.fn()}
        scenarioName="secrets-manager"
        onScenarioNameChange={vi.fn()}
        selectionSource="inventory"
        onOpenSigningTab={onOpenSigningTab}
      />,
    );

    expect(screen.getByText("Loaded: secrets-manager.")).toBeInTheDocument();
    expect(
      screen.getByText("Application: Secrets Manager Desktop"),
    ).toBeInTheDocument();
    expect(screen.getByText("Template: react-vite")).toBeInTheDocument();
    expect(
      screen.getByText("Output: /bundles/secrets-manager"),
    ).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Open signing" }));
    expect(onOpenSigningTab).toHaveBeenCalledWith("secrets-manager");
  });

  it("clears a custom path when the operator returns to standard output", () => {
    const current = state();
    useGeneratorFormStateMock.mockReturnValue(current);

    render(
      <GeneratorForm
        selectedTemplate="react-vite"
        onTemplateChange={vi.fn()}
        scenarioName="secrets-manager"
        onScenarioNameChange={vi.fn()}
        onOpenSigningTab={vi.fn()}
      />,
    );

    fireEvent.click(
      screen.getByRole("button", { name: "Use standard output" }),
    );
    expect(current.setLocationMode).toHaveBeenCalledWith("proper");
    expect(current.setOutputPath).toHaveBeenCalledWith("");
  });

  it("wires generator configuration controls to their owning state transitions", () => {
    const current = state({
      selectedScenario: {
        connection_config: { proxy_url: "https://saved.example" },
      },
    });
    useGeneratorFormStateMock.mockReturnValue(current);
    const onTemplateChange = vi.fn();
    const onScenarioNameChange = vi.fn();
    render(
      <GeneratorForm
        selectedTemplate="react-vite"
        onTemplateChange={onTemplateChange}
        scenarioName="secrets-manager"
        onScenarioNameChange={onScenarioNameChange}
        onOpenSigningTab={vi.fn()}
      />,
    );

    fireEvent.click(
      screen.getByRole("button", { name: "Scenario: secrets-manager" }),
    );
    fireEvent.click(screen.getByRole("button", { name: "Choose scenario" }));
    fireEvent.click(
      screen.getByRole("button", { name: "Load saved connection" }),
    );
    fireEvent.click(screen.getByRole("button", { name: "Choose framework" }));
    fireEvent.click(screen.getByRole("button", { name: "Choose template" }));
    fireEvent.click(screen.getByRole("button", { name: "Choose deployment" }));
    fireEvent.click(
      screen.getByRole("button", { name: "Connection selection" }),
    );
    fireEvent.click(screen.getByRole("button", { name: "Change runtime" }));
    fireEvent.click(screen.getByRole("checkbox", { name: "Windows" }));
    fireEvent.click(
      screen.getByRole("button", { name: "Enable build signing" }),
    );
    fireEvent.click(screen.getByRole("button", { name: "Refresh signing" }));
    fireEvent.click(
      screen.getByRole("button", { name: "Output: /bundles/secrets-manager" }),
    );
    fireEvent.click(
      screen.getByRole("button", { name: "Configuration modals" }),
    );

    expect(current.setScenarioLocked).toHaveBeenCalledWith(false);
    expect(current.openModal).toHaveBeenCalledWith("scenario");
    expect(current.applySavedConnection).toHaveBeenCalledWith({
      proxy_url: "https://saved.example",
    });
    expect(current.openModal).toHaveBeenCalledWith("framework");
    expect(current.openModal).toHaveBeenCalledWith("template");
    expect(current.openModal).toHaveBeenCalledWith("deployment");
    expect(current.setProxyUrl).toHaveBeenCalledWith("https://remote.example");
    expect(current.setAutoManageTier1).toHaveBeenCalledWith(false);
    expect(current.setVrooliBinaryPath).toHaveBeenCalledWith("/bin/vrooli");
    expect(current.setServerPort).toHaveBeenCalledWith(4000);
    expect(current.setLocalServerPath).toHaveBeenCalledWith("api/main.js");
    expect(current.setLocalApiEndpoint).toHaveBeenCalledWith(
      "http://127.0.0.1:4000",
    );
    expect(current.handlePlatformChange).toHaveBeenCalled();
    expect(current.setSigningEnabledForBuild).toHaveBeenCalledWith(true);
    expect(current.refreshSigning).toHaveBeenCalled();
    expect(current.setOutputPath).toHaveBeenCalledWith("/custom/output");
    expect(onScenarioNameChange).toHaveBeenCalledWith("other");
    expect(onTemplateChange).toHaveBeenCalledWith("minimal");
    expect(current.setFramework).toHaveBeenCalledWith("electron");
    expect(current.handleDeploymentChange).toHaveBeenCalledWith("bundled");
    expect(current.setServerType).toHaveBeenCalledWith("node");
    expect(current.closeModal).toHaveBeenCalledWith("scenario");
  });
});
