/**
 * Types for secrets management in the deployment wizard.
 * These mirror the backend types from secrets-manager and scenario-to-cloud API.
 */

/**
 * A secret plan from secrets-manager describing how to handle a secret during deployment.
 */
export interface BundleSecretPlan {
  /** Unique identifier for this secret */
  id: string;
  /** Secret classification determining handling strategy */
  class: "per_install_generated" | "user_prompt" | "remote_fetch" | "infrastructure";
  /** Whether this secret is required for deployment */
  required: boolean;
  /** Human-readable description of what this secret is for */
  description?: string;
  /** Validation pattern for the secret value */
  format?: string;
  /** How the secret will be injected at runtime */
  target: BundleSecretTarget;
  /** Metadata for user_prompt secrets (labels, descriptions for UI) */
  prompt?: SecretPromptMetadata;
  /** Generator configuration for per_install_generated secrets */
  generator?: Record<string, unknown>;
}

export interface BundleSecretTarget {
  /** Injection type: "env" for environment variable, "file" for file path */
  type: "env" | "file";
  /** Environment variable name or file path */
  name: string;
}

export interface SecretPromptMetadata {
  /** Display label for the input field */
  label?: string;
  /** Help text describing what value to provide */
  description?: string;
}

/**
 * Summary statistics for secrets in a deployment.
 */
export interface SecretsSummary {
  /** Total number of secrets */
  total_secrets: number;
  /** Secrets that will be auto-generated during deployment */
  per_install_generated: number;
  /** Secrets that require user input */
  user_prompt: number;
  /** Secrets that will be fetched from external sources */
  remote_fetch: number;
  /** Infrastructure secrets (config values set by the deployment system) */
  infrastructure: number;
}

/**
 * Complete secrets manifest for a deployment.
 */
export interface SecretsManifest {
  /** List of all secrets needed for this deployment */
  bundle_secrets: BundleSecretPlan[];
  /** Summary statistics */
  summary: SecretsSummary;
}

/**
 * User-provided values for user_prompt secrets.
 * Keys are secret target names (env var or file path).
 */
export type ProvidedSecrets = Record<string, string>;
