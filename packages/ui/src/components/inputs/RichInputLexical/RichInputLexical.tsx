import { Box, useTheme } from "@mui/material";
import "highlight.js/styles/monokai-sublime.css";
import { CSSProperties, useCallback, useEffect, useLayoutEffect, useMemo, useState } from "react";
import { ListObject } from "utils/display/listTools";
import { LINE_HEIGHT_MULTIPLIER } from "../RichInput/RichInput";
import { RichInputTagDropdown, useTagDropdown } from "../RichInputTagDropdown/RichInputTagDropdown";
import { defaultActiveStates } from "../RichInputToolbar/RichInputToolbar";
import { RichInputAction, RichInputActiveStates, RichInputLexicalProps } from "../types";
import { $convertFromMarkdownString, $convertToMarkdownString, registerMarkdownShortcuts } from "./builder";
import { CODE_BLOCK_COMMAND, FORMAT_TEXT_COMMAND, INSERT_CHECK_LIST_COMMAND, INSERT_ORDERED_LIST_COMMAND, INSERT_TABLE_COMMAND, INSERT_UNORDERED_LIST_COMMAND, SELECTION_CHANGE_COMMAND, TOGGLE_LINK_COMMAND } from "./commands";
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
import { SpoilerPlugin } from "./plugins/SpoilerPlugin";
import { TablePlugin } from "./plugins/TablePlugin";
import { RangeSelection } from "./selection";
import "./theme.css";
import { ELEMENT_TRANSFORMERS } from "./transformers/elementTransformers";
import { TEXT_TRANSFORMERS, applyTextTransformers } from "./transformers/textFormatTransformers";
import { TEXT_MATCH_TRANSFORMERS } from "./transformers/textMatchTransformers";
import { HeadingTagType, LexicalTransformer } from "./types";
import { $findMatchingParent, $getNearestNodeOfType, $getNodeByKey, $getRoot, $getSelection, $isAtNodeEnd, $isNode, $isRangeSelection, $isRootOrShadowRoot, getParent, getTopLevelElementOrThrow } from "./utils";

const HISTORY_MERGE_OPTIONS = { tag: "history-merge" };

/** Every supported block type (e.g. lists, headers, quote) */
const blockTypeToActionName: { [x: string]: RichInputAction | `${RichInputAction}` } = {
    bullet: "ListBullet",
    check: "ListCheckbox",
    code: "Code",
    h1: "Header1",
    h2: "Header2",
    h3: "Header3",
    h4: "Header4",
    h5: "Header5",
    h6: "Header6",
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

const applyStyles = (
    text: string,
    format: number,
    parentNode: LexicalNode | null,
    canApplyHeader = true,
) => {
    text = applyTextTransformers(text, format);
    if (!parentNode) return text;

    // If the parent node is a heading, add the appropriate number of "#" characters
    if (parentNode?.getType() === "Heading" && canApplyHeader) {
        text = "#".repeat((parentNode as HeadingNode).__size) + " " + text;
    }

    return text;
};

export const MarkdownShortcutPlugin = ({
    transformers,
}: Readonly<{
    transformers: LexicalTransformer[];
}>): null => {
    const editor = useLexicalComposerContext();

    useEffect(() => {
        if (!editor) return;
        return registerMarkdownShortcuts(editor, transformers);
    }, [editor, transformers]);

    return null;
};

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
    autoFocus?: boolean;
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
    autoFocus,
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
            autoFocus={autoFocus}
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

/** TextInput for entering WYSIWYG text */
export const RichInputLexicalComponents = ({
    autoFocus = false,
    error = false,
    getTaggableItems,
    id,
    maxRows,
    minRows = 4,
    name,
    onActiveStatesChange,
    onBlur,
    onFocus,
    onChange,
    openAssistantDialog,
    placeholder = "",
    redo,
    setHandleAction,
    tabIndex,
    toggleMarkdown,
    undo,
    value,
    sxs,
}: RichInputLexicalProps) => {
    const { palette, typography } = useTheme();
    const editor = useLexicalComposerContext();

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
    useEffect(() => {
        return editor?.registerCommand(
            SELECTION_CHANGE_COMMAND,
            (_payload, newEditor) => {
                $updateToolbar();
                setActiveEditor(newEditor);
                return false;
            },
            COMMAND_PRIORITY_CRITICAL,
        );
    }, [editor, $updateToolbar]);

    const triggerEditorChange = useCallback(() => {
        console.log("updating editor value", value);
        editor?.update(() => {
            console.log("calling convertfrommarkdownstring in triggereditrochange");
            $convertFromMarkdownString(value, ALL_TRANSFORMERS);
        }, HISTORY_MERGE_OPTIONS);
    }, [editor, value]);

    useLayoutEffect(() => {
        if (!editor) return;
        return editor.registerListener("textcontent", (updatedMarkdown) => {
            console.log("textcontent listener triggered😽", updatedMarkdown);
            onChange(updatedMarkdown);
        });
    }, [editor, onChange]);

    // Toolbar actions
    const toggleHeading = useCallback((headingSize: HeadingTagType) => {
        editor?.update(() => {
            const selection = $getSelection();
            //TODO fix this or replace it. If replacing, you can also remove $setBlocksType
            // if (selection && $INTERNAL_isPointSelection(selection)) {
            //     $setBlocksType(selection as INTERNAL_PointSelection, () => activeStates[blockTypeToActionName[headingSize]] === true ? $createNode("Paragraph", {}) : $createNode("Heading", { headingSize }));
            // }
        });
    }, [activeStates, editor]);

    useEffect(() => {
        if (!setHandleAction || !editor) return;
        setHandleAction((action, data) => {
            const actionMap = {
                "Assistant": () => {
                    editor.update(() => {
                        const selection = $getSelection() as RangeSelection | null;
                        let selectedText = "";
                        if (selection !== null) {
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
                        const fullText = $convertToMarkdownString(ALL_TRANSFORMERS, $getRoot());
                        openAssistantDialog(selectedText, fullText);
                    });
                },
                "Bold": () => editor.dispatchCommand(FORMAT_TEXT_COMMAND, "BOLD"),
                "Code": () => editor.dispatchCommand(CODE_BLOCK_COMMAND, (void 0)),
                "Header1": () => toggleHeading("h1"),
                "Header2": () => toggleHeading("h2"),
                "Header3": () => toggleHeading("h3"),
                "Header4": () => toggleHeading("h4"),
                "Header5": () => toggleHeading("h5"),
                "Header6": () => toggleHeading("h6"),
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
            if (actionFunction) actionFunction();
        });
    }, [editor, openAssistantDialog, redo, setHandleAction, toggleHeading, triggerEditorChange, undo]);

    const lineHeight = useMemo(() => Math.round(typography.fontSize * LINE_HEIGHT_MULTIPLIER), [typography.fontSize]);

    return (
        <Box
            id={id}
            component="div"
            onBlur={onBlur}
            onFocus={onFocus}
            sx={{
                position: "relative",
                display: "grid",
                padding: "16.5px 14px",
                minWidth: "-webkit-fill-available",
                maxWidth: "-webkit-fill-available",
                borderRadius: "0 0 4px 4px",
                borderTop: "none",
                fontFamily: typography.fontFamily,
                fontSize: typography.fontSize + 2,
                lineHeight: `${lineHeight}px`,
                minHeight: lineHeight * minRows + 16.5,
                maxHeight: maxRows ? (lineHeight * (maxRows ?? 4) + 16.5) : "none",
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
            }}>
            <RichTextPlugin
                contentEditable={<ContentEditable
                    id={`${id}-contenteditable`}
                    autoFocus={autoFocus}
                    tabIndex={tabIndex}
                    style={{
                        outline: "none",
                        resize: "none",
                        overflow: "auto",
                        ...sxs?.textArea,
                    } as CSSProperties}
                />}
            />
            {(!editor || value.length === 0) && <div style={{
                color: palette.background.textSecondary,
                position: "absolute",
                padding: "16.5px 14px",
                pointerEvents: "none",
                top: 0,
                left: 0,
            }}
            >
                {!editor ? "Loading..." : (placeholder ?? "Enter some text...")}
            </div>}
            <CheckListPlugin />
            <LinkPlugin />
            <ListPlugin />
            <MarkdownShortcutPlugin transformers={ALL_TRANSFORMERS} />
            <RichInputTagDropdown {...tagData} selectDropdownItem={selectDropdownItem} />
            <TablePlugin />
            <SpoilerPlugin />
            <CodeBlockPlugin />
        </Box>
    );
};

/** TextInput for entering WYSIWYG text */
export const RichInputLexical = ({
    disabled,
    value,
    ...props
}: RichInputLexicalProps) => {

    // Set up Lexical editor
    const [editor, setEditor] = useState<LexicalEditor | null>(null);
    useEffect(() => {
        const initializeAsync = async () => {
            // Asynchronously create the editor instance
            const newEditor = await createEditor({ namespace: "RichInputEditor" });

            // Initialize editor with current value
            newEditor.update(() => { $convertFromMarkdownString(value, ALL_TRANSFORMERS); }, HISTORY_MERGE_OPTIONS);

            setEditor(newEditor);
        };

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
};
