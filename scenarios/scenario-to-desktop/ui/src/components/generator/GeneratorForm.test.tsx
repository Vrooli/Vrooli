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
  ScenarioSelector: ({ scenarioName, onUnlock }: { scenarioName: string; onUnlock: () => void }) => (
    <button type="button" onClick={onUnlock}>Scenario: {scenarioName}</button>
  ),
  AppMetadataSection: ({ appDisplayName }: { appDisplayName: string }) => <div>Application: {appDisplayName}</div>,
  FrameworkTemplateSection: ({ selectedTemplate }: { selectedTemplate: string }) => <div>Template: {selectedTemplate}</div>,
  SigningInlineSection: ({ onOpenSigning }: { onOpenSigning: () => void }) => (
    <button type="button" onClick={onOpenSigning}>Open signing</button>
  ),
  OutputLocationSelector: ({ onChange }: { onChange: (mode: "proper" | "custom") => void }) => (
    <>
      <button type="button" onClick={() => onChange("custom")}>Use custom output</button>
      <button type="button" onClick={() => onChange("proper")}>Use standard output</button>
    </>
  ),
  OutputPathField: ({ outputPath }: { outputPath: string }) => <div>Output: {outputPath}</div>,
  PlatformSelector: () => <div>Platform selection</div>,
  DeploymentSummarySection: () => <div>Deployment summary</div>,
  ConnectionSectionRouter: () => <div>Connection selection</div>,
  GeneratorFormHeader: () => <div>Form status</div>,
  GeneratorFormFooter: () => <button type="submit">Generate</button>,
  GeneratorModalsContainer: () => <div>Configuration modals</div>,
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
    handleSubmit: vi.fn((event: React.FormEvent) => event.preventDefault()),
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
      />
    );

    expect(screen.getByText("Loaded: secrets-manager.")).toBeInTheDocument();
    expect(screen.getByText("Application: Secrets Manager Desktop")).toBeInTheDocument();
    expect(screen.getByText("Template: react-vite")).toBeInTheDocument();
    expect(screen.getByText("Output: /bundles/secrets-manager")).toBeInTheDocument();

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
      />
    );

    fireEvent.click(screen.getByRole("button", { name: "Use standard output" }));
    expect(current.setLocationMode).toHaveBeenCalledWith("proper");
    expect(current.setOutputPath).toHaveBeenCalledWith("");
  });
});
