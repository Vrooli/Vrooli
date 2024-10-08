import { RangeSelection } from "../selection";
import { BaseSelection, DOMConversionMap, DOMConversionOutput, DOMExportOutput, EditorConfig, NodeConstructorPayloads, NodeKey, NodeType, SerializedListItemNode } from "../types";
import { $createNode, $isNode, $isRangeSelection, append, getNextSibling, getNextSiblings, getParent, getPreviousSibling, getPreviousSiblings, isHTMLElement, isNestedListNode } from "../utils";
import { ElementNode } from "./ElementNode";
import { type LexicalNode } from "./LexicalNode";
import { mergeLists, type ListNode } from "./ListNode";
import { type ParagraphNode } from "./ParagraphNode";

export class ListItemNode extends ElementNode {
    static __type: NodeType = "ListItem";
    __value: number;
    __checked?: boolean;

    static clone(node: ListItemNode): ListItemNode {
        const { __value, __checked, __key } = node;
        return $createNode("ListItem", { value: __value, checked: __checked, key: __key });
    }

    constructor({ value, checked, ...rest }: NodeConstructorPayloads["ListItem"]) {
        super(rest);
        this.__value = value === undefined ? 1 : value;
        this.__checked = checked;
    }

    createDOM(): HTMLElement {
        const element = document.createElement("li");
        const parent = getParent(this);
        if ($isNode("List", parent) && parent.getListType() === "check") {
            updateListItemChecked(element, this, null, parent);
        }
        element.value = this.__value;
        return element;
    }

    updateDOM(
        prevNode: ListItemNode,
        dom: HTMLElement,
        config: EditorConfig,
    ): boolean {
        const parent = getParent(this);
        if ($isNode("List", parent) && parent.getListType() === "check") {
            updateListItemChecked(dom, this, prevNode, parent);
        }
        // @ts-expect-error - this is always HTMLListItemElement
        dom.value = this.__value;

        return false;
    }

    static transform(): (node: LexicalNode) => void {
        return (node: LexicalNode) => {
            if (!$isNode("ListItem", node)) {
                throw new Error("node is not a ListItemNode");
            }
            if (node.__checked === null) {
                return;
            }
            const parent = getParent(node);
            if ($isNode("List", parent)) {
                if (parent.getListType() !== "check" && node.getChecked() !== null) {
                    node.setChecked(undefined);
                }
            }
        };
    }

    static importDOM(): DOMConversionMap {
        return {
            li: (node: Node) => ({
                conversion: convertListItemElement,
                priority: 0,
            }),
        };
    }

    static importJSON({ checked, direction, format, value }: SerializedListItemNode): ListItemNode {
        const node = $createNode("ListItem", { checked, value });
        node.setFormat(format);
        node.setDirection(direction);
        return node;
    }

    exportDOM(): DOMExportOutput {
        const element = this.createDOM();
        element.style.textAlign = this.getFormatType();
        return {
            element,
        };
    }

    exportJSON(): SerializedListItemNode {
        return {
            ...super.exportJSON(),
            __type: "ListItem",
            checked: this.getChecked(),
            value: this.getValue(),
            version: 1,
        };
    }

    append(...nodes: LexicalNode[]): this {
        for (let i = 0; i < nodes.length; i++) {
            const node = nodes[i];

            if ($isNode("Element", node) && this.canMergeWith(node)) {
                const children = node.getChildren();
                this.append(...children);
                node.remove();
            } else {
                super.append(node);
            }
        }

        return this;
    }

    replace<N extends LexicalNode>(
        replaceWithNode: N,
        includeChildren?: boolean,
    ): N {
        if ($isNode("ListItem", replaceWithNode)) {
            return super.replace(replaceWithNode);
        }
        this.setIndent(0);
        const list = getParent(this, { throwIfNull: true });
        if (!$isNode("List", list)) {
            return replaceWithNode;
        }
        if (list.__first === this.__key) {
            list.insertBefore(replaceWithNode);
        } else if (list.__last === this.__key) {
            list.insertAfter(replaceWithNode);
        } else {
            // Split the list
            const newList = $createNode("List", { listType: list.getListType() });
            let nextSibling = getNextSibling(this);
            while (nextSibling) {
                const nodeToAppend = nextSibling;
                nextSibling = getNextSibling(nextSibling);
                newList.append(nodeToAppend);
            }
            list.insertAfter(replaceWithNode);
            replaceWithNode.insertAfter(newList);
        }
        if (includeChildren) {
            if (!$isNode("Element", replaceWithNode)) {
                throw new Error("includeChildren should only be true for ElementNodes");
            }
            this.getChildren().forEach((child: LexicalNode) => {
                replaceWithNode.append(child);
            });
        }
        this.remove();
        if (list.getChildrenSize() === 0) {
            list.remove();
        }
        return replaceWithNode;
    }

    insertAfter(node: LexicalNode, restoreSelection = true): LexicalNode {
        const listNode = getParent(this, { throwIfNull: true });

        if (!$isNode("List", listNode)) {
            throw new Error("insertAfter: list node is not parent of list item node");
        }

        if ($isNode("ListItem", node)) {
            return super.insertAfter(node, restoreSelection);
        }

        const siblings = getNextSiblings(this);

        // Split the lists and insert the node in between them
        listNode.insertAfter(node, restoreSelection);

        if (siblings.length !== 0) {
            const newListNode = $createNode("List", { listType: listNode.getListType() });

            siblings.forEach((sibling) => newListNode.append(sibling));

            node.insertAfter(newListNode, restoreSelection);
        }

        return node;
    }

    remove(preserveEmptyParent?: boolean): void {
        const prevSibling = getPreviousSibling(this);
        const nextSibling = getNextSibling(this);
        super.remove(preserveEmptyParent);

        if (
            prevSibling &&
            nextSibling &&
            isNestedListNode(prevSibling) &&
            isNestedListNode(nextSibling)
        ) {
            mergeLists(prevSibling.getFirstChild(), nextSibling.getFirstChild());
            nextSibling.remove();
        }
    }

    insertNewAfter(
        _: RangeSelection,
        restoreSelection = true,
    ): ListItemNode | ParagraphNode {
        const checked = this.__checked === null ? undefined : false;
        const value = this.__value > 0 ? this.__value + 1 : 1;
        const newElement = $createNode("ListItem", { checked, value });
        this.insertAfter(newElement, restoreSelection);

        return newElement;
    }

    collapseAtStart(selection: RangeSelection): true {
        const paragraph = $createNode("Paragraph", {});
        const children = this.getChildren();
        children.forEach((child) => paragraph.append(child));
        const listNode = getParent(this, { throwIfNull: true });
        const listNodeParent = getParent(listNode, { throwIfNull: true });
        const isIndented = $isNode("ListItem", listNodeParent);

        if (listNode.getChildrenSize() === 1) {
            if (isIndented) {
                // if the list node is nested, we just want to remove it,
                // effectively unindenting it.
                listNode.remove();
                listNodeParent.select();
            } else {
                listNode.insertBefore(paragraph);
                listNode.remove();
                // If we have selection on the list item, we'll need to move it
                // to the paragraph
                const anchor = selection.anchor;
                const focus = selection.focus;
                const key = paragraph.__key;

                if (anchor.type === "element" && anchor.getNode().is(this)) {
                    anchor.set(key, anchor.offset, "element");
                }

                if (focus.type === "element" && focus.getNode().is(this)) {
                    focus.set(key, focus.offset, "element");
                }
            }
        } else {
            listNode.insertBefore(paragraph);
            this.remove();
        }

        return true;
    }

    getValue(): number {
        const self = this.getLatest();

        return self.__value;
    }

    setValue(value: number): void {
        const self = this.getWritable();
        self.__value = value;
    }

    getChecked(): boolean | undefined {
        const self = this.getLatest();

        return self.__checked;
    }

    setChecked(checked?: boolean): void {
        const self = this.getWritable();
        self.__checked = checked;
    }

    toggleChecked(): void {
        this.setChecked(!this.__checked);
    }

    getIndent(): number {
        // If we don't have a parent, we are likely serializing
        const parent = getParent(this);
        if (parent === null) {
            return this.getLatest().__indent;
        }
        // ListItemNode should always have a ListNode for a parent.
        let listNodeParent = getParent(parent, { throwIfNull: true });
        let indentLevel = 0;
        while ($isNode("ListItem", listNodeParent)) {
            // Get grandparent
            listNodeParent = getParent(listNodeParent, { skip: 1, throwIfNull: true });
            indentLevel++;
        }

        return indentLevel;
    }

    setIndent(indent: number): this {
        if (typeof indent !== "number" || indent < 0) {
            throw new Error("Invalid indent value.");
        }
        let currentIndent = this.getIndent();
        while (currentIndent !== indent) {
            if (currentIndent < indent) {
                $handleIndent(this);
                currentIndent++;
            } else {
                $handleOutdent(this);
                currentIndent--;
            }
        }

        return this;
    }

    canInsertAfter(node: LexicalNode): boolean {
        return $isNode("ListItem", node);
    }

    canReplaceWith(replacement: LexicalNode): boolean {
        return $isNode("ListItem", replacement);
    }

    canMergeWith(node: LexicalNode): boolean {
        return $isNode("Paragraph", node) || $isNode("ListItem", node);
    }

    extractWithChild(child: LexicalNode, selection: BaseSelection): boolean {
        if (!$isRangeSelection(selection)) {
            return false;
        }

        const anchorNode = selection.anchor.getNode();
        const focusNode = selection.focus.getNode();

        return (
            this.isParentOf(anchorNode) &&
            this.isParentOf(focusNode) &&
            this.getTextContent().length === selection.getTextContent().length
        );
    }

    isParentRequired(): true {
        return true;
    }

    createParentElementNode(): ElementNode {
        return $createNode("List", { listType: "bullet" });
    }
}

function updateListItemChecked(
    dom: HTMLElement,
    listItemNode: ListItemNode,
    prevListItemNode: ListItemNode | null,
    listNode: ListNode,
): void {
    // Only add attributes for leaf list items
    if ($isNode("List", listItemNode.getFirstChild())) {
        dom.removeAttribute("role");
        dom.removeAttribute("tabIndex");
        dom.removeAttribute("aria-checked");
    } else {
        dom.setAttribute("role", "checkbox");
        dom.setAttribute("tabIndex", "-1");

        if (
            !prevListItemNode ||
            listItemNode.__checked !== prevListItemNode.__checked
        ) {
            dom.setAttribute(
                "aria-checked",
                listItemNode.getChecked() ? "true" : "false",
            );
        }
    }
}

function convertListItemElement(domNode: Node): DOMConversionOutput {
    const checked = isHTMLElement(domNode) && domNode.getAttribute("aria-checked") === "true";
    const value = isHTMLElement(domNode) && domNode.getAttribute("value") ? parseInt(domNode.getAttribute("value") || "1", 10) : 1;
    return { node: $createNode("ListItem", { checked, value }) };
}

/**
 * Adds an empty ListNode/ListItemNode chain at listItemNode, so as to
 * create an indent effect. Won't indent ListItemNodes that have a ListNode as
 * a child, but does merge sibling ListItemNodes if one has a nested ListNode.
 * @param listItemNode - The ListItemNode to be indented.
 */
export function $handleIndent(listItemNode: ListItemNode): void {
    // go through each node and decide where to move it.
    const removed = new Set<NodeKey>();

    if (isNestedListNode(listItemNode) || removed.has(listItemNode.__key)) {
        return;
    }

    const parent = getParent(listItemNode);

    // We can cast both of the below `isNestedListNode` only returns a boolean type instead of a user-defined type guards
    const nextSibling = getNextSibling<ListItemNode>(listItemNode);
    const previousSibling = getPreviousSibling<ListItemNode>(listItemNode);
    // if there are nested lists on either side, merge them all together.

    if (isNestedListNode(nextSibling) && isNestedListNode(previousSibling)) {
        const innerList = previousSibling.getFirstChild();

        if ($isNode("List", innerList)) {
            innerList.append(listItemNode);
            const nextInnerList = nextSibling.getFirstChild();

            if ($isNode("List", nextInnerList)) {
                const children = nextInnerList.getChildren();
                append(innerList, children);
                nextSibling.remove();
                removed.add(nextSibling.__key);
            }
        }
    } else if (isNestedListNode(nextSibling)) {
        // if the ListItemNode is next to a nested ListNode, merge them
        const innerList = nextSibling.getFirstChild();

        if ($isNode("List", innerList)) {
            const firstChild = innerList.getFirstChild();

            if (firstChild !== null) {
                firstChild.insertBefore(listItemNode);
            }
        }
    } else if (isNestedListNode(previousSibling)) {
        const innerList = previousSibling.getFirstChild();

        if ($isNode("List", innerList)) {
            innerList.append(listItemNode);
        }
    } else {
        // otherwise, we need to create a new nested ListNode

        if ($isNode("List", parent)) {
            const listType = parent.getListType();
            const value = listType === "number" ? 1 : undefined;
            const newListItem = $createNode("ListItem", { value });
            const newList = $createNode("List", { listType });
            newListItem.append(newList);
            newList.append(listItemNode);

            if (previousSibling) {
                previousSibling.insertAfter(newListItem);
            } else if (nextSibling) {
                nextSibling.insertBefore(newListItem);
            } else {
                parent.append(newListItem);
            }
        }
    }
}

/**
 * Removes an indent by removing an empty ListNode/ListItemNode chain. An indented ListItemNode
 * has a great grandparent node of type ListNode, which is where the ListItemNode will reside
 * within as a child.
 * @param listItemNode - The ListItemNode to remove the indent (outdent).
 */
export function $handleOutdent(listItemNode: ListItemNode): void {
    // go through each node and decide where to move it.

    if (isNestedListNode(listItemNode)) {
        return;
    }
    const parentList = getParent(listItemNode);
    const grandparentListItem = parentList ? getParent(parentList) : undefined;
    const greatGrandparentList = grandparentListItem
        ? getParent(grandparentListItem)
        : undefined;
    // If it doesn't have these ancestors, it's not indented.

    if (
        $isNode("List", greatGrandparentList) &&
        $isNode("ListItem", grandparentListItem) &&
        $isNode("List", parentList)
    ) {
        // if it's the first child in it's parent list, insert it into the
        // great grandparent list before the grandparent
        const firstChild = parentList ? parentList.getFirstChild() : undefined;
        const lastChild = parentList ? parentList.getLastChild() : undefined;

        if (listItemNode.is(firstChild)) {
            grandparentListItem.insertBefore(listItemNode);

            if (parentList.isEmpty()) {
                grandparentListItem.remove();
            }
            // if it's the last child in it's parent list, insert it into the
            // great grandparent list after the grandparent.
        } else if (listItemNode.is(lastChild)) {
            grandparentListItem.insertAfter(listItemNode);

            if (parentList.isEmpty()) {
                grandparentListItem.remove();
            }
        } else {
            // otherwise, we need to split the siblings into two new nested lists
            const listType = parentList.getListType();
            const previousSiblingsListItem = $createNode("ListItem", {});
            const previousSiblingsList = $createNode("List", { listType });
            previousSiblingsListItem.append(previousSiblingsList);
            getPreviousSiblings(listItemNode).forEach((sibling) => previousSiblingsList.append(sibling));
            const nextSiblingsListItem = $createNode("ListItem", {});
            const nextSiblingsList = $createNode("List", { listType });
            nextSiblingsListItem.append(nextSiblingsList);
            append(nextSiblingsList, getNextSiblings(listItemNode));
            // put the sibling nested lists on either side of the grandparent list item in the great grandparent.
            grandparentListItem.insertBefore(previousSiblingsListItem);
            grandparentListItem.insertAfter(nextSiblingsListItem);
            // replace the grandparent list item (now between the siblings) with the outdented list item.
            grandparentListItem.replace(listItemNode);
        }
    }
}
