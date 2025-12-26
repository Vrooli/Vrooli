/**
 * Action Parser Tests
 */

import { parseLLMResponse, extractReasoning, ActionParseError } from '../../../../src/ai/action/parser';

describe('parseLLMResponse', () => {
  describe('ACTION: syntax parsing', () => {
    it('parses click action with element ID', () => {
      const response = `
        I can see the login button labeled [5].
        I'll click on it to proceed.

        ACTION: click(5)
      `;

      const action = parseLLMResponse(response);

      expect(action).toEqual({
        type: 'click',
        elementId: 5,
      });
    });

    it('parses click action with coordinates', () => {
      const response = `
        The button is at coordinates (150, 300).

        ACTION: click(150, 300)
      `;

      const action = parseLLMResponse(response);

      expect(action).toEqual({
        type: 'click',
        coordinates: { x: 150, y: 300 },
      });
    });

    it('parses click action with variant', () => {
      const response = 'ACTION: click(5, "double")';

      const action = parseLLMResponse(response);

      expect(action).toEqual({
        type: 'click',
        elementId: 5,
        variant: 'double',
      });
    });

    it('parses type action with element ID and text', () => {
      const response = `
        I'll type the username into the input field [3].

        ACTION: type(3, "john.doe@example.com")
      `;

      const action = parseLLMResponse(response);

      expect(action).toEqual({
        type: 'type',
        elementId: 3,
        text: 'john.doe@example.com',
      });
    });

    it('parses type action with just text', () => {
      const response = 'ACTION: type("hello world")';

      const action = parseLLMResponse(response);

      expect(action).toEqual({
        type: 'type',
        text: 'hello world',
      });
    });

    it('parses type action with clearFirst flag', () => {
      const response = 'ACTION: type(3, "new text", true)';

      const action = parseLLMResponse(response);

      expect(action).toEqual({
        type: 'type',
        elementId: 3,
        text: 'new text',
        clearFirst: true,
      });
    });

    it('parses scroll action', () => {
      const response = 'ACTION: scroll(down)';

      const action = parseLLMResponse(response);

      expect(action).toEqual({
        type: 'scroll',
        direction: 'down',
      });
    });

    it('parses scroll action with amount', () => {
      const response = 'ACTION: scroll(up, 500)';

      const action = parseLLMResponse(response);

      expect(action).toEqual({
        type: 'scroll',
        direction: 'up',
        amount: 500,
      });
    });

    it('parses navigate action', () => {
      const response = 'ACTION: navigate("https://example.com/login")';

      const action = parseLLMResponse(response);

      expect(action).toEqual({
        type: 'navigate',
        url: 'https://example.com/login',
      });
    });

    it('parses hover action with element ID', () => {
      const response = 'ACTION: hover(7)';

      const action = parseLLMResponse(response);

      expect(action).toEqual({
        type: 'hover',
        elementId: 7,
      });
    });

    it('parses hover action with coordinates', () => {
      const response = 'ACTION: hover(100, 200)';

      const action = parseLLMResponse(response);

      expect(action).toEqual({
        type: 'hover',
        coordinates: { x: 100, y: 200 },
      });
    });

    it('parses select action', () => {
      const response = 'ACTION: select(4, "option-value")';

      const action = parseLLMResponse(response);

      expect(action).toEqual({
        type: 'select',
        elementId: 4,
        value: 'option-value',
      });
    });

    it('parses wait action with milliseconds', () => {
      const response = 'ACTION: wait(2000)';

      const action = parseLLMResponse(response);

      expect(action).toEqual({
        type: 'wait',
        ms: 2000,
      });
    });

    it('parses wait action with selector', () => {
      const response = 'ACTION: wait("#loading-spinner")';

      const action = parseLLMResponse(response);

      expect(action).toEqual({
        type: 'wait',
        selector: '#loading-spinner',
      });
    });

    it('parses keypress action', () => {
      const response = 'ACTION: keypress("Enter")';

      const action = parseLLMResponse(response);

      expect(action).toEqual({
        type: 'keypress',
        key: 'Enter',
      });
    });

    it('parses keypress action with modifiers', () => {
      const response = 'ACTION: keypress("a", "Control")';

      const action = parseLLMResponse(response);

      expect(action).toEqual({
        type: 'keypress',
        key: 'a',
        modifiers: ['Control'],
      });
    });

    it('parses done action with success', () => {
      const response = `
        The order has been placed successfully.

        ACTION: done(true, "Order #12345 placed for chicken")
      `;

      const action = parseLLMResponse(response);

      expect(action).toEqual({
        type: 'done',
        success: true,
        result: 'Order #12345 placed for chicken',
      });
    });

    it('parses done action with failure', () => {
      const response = 'ACTION: done(false, "Could not find checkout button")';

      const action = parseLLMResponse(response);

      expect(action).toEqual({
        type: 'done',
        success: false,
        result: 'Could not find checkout button',
      });
    });
  });

  describe('JSON block parsing', () => {
    it('parses JSON code block', () => {
      const response = `
        I'll click the button.

        \`\`\`json
        {"type": "click", "elementId": 5}
        \`\`\`
      `;

      const action = parseLLMResponse(response);

      expect(action).toEqual({
        type: 'click',
        elementId: 5,
      });
    });

    it('parses code block without language tag', () => {
      const response = `
        \`\`\`
        {"type": "type", "text": "hello", "elementId": 3}
        \`\`\`
      `;

      const action = parseLLMResponse(response);

      expect(action).toEqual({
        type: 'type',
        text: 'hello',
        elementId: 3,
      });
    });
  });

  describe('Claude XML parsing', () => {
    it('parses action XML tag', () => {
      const response = `
        I can see the login button.
        <action>{"type": "click", "elementId": 5}</action>
      `;

      const action = parseLLMResponse(response);

      expect(action).toEqual({
        type: 'click',
        elementId: 5,
      });
    });

    it('parses multiline action XML', () => {
      const response = `
        <action>
        {
          "type": "type",
          "elementId": 3,
          "text": "user@example.com"
        }
        </action>
      `;

      const action = parseLLMResponse(response);

      expect(action).toEqual({
        type: 'type',
        elementId: 3,
        text: 'user@example.com',
      });
    });
  });

  describe('inline JSON parsing', () => {
    it('parses inline JSON object', () => {
      const response = 'I will perform this action: {"type": "scroll", "direction": "down"}';

      const action = parseLLMResponse(response);

      expect(action).toEqual({
        type: 'scroll',
        direction: 'down',
      });
    });
  });

  describe('error handling', () => {
    it('throws on unrecognized response format', () => {
      const response = 'I am confused and cannot proceed.';

      expect(() => parseLLMResponse(response)).toThrow(ActionParseError);
      expect(() => parseLLMResponse(response)).toThrow(
        'Could not parse action from LLM response'
      );
    });

    it('throws on invalid action type', () => {
      const response = '{"type": "invalid_action"}';

      expect(() => parseLLMResponse(response)).toThrow(ActionParseError);
      expect(() => parseLLMResponse(response)).toThrow('Invalid action type');
    });

    it('throws on click without elementId or coordinates', () => {
      const response = '{"type": "click"}';

      expect(() => parseLLMResponse(response)).toThrow(ActionParseError);
      expect(() => parseLLMResponse(response)).toThrow(
        'Click action requires elementId or coordinates'
      );
    });

    it('throws on type without text', () => {
      const response = '{"type": "type", "elementId": 3}';

      expect(() => parseLLMResponse(response)).toThrow(ActionParseError);
      expect(() => parseLLMResponse(response)).toThrow(
        'Type action requires text string'
      );
    });

    it('throws on scroll without direction', () => {
      const response = '{"type": "scroll"}';

      expect(() => parseLLMResponse(response)).toThrow(ActionParseError);
      expect(() => parseLLMResponse(response)).toThrow(
        'Scroll action requires direction'
      );
    });

    it('preserves raw response in error', () => {
      const response = 'No action here';

      try {
        parseLLMResponse(response);
        fail('Should have thrown');
      } catch (error) {
        expect(error).toBeInstanceOf(ActionParseError);
        expect((error as ActionParseError).rawResponse).toBe(response);
      }
    });
  });

  describe('edge cases', () => {
    it('handles text with special characters', () => {
      const response = 'ACTION: type(3, "hello, \\"world\\"!")';

      // This should parse without error
      const action = parseLLMResponse(response);
      expect(action.type).toBe('type');
    });

    it('handles mixed case ACTION keyword', () => {
      const response = 'action: click(5)';

      const action = parseLLMResponse(response);

      expect(action).toEqual({
        type: 'click',
        elementId: 5,
      });
    });

    it('handles extra whitespace', () => {
      const response = '  ACTION:   click(  5  )  ';

      const action = parseLLMResponse(response);

      expect(action).toEqual({
        type: 'click',
        elementId: 5,
      });
    });

    it('handles single-quoted strings', () => {
      const response = "ACTION: type(3, 'hello')";

      const action = parseLLMResponse(response);

      expect(action).toEqual({
        type: 'type',
        elementId: 3,
        text: 'hello',
      });
    });
  });
});

describe('extractReasoning', () => {
  it('extracts reasoning before ACTION line', () => {
    const response = `
      I can see the login button labeled [5].
      I'll click on it to proceed.

      ACTION: click(5)
    `;

    const reasoning = extractReasoning(response);

    expect(reasoning).toContain('login button labeled [5]');
    expect(reasoning).toContain("click on it to proceed");
    expect(reasoning).not.toContain('ACTION');
  });

  it('removes JSON code blocks', () => {
    const response = `
      Entering the email address.
      \`\`\`json
      {"type": "type"}
      \`\`\`
    `;

    const reasoning = extractReasoning(response);

    expect(reasoning).toBe('Entering the email address.');
    expect(reasoning).not.toContain('type');
  });

  it('removes action XML tags', () => {
    const response = `
      Clicking the button.
      <action>{"type": "click"}</action>
    `;

    const reasoning = extractReasoning(response);

    expect(reasoning).toBe('Clicking the button.');
  });

  it('handles response with no action', () => {
    const response = 'Just some text with no action.';

    const reasoning = extractReasoning(response);

    expect(reasoning).toBe('Just some text with no action.');
  });
});
