/**
 * Screenshot Module
 *
 * This module provides screenshot capture and annotation for the vision agent.
 */

// Types
export * from './types';

// Capture
export {
  createScreenshotCapture,
  captureScreenshot,
} from './capture';

// Annotator
export {
  createElementAnnotator,
  extractInteractiveElements,
  formatElementLabelsForPrompt,
  type AnnotatorConfig,
} from './annotate';
