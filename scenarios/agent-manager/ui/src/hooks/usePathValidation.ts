import { useState, useEffect } from "react";
import { validatePath } from "../lib/api";

/** Validation states for path fields */
export type PathValidationStatus =
  | "idle"
  | "validating"
  | "valid"
  | "invalid"
  | "outside";

export interface PathValidation {
  status: PathValidationStatus;
  message?: string;
}

/** Paths that should never be allowed as project roots or scope paths */
const DANGEROUS_PATHS = [
  "/",
  "/bin",
  "/sbin",
  "/usr",
  "/etc",
  "/var",
  "/tmp",
  "/root",
  "/home",
];

/**
 * Validates a project root path with debounced client-side + server-side checks.
 *
 * Client-side: rejects non-absolute paths and dangerous system directories.
 * Server-side: verifies existence and that it's a directory (via workspace-sandbox).
 */
export function useProjectRootValidation(projectRoot: string): PathValidation {
  const [validation, setValidation] = useState<PathValidation>({
    status: "idle",
  });

  useEffect(() => {
    if (!projectRoot.trim()) {
      setValidation({ status: "idle" });
      return;
    }

    setValidation({ status: "validating" });

    const timer = setTimeout(() => {
      void (async () => {
        const trimmed = projectRoot.trim();

        if (!trimmed.startsWith("/")) {
          setValidation({
            status: "invalid",
            message: "Path must be absolute (start with /)",
          });
          return;
        }

        const normalized = trimmed.replace(/\/+$/, "") || "/";
        if (DANGEROUS_PATHS.includes(normalized)) {
          setValidation({
            status: "invalid",
            message: "Cannot use system directories as project root",
          });
          return;
        }

        try {
          const result = await validatePath(trimmed);
          if (!result.valid) {
            setValidation({
              status: "invalid",
              message: result.error || "Invalid path",
            });
            return;
          }
          setValidation({ status: "valid", message: "Path is valid" });
        } catch {
          // API unavailable — accept format-only validation
          setValidation({ status: "valid", message: "Path format is valid" });
        }
      })();
    }, 300);

    return () => clearTimeout(timer);
  }, [projectRoot]);

  return validation;
}

/**
 * Validates a scope path with debounced client-side + server-side checks.
 *
 * Scope paths can be relative (resolved against project root) or absolute.
 * Absolute paths are checked against dangerous paths and must be within
 * the project root.
 */
export function useScopePathValidation(
  scopePath: string,
  projectRoot: string,
  defaultProjectRoot?: string
): PathValidation {
  const [validation, setValidation] = useState<PathValidation>({
    status: "idle",
  });

  useEffect(() => {
    if (!scopePath.trim()) {
      setValidation({ status: "idle" });
      return;
    }

    setValidation({ status: "validating" });

    const timer = setTimeout(() => {
      void (async () => {
        const trimmed = scopePath.trim();

        // Relative paths (like ".") are always format-valid — server resolves them
        if (!trimmed.startsWith("/")) {
          setValidation({ status: "valid", message: "Relative path" });
          return;
        }

        // Absolute path checks
        const normalized = trimmed.replace(/\/+$/, "") || "/";
        if (DANGEROUS_PATHS.includes(normalized)) {
          setValidation({
            status: "invalid",
            message: "Cannot use system directories as scope paths",
          });
          return;
        }

        // Check within project root (client-side)
        const effectiveRoot = projectRoot.trim() || defaultProjectRoot;
        if (effectiveRoot) {
          const normalizedRoot = effectiveRoot.replace(/\/+$/, "");
          if (
            normalized !== normalizedRoot &&
            !normalized.startsWith(normalizedRoot + "/")
          ) {
            setValidation({
              status: "outside",
              message: `Must be within ${effectiveRoot}`,
            });
            return;
          }
        }

        // Server-side validation
        try {
          const result = await validatePath(trimmed, effectiveRoot);
          if (!result.valid) {
            if (result.withinProjectRoot === false) {
              setValidation({
                status: "outside",
                message: result.error || `Must be within ${effectiveRoot}`,
              });
            } else {
              setValidation({
                status: "invalid",
                message: result.error || "Invalid path",
              });
            }
            return;
          }
          setValidation({ status: "valid", message: "Path is valid" });
        } catch {
          // API unavailable — accept format-only validation
          setValidation({ status: "valid", message: "Path format is valid" });
        }
      })();
    }, 300);

    return () => clearTimeout(timer);
  }, [scopePath, projectRoot, defaultProjectRoot]);

  return validation;
}
