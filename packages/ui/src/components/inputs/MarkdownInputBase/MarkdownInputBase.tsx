/**
 * TextField for entering (and previewing) markdown.
 */
import { Box, CircularProgress, IconButton, List, ListItem, Popover, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { ColorIconButton } from "components/buttons/ColorIconButton/ColorIconButton";
import { CharLimitIndicator } from "components/CharLimitIndicator/CharLimitIndicator";
import { AssistantDialog } from "components/dialogs/AssistantDialog/AssistantDialog";
import { AssistantDialogProps } from "components/dialogs/types";
import { MarkdownDisplay } from "components/text/MarkdownDisplay/MarkdownDisplay";
import { BoldIcon, Header1Icon, Header2Icon, Header3Icon, HeaderIcon, InvisibleIcon, ItalicIcon, LinkIcon, ListBulletIcon, ListCheckIcon, ListIcon, ListNumberIcon, MagicIcon, RedoIcon, StrikethroughIcon, UndoIcon, VisibleIcon } from "icons";
import React, { useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { linkColors, noSelect } from "styles";
import { getCurrentUser } from "utils/authentication/session";
import { keyComboToString } from "utils/display/device";
import { getDisplay, ListObjectType } from "utils/display/listTools";
import { useDebounce } from "utils/hooks/useDebounce";
import { getObjectUrl } from "utils/navigation/openObject";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { MarkdownInputBaseProps } from "../types";

interface TagItemDropdownPros {
    anchorEl: HTMLElement | null;
    focusedIndex: number;
    handleClose: () => void;
    handleFocusChange: (index: number) => void;
    handleSelect: (item: ListObjectType) => void;
    items: ListObjectType[];
}

/** When using "@" to tag an object, displays options */
const TagItemDropdown = ({
    anchorEl,
    focusedIndex,
    handleClose,
    handleFocusChange,
    handleSelect,
    items,
}: TagItemDropdownPros) => {
    const [popoverWidth, setPopoverWidth] = useState<number | null>(null);

    useEffect(() => {
        if (anchorEl) {
            setPopoverWidth(anchorEl.offsetWidth);
        }
    }, [anchorEl]);

    const listItems = items.map((item, index) => (
        <ListItem
            key={item.id}
            selected={focusedIndex === index}
            onMouseEnter={() => handleFocusChange(index)}
            onClick={() => handleSelect(item)}
            sx={{
                ...noSelect,
                cursor: "pointer",
            }}
        >
            {item.name}
        </ListItem>
    ));

    return (
        <Popover
            disableAutoFocus={true}
            open={Boolean(anchorEl)}
            anchorEl={anchorEl}
            onClose={handleClose}
            anchorOrigin={{
                vertical: "top",
                horizontal: "left",
            }}
            transformOrigin={{
                vertical: "bottom",
                horizontal: "left",
            }}
            sx={{
                "& .MuiPopover-paper": {
                    width: popoverWidth,
                },
            }}
        >
            {items.length > 0 ? <List>
                {listItems}
            </List> :
                <Box sx={{
                    padding: 2,
                    display: "flex",
                    justifyContent: "center",
                    alignItems: "center",
                }}>
                    <CircularProgress size={32} />
                </Box>
            }
        </Popover>
    );
};

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

const dropDownButtonProps = ({
    borderRadius: 0,
    width: "48px",
    height: "48px",
});

/**
 * Determines start index of the current line.
 * @param text Text to search.
 * @param selectionStart Index of cursor or selection start.
 * @returns Index of start of current line.
 */
const getLineStart = (text: string, selectionStart: number) => {
    if (selectionStart < 0 || selectionStart > text.length) return 0;
    return text.substring(0, selectionStart).lastIndexOf("\n") + 1;
};

/**
 * Determines end index of the current line.
 * @param text Text to search.
 * @param selectionStart Index of cursor or selection start.
 * @returns Index of end of current line.
 */
const getLineEnd = (text: string, selectionStart: number) => {
    if (selectionStart < 0 || selectionStart > text.length) return text.length;
    return text.substring(selectionStart).indexOf("\n") + selectionStart;
};

/**
 * Determines line the specified index is on.
 * @param text The entire text
 * @param index The index to search for
 * @returns The line the cursor is on (or null), as well as its start and end index
 */
const getLineAtIndex = (text: string, index: number): [string, number, number] => {
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
 * Uses element ID to get selectionStart, selectionEnd, and element.
 * @param id The ID of the element to get the selection of
 * @returns Object containing selectionStart, selectionEnd, and element
 */
const getSelection = (id: string): { selectionStart: number, selectionEnd: number, textArea: HTMLTextAreaElement } => {
    const textArea = document.getElementById(id);
    if (!textArea || !(textArea instanceof HTMLTextAreaElement)) throw new Error(`Element not found: ${id}`);
    return { selectionStart: textArea.selectionStart, selectionEnd: textArea.selectionEnd, textArea };
};

export const MarkdownInputBase = ({
    actionButtons,
    autoFocus = false,
    disabled = false,
    disableAssistant = false,
    error = false,
    getTaggableItems,
    helperText,
    maxChars,
    maxRows,
    minRows = 4,
    name,
    onBlur,
    onChange,
    placeholder = "",
    tabIndex,
    value,
    sxs,
    zIndex,
}: MarkdownInputBaseProps) => {
    const { palette, typography } = useTheme();
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    const { hasPremium } = useMemo(() => getCurrentUser(session), [session]);

    // Stores previous states for undo/redo (since we can't use the browser's undo/redo due to programmatic changes)
    const changeStack = useRef<string[]>([value]);
    const [changeStackIndex, setChangeStackIndex] = useState<number>(0);

    // Internal value (since value passed back is debounced)
    const [internalValue, setInternalValue] = useState<string>(value);
    useEffect(() => {
        // If new value is one of the recent items in the stack 
        // (i.e. debounce is firing while user is still typing),
        // then don't update the internal value
        const recentItems = changeStack.current.slice(Math.max(changeStack.current.length - 5, 0));
        if (value === "" || !recentItems.includes(value)) {
            setInternalValue(value);
        }
    }, [value]);
    // Debounce text change
    const onChangeDebounced = useDebounce(onChange, 200);

    // Flash preview button when user tries to edit text while preview is on
    const [flashPreview, setFlashPreview] = useState(false);
    useEffect(() => {
        let timer: NodeJS.Timeout;
        if (flashPreview) {
            timer = setTimeout(() => {
                setFlashPreview(false);
            }, 500);
        }
        return () => {
            clearTimeout(timer);
        };
    }, [flashPreview]);

    const isButtonDebounced = useRef(false);
    const startDebounce = useCallback(() => {
        isButtonDebounced.current = true;
        setTimeout(() => {
            isButtonDebounced.current = false;
        }, 100);
    }, []);

    const [isPreviewOn, setIsPreviewOn] = useState(false);
    const togglePreview = useCallback(() => { setIsPreviewOn(on => !on); }, []);
    const togglePreviewDebounce = useDebounce(togglePreview, 50);
    const checkIfCanEdit = useCallback(() => {
        if (isPreviewOn) {
            setFlashPreview(true);
            return false;
        }
        return true;
    }, [isPreviewOn]);
    // Switch focus when preview is toggled
    useEffect(() => {
        // Ignore if no changes have been made (starts with initial value, so length is 1)
        if (changeStack.current.length <= 1) return;
        // Set focus after a short delay to allow for debouncing of togglePreview
        console.log("switching focus", isPreviewOn, document.getElementById(`markdown-input-base-${name}`));
        // If turning preview on, change focus to the full markdown input component
        if (isPreviewOn) document.getElementById(`markdown-input-base-${name}`)?.focus();
        // Otherwise, change focus to the input element
        else document.getElementById(`markdown-input-${name}`)?.focus();
    }, [isPreviewOn, name]);

    /** Moves back one in the change stack */
    const undo = useCallback(() => {
        if (checkIfCanEdit() && changeStackIndex > 0) {
            setChangeStackIndex(changeStackIndex - 1);
            setInternalValue(changeStack.current[changeStackIndex - 1]);
            onChangeDebounced(changeStack.current[changeStackIndex - 1]);
        }
    }, [changeStackIndex, checkIfCanEdit, onChangeDebounced]);
    const canUndo = useMemo(() => changeStackIndex > 0 && changeStack.current.length > 0, [changeStackIndex]);
    /** Moves forward one in the change stack */
    const redo = useCallback(() => {
        if (checkIfCanEdit() && changeStackIndex < changeStack.current.length - 1) {
            setChangeStackIndex(changeStackIndex + 1);
            setInternalValue(changeStack.current[changeStackIndex + 1]);
            onChangeDebounced(changeStack.current[changeStackIndex + 1]);
        }
    }, [changeStackIndex, checkIfCanEdit, onChangeDebounced]);
    const canRedo = useMemo(() => changeStackIndex < changeStack.current.length - 1 && changeStack.current.length > 0, [changeStackIndex]);
    /**
     * Adds, to change stack, and removes anything from the change stack after the current index
     */
    const handleChange = useCallback((updatedText: string) => {
        const newChangeStack = [...changeStack.current];
        newChangeStack.splice(changeStackIndex + 1, newChangeStack.length - changeStackIndex - 1);
        newChangeStack.push(updatedText);
        changeStack.current = newChangeStack;
        setChangeStackIndex(newChangeStack.length - 1);
        setInternalValue(updatedText);
        onChangeDebounced(updatedText);
    }, [changeStackIndex, onChangeDebounced]);

    const [assistantDialogProps, setAssistantDialogProps] = useState<AssistantDialogProps>({
        context: undefined,
        isOpen: false,
        task: "note",
        handleClose: () => { setAssistantDialogProps(props => ({ ...props, isOpen: false })); },
        handleComplete: (data) => { console.log("completed", data); setAssistantDialogProps(props => ({ ...props, isOpen: false })); },
        zIndex: zIndex + 1,
    });
    const openAssistantDialog = useCallback(() => {
        if (!checkIfCanEdit()) return;
        // We want to provide the assistant with the most relevant context
        let context: string | undefined = undefined;
        const maxContextLength = 1500;
        // Get highlighted text
        const { selectionStart, selectionEnd, textArea } = getSelection(`markdown-input-${name}`);
        let highlightedText: string | undefined = textArea.value.substring(selectionStart, selectionEnd).trim();
        if (highlightedText.length > maxContextLength) highlightedText = highlightedText.substring(0, maxContextLength);
        if (highlightedText.length > 0) context = highlightedText;
        // If there's not highlighted text, provide the full text if it's not too long
        else if (internalValue.length <= maxContextLength) context = internalValue;
        // Otherwise, provide the last 1500 characters
        else context = internalValue.substring(internalValue.length - maxContextLength, internalValue.length);
        // Open the assistant dialog
        console.log("context here", context, highlightedText, selectionStart, selectionEnd, textArea.value);
        setAssistantDialogProps(props => ({ ...props, isOpen: true, context: context ? `\`\`\`\n${context}\n\`\`\`\n\n` : undefined }));
    }, [checkIfCanEdit, internalValue, name]);

    const [headerAnchorEl, setHeaderAnchorEl] = useState<HTMLElement | null>(null);
    const openHeaderSelect = (event: React.MouseEvent<HTMLElement>) => {
        if (!checkIfCanEdit()) return;
        setHeaderAnchorEl(event.currentTarget);
    };
    const closeHeader = () => { setHeaderAnchorEl(null); };
    const headerSelectOpen = Boolean(headerAnchorEl);

    const [listAnchorEl, setListAnchorEl] = useState<HTMLElement | null>(null);
    const openListSelect = (event: React.MouseEvent<HTMLElement>) => {
        if (!checkIfCanEdit()) return;
        setListAnchorEl(event.currentTarget);
    };
    const closeList = () => { setListAnchorEl(null); };
    const listSelectOpen = Boolean(listAnchorEl);

    const insertHeader = useCallback((header: Headers) => {
        if (!checkIfCanEdit()) return;
        // Find the current selection
        const { selectionStart, textArea } = getSelection(`markdown-input-${name}`);
        // Find the start of the line which the select starts on
        const startLine = getLineStart(textArea.value, selectionStart);
        // Determine header to insert
        const headerText = headerMarkdowns[header];
        // Check if the line already starts with the header
        if (textArea.value.substring(startLine, startLine + headerText.length) === headerText) {
            // If so, remove the header
            textArea.value = replaceText(textArea.value, "", startLine, startLine + headerText.length);
        } else {
            // If not, insert the header
            textArea.value = replaceText(textArea.value, headerText, startLine, startLine);
        }
        handleChange(textArea.value);
        closeHeader();
    }, [checkIfCanEdit, handleChange, name]);

    /**
     * Pads selection with the given substring
     * @param padStart The substring to add before the selection
     * @param padEnd The substring to add after the selection
     */
    const padSelection = useCallback((padStart: string, padEnd: string) => {
        if (!checkIfCanEdit()) return;
        // Find the current selection
        const { selectionStart, selectionEnd, textArea } = getSelection(`markdown-input-${name}`);
        // If no selection, return
        if (selectionStart === selectionEnd) {
            PubSub.get().publishSnack({ messageKey: "NoTextSelected", severity: "Error" });
            return;
        }
        // Insert ~~ before the selection, and ~~ after the selection
        textArea.value = textArea.value.substring(0, selectionStart) + padStart + textArea.value.substring(selectionStart, selectionEnd) + padEnd + textArea.value.substring(selectionEnd);
        handleChange(textArea.value);
    }, [checkIfCanEdit, handleChange, name]);

    const strikethrough = useCallback(() => { padSelection("~~", "~~"); }, [padSelection]);
    const bold = useCallback(() => { padSelection("**", "**"); }, [padSelection]);
    const italic = useCallback(() => { padSelection("*", "*"); }, [padSelection]);

    const insertLink = useCallback(() => {
        if (!checkIfCanEdit()) return;
        // Find the current selection
        const { selectionStart, selectionEnd, textArea } = getSelection(`markdown-input-${name}`);
        // If no selection, insert [link](url) at the cursor
        if (selectionStart === selectionEnd) {
            textArea.value = textArea.value.substring(0, selectionStart) + "[display text](url)" + textArea.value.substring(selectionEnd);
            onChange(textArea.value);
            return;
        }
        // Otherwise, call padSelection
        padSelection("[", "](url)");
    }, [checkIfCanEdit, name, onChange, padSelection]);

    const insertBulletList = useCallback(() => {
        if (!checkIfCanEdit()) return;
        const { selectionStart, selectionEnd, textArea } = getSelection(`markdown-input-${name}`);
        const [lines, linesStart, linesEnd] = (getLinesAtRange(textArea.value, selectionStart, selectionEnd) ?? []);
        const newValue = replaceText(textArea.value, lines.map(line => `* ${line}`).join("\n"), linesStart, linesEnd);
        textArea.value = newValue;
        handleChange(newValue);
        closeList();
    }, [checkIfCanEdit, handleChange, name]);

    const insertNumberList = useCallback(() => {
        if (!checkIfCanEdit()) return;
        const { selectionStart, selectionEnd, textArea } = getSelection(`markdown-input-${name}`);
        const [lines, linesStart, linesEnd] = (getLinesAtRange(textArea.value, selectionStart, selectionEnd) ?? []);
        const newValue = replaceText(textArea.value, lines.map((line, i) => `${i + 1}. ${line}`).join("\n"), linesStart, linesEnd);
        textArea.value = newValue;
        handleChange(newValue);
        closeList();
    }, [checkIfCanEdit, handleChange, name]);

    const insertCheckboxList = useCallback(() => {
        if (!checkIfCanEdit()) return;
        const { selectionStart, selectionEnd, textArea } = getSelection(`markdown-input-${name}`);
        const [lines, linesStart, linesEnd] = (getLinesAtRange(textArea.value, selectionStart, selectionEnd) ?? []);
        const newValue = replaceText(textArea.value, lines.map(line => `- [ ] ${line}`).join("\n"), linesStart, linesEnd);
        textArea.value = newValue;
        handleChange(newValue);
        closeList();
    }, [checkIfCanEdit, handleChange, name]);

    // Prevents the textArea from removing its highlight when one 
    // of the buttons is clicked
    const handleMouseDown = useCallback((e) => {
        if (isPreviewOn) return;
        // Get selection data
        const { selectionStart, selectionEnd } = getSelection(`markdown-input-${name}`);
        // Get target element id
        const targetId = e.target.id;
        // If the target is not the textArea, and the selection is not empty, then prevent default
        if (targetId !== `markdown-input-${name}` && selectionStart !== selectionEnd) {
            e.preventDefault();
            e.stopPropagation();
        }
        // e.preventDefault() 
    }, [isPreviewOn, name]);

    // Handle "@" dropdown for tagging users or any other items
    const [dropdownAnchorEl, setDropdownAnchorEl] = useState<HTMLElement | null>(null);
    const [dropdownTabIndex, setDropdownTabIndex] = useState(0); // The index of the currently selected item in the dropdown
    const [tagString, setTagString] = useState(""); // What has been typed after the "@"
    const [dropdownList, setDropdownList] = useState<ListObjectType[]>([]); // The list of items currently being displayed
    const [cachedTags, setCachedTags] = useState<Record<string, ListObjectType[]>>({}); // The cached list of items for each tag string, to prevent unnecessary queries
    // Callback to parent to get items, or to get cached items if they exist
    useEffect(() => {
        if (!dropdownAnchorEl || typeof getTaggableItems !== "function") return;

        const fetchItems = async () => {
            // Check the cache
            if (cachedTags[tagString] !== undefined) {
                setDropdownList(cachedTags[tagString]);
            }
            // Fallback to calling the parent
            else {
                const items = await getTaggableItems(tagString);
                // Store the items in the cache
                setCachedTags(prev => ({ ...prev, [tagString]: items }));

                // If the list is empty, end the query
                if (items.length === 0) {
                    setDropdownAnchorEl(null);
                }
                // Otherwise, set the list
                else {
                    setDropdownList(items);
                }
            }
        };

        fetchItems();
    }, [cachedTags, dropdownAnchorEl, getTaggableItems, tagString]);
    const selectDropdownItem = useCallback((item: ListObjectType) => {
        // Tagged item is inserted as a link
        const asLink = `[@${getDisplay(item).title}](${window.location.origin}${getObjectUrl(item)})`;
        // Insert the link, replacing the tag string and the "@" symbol
        const { textArea } = getSelection(`markdown-input-${name}`);
        textArea.value = replaceText(
            textArea.value,
            asLink,
            textArea.selectionStart - tagString.length - 1,
            textArea.selectionEnd,
        );
        handleChange(textArea.value);
        // Close the dropdown
        setDropdownAnchorEl(null);
    }, [handleChange, name, tagString]);

    // Listen for text input changes
    useEffect(() => {
        // Map keyboard shortcuts to their respective functions
        const keyMappings = {
            "1": () => insertHeader(Headers.H1), // ALT + 1 - Insert header 1
            "2": () => insertHeader(Headers.H2), // ALT + 2 - Insert header 2
            "3": () => insertHeader(Headers.H3), // ALT + 3 - Insert header 3
            "4": () => insertBulletList(), // ALT + 4 - Bullet list
            "5": () => insertNumberList(), // ALT + 5 - Number list
            "6": () => insertCheckboxList(), // ALT + 6 - Checkbox list
            "7": () => togglePreview(), // ALT + 7 - Toggle preview
            "b": () => bold(), // CTRL + B - Bold
            "i": () => italic(), // CTRL + I - Italic
            "k": () => insertLink(), // CTRL + K - Insert link
            "z": () => undo(), // CTRL + Z - Undo
            "Z": () => redo(), // CTRL + SHIFT + Z = Redo
            "S": () => strikethrough(), // CTRL + SHIFT + S - Strikethrough
            "y": () => redo(), // CTRL + Y = Redo
        };
        // Handle key press events for textarea
        const handleTextareaKeyDown = (e: any) => {
            // Check if either alt or ctrl key is pressed
            if (e.altKey || e.ctrlKey) {
                const action = keyMappings[e.key]; // Find the function mapped to the key press
                // If a function is found, prevent default action and call the function
                if (action && !isButtonDebounced.current) {
                    console.log("textarea action triggered");
                    e.preventDefault();
                    startDebounce();
                    action();
                    return;
                }
            }
            // Handle tag dropdown. Triggered by "@" key press
            if (!dropdownAnchorEl && typeof getTaggableItems === "function" && e.key === "@") {
                setTagString("");
                setDropdownList([]);
                setDropdownAnchorEl(textAreaRef.current);
            } else if (dropdownAnchorEl) {
                // Normal characters (e.g. letters, numbers, emojis) are added to the tag string
                if (e.key.length === 1) {
                    setTagString(tagString + e.key);
                }
                // Backspace removes the last character from the tag string
                else if (e.key === "Backspace") {
                    setTagString(tagString.slice(0, -1));
                }
                // Escape ends the query
                else if (e.key === "Escape") {
                    setDropdownAnchorEl(null);
                }
                // Tab and arrow keys cycle through the dropdown items
                else if (e.key === "Tab" || e.key === "ArrowDown" || e.key === "ArrowUp") {
                    e.preventDefault();
                    let newIndex = dropdownTabIndex;
                    // Increment the index if tab or arrow down is pressed without the shift key
                    if ((e.key === "Tab" && !e.shiftKey) || e.key === "ArrowDown") {
                        newIndex = (newIndex + 1) % dropdownList.length;
                    }
                    // Decrement the index if shift+tab or arrow up is pressed
                    else if ((e.key === "Tab" && e.shiftKey) || e.key === "ArrowUp") {
                        newIndex = (newIndex - 1 + dropdownList.length) % dropdownList.length;
                    }
                    setDropdownTabIndex(newIndex);
                }
                // Enter selects the first item in the dropdown
                else if (e.key === "Enter") {
                    if (dropdownList.length > 0) {
                        e.preventDefault();
                        const selectedItem = dropdownTabIndex >= 0 && dropdownTabIndex < dropdownList.length ?
                            dropdownList[dropdownTabIndex] :
                            dropdownList[0];
                        selectDropdownItem(selectedItem);
                    }
                    setDropdownAnchorEl(null);
                    // Return so that the enter key press is not handled by the textarea
                    return;
                }
            }
            // On enter key press
            if (e.key === "Enter") {
                const { selectionStart, selectionEnd, value } = e.target;
                let [trimmedLine] = getLineAtIndex(value, selectionStart);
                trimmedLine = trimmedLine.trimStart();
                const isNumberList = /^\d+\.\s/.test(trimmedLine);
                const isCheckboxList = trimmedLine.startsWith("- [ ] ") || trimmedLine.startsWith("- [x] ");
                const isBulletDashList = trimmedLine.startsWith("- ");
                const isBulletStarList = trimmedLine.startsWith("* ");
                // If the current line is a list
                if (isNumberList || isCheckboxList || isBulletDashList || isBulletStarList) {
                    e.preventDefault();
                    const { textArea } = getSelection(`markdown-input-${name}`);
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
                    textArea.value = replaceText(value, textToInsert, selectionStart, selectionEnd);
                    handleChange(textArea.value);
                }
            }
        };
        // Handle key press events for preview
        const handleFullComponentKeyDown = (e: any) => {
            if (isButtonDebounced.current) return;
            // Only check for the toggle preview shortcut
            if (e.altKey && e.key === "6") {
                console.log("fullcomponent action triggered");
                e.preventDefault();
                startDebounce();
                togglePreview();
            }
            // If any letters or numbers are pressed (i.e. user is trying to type and possibly 
            // frustrated that nothing is happending), trigger preview button flash
            else if (/^[a-zA-Z0-9]$/.test(e.key)) {
                setFlashPreview(true);
            }
        };
        // Find textarea or full component
        const textarea = document.getElementById(`markdown-input-${name}`);
        const fullComponent = document.getElementById(`markdown-input-base-${name}`);
        // Add appropriate listeners
        if (textarea) textarea.addEventListener("keydown", handleTextareaKeyDown);
        else if (fullComponent) fullComponent.addEventListener("keydown", handleFullComponentKeyDown);
        // Return a cleanup function to remove the listeners on unmount
        return () => {
            textarea?.removeEventListener("keydown", handleTextareaKeyDown);
            fullComponent?.removeEventListener("keydown", handleFullComponentKeyDown);
        };
    }, [bold, dropdownAnchorEl, dropdownList, dropdownTabIndex, getTaggableItems, handleChange, insertBulletList, insertCheckboxList, insertHeader, insertLink, insertNumberList, italic, name, redo, selectDropdownItem, startDebounce, strikethrough, tagString, togglePreview, togglePreviewDebounce, undo]);

    // Resize textarea to fit content
    const LINE_HEIGHT_MULTIPLIER = 1.5;
    const textAreaRef = useRef<HTMLTextAreaElement>(null);
    useEffect(() => {
        if (!textAreaRef.current) return;
        const lines = (value.match(/\n/g)?.length || 0) + 1;
        const lineHeight = Math.round(typography.fontSize * LINE_HEIGHT_MULTIPLIER);
        const minRowsNum = minRows ? Number.parseInt(minRows + "") : 2;
        const maxRowsNum = maxRows ? Number.parseInt(maxRows + "") : lines;
        const linesShown = Math.max(minRowsNum, Math.min(lines, maxRowsNum));
        const padding = 34;
        textAreaRef.current.style.height = `${linesShown * lineHeight + padding}px`;
    }, [isPreviewOn, minRows, maxRows, typography, value]);

    return (
        <>
            {/* Assistant dialog for generating text */}
            <AssistantDialog {...assistantDialogProps} />
            {/* Dropdown for tagging items */}
            <TagItemDropdown
                anchorEl={dropdownAnchorEl}
                focusedIndex={dropdownTabIndex}
                handleClose={() => setDropdownAnchorEl(null)}
                handleFocusChange={setDropdownTabIndex}
                handleSelect={selectDropdownItem}
                items={dropdownList}
            />
            <Stack
                id={`markdown-input-base-${name}`}
                direction="column"
                spacing={0}
                onMouseDown={handleMouseDown}
                sx={{ ...(sxs?.root ?? {}) }}
            >
                {/* Bar above TextField, for inserting markdown and previewing */}
                <Box sx={{
                    display: "flex",
                    width: "100%",
                    padding: "0.5rem",
                    background: palette.primary.light,
                    color: palette.primary.contrastText,
                    borderRadius: "0.5rem 0.5rem 0 0",
                    ...(sxs?.bar ?? {}),
                }}>
                    {/* To the left is a stack for ai assistant, inserting titles, italics/bold, lists, and links */}
                    <Stack
                        direction="row"
                        spacing={{ xs: 0, sm: 0.5, md: 1 }}
                        sx={{ marginRight: "auto" }}
                    >
                        {/* AI assistant */}
                        {hasPremium && !disableAssistant && <Tooltip title={t("AIAssistant")} placement="top">
                            <IconButton
                                aria-describedby={`markdown-input-assistant-popover-${name}`}
                                disabled={disabled}
                                size="small"
                                onClick={openAssistantDialog}
                            >
                                <MagicIcon fill={palette.primary.contrastText} />
                            </IconButton>
                        </Tooltip>}
                        {/* Insert header selector */}
                        <Tooltip title={t("HeaderInsert")} placement="top">
                            <IconButton
                                aria-describedby={`markdown-input-header-popover-${name}`}
                                disabled={disabled}
                                size="small"
                                onClick={openHeaderSelect}
                            >
                                <HeaderIcon fill={palette.primary.contrastText} />
                            </IconButton>
                        </Tooltip>
                        <Popover
                            id={`markdown-input-header-popover-${name}`}
                            open={headerSelectOpen}
                            anchorEl={headerAnchorEl}
                            onClose={closeHeader}
                            anchorOrigin={{
                                vertical: "bottom",
                                horizontal: "center",
                            }}
                        >
                            {/* When opened, button row of 1-6, for each size */}
                            <Stack direction="row" spacing={0} sx={{
                                background: palette.primary.light,
                                color: palette.primary.contrastText,
                            }}>
                                <Tooltip title={`${t("Header1")} (${keyComboToString("Alt", "1")})`} placement="top">
                                    <IconButton
                                        onClick={() => insertHeader(Headers.H1)}
                                        sx={dropDownButtonProps}
                                    >
                                        <Header1Icon fill={palette.primary.contrastText} />
                                    </IconButton>
                                </Tooltip>
                                <Tooltip title={`${t("Header2")} (${keyComboToString("Alt", "2")})`} placement="top">
                                    <IconButton
                                        onClick={() => insertHeader(Headers.H2)}
                                        sx={dropDownButtonProps}
                                    >
                                        <Header2Icon fill={palette.primary.contrastText} />
                                    </IconButton>
                                </Tooltip>
                                <Tooltip title={`${t("Header3")} (${keyComboToString("Alt", "3")})`} placement="top">
                                    <IconButton
                                        onClick={() => insertHeader(Headers.H3)}
                                        sx={dropDownButtonProps}
                                    >
                                        <Header3Icon fill={palette.primary.contrastText} />
                                    </IconButton>
                                </Tooltip>
                            </Stack>
                        </Popover>
                        {/* Button for bold */}
                        <Tooltip title={`${t("Bold")} (${keyComboToString("Ctrl", "B")})`} placement="top">
                            <IconButton
                                disabled={disabled}
                                size="small"
                                onClick={bold}
                            >
                                <BoldIcon fill={palette.primary.contrastText} />
                            </IconButton>
                        </Tooltip>
                        {/* Button for italic */}
                        <Tooltip title={`${t("Italic")} (${keyComboToString("Ctrl", "I")})`} placement="top">
                            <IconButton
                                disabled={disabled}
                                size="small"
                                onClick={italic}
                            >
                                <ItalicIcon fill={palette.primary.contrastText} />
                            </IconButton>
                        </Tooltip>
                        {/* Button for strikethrough */}
                        <Tooltip title={`${t("Strikethrough")} (${keyComboToString("Ctrl", "Shift", "S")})`} placement="top">
                            <IconButton
                                disabled={disabled}
                                size="small"
                                onClick={strikethrough}
                            >
                                <StrikethroughIcon fill={palette.primary.contrastText} />
                            </IconButton>
                        </Tooltip>
                        {/* Insert bulleted or numbered list selector */}
                        <Tooltip title={t("ListInsert")} placement="top">
                            <IconButton
                                aria-describedby={`markdown-input-list-popover-${name}`}
                                disabled={disabled}
                                size="small"
                                onClick={openListSelect}
                            >
                                <ListIcon fill={palette.primary.contrastText} />
                            </IconButton>
                        </Tooltip>
                        <Popover
                            id={`markdown-input-list-popover-${name}`}
                            open={listSelectOpen}
                            anchorEl={listAnchorEl}
                            onClose={closeList}
                            anchorOrigin={{
                                vertical: "bottom",
                                horizontal: "center",
                            }}
                        >
                            <Stack direction="row" spacing={0} sx={{
                                background: palette.primary.light,
                                color: palette.primary.contrastText,
                            }}>
                                <Tooltip title={`${t("ListBulleted")} (${keyComboToString("Alt", "4")})`} placement="top">
                                    <IconButton
                                        onClick={insertBulletList}
                                        sx={dropDownButtonProps}
                                    >
                                        <ListBulletIcon fill={palette.primary.contrastText} />
                                    </IconButton>
                                </Tooltip>
                                <Tooltip title={`${t("ListNumbered")} (${keyComboToString("Alt", "5")})`} placement="top">
                                    <IconButton
                                        onClick={insertNumberList}
                                        sx={dropDownButtonProps}
                                    >
                                        <ListNumberIcon fill={palette.primary.contrastText} />
                                    </IconButton>
                                </Tooltip>
                                <Tooltip title={`${t("ListCheckbox")} (${keyComboToString("Alt", "6")})`} placement="top">
                                    <IconButton
                                        onClick={insertCheckboxList}
                                        sx={dropDownButtonProps}
                                    >
                                        <ListCheckIcon fill={palette.primary.contrastText} />
                                    </IconButton>
                                </Tooltip>
                            </Stack>
                        </Popover>
                        {/* Button for inserting link */}
                        <Tooltip title={`${t("LinkInsert")} (${keyComboToString("Ctrl", "K")})`} placement="top">
                            <IconButton
                                disabled={disabled}
                                size="small"
                                onClick={insertLink}
                            >
                                <LinkIcon fill={palette.primary.contrastText} />
                            </IconButton>
                        </Tooltip>
                    </Stack>
                    {/* To the right is buttons for undo, redo, and previewing the markdown */}
                    <Stack
                        direction="row"
                        spacing={{ xs: 0, sm: 0.5, md: 1 }}
                    >
                        {/* Undo */}
                        {(canUndo || canRedo) && <Tooltip title={canUndo ? `${t("Undo")} (${keyComboToString("Ctrl", "Z")})` : ""}>
                            <IconButton
                                id="undo-button"
                                disabled={!canUndo}
                                onClick={undo}
                                aria-label="Undo"
                                size="small"
                            >
                                <UndoIcon fill={palette.primary.contrastText} />
                            </IconButton>
                        </Tooltip>}
                        {/* Redo */}
                        {(canUndo || canRedo) && <Tooltip title={canRedo ? `${t("Redo")} (${keyComboToString("Ctrl", "Y")})` : ""}>
                            <IconButton
                                id="redo-button"
                                disabled={!canRedo}
                                onClick={redo}
                                aria-label="Redo"
                                size="small"
                            >
                                <RedoIcon fill={palette.primary.contrastText} />
                            </IconButton>
                        </Tooltip>}
                        {/* Preview */}
                        <Tooltip title={isPreviewOn ? `${t("PressToEdit")} (${keyComboToString("Alt", "7")})` : `${t("PressToPreview")} (${keyComboToString("Alt", "7")})`} placement="top">
                            <IconButton
                                size="small"
                                onClick={togglePreview}
                                sx={{
                                    backgroundColor: flashPreview ? palette.error.main : "transparent",
                                    transition: "backgroundColor 1s ease-in-out",
                                }}
                            >
                                {
                                    isPreviewOn ?
                                        <InvisibleIcon fill={flashPreview ? palette.error.contrastText : palette.primary.contrastText} /> :
                                        <VisibleIcon fill={flashPreview ? palette.error.contrastText : palette.primary.contrastText} />
                                }
                            </IconButton>
                        </Tooltip>
                    </Stack>
                </Box>
                {/* TextField for entering markdown, or markdown display if previewing */}
                {
                    isPreviewOn ?
                        (
                            <Box
                                id={`markdown-preview-${name}`}
                                sx={{
                                    border: `1px solid ${error ? "red" : "black"}`,
                                    borderRadius: "0 0 0.5rem 0.5rem",
                                    padding: "12px",
                                    wordBreak: "break-word",
                                    overflow: "auto",
                                    backgroundColor: palette.background.paper,
                                    color: palette.text.primary,
                                    ...linkColors(palette),
                                    ...sxs?.textArea,
                                }}>
                                <MarkdownDisplay
                                    content={internalValue}
                                    isEditable={!disabled}
                                    onChange={handleChange}
                                    zIndex={zIndex}
                                />
                            </Box>
                        ) :
                        (
                            <textarea
                                id={`markdown-input-${name}`}
                                ref={textAreaRef}
                                autoFocus={autoFocus}
                                disabled={disabled}
                                name={name}
                                placeholder={placeholder}
                                rows={minRows}
                                value={internalValue}
                                onBlur={onBlur}
                                onChange={(e) => { handleChange(e.target.value); }}
                                tabIndex={tabIndex}
                                style={{
                                    padding: "16.5px 14px",
                                    minWidth: "-webkit-fill-available",
                                    maxWidth: "-webkit-fill-available",
                                    outline: "none",
                                    resize: "none",
                                    borderColor: error ? "red" : palette.divider,
                                    borderRadius: "0 0 4px 4px",
                                    borderTop: "none",
                                    fontFamily: typography.fontFamily,
                                    fontSize: typography.fontSize + 2,
                                    lineHeight: `${Math.round(typography.fontSize * LINE_HEIGHT_MULTIPLIER)}px`,
                                    backgroundColor: palette.background.paper,
                                    color: palette.text.primary,
                                    ...sxs?.textArea,
                                } as const}
                            />
                        )
                }
                {/* Help text, characters remaining indicator, and action buttons */}
                {(helperText || maxChars || (Array.isArray(actionButtons) && actionButtons.length > 0)) && <Stack
                    direction="row"
                    justifyContent="space-between"
                    alignItems="center"
                    spacing={1}
                    sx={{
                        padding: 1,
                    }}
                >
                    {/* Helper text label */}
                    {
                        helperText &&
                        <Typography variant="body1" sx={{ color: "red", paddingTop: 1 }}>
                            {helperText}
                        </Typography>
                    }
                    <Stack direction="row" ml="auto" spacing={1}>
                        {/* Characters remaining indicator */}
                        {
                            !disabled && maxChars !== undefined &&
                            <CharLimitIndicator
                                chars={internalValue.length}
                                maxChars={maxChars}
                            />
                        }
                        {/* Action buttons */}
                        {
                            actionButtons?.map(({ disabled: buttonDisabled, Icon, onClick, tooltip }, index) => (
                                <Tooltip key={index} title={tooltip} placement="top">
                                    <ColorIconButton
                                        background={palette.secondary.main}
                                        disabled={disabled || buttonDisabled}
                                        size="small"
                                        onClick={onClick}
                                    >
                                        <Icon fill={palette.primary.contrastText} />
                                    </ColorIconButton>
                                </Tooltip>
                            ))
                        }
                    </Stack>
                </Stack>}
            </Stack>
        </>
    );
};
