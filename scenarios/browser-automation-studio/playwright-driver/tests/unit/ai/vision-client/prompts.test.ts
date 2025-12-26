/**
 * Vision Agent Prompts Tests
 */

import {
  generateSystemPrompt,
  generateUserPrompt,
  formatElementLabelsCompact,
  generateContinuationPrompt,
  generateVerificationPrompt,
} from '../../../../src/ai/vision-client/prompts';
import type { ElementLabel } from '../../../../src/ai/vision-client/types';

describe('prompts', () => {
  describe('generateSystemPrompt', () => {
    it('returns a non-empty string', () => {
      const prompt = generateSystemPrompt();
      expect(prompt.length).toBeGreaterThan(100);
    });

    it('includes action syntax documentation', () => {
      const prompt = generateSystemPrompt();
      expect(prompt).toContain('ACTION: click');
      expect(prompt).toContain('ACTION: type');
      expect(prompt).toContain('ACTION: scroll');
      expect(prompt).toContain('ACTION: done');
    });

    it('explains element labels', () => {
      const prompt = generateSystemPrompt();
      expect(prompt).toContain('[1]');
      expect(prompt).toContain('[2]');
      expect(prompt).toContain('[3]');
    });

    it('includes response format instructions', () => {
      const prompt = generateSystemPrompt();
      expect(prompt).toContain('reasoning');
      expect(prompt).toContain('ACTION');
    });
  });

  describe('generateUserPrompt', () => {
    const sampleLabels: ElementLabel[] = [
      {
        id: 1,
        selector: '#login',
        tagName: 'button',
        bounds: { x: 100, y: 200, width: 80, height: 30 },
        text: 'Login',
      },
      {
        id: 2,
        selector: '#email',
        tagName: 'input',
        bounds: { x: 100, y: 150, width: 200, height: 30 },
        placeholder: 'Enter email',
        role: 'textbox',
      },
      {
        id: 3,
        selector: '[aria-label="Menu"]',
        tagName: 'button',
        bounds: { x: 50, y: 50, width: 40, height: 40 },
        ariaLabel: 'Open menu',
      },
    ];

    it('includes goal in prompt', () => {
      const prompt = generateUserPrompt({
        goal: 'Log into the website',
        currentUrl: 'https://example.com',
        elementLabels: [],
        stepNumber: 1,
      });

      expect(prompt).toContain('Log into the website');
    });

    it('includes current URL', () => {
      const prompt = generateUserPrompt({
        goal: 'Some goal',
        currentUrl: 'https://example.com/page',
        elementLabels: [],
        stepNumber: 1,
      });

      expect(prompt).toContain('https://example.com/page');
    });

    it('includes step number', () => {
      const prompt = generateUserPrompt({
        goal: 'Some goal',
        currentUrl: 'https://example.com',
        elementLabels: [],
        stepNumber: 5,
      });

      expect(prompt).toContain('Step 5');
    });

    it('formats element labels with IDs', () => {
      const prompt = generateUserPrompt({
        goal: 'Some goal',
        currentUrl: 'https://example.com',
        elementLabels: sampleLabels,
        stepNumber: 1,
      });

      expect(prompt).toContain('[1]');
      expect(prompt).toContain('[2]');
      expect(prompt).toContain('[3]');
    });

    it('includes element text', () => {
      const prompt = generateUserPrompt({
        goal: 'Some goal',
        currentUrl: 'https://example.com',
        elementLabels: sampleLabels,
        stepNumber: 1,
      });

      expect(prompt).toContain('Login');
    });

    it('includes placeholder', () => {
      const prompt = generateUserPrompt({
        goal: 'Some goal',
        currentUrl: 'https://example.com',
        elementLabels: sampleLabels,
        stepNumber: 1,
      });

      expect(prompt).toContain('Enter email');
    });

    it('includes aria-label', () => {
      const prompt = generateUserPrompt({
        goal: 'Some goal',
        currentUrl: 'https://example.com',
        elementLabels: sampleLabels,
        stepNumber: 1,
      });

      expect(prompt).toContain('Open menu');
    });

    it('includes tag names', () => {
      const prompt = generateUserPrompt({
        goal: 'Some goal',
        currentUrl: 'https://example.com',
        elementLabels: sampleLabels,
        stepNumber: 1,
      });

      expect(prompt).toContain('<button>');
      expect(prompt).toContain('<input>');
    });

    it('handles empty element labels', () => {
      const prompt = generateUserPrompt({
        goal: 'Some goal',
        currentUrl: 'https://example.com',
        elementLabels: [],
        stepNumber: 1,
      });

      expect(prompt).toContain('No interactive elements detected');
    });

    it('includes previous actions when provided', () => {
      const prompt = generateUserPrompt({
        goal: 'Some goal',
        currentUrl: 'https://example.com',
        elementLabels: [],
        stepNumber: 3,
        previousActions: [
          'click(1)',
          'type(2, "hello")',
        ],
      });

      expect(prompt).toContain('Previous Actions');
      expect(prompt).toContain('click(1)');
      expect(prompt).toContain('type(2, "hello")');
    });

    it('truncates long text', () => {
      const longTextLabel: ElementLabel[] = [
        {
          id: 1,
          selector: '#long-text',
          tagName: 'button',
          bounds: { x: 0, y: 0, width: 100, height: 30 },
          text: 'This is a very long text that should be truncated because it exceeds the maximum length allowed in the prompt',
        },
      ];

      const prompt = generateUserPrompt({
        goal: 'Some goal',
        currentUrl: 'https://example.com',
        elementLabels: longTextLabel,
        stepNumber: 1,
      });

      // Should be truncated with ...
      expect(prompt).toContain('...');
      expect(prompt.length).toBeLessThan(prompt.length + 100);
    });
  });

  describe('formatElementLabelsCompact', () => {
    it('formats labels in compact format', () => {
      const labels: ElementLabel[] = [
        {
          id: 1,
          selector: '#btn',
          tagName: 'button',
          bounds: { x: 0, y: 0, width: 80, height: 30 },
          text: 'Click me',
        },
      ];

      const result = formatElementLabelsCompact(labels);
      expect(result).toContain('[1]');
      expect(result).toContain('button');
      expect(result).toContain('Click me');
    });

    it('handles empty labels', () => {
      const result = formatElementLabelsCompact([]);
      expect(result).toContain('No interactive elements');
    });

    it('includes placeholder in parentheses', () => {
      const labels: ElementLabel[] = [
        {
          id: 1,
          selector: '#input',
          tagName: 'input',
          bounds: { x: 0, y: 0, width: 200, height: 30 },
          placeholder: 'Enter text',
        },
      ];

      const result = formatElementLabelsCompact(labels);
      expect(result).toContain('(Enter text)');
    });
  });

  describe('generateContinuationPrompt', () => {
    it('includes previous reasoning', () => {
      const prompt = generateContinuationPrompt({
        previousReasoning: 'I tried to click the button but it was not clickable.',
      });

      expect(prompt).toContain('I tried to click the button');
    });

    it('includes error message when provided', () => {
      const prompt = generateContinuationPrompt({
        previousReasoning: 'Attempted action',
        errorMessage: 'Element not found',
      });

      expect(prompt).toContain('Element not found');
      expect(prompt).toContain('Error');
    });

    it('suggests trying different approach', () => {
      const prompt = generateContinuationPrompt({
        previousReasoning: 'Some reasoning',
      });

      expect(prompt).toContain('different approach');
    });
  });

  describe('generateVerificationPrompt', () => {
    it('includes original goal', () => {
      const prompt = generateVerificationPrompt({
        goal: 'Order a pizza',
        claimedResult: 'Pizza ordered successfully',
      });

      expect(prompt).toContain('Order a pizza');
    });

    it('includes claimed result', () => {
      const prompt = generateVerificationPrompt({
        goal: 'Order a pizza',
        claimedResult: 'Pizza ordered successfully',
      });

      expect(prompt).toContain('Pizza ordered successfully');
    });

    it('includes verification instructions', () => {
      const prompt = generateVerificationPrompt({
        goal: 'Some goal',
        claimedResult: 'Some result',
      });

      expect(prompt).toContain('done(true');
      expect(prompt).toContain('done(false');
    });
  });
});
