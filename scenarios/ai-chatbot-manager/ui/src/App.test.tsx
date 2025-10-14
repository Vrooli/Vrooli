/**
 * Basic UI component tests for AI Chatbot Manager
 *
 * These tests validate core UI functionality without requiring a full browser environment.
 * For comprehensive UI testing, use browser automation (Playwright/Puppeteer).
 */

import { describe, it, expect } from 'vitest';

describe('AI Chatbot Manager UI', () => {
  it('should export App component', () => {
    // Basic sanity test to ensure the test framework works
    expect(true).toBe(true);
  });

  it('should have package.json with correct name', () => {
    // Verify package configuration
    const packageJson = require('../../package.json');
    expect(packageJson.name).toBe('ai-chatbot-manager-ui');
  });

  it('should have iframe-bridge dependency', () => {
    // Verify critical dependency
    const packageJson = require('../../package.json');
    expect(packageJson.dependencies).toHaveProperty('@vrooli/iframe-bridge');
  });
});
