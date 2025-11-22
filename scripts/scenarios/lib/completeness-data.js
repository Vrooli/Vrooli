#!/usr/bin/env node
'use strict';

/**
 * Data collection for scenario completeness scoring
 * Gathers metrics from requirements, operational targets, and test results
 * @module scenarios/lib/completeness-data
 */

const fs = require('node:fs');
const path = require('node:path');

/**
 * Load service.json to get scenario category
 * @param {string} scenarioRoot - Scenario root directory
 * @returns {object} Service configuration
 */
function loadServiceConfig(scenarioRoot) {
  const servicePath = path.join(scenarioRoot, '.vrooli', 'service.json');
  try {
    const content = fs.readFileSync(servicePath, 'utf8');
    return JSON.parse(content);
  } catch (error) {
    console.warn(`Unable to load service.json from ${servicePath}: ${error.message}`);
    return { category: 'utility' };  // Default category
  }
}

/**
 * Load requirements from index.json or module.json files
 * Supports both imports-based (BAS) and module-based (deployment-manager) architectures
 * @param {string} scenarioRoot - Scenario root directory
 * @returns {array} Array of requirement objects
 */
function loadRequirements(scenarioRoot) {
  const requirementsDir = path.join(scenarioRoot, 'requirements');

  if (!fs.existsSync(requirementsDir)) {
    return [];
  }

  const requirements = [];
  const loadedFiles = new Set(); // Prevent duplicate loading

  /**
   * Load requirements from a single JSON file
   */
  const loadFromFile = (filePath) => {
    if (loadedFiles.has(filePath)) return;
    loadedFiles.add(filePath);

    try {
      const content = fs.readFileSync(filePath, 'utf8');
      const data = JSON.parse(content);
      if (Array.isArray(data.requirements)) {
        requirements.push(...data.requirements);
      }
    } catch (error) {
      console.warn(`Unable to parse ${filePath}: ${error.message}`);
    }
  };

  // Try to load from index.json first
  const indexPath = path.join(requirementsDir, 'index.json');
  if (fs.existsSync(indexPath)) {
    try {
      const content = fs.readFileSync(indexPath, 'utf8');
      const data = JSON.parse(content);

      // Load parent requirements from index.json
      if (Array.isArray(data.requirements)) {
        requirements.push(...data.requirements);
      }

      // Check for imports array (BAS architecture)
      if (Array.isArray(data.imports)) {
        for (const importPath of data.imports) {
          const fullPath = path.join(requirementsDir, importPath);
          loadFromFile(fullPath);
        }
      }
    } catch (error) {
      console.warn(`Unable to parse ${indexPath}: ${error.message}`);
    }
  }

  // Also scan for module.json files (hierarchical requirements - deployment-manager style)
  const scanModules = (dir) => {
    try {
      const entries = fs.readdirSync(dir, { withFileTypes: true });
      for (const entry of entries) {
        const fullPath = path.join(dir, entry.name);
        if (entry.isDirectory()) {
          scanModules(fullPath);
        } else if (entry.name === 'module.json') {
          loadFromFile(fullPath);
        }
      }
    } catch (error) {
      // Ignore directory read errors
    }
  };

  scanModules(requirementsDir);

  return requirements;
}

/**
 * Extract operational targets from requirements and sync metadata
 * Supports both OT-P0-001 format (deployment-manager) and folder-based format (BAS)
 * @param {array} requirements - Array of requirement objects
 * @param {object} syncMetadata - Sync metadata object (may contain operational_targets)
 * @returns {array} Array of unique operational target objects
 */
function extractOperationalTargets(requirements, syncMetadata) {
  const targets = new Map(); // Use Map to store target objects with IDs as keys

  // Method 1: Extract OT-P0-001 format from requirement prd_ref (deployment-manager style)
  for (const req of requirements) {
    if (req.prd_ref) {
      const match = req.prd_ref.match(/OT-[Pp][0-2]-\d{3}/);
      if (match) {
        const targetId = match[0].toUpperCase();
        if (!targets.has(targetId)) {
          targets.set(targetId, {
            id: targetId,
            type: 'prd_ref',
            requirements: []
          });
        }
        targets.get(targetId).requirements.push(req.id);
      }
    }
  }

  // Method 2: Extract from sync metadata operational_targets (BAS style)
  if (syncMetadata && syncMetadata.operational_targets && Array.isArray(syncMetadata.operational_targets)) {
    for (const ot of syncMetadata.operational_targets) {
      const targetId = ot.key || ot.folder_hint || ot.target_id;
      if (targetId && !targets.has(targetId)) {
        targets.set(targetId, {
          id: targetId,
          type: 'folder',
          status: ot.status,
          criticality: ot.criticality,
          counts: ot.counts,
          requirements: (ot.requirements || []).map(r => r.id)
        });
      }
    }
  }

  return Array.from(targets.values());
}

/**
 * Load sync metadata from coverage/sync or coverage/requirements-sync directory
 * Supports both legacy (sync/) and new (requirements-sync/latest.json) formats
 * @param {string} scenarioRoot - Scenario root directory
 * @returns {object} Full sync metadata object including requirements and operational_targets
 */
function loadSyncMetadata(scenarioRoot) {
  // Try new format first: coverage/requirements-sync/latest.json (BAS format)
  const newSyncPath = path.join(scenarioRoot, 'coverage', 'requirements-sync', 'latest.json');
  if (fs.existsSync(newSyncPath)) {
    try {
      const content = fs.readFileSync(newSyncPath, 'utf8');
      const data = JSON.parse(content);
      // Return full object with requirements, operational_targets, etc.
      return data;
    } catch (error) {
      console.warn(`Unable to parse ${newSyncPath}: ${error.message}`);
    }
  }

  // Fallback to legacy format: coverage/sync/*.json
  const legacySyncDir = path.join(scenarioRoot, 'coverage', 'sync');
  if (fs.existsSync(legacySyncDir)) {
    const metadata = { requirements: {} };
    try {
      const files = fs.readdirSync(legacySyncDir).filter(f => f.endsWith('.json'));
      for (const file of files) {
        const filePath = path.join(legacySyncDir, file);
        try {
          const content = fs.readFileSync(filePath, 'utf8');
          const data = JSON.parse(content);
          if (data.requirements) {
            Object.assign(metadata.requirements, data.requirements);
          }
        } catch (error) {
          console.warn(`Unable to parse ${filePath}: ${error.message}`);
        }
      }
    } catch (error) {
      console.warn(`Unable to read sync directory ${legacySyncDir}: ${error.message}`);
    }
    return metadata;
  }

  return { requirements: {} };
}

/**
 * Load test results from test phase outputs
 * Supports both single-file (test-results.json) and phase-based (phase-results/*.json) formats
 * @param {string} scenarioRoot - Scenario root directory
 * @returns {object} Test results summary
 */
function loadTestResults(scenarioRoot) {
  let totalPassing = 0;
  let totalFailing = 0;
  let latestTimestamp = null;

  // Try phase-based results first (BAS format: coverage/phase-results/*.json)
  const phaseResultsDir = path.join(scenarioRoot, 'coverage', 'phase-results');
  if (fs.existsSync(phaseResultsDir)) {
    try {
      const files = fs.readdirSync(phaseResultsDir).filter(f => f.endsWith('.json'));
      let hasResults = false;

      for (const file of files) {
        const filePath = path.join(phaseResultsDir, file);
        try {
          const content = fs.readFileSync(filePath, 'utf8');
          const data = JSON.parse(content);

          // Aggregate requirement-level pass/fail from phase results
          if (Array.isArray(data.requirements)) {
            hasResults = true;
            for (const req of data.requirements) {
              if (req.status === 'passed') {
                totalPassing++;
              } else if (req.status === 'failed') {
                totalFailing++;
              }
              // Skip 'skipped', 'unknown', 'not_run' - don't count as tests
            }

            // Track latest timestamp
            if (data.updated_at) {
              const timestamp = new Date(data.updated_at);
              if (!latestTimestamp || timestamp > latestTimestamp) {
                latestTimestamp = timestamp;
              }
            }
          }
        } catch (error) {
          console.warn(`Unable to parse ${filePath}: ${error.message}`);
        }
      }

      if (hasResults) {
        return {
          total: totalPassing + totalFailing,
          passing: totalPassing,
          failing: totalFailing,
          lastRun: latestTimestamp ? latestTimestamp.toISOString() : null
        };
      }
    } catch (error) {
      console.warn(`Unable to read phase results directory ${phaseResultsDir}: ${error.message}`);
    }
  }

  // Fallback to single test-results.json (legacy format)
  const testResultsPath = path.join(scenarioRoot, 'coverage', 'test-results.json');
  if (fs.existsSync(testResultsPath)) {
    try {
      const content = fs.readFileSync(testResultsPath, 'utf8');
      const data = JSON.parse(content);
      return {
        total: (data.passed || 0) + (data.failed || 0),
        passing: data.passed || 0,
        failing: data.failed || 0,
        lastRun: data.timestamp || null
      };
    } catch (error) {
      console.warn(`Unable to parse ${testResultsPath}: ${error.message}`);
    }
  }

  return { total: 0, passing: 0, failing: 0, lastRun: null };
}

/**
 * Calculate requirement pass status from sync metadata
 * @param {array} requirements - Array of requirement objects
 * @param {object} syncData - Full sync metadata object
 * @returns {object} Requirement pass statistics
 */
function calculateRequirementPass(requirements, syncData) {
  let passing = 0;
  let total = requirements.length;

  const syncMetadata = syncData.requirements || syncData;

  for (const req of requirements) {
    const reqMeta = syncMetadata[req.id];

    // Requirement is passing if:
    // 1. Requirement status is "validated", "implemented", or "complete"
    // 2. OR sync metadata exists with status "complete" or "validated"
    // 3. OR sync metadata shows all_tests_passing is true
    if (req.status === 'validated' || req.status === 'implemented' || req.status === 'complete') {
      passing++;
    } else if (reqMeta) {
      if (reqMeta.status === 'complete' || reqMeta.status === 'validated') {
        passing++;
      } else if (reqMeta.sync_metadata && reqMeta.sync_metadata.all_tests_passing === true) {
        passing++;
      }
    }
  }

  return { total, passing };
}

/**
 * Calculate operational target pass status
 * Supports both OT-P0-001 format and folder-based targets
 * @param {array} targets - Array of operational target objects
 * @param {array} requirements - Array of requirement objects
 * @param {object} syncData - Full sync metadata object
 * @returns {object} Target pass statistics
 */
function calculateTargetPass(targets, requirements, syncData) {
  const total = targets.length;
  let passing = 0;

  const syncMetadata = syncData.requirements || syncData;

  for (const target of targets) {
    let isTargetPassing = false;

    if (target.type === 'folder') {
      // Folder-based target (BAS style): use counts or status from sync metadata
      if (target.status === 'complete') {
        isTargetPassing = true;
      } else if (target.counts) {
        // Target is passing if >50% of requirements are complete
        const completeRatio = target.counts.complete / target.counts.total;
        isTargetPassing = completeRatio > 0.5;
      }
    } else {
      // OT-P0-001 style target: check linked requirements
      const targetId = target.id;
      const linkedReqs = requirements.filter(req => {
        if (!req.prd_ref) return false;
        const match = req.prd_ref.match(/OT-[Pp][0-2]-\d{3}/);
        return match && match[0].toUpperCase() === targetId;
      });

      // Target is passing if at least one linked requirement is passing
      isTargetPassing = linkedReqs.some(req => {
        const reqMeta = syncMetadata[req.id];

        // Check requirement status
        if (req.status === 'validated' || req.status === 'implemented' || req.status === 'complete') {
          return true;
        }

        // Check sync metadata
        if (reqMeta) {
          if (reqMeta.status === 'complete' || reqMeta.status === 'validated') {
            return true;
          }
          if (reqMeta.sync_metadata && reqMeta.sync_metadata.all_tests_passing === true) {
            return true;
          }
        }

        return false;
      });
    }

    if (isTargetPassing) {
      passing++;
    }
  }

  return { total, passing };
}

/**
 * All supported UI source file extensions
 */
const SOURCE_EXTENSIONS = ['.tsx', '.ts', '.jsx', '.js', '.vue', '.svelte'];

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

  // Fallback to ui/ root for flat structures (ecosystem-manager, picker-wheel, etc.)
  return flatPath;
}

/**
 * Find the main app entry point file
 * @param {string} scenarioRoot - Scenario root directory
 * @returns {string|null} Path to app entry point or null if not found
 */
function findAppEntryPoint(scenarioRoot) {
  // Try common entry point patterns in priority order
  const candidates = [
    // React/TypeScript patterns
    path.join(scenarioRoot, 'ui', 'src', 'App.tsx'),
    path.join(scenarioRoot, 'ui', 'src', 'App.jsx'),
    path.join(scenarioRoot, 'ui', 'src', 'main.tsx'),
    path.join(scenarioRoot, 'ui', 'src', 'main.jsx'),
    path.join(scenarioRoot, 'ui', 'src', 'index.tsx'),
    path.join(scenarioRoot, 'ui', 'src', 'index.jsx'),
    // Vanilla JS patterns (flat structure)
    path.join(scenarioRoot, 'ui', 'app.js'),
    path.join(scenarioRoot, 'ui', 'script.js'),
    path.join(scenarioRoot, 'ui', 'index.js'),
    path.join(scenarioRoot, 'ui', 'main.js'),
    // Vue patterns
    path.join(scenarioRoot, 'ui', 'src', 'App.vue'),
    path.join(scenarioRoot, 'ui', 'src', 'main.js')
  ];

  for (const candidate of candidates) {
    if (fs.existsSync(candidate)) {
      return candidate;
    }
  }

  return null;
}

/**
 * Find all potential routing configuration files
 * @param {string} scenarioRoot - Scenario root directory
 * @returns {array} Array of file paths that might contain routing config
 */
function findRoutingConfigFiles(scenarioRoot) {
  const uiSrcDir = getUiSourceDir(scenarioRoot);
  const files = [];

  const routingFileNames = [
    'routes.tsx', 'routes.ts', 'routes.jsx', 'routes.js',
    'router.tsx', 'router.ts', 'router.jsx', 'router.js',
    'routing.tsx', 'routing.ts', 'routing.jsx', 'routing.js'
  ];

  // Check root of src directory
  for (const fileName of routingFileNames) {
    const filePath = path.join(uiSrcDir, fileName);
    if (fs.existsSync(filePath)) {
      files.push(filePath);
    }
  }

  // Check common subdirectories
  const commonDirs = ['config', 'routing', 'router', 'navigation', 'pages'];
  for (const dir of commonDirs) {
    const dirPath = path.join(uiSrcDir, dir);
    if (fs.existsSync(dirPath)) {
      for (const fileName of routingFileNames) {
        const filePath = path.join(dirPath, fileName);
        if (fs.existsSync(filePath)) {
          files.push(filePath);
        }
      }

      // Also check for index files in these directories
      const indexFiles = ['index.tsx', 'index.ts', 'index.jsx', 'index.js'];
      for (const indexFile of indexFiles) {
        const filePath = path.join(dirPath, indexFile);
        if (fs.existsSync(filePath)) {
          files.push(filePath);
        }
      }
    }
  }

  return files;
}

/**
 * Detect if UI is still the unmodified template
 * @param {string} scenarioRoot - Scenario root directory
 * @returns {boolean} True if UI appears to be template
 */
function detectTemplateUI(scenarioRoot) {
  const appEntryPoint = findAppEntryPoint(scenarioRoot);

  if (!appEntryPoint) {
    return false; // No UI at all
  }

  try {
    const content = fs.readFileSync(appEntryPoint, 'utf8');
    const lines = content.split('\n').length;

    // Template signatures
    const hasTemplateText = content.includes('Scenario Template');
    const hasTemplateMessage = content.includes('This starter UI is intentionally minimal');
    const hasTemplateMessage2 = content.includes('Replace it with your scenario-specific');
    const isSmallFile = lines < 60; // Template is 45 lines

    // High confidence template detection
    if (hasTemplateText && hasTemplateMessage) {
      return true;
    }

    // Medium confidence - small file with template message
    if (hasTemplateMessage && hasTemplateMessage2 && isSmallFile) {
      return true;
    }

    return false;
  } catch (error) {
    console.warn(`Unable to read app entry point: ${error.message}`);
    return false;
  }
}

/**
 * Count files recursively matching given extensions
 * @param {string} dir - Directory to scan
 * @param {array} extensions - Array of extensions to match (e.g., ['.tsx', '.ts'])
 * @returns {number} Count of matching files
 */
function countFilesRecursive(dir, extensions) {
  if (!fs.existsSync(dir)) {
    return 0;
  }

  let count = 0;

  try {
    const entries = fs.readdirSync(dir, { withFileTypes: true });
    for (const entry of entries) {
      const fullPath = path.join(dir, entry.name);

      if (entry.isDirectory()) {
        // Skip node_modules, dist, build, coverage
        if (!['node_modules', 'dist', 'build', 'coverage', '.git'].includes(entry.name)) {
          count += countFilesRecursive(fullPath, extensions);
        }
      } else if (entry.isFile()) {
        const ext = path.extname(entry.name);
        if (extensions.includes(ext)) {
          count++;
        }
      }
    }
  } catch (error) {
    // Ignore read errors
  }

  return count;
}

/**
 * Get total lines of code in directory
 * @param {string} dir - Directory to scan
 * @returns {number} Total lines of code
 */
function getTotalLOC(dir) {
  if (!fs.existsSync(dir)) {
    return 0;
  }

  let totalLines = 0;

  const countLinesInFile = (filePath) => {
    try {
      const content = fs.readFileSync(filePath, 'utf8');
      return content.split('\n').length;
    } catch (error) {
      return 0;
    }
  };

  const scanDir = (directory) => {
    try {
      const entries = fs.readdirSync(directory, { withFileTypes: true });
      for (const entry of entries) {
        const fullPath = path.join(directory, entry.name);

        if (entry.isDirectory()) {
          if (!['node_modules', 'dist', 'build', 'coverage', '.git'].includes(entry.name)) {
            scanDir(fullPath);
          }
        } else if (entry.isFile()) {
          const ext = path.extname(entry.name);
          if (['.ts', '.tsx', '.js', '.jsx'].includes(ext)) {
            totalLines += countLinesInFile(fullPath);
          }
        }
      }
    } catch (error) {
      // Ignore read errors
    }
  };

  scanDir(dir);
  return totalLines;
}

/**
 * Detect routing in a single file
 * @param {string} content - File content
 * @returns {object} Routing information {hasRouting, routeCount}
 */
function detectRoutingInFile(content) {
  // Method 1: react-router based routing
  const hasRouterImport = content.includes('react-router') ||
                          content.includes('BrowserRouter') ||
                          content.includes('Routes');

  const routeMatches = content.match(/<Route\s/g);
  const reactRouterCount = routeMatches ? routeMatches.length : 0;

  if (hasRouterImport && reactRouterCount > 0) {
    return {
      hasRouting: true,
      routeCount: reactRouterCount
    };
  }

  // Method 2: State-based view navigation
  // Pattern: type AppView = "dashboard" | "project-detail" | ...
  const viewTypeMatch = content.match(/type\s+\w*[Vv]iew\w*\s*=\s*["']([^"']+)["']/);
  if (viewTypeMatch) {
    // Count the number of view options in the union type
    const viewTypeDeclaration = content.match(/type\s+\w*[Vv]iew\w*\s*=\s*([^;]+)/);
    if (viewTypeDeclaration) {
      const viewOptions = viewTypeDeclaration[1].match(/["']([^"']+)["']/g);
      if (viewOptions && viewOptions.length >= 2) {
        return {
          hasRouting: true,
          routeCount: viewOptions.length
        };
      }
    }
  }

  // Method 3: Conditional view rendering with state
  // Pattern: currentView === "view-name" or view === "view-name"
  const viewConditionMatches = content.match(/(?:current)?[Vv]iew\s*===\s*["']([^"']+)["']/g);
  if (viewConditionMatches && viewConditionMatches.length >= 2) {
    // Extract unique view names
    const uniqueViews = new Set();
    for (const match of viewConditionMatches) {
      const viewName = match.match(/["']([^"']+)["']/);
      if (viewName) {
        uniqueViews.add(viewName[1]);
      }
    }

    if (uniqueViews.size >= 2) {
      return {
        hasRouting: true,
        routeCount: uniqueViews.size
      };
    }
  }

  // Method 4: Lazy-loaded page components (indicator of multi-view app)
  // Pattern: const PageName = lazy(() => import('./pages/PageName'))
  const lazyPageMatches = content.match(/const\s+[A-Z]\w+\s*=\s*lazy\s*\(\s*\(\s*\)\s*=>\s*import\s*\(\s*['"][^'"]*(?:pages|views|screens)[^'"]*['"]\s*\)/g);
  if (lazyPageMatches && lazyPageMatches.length >= 2) {
    return {
      hasRouting: true,
      routeCount: lazyPageMatches.length
    };
  }

  return { hasRouting: false, routeCount: 0 };
}

/**
 * Detect routing usage across app entry point and routing config files
 * Supports both react-router and state-based navigation patterns
 * @param {string|null} appEntryPoint - Path to app entry point
 * @param {string} scenarioRoot - Scenario root directory
 * @returns {object} Routing information
 */
function detectRouting(appEntryPoint, scenarioRoot) {
  if (!appEntryPoint || !fs.existsSync(appEntryPoint)) {
    return { hasRouting: false, routeCount: 0 };
  }

  try {
    // Check main app entry point first
    const mainContent = fs.readFileSync(appEntryPoint, 'utf8');
    const mainResult = detectRoutingInFile(mainContent);

    if (mainResult.hasRouting) {
      return mainResult;
    }

    // If no routing found in main file, check dedicated routing files
    const routingFiles = findRoutingConfigFiles(scenarioRoot);
    for (const routingFile of routingFiles) {
      try {
        const content = fs.readFileSync(routingFile, 'utf8');
        const result = detectRoutingInFile(content);
        if (result.hasRouting) {
          return result;
        }
      } catch (error) {
        // Ignore read errors for individual routing files
      }
    }

    return { hasRouting: false, routeCount: 0 };
  } catch (error) {
    return { hasRouting: false, routeCount: 0 };
  }
}

/**
 * Extract API endpoints called from UI code
 * @param {string} scenarioRoot - Scenario root directory
 * @returns {object} API integration metrics
 */
function extractAPIEndpoints(scenarioRoot) {
  const uiSrcDir = getUiSourceDir(scenarioRoot);

  if (!fs.existsSync(uiSrcDir)) {
    return { endpoints: [], uniqueCount: 0, beyondHealth: 0 };
  }

  const endpoints = new Set();

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
            try {
              const content = fs.readFileSync(fullPath, 'utf8');

              // Look for API endpoint patterns
              // Pattern 1: '/api/...' or "/api/..."
              const apiMatches = content.matchAll(/['"](\/api\/[^'"]+)['"]/g);
              for (const match of apiMatches) {
                endpoints.add(match[1]);
              }

              // Pattern 2: `/api/${...}` template literals
              const templateMatches = content.matchAll(/`(\/api\/[^`]+)`/g);
              for (const match of templateMatches) {
                // Simplify template by removing ${...} placeholders
                const simplified = match[1].replace(/\$\{[^}]+\}/g, ':param');
                endpoints.add(simplified);
              }

              // Pattern 3: config.API_URL or similar base URL variables
              // Matches: config.API_URL + '/path', `${config.API_URL}/path`, etc.
              const configApiPattern = /(?:config\.API_URL|API_BASE|API_URL|BASE_URL|getApiBase\(\))[\s]*(?:\+[\s]*)?[`'"]([^`'"]+)[`'"]/g;
              const configMatches = content.matchAll(configApiPattern);
              for (const match of configMatches) {
                const endpoint = match[1];
                // Only include if it starts with / (relative path)
                if (endpoint && endpoint.startsWith('/') && !endpoint.startsWith('//')) {
                  // Clean up template variables
                  const cleaned = endpoint
                    .split('?')[0]  // Remove query strings
                    .replace(/\$\{[^}]+\}/g, ':param');  // Replace ${var} with :param
                  endpoints.add(cleaned);
                }
              }

              // Pattern 4: Template literals with variable base URL
              // Matches: `${config.API_URL}/path`, `${API_BASE}/path`, `${this.apiBase}/path`
              const templateVarPattern = /`\$\{(?:config\.API_URL|API_BASE|API_URL|BASE_URL|getApiBase\(\)|this\.apiBase)\}([^`]+)`/g;
              const templateVarMatches = content.matchAll(templateVarPattern);
              for (const match of templateVarMatches) {
                const endpoint = match[1];
                if (endpoint && endpoint.startsWith('/') && !endpoint.startsWith('//')) {
                  const cleaned = endpoint
                    .split('?')[0]
                    .replace(/\$\{[^}]+\}/g, ':param');
                  endpoints.add(cleaned);
                }
              }

              // Pattern 5: this.apiBase concatenation
              // Matches: this.apiBase + '/path' or this.apiBase + "/path"
              const apiBasePattern = /this\.apiBase[\s]*\+[\s]*[`'"]([^`'"]+)[`'"]/g;
              const apiBaseMatches = content.matchAll(apiBasePattern);
              for (const match of apiBaseMatches) {
                const endpoint = match[1];
                if (endpoint && endpoint.startsWith('/') && !endpoint.startsWith('//')) {
                  const cleaned = endpoint
                    .split('?')[0]
                    .replace(/\$\{[^}]+\}/g, ':param');
                  endpoints.add(cleaned);
                }
              }

              // Pattern 6: Function call arguments (buildApiUrl('/path'), apiClient('/path'))
              // Matches common wrapper functions: buildApiUrl, apiUrl, apiCall, makeRequest, api, request, etc.
              const functionCallPattern = /(?:buildApiUrl|buildUrl|apiUrl|apiCall|makeRequest|apiClient|makeApiCall|callApi|api|request)\s*\(\s*[`'"]([^`'"]+)[`'"]/g;
              const functionMatches = content.matchAll(functionCallPattern);
              for (const match of functionMatches) {
                const endpoint = match[1];
                // Accept paths starting with / (both /api/... and /catalog/... etc)
                if (endpoint && endpoint.startsWith('/') && !endpoint.startsWith('//')) {
                  const cleaned = endpoint
                    .split('?')[0]  // Remove query strings
                    .replace(/\$\{[^}]+\}/g, ':param');  // Replace ${var} with :param
                  endpoints.add(cleaned);
                }
              }
            } catch (error) {
              // Ignore read errors
            }
          }
        }
      }
    } catch (error) {
      // Ignore read errors
    }
  };

  scanForEndpoints(uiSrcDir);

  const endpointArray = Array.from(endpoints);
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

  const isTemplate = detectTemplateUI(scenarioRoot);
  const fileCount = countFilesRecursive(uiSrcDir, SOURCE_EXTENSIONS);
  const componentCount = countFilesRecursive(path.join(uiSrcDir, 'components'), SOURCE_EXTENSIONS);
  const pageCount = Math.max(
    countFilesRecursive(path.join(uiSrcDir, 'pages'), SOURCE_EXTENSIONS),
    countFilesRecursive(path.join(uiSrcDir, 'views'), SOURCE_EXTENSIONS)
  );
  const totalLOC = getTotalLOC(uiSrcDir);
  const routing = detectRouting(appEntryPoint, scenarioRoot);
  const apiIntegration = extractAPIEndpoints(scenarioRoot);

  // Get app entry point line count
  let appEntryLines = 0;
  if (appEntryPoint && fs.existsSync(appEntryPoint)) {
    try {
      const content = fs.readFileSync(appEntryPoint, 'utf8');
      appEntryLines = content.split('\n').length;
    } catch (error) {
      // Ignore
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

/**
 * Collect all metrics for a scenario
 * @param {string} scenarioRoot - Scenario root directory
 * @returns {object} Complete metrics object
 */
function collectMetrics(scenarioRoot) {
  const serviceConfig = loadServiceConfig(scenarioRoot);
  const requirements = loadRequirements(scenarioRoot);
  const syncData = loadSyncMetadata(scenarioRoot);
  const operationalTargets = extractOperationalTargets(requirements, syncData);
  const testResults = loadTestResults(scenarioRoot);
  const uiMetrics = collectUIMetrics(scenarioRoot);

  const requirementPass = calculateRequirementPass(requirements, syncData);
  const targetPass = calculateTargetPass(operationalTargets, requirements, syncData);

  return {
    scenario: path.basename(scenarioRoot),
    category: serviceConfig.category || 'utility',
    requirements: requirementPass,
    targets: targetPass,
    tests: {
      total: testResults.total,
      passing: testResults.passing
    },
    lastTestRun: testResults.lastRun,
    rawRequirements: requirements,  // Include for depth calculation
    ui: uiMetrics  // Include UI metrics
  };
}

module.exports = {
  loadServiceConfig,
  loadRequirements,
  extractOperationalTargets,
  loadSyncMetadata,
  loadTestResults,
  calculateRequirementPass,
  calculateTargetPass,
  collectMetrics
};
