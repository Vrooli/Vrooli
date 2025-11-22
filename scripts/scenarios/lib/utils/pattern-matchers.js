'use strict';

/**
 * Regex patterns for detecting various code structures
 * Each pattern is documented with examples and use cases
 * @module scenarios/lib/utils/pattern-matchers
 */

/**
 * Operational target patterns
 */
const OPERATIONAL_TARGETS = {
  /**
   * Matches operational target IDs in prd_ref format
   * Format: OT-P[0-2]-XXX where XXX is a 3-digit number
   * Examples:
   *   - "OT-P0-001" (P0 = critical)
   *   - "OT-P1-042" (P1 = high priority)
   *   - "OT-p2-123" (case insensitive)
   * Used in: requirement.prd_ref field
   */
  prdRef: /OT-[Pp][0-2]-\d{3}/
};

/**
 * API endpoint detection patterns
 * Used to identify API integrations in UI code
 */
const API_ENDPOINTS = {
  /**
   * Pattern 1: Simple quoted API paths
   * Matches: '/api/users', "/api/projects"
   * Examples in code:
   *   fetch('/api/users')
   *   axios.get("/api/projects")
   */
  simple: /['"](\/api\/[^'"]+)['"]/g,

  /**
   * Pattern 2: Template literal API paths
   * Matches: `/api/users/${id}`, `/api/projects/${projectId}/tasks`
   * Examples in code:
   *   fetch(`/api/users/${userId}`)
   *   Note: Placeholders are simplified to :param
   */
  template: /`(\/api\/[^`]+)`/g,

  /**
   * Pattern 3: Config-based API URLs with concatenation
   * Matches: config.API_URL + '/users', API_BASE + "/projects"
   * Examples in code:
   *   fetch(config.API_URL + '/users')
   *   axios.get(API_BASE + "/projects")
   * Captures only the path portion (after the base URL)
   */
  configUrl: /(?:config\.API_URL|API_BASE|API_URL|BASE_URL|getApiBase\(\))[\s]*(?:\+[\s]*)?[`'"]([^`'"]+)[`'"]/g,

  /**
   * Pattern 4: Template literals with variable base URLs
   * Matches: `${config.API_URL}/users`, `${API_BASE}/projects`
   * Examples in code:
   *   fetch(`${config.API_URL}/users`)
   *   axios.get(`${this.apiBase}/projects`)
   */
  templateVar: /`\$\{(?:config\.API_URL|API_BASE|API_URL|BASE_URL|getApiBase\(\)|this\.apiBase)\}([^`]+)`/g,

  /**
   * Pattern 5: this.apiBase concatenation
   * Matches: this.apiBase + '/users', this.apiBase + "/projects"
   * Examples in code:
   *   fetch(this.apiBase + '/users')
   *   Used in class-based components
   */
  apiBaseConcat: /this\.apiBase[\s]*\+[\s]*[`'"]([^`'"]+)[`'"]/g,

  /**
   * Pattern 6: Wrapper function calls
   * Matches: buildApiUrl('/users'), apiClient('/projects')
   * Examples in code:
   *   fetch(buildApiUrl('/users'))
   *   apiClient('/projects').then(...)
   * Supported function names: buildApiUrl, apiUrl, apiCall, makeRequest, etc.
   */
  functionCall: /(?:buildApiUrl|buildUrl|apiUrl|apiCall|makeRequest|apiClient|makeApiCall|callApi|api|request)\s*\(\s*[`'"]([^`'"]+)[`'"]/g
};

/**
 * Routing detection patterns
 * Used to identify navigation and routing in UI applications
 */
const ROUTING = {
  /**
   * React Router: Import detection
   * Checks if react-router library is used
   * Examples:
   *   import { BrowserRouter } from 'react-router-dom'
   *   import { Routes, Route } from 'react-router-dom'
   */
  reactRouterImports: [
    /react-router/,
    /BrowserRouter/,
    /Routes/
  ],

  /**
   * React Router: Route component usage
   * Matches: <Route path="/users" ...>, <Route path="/projects/:id" ...>
   * Examples in JSX:
   *   <Route path="/users" element={<Users />} />
   *   <Route path="/projects/:id" component={ProjectDetail} />
   */
  reactRouteComponent: /<Route\s/g,

  /**
   * State-based routing: View type definitions
   * Matches: type AppView = "dashboard" | "project-detail"
   * Examples:
   *   type AppView = "dashboard" | "settings" | "profile"
   *   type View = "home" | "about"
   * Used for apps that don't use react-router
   */
  viewTypeDeclaration: /type\s+\w*[Vv]iew\w*\s*=\s*([^;]+)/,

  /**
   * State-based routing: View type union member
   * Extracts individual view options from type declaration
   * Matches: "dashboard", "settings", "profile"
   */
  viewTypeOptions: /["']([^"']+)["']/g,

  /**
   * Conditional view rendering
   * Matches: view === "dashboard", currentView === "settings"
   * Examples in code:
   *   {view === "dashboard" && <Dashboard />}
   *   {currentView === "settings" ? <Settings /> : null}
   */
  viewCondition: /(?:current)?[Vv]iew\s*===\s*["']([^"']+)["']/g,

  /**
   * Lazy-loaded page components
   * Matches: const Dashboard = lazy(() => import('./pages/Dashboard'))
   * Examples:
   *   const Users = lazy(() => import('./pages/Users'))
   *   const Settings = lazy(() => import('./views/Settings'))
   * Indicates a multi-page application structure
   */
  lazyLoadedPages: /const\s+[A-Z]\w+\s*=\s*lazy\s*\(\s*\(\s*\)\s*=>\s*import\s*\(\s*['"][^'"]*(?:pages|views|screens)[^'"]*['"]\s*\)/g
};

/**
 * Template UI detection patterns
 */
const TEMPLATE = {
  /**
   * Template signature text patterns
   * If these exact strings are found, file is likely template
   */
  signatures: [
    /Scenario Template/,
    /This starter UI is intentionally minimal/,
    /Replace it with your scenario-specific/
  ]
};

/**
 * Helper: Check if content has any React Router imports
 * @param {string} content - File content
 * @returns {boolean} True if any React Router import is found
 */
function hasReactRouterImport(content) {
  return ROUTING.reactRouterImports.some(pattern => pattern.test(content));
}

/**
 * Helper: Extract unique view names from conditional rendering
 * @param {string} content - File content
 * @returns {Set<string>} Set of unique view names
 */
function extractViewNames(content) {
  const viewNames = new Set();
  const matches = content.matchAll(ROUTING.viewCondition);

  for (const match of matches) {
    if (match[1]) {
      viewNames.add(match[1]);
    }
  }

  return viewNames;
}

/**
 * Helper: Check if content has template signatures
 * @param {string} content - File content
 * @returns {boolean} True if any template signature is found
 */
function hasTemplateSignature(content) {
  return TEMPLATE.signatures.some(pattern => pattern.test(content));
}

/**
 * Helper: Extract API endpoints from content using all patterns
 * @param {string} content - File content
 * @returns {Set<string>} Set of unique endpoint paths
 */
function extractApiEndpoints(content) {
  const endpoints = new Set();

  // Pattern 1: Simple quoted paths
  const simpleMatches = content.matchAll(API_ENDPOINTS.simple);
  for (const match of simpleMatches) {
    endpoints.add(match[1]);
  }

  // Pattern 2: Template literals
  const templateMatches = content.matchAll(API_ENDPOINTS.template);
  for (const match of templateMatches) {
    const simplified = match[1].replace(/\$\{[^}]+\}/g, ':param');
    endpoints.add(simplified);
  }

  // Pattern 3: Config URL concatenation
  const configMatches = content.matchAll(API_ENDPOINTS.configUrl);
  for (const match of configMatches) {
    const endpoint = match[1];
    if (endpoint && endpoint.startsWith('/') && !endpoint.startsWith('//')) {
      const cleaned = endpoint
        .split('?')[0]
        .replace(/\$\{[^}]+\}/g, ':param');
      endpoints.add(cleaned);
    }
  }

  // Pattern 4: Template variables
  const templateVarMatches = content.matchAll(API_ENDPOINTS.templateVar);
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
  const apiBaseMatches = content.matchAll(API_ENDPOINTS.apiBaseConcat);
  for (const match of apiBaseMatches) {
    const endpoint = match[1];
    if (endpoint && endpoint.startsWith('/') && !endpoint.startsWith('//')) {
      const cleaned = endpoint
        .split('?')[0]
        .replace(/\$\{[^}]+\}/g, ':param');
      endpoints.add(cleaned);
    }
  }

  // Pattern 6: Function calls
  const functionMatches = content.matchAll(API_ENDPOINTS.functionCall);
  for (const match of functionMatches) {
    const endpoint = match[1];
    if (endpoint && endpoint.startsWith('/') && !endpoint.startsWith('//')) {
      const cleaned = endpoint
        .split('?')[0]
        .replace(/\$\{[^}]+\}/g, ':param');
      endpoints.add(cleaned);
    }
  }

  return endpoints;
}

module.exports = {
  OPERATIONAL_TARGETS,
  API_ENDPOINTS,
  ROUTING,
  TEMPLATE,
  // Helper functions
  hasReactRouterImport,
  extractViewNames,
  hasTemplateSignature,
  extractApiEndpoints
};
