'use strict';

/**
 * Constants and configuration for completeness scoring system
 * @module scenarios/lib/constants
 */

/**
 * File extensions for UI source files
 * Used for counting components, pages, and calculating LOC
 */
const SOURCE_EXTENSIONS = ['.tsx', '.ts', '.jsx', '.js', '.vue', '.svelte'];

/**
 * File extensions specifically for LOC counting
 * Subset of SOURCE_EXTENSIONS focusing on code files
 */
const CODE_EXTENSIONS = ['.ts', '.tsx', '.js', '.jsx'];

/**
 * Directories to exclude from file scanning
 * These are build artifacts, dependencies, or git metadata
 */
const EXCLUDED_DIRS = ['node_modules', 'dist', 'build', 'coverage', '.git'];

/**
 * Template detection configuration
 */
const TEMPLATE_DETECTION = {
  /**
   * Maximum number of lines for a file to be considered a template
   * Template App.tsx is typically 45 lines
   */
  MAX_LINES: 60,

  /**
   * Text signatures that indicate template UI
   * If any of these are found, the file is likely a template
   */
  SIGNATURES: [
    'Scenario Template',
    'This starter UI is intentionally minimal',
    'Replace it with your scenario-specific'
  ]
};

/**
 * Operational target pass threshold
 * For folder-based targets, pass rate must exceed this ratio
 */
const TARGET_PASS_THRESHOLD = 0.5;

/**
 * Requirement status values that indicate a requirement is passing
 */
const PASSING_REQUIREMENT_STATUSES = ['validated', 'implemented', 'complete'];

/**
 * Sync metadata status values that indicate completion
 */
const PASSING_SYNC_STATUSES = ['complete', 'validated'];

/**
 * Common entry point file patterns (in priority order)
 * Used to find the main app file for template detection
 */
const ENTRY_POINT_PATTERNS = [
  // React/TypeScript patterns
  'ui/src/App.tsx',
  'ui/src/App.jsx',
  'ui/src/main.tsx',
  'ui/src/main.jsx',
  'ui/src/index.tsx',
  'ui/src/index.jsx',
  // Vanilla JS patterns (flat structure)
  'ui/app.js',
  'ui/script.js',
  'ui/index.js',
  'ui/main.js',
  // Vue patterns
  'ui/src/App.vue',
  'ui/src/main.js'
];

/**
 * Routing configuration file names
 * Used to locate dedicated routing configuration
 */
const ROUTING_FILE_NAMES = [
  'routes.tsx', 'routes.ts', 'routes.jsx', 'routes.js',
  'router.tsx', 'router.ts', 'router.jsx', 'router.js',
  'routing.tsx', 'routing.ts', 'routing.jsx', 'routing.js'
];

/**
 * Common directories that might contain routing configuration
 */
const ROUTING_SUBDIRS = ['config', 'routing', 'router', 'navigation', 'pages'];

/**
 * Index file patterns for routing directories
 */
const INDEX_FILE_PATTERNS = ['index.tsx', 'index.ts', 'index.jsx', 'index.js'];

/**
 * Default service category when not specified
 */
const DEFAULT_SERVICE_CATEGORY = 'utility';

module.exports = {
  SOURCE_EXTENSIONS,
  CODE_EXTENSIONS,
  EXCLUDED_DIRS,
  TEMPLATE_DETECTION,
  TARGET_PASS_THRESHOLD,
  PASSING_REQUIREMENT_STATUSES,
  PASSING_SYNC_STATUSES,
  ENTRY_POINT_PATTERNS,
  ROUTING_FILE_NAMES,
  ROUTING_SUBDIRS,
  INDEX_FILE_PATTERNS,
  DEFAULT_SERVICE_CATEGORY
};
