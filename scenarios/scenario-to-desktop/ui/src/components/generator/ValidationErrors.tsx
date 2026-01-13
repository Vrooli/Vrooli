/**
 * Validation errors display component for the generator form.
 */

import { AlertTriangle, X } from "lucide-react";
import { Button } from "../ui/button";

export interface ValidationError {
  id: string;
  message: string;
  field?: string;
}

export interface ValidationErrorsProps {
  errors: ValidationError[];
  onDismiss?: () => void;
  className?: string;
}

export function ValidationErrors({ errors, onDismiss, className = "" }: ValidationErrorsProps) {
  if (errors.length === 0) return null;

  return (
    <div className={`rounded-lg border border-red-800/60 bg-red-950/30 p-4 ${className}`}>
      <div className="flex items-start justify-between gap-3">
        <div className="flex items-start gap-3">
          <AlertTriangle className="h-5 w-5 text-red-400 flex-shrink-0 mt-0.5" />
          <div className="space-y-2">
            <p className="text-sm font-medium text-red-200">
              Please fix the following {errors.length === 1 ? "issue" : "issues"} before generating:
            </p>
            <ul className="space-y-1 text-sm text-red-300">
              {errors.map((error) => (
                <li key={error.id} className="flex items-start gap-2">
                  <span className="text-red-400">•</span>
                  <span>{error.message}</span>
                </li>
              ))}
            </ul>
          </div>
        </div>
        {onDismiss && (
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={onDismiss}
            className="text-red-400 hover:text-red-300 hover:bg-red-900/30 -mt-1 -mr-1"
          >
            <X className="h-4 w-4" />
          </Button>
        )}
      </div>
    </div>
  );
}

/**
 * Validates form inputs and returns validation errors.
 */
export interface ValidateFormInputsParams {
  scenarioName: string;
  selectedPlatforms: string[];
  isBundled: boolean;
  requiresProxyUrl: boolean;
  bundleManifestPath: string;
  proxyUrl: string;
  appDisplayName: string;
  appDescription: string;
  locationMode: string;
  outputPath: string;
  preflightResult: unknown;
  preflightOk: boolean;
  preflightOverride: boolean;
  signingEnabledForBuild: boolean;
  signingConfig: { enabled?: boolean } | null | undefined;
  signingReadiness: { ready?: boolean; issues?: string[] } | undefined;
}

export function validateFormInputs(params: ValidateFormInputsParams): ValidationError[] {
  const errors: ValidationError[] = [];

  // Scenario selection
  if (!params.scenarioName) {
    errors.push({
      id: "no-scenario",
      message: "Please select a scenario before generating a desktop app.",
      field: "scenarioName",
    });
  }

  // Platform selection
  if (params.selectedPlatforms.length === 0) {
    errors.push({
      id: "no-platforms",
      message: "Select at least one target platform (Windows, macOS, or Linux).",
      field: "platforms",
    });
  }

  // Bundled mode requirements
  if (params.isBundled) {
    if (!params.bundleManifestPath.trim()) {
      errors.push({
        id: "no-bundle-manifest",
        message: "Provide a bundle manifest path for bundled runtime mode.",
        field: "bundleManifestPath",
      });
    }

    if (!params.preflightResult) {
      errors.push({
        id: "no-preflight",
        message: "Run preflight validation before generating a bundled desktop app.",
        field: "preflight",
      });
    } else if (!params.preflightOk && !params.preflightOverride) {
      errors.push({
        id: "preflight-failed",
        message: "Preflight validation failed. Fix the issues or enable override to continue.",
        field: "preflight",
      });
    }
  }

  // Remote server requirements
  if (params.requiresProxyUrl && !params.proxyUrl.trim()) {
    errors.push({
      id: "no-proxy-url",
      message: "Provide a proxy URL for remote server mode.",
      field: "proxyUrl",
    });
  }

  // App metadata
  if (!params.appDisplayName.trim()) {
    errors.push({
      id: "no-display-name",
      message: "Provide an app display name.",
      field: "appDisplayName",
    });
  }

  if (!params.appDescription.trim()) {
    errors.push({
      id: "no-description",
      message: "Provide an app description.",
      field: "appDescription",
    });
  }

  // Custom output path
  if (params.locationMode === "custom" && !params.outputPath.trim()) {
    errors.push({
      id: "no-output-path",
      message: "Provide an output path when using custom location mode.",
      field: "outputPath",
    });
  }

  // Signing
  if (params.signingEnabledForBuild) {
    if (!params.signingConfig || !params.signingConfig.enabled) {
      errors.push({
        id: "no-signing-config",
        message: "Signing is enabled but no signing config is saved. Open the Signing tab to add certificates.",
        field: "signing",
      });
    } else if (params.signingReadiness && !params.signingReadiness.ready) {
      const issue = params.signingReadiness.issues?.[0] || "Signing prerequisites not met.";
      errors.push({
        id: "signing-not-ready",
        message: `Signing is not ready: ${issue}`,
        field: "signing",
      });
    }
  }

  return errors;
}
