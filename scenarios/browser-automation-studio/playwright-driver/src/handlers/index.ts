/**
 * Handlers Module - Instruction Execution for Browser Automation
 *
 * ┌─────────────────────────────────────────────────────────────────────────┐
 * │ FILE GUIDE:                                                             │
 * │                                                                         │
 * │ CORE (stable, rarely changes):                                          │
 * │   base.ts      - InstructionHandler interface, HandlerContext/Result    │
 * │   registry.ts  - Handler lookup by instruction type                     │
 * │                                                                         │
 * │ HANDLERS (one file per instruction category):                           │
 * │   navigation.ts    - goto, goBack, goForward, reload                    │
 * │   interaction.ts   - click, hover, type, focus, blur                    │
 * │   wait.ts          - wait-for-selector, wait-for-timeout, etc.          │
 * │   assertion.ts     - assert-element-exists, assert-text-contains, etc.  │
 * │   extraction.ts    - extract-text, extract-attribute, etc.              │
 * │   screenshot.ts    - screenshot                                         │
 * │   scroll.ts        - scroll-to, scroll-by                               │
 * │   frame.ts         - enter-frame, exit-frame                            │
 * │   tab.ts           - new-tab, switch-tab, close-tab                     │
 * │   select.ts        - select-option                                      │
 * │   keyboard.ts      - press-key, type-text                               │
 * │   cookie-storage.ts - get/set cookies, localStorage                     │
 * │   gesture.ts       - drag, swipe                                        │
 * │   upload.ts        - set-input-files                                    │
 * │   download.ts      - download files                                     │
 * │   network.ts       - network-mock, wait-for-response                    │
 * │   device.ts        - set-viewport, emulate-device                       │
 * └─────────────────────────────────────────────────────────────────────────┘
 *
 * ADDING A NEW INSTRUCTION TYPE:
 * 1. Create handler file implementing InstructionHandler (see base.ts)
 * 2. Add Zod schema to types/instruction.ts for parameter validation
 * 3. Export handler from this index file
 * 4. Register handler in server.ts:registerHandlers()
 */

// Core abstractions
export * from './base';
export * from './registry';

// Navigation & Frames
export * from './navigation';
export * from './frame';
export * from './tab';

// Interaction
export * from './interaction';
export * from './gesture';
export * from './keyboard';
export * from './select';

// Wait & Assert
export * from './wait';
export * from './assertion';

// Data
export * from './extraction';
export * from './screenshot';
export * from './cookie-storage';

// IO
export * from './upload';
export * from './download';
export * from './scroll';

// Network & Device
export * from './network';
export * from './device';
