/**
 * Generator page - main page for generating desktop applications.
 * Composes all sections with the new sidebar layout.
 */

import { useRef, useState, useEffect, useMemo, useCallback } from "react";
import { GeneratorLayout } from "../components/layout";
import { GeneratorForm } from "../components/generator/GeneratorForm";
import type {
  ExposedFormState,
  ValidationState,
} from "../components/generator/types";
import {
  SectionCard,
  BundleSection,
  PreflightSection,
  GenerateSection,
  BuildSection,
  SmokeTestSection,
  DeploySection,
} from "../components/sections";
import { useSidebarStore, type SectionId } from "../store/sidebarStore";
import { buildUrl } from "../lib/api";
import { selectors } from "../consts/selectors";
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
}

export function GeneratorPage({
  scenarioName,
  onScenarioNameChange,
  selectedTemplate,
  onTemplateChange,
  selectionSource,
  onOpenSigningTab,
}: GeneratorPageProps) {
  const setActiveSection = useSidebarStore((s) => s.setActiveSection);
  const basEvidenceFixture = useMemo(
    () =>
      typeof window !== "undefined" &&
      new URLSearchParams(window.location.search).get("bas_fixture") === "1",
    [],
  );
  const [fixtureStatus, setFixtureStatus] = useState<string | null>(null);
  const [fixtureError, setFixtureError] = useState<string | null>(null);
  const runEvidenceFixture = useCallback(async () => {
    setFixtureStatus("running");
    setFixtureError(null);
    try {
      const response = await fetch(
        buildUrl("/test-fixtures/desktop-evidence"),
        {
          method: "POST",
        },
      );
      if (!response.ok) {
        throw new Error(
          `Fixture request failed with ${String(response.status)}`,
        );
      }
      const result = (await response.json()) as {
        artifact?: string;
        smoke_report?: string;
        test_root_writes?: number;
      };
      if ((result.test_root_writes ?? 0) < 3) {
        throw new Error(
          "Fixture did not record its required routed file writes",
        );
      }
      setFixtureStatus(
        `passed: ${result.artifact ?? "artifact"} and ${result.smoke_report ?? "smoke report"}; leased writes: ${String(result.test_root_writes)}`,
      );
    } catch (error) {
      setFixtureStatus(null);
      setFixtureError(error instanceof Error ? error.message : String(error));
    }
  }, []);

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
  const [validationState, setValidationState] =
    useState<ValidationState | null>(null);
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
  const deployRef = useRef<HTMLDivElement>(null);

  const sectionRefs = useMemo(
    () => ({
      configuration: configurationRef,
      bundle: bundleRef,
      preflight: preflightRef,
      generate: generateRef,
      build: buildRef,
      smoketest: smoketestRef,
      deploy: deployRef,
    }),
    [],
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
        "deploy",
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
    return () => {
      window.removeEventListener("scroll", handleScroll);
    };
  }, [setActiveSection, sectionRefs]);

  return (
    <GeneratorLayout sectionRefs={sectionRefs}>
      <div className="space-y-4 md:space-y-8">
        {basEvidenceFixture && (
          <section className="rounded-lg border border-amber-700/70 bg-amber-950/20 p-4">
            <h2 className="text-sm font-semibold text-amber-100">
              BAS isolated evidence fixture
            </h2>
            <p className="mt-1 text-xs text-amber-200/80">
              Available only to routed test-mode requests. It creates a
              deterministic leased artifact and smoke report; it never creates a
              production desktop application.
            </p>
            <button
              type="button"
              className="mt-3 rounded bg-amber-500 px-3 py-2 text-sm font-medium text-slate-950 hover:bg-amber-400 disabled:opacity-60"
              data-testid={selectors.generator.evidenceFixtureRun}
              disabled={fixtureStatus === "running"}
              onClick={() => {
                void runEvidenceFixture();
              }}
            >
              {fixtureStatus === "running"
                ? "Creating isolated evidence…"
                : "Create isolated evidence"}
            </button>
            {(fixtureStatus || fixtureError) && (
              <p
                className={
                  fixtureError
                    ? "mt-2 text-sm text-red-300"
                    : "mt-2 text-sm text-emerald-300"
                }
                data-testid={selectors.generator.evidenceFixtureResult}
              >
                {fixtureError ?? fixtureStatus}
              </p>
            )}
          </section>
        )}
        {/* Section 0: Configuration */}
        <SectionCard
          ref={sectionRefs.configuration}
          sectionId="configuration"
          title="Configuration"
          subtitle="Set up your desktop application"
          variant="pipeline"
          collapsible={true}
        >
          <GeneratorForm
            selectedTemplate={selectedTemplate}
            onTemplateChange={onTemplateChange}
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
        </SectionCard>

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
        <BuildSection ref={sectionRefs.build} scenarioName={scenarioName} />

        {/* Section 5: Smoke Test */}
        <SmokeTestSection
          ref={sectionRefs.smoketest}
          scenarioName={scenarioName}
        />

        {/* Section 6: Deploy */}
        <DeploySection ref={sectionRefs.deploy} scenarioName={scenarioName} />
      </div>
    </GeneratorLayout>
  );
}
