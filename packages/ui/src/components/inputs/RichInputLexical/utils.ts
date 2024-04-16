/* eslint-disable @typescript-eslint/ban-ts-comment */
import { $insertDataTransferForRichText, copyToClipboard } from "./clipboard";
import { CUT_COMMAND, PASTE_COMMAND } from "./commands";
import { CAN_USE_DOM, COMPOSITION_SUFFIX, DOM_TEXT_TYPE, HAS_DIRTY_NODES, IGNORE_TAGS, IS_APPLE, IS_APPLE_WEBKIT, IS_FIREFOX, IS_IOS, IS_SAFARI, LTR_REGEX, RTL_REGEX } from "./consts";
import { type EditorState, type LexicalEditor } from "./editor";
import { DecoratorNode } from "./nodes/DecoratorNode";
import { type ElementNode } from "./nodes/ElementNode";
import { type LexicalNode } from "./nodes/LexicalNode";
import { $isLineBreakNode, LineBreakNode } from "./nodes/LineBreakNode";
import { $createListItemNode, ListItemNode } from "./nodes/ListItemNode";
import { ListNode } from "./nodes/ListNode";
import { $createParagraphNode, $isParagraphNode, ParagraphNode } from "./nodes/ParagraphNode";
import { QuoteNode } from "./nodes/QuoteNode";
import { RootNode } from "./nodes/RootNode";
import { $isTableCellNode, TableCellNode } from "./nodes/TableCellNode";
import { TableNode } from "./nodes/TableNode";
import { $isTableRowNode } from "./nodes/TableRowNode";
import { TextNode } from "./nodes/TextNode";
import { $getCharacterOffsets, $getPreviousSelection, NodeSelection, Point, RangeSelection } from "./selection";
import { BaseSelection, CommandPayloadType, DOMChildConversion, DOMConversion, DOMConversionFn, DOMConversionOutput, EditorConfig, EditorThemeClasses, IntentionallyMarkedAsDirtyElement, Klass, LexicalCommand, MutatedNodes, MutationListeners, NodeKey, NodeMap, NodeMutation, ObjectKlass, PasteCommandType, PointType, RegisteredNodes, ShadowRootNode, Spread, TableMapType, TableMapValueType } from "./types";
import { errorOnInfiniteTransforms, errorOnReadOnly, getActiveEditor, getActiveEditorState, isCurrentlyReadOnlyMode, triggerCommandListeners, updateEditor } from "./updates";

export const getWindow = (editor: LexicalEditor): Window => {
    const windowObj = editor._window;
    if (windowObj === null) {
        throw new Error("window object not found");
    }
    return windowObj;
};

export const getDOMSelection = (targetWindow: null | Window): null | Selection => {
    return !CAN_USE_DOM ? null : (targetWindow || window).getSelection();
};

export const getDefaultView = (domElem: HTMLElement): Window | null => {
    const ownerDoc = domElem.ownerDocument;
    return (ownerDoc && ownerDoc.defaultView) || null;
};

export const isFirefoxClipboardEvents = (editor: LexicalEditor): boolean => {
    const event = getWindow(editor).event;
    const inputType = event && (event as InputEvent).inputType;
    return (
        inputType === "insertFromPaste" ||
        inputType === "insertFromPasteAsQuotation"
    );
};

export const dispatchCommand = <TCommand extends LexicalCommand<unknown>>(
    editor: LexicalEditor,
    command: TCommand,
    payload: CommandPayloadType<TCommand>,
): boolean => {
    return triggerCommandListeners(editor, command, payload);
};

export const normalizeClassNames = (
    ...classNames: Array<typeof undefined | boolean | null | string>
): Array<string> => {
    const rval: string[] = [];
    for (const className of classNames) {
        if (className && typeof className === "string") {
            for (const [s] of className.matchAll(/\S+/g)) {
                rval.push(s);
            }
        }
    }
    return rval;
};

export const getCachedClassNameArray = (
    classNamesTheme: EditorThemeClasses,
    classNameThemeType: string,
): Array<string> => {
    if (classNamesTheme.__lexicalClassNameCache === undefined) {
        classNamesTheme.__lexicalClassNameCache = {};
    }
    const classNamesCache = classNamesTheme.__lexicalClassNameCache;
    const cachedClassNames = classNamesCache[classNameThemeType];
    if (cachedClassNames !== undefined) {
        return cachedClassNames;
    }
    const classNames = classNamesTheme[classNameThemeType];
    // As we're using classList, we need
    // to handle className tokens that have spaces.
    // The easiest way to do this to convert the
    // className tokens to an array that can be
    // applied to classList.add()/remove().
    if (typeof classNames === "string") {
        const classNamesArr = normalizeClassNames(classNames);
        classNamesCache[classNameThemeType] = classNamesArr;
        return classNamesArr;
    }
    return classNames;
};

export const markAllNodesAsDirty = (editor: LexicalEditor, type: string): void => {
    // Mark all existing text nodes as dirty
    updateEditor(
        editor,
        () => {
            const editorState = getActiveEditorState();
            if (editorState.isEmpty()) {
                return;
            }
            if (type === "root") {
                $getRoot().markDirty();
                return;
            }
            const nodeMap = editorState._nodeMap;
            for (const [, node] of nodeMap) {
                node.markDirty();
            }
        },
        editor._pendingEditorState === null
            ? {
                tag: "history-merge",
            }
            : undefined,
    );
};

export const $getCompositionKey = (): null | NodeKey => {
    if (isCurrentlyReadOnlyMode()) {
        return null;
    }
    const editor = getActiveEditor();
    return editor._compositionKey;
};

export const $setCompositionKey = (compositionKey: null | NodeKey): void => {
    errorOnReadOnly();
    const editor = getActiveEditor();
    const previousCompositionKey = editor._compositionKey;
    if (compositionKey !== previousCompositionKey) {
        editor._compositionKey = compositionKey;
        if (previousCompositionKey !== null) {
            const node = $getNodeByKey(previousCompositionKey);
            if (node !== null) {
                node.getWritable();
            }
        }
        if (compositionKey !== null) {
            const node = $getNodeByKey(compositionKey);
            if (node !== null) {
                node.getWritable();
            }
        }
    }
};

export const $getRoot = (): RootNode => {
    return getActiveEditorState()._nodeMap.get("root") as RootNode;
};

export const $getSelection = (): BaseSelection | null => {
    return getActiveEditorState()._selection;
};

/**
 * @param x - The element being tested
 * @returns Returns true if x is an HTML anchor tag, false otherwise
 */
export const isHTMLAnchorElement = (x: Node): x is HTMLAnchorElement => {
    return isHTMLElement(x) && x.tagName === "A";
};

export const $isDecoratorNode = <T>(
    node: LexicalNode | null | undefined,
): node is DecoratorNode<T> => {
    return node instanceof DecoratorNode;
};

export const $isListNode = (
    node: LexicalNode | null | undefined,
): node is ListNode => {
    return node instanceof ListNode;
};

export const $isListItemNode = (
    node: LexicalNode | null | undefined,
): node is ListItemNode => {
    return node instanceof ListItemNode;
};

export const $isQuoteNode = (
    node: LexicalNode | null | undefined,
): node is QuoteNode => {
    return node instanceof QuoteNode;
};

export const $isSelectionCapturedInDecorator = (node: Node): boolean => {
    return $isDecoratorNode($getNearestNodeFromDOMNode(node));
};

export const $isElementNode = (
    node: LexicalNode | null | undefined,
): node is ElementNode => {
    const editor = getActiveEditor();
    const ElementNode = Object.getPrototypeOf(
        editor._nodes.get("paragraph")!.klass,
    );

    return node instanceof ElementNode;
};

export const $isTextNode = (
    node: LexicalNode | null | undefined,
): node is TextNode => {
    const editor = getActiveEditor();
    const TextNode = editor._nodes.get("text")!.klass;

    return node instanceof TextNode;
};

export const $isRootNode = (
    node: RootNode | LexicalNode | null | undefined,
): node is RootNode => {
    return node instanceof RootNode;
};

export const isHTMLElement = (x: unknown): x is HTMLElement => {
    return x instanceof HTMLElement;
};

export const $isRangeSelection = (x: unknown): x is RangeSelection => {
    // Duck typing :P (and not instanceof RangeSelection) because extension operates
    // from different JS bundle and has no reference to the RangeSelection used on the page
    return x != null && typeof x === "object" && "applyDOMRange" in x;
};

export const $isNodeSelection = (x: unknown): x is NodeSelection => {
    // Duck typing :P (and not instanceof NodeSelection) because extension operates
    // from different JS bundle and has no reference to the NodeSelection used on the page
    return x != null && typeof x === "object" && "_nodes" in x;
};

export const $isRootOrShadowRoot = (
    node: null | LexicalNode,
): node is RootNode | ShadowRootNode => {
    return $isRootNode(node) || ($isElementNode(node) && node.isShadowRoot());
};

export const $setSelection = (selection: null | BaseSelection): void => {
    errorOnReadOnly();
    const editorState = getActiveEditorState();
    if (selection !== null) {
        selection.dirty = true;
        selection.setCachedNodes(null);
    }
    editorState._selection = selection;
};

export function $nodesOfType<T extends LexicalNode>(klass: Klass<T>): Array<T> {
    const editorState = getActiveEditorState();
    const readOnly = editorState._readOnly;
    const klassType = klass.getType();
    const nodes = editorState._nodeMap;
    const nodesOfType: Array<T> = [];
    for (const [, node] of nodes) {
        if (
            node instanceof klass &&
            node.__type === klassType &&
            (readOnly || node.isAttached())
        ) {
            nodesOfType.push(node as T);
        }
    }
    return nodesOfType;
}

/**
 * Takes a node and traverses up its ancestors (toward the root node)
 * in order to find a specific type of node.
 * @param node - the node to begin searching.
 * @param klass - an instance of the type of node to look for.
 * @returns the node of type klass that was passed, or null if none exist.
 */
export const $getNearestNodeOfType = <T extends ElementNode>(
    node: LexicalNode,
    klass: Klass<T>,
): T | null => {
    let parent: ElementNode | LexicalNode | null = node;

    while (parent != null) {
        if (parent instanceof klass) {
            return parent as T;
        }

        // @ts-ignore TODO
        parent = parent.getParent();
    }

    return null;
};

export const $updateTextNodeFromDOMContent = (
    textNode: TextNode,
    textContent: string,
    anchorOffset: null | number,
    focusOffset: null | number,
    compositionEnd: boolean,
): void => {
    let node = textNode;

    if (node.isAttached() && (compositionEnd || !node.isDirty())) {
        const isComposing = node.isComposing();
        let normalizedTextContent = textContent;

        if (
            (isComposing || compositionEnd) &&
            textContent[textContent.length - 1] === COMPOSITION_SUFFIX
        ) {
            normalizedTextContent = textContent.slice(0, -1);
        }
        const prevTextContent = node.getTextContent();

        if (compositionEnd || normalizedTextContent !== prevTextContent) {
            if (normalizedTextContent === "") {
                $setCompositionKey(null);
                if (!IS_SAFARI && !IS_IOS && !IS_APPLE_WEBKIT) {
                    // For composition (mainly Android), we have to remove the node on a later update
                    const editor = getActiveEditor();
                    setTimeout(() => {
                        editor.update(() => {
                            if (node.isAttached()) {
                                node.remove();
                            }
                        });
                    }, 20);
                } else {
                    node.remove();
                }
                return;
            }
            const parent = node.getParent();
            const prevSelection = $getPreviousSelection();
            const prevTextContentSize = node.getTextContentSize();
            const compositionKey = $getCompositionKey();
            const nodeKey = node.getKey();

            if (
                node.isToken() ||
                (compositionKey !== null &&
                    nodeKey === compositionKey &&
                    !isComposing) ||
                // Check if character was added at the start or boundaries when not insertable, and we need
                // to clear this input from occurring as that action wasn't permitted.
                ($isRangeSelection(prevSelection) &&
                    ((parent !== null &&
                        !parent.canInsertTextBefore() &&
                        prevSelection.anchor.offset === 0) ||
                        (prevSelection.anchor.key === textNode.__key &&
                            prevSelection.anchor.offset === 0 &&
                            !node.canInsertTextBefore() &&
                            !isComposing) ||
                        (prevSelection.focus.key === textNode.__key &&
                            prevSelection.focus.offset === prevTextContentSize &&
                            !node.canInsertTextAfter() &&
                            !isComposing)))
            ) {
                node.markDirty();
                return;
            }
            const selection = $getSelection();

            if (
                !$isRangeSelection(selection) ||
                anchorOffset === null ||
                focusOffset === null
            ) {
                node.setTextContent(normalizedTextContent);
                return;
            }
            selection.setTextNodeRange(node, anchorOffset, node, focusOffset);

            if (node.isSegmented()) {
                const originalTextContent = node.getTextContent();
                const replacement = $createTextNode(originalTextContent);
                node.replace(replacement);
                node = replacement;
            }
            node.setTextContent(normalizedTextContent);
        }
    }
};

/**
 * Determines if the current selection is at the end of the node.
 * @param point - The point of the selection to test.
 * @returns true if the provided point offset is in the last possible position, false otherwise.
 */
export const $isAtNodeEnd = (point: Point): boolean => {
    if (point.type === "text") {
        return point.offset === point.getNode().getTextContentSize();
    }
    const node = point.getNode();
    if (!$isElementNode(node)) {
        throw new Error("isAtNodeEnd: node must be a TextNode or ElementNode");
    }

    return point.offset === node.getChildrenSize();
};

/**
 * Starts with a node and moves up the tree (toward the root node) to find a matching node based on
 * the search parameters of the findFn. (Consider JavaScripts' .find() function where a testing function must be
 * passed as an argument. eg. if( (node) => node.__type === 'div') ) return true; otherwise return false
 * @param startingNode - The node where the search starts.
 * @param findFn - A testing function that returns true if the current node satisfies the testing parameters.
 * @returns A parent node that matches the findFn parameters, or null if one wasn't found.
 */
export const $findMatchingParent: {
    <T extends LexicalNode>(
        startingNode: LexicalNode,
        findFn: (node: LexicalNode) => node is T,
    ): T | null;
    (
        startingNode: LexicalNode,
        findFn: (node: LexicalNode) => boolean,
    ): LexicalNode | null;
} = (
    startingNode: LexicalNode,
    findFn: (node: LexicalNode) => boolean,
): LexicalNode | null => {
        let curr: ElementNode | LexicalNode | null = startingNode;

        while (curr !== $getRoot() && curr != null) {
            if (findFn(curr)) {
                return curr;
            }

            curr = curr.getParent();
        }

        return null;
    };

export const getEditorsToPropagate = (
    editor: LexicalEditor,
): Array<LexicalEditor> => {
    const editorsToPropagate: LexicalEditor[] = [];
    let currentEditor: LexicalEditor | null = editor;
    while (currentEditor !== null) {
        editorsToPropagate.push(currentEditor);
        currentEditor = currentEditor._parentEditor;
    }
    return editorsToPropagate;
};

export const scheduleMicroTask: (fn: () => void) => void =
    typeof queueMicrotask === "function"
        ? queueMicrotask
        : (fn) => {
            // No window prefix intended (#1400)
            Promise.resolve().then(fn);
        };

export const getEditorStateTextContent = (editorState: EditorState): string => {
    return editorState.read(() => $getRoot().getTextContent());
};

export const removeDOMBlockCursorElement = (
    blockCursorElement: HTMLElement,
    editor: LexicalEditor,
    rootElement: HTMLElement,
) => {
    rootElement.style.removeProperty("caret-color");
    editor._blockCursorElement = null;
    const parentElement = blockCursorElement.parentElement;
    if (parentElement !== null) {
        parentElement.removeChild(blockCursorElement);
    }
};

const createBlockCursorElement = (editorConfig: EditorConfig): HTMLDivElement => {
    const theme = editorConfig.theme;
    const element = document.createElement("div");
    element.contentEditable = "false";
    element.setAttribute("data-lexical-cursor", "true");
    let blockCursorTheme = theme.blockCursor;
    if (blockCursorTheme !== undefined) {
        if (typeof blockCursorTheme === "string") {
            const classNamesArr = normalizeClassNames(blockCursorTheme);
            // @ts-expect-error: intentional
            blockCursorTheme = theme.blockCursor = classNamesArr;
        }
        if (blockCursorTheme !== undefined) {
            element.classList.add(...blockCursorTheme);
        }
    }
    return element;
};

export const $applyNodeReplacement = <N extends LexicalNode>(
    node: LexicalNode,
): N => {
    const editor = getActiveEditor();
    const nodeType = node.constructor.getType();
    const registeredNode = editor._nodes.get(nodeType);
    if (registeredNode === undefined) {
        throw new Error("$initializeNode failed. Ensure node has been registered to the editor. You can do this by passing the node class via the \"nodes\" array in the editor config.");
    }
    const replaceFunc = registeredNode.replace;
    if (replaceFunc !== null) {
        const replacementNode = replaceFunc(node) as N;
        if (!(replacementNode instanceof node.constructor)) {
            throw new Error("$initializeNode failed. Ensure replacement node is a subclass of the original node.");
        }
        return replacementNode;
    }
    return node as N;
};

export const $createTextNode = (text = ""): TextNode => {
    return $applyNodeReplacement(new TextNode(text));
};

const needsBlockCursor = (node: null | LexicalNode): boolean => {
    return (
        ($isDecoratorNode(node) || ($isElementNode(node) && !node.canBeEmpty())) &&
        !node.isInline()
    );
};

export const updateDOMBlockCursorElement = (
    editor: LexicalEditor,
    rootElement: HTMLElement,
    nextSelection: null | BaseSelection,
) => {
    let blockCursorElement = editor._blockCursorElement;

    if (
        $isRangeSelection(nextSelection) &&
        nextSelection.isCollapsed() &&
        nextSelection.anchor.type === "element" &&
        rootElement.contains(document.activeElement)
    ) {
        const anchor = nextSelection.anchor;
        const elementNode = anchor.getNode();
        const offset = anchor.offset;
        const elementNodeSize = elementNode.getChildrenSize();
        let isBlockCursor = false;
        let insertBeforeElement: null | HTMLElement = null;

        if (offset === elementNodeSize) {
            const child = elementNode.getChildAtIndex(offset - 1);
            if (needsBlockCursor(child)) {
                isBlockCursor = true;
            }
        } else {
            const child = elementNode.getChildAtIndex(offset);
            if (needsBlockCursor(child)) {
                const sibling = (child as LexicalNode).getPreviousSibling();
                if (sibling === null || needsBlockCursor(sibling)) {
                    isBlockCursor = true;
                    insertBeforeElement = editor.getElementByKey(
                        (child as LexicalNode).__key,
                    );
                }
            }
        }
        if (isBlockCursor) {
            const elementDOM = editor.getElementByKey(
                elementNode.__key,
            ) as HTMLElement;
            if (blockCursorElement === null) {
                editor._blockCursorElement = blockCursorElement =
                    createBlockCursorElement(editor._config);
            }
            rootElement.style.caretColor = "transparent";
            if (insertBeforeElement === null) {
                elementDOM.appendChild(blockCursorElement);
            } else {
                elementDOM.insertBefore(blockCursorElement, insertBeforeElement);
            }
            return;
        }
    }
    // Remove cursor
    if (blockCursorElement !== null) {
        removeDOMBlockCursorElement(blockCursorElement, editor, rootElement);
    }
};

export const getParentElement = (node: Node): HTMLElement | null => {
    const parentElement =
        (node as HTMLSlotElement).assignedSlot || node.parentElement;
    return parentElement !== null && parentElement.nodeType === 11
        ? ((parentElement as unknown as ShadowRoot).host as HTMLElement)
        : parentElement;
};

export const getTextNodeOffset = (
    node: TextNode,
    moveSelectionToEnd: boolean,
): number => {
    return moveSelectionToEnd ? node.getTextContentSize() : 0;
};

export const $getNodeByKey = <T extends LexicalNode>(
    key: NodeKey,
    _editorState?: EditorState,
): T | null => {
    const editorState = _editorState || getActiveEditorState();
    const node = editorState._nodeMap.get(key) as T;
    if (node === undefined) {
        return null;
    }
    return node;
};

export const getNodeFromDOMNode = (
    dom: Node,
    editorState?: EditorState,
): LexicalNode | null => {
    const editor = getActiveEditor();
    // @ts-ignore We intentionally add this to the Node.
    const key = dom[`__lexicalKey_${editor._key}`];
    if (key !== undefined) {
        return $getNodeByKey(key, editorState);
    }
    return null;
};

export const getNodeFromDOM = (dom: Node): null | LexicalNode => {
    const editor = getActiveEditor();
    const nodeKey = getNodeKeyFromDOM(dom, editor);
    if (nodeKey === null) {
        const rootElement = editor.getRootElement();
        if (dom === rootElement) {
            return $getNodeByKey("root");
        }
        return null;
    }
    return $getNodeByKey(nodeKey);
};

const getNodeKeyFromDOM = (
    // Note that node here refers to a DOM Node, not an Lexical Node
    dom: Node,
    editor: LexicalEditor,
): NodeKey | null => {
    let node: Node | null = dom;
    while (node != null) {
        // @ts-ignore We intentionally add this to the Node.
        const key: NodeKey = node[`__lexicalKey_${editor._key}`];
        if (key !== undefined) {
            return key;
        }
        node = getParentElement(node);
    }
    return null;
};

const isDOMNodeLexicalTextNode = (node: Node): node is Text => {
    return node.nodeType === DOM_TEXT_TYPE;
};

export const getDOMTextNode = (element: Node | null): Text | null => {
    let node = element;
    while (node != null) {
        if (isDOMNodeLexicalTextNode(node)) {
            return node;
        }
        node = node.firstChild;
    }
    return null;
};

export const $getNearestNodeFromDOMNode = (
    startingDOM: Node,
    editorState?: EditorState,
): LexicalNode | null => {
    let dom: Node | null = startingDOM;
    while (dom != null) {
        const node = getNodeFromDOMNode(dom, editorState);
        if (node !== null) {
            return node;
        }
        dom = getParentElement(dom);
    }
    return null;
};

export const isSelectionCapturedInDecoratorInput = (anchorDOM: Node): boolean => {
    const activeElement = document.activeElement as HTMLElement;

    if (activeElement === null) {
        return false;
    }
    const nodeName = activeElement.nodeName;

    return (
        $isDecoratorNode($getNearestNodeFromDOMNode(anchorDOM)) &&
        (nodeName === "INPUT" ||
            nodeName === "TEXTAREA" ||
            (activeElement.contentEditable === "true" &&
                // @ts-ignore iternal field
                activeElement.__lexicalEditor == null))
    );
};

export const getNearestEditorFromDOMNode = (
    node: Node | null,
): LexicalEditor | null => {
    let currentNode = node;
    while (currentNode != null) {
        // @ts-expect-error: internal field
        const editor: LexicalEditor = currentNode.__lexicalEditor;
        if (editor != null) {
            return editor;
        }
        currentNode = getParentElement(currentNode);
    }
    return null;
};

export const isSelectionWithinEditor = (
    editor: LexicalEditor,
    anchorDOM: null | Node,
    focusDOM: null | Node,
): boolean => {
    const rootElement = editor.getRootElement();
    try {
        return (
            rootElement !== null &&
            rootElement.contains(anchorDOM) &&
            rootElement.contains(focusDOM) &&
            // Ignore if selection is within nested editor
            anchorDOM !== null &&
            !isSelectionCapturedInDecoratorInput(anchorDOM as Node) &&
            getNearestEditorFromDOMNode(anchorDOM) === editor
        );
    } catch (error) {
        return false;
    }
};

export const getElementByKeyOrThrow = (
    editor: LexicalEditor,
    key: NodeKey,
): HTMLElement => {
    const element = editor._keyToDOMMap.get(key);

    if (element === undefined) {
        throw new Error("Reconciliation: could not find DOM element for node key");
    }

    return element;
};

export const scrollIntoViewIfNeeded = (
    editor: LexicalEditor,
    selectionRect: DOMRect,
    rootElement: HTMLElement,
): void => {
    const doc = rootElement.ownerDocument;
    const defaultView = doc.defaultView;

    if (defaultView === null) {
        return;
    }
    let { top: currentTop, bottom: currentBottom } = selectionRect;
    let targetTop = 0;
    let targetBottom = 0;
    let element: HTMLElement | null = rootElement;

    while (element !== null) {
        const isBodyElement = element === doc.body;
        if (isBodyElement) {
            targetTop = 0;
            targetBottom = getWindow(editor).innerHeight;
        } else {
            const targetRect = element.getBoundingClientRect();
            targetTop = targetRect.top;
            targetBottom = targetRect.bottom;
        }
        let diff = 0;

        if (currentTop < targetTop) {
            diff = -(targetTop - currentTop);
        } else if (currentBottom > targetBottom) {
            diff = currentBottom - targetBottom;
        }

        if (diff !== 0) {
            if (isBodyElement) {
                // Only handles scrolling of Y axis
                defaultView.scrollBy(0, diff);
            } else {
                const scrollTop = element.scrollTop;
                element.scrollTop += diff;
                const yOffset = element.scrollTop - scrollTop;
                currentTop -= yOffset;
                currentBottom -= yOffset;
            }
        }
        if (isBodyElement) {
            break;
        }
        element = getParentElement(element);
    }
};

export const isTab = (
    keyCode: number,
    altKey: boolean,
    ctrlKey: boolean,
    metaKey: boolean,
): boolean => {
    return keyCode === 9 && !altKey && !ctrlKey && !metaKey;
};

export const isBold = (
    keyCode: number,
    altKey: boolean,
    metaKey: boolean,
    ctrlKey: boolean,
): boolean => {
    return keyCode === 66 && !altKey && controlOrMeta(metaKey, ctrlKey);
};

export const isItalic = (
    keyCode: number,
    altKey: boolean,
    metaKey: boolean,
    ctrlKey: boolean,
): boolean => {
    return keyCode === 73 && !altKey && controlOrMeta(metaKey, ctrlKey);
};

export const isUnderline = (
    keyCode: number,
    altKey: boolean,
    metaKey: boolean,
    ctrlKey: boolean,
): boolean => {
    return keyCode === 85 && !altKey && controlOrMeta(metaKey, ctrlKey);
};

export const isParagraph = (keyCode: number, shiftKey: boolean): boolean => {
    return isReturn(keyCode) && !shiftKey;
};

export const isLineBreak = (keyCode: number, shiftKey: boolean): boolean => {
    return isReturn(keyCode) && shiftKey;
};

// Inserts a new line after the selection

export const isOpenLineBreak = (keyCode: number, ctrlKey: boolean): boolean => {
    // 79 = KeyO
    return IS_APPLE && ctrlKey && keyCode === 79;
};

export const isDeleteWordBackward = (
    keyCode: number,
    altKey: boolean,
    ctrlKey: boolean,
): boolean => {
    return isBackspace(keyCode) && (IS_APPLE ? altKey : ctrlKey);
};

export const isDeleteWordForward = (
    keyCode: number,
    altKey: boolean,
    ctrlKey: boolean,
): boolean => {
    return isDelete(keyCode) && (IS_APPLE ? altKey : ctrlKey);
};

export const isDeleteLineBackward = (
    keyCode: number,
    metaKey: boolean,
): boolean => {
    return IS_APPLE && metaKey && isBackspace(keyCode);
};

export const isDeleteLineForward = (
    keyCode: number,
    metaKey: boolean,
): boolean => {
    return IS_APPLE && metaKey && isDelete(keyCode);
};

export const isDeleteBackward = (
    keyCode: number,
    altKey: boolean,
    metaKey: boolean,
    ctrlKey: boolean,
): boolean => {
    if (IS_APPLE) {
        if (altKey || metaKey) {
            return false;
        }
        return isBackspace(keyCode) || (keyCode === 72 && ctrlKey);
    }
    if (ctrlKey || altKey || metaKey) {
        return false;
    }
    return isBackspace(keyCode);
};

export const isDeleteForward = (
    keyCode: number,
    ctrlKey: boolean,
    shiftKey: boolean,
    altKey: boolean,
    metaKey: boolean,
): boolean => {
    if (IS_APPLE) {
        if (shiftKey || altKey || metaKey) {
            return false;
        }
        return isDelete(keyCode) || (keyCode === 68 && ctrlKey);
    }
    if (ctrlKey || altKey || metaKey) {
        return false;
    }
    return isDelete(keyCode);
};

export const isUndo = (
    keyCode: number,
    shiftKey: boolean,
    metaKey: boolean,
    ctrlKey: boolean,
): boolean => {
    return keyCode === 90 && !shiftKey && controlOrMeta(metaKey, ctrlKey);
};

export const isRedo = (
    keyCode: number,
    shiftKey: boolean,
    metaKey: boolean,
    ctrlKey: boolean,
): boolean => {
    if (IS_APPLE) {
        return keyCode === 90 && metaKey && shiftKey;
    }
    return (keyCode === 89 && ctrlKey) || (keyCode === 90 && ctrlKey && shiftKey);
};

export const isCopy = (
    keyCode: number,
    shiftKey: boolean,
    metaKey: boolean,
    ctrlKey: boolean,
): boolean => {
    if (shiftKey) {
        return false;
    }
    if (keyCode === 67) {
        return IS_APPLE ? metaKey : ctrlKey;
    }

    return false;
};

export const isCut = (
    keyCode: number,
    shiftKey: boolean,
    metaKey: boolean,
    ctrlKey: boolean,
): boolean => {
    if (shiftKey) {
        return false;
    }
    if (keyCode === 88) {
        return IS_APPLE ? metaKey : ctrlKey;
    }

    return false;
};

const isArrowLeft = (keyCode: number): boolean => {
    return keyCode === 37;
};

const isArrowRight = (keyCode: number): boolean => {
    return keyCode === 39;
};

const isArrowUp = (keyCode: number): boolean => {
    return keyCode === 38;
};

const isArrowDown = (keyCode: number): boolean => {
    return keyCode === 40;
};

export const isMoveBackward = (
    keyCode: number,
    ctrlKey: boolean,
    altKey: boolean,
    metaKey: boolean,
): boolean => {
    return isArrowLeft(keyCode) && !ctrlKey && !metaKey && !altKey;
};

export const isMoveToStart = (
    keyCode: number,
    ctrlKey: boolean,
    shiftKey: boolean,
    altKey: boolean,
    metaKey: boolean,
): boolean => {
    return isArrowLeft(keyCode) && !altKey && !shiftKey && (ctrlKey || metaKey);
};

export const isMoveForward = (
    keyCode: number,
    ctrlKey: boolean,
    altKey: boolean,
    metaKey: boolean,
): boolean => {
    return isArrowRight(keyCode) && !ctrlKey && !metaKey && !altKey;
};

export const isMoveToEnd = (
    keyCode: number,
    ctrlKey: boolean,
    shiftKey: boolean,
    altKey: boolean,
    metaKey: boolean,
): boolean => {
    return isArrowRight(keyCode) && !altKey && !shiftKey && (ctrlKey || metaKey);
};

export const isMoveUp = (
    keyCode: number,
    ctrlKey: boolean,
    metaKey: boolean,
): boolean => {
    return isArrowUp(keyCode) && !ctrlKey && !metaKey;
};

export const isMoveDown = (
    keyCode: number,
    ctrlKey: boolean,
    metaKey: boolean,
): boolean => {
    return isArrowDown(keyCode) && !ctrlKey && !metaKey;
};

export const isModifier = (
    ctrlKey: boolean,
    shiftKey: boolean,
    altKey: boolean,
    metaKey: boolean,
): boolean => {
    return ctrlKey || shiftKey || altKey || metaKey;
};

export const isSpace = (keyCode: number): boolean => {
    return keyCode === 32;
};

export const controlOrMeta = (metaKey: boolean, ctrlKey: boolean): boolean => {
    if (IS_APPLE) {
        return metaKey;
    }
    return ctrlKey;
};

export const isReturn = (keyCode: number): boolean => {
    return keyCode === 13;
};

export const isBackspace = (keyCode: number): boolean => {
    return keyCode === 8;
};

export const isEscape = (keyCode: number): boolean => {
    return keyCode === 27;
};

export const isDelete = (keyCode: number): boolean => {
    return keyCode === 46;
};

export const isSelectAll = (
    keyCode: number,
    metaKey: boolean,
    ctrlKey: boolean,
): boolean => {
    return keyCode === 65 && controlOrMeta(metaKey, ctrlKey);
};

export const removeFromParent = (node: LexicalNode) => {
    const oldParent = node.getParent();
    if (oldParent !== null) {
        const writableNode = node.getWritable();
        const writableParent = oldParent.getWritable();
        const prevSibling = node.getPreviousSibling();
        const nextSibling = node.getNextSibling();
        // TODO: this function duplicates a bunch of operations, can be simplified.
        if (prevSibling === null) {
            if (nextSibling !== null) {
                const writableNextSibling = nextSibling.getWritable();
                writableParent.__first = nextSibling.__key;
                writableNextSibling.__prev = null;
            } else {
                writableParent.__first = null;
            }
        } else {
            const writablePrevSibling = prevSibling.getWritable();
            if (nextSibling !== null) {
                const writableNextSibling = nextSibling.getWritable();
                writableNextSibling.__prev = writablePrevSibling.__key;
                writablePrevSibling.__next = writableNextSibling.__key;
            } else {
                writablePrevSibling.__next = null;
            }
            writableNode.__prev = null;
        }
        if (nextSibling === null) {
            if (prevSibling !== null) {
                const writablePrevSibling = prevSibling.getWritable();
                writableParent.__last = prevSibling.__key;
                writablePrevSibling.__next = null;
            } else {
                writableParent.__last = null;
            }
        } else {
            const writableNextSibling = nextSibling.getWritable();
            if (prevSibling !== null) {
                const writablePrevSibling = prevSibling.getWritable();
                writablePrevSibling.__next = writableNextSibling.__key;
                writableNextSibling.__prev = writablePrevSibling.__key;
            } else {
                writableNextSibling.__prev = null;
            }
            writableNode.__next = null;
        }
        writableParent.__size--;
        writableNode.__parent = null;
    }
};

/**
 * Takes an HTML element and adds the classNames passed within an array,
 * ignoring any non-string types. A space can be used to add multiple classes
 * eg. addClassNamesToElement(element, ['element-inner active', true, null])
 * will add both 'element-inner' and 'active' as classes to that element.
 * @param element - The element in which the classes are added
 * @param classNames - An array defining the class names to add to the element
 */
export const addClassNamesToElement = (
    element: HTMLElement,
    ...classNames: Array<typeof undefined | boolean | null | string>
) => {
    const classesToAdd = normalizeClassNames(...classNames);
    if (classesToAdd.length > 0) {
        element.classList.add(...classesToAdd);
    }
};

export const getAnchorTextFromDOM = (anchorNode: Node): null | string => {
    if (anchorNode.nodeType === DOM_TEXT_TYPE) {
        return anchorNode.nodeValue;
    }
    return null;
};

export const $updateSelectedTextFromDOM = (
    isCompositionEnd: boolean,
    editor: LexicalEditor,
    data?: string,
) => {
    // Update the text content with the latest composition text
    const domSelection = getDOMSelection(editor._window);
    if (domSelection === null) {
        return;
    }
    const anchorNode = domSelection.anchorNode;
    let { anchorOffset, focusOffset } = domSelection;
    if (anchorNode !== null) {
        let textContent = getAnchorTextFromDOM(anchorNode);
        const node = $getNearestNodeFromDOMNode(anchorNode);
        if (textContent !== null && $isTextNode(node)) {
            // Data is intentionally truthy, as we check for boolean, null and empty string.
            if (textContent === COMPOSITION_SUFFIX && data) {
                const offset = data.length;
                textContent = data;
                anchorOffset = offset;
                focusOffset = offset;
            }

            if (textContent !== null) {
                $updateTextNodeFromDOMContent(
                    node,
                    textContent,
                    anchorOffset,
                    focusOffset,
                    isCompositionEnd,
                );
            }
        }
    }
};

/**
 * Checks if a string contains a grapheme, which is a single 
 * unit of a writing system that is treated as a single character (eg. a letter, an ideogram, etc.)
 */
export const doesContainGrapheme = (str: string): boolean => {
    return /[\uD800-\uDBFF][\uDC00-\uDFFF]/g.test(str);
};

export const $isTokenOrSegmented = (node: TextNode): boolean => {
    return node.isToken() || node.isSegmented();
};

/**
 * Returns a function that will execute all functions passed when called. It is generally used
 * to register multiple lexical listeners and then tear them down with a single function call, such
 * as React's useEffect hook.
 * @example
 * ```ts
 * useEffect(() => {
 *   return mergeRegister(
 *     editor.registerCommand(...registerCommand1 logic),
 *     editor.registerCommand(...registerCommand2 logic),
 *     editor.registerCommand(...registerCommand3 logic)
 *   )
 * }, [editor])
 * ```
 * In this case, useEffect is returning the function returned by mergeRegister as a cleanup
 * function to be executed after either the useEffect runs again (due to one of its dependencies
 * updating) or the component it resides in unmounts.
 * Note the functions don't neccesarily need to be in an array as all arguements
 * are considered to be the func argument and spread from there.
 * @param func - An array of functions meant to be executed by the returned function.
 * @returns the function which executes all the passed register command functions.
 */
export const mergeRegister = (...func: Array<(() => void)>): (() => void) => {
    return () => {
        func.forEach((f) => f());
    };
};

export const $getAncestor = <NodeType extends LexicalNode = LexicalNode>(
    node: LexicalNode,
    predicate: (ancestor: LexicalNode) => ancestor is NodeType,
) => {
    let parent = node;
    while (parent !== null && parent.getParent() !== null && !predicate(parent)) {
        parent = parent.getParentOrThrow();
    }
    return predicate(parent) ? parent : null;
};

export const $getNearestRootOrShadowRoot = (
    node: LexicalNode,
): RootNode | ElementNode => {
    let parent = node.getParentOrThrow();
    while (parent !== null) {
        if ($isRootOrShadowRoot(parent)) {
            return parent;
        }
        parent = parent.getParentOrThrow();
    }
    return parent;
};

export const $hasAncestor = (
    child: LexicalNode,
    targetNode: LexicalNode,
): boolean => {
    let parent = child.getParent();
    while (parent !== null) {
        if (parent.is(targetNode)) {
            return true;
        }
        parent = parent.getParent();
    }
    return false;
};

export const $maybeMoveChildrenSelectionToParent = (
    parentNode: LexicalNode,
): BaseSelection | null => {
    const selection = $getSelection();
    if (!$isRangeSelection(selection) || !$isElementNode(parentNode)) {
        return selection;
    }
    const { anchor, focus } = selection;
    const anchorNode = anchor.getNode();
    const focusNode = focus.getNode();
    if ($hasAncestor(anchorNode, parentNode)) {
        anchor.set(parentNode.__key, 0, "element");
    }
    if ($hasAncestor(focusNode, parentNode)) {
        focus.set(parentNode.__key, 0, "element");
    }
    return selection;
};

const resolveElement = (
    element: ElementNode,
    isBackward: boolean,
    focusOffset: number,
): LexicalNode | null => {
    const parent = element.getParent();
    let offset = focusOffset;
    let block = element;
    if (parent !== null) {
        if (isBackward && focusOffset === 0) {
            offset = block.getIndexWithinParent();
            block = parent;
        } else if (!isBackward && focusOffset === block.getChildrenSize()) {
            offset = block.getIndexWithinParent() + 1;
            block = parent;
        }
    }
    return block.getChildAtIndex(isBackward ? offset - 1 : offset);
};

export const $getAdjacentNode = (
    focus: PointType,
    isBackward: boolean,
): LexicalNode | null => {
    const focusOffset = focus.offset;
    if (focus.type === "element") {
        const block = focus.getNode();
        return resolveElement(block, isBackward, focusOffset);
    } else {
        const focusNode = focus.getNode();
        if (
            (isBackward && focusOffset === 0) ||
            (!isBackward && focusOffset === focusNode.getTextContentSize())
        ) {
            const possibleNode = isBackward
                ? focusNode.getPreviousSibling()
                : focusNode.getNextSibling();
            if (possibleNode === null) {
                return resolveElement(
                    focusNode.getParentOrThrow(),
                    isBackward,
                    focusNode.getIndexWithinParent() + (isBackward ? 0 : 1),
                );
            }
            return possibleNode;
        }
    }
    return null;
};

const internalMarkParentElementsAsDirty = (
    parentKey: NodeKey,
    nodeMap: NodeMap,
    dirtyElements: Map<NodeKey, IntentionallyMarkedAsDirtyElement>,
): void => {
    let nextParentKey: string | null = parentKey;
    while (nextParentKey !== null) {
        if (dirtyElements.has(nextParentKey)) {
            return;
        }
        const node = nodeMap.get(nextParentKey);
        if (node === undefined) {
            break;
        }
        dirtyElements.set(nextParentKey, false);
        nextParentKey = node.__parent;
    }
};

// Never use this function directly! It will break
// the cloning heuristic. Instead use node.getWritable().
export const internalMarkNodeAsDirty = (node: LexicalNode): void => {
    errorOnInfiniteTransforms();
    const latest = node.getLatest();
    const parent = latest.__parent;
    const editorState = getActiveEditorState();
    const editor = getActiveEditor();
    const nodeMap = editorState._nodeMap;
    const dirtyElements = editor._dirtyElements;
    if (parent !== null) {
        internalMarkParentElementsAsDirty(parent, nodeMap, dirtyElements);
    }
    const key = latest.__key;
    editor._dirtyType = HAS_DIRTY_NODES;
    if ($isElementNode(node)) {
        dirtyElements.set(key, true);
    } else {
        // TODO split internally MarkNodeAsDirty into two dedicated Element/leave functions
        editor._dirtyLeaves.add(key);
    }
};

export const internalMarkSiblingsAsDirty = (node: LexicalNode) => {
    const previousNode = node.getPreviousSibling();
    const nextNode = node.getNextSibling();
    if (previousNode !== null) {
        internalMarkNodeAsDirty(previousNode);
    }
    if (nextNode !== null) {
        internalMarkNodeAsDirty(nextNode);
    }
};

/**
 * Wraps a node into a ListItemNode.
 * @param node - The node to be wrapped into a ListItemNode
 * @returns The ListItemNode which the passed node is wrapped in.
 */
export const wrapInListItem = (node: LexicalNode): ListItemNode => {
    const listItemWrapper = $createListItemNode();
    return listItemWrapper.append(node);
};

let keyCounter = 1;
export const generateRandomKey = (): string => {
    return "key-" + keyCounter++;
};

export const $setNodeKey = (
    node: LexicalNode,
    existingKey: NodeKey | null | undefined,
): void => {
    if (existingKey != null) {
        node.__key = existingKey;
        return;
    }
    errorOnReadOnly();
    errorOnInfiniteTransforms();
    const editor = getActiveEditor();
    const editorState = getActiveEditorState();
    const key = generateRandomKey();
    editorState._nodeMap.set(key, node);
    // TODO Split this function into leaf/element
    if ($isElementNode(node)) {
        editor._dirtyElements.set(key, true);
    } else {
        editor._dirtyLeaves.add(key);
    }
    editor._cloneNotNeeded.add(key);
    editor._dirtyType = HAS_DIRTY_NODES;
    node.__key = key;
};

export const errorOnInsertTextNodeOnRoot = (
    node: LexicalNode,
    insertNode: LexicalNode,
): void => {
    const parentNode = node.getParent();
    if (
        $isRootNode(parentNode) &&
        !$isElementNode(insertNode) &&
        !$isDecoratorNode(insertNode)
    ) {
        throw new Error("Only element or decorator nodes can be inserted in to the root node");
    }
};

/**
 * Takes an HTML element and removes the classNames passed within an array,
 * ignoring any non-string types. A space can be used to remove multiple classes
 * eg. removeClassNamesFromElement(element, ['active small', true, null])
 * will remove both the 'active' and 'small' classes from that element.
 * @param element - The element in which the classes are removed
 * @param classNames - An array defining the class names to remove from the element
 */
export const removeClassNamesFromElement = (
    element: HTMLElement,
    ...classNames: Array<typeof undefined | boolean | null | string>
): void => {
    const classesToRemove = normalizeClassNames(...classNames);
    if (classesToRemove.length > 0) {
        element.classList.remove(...classesToRemove);
    }
};

export const $computeTableMap = (
    grid: TableNode,
    cellA: TableCellNode,
    cellB: TableCellNode,
): [TableMapType, TableMapValueType, TableMapValueType] => {
    const tableMap: TableMapType = [];
    let cellAValue: null | TableMapValueType = null;
    let cellBValue: null | TableMapValueType = null;
    function write(startRow: number, startColumn: number, cell: TableCellNode) {
        const value = {
            cell,
            startColumn,
            startRow,
        };
        const rowSpan = cell.__rowSpan;
        const colSpan = cell.__colSpan;
        for (let i = 0; i < rowSpan; i++) {
            if (tableMap[startRow + i] === undefined) {
                tableMap[startRow + i] = [];
            }
            for (let j = 0; j < colSpan; j++) {
                tableMap[startRow + i][startColumn + j] = value;
            }
        }
        if (cellA.is(cell)) {
            cellAValue = value;
        }
        if (cellB.is(cell)) {
            cellBValue = value;
        }
    }
    function isEmpty(row: number, column: number) {
        return tableMap[row] === undefined || tableMap[row][column] === undefined;
    }

    const gridChildren = grid.getChildren();
    for (let i = 0; i < gridChildren.length; i++) {
        const row = gridChildren[i];
        if (!$isTableRowNode(row)) {
            throw new Error("Expected GridNode children to be TableRowNode");
        }
        const rowChildren = row.getChildren();
        let j = 0;
        for (const cell of rowChildren) {
            if (!$isTableCellNode(cell)) {
                throw new Error("Expected TableRowNode children to be TableCellNode");
            }
            while (!isEmpty(i, j)) {
                j++;
            }
            write(i, j, cell);
            j += cell.__colSpan;
        }
    }
    if (cellAValue === null) {
        throw new Error("Anchor not found in Grid");
    }
    if (cellBValue === null) {
        throw new Error("Focus not found in Grid");
    }
    return [tableMap, cellAValue, cellBValue];
};

/**
 * Calculates the zoom level of an element as a result of using
 * css zoom property.
 * @param element
 */
export const calculateZoomLevel = (element: Element | null): number => {
    if (IS_FIREFOX) {
        return 1;
    }
    let zoom = 1;
    while (element) {
        zoom *= Number(window.getComputedStyle(element).getPropertyValue("zoom"));
        element = element.parentElement;
    }
    return zoom;
};

const $previousSiblingDoesNotAcceptText = (node: TextNode): boolean => {
    const previousSibling = node.getPreviousSibling();

    return (
        ($isTextNode(previousSibling) ||
            ($isElementNode(previousSibling) && previousSibling.isInline())) &&
        !previousSibling.canInsertTextAfter()
    );
};

// This function is connected to $shouldPreventDefaultAndInsertText and determines whether the
// TextNode boundaries are writable or we should use the previous/next sibling instead. For example,
// in the case of a LinkNode, boundaries are not writable.
export const $shouldInsertTextAfterOrBeforeTextNode = (
    selection: RangeSelection,
    node: TextNode,
): boolean => {
    if (node.isSegmented()) {
        return true;
    }
    if (!selection.isCollapsed()) {
        return false;
    }
    const offset = selection.anchor.offset;
    const parent = node.getParentOrThrow();
    const isToken = node.isToken();
    if (offset === 0) {
        return (
            !node.canInsertTextBefore() ||
            (!parent.canInsertTextBefore() && !node.isComposing()) ||
            isToken ||
            $previousSiblingDoesNotAcceptText(node)
        );
    } else if (offset === node.getTextContentSize()) {
        return (
            !node.canInsertTextAfter() ||
            (!parent.canInsertTextAfter() && !node.isComposing()) ||
            isToken
        );
    } else {
        return false;
    }
};

export const $textContentRequiresDoubleLinebreakAtEnd = (
    node: ElementNode,
): boolean => {
    return !$isRootNode(node) && !node.isLastChild() && !node.isInline();
};

export const setMutatedNode = (
    mutatedNodes: MutatedNodes,
    registeredNodes: RegisteredNodes,
    mutationListeners: MutationListeners,
    node: LexicalNode,
    mutation: NodeMutation,
) => {
    if (mutationListeners.size === 0) {
        return;
    }
    const nodeType = node.__type;
    const nodeKey = node.__key;
    const registeredNode = registeredNodes.get(nodeType);
    if (registeredNode === undefined) {
        throw new Error(`Type ${nodeType} not in registeredNodes`);
    }
    const klass = registeredNode.klass;
    let mutatedNodesByType = mutatedNodes.get(klass);
    if (mutatedNodesByType === undefined) {
        mutatedNodesByType = new Map();
        mutatedNodes.set(klass, mutatedNodesByType);
    }
    const prevMutation = mutatedNodesByType.get(nodeKey);
    // If the node has already been "destroyed", yet we are
    // re-making it, then this means a move likely happened.
    // We should change the mutation to be that of "updated"
    // instead.
    const isMove = prevMutation === "destroyed" && mutation === "created";
    if (prevMutation === undefined || isMove) {
        mutatedNodesByType.set(nodeKey, isMove ? "updated" : mutation);
    }
};

export const getTextDirection = (text: string): "ltr" | "rtl" | null => {
    if (RTL_REGEX.test(text)) {
        return "rtl";
    }
    if (LTR_REGEX.test(text)) {
        return "ltr";
    }
    return null;
};

export const cloneDecorators = (
    editor: LexicalEditor,
): Record<NodeKey, unknown> => {
    const currentDecorators = editor._decorators;
    const pendingDecorators = Object.assign({}, currentDecorators);
    editor._pendingDecorators = pendingDecorators;
    return pendingDecorators;
};

export const $isTargetWithinDecorator = (target: HTMLElement): boolean => {
    const node = $getNearestNodeFromDOMNode(target);
    return $isDecoratorNode(node);
};

/**
 * Returns the element node of the nearest ancestor, otherwise throws an error.
 * @param startNode - The starting node of the search
 * @returns The ancestor node found
 */
export const $getNearestBlockElementAncestorOrThrow = (
    startNode: LexicalNode,
): ElementNode => {
    const blockNode = $findMatchingParent(
        startNode,
        (node) => $isElementNode(node) && !node.isInline(),
    );
    if (!$isElementNode(blockNode)) {
        throw new Error(`Expected node ${startNode.__key} to have closest block element node.`);
    }
    return blockNode;
};

export const handleIndentAndOutdent = (
    indentOrOutdent: (block: ElementNode) => void,
): boolean => {
    const selection = $getSelection();
    if (!$isRangeSelection(selection)) {
        return false;
    }
    const alreadyHandled = new Set();
    const nodes = selection.getNodes();
    for (let i = 0; i < nodes.length; i++) {
        const node = nodes[i];
        const key = node.getKey();
        if (alreadyHandled.has(key)) {
            continue;
        }
        const parentBlock = $getNearestBlockElementAncestorOrThrow(node);
        const parentKey = parentBlock.getKey();
        if (parentBlock.canIndent() && !alreadyHandled.has(parentKey)) {
            alreadyHandled.add(parentKey);
            indentOrOutdent(parentBlock);
        }
    }
    return alreadyHandled.size > 0;
};

export const $isSelectionAtEndOfRoot = (selection: RangeSelection) => {
    const focus = selection.focus;
    return focus.key === "root" && focus.offset === $getRoot().getChildrenSize();
};

// Clipboard may contain files that we aren't allowed to read. While the event is arguably useless,
// in certain occasions, we want to know whether it was a file transfer, as opposed to text. We
// control this with the first boolean flag.
export const eventFiles = (
    event: DragEvent | PasteCommandType,
): [boolean, Array<File>, boolean] => {
    let dataTransfer: null | DataTransfer = null;
    if (objectKlassEquals(event, DragEvent)) {
        dataTransfer = (event as DragEvent).dataTransfer;
    } else if (objectKlassEquals(event, ClipboardEvent)) {
        dataTransfer = (event as ClipboardEvent).clipboardData;
    }

    if (dataTransfer === null) {
        return [false, [], false];
    }

    const types = dataTransfer.types;
    const hasFiles = types.includes("Files");
    const hasContent =
        types.includes("text/html") || types.includes("text/plain");
    return [hasFiles, Array.from(dataTransfer.files), hasContent];
};

/**
 * @param object = The instance of the type
 * @param objectClass = The class of the type
 * @returns Whether the object is has the same Klass of the objectClass, ignoring the difference across window (e.g. different iframs)
 */
export const objectKlassEquals = <T>(
    object: unknown,
    objectClass: ObjectKlass<T>,
): boolean => {
    return object !== null
        ? Object.getPrototypeOf(object).constructor.name === objectClass.name
        : false;
};

/**
 * Creates an object containing all the styles and their values provided in the CSS string.
 * @param css - The CSS string of styles and their values.
 * @returns The styleObject containing all the styles and their values.
 */
export const getStyleObjectFromRawCSS = (css: string): Record<string, string> => {
    const styleObject: Record<string, string> = {};
    const styles = css.split(";");

    for (const style of styles) {
        if (style !== "") {
            const [key, value] = style.split(/:([^]+)/); // split on first colon
            if (key && value) {
                styleObject[key.trim()] = value.trim();
            }
        }
    }

    return styleObject;
};

export const $isLeafNode = (
    node: LexicalNode | null | undefined,
): node is TextNode | LineBreakNode | DecoratorNode<unknown> => {
    return $isTextNode(node) || $isLineBreakNode(node) || $isDecoratorNode(node);
};

/**
 * Finds the nearest ancestral ListNode and returns it, throws an invariant if listItem is not a ListItemNode.
 * @param listItem - The node to be checked.
 * @returns The ListNode found.
 */
export const $getTopListNode = (listItem: LexicalNode): ListNode => {
    let list = listItem.getParent<ListNode>();

    if (!$isListNode(list)) {
        throw new Error("A ListItemNode must have a ListNode for a parent.");
    }

    let parent: ListNode | null = list;

    while (parent !== null) {
        parent = parent.getParent();

        if ($isListNode(parent)) {
            list = parent;
        }
    }

    return list;
};

/**
 * Checks to see if the passed node is a ListItemNode and has a ListNode as a child.
 * @param node - The node to be checked.
 * @returns true if the node is a ListItemNode and has a ListNode child, false otherwise.
 */
export const isNestedListNode = (
    node: LexicalNode | null | undefined,
): node is Spread<
    { getFirstChild(): ListNode },
    ListItemNode
> => {
    return $isListItemNode(node) && $isListNode(node.getFirstChild());
};

/**
 * A recursive Depth-First Search (Postorder Traversal) that finds all of a node's children
 * that are of type ListItemNode and returns them in an array.
 * @param node - The ListNode to start the search.
 * @returns An array containing all nodes of type ListItemNode found.
 */
// This should probably be $getAllChildrenOfType
export const $getAllListItems = (node: ListNode): Array<ListItemNode> => {
    let listItemNodes: Array<ListItemNode> = [];
    const listChildren: Array<ListItemNode> = node
        .getChildren()
        .filter($isListItemNode);

    for (let i = 0; i < listChildren.length; i++) {
        const listItemNode = listChildren[i];
        const firstChild = listItemNode.getFirstChild();

        if ($isListNode(firstChild)) {
            listItemNodes = listItemNodes.concat($getAllListItems(firstChild));
        } else {
            listItemNodes.push(listItemNode);
        }
    }

    return listItemNodes;
};

/**
 * Takes a deeply nested ListNode or ListItemNode and traverses up the branch to delete the first
 * ancestral ListNode (which could be the root ListNode) or ListItemNode with siblings, essentially
 * bringing the deeply nested node up the branch once. Would remove sublist if it has siblings.
 * Should not break ListItem -> List -> ListItem chain as empty List/ItemNodes should be removed on .remove().
 * @param sublist - The nested ListNode or ListItemNode to be brought up the branch.
 */
export const $removeHighestEmptyListParent = (
    sublist: ListItemNode | ListNode,
) => {
    // Nodes may be repeatedly indented, to create deeply nested lists that each
    // contain just one bullet.
    // Our goal is to remove these (empty) deeply nested lists. The easiest
    // way to do that is crawl back up the tree until we find a node that has siblings
    // (e.g. is actually part of the list contents) and delete that, or delete
    // the root of the list (if no list nodes have siblings.)
    let emptyListPtr = sublist;

    while (
        emptyListPtr.getNextSibling() == null &&
        emptyListPtr.getPreviousSibling() == null
    ) {
        const parent = emptyListPtr.getParent<ListItemNode | ListNode>();

        if (
            parent == null ||
            !($isListItemNode(emptyListPtr) || $isListNode(emptyListPtr))
        ) {
            break;
        }

        emptyListPtr = parent;
    }

    emptyListPtr.remove();
};

/**
 * Checks the depth of listNode from the root node.
 * @param listNode - The ListNode to be checked.
 * @returns The depth of the ListNode.
 */
export const $getListDepth = (listNode: ListNode): number => {
    let depth = 1;
    let parent = listNode.getParent();

    while (parent != null) {
        if ($isListItemNode(parent)) {
            const parentList = parent.getParent();

            if ($isListNode(parentList)) {
                depth++;
                parent = parentList.getParent();
                continue;
            }
            throw new Error("A ListItemNode must have a ListNode for a parent.");
        }

        return depth;
    }

    return depth;
};

const $normalizePoint = (point: PointType): void => {
    while (point.type === "element") {
        const node = point.getNode();
        const offset = point.offset;
        let nextNode;
        let nextOffsetAtEnd;
        if (offset === node.getChildrenSize()) {
            nextNode = node.getChildAtIndex(offset - 1);
            nextOffsetAtEnd = true;
        } else {
            nextNode = node.getChildAtIndex(offset);
            nextOffsetAtEnd = false;
        }
        if ($isTextNode(nextNode)) {
            point.set(
                nextNode.__key,
                nextOffsetAtEnd ? nextNode.getTextContentSize() : 0,
                "text",
            );
            break;
        } else if (!$isElementNode(nextNode)) {
            break;
        }
        point.set(
            nextNode.__key,
            nextOffsetAtEnd ? nextNode.getChildrenSize() : 0,
            "element",
        );
    }
};

export const $normalizeSelection = (selection: RangeSelection): RangeSelection => {
    $normalizePoint(selection.anchor);
    $normalizePoint(selection.focus);
    return selection;
};

export const $selectAll = (): void => {
    const root = $getRoot();
    const selection = root.select(0, root.getChildrenSize());
    $setSelection($normalizeSelection(selection));
};

export const append = (node: ElementNode, nodesToAppend: LexicalNode[]) => {
    node.splice(node.getChildrenSize(), 0, nodesToAppend);
};

export const onCutForRichText = async (
    event: CommandPayloadType<typeof CUT_COMMAND>,
    editor: LexicalEditor,
): Promise<void> => {
    await copyToClipboard(
        editor,
        objectKlassEquals(event, ClipboardEvent) ? (event as ClipboardEvent) : null,
    );
    editor.update(() => {
        const selection = $getSelection();
        if ($isRangeSelection(selection)) {
            selection.removeText();
        } else if ($isNodeSelection(selection)) {
            selection.getNodes().forEach((node) => node.remove());
        }
    });
};

export const onPasteForRichText = (
    event: CommandPayloadType<typeof PASTE_COMMAND>,
    editor: LexicalEditor,
): void => {
    event.preventDefault();
    editor.update(
        () => {
            const selection = $getSelection();
            const clipboardData =
                objectKlassEquals(event, InputEvent) ||
                    objectKlassEquals(event, KeyboardEvent)
                    ? null
                    : (event as ClipboardEvent).clipboardData;
            if (clipboardData != null && selection !== null) {
                $insertDataTransferForRichText(clipboardData, selection, editor);
            }
        },
        {
            tag: "paste",
        },
    );
};

export function $copyNode<T extends LexicalNode>(node: T): T {
    const copy = node.constructor.clone(node);
    $setNodeKey(copy, null);
    return copy as T;
}

export function $splitNode(
    node: ElementNode,
    offset: number,
): [ElementNode | null, ElementNode] {
    let startNode = node.getChildAtIndex(offset);
    if (startNode == null) {
        startNode = node;
    }

    if ($isRootOrShadowRoot(node)) {
        throw new Error("Can not call $splitNode() on root element");
    }

    const recurse = <T extends LexicalNode>(
        currentNode: T,
    ): [ElementNode, ElementNode, T] => {
        const parent = currentNode.getParentOrThrow();
        const isParentRoot = $isRootOrShadowRoot(parent);
        // The node we start split from (leaf) is moved, but its recursive
        // parents are copied to create separate tree
        const nodeToMove =
            currentNode === startNode && !isParentRoot
                ? currentNode
                : $copyNode(currentNode);

        if (isParentRoot) {
            if (!$isElementNode(currentNode) || !$isElementNode(nodeToMove)) {
                throw new Error("Children of a root must be ElementNode");
            }

            currentNode.insertAfter(nodeToMove);
            return [currentNode, nodeToMove, nodeToMove];
        } else {
            const [leftTree, rightTree, newParent] = recurse(parent);
            const nextSiblings = currentNode.getNextSiblings();

            newParent.append(nodeToMove, ...nextSiblings);
            return [leftTree, rightTree, nodeToMove];
        }
    };

    const [leftTree, rightTree] = recurse(startNode);

    return [leftTree, rightTree];
}

/**
 * If the selected insertion area is the root/shadow root node (see {@link lexical!$isRootOrShadowRoot}),
 * the node will be appended there, otherwise, it will be inserted before the insertion area.
 * If there is no selection where the node is to be inserted, it will be appended after any current nodes
 * within the tree, as a child of the root node. A paragraph node will then be added after the inserted node and selected.
 * @param node - The node to be inserted
 * @returns The node after its insertion
 */
export const $insertNodeToNearestRoot = <T extends LexicalNode>(node: T): T => {
    const selection = $getSelection() || $getPreviousSelection();

    if ($isRangeSelection(selection)) {
        const { focus } = selection;
        const focusNode = focus.getNode();
        const focusOffset = focus.offset;

        if ($isRootOrShadowRoot(focusNode)) {
            const focusChild = focusNode.getChildAtIndex(focusOffset);
            if (focusChild == null) {
                focusNode.append(node);
            } else {
                focusChild.insertBefore(node);
            }
            node.selectNext();
        } else {
            let splitNode: ElementNode;
            let splitOffset: number;
            if ($isTextNode(focusNode)) {
                splitNode = focusNode.getParentOrThrow();
                splitOffset = focusNode.getIndexWithinParent();
                if (focusOffset > 0) {
                    splitOffset += 1;
                    focusNode.splitText(focusOffset);
                }
            } else {
                splitNode = focusNode;
                splitOffset = focusOffset;
            }
            const [, rightTree] = $splitNode(splitNode, splitOffset);
            rightTree.insertBefore(node);
            rightTree.selectStart();
        }
    } else {
        if (selection != null) {
            const nodes = selection.getNodes();
            nodes[nodes.length - 1].getTopLevelElementOrThrow().insertAfter(node);
        } else {
            const root = $getRoot();
            root.append(node);
        }
        const paragraphNode = $createParagraphNode();
        node.insertAfter(paragraphNode);
        paragraphNode.select();
    }
    return node.getLatest();
};

function getConversionFunction(
    domNode: Node,
    editor: LexicalEditor,
): DOMConversionFn | null {
    const { nodeName } = domNode;

    const cachedConversions = editor._htmlConversions.get(nodeName.toLowerCase());

    let currentConversion: DOMConversion | null = null;

    if (cachedConversions !== undefined) {
        for (const cachedConversion of cachedConversions) {
            const domConversion = cachedConversion(domNode);

            if (
                domConversion !== null &&
                (currentConversion === null ||
                    (currentConversion.priority || 0) < (domConversion.priority || 0))
            ) {
                currentConversion = domConversion;
            }
        }
    }

    return currentConversion !== null ? currentConversion.conversion : null;
}

function $createNodesFromDOM(
    node: Node,
    editor: LexicalEditor,
    forChildMap: Map<string, DOMChildConversion> = new Map(),
    parentLexicalNode?: LexicalNode | null | undefined,
): Array<LexicalNode> {
    let lexicalNodes: Array<LexicalNode> = [];

    if (IGNORE_TAGS.has(node.nodeName)) {
        return lexicalNodes;
    }

    let currentLexicalNode: LexicalNode | null | undefined = null;
    const transformFunction = getConversionFunction(node, editor);
    const transformOutput = transformFunction
        ? transformFunction(node as HTMLElement)
        : null;
    let postTransform: DOMConversionOutput["after"] | null = null;

    if (transformOutput !== null) {
        postTransform = transformOutput.after;
        const transformNodes = transformOutput.node;
        currentLexicalNode = Array.isArray(transformNodes)
            ? transformNodes[transformNodes.length - 1]
            : transformNodes;

        if (currentLexicalNode !== null) {
            for (const [, forChildFunction] of forChildMap) {
                currentLexicalNode = forChildFunction(
                    currentLexicalNode,
                    parentLexicalNode,
                );

                if (!currentLexicalNode) {
                    break;
                }
            }

            if (currentLexicalNode) {
                lexicalNodes.push(
                    ...(Array.isArray(transformNodes)
                        ? transformNodes
                        : [currentLexicalNode]),
                );
            }
        }

        if (transformOutput.forChild != null) {
            forChildMap.set(node.nodeName, transformOutput.forChild);
        }
    }

    // If the DOM node doesn't have a transformer, we don't know what
    // to do with it but we still need to process any childNodes.
    const children = node.childNodes;
    let childLexicalNodes: LexicalNode[] = [];

    for (let i = 0; i < children.length; i++) {
        childLexicalNodes.push(
            ...$createNodesFromDOM(
                children[i],
                editor,
                new Map(forChildMap),
                currentLexicalNode,
            ),
        );
    }

    if (postTransform != null) {
        childLexicalNodes = postTransform(childLexicalNodes);
    }

    if (currentLexicalNode == null) {
        // If it hasn't been converted to a LexicalNode, we hoist its children
        // up to the same level as it.
        lexicalNodes = lexicalNodes.concat(childLexicalNodes);
    } else {
        if ($isElementNode(currentLexicalNode)) {
            // If the current node is a ElementNode after conversion,
            // we can append all the children to it.
            currentLexicalNode.append(...childLexicalNodes);
        }
    }

    return lexicalNodes;
}

/**
 * How you parse your html string to get a document is left up to you. In the browser you can use the native
 * DOMParser API to generate a document (see clipboard.ts), but to use in a headless environment you can use JSDom
 * or an equivalent library and pass in the document here.
 */
export function $generateNodesFromDOM(
    editor: LexicalEditor,
    dom: Document,
): Array<LexicalNode> {
    const elements = dom.body ? dom.body.childNodes : [];
    let lexicalNodes: Array<LexicalNode> = [];
    for (let i = 0; i < elements.length; i++) {
        const element = elements[i];
        if (!IGNORE_TAGS.has(element.nodeName)) {
            const lexicalNode = $createNodesFromDOM(element, editor);
            if (lexicalNode !== null) {
                lexicalNodes = lexicalNodes.concat(lexicalNode);
            }
        }
    }

    return lexicalNodes;
}

function $updateElementNodeProperties<T extends ElementNode>(
    target: T,
    source: ElementNode,
): T {
    target.__first = source.__first;
    target.__last = source.__last;
    target.__size = source.__size;
    target.__format = source.__format;
    target.__indent = source.__indent;
    target.__dir = source.__dir;
    return target;
}

function $updateTextNodeProperties<T extends TextNode>(
    target: T,
    source: TextNode,
): T {
    target.__format = source.__format;
    target.__style = source.__style;
    target.__mode = source.__mode;
    target.__detail = source.__detail;
    return target;
}

function $updateParagraphNodeProperties<T extends ParagraphNode>(
    target: T,
    source: ParagraphNode,
): T {
    target.__textFormat = source.__textFormat;
    return target;
}


/**
 * Returns a copy of a node, but generates a new key for the copy.
 * @param node - The node to be cloned.
 * @returns The clone of the node.
 */
export function $cloneWithProperties<T extends LexicalNode>(node: T): T {
    const constructor = node.constructor;
    // @ts-expect-error
    const clone: T = constructor.clone(node);
    clone.__parent = node.__parent;
    clone.__next = node.__next;
    clone.__prev = node.__prev;

    if ($isElementNode(node) && $isElementNode(clone)) {
        return $updateElementNodeProperties(clone, node);
    }

    if ($isTextNode(node) && $isTextNode(clone)) {
        return $updateTextNodeProperties(clone, node);
    }

    if ($isParagraphNode(node) && $isParagraphNode(clone)) {
        return $updateParagraphNodeProperties(clone, node);
    }
    return clone;
}

/**
 * Generally used to append text content to HTML and JSON. Grabs the text content and "slices"
 * it to be generated into the new TextNode.
 * @param selection - The selection containing the node whose TextNode is to be edited.
 * @param textNode - The TextNode to be edited.
 * @returns The updated TextNode.
 */
export function $sliceSelectedTextNodeContent(
    selection: BaseSelection,
    textNode: TextNode,
): LexicalNode {
    const anchorAndFocus = selection.getStartEndPoints();
    if (
        textNode.isSelected(selection) &&
        !textNode.isSegmented() &&
        !textNode.isToken() &&
        anchorAndFocus !== null
    ) {
        const [anchor, focus] = anchorAndFocus;
        const isBackward = selection.isBackward();
        const anchorNode = anchor.getNode();
        const focusNode = focus.getNode();
        const isAnchor = textNode.is(anchorNode);
        const isFocus = textNode.is(focusNode);

        if (isAnchor || isFocus) {
            const [anchorOffset, focusOffset] = $getCharacterOffsets(selection);
            const isSame = anchorNode.is(focusNode);
            const isFirst = textNode.is(isBackward ? focusNode : anchorNode);
            const isLast = textNode.is(isBackward ? anchorNode : focusNode);
            let startOffset = 0;
            let endOffset: number | undefined = undefined;

            if (isSame) {
                startOffset = anchorOffset > focusOffset ? focusOffset : anchorOffset;
                endOffset = anchorOffset > focusOffset ? anchorOffset : focusOffset;
            } else if (isFirst) {
                const offset = isBackward ? focusOffset : anchorOffset;
                startOffset = offset;
                endOffset = undefined;
            } else if (isLast) {
                const offset = isBackward ? anchorOffset : focusOffset;
                startOffset = 0;
                endOffset = offset;
            }

            textNode.__text = textNode.__text.slice(startOffset, endOffset);
            return textNode;
        }
    }
    return textNode;
}
