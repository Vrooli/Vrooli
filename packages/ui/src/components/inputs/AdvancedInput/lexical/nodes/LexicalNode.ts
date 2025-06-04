/* eslint-disable @typescript-eslint/ban-ts-comment */
import { $moveSelectionPointToEnd, $updateElementSelectionOnCreateDeleteNode, type RangeSelection } from "../selection.js";
import { type DOMConversionMap, type DOMExportOutput, type EditorConfig, LexicalNodeBase, type LexicalNodeClass, type NodeKey, type NodeType, type SerializedLexicalNode } from "../types.js";
import { errorOnReadOnly, getActiveEditor, getActiveEditorState } from "../updates.js";
import { $createNode, $createNodeKey, $getCompositionKey, $getNodeByKey, $getSelection, $isNode, $isRangeSelection, $setCompositionKey, $setSelection, errorOnInsertTextNodeOnRoot, getCommonAncestor, getIndexWithinParent, getNextSibling, getParent, getPreviousSibling, internalMarkNodeAsDirty, removeFromParent, removeNode } from "../utils.js";
import { type ElementNode } from "./ElementNode.js";

export class LexicalNode extends LexicalNodeBase {
    static __type: NodeType = "Root";
    /** A unique key for the node. */
    __key: string;
    /** The key of the parent node. */
    __parent: null | NodeKey;
    /** The key of the previous sibling node. */
    __prev: null | NodeKey;
    /** The key of the next sibling node. */
    __next: null | NodeKey;

    constructor(key?: NodeKey) {
        super();
        this.__key = $createNodeKey(this, key);
        this.__parent = null;
        this.__prev = null;
        this.__next = null;
    }

    getType(): NodeType {
        return (this.constructor as typeof LexicalNode).__type;
    }
    static getType(): NodeType {
        return this.__type;
    }

    /**
     * Clones this node, creating a new node with a different key
     * and adding it to the EditorState (but not attaching it anywhere!). All nodes must
     * implement this method.
     *
     */
    static clone(_data: unknown): LexicalNode {
        throw new Error(`LexicalNode: Node ${this.name} does not implement .clone().`);
    }

    /**
     * Provides a map of DOM elements to their conversion functions. This allows us to 
     * turn HTML elements into lexical nodes
     */
    static importDOM: () => DOMConversionMap;

    /**
     * True if this node is inline, false otherwise. Inline nodes are nodes that can be
     * placed on the same line as other inline nodes, such as bold text or links. 
     * Block nodes, like paragraphs or headings, are not inline.
     */
    isInline(): boolean {
        throw new Error(`LexicalNode: Node ${this.constructor.name} does not implement .isInline().`);
    }

    /**
     * Returns true if the provided node is the exact same one as this node, from Lexical's perspective.
     * Always use this instead of referential equality.
     *
     * @param object - the node to perform the equality comparison on.
     */
    is(object: LexicalNode | null | undefined): boolean {
        if (object === null || object === undefined) {
            return false;
        }
        return this.__key === object.__key;
    }

    /**
     * Returns true if this node logical precedes the target node in the editor state.
     *
     * @param targetNode - the node we're testing to see if it's after this one.
     */
    isBefore(targetNode: LexicalNode): boolean {
        if (this === targetNode) {
            return false;
        }
        if (targetNode.isParentOf(this)) {
            return true;
        }
        if (this.isParentOf(targetNode)) {
            return false;
        }
        const commonAncestor = getCommonAncestor(this, targetNode);
        let indexA = 0;
        let indexB = 0;
        let node: this | ElementNode | LexicalNode = this;
        let loopCount = 0;
        while (loopCount < 100) {
            const parent: ElementNode = getParent(node, { throwIfNull: true });
            if (parent === commonAncestor) {
                indexA = getIndexWithinParent(node);
                break;
            }
            node = parent;
        }
        node = targetNode;
        loopCount = 0;
        while (loopCount < 100) {
            const parent: ElementNode = getParent(node, { throwIfNull: true });
            if (parent === commonAncestor) {
                indexB = getIndexWithinParent(node);
                break;
            }
            node = parent;
        }
        return indexA < indexB;
    }

    /**
     * Returns true if this node is the parent of the target node, false otherwise.
     *
     * @param targetNode - the would-be child node.
     */
    isParentOf(targetNode: LexicalNode): boolean {
        const key = this.__key;
        if (key === targetNode.__key) {
            return false;
        }
        let node: ElementNode | LexicalNode | null = targetNode;
        while (node !== null) {
            if (node.__key === key) {
                return true;
            }
            node = getParent(node);
        }
        return false;
    }

    // TO-DO: this function can be simplified a lot
    /**
     * Returns a list of nodes that are between this node and
     * the target node in the EditorState.
     *
     * @param targetNode - the node that marks the other end of the range of nodes to be returned.
     */
    getNodesBetween(targetNode: LexicalNode): Array<LexicalNode> {
        const isBefore = this.isBefore(targetNode);
        const nodes: LexicalNode[] = [];
        const visited = new Set();
        let node: LexicalNode | this | null = this;
        const loopCount = 0;
        while (loopCount < 100) {
            if (node === null) {
                break;
            }
            const key = node.__key;
            if (!visited.has(key)) {
                visited.add(key);
                nodes.push(node);
            }
            if (node === targetNode) {
                break;
            }
            const child: LexicalNode | null = $isNode("Element", node)
                ? isBefore
                    ? node.getFirstChild()
                    : node.getLastChild()
                : null;
            if (child !== null) {
                node = child;
                continue;
            }
            const nextSibling: LexicalNode | null = isBefore
                ? getNextSibling(node)
                : getPreviousSibling(node);
            if (nextSibling !== null) {
                node = nextSibling;
                continue;
            }
            const parent: LexicalNode | null = getParent(node, { throwIfNull: true });
            if (!visited.has(parent.__key)) {
                nodes.push(parent);
            }
            if (parent === targetNode) {
                break;
            }
            let parentSibling: LexicalNode | null = null;
            let ancestor: LexicalNode | null = parent;
            do {
                if (ancestor === null) {
                    throw new Error("getNodesBetween: ancestor is null");
                }
                parentSibling = isBefore
                    ? getNextSibling(ancestor)
                    : getPreviousSibling(ancestor);
                ancestor = getParent(ancestor);
                if (ancestor !== null) {
                    if (parentSibling === null && !visited.has(ancestor.__key)) {
                        nodes.push(ancestor);
                    }
                } else {
                    break;
                }
            } while (parentSibling === null);
            node = parentSibling;
        }
        if (!isBefore) {
            nodes.reverse();
        }
        return nodes;
    }

    /**
     * Returns true if this node has been marked dirty during this update cycle.
     *
     */
    isDirty(): boolean {
        const editor = getActiveEditor();
        const dirtyLeaves = editor._dirtyLeaves;
        return dirtyLeaves !== null && dirtyLeaves.has(this.__key);
    }

    /**
     * Returns the latest version of the node from the active EditorState.
     * This is used to avoid getting values from stale node references.
     *
     */
    getLatest(): this {
        const latest = $getNodeByKey<this>(this.__key);
        if (!latest) {
            throw new Error(`Lexical node ${this.__key} does not exist in active editor state. Avoid using the same node references between nested closures from editorState.read/editor.update.`);
        }
        return latest;
    }

    /**
     * Returns a mutable version of the node. Will throw an error if
     * called outside of a Lexical Editor callback.
     *
     */
    getWritable(): this {
        errorOnReadOnly();
        const editorState = getActiveEditorState();
        const editor = getActiveEditor();
        const nodeMap = editorState._nodeMap;
        const key = this.__key;
        // Ensure we get the latest node from pending state
        const latestNode = this.getLatest();
        const parent = latestNode.__parent;
        const cloneNotNeeded = editor._cloneNotNeeded;
        const selection = $getSelection();
        if (selection !== null) {
            selection.setCachedNodes(null);
        }
        if (cloneNotNeeded.has(key)) {
            // Transforms clear the dirty node set on each iteration to keep track on newly dirty nodes
            internalMarkNodeAsDirty(latestNode);
            return latestNode;
        }
        const constructor = latestNode.constructor;
        const mutableNode = (constructor as LexicalNodeClass).clone(latestNode);
        mutableNode.__parent = parent;
        mutableNode.__next = latestNode.__next;
        mutableNode.__prev = latestNode.__prev;
        if ($isNode("Element", latestNode) && $isNode("Element", mutableNode)) {
            if ($isNode("Paragraph", latestNode) && $isNode("Paragraph", mutableNode)) {
                mutableNode.__textFormat = latestNode.__textFormat;
            }
            mutableNode.__first = latestNode.__first;
            mutableNode.__last = latestNode.__last;
            mutableNode.__size = latestNode.__size;
            mutableNode.__indent = latestNode.__indent;
            mutableNode.__format = latestNode.__format;
            mutableNode.__dir = latestNode.__dir;
        } else if ($isNode("Text", latestNode) && $isNode("Text", mutableNode)) {
            mutableNode.__format = latestNode.__format;
            mutableNode.__style = latestNode.__style;
            mutableNode.__mode = latestNode.__mode;
            mutableNode.__detail = latestNode.__detail;
        }
        cloneNotNeeded.add(key);
        mutableNode.__key = key;
        internalMarkNodeAsDirty(mutableNode);
        // Update reference in node map
        nodeMap.set(key, mutableNode);

        return mutableNode;
    }

    /**
     * @returns Markdown content of the node. Override this for
     * custom nodes that should have a representation in markdown format.
     */
    getMarkdownContent(): string {
        return "";
    }

    /**
     * @returns Text content of the node. Override this for
     * custom nodes that should have a representation in plain text
     * format (for copy + paste, for example)
     */
    getTextContent(): string {
        return "";
    }

    /**
     * Returns the length of the string produced by calling getTextContent on this node.
     * We use this instead of getTextContent().length for performance reasons.
     */
    getTextContentSize(): number {
        return this.getTextContent().length;
    }

    // View

    /**
     * Called during the reconciliation process to determine which nodes
     * to insert into the DOM for this Lexical Node.
     *
     * This method must return exactly one HTMLElement. Nested elements are not supported.
     *
     * Do not attempt to update the Lexical EditorState during this phase of the update lifecyle.
     *
     * @param _config - allows access to things like the EditorTheme (to apply classes) during reconciliation.
     * @param _editor - allows access to the editor for context during reconciliation.
     *
     * */
    createDOM(): HTMLElement {
        throw new Error("createDOM: base method not extended");
    }

    /**
     * Called when a node changes and should update the DOM
     * in whatever way is necessary to make it align with any changes that might
     * have happened during the update.
     *
     * Returning "true" here will cause lexical to unmount and recreate the DOM node
     * (by calling createDOM). You would need to do this if the element tag changes,
     * for instance.
     *
     * */
    updateDOM(
        _prevNode: unknown,
        _dom: HTMLElement,
        _config: EditorConfig,
    ): boolean {
        throw new Error("updateDOM: base method not extended");
    }

    /**
     * Controls how the this node is serialized to HTML. This is important for
     * copy and paste between Lexical and non-Lexical editors, or Lexical editors with different namespaces,
     * in which case the primary transfer format is HTML. It's also important if you're serializing
     * to HTML for any other reason via {@link @lexical/html!$generateHtmlFromNodes}. You could
     * also use this method to build your own HTML renderer.
     *
     * */
    exportDOM(): DOMExportOutput {
        const element = this.createDOM();
        return { element };
    }

    /**
     * Controls how the this node is serialized to JSON. This is important for
     * copy and paste between Lexical editors sharing the same namespace. It's also important
     * if you're serializing to JSON for persistent storage somewhere.
     * See [Serialization & Deserialization](https://lexical.dev/docs/concepts/serialization#lexical---html).
     *
     * */
    exportJSON(): SerializedLexicalNode {
        throw new Error("exportJSON: base method not extended");
    }

    /**
     * Controls how the this node is deserialized from JSON. This is usually boilerplate,
     * but provides an abstraction between the node implementation and serialized interface that can
     * be important if you ever make breaking changes to a node schema (by adding or removing properties).
     * See [Serialization & Deserialization](https://lexical.dev/docs/concepts/serialization#lexical---html).
     *
     * */
    static importJSON(_serializedNode: SerializedLexicalNode): LexicalNode {
        throw new Error(`LexicalNode: Node ${this.name} does not implement .importJSON().`);
    }

    // Setters and mutators

    /**
     * Removes this LexicalNode from the EditorState. If the node isn't re-inserted
     * somewhere, the Lexical garbage collector will eventually clean it up.
     *
     * @param preserveEmptyParent - If falsy, the node's parent will be removed if
     * it's empty after the removal operation. This is the default behavior, subject to
     * other node heuristics such as {@link ElementNode#canBeEmpty}
     * */
    remove(preserveEmptyParent?: boolean): void {
        removeNode(this, true, preserveEmptyParent);
    }

    /**
     * Replaces this LexicalNode with the provided node, optionally transferring the children
     * of the replaced node to the replacing node.
     * 
     * An example of when this is useful is switching text from italic to a code block. We have to 
     * get the text from the italic node (TextNode), and use it to create a new code block node.
     *
     * @param replaceWith - The node to replace this one with.
     * @param includeChildren - Whether or not to transfer the children of this node to the replacing node.
     * */
    replace<N extends LexicalNode>(replaceWith: N, includeChildren?: boolean): N {
        errorOnReadOnly();
        let selection = $getSelection();
        if (selection !== null) {
            selection = selection.clone();
        }
        errorOnInsertTextNodeOnRoot(this, replaceWith);
        const self = this.getLatest();
        const toReplaceKey = this.__key;
        const key = replaceWith.__key;
        const writableReplaceWith = replaceWith.getWritable();
        const writableParent = getParent(this, { throwIfNull: true }).getWritable();
        const size = writableParent.__size;
        removeFromParent(writableReplaceWith);
        const prevSibling = getPreviousSibling(self);
        const nextSibling = getNextSibling(self);
        const prevKey = self.__prev;
        const nextKey = self.__next;
        const parentKey = self.__parent;
        removeNode(self, false, true);

        if (prevSibling === null) {
            writableParent.__first = key;
        } else {
            const writablePrevSibling = prevSibling.getWritable();
            writablePrevSibling.__next = key;
        }
        writableReplaceWith.__prev = prevKey;
        if (nextSibling === null) {
            writableParent.__last = key;
        } else {
            const writableNextSibling = nextSibling.getWritable();
            writableNextSibling.__prev = key;
        }
        writableReplaceWith.__next = nextKey;
        writableReplaceWith.__parent = parentKey;
        writableParent.__size = size;
        if (includeChildren) {
            if (!$isNode("Element", this) || !$isNode("Element", writableReplaceWith)) {
                throw new Error("includeChildren should only be true for ElementNodes");
            }
            this.getChildren().forEach((child: LexicalNode) => {
                writableReplaceWith.append(child);
            });
        }
        if ($isRangeSelection(selection)) {
            $setSelection(selection);
            const anchor = selection.anchor;
            const focus = selection.focus;
            if (anchor.key === toReplaceKey) {
                $moveSelectionPointToEnd(anchor, writableReplaceWith);
            }
            if (focus.key === toReplaceKey) {
                $moveSelectionPointToEnd(focus, writableReplaceWith);
            }
        }
        if ($getCompositionKey() === toReplaceKey) {
            $setCompositionKey(key);
        }
        return writableReplaceWith;
    }

    /**
     * Inserts a node after this LexicalNode (as the next sibling).
     *
     * @param nodeToInsert - The node to insert after this one.
     * @param restoreSelection - Whether or not to attempt to resolve the
     * selection to the appropriate place after the operation is complete.
     * */
    insertAfter(nodeToInsert: LexicalNode, restoreSelection = true): LexicalNode {
        errorOnReadOnly();
        errorOnInsertTextNodeOnRoot(this, nodeToInsert);
        const writableSelf = this.getWritable();
        const writableNodeToInsert = nodeToInsert.getWritable();
        const oldParent = getParent(writableNodeToInsert);
        const selection = $getSelection();
        let elementAnchorSelectionOnNode = false;
        let elementFocusSelectionOnNode = false;
        if (oldParent !== null) {
            const oldIndex = getIndexWithinParent(nodeToInsert);
            removeFromParent(writableNodeToInsert);
            if ($isRangeSelection(selection)) {
                const oldParentKey = oldParent.__key;
                const anchor = selection.anchor;
                const focus = selection.focus;
                elementAnchorSelectionOnNode =
                    anchor.type === "element" &&
                    anchor.key === oldParentKey &&
                    anchor.offset === oldIndex + 1;
                elementFocusSelectionOnNode =
                    focus.type === "element" &&
                    focus.key === oldParentKey &&
                    focus.offset === oldIndex + 1;
            }
        }
        const nextSibling = getNextSibling(this);
        const writableParent = getParent(this, { throwIfNull: true }).getWritable();
        const insertKey = writableNodeToInsert.__key;
        const nextKey = writableSelf.__next;
        if (nextSibling === null) {
            writableParent.__last = insertKey;
        } else {
            const writableNextSibling = nextSibling.getWritable();
            writableNextSibling.__prev = insertKey;
        }
        writableParent.__size++;
        writableSelf.__next = insertKey;
        writableNodeToInsert.__next = nextKey;
        writableNodeToInsert.__prev = writableSelf.__key;
        writableNodeToInsert.__parent = writableSelf.__parent;
        if (restoreSelection && $isRangeSelection(selection)) {
            const index = getIndexWithinParent(this);
            $updateElementSelectionOnCreateDeleteNode(
                selection,
                writableParent,
                index + 1,
            );
            const writableParentKey = writableParent.__key;
            if (elementAnchorSelectionOnNode) {
                selection.anchor.set(writableParentKey, index + 2, "element");
            }
            if (elementFocusSelectionOnNode) {
                selection.focus.set(writableParentKey, index + 2, "element");
            }
        }
        return nodeToInsert;
    }

    /**
     * Inserts a node before this LexicalNode (as the previous sibling).
     *
     * @param nodeToInsert - The node to insert before this one.
     * @param restoreSelection - Whether or not to attempt to resolve the
     * selection to the appropriate place after the operation is complete.
     * */
    insertBefore(
        nodeToInsert: LexicalNode,
        restoreSelection = true,
    ): LexicalNode {
        errorOnReadOnly();
        errorOnInsertTextNodeOnRoot(this, nodeToInsert);
        const writableSelf = this.getWritable();
        const writableNodeToInsert = nodeToInsert.getWritable();
        const insertKey = writableNodeToInsert.__key;
        removeFromParent(writableNodeToInsert);
        const prevSibling = getPreviousSibling(this);
        const writableParent = getParent(this, { throwIfNull: true }).getWritable();
        const prevKey = writableSelf.__prev;
        // TODO: this is O(n), can we improve?
        const index = getIndexWithinParent(this);
        if (prevSibling === null) {
            writableParent.__first = insertKey;
        } else {
            const writablePrevSibling = prevSibling.getWritable();
            writablePrevSibling.__next = insertKey;
        }
        writableParent.__size++;
        writableSelf.__prev = insertKey;
        writableNodeToInsert.__prev = prevKey;
        writableNodeToInsert.__next = writableSelf.__key;
        writableNodeToInsert.__parent = writableSelf.__parent;
        const selection = $getSelection();
        if (restoreSelection && $isRangeSelection(selection)) {
            const parent = getParent(this, { throwIfNull: true });
            $updateElementSelectionOnCreateDeleteNode(selection, parent, index);
        }
        return nodeToInsert;
    }

    /**
     * Whether or not this node has a required parent. Used during copy + paste operations
     * to normalize nodes that would otherwise be orphaned. For example, ListItemNodes without
     * a ListNode parent or TextNodes with a ParagraphNode parent.
     *
     * */
    isParentRequired(): boolean {
        return false;
    }

    /**
     * The creation logic for any required parent. Should be implemented if {@link isParentRequired} returns true.
     *
     * */
    createParentElementNode(): ElementNode {
        return $createNode("Paragraph", {});
    }

    selectStart(): RangeSelection {
        return this.selectPrevious();
    }

    selectEnd(): RangeSelection {
        return this.selectNext(0, 0);
    }

    /**
     * Moves selection to the previous sibling of this node, at the specified offsets.
     *
     * @param anchorOffset - The anchor offset for selection.
     * @param focusOffset -  The focus offset for selection
     * */
    selectPrevious(anchorOffset?: number, focusOffset?: number): RangeSelection {
        errorOnReadOnly();
        const prevSibling = getPreviousSibling(this);
        const parent = getParent(this, { throwIfNull: true });
        if (prevSibling === null) {
            return parent.select(0, 0);
        }
        if ($isNode("Element", prevSibling)) {
            return prevSibling.select();
        } else if (!$isNode("Text", prevSibling)) {
            const index = getIndexWithinParent(prevSibling) + 1;
            return parent.select(index, index);
        }
        return prevSibling.select(anchorOffset, focusOffset);
    }

    /**
     * Moves selection to the next sibling of this node, at the specified offsets.
     *
     * @param anchorOffset - The anchor offset for selection.
     * @param focusOffset -  The focus offset for selection
     * */
    selectNext(anchorOffset?: number, focusOffset?: number): RangeSelection {
        errorOnReadOnly();
        const nextSibling = getNextSibling(this);
        const parent = getParent(this, { throwIfNull: true });
        if (nextSibling === null) {
            return parent.select();
        }
        if ($isNode("Element", nextSibling)) {
            return nextSibling.select(0, 0);
        } else if (!$isNode("Text", nextSibling)) {
            const index = getIndexWithinParent(nextSibling);
            return parent.select(index, index);
        }
        return nextSibling.select(anchorOffset, focusOffset);
    }

    /**
     * Marks a node dirty, triggering transforms and
     * forcing it to be reconciled during the update cycle.
     *
     * */
    markDirty(): void {
        this.getWritable();
    }
}

/**
 * Insert a series of nodes after this LexicalNode (as next siblings)
 *
 * @param firstToInsert - The first node to insert after this one.
 * @param lastToInsert - The last node to insert after this one. Must be a
 * later sibling of FirstNode. If not provided, it will be its last sibling.
 */
export function insertRangeAfter(
    node: LexicalNode,
    firstToInsert: LexicalNode,
    lastToInsert?: LexicalNode,
) {
    const lastToInsert2 =
        lastToInsert || getParent(firstToInsert, { throwIfNull: true }).getLastChild()!;
    let current = firstToInsert;
    const nodesToInsert = [firstToInsert];
    while (current !== lastToInsert2) {
        if (!getNextSibling(current)) {
            throw new Error("insertRangeAfter: lastToInsert must be a later sibling of firstToInsert");
        }
        current = getNextSibling(current)!;
        nodesToInsert.push(current);
    }

    let currentNode: LexicalNode = node;
    for (const nodeToInsert of nodesToInsert) {
        currentNode = currentNode.insertAfter(nodeToInsert);
    }
}
