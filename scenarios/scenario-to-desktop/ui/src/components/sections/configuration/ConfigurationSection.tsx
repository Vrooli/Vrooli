/**
 * Configuration section - wraps GeneratorForm with the new section styling.
 * This uses the existing GeneratorForm to maintain compatibility while
 * providing the new visual structure.
 */

import { forwardRef } from "react";
import { SectionCard } from "../shared";
import { GeneratorForm } from "../../generator/GeneratorForm";

/** Exposed form state for sharing with other sections */
export interface ExposedFormState {
  bundleManifestPath: string;
  isBundled: boolean;
  bundleManifest?: unknown;
}

interface ConfigurationSectionProps {
  /** Currently selected template */
  selectedTemplate: string;
  /** Callback when template changes */
  onTemplateChange: (template: string) => void;
  /** Callback when build starts */
  onBuildStart: (buildId: string) => void;
  /** Currently selected scenario name */
  scenarioName: string;
  /** Callback when scenario name changes */
  onScenarioNameChange: (name: string) => void;
  /** How the scenario was selected */
  selectionSource?: "inventory" | "manual" | null;
  /** Callback to open the signing tab */
  onOpenSigningTab: (scenario?: string) => void;
  /** Form ID for external submit button */
  formId?: string;
  /** Whether to show the submit button */
  showSubmit?: boolean;
  /** Callback for generate state changes */
  onGenerateStateChange?: (state: { pending: boolean; error: string | null }) => void;
  /** Callback when form state changes that other sections need */
  onFormStateChange?: (state: ExposedFormState) => void;
}

export const ConfigurationSection = forwardRef<HTMLDivElement, ConfigurationSectionProps>(
  (
    {
      selectedTemplate,
      onTemplateChange,
      onBuildStart,
      scenarioName,
      onScenarioNameChange,
      selectionSource,
      onOpenSigningTab,
      formId = "generator-form",
      showSubmit = true,
      onGenerateStateChange,
      onFormStateChange,
    },
    ref
  ) => {
    return (
      <SectionCard
        ref={ref}
        sectionId="configuration"
        title="Configuration"
        subtitle="Set up your desktop application"
        variant="pipeline"
        collapsible={true}
      >
        <GeneratorForm
          selectedTemplate={selectedTemplate}
          onTemplateChange={onTemplateChange}
          onBuildStart={onBuildStart}
          scenarioName={scenarioName}
          onScenarioNameChange={onScenarioNameChange}
          selectionSource={selectionSource}
          onOpenSigningTab={onOpenSigningTab}
          formId={formId}
          showSubmit={showSubmit}
          onGenerateStateChange={onGenerateStateChange}
          onFormStateChange={onFormStateChange}
        />
      </SectionCard>
    );
  }
);

ConfigurationSection.displayName = "ConfigurationSection";
