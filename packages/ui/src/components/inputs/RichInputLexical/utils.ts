/* eslint-disable @typescript-eslint/ban-ts-comment */
import { $insertDataTransferForRichText, copyToClipboard } from "./clipboard";
import { CUT_COMMAND, PASTE_COMMAND } from "./commands";
import { CAN_USE_DOM, COMPOSITION_SUFFIX, DOM_TEXT_TYPE, HAS_DIRTY_NODES, IGNORE_TAGS, IS_APPLE, IS_APPLE_WEBKIT, IS_FIREFOX, IS_IOS, IS_SAFARI } from "./consts";
import { type EditorState, type LexicalEditor } from "./editor";
import { LexicalNodes } from "./nodes";
import { type DecoratorNode } from "./nodes/DecoratorNode";
import { type ElementNode } from "./nodes/ElementNode";
import { type LexicalNode } from "./nodes/LexicalNode";
import { type LineBreakNode } from "./nodes/LineBreakNode";
import { type ListItemNode } from "./nodes/ListItemNode";
import { type ListNode } from "./nodes/ListNode";
import { type ParagraphNode } from "./nodes/ParagraphNode";
import { type RootNode } from "./nodes/RootNode";
import { type TableCellNode } from "./nodes/TableCellNode";
import { type TableNode } from "./nodes/TableNode";
import { type TextNode } from "./nodes/TextNode";
import { $getCharacterOffsets, $getPreviousSelection, $updateElementSelectionOnCreateDeleteNode, NodeSelection, Point, RangeSelection, moveSelectionPointToSibling } from "./selection";
import { BaseSelection, CommandPayloadType, CustomDomElement, DOMChildConversion, DOMConversion, DOMConversionFn, DOMConversionOutput, EditorConfig, IntentionallyMarkedAsDirtyElement, LexicalNodeClass, MutatedNodes, NodeConstructorPayloads, NodeConstructors, NodeKey, NodeMap, NodeMutation, ObjectClass, PasteCommandType, PointType, RegisteredNodes, ShadowRootNode, Spread, TableMapType, TableMapValueType } from "./types";
import { errorOnInfiniteTransforms, errorOnReadOnly, getActiveEditor, getActiveEditorState, isCurrentlyReadOnlyMode, updateEditor } from "./updates";

export function getWindow(editor: LexicalEditor): Window {
    const windowObj = editor._window;
    if (windowObj === null) {
        throw new Error("window object not found");
    }
    return windowObj;
}

export function getDOMSelection(targetWindow: null | Window): null | Selection {
    return !CAN_USE_DOM ? null : (targetWindow || window).getSelection();
}

export function getDefaultView(domElem: HTMLElement): Window | null {
    const ownerDoc = domElem.ownerDocument;
    return (ownerDoc && ownerDoc.defaultView) || null;
}

export function isFirefoxClipboardEvents(editor: LexicalEditor): boolean {
    const event = getWindow(editor).event;
    const inputType = event && (event as InputEvent).inputType;
    return (
        inputType === "insertFromPaste" ||
        inputType === "insertFromPasteAsQuotation"
    );
}

/**
 * Normalizes a list of class name inputs, filtering out non-string values and splitting space-separated class names.
 * This function is designed to handle dynamic class name entries typically used in web development where class names
 * might be conditionally applied and could include multiple classes in a single string separated by spaces.
 *
 * @param classNames - A variable number of arguments that can be undefined, boolean, null, or string.
 *        Only string values are processed; other types are ignored. String values containing multiple class names separated by spaces are split.
 * @returnS An array of individual, trimmed class names with no duplicates and no empty or non-string values.
 *
 * @example
 * // Returns ['btn', 'primary', 'disabled', 'header', 'footer']
 * normalizeClassNames("btn primary", null, undefined, "disabled", false, " header  footer ");
 */
export function normalizeClassNames(
    ...classNames: Array<typeof undefined | boolean | null | string>
): Array<string> {
    const rval: string[] = [];
    for (const className of classNames) {
        if (className && typeof className === "string") {
            for (const [s] of className.matchAll(/\S+/g)) {
                rval.push(s);
            }
        }
    }
    return rval;
}

export function markAllNodesAsDirty(editor: LexicalEditor, type: string): void {
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
}

export function $getCompositionKey(): null | NodeKey {
    if (isCurrentlyReadOnlyMode()) {
        return null;
    }
    const editor = getActiveEditor();
    return editor._compositionKey;
}

export function $setCompositionKey(compositionKey: null | NodeKey): void {
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
}

export function $getRoot(): RootNode {
    return getActiveEditorState()._nodeMap.get("root") as RootNode;
}

export function $getSelection(): BaseSelection | null {
    return getActiveEditorState()._selection;
}

/**
 * @param x - The element being tested
 * @returns Returns true if x is an HTML anchor tag, false otherwise
 */
export function isHTMLAnchorElement(x: Node): x is HTMLAnchorElement {
    return isHTMLElement(x) && x.tagName === "A";
}

export function $isSelectionCapturedInDecorator(node: Node): boolean {
    return $isNode("Decorator", $getNearestNodeFromDOMNode(node));
}

export function isHTMLElement(x: unknown): x is HTMLElement {
    return x instanceof HTMLElement;
}

export function $isRangeSelection(x: unknown): x is RangeSelection {
    // Duck typing :P (and not instanceof RangeSelection) because extension operates
    // from different JS bundle and has no reference to the RangeSelection used on the page
    return x !== null && x !== undefined && typeof x === "object" && "applyDOMRange" in x;
}

export function $isNodeSelection(x: unknown): x is NodeSelection {
    // Duck typing :P (and not instanceof NodeSelection) because extension operates
    // from different JS bundle and has no reference to the NodeSelection used on the page
    return x !== null && x !== undefined && typeof x === "object" && "_nodes" in x;
}

export function $isRootOrShadowRoot(
    node: null | LexicalNode,
): node is RootNode | ShadowRootNode {
    return $isNode("Root", node) || ($isNode("Element", node) && node.isShadowRoot());
}

export function $setSelection(selection: null | BaseSelection): void {
    errorOnReadOnly();
    const editorState = getActiveEditorState();
    if (selection !== null) {
        selection.dirty = true;
        selection.setCachedNodes(null);
    }
    editorState._selection = selection;
}

export function $nodesOfType<K extends keyof NodeConstructors>(
    nodeType: K,
): Array<InstanceType<NodeConstructors[K]>> {
    const editorState = getActiveEditorState();
    const readOnly = editorState._readOnly;
    const nodes = editorState._nodeMap;
    const nodesOfType: Array<InstanceType<NodeConstructors[K]>> = [];

    for (const [, node] of nodes) {
        if (
            $isNode(nodeType, node) && // Matches class
            node.getType() === LexicalNodes.get(nodeType)?.getType() && // Matches type (i.e. not a subclass of the node)
            (readOnly || isAttachedToRoot(node))
        ) {
            nodesOfType.push(node as InstanceType<NodeConstructors[K]>);
        }
    }
    return nodesOfType;
}

/**
 * Takes a node and traverses up its ancestors (toward the root node)
 * in order to find a specific type of node.
 * @param node - the node to begin searching.
 * @param nodeType - the type of node to search for.
 * @returns the node of the requested type, or null if none exist.
 */
export function $getNearestNodeOfType<K extends keyof NodeConstructors>(
    nodeType: K,
    node: LexicalNode,
): InstanceType<NodeConstructors[K]> | null {
    let parent: LexicalNode | null = node;

    while (parent !== null) {
        if ($isNode(nodeType, parent)) {
            return parent as InstanceType<NodeConstructors[K]>;
        }

        parent = getParent(parent);
    }

    return null;
}

export function $updateTextNodeFromDOMContent(
    textNode: TextNode,
    textContent: string,
    anchorOffset: null | number,
    focusOffset: null | number,
    compositionEnd: boolean,
): void {
    let node = textNode;

    if (isAttachedToRoot(node) && (compositionEnd || !node.isDirty())) {
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
                            if (isAttachedToRoot(node)) {
                                node.remove();
                            }
                        });
                    }, 20);
                } else {
                    node.remove();
                }
                return;
            }
            const parent = getParent(node);
            const prevSelection = $getPreviousSelection();
            const prevTextContentSize = node.getTextContentSize();
            const compositionKey = $getCompositionKey();
            const nodeKey = node.__key;

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
                const replacement = $createNode("Text", { text: originalTextContent });
                node.replace(replacement);
                node = replacement;
            }
            node.setTextContent(normalizedTextContent);
        }
    }
}

/**
 * Determines if the current selection is at the end of the node.
 * @param point - The point of the selection to test.
 * @returns true if the provided point offset is in the last possible position, false otherwise.
 */
export function $isAtNodeEnd(point: Point): boolean {
    if (point.type === "text") {
        return point.offset === point.getNode().getTextContentSize();
    }
    const node = point.getNode();
    if (!$isNode("Element", node)) {
        throw new Error("isAtNodeEnd: node must be a TextNode or ElementNode");
    }

    return point.offset === node.getChildrenSize();
}

/**
 * Starts with a node and moves up the tree (toward the root node) to find a matching node based on
 * the search parameters of the findFn. (Consider JavaScripts' .find() function where a testing function must be
 * passed as an argument. eg. if( (node) => node.__type === "Paragraph") ) return true; otherwise return false
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

        while (curr !== $getRoot() && curr !== null) {
            if (findFn(curr)) {
                return curr;
            }

            curr = getParent(curr);
        }

        return null;
    };

export const scheduleMicroTask: (fn: () => void) => void =
    typeof queueMicrotask === "function"
        ? queueMicrotask
        : (fn) => {
            // No window prefix intended (#1400)
            Promise.resolve().then(fn);
        };

export function removeDOMBlockCursorElement(
    blockCursorElement: HTMLElement,
    editor: LexicalEditor,
    rootElement: HTMLElement,
) {
    rootElement.style.removeProperty("caret-color");
    editor._blockCursorElement = null;
    const parentElement = blockCursorElement.parentElement;
    if (parentElement !== null) {
        parentElement.removeChild(blockCursorElement);
    }
}

function createBlockCursorElement(editorConfig: EditorConfig): HTMLDivElement {
    // const theme = editorConfig.theme;
    const element = document.createElement("div");
    element.contentEditable = "false";
    element.setAttribute("data-lexical-cursor", "true");
    // let blockCursorTheme = theme.blockCursor;
    // if (blockCursorTheme !== undefined) {
    //     if (typeof blockCursorTheme === "string") {
    //         const classNamesArr = normalizeClassNames(blockCursorTheme);
    //         // @ts-expect-error: intentional
    //         blockCursorTheme = theme.blockCursor = classNamesArr;
    //     }
    //     if (blockCursorTheme !== undefined) {
    //         element.classList.add(...blockCursorTheme);
    //     }
    // }
    return element;
}

export function $applyNodeReplacement<N extends LexicalNode>(
    node: LexicalNode,
): N {
    const editor = getActiveEditor();
    const nodeType = node.getType();
    const registeredNode = editor._nodes[nodeType];
    if (registeredNode === undefined) {
        throw new Error("$initializeNode failed. Ensure node has been registered to the editor. You can do this by passing the node class via the \"nodes\" array in the editor config.");
    }
    return node as N;
}

function needsBlockCursor(node: null | LexicalNode): boolean {
    return (
        ($isNode("Decorator", node) || ($isNode("Element", node) && !node.canBeEmpty())) &&
        !node.isInline()
    );
}

export function updateDOMBlockCursorElement(
    editor: LexicalEditor,
    rootElement: HTMLElement,
    nextSelection: null | BaseSelection,
) {
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
                const sibling = getPreviousSibling((child as LexicalNode));
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
}

/**
 * Retrieves the parent element of a given DOM node, accounting for cases where the node is within a Shadow DOM or a slot.
 *
 * This function checks if the node is slotted and returns the slot as the parent if true. Otherwise, it returns the node's direct parent element.
 * If the parent is part of a Shadow DOM (an encapsulated, isolated DOM segment), the function returns the host element that contains the Shadow DOM.
 *
 * @param node - The DOM node whose parent element is to be determined.
 * @returns  The parent HTML element or null if there is no parent.
 */
export function getParentElement(node: Node): HTMLElement | null {
    // First check for node's direct parent if it's assigned to a slot.
    const parentElement = node instanceof HTMLElement && node.assignedSlot ? node.assignedSlot : node.parentElement;

    // Then check if the parent element is a shadow root and return its host.
    if (parentElement && parentElement.nodeType === Node.DOCUMENT_FRAGMENT_NODE) {
        return (parentElement as unknown as ShadowRoot).host as HTMLElement;
    }

    // If none of the above, return the parent element or null if there isn't any.
    return parentElement;
}

export function getTextNodeOffset(
    node: TextNode,
    moveSelectionToEnd: boolean,
): number {
    return moveSelectionToEnd ? node.getTextContentSize() : 0;
}

export function $getNodeByKey<T extends LexicalNode>(
    key: NodeKey,
    _editorState?: EditorState | null,
): T | null {
    const editorState = _editorState || getActiveEditorState();
    const node = editorState._nodeMap.get(key) as T;
    if (!node) {
        return null;
    }
    return node;
}

export function getNodeFromDOMNode(
    dom: Node,
    editorState?: EditorState | null,
): LexicalNode | null {
    const editor = getActiveEditor();
    const key = (dom as CustomDomElement)[`__lexicalKey_${editor._key}`];
    if (key) {
        return $getNodeByKey(key, editorState);
    }
    return null;
}

export function getNodeFromDOM(dom: Node): null | LexicalNode {
    const editor = getActiveEditor();
    const nodeKey = getNodeKeyFromDOM(dom, editor);
    if (!nodeKey) {
        const rootElement = editor.getRootElement();
        if (dom === rootElement) {
            return $getNodeByKey("root");
        }
        return null;
    }
    return $getNodeByKey(nodeKey);
}

function getNodeKeyFromDOM(
    // Note that node here refers to a DOM Node, not an Lexical Node
    dom: Node,
    editor: LexicalEditor,
): NodeKey | null {
    let node: Node | null = dom;
    while (node) {
        const key = (node as CustomDomElement)[`__lexicalKey_${editor._key}`];
        if (key) {
            return key;
        }
        node = getParentElement(node);
    }
    return null;
}

/**
 * Traverse down the DOM tree to find the nearest LexicalNode.
 * @param element The DOM element to start the search from.
 * @returns The nearest LexicalNode, or null if none is found.
 */
export function getDOMTextNode(element: Node | null): Text | null {
    let node = element;
    while (node) {
        if (node.nodeType === DOM_TEXT_TYPE) {
            return node as Text;
        }
        node = node.firstChild;
    }
    return null;
}

export function $getNearestNodeFromDOMNode(
    startingDOM: Node,
    editorState?: EditorState | null,
): LexicalNode | null {
    let dom: Node | null = startingDOM;
    while (dom) {
        const node = getNodeFromDOMNode(dom, editorState);
        if (node) {
            return node;
        }
        dom = getParentElement(dom);
    }
    return null;
}

export function isSelectionCapturedInDecoratorInput(anchorDOM: Node): boolean {
    const activeElement = document.activeElement as HTMLElement;

    if (!activeElement) {
        return false;
    }
    const nodeName = activeElement.nodeName;

    return (
        $isNode("Decorator", $getNearestNodeFromDOMNode(anchorDOM)) &&
        (nodeName === "INPUT" ||
            nodeName === "TEXTAREA" ||
            (activeElement.contentEditable === "true" &&
                (activeElement as CustomDomElement).__lexicalEditor === null))
    );
}

export function getNearestEditorFromDOMNode(
    node: Node | null,
): LexicalEditor | null {
    let currentNode = node;
    while (currentNode) {
        const editor = (currentNode as CustomDomElement).__lexicalEditor;
        if (editor) {
            return editor;
        }
        currentNode = getParentElement(currentNode);
    }
    return null;
}

export function isSelectionWithinEditor(
    editor: LexicalEditor,
    anchorDOM: null | Node,
    focusDOM: null | Node,
): boolean {
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
}

export function getElementByKeyOrThrow(
    editor: LexicalEditor,
    key: NodeKey,
): HTMLElement {
    const element = editor._keyToDOMMap.get(key);

    if (element === undefined) {
        throw new Error("Reconciliation: could not find DOM element for node key");
    }

    return element;
}

export function scrollIntoViewIfNeeded(
    editor: LexicalEditor,
    selectionRect: DOMRect,
    rootElement: HTMLElement,
): void {
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
}

type KeyEvent = Pick<KeyboardEvent, "altKey" | "ctrlKey" | "key" | "metaKey" | "shiftKey">;
type KeyEventHandler =
    | "tab"
    | "bold"
    | "controlOrMeta"
    | "copy"
    | "cut"
    | "deleteWordBackward"
    | "deleteWordForward"
    | "deleteLineBackward"
    | "deleteLineForward"
    | "deleteBackward"
    | "deleteForward"
    | "italic"
    | "underline"
    | "paragraph"
    | "lineBreak"
    | "openLineBreak"
    | "undo"
    | "redo"
    | "arrowLeft"
    | "arrowRight"
    | "arrowUp"
    | "arrowDown"
    | "moveBackward"
    | "moveToStart"
    | "moveForward"
    | "moveToEnd"
    | "moveUp"
    | "moveDown"
    | "modifier"
    | "space"
    | "return"
    | "backspace"
    | "escape"
    | "delete"
    | "selectAll";

export const IS_KEY: Record<KeyEventHandler, ((event: KeyEvent) => boolean)> = {
    tab: ({ altKey, ctrlKey, key, metaKey }) => key === "Tab" && !altKey && !ctrlKey && !metaKey,
    bold: (props) => props.key === "b" && !props.altKey && IS_KEY.controlOrMeta(props),
    controlOrMeta({ metaKey, ctrlKey }) {
        if (IS_APPLE) {
            return metaKey;
        }
        return ctrlKey;
    },
    copy({ ctrlKey, key, metaKey, shiftKey }) {
        if (shiftKey) {
            return false;
        }
        if (key === "c") {
            return IS_APPLE ? metaKey : ctrlKey;
        }

        return false;
    },
    cut({ ctrlKey, key, metaKey, shiftKey }) {
        if (shiftKey) {
            return false;
        }
        if (key === "x") {
            return IS_APPLE ? metaKey : ctrlKey;
        }
        return false;
    },
    deleteWordBackward: (props) => IS_KEY.backspace(props) && (IS_APPLE ? props.altKey : props.ctrlKey),
    deleteWordForward: (props) => IS_KEY.delete(props) && (IS_APPLE ? props.altKey : props.ctrlKey),
    deleteLineBackward: (props) => IS_APPLE && props.metaKey && IS_KEY.backspace(props),
    deleteLineForward: (props) => IS_APPLE && props.metaKey && IS_KEY.delete(props),
    deleteBackward(props) {
        if (IS_APPLE) {
            if (props.altKey || props.metaKey) {
                return false;
            }
            return IS_KEY.backspace(props) || (props.key === "h" && props.ctrlKey);
        }
        if (props.ctrlKey || props.altKey || props.metaKey) {
            return false;
        }
        return IS_KEY.backspace(props);
    },
    deleteForward(props) {
        if (IS_APPLE) {
            if (props.shiftKey || props.altKey || props.metaKey) {
                return false;
            }
            return IS_KEY.delete(props) || (props.key === "d" && props.ctrlKey);
        }
        if (props.ctrlKey || props.altKey || props.metaKey) {
            return false;
        }
        return IS_KEY.delete(props);
    },
    italic: (props) => props.key === "i" && !props.altKey && IS_KEY.controlOrMeta(props),
    underline: (props) => props.key === "u" && !props.altKey && IS_KEY.controlOrMeta(props),
    paragraph: (props) => IS_KEY.return(props) && !props.shiftKey,
    lineBreak: (props) => IS_KEY.return(props) && props.shiftKey,
    openLineBreak: ({ ctrlKey, key }) => IS_APPLE && ctrlKey && key === "o",
    undo: (props) => props.key === "z" && !props.shiftKey && IS_KEY.controlOrMeta(props),
    redo({ ctrlKey, key, metaKey, shiftKey }) {
        if (IS_APPLE) {
            return key === "z" && metaKey && shiftKey;
        }
        return (key === "y" && ctrlKey) || (key === "z" && ctrlKey && shiftKey);
    },
    arrowLeft: ({ key }) => key === "ArrowLeft" || key === "Left",
    arrowRight: ({ key }) => key === "ArrowRight" || key === "Right",
    arrowUp: ({ key }) => key === "ArrowUp" || key === "Up",
    arrowDown: ({ key }) => key === "ArrowDown" || key === "Down",
    moveBackward: (props) => IS_KEY.arrowLeft(props) && !props.ctrlKey && !props.metaKey && !props.altKey,
    moveToStart: (props) => IS_KEY.arrowLeft(props) && !props.altKey && !props.shiftKey && (props.ctrlKey || props.metaKey),
    moveForward: (props) => IS_KEY.arrowRight(props) && !props.ctrlKey && !props.metaKey && !props.altKey,
    moveToEnd: (props) => IS_KEY.arrowRight(props) && !props.altKey && !props.shiftKey && (props.ctrlKey || props.metaKey),
    moveUp: (props) => IS_KEY.arrowUp(props) && !props.ctrlKey && !props.metaKey,
    moveDown: (props) => IS_KEY.arrowDown(props) && !props.ctrlKey && !props.metaKey,
    modifier: ({ altKey, ctrlKey, metaKey, shiftKey }) => ctrlKey || shiftKey || altKey || metaKey,
    space: ({ key }) => key === " " || key === "Spacebar",
    return: ({ key }) => key === "Enter",
    backspace: ({ key }) => (!IS_APPLE && key === "Backspace") || (IS_APPLE && key === "Delete"),
    escape: ({ key }) => key === "Escape" || key === "Esc",
    delete: ({ key }) => key === "Delete",
    selectAll: (props) => props.key === "a" && IS_KEY.controlOrMeta(props),
};

/**
 * Removes a specified node from its parent in a doubly linked list structure.
 * It adjusts sibling and parent references to maintain the integrity of the list after the node is removed.
 * @param node - The LexicalNode to be removed.
 */
export function removeFromParent(node: LexicalNode) {
    const oldParent = getParent(node);
    if (oldParent !== null) {
        const writableNode = node.getWritable();
        const writableParent = oldParent.getWritable();

        const prevSibling = getPreviousSibling(node);
        const nextSibling = getNextSibling(node);

        // Update the links of previous and next siblings
        if (prevSibling !== null) {
            const writablePrevSibling = prevSibling.getWritable();
            writablePrevSibling.__next = nextSibling ? nextSibling.__key : null;
        } else {
            writableParent.__first = nextSibling ? nextSibling.__key : null;
        }

        if (nextSibling !== null) {
            const writableNextSibling = nextSibling.getWritable();
            writableNextSibling.__prev = prevSibling ? prevSibling.__key : null;
        } else {
            writableParent.__last = prevSibling ? prevSibling.__key : null;
        }

        // Update the node itself
        writableNode.__prev = null;
        writableNode.__next = null;
        writableNode.__parent = null;

        // Decrement the size of the parent
        writableParent.__size--;
    }
}

/**
 * Takes an HTML element and adds the classNames passed within an array,
 * ignoring any non-string types. A space can be used to add multiple classes
 * eg. addClassNamesToElement(element, ['element-inner active', true, null])
 * will add both 'element-inner' and 'active' as classes to that element.
 * @param element - The element in which the classes are added
 * @param classNames - An array defining the class names to add to the element
 */
export function addClassNamesToElement(
    element: HTMLElement,
    ...classNames: Array<typeof undefined | boolean | null | string>
) {
    const classesToAdd = normalizeClassNames(...classNames);
    if (classesToAdd.length > 0) {
        element.classList.add(...classesToAdd);
    }
}

export function getAnchorTextFromDOM(anchorNode: Node): null | string {
    if (anchorNode.nodeType === DOM_TEXT_TYPE) {
        return anchorNode.nodeValue;
    }
    return null;
}

export function $updateSelectedTextFromDOM(
    isCompositionEnd: boolean,
    editor: LexicalEditor,
    data?: string,
) {
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
        if (textContent !== null && $isNode("Text", node)) {
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
}

/**
 * Checks if a string contains a grapheme, which is a single 
 * unit of a writing system that is treated as a single character (eg. a letter, an ideogram, etc.)
 */
export function doesContainGrapheme(str: string): boolean {
    return /[\uD800-\uDBFF][\uDC00-\uDFFF]/g.test(str);
}

export function $isTokenOrSegmented(node: TextNode): boolean {
    return node.isToken() || node.isSegmented();
}

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
export function mergeRegister(...func: Array<(() => void)>): (() => void) {
    return () => {
        func.forEach((f) => f());
    };
}

export function $getAncestor<NodeType extends LexicalNode = LexicalNode>(
    node: LexicalNode,
    predicate: (ancestor: LexicalNode) => ancestor is NodeType,
) {
    let parent = node;
    while (parent !== null && getParent(parent) !== null && !predicate(parent)) {
        parent = getParent(parent, { throwIfNull: true });
    }
    return predicate(parent) ? parent : null;
}

export function $getNearestRootOrShadowRoot(
    node: LexicalNode,
): RootNode | ElementNode {
    let parent = getParent(node, { throwIfNull: true });
    while (parent !== null) {
        if ($isRootOrShadowRoot(parent)) {
            return parent;
        }
        parent = getParent(parent, { throwIfNull: true });
    }
    return parent;
}

export function $hasAncestor(
    child: LexicalNode,
    targetNode: LexicalNode,
): boolean {
    let parent = getParent(child);
    while (parent !== null) {
        if (parent.is(targetNode)) {
            return true;
        }
        parent = getParent(parent);
    }
    return false;
}

export function $maybeMoveChildrenSelectionToParent(
    parentNode: LexicalNode,
): BaseSelection | null {
    const selection = $getSelection();
    if (!$isRangeSelection(selection) || !$isNode("Element", parentNode)) {
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
}

function resolveElement(
    element: ElementNode,
    isBackward: boolean,
    focusOffset: number,
): LexicalNode | null {
    const parent = getParent(element);
    let offset = focusOffset;
    let block = element;
    if (parent !== null) {
        if (isBackward && focusOffset === 0) {
            offset = getIndexWithinParent(block);
            block = parent;
        } else if (!isBackward && focusOffset === block.getChildrenSize()) {
            offset = getIndexWithinParent(block) + 1;
            block = parent;
        }
    }
    return block.getChildAtIndex(isBackward ? offset - 1 : offset);
}

export function $getAdjacentNode(
    focus: PointType,
    isBackward: boolean,
): LexicalNode | null {
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
                ? getPreviousSibling(focusNode)
                : getNextSibling(focusNode);
            if (possibleNode === null) {
                return resolveElement(
                    getParent(focusNode, { throwIfNull: true }),
                    isBackward,
                    getIndexWithinParent(focusNode) + (isBackward ? 0 : 1),
                );
            }
            return possibleNode;
        }
    }
    return null;
}

function internalMarkParentElementsAsDirty(
    parentKey: NodeKey,
    nodeMap: NodeMap,
    dirtyElements: Map<NodeKey, IntentionallyMarkedAsDirtyElement>,
): void {
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
}

// Never use this function directly! It will break
// the cloning heuristic. Instead use node.getWritable().
export function internalMarkNodeAsDirty(node: LexicalNode): void {
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
    if ($isNode("Element", node)) {
        dirtyElements.set(key, true);
    } else {
        // TODO split internally MarkNodeAsDirty into two dedicated Element/leave functions
        editor._dirtyLeaves.add(key);
    }
}

export function internalMarkSiblingsAsDirty(node: LexicalNode) {
    const previousNode = getPreviousSibling(node);
    const nextNode = getNextSibling(node);
    if (previousNode !== null) {
        internalMarkNodeAsDirty(previousNode);
    }
    if (nextNode !== null) {
        internalMarkNodeAsDirty(nextNode);
    }
}

/**
 * Wraps a node into a ListItemNode.
 * @param node - The node to be wrapped into a ListItemNode
 * @returns The ListItemNode which the passed node is wrapped in.
 */
export function wrapInListItem(node: LexicalNode): ListItemNode {
    const listItemWrapper = $createNode("ListItem", {});
    return listItemWrapper.append(node);
}

let keyCounter = 1;
export function generateRandomKey(): string {
    return "key-" + keyCounter++;
}

export function $createNodeKey(
    node: LexicalNode,
    existingKey: NodeKey | null | undefined, // Only used for the root node
): NodeKey {
    // Should only be used for the root node. This is used to 
    // bypass the active editor stuff below
    if (existingKey !== undefined && existingKey !== null) {
        return existingKey;
    }
    errorOnReadOnly();
    errorOnInfiniteTransforms();
    const editor = getActiveEditor();
    const editorState = getActiveEditorState();
    const key = generateRandomKey();
    editorState._nodeMap.set(key, node);
    // TODO Split this function into leaf/element
    if ($isNode("Element", node)) {
        editor._dirtyElements.set(key, true);
    } else {
        editor._dirtyLeaves.add(key);
    }
    editor._cloneNotNeeded.add(key);
    editor._dirtyType = HAS_DIRTY_NODES;
    return key;
}

export function errorOnInsertTextNodeOnRoot(
    node: LexicalNode,
    insertNode: LexicalNode,
): void {
    const parentNode = getParent(node);
    if (
        $isNode("Root", parentNode) &&
        !$isNode("Element", insertNode) &&
        !$isNode("Decorator", insertNode)
    ) {
        throw new Error("Only element or decorator nodes can be inserted in to the root node");
    }
}

/**
 * Takes an HTML element and removes the classNames passed within an array,
 * ignoring any non-string types. A space can be used to remove multiple classes
 * eg. removeClassNamesFromElement(element, ['active small', true, null])
 * will remove both the 'active' and 'small' classes from that element.
 * @param element - The element in which the classes are removed
 * @param classNames - An array defining the class names to remove from the element
 */
export function removeClassNamesFromElement(
    element: HTMLElement,
    ...classNames: Array<typeof undefined | boolean | null | string>
): void {
    const classesToRemove = normalizeClassNames(...classNames);
    if (classesToRemove.length > 0) {
        element.classList.remove(...classesToRemove);
    }
}

export function $computeTableMap(
    grid: TableNode,
    cellA: TableCellNode,
    cellB: TableCellNode,
): [TableMapType, TableMapValueType, TableMapValueType] {
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
        if (!$isNode("TableRow", row)) {
            throw new Error("Expected GridNode children to be TableRowNode");
        }
        const rowChildren = row.getChildren();
        let j = 0;
        for (const cell of rowChildren) {
            if (!$isNode("TableCell", cell)) {
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
}

/**
 * Calculates the zoom level of an element as a result of using
 * css zoom property.
 * @param element
 */
export function calculateZoomLevel(element: Element | null): number {
    if (IS_FIREFOX) {
        return 1;
    }
    let zoom = 1;
    while (element) {
        zoom *= Number(window.getComputedStyle(element).getPropertyValue("zoom"));
        element = element.parentElement;
    }
    return zoom;
}

function $previousSiblingDoesNotAcceptText(node: TextNode): boolean {
    const previousSibling = getPreviousSibling(node);

    return (
        ($isNode("Text", previousSibling) ||
            ($isNode("Element", previousSibling) && previousSibling.isInline())) &&
        !previousSibling.canInsertTextAfter()
    );
}

// This function is connected to $shouldPreventDefaultAndInsertText and determines whether the
// TextNode boundaries are writable or we should use the previous/next sibling instead. For example,
// in the case of a LinkNode, boundaries are not writable.
export function $shouldInsertTextAfterOrBeforeTextNode(
    selection: RangeSelection,
    node: TextNode,
): boolean {
    if (node.isSegmented()) {
        return true;
    }
    if (!selection.isCollapsed()) {
        return false;
    }
    const offset = selection.anchor.offset;
    const parent = getParent(node, { throwIfNull: true });
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
}

export function setMutatedNode(
    mutatedNodes: MutatedNodes,
    registeredNodes: RegisteredNodes | undefined,
    node: LexicalNode,
    mutation: NodeMutation,
) {
    const nodeType = node.getType();
    const nodeKey = node.__key;
    const registeredNode = registeredNodes?.[nodeType];
    if (!registeredNode) {
        throw new Error(`Type ${nodeType} not in registeredNodes`);
    }
    let mutatedNodesByType = mutatedNodes[nodeType];
    if (!mutatedNodesByType) {
        mutatedNodesByType = {};
        mutatedNodes[nodeType] = mutatedNodesByType;
    }
    const prevMutation = mutatedNodesByType[nodeKey];
    // If the node has already been "destroyed", yet we are
    // re-making it, then this means a move likely happened.
    // We should change the mutation to be that of "updated"
    // instead.
    const isMove = prevMutation === "destroyed" && mutation === "created";
    if (prevMutation === undefined || isMove) {
        mutatedNodesByType[nodeKey] = isMove ? "updated" : mutation;
    }
}

export function cloneDecorators(
    editor: LexicalEditor,
): Record<NodeKey, unknown> {
    const currentDecorators = editor._decorators;
    const pendingDecorators = Object.assign({}, currentDecorators);
    editor._pendingDecorators = pendingDecorators;
    return pendingDecorators;
}

export function $isTargetWithinDecorator(target: HTMLElement): boolean {
    const node = $getNearestNodeFromDOMNode(target);
    return $isNode("Decorator", node);
}

/**
 * Returns the element node of the nearest ancestor, otherwise throws an error.
 * @param startNode - The starting node of the search
 * @returns The ancestor node found
 */
export function $getNearestBlockElementAncestorOrThrow(
    startNode: LexicalNode,
): ElementNode {
    const blockNode = $findMatchingParent(
        startNode,
        (node) => $isNode("Element", node) && !node.isInline(),
    );
    if (!$isNode("Element", blockNode)) {
        throw new Error(`Expected node ${startNode.__key} to have closest block element node.`);
    }
    return blockNode;
}

export function handleIndentAndOutdent(
    indentOrOutdent: (block: ElementNode) => void,
): boolean {
    const selection = $getSelection();
    if (!$isRangeSelection(selection)) {
        return false;
    }
    const alreadyHandled = new Set();
    const nodes = selection.getNodes();
    for (let i = 0; i < nodes.length; i++) {
        const node = nodes[i];
        const key = node.__key;
        if (alreadyHandled.has(key)) {
            continue;
        }
        const parentBlock = $getNearestBlockElementAncestorOrThrow(node);
        const parentKey = parentBlock.__key;
        if (parentBlock.canIndent() && !alreadyHandled.has(parentKey)) {
            alreadyHandled.add(parentKey);
            indentOrOutdent(parentBlock);
        }
    }
    return alreadyHandled.size > 0;
}

export function $isSelectionAtEndOfRoot(selection: RangeSelection) {
    const focus = selection.focus;
    return focus.key === "root" && focus.offset === $getRoot().getChildrenSize();
}

// Clipboard may contain files that we aren't allowed to read. While the event is arguably useless,
// in certain occasions, we want to know whether it was a file transfer, as opposed to text. We
// control this with the first boolean flag.
export function eventFiles(
    event: DragEvent | PasteCommandType,
): [boolean, Array<File>, boolean] {
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
}

/**
 * @param object = The instance of the type
 * @param objectClass = The class of the type
 * @returns Whether the object is has the same Klass of the objectClass, ignoring the difference across window (e.g. different iframs)
 */
export function objectKlassEquals(
    object: unknown,
    objectClass: ObjectClass,
): boolean {
    return object !== null
        ? Object.getPrototypeOf(object).constructor.name === objectClass.name
        : false;
}

/**
 * Creates an object containing all the styles and their values provided in the CSS string.
 * @param css - The CSS string of styles and their values.
 * @returns The styleObject containing all the styles and their values.
 */
export function getStyleObjectFromRawCSS(css: string): Record<string, string> {
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
}

export function $isLeafNode(
    node: LexicalNode | null | undefined,
): node is TextNode | LineBreakNode | DecoratorNode<unknown> {
    return $isNode("Text", node) || $isNode("LineBreak", node) || $isNode("Decorator", node);
}

/**
 * Finds the nearest ancestral ListNode and returns it, throws an invariant if listItem is not a ListItemNode.
 * @param listItem - The node to be checked.
 * @returns The ListNode found.
 */
export function $getTopListNode(listItem: LexicalNode): ListNode {
    let list = getParent<ListNode>(listItem);

    if (!$isNode("List", list)) {
        throw new Error("A ListItemNode must have a ListNode for a parent.");
    }

    let parent: ListNode | null = list;

    while (parent !== null) {
        parent = getParent(parent);

        if ($isNode("List", parent)) {
            list = parent;
        }
    }

    return list;
}

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
    return $isNode("ListItem", node) && $isNode("List", node.getFirstChild());
};

/**
 * A recursive Depth-First Search (Postorder Traversal) that finds all of a node's children
 * that are of type ListItemNode and returns them in an array.
 * @param node - The ListNode to start the search.
 * @returns An array containing all nodes of type ListItemNode found.
 */
// This should probably be $getAllChildrenOfType
export const $getAllListItems = (node: ListNode): Array<ListItemNode> => {
    let listItemNodes: ListItemNode[] = [];
    const listChildren = node
        .getChildren()
        .filter((node) => $isNode("ListItem", node)) as ListItemNode[];

    for (let i = 0; i < listChildren.length; i++) {
        const listItemNode = listChildren[i];
        const firstChild = listItemNode.getFirstChild();

        if ($isNode("List", firstChild)) {
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
        getNextSibling(emptyListPtr) === null &&
        getPreviousSibling(emptyListPtr) === null
    ) {
        const parent = getParent<ListItemNode | ListNode>(emptyListPtr);

        if (
            parent === null ||
            !($isNode("ListItem", emptyListPtr) || $isNode("List", emptyListPtr))
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
    let parent = getParent(listNode);

    while (parent !== null) {
        if ($isNode("ListItem", parent)) {
            const parentList = getParent(parent);

            if ($isNode("List", parentList)) {
                depth++;
                parent = getParent(parentList);
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
        if ($isNode("Text", nextNode)) {
            point.set(
                nextNode.__key,
                nextOffsetAtEnd ? nextNode.getTextContentSize() : 0,
                "text",
            );
            break;
        } else if (!$isNode("Element", nextNode)) {
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
            if (clipboardData !== null && selection !== null) {
                $insertDataTransferForRichText(clipboardData, selection, editor);
            }
        },
        {
            tag: "paste",
        },
    );
};

export function $copyNode<T extends LexicalNode>(node: T): T {
    const copy = (node.constructor as LexicalNodeClass).clone(node);
    copy.__key = $createNodeKey(copy, null);
    return copy as T;
}

export function $splitNode(
    node: ElementNode,
    offset: number,
): [ElementNode | null, ElementNode] {
    let startNode = node.getChildAtIndex(offset);
    if (startNode === null) {
        startNode = node;
    }

    if ($isRootOrShadowRoot(node)) {
        throw new Error("Can not call $splitNode() on root element");
    }

    const recurse = <T extends LexicalNode>(
        currentNode: T,
    ): [ElementNode, ElementNode, T] => {
        const parent = getParent(currentNode, { throwIfNull: true });
        const isParentRoot = $isRootOrShadowRoot(parent);
        // The node we start split from (leaf) is moved, but its recursive
        // parents are copied to create separate tree
        const nodeToMove =
            currentNode === startNode && !isParentRoot
                ? currentNode
                : $copyNode(currentNode);

        if (isParentRoot) {
            if (!$isNode("Element", currentNode) || !$isNode("Element", nodeToMove)) {
                throw new Error("Children of a root must be ElementNode");
            }

            currentNode.insertAfter(nodeToMove);
            return [currentNode, nodeToMove, nodeToMove];
        } else {
            const [leftTree, rightTree, newParent] = recurse(parent);
            const nextSiblings = getNextSiblings(currentNode);

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
            if (focusChild === null) {
                focusNode.append(node);
            } else {
                focusChild.insertBefore(node);
            }
            node.selectNext();
        } else {
            let splitNode: ElementNode;
            let splitOffset: number;
            if ($isNode("Text", focusNode)) {
                splitNode = getParent(focusNode, { throwIfNull: true });
                splitOffset = getIndexWithinParent(focusNode);
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
        if (selection !== null) {
            const nodes = selection.getNodes();
            getTopLevelElementOrThrow(nodes[nodes.length - 1]).insertAfter(node);
        } else {
            const root = $getRoot();
            root.append(node);
        }
        const paragraphNode = $createNode("Paragraph", {});
        node.insertAfter(paragraphNode);
        paragraphNode.select();
    }
    return node.getLatest();
};

const getConversionFunction = (domNode: Node): DOMConversionFn | null => {
    const { nodeName } = domNode;

    const cachedConversions = LexicalNodes.getConversionsForTag(nodeName.toLowerCase());

    let currentConversion: DOMConversion | null = null;

    if (cachedConversions !== undefined) {
        for (const cachedConversion of cachedConversions) {
            const domConversion = cachedConversion(domNode as HTMLElement);

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
};

const $createNodesFromDOM = (
    node: Node,
    forChildMap: Map<string, DOMChildConversion> = new Map(),
    parentLexicalNode?: LexicalNode | null | undefined,
): Array<LexicalNode> => {
    let lexicalNodes: Array<LexicalNode> = [];

    if (IGNORE_TAGS.has(node.nodeName)) {
        return lexicalNodes;
    }

    let currentLexicalNode: LexicalNode | null | undefined = null;
    const transformFunction = getConversionFunction(node);
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

        if (transformOutput.forChild !== null && transformOutput.forChild !== undefined) {
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
                new Map(forChildMap),
                currentLexicalNode,
            ),
        );
    }

    if (postTransform !== null && postTransform !== undefined) {
        childLexicalNodes = postTransform(childLexicalNodes);
    }

    if (currentLexicalNode === null) {
        // If it hasn't been converted to a LexicalNode, we hoist its children
        // up to the same level as it.
        lexicalNodes = lexicalNodes.concat(childLexicalNodes);
    } else {
        if ($isNode("Element", currentLexicalNode)) {
            // If the current node is a ElementNode after conversion,
            // we can append all the children to it.
            currentLexicalNode.append(...childLexicalNodes);
        }
    }

    return lexicalNodes;
};

/**
 * How you parse your html string to get a document is left up to you. In the browser you can use the native
 * DOMParser API to generate a document (see clipboard.ts), but to use in a headless environment you can use JSDom
 * or an equivalent library and pass in the document here.
 */
export const $generateNodesFromDOM = (dom: Document): Array<LexicalNode> => {
    const elements = dom.body ? dom.body.childNodes : [];
    let lexicalNodes: Array<LexicalNode> = [];
    for (let i = 0; i < elements.length; i++) {
        const element = elements[i];
        if (!IGNORE_TAGS.has(element.nodeName)) {
            const lexicalNode = $createNodesFromDOM(element);
            if (lexicalNode !== null) {
                lexicalNodes = lexicalNodes.concat(lexicalNode);
            }
        }
    }

    return lexicalNodes;
};

const $updateElementNodeProperties = <T extends ElementNode>(
    target: T,
    source: ElementNode,
): T => {
    target.__first = source.__first;
    target.__last = source.__last;
    target.__size = source.__size;
    target.__format = source.__format;
    target.__indent = source.__indent;
    target.__dir = source.__dir;
    return target;
};

const $updateTextNodeProperties = <T extends TextNode>(
    target: T,
    source: TextNode,
): T => {
    target.__format = source.__format;
    target.__style = source.__style;
    target.__mode = source.__mode;
    target.__detail = source.__detail;
    return target;
};

const $updateParagraphNodeProperties = <T extends ParagraphNode>(
    target: T,
    source: ParagraphNode,
): T => {
    target.__textFormat = source.__textFormat;
    return target;
};


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

    if ($isNode("Element", node) && $isNode("Element", clone)) {
        return $updateElementNodeProperties(clone, node);
    }

    if ($isNode("Text", node) && $isNode("Text", clone)) {
        return $updateTextNodeProperties(clone, node);
    }

    if ($isNode("Paragraph", node) && $isNode("Paragraph", clone)) {
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
        isSelected(textNode, selection) &&
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

export function $createNode<K extends keyof NodeConstructorPayloads>(
    nodeType: K,
    params: NodeConstructorPayloads[K],
): InstanceType<NodeConstructors[K]> {
    const NodeClass = LexicalNodes.get(nodeType);
    if (!NodeClass) {
        throw new Error(`No constructor found for node type: ${nodeType}`);
    }
    const node = new NodeClass(params);
    return $applyNodeReplacement(node) as InstanceType<NodeConstructors[K]>;
}

export function $isNode<K extends keyof NodeConstructors>(
    nodeType: K,
    node: LexicalNode | null | undefined,
): node is InstanceType<NodeConstructors[K]> {
    const NodeClass = LexicalNodes.get(nodeType);
    return !!node && NodeClass ? node instanceof NodeClass : false;
}

export function removeNode(
    nodeToRemove: LexicalNode,
    restoreSelection: boolean,
    preserveEmptyParent?: boolean,
) {
    errorOnReadOnly();
    const key = nodeToRemove.__key;
    const parent = getParent(nodeToRemove);
    if (parent === null) {
        return;
    }
    const selection = $maybeMoveChildrenSelectionToParent(nodeToRemove);
    let selectionMoved = false;
    if ($isRangeSelection(selection) && restoreSelection) {
        const anchor = selection.anchor;
        const focus = selection.focus;
        if (anchor.key === key) {
            moveSelectionPointToSibling(
                anchor,
                nodeToRemove,
                parent,
                getPreviousSibling(nodeToRemove),
                getNextSibling(nodeToRemove),
            );
            selectionMoved = true;
        }
        if (focus.key === key) {
            moveSelectionPointToSibling(
                focus,
                nodeToRemove,
                parent,
                getPreviousSibling(nodeToRemove),
                getNextSibling(nodeToRemove),
            );
            selectionMoved = true;
        }
    } else if (
        $isNodeSelection(selection) &&
        restoreSelection &&
        isSelected(nodeToRemove)
    ) {
        nodeToRemove.selectPrevious();
    }

    if ($isRangeSelection(selection) && restoreSelection && !selectionMoved) {
        const index = getIndexWithinParent(nodeToRemove);
        removeFromParent(nodeToRemove);
        $updateElementSelectionOnCreateDeleteNode(selection, parent, index, -1);
    } else {
        removeFromParent(nodeToRemove);
    }

    if (
        !preserveEmptyParent &&
        !$isRootOrShadowRoot(parent) &&
        !parent.canBeEmpty() &&
        parent.isEmpty()
    ) {
        removeNode(parent, restoreSelection);
    }
    if (restoreSelection && $isNode("Root", parent) && parent.isEmpty()) {
        parent.selectEnd();
    }
}

/**
 * Returns true if there is a path between the provided node and the RootNode, false otherwise.
 * This is a way of determining if the node is "attached" EditorState. Unattached nodes
 * won't be reconciled and will ultimately be cleaned up by the Lexical GC.
 */
export function isAttachedToRoot(node: LexicalNode): boolean {
    let nodeKey: string | null = node.__key;
    while (nodeKey !== null) {
        // Base case - root node
        if (nodeKey === "root") {
            return true;
        }

        // Traverse up parent chain
        const node: LexicalNode | null = $getNodeByKey(nodeKey);
        if (node === null) {
            break;
        }
        nodeKey = node.__parent;
    }
    return false;
}

/**
 * Returns true if the provided node is contained within the provided Selection, false otherwise.
 * Relies on the algorithms implemented in {@link BaseSelection.getNodes} to determine
 * what's included.
 *
 * @param selection - The selection that we want to determine if the node is in.
 */
export function isSelected(node: LexicalNode, selection?: null | BaseSelection): boolean {
    const targetSelection = selection || $getSelection();
    if (targetSelection === null) {
        return false;
    }

    const isSelected = targetSelection
        .getNodes()
        .some((n) => n.__key === node.__key);

    if ($isNode("Text", node)) {
        return isSelected;
    }
    // For inline images inside of element nodes.
    // Without this change the image will be selected if the cursor is before or after it.
    if (
        $isRangeSelection(targetSelection) &&
        targetSelection.anchor.type === "element" &&
        targetSelection.focus.type === "element" &&
        targetSelection.anchor.key === targetSelection.focus.key &&
        targetSelection.anchor.offset === targetSelection.focus.offset
    ) {
        return false;
    }
    return isSelected;
}

/**
 * Returns the zero-based index of the provided node within the parent.
 * TODO This is O(n) and can be improved.
 */
export function getIndexWithinParent(node: LexicalNode): number {
    const parent = getParent(node);
    if (parent === null) {
        return -1;
    }
    let currChild = parent.getFirstChild();
    let index = 0;
    while (currChild !== null) {
        if (node.__key === currChild.__key) {
            return index;
        }
        index++;
        currChild = getNextSibling(currChild);
    }
    return -1;
}

/**
 * Returns the parent of the provided node, or null if none is found.
 */
export function getParent<T extends ElementNode, E extends boolean = false>(
    node: LexicalNode,
    options?: {
        /** Will skip the provided number of parents (e.g. 1 will return the grandparent) */
        skip?: number,
        /** Will throw an error if the parent is not found */
        throwIfNull?: E
    },
): E extends true ? T : T | null {
    let current: LexicalNode | null = node.getLatest();
    let skip = options?.skip ?? 0;

    while (skip >= 0 && current) {
        const parentKey = current.__parent;
        current = parentKey ? $getNodeByKey<T>(parentKey) : null;
        if (current === null) {
            if (skip === 0 && options?.throwIfNull) {
                throw new Error(`Expected node ${node.__key} to have a parent.`);
            }
            break;
        }
        skip--;
    }

    return current as E extends true ? T : T | null;
}

/**
 * Returns the highest (in the EditorState tree)
 * non-root ancestor of the provided node, or null if none is found. See {@link lexical!$isRootOrShadowRoot}
 * for more information on which Elements comprise "roots".
 */
export function getTopLevelElement(node: LexicalNode): ElementNode | null {
    let currNode = getParent(node);
    while (currNode !== null) {
        if ($isRootOrShadowRoot(currNode)) {
            return currNode;
        }
        currNode = getParent(currNode);
    }
    return null;
}

/**
 * Like `getTopLevelElement`, but throws an error if no parent is found.
 */
export function getTopLevelElementOrThrow(node: LexicalNode): ElementNode {
    const parent = getTopLevelElement(node);
    if (parent === null) {
        throw new Error(`Expected node ${node.__key} to have a top parent element.`);
    }
    return parent;
}

const MAX_PARENT_SEARCH_DEPTH = 25;
/**
 * Returns a list of the every ancestor of the provided node,
 * all the way up to the RootNode.
 */
export const getParents = (node: LexicalNode): ElementNode[] => {
    const parents: ElementNode[] = [];
    let currNode = getParent(node);
    while (currNode && parents.length < MAX_PARENT_SEARCH_DEPTH) {
        parents.push(currNode);
        currNode = getParent(currNode);
    }
    return parents;
};

/**
 * Returns a list of the keys of every ancestor of this node,
 * all the way up to the RootNode.
 */
export const getParentKeys = (node: LexicalNode): NodeKey[] => {
    return getParents(node).map((parent) => parent.__key);
};

/**
 * Returns the "previous" siblings - that is, the node that comes
 * before this one in the same parent.
 */
export const getPreviousSibling = <T extends LexicalNode>(node: LexicalNode): T | null => {
    const self = node.getLatest();
    const prevKey = self.__prev;
    return prevKey === null ? null : $getNodeByKey<T>(prevKey);
};

/**
 * Returns the "previous" siblings - that is, the nodes that come between
 * this one and the first child of it's parent, inclusive.
 */
export const getPreviousSiblings = <T extends LexicalNode>(node: LexicalNode): Array<T> => {
    const siblings: T[] = [];
    const parent = getParent(node);
    if (parent === null) {
        return siblings;
    }
    let currNode: null | T = parent.getFirstChild();
    while (currNode !== null) {
        if (currNode.is(this)) {
            break;
        }
        siblings.push(currNode);
        currNode = getNextSibling(currNode);
    }
    return siblings;
};

/**
 * Returns the "next" siblings - that is, the node that comes
 * after this one in the same parent
 *
 */
export const getNextSibling = <T extends LexicalNode>(node: LexicalNode): T | null => {
    const self = node.getLatest();
    const nextKey = self.__next;
    return nextKey === null ? null : $getNodeByKey<T>(nextKey);
};

/**
 * Returns all "next" siblings - that is, the nodes that come between this
 * one and the last child of it's parent, inclusive.
 *
 */
export const getNextSiblings = <T extends LexicalNode>(node: LexicalNode): Array<T> => {
    const siblings: Array<T> = [];
    let currNode: null | T = getNextSibling(node);
    while (currNode !== null) {
        siblings.push(currNode);
        currNode = getNextSibling(currNode);
    }
    return siblings;
};

/**
 * Returns the closest common ancestor of the two nodes, or null if none is found.
 *
 * @param node - the other node to find the common ancestor of.
 */
export const getCommonAncestor = <T extends ElementNode = ElementNode>(
    nodeA: LexicalNode,
    nodeB: LexicalNode,
): T | null => {
    const a = getParents(nodeA);
    const b = getParents(nodeB);
    if ($isNode("Element", nodeA)) {
        a.unshift(nodeA);
    }
    if ($isNode("Element", nodeB)) {
        b.unshift(nodeB);
    }
    const aLength = a.length;
    const bLength = b.length;
    if (aLength === 0 || bLength === 0 || a[aLength - 1] !== b[bLength - 1]) {
        return null;
    }
    const bSet = new Set(b);
    for (let i = 0; i < aLength; i++) {
        const ancestor = a[i] as T;
        if (bSet.has(ancestor)) {
            return ancestor;
        }
    }
    return null;
};

