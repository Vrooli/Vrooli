/**
 * Browser Global Declarations
 *
 * These declarations provide minimal typing for browser globals
 * used within Playwright page.evaluate() callbacks.
 *
 * The actual code runs in browser context where these globals exist,
 * but TypeScript needs declarations since this is a Node.js project.
 */

/* eslint-disable @typescript-eslint/no-explicit-any */

declare const window: {
  getComputedStyle(element: any): CSSStyleDeclaration;
};

declare const document: {
  body: any;
  querySelectorAll(selectors: string): NodeListOf<any>;
};

declare const CSS: {
  escape(value: string): string;
};

interface CSSStyleDeclaration {
  display: string;
  visibility: string;
  opacity: string;
}
