'use strict';

/**
 * UI analyzer
 * Analyzes UI completeness including template detection, routing, and API integration
 * @module scenarios/lib/analyzers/ui-analyzer
 */

const fs = require('node:fs');
const path = require('node:path');
const {
  readTextFile,
  countFilesRecursive,
  getTotalLOC,
  findFirstFile,
  findFilesByName
} = require('../utils/file-utils');
const {
  ROUTING,
  hasReactRouterImport,
  extractViewNames,
  hasTemplateSignature,
  extractApiEndpoints
} = require('../utils/pattern-matchers');
const {
  SOURCE_EXTENSIONS,
  TEMPLATE_DETECTION,
  ENTRY_POINT_PATTERNS,
  ROUTING_FILE_NAMES,
  ROUTING_SUBDIRS,
  INDEX_FILE_PATTERNS
} = require('../constants');

/**
 * Determine the UI source directory (supports both ui/src/ and flat ui/ structure)
 * @param {string} scenarioRoot - Scenario root directory
 * @returns {string} Path to UI source directory
 */
function getUiSourceDir(scenarioRoot) {
  const srcPath = path.join(scenarioRoot, 'ui', 'src');
  const flatPath = path.join(scenarioRoot, 'ui');

  // Prefer ui/src/ if it exists and contains source files
  if (fs.existsSync(srcPath)) {
    try {
      const entries = fs.readdirSync(srcPath, { withFileTypes: true });
      const hasSourceFiles = entries.some(entry => {
        if (!entry.isFile()) return false;
        const ext = path.extname(entry.name);
        return SOURCE_EXTENSIONS.includes(ext);
      });
      if (hasSourceFiles) return srcPath;
    } catch (error) {
      // Fall through to flat path
    }
  }

  // Fallback to ui/ root for flat structures
  return flatPath;
}

/**
 * Find the main app entry point file
 * @param {string} scenarioRoot - Scenario root directory
 * @returns {string|null} Path to app entry point or null if not found
 */
function findAppEntryPoint(scenarioRoot) {
  return findFirstFile(scenarioRoot, ENTRY_POINT_PATTERNS);
}

/**
 * Detect if UI is still the unmodified template
 * @param {string} scenarioRoot - Scenario root directory
 * @param {string|null} appEntryPoint - Path to app entry point (optional)
 * @returns {boolean} True if UI appears to be template
 */
function detectTemplateUI(scenarioRoot, appEntryPoint = null) {
  const entryPoint = appEntryPoint || findAppEntryPoint(scenarioRoot);

  if (!entryPoint) {
    return false; // No UI at all
  }

  const content = readTextFile(entryPoint, { silent: true });
  if (!content) {
    return false;
  }

  const lines = content.split('\n').length;

  // High confidence template detection
  if (hasTemplateSignature(content)) {
    return true;
  }

  // Medium confidence - small file with template message indicators
  const hasTemplateMessage = content.includes('This starter UI is intentionally minimal');
  const hasTemplateMessage2 = content.includes('Replace it with your scenario-specific');
  const isSmallFile = lines < TEMPLATE_DETECTION.MAX_LINES;

  if (hasTemplateMessage && hasTemplateMessage2 && isSmallFile) {
    return true;
  }

  return false;
}

/**
 * Detect React Router-based routing
 * @param {string} content - File content
 * @returns {object} Routing information {hasRouting, routeCount}
 */
function detectReactRouter(content) {
  const hasRouterImport = hasReactRouterImport(content);
  const routeMatches = content.match(ROUTING.reactRouteComponent);
  const reactRouterCount = routeMatches ? routeMatches.length : 0;

  if (hasRouterImport && reactRouterCount > 0) {
    return {
      hasRouting: true,
      routeCount: reactRouterCount
    };
  }

  return { hasRouting: false, routeCount: 0 };
}

/**
 * Detect state-based view navigation via type declarations
 * @param {string} content - File content
 * @returns {object} Routing information {hasRouting, routeCount}
 */
function detectStateBasedViews(content) {
  // Pattern: type AppView = "dashboard" | "project-detail" | ...
  const viewTypeMatch = content.match(/type\s+\w*[Vv]iew\w*\s*=\s*["']([^"']+)["']/);
  if (viewTypeMatch) {
    // Count the number of view options in the union type
    const viewTypeDeclaration = content.match(ROUTING.viewTypeDeclaration);
    if (viewTypeDeclaration) {
      const viewOptions = viewTypeDeclaration[1].match(ROUTING.viewTypeOptions);
      if (viewOptions && viewOptions.length >= 2) {
        return {
          hasRouting: true,
          routeCount: viewOptions.length
        };
      }
    }
  }

  return { hasRouting: false, routeCount: 0 };
}

/**
 * Detect conditional view rendering with state
 * @param {string} content - File content
 * @returns {object} Routing information {hasRouting, routeCount}
 */
function detectConditionalViews(content) {
  // Pattern: currentView === "view-name" or view === "view-name"
  const uniqueViews = extractViewNames(content);

  if (uniqueViews.size >= 2) {
    return {
      hasRouting: true,
      routeCount: uniqueViews.size
    };
  }

  return { hasRouting: false, routeCount: 0 };
}

/**
 * Detect lazy-loaded page components (indicator of multi-view app)
 * @param {string} content - File content
 * @returns {object} Routing information {hasRouting, routeCount}
 */
function detectLazyLoadedPages(content) {
  // Pattern: const PageName = lazy(() => import('./pages/PageName'))
  const lazyPageMatches = content.match(ROUTING.lazyLoadedPages);

  if (lazyPageMatches && lazyPageMatches.length >= 2) {
    return {
      hasRouting: true,
      routeCount: lazyPageMatches.length
    };
  }

  return { hasRouting: false, routeCount: 0 };
}

/**
 * Detect routing in a single file using multiple detection methods
 * @param {string} content - File content
 * @returns {object} Routing information {hasRouting, routeCount}
 */
function detectRoutingInFile(content) {
  // Try each detection method in priority order
  const methods = [
    detectReactRouter,
    detectStateBasedViews,
    detectConditionalViews,
    detectLazyLoadedPages
  ];

  for (const method of methods) {
    const result = method(content);
    if (result.hasRouting) {
      return result;
    }
  }

  return { hasRouting: false, routeCount: 0 };
}

/**
 * Find all potential routing configuration files
 * @param {string} scenarioRoot - Scenario root directory
 * @returns {array} Array of file paths that might contain routing config
 */
function findRoutingConfigFiles(scenarioRoot) {
  const uiSrcDir = getUiSourceDir(scenarioRoot);
  const files = [];

  // Check root of src directory
  files.push(...findFilesByName(uiSrcDir, ROUTING_FILE_NAMES));

  // Check common subdirectories
  for (const subdir of ROUTING_SUBDIRS) {
    const subdirPath = path.join(uiSrcDir, subdir);
    if (fs.existsSync(subdirPath)) {
      // Check for routing files
      files.push(...findFilesByName(subdirPath, ROUTING_FILE_NAMES));

      // Also check for index files in these directories
      files.push(...findFilesByName(subdirPath, INDEX_FILE_PATTERNS));
    }
  }

  return files;
}

/**
 * Detect routing usage across app entry point and routing config files
 * Supports both react-router and state-based navigation patterns
 * @param {string} scenarioRoot - Scenario root directory
 * @param {string|null} appEntryPoint - Path to app entry point (optional)
 * @returns {object} Routing information {hasRouting, routeCount}
 */
function detectRouting(scenarioRoot, appEntryPoint = null) {
  const entryPoint = appEntryPoint || findAppEntryPoint(scenarioRoot);

  if (!entryPoint || !fs.existsSync(entryPoint)) {
    return { hasRouting: false, routeCount: 0 };
  }

  // Check main app entry point first
  const mainContent = readTextFile(entryPoint, { silent: true });
  if (mainContent) {
    const mainResult = detectRoutingInFile(mainContent);
    if (mainResult.hasRouting) {
      return mainResult;
    }
  }

  // If no routing found in main file, check dedicated routing files
  const routingFiles = findRoutingConfigFiles(scenarioRoot);
  for (const routingFile of routingFiles) {
    const content = readTextFile(routingFile, { silent: true });
    if (content) {
      const result = detectRoutingInFile(content);
      if (result.hasRouting) {
        return result;
      }
    }
  }

  return { hasRouting: false, routeCount: 0 };
}

/**
 * Extract API endpoints called from UI code
 * @param {string} scenarioRoot - Scenario root directory
 * @returns {object} API integration metrics {endpoints, uniqueCount, beyondHealth}
 */
function extractAPIEndpointsFromUI(scenarioRoot) {
  const uiSrcDir = getUiSourceDir(scenarioRoot);

  if (!fs.existsSync(uiSrcDir)) {
    return { endpoints: [], uniqueCount: 0, beyondHealth: 0 };
  }

  const allEndpoints = new Set();

  const scanForEndpoints = (dir) => {
    try {
      const entries = fs.readdirSync(dir, { withFileTypes: true });
      for (const entry of entries) {
        const fullPath = path.join(dir, entry.name);

        if (entry.isDirectory()) {
          if (!['node_modules', 'dist', 'build', 'coverage'].includes(entry.name)) {
            scanForEndpoints(fullPath);
          }
        } else if (entry.isFile()) {
          const ext = path.extname(entry.name);
          if (['.ts', '.tsx', '.js', '.jsx'].includes(ext)) {
            const content = readTextFile(fullPath, { silent: true });
            if (content) {
              const endpoints = extractApiEndpoints(content);
              endpoints.forEach(ep => allEndpoints.add(ep));
            }
          }
        }
      }
    } catch (error) {
      // Silently ignore read errors
    }
  };

  scanForEndpoints(uiSrcDir);

  const endpointArray = Array.from(allEndpoints);
  const beyondHealth = endpointArray.filter(e =>
    !e.includes('/health') && !e.includes('/status')
  ).length;

  return {
    endpoints: endpointArray,
    uniqueCount: endpointArray.length,
    beyondHealth: beyondHealth
  };
}

/**
 * Collect UI completeness metrics
 * @param {string} scenarioRoot - Scenario root directory
 * @returns {object} UI metrics
 */
function collectUIMetrics(scenarioRoot) {
  const uiSrcDir = getUiSourceDir(scenarioRoot);
  const appEntryPoint = findAppEntryPoint(scenarioRoot);

  const isTemplate = detectTemplateUI(scenarioRoot, appEntryPoint);
  const fileCount = countFilesRecursive(uiSrcDir, SOURCE_EXTENSIONS);
  const componentCount = countFilesRecursive(path.join(uiSrcDir, 'components'), SOURCE_EXTENSIONS);
  const pageCount = Math.max(
    countFilesRecursive(path.join(uiSrcDir, 'pages'), SOURCE_EXTENSIONS),
    countFilesRecursive(path.join(uiSrcDir, 'views'), SOURCE_EXTENSIONS)
  );
  const totalLOC = getTotalLOC(uiSrcDir);
  const routing = detectRouting(scenarioRoot, appEntryPoint);
  const apiIntegration = extractAPIEndpointsFromUI(scenarioRoot);

  // Get app entry point line count
  let appEntryLines = 0;
  if (appEntryPoint && fs.existsSync(appEntryPoint)) {
    const content = readTextFile(appEntryPoint, { silent: true });
    if (content) {
      appEntryLines = content.split('\n').length;
    }
  }

  return {
    is_template: isTemplate,
    file_count: fileCount,
    component_count: componentCount,
    page_count: pageCount,
    total_loc: totalLOC,
    app_tsx_lines: appEntryLines,
    has_routing: routing.hasRouting,
    route_count: routing.routeCount,
    api_endpoints: apiIntegration.uniqueCount,
    api_beyond_health: apiIntegration.beyondHealth
  };
}

module.exports = {
  getUiSourceDir,
  findAppEntryPoint,
  detectTemplateUI,
  detectRouting,
  extractAPIEndpointsFromUI,
  collectUIMetrics
};
