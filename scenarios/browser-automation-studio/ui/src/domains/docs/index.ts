/**
 * Documentation feature - in-app documentation hub
 *
 * Provides access to:
 * - Getting Started guide
 * - Workflow Methods (Recording, AI-Assisted, Visual Builder)
 * - Sessions (browser state and fingerprinting)
 * - Replay & Export (video, GIF, JSON exports)
 * - Scheduling (automated workflow runs)
 * - Node Reference (all 30+ node types with full documentation)
 * - Schema Reference (workflow JSON schema for programmatic use)
 * - Privacy & Data (how we handle your data)
 * - About (support, feedback, contact)
 */

export { DocsHub, DocsModal } from "./DocsHub";
export { GettingStarted } from "./GettingStarted";
export { WorkflowMethods } from "./WorkflowMethods";
export { Sessions } from "./Sessions";
export { ReplayExport } from "./ReplayExport";
export { Scheduling } from "./Scheduling";
export { NodeReference } from "./NodeReference";
export { SchemaReference } from "./SchemaReference";
export { PrivacyData } from "./PrivacyData";
export { About } from "./About";
export { MarkdownRenderer } from "./MarkdownRenderer";

// Re-export content utilities for node-level help
export {
  NODE_DOCUMENTATION,
  getNodeDocumentation,
  getNodeDocumentationByCategory,
  CATEGORY_ORDER,
  type NodeDocEntry,
} from "./content/nodeDocumentation";
