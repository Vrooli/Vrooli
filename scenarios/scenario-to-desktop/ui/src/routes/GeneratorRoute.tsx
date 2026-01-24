/**
 * Generator route - thin wrapper component.
 * Uses URL state to provide scenario context to existing page components.
 *
 * This follows the pattern: Route (thin) -> Hook (fat) -> Page (presentational)
 *
 * This route integrates with the existing GeneratorPage (full layout) or
 * can be used to compose the useGeneratorPage hook with custom UI.
 */

import { useCallback, useState } from "react";
import { useUrlState } from "../hooks/useUrlState";
import { GeneratorPage } from "../pages/GeneratorPage";

export interface GeneratorRouteProps {
  /** Callback when build starts (navigates to build tab) */
  onBuildStart?: (buildId: string) => void;
  /** Callback to open signing tab */
  onOpenSigningTab?: (scenario?: string) => void;
  /** Current build ID if available */
  buildId?: string | null;
  /** Initial template selection */
  initialTemplate?: string;
  /** Selection source (inventory or manual) */
  selectionSource?: "inventory" | "manual" | null;
}

/**
 * Route component that manages URL state and renders GeneratorPage.
 * This provides a clean integration point for URL-based state management.
 */
export function GeneratorRoute(props: GeneratorRouteProps) {
  const {
    onBuildStart,
    onOpenSigningTab,
    buildId = null,
    initialTemplate = "basic",
    selectionSource = null,
  } = props;

  // URL state management
  const urlState = useUrlState();
  const { scenarioName, setScenarioName } = urlState;

  // Local state for template (not in URL)
  const [selectedTemplate, setSelectedTemplate] = useState(initialTemplate);

  // Callbacks for URL state changes
  const handleScenarioNameChange = useCallback(
    (name: string) => {
      setScenarioName(name);
    },
    [setScenarioName]
  );

  const handleTemplateChange = useCallback(
    (template: string) => {
      setSelectedTemplate(template);
    },
    []
  );

  const handleBuildStart = useCallback(
    (buildId: string) => {
      onBuildStart?.(buildId);
    },
    [onBuildStart]
  );

  const handleOpenSigningTab = useCallback(
    (scenario?: string) => {
      onOpenSigningTab?.(scenario);
    },
    [onOpenSigningTab]
  );

  return (
    <GeneratorPage
      scenarioName={scenarioName || ""}
      selectedTemplate={selectedTemplate || "basic"}
      selectionSource={selectionSource}
      onScenarioNameChange={handleScenarioNameChange}
      onTemplateChange={handleTemplateChange}
      onOpenSigningTab={handleOpenSigningTab}
      buildId={buildId}
      onBuildStart={handleBuildStart}
    />
  );
}
