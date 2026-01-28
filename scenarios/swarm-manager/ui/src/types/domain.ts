/**
 * Domain types for Swarm Manager
 *
 * This module contains the core domain types that represent the business concepts.
 * These types are shared across the UI and should match the API contract.
 *
 * DOC: docs/concepts/ARCHITECTURE.md#domain-concepts
 * DOC: docs/internal/SEAMS.md#module-boundaries
 * DOC: docs/internal/INTENT.md#module-responsibilities
 */

// ============================================================================
// Ideas Domain
// ============================================================================

/**
 * Valid lifecycle states for an idea
 */
export type IdeaStatus =
  | "backlog"
  | "researching"
  | "ready"
  | "queued"
  | "in_progress"
  | "completed"
  | "archived";

/**
 * An idea represents a proposal for a new scenario in the Vrooli ecosystem.
 * Ideas are stored as git-tracked folders and progress through a defined lifecycle.
 */
export interface Idea {
  /** Unique identifier (folder name) */
  name: string;
  /** Human-readable title */
  title: string;
  /** Detailed description of the idea */
  description: string;
  /** Current lifecycle state */
  status: IdeaStatus;
  /** Priority level (1 = highest) */
  priority: number;
  /** Categorization tags */
  tags: string[];
  /** ISO timestamp of creation */
  created: string;
  /** ISO timestamp of last update */
  updated: string;
}

/**
 * File type in the idea file tree
 */
export type IdeaFileType = "file" | "directory";

/**
 * Represents a file or directory within an idea folder.
 * Used for the file tree view in the idea details page.
 */
export interface IdeaFile {
  /** File or directory name */
  name: string;
  /** Relative path from the idea root */
  path: string;
  /** Whether this is a file or directory */
  type: IdeaFileType;
  /** File size in bytes (only for files) */
  size?: number;
  /** Child files (only for directories) */
  children?: IdeaFile[];
}

// ============================================================================
// Scenarios Domain
// ============================================================================

/**
 * Valid runtime states for a scenario
 */
export type ScenarioStatus = "running" | "stopped" | "error" | "unknown";

/**
 * A scenario represents a deployed application in the Vrooli ecosystem.
 * [REQ:REQ-P0-007] Includes metadata for greenfield toggle and recommendations
 */
export interface Scenario {
  /** Unique identifier (folder name) */
  name: string;
  /** Human-readable name from service.json */
  displayName: string;
  /** Scenario description */
  description: string;
  /** Current runtime state */
  status: ScenarioStatus;
  /** Priority level for development focus */
  priority: number;
  /** Completeness score (0-100) if available */
  completenessScore?: number;
  /** Whether this is a new scenario without existing code */
  isGreenfield: boolean;
  /** Categorization tags */
  tags: string[];
  /** Whether recommendations engine is enabled for this scenario */
  recommendationsEnabled: boolean;
}

/**
 * Request to update scenario metadata
 * [REQ:REQ-P0-007] Update metadata for greenfield and recommendations
 */
export interface UpdateScenarioMetadataRequest {
  /** Set to toggle greenfield status */
  isGreenfield?: boolean;
  /** Set to enable/disable recommendations */
  recommendationsEnabled?: boolean;
}

/**
 * Response from scenario deletion
 * [REQ:REQ-P0-008] Deletion confirmation with archive status
 */
export interface DeleteScenarioResponse {
  /** Name of the deleted scenario */
  name: string;
  /** Whether the scenario was archived to ideas backlog */
  archived: boolean;
  /** Human-readable message describing the result */
  message: string;
}

// ============================================================================
// Recommendations Domain
// ============================================================================

/**
 * Valid states for a recommendation
 */
export type RecommendationStatus = "pending" | "approved" | "rejected";

/**
 * A recommendation is a system-generated suggestion for improvement.
 */
export interface Recommendation {
  /** Unique identifier */
  id: string;
  /** Target scenario name */
  scenarioName: string;
  /** Type of improvement */
  type: "test" | "feature" | "refactor" | "docs";
  /** Description of the recommendation */
  description: string;
  /** Current state */
  status: RecommendationStatus;
  /** Priority level (1 = highest) */
  priority: number;
  /** ISO timestamp of creation */
  created: string;
}

// ============================================================================
// Settings Domain
// ============================================================================

/**
 * Recommendation engine operating mode
 */
export type RecommendationMode = "off" | "suggestions" | "yolo";

/**
 * User preferences and configuration
 */
export interface Settings {
  /** UI theme preference */
  theme: "dark" | "light" | "system";
  /** Recommendation engine mode */
  recommendationMode: RecommendationMode;
  /** Optional custom focus for recommendations */
  customFocus?: string;
  /** Whether insights engine is enabled */
  insightsEnabled: boolean;
  /** Auto-analyze on scenario changes */
  insightsAutoAnalyze: boolean;
}
