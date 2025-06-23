import Box from "@mui/material/Box";
import { styled } from "@mui/material/styles";
import { useTheme } from "@mui/material/styles";
import { type CSSProperties, forwardRef, useCallback, useEffect, useImperativeHandle, useLayoutEffect, useMemo, useRef, useState } from "react";
import { DEFAULT_MIN_ROWS } from "../../../../utils/consts.js";
const LINE_HEIGHT_MULTIPLIER = 1.5;
import { defaultActiveStates } from "../AdvancedInputToolbar.js";
import { type AdvancedInputAction, type AdvancedInputActiveStates, type AdvancedInputLexicalProps, Headers } from "../utils.js";
import { $convertFromMarkdownString, registerMarkdownShortcuts } from "./builder.js";
import { CODE_BLOCK_COMMAND, FORMAT_TEXT_COMMAND, INSERT_CHECK_LIST_COMMAND, INSERT_ORDERED_LIST_COMMAND, INSERT_TABLE_COMMAND, INSERT_UNORDERED_LIST_COMMAND, KEY_ENTER_COMMAND, SELECTION_CHANGE_COMMAND, TOGGLE_LINK_COMMAND } from "./commands.js";
import { COMMAND_PRIORITY_CRITICAL } from "./consts.js";
import { LexicalComposerContext, useLexicalComposerContext } from "./context.js";
import { type LexicalEditor, createEditor } from "./editor.js";
import { type ElementNode } from "./nodes/ElementNode.js";
import { type HeadingNode } from "./nodes/HeadingNode.js";
import { type LexicalNode } from "./nodes/LexicalNode.js";
import { type TextNode } from "./nodes/TextNode.js";
import { CheckListPlugin } from "./plugins/CheckListPlugin.js";
import { CodeBlockPlugin } from "./plugins/CodePlugin.js";
import { LinkPlugin } from "./plugins/LinkPlugin.js";
import { ListPlugin } from "./plugins/ListPlugin.js";
import { RichTextPlugin } from "./plugins/RichTextPlugin.js";
import { TablePlugin } from "./plugins/TablePlugin.js";
import { $setBlocksType, type RangeSelection } from "./selection.js";
import { ELEMENT_TRANSFORMERS } from "./transformers/elementTransformers.js";
import { TEXT_TRANSFORMERS, applyTextTransformers } from "./transformers/textFormatTransformers.js";
import { TEXT_MATCH_TRANSFORMERS } from "./transformers/textMatchTransformers.js";
import { type LexicalTransformer } from "./types.js";
import { $createNode, $findMatchingParent, $getNearestNodeOfType, $getSelection, $isAtNodeEnd, $isNode, $isRangeSelection, $isRootOrShadowRoot, getParent, getTopLevelElementOrThrow } from "./utils.js";

const HISTORY_MERGE_OPTIONS = { tag: "history-merge" };
const PADDING_HEIGHT_PX = 16.5;
const DEBOUNCE_DELAY_MS = 300;

/** Every supported block type (e.g. lists, headers, quote) */
const blockTypeToActionName: { [x: string]: AdvancedInputAction | `${AdvancedInputAction}` } = {
    bullet: "ListBullet",
    check: "ListCheckbox",
    code: "Code",
    [Headers.h1]: "Header1",
    [Headers.h2]: "Header2",
    [Headers.h3]: "Header3",
    [Headers.h4]: "Header4",
    [Headers.h5]: "Header5",
    [Headers.h6]: "Header6",
    number: "ListNumber",
    quote: "Quote",
};

/**
 * Retrieves the most relevant node from a given range selection within the Lexical editor.
 *
 * This function determines the most appropriate node to consider as "selected" based on
 * the current range selection's anchor and focus points. It is particularly useful in
 * scenarios where operations need to be performed on a single node, and there's a need
 * to decide which node (anchor or focus) should be the target based on the selection direction
 * and position.
 *
 * - If the anchor and focus of the selection are within the same node, that node is returned.
 * - If the selection is backward (focus is before anchor), the function returns:
 *   - The anchor node if the focus is at the end of its node, implying the selection started
 *     from the end of the focus node and extended into the anchor node.
 *   - The focus node otherwise, implying the main part of the selection is within the focus node.
 * - If the selection is forward (anchor is before focus), the function returns:
 *   - The anchor node if the anchor is at the end of its node, implying the selection started
 *     at the end of the anchor node and extended into the focus node.
 *   - The focus node otherwise, implying the main part of the selection is within the focus node.
 *
 * Note: The function assumes that the selection is a `RangeSelection` and that both the anchor
 * and focus are within text nodes or element nodes. It does not handle other types of selections
 * or nodes (e.g., root or line break nodes).
 *
 * @param {RangeSelection} selection - The current range selection from which the node is to be determined.
 * @returns {TextNode | ElementNode} - The selected node based on the logic described above, which will
 * either be a `TextNode` or an `ElementNode`.
 */
export function getSelectedNode(
    selection: RangeSelection,
): TextNode | ElementNode {
    const anchor = selection.anchor;
    const focus = selection.focus;
    const anchorNode = selection.anchor.getNode();
    const focusNode = selection.focus.getNode();
    if (anchorNode === focusNode) {
        return anchorNode;
    }
    const isBackward = selection.isBackward();
    if (isBackward) {
        return $isAtNodeEnd(focus) ? anchorNode : focusNode;
    } else {
        return $isAtNodeEnd(anchor) ? anchorNode : focusNode;
    }
}

const ALL_TRANSFORMERS: LexicalTransformer[] = [
    ...ELEMENT_TRANSFORMERS,
    ...TEXT_TRANSFORMERS,
    ...TEXT_MATCH_TRANSFORMERS,
];

function applyStyles(
    text: string,
    format: number,
    parentNode: LexicalNode | null,
    canApplyHeader = true,
) {
    text = applyTextTransformers(text, format);
    if (!parentNode) return text;

    // If the parent node is a heading, add the appropriate number of "#" characters
    if (parentNode?.getType() === "Heading" && canApplyHeader) {
        text = "#".repeat((parentNode as HeadingNode).__size) + " " + text;
    }

    return text;
}

export function MarkdownShortcutPlugin({
    transformers,
}: Readonly<{
    transformers: LexicalTransformer[];
}>): null {
    const editor = useLexicalComposerContext();

    useEffect(function registerMarkdownShortcutsEffect() {
        if (!editor) return;
        return registerMarkdownShortcuts(editor, transformers);
    }, [editor, transformers]);

    return null;
}

export type ContentEditableProps = {
    ariaActiveDescendant?: React.AriaAttributes["aria-activedescendant"];
    ariaAutoComplete?: React.AriaAttributes["aria-autocomplete"];
    ariaControls?: React.AriaAttributes["aria-controls"];
    ariaDescribedBy?: React.AriaAttributes["aria-describedby"];
    ariaExpanded?: React.AriaAttributes["aria-expanded"];
    ariaLabel?: React.AriaAttributes["aria-label"];
    ariaLabelledBy?: React.AriaAttributes["aria-labelledby"];
    ariaMultiline?: React.AriaAttributes["aria-multiline"];
    ariaOwns?: React.AriaAttributes["aria-owns"];
    ariaRequired?: React.AriaAttributes["aria-required"];
    autoCapitalize?: HTMLDivElement["autocapitalize"];
    "data-testid"?: string | null | undefined;
} & React.AllHTMLAttributes<HTMLDivElement>;

export function ContentEditable({
    ariaActiveDescendant,
    ariaAutoComplete,
    ariaControls,
    ariaDescribedBy,
    ariaExpanded,
    ariaLabel,
    ariaLabelledBy,
    ariaMultiline,
    ariaOwns,
    ariaRequired,
    autoCapitalize,
    className,
    role = "textbox",
    spellCheck = true,
    style,
    tabIndex,
    "data-testid": testid,
    ...rest
}: ContentEditableProps): JSX.Element {
    const editor = useLexicalComposerContext();
    const [isEditable, setEditable] = useState(false);

    // Set the DOM root element for the editor, which we can attach 
    // the editor's node tree to. Without this, the editor won't 
    // display anything.
    const ref = useCallback(
        (rootElement: null | HTMLElement) => {
            // defaultView is required for a root element.
            // In multi-window setups, the defaultView may not exist at certain points.
            if (
                editor &&
                rootElement &&
                rootElement.ownerDocument &&
                rootElement.ownerDocument.defaultView
            ) {
                editor.setRootElement(rootElement);
            }
        },
        [editor],
    );

    useLayoutEffect(() => {
        setEditable(editor?.isEditable() ?? false);
        return editor?.registerListener("editable", (currentIsEditable) => {
            setEditable(currentIsEditable);
        });
    }, [editor]);

    return (
        // eslint-disable-next-line jsx-a11y/aria-activedescendant-has-tabindex
        <div
            {...rest}
            aria-activedescendant={(!isEditable || tabIndex === undefined) ? undefined : ariaActiveDescendant}
            aria-autocomplete={!isEditable ? "none" : ariaAutoComplete}
            aria-controls={!isEditable ? undefined : ariaControls}
            aria-describedby={ariaDescribedBy}
            aria-expanded={
                !isEditable
                    ? undefined
                    : role === "combobox"
                        ? !!ariaExpanded
                        : undefined
            }
            aria-label={ariaLabel}
            aria-labelledby={ariaLabelledBy}
            aria-multiline={ariaMultiline}
            aria-owns={!isEditable ? undefined : ariaOwns}
            aria-readonly={!isEditable ? true : undefined}
            aria-required={ariaRequired}
            autoCapitalize={autoCapitalize}
            className={className}
            contentEditable={isEditable}
            data-testid={testid}
            ref={ref}
            role={role}
            spellCheck={spellCheck}
            style={style}
            tabIndex={tabIndex}
        />
    );
}

const LoadingPlaceholderBox = styled(Box)(({ theme }) => ({
    color: theme.palette.background.textSecondary,
    position: "absolute",
    padding: 0,
    pointerEvents: "none",
    top: 0,
    left: 0,
    fontSize: theme.typography.caption.fontSize,
}));

/** TextInput for entering WYSIWYG text */
export function AdvancedInputLexicalComponents({
    enterWillSubmit,
    maxRows,
    minRows = DEFAULT_MIN_ROWS,
    onActiveStatesChange,
    onBlur,
    onFocus,
    onChange,
    onSubmit,
    placeholder = "",
    redo,
    setHandleAction,
    tabIndex,
    undo,
    value,
    mergedFeatures,
}: AdvancedInputLexicalProps) {
    const { palette, typography } = useTheme();
    const editor = useLexicalComposerContext();

    const valueRef = useRef(value);
    useEffect(function updateValueRefEffect() {
        valueRef.current = value;
    }, [value]);

    /** Store current text properties. Logic inspired by https://github.com/facebook/lexical/blob/9e83533d52fe934bd91aaa5baaf156f682577dcf/packages/lexical-playground/src/plugins/ToolbarPlugin/index.tsx#L484 */
    const [activeEditor, setActiveEditor] = useState(editor);
    const [activeStates, setActiveStates] = useState<Omit<AdvancedInputActiveStates, "SetValue">>({ ...defaultActiveStates });
    const $updateToolbar = useCallback(() => {
        if (!activeEditor) return;
        const updatedStates = { ...defaultActiveStates };
        const selection = $getSelection();
        if ($isRangeSelection(selection)) {
            const anchorNode = selection.anchor.getNode();
            let element =
                anchorNode.__key === "root"
                    ? anchorNode
                    : $findMatchingParent(anchorNode, (e) => {
                        const parent = getParent(e);
                        return parent !== null && $isRootOrShadowRoot(parent);
                    });

            if (element === null) {
                element = getTopLevelElementOrThrow(anchorNode);
            }

            const elementKey = element.__key;
            const elementDOM = activeEditor.getElementByKey(elementKey);

            // Find text formats
            updatedStates.Bold = selection.hasFormat("BOLD");
            updatedStates.Code = selection.hasFormat("CODE_INLINE");
            updatedStates.Italic = selection.hasFormat("ITALIC");
            updatedStates.Underline = selection.hasFormat("UNDERLINE_LINES") || selection.hasFormat("UNDERLINE_TAGS");
            updatedStates.Spoiler = selection.hasFormat("SPOILER_LINES") || selection.hasFormat("SPOILER_TAGS");
            updatedStates.Strikethrough = selection.hasFormat("STRIKETHROUGH");
            // Check if link
            const node = getSelectedNode(selection);
            const parent = getParent(node);
            if ($isNode("Link", parent) || $isNode("Link", node)) {
                updatedStates.Link = true;
            }
            // Check if table
            const tableNode = $findMatchingParent(node, (node) => $isNode("Table", node));
            if ($isNode("Table", tableNode)) {
                updatedStates.Table = true;
            }

            if (elementDOM !== null) {
                if ($isNode("List", element)) {
                    const parentList = $getNearestNodeOfType("List", anchorNode);
                    const type = parentList
                        ? parentList.getListType()
                        : element.getListType();
                    console.log("got block type 1", type);
                    if (type in blockTypeToActionName) {
                        updatedStates[blockTypeToActionName[type as keyof typeof blockTypeToActionName]] = true;
                    }
                } else {
                    const type = $isNode("Heading", element)
                        ? element.getTag()
                        : element.getType();
                    if (type in blockTypeToActionName) {
                        console.log("got block type 2", type);
                        updatedStates[blockTypeToActionName[type as keyof typeof blockTypeToActionName]] = true;
                    }
                }
            }
            // TODO need to find  headers, code and quote blocks, and lists
            setActiveStates({ ...updatedStates });
            onActiveStatesChange({ ...updatedStates });
        }
    }, [activeEditor, onActiveStatesChange]);

    useEffect(function registerSelectionChangeEffect() {
        if (!editor) return;
        return editor.registerCommand(
            SELECTION_CHANGE_COMMAND,
            function selectionChangeListener(_, newEditor) {
                $updateToolbar();
                setActiveEditor(newEditor);
                return false;
            },
            COMMAND_PRIORITY_CRITICAL,
        );
    }, [editor, $updateToolbar]);

    useEffect(function registerKeyEnterEffect() {
        if (!editor) return;
        return editor.registerCommand(
            KEY_ENTER_COMMAND,
            function enterKeyListener(event, newEditor) {
                console.log("in enterKeyListener", event);
                // Check if any other modifier keys are pressed
                if (!event || event.shiftKey || event.ctrlKey || event.altKey || event.metaKey) {
                    return false;
                }
                if (enterWillSubmit) {
                    console.log("enter will submit");
                    event.preventDefault();
                    onSubmit?.(valueRef.current);
                    // Clear the editor
                    newEditor.update(() => { $convertFromMarkdownString("", []); }, HISTORY_MERGE_OPTIONS);
                    return true;
                }
                return false;
            },
            COMMAND_PRIORITY_CRITICAL, // Critical so we can cancel the default behavior
        );
    }, [editor, enterWillSubmit, onSubmit]);

    const triggerEditorChange = useCallback(() => {
        if (!editor) return;
        editor.update(() => {
            $convertFromMarkdownString(value, ALL_TRANSFORMERS);
        }, HISTORY_MERGE_OPTIONS);
    }, [editor, value]);

    const lastValueFromEditorRef = useRef(value);
    useLayoutEffect(() => {
        if (!editor) return;
        return editor.registerListener("textcontent", function handleTextContentChanged(updatedMarkdown) {
            lastValueFromEditorRef.current = updatedMarkdown;
            onChange(updatedMarkdown);
        });
    }, [editor, onChange]);
    // NOTE: If there are performance issues, this could be the cause. We 
    // need this to update the editor when the value prop is changed by the parent. 
    // There might be a more efficient way to do this.
    useEffect(() => {
        if (!editor || value === lastValueFromEditorRef.current) return;

        // Check if the editor is focused, to prevent updates while typing
        const rootElement = editor.getRootElement();
        if (rootElement && rootElement === document.activeElement) {
            // The editor is focused, skip the update to avoid key loss
            return;
        }

        // Debounced update to reduce frequent updates
        const timeoutId = setTimeout(() => {
            editor.update(() => {
                $convertFromMarkdownString(value, ALL_TRANSFORMERS);
            }, HISTORY_MERGE_OPTIONS);
            lastValueFromEditorRef.current = value; // Update the ref to match the current value
        }, DEBOUNCE_DELAY_MS);

        return () => clearTimeout(timeoutId); // Cleanup timeout
    }, [value, editor]);

    // Toolbar actions
    const toggleHeading = useCallback((headingSize: Headers) => {
        editor?.update(() => {
            const selection = $getSelection();
            if (!selection) return;
            const isAlreadyApplied = activeStates[blockTypeToActionName[headingSize]] === true;
            $setBlocksType(selection, () => isAlreadyApplied ? $createNode("Paragraph", {}) : $createNode("Heading", { tag: headingSize }));
        });
    }, [activeStates, editor]);

    useEffect(function setHandleActionEffect() {
        if (!setHandleAction || !editor) return;
        setHandleAction((action, data) => {
            const actionMap = {
                "Bold": () => editor.dispatchCommand(FORMAT_TEXT_COMMAND, "BOLD"),
                "Code": () => editor.dispatchCommand(CODE_BLOCK_COMMAND, (void 0)),
                "Header1": () => toggleHeading(Headers.h1),
                "Header2": () => toggleHeading(Headers.h2),
                "Header3": () => toggleHeading(Headers.h3),
                "Header4": () => toggleHeading(Headers.h4),
                "Header5": () => toggleHeading(Headers.h5),
                "Header6": () => toggleHeading(Headers.h6),
                "Italic": () => editor.dispatchCommand(FORMAT_TEXT_COMMAND, "ITALIC"),
                "Link": () => editor.dispatchCommand(TOGGLE_LINK_COMMAND, "https://"), //TODO not working
                "ListBullet": () => editor.dispatchCommand(INSERT_UNORDERED_LIST_COMMAND, (void 0)),
                "ListCheckbox": () => editor.dispatchCommand(INSERT_CHECK_LIST_COMMAND, (void 0)), // TODO not working
                "ListNumber": () => editor.dispatchCommand(INSERT_ORDERED_LIST_COMMAND, (void 0)),
                "Quote": () => editor.dispatchCommand(FORMAT_TEXT_COMMAND, "QUOTE"), // Implement quote formatting
                "Redo": () => {
                    redo();
                    triggerEditorChange();
                },
                "SetValue": () => {
                    if (typeof data !== "string") {
                        console.error("Invalid data type for SetValue action", data);
                        return;
                    }
                    // set value without triggering onChange
                    editor.update(() => {
                        console.log("calling convertfrommarkdownstring in sethandleaction");
                        $convertFromMarkdownString(data, ALL_TRANSFORMERS);
                    }, HISTORY_MERGE_OPTIONS);
                },
                "Spoiler": () => editor.dispatchCommand(FORMAT_TEXT_COMMAND, "SPOILER_TAGS"),
                "Strikethrough": () => editor.dispatchCommand(FORMAT_TEXT_COMMAND, "STRIKETHROUGH"),
                "Table": () => editor.dispatchCommand(INSERT_TABLE_COMMAND, data as { rows: number, columns: number }),
                "Underline": () => editor.dispatchCommand(FORMAT_TEXT_COMMAND, "UNDERLINE_TAGS"),
                "Undo": () => {
                    undo();
                    triggerEditorChange();
                },
            };
            const actionFunction = actionMap[action];
            console.log("handling action", actionFunction, action);
            if (actionFunction) actionFunction();
        });
    }, [editor, redo, setHandleAction, toggleHeading, triggerEditorChange, undo]);

    const lineHeight = useMemo(() => Math.round(typography.fontSize * LINE_HEIGHT_MULTIPLIER), [typography.fontSize]);

    const outerBoxStyle = useMemo(function outerBoxStyleMemo() {
        return {
            position: "relative",
            display: "grid",
            padding: 0,
            minWidth: "-webkit-fill-available",
            maxWidth: "-webkit-fill-available",
            borderRadius: "0 0 4px 4px",
            borderTop: "none",
            fontFamily: typography.fontFamily,
            fontSize: typography.fontSize + 2,
            lineHeight: `${lineHeight}px`,
            minHeight: lineHeight * minRows + PADDING_HEIGHT_PX,
            maxHeight: maxRows ? (lineHeight * (maxRows ?? DEFAULT_MIN_ROWS) + PADDING_HEIGHT_PX) : "none",
            overflow: "auto",
            backgroundColor: "transparent",
            color: palette.text.primary,
            border: "none",
            "&:hover": {
                border: "none",
            },
            "&:focus-within": {
                border: "none",
                outline: "none",
            },
            "& .RichInput__textCode": {
                backgroundColor: palette.grey[100],
                padding: "1px 0.25rem",
                fontFamily: "Menlo, Consolas, Monaco, monospace",
                fontSize: "94%",
            },
            "& .RichInput_highlight": {
                backgroundColor: "rgba(255, 212, 0, 0.14)",
                borderBottom: "2px solid rgba(255, 212, 0, 0.3)",
                paddingBottom: "2px",
            },
            "& .spoiler": {
                cursor: "pointer",
                padding: "0.25rem",
                transition: "color 0.4s ease-in-out, background 0.4s ease-in-out",
                "&.hidden": {
                    color: "black",
                    background: "black",
                    filter: "blur(2px)",
                    "&:hover": {
                        filter: "blur(1px) brightness(1.1)",
                    },
                },
                "&.revealed": {
                    color: "inherit",
                    background: "rgba(0, 0, 0, 0.3)",
                    filter: "none",
                },
            },
            "& .RichInput__ltr": {
                textAlign: "left",
            },
            "& .RichInput__rtl": {
                textAlign: "right",
            },
            "& .RichInput__paragraph": {
                margin: 0,
                position: "relative",
            },
            "& .RichInput__quote": {
                margin: "0 0 10px 20px",
                fontSize: 15,
                color: palette.text.secondary,
                borderLeft: `4px solid ${palette.divider}`,
                paddingLeft: 16,
            },
            "& .RichInput__indent": {
                "--lexical-indent-base-value": "40px",
            },
            "& .RichInput__textStrikethrough": {
                textDecoration: "line-through",
            },
            "& .RichInput__textUnderlineStrikethrough": {
                textDecoration: "underline line-through",
            },
            "& .RichInput__textSubscript": {
                fontSize: "0.8em",
                verticalAlign: "sub !important",
            },
            "& .RichInput__textSuperscript": {
                fontSize: "0.8em",
                verticalAlign: "super",
            },
            "& .RichInput__hashtag": {
                backgroundColor: "rgba(88, 144, 255, 0.15)",
                borderBottom: "1px solid rgba(88, 144, 255, 0.3)",
            },
            "& .RichInput__table": {
                borderCollapse: "collapse",
                borderSpacing: 0,
                maxWidth: "100%",
                overflowY: "scroll",
                tableLayout: "fixed",
                width: "calc(100% - 25px)",
                margin: "30px 0",
                "&.selected": {
                    outline: `2px solid ${palette.primary.main}`,
                },
            },
            "& .RichInput__tableCell": {
                border: `1px solid ${palette.divider}`,
                minWidth: 75,
                verticalAlign: "top",
                textAlign: "start",
                padding: "6px 8px",
                position: "relative",
                cursor: "default",
                outline: "none",
            },
            "& .RichInput__tableCellResizer": {
                position: "absolute",
                right: -4,
                height: "100%",
                width: 8,
                cursor: "ew-resize",
                zIndex: 10,
                top: 0,
            },
            "& .RichInput__tableCellHeader": {
                backgroundColor: palette.grey[100],
            },
            "& .RichInput__tableCellSelected": {
                backgroundColor: palette.action.selected,
            },
            "& .RichInput__tableCellEditing": {
                boxShadow: "0 0 5px rgba(0, 0, 0, 0.4)",
                borderRadius: "3px",
            },
            "& .RichInput__tableAddColumns, & .RichInput__tableAddRows": {
                position: "absolute",
                backgroundColor: palette.grey[200],
                border: 0,
                cursor: "pointer",
                "&:hover": {
                    backgroundColor: palette.action.hover,
                },
                "&.columns": {
                    top: 0,
                    width: 20,
                    height: "100%",
                    right: 0,
                },
                "&.rows": {
                    bottom: -25,
                    width: "calc(100% - 25px)",
                    height: 20,
                    left: 0,
                },
            },
            "& .RichInput__ol1, & .RichInput__ol2, & .RichInput__ol3, & .RichInput__ol4, & .RichInput__ol5": {
                margin: "0 0 0 16px",
                padding: 0,
                listStyleType: "decimal",
            },
            "& .RichInput__ul": {
                margin: "0 0 0 16px",
                padding: 0,
                listStyleType: "disc",
            },
            "& .RichInput__listItem": {
                margin: "0 32px",
            },
            "& .RichInput__listItemChecked, & .RichInput__listItemUnchecked": {
                listStyleType: "none",
                padding: "0 0 0 24px",
                position: "relative",
                "&:before": {
                    content: "\"\"",
                    width: 16,
                    height: 16,
                    position: "absolute",
                    left: 0,
                    top: "50%",
                    transform: "translateY(-50%)",
                    border: `1px solid ${palette.text.primary}`,
                    borderRadius: 3,
                },
            },
            "& .RichInput__listItemChecked": {
                "&:before": {
                    backgroundColor: palette.primary.main,
                    borderColor: palette.primary.main,
                },
                "&:after": {
                    content: "\"\"",
                    position: "absolute",
                    left: 3,
                    top: "50%",
                    transform: "translateY(-70%) rotate(-45deg)",
                    width: 10,
                    height: 5,
                    borderLeft: `2px solid ${palette.common.white}`,
                    borderBottom: `2px solid ${palette.common.white}`,
                },
            },
            "& .RichInput__tokenComment": {
                color: palette.text.disabled,
            },
            "& .RichInput__tokenPunctuation": {
                color: palette.text.primary,
            },
            "& .RichInput__tokenProperty": {
                color: palette.primary.main,
            },
            "& .RichInput__tokenSelector": {
                color: palette.secondary.main,
            },
            "& .RichInput__tokenOperator": {
                color: palette.error.main,
            },
            "& .RichInput__tokenAttr": {
                color: palette.warning.main,
            },
            "& .RichInput__tokenVariable": {
                color: palette.info.main,
            },
            "& .RichInput__tokenFunction": {
                color: palette.success.main,
            },
            "& .RichInput__mark": {
                backgroundColor: palette.warning.main,
                "&.overlap": {
                    backgroundColor: palette.warning.light,
                },
                "&.selected": {
                    backgroundColor: palette.primary.main,
                    "&.overlap": {
                        backgroundColor: palette.primary.light,
                    },
                },
            },
            "& .RichInput__embedBlock": {
                position: "relative",
                "&.focused": {
                    outline: `2px solid ${palette.primary.main}`,
                },
            },
        } as const;
    }, [lineHeight, maxRows, minRows, palette, typography.fontFamily, typography.fontSize]);

    return (
        <Box
            component="div"
            onBlur={onBlur}
            onFocus={onFocus}
            sx={outerBoxStyle}>
            <RichTextPlugin
                contentEditable={<ContentEditable
                    data-testid="lexical-editor"
                    tabIndex={tabIndex}
                    style={{
                        outline: "none",
                        resize: "none",
                        overflow: "auto",
                        border: "none",
                    } as CSSProperties}
                    spellCheck={mergedFeatures?.allowSpellcheck !== false}
                />}
            />
            {(!editor || (typeof value === "string" && value.length === 0)) && <LoadingPlaceholderBox>
                {!editor ? "Loading..." : (placeholder ?? "Enter some text...")}
            </LoadingPlaceholderBox>}
            <CheckListPlugin />
            <LinkPlugin />
            <ListPlugin />
            <MarkdownShortcutPlugin transformers={ALL_TRANSFORMERS} />
            <TablePlugin />
            <CodeBlockPlugin />
        </Box>
    );
}

/** TextInput for entering WYSIWYG text */
export const AdvancedInputLexical = forwardRef<
    { focus: (options?: FocusOptions) => void },
    Omit<AdvancedInputLexicalProps, "ref">
>(function AdvancedInputLexical({
    disabled,
    value,
    mergedFeatures,
    ...props
}, ref) {
    // Set up Lexical editor
    const [editor, setEditor] = useState<LexicalEditor | null>(null);

    useEffect(function initLexicalEditorEffect() {
        async function initializeAsync() {
            // Asynchronously create the editor instance
            const newEditor = await createEditor({ namespace: "RichInputEditor" });

            // Initialize editor with current value
            newEditor.update(() => { $convertFromMarkdownString(value, ALL_TRANSFORMERS); }, HISTORY_MERGE_OPTIONS);

            setEditor(newEditor);
        }

        initializeAsync();
        // Purposely only run once
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    useLayoutEffect(() => {
        if (!editor) return;
        const isEditable = typeof disabled === "boolean" ? !disabled : true;
        editor.setEditable(isEditable !== undefined ? isEditable : true);
    }, [disabled, editor]);

    // Connect the ref to focus the editor
    useImperativeHandle(ref, () => ({
        focus: (options?: FocusOptions) => {
            if (editor) {
                editor.focus();
            }
        },
    }), [editor]);

    return (
        <LexicalComposerContext.Provider value={editor}>
            <AdvancedInputLexicalComponents value={value} mergedFeatures={mergedFeatures} {...props} />
        </LexicalComposerContext.Provider>
    );
});

// Add display name for debugging and dev tools
AdvancedInputLexical.displayName = "AdvancedInputLexical";
