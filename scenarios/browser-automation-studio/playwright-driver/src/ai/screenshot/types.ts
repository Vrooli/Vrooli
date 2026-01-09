/**
 * Screenshot Module Types
 *
 * STABILITY: STABLE CONTRACT
 *
 * Types for screenshot capture and element annotation.
 */

/**
 * Screenshot capture result.
 */
export interface ScreenshotResult {
  /** Raw screenshot buffer (PNG or JPEG) */
  buffer: Buffer;
  /** Media type */
  mediaType: 'image/png' | 'image/jpeg';
  /** Screenshot dimensions */
  width: number;
  height: number;
}

/**
 * Options for screenshot capture.
 */
export interface CaptureOptions {
  /** Image format */
  format?: 'png' | 'jpeg';
  /** JPEG quality (1-100) */
  quality?: number;
  /** Capture full page or just viewport */
  fullPage?: boolean;
  /** Maximum size in bytes (will reduce quality if exceeded) */
  maxSizeBytes?: number;
}

/**
 * Annotated screenshot with element labels.
 */
export interface AnnotatedScreenshotResult {
  /** Annotated screenshot buffer */
  buffer: Buffer;
  /** Media type */
  mediaType: 'image/png' | 'image/jpeg';
  /** Width */
  width: number;
  /** Height */
  height: number;
  /** Element labels drawn on the screenshot */
  labels: AnnotatedElement[];
}

/**
 * Element that has been annotated on the screenshot.
 */
export interface AnnotatedElement {
  /** Label ID (number shown on screenshot) */
  id: number;
  /** CSS selector */
  selector: string;
  /** Element tag name */
  tagName: string;
  /** Bounding box */
  bounds: {
    x: number;
    y: number;
    width: number;
    height: number;
  };
  /** Text content (truncated) */
  text?: string;
  /** ARIA role or role attribute */
  role?: string;
  /** Placeholder text */
  placeholder?: string;
  /** ARIA label */
  ariaLabel?: string;
  /** Whether the element is interactive */
  interactive: boolean;
}
