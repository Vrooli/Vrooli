import { handleRegex, urlRegex, urlRegexDev, walletAddressRegex } from "@local/shared";

/**
 * Returns the first non-blank, non-whitespace string
 * @param strings Strings to check
 * @returns First non-blank, non-whitespace string, or empty string if none found
 */
export function firstString(...strings: unknown[]): string {
    for (const obj of strings) {
        let str: string | undefined;

        if (typeof obj === "string") {
            str = obj;
        } else if (typeof obj === "function") {
            const result = obj();
            if (typeof result === "string") {
                str = result;
            }
        }

        if (str && str.trim() !== "") return str;
    }

    return "";
}

/**
 * Displays a date in a human readable format
 * @param timestamp Timestamp of date to display as seconds since epoch, or an ISO string
 * @param showDateAndTime Whether to display the time and date, or just the date
 */
export function displayDate(timestamp: number | string, showDateAndTime = true): string | null {
    let date: Date;

    if (typeof timestamp === "number") {
        if (isNaN(timestamp) || timestamp < 0) {
            return null;
        }
        date = new Date(timestamp);
    } else if (typeof timestamp === "string") {
        date = new Date(timestamp);
        if (isNaN(date.getTime())) {
            return null;
        }
    } else {
        return null;
    }

    const currentDate = new Date();

    const isCurrentYear = date.getUTCFullYear() === currentDate.getUTCFullYear();
    const isCurrentMonth = date.getUTCMonth() === currentDate.getUTCMonth();
    const isCurrentDay = date.getUTCDate() === currentDate.getUTCDate();
    const isToday = isCurrentYear && isCurrentMonth && isCurrentDay;

    const year = !isCurrentYear ? "numeric" : undefined;
    const month = !isToday ? "short" : undefined;
    const day = !isToday ? "numeric" : undefined;

    const dateString = !isToday ? date.toLocaleDateString(navigator.language, { year, month, day }) : "Today at";
    const timeString = date.toLocaleTimeString(navigator.language, { timeZone: "UTC" });
    // If joined today, display time instead of date
    if (isToday) {
        return timeString;
    }
    // Return date and/or time string
    return showDateAndTime ? `${dateString}${!isToday ? "," : ""} ${timeString}` : dateString;
}

/**
 * Converts font sizes (rem, em, px) to pixels
 * @param size - font size as string or number
 * @param id - id of element, used to get em size
 * @returns font size number in pixels
 */
export function fontSizeToPixels(size: string | number, id?: string): number {
    if (typeof size === "number") {
        return size;
    }
    if (size.includes("px")) {
        return parseFloat(size);
    }
    if (size.includes("rem")) {
        return parseFloat(size) * 16;
    }
    if (size.includes("em")) {
        if (!id) {
            console.error("Must provide id to convert em to px");
            return 0;
        }
        const el = document.getElementById(id);
        if (el) {
            const fontSize = window.getComputedStyle(el).fontSize;
            return parseFloat(size) * parseFloat(fontSize);
        }
    }
    return 0;
}

/**
 * Replaces selected text with new text.
 * @param text Text to replace in.
 * @param newText Text to replace with.
 * @param start Index of cursor or selection start.
 * @param end Index of cursor or selection end.
 * @returns New text with selected text replaced.
 */
export function replaceText(text: string, newText: string, start: number, end: number): string {
    // Ignores out of bounds indexes
    if (start < 0 || end < 0 || start > text.length || end > text.length) {
        return text;
    }
    return text.substring(0, start) + newText + text.substring(end);
}

/**
 * Uses element ID to get start, end, and element.
 * @param id The ID of the element to get the selection of
 * @returns Object containing start, end, and element
 */
export function getTextSelection(id: string) {
    const element = document.getElementById(id);
    if (!element || element.tagName !== "TEXTAREA") {
        console.error(`Element not found or is not a textarea: ${id}`);
        return { start: 0, end: 0, selected: "", inputElement: null };
    }
    const textArea = element as HTMLTextAreaElement;
    return {
        start: textArea.selectionStart,
        end: textArea.selectionEnd,
        selected: textArea.value.substring(textArea.selectionStart, textArea.selectionEnd),
        inputElement: textArea,
    };
}

/**
 * Determines start index of the current line.
 * @param text Text to search.
 * @param start Index of cursor or selection start.
 * @returns Index of start of current line.
 */
export function getLineStart(text: string, start: number) {
    if (start < 0 || start > text.length) return 0;
    return text.substring(0, start).lastIndexOf("\n") + 1;
}

/**
 * Determines end index of the current line.
 * @param text Text to search.
 * @param start Index of cursor or selection start.
 * @returns Index of end of current line.
 */
export function getLineEnd(text: string, start: number) {
    if (start < 0 || start > text.length) return text.length;

    // Find the index of the next newline character after the start index
    const endIndex = text.substring(start).indexOf("\n");

    // If no newline character is found, return the length of the text (end of the last line)
    if (endIndex === -1) {
        return text.length;
    }

    // Otherwise, adjust the end index to be relative to the entire text
    return endIndex + start;
}

/**
 * Finds the line the specified index is on.
 * @returns The line's text, as well as its start and end index
 */
export function getLineAtIndex(text: string | null | undefined, index: number): [string, number, number] {
    if (!text || index < 0 || index > text.length) return ["", 0, 0];
    const start = getLineStart(text, index);
    const end = getLineEnd(text, index);
    const line = text.substring(start, end);
    return [line, start, end];
}

/**
 * Determines all lines the cursor or highlighted text is on
 * @param text The entire text
 * @param start The index of the cursor, or start of highlighted text
 * @param end The index of the end of highlighted text
 * @returns The lines the cursor is on (or null), as well as their start and end index
 */
export function getLinesAtRange(text: string, start: number, end: number): [string[], number, number] {
    // If out of bounds, return empty array
    if (typeof text !== "string" || text.length === 0 || start < 0 || end < 0 || start > text.length || end > text.length) {
        return [[], 0, 0];
    }
    const lineStart = getLineStart(text, start);
    const lineEnd = getLineEnd(text, end);
    const lines = lineStart === lineEnd ? [] : text.substring(lineStart, lineEnd).split("\n");
    return [lines, lineStart, lineEnd];
}

export enum Headers {
    h1 = "h1",
    h2 = "h2",
    h3 = "h3",
    h4 = "h4",
    h5 = "h5",
    h6 = "h6",
}

export const headerMarkdowns = {
    [Headers.h1]: "# ",
    [Headers.h2]: "## ",
    [Headers.h3]: "### ",
    [Headers.h4]: "#### ",
    [Headers.h5]: "##### ",
    [Headers.h6]: "###### ",
};

export type TextStyleResult = {
    text: string;
    start: number;
    end: number;
};

/**
 * Inserts or removes a markdown header from the specified position in the text.
 * 
 * @param header - The header to insert or remove.
 * @param text - The full text from the input field.
 * @param start - The start index of the cursor/selection.
 * @param end - The end index of the cursor/selection.
 * @returns The updated text, start, and end index of the cursor.
 */
export function insertHeader(header: Headers, text: string, start: number, end: number): TextStyleResult {
    let updatedText = text;
    const startLine = getLineStart(updatedText, start);
    const headerText = headerMarkdowns[header];
    let headerAdded = false;

    // Define all possible markdown headers (assuming headerMarkdowns is a predefined list of headers)
    const allHeaders = Object.values(headerMarkdowns);

    // Find if there's any existing header at the startLine
    const existingHeader = allHeaders.find(h => updatedText.startsWith(h, startLine));

    if (existingHeader) {
        // If an existing header is found and it's the same as the one to insert, remove it
        if (existingHeader === headerText) {
            updatedText = replaceText(updatedText, "", startLine, startLine + existingHeader.length);
            headerAdded = false; // Mark as removed
        } else {
            // If a different header is found, replace it
            updatedText = replaceText(updatedText, headerText, startLine, startLine + existingHeader.length);
            headerAdded = true; // Mark as replaced
        }
    } else {
        // If no header is found, insert the new header
        updatedText = replaceText(updatedText, headerText, startLine, startLine);
        headerAdded = true; // Mark as added
    }

    // Adjust cursor position based on operation performed
    let cursorAdjustment = headerText.length;
    if (!headerAdded && existingHeader) {
        cursorAdjustment = -existingHeader.length; // Adjust for header removal
    } else if (headerAdded && existingHeader) {
        cursorAdjustment = headerText.length - existingHeader.length; // Adjust for header replacement
    }

    const updatedStart = start + cursorAdjustment;
    const updatedEnd = end + cursorAdjustment;

    return { text: updatedText, start: updatedStart, end: updatedEnd };
}

/**
 * Toggles padding around the selection, with special handling for Markdown italics and bold.
 * Adds padding if not present, removes it if present, and considers bold formatting.
 * @param padStart The substring to add before the selection
 * @param padEnd The substring to add after the selection
 * @param text The full text from the input field
 * @param start The start index of the cursor/selection
 * @param end The end index of the cursor/selection
 * @returns The updated text, start, and end index of the cursor
 */
export function padSelection(
    padStart: string,
    padEnd: string,
    text: string,
    start: number,
    end: number,
): TextStyleResult {
    let updatedText = text;
    let updatedStart = start;
    let updatedEnd = end;

    // Special handling for "*" (Markdown italics and bold)
    if (padStart === "*" && padEnd === "*") {
        const beforeText = text.substring(0, start);
        const afterText = text.substring(end);
        const regexPattern = /\*+/g;

        const beforeMatches = beforeText.match(regexPattern);
        const afterMatches = afterText.match(regexPattern);

        const lastBeforeMatch = beforeMatches ? beforeMatches[beforeMatches.length - 1] : "";
        const firstAfterMatch = afterMatches ? afterMatches[0] : "";

        // Check if we're exactly surrounded by "**" which indicates bold
        if (lastBeforeMatch === "**" && firstAfterMatch === "**") {
            // Add italics inside bold
            updatedText = beforeText + padStart + text.substring(start, end) + padEnd + afterText;
            updatedStart = start + padStart.length;
            updatedEnd = end + padEnd.length;
        }
        else if (lastBeforeMatch.endsWith("*") && firstAfterMatch.startsWith("*")) {
            // Remove existing asterisks (just the italics, not any of the extra asterisks)
            updatedText = beforeText.substring(0, beforeText.length - 1) + text.substring(start, end) + afterText.substring(1);
            updatedStart = start - 1;
            updatedEnd = end - 1;
        } else {
            // Add italics if no surrounding asterisks
            updatedText = beforeText + padStart + text.substring(start, end) + padEnd + afterText;
            updatedStart = start + padStart.length;
            updatedEnd = end + padEnd.length;
        }
    } else {
        // Normal handling for other paddings
        const isPaddedStart = text.substring(start - padStart.length, start) === padStart;
        const isPaddedEnd = text.substring(end, end + padEnd.length) === padEnd;

        if (isPaddedStart && isPaddedEnd) {
            // Remove padding
            updatedText = text.substring(0, start - padStart.length) + text.substring(start, end) + text.substring(end + padEnd.length);
            updatedStart = start - padStart.length;
            updatedEnd = end - padStart.length;
        } else {
            // Add padding
            updatedText = text.substring(0, start) + padStart + text.substring(start, end) + padEnd + text.substring(end);
            updatedStart = start + padStart.length;
            updatedEnd = end + padStart.length;
        }
    }

    return { text: updatedText, start: updatedStart, end: updatedEnd };
}

/**
 * Inserts a markdown link at the cursor or around the selected text.
 * @param string The full text from the input field
 * @param start The start index of the cursor/selection
 * @param end The end index of the cursor/selection
 * @returns The updated text, start, and end index of the cursor
 */
export function insertLink(text: string, start: number, end: number): TextStyleResult {
    const isValidUrl = (url: string): boolean => {
        return urlRegex.test(url)
            || urlRegexDev.test(url)
            || walletAddressRegex.test(url)
            || (url.startsWith("@") && handleRegex.test(url.substring(1))); // Remove leading"@"
    };

    // No selection: insert a placeholder link
    if (start === end) {
        const placeholder = "[label](url)";
        const updatedText = text.substring(0, start) + placeholder + text.substring(end);
        // Place cursor inside the placeholder URL for immediate editing
        const updatedStart = start + placeholder.length - 4;
        const updatedEnd = start + placeholder.length - 1;
        return { text: updatedText, start: updatedStart, end: updatedEnd };
    }

    // Handle selection
    const selection = text.substring(start, end).trim();
    let updatedText, updatedStart, updatedEnd;

    if (isValidUrl(selection)) {
        // Selection is a valid URL: use it as the URL in the markdown link
        const markdown = `[label](${selection})`;
        updatedText = text.substring(0, start) + markdown + text.substring(end);
        // Adjust cursor to allow immediate editing of the display text
        updatedStart = start + 1; // Just after '['
        updatedEnd = start + 6; // Just before ']'
    } else {
        // Selection is not a URL: treat it as display text
        const markdown = `[${selection}](url)`;
        updatedText = text.substring(0, start) + markdown + text.substring(end);
        // Adjust cursor to allow immediate editing of the placeholder URL
        updatedStart = end + 3; // Account for "](".length
        updatedEnd = updatedStart + "url".length;
    }

    return { text: updatedText, start: updatedStart, end: updatedEnd };
}

/**
 * Toggles a bullet list item at the beginning of each line in the selection.
 * @param text The full text from the input field
 * @param start The start index of the cursor/selection
 * @param end The end index of the cursor/selection
 * @returns The updated text, start, and end index of the cursor
 */
export function insertBulletList(text: string, start: number, end: number): TextStyleResult {
    // Helper function to detect if a line is bulleted
    const isBulleted = (line: string) => line.startsWith("* ");

    // Get all lines in the selection
    // eslint-disable-next-line prefer-const
    let [lines, linesStart, linesEnd] = getLinesAtRange(text, start, end);
    // Make sure we're always working with at least one line
    if (lines.length === 0) {
        lines = [""];
    }

    // Check if all lines are already bulleted
    const allBulleted = lines.every(isBulleted);
    // If so, remove the bullets
    if (allBulleted) {
        const updatedLines = lines.map(line => line.substring(2));
        const updatedText = replaceText(text, updatedLines.join("\n"), linesStart, linesEnd);
        // Keep the selection in the same relative position. This means the start should move 
        // at most 2 characters to the left, making sure not to move back past the start of the line
        const updatedStart = Math.max(start - 2, linesStart);
        const updatedEnd = end - (2 * lines.length);
        return { text: updatedText, start: updatedStart, end: updatedEnd };
    }

    // Otherwise, insert a bullet at the start of each line
    const updatedLines = lines.map(line => `* ${line}`);
    const updatedText = replaceText(text, updatedLines.join("\n"), linesStart, linesEnd);
    // Keep the selection in the same relative position
    const updatedStart = start + 2;
    const updatedEnd = end + (2 * lines.length);
    return { text: updatedText, start: updatedStart, end: updatedEnd };
}

/**
 * Toggles a number list item at the beginning of each line in the selection.
 * @param text The full text from the input field
 * @param start The start index of the cursor/selection
 * @param end The end index of the cursor/selection
 * @returns The updated text, start, and end index of the cursor
 */
export function insertNumberList(text: string, start: number, end: number): TextStyleResult {
    // Get all lines in the selection
    // eslint-disable-next-line prefer-const
    let [lines, linesStart, linesEnd] = getLinesAtRange(text, start, end);
    // Make sure we're always working with at least one line
    if (lines.length === 0) {
        lines = [""];
    }

    // Detect existing numbering and determine if all selected lines are already numbered
    const numberPrefixRegex = /^\d+\.\s+/;
    const allNumbered = lines.every(line => numberPrefixRegex.test(line));

    let updatedLines: string[], updatedStart: number, updatedEnd: number;

    if (allNumbered) {
        // Remove numbering from all lines
        updatedLines = lines.map(line => line.replace(numberPrefixRegex, ""));
        const totalRemovedLength = lines.reduce((acc, line) => acc + (line.match(numberPrefixRegex)?.[0].length ?? 0), 0);
        updatedStart = Math.max(linesStart, start - 3);
        updatedEnd = Math.max(linesStart, end - totalRemovedLength);
    } else {
        // Add numbering to all lines
        updatedLines = lines.map((line, index) => `${index + 1}. ${line}`);
        const numberLengthDifference = updatedLines.reduce((acc, line) => acc + line.split(".")[0].length + 2, 0);// - lines.join("\n").length;
        updatedStart = start + 3;
        updatedEnd = end + numberLengthDifference;
    }

    const updatedText = replaceText(text, updatedLines.join("\n"), linesStart, linesEnd);

    return { text: updatedText, start: updatedStart, end: updatedEnd };
}

/**
 * Toggles a checkbox list item at the beginning of each line in the selection.
 * @param text The full text from the input field
 * @param start The start index of the cursor/selection
 * @param end The end index of the cursor/selection
 * @returns The updated text, start, and end index of the cursor
 */
export function insertCheckboxList(text: string, start: number, end: number): TextStyleResult {
    // Get all lines in the selection
    // eslint-disable-next-line prefer-const
    let [lines, linesStart, linesEnd] = getLinesAtRange(text, start, end);
    // Make sure we're always working with at least one line
    if (lines.length === 0) {
        lines = [""];
    }

    const checkboxRegex = /^- \[[x ]\] /;

    // Determine if all selected lines have a checkbox
    const allHaveCheckbox = lines.every(line => checkboxRegex.test(line));

    let updatedLines: string[], updatedStart: number, updatedEnd: number;

    if (allHaveCheckbox) {
        // Remove checkboxes from all lines
        updatedLines = lines.map(line => line.replace(checkboxRegex, ""));
        const totalRemovedLength = lines.reduce((acc, line) => acc + (line.match(checkboxRegex)?.[0].length ?? 0), 0);
        updatedStart = Math.max(linesStart, start - totalRemovedLength / lines.length); // Average removal length per line
        updatedEnd = Math.max(linesStart, end - totalRemovedLength);
    } else {
        // Add unchecked checkboxes to all lines
        updatedLines = lines.map(line => `- [ ] ${line}`);
        const checkboxLength = "- [ ] ".length;
        updatedStart = start + checkboxLength;
        updatedEnd = end + (checkboxLength * lines.length);
    }

    const updatedText = replaceText(text, updatedLines.join("\n"), linesStart, linesEnd);

    return { text: updatedText, start: updatedStart, end: updatedEnd };
}

/**
 * Toggles code quotes or a code block, depending on the selection.
 * @param text The full text from the input field
 * @param start The start index of the cursor/selection
 * @param end The end index of the cursor/selection
 * @returns The updated text, start, and end index of the cursor
 */
export function insertCode(text: string, start: number, end: number): TextStyleResult {
    const selection = text.substring(start, end);
    const isMultiLine = selection.includes("\n");

    const inlineCodeMarker = "`";
    const codeBlockMarker = "```";

    if (isMultiLine) {
        // For multi-line, check if it's already a code block considering "\n```" before and after
        const preText = text.substring(start - codeBlockMarker.length - 1, start);
        const postText = text.substring(end, end + codeBlockMarker.length + 1);
        if (preText === `${codeBlockMarker}\n` && postText === `\n${codeBlockMarker}`) {
            // Remove code block markers including new lines
            return {
                text: text.substring(0, start - codeBlockMarker.length - 1) + selection + text.substring(end + codeBlockMarker.length + 1),
                start: start - codeBlockMarker.length - 1,
                end: end - codeBlockMarker.length - 1,
            };
        }

        // Check if the start and end of the selection are inline code markers
        if (selection.startsWith(codeBlockMarker + "\n") && selection.endsWith("\n" + codeBlockMarker)) {
            // Remove inline code markers
            return {
                text: text.substring(0, start) + selection.substring(4, selection.length - 4) + text.substring(end),
                start,
                end: end - 8,
            };
        }

        // Otherwise, Add code block markers with new lines around the selection
        return {
            text: text.substring(0, start) + `${codeBlockMarker}\n` + selection + `\n${codeBlockMarker}` + text.substring(end),
            start: start + codeBlockMarker.length + 1,
            end: end + codeBlockMarker.length + 1,
        };
    } else {
        // Handle single-line selections for inline code
        // Check if the selection is already surrounded by inline code markers
        const paddedWithInlineCode = text.substring(start - inlineCodeMarker.length, start) === inlineCodeMarker
            && text.substring(end, end + inlineCodeMarker.length) === inlineCodeMarker;
        if (paddedWithInlineCode) {
            // Remove inline code markers
            return {
                text: text.substring(0, start - inlineCodeMarker.length) + selection + text.substring(end + inlineCodeMarker.length),
                start: start - inlineCodeMarker.length,
                end: end - inlineCodeMarker.length,
            };
        }

        // Check if the start and end of the selection are inline code markers
        if (text[start] === inlineCodeMarker && text[end - 1] === inlineCodeMarker) {
            // Remove inline code markers
            return {
                text: text.substring(0, start) + selection.substring(1, selection.length - 1) + text.substring(end),
                start,
                end: end - 2,
            };
        }

        // Otherwise, add inline code markers around the selection
        return padSelection(inlineCodeMarker, inlineCodeMarker, text, start, end);
    }
}

/**
 * Inserts quotes or a quote block, depending on the selection.
 * @param text The full text from the input field
 * @param start The start index of the cursor/selection
 * @param end The end index of the cursor/selection
 * @returns The updated text, start, and end index of the cursor
 */
export function insertQuote(text: string, start: number, end: number): TextStyleResult {
    // Get all lines in the selection
    // eslint-disable-next-line prefer-const
    let [lines, linesStart, linesEnd] = getLinesAtRange(text, start, end);
    // Make sure we're always working with at least one line
    if (lines.length === 0) {
        lines = [""];
    }

    // Check if all lines are already quoted
    const quotePattern = /^(>+)(\s)?/; // Matches sequences of ">" at the start of a line, optionally followed by a space
    const allLinesQuoted = lines.every(line => quotePattern.test(line));

    const updatedLines: string[] = [];

    lines.forEach((line, index) => {
        const match = line.match(quotePattern);

        // If all lines are quoted, remove the quoting
        if (allLinesQuoted) {
            if (!match) {
                console.error("No match found for quoted line", line);
                return;
            }
            // Remove one level of quoting
            const isNested = line.startsWith(">>"); // Has at least two levels of quoting
            const quotes = match[0];
            const updatedLine = isNested ? line.substring(1) : line.substring(quotes.length);
            updatedLines.push(updatedLine);
            // If the selection start is on this line, adjust it to account for the removed quote
            if (index === 0) {
                start -= isNested ? 1 : quotes.length;
            }
            // Continually adjust the end index
            end -= isNested ? 1 : quotes.length;
        }
        // Otherwise, add quoting
        else {
            const updatedLine = match ? ">" + line : "> " + line;
            updatedLines.push(updatedLine);
            // If the selection start is on this line, adjust it to account for the added quote
            if (index === 0) {
                start += match ? 1 : 2;
            }
            // Continually adjust the end index
            end += match ? 1 : 2;
        }
    });

    const updatedText = replaceText(text, updatedLines.join("\n"), linesStart, linesEnd);

    // Ensure the start index does not go to the previous line
    start = Math.max(linesStart, start);

    return { text: updatedText, start, end };
}

/**
 * Inserts table markdown with the required number of rows and columns.
 * @param rows The number of rows in the table
 * @param cols The number of columns in the table
 * @param text The full text from the input field
 * @param start The start index of the cursor/selection
 * @param end The end index of the cursor/selection
 * @returns The updated text, start, and end index of the cursor
 */
export function insertTable(rows: number, cols: number, text: string, start: number, end: number): TextStyleResult {
    // Don't do anything if rows or cols are invalid
    if (rows <= 0 || cols <= 0) {
        return { text, start, end };
    }
    // Generate table markdown based on rows and cols
    let tableStr = "|";
    for (let c = 0; c < cols; c++) {
        tableStr += " Header |";
    }
    tableStr += "\n|";
    for (let c = 0; c < cols; c++) {
        tableStr += " ------- |";
    }
    for (let r = 0; r < rows; r++) {
        tableStr += "\n|";
        for (let c = 0; c < cols; c++) {
            tableStr += "   |";
        }
    }
    // Get starting line
    const [firstLine, _, firstLineEnd] = getLineAtIndex(text, start);
    let tableStart = start;
    let tableEnd = end;
    // If the first line has non-whitespace characters, insert the table on the next line
    if (firstLine.trim() !== "") {
        console.log("inserting at next line");
        // Handle case where there is no next line
        if (firstLineEnd === text.length) {
            tableStr = "\n" + tableStr;
            tableStart = firstLineEnd;
        } else {
            tableStart = firstLineEnd + 1;
        }
        tableEnd = tableStart;
        tableStr += "\n";
    }
    // Move the selection to the first cell
    const updatedStart = (firstLine.trim() !== "" && firstLineEnd === text.length) ? tableStart + 3 : tableStart + 2;
    const updatedEnd = updatedStart + "Header".length;
    // Insert table markdown at the cursor
    const updatedText = replaceText(text, tableStr, tableStart, tableEnd);
    return { text: updatedText, start: updatedStart, end: updatedEnd };
}

const MAX_CONTENT_LENGTH = 1500;

/**
 * Generates a context string from a selection of text, or most/all of the text if no selection.
 * @param selected The current selection
 * @param fullText The full text from the input field
 * @returns The formatted context string
 */
export function generateContext(selected: string, fullText: string): string {
    let context = selected.trim();
    const ellipsis = "…";
    // If selected text is too long, provide the last 1500 characters
    if (context.length > MAX_CONTENT_LENGTH) {
        context = ellipsis + context.substring(context.length - MAX_CONTENT_LENGTH - ellipsis.length, context.length);
        return context.trim();
    } else if (context.length > 0) {
        return context.trim();
    }
    const fullValue = fullText.trim();
    if (fullValue.length <= 0) {
        return "";
    }
    // If there's no selection, provide the full text if it's not too long
    if (fullValue.length <= MAX_CONTENT_LENGTH) context = fullValue;
    // Otherwise, provide the last 1500 characters
    else context = ellipsis + fullValue.substring(fullValue.length - MAX_CONTENT_LENGTH - ellipsis.length, fullValue.length);
    return context.trim();
}
