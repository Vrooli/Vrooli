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
    let callback;
    beforeEach(() => {
        callback = jest.fn();
    });

    // Outside tests
    test('keep adding to outside when not on a slash - space', () => {
        const buffer = "asdf";
        const result = handleCommandTransition({
            curr: ' ',
            section: 'outside',
            buffer,
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'outside', buffer: buffer + " " });
    });
    test('keep adding to outside when not on a slash - tab', () => {
        const buffer = "asdf";
        const result = handleCommandTransition({
            curr: '\t',
            section: 'outside',
            buffer,
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'outside', buffer: buffer + "\t" });
    });
    test('keep adding to outside when not on a slash - single quote', () => {
        const buffer = "asdf";
        const result = handleCommandTransition({
            curr: `'`,
            section: 'outside',
            buffer,
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'outside', buffer: buffer + `'` });
    });
    test('keep adding to outside when not on a slash - double quote', () => {
        const buffer = "asdf";
        const result = handleCommandTransition({
            curr: `"`,
            section: 'outside',
            buffer,
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'outside', buffer: buffer + `"` });
    });
    test('keep adding to outside when not on a slash - equals sign', () => {
        const buffer = "asdf";
        const result = handleCommandTransition({
            curr: '=',
            section: 'outside',
            buffer,
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'outside', buffer: buffer + "=" });
    });
    test('keep adding to outside when not on a slash - letter', () => {
        const buffer = "asdf";
        const result = handleCommandTransition({
            curr: 'a',
            section: 'outside',
            buffer,
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'outside', buffer: buffer + "a" });
    });
    test('keep adding to outside when not on a slash - number', () => {
        const buffer = "asdf";
        const result = handleCommandTransition({
            curr: '1',
            section: 'outside',
            buffer,
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'outside', buffer: buffer + "1" });
    });
    test('keep adding to outside when not on a slash - newline', () => {
        const buffer = "asdf";
        const result = handleCommandTransition({
            curr: '\n',
            section: 'outside',
            buffer,
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'outside', buffer: buffer + "\n" });
    });
    test('keep adding to outside when not on a slash - other alphabets', () => {
        const buffer = "asdf";
        const result = handleCommandTransition({
            curr: '你',
            section: 'outside',
            buffer,
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'outside', buffer: buffer + "你" });
    });
    test('keep adding to outside when not on a slash - emojis', () => {
        const buffer = "asdf";
        const result = handleCommandTransition({
            curr: '👋',
            section: 'outside',
            buffer,
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'outside', buffer: buffer + "👋" });
    });
    test('keep adding to outside when not on a slash - symbols', () => {
        const buffer = "asdf";
        const result = handleCommandTransition({
            curr: '!',
            section: 'outside',
            buffer,
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'outside', buffer: buffer + "!" });
    });

    // Command tests
    test('does not start a command when the slash is not preceeded by whitespace - letter', () => {
        const buffer = "asdf";
        const result = handleCommandTransition({
            curr: '/',
            section: 'outside',
            buffer,
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'outside', buffer: buffer + "/" });
    });
    test('does not start a command when the slash is not preceeded by whitespace - number', () => {
        const buffer = "1234";
        const result = handleCommandTransition({
            curr: '/',
            section: 'outside',
            buffer,
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'outside', buffer: buffer + "/" });
    });
    test('does not start a command when the slash is not preceeded by whitespace - symbol', () => {
        const buffer = "!@#$";
        const result = handleCommandTransition({
            curr: '/',
            section: 'outside',
            buffer,
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'outside', buffer: buffer + "/" });
    });
    test('does not start a command when the slash is not preceeded by whitespace - emoji', () => {
        const buffer = "🙌💃";
        const result = handleCommandTransition({
            curr: '/',
            section: 'outside',
            buffer,
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'outside', buffer: buffer + "/" });
    });
    test('starts a command when buffer is empty', () => {
        const result = handleCommandTransition({
            curr: '/',
            section: 'outside',
            buffer: '',
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'command', buffer: "" });
    });
    test('starts a command after a newline', () => {
        const result = handleCommandTransition({
            curr: '/',
            section: 'outside',
            buffer: 'asdf\n',
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'command', buffer: "" });
    });
    test('starts a command after whitespace - space', () => {
        const result = handleCommandTransition({
            curr: '/',
            section: 'outside',
            buffer: 'asdf ',
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'command', buffer: "" });
    });
    test('starts a command after whitespace - tab', () => {
        const result = handleCommandTransition({
            curr: '/',
            section: 'outside',
            buffer: 'asdf\t',
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'command', buffer: "" });
    });
    test('adds letter to command buffer', () => {
        const buffer = "test"
        const result = handleCommandTransition({
            curr: 'a',
            section: 'command',
            buffer,
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'command', buffer: buffer + 'a' });
    });
    test('adds number to command buffer', () => {
        const buffer = "test"
        const result = handleCommandTransition({
            curr: '1',
            section: 'command',
            buffer,
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'command', buffer: buffer + "1" });
    });
    test('resets to outside on newline', () => {
        const result = handleCommandTransition({
            curr: '\n',
            section: 'command',
            buffer: 'test',
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'outside', buffer: '' });
    });
    test('resets to outside on other alphabets', () => {
        const result = handleCommandTransition({
            curr: '你',
            section: 'command',
            buffer: 'test',
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'outside', buffer: '' });
    });
    test('resets to outside on emojis', () => {
        const result = handleCommandTransition({
            curr: '👋',
            section: 'command',
            buffer: 'test',
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'outside', buffer: '' });
    });
    test('resets to outside on symbols', () => {
        const result = handleCommandTransition({
            curr: '!',
            section: 'command',
            buffer: 'test',
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'outside', buffer: '' });
    });

    // Pending (when we're not sure if it's an action or a property yet) tests
    test('starts pending when we encounter the first space after a command - space', () => {
        const result = handleCommandTransition({
            curr: ' ',
            section: 'command',
            buffer: 'test',
            callback,
        });
        expect(callback).toHaveBeenCalledWith('command', 'test');
        expect(result).toEqual({ section: 'pending', buffer: '' });
    });
    test('starts pending when we encounter the first space after a command - tab', () => {
        const result = handleCommandTransition({
            curr: '\t',
            section: 'command',
            buffer: 'test',
            callback,
        });
        expect(callback).toHaveBeenCalledWith('command', 'test');
        expect(result).toEqual({ section: 'pending', buffer: '' });
    });
    test('does not start pending for newline', () => {
        const result = handleCommandTransition({
            curr: '\n',
            section: 'command',
            buffer: 'test',
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'outside', buffer: "" });
    });
    test('does not start pending for other alphabets', () => {
        const result = handleCommandTransition({
            curr: '你',
            section: 'command',
            buffer: 'test',
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'outside', buffer: "" });
    });
    test('does not start pending for emojis', () => {
        const result = handleCommandTransition({
            curr: '👋',
            section: 'command',
            buffer: 'test',
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'outside', buffer: "" });
    });
    test('does not start pending for symbols', () => {
        const result = handleCommandTransition({
            curr: '!',
            section: 'command',
            buffer: 'test',
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'outside', buffer: "" });
    });
    test('cancels pending on other alphabets', () => {
        const result = handleCommandTransition({
            curr: '你',
            section: 'pending',
            buffer: 'test',
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'outside', buffer: "" });
    });
    test('cancels pending on emojis', () => {
        const result = handleCommandTransition({
            curr: '👋',
            section: 'pending',
            buffer: 'test',
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'outside', buffer: "" });
    });
    test('cancels pending on symbols', () => {
        const result = handleCommandTransition({
            curr: '!',
            section: 'pending',
            buffer: 'test',
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'outside', buffer: "" });
    });

    // Action tests
    test('commits pending buffer to action on whitespace - space', () => {
        const result = handleCommandTransition({
            curr: ' ',
            section: 'pending',
            buffer: 'create',
            callback,
        });
        expect(callback).toHaveBeenCalledWith('action', 'create');
        expect(result).toEqual({ section: 'propName', buffer: '' });
    });
    test('commits pending buffer to action on whitespace - tab', () => {
        const result = handleCommandTransition({
            curr: '\t',
            section: 'pending',
            buffer: 'create',
            callback,
        });
        expect(callback).toHaveBeenCalledWith('action', 'create');
        expect(result).toEqual({ section: 'propName', buffer: '' });
    });
    test('commits pending buffer to action on newline', () => {
        const result = handleCommandTransition({
            curr: '\n',
            section: 'pending',
            buffer: 'create',
            callback,
        });
        expect(callback).toHaveBeenCalledWith('action', 'create');
        expect(result).toEqual({ section: 'outside', buffer: '' });
    });

    // Property name tests
    test('commits pending buffer to property name on equals sign', () => {
        const result = handleCommandTransition({
            curr: '=',
            section: 'pending',
            buffer: 'name',
            callback,
        });
        expect(callback).toHaveBeenCalledWith('propName', 'name');
        expect(result).toEqual({ section: 'propValue', buffer: '' });
    });
    test('commits property name buffer to property name on equals sign', () => {
        const result = handleCommandTransition({
            curr: '=',
            section: 'propName',
            buffer: 'name',
            callback,
        });
        expect(callback).toHaveBeenCalledWith('propName', 'name');
        expect(result).toEqual({ section: 'propValue', buffer: '' });
    });
    test('cancels property name buffer on whitespace - space', () => {
        const result = handleCommandTransition({
            curr: ' ',
            section: 'propName',
            buffer: 'name',
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'outside', buffer: '' });
    });
    test('cancels property name buffer on whitespace - tab', () => {
        const result = handleCommandTransition({
            curr: '\t',
            section: 'propName',
            buffer: 'name',
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'outside', buffer: '' });
    });
    test('cancels property name buffer on newline', () => {
        const result = handleCommandTransition({
            curr: '\n',
            section: 'propName',
            buffer: 'name',
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'outside', buffer: '' });
    });
    test('cancels property name buffer on other alphabets', () => {
        const result = handleCommandTransition({
            curr: '你',
            section: 'propName',
            buffer: 'name',
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'outside', buffer: '' });
    });
    test('cancels property name buffer on emojis', () => {
        const result = handleCommandTransition({
            curr: '👋',
            section: 'propName',
            buffer: 'name',
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'outside', buffer: '' });
    });
    test('cancels property name buffer on symbols', () => {
        const result = handleCommandTransition({
            curr: '!',
            section: 'propName',
            buffer: 'name',
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'outside', buffer: '' });
    });
    test('cancels property name buffer on slash', () => {
        const result = handleCommandTransition({
            curr: '/',
            section: 'propName',
            buffer: 'name',
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'outside', buffer: '' });
    });

    // Property value tests
    test('starts property value on single quote', () => {
        const result = handleCommandTransition({
            curr: "'",
            section: 'propValue', // Should already be marked as propValue because of the equals sign
            buffer: '',
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        // Includes the quote in the buffer
        expect(result).toEqual({ section: 'propValue', buffer: `'` });
    });
    test('starts property value on double quote', () => {
        const result = handleCommandTransition({
            curr: '"',
            section: 'propValue', // Should already be marked as propValue because of the equals sign
            buffer: '',
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        // Includes the quote in the buffer
        expect(result).toEqual({ section: 'propValue', buffer: `"` });
    });
    test('cancels property value if whitespace before quote - space', () => {
        const result = handleCommandTransition({
            curr: ' ',
            section: 'propValue',
            buffer: '',
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'outside', buffer: '' });
    });
    test('cancels property value if whitespace before quote - tab', () => {
        const result = handleCommandTransition({
            curr: '\t',
            section: 'propValue',
            buffer: '',
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'outside', buffer: '' });
    });
    test('cancels property value if newline before quote', () => {
        const result = handleCommandTransition({
            curr: '\n',
            section: 'propValue',
            buffer: '',
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'outside', buffer: '' });
    });
    test('cancels property value if other alphabets before quote', () => {
        const result = handleCommandTransition({
            curr: '你',
            section: 'propValue',
            buffer: '',
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'outside', buffer: '' });
    });
    test('cancels property value if emojis before quote', () => {
        const result = handleCommandTransition({
            curr: '👋',
            section: 'propValue',
            buffer: '',
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'outside', buffer: '' });
    });
    test('cancels property value if symbols before quote', () => {
        const result = handleCommandTransition({
            curr: '!',
            section: 'propValue',
            buffer: '',
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'outside', buffer: '' });
    });
    test('cancels property value if slash before quote', () => {
        const result = handleCommandTransition({
            curr: '/',
            section: 'propValue',
            buffer: '',
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'outside', buffer: '' });
    });
    test('continues property value if already in quotes - letter with single quote start', () => {
        const result = handleCommandTransition({
            curr: 'a',
            section: 'propValue',
            buffer: "'",
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'propValue', buffer: "'a" });
    });
    test('continues property value if already in quotes - letter with double quote start', () => {
        const result = handleCommandTransition({
            curr: 'a',
            section: 'propValue',
            buffer: '"',
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'propValue', buffer: '"a' });
    });
    test('continues property value if already in quotes - letter with single quote start and other text in buffer', () => {
        const result = handleCommandTransition({
            curr: 'a',
            section: 'propValue',
            buffer: "'test",
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'propValue', buffer: "'testa" });
    });
    test('continues property value if already in quotes - other language', () => {
        const result = handleCommandTransition({
            curr: '你',
            section: 'propValue',
            buffer: "'test",
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'propValue', buffer: "'test你" });
    });
    test('continues property value if already in quotes - emoji', () => {
        const result = handleCommandTransition({
            curr: '👋',
            section: 'propValue',
            buffer: "'test",
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'propValue', buffer: "'test👋" });
    });
    test('continues property value if already in quotes - symbol', () => {
        const result = handleCommandTransition({
            curr: '!',
            section: 'propValue',
            buffer: "'test",
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'propValue', buffer: "'test!" });
    });
    test('continues property value if already in quotes - slash', () => {
        const result = handleCommandTransition({
            curr: '/',
            section: 'propValue',
            buffer: "'test",
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'propValue', buffer: "'test/" });
    });
    test('continues property value if already in quotes - newline', () => {
        const result = handleCommandTransition({
            curr: '\n',
            section: 'propValue',
            buffer: "'test",
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'propValue', buffer: "'test\n" });
    });
    test('continues property value if already in quotes - space', () => {
        const result = handleCommandTransition({
            curr: ' ',
            section: 'propValue',
            buffer: '"test',
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'propValue', buffer: '"test ' });
    });
    test('continues property value if already in quotes - tab', () => {
        const result = handleCommandTransition({
            curr: '\t',
            section: 'propValue',
            buffer: '"test',
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'propValue', buffer: '"test\t' });
    });
    test('continues property value when curr is a different quote type than the starting quote - double with single start', () => {
        const result = handleCommandTransition({
            curr: '"',
            section: 'propValue',
            buffer: "'",
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'propValue', buffer: "'\"" });
    });
    test('continues property value when curr is a different quote type than the starting quote - single with double start', () => {
        const result = handleCommandTransition({
            curr: "'",
            section: 'propValue',
            buffer: '"',
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'propValue', buffer: '"\'' });
    });
    test('continues property value for escaped characters - curr is escape character, buffer is quote', () => {
        const result = handleCommandTransition({
            curr: '\\',
            section: 'propValue',
            buffer: "'",
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'propValue', buffer: `'\\` });
    });
    test('continues property value for escaped characters - curr is single quote, buffer is double quote and escape character', () => {
        const result = handleCommandTransition({
            curr: `"`,
            section: 'propValue',
            buffer: `"\\`,
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'propValue', buffer: `"\\"` });
    });
    test('continues property value for escaped characters - curr is single quote, buffer is single quote and escape character', () => {
        const result = handleCommandTransition({
            curr: `'`,
            section: 'propValue',
            buffer: `'\\`,
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'propValue', buffer: `'\\'` });
    });
    test('completes property value for an even number of escape characters', () => {
        const result = handleCommandTransition({
            curr: `'`,
            section: 'propValue',
            buffer: `'\\\\`,
            callback,
        });
        expect(callback).toHaveBeenCalledWith('propValue', `\\\\`);
        expect(result).toEqual({ section: 'propName', buffer: '' });
    });
    test('continues property value for an odd number of escape characters', () => {
        const result = handleCommandTransition({
            curr: `'`,
            section: 'propValue',
            buffer: `'\\\\\\`,
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'propValue', buffer: `'\\\\\\'` });
    });
    test('commits property value on closing quote - single quote', () => {
        const result = handleCommandTransition({
            curr: "'",
            section: 'propValue',
            buffer: "'test",
            callback,
        });
        expect(callback).toHaveBeenCalledWith('propValue', "test");
        expect(result).toEqual({ section: 'propName', buffer: '' });
    });
    test('commits property value on closing quote - double quote', () => {
        const result = handleCommandTransition({
            curr: '"',
            section: 'propValue',
            buffer: '"test',
            callback,
        });
        expect(callback).toHaveBeenCalledWith('propValue', 'test');
        expect(result).toEqual({ section: 'propName', buffer: '' });
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

