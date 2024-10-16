import { ListObject } from "@local/shared";
import { Box, styled, useTheme } from "@mui/material";
import { CSSProperties, useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import { DEFAULT_MIN_ROWS } from "utils/consts";
import { Headers } from "utils/display/stringTools";
import { LINE_HEIGHT_MULTIPLIER } from "../RichInput/RichInput";
import { RichInputTagDropdown, useTagDropdown } from "../RichInputTagDropdown/RichInputTagDropdown";
import { defaultActiveStates } from "../RichInputToolbar/RichInputToolbar";
import { RichInputAction, RichInputActiveStates, RichInputLexicalProps } from "../types";
import { $convertFromMarkdownString, registerMarkdownShortcuts } from "./builder";
import { CODE_BLOCK_COMMAND, FORMAT_TEXT_COMMAND, INSERT_CHECK_LIST_COMMAND, INSERT_ORDERED_LIST_COMMAND, INSERT_TABLE_COMMAND, INSERT_UNORDERED_LIST_COMMAND, KEY_ENTER_COMMAND, SELECTION_CHANGE_COMMAND, TOGGLE_LINK_COMMAND } from "./commands";
import { COMMAND_PRIORITY_CRITICAL } from "./consts";
import { LexicalComposerContext, useLexicalComposerContext } from "./context";
import { LexicalEditor, createEditor } from "./editor";
import { type ElementNode } from "./nodes/ElementNode";
import { type HeadingNode } from "./nodes/HeadingNode";
import { type LexicalNode } from "./nodes/LexicalNode";
import { type TextNode } from "./nodes/TextNode";
import { CheckListPlugin } from "./plugins/CheckListPlugin";
import { CodeBlockPlugin } from "./plugins/CodePlugin";
import { LinkPlugin } from "./plugins/LinkPlugin";
import { ListPlugin } from "./plugins/ListPlugin";
import { RichTextPlugin } from "./plugins/RichTextPlugin";
import { TablePlugin } from "./plugins/TablePlugin";
import { $setBlocksType, RangeSelection } from "./selection";
import "./theme.css";
import { ELEMENT_TRANSFORMERS } from "./transformers/elementTransformers";
import { TEXT_TRANSFORMERS, applyTextTransformers } from "./transformers/textFormatTransformers";
import { TEXT_MATCH_TRANSFORMERS } from "./transformers/textMatchTransformers";
import { LexicalTransformer } from "./types";
import { $createNode, $findMatchingParent, $getNearestNodeOfType, $getNodeByKey, $getRoot, $getSelection, $isAtNodeEnd, $isNode, $isRangeSelection, $isRootOrShadowRoot, getParent, getTopLevelElementOrThrow } from "./utils";

const HISTORY_MERGE_OPTIONS = { tag: "history-merge" };
const PADDING_HEIGHT_PX = 16.5;

/** Every supported block type (e.g. lists, headers, quote) */
const blockTypeToActionName: { [x: string]: RichInputAction | `${RichInputAction}` } = {
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
    id,
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
            id={id ?? "lexical-editor"}
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
    padding: `${PADDING_HEIGHT_PX}px 14px`,
    pointerEvents: "none",
    top: 0,
    left: 0,
}));

/** TextInput for entering WYSIWYG text */
export function RichInputLexicalComponents({
    enterWillSubmit,
    getTaggableItems,
    id,
    maxRows,
    minRows = DEFAULT_MIN_ROWS,
    onActiveStatesChange,
    onBlur,
    onFocus,
    onChange,
    onSubmit,
    openAssistantDialog,
    placeholder = "",
    redo,
    setHandleAction,
    tabIndex,
    undo,
    value,
    sxs,
}: RichInputLexicalProps) {
    const { palette, typography } = useTheme();
    const editor = useLexicalComposerContext();

    const valueRef = useRef(value);
    useEffect(function updateValueRefEffect() {
        valueRef.current = value;
    }, [value]);

    const tagData = useTagDropdown({ getTaggableItems });
    const selectDropdownItem = useCallback((item: ListObject) => {
        //TODO
    }, []);

    /** Store current text properties. Logic inspired by https://github.com/facebook/lexical/blob/9e83533d52fe934bd91aaa5baaf156f682577dcf/packages/lexical-playground/src/plugins/ToolbarPlugin/index.tsx#L484 */
    const [activeEditor, setActiveEditor] = useState(editor);
    const [activeStates, setActiveStates] = useState<Omit<RichInputActiveStates, "SetValue">>({ ...defaultActiveStates });
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
        }, 300); // Adjust delay as needed for performance

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
                "Assistant": () => {
                    console.log("in assistant action");
                    editor.update(() => {
                        console.log("in assistant action editor update");
                        const selection = $getSelection() as RangeSelection | null;
                        let selectedText = "";
                        if (selection) {
                            const anchorNode = $getNodeByKey(selection.anchor.key);
                            const focusNode = $getNodeByKey(selection.focus.key);

                            if (anchorNode && focusNode) {
                                // Assuming anchorNode comes before focusNode in the document
                                const nodes = anchorNode.getNodesBetween(focusNode);

                                let canApplyHeader = true; // Can only apply header style once per line
                                // Concatenate the text from these nodes
                                selectedText = nodes.map((node) => {
                                    // If normal or stylized text
                                    if (node.getType() === "Text") {
                                        const parentNode = node.__parent ? $getNodeByKey(node.__parent) : null;
                                        const formattedText = applyStyles(node.getTextContent(), (node as TextNode).__format, parentNode, canApplyHeader);
                                        if (parentNode?.getType() === "Heading") {
                                            canApplyHeader = false;
                                        }
                                        return formattedText;
                                    }
                                    // If a newline (this node type might be used in other cases, but we're not sure yet)
                                    else if (node.getType() === "Paragraph") {
                                        canApplyHeader = true;
                                        return "\n";
                                    }
                                    return "";
                                }).join("");
                            }
                        }
                        const fullText = $getRoot().getMarkdownContent();
                        openAssistantDialog(selectedText, fullText);
                    });
                },
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
                "Quote": () => { }, //TODO
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
    }, [editor, openAssistantDialog, redo, setHandleAction, toggleHeading, triggerEditorChange, undo]);

    const lineHeight = useMemo(() => Math.round(typography.fontSize * LINE_HEIGHT_MULTIPLIER), [typography.fontSize]);

    const outerBoxStyle = useMemo(function outerBoxStyleMemo() {
        return {
            position: "relative",
            display: "grid",
            padding: `${PADDING_HEIGHT_PX}px 14px`,
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
            border: `1px solid ${palette.divider}`,
            "&:hover": {
                border: `1px solid ${palette.background.textPrimary}`,
            },
            "&:focus-within": {
                border: `2px solid ${palette.primary.main}`,
            },
            ...sxs?.inputRoot,
        } as const;
    }, [lineHeight, maxRows, minRows, palette.background.textPrimary, palette.divider, palette.primary.main, palette.text.primary, sxs?.inputRoot, typography.fontFamily, typography.fontSize]);

    return (
        <Box
            id={id}
            component="div"
            onBlur={onBlur}
            onFocus={onFocus}
            sx={outerBoxStyle}>
            <RichTextPlugin
                contentEditable={<ContentEditable
                    id={`${id}-contenteditable`}
                    tabIndex={tabIndex}
                    style={{
                        outline: "none",
                        resize: "none",
                        overflow: "auto",
                        ...sxs?.textArea,
                    } as CSSProperties}
                />}
            />
            {(!editor || (typeof value === "string" && value.length === 0)) && <LoadingPlaceholderBox>
                {!editor ? "Loading..." : (placeholder ?? "Enter some text...")}
            </LoadingPlaceholderBox>}
            <CheckListPlugin />
            <LinkPlugin />
            <ListPlugin />
            <MarkdownShortcutPlugin transformers={ALL_TRANSFORMERS} />
            <RichInputTagDropdown {...tagData} selectDropdownItem={selectDropdownItem} />
            <TablePlugin />
            <CodeBlockPlugin />
        </Box>
    );
}

/** TextInput for entering WYSIWYG text */
export function RichInputLexical({
    disabled,
    value,
    ...props
}: RichInputLexicalProps) {

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

    return (
        <LexicalComposerContext.Provider value={editor}>
            <RichInputLexicalComponents value={value} {...props} />
        </LexicalComposerContext.Provider>
    );
}
