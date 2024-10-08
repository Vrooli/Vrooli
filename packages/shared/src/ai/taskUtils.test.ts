/* eslint-disable @typescript-eslint/ban-ts-comment */
import { LlmTask } from "../api/generated/graphqlTypes";
import { pascalCase } from "../utils/casing";
import { importCommandToTask } from "./config";
import { detectWrappedTasks, extractTasksFromJson, extractTasksFromText, filterInvalidTasks, findCharWithLimit, getValidTasksFromMessage, handleTaskTransitionAction, handleTaskTransitionCode, handleTaskTransitionCommand, handleTaskTransitionOutside, handleTaskTransitionPropName, handleTaskTransitionPropValue, isAlphaNum, isNewline, isWhitespace, removeTasks } from "./taskUtils";
import { MaybeLlmTaskInfo, PartialTaskInfo } from "./types";

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

function bufferPrev(buffer: string[]) {
    return buffer[buffer.length - 1];
}
function withBuffer<T extends object>(buffer: string[], rest: T) {
    return {
        buffer: [...buffer], // clone buffer
        prev: bufferPrev(buffer),
        ...rest,
    };
}

describe("handleTaskTransitionOutside", () => {
    const section = "outside";
    let onCommit, onComplete, onCancel, onStart;
    let rest;
    beforeEach(() => {
        onCommit = jest.fn();
        onComplete = jest.fn();
        onCancel = jest.fn();
        onStart = jest.fn();
        rest = { onCommit, onComplete, onCancel, onStart, hasOpenBracket: false };
    });

    describe("Reset buffer on outside when whitespace encountered", () => {
        test("space", () => {
            const buffer = "asdf".split("");
            const result = handleTaskTransitionOutside({
                curr: " ",
                ...withBuffer(buffer, rest),
            });
            expect(onCommit).not.toHaveBeenCalled();
            expect(result).toMatchObject({ section: "outside", buffer: [] });
        });
        test("tab", () => {
            const buffer = "asdf".split("");
            const result = handleTaskTransitionOutside({
                curr: "\t",
                ...withBuffer(buffer, rest),
            });
            expect(onCommit).not.toHaveBeenCalled();
            expect(result).toMatchObject({ section: "outside", buffer: [] });
        });
        test("newline", () => {
            const buffer = "asdf".split("");
            const result = handleTaskTransitionOutside({
                curr: "\n",
                ...withBuffer(buffer, rest),
            });
            expect(onCommit).not.toHaveBeenCalled();
            expect(result).toMatchObject({ section: "outside", buffer: [] });
        });
    });

    describe("keep adding to outside when not on a slash", () => {
        test("single quote", () => {
            const buffer = "asdf".split("");
            const result = handleTaskTransitionOutside({
                curr: "'",
                ...withBuffer(buffer, rest),
            });
            expect(onCommit).not.toHaveBeenCalled();
            expect(result).toMatchObject({ section, buffer: [...buffer, "'"] });
        });
        test("double quote", () => {
            const buffer = "asdf".split("");
            const result = handleTaskTransitionOutside({
                curr: "\"",
                ...withBuffer(buffer, rest),
            });
            expect(onCommit).not.toHaveBeenCalled();
            expect(result).toMatchObject({ section, buffer: [...buffer, "\""] });
        });
        test("equals sign", () => {
            const buffer = "asdf".split("");
            const result = handleTaskTransitionOutside({
                curr: "=",
                ...withBuffer(buffer, rest),
            });
            expect(onCommit).not.toHaveBeenCalled();
            expect(result).toMatchObject({ section, buffer: [...buffer, "="] });
        });
        test("letter", () => {
            const buffer = "asdf".split("");
            const result = handleTaskTransitionOutside({
                curr: "a",
                ...withBuffer(buffer, rest),
            });
            expect(onCommit).not.toHaveBeenCalled();
            expect(result).toMatchObject({ section, buffer: [...buffer, "a"] });
        });
        test("number", () => {
            const buffer = "asdf".split("");
            const result = handleTaskTransitionOutside({
                curr: "1",
                ...withBuffer(buffer, rest),
            });
            expect(onCommit).not.toHaveBeenCalled();
            expect(result).toMatchObject({ section, buffer: [...buffer, "1"] });
        });
        test("other alphabets", () => {
            const buffer = "asdf".split("");
            const result = handleTaskTransitionOutside({
                curr: "你",
                ...withBuffer(buffer, rest),
            });
            expect(onCommit).not.toHaveBeenCalled();
            expect(result).toMatchObject({ section, buffer: [...buffer, "你"] });
        });
        test("emojis", () => {
            const buffer = "asdf".split("");
            const result = handleTaskTransitionOutside({
                curr: "👋",
                ...withBuffer(buffer, rest),
            });
            expect(onCommit).not.toHaveBeenCalled();
            expect(result).toMatchObject({ section, buffer: [...buffer, "👋"] });
        });
        test("symbols", () => {
            const buffer = "asdf".split("");
            const result = handleTaskTransitionOutside({
                curr: "!",
                ...withBuffer(buffer, rest),
            });
            expect(onCommit).not.toHaveBeenCalled();
            expect(result).toMatchObject({ section, buffer: [...buffer, "!"] });
        });
    });

    describe("invalid command starts", () => {
        describe("not preceded by whitespace", () => {
            test("letter", () => {
                const buffer = "asdf".split("");
                const result = handleTaskTransitionOutside({
                    curr: "/",
                    ...withBuffer(buffer, rest),
                });
                expect(onCommit).not.toHaveBeenCalled();
                expect(result).toMatchObject({ section, buffer: [...buffer, "/"] });
            });
            test("number", () => {
                const buffer = "1234".split("");
                const result = handleTaskTransitionOutside({
                    curr: "/",
                    ...withBuffer(buffer, rest),
                });
                expect(onCommit).not.toHaveBeenCalled();
                expect(result).toMatchObject({ section, buffer: [...buffer, "/"] });
            });
            test("symbol", () => {
                const buffer = "!@#$".split("");
                const result = handleTaskTransitionOutside({
                    curr: "/",
                    ...withBuffer(buffer, rest),
                });
                expect(onCommit).not.toHaveBeenCalled();
                expect(result).toMatchObject({ section, buffer: [...buffer, "/"] });
            });
            test("emoji", () => {
                const buffer = "🙌💃".split("");
                const result = handleTaskTransitionOutside({
                    curr: "/",
                    ...withBuffer(buffer, rest),
                });
                expect(onCommit).not.toHaveBeenCalled();
                expect(result).toMatchObject({ section, buffer: [...buffer, "/"] });
            });
            test("another slash", () => {
                const buffer = "/".split("");
                const result = handleTaskTransitionOutside({
                    curr: "/",
                    ...withBuffer(buffer, rest),
                });
                expect(onCommit).not.toHaveBeenCalled();
                expect(result).toMatchObject({ section, buffer: [...buffer, "/"] });
            });
        });
    });

    describe("valid command starts", () => {
        test("buffer is empty", () => {
            const buffer = "".split("");
            const result = handleTaskTransitionOutside({
                curr: "/",
                ...withBuffer(buffer, rest),
            });
            expect(onCommit).not.toHaveBeenCalled();
            expect(result).toMatchObject({ section: "command", buffer: [] });
        });
        describe("after whitespace", () => {
            test("newline", () => {
                const buffer = "asdf\n".split("");
                const result = handleTaskTransitionOutside({
                    curr: "/",
                    ...withBuffer(buffer, rest),
                });
                expect(onCommit).not.toHaveBeenCalled();
                expect(result).toMatchObject({ section: "command", buffer: [] });
            });
            test("space", () => {
                const buffer = "asdf ".split("");
                const result = handleTaskTransitionOutside({
                    curr: "/",
                    ...withBuffer(buffer, rest),
                });
                expect(onCommit).not.toHaveBeenCalled();
                expect(result).toMatchObject({ section: "command", buffer: [] });
            });
            test("tab", () => {
                const buffer = "asdf\t".split("");
                const result = handleTaskTransitionOutside({
                    curr: "/",
                    ...withBuffer(buffer, rest),
                });
                expect(onCommit).not.toHaveBeenCalled();
                expect(result).toMatchObject({ section: "command", buffer: [] });
            });
        });
        describe("after open bracket (used for suggested commands)", () => {
            test("newline", () => {
                const buffer = "asdf\n[".split("");
                const result = handleTaskTransitionOutside({
                    curr: "/",
                    ...withBuffer(buffer, rest),
                    hasOpenBracket: true,
                });
                expect(onCommit).not.toHaveBeenCalled();
                expect(result).toMatchObject({ section: "command", buffer: [] });
            });
            test("space", () => {
                const buffer = "asdf [".split("");
                const result = handleTaskTransitionOutside({
                    curr: "/",
                    ...withBuffer(buffer, rest),
                    hasOpenBracket: true,
                });
                expect(onCommit).not.toHaveBeenCalled();
                expect(result).toMatchObject({ section: "command", buffer: [] });
            });
            test("tab", () => {
                const buffer = "asdf\t[".split("");
                const result = handleTaskTransitionOutside({
                    curr: "/",
                    ...withBuffer(buffer, rest),
                    hasOpenBracket: true,
                });
                expect(onCommit).not.toHaveBeenCalled();
                expect(result).toMatchObject({ section: "command", buffer: [] });
            });
            test("letter", () => {
                const buffer = "asdf[".split("");
                const result = handleTaskTransitionOutside({
                    curr: "/",
                    ...withBuffer(buffer, rest),
                    hasOpenBracket: true,
                });
                expect(onCommit).not.toHaveBeenCalled();
                expect(result).toMatchObject({ section: "command", buffer: [] });
            });
        });
    });

    describe("invalid or not yet code blocks", () => {
        describe("single backtick - code is triggered on first non-tick", () => {
            test("buffer is empty", () => {
                const buffer = "".split("");
                const result = handleTaskTransitionOutside({
                    curr: "`",
                    ...withBuffer(buffer, rest),
                });
                expect(onCommit).not.toHaveBeenCalled();
                expect(result).toMatchObject({ section, buffer: [...buffer, "`"] });
            });
            test("buffer is not empty", () => {
                const buffer = "asdf".split("");
                const result = handleTaskTransitionOutside({
                    curr: "`",
                    ...withBuffer(buffer, rest),
                });
                expect(onCommit).not.toHaveBeenCalled();
                expect(result).toMatchObject({ section, buffer: [...buffer, "`"] });
            });
        });
        test("double backtick", () => {
            const buffer = "asdf`".split("");
            const result = handleTaskTransitionOutside({
                curr: "`",
                ...withBuffer(buffer, rest),
            });
            expect(onCommit).not.toHaveBeenCalled();
            expect(result).toMatchObject({ section, buffer: [...buffer, "`"] });
        });
        test("triple backtick", () => {
            const buffer = "asdf``".split("");
            const result = handleTaskTransitionOutside({
                curr: "`",
                ...withBuffer(buffer, rest),
            });
            expect(onCommit).not.toHaveBeenCalled();
            expect(result).toMatchObject({ section, buffer: [...buffer, "`"] });
        });
        test("quadruple backtick", () => {
            const buffer = "asdf```".split("");
            const result = handleTaskTransitionOutside({
                curr: "`",
                ...withBuffer(buffer, rest),
            });
            expect(onCommit).not.toHaveBeenCalled();
            expect(result).toMatchObject({ section, buffer: [...buffer, "`"] });
        });
        test("newline after single backtick", () => {
            const buffer = "`".split("");
            const result = handleTaskTransitionOutside({
                curr: "\n",
                ...withBuffer(buffer, rest),
            });
            expect(onCommit).not.toHaveBeenCalled();
            expect(result).toMatchObject({ section, buffer: [] });
        });
    });

    describe("detecting code blocks", () => {
        describe("single backtick code blocks", () => {
            test("buffer is empty besides backtick", () => {
                const buffer = "`".split("");
                const result = handleTaskTransitionOutside({
                    curr: "1",
                    ...withBuffer(buffer, rest),
                });
                expect(onCommit).not.toHaveBeenCalled();
                expect(result).toMatchObject({ section: "code", buffer: ["`", "1"] }); // Keep only the backtick and curr
            });
            test("buffer is not empty besides backtick", () => {
                const buffer = "asdf`".split("");
                const result = handleTaskTransitionOutside({
                    curr: "f",
                    ...withBuffer(buffer, rest),
                });
                expect(onCommit).not.toHaveBeenCalled();
                expect(result).toMatchObject({ section: "code", buffer: ["`", "f"] }); // Keep only the backtick and curr
            });
        });
        describe("triple backtick code blocks", () => {
            test("buffer is empty besides backticks", () => {
                const buffer = "```".split("");
                const result = handleTaskTransitionOutside({
                    curr: "a",
                    ...withBuffer(buffer, rest),
                });
                expect(onCommit).not.toHaveBeenCalled();
                expect(result).toMatchObject({ section: "code", buffer: "```a".split("") }); // Keep only the backticks and curr
            });
            test("buffer is not empty besides backticks", () => {
                const buffer = "asdf$#@$#```".split("");
                const result = handleTaskTransitionOutside({
                    curr: "x",
                    ...withBuffer(buffer, rest),
                });
                expect(onCommit).not.toHaveBeenCalled();
                expect(result).toMatchObject({ section: "code", buffer: "```x".split("") }); // Keep only the backticks and curr
            });
        });
        describe("HTML-style code tags", () => {
            test("preceded by whitespace", () => {
                const buffer = "asdf <code".split("");
                const result = handleTaskTransitionOutside({
                    curr: ">",
                    ...withBuffer(buffer, rest),
                });
                expect(onCommit).not.toHaveBeenCalled();
                expect(result).toMatchObject({ section: "code", buffer: "<code>".split("") }); // Keep only the tag
            });
            test("preceded by non-whitespace", () => {
                const buffer = "asdf<code".split("");
                const result = handleTaskTransitionOutside({
                    curr: ">",
                    ...withBuffer(buffer, rest),
                });
                expect(onCommit).not.toHaveBeenCalled();
                expect(result).toMatchObject({ section: "code", buffer: "<code>".split("") }); // Keep only the tag
            });
            test("not fully closed", () => {
                const buffer = "asdf <code".split("");
                const result = handleTaskTransitionOutside({
                    curr: "x",
                    ...withBuffer(buffer, rest),
                });
                expect(onCommit).not.toHaveBeenCalled();
                expect(result).toMatchObject({ section: "outside", buffer: [...buffer, "x"] });
            });
        });
    });
});

describe("handleTaskTransitionCode", () => {
    const section = "code";
    let onCancel;
    let rest;
    beforeEach(() => {
        onCancel = jest.fn();
        rest = { onCancel, hasOpenBracket: false };
    });

    test("shouldn't be here if buffer is empty (backticks/tag is carried over from 'outside' buffer)", () => {
        const buffer = "".split("");
        const result = handleTaskTransitionCode({
            curr: "`",
            ...withBuffer(buffer, rest),
        });
        expect(onCancel).toHaveBeenCalled();
        expect(result).toMatchObject({ section: "outside", buffer: [] });
    });

    describe("can't determine code block type", () => {
        describe("single backtick", () => {
            describe("additional characters at start of buffer", () => {
                test("letter", () => {
                    const buffer = "afdsa`".split("");
                    const result = handleTaskTransitionCode({
                        curr: "a",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCancel).toHaveBeenCalled();
                    expect(result).toMatchObject({ section: "outside", buffer: [] });
                });
                test("number", () => {
                    const buffer = "1234`".split("");
                    const result = handleTaskTransitionCode({
                        curr: "1",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCancel).toHaveBeenCalled();
                    expect(result).toMatchObject({ section: "outside", buffer: [] });
                });
                test("special character", () => {
                    const buffer = "!@#$`".split("");
                    const result = handleTaskTransitionCode({
                        curr: "@",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCancel).toHaveBeenCalled();
                    expect(result).toMatchObject({ section: "outside", buffer: [] });
                });
            });
        });
        describe("triple backtick", () => {
            describe("additional characters at start of buffer", () => {
                test("letter", () => {
                    const buffer = "afdsa```".split("");
                    const result = handleTaskTransitionCode({
                        curr: "a",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCancel).toHaveBeenCalled();
                    expect(result).toMatchObject({ section: "outside", buffer: [] });
                });
                test("number", () => {
                    const buffer = "1234```".split("");
                    const result = handleTaskTransitionCode({
                        curr: "1",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCancel).toHaveBeenCalled();
                    expect(result).toMatchObject({ section: "outside", buffer: [] });
                });
                test("special character", () => {
                    const buffer = "!@#$```".split("");
                    const result = handleTaskTransitionCode({
                        curr: "@",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCancel).toHaveBeenCalled();
                    expect(result).toMatchObject({ section: "outside", buffer: [] });
                });
            });
        });
        describe("HTML-style code tags", () => {
            describe("additional characters at start of buffer", () => {
                test("letter", () => {
                    const buffer = "afdsa<code>".split("");
                    const result = handleTaskTransitionCode({
                        curr: "a",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCancel).toHaveBeenCalled();
                    expect(result).toMatchObject({ section: "outside", buffer: [] });
                });
                test("number", () => {
                    const buffer = "1234<code>".split("");
                    const result = handleTaskTransitionCode({
                        curr: "1",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCancel).toHaveBeenCalled();
                    expect(result).toMatchObject({ section: "outside", buffer: [] });
                });
                test("special character", () => {
                    const buffer = "!@#$<code>".split("");
                    const result = handleTaskTransitionCode({
                        curr: "@",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCancel).toHaveBeenCalled();
                    expect(result).toMatchObject({ section: "outside", buffer: [] });
                });
            });
        });
    });

    describe("single backtick code blocks", () => {
        describe("continues buffering", () => {
            describe("only tick in buffer", () => {
                test("letter", () => {
                    const buffer = "`".split("");
                    const result = handleTaskTransitionCode({
                        curr: "a",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCancel).not.toHaveBeenCalled();
                    expect(result).toMatchObject({ section, buffer: [...buffer, "a"] });
                });
                test("number", () => {
                    const buffer = "`".split("");
                    const result = handleTaskTransitionCode({
                        curr: "1",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCancel).not.toHaveBeenCalled();
                    expect(result).toMatchObject({ section, buffer: [...buffer, "1"] });
                });
                test("special character", () => {
                    const buffer = "`".split("");
                    const result = handleTaskTransitionCode({
                        curr: "@",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCancel).not.toHaveBeenCalled();
                    expect(result).toMatchObject({ section, buffer: [...buffer, "@"] });
                });
                test("space", () => {
                    const buffer = "`".split("");
                    const result = handleTaskTransitionCode({
                        curr: " ",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCancel).not.toHaveBeenCalled();
                    expect(result).toMatchObject({ section, buffer: [...buffer, " "] });
                });
                test("tab", () => {
                    const buffer = "`".split("");
                    const result = handleTaskTransitionCode({
                        curr: "\t",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCancel).not.toHaveBeenCalled();
                    expect(result).toMatchObject({ section, buffer: [...buffer, "\t"] });
                });
                test("emoji", () => {
                    const buffer = "`".split("");
                    const result = handleTaskTransitionCode({
                        curr: "👋",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCancel).not.toHaveBeenCalled();
                    expect(result).toMatchObject({ section, buffer: [...buffer, "👋"] });
                });
            });
            describe("more than tick in buffer", () => {
                test("letter", () => {
                    const buffer = "`fdsaf".split("");
                    const result = handleTaskTransitionCode({
                        curr: "a",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCancel).not.toHaveBeenCalled();
                    expect(result).toMatchObject({ section, buffer: [...buffer, "a"] });
                });
                test("number", () => {
                    const buffer = "` fdsaf".split("");
                    const result = handleTaskTransitionCode({
                        curr: "1",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCancel).not.toHaveBeenCalled();
                    expect(result).toMatchObject({ section, buffer: [...buffer, "1"] });
                });
                test("special character", () => {
                    const buffer = "`fdsaffdksa;f".split("");
                    const result = handleTaskTransitionCode({
                        curr: "@",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCancel).not.toHaveBeenCalled();
                    expect(result).toMatchObject({ section, buffer: [...buffer, "@"] });
                });
                test("space", () => {
                    const buffer = "`;2342".split("");
                    const result = handleTaskTransitionCode({
                        curr: " ",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCancel).not.toHaveBeenCalled();
                    expect(result).toMatchObject({ section, buffer: [...buffer, " "] });
                });
                test("tab", () => {
                    const buffer = "`!@#".split("");
                    const result = handleTaskTransitionCode({
                        curr: "\t",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCancel).not.toHaveBeenCalled();
                    expect(result).toMatchObject({ section, buffer: [...buffer, "\t"] });
                });
                test("emoji", () => {
                    const buffer = "`zzz".split("");
                    const result = handleTaskTransitionCode({
                        curr: "👋",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCancel).not.toHaveBeenCalled();
                    expect(result).toMatchObject({ section, buffer: [...buffer, "👋"] });
                });
            });
            test("adding another tick (could be typing triple tick, so we can't cancel out", () => {
                const buffer = "`".split("");
                const result = handleTaskTransitionCode({
                    curr: "`",
                    ...withBuffer(buffer, rest),
                });
                expect(onCancel).not.toHaveBeenCalled();
                expect(result).toMatchObject({ section, buffer: [...buffer, "`"] });
            });
        });
        describe("stops buffering", () => {
            describe("only tick in buffer", () => {
                test("newline", () => {
                    const buffer = "`".split("");
                    const result = handleTaskTransitionCode({
                        curr: "\n",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCancel).toHaveBeenCalled();
                    expect(result).toMatchObject({ section: "outside", buffer: [] });
                });
            });
            describe("more than tick in buffer", () => {
                test("newline", () => {
                    const buffer = "fdsaf`".split("");
                    const result = handleTaskTransitionCode({
                        curr: "\n",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCancel).toHaveBeenCalled();
                    expect(result).toMatchObject({ section: "outside", buffer: [] });
                });
            });
        });
    });

    describe("triple backtick code blocks", () => {
        describe("continues buffering", () => {
            describe("only ticks in buffer", () => {
                test("letter", () => {
                    const buffer = "```".split("");
                    const result = handleTaskTransitionCode({
                        curr: "a",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCancel).not.toHaveBeenCalled();
                    expect(result).toMatchObject({ section, buffer: [...buffer, "a"] });
                });
                test("number", () => {
                    const buffer = "```".split("");
                    const result = handleTaskTransitionCode({
                        curr: "1",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCancel).not.toHaveBeenCalled();
                    expect(result).toMatchObject({ section, buffer: [...buffer, "1"] });
                });
                test("special character", () => {
                    const buffer = "```".split("");
                    const result = handleTaskTransitionCode({
                        curr: "@",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCancel).not.toHaveBeenCalled();
                    expect(result).toMatchObject({ section, buffer: [...buffer, "@"] });
                });
                test("space", () => {
                    const buffer = "```".split("");
                    const result = handleTaskTransitionCode({
                        curr: " ",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCancel).not.toHaveBeenCalled();
                    expect(result).toMatchObject({ section, buffer: [...buffer, " "] });
                });
                test("tab", () => {
                    const buffer = "```".split("");
                    const result = handleTaskTransitionCode({
                        curr: "\t",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCancel).not.toHaveBeenCalled();
                    expect(result).toMatchObject({ section, buffer: [...buffer, "\t"] });
                });
                test("emoji", () => {
                    const buffer = "```".split("");
                    const result = handleTaskTransitionCode({
                        curr: "👋",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCancel).not.toHaveBeenCalled();
                    expect(result).toMatchObject({ section, buffer: [...buffer, "👋"] });
                });
                test("newline", () => {
                    const buffer = "```".split("");
                    const result = handleTaskTransitionCode({
                        curr: "\n",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCancel).not.toHaveBeenCalled();
                    expect(result).toMatchObject({ section, buffer: [...buffer, "\n"] });
                });
            });
            describe("more than ticks in buffer", () => {
                test("letter", () => {
                    const buffer = "```fdsaf".split("");
                    const result = handleTaskTransitionCode({
                        curr: "a",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCancel).not.toHaveBeenCalled();
                    expect(result).toMatchObject({ section, buffer: [...buffer, "a"] });
                });
                test("number", () => {
                    const buffer = "``` fdsaf".split("");
                    const result = handleTaskTransitionCode({
                        curr: "1",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCancel).not.toHaveBeenCalled();
                    expect(result).toMatchObject({ section, buffer: [...buffer, "1"] });
                });
                test("special character", () => {
                    const buffer = "```fdsaffdksa;f".split("");
                    const result = handleTaskTransitionCode({
                        curr: "@",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCancel).not.toHaveBeenCalled();
                    expect(result).toMatchObject({ section, buffer: [...buffer, "@"] });
                });
                test("space", () => {
                    const buffer = "```;2342".split("");
                    const result = handleTaskTransitionCode({
                        curr: " ",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCancel).not.toHaveBeenCalled();
                    expect(result).toMatchObject({ section, buffer: [...buffer, " "] });
                });
                test("tab", () => {
                    const buffer = "```!@#".split("");
                    const result = handleTaskTransitionCode({
                        curr: "\t",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCancel).not.toHaveBeenCalled();
                    expect(result).toMatchObject({ section, buffer: [...buffer, "\t"] });
                });
                test("emoji", () => {
                    const buffer = "```zzz".split("");
                    const result = handleTaskTransitionCode({
                        curr: "👋",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCancel).not.toHaveBeenCalled();
                    expect(result).toMatchObject({ section, buffer: [...buffer, "👋"] });
                });
                test("newline", () => {
                    const buffer = "```fdsaf".split("");
                    const result = handleTaskTransitionCode({
                        curr: "\n",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCancel).not.toHaveBeenCalled();
                    expect(result).toMatchObject({ section, buffer: [...buffer, "\n"] });
                });
            });
            describe("ticks (but not enough to close)", () => {
                test("1 tick with text between", () => {
                    const buffer = "```a".split("");
                    const result = handleTaskTransitionCode({
                        curr: "`",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCancel).not.toHaveBeenCalled();
                    expect(result).toMatchObject({ section, buffer: [...buffer, "`"] });
                });
                test("2 ticks with text between", () => {
                    const buffer = "```a`".split("");
                    const result = handleTaskTransitionCode({
                        curr: "`",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCancel).not.toHaveBeenCalled();
                    expect(result).toMatchObject({ section, buffer: [...buffer, "`"] });
                });
            });
        });
        describe("stops buffering", () => {
            test("4 ticks in a row - don't support empty code blocks", () => {
                const buffer = "```".split("");
                const result = handleTaskTransitionCode({
                    curr: "`",
                    ...withBuffer(buffer, rest),
                });
                expect(onCancel).toHaveBeenCalled();
                expect(result).toMatchObject({ section: "outside", buffer: [] });
            });
            test("5 ticks in a row - don't support empty code blocks", () => {
                const buffer = "````".split("");
                const result = handleTaskTransitionCode({
                    curr: "`",
                    ...withBuffer(buffer, rest),
                });
                expect(onCancel).toHaveBeenCalled();
                expect(result).toMatchObject({ section: "outside", buffer: [] });
            });
            test("6th tick - only ticks in buffer", () => {
                const buffer = "`````".split("");
                const result = handleTaskTransitionCode({
                    curr: "`",
                    ...withBuffer(buffer, rest),
                });
                expect(onCancel).toHaveBeenCalled();
                expect(result).toMatchObject({ section: "outside", buffer: [] });
            });
            test("6th tick - letters between", () => {
                const buffer = "```aasdf``".split("");
                const result = handleTaskTransitionCode({
                    curr: "`",
                    ...withBuffer(buffer, rest),
                });
                expect(onCancel).toHaveBeenCalled();
                expect(result).toMatchObject({ section: "outside", buffer: [] });
            });
            test("6th tick - lots of stuff between", () => {
                const buffer = "```fdsafds\nhello`fdsafdsaf;3423243``".split("");
                const result = handleTaskTransitionCode({
                    curr: "`",
                    ...withBuffer(buffer, rest),
                });
                expect(onCancel).toHaveBeenCalled();
                expect(result).toMatchObject({ section: "outside", buffer: [] });
            });
        });
    });

    describe("HTML-style code tags", () => {
        describe("continuing", () => {
            describe("valid characters", () => {
                test("space", () => {
                    const buffer = "<code>".split("");
                    const result = handleTaskTransitionCode({
                        curr: " ",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCancel).not.toHaveBeenCalled();
                    expect(result).toMatchObject({ section, buffer: [...buffer, " "] });
                });
                test("tab", () => {
                    const buffer = "<code>".split("");
                    const result = handleTaskTransitionCode({
                        curr: "\t",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCancel).not.toHaveBeenCalled();
                    expect(result).toMatchObject({ section, buffer: [...buffer, "\t"] });
                });
                test("newline", () => {
                    const buffer = "<code>".split("");
                    const result = handleTaskTransitionCode({
                        curr: "\n",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCancel).not.toHaveBeenCalled();
                    expect(result).toMatchObject({ section, buffer: [...buffer, "\n"] });
                });
                test("letter", () => {
                    const buffer = "<code>".split("");
                    const result = handleTaskTransitionCode({
                        curr: "a",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCancel).not.toHaveBeenCalled();
                    expect(result).toMatchObject({ section, buffer: [...buffer, "a"] });
                });
                test("number", () => {
                    const buffer = "<code>".split("");
                    const result = handleTaskTransitionCode({
                        curr: "1",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCancel).not.toHaveBeenCalled();
                    expect(result).toMatchObject({ section, buffer: [...buffer, "1"] });
                });
                test("special character", () => {
                    const buffer = "<code>".split("");
                    const result = handleTaskTransitionCode({
                        curr: "@",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCancel).not.toHaveBeenCalled();
                    expect(result).toMatchObject({ section, buffer: [...buffer, "@"] });
                });
                test("emoji", () => {
                    const buffer = "<code>".split("");
                    const result = handleTaskTransitionCode({
                        curr: "👋",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCancel).not.toHaveBeenCalled();
                    expect(result).toMatchObject({ section, buffer: [...buffer, "👋"] });
                });
                test("other alphabets", () => {
                    const buffer = "<code>".split("");
                    const result = handleTaskTransitionCode({
                        curr: "你",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCancel).not.toHaveBeenCalled();
                    expect(result).toMatchObject({ section, buffer: [...buffer, "你"] });
                });
            });
            describe("with stuff in code block", () => {
                test("test 1", () => {
                    const buffer = "<code>asdf".split("");
                    const result = handleTaskTransitionCode({
                        curr: " ",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCancel).not.toHaveBeenCalled();
                    expect(result).toMatchObject({ section, buffer: [...buffer, " "] });
                });
                test("test 2", () => {
                    const buffer = "<code>fkdsa;f4382483fdsfdsaf; fcofndsafdsaf\n\n\nfjdsaf;!!%$#@3".split("");
                    const result = handleTaskTransitionCode({
                        curr: " ",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCancel).not.toHaveBeenCalled();
                    expect(result).toMatchObject({ section, buffer: [...buffer, " "] });
                });
            });
            describe("closing tag not finished or invalid", () => {
                test("partial closing tag 1", () => {
                    const buffer = "<code>fdasfd".split("");
                    const result = handleTaskTransitionCode({
                        curr: "<",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCancel).not.toHaveBeenCalled();
                    expect(result).toMatchObject({ section, buffer: [...buffer, "<"] });
                });
                test("partial closing tag 2", () => {
                    const buffer = "<code>fdasfd<".split("");
                    const result = handleTaskTransitionCode({
                        curr: "/",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCancel).not.toHaveBeenCalled();
                    expect(result).toMatchObject({ section, buffer: [...buffer, "/"] });
                });
                test("partial closing tag 3", () => {
                    const buffer = "<code>fdasfd</".split("");
                    const result = handleTaskTransitionCode({
                        curr: "c",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCancel).not.toHaveBeenCalled();
                    expect(result).toMatchObject({ section, buffer: [...buffer, "c"] });
                });
                test("partial closing tag 4", () => {
                    const buffer = "<code>fdasfd</c".split("");
                    const result = handleTaskTransitionCode({
                        curr: "o",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCancel).not.toHaveBeenCalled();
                    expect(result).toMatchObject({ section, buffer: [...buffer, "o"] });
                });
                test("partial closing tag 5", () => {
                    const buffer = "<code>fdasfd</co".split("");
                    const result = handleTaskTransitionCode({
                        curr: "d",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCancel).not.toHaveBeenCalled();
                    expect(result).toMatchObject({ section, buffer: [...buffer, "d"] });
                });
                test("partial closing tag 6", () => {
                    const buffer = "<code>fdasfd</cod".split("");
                    const result = handleTaskTransitionCode({
                        curr: "e",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCancel).not.toHaveBeenCalled();
                    expect(result).toMatchObject({ section, buffer: [...buffer, "e"] });
                });
                test("invalid closing tag", () => {
                    const buffer = "<code>fdasdf</c0de".split("");
                    const result = handleTaskTransitionCode({
                        curr: ">",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCancel).not.toHaveBeenCalled();
                    expect(result).toMatchObject({ section, buffer: [...buffer, ">"] });
                });
            });
        });
        describe("canceling", () => {
            test("valid closing tag", () => {
                const buffer = "<code>fdasfd</code".split("");
                const result = handleTaskTransitionCode({
                    curr: ">",
                    ...withBuffer(buffer, rest),
                });
                expect(onCancel).toHaveBeenCalled();
                expect(result).toMatchObject({ section: "outside", buffer: [] });
            });
        });
    });
});

describe("handleTaskTransitionCommand", () => {
    const section = "command";
    let onCancel, onCommit, onComplete;
    let rest;
    beforeEach(() => {
        onCancel = jest.fn();
        onCommit = jest.fn();
        onComplete = jest.fn();
        rest = { onCancel, onCommit, onComplete, hasOpenBracket: false };
    });

    describe("continues buffering", () => {
        test("letter", () => {
            const buffer = "test".split("");
            const result = handleTaskTransitionCommand({
                curr: "a",
                ...withBuffer(buffer, rest),
            });
            expect(onCommit).not.toHaveBeenCalled();
            expect(result).toMatchObject({ section, buffer: [...buffer, "a"] });
        });
        test("number", () => {
            const buffer = "test".split("");
            const result = handleTaskTransitionCommand({
                curr: "1",
                ...withBuffer(buffer, rest),
            });
            expect(onCommit).not.toHaveBeenCalled();
            expect(result).toMatchObject({ section, buffer: [...buffer, "1"] });
        });
    });

    describe("commits", () => {
        test("newline", () => {
            const buffer = "test".split("");
            const result = handleTaskTransitionCommand({
                curr: "\n",
                ...withBuffer(buffer, rest),
            });
            expect(onCommit).toHaveBeenCalledWith("command", buffer.join(""));
            expect(result).toMatchObject({ section: "outside", buffer: [] });
        });
    });

    describe("resets", () => {
        test("other alphabets", () => {
            const buffer = "test".split("");
            const result = handleTaskTransitionCommand({
                curr: "你",
                ...withBuffer(buffer, rest),
            });
            expect(onCommit).not.toHaveBeenCalled();
            expect(result).toMatchObject({ section: "outside", buffer: [] });
        });
        test("emojis", () => {
            const buffer = "test".split("");
            const result = handleTaskTransitionCommand({
                curr: "👋",
                ...withBuffer(buffer, rest),
            });
            expect(onCommit).not.toHaveBeenCalled();
            expect(result).toMatchObject({ section: "outside", buffer: [] });
        });
        test("symbols", () => {
            const buffer = "test".split("");
            const result = handleTaskTransitionCommand({
                curr: "!",
                ...withBuffer(buffer, rest),
            });
            expect(onCommit).not.toHaveBeenCalled();
            expect(result).toMatchObject({ section: "outside", buffer: [] });
        });
    });

    describe("changes to action state", () => {
        test("space", () => {
            const buffer = "test".split("");
            const result = handleTaskTransitionCommand({
                curr: " ",
                ...withBuffer(buffer, rest),
            });
            expect(onCommit).toHaveBeenCalledWith("command", buffer.join(""));
            expect(result).toMatchObject({ section: "action", buffer: [] });
        });
        test("tab", () => {
            const buffer = "test".split("");
            const result = handleTaskTransitionCommand({
                curr: "\t",
                ...withBuffer(buffer, rest),
            });
            expect(onCommit).toHaveBeenCalledWith("command", buffer.join(""));
            expect(result).toMatchObject({ section: "action", buffer: [] });
        });
    });
});

describe("handleTaskTransitionAction", () => {
    const section = "action";
    let onCancel, onCommit, onComplete;
    let rest;
    beforeEach(() => {
        onCancel = jest.fn();
        onCommit = jest.fn();
        onComplete = jest.fn();
        rest = { onCancel, onCommit, onComplete, hasOpenBracket: false };
    });

    describe("continues buffering", () => {
        test("letter", () => {
            const buffer = "test".split("");
            const result = handleTaskTransitionAction({
                curr: "a",
                ...withBuffer(buffer, rest),
            });
            expect(onCommit).not.toHaveBeenCalled();
            expect(result).toMatchObject({ section, buffer: [...buffer, "a"] });
        });
        test("number", () => {
            const buffer = "test".split("");
            const result = handleTaskTransitionAction({
                curr: "1",
                ...withBuffer(buffer, rest),
            });
            expect(onCommit).not.toHaveBeenCalled();
            expect(result).toMatchObject({ section, buffer: [...buffer, "1"] });
        });
    });

    describe("cancels", () => {
        test("other alphabets", () => {
            const buffer = "test".split("");
            const result = handleTaskTransitionAction({
                curr: "你",
                ...withBuffer(buffer, rest),
            });
            expect(onCommit).not.toHaveBeenCalled();
            expect(result).toMatchObject({ section: "outside", buffer: [] });
        });
        test("emojis", () => {
            const buffer = "test".split("");
            const result = handleTaskTransitionAction({
                curr: "👋",
                ...withBuffer(buffer, rest),
            });
            expect(onCommit).not.toHaveBeenCalled();
            expect(result).toMatchObject({ section: "outside", buffer: [] });
        });
        test("symbols", () => {
            const buffer = "test".split("");
            const result = handleTaskTransitionAction({
                curr: "!",
                ...withBuffer(buffer, rest),
            });
            expect(onCommit).not.toHaveBeenCalled();
            expect(result).toMatchObject({ section: "outside", buffer: [] });
        });
        test("too long", () => {
            const buffer = "test".repeat(10).split("");
            const result = handleTaskTransitionAction({
                curr: "a",
                ...withBuffer(buffer, rest),
                bufferLengthLimit: 4,
            });
            expect(onCommit).not.toHaveBeenCalled();
            expect(onComplete).toHaveBeenCalled();
            expect(result).toMatchObject({ section: "outside", buffer: [] });
        });
    });

    describe("commits and moves to propName", () => {
        test("space", () => {
            const buffer = "add".split("");
            const result = handleTaskTransitionAction({
                curr: " ",
                ...withBuffer(buffer, rest),
            });
            expect(onCommit).toHaveBeenCalledWith("action", buffer.join(""));
            expect(result).toMatchObject({ section: "propName", buffer: [] });
        });
        test("tab", () => {
            const buffer = "add".split("");
            const result = handleTaskTransitionAction({
                curr: "\t",
                ...withBuffer(buffer, rest),
            });
            expect(onCommit).toHaveBeenCalledWith("action", buffer.join(""));
            expect(result).toMatchObject({ section: "propName", buffer: [] });
        });
    });

    describe("commits and moves outside", () => {
        test("newline", () => {
            const buffer = "add".split("");
            const result = handleTaskTransitionAction({
                curr: "\n",
                ...withBuffer(buffer, rest),
            });
            expect(onCommit).toHaveBeenCalledWith("action", buffer.join(""));
            expect(result).toMatchObject({ section: "outside", buffer: [] });
        });
    });

    test("commits as property name on equals sign", () => {
        const buffer = "name".split("");
        const result = handleTaskTransitionAction({
            curr: "=",
            ...withBuffer(buffer, rest),
        });
        expect(onCommit).toHaveBeenCalledWith("propName", buffer.join(""));
        expect(result).toMatchObject({ section: "propValue", buffer: [] });
    });
});

describe("handleTaskTransitionPropName", () => {
    const section = "propName";
    let onCancel, onCommit, onComplete;
    let rest;
    beforeEach(() => {
        onCancel = jest.fn();
        onCommit = jest.fn();
        onComplete = jest.fn();
        rest = { onCancel, onCommit, onComplete, hasOpenBracket: false };
    });

    describe("continues buffering", () => {
        test("letter", () => {
            const buffer = "name".split("");
            const result = handleTaskTransitionPropName({
                curr: "a",
                ...withBuffer(buffer, rest),
            });
            expect(onCommit).not.toHaveBeenCalled();
            expect(result).toMatchObject({ section, buffer: [...buffer, "a"] });
        });
        test("number", () => {
            const buffer = "name".split("");
            const result = handleTaskTransitionPropName({
                curr: "1",
                ...withBuffer(buffer, rest),
            });
            expect(onCommit).not.toHaveBeenCalled();
            expect(result).toMatchObject({ section, buffer: [...buffer, "1"] });
        });
    });

    describe("cancels", () => {
        test("space", () => {
            const buffer = "name".split("");
            const result = handleTaskTransitionPropName({
                curr: " ",
                ...withBuffer(buffer, rest),
            });
            expect(onCommit).not.toHaveBeenCalled();
            expect(result).toMatchObject({ section: "outside", buffer: [] });
        });
        test("tab", () => {
            const buffer = "name".split("");
            const result = handleTaskTransitionPropName({
                curr: "\t",
                ...withBuffer(buffer, rest),
            });
            expect(onCommit).not.toHaveBeenCalled();
            expect(result).toMatchObject({ section: "outside", buffer: [] });
        });
        test("newline", () => {
            const buffer = "name".split("");
            const result = handleTaskTransitionPropName({
                curr: "\n",
                ...withBuffer(buffer, rest),
            });
            expect(onCommit).not.toHaveBeenCalled();
            expect(result).toMatchObject({ section: "outside", buffer: [] });
        });
        test("other alphabets", () => {
            const buffer = "name".split("");
            const result = handleTaskTransitionPropName({
                curr: "你",
                ...withBuffer(buffer, rest),
            });
            expect(onCommit).not.toHaveBeenCalled();
            expect(result).toMatchObject({ section: "outside", buffer: [] });
        });
        test("emojis", () => {
            const buffer = "name".split("");
            const result = handleTaskTransitionPropName({
                curr: "👋",
                ...withBuffer(buffer, rest),
            });
            expect(onCommit).not.toHaveBeenCalled();
            expect(result).toMatchObject({ section: "outside", buffer: [] });
        });
        test("symbols", () => {
            const buffer = "name".split("");
            const result = handleTaskTransitionPropName({
                curr: "!",
                ...withBuffer(buffer, rest),
            });
            expect(onCommit).not.toHaveBeenCalled();
            expect(result).toMatchObject({ section: "outside", buffer: [] });
        });
        test("slash", () => {
            const buffer = "name".split("");
            const result = handleTaskTransitionPropName({
                curr: "/",
                ...withBuffer(buffer, rest),
            });
            expect(onCommit).not.toHaveBeenCalled();
            expect(result).toMatchObject({ section: "outside", buffer: [] });
        });
    });

    describe("commits", () => {
        test("equals sign", () => {
            const buffer = "name".split("");
            const result = handleTaskTransitionPropName({
                curr: "=",
                ...withBuffer(buffer, rest),
            });
            expect(onCommit).toHaveBeenCalledWith("propName", buffer.join(""));
            expect(result).toMatchObject({ section: "propValue", buffer: [] });
        });
    });
});

describe("handleTaskTransitionPropValue", () => {
    const section = "propValue";
    let onCommit, onComplete, onCancel, onStart;
    let rest;
    beforeEach(() => {
        onCommit = jest.fn();
        onComplete = jest.fn();
        onCancel = jest.fn();
        onStart = jest.fn();
        rest = { onCommit, onComplete, onCancel, onStart, hasOpenBracket: false };
    });

    describe("continues buffering", () => {
        describe("empty buffer", () => {
            test("single quote", () => {
                const buffer = "".split("");
                const result = handleTaskTransitionPropValue({
                    curr: "'",
                    ...withBuffer(buffer, rest),
                });
                expect(onCommit).not.toHaveBeenCalled();
                expect(result).toMatchObject({ section, buffer: [...buffer, "'"] });
            });
            test("double quote", () => {
                const buffer = "".split("");
                const result = handleTaskTransitionPropValue({
                    curr: "\"",
                    ...withBuffer(buffer, rest),
                });
                expect(onCommit).not.toHaveBeenCalled();
                expect(result).toMatchObject({ section, buffer: [...buffer, "\""] });
            });
        });
        describe("number, or might be a number with continued buffering", () => {
            test("first digit", () => {
                const buffer = "".split("");
                const result = handleTaskTransitionPropValue({
                    curr: "1",
                    ...withBuffer(buffer, rest),
                });
                expect(onCommit).not.toHaveBeenCalled();
                expect(result).toMatchObject({ section, buffer: [...buffer, "1"] });
            });
            test("first negative sign", () => {
                const buffer = "".split("");
                const result = handleTaskTransitionPropValue({
                    curr: "-",
                    ...withBuffer(buffer, rest),
                });
                expect(onCommit).not.toHaveBeenCalled();
                expect(result).toMatchObject({ section, buffer: [...buffer, "-"] });
            });
            test("first digit after negative sign", () => {
                const buffer = "-".split("");
                const result = handleTaskTransitionPropValue({
                    curr: "9",
                    ...withBuffer(buffer, rest),
                });
                expect(onCommit).not.toHaveBeenCalled();
                expect(result).toMatchObject({ section, buffer: [...buffer, "9"] });
            });
            test("first period with empty buffer", () => {
                const buffer = "".split("");
                const result = handleTaskTransitionPropValue({
                    curr: ".",
                    ...withBuffer(buffer, rest),
                });
                expect(onCommit).not.toHaveBeenCalled();
                expect(result).toMatchObject({ section, buffer: [...buffer, "."] });
            });
            test("first period after positive number", () => {
                const buffer = "3".split("");
                const result = handleTaskTransitionPropValue({
                    curr: ".",
                    ...withBuffer(buffer, rest),
                });
                expect(onCommit).not.toHaveBeenCalled();
                expect(result).toMatchObject({ section, buffer: [...buffer, "."] });
            });
            test("first period after negative number", () => {
                const buffer = "-3".split("");
                const result = handleTaskTransitionPropValue({
                    curr: ".",
                    ...withBuffer(buffer, rest),
                });
                expect(onCommit).not.toHaveBeenCalled();
                expect(result).toMatchObject({ section, buffer: [...buffer, "."] });
            });
        });
        describe("null, or might be null with continued buffering", () => {
            test("first n", () => {
                const buffer = "".split("");
                const result = handleTaskTransitionPropValue({
                    curr: "n",
                    ...withBuffer(buffer, rest),
                });
                expect(onCommit).not.toHaveBeenCalled();
                expect(result).toMatchObject({ section, buffer: [...buffer, "n"] });
            });
            test("first u", () => {
                const buffer = "n".split("");
                const result = handleTaskTransitionPropValue({
                    curr: "u",
                    ...withBuffer(buffer, rest),
                });
                expect(onCommit).not.toHaveBeenCalled();
                expect(result).toMatchObject({ section, buffer: [...buffer, "u"] });
            });
            test("first l", () => {
                const buffer = "nu".split("");
                const result = handleTaskTransitionPropValue({
                    curr: "l",
                    ...withBuffer(buffer, rest),
                });
                expect(onCommit).not.toHaveBeenCalled();
                expect(result).toMatchObject({ section, buffer: [...buffer, "l"] });
            });
            // See "cancels" and "commits" sections for other null tests
        });
        describe("boolean, or might be boolean with continued buffering", () => {
            // True
            test("first t", () => {
                const buffer = "".split("");
                const result = handleTaskTransitionPropValue({
                    curr: "t",
                    ...withBuffer(buffer, rest),
                });
                expect(onCommit).not.toHaveBeenCalled();
                expect(result).toMatchObject({ section, buffer: [...buffer, "t"] });
            });
            test("first r", () => {
                const buffer = "t".split("");
                const result = handleTaskTransitionPropValue({
                    curr: "r",
                    ...withBuffer(buffer, rest),
                });
                expect(onCommit).not.toHaveBeenCalled();
                expect(result).toMatchObject({ section, buffer: [...buffer, "r"] });
            });
            test("first u", () => {
                const buffer = "tr".split("");
                const result = handleTaskTransitionPropValue({
                    curr: "u",
                    ...withBuffer(buffer, rest),
                });
                expect(onCommit).not.toHaveBeenCalled();
                expect(result).toMatchObject({ section, buffer: [...buffer, "u"] });
            });
            // False
            test("first f", () => {
                const buffer = "".split("");
                const result = handleTaskTransitionPropValue({
                    curr: "f",
                    ...withBuffer(buffer, rest),
                });
                expect(onCommit).not.toHaveBeenCalled();
                expect(result).toMatchObject({ section, buffer: [...buffer, "f"] });
            });
            test("first a", () => {
                const buffer = "f".split("");
                const result = handleTaskTransitionPropValue({
                    curr: "a",
                    ...withBuffer(buffer, rest),
                });
                expect(onCommit).not.toHaveBeenCalled();
                expect(result).toMatchObject({ section, buffer: [...buffer, "a"] });
            });
            test("first l", () => {
                const buffer = "fa".split("");
                const result = handleTaskTransitionPropValue({
                    curr: "l",
                    ...withBuffer(buffer, rest),
                });
                expect(onCommit).not.toHaveBeenCalled();
                expect(result).toMatchObject({ section, buffer: [...buffer, "l"] });
            });
            test("first s", () => {
                const buffer = "fal".split("");
                const result = handleTaskTransitionPropValue({
                    curr: "s",
                    ...withBuffer(buffer, rest),
                });
                expect(onCommit).not.toHaveBeenCalled();
                expect(result).toMatchObject({ section, buffer: [...buffer, "s"] });
            });
            // See "cancels" and "commits" sections for other boolean tests
        });
        describe("valid character in quote", () => {
            describe("single quote start", () => {
                test("letter", () => {
                    const buffer = "'".split("");
                    const result = handleTaskTransitionPropValue({
                        curr: "a",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCommit).not.toHaveBeenCalled();
                    expect(result).toMatchObject({ section, buffer: [...buffer, "a"] });
                });
                test("letter with other text already in quote", () => {
                    const buffer = "'fdsaf".split("");
                    const result = handleTaskTransitionPropValue({
                        curr: "a",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCommit).not.toHaveBeenCalled();
                    expect(result).toMatchObject({ section, buffer: [...buffer, "a"] });
                });
                test("other alphabets", () => {
                    const buffer = "'你".split("");
                    const result = handleTaskTransitionPropValue({
                        curr: "你",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCommit).not.toHaveBeenCalled();
                    expect(result).toMatchObject({ section, buffer: [...buffer, "你"] });
                });
                test("emoji", () => {
                    const buffer = "'👋".split("");
                    const result = handleTaskTransitionPropValue({
                        curr: "👋",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCommit).not.toHaveBeenCalled();
                    expect(result).toMatchObject({ section, buffer: [...buffer, "👋"] });
                });
                test("symbol", () => {
                    const buffer = "'!@#".split("");
                    const result = handleTaskTransitionPropValue({
                        curr: "!",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCommit).not.toHaveBeenCalled();
                    expect(result).toMatchObject({ section, buffer: [...buffer, "!"] });
                });
                test("slash", () => {
                    const buffer = "'/".split("");
                    const result = handleTaskTransitionPropValue({
                        curr: "/",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCommit).not.toHaveBeenCalled();
                    expect(result).toMatchObject({ section, buffer: [...buffer, "/"] });
                });
                describe("whitespace", () => {
                    test("space", () => {
                        const buffer = "'".split("");
                        const result = handleTaskTransitionPropValue({
                            curr: " ",
                            ...withBuffer(buffer, rest),
                        });
                        expect(onCommit).not.toHaveBeenCalled();
                        expect(result).toMatchObject({ section, buffer: [...buffer, " "] });
                    });
                    test("tab", () => {
                        const buffer = "'".split("");
                        const result = handleTaskTransitionPropValue({
                            curr: "\t",
                            ...withBuffer(buffer, rest),
                        });
                        expect(onCommit).not.toHaveBeenCalled();
                        expect(result).toMatchObject({ section, buffer: [...buffer, "\t"] });
                    });
                    test("newline", () => {
                        const buffer = "'".split("");
                        const result = handleTaskTransitionPropValue({
                            curr: "\n",
                            ...withBuffer(buffer, rest),
                        });
                        expect(onCommit).not.toHaveBeenCalled();
                        expect(result).toMatchObject({ section, buffer: [...buffer, "\n"] });
                    });
                });
                test("double quote", () => {
                    const buffer = "'".split("");
                    const result = handleTaskTransitionPropValue({
                        curr: "\"",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCommit).not.toHaveBeenCalled();
                    expect(result).toMatchObject({ section, buffer: [...buffer, "\""] });
                });
                test("escape character", () => {
                    const buffer = "'".split("");
                    const result = handleTaskTransitionPropValue({
                        curr: "\\",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCommit).not.toHaveBeenCalled();
                    expect(result).toMatchObject({ section, buffer: [...buffer, "\\"] });
                });
                test("escaped single quote", () => {
                    const buffer = "'\\".split("");
                    const result = handleTaskTransitionPropValue({
                        curr: "'",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCommit).not.toHaveBeenCalled();
                    expect(result).toMatchObject({ section, buffer: [...buffer, "'"] });
                });
                test("escaped double quote", () => {
                    const buffer = "'\\".split("");
                    const result = handleTaskTransitionPropValue({
                        curr: "\"",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCommit).not.toHaveBeenCalled();
                    expect(result).toMatchObject({ section, buffer: [...buffer, "\""] });
                });
                test("odd number of escape characters", () => {
                    const buffer = "'\\\\\\".split("");
                    const result = handleTaskTransitionPropValue({
                        curr: "'",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCommit).not.toHaveBeenCalled();
                    expect(result).toMatchObject({ section: "propValue", buffer: [...buffer, "'"] });
                });
            });
            describe("double quote start", () => {
                test("letter", () => {
                    const buffer = "\"".split("");
                    const result = handleTaskTransitionPropValue({
                        curr: "a",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCommit).not.toHaveBeenCalled();
                    expect(result).toMatchObject({ section, buffer: [...buffer, "a"] });
                });
                test("letter with other text already in quote", () => {
                    const buffer = "\"fdsaf".split("");
                    const result = handleTaskTransitionPropValue({
                        curr: "a",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCommit).not.toHaveBeenCalled();
                    expect(result).toMatchObject({ section, buffer: [...buffer, "a"] });
                });
                test("other alphabets", () => {
                    const buffer = "\"你".split("");
                    const result = handleTaskTransitionPropValue({
                        curr: "你",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCommit).not.toHaveBeenCalled();
                    expect(result).toMatchObject({ section, buffer: [...buffer, "你"] });
                });
                test("emoji", () => {
                    const buffer = "\"👋".split("");
                    const result = handleTaskTransitionPropValue({
                        curr: "👋",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCommit).not.toHaveBeenCalled();
                    expect(result).toMatchObject({ section, buffer: [...buffer, "👋"] });
                });
                test("single quote", () => {
                    const buffer = "\"".split("");
                    const result = handleTaskTransitionPropValue({
                        curr: "'",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCommit).not.toHaveBeenCalled();
                    expect(result).toMatchObject({ section, buffer: [...buffer, "'"] });
                });
                test("escape character", () => {
                    const buffer = "\"".split("");
                    const result = handleTaskTransitionPropValue({
                        curr: "\\",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCommit).not.toHaveBeenCalled();
                    expect(result).toMatchObject({ section, buffer: [...buffer, "\\"] });
                });
                test("escaped single quote", () => {
                    const buffer = "\"\\".split("");
                    const result = handleTaskTransitionPropValue({
                        curr: "'",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCommit).not.toHaveBeenCalled();
                    expect(result).toMatchObject({ section, buffer: [...buffer, "'"] });
                });
                test("escaped double quote", () => {
                    const buffer = "\"\\".split("");
                    const result = handleTaskTransitionPropValue({
                        curr: "\"",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCommit).not.toHaveBeenCalled();
                    expect(result).toMatchObject({ section, buffer: [...buffer, "\""] });
                });
                test("odd number of escape characters", () => {
                    const buffer = "'\\\\\\".split("");
                    const result = handleTaskTransitionPropValue({
                        curr: "\"",
                        ...withBuffer(buffer, rest),
                    });
                    expect(onCommit).not.toHaveBeenCalled();
                    expect(result).toMatchObject({ section: "propValue", buffer: [...buffer, "\""] });
                });
            });
        });
    });

    describe("cancels", () => {
        describe("invalid number", () => {
            test("double negative", () => {
                const buffer = "-".split("");
                const result = handleTaskTransitionPropValue({
                    curr: "-",
                    ...withBuffer(buffer, rest),
                });
                expect(onCommit).not.toHaveBeenCalled();
                expect(result).toMatchObject({ section: "outside", buffer: [] });
            });
            test("negative after number", () => {
                const buffer = "1".split("");
                const result = handleTaskTransitionPropValue({
                    curr: "-",
                    ...withBuffer(buffer, rest),
                });
                expect(onCommit).not.toHaveBeenCalled();
                expect(result).toMatchObject({ section: "outside", buffer: [] });
            });
            test("double period, sequential", () => {
                const buffer = "3.".split("");
                const result = handleTaskTransitionPropValue({
                    curr: ".",
                    ...withBuffer(buffer, rest),
                });
                expect(onCommit).not.toHaveBeenCalled();
                expect(result).toMatchObject({ section: "outside", buffer: [] });
            });
            test("double period, number inbetween", () => {
                const buffer = "1.2".split("");
                const result = handleTaskTransitionPropValue({
                    curr: ".",
                    ...withBuffer(buffer, rest),
                });
                expect(onCommit).not.toHaveBeenCalled();
                expect(result).toMatchObject({ section: "outside", buffer: [] });
            });
            test("letter after number", () => {
                const buffer = "1".split("");
                const result = handleTaskTransitionPropValue({
                    curr: "a",
                    ...withBuffer(buffer, rest),
                });
                expect(onCommit).not.toHaveBeenCalled();
                expect(result).toMatchObject({ section: "outside", buffer: [] });
            });
            test("other alphabets after number", () => {
                const buffer = "1".split("");
                const result = handleTaskTransitionPropValue({
                    curr: "你",
                    ...withBuffer(buffer, rest),
                });
                expect(onCommit).not.toHaveBeenCalled();
                expect(result).toMatchObject({ section: "outside", buffer: [] });
            });
            test("emoji after number", () => {
                const buffer = "1".split("");
                const result = handleTaskTransitionPropValue({
                    curr: "👋",
                    ...withBuffer(buffer, rest),
                });
                expect(onCommit).not.toHaveBeenCalled();
                expect(result).toMatchObject({ section: "outside", buffer: [] });
            });
            test("symbol after number", () => {
                const buffer = "1".split("");
                const result = handleTaskTransitionPropValue({
                    curr: "!",
                    ...withBuffer(buffer, rest),
                });
                expect(onCommit).not.toHaveBeenCalled();
                expect(result).toMatchObject({ section: "outside", buffer: [] });
            });
        });
        describe("invalid null", () => {
            test("double n", () => {
                const buffer = "n".split("");
                const result = handleTaskTransitionPropValue({
                    curr: "n",
                    ...withBuffer(buffer, rest),
                });
                expect(onCommit).not.toHaveBeenCalled();
                expect(result).toMatchObject({ section: "outside", buffer: [] });
            });
            test("double u", () => {
                const buffer = "nu".split("");
                const result = handleTaskTransitionPropValue({
                    curr: "u",
                    ...withBuffer(buffer, rest),
                });
                expect(onCommit).not.toHaveBeenCalled();
                expect(result).toMatchObject({ section: "outside", buffer: [] });
            });
            test("capital N", () => {
                const buffer = "N".split("");
                const result = handleTaskTransitionPropValue({
                    curr: "N",
                    ...withBuffer(buffer, rest),
                });
                expect(onCommit).not.toHaveBeenCalled();
                expect(result).toMatchObject({ section: "outside", buffer: [] });
            });
        });
        describe("invalid boolean", () => {
            test("double f", () => {
                const buffer = "f".split("");
                const result = handleTaskTransitionPropValue({
                    curr: "f",
                    ...withBuffer(buffer, rest),
                });
                expect(onCommit).not.toHaveBeenCalled();
                expect(result).toMatchObject({ section: "outside", buffer: [] });
            });
            test("double t", () => {
                const buffer = "t".split("");
                const result = handleTaskTransitionPropValue({
                    curr: "t",
                    ...withBuffer(buffer, rest),
                });
                expect(onCommit).not.toHaveBeenCalled();
                expect(result).toMatchObject({ section: "outside", buffer: [] });
            });
            test("capital F", () => {
                const buffer = "F".split("");
                const result = handleTaskTransitionPropValue({
                    curr: "F",
                    ...withBuffer(buffer, rest),
                });
                expect(onCommit).not.toHaveBeenCalled();
                expect(result).toMatchObject({ section: "outside", buffer: [] });
            });
        });
        describe("whitespace before quote", () => {
            test("space", () => {
                const buffer = "".split("");
                const result = handleTaskTransitionPropValue({
                    curr: " ",
                    ...withBuffer(buffer, rest),
                });
                expect(onCommit).not.toHaveBeenCalled();
                expect(result).toMatchObject({ section: "outside", buffer: [] });
            });
            test("tab", () => {
                const buffer = "".split("");
                const result = handleTaskTransitionPropValue({
                    curr: "\t",
                    ...withBuffer(buffer, rest),
                });
                expect(onCommit).not.toHaveBeenCalled();
                expect(result).toMatchObject({ section: "outside", buffer: [] });
            });
            test("newline", () => {
                const buffer = "".split("");
                const result = handleTaskTransitionPropValue({
                    curr: "\n",
                    ...withBuffer(buffer, rest),
                });
                expect(onCommit).not.toHaveBeenCalled();
                expect(result).toMatchObject({ section: "outside", buffer: [] });
            });
        });
        test("letter before quote (that can't be a possible null/boolean value)", () => {
            const buffer = "".split("");
            const result = handleTaskTransitionPropValue({
                curr: "a",
                ...withBuffer(buffer, rest),
            });
            expect(onCommit).not.toHaveBeenCalled();
            expect(result).toMatchObject({ section: "outside", buffer: [] });
        });
        test("other alphabets before quote", () => {
            const buffer = "".split("");
            const result = handleTaskTransitionPropValue({
                curr: "你",
                ...withBuffer(buffer, rest),
            });
            expect(onCommit).not.toHaveBeenCalled();
            expect(result).toMatchObject({ section: "outside", buffer: [] });
        });
        test("emoji before quote", () => {
            const buffer = "".split("");
            const result = handleTaskTransitionPropValue({
                curr: "👋",
                ...withBuffer(buffer, rest),
            });
            expect(onCommit).not.toHaveBeenCalled();
            expect(result).toMatchObject({ section: "outside", buffer: [] });
        });
        test("symbol before quote", () => {
            const buffer = "".split("");
            const result = handleTaskTransitionPropValue({
                curr: "!",
                ...withBuffer(buffer, rest),
            });
            expect(onCommit).not.toHaveBeenCalled();
            expect(result).toMatchObject({ section: "outside", buffer: [] });
        });
        test("slash before quote", () => {
            const buffer = "".split("");
            const result = handleTaskTransitionPropValue({
                curr: "/",
                ...withBuffer(buffer, rest),
            });
            expect(onCommit).not.toHaveBeenCalled();
            expect(result).toMatchObject({ section: "outside", buffer: [] });
        });
    });

    describe("commits", () => {
        describe("single quote", () => {
            test("empty quote (i.e. only single quote in buffer)", () => {
                const buffer = "'".split("");
                const result = handleTaskTransitionPropValue({
                    curr: "'",
                    ...withBuffer(buffer, rest),
                });
                expect(onCommit).toHaveBeenCalledWith("propValue", "");
                expect(result).toMatchObject({ section: "propName", buffer: [] });
            });
            test("quote with text", () => {
                const buffer = "'test".split("");
                const result = handleTaskTransitionPropValue({
                    curr: "'",
                    ...withBuffer(buffer, rest),
                });
                expect(onCommit).toHaveBeenCalledWith("propValue", "test");
                expect(result).toMatchObject({ section: "propName", buffer: [] });
            });
            test("quote that looks like a number", () => {
                const buffer = "'123".split("");
                const result = handleTaskTransitionPropValue({
                    curr: "'",
                    ...withBuffer(buffer, rest),
                });
                expect(onCommit).toHaveBeenCalledWith("propValue", "123");
                expect(result).toMatchObject({ section: "propName", buffer: [] });
            });
            test("ending quote with even number of escape characters", () => {
                const buffer = "'\\\\".split("");
                const result = handleTaskTransitionPropValue({
                    curr: "'",
                    ...withBuffer(buffer, rest),
                });
                expect(onCommit).toHaveBeenCalledWith("propValue", "\\\\");
                expect(result).toMatchObject({ section: "propName", buffer: [] });
            });
        });
        describe("double quote", () => {
            test("empty quote (i.e. only double quote in buffer)", () => {
                const buffer = "\"".split("");
                const result = handleTaskTransitionPropValue({
                    curr: "\"",
                    ...withBuffer(buffer, rest),
                });
                expect(onCommit).toHaveBeenCalledWith("propValue", "");
                expect(result).toMatchObject({ section: "propName", buffer: [] });
            });
            test("quote with text", () => {
                const buffer = "\"test".split("");
                const result = handleTaskTransitionPropValue({
                    curr: "\"",
                    ...withBuffer(buffer, rest),
                });
                expect(onCommit).toHaveBeenCalledWith("propValue", "test");
                expect(result).toMatchObject({ section: "propName", buffer: [] });
            });
            test("quote that looks like a number", () => {
                const buffer = "\"123".split("");
                const result = handleTaskTransitionPropValue({
                    curr: "\"",
                    ...withBuffer(buffer, rest),
                });
                expect(onCommit).toHaveBeenCalledWith("propValue", "123");
                expect(result).toMatchObject({ section: "propName", buffer: [] });
            });
            test("ending quote with even number of escape characters", () => {
                const buffer = "\"\\\\".split("");
                const result = handleTaskTransitionPropValue({
                    curr: "\"",
                    ...withBuffer(buffer, rest),
                });
                expect(onCommit).toHaveBeenCalledWith("propValue", "\\\\");
                expect(result).toMatchObject({ section: "propName", buffer: [] });
            });
        });
        describe("number property on whitespace", () => {
            test("space", () => {
                const buffer = "000123".split("");
                const result = handleTaskTransitionPropValue({
                    curr: " ",
                    ...withBuffer(buffer, rest),
                });
                expect(onCommit).toHaveBeenCalledWith("propValue", 123);
                expect(result).toMatchObject({ section: "propName", buffer: [] });
            });
            test("tab", () => {
                const buffer = "1.23".split("");
                const result = handleTaskTransitionPropValue({
                    curr: "\t",
                    ...withBuffer(buffer, rest),
                });
                expect(onCommit).toHaveBeenCalledWith("propValue", 1.23);
                expect(result).toMatchObject({ section: "propName", buffer: [] });
            });
            test("newline", () => {
                const buffer = "-123.".split("");
                const result = handleTaskTransitionPropValue({
                    curr: "\n",
                    ...withBuffer(buffer, rest),
                });
                expect(onCommit).toHaveBeenCalledWith("propValue", -123);
                expect(result).toMatchObject({ section: "outside", buffer: [] });
            });
        });
    });
});

function getStartEnd(str: string, sub: string): { start: number, end: number } {
    return {
        start: str.indexOf(sub),
        end: str.indexOf(sub) + sub.length, // Index after the last character
    };
}

describe("findCharWithLimit", () => {
    const testString1 = "[/command1]";
    const testString2 = "  [/command1]  ";
    const testString3 = "  [  /command1  ]  ";
    const testString4 = " \n[/command1]\n  ";
    const allTests = [testString1, testString2, testString3, testString4];

    for (const testString of allTests) {
        const openIndex = testString.indexOf("[");
        const closeIndex = testString.indexOf("]");
        test("finds \"[\" starting at beginning and going forward", () => {
            // @ts-ignore: expect-message
            expect(findCharWithLimit(0, true, "[", testString, 5), `input: ${testString}`).toBe(openIndex);
        });
        test("doesn't find \"[\" starting at end and going forward", () => {
            // @ts-ignore: expect-message
            expect(findCharWithLimit(testString.length - 1, true, "[", testString, 5), `input: ${testString}`).toBeNull();
        });
        test("finds \"]\" starting at end and going backward", () => {
            // @ts-ignore: expect-message
            expect(findCharWithLimit(testString.length - 1, false, "]", testString, 5), `input: ${testString}`).toBe(closeIndex);
        });
        test("doesn't find \"]\" starting at beginning and going backward", () => {
            // @ts-ignore: expect-message
            expect(findCharWithLimit(0, false, "]", testString, 5), `input: ${testString}`).toBeNull();
        });
        test("doesn't find \"[\" starting at middle and going forward - should already be past it", () => {
            // @ts-ignore: expect-message
            expect(findCharWithLimit(3, true, "[", testString, 5), `input: ${testString}`).toBeNull();
        });
    }
});

// Mock commandToTask
function commandToTask(command: string, action?: string | null): LlmTask {
    if (action) return `${command} ${action}` as LlmTask;
    return command as LlmTask;
}

/**
 * Helper function to simplify testing of `detectWrappedTasks`
 */
function detectWrappedTasksTester({
    start,
    delimiter,
    input,
    expected,
}: {
    start: string,
    delimiter: string | null,
    input: string,
    expected: {
        taskIndices: number[],
        wrapperStart: number,
        wrapperEnd: number,
    }[],
}) {
    const allCommands = extractTasksFromText(input, commandToTask);
    const detectedCommands = detectWrappedTasks({
        start,
        delimiter,
        commands: allCommands,
        messageString: input,
    });
    // @ts-ignore: expect-message
    expect(detectedCommands, `input: ${input}`).toEqual(expected);
}

describe("detectWrappedTasks", () => {
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
        describe("returns empty array when no commands are present", () => {
            test("test 1", () => {
                detectWrappedTasksTester({
                    input: "a/command",
                    expected: [],
                    ...wrapper,
                });
            });
            test("test 2", () => {
                detectWrappedTasksTester({
                    input: "/command🥴",
                    expected: [],
                    ...wrapper,
                });
            });
        });
        describe("returns empty array when commands are not wrapped", () => {
            test("test 1", () => {
                detectWrappedTasksTester({
                    input: "/command1 /command2",
                    expected: [],
                    ...wrapper,
                });
            });
            test("test 2", () => {
                detectWrappedTasksTester({
                    input: `${start.slice(0, -1)}: [/command1]`,
                    expected: [],
                    ...wrapper,
                });
            });
            test("test 3", () => {
                detectWrappedTasksTester({
                    input: `${start}: [/command1`,
                    expected: [],
                    ...wrapper,
                });
            });
            test("test 4", () => {
                detectWrappedTasksTester({
                    input: `${start}: [/command1\n]`,
                    expected: [],
                    ...wrapper,
                });
            });
            test("test 5", () => {
                detectWrappedTasksTester({
                    input: `${start}: [/command1${delimiter}\n/command2]`,
                    expected: [],
                    ...wrapper,
                });
            });
            test("test 6", () => {
                detectWrappedTasksTester({
                    input: `${start}:[/command1 ${start}:]`,
                    expected: [],
                    ...wrapper,
                });
            });
        });
        describe("returns correct indices for a single wrapped command", () => {
            test("test 1", () => {
                const input = `${start}:[/command1]`;
                detectWrappedTasksTester({
                    input,
                    expected: [{
                        taskIndices: [0],
                        wrapperStart: 0,
                        wrapperEnd: input.length - 1,
                    }],
                    ...wrapper,
                });
            });
            test("test 2", () => {
                const input = `boop ${start}: [/command1]`;
                detectWrappedTasksTester({
                    input,
                    expected: [{
                        taskIndices: [0],
                        wrapperStart: 5,
                        wrapperEnd: input.length - 1,
                    }],
                    ...wrapper,
                });
            });
            test("test 3", () => {
                const input = `/firstCommand hello ${start}:[/command1]`;
                detectWrappedTasksTester({
                    input,
                    expected: [{
                        taskIndices: [1],
                        wrapperStart: "/firstCommand hello ".length,
                        wrapperEnd: input.length - 1,
                    }],
                    ...wrapper,
                });
            });
            test("test 4", () => {
                const input = `/firstCommand hello ${start}: [/command1]}`;
                detectWrappedTasksTester({
                    input,
                    expected: [{
                        taskIndices: [1],
                        wrapperStart: "/firstCommand hello ".length,
                        wrapperEnd: input.length - 2,
                    }],
                    ...wrapper,
                });
            });
            test("test 5", () => {
                const input = `/firstCommand hello ${start}: [  /command1  ]`;
                detectWrappedTasksTester({
                    input,
                    expected: [{
                        taskIndices: [1],
                        wrapperStart: "/firstCommand hello ".length,
                        wrapperEnd: input.length - 1,
                    }],
                    ...wrapper,
                });
            });
            test("test 6", () => {
                const input = `/firstCommand hello name="hi" ${start}: [/command1]`;
                detectWrappedTasksTester({
                    input,
                    expected: [{
                        taskIndices: [1],
                        wrapperStart: "/firstCommand hello name=\"hi\" ".length,
                        wrapperEnd: input.length - 1,
                    }],
                    ...wrapper,
                });
            });
            test("test 7", () => {
                const input = `${start}: ${start}: [/command1 hello]`;
                detectWrappedTasksTester({
                    input,
                    expected: [{
                        taskIndices: [0],
                        wrapperStart: `${start}: `.length, // First ${start} is ignored
                        wrapperEnd: input.length - 1,
                    }],
                    ...wrapper,
                });
            });
            test("test 8", () => {
                const input = `/hi there ${start}: [/command1 hello name="hi" value=123 thing='hi']`;
                detectWrappedTasksTester({
                    input,
                    expected: [{
                        taskIndices: [1],
                        wrapperStart: "/hi there ".length,
                        wrapperEnd: input.length - 1,
                    }],
                    ...wrapper,
                });
            });
            test("test 9", () => {
                const input = `${start}:\n[/command1] `;
                detectWrappedTasksTester({
                    input,
                    expected: [{
                        taskIndices: [0],
                        wrapperStart: 0,
                        wrapperEnd: input.length - 2, // Excludes space at end
                    }],
                    ...wrapper,
                });
            });
            test("test 10", () => {
                const input = `/task boop ${start}: \n [/command1]`;
                detectWrappedTasksTester({
                    input,
                    expected: [{
                        taskIndices: [1],
                        wrapperStart: "/task boop ".length,
                        wrapperEnd: input.length - 1,
                    }],
                    ...wrapper,
                });
            });
        });
        describe("returns correct indices for multiple wrapped commands", () => {
            test("test 1", () => {
                const input = `${start}:[/command1${delimiter ?? ""} /command2]`;
                detectWrappedTasksTester({
                    input,
                    expected: delimiter ? [{
                        taskIndices: [0, 1],
                        wrapperStart: 0,
                        wrapperEnd: input.length - 1,
                    }] : [],
                    ...wrapper,
                });
            });
            test("test 2", () => {
                const input = `${start}: [/command1 ${delimiter ?? ""} /command2 ]`;
                detectWrappedTasksTester({
                    input,
                    expected: delimiter ? [{
                        taskIndices: [0, 1],
                        wrapperStart: 0,
                        wrapperEnd: input.length - 1,
                    }] : [],
                    ...wrapper,
                });
            });
            test("test 3", () => {
                const input = `${start}: [/command1 ${delimiter ?? ""} /command2 action name='hi']`;
                detectWrappedTasksTester({
                    input,
                    expected: delimiter ? [{
                        taskIndices: [0, 1],
                        wrapperStart: 0,
                        wrapperEnd: input.length - 1,
                    }] : [],
                    ...wrapper,
                });
            });
        });
        describe("handles property value trickery", () => {
            test("test 1", () => {
                detectWrappedTasksTester({
                    input: `/command1 action name='${start}: [/command2]'`,
                    expected: [],
                    ...wrapper,
                });
            });
            test("test 2", () => {
                detectWrappedTasksTester({
                    input: `/command1 action name='${start}:' [/command2]`,
                    expected: [],
                    ...wrapper,
                });
            });
        });
        describe("real world examples", () => {
            test("test 1", () => {
                const input = `${start}: [ /bot find]`; // Has a space before command for some reason
                detectWrappedTasksTester({
                    input,
                    expected: [{
                        taskIndices: [0],
                        wrapperStart: 0,
                        wrapperEnd: input.length - 1,
                    }],
                    ...wrapper,
                });

            });
        });
    }
});

function enCommandToTask(command: string, action?: string | null) {
    let result: string;
    if (action) result = `${pascalCase(command)}${pascalCase(action)}`;
    else result = pascalCase(command);
    if (Object.keys(LlmTask).includes(result)) return result as LlmTask;
    return null;
}

/**
 * Helper function to simplify testing of `extractTasks`
 */
function extractTasksFromTextTester({
    input,
    expected,
}: {
    input: string,
    expected: (Omit<MaybeLlmTaskInfo, "start" | "end"> & { match: string })[],
}) {
    const commands = extractTasksFromText(input, commandToTask);
    const expectedCommands = expected.map(({ properties, task, match }) => ({
        task,
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
}

describe("extractTasksFromText", () => {
    describe("ignores non-command slashes", () => {
        test("test 1", () => {
            extractTasksFromTextTester({
                input: "a/command",
                expected: [],
            });
        });
        test("test 2", () => {
            extractTasksFromTextTester({
                input: "1/3",
                expected: [],
            });
        });
        test("test 3", () => {
            extractTasksFromTextTester({
                input: "/boop.",
                expected: [],
            });
        });
        test("test 4", () => {
            extractTasksFromTextTester({
                input: "/boop!",
                expected: [],
            });
        });
        test("test 5", () => {
            extractTasksFromTextTester({
                input: "/boop你",
                expected: [],
            });
        });
        test("test 6", () => {
            extractTasksFromTextTester({
                input: "/boop👋",
                expected: [],
            });
        });
        test("test 7", () => {
            extractTasksFromTextTester({
                input: "/boop/",
                expected: [],
            });
        });
        test("test 8", () => {
            extractTasksFromTextTester({
                input: "/boop\\",
                expected: [],
            });
        });
        test("test 9", () => {
            extractTasksFromTextTester({
                input: "/boop=",
                expected: [],
            });
        });
        test("test 10", () => {
            extractTasksFromTextTester({
                input: "/boop-",
                expected: [],
            });
        });
        test("test 11", () => {
            extractTasksFromTextTester({
                input: "/boop.",
                expected: [],
            });
        });
        test("test 12", () => {
            extractTasksFromTextTester({
                input: "//boop",
                expected: [],
            });
        });
        test("test 13", () => {
            extractTasksFromTextTester({
                input: "//bippity /boppity! /boop💃 /realCommand",
                expected: [{
                    task: commandToTask("realCommand"),
                    properties: {},
                    match: "/realCommand",
                }],
            });
        });
    });

    describe("code blocks", () => {
        describe("ignores commands in code blocks", () => {
            test("test 1", () => {
                extractTasksFromTextTester({
                    input: "```/command```",
                    expected: [],
                });
            });
            test("test 2", () => {
                extractTasksFromTextTester({
                    input: "here is some code:\n```/command action```\n",
                    expected: [],
                });
            });
            test("test 3", () => {
                extractTasksFromTextTester({
                    input: "here is some code:\n```bloop /command action\nother words```",
                    expected: [],
                });
            });
            test("test 4", () => {
                extractTasksFromTextTester({
                    input: "```command inside code block: /codeCommand action```command outside code block: /otherCommand action\n",
                    expected: [{
                        task: commandToTask("otherCommand", "action"),
                        properties: {},
                        match: "/otherCommand action",
                    }],
                });
            });
            test("test 5", () => {
                extractTasksFromTextTester({
                    input: "Single-tick code block: `/command action`",
                    expected: [],
                });
            });
            test("test 6", () => {
                extractTasksFromTextTester({
                    input: "<code>/command action</code>",
                    expected: [],
                });
            });
            test("test 7", () => {
                extractTasksFromTextTester({
                    input: "hello <code>\n/command action\n</code> there",
                    expected: [],
                });
            });
        });

        describe("doesn't ignore code-looking commands when there's property value trickery", () => {
            test("test 1", () => {
                extractTasksFromTextTester({
                    input: "/command1 action1 text='```/note find```'",
                    expected: [{
                        task: commandToTask("command1", "action1"),
                        properties: { text: "```/note find```" },
                        match: "/command1 action1 text='```/note find```'",
                    }],
                });
            });
            test("test 2", () => {
                extractTasksFromTextTester({
                    input: "/command1 action1 text='```' /note find text='```'",
                    expected: [{
                        task: commandToTask("command1", "action1"),
                        properties: { text: "```" },
                        match: "/command1 action1 text='```'",
                    }, {
                        task: commandToTask("note", "find"),
                        properties: { text: "```" },
                        match: "/note find text='```'",
                    }],
                });
            });
            test("test 3", () => {
                extractTasksFromTextTester({
                    input: "/command1 action1 text='<code>/note find</code>'",
                    expected: [{
                        task: commandToTask("command1", "action1"),
                        properties: { text: "<code>/note find</code>" },
                        match: "/command1 action1 text='<code>/note find</code>'",
                    }],
                });
            });
            test("test 4", () => {
                extractTasksFromTextTester({
                    input: "/command1 action1 text='<code>' /note find text='<code>'",
                    expected: [{
                        task: commandToTask("command1", "action1"),
                        properties: { text: "<code>" },
                        match: "/command1 action1 text='<code>'",
                    }, {
                        task: commandToTask("note", "find"),
                        properties: { text: "<code>" },
                        match: "/note find text='<code>'",
                    }],
                });
            });
        });

        test("doesn't ignore double-tick code blocks", () => {
            extractTasksFromTextTester({
                input: "Double-tick code block: `` /command action ``",
                expected: [{
                    task: commandToTask("command", "action"),
                    properties: {},
                    match: "/command action",
                }],
            });
        });

        test("doesn't ignore quadruple-tick code blocks", () => {
            extractTasksFromTextTester({
                input: "Quadruple-tick code block: ```` /command action ````",
                expected: [{
                    task: commandToTask("command", "action"),
                    properties: {},
                    match: "/command action",
                }],
            });
        });

        test("doesn't ignore quintuple-tick code blocks", () => {
            extractTasksFromTextTester({
                input: "Quintuple-tick code block: ````` /command action `````",
                expected: [{
                    task: commandToTask("command", "action"),
                    properties: {},
                    match: "/command action",
                }],
            });
        });

        test("doesn't ignore single-tick code block when newline is encountered", () => {
            extractTasksFromTextTester({
                input: "Invalid single-tick code block: `\n/command action `",
                expected: [{
                    task: commandToTask("command", "action"),
                    properties: {},
                    match: "/command action",
                }],
            });
        });
    });

    describe("extracts simple command without action or properties", () => {
        test("test 1", () => {
            extractTasksFromTextTester({
                input: "/command",
                expected: [{
                    task: commandToTask("command"),
                    properties: {},
                    match: "/command",
                }],
            });
        });
        test("test 2", () => {
            extractTasksFromTextTester({
                input: "  /command",
                expected: [{
                    task: commandToTask("command"),
                    properties: {},
                    match: "/command",
                }],
            });
        });
        test("test 3", () => {
            extractTasksFromTextTester({
                input: "/command  ",
                expected: [{
                    task: commandToTask("command"),
                    properties: {},
                    match: "/command",
                }],
            });
        });
        test("test 4", () => {
            extractTasksFromTextTester({
                input: "aasdf\n/command",
                expected: [{
                    task: commandToTask("command"),
                    properties: {},
                    match: "/command",
                }],
            });
        });
        test("test 5", () => {
            extractTasksFromTextTester({
                input: "/command\n",
                expected: [{
                    task: commandToTask("command"),
                    properties: {},
                    match: "/command",
                }],
            });
        });
        test("test 6", () => {
            extractTasksFromTextTester({
                input: "/command\t",
                expected: [{
                    task: commandToTask("command"),
                    properties: {},
                    match: "/command",
                }],
            });
        });
        test("test 7", () => {
            extractTasksFromTextTester({
                input: "/command invalidActionBecauseSymbol!",
                expected: [{
                    task: commandToTask("command"),
                    properties: {},
                    match: "/command",
                }],
            });
        });
        test("test 8", () => {
            extractTasksFromTextTester({
                input: "/command invalidActionBecauseLanguage你",
                expected: [{
                    task: commandToTask("command"),
                    properties: {},
                    match: "/command",
                }],
            });
        });
    });

    describe("extracts command with action", () => {
        test("test 1", () => {
            extractTasksFromTextTester({
                input: "/command action",
                expected: [{
                    task: commandToTask("command", "action"),
                    properties: {},
                    match: "/command action",
                }],
            });
        });
        test("test 2", () => {
            extractTasksFromTextTester({
                input: "/command action other words",
                expected: [{
                    task: commandToTask("command", "action"),
                    properties: {},
                    match: "/command action",
                }],
            });
        });
        test("test 3", () => {
            extractTasksFromTextTester({
                input: "/command action invalidProp= 'space after equals'",
                expected: [{
                    task: commandToTask("command", "action"),
                    properties: {},
                    match: "/command action",
                }],
            });
        });
        test("test 4", () => {
            extractTasksFromTextTester({
                input: "/command action\t",
                expected: [{
                    task: commandToTask("command", "action"),
                    properties: {},
                    match: "/command action",
                }],
            });
        });
        test("test 5", () => {
            extractTasksFromTextTester({
                input: "/command action\n",
                expected: [{
                    task: commandToTask("command", "action"),
                    properties: {},
                    match: "/command action",
                }],
            });
        });
        test("test 6", () => {
            extractTasksFromTextTester({
                input: "/command\taction",
                expected: [{
                    task: commandToTask("command", "action"),
                    properties: {},
                    match: "/command\taction",
                }],
            });
        });
    });

    describe("handles command with properties", () => {
        test("test 1", () => {
            extractTasksFromTextTester({
                input: "/command prop1=123 prop2='value' prop3=null",
                expected: [{
                    task: commandToTask("command"),
                    properties: { prop1: 123, prop2: "value", prop3: null },
                    match: "/command prop1=123 prop2='value' prop3=null",
                }],
            });
        });
        test("test 2", () => {
            extractTasksFromTextTester({
                input: "/command prop1=\"123\" prop2='value' prop3=\"null\"",
                expected: [{
                    task: commandToTask("command"),
                    properties: { prop1: "123", prop2: "value", prop3: "null" },
                    match: "/command prop1=\"123\" prop2='value' prop3=\"null\"",
                }],
            });
        });
        test("test 3", () => {
            extractTasksFromTextTester({
                input: "/command prop1=0.3 prop2='val\"ue' prop3=\"asdf\nfdsa\"",
                expected: [{
                    task: commandToTask("command"),
                    properties: { prop1: 0.3, prop2: "val\"ue", prop3: "asdf\nfdsa" },
                    match: "/command prop1=0.3 prop2='val\"ue' prop3=\"asdf\nfdsa\"",
                }],
            });
        });
        test("test 4", () => {
            extractTasksFromTextTester({
                input: "/command prop1=0.3\" prop2='value' prop3=null",
                expected: [{
                    task: commandToTask("command"),
                    properties: {},
                    match: "/command",
                }],
            });
        });
        test("test 5", () => {
            extractTasksFromTextTester({
                input: "/command prop1=-2.3 prop2=.2 prop3=\"one\\\"\"",
                expected: [{
                    task: commandToTask("command"),
                    properties: { prop1: -2.3, prop2: .2, prop3: "one\\\"" },
                    match: "/command prop1=-2.3 prop2=.2 prop3=\"one\\\"\"",
                }],
            });
        });
        test("test 6", () => {
            extractTasksFromTextTester({
                input: "/command prop1=123 prop2='value' prop3=null\"",
                expected: [{
                    task: commandToTask("command"),
                    properties: { prop1: 123, prop2: "value" },
                    match: "/command prop1=123 prop2='value'",
                }],
            });
        });
        test("test 7", () => {
            extractTasksFromTextTester({
                input: "/command\tprop1=123 prop2='value' prop3=null\n",
                expected: [{
                    task: commandToTask("command"),
                    properties: { prop1: 123, prop2: "value", prop3: null },
                    match: "/command\tprop1=123 prop2='value' prop3=null",
                }],
            });
        });
        test("test 8", () => {
            extractTasksFromTextTester({
                input: "/command prop1=123 prop2='value' prop3=null ",
                expected: [{
                    task: commandToTask("command"),
                    properties: { prop1: 123, prop2: "value", prop3: null },
                    match: "/command prop1=123 prop2='value' prop3=null",
                }],
            });
        });
        test("test 9", () => {
            extractTasksFromTextTester({
                input: "/command prop1=123 prop2='val\\'u\"e' prop3=null\t",
                expected: [{
                    task: commandToTask("command"),
                    properties: { prop1: 123, prop2: "val\\'u\"e", prop3: null },
                    match: "/command prop1=123 prop2='val\\'u\"e' prop3=null",
                }],
            });
        });
        test("test 10", () => {
            extractTasksFromTextTester({
                input: "/command prop1=123 prop2='value' prop3=null notaprop",
                expected: [{
                    task: commandToTask("command"),
                    properties: { prop1: 123, prop2: "value", prop3: null },
                    match: "/command prop1=123 prop2='value' prop3=null",
                }],
            });
        });
    });

    describe("handles wrapped commands", () => {
        describe("ideal format", () => {
            test("test 1", () => {
                extractTasksFromTextTester({
                    input: "suggested: [/command1]",
                    expected: [{
                        task: commandToTask("command1"),
                        properties: {},
                        match: "/command1",
                    }],
                });
            });
            test("test 2", () => {
                extractTasksFromTextTester({
                    input: "/command1 suggested: [/command2]",
                    expected: [{
                        task: commandToTask("command1"),
                        properties: {},
                        match: "/command1",
                    }, {
                        task: commandToTask("command2"),
                        properties: {},
                        match: "/command2",
                    }],
                });
            });
            test("test 3", () => {
                extractTasksFromTextTester({
                    input: "suggested: [/command1] recommended: [/command2]",
                    expected: [{
                        task: commandToTask("command1"),
                        properties: {},
                        match: "/command1",
                    }, {
                        task: commandToTask("command2"),
                        properties: {},
                        match: "/command2",
                    }],
                });
            });
            test("test 4", () => {
                extractTasksFromTextTester({
                    input: "suggested: [/command1 action]",
                    expected: [{
                        task: commandToTask("command1", "action"),
                        properties: {},
                        match: "/command1 action",
                    }],
                });
            });
            test("test 5", () => {
                extractTasksFromTextTester({
                    input: "suggested: [/command1 action, /command2 action2]",
                    expected: [{
                        task: commandToTask("command1", "action"),
                        properties: {},
                        match: "/command1 action",
                    }, {
                        task: commandToTask("command2", "action2"),
                        properties: {},
                        match: "/command2 action2",
                    }],
                });
            });
            test("test 6", () => {
                extractTasksFromTextTester({
                    input: "suggested: [/command1 action name='value' prop2=123 thing=\"asdf\"]",
                    expected: [{
                        task: commandToTask("command1", "action"),
                        properties: { name: "value", prop2: 123, thing: "asdf" },
                        match: "/command1 action name='value' prop2=123 thing=\"asdf\"",
                    }],
                });
            });
            test("test 7", () => {
                extractTasksFromTextTester({
                    input: "suggested: [/command1 action name='valu\"e' prop2=123 thing=\"asdf\", /command2 action2]",
                    expected: [{
                        task: commandToTask("command1", "action"),
                        properties: { name: "valu\"e", prop2: 123, thing: "asdf" },
                        match: "/command1 action name='valu\"e' prop2=123 thing=\"asdf\"",
                    }, {
                        task: commandToTask("command2", "action2"),
                        properties: {},
                        match: "/command2 action2",
                    }],
                });
            });
        });
        describe("non-ideal format (llms aren't perfect)", () => {
            describe("leading whitespace", () => {
                test("test 1", () => {
                    extractTasksFromTextTester({
                        input: "suggested: [ /command1]",
                        expected: [{
                            task: commandToTask("command1"),
                            properties: {},
                            match: "/command1",
                        }],
                    });
                });
                test("test 2", () => {
                    extractTasksFromTextTester({
                        input: "suggested: [  /command1]",
                        expected: [{
                            task: commandToTask("command1"),
                            properties: {},
                            match: "/command1",
                        }],
                    });
                });
                test("test 3", () => {
                    extractTasksFromTextTester({
                        input: "suggested: [ /command1 action]",
                        expected: [{
                            task: commandToTask("command1", "action"),
                            properties: {},
                            match: "/command1 action",
                        }],
                    });
                });
                test("test 4", () => {
                    extractTasksFromTextTester({
                        input: "suggested: [ /command1 action name='value' prop2=123 thing=\"asdf\"]",
                        expected: [{
                            task: commandToTask("command1", "action"),
                            properties: { name: "value", prop2: 123, thing: "asdf" },
                            match: "/command1 action name='value' prop2=123 thing=\"asdf\"",
                        }],
                    });
                });
                test("test 5", () => {
                    extractTasksFromTextTester({
                        input: "suggested: [ /command1 action name='valu\"e' prop2=123 thing=\"asdf\", /command2 action2]",
                        expected: [{
                            task: commandToTask("command1", "action"),
                            properties: { name: "valu\"e", prop2: 123, thing: "asdf" },
                            match: "/command1 action name='valu\"e' prop2=123 thing=\"asdf\"",
                        }, {
                            task: commandToTask("command2", "action2"),
                            properties: {},
                            match: "/command2 action2",
                        }],
                    });
                });
            });
            describe("trailing whitespace", () => {
                test("test 1", () => {
                    extractTasksFromTextTester({
                        input: "suggested: [/command1 ]",
                        expected: [{
                            task: commandToTask("command1"),
                            properties: {},
                            match: "/command1",
                        }],
                    });
                });
                test("test 2", () => {
                    extractTasksFromTextTester({
                        input: "suggested: [/command1  ]",
                        expected: [{
                            task: commandToTask("command1"),
                            properties: {},
                            match: "/command1",
                        }],
                    });
                });
                test("test 3", () => {
                    extractTasksFromTextTester({
                        input: "suggested: [/command1 action ]",
                        expected: [{
                            task: commandToTask("command1", "action"),
                            properties: {},
                            match: "/command1 action",
                        }],
                    });
                });
                test("test 4", () => {
                    extractTasksFromTextTester({
                        input: "suggested: [/command1 action name='value' prop2=123 thing=\"asdf\"] ",
                        expected: [{
                            task: commandToTask("command1", "action"),
                            properties: { name: "value", prop2: 123, thing: "asdf" },
                            match: "/command1 action name='value' prop2=123 thing=\"asdf\"",
                        }],
                    });
                });
                test("test 5", () => {
                    extractTasksFromTextTester({
                        input: "suggested: [/command1 action name='valu\"e' prop2=123 thing=\"asdf\", /command2 action2] ",
                        expected: [{
                            task: commandToTask("command1", "action"),
                            properties: { name: "valu\"e", prop2: 123, thing: "asdf" },
                            match: "/command1 action name='valu\"e' prop2=123 thing=\"asdf\"",
                        }, {
                            task: commandToTask("command2", "action2"),
                            properties: {},
                            match: "/command2 action2",
                        }],
                    });
                });
            });
            describe("leading and trailing whitespace", () => {
                test("test 1", () => {
                    extractTasksFromTextTester({
                        input: "suggested: [ /command1 ]",
                        expected: [{
                            task: commandToTask("command1"),
                            properties: {},
                            match: "/command1",
                        }],
                    });
                });
                test("test 2", () => {
                    extractTasksFromTextTester({
                        input: "suggested: [  /command1  ]",
                        expected: [{
                            task: commandToTask("command1"),
                            properties: {},
                            match: "/command1",
                        }],
                    });
                });
                test("test 3", () => {
                    extractTasksFromTextTester({
                        input: "suggested: [ /command1 action ]",
                        expected: [{
                            task: commandToTask("command1", "action"),
                            properties: {},
                            match: "/command1 action",
                        }],
                    });
                });
                test("test 4", () => {
                    extractTasksFromTextTester({
                        input: "suggested: [ /command1 action name='value' prop2=123 thing=\"asdf\"] ",
                        expected: [{
                            task: commandToTask("command1", "action"),
                            properties: { name: "value", prop2: 123, thing: "asdf" },
                            match: "/command1 action name='value' prop2=123 thing=\"asdf\"",
                        }],
                    });
                });
                test("test 5", () => {
                    extractTasksFromTextTester({
                        input: "suggested: [ /command1 action name='valu\"e' prop2=123 thing=\"asdf\", /command2 action2] ",
                        expected: [{
                            task: commandToTask("command1", "action"),
                            properties: { name: "valu\"e", prop2: 123, thing: "asdf" },
                            match: "/command1 action name='valu\"e' prop2=123 thing=\"asdf\"",
                        }, {
                            task: commandToTask("command2", "action2"),
                            properties: {},
                            match: "/command2 action2",
                        }],
                    });
                });
            });
            describe("messiness before wrapper", () => {
                test("test 1", () => {
                    extractTasksFromTextTester({
                        input: "fdksaf; [] fdsafsdfds[ fdks;lfadksaf suggested: [/command1]",
                        expected: [{
                            task: commandToTask("command1"),
                            properties: {},
                            match: "/command1",
                        }],
                    });
                });
            });
        });
        describe("invalid formats that might trick our system", () => {
            test("non-whitespace after opening bracket", () => {
                extractTasksFromTextTester({
                    input: "suggested: [. /command1]",
                    expected: [],
                });
            });
            test("newline after opening bracket", () => {
                extractTasksFromTextTester({
                    input: "suggested: [\n/command1]",
                    expected: [],
                });
            });
        });
    });

    describe("whitespace", () => {
        test("handles newline properly", () => {
            extractTasksFromTextTester({
                input: "/command1\n/command2",
                expected: [{
                    task: commandToTask("command1"),
                    properties: {},
                    match: "/command1",
                }, {
                    task: commandToTask("command2"),
                    properties: {},
                    match: "/command2",
                }],
            });
        });

        test("handles space properly", () => {
            extractTasksFromTextTester({
                input: "/command1 /command2",
                expected: [{
                    task: commandToTask("command1"),
                    properties: {},
                    match: "/command1",
                }, {
                    task: commandToTask("command2"),
                    properties: {},
                    match: "/command2",
                }],
            });
        });

        test("handles tab properly", () => {
            extractTasksFromTextTester({
                input: "/command1\t/command2",
                expected: [{
                    task: commandToTask("command1"),
                    properties: {},
                    match: "/command1",
                }, {
                    task: commandToTask("command2"),
                    properties: {},
                    match: "/command2",
                }],
            });
        });
    });

    test("handles complex scenario with multiple commands and properties - test 1", () => {
        extractTasksFromTextTester({
            input: "/cmd1  prop1=123 /cmd2 action2 \tprop2='text' prop3=4.56\n/cmd3 prop4=null \nprop5='invalid because newline",
            expected: [{
                task: commandToTask("cmd1"),
                properties: { prop1: 123 },
                match: "/cmd1  prop1=123",
            }, {
                task: commandToTask("cmd2", "action2"),
                properties: { prop2: "text", prop3: 4.56 },
                match: "/cmd2 action2 \tprop2='text' prop3=4.56",
            }, {
                task: commandToTask("cmd3"),
                properties: { prop4: null },
                match: "/cmd3 prop4=null",
            }],
        });
    });

    describe("does nothing for non-command test", () => {
        test("test 1", () => {
            extractTasksFromTextTester({
                input: "this is a test",
                expected: [],
            });
        });
        test("test 2", () => {
            extractTasksFromTextTester({
                input: "/a".repeat(1000),
                expected: [],
            });
        });
        test("test 3", () => {
            extractTasksFromTextTester({
                input: "",
                expected: [],
            });
        });
        test("test 4", () => {
            extractTasksFromTextTester({
                input: " ",
                expected: [],
            });
        });
        test("test 5", () => {
            extractTasksFromTextTester({
                input: "\n",
                expected: [],
            });
        });
        test("test 6", () => {
            extractTasksFromTextTester({
                input: "\t",
                expected: [],
            });
        });
        test("test 7", () => {
            extractTasksFromTextTester({
                input: "💃",
                expected: [],
            });
        });
        test("test 8", () => {
            extractTasksFromTextTester({
                input: "你",
                expected: [],
            });
        });
        test("test 9", () => {
            extractTasksFromTextTester({
                input: "!",
                expected: [],
            });
        });
        test("test 10", () => {
            extractTasksFromTextTester({
                input: ".",
                expected: [],
            });
        });
        test("test 11", () => {
            extractTasksFromTextTester({
                input: "123",
                expected: [],
            });
        });
        test("test 12", () => {
            extractTasksFromTextTester({
                input: "-123",
                expected: [],
            });
        });
        test("test 13", () => {
            extractTasksFromTextTester({
                input: "0.123",
                expected: [],
            });
        });
        test("test 14", () => {
            extractTasksFromTextTester({
                input: "null",
                expected: [],
            });
        });
        test("test 15", () => {
            extractTasksFromTextTester({
                input: "true",
                expected: [],
            });
        });
        test("test 16", () => {
            extractTasksFromTextTester({
                input: "false",
                expected: [],
            });
        });
        test("test 17", () => {
            extractTasksFromTextTester({
                input: "123你",
                expected: [],
            });
        });
        test("test 18", () => {
            extractTasksFromTextTester({
                input: "To whom it may concern,\n\nI am writing to inform you that I am a giant purple dinosaur. I cannot be stopped. I am inevitable. I am Barney.\n\nSincerely,\nYour Worst Nightmare",
                expected: [],
            });
        });
    });

    describe("real world examples", () => {
        test("test 1", () => {
            extractTasksFromTextTester({
                input: "/add name='Get Oat Milk' description='Reminder to buy oat milk' dueDate='2023-10-06T09:00:00Z' isComplete=false",
                expected: [{
                    task: commandToTask("add"),
                    properties: { name: "Get Oat Milk", description: "Reminder to buy oat milk", dueDate: "2023-10-06T09:00:00Z", isComplete: false },
                    match: "/add name='Get Oat Milk' description='Reminder to buy oat milk' dueDate='2023-10-06T09:00:00Z' isComplete=false",
                }],
            });
        });
        test("test 2", () => {
            extractTasksFromTextTester({
                input: "/bot find searchString=\"big bird\"",
                expected: [{
                    task: commandToTask("bot", "find"),
                    properties: { searchString: "big bird" },
                    match: "/bot find searchString=\"big bird\"",
                }],
            });
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
    //     console.time('extractTasks with 10 commands');
    //     extractTasks(input, commandToTask);
    //     console.timeEnd('extractTasks with 10 commands');
    // });
    // test('performance test - 100 commands', () => {
    //     const input = generateInput(100);
    //     console.time('extractTasks with 100 commands');
    //     extractTasks(input, commandToTask);
    //     console.timeEnd('extractTasks with 100 commands');
    // });
    // test('performance test - 1000 commands', () => {
    //     const input = generateInput(1000);
    //     console.time('extractTasks with 1000 commands');
    //     extractTasks(input, commandToTask);
    //     console.timeEnd('extractTasks with 1000 commands');
    // });
});

function extractTasksFromJsonTester({
    input,
    expected,
    taskMode,
}: {
    input: object;
    expected: (Omit<MaybeLlmTaskInfo, "start" | "end">)[];
    taskMode: LlmTask | `${LlmTask}`;
}) {
    const stringifiedInput = JSON.stringify(input);
    const start = 0;
    const end = stringifiedInput.length;
    const result = extractTasksFromJson(stringifiedInput, taskMode, enCommandToTask);
    expect(result).toEqual(expected.map((task) => ({ ...task, start, end })));
}

describe("extractTasksFromJson", () => {
    describe("returns no tasks when text isn't JSON", () => {
        it("empty string", () => {
            const result = extractTasksFromJson("", "BotUpdate", commandToTask);
            expect(result).toEqual([]);
        });
        it("whitespace", () => {
            const result = extractTasksFromJson("   \n ", "ApiDelete", commandToTask);
            expect(result).toEqual([]);
        });
        it("non-JSON text", () => {
            const result = extractTasksFromJson("this is a test!", "RoutineFind", commandToTask);
            expect(result).toEqual([]);
        });
        it("task structure in text mode", () => {
            const result = extractTasksFromJson("/bot update name=123", "BotUpdate", commandToTask);
            expect(result).toEqual([]);
        });
        it("numbers, special characters, and Chinese characters", () => {
            const result = extractTasksFromJson("1@#<>{}23你", "RoutineFind", commandToTask);
            expect(result).toEqual([]);
        });
        it("starts like JSON, but not closed", () => {
            const result = extractTasksFromJson("{ \"name\": \"value\"", "RoutineFind", commandToTask);
            expect(result).toEqual([]);
        });
    });

    describe("handles invalid JSON", () => {
        it("empty object", () => {
            const result = extractTasksFromJson("{}", "RoutineFind", commandToTask);
            expect(result).toEqual([]);
        });
        it("empty array", () => {
            const result = extractTasksFromJson("[]", "RoutineFind", commandToTask);
            expect(result).toEqual([]);
        });
    });

    // NOTE: This doesn't need to be too foolproof, as there shouldn't be text before/after the JSON to begin with
    describe("handles additional text outside JSON", () => {
        it("text before JSON", () => {
            const input = {
                command: "api",
                action: "find",
                properties: { searchString: "good stuff" },
            };
            const taskMode = "Start";
            const stuffBeforeJson = "This is some text before the JSON: ";
            const stringifiedInput = JSON.stringify(input);
            const fullInputString = stuffBeforeJson + stringifiedInput;
            const start = stuffBeforeJson.length;
            const end = start + stringifiedInput.length;
            const result = extractTasksFromJson(fullInputString, taskMode, enCommandToTask);
            expect(result).toEqual([{
                task: enCommandToTask(input.command, input.action) as LlmTask,
                properties: { searchString: "good stuff" },
                start,
                end,
            }]);
        });
        it("text after JSON", () => {
            const input = {
                command: "api",
                action: "find",
                properties: { searchString: "good stuff" },
            };
            const taskMode = enCommandToTask(input.command, input.action) as LlmTask;
            const stuffAfterJson = "This is some text after the JSON.";
            const stringifiedInput = JSON.stringify(input);
            const fullInputString = stringifiedInput + stuffAfterJson;
            const start = 0;
            const end = stringifiedInput.length;
            const result = extractTasksFromJson(fullInputString, taskMode, enCommandToTask);
            expect(result).toEqual([{
                task: taskMode,
                properties: { searchString: "good stuff" },
                start,
                end,
            }]);
        });
    });

    describe("handles command/action/properties format", () => {
        describe("single object", () => {
            it("action provided", () => {
                const input = {
                    command: "routine",
                    action: "find",
                    properties: { text: "hello" },
                };
                const taskMode = enCommandToTask(input.command, input.action) as LlmTask;
                extractTasksFromJsonTester({
                    input,
                    expected: [{
                        task: taskMode,
                        properties: { text: "hello" },
                    }],
                    taskMode,
                });
            });
            it("action omitted", () => {
                const input = {
                    command: "routine",
                    properties: { text: "hello" },
                };
                const taskMode = "RoutineFind";
                extractTasksFromJsonTester({
                    input,
                    expected: [{
                        task: taskMode,
                        properties: { text: "hello" },
                    }],
                    taskMode,
                });
            });
            it("action and command omitted", () => {
                const input = {
                    properties: { text: "hello" },
                };
                const taskMode = "RoutineUpdate";
                extractTasksFromJsonTester({
                    input,
                    expected: [{
                        task: taskMode,
                        properties: { text: "hello" },
                    }],
                    taskMode,
                });
            });
        });
        describe("array", () => {
            it("action provided", () => {
                const input = [{
                    command: "routine",
                    action: "find",
                    properties: { text: "hello", num: -69.420 },
                }, {
                    command: "bot",
                    action: "update",
                    properties: { name: "value", yes: true, asdf: null },
                }];
                const taskMode = "RoutineFind";
                extractTasksFromJsonTester({
                    input,
                    expected: [{
                        task: "RoutineFind",
                        properties: { text: "hello", num: -69.420 },
                    }, {
                        task: "BotUpdate",
                        properties: { name: "value", yes: true, asdf: null },
                    }],
                    taskMode,
                });
            });
            it("action omitted", () => {
                const input = [{
                    command: "routine",
                    properties: { text: "hello", num: -69.420 },
                }, {
                    command: "bot",
                    properties: { name: "value", yes: true, asdf: null },
                }];
                const taskMode = "RoutineFind";
                extractTasksFromJsonTester({
                    input,
                    expected: [{
                        task: taskMode, // Falls back to taskMode
                        properties: { text: "hello", num: -69.420 },
                    }, {
                        task: taskMode, // Falls back to taskMode
                        properties: { name: "value", yes: true, asdf: null },
                    }],
                    taskMode,
                });
            });
            it("action and command omitted", () => {
                const input = [{
                    properties: { text: "hello", num: -69.420 },
                }, {
                    properties: { name: "value", yes: true, asdf: null },
                }];
                const taskMode = "RoutineFind";
                extractTasksFromJsonTester({
                    input,
                    expected: [{
                        task: taskMode, // Falls back to taskMode
                        properties: { text: "hello", num: -69.420 },
                    }, {
                        task: taskMode, // Falls back to taskMode
                        properties: { name: "value", yes: true, asdf: null },
                    }],
                    taskMode,
                });
            });
        });
    });

    describe("handles command/action/properties format but wrapped in additional property", () => {
        describe("wrapped in \"tasks\" property", () => {
            describe("outer object single", () => {
                it("action provided", () => {
                    const input = {
                        tasks: {
                            command: "api",
                            action: "update",
                            properties: { id: 123 },
                        },
                    };
                    const taskMode = "Start";
                    console.log("yeet before");
                    extractTasksFromJsonTester({
                        input,
                        expected: [{
                            task: enCommandToTask(input.tasks.command, input.tasks.action),
                            properties: { id: 123 },
                        }],
                        taskMode,
                    });
                });
                it("action omitted", () => {
                    const input = {
                        tasks: {
                            command: "api",
                            properties: { id: 123 },
                        },
                    };
                    const taskMode = "ApiFind";
                    extractTasksFromJsonTester({
                        input,
                        expected: [{
                            task: taskMode,
                            properties: { id: 123 },
                        }],
                        taskMode,
                    });
                });
            });
            describe("outer object array", () => {
                it("action provided", () => {
                    const input = {
                        tasks: [{
                            command: "api",
                            action: "update",
                            properties: { id: 123 },
                        }, {
                            command: "bot",
                            action: "delete",
                            properties: { name: "test" },
                        }],
                    };
                    const taskMode = "Start";
                    extractTasksFromJsonTester({
                        input,
                        expected: [{
                            task: "ApiUpdate",
                            properties: { id: 123 },
                        }, {
                            task: "BotDelete",
                            properties: { name: "test" },
                        }],
                        taskMode,
                    });
                });
                it("action omitted", () => {
                    const input = {
                        tasks: [{
                            command: "api",
                            properties: { id: 123 },
                        }, {
                            command: "bot",
                            properties: { name: "test" },
                        }],
                    };
                    const taskMode = "Start";
                    extractTasksFromJsonTester({
                        input,
                        expected: [{
                            task: taskMode, // Falls back to taskMode
                            properties: { id: 123 },
                        }, {
                            task: taskMode, // Falls back to taskMode
                            properties: { name: "test" },
                        }],
                        taskMode,
                    });
                });
            });
            describe("inner object single", () => {
                it("action provided", () => {
                    const input = {
                        tasks: {
                            command: "api",
                            action: "update",
                            properties: { id: 123 },
                        },
                    };
                    const taskMode = "Start";
                    extractTasksFromJsonTester({
                        input,
                        expected: [{
                            task: "ApiUpdate",
                            properties: { id: 123 },
                        }],
                        taskMode,
                    });
                });
                it("action omitted", () => {
                    const input = {
                        tasks: {
                            command: "api",
                            properties: { id: 123 },
                        },
                    };
                    const taskMode = "Start";
                    extractTasksFromJsonTester({
                        input,
                        expected: [{
                            task: taskMode, // Falls back to taskMode
                            properties: { id: 123 },
                        }],
                        taskMode,
                    });
                });
            });
            describe("inner object array", () => {
                it("action provided", () => {
                    const input = {
                        tasks: [{
                            command: "api",
                            action: "update",
                            properties: { id: 123 },
                        }, {
                            command: "bot",
                            action: "delete",
                            properties: { name: "test" },
                        }],
                    };
                    const taskMode = "Start";
                    extractTasksFromJsonTester({
                        input,
                        expected: [{
                            task: "ApiUpdate",
                            properties: { id: 123 },
                        }, {
                            task: "BotDelete",
                            properties: { name: "test" },
                        }],
                        taskMode,
                    });
                });
                it("action omitted", () => {
                    const input = {
                        tasks: [{
                            command: "api",
                            properties: { id: 123 },
                        }, {
                            command: "bot",
                            properties: { name: "test" },
                        }],
                    };
                    const taskMode = "Start";
                    extractTasksFromJsonTester({
                        input,
                        expected: [{
                            task: taskMode, // Falls back to taskMode
                            properties: { id: 123 },
                        }, {
                            task: taskMode, // Falls back to taskMode
                            properties: { name: "test" },
                        }],
                        taskMode,
                    });
                });
            });
        });
        describe("wrapped in \"task\" property", () => {
            describe("outer object single", () => {
                it("action provided", () => {
                    const input = {
                        task: {
                            command: "api",
                            action: "update",
                            properties: { id: 123 },
                        },
                    };
                    const taskMode = "Start";
                    console.log("yeet before");
                    extractTasksFromJsonTester({
                        input,
                        expected: [{
                            task: enCommandToTask(input.task.command, input.task.action),
                            properties: { id: 123 },
                        }],
                        taskMode,
                    });
                });
                it("action omitted", () => {
                    const input = {
                        task: {
                            command: "api",
                            properties: { id: 123 },
                        },
                    };
                    const taskMode = "ApiFind";
                    extractTasksFromJsonTester({
                        input,
                        expected: [{
                            task: taskMode,
                            properties: { id: 123 },
                        }],
                        taskMode,
                    });
                });
            });
            describe("outer object array", () => {
                it("action provided", () => {
                    const input = {
                        task: [{
                            command: "api",
                            action: "update",
                            properties: { id: 123 },
                        }, {
                            command: "bot",
                            action: "delete",
                            properties: { name: "test" },
                        }],
                    };
                    const taskMode = "Start";
                    extractTasksFromJsonTester({
                        input,
                        expected: [{
                            task: "ApiUpdate",
                            properties: { id: 123 },
                        }, {
                            task: "BotDelete",
                            properties: { name: "test" },
                        }],
                        taskMode,
                    });
                });
                it("action omitted", () => {
                    const input = {
                        task: [{
                            command: "api",
                            properties: { id: 123 },
                        }, {
                            command: "bot",
                            properties: { name: "test" },
                        }],
                    };
                    const taskMode = "Start";
                    extractTasksFromJsonTester({
                        input,
                        expected: [{
                            task: taskMode, // Falls back to taskMode
                            properties: { id: 123 },
                        }, {
                            task: taskMode, // Falls back to taskMode
                            properties: { name: "test" },
                        }],
                        taskMode,
                    });
                });
            });
            describe("inner object single", () => {
                it("action provided", () => {
                    const input = {
                        task: {
                            command: "api",
                            action: "update",
                            properties: { id: 123 },
                        },
                    };
                    const taskMode = "Start";
                    extractTasksFromJsonTester({
                        input,
                        expected: [{
                            task: "ApiUpdate",
                            properties: { id: 123 },
                        }],
                        taskMode,
                    });
                });
                it("action omitted", () => {
                    const input = {
                        task: {
                            command: "api",
                            properties: { id: 123 },
                        },
                    };
                    const taskMode = "Start";
                    extractTasksFromJsonTester({
                        input,
                        expected: [{
                            task: taskMode, // Falls back to taskMode
                            properties: { id: 123 },
                        }],
                        taskMode,
                    });
                });
            });
            describe("inner object array", () => {
                it("action provided", () => {
                    const input = {
                        task: [{
                            command: "api",
                            action: "update",
                            properties: { id: 123 },
                        }, {
                            command: "bot",
                            action: "delete",
                            properties: { name: "test" },
                        }],
                    };
                    const taskMode = "Start";
                    extractTasksFromJsonTester({
                        input,
                        expected: [{
                            task: "ApiUpdate",
                            properties: { id: 123 },
                        }, {
                            task: "BotDelete",
                            properties: { name: "test" },
                        }],
                        taskMode,
                    });
                });
                it("action omitted", () => {
                    const input = {
                        task: [{
                            command: "api",
                            properties: { id: 123 },
                        }, {
                            command: "bot",
                            properties: { name: "test" },
                        }],
                    };
                    const taskMode = "Start";
                    extractTasksFromJsonTester({
                        input,
                        expected: [{
                            task: taskMode, // Falls back to taskMode
                            properties: { id: 123 },
                        }, {
                            task: taskMode, // Falls back to taskMode
                            properties: { name: "test" },
                        }],
                        taskMode,
                    });
                });
            });
        });
    });

    describe("handles object with properties as keys and values", () => {
        describe("single object", () => {
            it("with command and action", () => {
                const input = {
                    command: "routine",
                    action: "find",
                    text: "hello",
                    num: 123,
                    isComplete: false,
                };
                const taskMode = "Start";
                extractTasksFromJsonTester({
                    input,
                    expected: [{
                        task: "RoutineFind",
                        properties: { text: "hello", num: 123, isComplete: false },
                    }],
                    taskMode,
                });
            });
            it("with command and no action", () => {
                const input = {
                    command: "routine",
                    text: "hello",
                    num: 123,
                    isComplete: false,
                };
                const taskMode = "Start";
                extractTasksFromJsonTester({
                    input,
                    expected: [{
                        task: taskMode, // Falls back to taskMode
                        properties: { text: "hello", num: 123, isComplete: false },
                    }],
                    taskMode,
                });
            });
            it("with properties only", () => {
                const input = {
                    text: "hello",
                    num: 123,
                    isComplete: false,
                };
                const taskMode = "Start";
                extractTasksFromJsonTester({
                    input,
                    expected: [{
                        task: taskMode, // Falls back to taskMode
                        properties: { text: "hello", num: 123, isComplete: false },
                    }],
                    taskMode,
                });
            });
        });
        describe("array", () => {
            it("with command and action", () => {
                const input = [{
                    command: "routine",
                    action: "find",
                    text: "hello",
                    num: 123,
                    isComplete: false,
                }, {
                    command: "bot",
                    action: "update",
                    name: "value",
                    yes: true,
                    no: null,
                }];
                const taskMode = "Start";
                extractTasksFromJsonTester({
                    input,
                    expected: [{
                        task: "RoutineFind",
                        properties: { text: "hello", num: 123, isComplete: false },
                    }, {
                        task: "BotUpdate",
                        properties: { name: "value", yes: true, no: null },
                    }],
                    taskMode,
                });
            });
            it("with command and no action", () => {
                const input = [{
                    command: "routine",
                    text: "hello",
                    num: 123,
                    isComplete: false,
                }, {
                    command: "bot",
                    name: "value",
                    yes: true,
                    no: null,
                }];
                const taskMode = "Start";
                extractTasksFromJsonTester({
                    input,
                    expected: [{
                        task: taskMode, // Falls back to taskMode
                        properties: { text: "hello", num: 123, isComplete: false },
                    }, {
                        task: taskMode, // Falls back to taskMode
                        properties: { name: "value", yes: true, no: null },
                    }],
                    taskMode,
                });
            });
            it("with properties only", () => {
                const input = [{
                    text: "hello",
                    num: 123,
                    isComplete: false,
                }, {
                    name: "value",
                    yes: true,
                    no: null,
                }];
                const taskMode = "Start";
                extractTasksFromJsonTester({
                    input,
                    expected: [{
                        task: taskMode, // Falls back to taskMode
                        properties: { text: "hello", num: 123, isComplete: false },
                    }, {
                        task: taskMode, // Falls back to taskMode
                        properties: { name: "value", yes: true, no: null },
                    }],
                    taskMode,
                });
            });
        });
    });
});

describe("filterInvalidTasks", () => {

    test("Filters: all valid except for task", async () => {
        const potentialCommands: MaybeLlmTaskInfo[] = [{
            task: "asjflkdjslafkjslaf" as LlmTask,
            properties: null,
            start: 0,
            end: 10,
        }];

        const result = await filterInvalidTasks(potentialCommands, "Start", {}, enCommandToTask, "en", console);
        expect(result).toEqual([]);
    });

    test("Accepts: properties missing, but we're in the Start mode", async () => {
        // The "Start" task allows us to trigger/suggest several commands, but we're 
        // not given any properties. This should be allowed, even though they'd be 
        // filtered out in other modes.
        const potentialCommands: MaybeLlmTaskInfo[] = [{
            task: enCommandToTask("bot", "add"),
            properties: null,
            start: 0,
            end: 10,
        }];

        const result = await filterInvalidTasks(potentialCommands, "Start", {}, enCommandToTask, "en", console);
        expect(result).toEqual(potentialCommands);
    });

    test("Filters: properties missing, and we're not the Start mode", async () => {
        const potentialCommands: MaybeLlmTaskInfo[] = [{
            task: enCommandToTask("bot", "add"),
            properties: null,
            start: 0,
            end: 10,
        }];

        const result = await filterInvalidTasks(potentialCommands, "BotAdd", {}, enCommandToTask, "en", console);
        expect(result).toEqual([]); // The "BotAdd" task requires a "name" property, but we're not given any properties.
    });

    test("Accepts: properties missing, but they are provided in existingData", async () => {
        const potentialCommands: MaybeLlmTaskInfo[] = [{
            task: enCommandToTask("bot", "add"),
            properties: null,
            start: 0,
            end: 10,
        }];

        const result = await filterInvalidTasks(potentialCommands, "BotAdd", { name: "hello" }, enCommandToTask, "en", console);
        expect(result).toEqual(potentialCommands);
    });

    describe("Valid potential command, no trickery", () => {
        test("Accepts: task is the same as taskMode", async () => {
            const potentialCommands: MaybeLlmTaskInfo[] = [{
                task: "BotAdd",
                properties: { name: "hello" },
                start: 0,
                end: 10,
            }];

            const result = await filterInvalidTasks(potentialCommands, "BotAdd", {}, enCommandToTask, "en", console);
            expect(result).toEqual(potentialCommands);
        });

        test("Accepts: task is different than taskMode, but still valid", async () => {
            const potentialCommands: MaybeLlmTaskInfo[] = [{
                task: enCommandToTask("routine", "find"),
                properties: { searchString: "hello" }, // Valid property if we're in RoutineFind mode. But we're in Start mode, so it will be removed
                start: 0,
                end: 10,
            }];

            const result = await filterInvalidTasks(potentialCommands, "Start", {}, enCommandToTask, "en", console);
            expect(result).toEqual([{ ...potentialCommands[0], properties: {} }]);
        });

        test("Omits: task is different than taskMode, and invalid", async () => {
            const potentialCommands: MaybeLlmTaskInfo[] = [{
                task: "RunProjectStart", // Not available in Start mode
                properties: { name: "hello" },
                start: 0,
                end: 10,
            }];

            const result = await filterInvalidTasks(potentialCommands, "Start", {}, enCommandToTask, "en", console);
            expect(result).toEqual([]);
        });
    });

    test("omits invalid properties", async () => {
        const potentialCommands: MaybeLlmTaskInfo[] = [{
            task: enCommandToTask("routine", "add"),
            properties: { "name": "value", "jeff": "bunny" },
            start: 0,
            end: 10,
        }];

        const result = await filterInvalidTasks(potentialCommands, "RoutineAdd", {}, enCommandToTask, "en", console);
        expect(result.length).toBe(1);
        expect(result[0].task).toBe(enCommandToTask("routine", "add"));
        expect(result[0].properties).toEqual({ "name": "value" });
    });

    test("Filters: missing required properties", async () => {
        const potentialCommands: MaybeLlmTaskInfo[] = [{
            task: enCommandToTask("routine", "add"),
            properties: {},
            start: 0,
            end: 10,
        }];

        const result = await filterInvalidTasks(potentialCommands, "RoutineAdd", {}, enCommandToTask, "en", console);
        expect(result).toEqual([]); // Missing required property
    });
});

describe("removeTasks", () => {
    test("removes a single command from the string", () => {
        const input = "/command1 action1 prop1=value1";
        const commands: PartialTaskInfo[] = [{
            ...getStartEnd(input, "/command1 action1 prop1=value1"),
            task: "Start",
            properties: { prop1: "value1" },
        }];
        expect(removeTasks(input, commands)).toBe("");
    });

    test("removes multiple commands from the string", () => {
        const input = "Text before /command1 action1 prop1=value1 and text after /command2";
        const commands: PartialTaskInfo[] = [
            {
                ...getStartEnd(input, "/command1 action1 prop1=value1"),
                task: "Start",
                properties: { prop1: "value1" },
            },
            {
                ...getStartEnd(input, "/command2"),
                task: "Start",
                properties: null,
            },
        ];
        expect(removeTasks(input, commands)).toBe("Text before  and text after ");
    });

    test("handles commands at the start and end of the string", () => {
        const input = "/command1 at start /command2 at end";
        const commands: PartialTaskInfo[] = [
            {
                ...getStartEnd(input, "/command1 at start"),
                task: "Start",
                properties: null,
            },
            {
                ...getStartEnd(input, "/command2 at end"),
                task: "Start",
                properties: null,
            },
        ];
        expect(removeTasks(input, commands)).toBe(" ");
    });

    test("handles overlapping commands", () => {
        const input = "/command1 /nestedCommand2 /nestedCommand3";
        const commands: PartialTaskInfo[] = [
            {
                ...getStartEnd(input, "/command1 /nestedCommand2"),
                task: "Start",
                properties: null,
            },
            {
                ...getStartEnd(input, "/nestedCommand2 /nestedCommand3"),
                task: "Start",
                properties: null,
            },
            {
                ...getStartEnd(input, "/nestedCommand3"),
                task: "Start",
                properties: null,
            },
        ];
        expect(removeTasks(input, commands)).toBe("");
    });

    test("leaves string untouched if no commands match", () => {
        const input = "This string has no commands";
        const commands: PartialTaskInfo[] = [];
        expect(removeTasks(input, commands)).toBe(input);
    });

    // Add more tests for other edge cases as needed
});

//TODO add more tests, including for JSON mode
describe("getValidTasksFromMessage", () => {
    let commandToTask;
    const language = "en";
    const logger = console;

    beforeEach(async () => {
        commandToTask = await importCommandToTask(language, logger);
    });

    test("real world examples", async () => {
        const message = "/bot find searchString=\"big bird\"";
        const result = await getValidTasksFromMessage({
            commandToTask,
            existingData: null,
            language,
            logger,
            message,
            mode: "text",
            taskMode: "Start",
        });

        expect(result).toEqual({
            messageWithoutTasks: "",
            tasksToRun: [{
                task: "BotFind",
                label: "Find Bot",
                properties: {}, //Search string is a property in taskMode "BotFind", not "Start"
                start: 0,
                end: message.length,
                taskId: expect.any(String),
            }],
            tasksToSuggest: [],
        });
    });
});
