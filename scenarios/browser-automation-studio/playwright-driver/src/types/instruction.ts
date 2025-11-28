import { z } from 'zod';

/**
 * Parameter schemas for all instruction types.
 * Uses Zod for runtime validation.
 */

// Navigation
export const NavigateParamsSchema = z.object({
  url: z.string().optional(),
  target: z.string().optional(),
  href: z.string().optional(),
  timeoutMs: z.number().optional(),
  waitUntil: z.enum(['load', 'domcontentloaded', 'networkidle']).optional(),
});
export type NavigateParams = z.infer<typeof NavigateParamsSchema>;

// Interaction
export const ClickParamsSchema = z.object({
  selector: z.string(),
  timeoutMs: z.number().optional(),
  clickCount: z.number().optional(),
  button: z.enum(['left', 'right', 'middle']).optional(),
  modifiers: z.array(z.string()).optional(),
});
export type ClickParams = z.infer<typeof ClickParamsSchema>;

export const HoverParamsSchema = z.object({
  selector: z.string(),
  timeoutMs: z.number().optional(),
});
export type HoverParams = z.infer<typeof HoverParamsSchema>;

export const TypeParamsSchema = z.object({
  selector: z.string(),
  text: z.string().optional(),
  value: z.string().optional(),
  timeoutMs: z.number().optional(),
  delay: z.number().optional(),
});
export type TypeParams = z.infer<typeof TypeParamsSchema>;

export const FocusParamsSchema = z.object({
  selector: z.string(),
  scroll: z.boolean().optional(),
  timeoutMs: z.number().optional(),
});
export type FocusParams = z.infer<typeof FocusParamsSchema>;

export const BlurParamsSchema = z.object({
  selector: z.string().optional(),
  timeoutMs: z.number().optional(),
});
export type BlurParams = z.infer<typeof BlurParamsSchema>;

// Wait
export const WaitParamsSchema = z.object({
  selector: z.string().optional(),
  timeoutMs: z.number().optional(),
  ms: z.number().optional(),
  state: z.enum(['attached', 'detached', 'visible', 'hidden']).optional(),
});
export type WaitParams = z.infer<typeof WaitParamsSchema>;

// Assert
export const AssertParamsSchema = z.object({
  selector: z.string(),
  mode: z.string().optional(),
  kind: z.string().optional(),
  expected: z.unknown().optional(),
  expectedValue: z.unknown().optional(), // Workflow schema uses expectedValue, handler checks both
  text: z.string().optional(),
  attribute: z.string().optional(),
  attr: z.string().optional(),
  attributeName: z.string().optional(), // Workflow schema uses attributeName, handler checks attribute/attr/attributeName
  value: z.string().optional(),
  contains: z.boolean().optional(),
  timeoutMs: z.number().optional(),
});
export type AssertParams = z.infer<typeof AssertParamsSchema>;

// Extraction
export const ExtractParamsSchema = z.object({
  selector: z.string(),
  timeoutMs: z.number().optional(),
});
export type ExtractParams = z.infer<typeof ExtractParamsSchema>;

export const EvaluateParamsSchema = z.object({
  script: z.string().optional(),
  expression: z.string().optional(),
  args: z.record(z.unknown()).optional(),
});
export type EvaluateParams = z.infer<typeof EvaluateParamsSchema>;

// Upload/Download
export const UploadFileParamsSchema = z.object({
  selector: z.string(),
  filePath: z.string().optional(),
  file_path: z.string().optional(),
  path: z.string().optional(),
  timeoutMs: z.number().optional(),
});
export type UploadFileParams = z.infer<typeof UploadFileParamsSchema>;

export const DownloadParamsSchema = z.object({
  selector: z.string().optional(),
  url: z.string().optional(),
  timeoutMs: z.number().optional(),
});
export type DownloadParams = z.infer<typeof DownloadParamsSchema>;

// Scroll
export const ScrollParamsSchema = z.object({
  x: z.number().optional(),
  y: z.number().optional(),
  selector: z.string().optional(),
  behavior: z.enum(['auto', 'smooth']).optional(),
});
export type ScrollParams = z.infer<typeof ScrollParamsSchema>;

// Screenshot
export const ScreenshotParamsSchema = z.object({
  fullPage: z.boolean().optional(),
  quality: z.number().optional(),
});
export type ScreenshotParams = z.infer<typeof ScreenshotParamsSchema>;

// Frame operations
export const FrameSwitchParamsSchema = z.object({
  action: z.enum(['enter', 'exit', 'parent']),
  selector: z.string().optional(),
  frameId: z.string().optional(),
  frameUrl: z.string().optional(),
  timeoutMs: z.number().optional(),
});
export type FrameSwitchParams = z.infer<typeof FrameSwitchParamsSchema>;

// Tab operations
export const TabSwitchParamsSchema = z.object({
  action: z.enum(['open', 'switch', 'close', 'list']),
  url: z.string().optional(),
  index: z.number().optional(),
  title: z.string().optional(),
  urlPattern: z.string().optional(),
});
export type TabSwitchParams = z.infer<typeof TabSwitchParamsSchema>;

// Cookie/Storage
export const CookieStorageParamsSchema = z.object({
  operation: z.enum(['get', 'set', 'delete', 'clear']),
  storageType: z.enum(['cookie', 'localStorage', 'sessionStorage']),
  key: z.string().optional(),
  name: z.string().optional(),
  value: z.string().optional(),
  cookieOptions: z
    .object({
      domain: z.string().optional(),
      path: z.string().optional(),
      expires: z.number().optional(),
      httpOnly: z.boolean().optional(),
      secure: z.boolean().optional(),
      sameSite: z.enum(['Strict', 'Lax', 'None']).optional(),
    })
    .optional(),
});
export type CookieStorageParams = z.infer<typeof CookieStorageParamsSchema>;

// Select
export const SelectParamsSchema = z.object({
  selector: z.string(),
  value: z.union([z.string(), z.array(z.string())]).optional(),
  label: z.union([z.string(), z.array(z.string())]).optional(),
  index: z.union([z.number(), z.array(z.number())]).optional(),
  timeoutMs: z.number().optional(),
});
export type SelectParams = z.infer<typeof SelectParamsSchema>;

// Keyboard
export const KeyboardParamsSchema = z.object({
  key: z.string().optional(),
  keys: z.array(z.string()).optional(),
  modifiers: z.array(z.string()).optional(),
  action: z.enum(['press', 'down', 'up']).optional(),
});
export type KeyboardParams = z.infer<typeof KeyboardParamsSchema>;

export const ShortcutParamsSchema = z.object({
  shortcut: z.string(),
  selector: z.string().optional(),
});
export type ShortcutParams = z.infer<typeof ShortcutParamsSchema>;

// Drag/Drop
export const DragDropParamsSchema = z.object({
  sourceSelector: z.string(),
  targetSelector: z.string().optional(),
  offsetX: z.number().optional(),
  offsetY: z.number().optional(),
  steps: z.number().optional(),
  delayMs: z.number().optional(),
  timeoutMs: z.number().optional(),
});
export type DragDropParams = z.infer<typeof DragDropParamsSchema>;

// Gesture
export const GestureParamsSchema = z.object({
  type: z.enum(['swipe', 'pinch', 'zoom']),
  selector: z.string().optional(),
  direction: z.enum(['up', 'down', 'left', 'right']).optional(),
  distance: z.number().optional(),
  scale: z.number().optional(),
});
export type GestureParams = z.infer<typeof GestureParamsSchema>;

// Network mock
export const NetworkMockParamsSchema = z.object({
  operation: z.enum(['mock', 'block', 'modifyRequest', 'modifyResponse', 'clear']),
  urlPattern: z.string(),
  method: z.string().optional(),
  statusCode: z.number().optional(),
  headers: z.record(z.string()).optional(),
  body: z.union([z.string(), z.record(z.unknown())]).optional(),
  delayMs: z.number().optional(),
});
export type NetworkMockParams = z.infer<typeof NetworkMockParamsSchema>;

// Device
export const RotateParamsSchema = z.object({
  orientation: z.enum(['portrait', 'landscape']),
  angle: z.union([z.literal(0), z.literal(90), z.literal(180), z.literal(270)]).optional(),
});
export type RotateParams = z.infer<typeof RotateParamsSchema>;
