/**
 * Returns the first non-blank, non-whitespace string
 * @param strings Strings to check
 * @returns First non-blank, non-whitespace string, or empty string if none found
 */
export const firstString = (...strings: unknown[]): string => {
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
};

/**
 * Displays a date in a human readable format
 * @param timestamp Timestamp of date to display
 * @param showDateAndTime Whether to display the time and date, or just the date
 */
export const displayDate = (timestamp: number, showDateAndTime = true): string | null => {
    if (isNaN(timestamp) || timestamp < 0) {
        return null;
    }

    const date = new Date(timestamp);
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
};

/**
 * Converts font sizes (rem, em, px) to pixels
 * @param size - font size as string or number
 * @param id - id of element, used to get em size
 * @returns font size number in pixels
 */
export const fontSizeToPixels = (size: string | number, id?: string): number => {
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
};

/**
 * Replaces selected text with new text.
 * @param text Text to replace in.
 * @param newText Text to replace with.
 * @param start Index of cursor or selection start.
 * @param end Index of cursor or selection end.
 * @returns New text with selected text replaced.
 */
export const replaceText = (text: string, newText: string, start: number, end: number): string => {
    // Ignores out of bounds indexes
    if (start < 0 || end < 0 || start > text.length || end > text.length) {
        return text;
    }
    return text.substring(0, start) + newText + text.substring(end);
};

/**
 * Uses element ID to get start, end, and element.
 * @param id The ID of the element to get the selection of
 * @returns Object containing start, end, and element
 */
export const getTextSelection = (id: string) => {
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
};

/**
 * Determines start index of the current line.
 * @param text Text to search.
 * @param start Index of cursor or selection start.
 * @returns Index of start of current line.
 */
export const getLineStart = (text: string, start: number) => {
    if (start < 0 || start > text.length) return 0;
    return text.substring(0, start).lastIndexOf("\n") + 1;
};

/**
 * Determines end index of the current line.
 * @param text Text to search.
 * @param start Index of cursor or selection start.
 * @returns Index of end of current line.
 */
export const getLineEnd = (text: string, start: number) => {
    if (start < 0 || start > text.length) return text.length;

    // Find the index of the next newline character after the start index
    const endIndex = text.substring(start).indexOf("\n");

    // If no newline character is found, return the length of the text (end of the last line)
    if (endIndex === -1) {
        return text.length;
    }

    // Otherwise, adjust the end index to be relative to the entire text
    return endIndex + start;
};

/**
 * Finds the line the specified index is on.
 * @returns The line's text, as well as its start and end index
 */
export const getLineAtIndex = (text: string | null | undefined, index: number): [string, number, number] => {
    if (!text || index < 0 || index > text.length) return ["", 0, 0];
    const start = getLineStart(text, index);
    const end = getLineEnd(text, index);
    const line = text.substring(start, end);
    return [line, start, end];
};

/**
 * Determines all lines the cursor or highlighted text is on
 * @param text The entire text
 * @param start The index of the cursor, or start of highlighted text
 * @param end The index of the end of highlighted text
 * @returns The lines the cursor is on (or null), as well as their start and end index
 */
export const getLinesAtRange = (text: string, start: number, end: number): [string[], number, number] => {
    // If out of bounds, return empty array
    if (typeof text !== "string" || text.length === 0 || start < 0 || end < 0 || start > text.length || end > text.length) {
        return [[], 0, 0];
    }
    const lineStart = getLineStart(text, start);
    const lineEnd = getLineEnd(text, end);
    const lines = lineStart === lineEnd ? [] : text.substring(lineStart, lineEnd).split("\n");
    return [lines, lineStart, lineEnd];
};

export enum Headers {
    H1 = "h1",
    H2 = "h2",
    H3 = "h3",
    H4 = "h4",
    H5 = "h5",
    H6 = "h6",
}

export const headerMarkdowns = {
    [Headers.H1]: "# ",
    [Headers.H2]: "## ",
    [Headers.H3]: "### ",
    [Headers.H4]: "#### ",
    [Headers.H5]: "##### ",
    [Headers.H6]: "###### ",
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
export const insertHeader = (header: Headers, text: string, start: number, end: number): TextStyleResult => {
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
};

/**
 * Pads selection with the given substring
 * @param padStart The substring to add before the selection
 * @param padEnd The substring to add after the selection
 * @param text The full text from the input field
 * @param start The start index of the cursor/selection
 * @param end The end index of the cursor/selection
 * @returns The updated text, start, and end index of the cursor
 */
export const padSelection = (
    padStart: string,
    padEnd: string,
    text: string,
    start: number,
    end: number,
): TextStyleResult => {
    // Insert pad around selection
    const updatedText = text.substring(0, start) + padStart + text.substring(start, end) + padEnd + text.substring(end);
    // Keep the selection in the same relative position
    const updatedStart = start + padStart.length;
    const updatedEnd = end + padStart.length;
    return { text: updatedText, start: updatedStart, end: updatedEnd };
};

/**
 * Inserts a markdown link at the cursor or around the selected text.
 * @param string The full text from the input field
 * @param start The start index of the cursor/selection
 * @param end The end index of the cursor/selection
 * @returns The updated text, start, and end index of the cursor
 */
export const insertLink = (text: string, start: number, end: number): TextStyleResult => {
    // If no selection, insert [link](url) at the cursor
    if (start === end) {
        const placeholder = "[display text](url)";
        const updatedText = text.substring(0, start) + placeholder + text.substring(end);
        // Place cursor at the end of the "url" word
        const updatedStart = start + placeholder.length - 1;
        const updatedEnd = updatedStart;
        return { text: updatedText, start: updatedStart, end: updatedEnd };
    }
    // Otherwise, call padSelection
    return padSelection("[", "](url)", text, start, end);
};

/**
 * Inserts a bullet list item at the beginning of each line in the selection.
 * @param text The full text from the input field
 * @param start The start index of the cursor/selection
 * @param end The end index of the cursor/selection
 * @returns The updated text, start, and end index of the cursor
 */
export const insertBulletList = (text: string, start: number, end: number): TextStyleResult => {
    const [lines, linesStart, linesEnd] = getLinesAtRange(text, start, end);
    const updatedText = replaceText(text, lines.map(line => `* ${line}`).join("\n"), linesStart, linesEnd);
    //TODO not sure how to handle selection yet
    return { text: updatedText, start, end };
};

/**
 * Inserts a number list item at the beginning of each line in the selection.
 * @param text The full text from the input field
 * @param start The start index of the cursor/selection
 * @param end The end index of the cursor/selection
 * @returns The updated text, start, and end index of the cursor
 */
export const insertNumberList = (text: string, start: number, end: number): TextStyleResult => {
    const [lines, linesStart, linesEnd] = getLinesAtRange(text, start, end);
    const updatedText = replaceText(text, lines.map((line, i) => `${i + 1}. ${line}`).join("\n"), linesStart, linesEnd);
    //TODO not sure how to handle selection yet
    return { text: updatedText, start, end };
};

/**
 * Inserts a checkbox list item at the beginning of each line in the selection.
 * @param text The full text from the input field
 * @param start The start index of the cursor/selection
 * @param end The end index of the cursor/selection
 * @returns The updated text, start, and end index of the cursor
 */
export const insertCheckboxList = (text: string, start: number, end: number): TextStyleResult => {
    const [lines, linesStart, linesEnd] = getLinesAtRange(text, start, end);
    const updatedText = replaceText(text, lines.map(line => `- [ ] ${line}`).join("\n"), linesStart, linesEnd);
    //TODO not sure how to handle selection yet
    return { text: updatedText, start, end };
};

/**
 * Inserts code quotes or a code block, depending on the selection.
 * @param text The full text from the input field
 * @param start The start index of the cursor/selection
 * @param end The end index of the cursor/selection
 * @returns The updated text, start, and end index of the cursor
 */
export const insertCode = (text: string, start: number, end: number): TextStyleResult => {
    // TODO
    return { text, start, end };
};

/**
 * Inserts quotes or a quote block, depending on the selection.
 * @param text The full text from the input field
 * @param start The start index of the cursor/selection
 * @param end The end index of the cursor/selection
 * @returns The updated text, start, and end index of the cursor
 */
export const insertQuote = (text: string, start: number, end: number): TextStyleResult => {
    // TODO
    return { text, start, end };
};

/**
 * Inserts table markdown with the required number of rows and columns.
 * @param rows The number of rows in the table
 * @param cols The number of columns in the table
 * @param text The full text from the input field
 * @param start The start index of the cursor/selection
 * @param end The end index of the cursor/selection
 * @returns The updated text, start, and end index of the cursor
 */
export const insertTable = (rows: number, cols: number, text: string, start: number, end: number): TextStyleResult => {
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
    // Insert table markdown at the cursor
    const updatedText = replaceText(text, tableStr, start, end);
    // Move the selection to the first cell
    const updatedStart = start + 2;
    const updatedEnd = updatedStart;
    return { text: updatedText, start: updatedStart, end: updatedEnd };
};
