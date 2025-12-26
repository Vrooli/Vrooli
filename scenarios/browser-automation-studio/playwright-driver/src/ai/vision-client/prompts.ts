/**
 * Vision Agent Prompts
 *
 * STABILITY: EVOLVING
 *
 * This module contains the system prompts for browser automation.
 * Prompt engineering is crucial for getting good results from vision models.
 *
 * The prompts instruct the model to:
 * 1. Analyze the screenshot and element labels
 * 2. Reason about the current state and next step
 * 3. Output an action in the expected format
 */

import type { ElementLabel } from './types';

/**
 * Generate the system prompt for browser automation.
 */
export function generateSystemPrompt(): string {
  return `You are a browser automation agent. Your task is to analyze screenshots of web pages and determine the next action to take to achieve a user's goal.

## How You Work

You will receive:
1. A screenshot of the current web page
2. A list of interactive elements with their label numbers (e.g., [1], [2], [3])
3. The user's goal
4. The current URL

## Element Labels

Interactive elements on the page are labeled with numbers in square brackets like [1], [2], [3].
When you want to interact with an element, use its label number.

## Available Actions

You can perform these actions using the ACTION syntax:

### Click on an element
\`\`\`
ACTION: click(ELEMENT_ID)
\`\`\`
Example: \`ACTION: click(5)\` to click element [5]

### Type text into an input
\`\`\`
ACTION: type(ELEMENT_ID, "TEXT")
\`\`\`
Example: \`ACTION: type(3, "hello@example.com")\` to type into element [3]

If you want to type into the currently focused element:
\`\`\`
ACTION: type("TEXT")
\`\`\`

### Scroll the page
\`\`\`
ACTION: scroll(DIRECTION)
\`\`\`
Direction can be: up, down, left, right
Example: \`ACTION: scroll(down)\`

### Press a keyboard key
\`\`\`
ACTION: keypress(KEY)
\`\`\`
Examples:
- \`ACTION: keypress(Enter)\`
- \`ACTION: keypress(Tab)\`
- \`ACTION: keypress(Escape)\`

### Hover over an element
\`\`\`
ACTION: hover(ELEMENT_ID)
\`\`\`
Example: \`ACTION: hover(7)\` to hover over element [7]

### Select from dropdown
\`\`\`
ACTION: select(ELEMENT_ID, "VALUE")
\`\`\`
Example: \`ACTION: select(4, "option1")\`

### Wait for content to load
\`\`\`
ACTION: wait(MILLISECONDS)
\`\`\`
Example: \`ACTION: wait(2000)\` to wait 2 seconds

### Navigate to a URL
\`\`\`
ACTION: navigate("URL")
\`\`\`
Example: \`ACTION: navigate("https://example.com/page")\`

### Mark task as complete
\`\`\`
ACTION: done(SUCCESS, "RESULT_DESCRIPTION")
\`\`\`
Examples:
- \`ACTION: done(true, "Successfully logged in and reached dashboard")\`
- \`ACTION: done(false, "Could not find the checkout button")\`

## Response Format

Always respond with:
1. Brief reasoning about the current state and what you observe
2. Your decision about what to do next
3. The ACTION line with your chosen action

## Important Guidelines

- Always explain your reasoning before the action
- Use element IDs from the labels when possible (more reliable than coordinates)
- If an element isn't visible, try scrolling to find it
- If a form submission fails, check for error messages
- If you're stuck in a loop, try a different approach
- When the goal is achieved, use ACTION: done(true, "description")
- If you cannot achieve the goal, use ACTION: done(false, "reason")
- Be efficient - don't take unnecessary actions
- Pay attention to loading states and wait when needed`;
}

/**
 * Generate the user prompt for a specific navigation step.
 */
export function generateUserPrompt(params: {
  goal: string;
  currentUrl: string;
  elementLabels: ElementLabel[];
  stepNumber: number;
  previousActions?: string[];
}): string {
  const { goal, currentUrl, elementLabels, stepNumber, previousActions } = params;

  let prompt = `## Goal
${goal}

## Current URL
${currentUrl}

## Step ${stepNumber}
`;

  if (previousActions && previousActions.length > 0) {
    prompt += `\n## Previous Actions\n`;
    previousActions.forEach((action, i) => {
      prompt += `${i + 1}. ${action}\n`;
    });
  }

  prompt += `\n## Interactive Elements on Page\n`;

  if (elementLabels.length === 0) {
    prompt += `No interactive elements detected. You may need to scroll or wait for content to load.\n`;
  } else {
    for (const element of elementLabels) {
      const parts: string[] = [];

      parts.push(`[${element.id}]`);
      parts.push(`<${element.tagName}>`);

      if (element.text) {
        const truncated = element.text.length > 50
          ? element.text.substring(0, 50) + '...'
          : element.text;
        parts.push(`"${truncated}"`);
      }

      if (element.placeholder) {
        parts.push(`placeholder="${element.placeholder}"`);
      }

      if (element.ariaLabel) {
        parts.push(`aria-label="${element.ariaLabel}"`);
      }

      if (element.role) {
        parts.push(`role="${element.role}"`);
      }

      prompt += `${parts.join(' ')}\n`;
    }
  }

  prompt += `\nAnalyze the screenshot and determine the next action to achieve the goal.`;

  return prompt;
}

/**
 * Format element labels for inclusion in a prompt.
 * This is a simpler format for models that prefer concise input.
 */
export function formatElementLabelsCompact(labels: ElementLabel[]): string {
  if (labels.length === 0) {
    return 'No interactive elements detected.';
  }

  return labels
    .map((el) => {
      const parts = [`[${el.id}]`, el.tagName];
      if (el.text) parts.push(`"${el.text.substring(0, 30)}"`);
      if (el.placeholder) parts.push(`(${el.placeholder})`);
      return parts.join(' ');
    })
    .join('\n');
}

/**
 * Generate a continuation prompt when the model needs more context.
 */
export function generateContinuationPrompt(params: {
  previousReasoning: string;
  errorMessage?: string;
}): string {
  const { previousReasoning, errorMessage } = params;

  let prompt = `The previous action could not be completed.

## Previous Reasoning
${previousReasoning}
`;

  if (errorMessage) {
    prompt += `\n## Error
${errorMessage}
`;
  }

  prompt += `\nPlease analyze the current screenshot and try a different approach.`;

  return prompt;
}

/**
 * Generate a prompt for goal verification.
 * Used after the model says it's done to verify the goal was achieved.
 */
export function generateVerificationPrompt(params: {
  goal: string;
  claimedResult: string;
}): string {
  return `Please verify that the following goal was achieved:

## Original Goal
${params.goal}

## Claimed Result
${params.claimedResult}

Look at the current screenshot and confirm whether the goal was truly achieved.
Respond with:
- \`ACTION: done(true, "Verified: <confirmation>")\` if the goal was achieved
- \`ACTION: done(false, "Not verified: <reason>")\` if the goal was not achieved`;
}
