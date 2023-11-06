import { $isCodeNode, CodeHighlightNode, CodeNode, CODE_LANGUAGE_MAP } from "@lexical/code";
import { HashtagNode } from "@lexical/hashtag";
import { $isLinkNode, AutoLinkNode, LinkNode, TOGGLE_LINK_COMMAND } from "@lexical/link";
import { $isListNode, INSERT_CHECK_LIST_COMMAND, INSERT_ORDERED_LIST_COMMAND, INSERT_UNORDERED_LIST_COMMAND, ListItemNode, ListNode } from "@lexical/list";
import { $convertFromMarkdownString, $convertToMarkdownString, ElementTransformer, TextMatchTransformer, TRANSFORMERS } from "@lexical/markdown";
import { CheckListPlugin } from "@lexical/react/LexicalCheckListPlugin";
import { InitialEditorStateType } from "@lexical/react/LexicalComposer";
import { createLexicalComposerContext, LexicalComposerContext, LexicalComposerContextType, useLexicalComposerContext } from "@lexical/react/LexicalComposerContext";
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
import { $findMatchingParent, $getNearestNodeOfType, mergeRegister } from "@lexical/utils";
import { Box, useTheme } from "@mui/material";
import "highlight.js/styles/monokai-sublime.css";
import { $applyNodeReplacement, $createParagraphNode, $getRoot, $getSelection, $isRangeSelection, $isRootOrShadowRoot, COMMAND_PRIORITY_CRITICAL, COMMAND_PRIORITY_EDITOR, createCommand, createEditor, DEPRECATED_$isGridSelection, DOMConversionMap, DOMConversionOutput, EditorConfig, EditorState, EditorThemeClasses, ElementNode, FORMAT_TEXT_COMMAND, LexicalCommand, LexicalEditor, LexicalNode, LineBreakNode, NodeKey, ParagraphNode, RangeSelection, SELECTION_CHANGE_COMMAND, SerializedLexicalNode, Spread, TextFormatType, TextNode } from "lexical";
import { CSSProperties, useCallback, useEffect, useLayoutEffect, useMemo, useState } from "react";
import { ListObject } from "utils/display/listTools";
import { LINE_HEIGHT_MULTIPLIER } from "../RichInputBase/RichInputBase";
import { RichInputTagDropdown, useTagDropdown } from "../RichInputTagDropdown/RichInputTagDropdown";
import { defaultActiveStates } from "../RichInputToolbar/RichInputToolbar";
import { RichInputAction, RichInputActiveStates, RichInputLexicalProps } from "../types";
import "./theme.css";

export const CAN_USE_DOM: boolean =
    typeof window !== "undefined" &&
    typeof window.document !== "undefined" &&
    typeof window.document.createElement !== "undefined";
const HISTORY_MERGE_OPTIONS = { tag: "history-merge" };

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

type SerializedSpoilerNode = Spread<
    {
        children: any[];
    },
    SerializedLexicalNode
>;
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

    static importJSON(serializedNode: SerializedSpoilerNode): SpoilerNode {
        if (serializedNode.type === "spoiler" && serializedNode.version === 1) {
            const node = new SpoilerNode();
            serializedNode.children.forEach(childJSON => {
                node.append(childJSON);
            });
            return node;
        }
        throw new Error("Invalid serialized spoiler node.");
    }

    exportJSON(): any {//SerializedSpoilerNode { TODO
        const childrenJSON = this.getChildren().map(child => child.exportJSON());
        return {
            type: "spoiler",
            version: 1,
            children: childrenJSON,
        } as any; //const;
    }
}
export function $createSpoilerNode(): SpoilerNode {
    return $applyNodeReplacement(new SpoilerNode());
}
// Create a command to toggle spoilers
const SPOILER_COMMAND: LexicalCommand<void> = createCommand("FORMAT_TEXT_COMMAND");

/** Register commands for custom components (i.e. spoiler) */
const registerCustomCommands = (editor: LexicalEditor): (() => unknown) => {
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
                    const spoilerNode = $createSpoilerNode();

                    const startNode = nodes[0];
                    console.log("got start node", startNode.getTextContent());
                    const endNode = nodes[nodes.length - 1];
                    console.log("got end node", endNode.getTextContent());
                    const startParent = startNode.getParent();
                    console.log("start parent at beginning", startParent?.getTextContent());
                    const nodeAfterStartNode = startNode.getNextSibling();

                    // Get offsets from smallest to largest
                    const [startOffset, endOffset] = [selection.anchor.offset, selection.focus.offset].sort((a, b) => a - b);
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

                    // If the spoiler spans multiple nodes:
                    if (startNode !== endNode) {
                        let currentNode = startNode.getNextSibling();
                        while (currentNode && currentNode !== endNode) {
                            spoilerNode.append(currentNode);
                            currentNode = currentNode.getNextSibling();
                        }
                    }

                    // Split the endNode if needed
                    if (endOffset < endNode.getTextContentSize()) {
                        const afterText = endNode.getTextContent().substring(endOffset - startOffset);
                        const newNode = new TextNode(afterText);
                        console.log("new node 2", newNode.getTextContent());
                        endNode.insertAfter(newNode);
                        endNode.setTextContent(endNode.getTextContent().substring(0, endOffset - startOffset));
                    }
                    console.log("got end node", endNode.getTextContent());

                    // Append the (possibly split) endNode to spoilerNode
                    spoilerNode.append(endNode);

                    // Insert the spoilerNode in the correct location
                    console.log("insert the spoiler node", startNode.getTextContent(), spoilerNode.getTextContent());
                    if (startParent) {
                        startParent.insertAfter(spoilerNode);
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
// Spoiler with ||spoiler|| syntax
const SPOILER_LINES: TextMatchTransformer = {
    dependencies: [],
    export: (node, exportChildren, exportFormat) => {
        const isSpoiler = (node as TextNode).__type === "spoiler";
        const shouldExportChildren = node instanceof ElementNode;
        if (isSpoiler) {
            return `||${shouldExportChildren ? exportChildren(node as ElementNode) : (node as TextNode).__text}||`;
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
// Spoiler with <spoiler>spoiler</spoiler> syntax
const SPOILER_TAGS: TextMatchTransformer = {
    dependencies: [],
    export: (node, exportChildren, exportFormat) => {
        const isSpoiler = (node as TextNode).__type === "spoiler";
        const shouldExportChildren = node instanceof ElementNode;
        if (isSpoiler) {
            return `<spoiler>${shouldExportChildren ? exportChildren(node as ElementNode) : (node as TextNode).__text}</spoiler>`;
        }
        return null;
    },
    importRegExp: /<spoiler>(.*?)<\/spoiler>/,
    regExp: /<spoiler>(.*?)<\/spoiler>$/,
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
    trigger: "<spoiler>",
    type: "text-match",
};
// Empty lines are converted to <br> tags
export const EMPTY_LINE_BREAKS: ElementTransformer = {
    dependencies: [ParagraphNode],
    export: () => { return null; },
    regExp: /^$/,
    replace: (textNode, nodes, _, isImport) => {
        if (isImport && nodes.length === 1) {
            console.log(textNode);
            nodes[0].replace($createParagraphNode());
        }
    },
    type: "element",
};

const CUSTOM_TEXT_TRANSFORMERS: Array<TextMatchTransformer | ElementTransformer> = [
    UNDERLINE,
    SPOILER_LINES,
    SPOILER_TAGS,
    EMPTY_LINE_BREAKS,
];

const ALL_TRANSFORMERS = [...TRANSFORMERS, ...CUSTOM_TEXT_TRANSFORMERS];

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
    onChange,
    openAssistantDialog,
    placeholder = "",
    redo,
    setHandleAction,
    tabIndex,
    toggleMarkdown,
    undo,
    value,
    sx,
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
    const [codeLanguage, setCodeLanguage] = useState<string>("");
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
            // updatedStates.Code = selection.hasFormat("code");
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
                    if ($isCodeNode(element)) {
                        const language =
                            element.getLanguage() as keyof typeof CODE_LANGUAGE_MAP;
                        console.log("got code language", language);
                        setCodeLanguage(
                            language ? CODE_LANGUAGE_MAP[language] || language : "",
                        );
                        return;
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
            if (
                $isRangeSelection(selection) ||
                DEPRECATED_$isGridSelection(selection)
            ) {
                $setBlocksType(selection, () => activeStates[blockTypeToActionName[headingSize]] === true ? $createParagraphNode() : $createHeadingNode(headingSize));
            }
        });
    }, [activeStates, editor]);
    const toggleFormat = useCallback((formatType: TextFormatType) => {
        editor.dispatchCommand(FORMAT_TEXT_COMMAND, formatType);
    }, [editor]);
    const toggleSpoiler = useCallback(() => {
        editor.dispatchCommand(SPOILER_COMMAND, (void 0));
    }, [editor]);


    useEffect(() => {
        if (!setHandleAction) return;
        setHandleAction((action, data) => {
            console.log("in RichInputlExical handleAction", action);
            const dispatch = editor.dispatchCommand;
            const actionMap = {
                "Assistant": () => openAssistantDialog(""),
                "Bold": () => toggleFormat("bold"),
                "Code": () => { }, //TODO
                "Header1": () => toggleHeading("h1"),
                "Header2": () => toggleHeading("h2"),
                "Header3": () => toggleHeading("h3"),
                "Header4": () => toggleHeading("h4"),
                "Header5": () => toggleHeading("h5"),
                "Header6": () => toggleHeading("h6"),
                "Italic": () => toggleFormat("italic"),
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
                        $convertFromMarkdownString(data, ALL_TRANSFORMERS);
                    }, HISTORY_MERGE_OPTIONS);
                },
                "Spoiler": toggleSpoiler,
                "Strikethrough": () => toggleFormat("strikethrough"),
                "Table": () => { }, //TODO
                "Underline": () => toggleFormat("underline"),
                "Undo": () => {
                    undo();
                    triggerEditorChange();
                },
            };
            const actionFunction = actionMap[action];
            if (actionFunction) actionFunction();
        });
    }, [editor, openAssistantDialog, redo, setHandleAction, toggleFormat, toggleHeading, toggleSpoiler, triggerEditorChange, undo]);

    return (
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
            <CheckListPlugin />
            <LinkPlugin />
            <ListPlugin />
            <OnChangePlugin onChange={handleChange} />
            <MarkdownShortcutPlugin transformers={ALL_TRANSFORMERS} />
            <RichInputTagDropdown {...tagData} selectDropdownItem={selectDropdownItem} />
            <TablePlugin />
        </Box>
    );
};

/** TextField for entering WYSIWYG text */
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
        editorState: () => $convertFromMarkdownString(value, ALL_TRANSFORMERS),
        namespace: "RichInputEditor",
        nodes: [AutoLinkNode, CodeNode, CodeHighlightNode, HashtagNode, HeadingNode, HorizontalRuleNode, LineBreakNode, LinkNode, ListNode, ListItemNode, ParagraphNode, QuoteNode, SpoilerNode, TableNode, TableCellNode, TableRowNode],
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

    return (
        <LexicalComposerContext.Provider value={composerContext}>
            <RichInputLexicalComponents value={value} {...props} />
        </LexicalComposerContext.Provider>
    );
};
