#!/usr/bin/env node
'use strict';

/**
 * Type definitions for the requirements system
 * These are used throughout the codebase for better IDE support and documentation
 */

/**
 * @typedef {Object} Requirement
 * @property {string} id - Unique requirement identifier (e.g., "BAS-WORKFLOW-PERSIST-CRUD")
 * @property {string} status - Declared status: 'complete'|'in_progress'|'pending'|'planned'
 * @property {string} criticality - Priority level: 'P0'|'P1'|'P2'
 * @property {string} title - Short title describing the requirement
 * @property {string} [category] - Hierarchical category (e.g., 'workflow.builder')
 * @property {string} [description] - Detailed description
 * @property {string} [prd_ref] - Reference to PRD section
 * @property {Validation[]} [validations] - Array of validation entries
 * @property {string[]} [children] - Child requirement IDs
 * @property {string[]} [tags] - Additional tags
 * @property {string[]} [depends_on] - Dependency requirement IDs
 * @property {string[]} [blocks] - Blocked requirement IDs
 * @property {string} [liveStatus] - Live test status: 'passed'|'failed'|'skipped'|'not_run'|'unknown'
 * @property {EvidenceRecord[]} [liveEvidence] - Live test evidence from phase results
 * @property {RequirementMetadata} [__meta] - Internal metadata (non-enumerable)
 */

/**
 * @typedef {Object} Validation
 * @property {string} type - Validation type: 'test'|'automation'|'manual'
 * @property {string} [ref] - File path reference (e.g., "api/main_test.go")
 * @property {string} [workflow_id] - Workflow identifier for automations
 * @property {string} status - Validation status: 'implemented'|'failing'|'planned'|'not_implemented'
 * @property {string} [phase] - Test phase: 'structure'|'dependencies'|'unit'|'integration'|'business'|'performance'
 * @property {string} [notes] - Additional notes
 * @property {string} [scenario] - Related scenario name
 * @property {string} [folder] - Related folder path
 * @property {*} [metadata] - Additional metadata
 * @property {string} [liveStatus] - Live test status from phase results
 * @property {ValidationSource} [liveSource] - Detected source information
 * @property {LiveDetails} [liveDetails] - Live test execution details
 * @property {ValidationMetadata} [__meta] - Internal metadata (non-enumerable)
 */

/**
 * @typedef {Object} EvidenceRecord
 * @property {string} id - Requirement ID
 * @property {string} status - Test status: 'passed'|'failed'|'skipped'|'not_run'
 * @property {string} phase - Test phase name
 * @property {string|null} evidence - Evidence description
 * @property {string|null} updated_at - ISO timestamp of last update
 * @property {number|null} duration_seconds - Test execution duration
 */

/**
 * @typedef {Object} ValidationSource
 * @property {string} kind - Source kind: 'phase'|'automation'
 * @property {string} name - Source name (phase name or automation slug)
 */

/**
 * @typedef {Object} LiveDetails
 * @property {string|null} updated_at - ISO timestamp of last update
 * @property {number|null} duration_seconds - Test execution duration
 * @property {RequirementEvidence} [requirement] - Requirement-specific evidence
 */

/**
 * @typedef {Object} RequirementEvidence
 * @property {string} id - Requirement ID
 * @property {string} phase - Test phase name
 * @property {string} status - Test status
 * @property {string|null} evidence - Evidence description
 */

/**
 * @typedef {Object} RequirementMetadata
 * @property {string} filePath - Path to source JSON file
 * @property {string} originalStatus - Original status before enrichment
 */

/**
 * @typedef {Object} ValidationMetadata
 * @property {string} filePath - Path to source JSON file
 */

/**
 * @typedef {Object} PhaseResult
 * @property {string} phase - Phase name
 * @property {string} status - Phase status: 'passed'|'failed'|'skipped'
 * @property {string} [updated_at] - ISO timestamp of last update
 * @property {number} [duration_seconds] - Phase execution duration
 * @property {number} [duration] - Alternative duration field
 * @property {RequirementEntry[]} [requirements] - Requirement entries from phase
 */

/**
 * @typedef {Object} RequirementEntry
 * @property {string} id - Requirement ID
 * @property {string} status - Test status
 * @property {string} [evidence] - Evidence description
 * @property {string} [updated_at] - ISO timestamp
 * @property {number} [duration_seconds] - Duration in seconds
 */

/**
 * @typedef {Object} RequirementFile
 * @property {string} path - Absolute path to requirement file
 * @property {string} relative - Path relative to scenario root
 * @property {boolean} isIndex - Whether this is an index.json file
 */

/**
 * @typedef {Object} VitestFile
 * @property {string} ref - Test file reference
 * @property {string} phase - Test phase (usually 'unit')
 * @property {string} status - Test status
 */

/**
 * @typedef {Object} SyncUpdate
 * @property {string} type - Update type: 'requirement'|'validation'|'add_validation'
 * @property {string} requirement - Requirement ID
 * @property {string} status - New status
 * @property {string} file - File path
 * @property {number} [index] - Validation index (for validation updates)
 * @property {string} [validation] - Validation reference (for add_validation)
 */

/**
 * @typedef {Object} OrphanedValidation
 * @property {string} requirement - Requirement ID
 * @property {string} ref - Validation reference
 * @property {string} phase - Test phase
 * @property {string} file - File path
 */

/**
 * @typedef {Object} SyncResult
 * @property {SyncUpdate[]} statusUpdates - Status change updates
 * @property {SyncUpdate[]} addedValidations - Added validation entries
 * @property {OrphanedValidation[]} orphanedValidations - Detected orphaned validations
 * @property {OrphanedValidation[]} removedValidations - Removed orphaned validations
 */

/**
 * @typedef {Object} Summary
 * @property {number} total - Total requirement count
 * @property {Object.<string, number>} byStatus - Count by declared status
 * @property {Object.<string, number>} liveStatus - Count by live status
 * @property {number} criticalityGap - Count of incomplete P0/P1 requirements
 */

/**
 * @typedef {Object} EnrichmentContext
 * @property {Object.<string, PhaseResult>} phaseResults - Phase results by phase name
 * @property {Object.<string, EvidenceRecord[]>} requirementEvidence - Evidence records by requirement ID
 */

/**
 * @typedef {Object} ParseOptions
 * @property {string} scenario - Scenario name
 * @property {string} format - Output format: 'json'|'markdown'|'trace'
 * @property {boolean} includePending - Include pending requirements in output
 * @property {string} output - Output file path
 * @property {string} mode - Execution mode: 'report'|'phase-inspect'|'validate'|'sync'
 * @property {string} phase - Phase name (for phase-inspect mode)
 * @property {boolean} pruneStale - Remove stale validations during sync
 */

module.exports = {
  // This file only exports type definitions via JSDoc
  // No runtime exports needed
};
