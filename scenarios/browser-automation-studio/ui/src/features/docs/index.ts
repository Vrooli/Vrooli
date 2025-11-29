/**
 * Documentation feature - in-app documentation hub
 *
 * Provides access to:
 * - Getting Started guide
 * - Node Reference (all 30+ node types with full documentation)
 * - Schema Reference (workflow JSON schema for programmatic use)
 */

export { DocsHub, DocsModal } from "./DocsHub";
export { GettingStarted } from "./GettingStarted";
export { NodeReference } from "./NodeReference";
export { SchemaReference } from "./SchemaReference";
export { MarkdownRenderer } from "./MarkdownRenderer";

// Re-export content utilities for node-level help
export {
  NODE_DOCUMENTATION,
  getNodeDocumentation,
  getNodeDocumentationByCategory,
  CATEGORY_ORDER,
  type NodeDocEntry,
} from "./content/nodeDocumentation";
