/* eslint-disable @typescript-eslint/ban-ts-comment */
import { LlmTask } from "@local/shared";
import { LlmCommand, MaybeLlmCommand, detectWrappedCommands, extractCommands, filterInvalidCommands, findCharWithLimit, handleCommandTransition, isAlphaNum, isNewline, isWhitespace, removeCommands } from "./commands";
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
    let rest;
    beforeEach(() => {
        onCommit = jest.fn();
        onComplete = jest.fn();
        onCancel = jest.fn();
        onStart = jest.fn();
        rest = { onCommit, onComplete, onCancel, onStart, hasOpenBrackets: false };
    });

    // Outside tests
    test("Reset buffer on outside when whitespace encountered - space", () => {
        const buffer = "asdf";
        const result = handleCommandTransition({
            curr: " ",
            prev: buffer[buffer.length - 1],
            section: "outside",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "outside", buffer: [] });
    });
    test("Reset buffer on outside when whitespace encountered - tab", () => {
        const buffer = "asdf";
        const result = handleCommandTransition({
            curr: "\t",
            prev: buffer[buffer.length - 1],
            section: "outside",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "outside", buffer: [] });
    });
    test("Reset buffer on outside when whitespace encountered - newline", () => {
        const buffer = "asdf";
        const result = handleCommandTransition({
            curr: "\n",
            prev: buffer[buffer.length - 1],
            section: "outside",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "outside", buffer: [] });
    });
    test("keep adding to outside when not on a slash - single quote", () => {
        const buffer = "asdf";
        const result = handleCommandTransition({
            curr: "'",
            prev: buffer[buffer.length - 1],
            section: "outside",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "outside", buffer: [...buffer.split(""), "'"] });
    });
    test("keep adding to outside when not on a slash - double quote", () => {
        const buffer = "asdf";
        const result = handleCommandTransition({
            curr: "\"",
            prev: buffer[buffer.length - 1],
            section: "outside",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "outside", buffer: [...buffer.split(""), "\""] });
    });
    test("keep adding to outside when not on a slash - equals sign", () => {
        const buffer = "asdf";
        const result = handleCommandTransition({
            curr: "=",
            prev: buffer[buffer.length - 1],
            section: "outside",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "outside", buffer: [...buffer.split(""), "="] });
    });
    test("keep adding to outside when not on a slash - letter", () => {
        const buffer = "asdf";
        const result = handleCommandTransition({
            curr: "a",
            prev: buffer[buffer.length - 1],
            section: "outside",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "outside", buffer: [...buffer.split(""), "a"] });
    });
    test("keep adding to outside when not on a slash - number", () => {
        const buffer = "asdf";
        const result = handleCommandTransition({
            curr: "1",
            prev: buffer[buffer.length - 1],
            section: "outside",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "outside", buffer: [...buffer.split(""), "1"] });
    });
    test("keep adding to outside when not on a slash - other alphabets", () => {
        const buffer = "asdf";
        const result = handleCommandTransition({
            curr: "你",
            prev: buffer[buffer.length - 1],
            section: "outside",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "outside", buffer: [...buffer.split(""), "你"] });
    });
    test("keep adding to outside when not on a slash - emojis", () => {
        const buffer = "asdf";
        const result = handleCommandTransition({
            curr: "👋",
            prev: buffer[buffer.length - 1],
            section: "outside",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "outside", buffer: [...buffer.split(""), "👋"] });
    });
    test("keep adding to outside when not on a slash - symbols", () => {
        const buffer = "asdf";
        const result = handleCommandTransition({
            curr: "!",
            prev: buffer[buffer.length - 1],
            section: "outside",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "outside", buffer: [...buffer.split(""), "!"] });
    });

    // Command tests
    test("does not start a command when the slash is not preceeded by whitespace - letter", () => {
        const buffer = "asdf";
        const result = handleCommandTransition({
            curr: "/",
            prev: buffer[buffer.length - 1],
            section: "outside",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "outside", buffer: [...buffer.split(""), "/"] });
    });
    test("does not start a command when the slash is not preceeded by whitespace - number", () => {
        const buffer = "1234";
        const result = handleCommandTransition({
            curr: "/",
            prev: buffer[buffer.length - 1],
            section: "outside",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "outside", buffer: [...buffer.split(""), "/"] });
    });
    test("does not start a command when the slash is not preceeded by whitespace - symbol", () => {
        const buffer = "!@#$";
        const result = handleCommandTransition({
            curr: "/",
            prev: buffer[buffer.length - 1],
            section: "outside",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({
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
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "outside", buffer: [...buffer.split(""), "/"] });
    });
    test("starts a command when buffer is empty", () => {
        const buffer = "";
        const result = handleCommandTransition({
            curr: "/",
            prev: buffer[buffer.length - 1],
            section: "outside",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "command", buffer: [] });
    });
    test("starts a command after a newline", () => {
        const buffer = "asdf\n";
        const result = handleCommandTransition({
            curr: "/",
            prev: buffer[buffer.length - 1],
            section: "outside",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "command", buffer: [] });
    });
    test("starts a command after whitespace - space", () => {
        const buffer = "asdf ";
        const result = handleCommandTransition({
            curr: "/",
            prev: buffer[buffer.length - 1],
            section: "outside",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "command", buffer: [] });
    });
    test("starts a command after whitespace - tab", () => {
        const buffer = "asdf\t";
        const result = handleCommandTransition({
            curr: "/",
            prev: buffer[buffer.length - 1],
            section: "outside",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "command", buffer: [] });
    });
    test("adds letter to command buffer", () => {
        const buffer = "test";
        const result = handleCommandTransition({
            curr: "a",
            prev: buffer[buffer.length - 1],
            section: "command",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "command", buffer: [...buffer.split(""), "a"] });
    });
    test("adds number to command buffer", () => {
        const buffer = "test";
        const result = handleCommandTransition({
            curr: "1",
            prev: buffer[buffer.length - 1],
            section: "command",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "command", buffer: [...buffer.split(""), "1"] });
    });
    test("commmits on newline", () => {
        const buffer = "test";
        const result = handleCommandTransition({
            curr: "\n",
            prev: buffer[buffer.length - 1],
            section: "command",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).toHaveBeenCalledWith("command", buffer);
        expect(result).toMatchObject({ section: "outside", buffer: [] });
    });
    test("resets to outside on other alphabets", () => {
        const buffer = "test";
        const result = handleCommandTransition({
            curr: "你",
            prev: buffer[buffer.length - 1],
            section: "command",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "outside", buffer: [] });
    });
    test("resets to outside on emojis", () => {
        const buffer = "test";
        const result = handleCommandTransition({
            curr: "👋",
            prev: buffer[buffer.length - 1],
            section: "command",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "outside", buffer: [] });
    });
    test("resets to outside on symbols", () => {
        const buffer = "test";
        const result = handleCommandTransition({
            curr: "!",
            prev: buffer[buffer.length - 1],
            section: "command",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "outside", buffer: [] });
    });

    // Pending (when we're not sure if it's an action or a property yet) tests
    test("starts pending actionwhen we encounter the first space after a command - space", () => {
        const buffer = "test";
        const result = handleCommandTransition({
            curr: " ",
            prev: buffer[buffer.length - 1],
            section: "command",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).toHaveBeenCalledWith("command", buffer);
        expect(result).toMatchObject({ section: "action", buffer: [] });
    });
    test("starts pending action when we encounter the first space after a command - tab", () => {
        const buffer = "test";
        const result = handleCommandTransition({
            curr: "\t",
            prev: buffer[buffer.length - 1],
            section: "command",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).toHaveBeenCalledWith("command", buffer);
        expect(result).toMatchObject({ section: "action", buffer: [] });
    });
    test("does not start pending action for newline", () => {
        const buffer = "test";
        const result = handleCommandTransition({
            curr: "\n",
            prev: buffer[buffer.length - 1],
            section: "command",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).toHaveBeenCalledWith("command", buffer);
        expect(result).toMatchObject({ section: "outside", buffer: [] });
    });
    test("does not start pending action for other alphabets", () => {
        const buffer = "test";
        const result = handleCommandTransition({
            curr: "你",
            prev: buffer[buffer.length - 1],
            section: "command",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "outside", buffer: [] });
    });
    test("does not start pending action for emojis", () => {
        const buffer = "test";
        const result = handleCommandTransition({
            curr: "👋",
            prev: buffer[buffer.length - 1],
            section: "command",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "outside", buffer: [] });
    });
    test("does not start pending action for symbols", () => {
        const buffer = "test";
        const result = handleCommandTransition({
            curr: "!",
            prev: buffer[buffer.length - 1],
            section: "command",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "outside", buffer: [] });
    });
    test("cancels pending action on other alphabets", () => {
        const buffer = "test";
        const result = handleCommandTransition({
            curr: "你",
            prev: buffer[buffer.length - 1],
            section: "action",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "outside", buffer: [] });
    });
    test("cancels pending action on emojis", () => {
        const buffer = "test";
        const result = handleCommandTransition({
            curr: "👋",
            prev: buffer[buffer.length - 1],
            section: "action",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "outside", buffer: [] });
    });
    test("cancels pending action on symbols", () => {
        const buffer = "test";
        const result = handleCommandTransition({
            curr: "!",
            prev: buffer[buffer.length - 1],
            section: "action",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "outside", buffer: [] });
    });

    // Action tests
    test("commits pending action buffer to action on whitespace - space", () => {
        const buffer = "add";
        const result = handleCommandTransition({
            curr: " ",
            prev: buffer[buffer.length - 1],
            section: "action",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).toHaveBeenCalledWith("action", buffer);
        expect(result).toMatchObject({ section: "propName", buffer: [] });
    });
    test("commits pending action buffer to action on whitespace - tab", () => {
        const buffer = "add";
        const result = handleCommandTransition({
            curr: "\t",
            prev: buffer[buffer.length - 1],
            section: "action",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).toHaveBeenCalledWith("action", buffer);
        expect(result).toMatchObject({ section: "propName", buffer: [] });
    });
    test("commits pending action buffer to action on newline", () => {
        const buffer = "add";
        const result = handleCommandTransition({
            curr: "\n",
            prev: buffer[buffer.length - 1],
            section: "action",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).toHaveBeenCalledWith("action", buffer);
        expect(result).toMatchObject({ section: "outside", buffer: [] });
    });

    // Property name tests
    test("commits pending action buffer to property name on equals sign", () => {
        const buffer = "name";
        const result = handleCommandTransition({
            curr: "=",
            prev: buffer[buffer.length - 1],
            section: "action",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).toHaveBeenCalledWith("propName", buffer);
        expect(result).toMatchObject({ section: "propValue", buffer: [] });
    });
    test("commits property name buffer to property name on equals sign", () => {
        const buffer = "name";
        const result = handleCommandTransition({
            curr: "=",
            prev: buffer[buffer.length - 1],
            section: "propName",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).toHaveBeenCalledWith("propName", buffer);
        expect(result).toMatchObject({ section: "propValue", buffer: [] });
    });
    test("cancels property name buffer on whitespace - space", () => {
        const buffer = "name";
        const result = handleCommandTransition({
            curr: " ",
            prev: buffer[buffer.length - 1],
            section: "propName",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "outside", buffer: [] });
    });
    test("cancels property name buffer on whitespace - tab", () => {
        const buffer = "name";
        const result = handleCommandTransition({
            curr: "\t",
            prev: buffer[buffer.length - 1],
            section: "propName",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "outside", buffer: [] });
    });
    test("cancels property name buffer on newline", () => {
        const buffer = "name";
        const result = handleCommandTransition({
            curr: "\n",
            prev: buffer[buffer.length - 1],
            section: "propName",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "outside", buffer: [] });
    });
    test("cancels property name buffer on other alphabets", () => {
        const buffer = "name";
        const result = handleCommandTransition({
            curr: "你",
            prev: buffer[buffer.length - 1],
            section: "propName",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "outside", buffer: [] });
    });
    test("cancels property name buffer on emojis", () => {
        const buffer = "name";
        const result = handleCommandTransition({
            curr: "👋",
            prev: buffer[buffer.length - 1],
            section: "propName",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "outside", buffer: [] });
    });
    test("cancels property name buffer on symbols", () => {
        const buffer = "name";
        const result = handleCommandTransition({
            curr: "!",
            prev: buffer[buffer.length - 1],
            section: "propName",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "outside", buffer: [] });
    });
    test("cancels property name buffer on slash", () => {
        const buffer = "name";
        const result = handleCommandTransition({
            curr: "/",
            prev: buffer[buffer.length - 1],
            section: "propName",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "outside", buffer: [] });
    });

    // Property value tests
    test("starts property value on single quote", () => {
        const buffer = "";
        const result = handleCommandTransition({
            curr: "'",
            prev: buffer[buffer.length - 1],
            section: "propValue", // Should already be marked as propValue because of the equals sign
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        // Includes the quote in the buffer
        expect(result).toMatchObject({ section: "propValue", buffer: ["'"] });
    });
    test("starts property value on double quote", () => {
        const buffer = "";
        const result = handleCommandTransition({
            curr: "\"",
            prev: buffer[buffer.length - 1],
            section: "propValue", // Should already be marked as propValue because of the equals sign
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        // Includes the quote in the buffer
        expect(result).toMatchObject({ section: "propValue", buffer: [...buffer.split(""), "\""] });
    });
    test("continues property value if buffer + curr might be a number - test 1", () => {
        const buffer = "";
        const result = handleCommandTransition({
            curr: "1",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "propValue", buffer: [...buffer.split(""), "1"] });
    });
    test("continues property value if buffer + curr might be a number - test 2", () => {
        const buffer = "-";
        const result = handleCommandTransition({
            curr: "1",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "propValue", buffer: [...buffer.split(""), "1"] });
    });
    test("continues property value if buffer + curr might be a number - test 3", () => {
        const buffer = "";
        const result = handleCommandTransition({
            curr: ".",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "propValue", buffer: [...buffer.split(""), "."] });
    });
    test("continues property value if buffer + curr might be a number - test 4", () => {
        const buffer = "3";
        const result = handleCommandTransition({
            curr: ".",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "propValue", buffer: [...buffer.split(""), "."] });
    });
    test("cancels property value if buffer + curr is an invalid number - test 1", () => {
        const buffer = "-";
        const result = handleCommandTransition({
            curr: "-",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "outside", buffer: [] });
    });
    test("cancels property value if buffer + curr is an invalid number - test 2", () => {
        const buffer = "1";
        const result = handleCommandTransition({
            curr: "-",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "outside", buffer: [] });
    });
    test("cancels property value if buffer + curr is an invalid number - test 3", () => {
        const buffer = "3.";
        const result = handleCommandTransition({
            curr: ".",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "outside", buffer: [] });
    });
    test("cancels property value if buffer + curr is an invalid number - test 4", () => {
        const buffer = "1.2";
        const result = handleCommandTransition({
            curr: ".",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "outside", buffer: [] });
    });
    test("continues property value if buffer + curr might be null - test 1", () => {
        const buffer = "";
        const result = handleCommandTransition({
            curr: "n",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "propValue", buffer: ["n"] });
    });
    test("continues property value if buffer + curr might be null - test 2", () => {
        const buffer = "nul";
        const result = handleCommandTransition({
            curr: "l",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "propValue", buffer: [...buffer.split(""), "l"] });
    });
    test("cancels property value if buffer + curr is not null", () => {
        const buffer = "null";
        const result = handleCommandTransition({
            curr: "l",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "outside", buffer: [] });
    });
    test("cancels property value if letter before quote", () => {
        const buffer = "";
        const result = handleCommandTransition({
            curr: "a",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "outside", buffer: [] });
    });
    test("cancels property value if whitespace before quote - space", () => {
        const buffer = "";
        const result = handleCommandTransition({
            curr: " ",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "outside", buffer: [] });
    });
    test("cancels property value if whitespace before quote - tab", () => {
        const buffer = "";
        const result = handleCommandTransition({
            curr: "\t",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "outside", buffer: [] });
    });
    test("cancels property value if newline before quote", () => {
        const buffer = "";
        const result = handleCommandTransition({
            curr: "\n",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "outside", buffer: [] });
    });
    test("cancels property value if other alphabets before quote", () => {
        const buffer = "";
        const result = handleCommandTransition({
            curr: "你",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "outside", buffer: [] });
    });
    test("cancels property value if emojis before quote", () => {
        const buffer = "";
        const result = handleCommandTransition({
            curr: "👋",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "outside", buffer: [] });
    });
    test("cancels property value if symbols before quote", () => {
        const buffer = "";
        const result = handleCommandTransition({
            curr: "!",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "outside", buffer: [] });
    });
    test("cancels property value if slash before quote", () => {
        const buffer = "";
        const result = handleCommandTransition({
            curr: "/",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "outside", buffer: [] });
    });
    test("continues property value if already in quotes - letter with single quote start", () => {
        const buffer = "'";
        const result = handleCommandTransition({
            curr: "a",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "propValue", buffer: [...buffer.split(""), "a"] });
    });
    test("continues property value if already in quotes - letter with double quote start", () => {
        const buffer = "\"";
        const result = handleCommandTransition({
            curr: "a",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "propValue", buffer: [...buffer.split(""), "a"] });
    });
    test("continues property value if already in quotes - letter with single quote start and other text in buffer", () => {
        const buffer = "'test";
        const result = handleCommandTransition({
            curr: "a",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "propValue", buffer: [...buffer.split(""), "a"] });
    });
    test("continues property value if already in quotes - other language", () => {
        const buffer = "'test";
        const result = handleCommandTransition({
            curr: "你",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "propValue", buffer: [...buffer.split(""), "你"] });
    });
    test("continues property value if already in quotes - emoji", () => {
        const buffer = "'test";
        const result = handleCommandTransition({
            curr: "👋",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "propValue", buffer: [...buffer.split(""), "👋"] });
    });
    test("continues property value if already in quotes - symbol", () => {
        const buffer = "'test";
        const result = handleCommandTransition({
            curr: "!",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "propValue", buffer: [...buffer.split(""), "!"] });
    });
    test("continues property value if already in quotes - slash", () => {
        const buffer = "'test";
        const result = handleCommandTransition({
            curr: "/",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "propValue", buffer: [...buffer.split(""), "/"] });
    });
    test("continues property value if already in quotes - newline", () => {
        const buffer = "'test";
        const result = handleCommandTransition({
            curr: "\n",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "propValue", buffer: [...buffer.split(""), "\n"] });
    });
    test("continues property value if already in quotes - space", () => {
        const buffer = "\"test";
        const result = handleCommandTransition({
            curr: " ",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "propValue", buffer: [...buffer.split(""), " "] });
    });
    test("continues property value if already in quotes - tab", () => {
        const buffer = "\"test";
        const result = handleCommandTransition({
            curr: "\t",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "propValue", buffer: [...buffer.split(""), "\t"] });
    });
    test("continues property value when curr is a different quote type than the starting quote - double with single start", () => {
        const buffer = "'";
        const result = handleCommandTransition({
            curr: "\"",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "propValue", buffer: ["'", "\""] });
    });
    test("continues property value when curr is a different quote type than the starting quote - single with double start", () => {
        const buffer = "\"";
        const result = handleCommandTransition({
            curr: "'",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "propValue", buffer: ["\"", "'"] });
    });
    test("continues property value for escaped characters - curr is escape character, buffer is quote", () => {
        const buffer = "'";
        const result = handleCommandTransition({
            curr: "\\",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "propValue", buffer: ["'", "\\"] });
    });
    test("continues property value for escaped characters - curr is single quote, buffer is double quote and escape character", () => {
        const buffer = "\"\\";
        const result = handleCommandTransition({
            curr: "\"",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "propValue", buffer: ["\"", "\\", "\""] });
    });
    test("continues property value for escaped characters - curr is single quote, buffer is single quote and escape character", () => {
        const buffer = "'\\";
        const result = handleCommandTransition({
            curr: "'",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "propValue", buffer: ["'", "\\", "'"] });
    });
    test("completes property value for an even number of escape characters", () => {
        const buffer = "'\\\\";
        const result = handleCommandTransition({
            curr: "'",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).toHaveBeenCalledWith("propValue", "\\\\");
        expect(result).toMatchObject({ section: "propName", buffer: [] });
    });
    test("continues property value for an odd number of escape characters", () => {
        const buffer = "'\\\\\\";
        const result = handleCommandTransition({
            curr: "'",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "propValue", buffer: [...buffer.split(""), "'"] });
    });
    test("commits property value on closing quote - single quote", () => {
        const buffer = "'test";
        const result = handleCommandTransition({
            curr: "'",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).toHaveBeenCalledWith("propValue", "test");
        expect(result).toMatchObject({ section: "propName", buffer: [] });
    });
    test("commits property value on closing quote - double quote", () => {
        const buffer = "\"test";
        const result = handleCommandTransition({
            curr: "\"",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).toHaveBeenCalledWith("propValue", "test");
        expect(result).toMatchObject({ section: "propName", buffer: [] });
    });
    test("commits number property value on space - test 1", () => {
        const buffer = "123";
        const result = handleCommandTransition({
            curr: " ",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).toHaveBeenCalledWith("propValue", 123);
        expect(result).toMatchObject({ section: "propName", buffer: [] });
    });
    test("commits number property value on space - test 2", () => {
        const buffer = "-123";
        const result = handleCommandTransition({
            curr: " ",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).toHaveBeenCalledWith("propValue", -123);
        expect(result).toMatchObject({ section: "propName", buffer: [] });
    });
    test("commits number property value on space - test 3", () => {
        const buffer = "-1.23";
        const result = handleCommandTransition({
            curr: " ",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).toHaveBeenCalledWith("propValue", -1.23);
        expect(result).toMatchObject({ section: "propName", buffer: [] });
    });
    test("commits number property value on tab", () => {
        const buffer = "-1.23";
        const result = handleCommandTransition({
            curr: "\t",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).toHaveBeenCalledWith("propValue", -1.23);
        expect(result).toMatchObject({ section: "propName", buffer: [] });
    });
    test("commits number property value on newline", () => {
        const buffer = "-1.23";
        const result = handleCommandTransition({
            curr: "\n",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).toHaveBeenCalledWith("propValue", -1.23);
        expect(result).toMatchObject({ section: "outside", buffer: [] });
    });
    test("cancels number property value on letter", () => {
        const buffer = "-1.23";
        const result = handleCommandTransition({
            curr: "a",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "outside", buffer: [] });
    });
    test("cancels number property value on other alphabet", () => {
        const buffer = "-1.23";
        const result = handleCommandTransition({
            curr: "你",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "outside", buffer: [] });
    });
    test("cancels number property value on emoji", () => {
        const buffer = "-1.23";
        const result = handleCommandTransition({
            curr: "👋",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "outside", buffer: [] });
    });
    test("cancels number property value on symbol", () => {
        const buffer = "-1.23";
        const result = handleCommandTransition({
            curr: "!",
            prev: buffer[buffer.length - 1],
            section: "propValue",
            buffer: buffer.split(""),
            ...rest,
        });
        expect(onCommit).not.toHaveBeenCalled();
        expect(result).toMatchObject({ section: "outside", buffer: [] });
    });
});

const getStartEnd = (str: string, sub: string): { start: number, end: number } => ({
    start: str.indexOf(sub),
    end: str.indexOf(sub) + sub.length, // Index after the last character
});

describe("findCharWithLimit", () => {
    const testString1 = "[/command1]";
    const testString2 = "  [/command1]  ";
    const testString3 = "  [  /command1  ]  ";
    const allTests = [testString1, testString2, testString3];

    for (const testString of allTests) {
        const openIndex = testString.indexOf("[");
        const closeIndex = testString.indexOf("]");
        test("finds \"[\" starting at beginning", () => {
            // @ts-ignore: expect-message
            expect(findCharWithLimit(0, true, "[", testString, 5), `input: ${testString}`).toBe(openIndex);
        });
        test("doesn't find \"[\" starting at end - runs into non-whitespace", () => {
            // @ts-ignore: expect-message
            expect(findCharWithLimit(testString.length - 1, true, "[", testString, 5), `input: ${testString}`).toBeNull();
        });
        test("finds \"]\" starting at end", () => {
            // @ts-ignore: expect-message
            expect(findCharWithLimit(testString.length - 1, false, "]", testString, 5), `input: ${testString}`).toBe(closeIndex);
        });
        test("doesn't find \"]\" starting at beginning - runs into non-whitespace", () => {
            // @ts-ignore: expect-message
            expect(findCharWithLimit(0, false, "]", testString, 5), `input: ${testString}`).toBeNull();
        });
        test("doesn't find \"[\" starting at middle - runs into non-whitespace", () => {
            // @ts-ignore: expect-message
            expect(findCharWithLimit(3, true, "[", testString, 5), `input: ${testString}`).toBeNull();
        });
    }

    test("doesn't find \"]\" when a newline is in the way", () => {
        const testString = " \n ]";
        expect(findCharWithLimit(0, false, "]", testString, 5)).toBeNull();
    });
});

// Mock commandToTask
const commandToTask: CommandToTask = (command, action) => {
    if (action) return `${command} ${action}` as LlmTask;
    return command as LlmTask;
};

/**
 * Helper function to simplify testing of `detectWrappedCommands`
 */
const detectWrappedCommandsTester = ({
    start,
    delimiter,
    input,
    expected,
}: {
    start: string,
    delimiter: string | null,
    input: string,
    expected: number[],
}) => {
    const allCommands = extractCommands(input, commandToTask);
    console.log("got these commands in detectCommandsTester", allCommands);
    const detectedCommands = detectWrappedCommands({
        start,
        delimiter,
        commands: allCommands,
        messageString: input,
    });
    // @ts-ignore: expect-message
    expect(detectedCommands, `input: ${input}`).toEqual(expected);
};

describe("detectWrappedCommands", () => {
    const wrapper1 = {
        start: "suggested",
        delimiter: ",", // Allows multiple
    };
    const wrapper2 = {
        start: "recommend",
        delimiter: null, // Does not allow multiple
    };
    const allWrappers = [wrapper1, wrapper2];

    // Loop through each wrapper and test the same cases for each
    for (const wrapper of allWrappers) {
        const { start, delimiter } = wrapper;
        test("returns empty array when no commands are present - test 1", () => {
            detectWrappedCommandsTester({
                input: "a/command",
                expected: [],
                ...wrapper,
            });
        });
        test("returns empty array when no commands are present - test 2", () => {
            detectWrappedCommandsTester({
                input: "/command🥴",
                expected: [],
                ...wrapper,
            });
        });
        test("returns empty array when commands are not wrapped - test 1", () => {
            detectWrappedCommandsTester({
                input: "/command1 /command2",
                expected: [],
                ...wrapper,
            });
        });
        test("returns empty array when commands are not wrapped - test 2", () => {
            detectWrappedCommandsTester({
                input: `${start.slice(0, -1)}: [/command1]`,
                expected: [],
                ...wrapper,
            });
        });
        test("returns empty array when commands are not wrapped - test 3", () => {
            detectWrappedCommandsTester({
                input: `${start}: [/command1`,
                expected: [],
                ...wrapper,
            });
        });
        test("returns empty array when commands are not wrapped - test 4", () => {
            detectWrappedCommandsTester({
                input: `${start}:\n[/command1]`,
                expected: [],
                ...wrapper,
            });
        });
        test("returns empty array when commands are not wrapped - test 5", () => {
            console.log("before not wrapped test 5");
            detectWrappedCommandsTester({
                input: `${start}: [/command1\n]`,
                expected: [],
                ...wrapper,
            });
            console.log("after not wrapped test 5");
        });
        test("returns empty array when commands are not wrapped - test 6", () => {
            detectWrappedCommandsTester({
                input: `${start}: [/command1${delimiter}\n/command2]`,
                expected: [],
                ...wrapper,
            });
        });
        test("returns empty array when commands are not wrapped - test 7", () => {
            console.log("before not wrapped test 7");
            detectWrappedCommandsTester({
                input: `${start}:[/command1 ${start}:]`,
                expected: [],
                ...wrapper,
            });
            console.log("after not wrapped test 7");
        });
        test("returns correct indices for a single wrapped command - test 1", () => {
            console.log("before single wrapped command 1");
            detectWrappedCommandsTester({
                input: `${start}:[/command1]`,
                expected: [0],
                ...wrapper,
            });
            console.log("after single wrapped command 1");
        });
        test("returns correct indices for a single wrapped command - test 2", () => {
            console.log("before single wrapped command 2");
            detectWrappedCommandsTester({
                input: `boop ${start}: [/command1]`,
                expected: [0],
                ...wrapper,
            });
            console.log("after single wrapped command 2");
        });
        test("returns correct indices for a single wrapped command - test 3", () => {
            console.log("before single wrapped command 3");
            detectWrappedCommandsTester({
                input: `/firstCommand hello ${start}:[/command1]`,
                expected: [1],
                ...wrapper,
            });
            console.log("after single wrapped command 3");
        });
        test("returns correct indices for a single wrapped command - test 4", () => {
            console.log("before single wrapped command 4");
            detectWrappedCommandsTester({
                input: `/firstCommand hello ${start}: [/command1]}`,
                expected: [1],
                ...wrapper,
            });
            console.log("after single wrapped command 4");
        });
        test("returns correct indices for a single wrapped command - test 5", () => {
            detectWrappedCommandsTester({
                input: `/firstCommand hello ${start}: [  /command1  ]`,
                expected: [1],
                ...wrapper,
            });
        });
        test("returns correct indices for a single wrapped command - test 6", () => {
            detectWrappedCommandsTester({
                input: `/firstCommand hello name="hi" ${start}: [/command1]`,
                expected: [1],
                ...wrapper,
            });
        });
        test("returns correct indices for a single wrapped command - test 7", () => {
            detectWrappedCommandsTester({
                input: `${start}: ${start}: [/command1 hello]`,
                expected: [0],
                ...wrapper,
            });
        });
        test("returns correct indices for multiple wrapped commands - test 1", () => {
            detectWrappedCommandsTester({
                input: `${start}:[/command1${delimiter ?? ""}/command2]`,
                expected: delimiter ? [0, 1] : [],
                ...wrapper,
            });
        });
        test("returns correct indices for multiple wrapped commands - test 2", () => {
            detectWrappedCommandsTester({
                input: `${start}: [/command1 ${delimiter ?? ""}/command2]`,
                expected: delimiter ? [0, 1] : [],
                ...wrapper,
            });
        });
        test("returns correct indices for multiple wrapped commands - test 3", () => {
            detectWrappedCommandsTester({
                input: `${start}: [/command1 ${delimiter ?? ""} /command2 ]`,
                expected: delimiter ? [0, 1] : [],
                ...wrapper,
            });
        });
        test("handles property value trickery - test 1", () => {
            detectWrappedCommandsTester({
                input: `/command1 action name='${start}: [/command2]'`,
                expected: [],
                ...wrapper,
            });
        });
        test("handles property value trickery - test 2", () => {
            detectWrappedCommandsTester({
                input: `/command1 action name='${start}:' [/command2]`,
                expected: [],
                ...wrapper,
            });
        });
    }
});

/**
 * Helper function to simplify testing of `extractCommands`
 */
const extractCommandsTester = ({
    input,
    expected,
}: {
    input: string,
    expected: (Omit<MaybeLlmCommand, "task" | "start" | "end"> & { match: string })[],
}) => {
    const commands = extractCommands(input, commandToTask);
    console.log("got commands", commands);
    const expectedCommands = expected.map(({ command, action, properties, match }) => ({
        task: commandToTask(command, action),
        command,
        action,
        properties: properties ? expect.objectContaining(properties) : undefined,
        ...getStartEnd(input, match),
    }));
    // @ts-ignore: expect-message
    expect(commands, `Should match expected commands. input: ${input}`).toEqual(expectedCommands);
    for (let i = 0; i < expectedCommands.length; i++) {
        const receivedPropertyLength = Object.keys(commands[i].properties ?? {}).length;
        const expectedPropertyLength = Object.keys(expected[i].properties ?? {}).length;
        // @ts-ignore: expect-message
        expect(receivedPropertyLength, `Should have same number of properties. input: ${input}. index: ${i}`).toBe(expectedPropertyLength);
    }
};

describe("extractCommands", () => {
    test("ignores non-command slashes - test 1", () => {
        extractCommandsTester({
            input: "a/command",
            expected: [],
        });
    });
    test("ignores non-command slashes - test 2", () => {
        extractCommandsTester({
            input: "1/3",
            expected: [],
        });
    });
    test("ignores non-command slashes - test 3", () => {
        extractCommandsTester({
            input: "/boop.",
            expected: [],
        });
    });
    test("ignores non-command slashes - test 4", () => {
        extractCommandsTester({
            input: "/boop!",
            expected: [],
        });
    });
    test("ignores non-command slashes - test 5", () => {
        extractCommandsTester({
            input: "/boop你",
            expected: [],
        });
    });
    test("ignores non-command slashes - test 6", () => {
        extractCommandsTester({
            input: "/boop👋",
            expected: [],
        });
    });
    test("ignores non-command slashes - test 7", () => {
        extractCommandsTester({
            input: "/boop/",
            expected: [],
        });
    });
    test("ignores non-command slashes - test 8", () => {
        extractCommandsTester({
            input: "/boop\\",
            expected: [],
        });
    });
    test("ignores non-command slashes - test 9", () => {
        extractCommandsTester({
            input: "/boop=",
            expected: [],
        });
    });
    test("ignores non-command slashes - test 10", () => {
        extractCommandsTester({
            input: "/boop-",
            expected: [],
        });
    });
    test("ignores non-command slashes - test 11", () => {
        extractCommandsTester({
            input: "/boop.",
            expected: [],
        });
    });
    test("ignores non-command slashes - test 12", () => {
        extractCommandsTester({
            input: "//boop",
            expected: [],
        });
    });
    test("ignores non-command slashes - test 13", () => {
        extractCommandsTester({
            input: "//bippity /boppity! /boop💃 /realCommand",
            expected: [{
                command: "realCommand",
                action: null,
                properties: {},
                match: "/realCommand",
            }],
        });
    });
    test("ignores commands in code blocks - test 1", () => {
        console.log("before code block test 1");
        extractCommandsTester({
            input: "```/command```",
            expected: [],
        });
        console.log("after code block test 1");
    });
    test("ignores commands in code blocks - test 2", () => {
        console.log("before code block test 2");
        extractCommandsTester({
            input: "here is some code:\n```/command action```\n",
            expected: [],
        });
        console.log("after code block test 2");
    });
    test("ignores commands in code blocks - test 3", () => {
        console.log("before code block test 3");
        extractCommandsTester({
            input: "here is some code:\n```bloop /command action\nother words```",
            expected: [],
        });
        console.log("after code block test 3");
    });
    test("ignores commands in code blocks - test 4", () => {
        console.log("before code block test 4");
        extractCommandsTester({
            input: "```command inside code block: /codeCommand action```command outside code block: /otherCommand action\n",
            expected: [{
                command: "otherCommand",
                action: "action",
                properties: {},
                match: "/otherCommand action",
            }],
        });
        console.log("after code block test 4");
    });
    test("ignores commands in code blocks - test 5", () => {
        console.log("before code block test 5");
        extractCommandsTester({
            input: "Single-tick code block: `/command action`",
            expected: [],
        });
        console.log("after code block test 5");
    });
    test("ignores commands inside code blocks - test 6", () => {
        console.log("before code block test 6");
        extractCommandsTester({
            input: "<code>/command action</code>",
            expected: [],
        });
        console.log("after code block test 6");
    });
    test("ignores commands inside code blocks - test 7", () => {
        console.log("before code block test 7");
        extractCommandsTester({
            input: "hello <code>\n/command action\n</code> there",
            expected: [],
        });
        console.log("after code block test 7");
    });
    test("doesn't ignore code-looking commands when there's property value trickery - test 1", () => {
        extractCommandsTester({
            input: "/command1 action1 text='```/note find```'",
            expected: [{
                command: "command1",
                action: "action1",
                properties: { text: "```/note find```" },
                match: "/command1 action1 text='```/note find```'",
            }],
        });
    });
    test("doesn't ignore code-looking commands when there's property value trickery - test 2", () => {
        extractCommandsTester({
            input: "/command1 action1 text='```' /note find text='```'",
            expected: [{
                command: "command1",
                action: "action1",
                properties: { text: "```" },
                match: "/command1 action1 text='```'",
            }, {
                command: "note",
                action: "find",
                properties: { text: "```" },
                match: "/note find text='```'",
            }],
        });
    });
    test("doesn't ignore code-looking commands when there's property value trickery - test 3", () => {
        extractCommandsTester({
            input: "/command1 action1 text='<code>/note find</code>'",
            expected: [{
                command: "command1",
                action: "action1",
                properties: { text: "<code>/note find</code>" },
                match: "/command1 action1 text='<code>/note find</code>'",
            }],
        });
    });
    test("doesn't ignore code-looking commands when there's property value trickery - test 4", () => {
        extractCommandsTester({
            input: "/command1 action1 text='<code>' /note find text='<code>'",
            expected: [{
                command: "command1",
                action: "action1",
                properties: { text: "<code>" },
                match: "/command1 action1 text='<code>'",
            }, {
                command: "note",
                action: "find",
                properties: { text: "<code>" },
                match: "/note find text='<code>'",
            }],
        });
    });
    test("doesn't ignore double-tick code blocks", () => {
        console.log("before double-tick code block test 1");
        extractCommandsTester({
            input: "Double-tick code block: `` /command action ``",
            expected: [{
                command: "command",
                action: "action",
                properties: {},
                match: "/command action",
            }],
        });
        console.log("after double-tick code block test 1");
    });
    test("doesn't ignore single-tick code block when newline is encountered", () => {
        extractCommandsTester({
            input: "Invalid single-tick code block: `\n/command action `",
            expected: [{
                command: "command",
                action: "action",
                properties: {},
                match: "/command action",
            }],
        });
    });

    test("extracts simple command without action or properties - test 1", () => {
        extractCommandsTester({
            input: "/command",
            expected: [{
                command: "command",
                action: null,
                properties: {},
                match: "/command",
            }],
        });
    });
    test("extracts simple command without action or properties - test 2", () => {
        extractCommandsTester({
            input: "  /command",
            expected: [{
                command: "command",
                action: null,
                properties: {},
                match: "/command",
            }],
        });
    });
    test("extracts simple command without action or properties - test 3", () => {
        extractCommandsTester({
            input: "/command  ",
            expected: [{
                command: "command",
                action: null,
                properties: {},
                match: "/command",
            }],
        });
    });
    test("extracts simple command without action or properties - test 4", () => {
        extractCommandsTester({
            input: "aasdf\n/command",
            expected: [{
                command: "command",
                action: null,
                properties: {},
                match: "/command",
            }],
        });
    });
    test("extracts simple command without action or properties - test 5", () => {
        extractCommandsTester({
            input: "/command\n",
            expected: [{
                command: "command",
                action: null,
                properties: {},
                match: "/command",
            }],
        });
    });
    test("extracts simple command without action or properties - test 6", () => {
        extractCommandsTester({
            input: "/command\t",
            expected: [{
                command: "command",
                action: null,
                properties: {},
                match: "/command",
            }],
        });
    });
    test("extracts simple command without action or properties - test 7", () => {
        extractCommandsTester({
            input: "/command invalidActionBecauseSymbol!",
            expected: [{
                command: "command",
                action: null,
                properties: {},
                match: "/command",
            }],
        });
    });
    test("extracts simple command without action or properties - test 8", () => {
        extractCommandsTester({
            input: "/command invalidActionBecauseLanguage你",
            expected: [{
                command: "command",
                action: null,
                properties: {},
                match: "/command",
            }],
        });
    });

    test("extracts command with action - test 1", () => {
        extractCommandsTester({
            input: "/command action",
            expected: [{
                command: "command",
                action: "action",
                properties: {},
                match: "/command action",
            }],
        });
    });
    test("extracts command with action - test 2", () => {
        extractCommandsTester({
            input: "/command action other words",
            expected: [{
                command: "command",
                action: "action",
                properties: {},
                match: "/command action",
            }],
        });
    });
    test("extracts command with action - test 3", () => {
        extractCommandsTester({
            input: "/command action invalidProp= 'space after equals'",
            expected: [{
                command: "command",
                action: "action",
                properties: {},
                match: "/command action",
            }],
        });
    });
    test("extracts command with action - test 4", () => {
        extractCommandsTester({
            input: "/command action\t",
            expected: [{
                command: "command",
                action: "action",
                properties: {},
                match: "/command action",
            }],
        });
    });
    test("extracts command with action - test 5", () => {
        extractCommandsTester({
            input: "/command action\n",
            expected: [{
                command: "command",
                action: "action",
                properties: {},
                match: "/command action",
            }],
        });
    });
    test("extracts command with action - test 6", () => {
        extractCommandsTester({
            input: "/command\taction",
            expected: [{
                command: "command",
                action: "action",
                properties: {},
                match: "/command\taction",
            }],
        });
    });

    test("handles command with properties - test 1", () => {
        extractCommandsTester({
            input: "/command prop1=123 prop2='value' prop3=null",
            expected: [{
                command: "command",
                action: null,
                properties: { prop1: 123, prop2: "value", prop3: null },
                match: "/command prop1=123 prop2='value' prop3=null",
            }],
        });
    });
    test("handles command with properties - test 2", () => {
        extractCommandsTester({
            input: "/command prop1=\"123\" prop2='value' prop3=\"null\"",
            expected: [{
                command: "command",
                action: null,
                properties: { prop1: "123", prop2: "value", prop3: "null" },
                match: "/command prop1=\"123\" prop2='value' prop3=\"null\"",
            }],
        });
    });
    test("handles command with properties - test 3", () => {
        extractCommandsTester({
            input: "/command prop1=0.3 prop2='val\"ue' prop3=\"asdf\nfdsa\"",
            expected: [{
                command: "command",
                action: null,
                properties: { prop1: 0.3, prop2: "val\"ue", prop3: "asdf\nfdsa" },
                match: "/command prop1=0.3 prop2='val\"ue' prop3=\"asdf\nfdsa\"",
            }],
        });
    });
    test("handles command with properties - test 4", () => {
        extractCommandsTester({
            input: "/command prop1=0.3\" prop2='value' prop3=null",
            expected: [{
                command: "command",
                action: null,
                properties: {},
                match: "/command",
            }],
        });
    });
    test("handles command with properties - test 5", () => {
        extractCommandsTester({
            input: "/command prop1=-2.3 prop2=.2 prop3=\"one\\\"\"",
            expected: [{
                command: "command",
                action: null,
                properties: { prop1: -2.3, prop2: .2, prop3: "one\\\"" },
                match: "/command prop1=-2.3 prop2=.2 prop3=\"one\\\"\"",
            }],
        });
    });
    test("handles command with properties - test 6", () => {
        extractCommandsTester({
            input: "/command prop1=123 prop2='value' prop3=null\"",
            expected: [{
                command: "command",
                action: null,
                properties: { prop1: 123, prop2: "value" },
                match: "/command prop1=123 prop2='value'",
            }],
        });
    });
    test("handles command with properties - test 7", () => {
        extractCommandsTester({
            input: "/command\tprop1=123 prop2='value' prop3=null\n",
            expected: [{
                command: "command",
                action: null,
                properties: { prop1: 123, prop2: "value", prop3: null },
                match: "/command\tprop1=123 prop2='value' prop3=null",
            }],
        });
    });
    test("handles command with properties - test 8", () => {
        extractCommandsTester({
            input: "/command prop1=123 prop2='value' prop3=null ",
            expected: [{
                command: "command",
                action: null,
                properties: { prop1: 123, prop2: "value", prop3: null },
                match: "/command prop1=123 prop2='value' prop3=null",
            }],
        });
    });
    test("handles command with properties - test 9", () => {
        extractCommandsTester({
            input: "/command prop1=123 prop2='val\\'u\"e' prop3=null\t",
            expected: [{
                command: "command",
                action: null,
                properties: { prop1: 123, prop2: "val\\'u\"e", prop3: null },
                match: "/command prop1=123 prop2='val\\'u\"e' prop3=null",
            }],
        });
    });
    test("handles command with properties - test 10", () => {
        extractCommandsTester({
            input: "/command prop1=123 prop2='value' prop3=null notaprop",
            expected: [{
                command: "command",
                action: null,
                properties: { prop1: 123, prop2: "value", prop3: null },
                match: "/command prop1=123 prop2='value' prop3=null",
            }],
        });
    });

    test("handles wrapped commands - test 1", () => {
        extractCommandsTester({
            input: "suggested: [/command1]",
            expected: [{
                command: "command1",
                action: null,
                properties: {},
                match: "/command1",
            }],
        });
    });
    test("handles wrapped commands - test 2", () => {
        extractCommandsTester({
            input: "/command1 suggested: [/command2]",
            expected: [{
                command: "command1",
                action: null,
                properties: {},
                match: "/command1",
            }, {
                command: "command2",
                action: null,
                properties: {},
                match: "/command2",
            }],
        });
    });
    test("handles wrapped commands - test 3", () => {
        extractCommandsTester({
            input: "suggested: [/command1] recommended: [/command2]",
            expected: [{
                command: "command1",
                action: null,
                properties: {},
                match: "/command1",
            }, {
                command: "command2",
                action: null,
                properties: {},
                match: "/command2",
            }],
        });
    });
    test("handles wrapped commands - test 4", () => {
        console.log("before wrapped commands 4");
        extractCommandsTester({
            input: "suggested: [/command1 action]",
            expected: [{
                command: "command1",
                action: "action",
                properties: {},
                match: "/command1 action",
            }],
        });
        console.log("after wrapped commands 4");
    });
    test("handles wrapped commands - test 5", () => {
        console.log("before wrapped commands 5");
        extractCommandsTester({
            input: "suggested: [/command1 action, /command2 action2]",
            expected: [{
                command: "command1",
                action: "action",
                properties: {},
                match: "/command1 action",
            }, {
                command: "command2",
                action: "action2",
                properties: {},
                match: "/command2 action2",
            }],
        });
        console.log("after wrapped commands 5");
    });
    test("handles wrapped commands - test 6", () => {
        console.log("before wrapped commands 6");
        extractCommandsTester({
            input: "suggested: [/command1 action name='value' prop2=123 thing=\"asdf\"]",
            expected: [{
                command: "command1",
                action: "action",
                properties: { name: "value", prop2: 123, thing: "asdf" },
                match: "/command1 action name='value' prop2=123 thing=\"asdf\"",
            }],
        });
        console.log("after wrapped commands 6");
    });
    test("handles wrapped commands - test 7", () => {
        console.log("before wrapped commands 7");
        extractCommandsTester({
            input: "suggested: [/command1 action name='valu\"e' prop2=123 thing=\"asdf\", /command2 action2]",
            expected: [{
                command: "command1",
                action: "action",
                properties: { name: "valu\"e", prop2: 123, thing: "asdf" },
                match: "/command1 action name='valu\"e' prop2=123 thing=\"asdf\"",
            }, {
                command: "command2",
                action: "action2",
                properties: {},
                match: "/command2 action2",
            }],
        });
        console.log("after wrapped commands 7");
    });

    test("handles newline properly", () => {
        extractCommandsTester({
            input: "/command1\n/command2",
            expected: [{
                command: "command1",
                action: null,
                properties: {},
                match: "/command1",
            }, {
                command: "command2",
                action: null,
                properties: {},
                match: "/command2",
            }],
        });
    });
    test("handles space properly", () => {
        extractCommandsTester({
            input: "/command1 /command2",
            expected: [{
                command: "command1",
                action: null,
                properties: {},
                match: "/command1",
            }, {
                command: "command2",
                action: null,
                properties: {},
                match: "/command2",
            }],
        });
    });
    test("handles tab properly", () => {
        extractCommandsTester({
            input: "/command1\t/command2",
            expected: [{
                command: "command1",
                action: null,
                properties: {},
                match: "/command1",
            }, {
                command: "command2",
                action: null,
                properties: {},
                match: "/command2",
            }],
        });
    });

    test("handles complex scenario with multiple commands and properties - test 1", () => {
        extractCommandsTester({
            input: "/cmd1  prop1=123 /cmd2 action2 \tprop2='text' prop3=4.56\n/cmd3 prop4=null \nprop5='invalid because newline",
            expected: [{
                command: "cmd1",
                action: null,
                properties: { prop1: 123 },
                match: "/cmd1  prop1=123",
            }, {
                command: "cmd2",
                action: "action2",
                properties: { prop2: "text", prop3: 4.56 },
                match: "/cmd2 action2 \tprop2='text' prop3=4.56",
            }, {
                command: "cmd3",
                action: null,
                properties: { prop4: null },
                match: "/cmd3 prop4=null",
            }],
        });
    });

    test("does nothing for non-command text - test 1", () => {
        extractCommandsTester({
            input: "this is a test",
            expected: [],
        });
    });
    test("does nothing for non-command text - test 2", () => {
        extractCommandsTester({
            input: "/a".repeat(1000),
            expected: [],
        });
    });
    test("does nothing for non-command text - test 3", () => {
        extractCommandsTester({
            input: "",
            expected: [],
        });
    });
    test("does nothing for non-command text - test 4", () => {
        extractCommandsTester({
            input: " ",
            expected: [],
        });
    });
    test("does nothing for non-command text - test 5", () => {
        extractCommandsTester({
            input: "\n",
            expected: [],
        });
    });
    test("does nothing for non-command text - test 6", () => {
        extractCommandsTester({
            input: "\t",
            expected: [],
        });
    });
    test("does nothing for non-command text - test 7", () => {
        extractCommandsTester({
            input: "💃",
            expected: [],
        });
    });
    test("does nothing for non-command text - test 8", () => {
        extractCommandsTester({
            input: "你",
            expected: [],
        });
    });
    test("does nothing for non-command text - test 9", () => {
        extractCommandsTester({
            input: "!",
            expected: [],
        });
    });
    test("does nothing for non-command text - test 10", () => {
        extractCommandsTester({
            input: ".",
            expected: [],
        });
    });
    test("does nothing for non-command text - test 11", () => {
        extractCommandsTester({
            input: "123",
            expected: [],
        });
    });
    test("does nothing for non-command text - test 12", () => {
        extractCommandsTester({
            input: "-123",
            expected: [],
        });
    });
    test("does nothing for non-command text - test 13", () => {
        extractCommandsTester({
            input: "0.123",
            expected: [],
        });
    });
    test("does nothing for non-command text - test 14", () => {
        extractCommandsTester({
            input: "null",
            expected: [],
        });
    });
    test("does nothing for non-command text - test 15", () => {
        extractCommandsTester({
            input: "true",
            expected: [],
        });
    });
    test("does nothing for non-command text - test 16", () => {
        extractCommandsTester({
            input: "false",
            expected: [],
        });
    });
    test("does nothing for non-command text - test 17", () => {
        extractCommandsTester({
            input: "123你",
            expected: [],
        });
    });
    test("does nothing for non-command text - test 18", () => {
        extractCommandsTester({
            input: "To whom it may concern,\n\nI am writing to inform you that I am a giant purple dinosaur. I cannot be stopped. I am inevitable. I am Barney.\n\nSincerely,\nYour Worst Nightmare",
            expected: [],
        });
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

