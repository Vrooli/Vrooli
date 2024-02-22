import { logger } from "../../events/logger";
import { CommandToTask, LlmCommandProperty, LlmTask, LlmTaskUnstructuredConfig, llmTasks } from "./config";

export type LlmCommand = {
    task: LlmTask;
    command: string;
    action: string | null;
    properties: {
        [key: string]: string | number | null;
    } | null;
    start: number;
    end: number;
};
export type MaybeLlmCommand = Omit<LlmCommand, "task"> & {
    task: LlmTask | null;
};
/** Properties stored as an array of [key, value, key, value, ...] */
type CurrentLlmCommand = Omit<LlmCommand, "properties" | "task"> & {
    properties: (string | number | null)[];
};

type CommandSection = "outside" | "command" | "action" | "propName" | "propValue";

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
}

/**
 * Handles the transition between parsing a command, action, property name, or property value. 
 * 
 * This is not meant to be the shortest possible function, but rather one that's the easiest to understand and 
 * maintain. The outer if-else structure which handles each section type chronologically, with each section
 * handling the transition between sections, invalid characters, and other edge cases.
 * @returns An object containing the new parsing section and buffer
 */
export const handleCommandTransition = ({
    curr,
    prev,
    section,
    buffer,
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
        // Start a command if there's a slash without a previous character
        if (curr === "/" && (!prev || isWhitespace(prev) || isNewline(prev))) {
            onStart();
            return { section: "command", buffer: [] }; // Don't include the slash in the buffer
        }
    }
    else if (section === "command") {
        // If we run into another slash, the command is invalid
        if (curr === "/") {
            onCancel();
            return { section: "outside", buffer: [] };
        }
        // Handle transition from command to action (might actually be property name,
        // but we're not sure yet
        if (isWhitespace(curr)) {
            onCommit("command", buffer.join(""));
            return { section: "action", buffer: [] };
        }
        // Commit on newline
        if (isNewline(curr)) {
            onCommit("command", buffer.join(""));
            onComplete();
            return { section: "outside", buffer: [] };
        }
        // Cancel on non alpha-numeric characters
        if (!isAlphaNum(curr)) {
            onCancel();
            return { section: "outside", buffer: [] };
        }
        // Cancel if the buffer is too long
        if (buffer.length >= MAX_COMMAND_ACTION_PROP_LENGTH) {
            onCancel();
            return { section: "outside", buffer: [] };
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
            return { section: "outside", buffer: [] };
        }
        // Handle whitespace
        if (isWhitespace(curr)) {
            // Skip if buffer is empty
            if (buffer.length === 0) return { section, buffer };
            // Othwerwise, commit as an action
            onCommit("action", buffer.join(""));
            return { section: "propName", buffer: [] };
        }
        // Handle slash
        if (curr === "/") {
            // If previous character is not a whitespace, it's not a valid action 
            // or start of a new command
            if (!isWhitespace(prev)) {
                onComplete();
                return { section: "outside", buffer: [] };
            }
            // If there's a buffer, commit it as an action
            if (buffer.length > 0) {
                onCommit("action", buffer.join(""));
            }
            // Start a new command
            onComplete();
            onStart();
            return { section: "command", buffer: [] };
        }
        // If it's an equals sign, commit as a property name 
        // instead of an action
        if (curr === "=") {
            onCommit("propName", buffer.join(""));
            return { section: "propValue", buffer: [] };
        }
        // Complete if it's not an alpha-numeric character
        if (!isAlphaNum(curr)) {
            // Commit action if previous character was whitespace
            if (isWhitespace(prev)) {
                onCommit("action", buffer.join(""));
            }
            onComplete();
            return { section: "outside", buffer: [] };
        }
        // Complete if the buffer is too long
        if (buffer.length >= MAX_COMMAND_ACTION_PROP_LENGTH) {
            onComplete();
            return { section: "outside", buffer: [] };
        }
    }
    else if (section === "propName") {
        // Commit on equals sign
        if (curr === "=") {
            onCommit("propName", buffer.join(""));
            return { section: "propValue", buffer: [] };
        }
        // Handle slash
        if (curr === "/") {
            // If previous character is not a whitespace, it's not a valid action 
            // or start of a new command
            if (!isWhitespace(prev)) {
                onComplete();
                return { section: "outside", buffer: [] };
            }
            // Start a new command. Do not commit the buffer, as it won't have an accompanying value
            onComplete();
            onStart();
            return { section: "command", buffer: [] };
        }
        // Handle whitespace
        if (isWhitespace(curr)) {
            // Skip if buffer is empty
            if (buffer.length === 0) return { section, buffer };
            // Otherwise, complete command
            if (buffer.length > 0) {
                onComplete();
                return { section: "outside", buffer: [] };
            }
        }
        // Complete on newline or non alpha-numeric characters
        if (isNewline(curr) || !isAlphaNum(curr)) {
            onComplete();
            return { section: "outside", buffer: [] };
        }
        // Complete if the buffer is too long
        if (buffer.length >= MAX_COMMAND_ACTION_PROP_LENGTH) {
            onComplete();
            return { section: "outside", buffer: [] };
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
            return { section: "propName", buffer: [] };
        }
        // Check if entering quotes
        if (!isInQuote && isQuote && buffer.length === 0) {
            return { section: "propValue", buffer: [curr] };
        }
        // Allow anything inside quotes
        if (isInQuote) {
            buffer.push(curr);
            return { section, buffer };
        }
        // Only numbers and null are allowed outside quotes
        const bufferString = buffer.join("");
        const withCurr = bufferString + curr;
        const maybeNumber = !isWhitespace(curr) && !isNewline(curr) && ((buffer.length === 0 && ["-", "."].includes(curr)) || Number.isFinite(+withCurr));
        const maybeNull = "null".startsWith(withCurr);
        if (maybeNumber || maybeNull) {
            buffer.push(curr);
            return { section, buffer };
        }
        // If we reached whitespace or a newline
        if (isWhitespace(curr) || isNewline(curr)) {
            // We've already accounted for quotes (i.e. strings). So we can only commit 
            // if it's a valid number or null
            const isNumber = Number.isFinite(+bufferString) && buffer.length > 0;
            const isNull = bufferString === "null";
            if (isNumber || isNull) {
                onCommit("propValue", isNumber ? +bufferString : null);
                if (isNewline(curr)) {
                    onComplete();
                    return { section: "outside", buffer: [] };
                }
                return { section: "propName", buffer: [] };
            }
        }
        // Otherwise it's invalid, so complete the command
        onComplete();
        return { section: "outside", buffer: [] };
    }

    // Otherwise, continue buffering
    buffer.push(curr);
    return { section, buffer };
};

/**
 * Extracts commands from a given string based on predefined formats.
 * 
 * The function searches for patterns that match commands in the format "/command action property1=value1 property2=value2".
 * It supports multiple commands within the same string, separated by spaces or other commands.
 * 
 * NOTE: This may capture commands (and command actions/properties) that are not valid. These should 
 * be ignored when comparing them to the available commands.
 * 
 * @param inputString The string containing one or more commands to be parsed.
 * @param commandToTask A function for converting a command and action to a valid task name.
 * @returns An array of Command objects, each containing the command, action (if any), an object of properties (if any), 
 * and the the task the command is associated with (if valid).
 */
export const extractCommands = (inputString: string, commandToTask: CommandToTask): MaybeLlmCommand[] => {
    const commands: MaybeLlmCommand[] = [];
    let currentCommand: CurrentLlmCommand | null = null;

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
    for (let i = 0; i < inputString.length; i++) {
        const curr = handleCommandTransition({
            curr: inputString[i],
            prev: i > 0 ? inputString[i - 1] : "\n", // Pretend there's a newline at the beginning
            section,
            buffer,
            onCommit: (section, text) => onCommit(section, text, i),
            onComplete,
            onStart: () => onStart(i),
            onCancel,
        });
        section = curr.section;
        buffer = curr.buffer;
    }
    // Call with newline to commit the last command
    handleCommandTransition({
        curr: "\n",
        prev: inputString.length > 0 ? inputString[inputString.length - 1] : "\n",
        section,
        buffer,
        onCommit: (section, text) => onCommit(section, text, inputString.length),
        onComplete,
        onCancel,
        onStart: () => onStart(inputString.length),
    });

    return commands;
};

/**
 * Filters out invalid commands based on the task and language.
 * 
 * @param potentialCommands The array of potential commands extracted from the input string.
 * @param taskConfig The unstructured task configuration object, containing information about 
 * the available commands, actions, and properties.
 * @returns An array of valid commands after filtering out the invalid ones, and removing any invlaid 
 * actions and properties.
 */
export const filterInvalidCommands = async (
    potentialCommands: MaybeLlmCommand[],
    taskConfig: LlmTaskUnstructuredConfig,
): Promise<LlmCommand[]> => {
    const result: LlmCommand[] = [];

    // Loop through each potential command
    for (const command of potentialCommands) {
        const modifiedCommand = { ...command };

        // If the task is not a valid task, skip it
        if (!command.task || !llmTasks.includes(command.task)) continue;

        // If the command is not in the task configuration, skip it
        if (!Object.prototype.hasOwnProperty.call(taskConfig.commands, command.command)) continue;

        // If the command has an invalid action, remove the action
        if (command.action && (!taskConfig.actions || !taskConfig.actions.includes(command.action))) {
            modifiedCommand.action = null;
        }

        // If the command has properties, filter out the invalid ones and check for required properties
        if (command.properties) {
            // Remove the properties so that we can add the valid ones back
            modifiedCommand.properties = {};

            // Identify all required properties from the task configuration to ensure they are present
            const requiredProperties = taskConfig.properties?.filter(prop => typeof prop === "object" && prop.is_required).map(prop => (prop as LlmCommandProperty).name) ?? [];

            // Check each property in the command
            Object.entries(command.properties).forEach(([key, value]) => {
                const propertyConfig = taskConfig.properties?.find(prop => (typeof prop === "object" ? prop.name : prop) === key);

                // If the property config exists, keep the property
                if (propertyConfig) {
                    modifiedCommand.properties![key] = value;
                    // Remove the property from requiredProperties if it's present and valid, indicating it's not missing
                    const requiredIndex = requiredProperties.indexOf(key);
                    if (requiredIndex !== -1) requiredProperties.splice(requiredIndex, 1);
                }
            });

            // After checking all properties, if any required properties are still missing, mark the command as having missing required properties
            if (requiredProperties.length > 0) {
                logger.error(`Command "${command.command}" is missing required properties: ${requiredProperties.join(", ")}`, { trace: "0045" });
            }
            // Otherwise, the command is valid
            else {
                result.push(modifiedCommand as LlmCommand);
            }
        } else {
            // If the command has no properties, it's still valid as long as it doesn't require any
            const requiresProperties = taskConfig.properties?.some(prop => typeof prop === "object" && prop.is_required) ?? false;
            if (!requiresProperties) {
                result.push(modifiedCommand as LlmCommand);
            }
        }
    }

    return result;
};

/**
 * Removes commands from a string
 * @param inputString The string containing the commands to be removed
 * @param commands The commands to be removed
 * @returns The string with the commands removed
 */
export const removeCommands = (inputString: string, commands: LlmCommand[]): string => {
    // Start with the full input string
    let resultString = inputString;

    // Iterate through the commands array backwards
    for (let i = commands.length - 1; i >= 0; i--) {
        const command = commands[i];

        // Use the start and end indices to slice the string and remove the command
        resultString = resultString.slice(0, command.start) + resultString.slice(command.end);
    }

    return resultString;
};
