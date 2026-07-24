import type { BundleSectionHandle, BundleResult } from "../sections/bundle/BundleSection";
import { DeploymentSummarySection } from "./DeploymentSummarySection";
import { PlatformSelector } from "./PlatformSelector";
import {
  ScenarioSelector,
  FrameworkTemplateSection,
  SigningInlineSection,
  OutputLocationSelector,
  OutputPathField,
  AppMetadataSection,
  ConnectionSectionRouter,
  GeneratorFormHeader,
  GeneratorFormFooter,
  GeneratorModalsContainer,
} from ".";
import type { DeploymentMode } from "../../domain/deployment";
import type { ServerType } from "../../domain/deployment";
import { PendingChangesAlert } from "../state/PendingChangesAlert";
import { useGeneratorFormState } from "./useGeneratorFormState";

/** Exposed form state for sharing with other sections */
export interface ExposedFormState {
  bundleManifestPath: string;
  isBundled: boolean;
  bundleManifest?: unknown;
  // Bundle-related handlers for BundleSection
  onBundleManifestChange: (path: string) => void;
  onBundleComplete: (result: BundleResult) => void;
  bundleHelperRef: React.RefObject<BundleSectionHandle>;
}

/** Validation state exposed to parent for the submit button in GenerateSection */
export interface ValidationState {
  errors: import("../../domain/generator").ValidationError[];
  clearErrors: () => void;
  isPending: boolean;
  isError: boolean;
  errorMessage: string | null;
  isUpdateMode: boolean;
}

interface GeneratorFormProps {
  selectedTemplate: string;
  onTemplateChange: (template: string) => void;
  onBuildStart?: (buildId: string) => void;
  scenarioName: string;
  onScenarioNameChange: (name: string) => void;
  selectionSource?: "inventory" | "manual" | null;
  onOpenSigningTab: (scenario?: string) => void;
  formId?: string;
  showSubmit?: boolean;
  onGenerateStateChange?: (state: { pending: boolean; error: string | null }) => void;
  /** Callback when form state changes that other sections need */
  onFormStateChange?: (state: ExposedFormState) => void;
  /** Callback when submit handler is ready - allows parent to trigger form submission */
  onSubmitHandlerReady?: (submitFn: () => void) => void;
  /** Callback when validation state changes - used by GenerateSection for submit button */
  onValidationStateChange?: (state: ValidationState) => void;
}

export function GeneratorForm({
  selectedTemplate,
  onTemplateChange,
  onBuildStart,
  scenarioName,
  onScenarioNameChange,
  selectionSource,
  onOpenSigningTab,
  formId,
  showSubmit = true,
  onGenerateStateChange,
  onFormStateChange,
  onSubmitHandlerReady,
  onValidationStateChange,
}: GeneratorFormProps) {
  const state = useGeneratorFormState({
    selectedTemplate,
    onTemplateChange,
    onBuildStart,
    scenarioName,
    onScenarioNameChange,
    selectionSource,
    onOpenSigningTab,
    onGenerateStateChange,
    onFormStateChange,
    onSubmitHandlerReady,
    onValidationStateChange,
  });

  return (
    <div className="space-y-4">
      <GeneratorFormHeader
        scenarioName={scenarioName}
        validationStatus={state.validationStatus}
        createdLabel={state.draftCreatedLabel}
        updatedLabel={state.draftUpdatedLabel}
        isSaving={state.stateSaving}
        onReset={state.handleReset}
      />

      {state.isStale && state.pendingChanges.length > 0 && (
        <PendingChangesAlert
          changes={state.pendingChanges}
          onDismiss={() => {
            // Dismiss by refreshing the page or invalidating cache
          }}
        />
      )}
      <form id={formId} onSubmit={state.handleSubmit} className="space-y-4">
          {selectionSource === "inventory" && scenarioName && (
            <div className="rounded-md border border-blue-800/60 bg-blue-950/30 px-2.5 py-1.5 text-xs md:text-sm text-blue-100">
              <span className="font-semibold text-blue-50">
                Loaded: {scenarioName}.
              </span>{" "}
              <span className="text-blue-100/90">
                Settings below regenerate the desktop wrapper&mdash;scenario code stays the same.
              </span>
            </div>
          )}
          <ScenarioSelector
            scenarioName={scenarioName}
            loadingScenarios={state.loadingScenarios}
            selectedScenario={state.selectedScenario}
            onOpenScenarioModal={() => state.openModal('scenario')}
            onLoadSaved={
              state.selectedScenario?.connection_config
                ? () => state.applySavedConnection(state.selectedScenario?.connection_config)
                : undefined
            }
            locked={state.scenarioLocked}
            onUnlock={() => state.setScenarioLocked(false)}
          />

          <AppMetadataSection
            scenarioName={scenarioName}
            appDisplayName={state.appDisplayName}
            onAppDisplayNameChange={state.setAppDisplayName}
            iconPath={state.iconPath}
            onIconPathChange={state.setIconPath}
            iconPreviewUrl={state.iconPreviewUrl}
            iconPreviewError={state.iconPreviewError}
            onIconPreviewError={state.setIconPreviewError}
            appDescription={state.appDescription}
            onAppDescriptionChange={state.setAppDescription}
          />

          <FrameworkTemplateSection
            framework={state.framework}
            onOpenFrameworkModal={() => state.openModal('framework')}
            selectedTemplate={selectedTemplate}
            onOpenTemplateModal={() => state.openModal('template')}
          />

          <DeploymentSummarySection
            deploymentMode={state.deploymentMode}
            serverType={state.serverType}
            onOpenDeploymentModal={() => state.openModal('deployment')}
          />

          <ConnectionSectionRouter
            connectionDecision={state.connectionDecision}
            scenarioName={scenarioName}
            proxyUrl={state.proxyUrl}
            onProxyUrlChange={state.setProxyUrl}
            proxyHints={state.proxyHints}
            connectionTester={state.connectionTester}
            connectionResult={state.connectionResult}
            connectionError={state.connectionError}
            autoManageTier1={state.autoManageTier1}
            onAutoManageTier1Change={state.setAutoManageTier1}
            vrooliBinaryPath={state.vrooliBinaryPath}
            onVrooliBinaryPathChange={state.setVrooliBinaryPath}
            serverPort={state.serverPort}
            onServerPortChange={state.setServerPort}
            localServerPath={state.localServerPath}
            onLocalServerPathChange={state.setLocalServerPath}
            localApiEndpoint={state.localApiEndpoint}
            onLocalApiEndpointChange={state.setLocalApiEndpoint}
          />

          <PlatformSelector platforms={state.platforms} onPlatformChange={state.handlePlatformChange} />

          <SigningInlineSection
            scenarioName={scenarioName}
            signingEnabled={state.signingEnabledForBuild}
            signingConfig={state.signingConfig}
            readiness={state.signingReadiness}
            loading={state.signingLoading}
            onToggleSigning={state.setSigningEnabledForBuild}
            onOpenSigning={() => onOpenSigningTab(scenarioName)}
            onRefresh={() => {
              state.refreshSigning();
            }}
          />

          <OutputLocationSelector
            locationMode={state.locationMode}
            onChange={(mode) => {
              state.setLocationMode(mode);
              if (mode !== "custom") {
                state.setOutputPath("");
              }
            }}
            standardPath={state.standardOutputPath}
            stagingPreview={state.stagingPreviewPath}
          />

          {state.isCustomLocation && (
            <OutputPathField
              outputPath={state.outputPath}
              onOutputPathChange={(value) => {
                state.setOutputPath(value);
              }}
            />
          )}

          <input type="hidden" name="scenarioName" value={scenarioName} />

          {showSubmit && (
            <GeneratorFormFooter
              validationErrors={state.validationErrors}
              onDismissErrors={state.clearValidationErrors}
              isPending={state.isSubmittingGenerate || state.isGenerating}
              isError={state.isGenerateError}
              errorMessage={state.generateErrorMessage}
              isUpdateMode={state.isUpdateMode}
            />
          )}
      </form>
      <GeneratorModalsContainer
        modals={state.modals}
        closeModal={state.closeModal}
        loadingScenarios={state.loadingScenarios}
        scenarios={state.scenariosData?.scenarios ?? []}
        selectedScenarioName={scenarioName}
        onScenarioSelect={onScenarioNameChange}
        selectedTemplate={selectedTemplate}
        onTemplateSelect={onTemplateChange}
        selectedFramework={state.framework}
        onFrameworkSelect={state.setFramework}
        deploymentMode={state.deploymentMode}
        serverType={state.serverType}
        allowedServerTypes={state.allowedServerTypes}
        onDeploymentChange={(nextMode: DeploymentMode, nextServerType?: ServerType) => {
          state.handleDeploymentChange(nextMode);
          if (nextServerType) {
            state.setServerType(nextServerType);
          }
        }}
      />
    </div>
  );
}
