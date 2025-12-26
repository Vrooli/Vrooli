/**
 * Action Parser
 *
 * STABILITY: STABLE CORE
 *
 * This module parses LLM responses into BrowserAction objects.
 * The parser supports multiple response formats to handle different models.
 *
 * Supported formats:
 * 1. ACTION: click(5) - Simple function-call syntax
 * 2. ACTION: type(3, "text") - Function with arguments
 * 3. JSON block: {"type": "click", "elementId": 5}
 * 4. Claude-style: <action>{"type": "click", "elementId": 5}</action>
 */

import type {
  BrowserAction,
  ClickAction,
  TypeAction,
  ScrollAction,
  NavigateAction,
  HoverAction,
  SelectAction,
  WaitAction,
  KeyPressAction,
  DoneAction,
} from './types';

/**
 * Error thrown when parsing fails.
 */
export class ActionParseError extends Error {
  constructor(
    message: string,
    public readonly rawResponse: string
  ) {
    super(message);
    this.name = 'ActionParseError';
  }
}

/**
 * Parse an LLM response into a BrowserAction.
 *
 * @param response - Raw LLM response text
 * @returns Parsed BrowserAction
 * @throws ActionParseError if parsing fails
 */
export function parseLLMResponse(response: string): BrowserAction {
  // Try each parsing strategy in order
  const strategies = [
    parseActionSyntax,
    parseJSONBlock,
    parseClaudeXML,
    parseInlineJSON,
  ];

  for (const strategy of strategies) {
    const result = strategy(response);
    if (result) {
      return validateAction(result);
    }
  }

  throw new ActionParseError(
    'Could not parse action from LLM response. ' +
    'Expected ACTION: syntax, JSON block, or <action> XML.',
    response
  );
}

/**
 * Parse ACTION: function-call syntax.
 *
 * Examples:
 * - ACTION: click(5)
 * - ACTION: click(150, 300)
 * - ACTION: type(3, "hello world")
 * - ACTION: scroll(down)
 * - ACTION: done(true, "Completed successfully")
 */
function parseActionSyntax(response: string): BrowserAction | null {
  // Match ACTION: followed by function call
  const actionMatch = response.match(
    /ACTION:\s*(\w+)\s*\(([^)]*)\)/i
  );

  if (!actionMatch) {
    return null;
  }

  const [, actionType, argsString] = actionMatch;
  const args = parseArguments(argsString);

  switch (actionType.toLowerCase()) {
    case 'click':
      return parseClickArgs(args);

    case 'type':
      return parseTypeArgs(args);

    case 'scroll':
      return parseScrollArgs(args);

    case 'navigate':
      return parseNavigateArgs(args);

    case 'hover':
      return parseHoverArgs(args);

    case 'select':
      return parseSelectArgs(args);

    case 'wait':
      return parseWaitArgs(args);

    case 'keypress':
    case 'key':
      return parseKeyPressArgs(args);

    case 'done':
      return parseDoneArgs(args);

    default:
      return null;
  }
}

/**
 * Parse arguments from a function call string.
 * Handles numbers, strings, and booleans.
 */
function parseArguments(argsString: string): (string | number | boolean)[] {
  const args: (string | number | boolean)[] = [];
  const trimmed = argsString.trim();

  if (!trimmed) {
    return args;
  }

  // State machine for parsing
  let current = '';
  let inString = false;
  let stringChar = '';

  for (let i = 0; i < trimmed.length; i++) {
    const char = trimmed[i];

    if (inString) {
      if (char === stringChar && trimmed[i - 1] !== '\\') {
        // End of string
        args.push(current);
        current = '';
        inString = false;
      } else {
        current += char;
      }
    } else if (char === '"' || char === "'") {
      inString = true;
      stringChar = char;
    } else if (char === ',') {
      if (current.trim()) {
        args.push(parseValue(current.trim()));
      }
      current = '';
    } else if (char === ' ' || char === '\t') {
      // Skip whitespace outside of strings (unless already accumulating)
      if (current.length > 0) {
        current += char;
      }
    } else {
      current += char;
    }
  }

  // Handle last argument
  if (current.trim()) {
    args.push(parseValue(current.trim()));
  }

  return args;
}

/**
 * Parse a single value (number, boolean, or string).
 */
function parseValue(value: string): string | number | boolean {
  // Boolean
  if (value.toLowerCase() === 'true') return true;
  if (value.toLowerCase() === 'false') return false;

  // Number
  const num = Number(value);
  if (!isNaN(num)) return num;

  // String (strip quotes if present)
  return value.replace(/^["']|["']$/g, '');
}

/**
 * Parse click action arguments.
 */
function parseClickArgs(args: (string | number | boolean)[]): ClickAction {
  if (args.length === 1 && typeof args[0] === 'number') {
    // click(5) - element ID
    return { type: 'click', elementId: args[0] };
  }

  if (args.length === 2 && typeof args[0] === 'number' && typeof args[1] === 'number') {
    // click(150, 300) - coordinates
    return { type: 'click', coordinates: { x: args[0], y: args[1] } };
  }

  if (args.length === 2 && typeof args[0] === 'number' && typeof args[1] === 'string') {
    // click(5, "double") - element ID with variant
    return {
      type: 'click',
      elementId: args[0],
      variant: args[1] as 'left' | 'right' | 'double',
    };
  }

  throw new ActionParseError(
    `Invalid click arguments: ${JSON.stringify(args)}`,
    ''
  );
}

/**
 * Parse type action arguments.
 */
function parseTypeArgs(args: (string | number | boolean)[]): TypeAction {
  if (args.length === 1 && typeof args[0] === 'string') {
    // type("text") - type into focused element
    return { type: 'type', text: args[0] };
  }

  if (args.length === 2 && typeof args[0] === 'number' && typeof args[1] === 'string') {
    // type(3, "text") - element ID and text
    return { type: 'type', elementId: args[0], text: args[1] };
  }

  if (args.length === 3 && typeof args[0] === 'number' && typeof args[1] === 'string') {
    // type(3, "text", true) - with clearFirst
    return {
      type: 'type',
      elementId: args[0],
      text: args[1],
      clearFirst: Boolean(args[2]),
    };
  }

  throw new ActionParseError(
    `Invalid type arguments: ${JSON.stringify(args)}`,
    ''
  );
}

/**
 * Parse scroll action arguments.
 */
function parseScrollArgs(args: (string | number | boolean)[]): ScrollAction {
  if (args.length >= 1 && typeof args[0] === 'string') {
    const direction = args[0].toLowerCase() as 'up' | 'down' | 'left' | 'right';
    return {
      type: 'scroll',
      direction,
      amount: typeof args[1] === 'number' ? args[1] : undefined,
    };
  }

  throw new ActionParseError(
    `Invalid scroll arguments: ${JSON.stringify(args)}`,
    ''
  );
}

/**
 * Parse navigate action arguments.
 */
function parseNavigateArgs(args: (string | number | boolean)[]): NavigateAction {
  if (args.length === 1 && typeof args[0] === 'string') {
    return { type: 'navigate', url: args[0] };
  }

  throw new ActionParseError(
    `Invalid navigate arguments: ${JSON.stringify(args)}`,
    ''
  );
}

/**
 * Parse hover action arguments.
 */
function parseHoverArgs(args: (string | number | boolean)[]): HoverAction {
  if (args.length === 1 && typeof args[0] === 'number') {
    return { type: 'hover', elementId: args[0] };
  }

  if (args.length === 2 && typeof args[0] === 'number' && typeof args[1] === 'number') {
    return { type: 'hover', coordinates: { x: args[0], y: args[1] } };
  }

  throw new ActionParseError(
    `Invalid hover arguments: ${JSON.stringify(args)}`,
    ''
  );
}

/**
 * Parse select action arguments.
 */
function parseSelectArgs(args: (string | number | boolean)[]): SelectAction {
  if (args.length === 2 && typeof args[0] === 'number' && typeof args[1] === 'string') {
    return { type: 'select', elementId: args[0], value: args[1] };
  }

  throw new ActionParseError(
    `Invalid select arguments: ${JSON.stringify(args)}`,
    ''
  );
}

/**
 * Parse wait action arguments.
 */
function parseWaitArgs(args: (string | number | boolean)[]): WaitAction {
  if (args.length === 0) {
    return { type: 'wait', ms: 1000 };
  }

  if (args.length === 1) {
    if (typeof args[0] === 'number') {
      return { type: 'wait', ms: args[0] };
    }
    if (typeof args[0] === 'string') {
      return { type: 'wait', selector: args[0] };
    }
  }

  throw new ActionParseError(
    `Invalid wait arguments: ${JSON.stringify(args)}`,
    ''
  );
}

/**
 * Parse keypress action arguments.
 */
function parseKeyPressArgs(args: (string | number | boolean)[]): KeyPressAction {
  if (args.length >= 1 && typeof args[0] === 'string') {
    const modifiers: ('Control' | 'Alt' | 'Shift' | 'Meta')[] = [];

    for (let i = 1; i < args.length; i++) {
      if (typeof args[i] === 'string') {
        modifiers.push(args[i] as 'Control' | 'Alt' | 'Shift' | 'Meta');
      }
    }

    return {
      type: 'keypress',
      key: args[0],
      modifiers: modifiers.length > 0 ? modifiers : undefined,
    };
  }

  throw new ActionParseError(
    `Invalid keypress arguments: ${JSON.stringify(args)}`,
    ''
  );
}

/**
 * Parse done action arguments.
 */
function parseDoneArgs(args: (string | number | boolean)[]): DoneAction {
  if (args.length >= 1) {
    const success = typeof args[0] === 'boolean' ? args[0] : true;
    const result = typeof args[1] === 'string' ? args[1] : 'Task completed';
    return { type: 'done', success, result };
  }

  return { type: 'done', success: true, result: 'Task completed' };
}

/**
 * Parse JSON block in response.
 * Looks for ```json ... ``` or ``` ... ``` blocks.
 */
function parseJSONBlock(response: string): BrowserAction | null {
  // Match code block with optional json language tag
  const codeBlockMatch = response.match(
    /```(?:json)?\s*\n?([\s\S]*?)\n?```/i
  );

  if (codeBlockMatch) {
    try {
      return JSON.parse(codeBlockMatch[1].trim()) as BrowserAction;
    } catch {
      return null;
    }
  }

  return null;
}

/**
 * Parse Claude-style <action> XML tag.
 */
function parseClaudeXML(response: string): BrowserAction | null {
  const xmlMatch = response.match(
    /<action>([\s\S]*?)<\/action>/i
  );

  if (xmlMatch) {
    try {
      return JSON.parse(xmlMatch[1].trim()) as BrowserAction;
    } catch {
      return null;
    }
  }

  return null;
}

/**
 * Parse inline JSON object in response.
 * Last resort - looks for any JSON object.
 */
function parseInlineJSON(response: string): BrowserAction | null {
  // Find JSON object pattern
  const jsonMatch = response.match(
    /\{[^{}]*"type"\s*:\s*"[^"]+"\s*[^{}]*\}/
  );

  if (jsonMatch) {
    try {
      return JSON.parse(jsonMatch[0]) as BrowserAction;
    } catch {
      return null;
    }
  }

  return null;
}

/**
 * Validate that a parsed action has all required fields.
 */
function validateAction(action: BrowserAction): BrowserAction {
  if (!action.type) {
    throw new ActionParseError('Action missing "type" field', JSON.stringify(action));
  }

  const validTypes = [
    'click', 'type', 'scroll', 'navigate', 'hover',
    'select', 'wait', 'keypress', 'done'
  ];

  if (!validTypes.includes(action.type)) {
    throw new ActionParseError(
      `Invalid action type: "${action.type}". Valid types: ${validTypes.join(', ')}`,
      JSON.stringify(action)
    );
  }

  // Type-specific validation
  switch (action.type) {
    case 'click':
      if (!action.elementId && !action.coordinates) {
        throw new ActionParseError(
          'Click action requires elementId or coordinates',
          JSON.stringify(action)
        );
      }
      break;

    case 'type':
      if (typeof action.text !== 'string') {
        throw new ActionParseError(
          'Type action requires text string',
          JSON.stringify(action)
        );
      }
      break;

    case 'scroll':
      if (!action.direction) {
        throw new ActionParseError(
          'Scroll action requires direction',
          JSON.stringify(action)
        );
      }
      break;

    case 'navigate':
      if (typeof action.url !== 'string') {
        throw new ActionParseError(
          'Navigate action requires url string',
          JSON.stringify(action)
        );
      }
      break;

    case 'select':
      if (typeof action.elementId !== 'number' || typeof action.value !== 'string') {
        throw new ActionParseError(
          'Select action requires elementId and value',
          JSON.stringify(action)
        );
      }
      break;

    case 'keypress':
      if (typeof action.key !== 'string') {
        throw new ActionParseError(
          'Keypress action requires key string',
          JSON.stringify(action)
        );
      }
      break;

    case 'done':
      if (typeof action.success !== 'boolean') {
        throw new ActionParseError(
          'Done action requires success boolean',
          JSON.stringify(action)
        );
      }
      break;
  }

  return action;
}

/**
 * Extract reasoning from LLM response.
 * Looks for reasoning before the ACTION line.
 */
export function extractReasoning(response: string): string {
  // Remove action line and everything after
  const actionIndex = response.search(/ACTION:/i);
  if (actionIndex > 0) {
    return response.substring(0, actionIndex).trim();
  }

  // Remove JSON blocks
  const cleaned = response
    .replace(/```[\s\S]*?```/g, '')
    .replace(/<action>[\s\S]*?<\/action>/gi, '')
    .replace(/\{[^{}]*"type"\s*:\s*"[^"]+"\s*[^{}]*\}/g, '');

  return cleaned.trim();
}
