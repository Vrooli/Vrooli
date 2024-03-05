import { LlmTask } from "@local/shared";
import { LlmCommand, MaybeLlmCommand, extractCommands, filterInvalidCommands, handleCommandTransition, isAlphaNum, isNewline, isWhitespace, removeCommands } from "./commands";
import { CommandToTask } from "./config";

describe("isNewline", () => {
    test("recognizes newline character", () => {
        expect(isNewline("\n")).toBe(true);
    });

    test("recognizes carriage return character", () => {
        expect(isNewline("\r")).toBe(true);
    });

    test("does not recognize space as newline", () => {
        expect(isNewline(" ")).toBe(false);
    });

    test("does not recognize alphabetic character as newline", () => {
        expect(isNewline("a")).toBe(false);
    });

    test("does not recognize numeric character as newline", () => {
        expect(isNewline("1")).toBe(false);
    });
});

describe("isWhitespace", () => {
    test("recognizes space as whitespace", () => {
        expect(isWhitespace(" ")).toBe(true);
    });

    test("recognizes tab as whitespace", () => {
        expect(isWhitespace("\t")).toBe(true);
    });

    test("does not recognize newline as whitespace", () => {
        expect(isWhitespace("\n")).toBe(false);
    });

    test("does not recognize carriage return as whitespace", () => {
        expect(isWhitespace("\r")).toBe(false);
    });

    test("does not recognize alphabetic character as whitespace", () => {
        expect(isWhitespace("a")).toBe(false);
    });

    test("does not recognize numeric character as whitespace", () => {
        expect(isWhitespace("1")).toBe(false);
    });
});

describe("isAlphaNum", () => {
    test("recognizes lowercase alphabetic character", () => {
        expect(isAlphaNum("a")).toBe(true);
        expect(isAlphaNum("z")).toBe(true);
    });

    test("recognizes uppercase alphabetic character", () => {
        expect(isAlphaNum("z")).toBe(true);
        expect(isAlphaNum("Z")).toBe(true);
    });

    test("recognizes numeric character", () => {
        expect(isAlphaNum("0")).toBe(true);
        expect(isAlphaNum("9")).toBe(true);
    });

    test("does not recognize space as alphanumeric", () => {
        expect(isAlphaNum(" ")).toBe(false);
    });

    test("does not recognize newline as alphanumeric", () => {
        expect(isAlphaNum("\n")).toBe(false);
    });

    test("does not recognize special character as alphanumeric", () => {
        expect(isAlphaNum("*")).toBe(false);
    });

    test("does not recognize accented characters as alphanumeric", () => {
        expect(isAlphaNum("á")).toBe(false);
        expect(isAlphaNum("ñ")).toBe(false);
    });

    test("does not recognize characters from other alphabets as alphanumeric", () => {
        expect(isAlphaNum("你")).toBe(false);
        expect(isAlphaNum("👋")).toBe(false);
    });
});

describe("handleCommandTransition", () => {
    let onCommit, onComplete, onCancel, onStart;
    beforeEach(() => {
        onCommit = jest.fn();
        onComplete = jest.fn();
        onCancel = jest.fn();
        onStart = jest.fn();
    });

    // Outside tests
    test("keep adding to outside when not on a slash - space", () => {
        const buffer = "asdf";
        const result = handleCommandTransition({
            curr: " ",
            prev: buffer[buffer.length - 1],
            section: "outside",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "outside", buffer: [...buffer.split(""), " "] });
    });
    test("keep adding to outside when not on a slash - tab", () => {
        const buffer = "asdf";
        const result = handleCommandTransition({
            curr: "\t",
            prev: buffer[buffer.length - 1],
            section: "outside",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "outside", buffer: [...buffer.split(""), "\t"] });
    });
    test("keep adding to outside when not on a slash - single quote", () => {
        const buffer = "asdf";
        const result = handleCommandTransition({
            curr: "'",
            prev: buffer[buffer.length - 1],
            section: "outside",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "outside", buffer: [...buffer.split(""), "'"] });
    });
    test("keep adding to outside when not on a slash - double quote", () => {
        const buffer = "asdf";
        const result = handleCommandTransition({
            curr: "\"",
            prev: buffer[buffer.length - 1],
            section: "outside",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "outside", buffer: [...buffer.split(""), "\""] });
    });
    test("keep adding to outside when not on a slash - equals sign", () => {
        const buffer = "asdf";
        const result = handleCommandTransition({
            curr: "=",
            prev: buffer[buffer.length - 1],
            section: "outside",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "outside", buffer: [...buffer.split(""), "="] });
    });
    test("keep adding to outside when not on a slash - letter", () => {
        const buffer = "asdf";
        const result = handleCommandTransition({
            curr: "a",
            prev: buffer[buffer.length - 1],
            section: "outside",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "outside", buffer: [...buffer.split(""), "a"] });
    });
    test("keep adding to outside when not on a slash - number", () => {
        const buffer = "asdf";
        const result = handleCommandTransition({
            curr: "1",
            prev: buffer[buffer.length - 1],
            section: "outside",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "outside", buffer: [...buffer.split(""), "1"] });
    });
    test("keep adding to outside when not on a slash - newline", () => {
        const buffer = "asdf";
        const result = handleCommandTransition({
            curr: "\n",
            prev: buffer[buffer.length - 1],
            section: "outside",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "outside", buffer: [...buffer.split(""), "\n"] });
    });
    test("keep adding to outside when not on a slash - other alphabets", () => {
        const buffer = "asdf";
        const result = handleCommandTransition({
            curr: "你",
            prev: buffer[buffer.length - 1],
            section: "outside",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "outside", buffer: [...buffer.split(""), "你"] });
    });
    test("keep adding to outside when not on a slash - emojis", () => {
        const buffer = "asdf";
        const result = handleCommandTransition({
            curr: "👋",
            prev: buffer[buffer.length - 1],
            section: "outside",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "outside", buffer: [...buffer.split(""), "👋"] });
    });
    test("keep adding to outside when not on a slash - symbols", () => {
        const buffer = "asdf";
        const result = handleCommandTransition({
            curr: "!",
            prev: buffer[buffer.length - 1],
            section: "outside",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "outside", buffer: [...buffer.split(""), "!"] });
    });

    // Command tests
    test("does not start a command when the slash is not preceeded by whitespace - letter", () => {
        const buffer = "asdf";
        const result = handleCommandTransition({
            curr: "/",
            prev: buffer[buffer.length - 1],
            section: "outside",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "outside", buffer: [...buffer.split(""), "/"] });
    });
    test("does not start a command when the slash is not preceeded by whitespace - number", () => {
        const buffer = "1234";
        const result = handleCommandTransition({
            curr: "/",
            prev: buffer[buffer.length - 1],
            section: "outside",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "outside", buffer: [...buffer.split(""), "/"] });
    });
    test("does not start a command when the slash is not preceeded by whitespace - symbol", () => {
        const buffer = "!@#$";
        const result = handleCommandTransition({
            curr: "/",
            prev: buffer[buffer.length - 1],
            section: "outside",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({
            section: "outside", buffer: [...buffer.split(""), "/"],
        });
    });
    test("does not start a command when the slash is not preceeded by whitespace - emoji", () => {
        const buffer = "🙌💃";
        const result = handleCommandTransition({
            curr: "/",
            prev: buffer[buffer.length - 1],
            section: "outside",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "outside", buffer: [...buffer.split(""), "/"] });
    });
    test("starts a command when buffer is empty", () => {
        const buffer = "";
        const result = handleCommandTransition({
            curr: "/",
            prev: buffer[buffer.length - 1],
            section: "outside",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "command", buffer: [] });
    });
    test("starts a command after a newline", () => {
        const buffer = "asdf\n";
        const result = handleCommandTransition({
            curr: "/",
            prev: buffer[buffer.length - 1],
            section: "outside",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "command", buffer: [] });
    });
    test("starts a command after whitespace - space", () => {
        const buffer = "asdf ";
        const result = handleCommandTransition({
            curr: "/",
            prev: buffer[buffer.length - 1],
            section: "outside",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "command", buffer: [] });
    });
    test("starts a command after whitespace - tab", () => {
        const buffer = "asdf\t";
        const result = handleCommandTransition({
            curr: "/",
            prev: buffer[buffer.length - 1],
            section: "outside",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "command", buffer: [] });
    });
    test("adds letter to command buffer", () => {
        const buffer = "test";
        const result = handleCommandTransition({
            curr: "a",
            prev: buffer[buffer.length - 1],
            section: "command",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "command", buffer: [...buffer.split(""), "a"] });
    });
    test("adds number to command buffer", () => {
        const buffer = "test";
        const result = handleCommandTransition({
            curr: "1",
            prev: buffer[buffer.length - 1],
            section: "command",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "command", buffer: [...buffer.split(""), "1"] });
    });
    test("commmits on newline", () => {
        const buffer = "test";
        const result = handleCommandTransition({
            curr: "\n",
            prev: buffer[buffer.length - 1],
            section: "command",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).toHaveBeenCalledWith("command", buffer);
        expect(result).toEqual({ section: "outside", buffer: [] });
    });
    test("resets to outside on other alphabets", () => {
        const buffer = "test";
        const result = handleCommandTransition({
            curr: "你",
            prev: buffer[buffer.length - 1],
            section: "command",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "outside", buffer: [] });
    });
    test("resets to outside on emojis", () => {
        const buffer = "test";
        const result = handleCommandTransition({
            curr: "👋",
            prev: buffer[buffer.length - 1],
            section: "command",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "outside", buffer: [] });
    });
    test("resets to outside on symbols", () => {
        const buffer = "test";
        const result = handleCommandTransition({
            curr: "!",
            prev: buffer[buffer.length - 1],
            section: "command",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "outside", buffer: [] });
    });

    // Pending (when we're not sure if it's an action or a property yet) tests
    test("starts pending actionwhen we encounter the first space after a command - space", () => {
        const buffer = "test";
        const result = handleCommandTransition({
            curr: " ",
            prev: buffer[buffer.length - 1],
            section: "command",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).toHaveBeenCalledWith("command", buffer);
        expect(result).toEqual({ section: "action", buffer: [] });
    });
    test("starts pending action when we encounter the first space after a command - tab", () => {
        const buffer = "test";
        const result = handleCommandTransition({
            curr: "\t",
            prev: buffer[buffer.length - 1],
            section: "command",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).toHaveBeenCalledWith("command", buffer);
        expect(result).toEqual({ section: "action", buffer: [] });
    });
    test("does not start pending action for newline", () => {
        const buffer = "test";
        const result = handleCommandTransition({
            curr: "\n",
            prev: buffer[buffer.length - 1],
            section: "command",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).toHaveBeenCalledWith("command", buffer);
        expect(result).toEqual({ section: "outside", buffer: [] });
    });
    test("does not start pending action for other alphabets", () => {
        const buffer = "test";
        const result = handleCommandTransition({
            curr: "你",
            prev: buffer[buffer.length - 1],
            section: "command",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "outside", buffer: [] });
    });
    test("does not start pending action for emojis", () => {
        const buffer = "test";
        const result = handleCommandTransition({
            curr: "👋",
            prev: buffer[buffer.length - 1],
            section: "command",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "outside", buffer: [] });
    });
    test("does not start pending action for symbols", () => {
        const buffer = "test";
        const result = handleCommandTransition({
            curr: "!",
            prev: buffer[buffer.length - 1],
            section: "command",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "outside", buffer: [] });
    });
    test("cancels pending action on other alphabets", () => {
        const buffer = "test";
        const result = handleCommandTransition({
            curr: "你",
            prev: buffer[buffer.length - 1],
            section: "action",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "outside", buffer: [] });
    });
    test("cancels pending action on emojis", () => {
        const buffer = "test";
        const result = handleCommandTransition({
            curr: "👋",
            prev: buffer[buffer.length - 1],
            section: "action",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "outside", buffer: [] });
    });
    test("cancels pending action on symbols", () => {
        const buffer = "test";
        const result = handleCommandTransition({
            curr: "!",
            prev: buffer[buffer.length - 1],
            section: "action",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "outside", buffer: [] });
    });

    // Action tests
    test("commits pending action buffer to action on whitespace - space", () => {
        const buffer = "add";
        const result = handleCommandTransition({
            curr: " ",
            prev: buffer[buffer.length - 1],
            section: "action",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).toHaveBeenCalledWith("action", buffer);
        expect(result).toEqual({ section: "propName", buffer: [] });
    });
    test("commits pending action buffer to action on whitespace - tab", () => {
        const buffer = "add";
        const result = handleCommandTransition({
            curr: "\t",
            prev: buffer[buffer.length - 1],
            section: "action",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).toHaveBeenCalledWith("action", buffer);
        expect(result).toEqual({ section: "propName", buffer: [] });
    });
    test("commits pending action buffer to action on newline", () => {
        const buffer = "add";
        const result = handleCommandTransition({
            curr: "\n",
            prev: buffer[buffer.length - 1],
            section: "action",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).toHaveBeenCalledWith("action", buffer);
        expect(result).toEqual({ section: "outside", buffer: [] });
    });

    // Property name tests
    test("commits pending action buffer to property name on equals sign", () => {
        const buffer = "name";
        const result = handleCommandTransition({
            curr: "=",
            prev: buffer[buffer.length - 1],
            section: "action",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).toHaveBeenCalledWith("propName", buffer);
        expect(result).toEqual({ section: "propValue", buffer: [] });
    });
    test("commits property name buffer to property name on equals sign", () => {
        const buffer = "name";
        const result = handleCommandTransition({
            curr: "=",
            prev: buffer[buffer.length - 1],
            section: "propName",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).toHaveBeenCalledWith("propName", buffer);
        expect(result).toEqual({ section: "propValue", buffer: [] });
    });
    test("cancels property name buffer on whitespace - space", () => {
        const buffer = "name";
        const result = handleCommandTransition({
            curr: " ",
            prev: buffer[buffer.length - 1],
            section: "propName",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "outside", buffer: [] });
    });
    test("cancels property name buffer on whitespace - tab", () => {
        const buffer = "name";
        const result = handleCommandTransition({
            curr: "\t",
            prev: buffer[buffer.length - 1],
            section: "propName",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "outside", buffer: [] });
    });
    test("cancels property name buffer on newline", () => {
        const buffer = "name";
        const result = handleCommandTransition({
            curr: "\n",
            prev: buffer[buffer.length - 1],
            section: "propName",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "outside", buffer: [] });
    });
    test("cancels property name buffer on other alphabets", () => {
        const buffer = "name";
        const result = handleCommandTransition({
            curr: "你",
            prev: buffer[buffer.length - 1],
            section: "propName",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "outside", buffer: [] });
    });
    test("cancels property name buffer on emojis", () => {
        const buffer = "name";
        const result = handleCommandTransition({
            curr: "👋",
            prev: buffer[buffer.length - 1],
            section: "propName",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "outside", buffer: [] });
    });
    test("cancels property name buffer on symbols", () => {
        const buffer = "name";
        const result = handleCommandTransition({
            curr: "!",
            prev: buffer[buffer.length - 1],
            section: "propName",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "outside", buffer: [] });
    });
    test("cancels property name buffer on slash", () => {
        const buffer = "name";
        const result = handleCommandTransition({
            curr: "/",
            prev: buffer[buffer.length - 1],
            section: "propName",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "outside", buffer: [] });
    });

    // Property value tests
    test("starts property value on single quote", () => {
        const buffer = "";
        const result = handleCommandTransition({
            curr: "'",
            prev: buffer[buffer.length - 1],
            section: "propValue", // Should already be marked as propValue because of the equals sign
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        // Includes the quote in the buffer
        expect(result).toEqual({ section: "propValue", buffer: ["'"] });
    });
    test("starts property value on double quote", () => {
        const buffer = "";
        const result = handleCommandTransition({
            curr: "\"",
            prev: buffer[buffer.length - 1],
            section: "propValue", // Should already be marked as propValue because of the equals sign
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        // Includes the quote in the buffer
        expect(result).toEqual({ section: "propValue", buffer: [...buffer.split(""), "\""] });
    });
    test("continues property value if buffer + curr might be a number - test 1", () => {
        const buffer = "";
        const result = handleCommandTransition({
            curr: "1",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "propValue", buffer: [...buffer.split(""), "1"] });
    });
    test("continues property value if buffer + curr might be a number - test 2", () => {
        const buffer = "-";
        const result = handleCommandTransition({
            curr: "1",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "propValue", buffer: [...buffer.split(""), "1"] });
    });
    test("continues property value if buffer + curr might be a number - test 3", () => {
        const buffer = "";
        const result = handleCommandTransition({
            curr: ".",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "propValue", buffer: [...buffer.split(""), "."] });
    });
    test("continues property value if buffer + curr might be a number - test 4", () => {
        const buffer = "3";
        const result = handleCommandTransition({
            curr: ".",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "propValue", buffer: [...buffer.split(""), "."] });
    });
    test("cancels property value if buffer + curr is an invalid number - test 1", () => {
        const buffer = "-";
        const result = handleCommandTransition({
            curr: "-",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "outside", buffer: [] });
    });
    test("cancels property value if buffer + curr is an invalid number - test 2", () => {
        const buffer = "1";
        const result = handleCommandTransition({
            curr: "-",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "outside", buffer: [] });
    });
    test("cancels property value if buffer + curr is an invalid number - test 3", () => {
        const buffer = "3.";
        const result = handleCommandTransition({
            curr: ".",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "outside", buffer: [] });
    });
    test("cancels property value if buffer + curr is an invalid number - test 4", () => {
        const buffer = "1.2";
        const result = handleCommandTransition({
            curr: ".",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "outside", buffer: [] });
    });
    test("continues property value if buffer + curr might be null - test 1", () => {
        const buffer = "";
        const result = handleCommandTransition({
            curr: "n",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "propValue", buffer: ["n"] });
    });
    test("continues property value if buffer + curr might be null - test 2", () => {
        const buffer = "nul";
        const result = handleCommandTransition({
            curr: "l",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "propValue", buffer: [...buffer.split(""), "l"] });
    });
    test("cancels property value if buffer + curr is not null", () => {
        const buffer = "null";
        const result = handleCommandTransition({
            curr: "l",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "outside", buffer: [] });
    });
    test("cancels property value if letter before quote", () => {
        const buffer = "";
        const result = handleCommandTransition({
            curr: "a",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "outside", buffer: [] });
    });
    test("cancels property value if whitespace before quote - space", () => {
        const buffer = "";
        const result = handleCommandTransition({
            curr: " ",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "outside", buffer: [] });
    });
    test("cancels property value if whitespace before quote - tab", () => {
        const buffer = "";
        const result = handleCommandTransition({
            curr: "\t",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "outside", buffer: [] });
    });
    test("cancels property value if newline before quote", () => {
        const buffer = "";
        const result = handleCommandTransition({
            curr: "\n",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "outside", buffer: [] });
    });
    test("cancels property value if other alphabets before quote", () => {
        const buffer = "";
        const result = handleCommandTransition({
            curr: "你",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "outside", buffer: [] });
    });
    test("cancels property value if emojis before quote", () => {
        const buffer = "";
        const result = handleCommandTransition({
            curr: "👋",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "outside", buffer: [] });
    });
    test("cancels property value if symbols before quote", () => {
        const buffer = "";
        const result = handleCommandTransition({
            curr: "!",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "outside", buffer: [] });
    });
    test("cancels property value if slash before quote", () => {
        const buffer = "";
        const result = handleCommandTransition({
            curr: "/",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "outside", buffer: [] });
    });
    test("continues property value if already in quotes - letter with single quote start", () => {
        const buffer = "'";
        const result = handleCommandTransition({
            curr: "a",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "propValue", buffer: [...buffer.split(""), "a"] });
    });
    test("continues property value if already in quotes - letter with double quote start", () => {
        const buffer = "\"";
        const result = handleCommandTransition({
            curr: "a",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "propValue", buffer: [...buffer.split(""), "a"] });
    });
    test("continues property value if already in quotes - letter with single quote start and other text in buffer", () => {
        const buffer = "'test";
        const result = handleCommandTransition({
            curr: "a",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "propValue", buffer: [...buffer.split(""), "a"] });
    });
    test("continues property value if already in quotes - other language", () => {
        const buffer = "'test";
        const result = handleCommandTransition({
            curr: "你",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "propValue", buffer: [...buffer.split(""), "你"] });
    });
    test("continues property value if already in quotes - emoji", () => {
        const buffer = "'test";
        const result = handleCommandTransition({
            curr: "👋",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "propValue", buffer: [...buffer.split(""), "👋"] });
    });
    test("continues property value if already in quotes - symbol", () => {
        const buffer = "'test";
        const result = handleCommandTransition({
            curr: "!",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "propValue", buffer: [...buffer.split(""), "!"] });
    });
    test("continues property value if already in quotes - slash", () => {
        const buffer = "'test";
        const result = handleCommandTransition({
            curr: "/",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "propValue", buffer: [...buffer.split(""), "/"] });
    });
    test("continues property value if already in quotes - newline", () => {
        const buffer = "'test";
        const result = handleCommandTransition({
            curr: "\n",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "propValue", buffer: [...buffer.split(""), "\n"] });
    });
    test("continues property value if already in quotes - space", () => {
        const buffer = "\"test";
        const result = handleCommandTransition({
            curr: " ",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "propValue", buffer: [...buffer.split(""), " "] });
    });
    test("continues property value if already in quotes - tab", () => {
        const buffer = "\"test";
        const result = handleCommandTransition({
            curr: "\t",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "propValue", buffer: [...buffer.split(""), "\t"] });
    });
    test("continues property value when curr is a different quote type than the starting quote - double with single start", () => {
        const buffer = "'";
        const result = handleCommandTransition({
            curr: "\"",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "propValue", buffer: ["'", "\""] });
    });
    test("continues property value when curr is a different quote type than the starting quote - single with double start", () => {
        const buffer = "\"";
        const result = handleCommandTransition({
            curr: "'",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "propValue", buffer: ["\"", "'"] });
    });
    test("continues property value for escaped characters - curr is escape character, buffer is quote", () => {
        const buffer = "'";
        const result = handleCommandTransition({
            curr: "\\",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "propValue", buffer: ["'", "\\"] });
    });
    test("continues property value for escaped characters - curr is single quote, buffer is double quote and escape character", () => {
        const buffer = "\"\\";
        const result = handleCommandTransition({
            curr: "\"",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "propValue", buffer: ["\"", "\\", "\""] });
    });
    test("continues property value for escaped characters - curr is single quote, buffer is single quote and escape character", () => {
        const buffer = "'\\";
        const result = handleCommandTransition({
            curr: "'",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "propValue", buffer: ["'", "\\", "'"] });
    });
    test("completes property value for an even number of escape characters", () => {
        const buffer = "'\\\\";
        const result = handleCommandTransition({
            curr: "'",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).toHaveBeenCalledWith("propValue", "\\\\");
        expect(result).toEqual({ section: "propName", buffer: [] });
    });
    test("continues property value for an odd number of escape characters", () => {
        const buffer = "'\\\\\\";
        const result = handleCommandTransition({
            curr: "'",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "propValue", buffer: [...buffer.split(""), "'"] });
    });
    test("commits property value on closing quote - single quote", () => {
        const buffer = "'test";
        const result = handleCommandTransition({
            curr: "'",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).toHaveBeenCalledWith("propValue", "test");
        expect(result).toEqual({ section: "propName", buffer: [] });
    });
    test("commits property value on closing quote - double quote", () => {
        const buffer = "\"test";
        const result = handleCommandTransition({
            curr: "\"",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).toHaveBeenCalledWith("propValue", "test");
        expect(result).toEqual({ section: "propName", buffer: [] });
    });
    test("commits number property value on space - test 1", () => {
        const buffer = "123";
        const result = handleCommandTransition({
            curr: " ",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).toHaveBeenCalledWith("propValue", 123);
        expect(result).toEqual({ section: "propName", buffer: [] });
    });
    test("commits number property value on space - test 2", () => {
        const buffer = "-123";
        const result = handleCommandTransition({
            curr: " ",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).toHaveBeenCalledWith("propValue", -123);
        expect(result).toEqual({ section: "propName", buffer: [] });
    });
    test("commits number property value on space - test 3", () => {
        const buffer = "-1.23";
        const result = handleCommandTransition({
            curr: " ",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).toHaveBeenCalledWith("propValue", -1.23);
        expect(result).toEqual({ section: "propName", buffer: [] });
    });
    test("commits number property value on tab", () => {
        const buffer = "-1.23";
        const result = handleCommandTransition({
            curr: "\t",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).toHaveBeenCalledWith("propValue", -1.23);
        expect(result).toEqual({ section: "propName", buffer: [] });
    });
    test("commits number property value on newline", () => {
        const buffer = "-1.23";
        const result = handleCommandTransition({
            curr: "\n",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).toHaveBeenCalledWith("propValue", -1.23);
        expect(result).toEqual({ section: "outside", buffer: [] });
    });
    test("cancels number property value on letter", () => {
        const buffer = "-1.23";
        const result = handleCommandTransition({
            curr: "a",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "outside", buffer: [] });
    });
    test("cancels number property value on other alphabet", () => {
        const buffer = "-1.23";
        const result = handleCommandTransition({
            curr: "你",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "outside", buffer: [] });
    });
    test("cancels number property value on emoji", () => {
        const buffer = "-1.23";
        const result = handleCommandTransition({
            curr: "👋",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "outside", buffer: [] });
    });
    test("cancels number property value on symbol", () => {
        const buffer = "-1.23";
        const result = handleCommandTransition({
            curr: "!",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            onCommit,
            onComplete,
            onCancel,
            onStart,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toEqual({ section: "outside", buffer: [] });
    });
});

const getStartEnd = (str: string, sub: string): { start: number, end: number } => ({
    start: str.indexOf(sub),
    end: str.indexOf(sub) + sub.length, // Index after the last character
});

describe("extractCommands", () => {
    // Mock commandToTask
    const commandToTask: CommandToTask = (command, action) => {
        if (action) return `${command} ${action}` as LlmTask;
        return command as LlmTask;
    };

    test("ignores non-command slashes - test 1", () => {
        const input = "a/command";
        const expected = [];
        expect(extractCommands(input, commandToTask)).toEqual(expected);
    });
    test("ignores non-command slashes - test 2", () => {
        const input = "1/3";
        const expected = [];
        expect(extractCommands(input, commandToTask)).toEqual(expected);
    });
    test("ignores non-command slashes - test 3", () => {
        const input = "/boop.";
        const expected = [];
        expect(extractCommands(input, commandToTask)).toEqual(expected);
    });
    test("ignores non-command slashes - test 4", () => {
        const input = "/boop!";
        const expected = [];
        expect(extractCommands(input, commandToTask)).toEqual(expected);
    });
    test("ignores non-command slashes - test 5", () => {
        const input = "/boop你";
        const expected = [];
        expect(extractCommands(input, commandToTask)).toEqual(expected);
    });
    test("ignores non-command slashes - test 6", () => {
        const input = "/boop👋";
        const expected = [];
        expect(extractCommands(input, commandToTask)).toEqual(expected);
    });
    test("ignores non-command slashes - test 7", () => {
        const input = "/boop/";
        const expected = [];
        expect(extractCommands(input, commandToTask)).toEqual(expected);
    });
    test("ignores non-command slashes - test 8", () => {
        const input = "/boop\\";
        const expected = [];
        expect(extractCommands(input, commandToTask)).toEqual(expected);
    });
    test("ignores non-command slashes - test 9", () => {
        const input = "/boop=";
        const expected = [];
        expect(extractCommands(input, commandToTask)).toEqual(expected);
    });
    test("ignores non-command slashes - test 10", () => {
        const input = "/boop-";
        const expected = [];
        expect(extractCommands(input, commandToTask)).toEqual(expected);
    });
    test("ignores non-command slashes - test 11", () => {
        const input = "/boop.";
        const expected = [];
        expect(extractCommands(input, commandToTask)).toEqual(expected);
    });
    test("ignores non-command slashes - test 12", () => {
        const input = "//boop";
        const expected = [];
        expect(extractCommands(input, commandToTask)).toEqual(expected);
    });
    test("ignores non-command slashes - test 13", () => {
        const input = "//bippity /boppity! /boop💃 /realCommand";
        const expected = [{
            task: "realCommand",
            command: "realCommand",
            action: null,
            properties: {},
            ...getStartEnd(input, "/realCommand"),
        }];
        expect(extractCommands(input, commandToTask)).toEqual(expected);
    });

    test("extracts simple command without action or properties - test 1", () => {
        const input = "/command";
        const expected = [{
            task: "command",
            command: "command",
            action: null,
            properties: {},
            ...getStartEnd(input, "/command"),
        }];
        expect(extractCommands(input, commandToTask)).toEqual(expected);
    });
    test("extracts simple command without action or properties - test 2", () => {
        const input = "  /command";
        const expected = [{
            task: "command",
            command: "command",
            action: null,
            properties: {},
            ...getStartEnd(input, "/command"),
        }];
        expect(extractCommands(input, commandToTask)).toEqual(expected);
    });
    test("extracts simple command without action or properties - test 3", () => {
        const input = "/command  ";
        const expected = [{
            task: "command",
            command: "command",
            action: null,
            properties: {},
            ...getStartEnd(input, "/command"),
        }];
        expect(extractCommands(input, commandToTask)).toEqual(expected);
    });
    test("extracts simple command without action or properties - test 4", () => {
        const input = "aasdf\n/command";
        const expected = [{
            task: "command",
            command: "command",
            action: null,
            properties: {},
            ...getStartEnd(input, "/command"),
        }];
        expect(extractCommands(input, commandToTask)).toEqual(expected);
    });
    test("extracts simple command without action or properties - test 5", () => {
        const input = "/command\n";
        const expected = [{
            task: "command",
            command: "command",
            action: null,
            properties: {},
            ...getStartEnd(input, "/command"),
        }];
        expect(extractCommands(input, commandToTask)).toEqual(expected);
    });
    test("extracts simple command without action or properties - test 6", () => {
        const input = "/command\t";
        const expected = [{
            task: "command",
            command: "command",
            action: null,
            properties: {},
            ...getStartEnd(input, "/command"),
        }];
        expect(extractCommands(input, commandToTask)).toEqual(expected);
    });
    test("extracts simple command without action or properties - test 7", () => {
        const input = "/command invalidActionBecauseSymbol!";
        const expected = [{
            task: "command",
            command: "command",
            action: null,
            properties: {},
            ...getStartEnd(input, "/command"),
        }];
        expect(extractCommands(input, commandToTask)).toEqual(expected);
    });
    test("extracts simple command without action or properties - test 8", () => {
        const input = "/command invalidActionBecauseLanguage你";
        const expected = [{
            task: "command",
            command: "command",
            action: null,
            properties: {},
            ...getStartEnd(input, "/command"),
        }];
        expect(extractCommands(input, commandToTask)).toEqual(expected);
    });

    test("extracts command with action - test 1", () => {
        const input = "/command action";
        const expected = [{
            task: "command action",
            command: "command",
            action: "action",
            properties: {},
            ...getStartEnd(input, "/command action"),
        }];
        expect(extractCommands(input, commandToTask)).toEqual(expected);
    });
    test("extracts command with action - test 2", () => {
        const input = "/command action other words";
        const expected = [{
            task: "command action",
            command: "command",
            action: "action",
            properties: {},
            ...getStartEnd(input, "/command action"),
        }];
        expect(extractCommands(input, commandToTask)).toEqual(expected);
    });
    test("extracts command with action - test 3", () => {
        const input = "/command action invalidProp= 'space after equals'";
        const expected = [{
            task: "command action",
            command: "command",
            action: "action",
            properties: {},
            ...getStartEnd(input, "/command action"),
        }];
        expect(extractCommands(input, commandToTask)).toEqual(expected);
    });
    test("extracts command with action - test 4", () => {
        const input = "/command action\t";
        const expected = [{
            task: "command action",
            command: "command",
            action: "action",
            properties: {},
            ...getStartEnd(input, "/command action"),
        }];
        expect(extractCommands(input, commandToTask)).toEqual(expected);
    });
    test("extracts command with action - test 5", () => {
        const input = "/command action\n";
        const expected = [{
            task: "command action",
            command: "command",
            action: "action",
            properties: {},
            ...getStartEnd(input, "/command action"),
        }];
        expect(extractCommands(input, commandToTask)).toEqual(expected);
    });
    test("extracts command with action - test 6", () => {
        const input = "/command\taction";
        const expected = [{
            task: "command action",
            command: "command",
            action: "action",
            properties: {},
            ...getStartEnd(input, "/command\taction"),
        }];
        expect(extractCommands(input, commandToTask)).toEqual(expected);
    });

    test("handles command with properties - test 1", () => {
        const input = "/command prop1=123 prop2='value' prop3=null";
        const expected = [{
            task: "command",
            command: "command",
            action: null,
            properties: expect.objectContaining({ prop1: 123, prop2: "value", prop3: null }),
            ...getStartEnd(input, input),
        }];
        const commands = extractCommands(input, commandToTask);
        expect(commands).toEqual(expected);
        expect(Object.keys(commands[0].properties ?? {}).length).toBe(3);
    });
    test("handles command with properties - test 2", () => {
        const input = "/command prop1=\"123\" prop2='value' prop3=\"null\"";
        const expected = [{
            task: "command",
            command: "command",
            action: null,
            properties: expect.objectContaining({ prop1: "123", prop2: "value", prop3: "null" }),
            ...getStartEnd(input, input),
        }];
        const commands = extractCommands(input, commandToTask);
        expect(commands).toEqual(expected);
        expect(Object.keys(commands[0].properties ?? {}).length).toBe(3);
    });
    test("handles command with properties - test 3", () => {
        const input = "/command prop1=0.3 prop2='val\"ue' prop3=\"asdf\nfdsa\"";
        const expected = [{
            task: "command",
            command: "command",
            action: null,
            properties: expect.objectContaining({ prop1: 0.3, prop2: "val\"ue", prop3: "asdf\nfdsa" }),
            ...getStartEnd(input, input),
        }];
        const commands = extractCommands(input, commandToTask);
        expect(commands).toEqual(expected);
        expect(Object.keys(commands[0].properties ?? {}).length).toBe(3);
    });
    test("handles command with properties - test 4", () => {
        const input = "/command prop1=0.3\" prop2='value' prop3=null";
        const expected = [{
            task: "command",
            command: "command",
            action: null,
            // Invalid prop1 because of erroneous quote, so nothing gets extracted
            properties: {},
            ...getStartEnd(input, "/command"),
        }];
        expect(extractCommands(input, commandToTask)).toEqual(expected);
    });
    test("handles command with properties - test 5", () => {
        const input = "/command prop1=-2.3 prop2=.2 prop3=\"one\\\"\"";
        const expected = [{
            task: "command",
            command: "command",
            action: null,
            properties: expect.objectContaining({ prop1: -2.3, prop2: .2, prop3: "one\\\"" }),
            ...getStartEnd(input, input),
        }];
        const commands = extractCommands(input, commandToTask);
        expect(commands).toEqual(expected);
        expect(Object.keys(commands[0].properties ?? {}).length).toBe(3);
    });
    test("handles command with properties - test 6", () => {
        const input = "/command prop1=123 prop2='value' prop3=null\"";
        const expected = [{
            task: "command",
            command: "command",
            action: null,
            // Invalid prop3 because of erroneous quote
            properties: expect.objectContaining({ prop1: 123, prop2: "value" }),
            ...getStartEnd(input, "/command prop1=123 prop2='value'"),
        }];
        const commands = extractCommands(input, commandToTask);
        expect(commands).toEqual(expected);
        expect(Object.keys(commands[0].properties ?? {}).length).toBe(2);
    });
    test("handles command with properties - test 7", () => {
        const input = "/command\tprop1=123 prop2='value' prop3=null\n";
        const expected = [{
            task: "command",
            command: "command",
            action: null,
            properties: expect.objectContaining({ prop1: 123, prop2: "value", prop3: null }),
            ...getStartEnd(input, "/command\tprop1=123 prop2='value' prop3=null"),
        }];
        const commands = extractCommands(input, commandToTask);
        expect(commands).toEqual(expected);
        expect(Object.keys(commands[0].properties ?? {}).length).toBe(3);
    });
    test("handles command with properties - test 8", () => {
        const input = "/command prop1=123 prop2='value' prop3=null ";
        const expected = [{
            task: "command",
            command: "command",
            action: null,
            properties: expect.objectContaining({ prop1: 123, prop2: "value", prop3: null }),
            ...getStartEnd(input, "/command prop1=123 prop2='value' prop3=null"),
        }];
        const commands = extractCommands(input, commandToTask);
        expect(commands).toEqual(expected);
        expect(Object.keys(commands[0].properties ?? {}).length).toBe(3);
    });
    test("handles command with properties - test 9", () => {
        const input = "/command prop1=123 prop2='val\\'u\"e' prop3=null\t";
        const expected = [{
            task: "command",
            command: "command",
            action: null,
            properties: expect.objectContaining({ prop1: 123, prop2: "val\\'u\"e", prop3: null }),
            ...getStartEnd(input, "/command prop1=123 prop2='val\\'u\"e' prop3=null"),
        }];
        const commands = extractCommands(input, commandToTask);
        expect(commands).toEqual(expected);
        expect(Object.keys(commands[0].properties ?? {}).length).toBe(3);
    });
    test("handles command with properties - test 10", () => {
        const input = "/command prop1=123 prop2='value' prop3=null notaprop";
        const expected = [{
            task: "command",
            command: "command",
            action: null,
            properties: expect.objectContaining({ prop1: 123, prop2: "value", prop3: null }),
            ...getStartEnd(input, "/command prop1=123 prop2='value' prop3=null"),
        }];
        const commands = extractCommands(input, commandToTask);
        expect(commands).toEqual(expected);
        expect(Object.keys(commands[0].properties ?? {}).length).toBe(3);
    });

    test("handles newline properly", () => {
        const input = "/command1\n/command2";
        const expected = [
            {
                task: "command1",
                command: "command1",
                action: null,
                properties: {},
                ...getStartEnd(input, "/command1"),
            },
            {
                task: "command2",
                command: "command2",
                action: null,
                properties: {},
                ...getStartEnd(input, "/command2"),
            },
        ];
        expect(extractCommands(input, commandToTask)).toEqual(expected);
    });
    test("handles space properly", () => {
        const input = "/command1 /command2";
        const expected = [
            {
                task: "command1",
                command: "command1",
                action: null,
                properties: {},
                ...getStartEnd(input, "/command1"),
            },
            {
                task: "command2",
                command: "command2",
                action: null,
                properties: {},
                ...getStartEnd(input, "/command2"),
            },
        ];
        expect(extractCommands(input, commandToTask)).toEqual(expected);
    });
    test("handles tab properly", () => {
        const input = "/command1\t/command2";
        const expected = [
            {
                task: "command1",
                command: "command1",
                action: null,
                properties: {},
                ...getStartEnd(input, "/command1"),
            },
            {
                task: "command2",
                command: "command2",
                action: null,
                properties: {},
                ...getStartEnd(input, "/command2"),
            },
        ];
        expect(extractCommands(input, commandToTask)).toEqual(expected);
    });

    test("handles complex scenario with multiple commands and properties - test 1", () => {
        const input = "/cmd1  prop1=123 /cmd2 action2 \tprop2='text' prop3=4.56\n/cmd3 prop4=null \nprop5='invalid because newline";
        const expected = [
            {
                task: "cmd1",
                command: "cmd1",
                action: null,
                properties: expect.objectContaining({ prop1: 123 }),
                ...getStartEnd(input, "/cmd1  prop1=123"),
            },
            {
                task: "cmd2 action2",
                command: "cmd2",
                action: "action2",
                properties: expect.objectContaining({ prop2: "text", prop3: 4.56 }),
                ...getStartEnd(input, "/cmd2 action2 \tprop2='text' prop3=4.56"),
            },
            {
                task: "cmd3",
                command: "cmd3",
                action: null,
                properties: expect.objectContaining({ prop4: null }),
                ...getStartEnd(input, "/cmd3 prop4=null"),
            },
        ];
        const commands = extractCommands(input, commandToTask);
        expect(commands).toEqual(expected);
        expect(Object.keys(commands[0].properties ?? {}).length).toBe(1);
        expect(Object.keys(commands[1].properties ?? {}).length).toBe(2);
        expect(Object.keys(commands[2].properties ?? {}).length).toBe(1);
    });

    test("does nothing for non-command text - test 1", () => {
        const input = "this is a test";
        const expected = [];
        expect(extractCommands(input, commandToTask)).toEqual(expected);
    });
    test("does nothing for non-command text - test 2", () => {
        const input = "/a".repeat(1000);
        const expected = [];
        expect(extractCommands(input, commandToTask)).toEqual(expected);
    });
    test("does nothing for non-command text - test 3", () => {
        const input = "";
        const expected = [];
        expect(extractCommands(input, commandToTask)).toEqual(expected);
    });
    test("does nothing for non-command text - test 4", () => {
        const input = " ";
        const expected = [];
        expect(extractCommands(input, commandToTask)).toEqual(expected);
    });
    test("does nothing for non-command text - test 5", () => {
        const input = "\n";
        const expected = [];
        expect(extractCommands(input, commandToTask)).toEqual(expected);
    });
    test("does nothing for non-command text - test 6", () => {
        const input = "\t";
        const expected = [];
        expect(extractCommands(input, commandToTask)).toEqual(expected);
    });
    test("does nothing for non-command text - test 7", () => {
        const input = "💃";
        const expected = [];
        expect(extractCommands(input, commandToTask)).toEqual(expected);
    });
    test("does nothing for non-command text - test 8", () => {
        const input = "你";
        const expected = [];
        expect(extractCommands(input, commandToTask)).toEqual(expected);
    });
    test("does nothing for non-command text - test 9", () => {
        const input = "!";
        const expected = [];
        expect(extractCommands(input, commandToTask)).toEqual(expected);
    });
    test("does nothing for non-command text - test 10", () => {
        const input = ".";
        const expected = [];
        expect(extractCommands(input, commandToTask)).toEqual(expected);
    });
    test("does nothing for non-command text - test 11", () => {
        const input = "123";
        const expected = [];
        expect(extractCommands(input, commandToTask)).toEqual(expected);
    });
    test("does nothing for non-command text - test 12", () => {
        const input = "-123";
        const expected = [];
        expect(extractCommands(input, commandToTask)).toEqual(expected);
    });
    test("does nothing for non-command text - test 13", () => {
        const input = "0.123";
        const expected = [];
        expect(extractCommands(input, commandToTask)).toEqual(expected);
    });
    test("does nothing for non-command text - test 14", () => {
        const input = "null";
        const expected = [];
        expect(extractCommands(input, commandToTask)).toEqual(expected);
    });
    test("does nothing for non-command text - test 15", () => {
        const input = "true";
        const expected = [];
        expect(extractCommands(input, commandToTask)).toEqual(expected);
    });
    test("does nothing for non-command text - test 16", () => {
        const input = "false";
        const expected = [];
        expect(extractCommands(input, commandToTask)).toEqual(expected);
    });
    test("does nothing for non-command text - test 17", () => {
        const input = "123你";
        const expected = [];
        expect(extractCommands(input, commandToTask)).toEqual(expected);
    });
    test("does nothing for non-command text - test 18", () => {
        const input = "To whom it may concern,\n\nI am writing to inform you that I am a giant purple dinosaur. I cannot be stopped. I am inevitable. I am Barney.\n\nSincerely,\nYour Worst Nightmare";
        const expected = [];
        expect(extractCommands(input, commandToTask)).toEqual(expected);
    });

    // // NOTE: Uncomment these when you want to test performance
    // /** Generates input for performance tests */
    // const generateInput = (numCommands: number) => {
    //     let input = '';
    //     for (let i = 0; i < numCommands; i++) {
    //         input += `/command${i} action${i} prop1=${i} prop2='value${i}' fjdklsafdksajf;ldks\n\n fdskafjsda;lf `;
    //     }
    //     return input;
    // };
    // test('performance test - 10 commands', () => {
    //     const input = generateInput(10);
    //     console.time('extractCommands with 10 commands');
    //     extractCommands(input, commandToTask);
    //     console.timeEnd('extractCommands with 10 commands');
    // });
    // test('performance test - 100 commands', () => {
    //     const input = generateInput(100);
    //     console.time('extractCommands with 100 commands');
    //     extractCommands(input, commandToTask);
    //     console.timeEnd('extractCommands with 100 commands');
    // });
    // test('performance test - 1000 commands', () => {
    //     const input = generateInput(1000);
    //     console.time('extractCommands with 1000 commands');
    //     extractCommands(input, commandToTask);
    //     console.timeEnd('extractCommands with 1000 commands');
    // });
});

describe("filterInvalidCommands", () => {
    test("filters out commands where the task is invalid", async () => {
        const potentialCommands: MaybeLlmCommand[] = [{
            task: "asjflkdjslafkjslaf" as LlmTask,
            command: "add",
            action: null,
            properties: null,
            start: 0,
            end: 10,
        }];

        const result = await filterInvalidCommands(potentialCommands, "RoutineAdd");
        expect(result).toEqual([]);
    });

    test("filters out commands not listed in taskConfig", async () => {
        const potentialCommands: MaybeLlmCommand[] = [{
            task: "Start",
            command: "poutine",
            action: "add",
            properties: null,
            start: 0,
            end: 10,
        }];

        const result = await filterInvalidCommands(potentialCommands, "Start");
        expect(result).toEqual([]);
    });

    test("filters out commands with invalid actions", async () => {
        const potentialCommands: MaybeLlmCommand[] = [{
            task: "Start",
            command: "routine",
            action: "fkdjsalfsda",
            properties: null,
            start: 0,
            end: 10,
        }];

        const result = await filterInvalidCommands(potentialCommands, "Start");
        expect(result).toEqual([]);
    });

    test("filters out invalid properties from commands", async () => {
        const potentialCommands: MaybeLlmCommand[] = [{
            task: "RoutineAdd",
            command: "add",
            action: null,
            properties: { "name": "value", "jeff": "bunny" },
            start: 0,
            end: 10,
        }];

        const result = await filterInvalidCommands(potentialCommands, "RoutineAdd");
        expect(result[0].command).toBe("add");
        expect(result[0].action).toBe(null);
        expect(result[0].properties).toEqual({ "name": "value" });
    });

    test("identifies missing required properties", async () => {
        const potentialCommands: MaybeLlmCommand[] = [{
            task: "RoutineAdd",
            command: "add",
            action: null,
            properties: {},
            start: 0,
            end: 10,
        }];

        const result = await filterInvalidCommands(potentialCommands, "RoutineAdd");
        expect(result).toEqual([]); // Missing required property
    });

    test("accepts valid commands with correct properties", async () => {
        const potentialCommands: MaybeLlmCommand[] = [{
            task: "RoutineAdd",
            command: "add",
            action: null,
            properties: { "name": "value" },
            start: 0,
            end: 10,
        }];

        const result = await filterInvalidCommands(potentialCommands, "RoutineAdd");
        expect(result).toEqual(potentialCommands);
    });

    test("corrects null task", async () => {
        const potentialCommands: MaybeLlmCommand[] = [{
            task: null,
            command: "add",
            action: null,
            properties: { "name": "value" },
            start: 0,
            end: 10,
        }];

        const result = await filterInvalidCommands(potentialCommands, "RoutineAdd");
        expect(result).toEqual([{
            ...potentialCommands[0],
            task: "RoutineAdd",
            command: "add",
            action: null,
            properties: { "name": "value" },
        }]);
    });

    test("corrects itself when command was passed in as an action", async () => {
        const potentialCommands: MaybeLlmCommand[] = [{
            task: "RoutineAdd",
            command: "boop", // Not a valid command
            action: "add", // A valid command
            properties: { "name": "value" },
            start: 0,
            end: 10,
        }];

        const result = await filterInvalidCommands(potentialCommands, "RoutineAdd");
        expect(result).toEqual([{
            ...potentialCommands[0],
            command: "add",
            action: null,
        }]);
    });
});

describe("removeCommands", () => {
    test("removes a single command from the string", () => {
        const input = "/command1 action1 prop1=value1";
        const commands: LlmCommand[] = [{
            ...getStartEnd(input, "/command1 action1 prop1=value1"),
            task: "Start",
            command: "command1",
            action: "action1",
            properties: { prop1: "value1" },
        }];
        expect(removeCommands(input, commands)).toBe("");
    });

    test("removes multiple commands from the string", () => {
        const input = "Text before /command1 action1 prop1=value1 and text after /command2";
        const commands: LlmCommand[] = [
            {
                ...getStartEnd(input, "/command1 action1 prop1=value1"),
                task: "Start",
                command: "command1",
                action: "action1",
                properties: { prop1: "value1" },
            },
            {
                ...getStartEnd(input, "/command2"),
                task: "Start",
                command: "command2",
                action: null,
                properties: null,
            },
        ];
        expect(removeCommands(input, commands)).toBe("Text before  and text after ");
    });

    test("handles commands at the start and end of the string", () => {
        const input = "/command1 at start /command2 at end";
        const commands: LlmCommand[] = [
            {
                ...getStartEnd(input, "/command1 at start"),
                task: "Start",
                command: "command1",
                action: null,
                properties: null,
            },
            {
                ...getStartEnd(input, "/command2 at end"),
                task: "Start",
                command: "command2",
                action: null,
                properties: null,
            },
        ];
        expect(removeCommands(input, commands)).toBe(" ");
    });

    test("handles overlapping commands", () => {
        const input = "/command1 /nestedCommand2 /nestedCommand3";
        const commands: LlmCommand[] = [
            {
                ...getStartEnd(input, "/command1 /nestedCommand2"),
                task: "Start",
                command: "command1",
                action: null,
                properties: null,
            },
            {
                ...getStartEnd(input, "/nestedCommand2 /nestedCommand3"),
                task: "Start",
                command: "nestedCommand2",
                action: null,
                properties: null,
            },
            {
                ...getStartEnd(input, "/nestedCommand3"),
                task: "Start",
                command: "nestedCommand3",
                action: null,
                properties: null,
            },
        ];
        expect(removeCommands(input, commands)).toBe("");
    });

    test("leaves string untouched if no commands match", () => {
        const input = "This string has no commands";
        const commands: LlmCommand[] = [];
        expect(removeCommands(input, commands)).toBe(input);
    });

    // Add more tests for other edge cases as needed
});
