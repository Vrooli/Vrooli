import { uuid } from "@local/shared";
import { LlmTask } from "../api/generated/graphqlTypes";
import { PassableLogger } from "../consts/commonTypes";
import { getUnstructuredTaskConfig, importConfig } from "./config";
import { CommandSection, CommandToTask, CommandTransitionTrack, ExistingTaskData, LanguageModelResponseMode, LlmTaskProperty, MaybeLlmTaskInfo, PartialTaskInfo, ServerLlmTaskInfo } from "./types";

/** Properties stored as an array of [key, value, key, value, ...] */
type CurrentLlmTaskInfo = Omit<PartialTaskInfo, "lastUpdated" | "properties" | "task"> & {
    action: string | null;
    command: string;
    properties: (string | number | boolean | null)[];
};

type GetValidTasksFromMessageParams = {
    /** A function for converting a command and action to a valid task name */
    commandToTask: CommandToTask,
    /**
     * Data we already have for the tasksToRun (e.g. if doing autofill, fields already filled in).
     *  
     * This data will be omitted when checking for required properties.
     */
    existingData: ExistingTaskData | null,
    /** The language of the user */
    language: string,
    logger: PassableLogger,
    /** The message containing the tasks */
    message: string,
    /**
    * The mode to use when generating the response
    */
    mode: LanguageModelResponseMode;
    /** The current task mode, which determines the tasks we're allowed to start next */
    taskMode: LlmTask | `${LlmTask}`,
};
type GetValidTasksFromMessageResult = {
    messageWithoutTasks: string,
    tasksToRun: ServerLlmTaskInfo[],
    tasksToSuggest: ServerLlmTaskInfo[],
};

export type CommandWrapper = {
    label: string,
    start: string,
    end: string,
    delimiter: string,
    allowMultiple: boolean,
};

type WrappedTasks = {
    taskIndices: number[],
    wrapperStart: number,
    wrapperEnd: number,
};

type HandleTaskTransitionParams = CommandTransitionTrack & {
    curr: string,
    prev: string,
    onCommit: (section: CommandSection, text: string | number | boolean | null) => unknown,
    onComplete: () => unknown, // When a full command (including any actions/props) is completed
    onCancel: () => unknown, // When a full command is cancelled. Typically only when there's a problem with the beginning slash command 
    onStart: () => unknown, // When a full command is started
};

type TaskTransitionHelperParams = Omit<HandleTaskTransitionParams, "section">;

/** Maximum length for a command, action, or property name */
const MAX_COMMAND_ACTION_PROP_LENGTH = 32;
const QUOTES = ["\"", "'"];

export function isNewline(char: string) {
    return char === "\n" || char === "\r";
}

export function isWhitespace(char: string): boolean {
    return char === " " || char === "\t";
}

export function isAlphaNum(char: string): boolean {
    const code = char.charCodeAt(0);
    // eslint-disable-next-line no-magic-numbers
    return (code > 47 && code < 58) || // numeric (0-9)
        // eslint-disable-next-line no-magic-numbers
        (code > 64 && code < 91) || // upper alpha (A-Z)
        // eslint-disable-next-line no-magic-numbers
        (code > 96 && code < 123);  // lower alpha (a-z)
}


function countEndSlashes(charArray: string[]): number {
    let count = 0;
    for (let i = charArray.length - 1; i >= 0; i--) {
        if (charArray[i] === "\\") {
            count++;
        } else {
            break; // Stop counting when a non-backslash character is encountered
        }
    }
    return count;
}

/**
 * Handles the transition logic when the parser is outside of any specific command,
 * action, or property block. It determines whether to start a new command or handle
 * other structures like code blocks.
 * 
 * @returns The updated state after processing.
 */
export function handleTaskTransitionOutside(params: TaskTransitionHelperParams): CommandTransitionTrack {
    const { buffer, curr, onStart, prev } = params;

    let hasOpenBracket = params.hasOpenBracket;
    // Set hasOpenBracket to true if the previous character was an open bracket
    if (prev === "[") {
        hasOpenBracket = true;
    }
    // Reset hasOpenBracket if the previous character was a close bracket, newline, or (non-whitespace and not a comma)
    else if (prev === "]" || isNewline(prev) || (!isWhitespace(prev) && prev !== ",")) {
        hasOpenBracket = false;
    }

    // Start a command if there's a slash without a previous character, 
    // or if the previous character is whitespace, newline, or open bracket (for wrapped commands)
    if (curr === "/" && (!prev || isWhitespace(prev) || isNewline(prev) || prev === "[")) {
        onStart();
        return {
            section: "command",
            buffer: [], // Don't include the slash in the buffer
            hasOpenBracket,
        };
    }
    // Reset buffer when there's whitespace=
    if (isWhitespace(curr)) {
        return { section: "outside", buffer: [], hasOpenBracket };
    }
    // Reset buffer when there's a newline
    if (isNewline(curr)) {
        return { section: "outside", buffer: [], hasOpenBracket };
    }
    // Start a code block if the end of buffer + curr "<code>"
    // eslint-disable-next-line no-magic-numbers
    if ((buffer.length >= 5 && curr === ">" && buffer[buffer.length - 1] === "e" && buffer[buffer.length - 2] === "d" && buffer[buffer.length - 3] === "o" && buffer[buffer.length - 4] === "c" && buffer[buffer.length - 5] === "<")) {
        return { section: "code", buffer: ["<", "c", "o", "d", "e", ">"], hasOpenBracket };
    }
    // Start a code block if curr is not a backtick, and the buffer ends with 1 or 3 backticks 
    // (Any other combination of sequential backticks is invalid)
    if (curr !== "`" && buffer.length >= 1 && buffer[buffer.length - 1] === "`" && buffer[buffer.length - 2] !== "`") {
        return { section: "code", buffer: ["`", curr], hasOpenBracket };
    }
    // eslint-disable-next-line no-magic-numbers
    if (curr !== "`" && buffer.length >= 3 && buffer[buffer.length - 1] === "`" && buffer[buffer.length - 2] === "`" && buffer[buffer.length - 3] === "`" && buffer[buffer.length - 4] !== "`") {
        return { section: "code", buffer: ["`", "`", "`", curr], hasOpenBracket };
    }
    // Continue buffering
    buffer.push(curr);
    return { section: "outside", buffer, hasOpenBracket };
}

/**
 * Manages the transitions within code blocks, handling different types of code
 * markers like single backticks, triple backticks, or HTML-style code tags.
 * 
 * NOTE: This function never calls `onComplete`, `onStart`, or `onCommit`, as code blocks are 
 * not a part of the command structure.
 * 
 * @param params - The current state and utility functions.
 * @returns The updated state after processing.
 */
export function handleTaskTransitionCode(params: TaskTransitionHelperParams): CommandTransitionTrack {
    const { buffer, curr, hasOpenBracket, onCancel } = params;

    // Shouldn't be here if the buffer is empty or malformed
    if (buffer.length === 0 || !["`", "<"].includes(buffer[0] as string)) {
        onCancel();
        return { section: "outside", buffer: [], hasOpenBracket };
    }
    // Determine which type of code block we're in
    let codeType: "single" | "multi" | "tag" | null = null;
    if (buffer[0] === "<") {
        codeType = "tag";
    } else if (
        (buffer.length > 2 && buffer[0] === "`" && buffer[1] === "`" && buffer[2] === "`") ||
        (buffer.length === 2 && buffer[0] === "`" && buffer[1] === "`" && curr === "`")
    ) {
        codeType = "multi";
    } else if (buffer[0] === "`") {
        codeType = "single";
    }
    // If we couldn't determine the code type, cancel the code block
    if (!codeType) {
        onCancel();
        return { section: "outside", buffer: [], hasOpenBracket };
    }
    // If we're in a single code block, cancel on newline
    if (codeType === "single" && isNewline(curr)) {
        onCancel();
        return { section: "outside", buffer: [], hasOpenBracket: false };
    }
    // If we're in a multi code block, cancel when encountering the closing backticks
    if (codeType === "multi") {
        // Cancel at fourth backtick in a row
        // eslint-disable-next-line no-magic-numbers
        if (buffer.length === 3 && buffer.every(b => b === "`") && curr === "`") {
            onCancel();
            return { section: "outside", buffer: [], hasOpenBracket };
        }
        // Cancel when closing triple backticks found
        // eslint-disable-next-line no-magic-numbers
        if (buffer.length > 3 && curr === "`" && buffer[buffer.length - 1] === "`" && buffer[buffer.length - 2] === "`") {
            onCancel();
            return { section: "outside", buffer: [], hasOpenBracket };
        }
    }
    // If we're in a tag code block
    if (codeType === "tag") {
        // When the buffer is short, stop if the tag never forms
        // eslint-disable-next-line no-magic-numbers
        if (buffer.length < 6) {
            // Cancel if we run into any characters not in "<code>"
            if (!"<code>".startsWith([...buffer, curr].join(""))) {
                onCancel();
                return { section: "outside", buffer: [], hasOpenBracket };
            }
        }
        // When the buffer is longer, stop if the tag is complete
        // eslint-disable-next-line no-magic-numbers
        if (buffer.length >= 13) {
            // Cancel if it end with "</code>"
            // eslint-disable-next-line no-magic-numbers
            if (buffer.slice(-6).join("") === "</code" && curr === ">") {
                onCancel();
                return { section: "outside", buffer: [], hasOpenBracket };
            }
        }
    }
    // Continue buffering
    buffer.push(curr);
    return { section: "code", buffer, hasOpenBracket };
}

/**
 * Manages transitions within a command section, dealing with the end of a command,
 * or transitioning to an action or property name.
 * 
 * @param params - The current state and utility functions.
 * @returns The updated state after processing.
 */
export function handleTaskTransitionCommand(params: TaskTransitionHelperParams): CommandTransitionTrack {
    const { curr, buffer, hasOpenBracket, onCommit, onComplete, onCancel } = params;

    // If we run into another slash, the command is invalid
    if (curr === "/") {
        onCancel();
        return { section: "outside", buffer: [], hasOpenBracket };
    }
    // Handle transition from command to action (might actually be property name,
    // but we're not sure yet
    if (isWhitespace(curr)) {
        onCommit("command", buffer.join(""));
        return { section: "action", buffer: [], hasOpenBracket };
    }
    // Commit on newline
    if (isNewline(curr)) {
        onCommit("command", buffer.join(""));
        onComplete();
        return { section: "outside", buffer: [], hasOpenBracket: false };
    }
    // If there is an open bracket
    if (hasOpenBracket) {
        if (curr === "]") {
            onCommit("command", buffer.join(""));
            onComplete();
            return { section: "outside", buffer: [], hasOpenBracket: false };
        }
        if (curr === ",") {
            onCommit("command", buffer.join(""));
            onComplete();
            return { section: "outside", buffer: [], hasOpenBracket };
        }
    }
    // Cancel on non alpha-numeric characters
    if (!isAlphaNum(curr)) {
        onCancel();
        return { section: "outside", buffer: [], hasOpenBracket };
    }
    // Cancel if the buffer is too long
    if (buffer.length >= MAX_COMMAND_ACTION_PROP_LENGTH) {
        onCancel();
        return { section: "outside", buffer: [], hasOpenBracket };
    }
    // Continue buffering
    buffer.push(curr);
    return { section: "command", buffer, hasOpenBracket };
}

/**
 * Handles transitions within the action part of a command, potentially moving
 * to property name or command completion.
 * 
 * @param params - The current state and utility functions.
 * @returns The updated state after processing.
 */
export function handleTaskTransitionAction(params: TaskTransitionHelperParams): CommandTransitionTrack {
    const { curr, buffer, hasOpenBracket, onCommit, onComplete, onStart, prev } = params;

    // If a newline is encountered
    if (isNewline(curr)) {
        // Commit as an action if the buffer is not empty
        if (buffer.length > 0) {
            onCommit("action", buffer.join(""));
        }
        onComplete();
        return { section: "outside", buffer: [], hasOpenBracket: false };
    }
    // Handle whitespace
    if (isWhitespace(curr)) {
        // Skip if buffer is empty
        if (buffer.length === 0) return { section: "action", buffer, hasOpenBracket };
        // Othwerwise, commit as an action
        onCommit("action", buffer.join(""));
        return { section: "propName", buffer: [], hasOpenBracket };
    }
    // Handle slash
    if (curr === "/") {
        // If previous character is not a whitespace, it's not a valid action 
        // or start of a new command
        if (!isWhitespace(prev)) {
            onComplete();
            return { section: "outside", buffer: [], hasOpenBracket };
        }
        // If there's a buffer, commit it as an action
        if (buffer.length > 0) {
            onCommit("action", buffer.join(""));
        }
        // Start a new command
        onComplete();
        onStart();
        return { section: "command", buffer: [], hasOpenBracket };
    }
    // If it's an equals sign, commit as a property name 
    // instead of an action
    if (curr === "=") {
        onCommit("propName", buffer.join(""));
        return { section: "propValue", buffer: [], hasOpenBracket };
    }
    // If there is an open bracket
    if (hasOpenBracket) {
        if (curr === "]") {
            // Only commit if buffer is not empty. An empty buffer indicates trailing whitespace
            // between the command and closing bracket (i.e. there isn't an action)
            if (buffer.length > 0) {
                onCommit("action", buffer.join(""));
            }
            onComplete();
            return { section: "outside", buffer: [], hasOpenBracket: false };
        }
        if (curr === ",") {
            onCommit("action", buffer.join(""));
            onComplete();
            return { section: "outside", buffer: [], hasOpenBracket };
        }
    }
    // Complete if it's not an alpha-numeric character
    if (!isAlphaNum(curr)) {
        // Commit if previous character was whitespace
        if (isWhitespace(prev)) {
            onCommit("action", buffer.join(""));
        }
        onComplete();
        // If it's a backtick, start a code block
        if (curr === "`") {
            return { section: "code", buffer: [curr], hasOpenBracket };
        }
        return { section: "outside", buffer: [], hasOpenBracket };
    }
    // Complete if the buffer is too long
    if (buffer.length >= MAX_COMMAND_ACTION_PROP_LENGTH) {
        onComplete();
        return { section: "outside", buffer: [], hasOpenBracket };
    }
    // Continue buffering
    buffer.push(curr);
    return { section: "action", buffer, hasOpenBracket };
}

/**
 * Manages transitions when handling a property name, deciding whether to commit
 * the property name and move to the property value or handle other transitions.
 * 
 * @param params - The current state and utility functions.
 * @returns The updated state after processing.
 */
export function handleTaskTransitionPropName(params: TaskTransitionHelperParams): CommandTransitionTrack {
    const { curr, buffer, hasOpenBracket, onCommit, onComplete, onStart, prev } = params;

    // Commit on equals sign
    if (curr === "=") {
        onCommit("propName", buffer.join(""));
        return { section: "propValue", buffer: [], hasOpenBracket };
    }
    // Handle slash
    if (curr === "/") {
        // If previous character is not a whitespace, it's not a valid action 
        // or start of a new command
        if (!isWhitespace(prev)) {
            onComplete();
            return { section: "outside", buffer: [], hasOpenBracket };
        }
        // Start a new command. Do not commit the buffer, as it won't have an accompanying value
        onComplete();
        onStart();
        return { section: "command", buffer: [], hasOpenBracket };
    }
    // Handle whitespace
    if (isWhitespace(curr)) {
        // Skip if buffer is empty
        if (buffer.length === 0) return { section: "propName", buffer, hasOpenBracket };
        // Otherwise, complete command
        if (buffer.length > 0) {
            onComplete();
            return { section: "outside", buffer: [], hasOpenBracket };
        }
    }
    // Complete on newline
    if (isNewline(curr)) {
        onComplete();
        return { section: "outside", buffer: [], hasOpenBracket: false };
    }
    // Complete on non alpha-numeric characters
    if (!isAlphaNum(curr)) {
        onComplete();
        // If previous character was whitespace and it's a backtick, start a code block
        if (isWhitespace(prev) && curr === "`") {
            return { section: "code", buffer: [curr], hasOpenBracket };
        }
        // Otherwise, move to the outside state
        return { section: "outside", buffer: [], hasOpenBracket };
    }
    // Complete if the buffer is too long
    if (buffer.length >= MAX_COMMAND_ACTION_PROP_LENGTH) {
        onComplete();
        return { section: "outside", buffer: [], hasOpenBracket };
    }
    // Continue buffering
    buffer.push(curr);
    return { section: "propName", buffer, hasOpenBracket };
}

/**
 * Manages transitions when handling a property value, particularly when dealing with
 * quoted strings, numbers, or the keyword 'null'.
 * 
 * @param params - The current state and utility functions.
 * @returns The updated state after processing.
 */
export function handleTaskTransitionPropValue(params: TaskTransitionHelperParams): CommandTransitionTrack {
    const { curr, buffer, hasOpenBracket, onCommit, onComplete } = params;

    const backslashes = countEndSlashes(buffer);
    const isEscaped = backslashes % 2 !== 0;
    const isQuote = QUOTES.includes(curr);
    const isInQuote = QUOTES.includes(buffer[0] ?? "");
    // Check if leaving quotes
    if (isQuote && buffer[0] === curr && !isEscaped) {
        onCommit("propValue", buffer.slice(1).join("")); // Exclude outer quotes
        return { section: "propName", buffer: [], hasOpenBracket };
    }
    // Check if entering quotes
    if (!isInQuote && isQuote && buffer.length === 0) {
        return { section: "propValue", buffer: [curr], hasOpenBracket };
    }
    // Allow anything inside quotes
    if (isInQuote) {
        buffer.push(curr);
        return { section: "propValue", buffer, hasOpenBracket };
    }
    // Only numbers and null are allowed outside quotes
    const bufferString = buffer.join("");
    const withCurr = bufferString + curr;
    const maybeNumber = !isWhitespace(curr) && !isNewline(curr) && ((buffer.length === 0 && ["-", "."].includes(curr)) || Number.isFinite(+withCurr));
    const maybeNull = "null".startsWith(withCurr);
    const maybeTrue = "true".startsWith(withCurr);
    const maybeFalse = "false".startsWith(withCurr);
    if (maybeNumber || maybeNull || maybeTrue || maybeFalse) {
        buffer.push(curr);
        return { section: "propValue", buffer, hasOpenBracket };
    }
    // If we reached whitespace, a newline, or a comma or end bracket, we'll probably commit
    if (isWhitespace(curr) || isNewline(curr)) {
        // We've already accounted for quotes (i.e. strings). So we can only commit 
        // if it's a valid number or null
        const isNumber = Number.isFinite(+bufferString) && buffer.length > 0;
        const isNull = bufferString === "null";
        const isTrue = bufferString === "true";
        const isFalse = bufferString === "false";
        if (isNumber || isNull || isTrue || isFalse) {
            onCommit("propValue", isNumber ? +bufferString : (isNull ? null : (isTrue ? true : false)));
            if (isNewline(curr)) {
                onComplete();
                return { section: "outside", buffer: [], hasOpenBracket: false };
            }
            // If there is an open bracket
            if (hasOpenBracket) {
                if (curr === "]") {
                    onComplete();
                    return { section: "outside", buffer: [], hasOpenBracket: false };
                }
                if (curr === ",") {
                    onComplete();
                    return { section: "outside", buffer: [], hasOpenBracket };
                }
            }
            return { section: "propName", buffer: [], hasOpenBracket };
        }
    }
    // Otherwise it's invalid, so complete the command
    onComplete();
    return { section: "outside", buffer: [], hasOpenBracket };
}

const handlerMap = {
    outside: handleTaskTransitionOutside,
    code: handleTaskTransitionCode,
    command: handleTaskTransitionCommand,
    action: handleTaskTransitionAction,
    propName: handleTaskTransitionPropName,
    propValue: handleTaskTransitionPropValue,
};

/**
 * Handles the transition between parsing a command, action, property name, or property value. 
 * 
 * This is not meant to be the shortest possible function, but rather one that's the easiest to understand and 
 * maintain. The outer if-else structure which handles each section type chronologically, with each section
 * handling the transition between sections, invalid characters, and other edge cases.
 * @returns An object containing the new parsing section and buffer
 */
export function handleTaskTransition({ section, ...rest }: HandleTaskTransitionParams): CommandTransitionTrack {
    return handlerMap[section](rest);
}

/**
 * Helper function which loops forward or backward from an index to find the matching character.
 * If it finds a non-whitespace character (excluding newline), passes the max length, or reaches the end of the string,
 * it returns null. Otherwise, it returns the index of the matching character.
 * 
 * @param index The index to start from.
 * @param forward The direction to search in. If true, it searches forward; otherwise, it searches backward.
 * @param char The character to search for.
 * @param input The string to search in.
 * @param maxLength The maximum number of whitespace characters to allow.
 * @returns The index of the matching character, or null if it's not found.
 */
export function findCharWithLimit(
    index: number,
    forward: boolean,
    char: string,
    input: string,
    maxLength: number,
): number | null {
    if (input[index] === char) return index; // If the current character is the target, return it

    // Determine the step direction: forward (1) or backward (-1)
    const step = forward ? 1 : -1;

    // If we're already at the start or end of the string, move the index back one
    if (index === 0 || index === input.length - 1) index -= step;

    let loopCount = 0;
    while (loopCount++ < maxLength) {
        index += step; // Move index in the specified direction

        // Check boundaries
        if (index < 0 || index >= input.length) {
            return null; // Reached the end or beginning of the string without finding the character
        }

        const currentChar = input[index];

        if (currentChar === char) {
            return index; // Found the target character
        } else if (!currentChar || (!isWhitespace(currentChar) && !isNewline(currentChar))) {
            // Found a non-whitespace character that is not the target
            return null;
        }
    }
    return null; // Exceeded the maximum loop count
}

/**
 * Extracts tasks from a given string (in "text" mode) based on predefined formats.
 * 
 * The function searches for patterns that match tasks in the format "/command action property1=value1 property2=value2".
 * It supports multiple tasks within the same string, separated by spaces or other tasks.
 * 
 * NOTE: This may capture tasks (and command actions/properties) that are not valid. These should 
 * be ignored when comparing them to the available tasks.
 * 
 * @param inputString The string containing one or more taaks to be parsed.
 * @param commandToTask A function for converting a command and action to a valid task name.
 * @returns An array of task data
 */
export function extractTasksFromText(
    inputString: string,
    commandToTask: CommandToTask,
): MaybeLlmTaskInfo[] {
    const commands: MaybeLlmTaskInfo[] = [];
    let currentCommand: CurrentLlmTaskInfo | null = null;

    /** When the full command is completed */
    function onComplete() {
        if (!currentCommand) return;
        const action = (currentCommand.action && currentCommand.action.length) ? currentCommand.action : null;
        const task = commandToTask(currentCommand.command, action);
        commands.push({
            end: currentCommand.end,
            start: currentCommand.start,
            task,
            // Convert properties array to object
            properties: currentCommand.properties.reduce((obj, item, index) => {
                if (index % 2 === 0) { // Even index, property name
                    // If it's the last element and there's no corresponding value, skip it
                    if (index === currentCommand!.properties.length - 1) return obj;
                    obj[item + ""] = currentCommand!.properties[index + 1];
                }
                return obj;
            }, {}),
        });
        currentCommand = null;
    }

    /** When one part of the command is committed */
    function onCommit(section: CommandSection, text: string | number | boolean | null, index: number) {
        if (!currentCommand) return;
        if (section === "command") {
            currentCommand.command = text + "";
            currentCommand.end = index;
        } else if (section === "action") {
            currentCommand.action = text + "";
            currentCommand.end = index;
        } else if (section === "propName") {
            currentCommand?.properties.push(text);
        } else if (section === "propValue") {
            currentCommand?.properties.push(text);
            currentCommand.end = typeof text === "string" ? index + 1 : index; // Add 1 for quote
        }
    }
    function onStart(start: number) {
        currentCommand = {
            command: "",
            action: null,
            properties: [],
            start,
            end: start,
        };
    }
    function onCancel() {
        currentCommand = null;
    }

    let section: CommandSection = "outside";
    let buffer: string[] = [];
    let hasOpenBracket = false;
    for (let i = 0; i < inputString.length; i++) {
        const curr = handleTaskTransition({
            curr: inputString[i] as string,
            prev: i > 0 ? inputString[i - 1] as string : "\n", // Pretend there's a newline at the beginning
            section,
            buffer,
            hasOpenBracket,
            onCommit: (section, text) => onCommit(section, text, i),
            onComplete,
            onStart: () => onStart(i),
            onCancel,
        });
        section = curr.section;
        buffer = curr.buffer;
        hasOpenBracket = curr.hasOpenBracket;
    }
    // Call with newline to commit the last command
    handleTaskTransition({
        curr: "\n",
        prev: inputString.length > 0 ? inputString[inputString.length - 1] as string : "\n",
        section,
        buffer,
        hasOpenBracket,
        onCommit: (section, text) => onCommit(section, text, inputString.length),
        onComplete,
        onCancel,
        onStart: () => onStart(inputString.length),
    });

    return commands;
}

/**
 * Extracts tasks from a given string (in "json" mode) based on predefined formats.
 * 
 * The function attempts to JSON parse the input string and extract tasks from the resulting object. It supports the following 
 * formats:
 * - An object with a `task` key containing the task name, an optional `action` key containing the action name, and an optional
 *  `properties` key containing an object with property names and values.
 * - An array of objects with the same structure as above.
 * - The same structures as above, but wrapped in a `task` or `tasks` key.
 * - An object with the properties as keys and values, in which case we'll use the provided task mode to set the task name and action.
 * @param inputString The string containing one or more taaks to be parsed.
 * @param taskMode The current task mode, which determines the tasks we're allowed to start next. 
 * This typically means only one task is allowed (e.g. "RoutineAdd" only lets you start the "RoutineAdd" task, which is 
 * why our fallback behavior of using the taskMode is usually fine).
 * @param commandToTask A function for converting a command and action to a valid task name.
 */
export function extractTasksFromJson(
    inputString: string,
    taskMode: LlmTask | `${LlmTask}`,
    commandToTask: CommandToTask,
): MaybeLlmTaskInfo[] {
    let start = 0;
    let end = inputString.length;
    let tasks: Pick<CurrentLlmTaskInfo, "command" | "action" | "properties">[] = [];

    try {
        // Remove any text before and after the JSON
        start = inputString.indexOf("{");
        end = inputString.lastIndexOf("}") + 1;
        // If the curly brackets are preceeded by square brackets, adjust the indexes (this means it's an array)
        if (inputString[start - 1] === "[" && inputString[end] === "]") {
            start--;
            end++;
        }
        const jsonString = inputString.slice(start, end);
        const inputData = JSON.parse(jsonString);

        if (Array.isArray(inputData)) {
            tasks = inputData;
        } else if (Object.prototype.hasOwnProperty.call(inputData, "task")) {
            tasks = Array.isArray(inputData.task) ? inputData.task : [inputData.task];
        } else if (Object.prototype.hasOwnProperty.call(inputData, "tasks")) {
            tasks = Array.isArray(inputData.tasks) ? inputData.tasks : [inputData.tasks];
        } else {
            tasks = [inputData];
        }
    } catch (e) {
        return [];
    }

    // Task data is not always in the correct format, so we'll try our best to extract the tasks
    return tasks.map((task) => {
        // Assume that "command", "action", and "properties" may be available at the top level. 
        const { command, action, properties, ...otherProperties } = task;
        // Any other top-level properties are assumed to be properties
        const taskProperties = properties || otherProperties;
        // Remove "command" and "action" from the properties
        delete (taskProperties as unknown as { command: unknown }).command;
        delete (taskProperties as unknown as { action: unknown }).action;

        // If it doesn't have any properties or command or action, we'll assume it's invalid
        if (!Object.keys(taskProperties).length && !command && !action) {
            return undefined;
        }

        // Otherwise, we'll return the task data
        return {
            properties: Object.keys(taskProperties).length > 0 ? taskProperties : undefined,
            task: (command && action) ? commandToTask(command, action) : taskMode,
            start,
            end,
        };
    }).filter(Boolean) as unknown as MaybeLlmTaskInfo[];
}

/**
 * Removes tasks from a string
 * @param inputString The string containing the tasks to be removed
 * @param tasks The tasks to be removed
 * @returns The string with the tasks removed
 */
export function removeTasks(inputString: string, tasks: PartialTaskInfo[]): string {
    // Start with the full input string
    let resultString = inputString;

    // Iterate through the tasks array backwards
    for (let i = tasks.length - 1; i >= 0; i--) {
        const command = tasks[i];
        if (!command) continue;

        // Use the start and end indices to slice the string and remove the command
        resultString = resultString.slice(0, command.start) + resultString.slice(command.end);
    }

    return resultString;
}

/**
 * Filters out invalid tasks based on the current task mode and language.
 * 
 * @param potentialTasks The array of potential tasks extracted from the input string.
 * @param taskMode The current task mode, which determines the tasks that can be triggered next.
 * @param existingData Data we already have for the task (e.g. if doing autofill, fields already filled in), 
 * which should be omitted when checking for required properties.
 * @param commandToTask A function for converting a command and action to a valid task name.
 * @param language The language the potential tasks were generated in.
 * @param logger The logger to use for logging errors.
 * @returns An array of valid tasks after filtering out the invalid ones, and removing any invlaid 
 * actions and properties.
 */
export async function filterInvalidTasks(
    potentialTasks: MaybeLlmTaskInfo[],
    taskMode: LlmTask | `${LlmTask}`,
    existingData: ExistingTaskData | null,
    commandToTask: CommandToTask,
    language: string,
    logger: PassableLogger,
): Promise<PartialTaskInfo[]> {
    const result: PartialTaskInfo[] = [];

    // Get task configuration
    const taskConfig = await getUnstructuredTaskConfig(taskMode, language, logger);

    // Determine available tasks based on the task configuration.
    // Rules:
    // 1. If there are 0-1 actions and one command, then the command is the same as the task mode. 
    // E.g. "RoutineAdd" only allows the "RoutineAdd" task. Its config looks like { commands: { add: "lorem ipsum" } }
    // 2. If there is more than one action or more than one command, then the task can trigger multiple tasks. 
    // In this case, we'll need to use `commandToTask` over each command/action pair to determine the available tasks.
    const availableActions = Array.isArray(taskConfig.actions) ? taskConfig.actions : [];
    const availableCommands = (typeof taskConfig.commands === "object" && taskConfig.commands !== null && !Array.isArray(taskConfig.commands))
        ? Object.keys(taskConfig.commands)
        : [];
    const availableTasks: (LlmTask | `${LlmTask}`)[] = [];
    for (const command of availableCommands) {
        for (const action of availableActions) {
            const currentTask = commandToTask(command, action);
            if (currentTask) {
                availableTasks.push(currentTask);
            }
        }
    }
    if (availableTasks.length === 0) {
        availableTasks.push(taskMode);
    }

    // Loop through each potential task
    for (const task of potentialTasks) {
        // Store copy so we can modify it
        const modifiedTask = { ...task };
        const existingKeys = existingData ? Object.keys(existingData) : [];

        // If the task is not a valid task, skip it
        if (!task.task) modifiedTask.task = taskMode;
        else if (!availableTasks.includes(task.task)) continue;

        // If the command has properties, filter out the invalid ones and check for required properties
        if (task.properties) {
            // Remove the properties so that we can add the valid ones back
            modifiedTask.properties = {};

            // Identify all required properties from the task configuration to ensure they are present
            const requiredProperties = taskConfig.properties
                ?.filter(prop => typeof prop === "object" && prop.is_required && !existingKeys.includes(prop.name))
                .map(prop => (prop as LlmTaskProperty).name) ?? [];

            // Check each property in the command and not already filled in
            Object.entries(task.properties).forEach(([key, value]) => {
                const propertyConfig = taskConfig.properties
                    ?.find(prop => {
                        const propName = typeof prop === "object" ? prop.name : prop;
                        return propName === key && !existingKeys.includes(propName);
                    });

                // If the property config exists, keep the property
                if (propertyConfig) {
                    modifiedTask.properties![key] = value;
                    // Remove the property from requiredProperties if it's present and valid, indicating it's not missing
                    const requiredIndex = requiredProperties.indexOf(key);
                    if (requiredIndex !== -1) requiredProperties.splice(requiredIndex, 1);
                }
            });

            // After checking all properties, if any required properties are still missing, mark the task as having missing required properties
            if (requiredProperties.length > 0) {
                logger.error(`Task "${modifiedTask.task}" is missing required properties: ${requiredProperties.join(", ")}`, { trace: "0045" });
            }
            // Otherwise, the task is valid
            else {
                result.push(modifiedTask as PartialTaskInfo);
            }
        } else {
            // If the task has no properties, it's still valid as long as 
            // it doesn't require any properties not already filled in
            const requiresProperties = taskConfig.properties ?
                taskConfig.properties.some(prop => typeof prop === "object" && prop.is_required && !existingKeys.includes(prop.name))
                : false;
            if (!requiresProperties) {
                result.push(modifiedTask as PartialTaskInfo);
            }
        }
    }

    return result;
}

/**
 * Detects tasks wrapped in square brackets, with the specified start substring and delimiter.
 * If a delimiter is not provided, then the wrapper is assumed to contain only one command.
 *
 * @returns An array of indices representing the tasks that are marked as suggested.
 */
export function detectWrappedTasks({
    start,
    delimiter,
    commands,
    messageString,
}: {
    start: string,
    delimiter?: string | null,
    commands: MaybeLlmTaskInfo[],
    messageString: string,
}): WrappedTasks[] {
    const result: WrappedTasks[] = [];
    const MAX_WHITESPACE_LENGTH = 5;
    const allowMultiple = typeof delimiter === "string" && delimiter.length;
    let startWrapperIndex = -1; // Index of the start substring for the current wrapper
    let lastValidIndex = -1; // Last index that matches current wrapper pattern

    for (const command of commands) {
        const { start: startIndex, end: endIndex } = command;

        // If we haven't found the start substring yet, search for it
        if (lastValidIndex < 0) {
            // Look for open bracket
            const openBracketIndex = findCharWithLimit(startIndex, false, "[", messageString, MAX_WHITESPACE_LENGTH);
            if (openBracketIndex === null) {
                lastValidIndex = -1;
                continue;
            }
            // Look for colon
            const colonIndex = findCharWithLimit(openBracketIndex, false, ":", messageString, MAX_WHITESPACE_LENGTH);
            if (colonIndex === null) {
                lastValidIndex = -1;
                continue;
            }
            // Check if the start substring is before the colon
            const possibleStartSubstring = messageString.substring(colonIndex - start.length, colonIndex);
            if (possibleStartSubstring !== start) {
                lastValidIndex = -1;
                continue;
            }
            // Set the last valid index and start wrapper index
            lastValidIndex = colonIndex;
            startWrapperIndex = colonIndex - start.length;
        }
        // Otherwise, make sure space between the last valid index and the current start index is whitespace
        else if (startIndex - lastValidIndex > MAX_WHITESPACE_LENGTH || messageString.substring(lastValidIndex + 1, startIndex).split("").some(c => !isWhitespace(c))) {
            lastValidIndex = -1;
            continue;
        }

        // Search for closing bracket from the command end
        const closingBracketIndex = findCharWithLimit(endIndex, true, "]", messageString, MAX_WHITESPACE_LENGTH);
        if (closingBracketIndex === null) {
            // If we allow multiple commands, check for delimiter
            if (allowMultiple) {
                const delimiterIndex = findCharWithLimit(endIndex, true, delimiter, messageString, MAX_WHITESPACE_LENGTH);
                if (delimiterIndex === null) {
                    lastValidIndex = -1;
                    continue;
                }
                // Set the last valid index to the current index
                lastValidIndex = delimiterIndex;
            }
            // Otherwise, the wrapper is invalid
            else {
                lastValidIndex = -1;
                continue;
            }
        }
        else {
            // If last index of command is actually a newline, cancel the wrapper
            if (messageString[endIndex] === "\n") {
                lastValidIndex = -1;
                continue;
            }
            // Commit all commands between the start index and the closing bracket index
            const taskIndices = commands.map((c, i) => c.start >= startWrapperIndex && c.end <= closingBracketIndex ? i : -1).filter(i => i >= 0) as number[];
            result.push({ taskIndices, wrapperStart: startWrapperIndex, wrapperEnd: closingBracketIndex });
        }
    }

    return result;
}

/**
 * `getValidTasksFromMessage` implementation for responses in "text" mode
 */
export async function getValidTasksFromText({
    commandToTask,
    existingData,
    language,
    logger,
    message,
    taskMode,
}: Omit<GetValidTasksFromMessageParams, "mode">): Promise<GetValidTasksFromMessageResult> {
    // Extract all possible commands from the message
    const maybeTasks = extractTasksFromText(message, commandToTask);
    // Filter out commands that are not valid for the current task
    const validTasks = await filterInvalidTasks(maybeTasks, taskMode, existingData, commandToTask, language, logger);

    // Add additional information to the valid tasks
    const config = await importConfig(language, logger);
    const labelledTasks = validTasks.map(command => ({
        ...command,
        label: config[command.task]().label,
        taskId: `task-${uuid()}`,
    })) as (PartialTaskInfo & { label: string, taskId: string })[];


    // Find commands that are being suggested rather than run right away
    const wrappedCommandList = detectWrappedTasks({
        start: config.__suggested_prefix,
        delimiter: ",",
        commands: labelledTasks,
        messageString: message,
    });
    // Only use one wrapper, as there should only be one suggested section
    const { taskIndices: suggestedIndices, wrapperStart, wrapperEnd } = wrappedCommandList.length > 0
        ? wrappedCommandList[wrappedCommandList.length - 1] as WrappedTasks
        : { taskIndices: [], wrapperStart: -1, wrapperEnd: -1 };
    const tasksToSuggest = suggestedIndices
        .map(i => labelledTasks[i])
        // Cannot suggest the current task or the "Start" task
        .filter((task) => task && !(task.task === taskMode || task.task === "Start")) as (PartialTaskInfo & { label: string, taskId: string })[];

    // find commands that should be run right away
    const tasksToRun = labelledTasks.filter((_, index) => !suggestedIndices.includes(index));

    // Remove suggested commands from the message
    let messageWithoutTasks = message;
    if (wrapperStart >= 0 && wrapperEnd >= 0) {
        messageWithoutTasks = messageWithoutTasks.slice(0, wrapperStart) + messageWithoutTasks.slice(wrapperEnd + 1);
    }
    // Remove the commands that should be run right away
    messageWithoutTasks = removeTasks(messageWithoutTasks, tasksToRun);
    // Trim the message
    messageWithoutTasks = messageWithoutTasks.trim();

    // Return the valid commands and the message without the commands
    return { tasksToRun, tasksToSuggest, messageWithoutTasks };
}

/**
 * `getValidTasksFromMessage` implementation for responses in "json" mode. 
 * 
 * NOTE: This version is intended to be used when we only want structured data, and nothing else. 
 * This means we can treat the whole message as a single task to run, and set tasksToSuggest and 
 * messageWithoutTasks as empty values.
 */
export async function getValidTasksFromJson({
    commandToTask,
    existingData,
    language,
    logger,
    message,
    taskMode,
}: Omit<GetValidTasksFromMessageParams, "mode">): Promise<GetValidTasksFromMessageResult> {
    // Extract all possible commands from the message
    const maybeTasks = extractTasksFromJson(message, taskMode, commandToTask);
    // Filter out commands that are not valid for the current task
    const validTasks = await filterInvalidTasks(maybeTasks, taskMode, existingData, commandToTask, language, logger);

    // Add additional information to the valid tasks
    const config = await importConfig(language, logger);
    const labelledTasks = validTasks.map(command => ({
        ...command,
        label: config[command.task]().label,
        taskId: `task-${uuid()}`,
    })) as (PartialTaskInfo & { label: string, taskId: string })[];

    // We only case about `tasksToRun` in "json" mode
    return { tasksToRun: labelledTasks, tasksToSuggest: [], messageWithoutTasks: "" };
}

/**
 * Extracts valid tasks from a message and returns the message without the tasks.
 * @returns An object containing tasks that should be run right away, suggested tasks, 
 * and the message without the tasks
 */
export async function getValidTasksFromMessage({
    mode,
    ...rest
}: GetValidTasksFromMessageParams): Promise<GetValidTasksFromMessageResult> {
    if (mode === "text") return await getValidTasksFromText(rest);
    if (mode === "json") return await getValidTasksFromJson(rest);
    throw new Error("Invalid mode provided");
}
