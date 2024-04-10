import { FC, useCallback, useEffect, useRef } from "react";
import { ListObject, getDisplay } from "utils/display/listTools";
import { Headers, TextStyleResult, getLineAtIndex, getTextSelection, insertBulletList, insertCheckboxList, insertCode, insertHeader, insertLink, insertNumberList, insertQuote, insertTable, padSelection, replaceText } from "utils/display/stringTools";
import { getObjectUrl } from "utils/navigation/openObject";
import { RichInputTagDropdown, useTagDropdown } from "../RichInputTagDropdown/RichInputTagDropdown";
import { TextInput } from "../TextInput/TextInput";
import { RichInputAction, RichInputMarkdownProps } from "../types";

type TextStyle = Extract<RichInputAction, "Bold" | "Code" | "Header1" | "Header2" | "Header3" | "Header4" | "Header5" | "Header6" | "Italic" | "Link" | "ListBullet" | "ListCheckbox" | "ListNumber" | "Quote" | "Spoiler" | "Strikethrough" | "Underline">;

const styleMap = (common: [string, number, number]): Record<TextStyle, () => TextStyleResult> => ({
    "Bold": () => padSelection("**", "**", ...common),
    "Code": () => insertCode(...common),
    "Header1": () => insertHeader(Headers.H1, ...common),
    "Header2": () => insertHeader(Headers.H2, ...common),
    "Header3": () => insertHeader(Headers.H3, ...common),
    "Header4": () => insertHeader(Headers.H4, ...common),
    "Header5": () => insertHeader(Headers.H5, ...common),
    "Header6": () => insertHeader(Headers.H6, ...common),
    "Italic": () => padSelection("*", "*", ...common),
    "Link": () => insertLink(...common),
    "ListBullet": () => insertBulletList(...common),
    "ListCheckbox": () => insertCheckboxList(...common),
    "ListNumber": () => insertNumberList(...common),
    "Quote": () => insertQuote(...common),
    "Spoiler": () => padSelection("||", "||", ...common),
    "Strikethrough": () => padSelection("~~", "~~", ...common),
    "Underline": () => padSelection("<u>", "</u>", ...common),
});

/** TextInput for entering markdown text */
export const RichInputMarkdown: FC<RichInputMarkdownProps> = ({
    autoFocus = false,
    disabled = false,
    enterWillSubmit,
    error = false,
    getTaggableItems,
    id,
    maxRows,
    minRows = 4,
    name,
    onBlur,
    onFocus,
    onChange,
    onSubmit,
    openAssistantDialog,
    placeholder = "",
    redo,
    setHandleAction,
    tabIndex,
    toggleMarkdown,
    undo,
    value,
    sxs,
}: RichInputMarkdownProps) => {
    const textAreaRef = useRef<HTMLElement>(null);

    const insertStyle = useCallback((style: TextStyle | `${TextStyle}`) => {
        if (disabled) return;
        // Find the current selection
        const { start, end, inputElement } = getTextSelection(id);
        if (!inputElement) return;
        // Modify text based on the style
        const styleFuncs = styleMap([inputElement.value, start, end]);
        if (!(style in styleFuncs)) {
            console.error("Invalid style", style);
            return;
        }
        const result = styleFuncs[style]();
        // Set the new value and selection
        inputElement.value = result.text;
        inputElement.selectionStart = result.start;
        inputElement.selectionEnd = result.end;
        // Update parent component
        onChange(result.text);
    }, [disabled, id, onChange]);

    const addTable = useCallback(({ rows, cols }: { rows: number, cols: number }) => {
        if (disabled) return;
        const { start, end, inputElement } = getTextSelection(id);
        if (!inputElement) return;
        const result = insertTable(rows, cols, inputElement.value, start, end);
        // Set the new value and selection
        inputElement.value = result.text;
        inputElement.selectionStart = result.start;
        inputElement.selectionEnd = result.end;
        // Update parent component
        onChange(result.text);
    }, [disabled, id, onChange]);

    const tagData = useTagDropdown({ getTaggableItems });
    const selectDropdownItem = useCallback((item: ListObject) => {
        // Tagged item is inserted as a link
        const asLink = `[@${getDisplay(item).title}](${window.location.origin}${getObjectUrl(item)})`;
        // Insert the link, replacing the tag string and the "@" symbol
        const { inputElement } = getTextSelection(id);
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

    useEffect(() => {
        if (!setHandleAction) return;
        setHandleAction((action, data) => {
            // Anything that isn't a style action goes in the actionMap
            const actionMap = {
                "Assistant": () => openAssistantDialog(getTextSelection(id).selected),
                "Redo": redo,
                "SetValue": () => {
                    if (typeof data !== "string") {
                        console.error("Invalid data for SetValue action", data);
                        return;
                    }
                    // Set value without triggering onChange
                    const { inputElement } = getTextSelection(id);
                    if (!inputElement) return;
                    inputElement.value = data;
                },
                "Table": () => addTable(data as { rows: number, cols: number }),
                "Undo": undo,
            };
            const actionFunction = actionMap[action];
            if (actionFunction) {
                actionFunction();
            }
            // Handle all style actions 
            else {
                insertStyle(action as TextStyle);
            }
        });
    }, [addTable, id, insertStyle, openAssistantDialog, redo, setHandleAction, undo]);

    // Listen for text input changes
    useEffect(() => {
        // Map keyboard shortcuts to their respective functions
        const keyMappings = {
            "1": () => insertStyle("Header1"), // ALT + 1 - Insert header 1
            "2": () => insertStyle("Header2"), // ALT + 2 - Insert header 2
            "3": () => insertStyle("Header3"), // ALT + 3 - Insert header 3
            "4": () => insertStyle("Header4"), // ALT + 4 - Insert header 4
            "5": () => insertStyle("Header5"), // ALT + 5 - Insert header 5
            "6": () => insertStyle("Header6"), // ALT + 6 - Insert header 6
            "7": () => insertStyle("ListBullet"), // ALT + 4 - Bullet list
            "8": () => insertStyle("ListNumber"), // ALT + 5 - Number list
            "9": () => insertStyle("ListCheckbox"), // ALT + 6 - Checklist
            "0": () => toggleMarkdown(), // ALT + 7 - Toggle preview
            "b": () => insertStyle("Bold"), // CTRL + B - Bold
            "i": () => insertStyle("Italic"), // CTRL + I - Italic
            "k": () => insertStyle("Link"), // CTRL + K - Insert link
            "z": () => undo(), // CTRL + Z - Undo
            "Z": () => redo(), // CTRL + SHIFT + Z = Redo
            "S": () => insertStyle("Strikethrough"), // CTRL + SHIFT + S - Strikethrough
            "l": () => insertStyle("Spoiler"), // CTRL + L - Spoiler
            "u": () => insertStyle("Underline"), // CTRL + U - Underline
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
                const { start, end, inputElement } = getTextSelection(id);
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
                    const { inputElement } = getTextSelection(id);
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
    }, [getTaggableItems, onChange, insertStyle, name, redo, toggleMarkdown, undo, id, tagData, selectDropdownItem, value]);

    return (
        <>
            <RichInputTagDropdown {...tagData} selectDropdownItem={selectDropdownItem} />
            <TextInput
                id={id}
                ref={textAreaRef}
                autoFocus={autoFocus}
                disabled={disabled}
                enterWillSubmit={enterWillSubmit}
                maxRows={maxRows}
                minRows={minRows}
                multiline
                name={name}
                placeholder={placeholder}
                value={value}
                onBlur={onBlur}
                onFocus={onFocus}
                onChange={(e) => { onChange(e.target.value); }}
                onSubmit={() => { onSubmit?.(value); }}
                tabIndex={tabIndex}
                spellCheck
                sx={{
                    minWidth: "-webkit-fill-available",
                    maxWidth: "-webkit-fill-available",
                    outline: "none",
                    resize: "none",
                    "& .MuiOutlinedInput-notchedOutline": {
                        borderRadius: "0 0 4px 4px",
                        borderTop: "none",
                    },
                    "& .MuiInputBase-root": {
                        "& > textarea": {
                            ...sxs?.inputRoot?.["& > textarea"],
                            ...sxs?.textArea,
                        },
                    },
                    ...sxs?.inputRoot,
                }}
            />
        </>
    );
};
