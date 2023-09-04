import { useTheme } from "@mui/material";
import { CSSProperties, FC, useCallback, useEffect, useRef } from "react";
import { getDisplay, ListObject } from "utils/display/listTools";
import { getObjectUrl } from "utils/navigation/openObject";
import { LINE_HEIGHT_MULTIPLIER } from "../RichInputBase/RichInputBase";
import { RichInputTagDropdown, useTagDropdown } from "../RichInputTagDropdown/RichInputTagDropdown";
import { RichInputChildView, RichInputMarkdownProps } from "../types";

enum Headers {
    H1 = "h1",
    H2 = "h2",
    H3 = "h3",
    H4 = "h4",
    H5 = "h5",
    H6 = "h6",
}

const headerMarkdowns = {
    [Headers.H1]: "# ",
    [Headers.H2]: "## ",
    [Headers.H3]: "### ",
    [Headers.H4]: "#### ",
    [Headers.H5]: "##### ",
    [Headers.H6]: "###### ",
};

/**
 * Determines start index of the current line.
 * @param text Text to search.
 * @param start Index of cursor or selection start.
 * @returns Index of start of current line.
 */
const getLineStart = (text: string, start: number) => {
    if (start < 0 || start > text.length) return 0;
    return text.substring(0, start).lastIndexOf("\n") + 1;
};

/**
 * Determines end index of the current line.
 * @param text Text to search.
 * @param start Index of cursor or selection start.
 * @returns Index of end of current line.
 */
const getLineEnd = (text: string, start: number) => {
    if (start < 0 || start > text.length) return text.length;
    return text.substring(start).indexOf("\n") + start;
};

/**
 * Finds the line the specified index is on.
 * @returns The line's text, as well as its start and end index
 */
const getLineAtIndex = (text: string | null | undefined, index: number): [string, number, number] => {
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
const getLinesAtRange = (text: string, start: number, end: number): [string[], number, number] => {
    const lineStart = getLineStart(text, start);
    const lineEnd = getLineEnd(text, end);
    const lines = text.substring(lineStart, lineEnd).split("\n");
    return [lines, lineStart, lineEnd];
};

/**
 * Replaces selected text with new text.
 * @param text Text to replace in.
 * @param newText Text to replace with.
 * @param start Index of cursor or selection start.
 * @param end Index of cursor or selection end.
 * @returns New text with selected text replaced.
 */
const replaceText = (text: string, newText: string, start: number, end: number): string => {
    return text.substring(0, start) + newText + text.substring(end);
};

/**
 * Uses element ID to get start, end, and element.
 * @param id The ID of the element to get the selection of
 * @returns Object containing start, end, and element
 */
const getSelection = (id: string) => {
    const textArea = document.getElementById(id);
    if (!textArea || !(textArea instanceof HTMLTextAreaElement)) {
        console.error(`Element not found: ${id}`);
        return { start: 0, end: 0, selected: "", inputElement: null };
    }
    return {
        start: textArea.selectionStart,
        end: textArea.selectionEnd,
        selected: textArea.value.substring(textArea.selectionStart, textArea.selectionEnd),
        inputElement: textArea,
    };
};

/** TextField for entering markdown text */
export const RichInputMarkdown: FC<RichInputMarkdownProps> = ({
    autoFocus = false,
    disabled = false,
    error = false,
    getTaggableItems,
    id,
    maxRows,
    minRows = 4,
    name,
    onBlur,
    onChange,
    openAssistantDialog,
    placeholder = "",
    redo,
    tabIndex,
    toggleMarkdown,
    undo,
    value,
    sx,
}: RichInputMarkdownProps) => {
    const { palette, typography } = useTheme();

    const textAreaRef = useRef<HTMLTextAreaElement>(null);

    const insertHeader = useCallback((header: Headers) => {
        if (disabled) return;
        // Find the current selection
        const { start, inputElement } = getSelection(id);
        if (!inputElement) return;
        // Find the start of the line which the select starts on
        const startLine = getLineStart(inputElement.value, start);
        // Determine header to insert
        const headerText = headerMarkdowns[header];
        // Check if the line already starts with the header
        if (inputElement.value.substring(startLine, startLine + headerText.length) === headerText) {
            // If so, remove the header
            inputElement.value = replaceText(inputElement.value, "", startLine, startLine + headerText.length);
        } else {
            // If not, insert the header
            inputElement.value = replaceText(inputElement.value, headerText, startLine, startLine);
        }
        onChange(inputElement.value);
    }, [disabled, id, onChange]);

    /**
     * Pads selection with the given substring
     * @param padStart The substring to add before the selection
     * @param padEnd The substring to add after the selection
     */
    const padSelection = useCallback((padStart: string, padEnd: string) => {
        if (disabled) return;
        // Find the current selection
        const { start, end, inputElement } = getSelection(id);
        if (!inputElement) return;
        // Insert pad around selection
        inputElement.value = inputElement.value.substring(0, start) + padStart + inputElement.value.substring(start, end) + padEnd + inputElement.value.substring(end);
        onChange(inputElement.value);
        // Move cursor to end of selection (by default it would be at the end of the padEnd string, which is not desired)
        inputElement.selectionStart = end + padStart.length;
        inputElement.selectionEnd = inputElement.selectionStart;
    }, [disabled, id, onChange]);

    const strikethrough = useCallback(() => { if (!disabled) padSelection("~~", "~~"); }, [disabled, padSelection]);
    const bold = useCallback(() => { if (!disabled) padSelection("**", "**"); }, [disabled, padSelection]);
    const italic = useCallback(() => { if (!disabled) padSelection("*", "*"); }, [disabled, padSelection]);
    const spoiler = useCallback(() => { if (!disabled) padSelection("||", "||"); }, [disabled, padSelection]);
    const underline = useCallback(() => { if (!disabled) padSelection("<u>", "</u>"); }, [disabled, padSelection]);

    const insertLink = useCallback(() => {
        if (disabled) return;
        // Find the current selection
        const { start, end, inputElement } = getSelection(id);
        if (!inputElement) return;
        // If no selection, insert [link](url) at the cursor
        if (start === end) {
            inputElement.value = inputElement.value.substring(0, start) + "[display text](url)" + inputElement.value.substring(end);
            onChange(inputElement.value);
            return;
        }
        // Otherwise, call padSelection
        padSelection("[", "](url)");
    }, [disabled, id, onChange, padSelection]);

    const insertBulletList = useCallback(() => {
        if (disabled) return;
        const { start, end, inputElement } = getSelection(id);
        if (!inputElement) return;
        const [lines, linesStart, linesEnd] = (getLinesAtRange(inputElement.value, start, end) ?? []);
        const newValue = replaceText(inputElement.value, lines.map(line => `* ${line}`).join("\n"), linesStart, linesEnd);
        inputElement.value = newValue;
        onChange(newValue);
    }, [disabled, id, onChange]);

    const insertNumberList = useCallback(() => {
        if (disabled) return;
        const { start, end, inputElement } = getSelection(id);
        if (!inputElement) return;
        const [lines, linesStart, linesEnd] = (getLinesAtRange(inputElement.value, start, end) ?? []);
        const newValue = replaceText(inputElement.value, lines.map((line, i) => `${i + 1}. ${line}`).join("\n"), linesStart, linesEnd);
        inputElement.value = newValue;
        onChange(newValue);
    }, [disabled, id, onChange]);

    const insertCheckboxList = useCallback(() => {
        if (disabled) return;
        const { start, end, inputElement } = getSelection(id);
        if (!inputElement) return;
        const [lines, linesStart, linesEnd] = (getLinesAtRange(inputElement.value, start, end) ?? []);
        const newValue = replaceText(inputElement.value, lines.map(line => `- [ ] ${line}`).join("\n"), linesStart, linesEnd);
        inputElement.value = newValue;
        onChange(newValue);
    }, [disabled, id, onChange]);

    const insertTable = useCallback(({ rows, cols }: { rows: number, cols: number }) => {
        console.log("in insertTable", rows, cols);
        if (disabled) return;
        const { start, inputElement } = getSelection(id);
        if (!inputElement) return;
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
        console.log("got tableStr", tableStr);
        // Insert the generated table into the text
        inputElement.value = replaceText(
            inputElement.value,
            tableStr + "\n", // Add an extra newline for separation
            start,
            start,
        );
        onChange(inputElement.value);
    }, [disabled, id, onChange]);


    const tagData = useTagDropdown({ getTaggableItems });
    const selectDropdownItem = useCallback((item: ListObject) => {
        // Tagged item is inserted as a link
        const asLink = `[@${getDisplay(item).title}](${window.location.origin}${getObjectUrl(item)})`;
        // Insert the link, replacing the tag string and the "@" symbol
        const { inputElement } = getSelection(id);
        if (!inputElement) return;
        inputElement.value = replaceText(
            inputElement.value,
            asLink,
            inputElement.selectionStart - tagData.tagString.length - 1,
            inputElement.selectionEnd,
        );
        onChange(inputElement.value);
        // Close the dropdown
        tagData.setAnchorEl(null);
    }, [id, onChange, tagData]);

    (RichInputMarkdown as unknown as RichInputChildView).handleAction = (action, data) => {
        const actionMap = {
            "Assistant": () => openAssistantDialog(getSelection(id).selected),
            "Bold": bold,
            "Code": () => { }, //TODO
            "Header1": () => insertHeader(Headers.H1),
            "Header2": () => insertHeader(Headers.H2),
            "Header3": () => insertHeader(Headers.H3),
            "Italic": italic,
            "Link": insertLink,
            "ListBullet": insertBulletList,
            "ListCheckbox": insertCheckboxList,
            "ListNumber": insertNumberList,
            "Quote": () => { }, //TODO
            "Redo": redo,
            "Spoiler": spoiler,
            "Strikethrough": strikethrough,
            "Table": () => insertTable(data as { rows: number, cols: number }),
            "Underline": underline,
            "Undo": undo,
        };
        const actionFunction = actionMap[action];
        if (actionFunction) actionFunction();
    };

    // Listen for text input changes
    useEffect(() => {
        // Map keyboard shortcuts to their respective functions
        const keyMappings = {
            "1": () => insertHeader(Headers.H1), // ALT + 1 - Insert header 1
            "2": () => insertHeader(Headers.H2), // ALT + 2 - Insert header 2
            "3": () => insertHeader(Headers.H3), // ALT + 3 - Insert header 3
            "4": () => insertHeader(Headers.H4), // ALT + 4 - Insert header 4
            "5": () => insertHeader(Headers.H5), // ALT + 5 - Insert header 5
            "6": () => insertHeader(Headers.H6), // ALT + 6 - Insert header 6
            "7": () => insertBulletList(), // ALT + 4 - Bullet list
            "8": () => insertNumberList(), // ALT + 5 - Number list
            "9": () => insertCheckboxList(), // ALT + 6 - Checklist
            "0": () => toggleMarkdown(), // ALT + 7 - Toggle preview
            "b": () => bold(), // CTRL + B - Bold
            "i": () => italic(), // CTRL + I - Italic
            "k": () => insertLink(), // CTRL + K - Insert link
            "z": () => undo(), // CTRL + Z - Undo
            "Z": () => redo(), // CTRL + SHIFT + Z = Redo
            "S": () => strikethrough(), // CTRL + SHIFT + S - Strikethrough
            "l": () => spoiler(), // CTRL + L - Spoiler
            "u": () => underline(), // CTRL + U - Underline
            "y": () => redo(), // CTRL + Y = Redo
        };
        // Handle key press events for textarea
        const handleTextareaKeyDown = (e: any) => {
            // Check if either alt or ctrl key is pressed
            if (e.altKey || e.ctrlKey) {
                const action = keyMappings[e.key]; // Find the function mapped to the key press
                // If a function is found, prevent default action and call the function
                if (action) {
                    console.log("textarea action triggered");
                    e.preventDefault();
                    action();
                    return;
                }
            }
            console.log("tag info:", tagData.anchorEl, typeof getTaggableItems, e.key);
            // Handle tag dropdown. Triggered by "@" key press
            if (!tagData.anchorEl && typeof getTaggableItems === "function" && e.key === "@") {
                console.log("opening dropdown for tags");
                tagData.setTagString("");
                tagData.setList([]);
                tagData.setAnchorEl(textAreaRef.current);
            } else if (tagData.anchorEl) {
                // Normal characters (e.g. letters, numbers, emojis) are added to the tag string
                if (e.key.length === 1) {
                    tagData.setTagString(tagData.tagString + e.key);
                }
                // Backspace removes the last character from the tag string
                else if (e.key === "Backspace") {
                    tagData.setTagString(tagData.tagString.slice(0, -1));
                }
                // Escape ends the query
                else if (e.key === "Escape") {
                    tagData.setAnchorEl(null);
                }
                // Tab and arrow keys cycle through the dropdown items
                else if (e.key === "Tab" || e.key === "ArrowDown" || e.key === "ArrowUp") {
                    e.preventDefault();
                    let newIndex = tagData.tabIndex;
                    // Increment the index if tab or arrow down is pressed without the shift key
                    if ((e.key === "Tab" && !e.shiftKey) || e.key === "ArrowDown") {
                        newIndex = (newIndex + 1) % tagData.list.length;
                    }
                    // Decrement the index if shift+tab or arrow up is pressed
                    else if ((e.key === "Tab" && e.shiftKey) || e.key === "ArrowUp") {
                        newIndex = (newIndex - 1 + tagData.list.length) % tagData.list.length;
                    }
                    tagData.setTabIndex(newIndex);
                }
                // Enter selects the first item in the dropdown
                else if (e.key === "Enter") {
                    if (tagData.list.length > 0) {
                        e.preventDefault();
                        const selectedItem = tagData.tabIndex >= 0 && tagData.tabIndex < tagData.list.length ?
                            tagData.list[tagData.tabIndex] :
                            tagData.list[0];
                        selectDropdownItem(selectedItem);
                    }
                    tagData.setAnchorEl(null);
                    // Return so that the enter key press is not handled by the textarea
                    return;
                }
            }
            // On enter key press
            if (e.key === "Enter") {
                // Find the line the start of the selection is on
                const { start, end, inputElement } = getSelection(id);
                let [trimmedLine] = getLineAtIndex(inputElement?.value, start);
                trimmedLine = trimmedLine.trimStart();
                console.log("enter key trimmed line", trimmedLine, value, start, end, Object.entries(e.target));
                const isNumberList = /^\d+\.\s/.test(trimmedLine);
                const isCheckboxList = trimmedLine.startsWith("- [ ] ") || trimmedLine.startsWith("- [x] ");
                const isBulletDashList = trimmedLine.startsWith("- ");
                const isBulletStarList = trimmedLine.startsWith("* ");
                console.log("key was enter", isNumberList, isCheckboxList, isBulletDashList, isBulletStarList);
                // If the current line is a list
                if (isNumberList || isCheckboxList || isBulletDashList || isBulletStarList) {
                    e.preventDefault();
                    const { inputElement } = getSelection(id);
                    if (!inputElement) return;
                    let textToInsert = "\n";
                    if (isNumberList) {
                        const number = Number(trimmedLine.match(/^\d+/)![0]) + 1;
                        textToInsert += `${number}. `;
                    } else if (isCheckboxList) {
                        textToInsert += "- [ ] ";
                    } else if (isBulletDashList) {
                        textToInsert += "- ";
                    } else if (isBulletStarList) {
                        textToInsert += "* ";
                    }
                    console.log("enter key inserting text", textToInsert, start, end);
                    inputElement.value = replaceText(value, textToInsert, start, end);
                    onChange(inputElement.value);
                }
            }
            // On "Esc" key press, don't propagate the event
            else if (e.key === "Escape") {
                e.stopPropagation();
            }
            console.log("made it to the end");
        };
        // Handle key press events for preview
        const handleFullComponentKeyDown = (e: any) => {
            // Only check for the toggle preview shortcut
            if (e.altKey && e.key === "6") {
                console.log("fullcomponent action triggered");
                e.preventDefault();
                toggleMarkdown();
            }
        };
        // Find textarea or full component
        const textarea = document.getElementById(id);
        const fullComponent = document.getElementById(`markdown-input-base-${name}`);
        // Add appropriate listeners
        if (textarea) textarea.addEventListener("keydown", handleTextareaKeyDown);
        else if (fullComponent) fullComponent.addEventListener("keydown", handleFullComponentKeyDown);
        // Return a cleanup function to remove the listeners on unmount
        return () => {
            textarea?.removeEventListener("keydown", handleTextareaKeyDown);
            fullComponent?.removeEventListener("keydown", handleFullComponentKeyDown);
        };
    }, [bold, getTaggableItems, onChange, insertBulletList, insertCheckboxList, insertHeader, insertLink, insertNumberList, italic, name, redo, strikethrough, toggleMarkdown, undo, id, tagData, selectDropdownItem, value, spoiler, underline]);

    return (
        <>
            <RichInputTagDropdown {...tagData} selectDropdownItem={selectDropdownItem} />
            <textarea
                id={id}
                ref={textAreaRef}
                autoFocus={autoFocus}
                disabled={disabled}
                name={name}
                placeholder={placeholder}
                rows={minRows}
                value={value}
                onBlur={onBlur}
                onChange={(e) => { onChange(e.target.value); }}
                tabIndex={tabIndex}
                spellCheck
                style={{
                    padding: "16.5px 14px",
                    minWidth: "-webkit-fill-available",
                    maxWidth: "-webkit-fill-available",
                    outline: "none",
                    resize: "none",
                    borderColor: error ? palette.error.main : palette.divider,
                    borderRadius: "0 0 4px 4px",
                    fontFamily: typography.fontFamily,
                    fontSize: typography.fontSize + 2,
                    lineHeight: `${Math.round(typography.fontSize * LINE_HEIGHT_MULTIPLIER)}px`,
                    backgroundColor: palette.background.paper,
                    color: palette.text.primary,
                    ...sx,
                } as CSSProperties}
            />
        </>
    );
};
