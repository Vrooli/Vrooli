import type { BundleArtifact, ValidationIssue } from "../lib/api";

export type DeploymentTarget = {
  type: "vps";
  vps: {
    host: string;
    port?: number;
    user?: string;
    key_path?: string;
    workdir?: string;
  };
};

export type DeploymentManifest = {
  version: string;
  target: DeploymentTarget;
  scenario: {
    id: string;
  };
  dependencies: {
    scenarios: string[];
    resources: string[];
  };
  bundle: {
    include_packages: boolean;
    include_autoheal: boolean;
  };
  /** Dynamic port mapping - keys are port names (ui, api, ws, playwright_driver, etc.) */
  ports: Record<string, number>;
  edge: {
    domain: string;
    caddy: {
      enabled: boolean;
      email?: string;
    };
  };
};

export type WizardStep =
  | "manifest"
  | "validate"
  | "build"
  | "preflight"
  | "deploy";

export const WIZARD_STEPS: Array<{ id: WizardStep; label: string; description: string }> = [
  { id: "manifest", label: "Manifest", description: "Configure deployment settings" },
  { id: "validate", label: "Validate", description: "Check manifest for errors" },
  { id: "build", label: "Build", description: "Create deployment bundle" },
  { id: "preflight", label: "Preflight", description: "Verify target server" },
  { id: "deploy", label: "Deploy", description: "Deploy to server" },
];

export type DeploymentState = {
  // Manifest
  manifestJson: string;
  parsedManifest: DeploymentManifest | null;
  manifestError: string | null;

  // Validation
  validationIssues: ValidationIssue[] | null;
  validationError: string | null;
  isValidating: boolean;

  // Bundle
  bundleArtifact: BundleArtifact | null;
  bundleError: string | null;
  isBuildingBundle: boolean;

  // Preflight (placeholder for future)
  preflightPassed: boolean | null;
  preflightError: string | null;
  isRunningPreflight: boolean;

  // Deploy (placeholder for future)
  deploymentStatus: "idle" | "deploying" | "success" | "failed";
  deploymentError: string | null;
};

export const DEFAULT_MANIFEST: DeploymentManifest = {
  version: "1.0.0",
  target: {
    type: "vps",
    vps: {
      host: "203.0.113.10",
    },
  },
  scenario: {
    id: "",
  },
  dependencies: {
    scenarios: [],
    resources: [],
  },
  bundle: {
    include_packages: true,
    include_autoheal: true,
  },
  ports: {},
  edge: {
    domain: "example.com",
    caddy: {
      enabled: true,
      email: "",
    },
  },
};

export const DEFAULT_MANIFEST_JSON = JSON.stringify(DEFAULT_MANIFEST, null, 2);
