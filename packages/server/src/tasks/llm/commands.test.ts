import { handleCommandTransition, isAlphaNum, isNewline, isWhitespace, parsePropertyValue } from './commands';

describe('parsePropertyValue', () => {
    // Test for numbers
    test('parses integers correctly', () => {
        const input = "123 ";
        const { value, endIndex } = parsePropertyValue(input, 0);
        expect(value).toBe(123);
        expect(endIndex).toBe(3);
    });
    test('parses negative integers correctly', () => {
        const input = "-456 ";
        const { value, endIndex } = parsePropertyValue(input, 0);
        expect(value).toBe(-456);
        expect(endIndex).toBe(4);
    });
    test('parses floats correctly', () => {
        const input = "123.456";
        const { value, endIndex } = parsePropertyValue(input, 0);
        expect(value).toBe(123.456);
        expect(endIndex).toBe(7);
    });

    // Test for quoted strings
    test('parses single-quoted strings correctly', () => {
        const input = "'hello world'";
        const { value, endIndex } = parsePropertyValue(input, 0);
        expect(value).toBe("hello world");
        expect(endIndex).toBe(13);
    });
    test('parses double-quoted strings correctly', () => {
        const input = '"hello world" ';
        const { value, endIndex } = parsePropertyValue(input, 0);
        expect(value).toBe("hello world");
        expect(endIndex).toBe(13);
    });

    // Test for special characters in strings
    test('handles escaped quotes within strings', () => {
        const input = "'hello \\'world\\'' ";
        const { value, endIndex } = parsePropertyValue(input, 0);
        expect(value).toBe("hello 'world'");
        expect(endIndex).toBe(17);
    });
    test('handles newlines within quoted strings', () => {
        const input = "'hello\nworld' ";
        const { value, endIndex } = parsePropertyValue(input, 0);
        expect(value).toBe("hello\nworld");
        expect(endIndex).toBe(13);
    });

    // Test for null
    test('parses null correctly', () => {
        const input = "null ";
        const { value, endIndex } = parsePropertyValue(input, 0);
        expect(value).toBeNull();
        expect(endIndex).toBe(4);
    });

    // Test for handling unexpected formats
    test('ignores content after space in unquoted values', () => {
        const input = "unquoted value";
        const { value, endIndex } = parsePropertyValue(input, 0);
        expect(value).toBe("unquoted");
        expect(endIndex).toBe(8);
    });

    // Add more tests as needed for other edge cases and scenarios...
});

describe('isNewline', () => {
    test('recognizes newline character', () => {
        expect(isNewline('\n')).toBe(true);
    });

    test('recognizes carriage return character', () => {
        expect(isNewline('\r')).toBe(true);
    });

    test('does not recognize space as newline', () => {
        expect(isNewline(' ')).toBe(false);
    });

    test('does not recognize alphabetic character as newline', () => {
        expect(isNewline('a')).toBe(false);
    });

    test('does not recognize numeric character as newline', () => {
        expect(isNewline('1')).toBe(false);
    });
});

describe('isWhitespace', () => {
    test('recognizes space as whitespace', () => {
        expect(isWhitespace(' ')).toBe(true);
    });

    test('recognizes tab as whitespace', () => {
        expect(isWhitespace('\t')).toBe(true);
    });

    test('does not recognize newline as whitespace', () => {
        expect(isWhitespace('\n')).toBe(false);
    });

    test('does not recognize carriage return as whitespace', () => {
        expect(isWhitespace('\r')).toBe(false);
    });

    test('does not recognize alphabetic character as whitespace', () => {
        expect(isWhitespace('a')).toBe(false);
    });

    test('does not recognize numeric character as whitespace', () => {
        expect(isWhitespace('1')).toBe(false);
    });
});

describe('isAlphaNum', () => {
    test('recognizes lowercase alphabetic character', () => {
        expect(isAlphaNum('a')).toBe(true);
    });

    test('recognizes uppercase alphabetic character', () => {
        expect(isAlphaNum('Z')).toBe(true);
    });

    test('recognizes numeric character', () => {
        expect(isAlphaNum('0')).toBe(true);
        expect(isAlphaNum('9')).toBe(true);
    });

    test('does not recognize space as alphanumeric', () => {
        expect(isAlphaNum(' ')).toBe(false);
    });

    test('does not recognize newline as alphanumeric', () => {
        expect(isAlphaNum('\n')).toBe(false);
    });

    test('does not recognize special character as alphanumeric', () => {
        expect(isAlphaNum('*')).toBe(false);
    });

    test('does not recognize accented characters as alphanumeric', () => {
        expect(isAlphaNum('á')).toBe(false);
        expect(isAlphaNum('ñ')).toBe(false);
    });

    test('does not recognize characters from other alphabets as alphanumeric', () => {
        expect(isAlphaNum('你')).toBe(false);
        expect(isAlphaNum('👋')).toBe(false);
    });
});

describe('handleCommandTransition', () => {
    // Outside tests
    test('keep adding to outside when not on a slash', () => {
        const buffer = "asdf";
        // Whitespace
        let result = handleCommandTransition({
            curr: ' ',
            next: ' ',
            section: 'outside',
            buffer,
            isInQuotes: false,
        });
        expect(result).toEqual({ section: 'outside', buffer: buffer + " ", isInQuotes: false });
        result = handleCommandTransition({
            curr: '\t',
            next: ' ',
            section: 'outside',
            buffer,
            isInQuotes: false,
        });
        expect(result).toEqual({ section: 'outside', buffer: buffer + "\t", isInQuotes: false });
        // Quotes
        result = handleCommandTransition({
            curr: `'`,
            next: ' ',
            section: 'outside',
            buffer,
            isInQuotes: false,
        });
        expect(result).toEqual({ section: 'outside', buffer: buffer + `'`, isInQuotes: false });
        result = handleCommandTransition({
            curr: `"`,
            next: ' ',
            section: 'outside',
            buffer,
            isInQuotes: false,
        });
        expect(result).toEqual({ section: 'outside', buffer: buffer + `"`, isInQuotes: false });
        // Equals sign
        result = handleCommandTransition({
            curr: '=',
            next: ' ',
            section: 'outside',
            buffer,
            isInQuotes: false,
        });
        expect(result).toEqual({ section: 'outside', buffer: buffer + "=", isInQuotes: false });
        // Alphanumeric
        result = handleCommandTransition({
            curr: 'a',
            next: ' ',
            section: 'outside',
            buffer,
            isInQuotes: false,
        });
        expect(result).toEqual({ section: 'outside', buffer: buffer + "a", isInQuotes: false });
        result = handleCommandTransition({
            curr: '1',
            next: ' ',
            section: 'outside',
            buffer,
            isInQuotes: false,
        });
        expect(result).toEqual({ section: 'outside', buffer: buffer + "1", isInQuotes: false });
        // Newline
        result = handleCommandTransition({
            curr: '\n',
            next: ' ',
            section: 'outside',
            buffer,
            isInQuotes: false,
        });
        expect(result).toEqual({ section: 'outside', buffer: buffer + "\n", isInQuotes: false });
        // Other alphabets
        result = handleCommandTransition({
            curr: '你',
            next: ' ',
            section: 'outside',
            buffer,
            isInQuotes: false,
        });
        expect(result).toEqual({ section: 'outside', buffer: buffer + "你", isInQuotes: false });
        result = handleCommandTransition({
            curr: '👋',
            next: ' ',
            section: 'outside',
            buffer,
            isInQuotes: false,
        });
        expect(result).toEqual({ section: 'outside', buffer: buffer + "👋", isInQuotes: false });
    });

    // Command tests
    test('does not start a command when the slash is not preceeded by whitespace', () => {
        let buffer = "asdf";
        let result = handleCommandTransition({
            curr: '/',
            next: ' ',
            section: 'outside',
            buffer,
            isInQuotes: false,
        });
        expect(result).toEqual({ section: 'outside', buffer: buffer + "/", isInQuotes: false });
        buffer = "!@#$";
        result = handleCommandTransition({
            curr: '/',
            next: ' ',
            section: 'outside',
            buffer,
            isInQuotes: false,
        });
        expect(result).toEqual({ section: 'outside', buffer: buffer + "/", isInQuotes: false });
        buffer = "🙌💃";
        result = handleCommandTransition({
            curr: '/',
            next: ' ',
            section: 'outside',
            buffer,
            isInQuotes: false,
        });
        expect(result).toEqual({ section: 'outside', buffer: buffer + "/", isInQuotes: false });
    });
    test('starts a command when buffer is empty', () => {
        const result = handleCommandTransition({
            curr: '/',
            next: ' ',
            section: 'outside',
            buffer: '',
            isInQuotes: false,
        });
        expect(result).toEqual({ section: 'command', buffer: "", isInQuotes: false });
    });
    test('starts a command after a newline', () => {
        console.log('testing start command after newline')
        const result = handleCommandTransition({
            curr: '/',
            next: ' ',
            section: 'outside',
            buffer: 'asdf\n',
            isInQuotes: false,
        });
        expect(result).toEqual({ section: 'command', buffer: "", isInQuotes: false });
    });
    test('starts a command after whitespace', () => {
        let result = handleCommandTransition({
            curr: '/',
            next: ' ',
            section: 'outside',
            buffer: 'asdf ',
            isInQuotes: false,
        });
        expect(result).toEqual({ section: 'command', buffer: "", isInQuotes: false });
        result = handleCommandTransition({
            curr: '/',
            next: ' ',
            section: 'outside',
            buffer: 'asdf\t',
            isInQuotes: false,
        });
        expect(result).toEqual({ section: 'command', buffer: "", isInQuotes: false });
    });
    test('adds alphanumeric to command buffer', () => {
        let result = handleCommandTransition({
            curr: 'a',
            next: ' ',
            section: 'command',
            buffer: 'test',
            isInQuotes: false,
        });
        expect(result).toEqual({ section: 'command', buffer: 'testa', isInQuotes: false });
        result = handleCommandTransition({
            curr: '1',
            next: ' ',
            section: 'command',
            buffer: 'test',
            isInQuotes: false,
        });
        expect(result).toEqual({ section: 'command', buffer: 'test1', isInQuotes: false });
    });
    test('resets to outside on newline', () => {
        const result = handleCommandTransition({
            curr: '\n',
            next: ' ',
            section: 'command',
            buffer: 'test',
            isInQuotes: false,
        });
        expect(result).toEqual({ section: 'outside', buffer: '', isInQuotes: false });
    });
    test('resets to outside on symbols/other alphabets', () => {
        let result = handleCommandTransition({
            curr: '你',
            next: ' ',
            section: 'command',
            buffer: 'test',
            isInQuotes: false,
        });
        expect(result).toEqual({ section: 'outside', buffer: '', isInQuotes: false });
        result = handleCommandTransition({
            curr: '👋',
            next: ' ',
            section: 'command',
            buffer: 'test',
            isInQuotes: false,
        });
        expect(result).toEqual({ section: 'outside', buffer: '', isInQuotes: false });
        result = handleCommandTransition({
            curr: '!',
            next: ' ',
            section: 'command',
            buffer: 'test',
            isInQuotes: false,
        });
        expect(result).toEqual({ section: 'outside', buffer: '', isInQuotes: false });
    });

    // Pending (when we're not sure if it's an action or a property yet) tests
    test('starts pending when we encounter the first space after a command', () => {
        let result = handleCommandTransition({
            curr: ' ',
            next: ' ',
            section: 'command',
            buffer: '/test',
            isInQuotes: false,
        });
        expect(result).toEqual({ section: 'pending', buffer: '', isInQuotes: false });
        result = handleCommandTransition({
            curr: '\t',
            next: ' ',
            section: 'command',
            buffer: 'test',
            isInQuotes: false,
        });
        expect(result).toEqual({ section: 'pending', buffer: '', isInQuotes: false });
    });
    test('does not start pending for newline', () => {
        const result = handleCommandTransition({
            curr: '\n',
            next: ' ',
            section: 'command',
            buffer: 'test',
            isInQuotes: false,
        });
        expect(result).toEqual({ section: 'outside', buffer: "", isInQuotes: false });
    });
    test('does not start pending for symbols/other alphabets', () => {
        let result = handleCommandTransition({
            curr: '你',
            next: ' ',
            section: 'command',
            buffer: 'test',
            isInQuotes: false,
        });
        expect(result).toEqual({ section: 'outside', buffer: "", isInQuotes: false });
        result = handleCommandTransition({
            curr: '👋',
            next: ' ',
            section: 'command',
            buffer: 'test',
            isInQuotes: false,
        });
        expect(result).toEqual({ section: 'outside', buffer: "", isInQuotes: false });
        result = handleCommandTransition({
            curr: '!',
            next: ' ',
            section: 'command',
            buffer: 'test',
            isInQuotes: false,
        });
        expect(result).toEqual({ section: 'outside', buffer: "", isInQuotes: false });
    });

    test('parses action when next character is whitespace', () => {
        const result = handleCommandTransition({
            curr: 'action',
            next: ' ',
            section: 'pending',
            buffer: '',
            isInQuotes: false,
        });
        expect(result).toEqual({ section: 'command', buffer: '', isInQuotes: false });  // Assuming it should reset to command
    });

    test('parses property name when next character is equals sign', () => {
        const result = handleCommandTransition({
            curr: 'propName',
            next: '=',
            section: 'pending',
            buffer: '',
            isInQuotes: false,
        });
        expect(result).toEqual({ section: 'propName', buffer: 'propName', isInQuotes: false });
    });

    test('resets on newline outside of quotes', () => {
        const result = handleCommandTransition({
            curr: '\n',
            next: ' ',
            section: 'command',
            buffer: '/test',
            isInQuotes: false,
        });
        expect(result).toEqual({ section: 'outside', buffer: '', isInQuotes: false });
    });

    test('enters propValue section when equals sign is encountered', () => {
        const result = handleCommandTransition({
            curr: '=',
            next: ' ',
            section: 'propName',
            buffer: 'key',
            isInQuotes: false,
        });
        expect(result).toEqual({ section: 'propValue', buffer: 'key', isInQuotes: false });
    });

    test('handles entering quotes for property value', () => {
        const result = handleCommandTransition({
            curr: '"',
            next: 'value',
            section: 'propValue',
            buffer: '',
            isInQuotes: false,
        });
        expect(result).toEqual({ section: 'propValue', buffer: '"', isInQuotes: true });
    });

    test('handles exiting quotes for property value', () => {
        const result = handleCommandTransition({
            curr: '"',
            next: ' ',
            section: 'propValue',
            buffer: '"value',
            isInQuotes: true,
        });
        expect(result).toEqual({ section: 'propName', buffer: '"value"', isInQuotes: false });
    });

    test('ignores command when preceded by non-whitespace character', () => {
        const result = handleCommandTransition({
            curr: '/',
            next: ' ',
            section: 'outside',
            buffer: 'text',
            isInQuotes: false,
        });
        expect(result).toEqual({ section: 'outside', buffer: '', isInQuotes: false });
    });

    test('transitions from pending to propName or command based on nextChar', () => {
        const propNameResult = handleCommandTransition({
            curr: 'name',
            next: '=',
            section: 'pending',
            buffer: '',
            isInQuotes: false,
        });
        expect(propNameResult).toEqual({ section: 'propName', buffer: 'name', isInQuotes: false });

        const actionResult = handleCommandTransition({
            curr: 'action',
            next: ' ',
            section: 'pending',
            buffer: '',
            isInQuotes: false,
        });
        expect(actionResult).toEqual({ section: 'command', buffer: '', isInQuotes: false });
    });
});



// describe('handleCommandActionTransition', () => {
//     // Test the transition from command to action
//     test('should transition from command to action on space', () => {
//         const result = handleCommandActionTransition(' ', 'command', '');
//         expect(result).toEqual({ newSection: 'action', newBuffer: '' });
//     });

//     // Test the transition from action to property name
//     test('should transition from action to property name on space', () => {
//         const result = handleCommandActionTransition(' ', 'action', 'create');
//         expect(result).toEqual({ newSection: 'propName', newBuffer: '' });
//     });

//     // Test the transition from action to property name on newline
//     test('should transition from action to property name on newline', () => {
//         const result = handleCommandActionTransition('\n', 'action', 'create');
//         expect(result).toEqual({ newSection: 'propName', newBuffer: '' });
//     });

//     // Test reset to command on newline
//     test('should reset to command on newline', () => {
//         const result = handleCommandActionTransition('\n', 'command', 'note');
//         expect(result).toEqual({ newSection: 'command', newBuffer: '' });
//     });

//     // Test ignoring non-whitespace characters before slash
//     test('should ignore non-whitespace characters before slash', () => {
//         const result = handleCommandActionTransition('/', 'command', 'a');
//         expect(result).toEqual({ newSection: 'command', newBuffer: 'a/' });
//     });

//     // Test maintaining current buffer and section when no transition is required
//     test('should maintain current buffer and section', () => {
//         const result = handleCommandActionTransition('x', 'command', '/');
//         expect(result).toEqual({ newSection: 'command', newBuffer: '/x' });
//     });
// });



// describe('extractCommands', () => {
//     test('parses a basic command with no action or properties', () => {
//         const input = "/viewDashboard";
//         const expected: LlmCommand[] = [{ command: "viewDashboard", action: null, properties: null }];
//         expect(extractCommands(input)).toEqual(expected);
//     });

//     test('parses a command with an action', () => {
//         const input = "/bot create";
//         const expected: LlmCommand[] = [{ command: "bot", action: "create", properties: null }];
//         expect(extractCommands(input)).toEqual(expected);
//     });

//     test('parses a command with properties of various types', () => {
//         const input = `/setTimer time=10 unit='minutes' other1=null other2='420' other3="double_quotes"`;
//         const expected: LlmCommand[] = [{
//             command: "setTimer",
//             action: null,
//             properties: { time: 10, unit: "minutes", other1: null, other2: "420", other3: "double_quotes" },
//         }];
//         expect(extractCommands(input)).toEqual(expected);
//     });

//     test('parses a command with an action and properties', () => {
//         const input = "/bot create name='Elon' occupation='CEO'";
//         const expected: LlmCommand[] = [{
//             command: "bot",
//             action: "create",
//             properties: { name: "Elon", occupation: "CEO" },
//         }];
//         expect(extractCommands(input)).toEqual(expected);
//     });

//     test('parses multiple commands in a single string', () => {
//         const input = "/start /stop reason='Finished tasks'";
//         const expected: LlmCommand[] = [
//             { command: "start", action: null, properties: null },
//             { command: "stop", action: null, properties: { reason: "Finished tasks" } },
//         ];
//         expect(extractCommands(input)).toEqual(expected);
//     });

//     test('handles edge cases with unexpected formats', () => {
//         const input = "/weirdFormat 123 /anotherCommand prop=123";
//         const expected: LlmCommand[] = [
//             { command: "weirdFormat", action: null, properties: null },
//             { command: "anotherCommand", action: null, properties: { prop: 123 } },
//         ];
//         expect(extractCommands(input)).toEqual(expected);
//     });

//     test('handles input with no commands', () => {
//         const input = "Just some regular text without any commands.";
//         const expected: LlmCommand[] = [];
//         expect(extractCommands(input)).toEqual(expected);
//     });

//     test('parses commands correctly within a larger text', () => {
//         const input = "Please execute /start and then /stop after 5 minutes.";
//         const expected: LlmCommand[] = [
//             { command: "start", action: "and", properties: null },
//             { command: "stop", action: "after", properties: null },
//         ];
//         expect(extractCommands(input)).toEqual(expected);
//     });

//     test('parses commands correctly with text between them', () => {
//         const input = "/start The system should then /pause '5m' before /stop";
//         const expected: LlmCommand[] = [
//             { command: "start", action: "The", properties: null },
//             { command: "pause", action: null, properties: null },
//             { command: "stop", action: null, properties: null },
//         ];
//         expect(extractCommands(input)).toEqual(expected);
//     });

//     test('ignores malformed commands', () => {
//         const input = "/ This is not a valid command /start and this /invalidCommand 'with' improper =format";
//         const expected: LlmCommand[] = [
//             { command: "start", action: "and", properties: null },
//             { command: "invalidCommand", action: null, properties: null },
//         ];
//         expect(extractCommands(input)).toEqual(expected);
//     });

//     test('works with positive, negative, 0, and decimal numbers', () => {
//         const input = "/testa create prop1=123 prop2=-456 prop3=0 prop4=3.14 /testb prop5=-3.14";
//         const expected: LlmCommand[] = [
//             { command: "testa", action: "create", properties: { prop1: 123, prop2: -456, prop3: 0, prop4: 3.14 } },
//             { command: "testb", action: null, properties: { prop5: -3.14 } },
//         ];
//         expect(extractCommands(input)).toEqual(expected);
//     });

//     test('works with non-ascii characters', () => {
//         const input = "boop /test create prop1='你好' prop2='👋' chicken";
//         const expected: LlmCommand[] = [
//             { command: "test", action: "create", properties: { prop1: "你好", prop2: "👋" } },
//         ];
//         expect(extractCommands(input)).toEqual(expected);
//     });

//     test('handles newlines correctly', () => {
//         const input = "/test create\nprop1=123\nprop2='456' /testb\nboop /testc hello prop3=789\nprop4='abc'";
//         const expected: LlmCommand[] = [
//             { command: "test", action: "create", properties: null },
//             { command: "testb", action: null, properties: null },
//             { command: "testc", action: "hello", properties: { prop3: 789 } },
//         ];
//         expect(extractCommands(input)).toEqual(expected);
//     });

//     test('handles invalid input', () => {
//         // @ts-ignore: Testing runtime scenario
//         expect(extractCommands(null)).toEqual([]);
//         // @ts-ignore: Testing runtime scenario
//         expect(extractCommands(undefined)).toEqual([]);
//         // @ts-ignore: Testing runtime scenario
//         expect(extractCommands(123)).toEqual([]);
//         // @ts-ignore: Testing runtime scenario
//         expect(extractCommands({})).toEqual([]);
//         // @ts-ignore: Testing runtime scenario
//         expect(extractCommands([])).toEqual([]);
//     });

//     test('handles slashes within quoted strings', () => {
//         const input = `/test prop1='/' prop2='\\' prop3="/"`;
//         const expected: LlmCommand[] = [{ command: "test", action: null, properties: { prop1: "/", prop2: "\\", prop3: "/" } }];
//         expect(extractCommands(input)).toEqual(expected);
//     });

//     //TODO ignores asdf/asdf (i.e. command with test before and no whitespace)
// });

