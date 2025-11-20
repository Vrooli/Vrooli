#!/usr/bin/env node
'use strict';

/**
 * Requirement file parsing and normalization
 * @module requirements/lib/parser
 */

const fs = require('node:fs');

/**
 * Parse a requirement file and normalize its structure
 * @param {string} filePath - Path to requirement JSON file
 * @returns {{ requirements: import('./types').Requirement[] }} Parsed requirements
 * @throws {Error} If file cannot be parsed
 */
function parseRequirementFile(filePath) {
  try {
    const content = fs.readFileSync(filePath, 'utf8');
    const parsed = JSON.parse(content);

    // Extract and normalize requirements array
    const requirements = (parsed.requirements || []).map((req) => {
      // Normalize requirement structure
      const requirement = {
        id: req.id || '',
        status: req.status || 'pending',
        criticality: req.criticality || '',
        title: req.title || '',
        category: req.category || '',
        validations: (req.validation || []).map((val) => ({
          type: val.type || '',
          ref: val.ref || '',
          status: val.status || '',
          notes: val.notes || '',
          phase: val.phase || '',
          workflow_id: val.workflow_id || '',
          scenario: val.scenario || '',
          folder: val.folder || '',
          metadata: val.metadata || null,
        })),
        children: req.children || [],
        description: req.description || '',
        prd_ref: req.prd_ref || '',
        tags: req.tags || [],
        depends_on: req.depends_on || [],
        blocks: req.blocks || [],
      };

      // Add metadata for tracking (used by sync operations)
      const meta = {
        filePath,
        originalStatus: requirement.status,
        originalSyncMetadata: req._sync_metadata || null,
      };
      Object.defineProperty(requirement, '__meta', {
        value: meta,
        enumerable: false,
      });

      // Add metadata to validations
      requirement.validations.forEach((validation, index) => {
        const validationMeta = {
          filePath,
          originalSyncMetadata: (req.validation && req.validation[index] && req.validation[index]._sync_metadata) || null,
        };
        Object.defineProperty(validation, '__meta', {
          value: validationMeta,
          enumerable: false,
        });
      });

      return requirement;
    });

    return { requirements };
  } catch (error) {
    throw new Error(`Failed to parse ${filePath}: ${error.message}`);
  }
}

module.exports = {
  parseRequirementFile,
};
