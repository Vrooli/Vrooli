import { ELEMENT_FORMAT_TO_TYPE, ELEMENT_TYPE_TO_FORMAT } from "../consts.js";
import { RangeSelection, internalMakeRangeSelection, moveSelectionPointToSibling } from "../selection.js";
import { BaseSelection, ElementFormatType, NodeConstructorPayloads, NodeKey, NodeType, PointType, SerializedElementNode } from "../types.js";
import { errorOnReadOnly, getActiveEditor } from "../updates.js";
import { $getNodeByKey, $getSelection, $isNode, $isRangeSelection, $isRootOrShadowRoot, getNextSibling, getParent, getPreviousSibling, removeFromParent } from "../utils.js";
import { LexicalNode } from "./LexicalNode.js";
import { type TextNode } from "./TextNode.js";

export class ElementNode extends LexicalNode {
    static __type: NodeType = "Element";
    __first: null | NodeKey;
    __last: null | NodeKey;
    __size: number;
    __format: number;
    __indent: number;
    __dir: "ltr" | "rtl" | null;

    constructor({ key }: NodeConstructorPayloads["Element"]) {
        super(key);
        this.__first = null;
        this.__last = null;
        this.__size = 0;
        this.__format = 0;
        this.__indent = 0;
        this.__dir = null;
    }

    getFormat(): number {
        const self = this.getLatest();
        return self.__format;
    }

    getFormatType(): ElementFormatType {
        const format = this.getFormat();
        return ELEMENT_FORMAT_TO_TYPE[format] || "";
    }

    getIndent(): number {
        const self = this.getLatest();
        return self.__indent;
    }

    getChildren<T extends LexicalNode>(): Array<T> {
        const children: Array<T> = [];
        let child: T | null = this.getFirstChild();
        while (child !== null) {
            children.push(child);
            child = getNextSibling(child);
        }
        return children;
    }

    getChildrenKeys(): Array<NodeKey> {
        const children: Array<NodeKey> = [];
        let child: LexicalNode | null = this.getFirstChild();
        while (child !== null) {
            children.push(child.__key);
            child = getNextSibling(child);
        }
        return children;
    }

    getChildrenSize(): number {
        const self = this.getLatest();
        return self.__size;
    }

    isEmpty(): boolean {
        return this.getChildrenSize() === 0;
    }

    isDirty(): boolean {
        const editor = getActiveEditor();
        const dirtyElements = editor._dirtyElements;
        return dirtyElements !== null && dirtyElements.has(this.__key);
    }

    isLastChild(): boolean {
        const self = this.getLatest();
        const parentLastChild = getParent(this, { throwIfNull: true }).getLastChild();
        return parentLastChild !== null && parentLastChild.is(self);
    }

    getAllTextNodes(): Array<TextNode> {
        const textNodes: TextNode[] = [];
        let child: LexicalNode | null = this.getFirstChild();
        while (child !== null) {
            if ($isNode("Text", child)) {
                textNodes.push(child);
            }
            if ($isNode("Element", child)) {
                const subChildrenNodes = child.getAllTextNodes();
                textNodes.push(...subChildrenNodes);
            }
            child = getNextSibling(child);
        }
        return textNodes;
    }

    getFirstDescendant<T extends LexicalNode>(): null | T {
        let node = this.getFirstChild<T>();
        while ($isNode("Element", node)) {
            const child = node.getFirstChild<T>();
            if (child === null) {
                break;
            }
            node = child;
        }
        return node;
    }

    getLastDescendant<T extends LexicalNode>(): null | T {
        let node = this.getLastChild<T>();
        while ($isNode("Element", node)) {
            const child = node.getLastChild<T>();
            if (child === null) {
                break;
            }
            node = child;
        }
        return node;
    }

    getDescendantByIndex<T extends LexicalNode>(index: number): null | T {
        const children = this.getChildren<T>();
        const childrenLength = children.length;
        // For non-empty element nodes, we resolve its descendant
        // (either a leaf node or the bottom-most element)
        if (index >= childrenLength) {
            const resolvedNode = children[childrenLength - 1];
            return (
                ($isNode("Element", resolvedNode) && resolvedNode.getLastDescendant()) ||
                resolvedNode ||
                null
            );
        }
        const resolvedNode = children[index];
        return (
            ($isNode("Element", resolvedNode) && resolvedNode.getFirstDescendant()) ||
            resolvedNode ||
            null
        );
    }

    getFirstChild<T extends LexicalNode>(): null | T {
        const self = this.getLatest();
        const firstKey = self.__first;
        return firstKey === null ? null : $getNodeByKey<T>(firstKey);
    }

    getFirstChildOrThrow<T extends LexicalNode>(): T {
        const firstChild = this.getFirstChild<T>();
        if (firstChild === null) {
            throw new Error(`Expected node ${this.__key} to have a first child.`);
        }
        return firstChild;
    }

    getLastChild<T extends LexicalNode>(): null | T {
        const self = this.getLatest();
        const lastKey = self.__last;
        return lastKey === null ? null : $getNodeByKey<T>(lastKey);
    }
    getLastChildOrThrow<T extends LexicalNode>(): T {
        const lastChild = this.getLastChild<T>();
        if (lastChild === null) {
            throw new Error(`Expected node ${this.__key} to have a last child.`);
        }
        return lastChild;
    }
    getChildAtIndex<T extends LexicalNode>(index: number): null | T {
        const size = this.getChildrenSize();
        let node: null | T;
        let i;
        if (index < size / 2) {
            node = this.getFirstChild<T>();
            i = 0;
            while (node !== null && i <= index) {
                if (i === index) {
                    return node;
                }
                node = getNextSibling(node);
                i++;
            }
            return null;
        }
        node = this.getLastChild<T>();
        i = size - 1;
        while (node !== null && i >= index) {
            if (i === index) {
                return node;
            }
            node = getPreviousSibling(node);
            i--;
        }
        return null;
    }

    getMarkdownContent() {
        let result = "";
        const children = this.getChildren();
        for (let i = 0; i < children.length; i++) {
            const child = children[i];
            const text = child.getMarkdownContent();
            if ($isNode("Element", child) && !child.isInline() && i < children.length - 1) {
                result += text + "\n";
            } else {
                result += text;
            }
        }
        return result;
    }

    getTextContent() {
        let result = "";
        const children = this.getChildren();
        for (let i = 0; i < children.length; i++) {
            const child = children[i];
            const text = child.getTextContent();
            if ($isNode("Element", child) && !child.isInline() && i < children.length - 1) {
                result += text + "\n";
            } else {
                result += text;
            }
        }
        return result;
    }

    getTextContentSize() {
        return this.getChildren().reduce((acc, child) => {
            const childSize = child.getTextContentSize();
            if ($isNode("Element", child) && !child.isInline()) {
                return acc + childSize + "\n".length;
            }
            return acc + childSize;
        }, 0);
    }

    getDirection(): "ltr" | "rtl" | null {
        const self = this.getLatest();
        return self.__dir;
    }
    hasFormat(type: ElementFormatType): boolean {
        if (type !== "") {
            const formatFlag = ELEMENT_TYPE_TO_FORMAT[type];
            return (this.getFormat() & formatFlag) !== 0;
        }
        return false;
    }

    // Mutators

    select(_anchorOffset?: number, _focusOffset?: number): RangeSelection {
        errorOnReadOnly();
        const selection = $getSelection();
        let anchorOffset = _anchorOffset;
        let focusOffset = _focusOffset;
        const childrenCount = this.getChildrenSize();
        if (!this.canBeEmpty()) {
            if (_anchorOffset === 0 && _focusOffset === 0) {
                const firstChild = this.getFirstChild();
                if ($isNode("Text", firstChild) || $isNode("Element", firstChild)) {
                    return firstChild.select(0, 0);
                }
            } else if (
                (_anchorOffset === undefined || _anchorOffset === childrenCount) &&
                (_focusOffset === undefined || _focusOffset === childrenCount)
            ) {
                const lastChild = this.getLastChild();
                if ($isNode("Text", lastChild) || $isNode("Element", lastChild)) {
                    return lastChild.select();
                }
            }
        }
        if (anchorOffset === undefined) {
            anchorOffset = childrenCount;
        }
        if (focusOffset === undefined) {
            focusOffset = childrenCount;
        }
        const key = this.__key;
        if (!$isRangeSelection(selection)) {
            return internalMakeRangeSelection(
                key,
                anchorOffset,
                key,
                focusOffset,
                "element",
                "element",
            );
        } else {
            selection.anchor.set(key, anchorOffset, "element");
            selection.focus.set(key, focusOffset, "element");
            selection.dirty = true;
        }
        return selection;
    }
    selectStart(): RangeSelection {
        const firstNode = this.getFirstDescendant();
        return firstNode ? firstNode.selectStart() : this.select();
    }
    selectEnd(): RangeSelection {
        const lastNode = this.getLastDescendant();
        return lastNode ? lastNode.selectEnd() : this.select();
    }
    clear(): this {
        const writableSelf = this.getWritable();
        const children = this.getChildren();
        children.forEach((child) => child.remove());
        return writableSelf;
    }
    append(...nodesToAppend: LexicalNode[]): this {
        return this.splice(this.getChildrenSize(), 0, nodesToAppend);
    }
    setDirection(direction: "ltr" | "rtl" | null): this {
        const self = this.getWritable();
        self.__dir = direction;
        return self;
    }
    setFormat(type: ElementFormatType): this {
        const self = this.getWritable();
        self.__format = type !== "" ? ELEMENT_TYPE_TO_FORMAT[type] : 0;
        return this;
    }
    setIndent(indentLevel: number): this {
        const self = this.getWritable();
        self.__indent = indentLevel;
        return this;
    }
    splice(
        start: number,
        deleteCount: number,
        nodesToInsert: Array<LexicalNode>,
    ): this {
        const nodesToInsertLength = nodesToInsert.length;
        const oldSize = this.getChildrenSize();
        const writableSelf = this.getWritable();
        const writableSelfKey = writableSelf.__key;
        const nodesToInsertKeys: string[] = [];
        const nodesToRemoveKeys: string[] = [];
        const nodeAfterRange = this.getChildAtIndex(start + deleteCount);
        let nodeBeforeRange: LexicalNode | null = null;
        let newSize = oldSize - deleteCount + nodesToInsertLength;

        if (start !== 0) {
            if (start === oldSize) {
                nodeBeforeRange = this.getLastChild();
            } else {
                const node = this.getChildAtIndex(start);
                if (node !== null) {
                    nodeBeforeRange = getPreviousSibling(node);
                }
            }
        }

        if (deleteCount > 0) {
            let nodeToDelete =
                nodeBeforeRange === null
                    ? this.getFirstChild()
                    : getNextSibling(nodeBeforeRange);
            for (let i = 0; i < deleteCount; i++) {
                if (nodeToDelete === null) {
                    throw new Error("splice: sibling not found");
                }
                const nextSibling = getNextSibling(nodeToDelete);
                const nodeKeyToDelete = nodeToDelete.__key;
                const writableNodeToDelete = nodeToDelete.getWritable();
                removeFromParent(writableNodeToDelete);
                nodesToRemoveKeys.push(nodeKeyToDelete);
                nodeToDelete = nextSibling;
            }
        }

        let prevNode = nodeBeforeRange;
        for (let i = 0; i < nodesToInsertLength; i++) {
            const nodeToInsert = nodesToInsert[i];
            if (prevNode !== null && nodeToInsert.is(prevNode)) {
                nodeBeforeRange = prevNode = getPreviousSibling(prevNode);
            }
            const writableNodeToInsert = nodeToInsert.getWritable();
            if (writableNodeToInsert.__parent === writableSelfKey) {
                newSize--;
            }
            removeFromParent(writableNodeToInsert);
            const nodeKeyToInsert = nodeToInsert.__key;
            if (prevNode === null) {
                writableSelf.__first = nodeKeyToInsert;
                writableNodeToInsert.__prev = null;
            } else {
                const writablePrevNode = prevNode.getWritable();
                writablePrevNode.__next = nodeKeyToInsert;
                writableNodeToInsert.__prev = writablePrevNode.__key;
            }
            if (nodeToInsert.__key === writableSelfKey) {
                throw new Error("append: attempting to append self");
            }
            // Set child parent to self
            writableNodeToInsert.__parent = writableSelfKey;
            nodesToInsertKeys.push(nodeKeyToInsert);
            prevNode = nodeToInsert;
        }

        if (start + deleteCount === oldSize) {
            if (prevNode !== null) {
                const writablePrevNode = prevNode.getWritable();
                writablePrevNode.__next = null;
                writableSelf.__last = prevNode.__key;
            }
        } else if (nodeAfterRange !== null) {
            const writableNodeAfterRange = nodeAfterRange.getWritable();
            if (prevNode !== null) {
                const writablePrevNode = prevNode.getWritable();
                writableNodeAfterRange.__prev = prevNode.__key;
                writablePrevNode.__next = nodeAfterRange.__key;
            } else {
                writableNodeAfterRange.__prev = null;
            }
        }

        writableSelf.__size = newSize;

        // In case of deletion we need to adjust selection, unlink removed nodes
        // and clean up node itself if it becomes empty. None of these needed
        // for insertion-only cases
        if (nodesToRemoveKeys.length) {
            // Adjusting selection, in case node that was anchor/focus will be deleted
            const selection = $getSelection();
            if ($isRangeSelection(selection)) {
                const nodesToRemoveKeySet = new Set(nodesToRemoveKeys);
                const nodesToInsertKeySet = new Set(nodesToInsertKeys);

                const { anchor, focus } = selection;
                if (isPointRemoved(anchor, nodesToRemoveKeySet, nodesToInsertKeySet)) {
                    moveSelectionPointToSibling(
                        anchor,
                        anchor.getNode(),
                        this,
                        nodeBeforeRange,
                        nodeAfterRange,
                    );
                }
                if (isPointRemoved(focus, nodesToRemoveKeySet, nodesToInsertKeySet)) {
                    moveSelectionPointToSibling(
                        focus,
                        focus.getNode(),
                        this,
                        nodeBeforeRange,
                        nodeAfterRange,
                    );
                }
                // Cleanup if node can't be empty
                if (newSize === 0 && !this.canBeEmpty() && !$isRootOrShadowRoot(this)) {
                    this.remove();
                }
            }
        }

        return writableSelf;
    }
    // JSON serialization
    exportJSON(): SerializedElementNode {
        return {
            __type: "Element",
            children: [],
            direction: this.getDirection(),
            format: this.getFormatType(),
            indent: this.getIndent(),
            version: 1,
        };
    }
    // These are intended to be extends for specific element heuristics.
    insertNewAfter(
        selection: RangeSelection,
        restoreSelection?: boolean,
    ): null | LexicalNode {
        return null;
    }
    canIndent(): boolean {
        return true;
    }
    /*
     * This method controls the behavior of a the node during backwards
     * deletion (i.e., backspace) when selection is at the beginning of
     * the node (offset 0)
     */
    collapseAtStart(selection: RangeSelection): boolean {
        return false;
    }
    excludeFromCopy(destination?: "clone" | "html"): boolean {
        return false;
    }
    canReplaceWith(replacement: LexicalNode): boolean {
        return true;
    }
    canInsertAfter(node: LexicalNode): boolean {
        return true;
    }
    canBeEmpty(): boolean {
        return true;
    }
    canInsertTextBefore(): boolean {
        return true;
    }
    canInsertTextAfter(): boolean {
        return true;
    }
    isInline(): boolean {
        return false;
    }
    /**
     * A shadow root is a Node that behaves like RootNode. The shadow root (and RootNode) mark the
     * end of the hiercharchy, most implementations should treat it as there's nothing (upwards)
     * beyond this point. For example, getTopLevelElement(node), when performed inside a TableCellNode
     * will return the immediate first child underneath TableCellNode instead of RootNode.
     */
    isShadowRoot(): boolean {
        return false;
    }
    canMergeWith(node: ElementNode): boolean {
        return false;
    }
    extractWithChild(
        child: LexicalNode,
        selection: BaseSelection | null,
        destination: "clone" | "html",
    ): boolean {
        return false;
    }
}

function isPointRemoved(
    point: PointType,
    nodesToRemoveKeySet: Set<NodeKey>,
    nodesToInsertKeySet: Set<NodeKey>,
): boolean {
    let node: ElementNode | TextNode | null = point.getNode();
    while (node) {
        const nodeKey = node.__key;
        if (nodesToRemoveKeySet.has(nodeKey) && !nodesToInsertKeySet.has(nodeKey)) {
            return true;
        }
        node = getParent(node);
    }
    return false;
}
