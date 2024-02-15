import { extractCommands, handleCommandTransition, isAlphaNum, isNewline, isWhitespace } from './commands';

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
    test('continues property value if buffer + curr might be a number - test 1', () => {
        const result = handleCommandTransition({
            curr: '1',
            section: 'propValue',
            buffer: '',
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'propValue', buffer: '1' });
    });
    test('continues property value if buffer + curr might be a number - test 2', () => {
        const result = handleCommandTransition({
            curr: '1',
            section: 'propValue',
            buffer: '-',
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'propValue', buffer: '-1' });
    });
    test('continues property value if buffer + curr might be a number - test 3', () => {
        const result = handleCommandTransition({
            curr: '.',
            section: 'propValue',
            buffer: '',
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'propValue', buffer: '.' });
    });
    test('continues property value if buffer + curr might be a number - test 4', () => {
        const result = handleCommandTransition({
            curr: '.',
            section: 'propValue',
            buffer: '3',
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'propValue', buffer: '3.' });
    });
    test('cancels property value if buffer + curr is an invalid number - test 1', () => {
        const result = handleCommandTransition({
            curr: '-',
            section: 'propValue',
            buffer: '-',
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'outside', buffer: '' });
    });
    test('cancels property value if buffer + curr is an invalid number - test 2', () => {
        const result = handleCommandTransition({
            curr: '-',
            section: 'propValue',
            buffer: '1',
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'outside', buffer: '' });
    });
    test('cancels property value if buffer + curr is an invalid number - test 3', () => {
        const result = handleCommandTransition({
            curr: '.',
            section: 'propValue',
            buffer: '3.',
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'outside', buffer: '' });
    });
    test('cancels property value if buffer + curr is an invalid number - test 4', () => {
        const result = handleCommandTransition({
            curr: '.',
            section: 'propValue',
            buffer: '1.2',
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'outside', buffer: '' });
    });
    test('continues property value if buffer + curr might be null - test 1', () => {
        const result = handleCommandTransition({
            curr: 'n',
            section: 'propValue',
            buffer: '',
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'propValue', buffer: 'n' });
    });
    test('continues property value if buffer + curr might be null - test 2', () => {
        const result = handleCommandTransition({
            curr: 'l',
            section: 'propValue',
            buffer: 'nul',
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'propValue', buffer: 'null' });
    });
    test('cancels property value if buffer + curr is not null', () => {
        const result = handleCommandTransition({
            curr: 'l',
            section: 'propValue',
            buffer: 'null',
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'outside', buffer: '' });
    });
    test('cancels property value if letter before quote', () => {
        const result = handleCommandTransition({
            curr: 'a',
            section: 'propValue',
            buffer: '',
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'outside', buffer: '' });
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
    test('commits number property value on space - test 1', () => {
        const result = handleCommandTransition({
            curr: ' ',
            section: 'propValue',
            buffer: '123',
            callback,
        });
        expect(callback).toHaveBeenCalledWith('propValue', 123);
        expect(result).toEqual({ section: 'propName', buffer: '' });
    });
    test('commits number property value on space - test 2', () => {
        const result = handleCommandTransition({
            curr: ' ',
            section: 'propValue',
            buffer: '-123',
            callback,
        });
        expect(callback).toHaveBeenCalledWith('propValue', -123);
        expect(result).toEqual({ section: 'propName', buffer: '' });
    });
    test('commits number property value on space - test 3', () => {
        const result = handleCommandTransition({
            curr: ' ',
            section: 'propValue',
            buffer: '-1.23',
            callback,
        });
        expect(callback).toHaveBeenCalledWith('propValue', -1.23);
        expect(result).toEqual({ section: 'propName', buffer: '' });
    });
    test('commits number property value on tab', () => {
        const result = handleCommandTransition({
            curr: '\t',
            section: 'propValue',
            buffer: '-1.23',
            callback,
        });
        expect(callback).toHaveBeenCalledWith('propValue', -1.23);
        expect(result).toEqual({ section: 'propName', buffer: '' });
    });
    test('commits number property value on newline', () => {
        const result = handleCommandTransition({
            curr: '\n',
            section: 'propValue',
            buffer: '-1.23',
            callback,
        });
        expect(callback).toHaveBeenCalledWith('propValue', -1.23);
        expect(result).toEqual({ section: 'outside', buffer: '' });
    });
    test('cancels number property value on letter', () => {
        const result = handleCommandTransition({
            curr: 'a',
            section: 'propValue',
            buffer: '-1.23',
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'outside', buffer: '' });
    });
    test('cancels number property value on other alphabet', () => {
        const result = handleCommandTransition({
            curr: '你',
            section: 'propValue',
            buffer: '-1.23',
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'outside', buffer: '' });
    });
    test('cancels number property value on emoji', () => {
        const result = handleCommandTransition({
            curr: '👋',
            section: 'propValue',
            buffer: '-1.23',
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'outside', buffer: '' });
    });
    test('cancels number property value on symbol', () => {
        const result = handleCommandTransition({
            curr: '!',
            section: 'propValue',
            buffer: '-1.23',
            callback,
        });
        expect(callback).not.toHaveBeenCalled();
        expect(result).toEqual({ section: 'outside', buffer: '' });
    });
});

describe('extractCommands', () => {
    test('ignores non-command slashes - test1', () => {
        const input = "a/command";
        const expected = [];
        expect(extractCommands(input)).toEqual(expected);
    });
    test('ignores non-command slashes - test2', () => {
        const input = "1/3";
        const expected = [];
        expect(extractCommands(input)).toEqual(expected);
    });
    test('ignores non-command slashes - test3', () => {
        const input = "/boop.";
        const expected = [];
        expect(extractCommands(input)).toEqual(expected);
    });
    test('ignores non-command slashes - test4', () => {
        const input = "/boop!";
        const expected = [];
        expect(extractCommands(input)).toEqual(expected);
    });
    test('ignores non-command slashes - test5', () => {
        const input = "/boop你";
        const expected = [];
        expect(extractCommands(input)).toEqual(expected);
    });
    test('ignores non-command slashes - test6', () => {
        const input = "/boop👋";
        const expected = [];
        expect(extractCommands(input)).toEqual(expected);
    });
    test('ignores non-command slashes - test7', () => {
        const input = "/boop/";
        const expected = [];
        expect(extractCommands(input)).toEqual(expected);
    });
    test('ignores non-command slashes - test8', () => {
        const input = "/boop\\";
        const expected = [];
        expect(extractCommands(input)).toEqual(expected);
    });
    test('ignores non-command slashes - test9', () => {
        const input = "/boop=";
        const expected = [];
        expect(extractCommands(input)).toEqual(expected);
    });
    test('ignores non-command slashes - test10', () => {
        const input = "/boop-";
        const expected = [];
        expect(extractCommands(input)).toEqual(expected);
    });
    test('ignores non-command slashes - test11', () => {
        const input = "/boop.";
        const expected = [];
        expect(extractCommands(input)).toEqual(expected);
    });
    test('ignores non-command slashes - test12', () => {
        const input = "//boop";
        const expected = [];
        expect(extractCommands(input)).toEqual(expected);
    });

    test('extracts simple command without action or properties - test 1', () => {
        const input = "/command";
        const expected = [{
            command: "command",
            action: null,
            properties: {},
            start: 0,
            end: input.length - 1,
        }];
        expect(extractCommands(input)).toEqual(expected);
    });
    test('extracts simple command without action or properties - test 2', () => {
        const input = "  /command";
        const expected = [{
            command: "command",
            action: null,
            properties: {},
            start: 2,
            end: input.length - 1,
        }];
        expect(extractCommands(input)).toEqual(expected);
    });
    test('extracts simple command without action or properties - test 3', () => {
        const input = "/command  ";
        const expected = [{
            command: "command",
            action: null,
            properties: {},
            start: 0,
            end: input.length - 3,
        }];
        expect(extractCommands(input)).toEqual(expected);
    });
    test('extracts simple command without action or properties - test 4', () => {
        const input = "\n/command";
        const expected = [{
            command: "command",
            action: null,
            properties: {},
            start: 1,
            end: input.length - 1,
        }];
        expect(extractCommands(input)).toEqual(expected);
    });
    test('extracts simple command without action or properties - test 5', () => {
        const input = "/command\n";
        const expected = [{
            command: "command",
            action: null,
            properties: {},
            start: 0,
            end: input.length - 2,
        }];
        expect(extractCommands(input)).toEqual(expected);
    });
    test('extracts simple command without action or properties - test 6', () => {
        const input = "/command\t";
        const expected = [{
            command: "command",
            action: null,
            properties: {},
            start: 0,
            end: input.length - 2,
        }];
        expect(extractCommands(input)).toEqual(expected);
    });

    test('extracts command with action - test1', () => {
        const input = "/command action";
        const expected = [{
            command: "command",
            action: "action",
            properties: {},
            start: 0,
            end: input.length - 1,
        }];
        expect(extractCommands(input)).toEqual(expected);
    });
    //TODO more

    test('handles command with mixed properties - test 1', () => {
        const input = "/command prop1=123 prop2='value' prop3=null";
        const expected = [{
            command: "command",
            action: null,
            properties: { prop1: 123, prop2: 'value', prop3: null },
            start: 0,
            end: input.length - 1,
        }];
        expect(extractCommands(input)).toEqual(expected);
    });
    //TODO more

    test('handles newline properly - test 1', () => {
        const input = "/command1\n/command2";
        const expected = [
            { command: "command1", action: null, properties: {}, start: 0, end: 9 },
            { command: "command2", action: null, properties: {}, start: 10, end: 19 }
        ];
        expect(extractCommands(input)).toEqual(expected);
    });
    //TODO more

    test('handles escaped quotes in property values - test 1', () => {
        const input = "/command prop='value with \\'escaped quotes\\''";
        const expected = [{
            command: "command",
            action: null,
            properties: { prop: "value with 'escaped quotes'" },
            start: 0,
            end: input.length - 1,
        }];
        expect(extractCommands(input)).toEqual(expected);
    });
    //TODO more

    test('handles complex scenario with multiple commands and properties - test 1', () => {
        const input = "/cmd1 prop1=123 /cmd2 action2 prop2='text' prop3=4.56\n/cmd3 prop4=null";
        const expected = [
            { command: "cmd1", action: null, properties: { prop1: 123 }, start: 0, end: 13 },
            { command: "cmd2", action: "action2", properties: { prop2: 'text', prop3: 4.56 }, start: 14, end: 44 },
            { command: "cmd3", action: null, properties: { prop4: null }, start: 45, end: 62 }
        ];
        expect(extractCommands(input)).toEqual(expected);
    });
    //TODO more

    // TODO tests that include non-command text, performance with long inputs, etc.
});
