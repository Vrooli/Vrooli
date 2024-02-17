export type LlmCommand = {
    command: string;
    action: string | null;
    properties: {
        [key: string]: string | number | null;
    } | null;
    start: number;
    end?: number;
}
/** Properties stored as an array of [key, value, key, value, ...] */
type CurrentLlmCommand = Omit<LlmCommand, "properties"> & {
    properties: (string | number | null)[];
};

type CommandSection = "outside" | "command" | "action" | "propName" | "propValue";

/** Maximum length for a command, action, or property name */
const MAX_COMMAND_ACTION_PROP_LENGTH = 32;
const QUOTES = ['"', "'"];

export const isNewline = (char: string) => char === '\n' || char === '\r';
export const isWhitespace = (char: string): boolean => char === ' ' || char === '\t';
export const isAlphaNum = (char: string): boolean => {
    const code = char.charCodeAt(0);
    return (code > 47 && code < 58) || // numeric (0-9)
        (code > 64 && code < 91) || // upper alpha (A-Z)
        (code > 96 && code < 123);  // lower alpha (a-z)
};


function countEndSlashes(charArray: string[]): number {
    let count = 0;
    for (let i = charArray.length - 1; i >= 0; i--) {
        if (charArray[i] === '\\') {
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
        const backslashes = countEndSlashes(buffer)
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
        const maybeNumber = !isWhitespace(curr) && !isNewline(curr) && ((buffer.length === 0 && ["-", "."].includes(curr)) || !Number.isNaN(+withCurr));
        const maybeNull = "null".startsWith(withCurr);
        if (maybeNumber || maybeNull) {
            buffer.push(curr);
            return { section, buffer };
        }
        // If we reached whitespace or a newline
        if (isWhitespace(curr) || isNewline(curr)) {
            // We've already accounted for quotes (i.e. strings). So we can only commit 
            // if it's a valid number or null
            const isNumber = !Number.isNaN(+bufferString) && buffer.length > 0;
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
 * @returns An array of Command objects, each containing the command, action (if any), and an object of properties (if any).
 */
export const extractCommands = (inputString: string): LlmCommand[] => {
    const commands: LlmCommand[] = [];
    let currentCommand: CurrentLlmCommand | null = null;

    /** When the full command is completed */
    const onComplete = () => {
        if (!currentCommand) return;
        commands.push({
            ...currentCommand,
            action: (currentCommand.action && currentCommand.action.length) ? currentCommand.action : null,
            // Convert properties array to object
            properties: currentCommand.properties.reduce((obj, item, index) => {
                if (index % 2 === 0) { // Even index, property name
                    // If it's the last element and there's no corresponding value, skip it
                    if (index === currentCommand!.properties.length - 1) return obj;
                    obj[item + ""] = currentCommand!.properties[index + 1];
                }
                return obj;
            }, {}),
        })
        currentCommand = null;
    };

    /** When one part of the command is committed */
    const onCommit = (section: CommandSection, text: string | number | null, index: number) => {
        if (!currentCommand) return;
        if (section === 'command') {
            currentCommand.command = text + "";
            currentCommand.end = index;
        } else if (section === 'action') {
            currentCommand.action = text + "";
            currentCommand.end = index;
        } else if (section === 'propName') {
            currentCommand?.properties.push(text);
        } else if (section === 'propValue') {
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
        }
    }
    const onCancel = () => {
        currentCommand = null;
    }

    let section: CommandSection = 'outside';
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
