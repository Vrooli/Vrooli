import { $isCodeNode, CodeHighlightNode, CodeNode, CODE_LANGUAGE_MAP } from "@lexical/code";
import { HashtagNode } from "@lexical/hashtag";
import { $isLinkNode, AutoLinkNode, LinkNode, TOGGLE_LINK_COMMAND } from "@lexical/link";
import { $isListNode, INSERT_CHECK_LIST_COMMAND, INSERT_ORDERED_LIST_COMMAND, INSERT_UNORDERED_LIST_COMMAND, ListItemNode, ListNode } from "@lexical/list";
import { $convertFromMarkdownString, TextMatchTransformer, TRANSFORMERS } from "@lexical/markdown";
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
import { $findMatchingParent, $getNearestNodeOfType, mergeRegister } from "@lexical/utils";
import { Box, useTheme } from "@mui/material";
import "highlight.js/styles/monokai-sublime.css";
import { $applyNodeReplacement, $createParagraphNode, $getRoot, $getSelection, $isElementNode, $isRangeSelection, $isRootOrShadowRoot, COMMAND_PRIORITY_CRITICAL, COMMAND_PRIORITY_EDITOR, createCommand, createEditor, DEPRECATED_$isGridSelection, DOMConversionMap, DOMConversionOutput, EditorConfig, EditorThemeClasses, ElementFormatType, ElementNode, FORMAT_TEXT_COMMAND, LexicalCommand, LexicalEditor, LexicalNode, NodeKey, RangeSelection, SELECTION_CHANGE_COMMAND, TextFormatType, TextNode } from "lexical";
import { CSSProperties, useCallback, useEffect, useLayoutEffect, useMemo, useState } from "react";
import { ListObject } from "utils/display/listTools";
import { LINE_HEIGHT_MULTIPLIER } from "../RichInputBase/RichInputBase";
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
    heading: {
        h1: "RichInput__h1",
        h2: "RichInput__h2",
        h3: "RichInput__h3",
        h4: "RichInput__h4",
        h5: "RichInput__h5",
        h6: "RichInput__h6",
    },
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

// Create a custom node for spoilers
export class SpoilerNode extends ElementNode {
    static getType(): string {
        return "spoiler";
    }

    constructor(key?: NodeKey) {
        super(key);
    }

    static clone(node: SpoilerNode): SpoilerNode {
        return new SpoilerNode(node.__key);
    }

    toggleSpoilerVisibility = (event: Event) => {
        const target = event.currentTarget as HTMLElement;
        if (target.classList.contains("revealed")) {
            target.classList.remove("revealed");
            target.classList.add("hidden");
        } else {
            target.classList.remove("hidden");
            target.classList.add("revealed");
        }
    };

    createDOM(_config: EditorConfig): HTMLElement {
        const element = document.createElement("span");
        element.classList.add("spoiler", "hidden"); // Start in hidden state
        element.setAttribute("spellcheck", "false"); // Disable spellcheck so that it doesn't block the toggle
        element.addEventListener("click", this.toggleSpoilerVisibility);
        return element;
    }

    updateDOM(_prevNode: SpoilerNode, span: HTMLElement, config: EditorConfig): boolean {
        // Assuming you might want to update some styles or attributes in the future, but for now, it's a no-op
        // Simply checks that the element is a spoiler and updates if needed
        if (!span.classList.contains("spoiler")) {
            span.classList.add("spoiler");
            span.setAttribute("spellcheck", "false");
            span.addEventListener("click", this.toggleSpoilerVisibility);
            return true; // Indicates that the DOM was updated
        }
        return false; // Indicates that no updates were necessary
    }

    static importDOM(): DOMConversionMap | null {
        return {
            span: (node: Node) => {
                // Check if the span has the correct class or attribute to identify it as a spoiler
                if ((node as HTMLElement).classList.contains("spoiler")) {
                    return {
                        conversion: SpoilerNode.fromDOMElement,
                        priority: 1,
                    };
                }
                return null;
            },
        };
    }

    static fromDOMElement(element: HTMLElement): DOMConversionOutput {
        // Creates a new SpoilerNode from the given DOM element
        return { node: new SpoilerNode() };
    }

    isInline(): true {
        return true;
    }
}
export function $createSpoilerNode(): SpoilerNode {
    return $applyNodeReplacement(new SpoilerNode());
}
// Create a command to toggle spoilers
const SPOILER_COMMAND: LexicalCommand<void> = createCommand("FORMAT_TEXT_COMMAND");

/** Register commands for custom components (i.e. spoiler) */
const registerCustomCommands = (editor: LexicalEditor): (() => void) => {
    const removeListener = mergeRegister(
        editor.registerCommand<void>(
            SPOILER_COMMAND,
            () => {
                const selection = $getSelection();
                if (!$isRangeSelection(selection)) {
                    return false;
                }
                const nodes = selection.getNodes();
                console.log("got nodes", nodes, nodes.map(node => node.getParent()));
                // Check if there is a spoiler node in the selection, or if the selection is a single node with a parent spoiler node
                const isAlreadySpoiled = (nodes.length > 0 && nodes.some((node) => node instanceof SpoilerNode)) ||
                    (nodes.length === 1 && nodes[0].getParent() instanceof SpoilerNode);
                console.log("is already spoiled", isAlreadySpoiled);
                if (isAlreadySpoiled) {
                    // Remove each spoiler effect by replacing it with its children
                    nodes.forEach((node) => {
                        let spoilerNode = node;
                        // If node isn't a spoiler, try the parent
                        if (!(spoilerNode instanceof SpoilerNode)) {
                            spoilerNode = node.getParent() as LexicalNode;
                            if (!(spoilerNode instanceof SpoilerNode)) {
                                return;
                            }
                        }
                        const children = spoilerNode.getChildren<LexicalNode>();
                        for (const child of children) {
                            spoilerNode.insertBefore(child);
                        }
                        spoilerNode.remove();
                    });
                } else if (nodes.length > 0) {
                    // // Apply spoiler effect
                    // const spoilerNode = $createSpoilerNode();

                    // // Pre-insert the spoiler node to its position in the hierarchy.
                    // // TODO this adds the spoiler to the whole line. Need to fix
                    // const firstNode = nodes[0];
                    // firstNode.insertBefore(spoilerNode);

                    // // Move nodes into the spoiler node.
                    // for (const node of nodes) {
                    //     if (!(node instanceof SpoilerNode)) {
                    //         console.log("adding node to spoiler", node.getTextContent());
                    //         spoilerNode.append(node);  // Assuming this moves the node
                    //     }
                    // }
                    // // Adjust the selection to the entire spoilerNode
                    // spoilerNode.select();


                    // const spoilerNode = $createSpoilerNode();

                    // // For the first node, if it's partially selected, split and take the part that's within the selection
                    // let startNode = nodes[0];
                    // console.log("first node", startNode.getTextContent());
                    // if (startNode.getTextContent().length !== selection.anchor.offset) {
                    //     startNode = startNode.splitText(selection.anchor.offset);
                    // }
                    // console.log("got start node", (startNode as unknown as any[]).map(node => node.getTextContent()));

                    // // For the last node, if it's partially selected, split and take the part that's within the selection
                    // const endNode = nodes[nodes.length - 1];
                    // if (endNode.getTextContent().length !== selection.focus.offset) {
                    //     endNode.splitText(selection.focus.offset);
                    // }
                    // console.log("got end node", endNode);

                    // // Now, from the startNode to endNode, wrap everything in between in the spoilerNode
                    // let currentNode = startNode;
                    // while (currentNode !== endNode) {
                    //     spoilerNode.append(currentNode);
                    //     const next = currentNode.getNextSibling();
                    //     if (next === null) {
                    //         break;
                    //     }
                    //     currentNode = next!;
                    // }
                    // spoilerNode.append(endNode);

                    // const parent = startNode.getParent();
                    // if (parent === null) {
                    //     return false;
                    // }
                    // console.log("got parent", parent);
                    // parent.insertBefore(spoilerNode, startNode);
                    // spoilerNode.select();




                    const spoilerNode = $createSpoilerNode();

                    const startNode = nodes[0];
                    console.log("got start node", startNode.getTextContent());
                    const endNode = nodes[nodes.length - 1];
                    const startParent = startNode.getParent();

                    // Get offsets
                    const [startOffset, endOffset] = [selection.anchor.offset, selection.focus.offset - selection.anchor.offset].sort();
                    console.log("got offsets", startOffset, endOffset);

                    // Split the startNode if needed
                    if (startNode.getTextContentSize() > startOffset) {
                        const beforeText = startNode.getTextContent().substring(0, startOffset);
                        const newNode = new TextNode(beforeText); // Assuming TextNode constructor accepts a string
                        console.log("new node 1", newNode.getTextContent());
                        startNode.insertBefore(newNode);
                        startNode.setTextContent(startNode.getTextContent().substring(startOffset));
                    }
                    console.log("got start node", startNode.getTextContent());

                    // Split the endNode if needed
                    if (endOffset < endNode.getTextContentSize()) {
                        const afterText = endNode.getTextContent().substring(endOffset);
                        const newNode = new TextNode(afterText);
                        console.log("new node 2", newNode.getTextContent());
                        endNode.insertAfter(newNode);
                        endNode.setTextContent(endNode.getTextContent().substring(0, endOffset));
                    }
                    console.log("got end node", endNode.getTextContent());

                    // At this point, our selection should be bounded by complete TextNodes.
                    // We now loop through all nodes and move them into the spoilerNode.
                    for (const node of nodes) {
                        if (!(node instanceof SpoilerNode)) {
                            console.log("adding node to spoiler", node.getTextContent());
                            spoilerNode.append(node);
                        }
                    }

                    // Insert the spoilerNode in the correct location
                    console.log("insert the spoiler node", startNode.getTextContent(), spoilerNode.getTextContent());
                    // startNode.getParent()?.insertBefore(spoilerNode);
                    if (startParent) {
                        startParent.insertBefore(spoilerNode);
                    } else {
                        // Handle cases where the startParent is not available
                        console.error("Failed to find a parent for the startNode!");
                    }

                    // Adjust the selection to the entire spoilerNode
                    spoilerNode.select();
                }
                return true;
            },
            COMMAND_PRIORITY_EDITOR,
        ),
    );
    return removeListener;
};

// Custom transformers for syntax not supported by CommonMark (Markdown's spec)
const UNDERLINE: TextMatchTransformer = {
    dependencies: [],
    export: (node, exportChildren, exportFormat) => {
        if (node.hasStyle("textDecoration", "underline")) {
            return `<u>${exportChildren(node as ElementNode)}</u>`;
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
const SPOILER: TextMatchTransformer = {
    dependencies: [],
    export: (node, exportChildren, exportFormat) => {
        if (node.hasStyle("backgroundColor", "black") && node.hasStyle("color", "black")) {
            return `||${exportChildren(node as ElementNode)}||`;
        }
        return null;
    },
    importRegExp: /\|\|(.*?)\|\|/,
    regExp: /\|\|(.*?)\|\|$/,
    replace: (textNode, match) => {
        // Extract the matched spoiler text
        const spoilerText = match[1];
        console.log("SPOILER TEXT", match, textNode);

        // Create a new styled text node with the matched spoiler text
        const spoilerTextNode = new TextNode(spoilerText);
        console.log("SPOILER TEXT NODE", spoilerTextNode);

        // Create a SpoilerNode
        const spoilerNode = $createSpoilerNode();
        spoilerNode.append(spoilerTextNode);

        // Replace the original text node with the spoiler node
        textNode.replace(spoilerNode);
    },
    trigger: "||",
    type: "text-match",
};

const CUSTOM_TEXT_TRANSFORMERS: Array<TextMatchTransformer> = [
    UNDERLINE,
    SPOILER,
];

const ALL_TRANSFORMERS = [...TRANSFORMERS, ...CUSTOM_TEXT_TRANSFORMERS];

/** TextField for entering WYSIWYG text */
export const RichInputLexical = ({
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
}: RichInputLexicalProps) => {
    const { palette, spacing, typography } = useTheme();

    const onError = (error: Error) => {
        console.error(error);
    };

    /** Configuration for lexical editor */
    const initialConfig = useMemo(() => ({
        editable: true,
        // Will need custom transformers if we want to support custom markdown syntax (e.g. underline, spoiler)
        editorState: () => $convertFromMarkdownString(value, ALL_TRANSFORMERS),
        namespace: "MyEditor",
        nodes: [AutoLinkNode, CodeNode, CodeHighlightNode, HashtagNode, HeadingNode, HorizontalRuleNode, LinkNode, ListNode, ListItemNode, QuoteNode, SpoilerNode],
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
        registerCustomCommands(editor);
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
        composerContext[0].dispatchCommand(FORMAT_TEXT_COMMAND, formatType);
    };
    const toggleSpoiler = () => {
        composerContext[0].dispatchCommand(SPOILER_COMMAND, (void 0));
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
            "Spoiler": toggleSpoiler,
            "Strikethrough": () => toggleFormat("strikethrough"),
            "Underline": () => toggleFormat("underline"),
            "Undo": undo,
        };
        const actionFunction = actionMap[action];
        if (actionFunction) actionFunction();
    };

    return (
        <LexicalComposerContext.Provider value={composerContext}>
            <Box
                id={id}
                sx={{
                    position: "relative",
                    display: "grid",
                    padding: "16.5px 14px",
                    minWidth: "-webkit-fill-available",
                    maxWidth: "-webkit-fill-available",
                    borderColor: error ? palette.error.main : palette.divider,
                    borderRadius: "0 0 4px 4px",
                    borderTop: "none",
                    fontFamily: typography.fontFamily,
                    fontSize: typography.fontSize + 2,
                    lineHeight: `${Math.round(typography.fontSize * LINE_HEIGHT_MULTIPLIER)}px`,
                    backgroundColor: palette.background.paper,
                    color: palette.text.primary,
                    ...sx,
                }}>
                <RichTextPlugin
                    contentEditable={<ContentEditable
                        style={{
                            outline: "none",
                            resize: "none",
                            overflow: "auto",
                        } as CSSProperties}
                    />}
                    placeholder={
                        <div style={{
                            color: palette.background.textSecondary,
                            position: "absolute",
                            padding: "16.5px 14px",
                            pointerEvents: "none",
                            top: 0,
                            left: 0,
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
                <MarkdownShortcutPlugin transformers={ALL_TRANSFORMERS} />
            </Box>
        </LexicalComposerContext.Provider>
    );
};
