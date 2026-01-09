/**
 * Constants for import functionality
 *
 * File validation rules, size limits, and supported types.
 */

/** Maximum file size in bytes (10MB) */
export const MAX_FILE_SIZE = 10 * 1024 * 1024;

/** Maximum file size for assets in bytes (5MB) */
export const MAX_ASSET_SIZE = 5 * 1024 * 1024;

/** Maximum image dimension in pixels */
export const MAX_IMAGE_DIMENSION = 4096;

/** Allowed MIME types for asset uploads */
export const ASSET_MIME_TYPES = [
  'image/png',
  'image/jpeg',
  'image/webp',
  'image/gif',
  'application/json',
  'text/csv',
  'text/plain',
  'text/markdown',
] as const;

/** Allowed MIME types for workflow files */
export const WORKFLOW_MIME_TYPES = ['application/json'] as const;

/** File extensions for assets */
export const ASSET_EXTENSIONS = [
  '.png',
  '.jpg',
  '.jpeg',
  '.webp',
  '.gif',
  '.json',
  '.csv',
  '.txt',
  '.md',
] as const;

/** File extensions for workflow files */
export const WORKFLOW_EXTENSIONS = ['.workflow.json', '.json'] as const;

/** Extension for workflow files */
export const WORKFLOW_FILE_SUFFIX = '.workflow.json';

/** BAS project metadata path */
export const PROJECT_METADATA_PATH = '.bas/project.json';

/** Workflows directory within a project */
export const WORKFLOWS_DIR = 'workflows';

/** Debounce delay for path validation (ms) */
export const PATH_VALIDATION_DEBOUNCE_MS = 300;

/** Default projects root directory (matches store.ts PROJECTS_ROOT) */
export const DEFAULT_PROJECTS_ROOT = 'scenarios/browser-automation-studio/data/projects';

/** File type information for display */
export const FILE_TYPE_INFO: Record<
  string,
  { label: string; color: string }
> = {
  'image/png': { label: 'PNG Image', color: 'text-blue-400' },
  'image/jpeg': { label: 'JPEG Image', color: 'text-blue-400' },
  'image/webp': { label: 'WebP Image', color: 'text-blue-400' },
  'image/gif': { label: 'GIF Image', color: 'text-blue-400' },
  'application/json': { label: 'JSON', color: 'text-yellow-400' },
  'text/csv': { label: 'CSV', color: 'text-green-400' },
  'text/plain': { label: 'Text', color: 'text-gray-400' },
  'text/markdown': { label: 'Markdown', color: 'text-purple-400' },
};

/** Get file type info with fallback */
export function getFileTypeInfo(file: File): { label: string; color: string } {
  const info = FILE_TYPE_INFO[file.type];
  if (info) return info;

  // Fallback based on extension
  const ext = file.name.toLowerCase().split('.').pop();
  if (ext === 'md') return FILE_TYPE_INFO['text/markdown'];
  if (ext === 'json') return FILE_TYPE_INFO['application/json'];

  return { label: 'File', color: 'text-gray-400' };
}

/** Format file size for display */
export function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

/** Check if file type is allowed for assets */
export function isAssetTypeAllowed(mimeType: string): boolean {
  return (ASSET_MIME_TYPES as readonly string[]).includes(mimeType);
}

/** Check if file is a workflow file */
export function isWorkflowFile(filename: string): boolean {
  const lower = filename.toLowerCase();
  return lower.endsWith(WORKFLOW_FILE_SUFFIX) || lower.endsWith('.json');
}

/** Get accept string for file inputs */
export function getAcceptString(extensions: readonly string[]): string {
  return extensions.join(',');
}
