/* eslint-disable @typescript-eslint/ban-ts-comment */
import { headerMarkdowns } from "utils/display/stringTools";
import { DOUBLE_LINE_BREAK, FULL_RECONCILE, IS_ALIGN_CENTER, IS_ALIGN_END, IS_ALIGN_JUSTIFY, IS_ALIGN_LEFT, IS_ALIGN_RIGHT, IS_ALIGN_START } from "./consts";
import { EditorState, LexicalEditor } from "./editor";
import { ElementNode } from "./nodes/ElementNode";
import { CustomDomElement, EditorConfig, IntentionallyMarkedAsDirtyElement, MutatedNodes, NodeKey, NodeMap, RegisteredNodes } from "./types";
import { $isNode, $textContentRequiresDoubleLinebreakAtEnd, cloneDecorators, getElementByKeyOrThrow, getTextDirection, setMutatedNode } from "./utils";

// Subtree information
/** Text representation of the current subtree being reconciled */
let subTreeTextContent = "";
/** Directioned text representation of the current subtree being reconciled */
let subTreeDirectionedTextContent = "";
/**
 * Alignment (e.g. left, right, center) if it's an element node, 
 * or its formatting (e.g. bold, italic) if it's a text node.
 */
let subTreeTextFormat: number | null = null;
/** Markdown representation of the current subtree being reconciled */
let subTreeMarkdownContent = "";

// Editor information
let editorTextContent = "";
let editorMarkdownContent = "";
let activeEditorConfig: EditorConfig;
let activeEditor: LexicalEditor;
let activeEditorNodes: RegisteredNodes | undefined;
let treatAllNodesAsDirty = false;
let activeEditorStateReadOnly = false;
let activeTextDirection: "ltr" | "rtl" | null = null;
let activeDirtyElements: Map<NodeKey, IntentionallyMarkedAsDirtyElement> | undefined;
let activeDirtyLeaves: Set<NodeKey> | undefined;
let activePrevNodeMap: NodeMap;
let activeNextNodeMap: NodeMap;
let activePrevKeyToDOMMap: Map<NodeKey, HTMLElement>;
let mutatedNodes: MutatedNodes;

/**
 * Removes a node (and its children) from the DOM and the keyToDOMMap. 
 * Does not delete any LexicalNode information (meaning we could re-create the DOM later).
 * @param key The key of the node to destroy
 * @param parentDOM The parent DOM of the node to destroy 
 */
const destroyNode = (key: NodeKey, parentDOM: HTMLElement | null): void => {
    const node = activePrevNodeMap.get(key);

    if (parentDOM) {
        const dom = getPrevElementByKeyOrThrow(key);
        if (dom.parentNode === parentDOM) {
            parentDOM.removeChild(dom);
        }
    }

    // This logic is really important, otherwise we will leak DOM nodes
    // when their corresponding LexicalNodes are removed from the editor state.
    if (!activeNextNodeMap.has(key)) {
        activeEditor._keyToDOMMap.delete(key);
    }

    if ($isNode("Element", node)) {
        const children = createChildrenArray(node, activePrevNodeMap);
        destroyChildren(children, 0, children.length - 1, null);
    }

    if (node) {
        setMutatedNode(mutatedNodes, activeEditorNodes, node, "destroyed");
    }
};

/**
 * Removes all children nodes from the DOM and the keyToDOMMap 
 * (including the children's children, and so on).
 * Does not delete any LexicalNode information (meaning we could re-create the DOM later).
 * @param children The keys of the children nodes to destroy
 * @param _startIndex The index of the first child to destroy
 * @param endIndex The index of the last child to destroy
 * @param dom The parent DOM of the children to destroy
 */
const destroyChildren = (
    children: NodeKey[],
    _startIndex: number,
    endIndex: number,
    dom: HTMLElement | null,
): void => {
    let startIndex = _startIndex;

    for (; startIndex <= endIndex; ++startIndex) {
        const child = children[startIndex];

        if (child !== undefined) {
            destroyNode(child, dom);
        }
    }
};

const DEFAULT_INDENT_VALUE = "40px";

/**
 * Applies indentation to an element node
 * @param dom The element node to apply indentation to
 * @param indent The amount of indentation to apply
 */
const setElementIndent = (dom: HTMLElement, indent: number): void => {
    const indentationBaseValue =
        getComputedStyle(dom).getPropertyValue("--lexical-indent-base-value") ||
        DEFAULT_INDENT_VALUE;

    dom.style.setProperty(
        "padding-inline-start",
        indent === 0 ? "" : `calc(${indent} * ${indentationBaseValue})`,
    );
};

/**
 * Applies text alignment to an element node
 * @param dom The element node to apply text alignment to
 * @param format The number representing the text alignment
 */
const setElementFormat = (dom: HTMLElement, format: number): void => {
    const domStyle = dom.style;
    const setTextAlign = (domStyle: CSSStyleDeclaration, value: string) => {
        domStyle.setProperty("text-align", value);
    };

    if (format === 0) {
        setTextAlign(domStyle, "");
    } else if (format === IS_ALIGN_LEFT) {
        setTextAlign(domStyle, "left");
    } else if (format === IS_ALIGN_CENTER) {
        setTextAlign(domStyle, "center");
    } else if (format === IS_ALIGN_RIGHT) {
        setTextAlign(domStyle, "right");
    } else if (format === IS_ALIGN_JUSTIFY) {
        setTextAlign(domStyle, "justify");
    } else if (format === IS_ALIGN_START) {
        setTextAlign(domStyle, "start");
    } else if (format === IS_ALIGN_END) {
        setTextAlign(domStyle, "end");
    }
};

/**
 * Create a new DOM node for a LexicalNode
 * @param key The key of the LexicalNode
 * @param parentDOM The DOM to insert the new node into
 * @param insertDOM A child of the parentDOM to insert the new node before
 * @returns The newly created DOM node
 */
const createNode = (
    key: NodeKey,
    parentDOM: HTMLElement | null,
    insertDOM: Node | null,
): HTMLElement => {
    // Get node information
    const node = activeNextNodeMap.get(key);
    if (!node) {
        throw new Error("createNode: node does not exist in nodeMap");
    }

    // Create the DOM element for the node, and store it in the keyToDOMMap
    const dom = node.createDOM();
    storeDOMWithKey(key, dom, activeEditor);

    // This helps preserve the text, and stops spell check tools from merging or 
    // break the spans (which happens if they are missing this attribute).
    if ($isNode("Text", node)) {
        dom.setAttribute("data-lexical-text", "true");
    } else if ($isNode("Decorator", node)) {
        dom.setAttribute("data-lexical-decorator", "true");
    }

    // If it's an element node, apply styling such as indentation and alignment
    if ($isNode("Element", node)) {
        const indent = node.__indent;
        const childrenSize = node.__size;

        if ($isNode("Heading", node)) {
            const markdown = headerMarkdowns[node.__tag];
            subTreeMarkdownContent += markdown;
            editorMarkdownContent += markdown;
        }

        if (indent !== 0) {
            setElementIndent(dom, indent);
        }
        if (childrenSize !== 0) {
            const endIndex = childrenSize - 1;
            const children = createChildrenArray(node, activeNextNodeMap);
            createChildrenWithDirection(children, endIndex, node, dom);
        }
        const format = node.__format;

        if (format !== 0) {
            setElementFormat(dom, format);
        }
        if (!node.isInline()) {
            reconcileElementTerminatingLineBreak(null, node, dom);
        }

        if ($textContentRequiresDoubleLinebreakAtEnd(node)) {
            subTreeTextContent += DOUBLE_LINE_BREAK;
            subTreeMarkdownContent += DOUBLE_LINE_BREAK;
            editorTextContent += DOUBLE_LINE_BREAK;
            editorMarkdownContent += DOUBLE_LINE_BREAK;
        }
    }
    // If it's a text or decorator node, track text and markdown content 
    else {
        const text = node.getTextContent();
        const markdown = node.getMarkdownContent();

        // Apply decorator
        if ($isNode("Decorator", node)) {
            const decorator = node.decorate(activeEditor, activeEditorConfig);

            if (decorator !== null) {
                reconcileDecorator(key, decorator);
            }
            // Decorators are always non editable
            dom.contentEditable = "false";
        } else if ($isNode("Text", node)) {
            if (!node.isDirectionless()) {
                subTreeDirectionedTextContent += text;
            }
        }
        subTreeTextContent += text;
        subTreeMarkdownContent += markdown;
        editorTextContent += text;
        editorMarkdownContent += markdown;
    }

    // Insert into the parent DOM if it exists
    if (parentDOM) {
        if (insertDOM) {
            parentDOM.insertBefore(dom, insertDOM);
        } else {
            const possibleLineBreak = (parentDOM as CustomDomElement).__lexicalLineBreak;

            if (possibleLineBreak) {
                parentDOM.insertBefore(dom, possibleLineBreak);
            } else {
                parentDOM.appendChild(dom);
            }
        }
    }

    // Mark the node as mutated for reconciliation
    setMutatedNode(mutatedNodes, activeEditorNodes, node, "created");

    // Return the DOM element we created
    return dom;
};

function createChildrenWithDirection(
    children: NodeKey[],
    endIndex: number,
    element: ElementNode,
    dom: HTMLElement,
): void {
    const previousSubTreeDirectionedTextContent = subTreeDirectionedTextContent;
    subTreeDirectionedTextContent = "";
    createChildren(children, element, 0, endIndex, dom, null);
    reconcileBlockDirection(element, dom);
    subTreeDirectionedTextContent = previousSubTreeDirectionedTextContent;
}

const createChildren = (
    children: NodeKey[],
    element: ElementNode,
    _startIndex: number,
    endIndex: number,
    dom: HTMLElement | null,
    insertDOM: HTMLElement | null,
): void => {
    const previousSubTreeTextContent = subTreeTextContent;
    const previousSubTreeMarkdownContent = subTreeMarkdownContent;
    subTreeTextContent = "";
    subTreeMarkdownContent = "";
    let startIndex = _startIndex;

    for (; startIndex <= endIndex; ++startIndex) {
        createNode(children[startIndex], dom, insertDOM);
        const node = activeNextNodeMap.get(children[startIndex]);
        if (node && subTreeTextFormat === null && $isNode("Text", node)) {
            subTreeTextFormat = node.getFormat();
        }
    }
    if ($textContentRequiresDoubleLinebreakAtEnd(element)) {
        subTreeTextContent += DOUBLE_LINE_BREAK;
        subTreeMarkdownContent += DOUBLE_LINE_BREAK;
    }
    (dom as CustomDomElement).__lexicalTextContent = subTreeTextContent;
    (dom as CustomDomElement).__lexicalMarkdownContent = subTreeMarkdownContent;
    subTreeTextContent = previousSubTreeTextContent + subTreeTextContent;
    subTreeMarkdownContent = previousSubTreeMarkdownContent + subTreeMarkdownContent;
};

const isLastChildLineBreakOrDecorator = (
    childKey: NodeKey,
    nodeMap: NodeMap,
): boolean => {
    const node = nodeMap.get(childKey);
    return $isNode("LineBreak", node) || ($isNode("Decorator", node) && node.isInline());
};

// If we end an element with a LineBreakNode, then we need to add an additional <br>
const reconcileElementTerminatingLineBreak = (
    prevElement: null | ElementNode,
    nextElement: ElementNode,
    dom: HTMLElement,
): void => {
    const prevLineBreak =
        prevElement !== null &&
        (prevElement.__size === 0 ||
            isLastChildLineBreakOrDecorator(
                prevElement.__last as NodeKey,
                activePrevNodeMap,
            ));
    const nextLineBreak =
        nextElement.__size === 0 ||
        isLastChildLineBreakOrDecorator(
            nextElement.__last as NodeKey,
            activeNextNodeMap,
        );

    if (prevLineBreak) {
        if (!nextLineBreak) {
            const element = (dom as CustomDomElement).__lexicalLineBreak;

            if (element) {
                dom.removeChild(element);
            }

            (dom as CustomDomElement).__lexicalLineBreak = null;
        }
    } else if (nextLineBreak) {
        const element = document.createElement("br");
        (dom as CustomDomElement).__lexicalLineBreak = element;
        dom.appendChild(element);
    }
};

const reconcileParagraphFormat = (element: ElementNode): void => {
    if (
        $isNode("Paragraph", element) &&
        subTreeTextFormat !== null &&
        subTreeTextFormat !== element.__textFormat
    ) {
        element.setTextFormat(subTreeTextFormat);
    }
};

const reconcileBlockDirection = (element: ElementNode, dom: HTMLElement): void => {
    const previousSubTreeDirectionTextContent = (dom as CustomDomElement).__lexicalDirTextContent;
    const previousDirection = (dom as CustomDomElement).__lexicalDir;

    if (
        previousSubTreeDirectionTextContent !== subTreeDirectionedTextContent ||
        previousDirection !== activeTextDirection
    ) {
        const hasEmptyDirectionedTextContent = subTreeDirectionedTextContent === "";
        const direction = hasEmptyDirectionedTextContent
            ? activeTextDirection
            : getTextDirection(subTreeDirectionedTextContent);

        if (direction !== previousDirection) {
            if (
                direction === null ||
                (hasEmptyDirectionedTextContent && direction === "ltr")
            ) {
                // Remove direction
                dom.removeAttribute("dir");
            } else {
                // Update direction
                dom.dir = direction;
            }

            if (!activeEditorStateReadOnly) {
                const writableNode = element.getWritable();
                writableNode.__dir = direction;
            }
        }

        activeTextDirection = direction;
        (dom as CustomDomElement).__lexicalDirTextContent = subTreeDirectionedTextContent;
        (dom as CustomDomElement).__lexicalDir = direction;
    }
};

const reconcileChildrenWithDirection = (
    prevElement: ElementNode,
    nextElement: ElementNode,
    dom: HTMLElement,
): void => {
    const previousSubTreeDirectionTextContent = subTreeDirectionedTextContent;
    subTreeDirectionedTextContent = "";
    subTreeTextFormat = null;
    reconcileChildren(prevElement, nextElement, dom);
    reconcileBlockDirection(nextElement, dom);
    reconcileParagraphFormat(nextElement);
    subTreeDirectionedTextContent = previousSubTreeDirectionTextContent;
    subTreeTextFormat = null;
};

const createChildrenArray = (
    element: ElementNode,
    nodeMap: NodeMap,
): NodeKey[] => {
    const children: string[] = [];
    let nodeKey = element.__first;
    while (nodeKey) {
        const node = nodeMap.get(nodeKey);
        if (!node) {
            throw new Error("createChildrenArray: node does not exist in nodeMap");
        }
        children.push(nodeKey);
        nodeKey = node.__next;
    }
    return children;
};

const reconcileChildren = (
    prevElement: ElementNode,
    nextElement: ElementNode,
    dom: HTMLElement,
): void => {
    const previousSubTreeTextContent = subTreeTextContent;
    const previousSubTreeMarkdownContent = subTreeMarkdownContent;
    const prevChildrenSize = prevElement.__size;
    const nextChildrenSize = nextElement.__size;
    subTreeTextContent = "";
    subTreeMarkdownContent = "";

    if (prevChildrenSize === 1 && nextChildrenSize === 1) {
        const prevFirstChildKey = prevElement.__first as NodeKey;
        const nextFrstChildKey = nextElement.__first as NodeKey;
        if (prevFirstChildKey === nextFrstChildKey) {
            reconcileNode(prevFirstChildKey, dom);
        } else {
            const lastDOM = getPrevElementByKeyOrThrow(prevFirstChildKey);
            const replacementDOM = createNode(nextFrstChildKey, null, null);
            dom.replaceChild(replacementDOM, lastDOM);
            destroyNode(prevFirstChildKey, null);
        }
        const nextChildNode = activeNextNodeMap.get(nextFrstChildKey);
        if (subTreeTextFormat === null && $isNode("Text", nextChildNode)) {
            subTreeTextFormat = nextChildNode.getFormat();
        }
    } else {
        const prevChildren = createChildrenArray(prevElement, activePrevNodeMap);
        const nextChildren = createChildrenArray(nextElement, activeNextNodeMap);

        if (prevChildrenSize === 0) {
            if (nextChildrenSize !== 0) {
                createChildren(
                    nextChildren,
                    nextElement,
                    0,
                    nextChildrenSize - 1,
                    dom,
                    null,
                );
            }
        } else if (nextChildrenSize === 0) {
            if (prevChildrenSize !== 0) {
                const lexicalLineBreak = (dom as CustomDomElement).__lexicalLineBreak;
                const canUseFastPath = lexicalLineBreak === null;
                destroyChildren(
                    prevChildren,
                    0,
                    prevChildrenSize - 1,
                    canUseFastPath ? null : dom,
                );

                if (canUseFastPath) {
                    // Fast path for removing DOM nodes
                    dom.textContent = "";
                }
            }
        } else {
            reconcileNodeChildren(
                nextElement,
                prevChildren,
                nextChildren,
                prevChildrenSize,
                nextChildrenSize,
                dom,
            );
        }
    }

    if ($textContentRequiresDoubleLinebreakAtEnd(nextElement)) {
        subTreeTextContent += DOUBLE_LINE_BREAK;
        subTreeMarkdownContent += DOUBLE_LINE_BREAK;
    }

    (dom as CustomDomElement).__lexicalTextContent = subTreeTextContent;
    (dom as CustomDomElement).__lexicalMarkdownContent = subTreeMarkdownContent;
    subTreeTextContent = previousSubTreeTextContent + subTreeTextContent;
    subTreeMarkdownContent = previousSubTreeMarkdownContent + subTreeMarkdownContent;
};

const reconcileNode = (
    key: NodeKey,
    parentDOM: HTMLElement | null,
): HTMLElement => {
    const prevNode = activePrevNodeMap.get(key);
    let nextNode = activeNextNodeMap.get(key);

    if (!prevNode || !nextNode) {
        throw new Error("reconcileNode: prevNode or nextNode does not exist in nodeMap");
    }

    const isDirty =
        treatAllNodesAsDirty ||
        (activeDirtyLeaves && activeDirtyLeaves.has(key)) ||
        (activeDirtyElements && activeDirtyElements.has(key));
    const dom = getElementByKeyOrThrow(activeEditor, key);

    // If the node key points to the same instance in both states
    // and isn't dirty, we just update the text content cache
    // and return the existing DOM Node.
    if (prevNode === nextNode && !isDirty) {
        if ($isNode("Element", prevNode)) {
            const {
                __lexicalTextContent: previousSubTreeTextContent,
                __lexicalMarkdownContent: previousSubTreeMarkdownContent,
            } = (dom as CustomDomElement);

            if ($isNode("Heading", prevNode)) {
                const markdown = headerMarkdowns[prevNode.__tag];
                subTreeMarkdownContent += markdown;
                editorMarkdownContent += markdown;
            }

            if (previousSubTreeTextContent) {
                subTreeTextContent += previousSubTreeTextContent;
                editorTextContent += previousSubTreeTextContent;
            }
            if (previousSubTreeMarkdownContent) {
                subTreeMarkdownContent += previousSubTreeMarkdownContent;
                editorMarkdownContent += previousSubTreeMarkdownContent;
            }

            const previousSubTreeDirectionTextContent = (dom as CustomDomElement).__lexicalDirTextContent;

            if (previousSubTreeDirectionTextContent) {
                subTreeDirectionedTextContent += previousSubTreeDirectionTextContent;
            }
        } else {
            const text = prevNode.getTextContent();
            const markdown = prevNode.getMarkdownContent();

            if ($isNode("Text", prevNode) && !prevNode.isDirectionless()) {
                subTreeDirectionedTextContent += text;
            }

            editorTextContent += text;
            subTreeTextContent += text;
            editorMarkdownContent += markdown;
            subTreeMarkdownContent += markdown;
        }

        return dom;
    }
    // If the node key doesn't point to the same instance in both maps,
    // it means it were cloned. If they're also dirty, we mark them as mutated.
    if (prevNode !== nextNode && isDirty) {
        setMutatedNode(mutatedNodes, activeEditorNodes, nextNode, "updated");
    }

    // Update node. If it returns true, we need to unmount and re-create the node
    if (nextNode.updateDOM(prevNode, dom, activeEditorConfig)) {
        const replacementDOM = createNode(key, null, null);

        if (parentDOM === null) {
            throw new Error("reconcileNode: parentDOM is null");
        }

        parentDOM.replaceChild(replacementDOM, dom);
        destroyNode(key, null);
        return replacementDOM;
    }

    if ($isNode("Element", prevNode) && $isNode("Element", nextNode)) {
        // Reconcile element children
        const nextIndent = nextNode.__indent;

        if (nextIndent !== prevNode.__indent) {
            setElementIndent(dom, nextIndent);
        }

        const nextFormat = nextNode.__format;

        if (nextFormat !== prevNode.__format) {
            setElementFormat(dom, nextFormat);
        }

        if ($isNode("Heading", nextNode)) {
            const markdown = headerMarkdowns[nextNode.__tag];
            subTreeMarkdownContent += markdown;
            editorMarkdownContent += markdown;
        }

        if (isDirty) {
            reconcileChildrenWithDirection(prevNode, nextNode, dom);
            if (!$isNode("Root", nextNode) && !nextNode.isInline()) {
                reconcileElementTerminatingLineBreak(prevNode, nextNode, dom);
            }
        }

        if ($textContentRequiresDoubleLinebreakAtEnd(nextNode)) {
            subTreeTextContent += DOUBLE_LINE_BREAK;
            subTreeMarkdownContent += DOUBLE_LINE_BREAK;
            editorTextContent += DOUBLE_LINE_BREAK;
            editorMarkdownContent += DOUBLE_LINE_BREAK;
        }
    } else {
        const text = nextNode.getTextContent();
        const markdown = nextNode.getMarkdownContent();

        if ($isNode("Decorator", nextNode)) {
            const decorator = nextNode.decorate(activeEditor, activeEditorConfig);

            if (decorator !== null) {
                reconcileDecorator(key, decorator);
            }
        } else if ($isNode("Text", nextNode) && !nextNode.isDirectionless()) {
            // Handle text content, for LTR, RTL cases.
            subTreeDirectionedTextContent += text;
        }

        subTreeTextContent += text;
        editorTextContent += text;
        subTreeMarkdownContent += markdown;
        editorMarkdownContent += markdown;
    }

    if (
        !activeEditorStateReadOnly &&
        $isNode("Root", nextNode) &&
        nextNode.__cachedText !== editorTextContent
    ) {
        // Cache the latest text and markdown content.
        const nextRootNode = nextNode.getWritable();
        nextRootNode.__cachedText = editorTextContent;
        nextRootNode.__cachedMarkdown = editorMarkdownContent;
        nextNode = nextRootNode;
    }

    return dom;
};

const reconcileDecorator = (key: NodeKey, decorator: unknown): void => {
    let pendingDecorators = activeEditor._pendingDecorators || {};
    const currentDecorators = activeEditor._decorators;

    if (pendingDecorators === null) {
        if (currentDecorators[key] === decorator) {
            return;
        }

        pendingDecorators = cloneDecorators(activeEditor);
    }

    pendingDecorators[key] = decorator;
};

const getFirstChild = (element: HTMLElement): Node | null => {
    return element.firstChild;
};

const getNextSibling = (element: HTMLElement): Node | null => {
    let nextSibling = element.nextSibling;
    if (
        nextSibling !== null &&
        nextSibling === activeEditor._blockCursorElement
    ) {
        nextSibling = nextSibling.nextSibling;
    }
    return nextSibling;
};

const reconcileNodeChildren = (
    nextElement: ElementNode,
    prevChildren: NodeKey[],
    nextChildren: NodeKey[],
    prevChildrenLength: number,
    nextChildrenLength: number,
    dom: HTMLElement,
): void => {
    const prevEndIndex = prevChildrenLength - 1;
    const nextEndIndex = nextChildrenLength - 1;
    let prevChildrenSet: Set<NodeKey> | undefined;
    let nextChildrenSet: Set<NodeKey> | undefined;
    let siblingDOM: null | Node = getFirstChild(dom);
    let prevIndex = 0;
    let nextIndex = 0;

    while (prevIndex <= prevEndIndex && nextIndex <= nextEndIndex) {
        const prevKey = prevChildren[prevIndex];
        const nextKey = nextChildren[nextIndex];

        if (prevKey === nextKey) {
            siblingDOM = getNextSibling(reconcileNode(nextKey, dom));
            prevIndex++;
            nextIndex++;
        } else {
            if (prevChildrenSet === undefined) {
                prevChildrenSet = new Set(prevChildren);
            }

            if (nextChildrenSet === undefined) {
                nextChildrenSet = new Set(nextChildren);
            }

            const nextHasPrevKey = nextChildrenSet.has(prevKey);
            const prevHasNextKey = prevChildrenSet.has(nextKey);

            if (!nextHasPrevKey) {
                // Remove prev
                siblingDOM = getNextSibling(getPrevElementByKeyOrThrow(prevKey));
                destroyNode(prevKey, dom);
                prevIndex++;
            } else if (!prevHasNextKey) {
                // Create next
                createNode(nextKey, dom, siblingDOM);
                nextIndex++;
            } else {
                // Move next
                const childDOM = getElementByKeyOrThrow(activeEditor, nextKey);

                if (childDOM === siblingDOM) {
                    siblingDOM = getNextSibling(reconcileNode(nextKey, dom));
                } else {
                    if (siblingDOM !== null) {
                        dom.insertBefore(childDOM, siblingDOM);
                    } else {
                        dom.appendChild(childDOM);
                    }

                    reconcileNode(nextKey, dom);
                }

                prevIndex++;
                nextIndex++;
            }
        }

        const node = activeNextNodeMap.get(nextKey);
        if (node && subTreeTextFormat === null && $isNode("Text", node)) {
            subTreeTextFormat = node.getFormat();
        }
    }

    const appendNewChildren = prevIndex > prevEndIndex;
    const removeOldChildren = nextIndex > nextEndIndex;

    if (appendNewChildren && !removeOldChildren) {
        const previousNode = nextChildren[nextEndIndex + 1];
        const insertDOM = !previousNode ? null : activeEditor.getElementByKey(previousNode);
        createChildren(
            nextChildren,
            nextElement,
            nextIndex,
            nextEndIndex,
            dom,
            insertDOM,
        );
    } else if (removeOldChildren && !appendNewChildren) {
        destroyChildren(prevChildren, prevIndex, prevEndIndex, dom);
    }
};

export function reconcileRoot(
    prevEditorState: EditorState,
    nextEditorState: EditorState,
    editor: LexicalEditor,
    dirtyType: 0 | 1 | 2,
    dirtyElements: Map<NodeKey, IntentionallyMarkedAsDirtyElement>,
    dirtyLeaves: Set<NodeKey>,
): MutatedNodes {
    // We cache text content to make retrieval more efficient.
    // The cache must be rebuilt during reconciliation to account for any changes.
    subTreeTextContent = "";
    subTreeMarkdownContent = "";
    editorTextContent = "";
    editorMarkdownContent = "";
    subTreeDirectionedTextContent = "";
    // Rather than pass around a load of arguments through the stack recursively
    // we instead set them as bindings within the scope of the module.
    treatAllNodesAsDirty = dirtyType === FULL_RECONCILE;
    activeTextDirection = null;
    activeEditor = editor;
    activeEditorConfig = editor._config;
    activeEditorNodes = editor._nodes;
    activeDirtyElements = dirtyElements;
    activeDirtyLeaves = dirtyLeaves;
    activePrevNodeMap = prevEditorState._nodeMap;
    activeNextNodeMap = nextEditorState._nodeMap;
    activeEditorStateReadOnly = nextEditorState._readOnly;
    activePrevKeyToDOMMap = new Map(editor._keyToDOMMap);
    // We keep track of mutated nodes so we can trigger mutation
    // listeners later in the update cycle.
    const currentMutatedNodes = {};
    mutatedNodes = currentMutatedNodes;
    reconcileNode("root", null);
    // Clear these fields to avoid memory leaks.
    // @ts-ignore TODO
    activeEditor = undefined;
    activeEditorNodes = undefined;
    activeDirtyElements = undefined;
    activeDirtyLeaves = undefined;
    // @ts-ignore TODO
    activePrevNodeMap = undefined;
    // @ts-ignore TODO
    activeNextNodeMap = undefined;
    // @ts-ignore TODO
    activeEditorConfig = undefined;
    // @ts-ignore TODO
    activePrevKeyToDOMMap = undefined;
    // @ts-ignore TODO
    mutatedNodes = undefined;

    return currentMutatedNodes;
}

export function storeDOMWithKey(
    key: NodeKey,
    dom: HTMLElement,
    editor: LexicalEditor,
): void {
    const keyToDOMMap = editor._keyToDOMMap;
    (dom as CustomDomElement)["__lexicalKey_" + editor._key] = key;
    keyToDOMMap.set(key, dom);
}

function getPrevElementByKeyOrThrow(key: NodeKey): HTMLElement {
    const element = activePrevKeyToDOMMap.get(key);

    if (element === undefined) {
        throw new Error(`Reconciliation: could not find DOM element for node key ${key}`);
    }

    return element;
}
