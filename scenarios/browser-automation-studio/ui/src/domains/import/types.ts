/**
 * Shared types for import functionality
 *
 * Used across project, asset, and routine import modals and hooks.
 */

/** Import workflow phase */
export type ImportPhase =
  | 'idle'
  | 'selecting'
  | 'validating'
  | 'previewing'
  | 'importing'
  | 'complete'
  | 'error';

/** Folder entry from filesystem scan */
export interface FolderEntry {
  /** Display name */
  name: string;
  /** Absolute path */
  path: string;
  /** Whether this is a detectable import target (project, workflow file, etc.) */
  isTarget: boolean;
  /** Whether this is already registered in the database */
  isRegistered: boolean;
  /** Database ID if registered */
  registeredId?: string;
  /** Suggested name from metadata */
  suggestedName?: string;
}

/** Result from folder scanning */
export interface ScanResult {
  /** Current path being scanned */
  path: string;
  /** Parent path (null if at root) */
  parent: string | null;
  /** Default starting path for this scan mode */
  defaultRoot: string;
  /** Scanned entries */
  entries: FolderEntry[];
}

/** Validation result for imported items */
export interface ValidationResult {
  isValid: boolean;
  errors: ValidationError[];
  warnings: ValidationWarning[];
}

export interface ValidationError {
  code: string;
  message: string;
  field?: string;
}

export interface ValidationWarning {
  code: string;
  message: string;
  field?: string;
}

/** Validation check status from API */
export type ValidationCheckStatus = 'pass' | 'warn' | 'error' | 'info';

/** Single validation check from API */
export interface ValidationCheck {
  id: string;
  status: ValidationCheckStatus;
  label: string;
  description: string;
  context?: Record<string, unknown>;
}

/** Validation summary from API - single source of truth for all validation info */
export interface ValidationSummary {
  overall_status: ValidationCheckStatus;
  pass_count: number;
  warn_count: number;
  error_count: number;
  info_count: number;
  checks: ValidationCheck[];
}

/** File information after selection */
export interface SelectedFile {
  file: File;
  /** Preview URL (blob:) for images */
  previewUrl?: string;
  /** Validation status */
  validation?: FileValidation;
}

export interface FileValidation {
  isValid: boolean;
  error?: string;
  /** Dimensions for images */
  dimensions?: { width: number; height: number };
}

/** Scan mode determines what we're looking for */
export type ScanMode = 'projects' | 'workflows' | 'assets' | 'files';

/** Drop zone variant */
export type DropZoneVariant = 'file' | 'folder' | 'both';

/** Import wizard step */
export interface ImportWizardStep {
  id: string;
  title: string;
  description?: string;
  isValid: boolean;
  isOptional?: boolean;
}

/** Common import modal props */
export interface ImportModalProps {
  isOpen: boolean;
  onClose: () => void;
}

/** Import project result - matches what the import API returns */
export interface ImportedProject {
  id: string;
  name: string;
  folder_path: string;
  created_at: string;
  updated_at: string;
}

/** Project import modal specific props */
export interface ProjectImportModalProps extends ImportModalProps {
  onSuccess?: (project: ImportedProject) => void;
}

/** Asset import modal specific props */
export interface AssetImportModalProps extends ImportModalProps {
  projectId: string;
  folder?: string;
  onSuccess?: (assetPath: string) => void;
}

/** Routine import modal specific props */
export interface RoutineImportModalProps extends ImportModalProps {
  projectId: string;
  onSuccess?: (workflow: { id: string; name: string }) => void;
}
