import { HashtagNode } from "@lexical/hashtag";
import { $isLinkNode, AutoLinkNode, LinkNode, TOGGLE_LINK_COMMAND } from "@lexical/link";
import { $isListNode, INSERT_CHECK_LIST_COMMAND, INSERT_ORDERED_LIST_COMMAND, INSERT_UNORDERED_LIST_COMMAND, ListItemNode, ListNode } from "@lexical/list";
import { $convertToMarkdownString, ElementTransformer, HEADING, ORDERED_LIST, QUOTE, TEXT_FORMAT_TRANSFORMERS, TEXT_MATCH_TRANSFORMERS, TextMatchTransformer, UNORDERED_LIST } from "@lexical/markdown";
import { CheckListPlugin } from "@lexical/react/LexicalCheckListPlugin";
import { InitialEditorStateType } from "@lexical/react/LexicalComposer";
import { LexicalComposerContext, LexicalComposerContextType, createLexicalComposerContext, useLexicalComposerContext } from "@lexical/react/LexicalComposerContext";
import { ContentEditable } from "@lexical/react/LexicalContentEditable";
import LexicalErrorBoundary from "@lexical/react/LexicalErrorBoundary";
import { HorizontalRuleNode } from "@lexical/react/LexicalHorizontalRuleNode";
import { LinkPlugin } from "@lexical/react/LexicalLinkPlugin";
import { ListPlugin } from "@lexical/react/LexicalListPlugin";
import { MarkdownShortcutPlugin } from "@lexical/react/LexicalMarkdownShortcutPlugin";
import { OnChangePlugin } from "@lexical/react/LexicalOnChangePlugin";
import { RichTextPlugin } from "@lexical/react/LexicalRichTextPlugin";
import { TablePlugin } from "@lexical/react/LexicalTablePlugin";
import { $createHeadingNode, $isHeadingNode, HeadingNode, HeadingTagType, QuoteNode } from "@lexical/rich-text";
import { $isAtNodeEnd, $setBlocksType } from "@lexical/selection";
import { $isTableNode, TableCellNode, TableNode, TableRowNode } from "@lexical/table";
import { $findMatchingParent, $getNearestNodeOfType } from "@lexical/utils";
import { Box, useTheme } from "@mui/material";
import "highlight.js/styles/monokai-sublime.css";
import { $INTERNAL_isPointSelection, $createParagraphNode, $getRoot, $getSelection, $isRangeSelection, $isRootOrShadowRoot, COMMAND_PRIORITY_CRITICAL, EditorState, EditorThemeClasses, ElementNode, FORMAT_TEXT_COMMAND, INTERNAL_PointSelection, LexicalEditor, LineBreakNode, NodeKey, ParagraphNode, RangeSelection, SELECTION_CHANGE_COMMAND, TextNode, createEditor } from "lexical";
import { CSSProperties, useCallback, useEffect, useLayoutEffect, useMemo, useState } from "react";
import { ListObject } from "utils/display/listTools";
import { LINE_HEIGHT_MULTIPLIER } from "../RichInput/RichInput";
import { RichInputTagDropdown, useTagDropdown } from "../RichInputTagDropdown/RichInputTagDropdown";
import { defaultActiveStates } from "../RichInputToolbar/RichInputToolbar";
import { RichInputAction, RichInputActiveStates, RichInputLexicalProps } from "../types";
import { $convertFromMarkdownString } from "./builder";
import { CODE_BLOCK_TRANSFORMER, CodeBlockNode, CodeBlockPlugin, TOGGLE_CODE_BLOCK_COMMAND } from "./plugins/code/CodePlugin";
import { SPOILER_LINES_TRANSFORMER, SPOILER_TAGS_TRANSFORMER, SpoilerNode, SpoilerPlugin, TOGGLE_SPOILER_COMMAND } from "./plugins/spoiler/SpoilerPlugin";
import "./theme.css";

export const CAN_USE_DOM: boolean =
    typeof window !== "undefined" &&
    typeof window.document !== "undefined" &&
    typeof window.document.createElement !== "undefined";
const HISTORY_MERGE_OPTIONS = { tag: "history-merge" };

/** Every supported block type (e.g. lists, headers, quote) */
const blockTypeToActionName: { [x: string]: RichInputAction } = {
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

const rootTypeToRootName = {
    root: "Root",
    table: "Table",
};

/** Maps Lexical classes to CSS classes defined in theme.css */
const theme: EditorThemeClasses = {
    blockCursor: "RichInput__blockCursor",
    characterLimit: "RichInput__characterLimit",
    code: "RichInput__code",
    codeHighlight: {
        atrule: "RichInput__tokenAttr",
        attr: "RichInput__tokenAttr",
        boolean: "RichInput__tokenProperty",
        builtin: "RichInput__tokenSelector",
        cdata: "RichInput__tokenComment",
        char: "RichInput__tokenSelector",
        class: "RichInput__tokenFunction",
        "class-name": "RichInput__tokenFunction",
        comment: "RichInput__tokenComment",
        constant: "RichInput__tokenProperty",
        deleted: "RichInput__tokenProperty",
        doctype: "RichInput__tokenComment",
        entity: "RichInput__tokenOperator",
        function: "RichInput__tokenFunction",
        important: "RichInput__tokenVariable",
        inserted: "RichInput__tokenSelector",
        keyword: "RichInput__tokenAttr",
        namespace: "RichInput__tokenVariable",
        number: "RichInput__tokenProperty",
        operator: "RichInput__tokenOperator",
        prolog: "RichInput__tokenComment",
        property: "RichInput__tokenProperty",
        punctuation: "RichInput__tokenPunctuation",
        regex: "RichInput__tokenVariable",
        selector: "RichInput__tokenSelector",
        string: "RichInput__tokenSelector",
        symbol: "RichInput__tokenProperty",
        tag: "RichInput__tokenProperty",
        url: "RichInput__tokenOperator",
        variable: "RichInput__tokenVariable",
    },
    embedBlock: {
        base: "RichInput__embedBlock",
        focus: "RichInput__embedBlockFocus",
    },
    hashtag: "RichInput__hashtag",
    image: "editor-image",
    indent: "RichInput__indent",
    list: {
        listitem: "RichInput__listItem",
        listitemChecked: "RichInput__listItemChecked",
        listitemUnchecked: "RichInput__listItemUnchecked",
        nested: {
            listitem: "RichInput__nestedListItem",
        },
        olDepth: [
            "RichInput__ol1",
            "RichInput__ol2",
            "RichInput__ol3",
            "RichInput__ol4",
            "RichInput__ol5",
        ],
        ul: "RichInput__ul",
    },
    ltr: "RichInput__ltr",
    mark: "RichInput__mark",
    markOverlap: "RichInput__markOverlap",
    paragraph: "RichInput__paragraph",
    quote: "RichInput__quote",
    rtl: "RichInput__rtl",
    table: "RichInput__table",
    tableAddColumns: "RichInput__tableAddColumns",
    tableAddRows: "RichInput__tableAddRows",
    tableCell: "RichInput__tableCell",
    tableCellActionButton: "RichInput__tableCellActionButton",
    tableCellActionButtonContainer:
        "RichInput__tableCellActionButtonContainer",
    tableCellEditing: "RichInput__tableCellEditing",
    tableCellHeader: "RichInput__tableCellHeader",
    tableCellPrimarySelected: "RichInput__tableCellPrimarySelected",
    tableCellResizer: "RichInput__tableCellResizer",
    tableCellSelected: "RichInput__tableCellSelected",
    tableCellSortedIndicator: "RichInput__tableCellSortedIndicator",
    tableResizeRuler: "RichInput__tableCellResizeRuler",
    tableSelected: "RichInput__tableSelected",
    text: {
        bold: "RichInput__textBold",
        code: "RichInput__textCode",
        italic: "RichInput__textItalic",
        strikethrough: "RichInput__textStrikethrough",
        subscript: "RichInput__textSubscript",
        superscript: "RichInput__textSuperscript",
        underline: "RichInput__textUnderline",
        underlineStrikethrough: "RichInput__textUnderlineStrikethrough",
    },
};

/**
 * Initializes the Lexical editor with a specified initial state.
 * 
 * The initial editor state can vary in type (string, object, or function) to provide
 * flexibility in defining the starting state of the editor. This allows for different
 * use cases such as starting with plain text, a predefined editor state, or a dynamic
 * state constructed at runtime.
 * 
 * - String: When the initial state is a string, it's assumed to be plain text or serialized
 *   editor state content that will be parsed and set as the editor's current state.
 * - Object: An object is assumed to be a directly usable editor state, typically derived from
 *   `editor.parseEditorState()` or stored from a previous editor state.
 * - Function: A function allows for dynamic state initialization, where the function receives
 *   the editor instance and can perform various operations, like inserting nodes or setting
 *   selection, based on custom logic.
 * 
 * If the initial state is `undefined`, the editor will be initialized with a single empty paragraph,
 * which is a common default for text editors. If the initial state is `null`, the initialization is
 * skipped, which might be used to indicate that the editor should not alter its current state.
 * 
 * @param editor - The Lexical editor instance to be initialized.
 * @param  [initialEditorState] - The desired initial state of the editor,
 * which can be a string (plain text or serialized state), an object (editor state), or a function
 * (custom initialization logic).
 */
function initializeEditor(
    editor: LexicalEditor,
    initialEditorState?: InitialEditorStateType,
): void {
    if (initialEditorState === null) {
        return;
    } else if (initialEditorState === undefined) {
        editor.update(() => {
            const root = $getRoot();
            if (root.isEmpty()) {
                const paragraph = $createParagraphNode();
                root.append(paragraph);
                const activeElement = CAN_USE_DOM ? document.activeElement : null;
                if (
                    $getSelection() !== null ||
                    (activeElement !== null && activeElement === editor.getRootElement())
                ) {
                    paragraph.select();
                }
            }
        }, HISTORY_MERGE_OPTIONS);
    } else {
        switch (typeof initialEditorState) {
            case "string": {
                const parsedEditorState = editor.parseEditorState(initialEditorState);
                editor.setEditorState(parsedEditorState, HISTORY_MERGE_OPTIONS);
                break;
            }
            case "object": {
                editor.setEditorState(initialEditorState, HISTORY_MERGE_OPTIONS);
                break;
            }
            case "function": {
                editor.update(() => {
                    const root = $getRoot();
                    if (root.isEmpty()) {
                        initialEditorState(editor);
                    }
                }, HISTORY_MERGE_OPTIONS);
                break;
            }
        }
    }
}

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

// Custom transformers for syntax not supported by CommonMark (Markdown's spec)
const UNDERLINE: TextMatchTransformer = {
    dependencies: [],
    export: (node, exportChildren, exportFormat) => {
        const isUnderlined = (node as TextNode).__style === "text-decoration: underline;";
        const shouldExportChildren = node instanceof ElementNode;
        if (isUnderlined) {
            return `<u>${shouldExportChildren ? exportChildren(node as ElementNode) : (node as TextNode).__text}</u>`;
        }
        return null;
    },
    importRegExp: /<u>(.*?)<\/u>/,
    regExp: /<u>(.*?)<\/u>$/,
    replace: (textNode, match) => {
        const newTextContent = match[1];
        const newTextNode = new TextNode(newTextContent);
        newTextNode.setStyle("text-decoration: underline;");
        textNode.replace(newTextNode);
    },
    trigger: "<u>",
    type: "text-match",
};

const CUSTOM_TEXT_TRANSFORMERS: Array<TextMatchTransformer | ElementTransformer> = [
    UNDERLINE,
    SPOILER_LINES_TRANSFORMER,
    SPOILER_TAGS_TRANSFORMER,
    CODE_BLOCK_TRANSFORMER,
];

const ALL_TRANSFORMERS = [
    ...CUSTOM_TEXT_TRANSFORMERS,
    // ...TRANSFORMERS,
    HEADING,
    QUOTE,
    // // CODE,
    UNORDERED_LIST,
    ORDERED_LIST,
    ...TEXT_FORMAT_TRANSFORMERS,
    ...TEXT_MATCH_TRANSFORMERS,
];

/** Actual components of RichInputLexical. Needed so that we can use the lexical provider */
const RichInputLexicalComponents = ({
    autoFocus = false,
    disabled = false,
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
    const { palette, spacing, typography } = useTheme();
    const [editor] = useLexicalComposerContext();

    const tagData = useTagDropdown({ getTaggableItems });
    const selectDropdownItem = useCallback((item: ListObject) => {
        //TODO
    }, []);

    /** Store current text properties. Logic inspired by https://github.com/facebook/lexical/blob/9e83533d52fe934bd91aaa5baaf156f682577dcf/packages/lexical-playground/src/plugins/ToolbarPlugin/index.tsx#L484 */
    const [activeEditor, setActiveEditor] = useState(editor);
    const [activeStates, setActiveStates] = useState<Omit<RichInputActiveStates, "SetValue">>({ ...defaultActiveStates });
    const [selectedElementKey, setSelectedElementKey] = useState<NodeKey | null>(null);
    const [isLink, setIsLink] = useState(false);
    const [isItalic, setIsItalic] = useState(false);
    const [isEditable, setIsEditable] = useState(() => editor.isEditable());
    const $updateToolbar = useCallback(() => {
        const updatedStates = { ...defaultActiveStates };
        const selection = $getSelection();
        if ($isRangeSelection(selection)) {
            const anchorNode = selection.anchor.getNode();
            let element =
                anchorNode.getKey() === "root"
                    ? anchorNode
                    : $findMatchingParent(anchorNode, (e) => {
                        const parent = e.getParent();
                        return parent !== null && $isRootOrShadowRoot(parent);
                    });

            if (element === null) {
                element = anchorNode.getTopLevelElementOrThrow();
            }

            const elementKey = element.getKey();
            const elementDOM = activeEditor.getElementByKey(elementKey);

            // Find text formats
            updatedStates.Bold = selection.hasFormat("bold");
            updatedStates.Italic = selection.hasFormat("italic");
            updatedStates.Underline = selection.hasFormat("underline");
            updatedStates.Strikethrough = selection.hasFormat("strikethrough");
            // updatedStates.Subscript = selection.hasFormat("subscript");
            // updatedStates.Superscript = selection.hasFormat("superscript");
            // Check if link
            const node = getSelectedNode(selection);
            const parent = node.getParent();
            if ($isLinkNode(parent) || $isLinkNode(node)) {
                updatedStates.Link = true;
            }
            // Check if table
            const tableNode = $findMatchingParent(node, $isTableNode);
            if ($isTableNode(tableNode)) {
                updatedStates.Table = true;
            }

            if (elementDOM !== null) {
                setSelectedElementKey(elementKey);
                if ($isListNode(element)) {
                    const parentList = $getNearestNodeOfType<ListNode>(
                        anchorNode,
                        ListNode,
                    );
                    const type = parentList
                        ? parentList.getListType()
                        : element.getListType();
                    console.log("got block type 1", type);
                    if (type in blockTypeToActionName) {
                        updatedStates[blockTypeToActionName[type as keyof typeof blockTypeToActionName]] = true;
                    }
                } else {
                    const type = $isHeadingNode(element)
                        ? element.getTag()
                        : element.getType();
                    if (type in blockTypeToActionName) {
                        console.log("got block type 2", type);
                        updatedStates[blockTypeToActionName[type as keyof typeof blockTypeToActionName]] = true;
                    }
                }
            }
            // TODO need to find spoilers, underlines, headers, code and quote blocks, and lists
            console.log("updated states", updatedStates);
            setActiveStates({ ...updatedStates });
            onActiveStatesChange({ ...updatedStates });
        }
    }, [activeEditor, onActiveStatesChange]);
    useEffect(() => {
        return editor.registerCommand(
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
        editor.update(() => {
            console.log("calling convertfrommarkdownstring in triggereditrochange");
            $convertFromMarkdownString(value, ALL_TRANSFORMERS);
        }, HISTORY_MERGE_OPTIONS);
    }, [editor, value]);

    const handleChange = useCallback((editorState: EditorState) => {
        const updatedMarkdown = editorState.read(() => {
            const root = $getRoot();
            return $convertToMarkdownString(ALL_TRANSFORMERS, root);
        });
        onChange(updatedMarkdown);
    }, [onChange]);

    // Toolbar actions
    const toggleHeading = useCallback((headingSize: HeadingTagType) => {
        editor.update(() => {
            const selection = $getSelection();
            if (selection && $INTERNAL_isPointSelection(selection)) {
                $setBlocksType(selection as INTERNAL_PointSelection, () => activeStates[blockTypeToActionName[headingSize]] === true ? $createParagraphNode() : $createHeadingNode(headingSize));
            }
        });
    }, [activeStates, editor]);


    useEffect(() => {
        if (!setHandleAction) return;
        setHandleAction((action, data) => {
            console.log("in RichInputlExical handleAction", action);
            const dispatch = editor.dispatchCommand;
            const actionMap = {
                "Assistant": () => openAssistantDialog(""),
                "Bold": () => dispatch(FORMAT_TEXT_COMMAND, "bold"),
                "Code": () => dispatch(TOGGLE_CODE_BLOCK_COMMAND, (void 0)),
                "Header1": () => toggleHeading("h1"),
                "Header2": () => toggleHeading("h2"),
                "Header3": () => toggleHeading("h3"),
                "Header4": () => toggleHeading("h4"),
                "Header5": () => toggleHeading("h5"),
                "Header6": () => toggleHeading("h6"),
                "Italic": () => dispatch(FORMAT_TEXT_COMMAND, "italic"),
                "Link": () => dispatch(TOGGLE_LINK_COMMAND, "https://"), //TODO not working
                "ListBullet": () => dispatch(INSERT_UNORDERED_LIST_COMMAND, (void 0)),
                "ListCheckbox": () => dispatch(INSERT_CHECK_LIST_COMMAND, (void 0)), // TODO not working
                "ListNumber": () => dispatch(INSERT_ORDERED_LIST_COMMAND, (void 0)),
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
                "Spoiler": dispatch(TOGGLE_SPOILER_COMMAND, (void 0)),
                "Strikethrough": () => dispatch(FORMAT_TEXT_COMMAND, "strikethrough"),
                "Table": () => { }, //TODO
                "Underline": () => dispatch(FORMAT_TEXT_COMMAND, "underline"),
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
                    style={{
                        outline: "none",
                        resize: "none",
                        overflow: "auto",
                        ...sxs?.textArea,
                    } as CSSProperties}
                />}
                placeholder={<></>} // Doesn't work for some reason, so we use our own placeholder
                ErrorBoundary={LexicalErrorBoundary}
            />
            {value.length === 0 && <div style={{
                color: palette.background.textSecondary,
                position: "absolute",
                padding: "16.5px 14px",
                pointerEvents: "none",
                top: 0,
                left: 0,
            }}
            >
                {placeholder ?? "Enter some text..."}
            </div>}
            <CheckListPlugin />
            <LinkPlugin />
            <ListPlugin />
            <OnChangePlugin onChange={handleChange} />
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
    value,
    ...props
}: RichInputLexicalProps) => {

    const onError = (error: Error) => {
        console.error(error);
    };

    /** Configuration for lexical editor */
    const initialConfig = useMemo(() => ({
        editable: true,
        // Will need custom transformers if we want to support custom markdown syntax (e.g. underline, spoiler)
        editorState: () => {
            console.log("calling convertfrommarkdownstring in initialconfig.editorState function");
            $convertFromMarkdownString(value, ALL_TRANSFORMERS);
        },
        namespace: "RichInputEditor",
        nodes: [AutoLinkNode, CodeBlockNode, HashtagNode, HeadingNode, HorizontalRuleNode, LineBreakNode, LinkNode, ListNode, ListItemNode, ParagraphNode, QuoteNode, SpoilerNode, TableNode, TableCellNode, TableRowNode],
        onError,
        theme,
    }), [value]);

    /** Lexical editor context, for finding and manipulating state */
    const composerContext: [LexicalEditor, LexicalComposerContextType] = useMemo(() => {
        const {
            editorState: initialEditorState,
            namespace,
            nodes,
            onError,
            theme,
        } = initialConfig;
        const context = createLexicalComposerContext(null, theme);
        const newEditor = createEditor({
            editable: initialConfig.editable,
            namespace,
            nodes,
            onError: (error) => onError(error),
            theme,
        });
        initializeEditor(newEditor, initialEditorState);
        const editor = newEditor;
        return [editor, context];
        // We only do this for init
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);
    useLayoutEffect(() => {
        const isEditable = initialConfig.editable;
        const [editor] = composerContext;
        editor.setEditable(isEditable !== undefined ? isEditable : true);
        // We only do this for init
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    return (
        <LexicalComposerContext.Provider value={composerContext}>
            <RichInputLexicalComponents value={value} {...props} />
        </LexicalComposerContext.Provider>
    );
};
