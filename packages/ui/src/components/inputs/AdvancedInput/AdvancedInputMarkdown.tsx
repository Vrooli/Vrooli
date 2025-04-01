import { useCallback, useEffect, useRef } from "react";
import { AdvancedInputMarkdownProps, AdvancedInputStylingAction, MarkdownUtils, advancedInputTextareaClassName } from "./utils.js";

const LINE_HEIGHT = 1.5;
const PIXELS_PER_EM = 16; // Standard conversion: 1em = 16px

const containerStyle = {
    width: "100%",
    display: "flex",
    flexDirection: "column",
    overflow: "hidden",
    margin: 0,
    padding: 0,
} as const;
const textareaStyle = {
    background: "inherit",
    border: "none",
    color: "inherit",
    width: "100%",
    padding: "8px 12px",
    margin: 0,
    fontFamily: "inherit",
    fontSize: `${PIXELS_PER_EM}px`,
    lineHeight: `${LINE_HEIGHT * PIXELS_PER_EM}px`,
    resize: "none",
    outline: "none",
    boxSizing: "border-box",
    display: "block",
    overflowY: "auto",
    "::placeholder": {
        color: "rgba(128, 128, 128, 0.7)",
    },
} as const;

/** TextInput for entering markdown text */
export function AdvancedInputMarkdown({
    disabled = false,
    enterWillSubmit,
    maxRows,
    minRows,
    name,
    onBlur,
    onFocus,
    onChange,
    onSubmit,
    placeholder = "",
    redo,
    setHandleAction,
    tabIndex,
    toggleMarkdown,
    undo,
    value,
}: AdvancedInputMarkdownProps) {
    console.log("rendering RichInputMarkdown", maxRows, minRows, value.length);
    const textAreaRef = useRef<HTMLTextAreaElement>(null);

    const insertStyle = useCallback((style: AdvancedInputStylingAction | `${AdvancedInputStylingAction}`) => {
        if (disabled) return;
        // Find the current selection
        const { start, end, inputElement } = MarkdownUtils.getTextSelection(textAreaRef.current);
        if (!inputElement) return;
        // Apply the style
        const result = MarkdownUtils.insertStyle(style, inputElement.value, start, end);
        // Set the new value and selection
        inputElement.value = result.text;
        inputElement.selectionStart = result.start;
        inputElement.selectionEnd = result.end;
        // Update parent component
        onChange(result.text);
    }, [disabled, onChange]);

    const addTable = useCallback(function addTableCallback({ rows, cols }: { rows: number, cols: number }) {
        if (disabled) return;
        const { start, end, inputElement } = MarkdownUtils.getTextSelection(textAreaRef.current);
        if (!inputElement) return;
        const result = MarkdownUtils.insertTable(rows, cols, inputElement.value, start, end);
        // Set the new value and selection
        inputElement.value = result.text;
        inputElement.selectionStart = result.start;
        inputElement.selectionEnd = result.end;
        // Update parent component
        onChange(result.text);
    }, [disabled, onChange]);

    useEffect(() => {
        if (!setHandleAction) return;
        setHandleAction((action, data) => {
            // Anything that isn't a style action goes in the actionMap
            const actionMap = {
                "Redo": redo,
                "SetValue": () => {
                    if (typeof data !== "string") {
                        console.error("Invalid data for SetValue action", data);
                        return;
                    }
                    // Set value without triggering onChange
                    const { inputElement } = MarkdownUtils.getTextSelection(textAreaRef.current);
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
                insertStyle(action as AdvancedInputStylingAction);
            }
        });
    }, [addTable, insertStyle, redo, setHandleAction, undo]);

    // Listen for text input changes
    useEffect(() => {
        const textarea = textAreaRef.current;
        if (!textarea) return;

        // Handle key press events for textarea
        function handleTextareaKeyDown(e: any) {
            // On enter key press
            if (e.key === "Enter") {
                // Find the line the start of the selection is on
                const { start, end, inputElement } = MarkdownUtils.getTextSelection(textAreaRef.current);
                let [trimmedLine] = MarkdownUtils.getLineAtIndex(inputElement?.value, start);
                trimmedLine = trimmedLine.trimStart();
                const isNumberList = /^\d+\.\s/.test(trimmedLine);
                const isCheckboxList = trimmedLine.startsWith("- [ ] ") || trimmedLine.startsWith("- [x] ");
                const isBulletDashList = trimmedLine.startsWith("- ");
                const isBulletStarList = trimmedLine.startsWith("* ");
                // If the current line is a list
                if (isNumberList || isCheckboxList || isBulletDashList || isBulletStarList) {
                    e.preventDefault();
                    const { inputElement } = MarkdownUtils.getTextSelection(textAreaRef.current);
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
                    inputElement.value = MarkdownUtils.replaceText(value, textToInsert, start, end);
                    onChange(inputElement.value);
                }
            }
            // On "Esc" key press, don't propagate the event
            else if (e.key === "Escape") {
                e.stopPropagation();
            }
        }

        // Add event listeners for dynamic content changes
        textarea.addEventListener("keydown", handleTextareaKeyDown);
        return () => {
            textarea.removeEventListener("keydown", handleTextareaKeyDown);
        };
    }, [onChange, insertStyle, name, redo, toggleMarkdown, undo, value]);

    useEffect(function adjustHeightEffect() {
        const textarea = textAreaRef.current;
        if (!textarea) return;

        // Need arrow function to access textarea
        // eslint-disable-next-line func-style
        const adjustHeight = () => {
            // Calculate heights including padding
            const paddingHeight = 16; // 8px top + 8px bottom
            const lineHeight = LINE_HEIGHT * PIXELS_PER_EM; // 24px
            const minHeight = Math.max(lineHeight + paddingHeight, minRows * lineHeight + paddingHeight);
            const maxHeight = maxRows * lineHeight + paddingHeight;

            // Reset height to auto to get the correct scrollHeight
            textarea.style.height = "auto";
            // Get the scrollHeight which accounts for wrapped text
            const valueHeight = textarea.scrollHeight;

            const newHeight = Math.min(
                Math.max(valueHeight, minHeight),
                maxHeight,
            );
            textarea.style.height = `${newHeight}px`;
        };

        adjustHeight();

        // Add event listeners for dynamic content changes
        textarea.addEventListener("input", adjustHeight);
        window.addEventListener("resize", adjustHeight);

        return () => {
            textarea.removeEventListener("input", adjustHeight);
            window.removeEventListener("resize", adjustHeight);
        };
    }, [value, minRows, maxRows]);

    const handleChange = useCallback((event: React.ChangeEvent<HTMLTextAreaElement>) => {
        onChange(event.target.value);
    }, [onChange]);

    return (
        <div style={containerStyle}>
            <textarea
                className={advancedInputTextareaClassName}
                disabled={disabled}
                name={name}
                placeholder={placeholder}
                value={value}
                onBlur={onBlur}
                onFocus={onFocus}
                onChange={handleChange}
                tabIndex={tabIndex}
                ref={textAreaRef}
                spellCheck
                style={textareaStyle}
            />
        </div>
    );
}
