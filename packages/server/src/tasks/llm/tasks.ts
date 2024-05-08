import { BotSettings, LlmTask, LlmTaskInfo, uuid } from "@local/shared";
import { logger } from "../../events/logger";
import { PreMapUserData } from "../../models/base/chatMessage";
import { SessionUserToken } from "../../types";
import { CommandToTask, LlmTaskProperty, getUnstructuredTaskConfig, importConfig } from "./config";
import { generateResponseWithFallback } from "./service";

export type ExistingTaskData = Record<string, string | number | boolean | null>;

export type PartialTaskInfo = Omit<LlmTaskInfo, "label" | "status"> & {
    start: number;
    end: number;
};
export type MaybeLlmTaskInfo = Omit<PartialTaskInfo, "id" | "task"> & {
    task: LlmTask | `${LlmTask}` | null;
};
/** Properties stored as an array of [key, value, key, value, ...] */
type CurrentLlmTaskInfo = Omit<PartialTaskInfo, "properties" | "task"> & {
    properties: (string | number | null)[];
};

export type CommandSection = "outside" | "code" | "command" | "action" | "propName" | "propValue";

/** Maximum length for a command, action, or property name */
const MAX_COMMAND_ACTION_PROP_LENGTH = 32;
const QUOTES = ["\"", "'"];

export const isNewline = (char: string) => char === "\n" || char === "\r";
export const isWhitespace = (char: string): boolean => char === " " || char === "\t";
export const isAlphaNum = (char: string): boolean => {
    const code = char.charCodeAt(0);
    return (code > 47 && code < 58) || // numeric (0-9)
        (code > 64 && code < 91) || // upper alpha (A-Z)
        (code > 96 && code < 123);  // lower alpha (a-z)
};


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

type CommandTransitionTrack = {
    section: CommandSection,
    buffer: string[],
    /** If true, command will be canceled later if we don't find a closing bracket */
    hasOpenBracket: boolean,
}

export type CommandWrapper = {
    label: string,
    start: string,
    end: string,
    delimiter: string,
    allowMultiple: boolean,
}

/**
 * Handles the transition between parsing a command, action, property name, or property value. 
 * 
 * This is not meant to be the shortest possible function, but rather one that's the easiest to understand and 
 * maintain. The outer if-else structure which handles each section type chronologically, with each section
 * handling the transition between sections, invalid characters, and other edge cases.
 * @returns An object containing the new parsing section and buffer
 */
export const handleTaskTransition = ({
    curr,
    prev,
    section,
    buffer,
    hasOpenBracket,
    onCommit,
    onComplete,
    onCancel,
    onStart,
}: CommandTransitionTrack & {
    curr: string,
    prev: string,
    onCommit: (section: CommandSection, text: string | number | null) => unknown,
    onComplete: () => unknown, // When a full command (including any actions/props) is completed
    onCancel: () => unknown, // When a full command is cancelled. Typically only when there's a problem with the beginning slash command 
    onStart: () => unknown, // When a full command is started
}): CommandTransitionTrack => {
    // Handle each section type
    if (section === "outside") {
        // Start a command if there's a slash without a previous character, 
        // or if the previous character is whitespace, newline, or open bracket (for wrapped commands)
        if (curr === "/" && (!prev || isWhitespace(prev) || isNewline(prev) || prev === "[")) {
            onStart();
            return {
                section: "command",
                buffer: [], // Don't include the slash in the buffer
                hasOpenBracket: hasOpenBracket || prev === "[", // Keep bracket status, or start a new one if the open bracket was found
            };
        }
        // Reset buffer when there's whitespace=
        if (isWhitespace(curr)) {
            return { section, buffer: [], hasOpenBracket };
        }
        if (isNewline(curr)) {
            return { section, buffer: [], hasOpenBracket: false };
        }
        // Reset buffer and hasOpenBracket when there's a newline
        // Start a code block if the buffer + curr is "`" or "```" or "<code>"
        if (
            (buffer.length === 0 && curr === "`") ||
            (buffer.length === 2 && buffer[0] === "`" && buffer[1] === "`" && curr === "`") ||
            (buffer.length === 5 && buffer[0] === "<" && buffer[1] === "c" && buffer[2] === "o" && buffer[3] === "d" && buffer[4] === "e" && curr === ">")
        ) {
            buffer.push(curr);
            return { section: "code", buffer, hasOpenBracket }; // Keep in buffer so we can match the closing backticks/tag
        }
    }
    else if (section === "code") {
        // Shouldn't be here if the buffer is empty
        if (buffer.length === 0) {
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
        } else if (buffer.length === 1 && buffer[0] === "`") {
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
            if (buffer.length === 3 && buffer.every(b => b === "`") && curr === "`") {
                onCancel();
                return { section: "outside", buffer: [], hasOpenBracket };
            }
            // Cancel when closing triple backticks found
            if (buffer.length > 3 && curr === "`" && buffer[buffer.length - 1] === "`" && buffer[buffer.length - 2] === "`") {
                onCancel();
                return { section: "outside", buffer: [], hasOpenBracket };
            }
        }
        // If we're in a tag code block
        if (codeType === "tag") {
            // When the buffer is short, stop if the tag never forms
            if (buffer.length < 6) {
                // Cancel if we run into any characters not in "<code>"
                if (!"<code>".startsWith([...buffer, curr].join(""))) {
                    onCancel();
                    return { section: "outside", buffer: [], hasOpenBracket };
                }
            }
            // When the buffer is longer, stop if the tag is complete
            if (buffer.length >= 13) {
                // Cancel if it end with "</code>"
                if (buffer.slice(-6).join("") === "</code" && curr === ">") {
                    onCancel();
                    return { section: "outside", buffer: [], hasOpenBracket };
                }
            }
        }
    }
    else if (section === "command") {
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
    }
    else if (section === "action") {
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
            if (buffer.length === 0) return { section, buffer, hasOpenBracket };
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
                onCommit("action", buffer.join(""));
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
    }
    else if (section === "propName") {
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
            if (buffer.length === 0) return { section, buffer, hasOpenBracket };
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
    }
    else if (section === "propValue") {
        const backslashes = countEndSlashes(buffer);
        const isEscaped = backslashes % 2 !== 0;
        const isQuote = QUOTES.includes(curr);
        const isInQuote = QUOTES.includes(buffer[0]);
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
            return { section, buffer, hasOpenBracket };
        }
        // Only numbers and null are allowed outside quotes
        const bufferString = buffer.join("");
        const withCurr = bufferString + curr;
        const maybeNumber = !isWhitespace(curr) && !isNewline(curr) && ((buffer.length === 0 && ["-", "."].includes(curr)) || Number.isFinite(+withCurr));
        const maybeNull = "null".startsWith(withCurr);
        if (maybeNumber || maybeNull) {
            buffer.push(curr);
            return { section, buffer, hasOpenBracket };
        }
        // If we reached whitespace, a newline, or a comma or end bracket, we'll probably commit
        if (isWhitespace(curr) || isNewline(curr)) {
            // We've already accounted for quotes (i.e. strings). So we can only commit 
            // if it's a valid number or null
            const isNumber = Number.isFinite(+bufferString) && buffer.length > 0;
            const isNull = bufferString === "null";
            if (isNumber || isNull) {
                onCommit("propValue", isNumber ? +bufferString : null);
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

    // Otherwise, continue buffering
    buffer.push(curr);
    return { section, buffer, hasOpenBracket };
};

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
export const findCharWithLimit = (
    index: number,
    forward: boolean,
    char: string,
    input: string,
    maxLength: number,
): number | null => {
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
        } else if (!isWhitespace(currentChar) && !isNewline(currentChar)) {
            // Found a non-whitespace character that is not the target
            return null;
        }
    }
    return null; // Exceeded the maximum loop count
};

type WrappedTasks = {
    taskIndices: number[],
    wrapperStart: number,
    wrapperEnd: number,
}

/**
 * Detects tasks wrapped in square brackets, with the specified start substring and delimiter.
 * If a delimiter is not provided, then the wrapper is assumed to contain only one command.
 *
 * @returns An array of indices representing the tasks that are marked as suggested.
 */
export const detectWrappedTasks = ({
    start,
    delimiter,
    commands,
    messageString,
}: {
    start: string,
    delimiter?: string | null,
    commands: MaybeLlmTaskInfo[],
    messageString: string,
}): WrappedTasks[] => {
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
};

/**
 * Extracts tasks from a given string based on predefined formats.
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
export const extractTasks = (
    inputString: string,
    commandToTask: CommandToTask,
): MaybeLlmTaskInfo[] => {
    const commands: MaybeLlmTaskInfo[] = [];
    let currentCommand: CurrentLlmTaskInfo | null = null;

    /** When the full command is completed */
    const onComplete = () => {
        if (!currentCommand) return;
        const action = (currentCommand.action && currentCommand.action.length) ? currentCommand.action : null;
        const task = commandToTask(currentCommand.command, action);
        commands.push({
            ...currentCommand,
            task,
            action,
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
    };

    /** When one part of the command is committed */
    const onCommit = (section: CommandSection, text: string | number | null, index: number) => {
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
    };
    const onStart = (start: number) => {
        currentCommand = {
            id: uuid(),
            command: "",
            action: null,
            properties: [],
            start,
            end: start,
        };
    };
    const onCancel = () => {
        currentCommand = null;
    };

    let section: CommandSection = "outside";
    let buffer: string[] = [];
    let hasOpenBracket = false;
    for (let i = 0; i < inputString.length; i++) {
        const curr = handleTaskTransition({
            curr: inputString[i],
            prev: i > 0 ? inputString[i - 1] : "\n", // Pretend there's a newline at the beginning
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
        prev: inputString.length > 0 ? inputString[inputString.length - 1] : "\n",
        section,
        buffer,
        hasOpenBracket,
        onCommit: (section, text) => onCommit(section, text, inputString.length),
        onComplete,
        onCancel,
        onStart: () => onStart(inputString.length),
    });

    return commands;
};

/**
 * Filters out invalid tasks based on the current task mode and language.
 * 
 * @param potentialTasks The array of potential tasks extracted from the input string.
 * @param taskMode The current task mode, which determines the tasks that can be triggered next.
 * @param existingData Data we already have for the task (e.g. if doing autofill, fields already filled in), 
 * which should be omitted when checking for required properties.
 * @param language The language the potential tasks were generated in.
 * @returns An array of valid tasks after filtering out the invalid ones, and removing any invlaid 
 * actions and properties.
 */
export const filterInvalidTasks = async (
    potentialTasks: MaybeLlmTaskInfo[],
    taskMode: LlmTask | `${LlmTask}`,
    existingData: ExistingTaskData | null,
    language: string,
): Promise<PartialTaskInfo[]> => {
    const result: PartialTaskInfo[] = [];

    // Get task configuration
    const taskConfig = await getUnstructuredTaskConfig(taskMode, language);

    // Loop through each potential task
    for (const task of potentialTasks) {
        // Store copy so we can modify it
        const modifiedTask = { ...task };
        const requiresAction = Boolean(taskConfig.actions?.length);
        const existingKeys = existingData ? Object.keys(existingData) : [];

        // If the task is not a valid task, skip it
        if (!task.task) modifiedTask.task = taskMode;
        else if (!Object.keys(LlmTask).includes(task.task)) continue;

        let commandIsValid = Object.prototype.hasOwnProperty.call(taskConfig.commands, task.command);

        // Attempt corrective measures when command or action are invalid.
        if (!commandIsValid) {
            const commandKeys = Object.keys(taskConfig.commands);
            // If the command provided is a valid action, and there's only one command available, 
            // then we can change the provided command to an action, and set the command to the only available command
            // E.g. "/action prop1='hello'" -> "/command action prop1='hello'"
            if (taskConfig.actions && taskConfig.actions.includes(task.command) && commandKeys.length === 1) {
                // Set action to the command
                modifiedTask.action = task.command;
                // Set command to the only command available in the task config
                modifiedTask.command = commandKeys[0];
                commandIsValid = true; // Mark the command as valid now
            }
            // If the command is invalid but matches an action, and this task doesn't have any actions, then 
            // we can change the action to a command, and set the action to null
            // E.g. "/invalid command prop1='hello'" -> "/command prop1='hello'"
            else if (task.action && commandKeys.includes(task.action) && !requiresAction) {
                // Set action to null
                modifiedTask.action = null;
                // Set command to the action
                modifiedTask.command = task.action;
                commandIsValid = true; // Mark the command as valid now
            }
        }

        // Skip if the command is still not valid
        if (!commandIsValid) continue;

        // If the command has an invalid action, then it is invalid
        if (requiresAction && (!modifiedTask.action || !taskConfig.actions?.includes(modifiedTask.action))) {
            continue;
        }
        // If the task config has no actions, make sure the action is null
        else if (!requiresAction && modifiedTask.action) {
            modifiedTask.action = null;
        }

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

            // After checking all properties, if any required properties are still missing, mark the command as having missing required properties
            if (requiredProperties.length > 0) {
                logger.error(`Command "${modifiedTask.command}" is missing required properties: ${requiredProperties.join(", ")}`, { trace: "0045" });
            }
            // Otherwise, the command is valid
            else {
                result.push(modifiedTask as PartialTaskInfo);
            }
        } else {
            // If the command has no properties, it's still valid as long as 
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
};

/**
 * Removes tasks from a string
 * @param inputString The string containing the tasks to be removed
 * @param tasks The tasks to be removed
 * @returns The string with the tasks removed
 */
export const removeTasks = (inputString: string, tasks: PartialTaskInfo[]): string => {
    // Start with the full input string
    let resultString = inputString;

    // Iterate through the tasks array backwards
    for (let i = tasks.length - 1; i >= 0; i--) {
        const command = tasks[i];

        // Use the start and end indices to slice the string and remove the command
        resultString = resultString.slice(0, command.start) + resultString.slice(command.end);
    }

    return resultString;
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
    /** The message containing the tasks */
    message: string,
    /** The current task mode, which determines the tasks we're allowed to start next */
    taskMode: LlmTask | `${LlmTask}`,
}

/**
 * Extracts valid tasks from a message and returns the message without the tasks.
 * @returns An object containing tasks that should be run right away, suggested tasks, 
 * and the message without the tasks
 */
export const getValidTasksFromMessage = async ({
    commandToTask,
    existingData,
    language,
    message,
    taskMode,
}: GetValidTasksFromMessageParams): Promise<{
    tasksToRun: Omit<LlmTaskInfo, "status">[],
    tasksToSuggest: Omit<LlmTaskInfo, "status">[],
    messageWithoutTasks: string,
}> => {
    // Extract all possible commands from the message
    const maybeTasks = extractTasks(message, commandToTask);
    // Filter out commands that are not valid for the current task
    const validTasks = await filterInvalidTasks(maybeTasks, taskMode, existingData, language);

    // Add labels to the valid commands
    const config = await importConfig(language);
    const labelledTasks = validTasks.map(command => ({
        ...command,
        label: config[command.task]().label,
    })) as (PartialTaskInfo & { label: string })[];


    // Find commands that are being suggested rather than run right away
    const wrappedCommandList = detectWrappedTasks({
        start: config.__suggested_prefix,
        delimiter: ",",
        commands: labelledTasks,
        messageString: message,
    });
    // Only use one wrapper, as there should only be one suggested section
    const { taskIndices: suggestedIndices, wrapperStart, wrapperEnd } = wrappedCommandList.length > 0
        ? wrappedCommandList[wrappedCommandList.length - 1]
        : { taskIndices: [], wrapperStart: -1, wrapperEnd: -1 };
    const tasksToSuggest = suggestedIndices
        .map(i => labelledTasks[i])
        // Cannot suggest the current task or the "Start" task
        .filter((task) => !(task.task === taskMode || task.task === "Start"));

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
};

export type ForceGetTaskParams = {
    chatId?: string | null,
    commandToTask: CommandToTask,
    existingData?: ExistingTaskData | null,
    language: string,
    participantsData?: Record<string, PreMapUserData> | null;
    respondingBotConfig: BotSettings,
    respondingBotId: string,
    respondingToMessage: {
        id?: string | null,
        text: string,
    } | null,
    task: LlmTask | `${LlmTask}`,
    userData: SessionUserToken,
}

/**
 * Repeatedly generates a bot response until it contains a valid task.
 * @returns The valid task and the rest of the message without the tasks
 */
export const forceGetTask = async ({
    chatId,
    commandToTask,
    existingData,
    language,
    participantsData,
    respondingBotConfig,
    respondingBotId,
    respondingToMessage,
    task,
    userData,
}: ForceGetTaskParams): Promise<{
    taskToRun: Omit<LlmTaskInfo, "status"> | null,
    tasksToSuggest: Omit<LlmTaskInfo, "status">[],
    messageWithoutTasks: string | null,
    cost: number
}> => {
    let retryCount = 0;
    const MAX_RETRIES = 3; // Set a maximum number of retries to avoid infinite loops
    let totalCost = 0;

    while (retryCount < MAX_RETRIES) {
        const { message, cost } = await generateResponseWithFallback({
            chatId,
            force: true, // Force the bot to respond with a task
            participantsData,
            respondingBotConfig,
            respondingBotId,
            respondingToMessage,
            stream: false,
            task,
            userData,
        });
        totalCost += cost;

        // Check for tasks in the start response
        const { tasksToRun, tasksToSuggest, messageWithoutTasks } = await getValidTasksFromMessage({
            commandToTask,
            existingData: existingData ?? null,
            language,
            message,
            taskMode: task,
        });

        // If a valid task is found, return it
        if (tasksToRun.length > 0) {
            // Only use the first task found
            return { taskToRun: tasksToRun[0], tasksToSuggest, messageWithoutTasks, cost };
        } else {
            // Increment the retry count if no task is found
            retryCount++;
            logger.warning(`No task found in start response. Retrying... (${retryCount}/${MAX_RETRIES})`, { trace: "0349", chatId, respondingBotId, task });
        }
    }

    logger.error("Failed to find a task in start response after maximum retries.", { trace: "0350", chatId, respondingBotId, task });
    return { taskToRun: null, tasksToSuggest: [], messageWithoutTasks: null, cost: totalCost };
};
