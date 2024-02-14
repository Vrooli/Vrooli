export type LlmCommand = {
    command: string;
    action: string | null;
    properties: {
        [key: string]: string | number | null;
    } | null;
}

type CommandSection = "outside" | "pending" | "command" | "action" | "propName" | "propValue";
type PropSection = "propName" | "propValue";
type PropertyValue = string | number | null;
type PropertyValueWithIndex = { value: PropertyValue, endIndex: number };

/** Maximum length for a command, action, or property name */
const MAX_COMMAND_ACTION_PROP_LENGTH = 32;

// export const extractCommands = (inputString: string): LlmCommand[] => {
//     const commands: LlmCommand[] = [];
//     if (typeof inputString !== 'string') return commands;
//     const lines = inputString.split('\n');

//     lines.forEach(line => {
//         const commandRegex = /\/(?<command>\w+)(?:\s+(?<action>[A-Za-z]+))?(?=\s|$)(?<properties>[^\/]*)/g;
//         // const commandRegex = /\/(?<command>\w+)(?:\s+(?<action>[A-Za-z]+))?(\s+(?<properties>(?:\w+=(?:'[^']*'|"[^"]*"|[^\/\s'"]+)(?:\s+|$))?)+)?/g;

//         let match;
//         while ((match = commandRegex.exec(line)) !== null) {
//             const { command, action, properties } = match.groups as { command: string; action?: string; properties?: string };
//             const propertiesObj: Record<string, string | number | null> = {};

//             if (properties?.trim()) {
//                 console.log('the properties', properties.trim());
//                 const propertiesParts = properties.trim().match(/\w+=(?:'[^']*'|"[^"]*"|-?\d+(\.\d+)?|\w+)/g) || [];
//                 // const propertiesParts = properties.trim().match(/\w+=(?:'([^'\\]*(?:\\.[^'\\]*)*)'|"[^"\\]*(?:\\.[^"\\]*)*"|\S+)/g) || [];
//                 propertiesParts.forEach(prop => {
//                     const [key, value] = prop.split('=');
//                     let formattedValue: string | number | null;

//                     if (value.startsWith("'") || value.startsWith("\"")) {
//                         formattedValue = value.replace(/^['"]|['"]$/g, ''); // Treat as string, remove quotes
//                     } else if (!isNaN(Number(value))) {
//                         formattedValue = Number(value); // Treat as number (including negatives and decimals)
//                     } else if (value === 'null') {
//                         formattedValue = null; // Convert 'null' string to null value
//                     } else {
//                         formattedValue = value; // Keep as string
//                     }

//                     propertiesObj[key] = formattedValue;
//                 });
//             }

//             commands.push({
//                 command,
//                 action: action || null, // Normalize undefined to null
//                 properties: Object.keys(propertiesObj).length > 0 ? propertiesObj : null
//             });
//         }
//     });

//     return commands;
// };


// export const extractCommands = (inputString: string): LlmCommand[] => {
//     const commands: LlmCommand[] = [];
//     if (typeof inputString !== 'string') return commands;

//     let command = '';
//     let action = '';
//     let propName = '';
//     let propValue = '';
//     let isReadingCommand = false;
//     let isReadingAction = false;
//     let isReadingPropName = false;
//     let isReadingPropValue = false;
//     let isInsideQuotes = false;
//     let quoteChar = '';
//     let currentCharLimit = 0;
//     const properties = {};

//     const commitProperty = () => {
//         if (propName && propName.length <= 32) {
//             properties[propName] = propValue.match(/^\d+$/) ? parseInt(propValue, 10)
//                 : propValue.match(/^\-?\d+(\.\d+)?$/) ? parseFloat(propValue)
//                     : propValue === 'null' ? null
//                         : propValue;
//             propName = '';
//             propValue = '';
//             isReadingPropName = false;
//             isReadingPropValue = false;
//         }
//     };

//     const commitCommand = () => {
//         if (command && command.length <= 32) {
//             commands.push({
//                 command,
//                 action: action.length <= 32 ? action || null : null,
//                 properties: Object.keys(properties).length > 0 ? { ...properties } : null
//             });
//             command = '';
//             action = '';
//             for (let prop in properties) delete properties[prop];
//         }
//     };

//     for (let i = 0; i < inputString.length; i++) {
//         let char = inputString[i];
//         let nextChar = inputString[i + 1];

//         // Handle commands
//         if (char === '/' && (i === 0 || inputString[i - 1] === ' ' || inputString[i - 1] === '\n')) {
//             commitProperty();
//             commitCommand();
//             isReadingCommand = true;
//             currentCharLimit = 0;
//             continue;
//         }

//         if (isReadingCommand) {
//             if (char === ' ' || char === '\n' || nextChar === undefined) {
//                 isReadingCommand = false;
//                 isReadingAction = true;
//             } else if (currentCharLimit < 32) {
//                 command += char;
//                 currentCharLimit++;
//             }
//             continue;
//         }

//         // Handle actions
//         if (isReadingAction) {
//             if (char === ' ' || char === '\n' || nextChar === undefined || nextChar === '/') {
//                 isReadingAction = false;
//                 isReadingPropName = true;
//             } else if (char !== ' ' && currentCharLimit < 32) {
//                 action += char;
//                 currentCharLimit++;
//             }
//             continue;
//         }

//         // Handle property names
//         if (isReadingPropName) {
//             if (char === '=') {
//                 isReadingPropName = false;
//                 isReadingPropValue = true;
//                 currentCharLimit = 0;
//                 continue;
//             } else if (char !== ' ' && currentCharLimit < 32) {
//                 propName += char;
//                 currentCharLimit++;
//             }
//             continue;
//         }

//         // Handle property values
//         if (isReadingPropValue) {
//             if (char === '"' || char === "'") {
//                 if (!isInsideQuotes) {
//                     isInsideQuotes = true;
//                     quoteChar = char;
//                 } else if (char === quoteChar) {
//                     isInsideQuotes = false;
//                     commitProperty();
//                 }
//             } else if (isInsideQuotes || (char !== ' ' && char !== '\n')) {
//                 propValue += char;
//             } else if (!isInsideQuotes && (char === ' ' || char === '\n' || nextChar === undefined || nextChar === '/')) {
//                 commitProperty();
//                 if (char === '\n') {
//                     commitCommand();
//                 }
//             }
//         }
//     }

//     // Commit any remaining properties or commands after loop completion
//     commitProperty();
//     commitCommand();

//     return commands;
// };

export const isNewline = (char: string) => char === '\n' || char === '\r';

export const isWhitespace = (char: string) => /^\s$/.test(char) && !isNewline(char);

export const isAlphaNum = (char: string) => /^[A-Za-z0-9]$/.test(char);


// export const handleCommandTransition = (
//     currentChar: string,
//     currentSection: CommandSection,
//     buffer: string,
//     isInQuotes: boolean,
// ): { section: CommandSection, buffer: string, isInQuotes: boolean } => {
//     const previousChar: string | undefined = buffer[buffer.length - 1];
//     const firstChar: string | undefined = buffer[0];
//     // Handle leaving quotes
//     if (isInQuotes) {
//         // If we encounter a non-escaped quote of the same type as the opening quote, we've
//         // reached the end of the quoted string
//         if (currentChar === firstChar && previousChar !== '\\') {
//             return { section: "propName", buffer: buffer + currentChar, isInQuotes: false };
//         }
//         // Otherwise, keep buffering
//         return { section: currentSection, buffer: buffer + currentChar, isInQuotes };
//     }
//     // Handle entering quotes
//     if ([`'`, '"'].includes(currentChar)) {
//         // Quotes are only allowed for property values
//         if (currentSection === 'propValue') {
//             // Make it the buffer start
//             return { section: 'propValue', buffer: currentChar, isInQuotes: true };
//         }
//         // If encountered within a word, reset without committing the buffer
//         if (isAlphaNum(previousChar)) {
//             return { section: 'outside', buffer: '', isInQuotes };
//         }
//         // Otherwise, reset and commit buffer
//         return { section: 'outside', buffer, isInQuotes };
//     }
//     // Handle entering property value
//     if (currentChar === "=") {
//         // If we're on a property name, an equals indicates the start of a property value
//         if (currentSection === 'propName') {
//             // Commit the buffer without the equals sign
//             return { section: 'propValue', buffer, isInQuotes };
//         }
//         // If encountered within a word, reset without committing the buffer
//         if (isAlphaNum(previousChar)) {
//             return { section: 'outside', buffer: '', isInQuotes };
//         }
//         // Otherwise, reset and commit buffer
//         return { section: 'outside', buffer, isInQuotes };
//     }
//     // Handle newline and non alpha-numeric characters, which are not valid in command,
//     // action, or property names
//     if (isNewline(currentChar) || !(isAlphaNum(currentChar) || isWhitespace(currentChar) || currentChar === "/")) {
//         // If encountered within a word, reset without committing the buffer
//         if (isAlphaNum(previousChar)) {
//             return { section: 'outside', buffer: '', isInQuotes };
//         }
//         // Otherwise, reset and commit buffer
//         return { section: 'outside', buffer, isInQuotes };
//     }
//     // Skip over whitespace
//     if (isWhitespace(currentChar)) {
//         return { section: currentSection, buffer, isInQuotes };
//     }
//     // Handle slash, which may indicate the start of a new command
//     if (currentChar === "/") {
//         // If encountered withing a word, reset without committing the buffer
//         if (isAlphaNum(previousChar)) {
//             return { section: 'outside', buffer: '', isInQuotes };
//         }
//         // Otherwise, start the command. Don't include the slash in the buffer
//         return { section: 'command', buffer: "", isInQuotes };
//     }
//     // Handle valid command, action and property name characters
//     if (isAlphaNum(currentChar)) {
//         // If we're inside a word, keep buffering
//         if (isAlphaNum(previousChar)) {
//             return { section: currentSection, buffer: buffer + currentChar, isInQuotes };
//         }
//         // If we're in the "outside" section, ignore
//         if (currentSection === 'outside') {
//             return { section: currentSection, buffer: '', isInQuotes };
//         }
//         // If we're in a command, this may be an action or a propName.
//         //TODO
//     }
//     // Otherwise, reset without committing the buffer
//     return { section: 'outside', buffer: '', isInQuotes };
// }

/**
 * Handles the transition between parsing a command, action, property name, or property value.
 * @returns An object containing the new parsing section, the updated buffer, and the updated isInQuotes flag.
 * 
 * NOTE: If the parsing section changed and the buffer is not empty, it should be committed to the previous section.
 */
export const handleCommandTransition = ({
    curr,
    next,
    section,
    buffer,
    isInQuotes,
}: {
    curr: string,
    next: string,
    section: CommandSection,
    buffer: string,
    isInQuotes: boolean,
}
): { section: CommandSection, buffer: string, isInQuotes: boolean } => {
    const previousChar = buffer[buffer.length - 1];

    // Handle quotes
    if (curr === '"' || curr === "'") {
        if (!isInQuotes && section === "propName") {  // Entering quotes
            return { section: 'propValue', buffer: curr, isInQuotes: true };
        } else if (buffer[0] === curr && buffer[buffer.length - 1] !== '\\') {  // Leaving quotes
            return { section: 'propName', buffer: buffer + curr, isInQuotes: false };
        }
    }
    // Handle equals sign
    else if (curr === '=' && (section === 'propName' || section === 'pending')) {
        return { section: 'propValue', buffer: buffer, isInQuotes };
    }
    // Handle slash, which may indicate the start of a new command
    else if (curr === "/") {
        console.log('in curr slash', previousChar, isWhitespace(previousChar));
        // If encountered within a word, don't start a new command
        if (previousChar && !isWhitespace(previousChar) && !isNewline(previousChar)) {
            return { section, buffer: buffer + curr, isInQuotes };
        }
        // Otherwise, start the command. Don't include the slash in the buffer
        return { section: 'command', buffer: "", isInQuotes };
    }
    // Handle transition from command to pending (action/propName undecided)
    else if (section === 'command' && isWhitespace(curr)) {
        return { section: 'pending', buffer: '', isInQuotes };
    }
    // Handle decision between action and propName based on lookahead
    else if (section === 'pending') {
        if (next === '=') {  // Next character is '=', so current buffer is a propName
            return { section: 'propName', buffer: buffer + curr, isInQuotes };
        } else if (isWhitespace(next)) {  // Next character is whitespace, so current buffer is an action
            return { section: 'command', buffer: '', isInQuotes };  // Return to 'command' to await next word or command
        }
    }
    // Handle newline and non alpha-numeric characters, which are not valid in command, action, or property names
    if (section !== "outside" && (isNewline(curr) || !(isAlphaNum(curr) || isWhitespace(curr) || curr === "/"))) {
        // If encountered within a word, reset without committing the buffer
        if (isAlphaNum(previousChar)) {
            return { section: 'outside', buffer: '', isInQuotes };
        }
        // Otherwise, reset and commit buffer
        return { section: 'outside', buffer, isInQuotes };
    }
    // Continue buffering within the current section
    return { section, buffer: buffer + curr, isInQuotes };
};


/**
 * Manages the transition from parsing a property name to parsing its value, considering newlines as a reset trigger.
 * @param currentChar The current character being processed.
 * @param buffer The current buffer holding the accumulated characters for the property name.
 * @param currentSection The current parsing section ('propName' or 'propValue').
 * @returns An object containing the property name (if transitioning to value parsing), the new parsing section, and the updated buffer.
 */
export const handlePropertyNameValueTransition = (
    currentChar: string,
    buffer: string,
    currentSection: PropSection
): { propName: string, newSection: PropSection | 'command', newBuffer: string } => {
    if (currentChar === '\n') {
        // Newline encountered, reset to parsing a new command
        return { propName: '', newSection: 'command', newBuffer: '' };
    }
    if (currentSection === 'propName' && currentChar === '=') {
        return { propName: buffer, newSection: 'propValue', newBuffer: '' };
    }
    return { propName: '', newSection: currentSection, newBuffer: buffer + currentChar };
}


/**
 * Parses a command's property value from a given input string starting at a specified index.
 * This function is designed to handle various types of values including:
 * - Integers and floating-point numbers
 * - Quoted strings, with support for escaped quotes within the string
 * - The special 'null' value
 * 
 * The function respects quoting for string values, allowing for spaces, newlines,
 * and special characters within quoted strings. It correctly handles escaped quotes
 * by including them as part of the string value.
 * 
 * @param input - The input string containing the property value to parse.
 * @param startIndex - The index within the input string to start parsing from.
 * @returns The parsed property value and the index of the last character parsed.
 */
export const parsePropertyValue = (input: string, startIndex: number): PropertyValueWithIndex => {
    let value = '';
    let index = startIndex;
    let isInsideQuotes = false;
    let quoteChar = '';
    let escapeNext = false;

    for (; index < input.length; index++) {
        const char = input[index];

        // When escape character is encountered, set escapeNext to true and skip this iteration
        if (char === '\\' && !escapeNext) {
            escapeNext = true;
            continue;
        }

        // Handle the end of a quoted string
        if (isInsideQuotes && char === quoteChar && !escapeNext) {
            isInsideQuotes = false;
            quoteChar = '';
            continue; // Skip adding the ending quote to the value
        }

        // Start of a quoted string
        if (!isInsideQuotes && (char === '"' || char === "'")) {
            isInsideQuotes = true;
            quoteChar = char;
            continue; // Skip adding the starting quote to the value
        }

        // Add character to value
        if (isInsideQuotes || char !== ' ') {
            value += char;
        }

        // Reset escapeNext flag after processing the current character
        escapeNext = false;

        // Break out of the loop if not inside quotes and encounter space or newline
        if (!isInsideQuotes && (char === ' ' || char === '\n')) {
            break;
        }
    }

    // Determine the type of the value
    let parsedValue;
    if (!isInsideQuotes && value.match(/^\d+$/)) {
        parsedValue = parseInt(value, 10);
    } else if (!isInsideQuotes && value.match(/^\-?\d+(\.\d+)?$/)) {
        parsedValue = parseFloat(value);
    } else if (!isInsideQuotes && value === 'null') {
        parsedValue = null;
    } else {
        // Remove any remaining escape characters for strings
        parsedValue = value.replace(/\\(['"])/g, '$1');
    }

    return { value: parsedValue, endIndex: index };
}

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
    if (typeof inputString !== 'string') return commands;

    let command = '';
    let action = '';
    let propName = '';
    let currentSection = 'command'; // can be 'command', 'action', 'propName', 'propValue'
    const properties = {};

    const addProperty = (propName, propValue) => {
        if (propName) {
            properties[propName] = propValue;
        }
    };

    const addCommand = () => {
        if (command) {
            commands.push({
                command,
                action: action || null,
                properties: Object.keys(properties).length > 0 ? { ...properties } : null
            });
            // Reset for the next command
            command = '';
            action = '';
            for (let prop in properties) delete properties[prop];
        }
    };

    for (let i = 0; i < inputString.length; i++) {
        const char = inputString[i];
        const prevChar = i > 0 ? inputString[i - 1] : '';

        // Check for command start, ensuring it's not preceded by a non-whitespace character
        if (char === '/' && (i === 0 || prevChar === '\n' || prevChar === ' ') && command.length === 0) {
            addCommand();
            currentSection = 'command';
        } else if (char === '\n') {
            // Newline signifies the end of the current command/action/property
            addCommand();
            currentSection = 'command';
        } else if (currentSection === 'command' && char !== ' ' && char !== '/' && command.length < MAX_COMMAND_ACTION_PROP_LENGTH) {
            command += char;
        } else if (currentSection === 'command' && (char === ' ' || char === '\n')) {
            currentSection = 'action';
        } else if (currentSection === 'action' && char !== ' ' && char !== '\n' && action.length < MAX_COMMAND_ACTION_PROP_LENGTH) {
            action += char;
        } else if (currentSection === 'action' && (char === ' ' || char === '\n')) {
            currentSection = 'propName';
        } else if (currentSection === 'propName' && char === '=') {
            currentSection = 'propValue';
        } else if (currentSection === 'propValue') {
            const { value, endIndex } = parsePropertyValue(inputString, i);
            addProperty(propName, value);
            i = endIndex - 1; // Adjust index to skip processed characters
            propName = '';
            currentSection = 'propName';
        } else if (currentSection === 'propName' && char !== ' ' && char !== '\n' && propName.length < MAX_COMMAND_ACTION_PROP_LENGTH) {
            propName += char;
        }
    }

    // Handle any remaining property or command at the end of the string
    if (currentSection === 'propValue') {
        const { value } = parsePropertyValue(inputString, inputString.length - propName.length - 1);
        addProperty(propName, value);
    }
    addCommand(); // Add the final command if any

    return commands;
};
