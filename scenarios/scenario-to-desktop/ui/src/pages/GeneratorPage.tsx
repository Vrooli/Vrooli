/**
 * Generator page - main page for generating desktop applications.
 * Composes all sections with the new sidebar layout.
 */

import { useRef, useState, useEffect, useMemo, useCallback } from "react";
import { GeneratorLayout } from "../components/layout";
import {
  ConfigurationSection,
  BundleSection,
  PreflightSection,
  GenerateSection,
  BuildSection,
  SmokeTestSection,
  DistributionSection,
  type ExposedFormState,
  type ValidationState,
} from "../components/sections";
import { useSidebarStore, type SectionId } from "../store/sidebarStore";
// NOTE: Pipeline scenario is set by App.tsx - no need to set it here

interface GeneratorPageProps {
  /** Currently selected scenario name */
  scenarioName: string;
  /** Callback when scenario name changes */
  onScenarioNameChange: (name: string) => void;
  /** Currently selected template */
  selectedTemplate: string;
  /** Callback when template changes */
  onTemplateChange: (template: string) => void;
  /** How the scenario was selected */
  selectionSource: "inventory" | "manual" | null;
  /** Callback to open the signing tab */
  onOpenSigningTab: (scenario?: string) => void;
  /** Current build ID (if any) */
  buildId: string | null;
  /** Callback when a build starts */
  onBuildStart: (buildId: string) => void;
}

export function GeneratorPage({
  scenarioName,
  onScenarioNameChange,
  selectedTemplate,
  onTemplateChange,
  selectionSource,
  onOpenSigningTab,
  buildId,
  onBuildStart,
}: GeneratorPageProps) {
  const setActiveSection = useSidebarStore((s) => s.setActiveSection);
  const [wrapperReady, setWrapperReady] = useState(false);

  // State shared between ConfigurationSection and PreflightSection/BundleSection
  const [formState, setFormState] = useState<ExposedFormState | null>(null);
  const handleFormStateChange = useCallback((state: ExposedFormState) => {
    setFormState(state);
  }, []);

  // Submit handler exposed from GeneratorForm for use in GenerateSection
  const [submitHandler, setSubmitHandler] = useState<(() => void) | null>(null);
  const handleSubmitHandlerReady = useCallback((fn: () => void) => {
    setSubmitHandler(() => fn);
  }, []);

  // Validation state for the submit button in GenerateSection
  const [validationState, setValidationState] = useState<ValidationState | null>(null);
  const handleValidationStateChange = useCallback((state: ValidationState) => {
    setValidationState(state);
  }, []);

  // Create refs for each section - memoize the object to prevent re-creation
  const configurationRef = useRef<HTMLDivElement>(null);
  const bundleRef = useRef<HTMLDivElement>(null);
  const preflightRef = useRef<HTMLDivElement>(null);
  const generateRef = useRef<HTMLDivElement>(null);
  const buildRef = useRef<HTMLDivElement>(null);
  const smoketestRef = useRef<HTMLDivElement>(null);
  const distributionRef = useRef<HTMLDivElement>(null);

  const sectionRefs = useMemo(
    () => ({
      configuration: configurationRef,
      bundle: bundleRef,
      preflight: preflightRef,
      generate: generateRef,
      build: buildRef,
      smoketest: smoketestRef,
      distribution: distributionRef,
    }),
    []
  );

  // NOTE: Pipeline scenario is set by App.tsx via setPipelineScenario
  // No need to set it here - avoid duplicate calls

  // Track scroll position to update active section
  useEffect(() => {
    const handleScroll = () => {
      const scrollPosition = window.scrollY + 200; // Offset for header

      // Find which section is currently in view
      const sectionIds: SectionId[] = [
        "configuration",
        "bundle",
        "preflight",
        "generate",
        "build",
        "smoketest",
        "distribution",
      ];

      for (let i = sectionIds.length - 1; i >= 0; i--) {
        const sectionId = sectionIds[i];
        if (!sectionId) continue;
        const ref = sectionRefs[sectionId];
        if (ref.current) {
          const rect = ref.current.getBoundingClientRect();
          const offsetTop = rect.top + window.scrollY;
          if (scrollPosition >= offsetTop) {
            setActiveSection(sectionId);
            break;
          }
        }
      }
    };

    window.addEventListener("scroll", handleScroll, { passive: true });
    return () => window.removeEventListener("scroll", handleScroll);
  }, [setActiveSection, sectionRefs]);

  const handleGenerateComplete = (newBuildId: string) => {
    setWrapperReady(true);
    onBuildStart(newBuildId);
  };

  return (
    <GeneratorLayout sectionRefs={sectionRefs}>
      <div className="space-y-8">
        {/* Section 0: Configuration */}
        <ConfigurationSection
          ref={sectionRefs.configuration}
          selectedTemplate={selectedTemplate}
          onTemplateChange={onTemplateChange}
          onBuildStart={handleGenerateComplete}
          scenarioName={scenarioName}
          onScenarioNameChange={onScenarioNameChange}
          selectionSource={selectionSource}
          onOpenSigningTab={onOpenSigningTab}
          formId="generator-form"
          showSubmit={false}
          onFormStateChange={handleFormStateChange}
          onSubmitHandlerReady={handleSubmitHandlerReady}
          onValidationStateChange={handleValidationStateChange}
        />

        {/* Section 1: Bundle */}
        <BundleSection
          ref={sectionRefs.bundle}
          scenarioName={scenarioName}
          isBundled={formState?.isBundled ?? false}
          bundleManifestPath={formState?.bundleManifestPath ?? ""}
          onBundleManifestChange={formState?.onBundleManifestChange}
          onBundleComplete={formState?.onBundleComplete}
          bundleHelperRef={formState?.bundleHelperRef}
        />

        {/* Section 2: Preflight */}
        <PreflightSection
          ref={sectionRefs.preflight}
          scenarioName={scenarioName}
          bundleManifestPath={formState?.bundleManifestPath ?? ""}
          bundleManifest={formState?.bundleManifest}
          isBundled={formState?.isBundled ?? false}
        />

        {/* Section 3: Generate - contains the submit button */}
        <GenerateSection
          ref={sectionRefs.generate}
          scenarioName={scenarioName}
          onSubmit={submitHandler ?? undefined}
          validationErrors={validationState?.errors ?? []}
          onDismissErrors={validationState?.clearErrors}
          isPending={validationState?.isPending ?? false}
          isError={validationState?.isError ?? false}
          errorMessage={validationState?.errorMessage ?? null}
          isUpdateMode={validationState?.isUpdateMode ?? false}
        />

        {/* Section 4: Build */}
        <BuildSection
          ref={sectionRefs.build}
          scenarioName={scenarioName}
          wrapperReady={wrapperReady || Boolean(buildId)}
        />

        {/* Section 5: Smoke Test */}
        <SmokeTestSection ref={sectionRefs.smoketest} scenarioName={scenarioName} />

        {/* Section 6: Distribution */}
        <DistributionSection ref={sectionRefs.distribution} scenarioName={scenarioName} />
      </div>
    </GeneratorLayout>
  );
}
