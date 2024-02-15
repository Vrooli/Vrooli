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

type CommandSection = "outside" | "pending" | "command" | "action" | "propName" | "propValue";

/** Maximum length for a command, action, or property name */
const MAX_COMMAND_ACTION_PROP_LENGTH = 32;

export const isNewline = (char: string) => char === '\n' || char === '\r';

export const isWhitespace = (char: string) => /^\s$/.test(char) && !isNewline(char);

export const isAlphaNum = (char: string) => /^[A-Za-z0-9]$/.test(char);

type CommandTransitionTrack = {
    section: CommandSection,
    buffer: string,
}

/**
 * Handles the transition between parsing a command, action, property name, or property value. 
 * Calls the callback when a section is completed.
 * @returns An object containing the new parsing section and buffer
 */
export const handleCommandTransition = ({
    curr,
    section,
    buffer,
    callback,
}: CommandTransitionTrack & {
    curr: string,
    callback: (section: CommandSection, text: string | number | null) => unknown,
}): CommandTransitionTrack => {
    const previousChar = buffer[buffer.length - 1];
    // Handle slash, which may indicate the start of a new command
    if (section === "outside" && curr === "/") {
        // If encountered within a word, don't start a new command
        if (previousChar && !isWhitespace(previousChar) && !isNewline(previousChar)) {
            return { section, buffer: buffer + curr };
        }
        // Otherwise, start the command. Don't include the slash in the buffer
        return { section: 'command', buffer: "" };
    }
    // Handle transition from command to pending (action/propName undecided)
    else if (section === 'command' && isWhitespace(curr)) {
        callback('command', buffer);
        return { section: 'pending', buffer: '' };
    }
    // Handle decision between action and propName
    else if (section === 'pending') {
        const isAction = isWhitespace(curr) || isNewline(curr);
        const isPropName = curr === "=";
        if (isAction) {
            callback("action", buffer);
            // There can only be one action, so switch to propName section if not a newline
            return { section: isNewline(curr) ? "outside" : "propName", buffer: "" };
        }
        else if (isPropName) {
            callback("propName", buffer);
            // Next we'll be reading the property value
            return { section: 'propValue', buffer: "" };
        }
    }
    // Handle equals sign
    else if (section === "propName") {
        // Commit on equals sign
        if (curr === "=") {
            callback("propName", buffer);
            return { section: 'propValue', buffer: "" };
        }
        // Ignore whitespace if buffer is empty
        if (isWhitespace(curr) && buffer.length === 0) {
            return { section, buffer };
        }
        // Cancel on whitespace, newline or non alpha-numeric characters
        if (isWhitespace(curr) || isNewline(curr) || !isAlphaNum(curr)) {
            return { section: 'outside', buffer: '' };
        }
    }
    // Handle propValue
    else if (section === "propValue") {
        const backslashes = buffer.match(/\\*$/)?.[0].length || 0;
        const isEscaped = backslashes % 2 !== 0;
        const isQuote = ['"', "'"].includes(curr);
        const isInQuote = ["'", '"'].includes(buffer[0]);
        // Check if leaving quotes
        if (isQuote && buffer[0] === curr && !isEscaped) {
            callback("propValue", buffer.slice(1)); // Exclude outer quotes
            return { section: 'propName', buffer: '' };
        }
        // Check if entering quotes
        if (!isInQuote && isQuote && buffer.length === 0) {
            return { section: 'propValue', buffer: curr };
        }
        // Allow anything inside quotes
        if (isInQuote) {
            return { section, buffer: buffer + curr };
        }
        // Only numbers and null are allowed outside quotes
        const withCurr = buffer + curr;
        const maybeNumber = !isWhitespace(curr) && !isNewline(curr) && ((buffer.length === 0 && ["-", "."].includes(curr)) || !Number.isNaN(+withCurr));
        const maybeNull = "null".startsWith(withCurr);
        if (maybeNumber || maybeNull) {
            return { section, buffer: withCurr };
        }
        // If we reached whitespace of a newline and the buffer is a valid number or null, commit the buffer
        if ((isWhitespace(curr) || isNewline(curr))) {
            const isNumber = !Number.isNaN(+buffer) && buffer.length > 0;
            const isNull = buffer === "null";
            if (isNumber || isNull) {
                callback("propValue", isNumber ? +buffer : null);
                return { section: isNewline(curr) ? 'outside' : "propName", buffer: '' };
            }
        }
        // Otherwise it's invalid, so don't commit the buffer
        return { section: 'outside', buffer: '' };
    }
    // Handle newline and non alpha-numeric characters, which are not valid in command, action, or property names
    if (section !== "outside" && (isNewline(curr) || !(isAlphaNum(curr) || isWhitespace(curr) || curr === "/"))) {
        // If encountered within a word, reset without committing the buffer
        if (isAlphaNum(previousChar)) {
            return { section: 'outside', buffer: '' };
        }
        // Otherwise, reset and commit buffer
        return { section: 'outside', buffer };
    }
    // Continue buffering within the current section
    return { section, buffer: buffer + curr };
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
    let currentSection: CommandSection = 'outside';
    let currentBuffer: string = '';
    let currentIndex = 0;

    const commitCommand = () => {
        if (!currentCommand) return;
        if (currentSection === "pending") {
            currentCommand.action = currentBuffer;
        }
        console.log('pushing new command', currentIndex)
        commands.push({
            ...currentCommand,
            end: currentIndex,
            // Convert properties array to object
            properties: currentCommand.properties.reduce((obj, item, index) => {
                if (index % 2 === 0) { // Even index, property name
                    // If it's the last element and there's no corresponding value, assign null
                    obj[item + ""] = index === currentCommand!.properties.length - 1 ? null : currentCommand!.properties[index + 1];
                }
                return obj;
            }, {}),
        })
    };

    const callback = (section: CommandSection, text: string | number | null) => {
        if (section === 'command') {
            commitCommand();
            currentCommand = {
                command: text + "",
                action: null,
                properties: [],
                start: currentIndex - (text + "").length,
            };
        } else if (section === 'action' && currentCommand) {
            currentCommand.action = text + "";
        } else if (['propName', 'propValue'].includes(section)) {
            currentCommand?.properties.push(text);
        }
    };

    for (let i = 0; i < inputString.length; i++) {
        const { section, buffer } = handleCommandTransition({
            curr: inputString[i],
            section: currentSection,
            buffer: currentBuffer,
            callback,
        });
        console.log('handled command transition', inputString[i], section, buffer)
        currentSection = section;
        currentBuffer = buffer;
        currentIndex = i;
    }

    // Finalize the last command if it exists
    if (!currentCommand && currentSection === 'command') {
        currentCommand = {
            command: currentBuffer,
            action: null,
            properties: [],
            start: currentIndex - currentBuffer.length,
        };
    }
    commitCommand();

    console.log('extracted commands', commands)
    return commands;
};
