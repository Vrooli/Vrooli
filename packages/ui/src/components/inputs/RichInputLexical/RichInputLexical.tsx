import { CodeHighlightNode, CodeNode } from "@lexical/code";
import { LinkNode, TOGGLE_LINK_COMMAND } from "@lexical/link";
import { $isListNode, INSERT_CHECK_LIST_COMMAND, INSERT_ORDERED_LIST_COMMAND, INSERT_UNORDERED_LIST_COMMAND, ListItemNode, ListNode } from "@lexical/list";
// import { AutocompleteNode, ImageNode, KeywordNode} from "@lexical/node";
import { $isCodeNode, CODE_LANGUAGE_MAP } from "@lexical/code";
import { HashtagNode } from "@lexical/hashtag";
import { $isLinkNode, AutoLinkNode } from "@lexical/link";
import { $convertFromMarkdownString, TRANSFORMERS } from "@lexical/markdown";
import { CheckListPlugin } from "@lexical/react/LexicalCheckListPlugin";
import { InitialEditorStateType } from "@lexical/react/LexicalComposer";
import { createLexicalComposerContext, LexicalComposerContext, LexicalComposerContextType } from "@lexical/react/LexicalComposerContext";
import { ContentEditable } from "@lexical/react/LexicalContentEditable";
import LexicalErrorBoundary from "@lexical/react/LexicalErrorBoundary";
import { HistoryPlugin } from "@lexical/react/LexicalHistoryPlugin";
import { HorizontalRuleNode } from "@lexical/react/LexicalHorizontalRuleNode";
import { LinkPlugin } from "@lexical/react/LexicalLinkPlugin";
import { ListPlugin } from "@lexical/react/LexicalListPlugin";
import { MarkdownShortcutPlugin } from "@lexical/react/LexicalMarkdownShortcutPlugin";
import { RichTextPlugin } from "@lexical/react/LexicalRichTextPlugin";
import { $createHeadingNode, $isHeadingNode, HeadingNode, HeadingTagType, QuoteNode } from "@lexical/rich-text";
import { $isAtNodeEnd, $isParentElementRTL, $setBlocksType } from "@lexical/selection";
import { $isTableNode } from "@lexical/table";
import { $findMatchingParent, $getNearestNodeOfType } from "@lexical/utils";
import { Box, useTheme } from "@mui/material";
import "highlight.js/styles/monokai-sublime.css";
import type { EditorThemeClasses } from "lexical";
import { $createParagraphNode, $getRoot, $getSelection, $isElementNode, $isRangeSelection, $isRootOrShadowRoot, COMMAND_PRIORITY_CRITICAL, createEditor, DEPRECATED_$isGridSelection, ElementFormatType, ElementNode, FORMAT_TEXT_COMMAND, LexicalEditor, NodeKey, RangeSelection, SELECTION_CHANGE_COMMAND, TextFormatType, TextNode } from "lexical";
import { CSSProperties, useCallback, useEffect, useLayoutEffect, useMemo, useState } from "react";
import { linkColors } from "styles";
import { ListObject } from "utils/display/listTools";
import { RichInputTagDropdown, useTagDropdown } from "../RichInputTagDropdown/RichInputTagDropdown";
import { RichInputAction } from "../RichInputToolbar/RichInputToolbar";
import { RichInputChildView, RichInputLexicalProps } from "../types";
import "./theme.css";

export const CAN_USE_DOM: boolean =
    typeof window !== "undefined" &&
    typeof window.document !== "undefined" &&
    typeof window.document.createElement !== "undefined";
const HISTORY_MERGE_OPTIONS = { tag: "history-merge" };

const blockTypeToBlockName = {
    bullet: "Bulleted List",
    check: "Check List",
    code: "Code Block",
    h1: "Heading 1",
    h2: "Heading 2",
    h3: "Heading 3",
    h4: "Heading 4",
    h5: "Heading 5",
    h6: "Heading 6",
    number: "Numbered List",
    paragraph: "Normal",
    quote: "Quote",
};

const rootTypeToRootName = {
    root: "Root",
    table: "Table",
};

const theme: EditorThemeClasses = {
    blockCursor: "PlaygroundEditorTheme__blockCursor",
    characterLimit: "PlaygroundEditorTheme__characterLimit",
    code: "PlaygroundEditorTheme__code",
    codeHighlight: {
        atrule: "PlaygroundEditorTheme__tokenAttr",
        attr: "PlaygroundEditorTheme__tokenAttr",
        boolean: "PlaygroundEditorTheme__tokenProperty",
        builtin: "PlaygroundEditorTheme__tokenSelector",
        cdata: "PlaygroundEditorTheme__tokenComment",
        char: "PlaygroundEditorTheme__tokenSelector",
        class: "PlaygroundEditorTheme__tokenFunction",
        "class-name": "PlaygroundEditorTheme__tokenFunction",
        comment: "PlaygroundEditorTheme__tokenComment",
        constant: "PlaygroundEditorTheme__tokenProperty",
        deleted: "PlaygroundEditorTheme__tokenProperty",
        doctype: "PlaygroundEditorTheme__tokenComment",
        entity: "PlaygroundEditorTheme__tokenOperator",
        function: "PlaygroundEditorTheme__tokenFunction",
        important: "PlaygroundEditorTheme__tokenVariable",
        inserted: "PlaygroundEditorTheme__tokenSelector",
        keyword: "PlaygroundEditorTheme__tokenAttr",
        namespace: "PlaygroundEditorTheme__tokenVariable",
        number: "PlaygroundEditorTheme__tokenProperty",
        operator: "PlaygroundEditorTheme__tokenOperator",
        prolog: "PlaygroundEditorTheme__tokenComment",
        property: "PlaygroundEditorTheme__tokenProperty",
        punctuation: "PlaygroundEditorTheme__tokenPunctuation",
        regex: "PlaygroundEditorTheme__tokenVariable",
        selector: "PlaygroundEditorTheme__tokenSelector",
        string: "PlaygroundEditorTheme__tokenSelector",
        symbol: "PlaygroundEditorTheme__tokenProperty",
        tag: "PlaygroundEditorTheme__tokenProperty",
        url: "PlaygroundEditorTheme__tokenOperator",
        variable: "PlaygroundEditorTheme__tokenVariable",
    },
    embedBlock: {
        base: "PlaygroundEditorTheme__embedBlock",
        focus: "PlaygroundEditorTheme__embedBlockFocus",
    },
    hashtag: "PlaygroundEditorTheme__hashtag",
    heading: {
        h1: "PlaygroundEditorTheme__h1",
        h2: "PlaygroundEditorTheme__h2",
        h3: "PlaygroundEditorTheme__h3",
        h4: "PlaygroundEditorTheme__h4",
        h5: "PlaygroundEditorTheme__h5",
        h6: "PlaygroundEditorTheme__h6",
    },
    image: "editor-image",
    indent: "PlaygroundEditorTheme__indent",
    link: "PlaygroundEditorTheme__link",
    list: {
        listitem: "PlaygroundEditorTheme__listItem",
        listitemChecked: "PlaygroundEditorTheme__listItemChecked",
        listitemUnchecked: "PlaygroundEditorTheme__listItemUnchecked",
        nested: {
            listitem: "PlaygroundEditorTheme__nestedListItem",
        },
        olDepth: [
            "PlaygroundEditorTheme__ol1",
            "PlaygroundEditorTheme__ol2",
            "PlaygroundEditorTheme__ol3",
            "PlaygroundEditorTheme__ol4",
            "PlaygroundEditorTheme__ol5",
        ],
        ul: "PlaygroundEditorTheme__ul",
    },
    ltr: "PlaygroundEditorTheme__ltr",
    mark: "PlaygroundEditorTheme__mark",
    markOverlap: "PlaygroundEditorTheme__markOverlap",
    paragraph: "PlaygroundEditorTheme__paragraph",
    quote: "PlaygroundEditorTheme__quote",
    rtl: "PlaygroundEditorTheme__rtl",
    table: "PlaygroundEditorTheme__table",
    tableAddColumns: "PlaygroundEditorTheme__tableAddColumns",
    tableAddRows: "PlaygroundEditorTheme__tableAddRows",
    tableCell: "PlaygroundEditorTheme__tableCell",
    tableCellActionButton: "PlaygroundEditorTheme__tableCellActionButton",
    tableCellActionButtonContainer:
        "PlaygroundEditorTheme__tableCellActionButtonContainer",
    tableCellEditing: "PlaygroundEditorTheme__tableCellEditing",
    tableCellHeader: "PlaygroundEditorTheme__tableCellHeader",
    tableCellPrimarySelected: "PlaygroundEditorTheme__tableCellPrimarySelected",
    tableCellResizer: "PlaygroundEditorTheme__tableCellResizer",
    tableCellSelected: "PlaygroundEditorTheme__tableCellSelected",
    tableCellSortedIndicator: "PlaygroundEditorTheme__tableCellSortedIndicator",
    tableResizeRuler: "PlaygroundEditorTheme__tableCellResizeRuler",
    tableSelected: "PlaygroundEditorTheme__tableSelected",
    text: {
        bold: "PlaygroundEditorTheme__textBold",
        code: "PlaygroundEditorTheme__textCode",
        italic: "PlaygroundEditorTheme__textItalic",
        strikethrough: "PlaygroundEditorTheme__textStrikethrough",
        subscript: "PlaygroundEditorTheme__textSubscript",
        superscript: "PlaygroundEditorTheme__textSuperscript",
        underline: "PlaygroundEditorTheme__textUnderline",
        underlineStrikethrough: "PlaygroundEditorTheme__textUnderlineStrikethrough",
    },
};

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
    } else if (initialEditorState !== null) {
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

/** TextField for entering WYSIWYG text */
export const RichInputLexical = ({
    autoFocus = false,
    disabled = false,
    error = false,
    getTaggableItems,
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
}: RichInputLexicalProps) => {
    const { palette, spacing } = useTheme();

    const onError = (error: Error) => {
        console.error(error);
    };

    /** Configuration for lexical editor */
    const initialConfig = useMemo(() => ({
        editable: true,
        editorState: () => $convertFromMarkdownString(value, TRANSFORMERS),
        namespace: "MyEditor",
        nodes: [AutoLinkNode, CodeNode, CodeHighlightNode, HashtagNode, HeadingNode, HorizontalRuleNode, LinkNode, ListNode, ListItemNode, QuoteNode],
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
        const context: LexicalComposerContextType = createLexicalComposerContext(
            null,
            theme,
        );
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
    const tagData = useTagDropdown({ getTaggableItems });
    const selectDropdownItem = useCallback((item: ListObject) => {
        //TODO
    }, []);

    /** Store current text properties */
    const [activeEditor, setActiveEditor] = useState(composerContext[0]);
    const [blockType, setBlockType] = useState<keyof typeof blockTypeToBlockName>("paragraph");
    const [rootType, setRootType] = useState<keyof typeof rootTypeToRootName>("root");
    const [selectedElementKey, setSelectedElementKey] = useState<NodeKey | null>(null);
    const [elementFormat, setElementFormat] = useState<ElementFormatType>("left");
    const [isLink, setIsLink] = useState(false);
    const [isBold, setIsBold] = useState(false);
    const [isItalic, setIsItalic] = useState(false);
    const [isUnderline, setIsUnderline] = useState(false);
    const [isStrikethrough, setIsStrikethrough] = useState(false);
    const [isSubscript, setIsSubscript] = useState(false);
    const [isSuperscript, setIsSuperscript] = useState(false);
    const [isCode, setIsCode] = useState(false);
    const [canUndo, setCanUndo] = useState(false);
    const [canRedo, setCanRedo] = useState(false);
    const [isRTL, setIsRTL] = useState(false);
    const [codeLanguage, setCodeLanguage] = useState<string>("");
    const [isEditable, setIsEditable] = useState(() => composerContext[0].isEditable());
    useEffect(() => {
        console.log("current editor status", {
            activeEditor,
            blockType,
            rootType,
            selectedElementKey,
            elementFormat,
            isLink,
            isBold,
            isItalic,
            isUnderline,
            isStrikethrough,
            isSubscript,
            isSuperscript,
            isCode,
            canUndo,
            canRedo,
            isRTL,
            codeLanguage,
            isEditable,
        });
    }, [activeEditor, blockType, rootType, selectedElementKey, elementFormat, isLink, isBold, isItalic, isUnderline, isStrikethrough, isSubscript, isSuperscript, isCode, canUndo, canRedo, isRTL, codeLanguage, isEditable]);
    const $updateToolbar = useCallback(() => {
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

            // Update text format
            setIsBold(selection.hasFormat("bold"));
            setIsItalic(selection.hasFormat("italic"));
            setIsUnderline(selection.hasFormat("underline"));
            setIsStrikethrough(selection.hasFormat("strikethrough"));
            setIsSubscript(selection.hasFormat("subscript"));
            setIsSuperscript(selection.hasFormat("superscript"));
            setIsCode(selection.hasFormat("code"));
            setIsRTL($isParentElementRTL(selection));

            // Update links
            const node = getSelectedNode(selection);
            const parent = node.getParent();
            if ($isLinkNode(parent) || $isLinkNode(node)) {
                setIsLink(true);
            } else {
                setIsLink(false);
            }

            const tableNode = $findMatchingParent(node, $isTableNode);
            if ($isTableNode(tableNode)) {
                setRootType("table");
            } else {
                setRootType("root");
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
                    setBlockType(type);
                } else {
                    const type = $isHeadingNode(element)
                        ? element.getTag()
                        : element.getType();
                    if (type in blockTypeToBlockName) {
                        setBlockType(type as keyof typeof blockTypeToBlockName);
                    }
                    if ($isCodeNode(element)) {
                        const language =
                            element.getLanguage() as keyof typeof CODE_LANGUAGE_MAP;
                        setCodeLanguage(
                            language ? CODE_LANGUAGE_MAP[language] || language : "",
                        );
                        return;
                    }
                }
            }
            // Handle buttons
            setElementFormat(
                ($isElementNode(node)
                    ? node.getFormatType()
                    : parent?.getFormatType()) || "left",
            );
        }
    }, [activeEditor]);
    useEffect(() => {
        return composerContext[0].registerCommand(
            SELECTION_CHANGE_COMMAND,
            (_payload, newEditor) => {
                $updateToolbar();
                setActiveEditor(newEditor);
                return false;
            },
            COMMAND_PRIORITY_CRITICAL,
        );
    }, [composerContext, $updateToolbar]);

    // Toolbar actions
    const toggleHeading = (headingSize: HeadingTagType) => {
        composerContext[0].update(() => {
            const selection = $getSelection();
            if (
                $isRangeSelection(selection) ||
                DEPRECATED_$isGridSelection(selection)
            ) {
                $setBlocksType(selection, () => blockType === headingSize ? $createParagraphNode() : $createHeadingNode(headingSize));
            }
        });
    };
    const toggleFormat = (formatType: TextFormatType) => {
        // composerContext[0].update(() => {
        //     const selection = $getSelection();
        //     console.log("got selection", selection, $isRangeSelection(selection));
        //     if ($isRangeSelection(selection)) {
        //         selection.toggleFormat(formatType);
        //     }
        // });
        composerContext[0].dispatchCommand(FORMAT_TEXT_COMMAND, formatType);
    };
    (RichInputLexical as unknown as RichInputChildView).handleAction = (action: RichInputAction) => {
        console.log("in RichInputlExical handleAction", action);
        const dispatch = composerContext[0].dispatchCommand;
        const actionMap: { [key in RichInputAction]: (() => unknown) } = {
            "Assistant": () => openAssistantDialog(""),
            "Bold": () => toggleFormat("bold"),
            "Header1": () => toggleHeading("h1"),
            "Header2": () => toggleHeading("h2"),
            "Header3": () => toggleHeading("h3"),
            "Italic": () => toggleFormat("italic"),
            "Link": () => dispatch(TOGGLE_LINK_COMMAND, "https://"), //TODO not working
            "ListBullet": () => dispatch(INSERT_UNORDERED_LIST_COMMAND, (void 0)),
            "ListCheckbox": () => dispatch(INSERT_CHECK_LIST_COMMAND, (void 0)), // TODO not working
            "ListNumber": () => dispatch(INSERT_ORDERED_LIST_COMMAND, (void 0)),
            "Mode": toggleMarkdown,
            "Redo": redo,
            "Strikethrough": () => toggleFormat("strikethrough"),
            "Undo": undo,
        };
        const actionFunction = actionMap[action];
        if (actionFunction) actionFunction();
    };

    return (
        <LexicalComposerContext.Provider value={composerContext}>
            <Box className="editor-container" sx={{
                border: `1px solid ${error ? "red" : "black"}`,
                borderRadius: "0 0 0.5rem 0.5rem",
                padding: "12px",
                wordBreak: "break-word",
                overflow: "auto",
                backgroundColor: palette.background.paper,
                color: palette.text.primary,
                position: "relative",
                ...linkColors(palette),
                ...sx,
            }}>
                <RichTextPlugin
                    contentEditable={<ContentEditable
                        className="editor-input"
                        // content={content ?? ""}
                        style={{
                            outline: "none",
                            border: "1px solid red",
                        } as CSSProperties}
                    />}
                    placeholder={
                        <div style={{
                            position: "absolute",
                            pointerEvents: "none",
                            top: 0,
                            left: 0,
                            padding: spacing(1),
                        }}
                        >
                            {placeholder ?? "Enter some text..."}
                        </div>
                    }
                    ErrorBoundary={LexicalErrorBoundary}
                />
                {/* <OnChangePlugin onChange={onChange} /> */}
                <CheckListPlugin />
                <HistoryPlugin />
                <LinkPlugin />
                <ListPlugin />
                <RichInputTagDropdown {...tagData} selectDropdownItem={selectDropdownItem} />
                <MarkdownShortcutPlugin transformers={TRANSFORMERS} />
            </Box>
        </LexicalComposerContext.Provider>
    );
};
