import { LexicalEditor } from "../editor";
import { RangeSelection } from "../selection";
import { BaseSelection, DOMConversionMap, DOMConversionOutput, DOMExportOutput, EditorConfig, EditorThemeClasses, NodeKey, SerializedListItemNode } from "../types";
import { $applyNodeReplacement, $isElementNode, $isRangeSelection, addClassNamesToElement, append, isHTMLElement, isNestedListNode, normalizeClassNames, removeClassNamesFromElement } from "../utils";
import { ElementNode } from "./ElementNode";
import { LexicalNode } from "./LexicalNode";
import { $createListNode, $isListNode, ListNode, mergeLists } from "./ListNode";
import { $createParagraphNode, $isParagraphNode, ParagraphNode } from "./ParagraphNode";

export class ListItemNode extends ElementNode {
    /** @internal */
    __value: number;
    /** @internal */
    __checked?: boolean;

    static getType(): string {
        return "listitem";
    }

    static clone(node: ListItemNode): ListItemNode {
        return new ListItemNode(node.__value, node.__checked, node.__key);
    }

    constructor(value?: number, checked?: boolean, key?: NodeKey) {
        super(key);
        this.__value = value === undefined ? 1 : value;
        this.__checked = checked;
    }

    createDOM(config: EditorConfig): HTMLElement {
        const element = document.createElement("li");
        const parent = this.getParent();
        if ($isListNode(parent) && parent.getListType() === "check") {
            updateListItemChecked(element, this, null, parent);
        }
        element.value = this.__value;
        $setListItemThemeClassNames(element, config.theme, this);
        return element;
    }

    updateDOM(
        prevNode: ListItemNode,
        dom: HTMLElement,
        config: EditorConfig,
    ): boolean {
        const parent = this.getParent();
        if ($isListNode(parent) && parent.getListType() === "check") {
            updateListItemChecked(dom, this, prevNode, parent);
        }
        // @ts-expect-error - this is always HTMLListItemElement
        dom.value = this.__value;
        $setListItemThemeClassNames(dom, config.theme, this);

        return false;
    }

    static transform(): (node: LexicalNode) => void {
        return (node: LexicalNode) => {
            if (!$isListItemNode(node)) {
                throw new Error("node is not a ListItemNode");
            }
            if (node.__checked == null) {
                return;
            }
            const parent = node.getParent();
            if ($isListNode(parent)) {
                if (parent.getListType() !== "check" && node.getChecked() != null) {
                    node.setChecked(undefined);
                }
            }
        };
    }

    static importDOM(): DOMConversionMap | null {
        return {
            li: (node: Node) => ({
                conversion: convertListItemElement,
                priority: 0,
            }),
        };
    }

    static importJSON(serializedNode: SerializedListItemNode): ListItemNode {
        const node = $createListItemNode();
        node.setChecked(serializedNode.checked);
        node.setValue(serializedNode.value);
        node.setFormat(serializedNode.format);
        node.setDirection(serializedNode.direction);
        return node;
    }

    exportDOM(editor: LexicalEditor): DOMExportOutput {
        const element = this.createDOM(editor._config);
        element.style.textAlign = this.getFormatType();
        return {
            element,
        };
    }

    exportJSON(): SerializedListItemNode {
        return {
            ...super.exportJSON(),
            checked: this.getChecked(),
            type: "listitem",
            value: this.getValue(),
            version: 1,
        };
    }

    append(...nodes: LexicalNode[]): this {
        for (let i = 0; i < nodes.length; i++) {
            const node = nodes[i];

            if ($isElementNode(node) && this.canMergeWith(node)) {
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
        if ($isListItemNode(replaceWithNode)) {
            return super.replace(replaceWithNode);
        }
        this.setIndent(0);
        const list = this.getParentOrThrow();
        if (!$isListNode(list)) {
            return replaceWithNode;
        }
        if (list.__first === this.getKey()) {
            list.insertBefore(replaceWithNode);
        } else if (list.__last === this.getKey()) {
            list.insertAfter(replaceWithNode);
        } else {
            // Split the list
            const newList = $createListNode(list.getListType());
            let nextSibling = this.getNextSibling();
            while (nextSibling) {
                const nodeToAppend = nextSibling;
                nextSibling = nextSibling.getNextSibling();
                newList.append(nodeToAppend);
            }
            list.insertAfter(replaceWithNode);
            replaceWithNode.insertAfter(newList);
        }
        if (includeChildren) {
            if (!$isElementNode(replaceWithNode)) {
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
        const listNode = this.getParentOrThrow();

        if (!$isListNode(listNode)) {
            throw new Error("insertAfter: list node is not parent of list item node");
        }

        if ($isListItemNode(node)) {
            return super.insertAfter(node, restoreSelection);
        }

        const siblings = this.getNextSiblings();

        // Split the lists and insert the node in between them
        listNode.insertAfter(node, restoreSelection);

        if (siblings.length !== 0) {
            const newListNode = $createListNode(listNode.getListType());

            siblings.forEach((sibling) => newListNode.append(sibling));

            node.insertAfter(newListNode, restoreSelection);
        }

        return node;
    }

    remove(preserveEmptyParent?: boolean): void {
        const prevSibling = this.getPreviousSibling();
        const nextSibling = this.getNextSibling();
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
        const newElement = $createListItemNode(
            this.__checked == null ? undefined : false,
        );
        this.insertAfter(newElement, restoreSelection);

        return newElement;
    }

    collapseAtStart(selection: RangeSelection): true {
        const paragraph = $createParagraphNode();
        const children = this.getChildren();
        children.forEach((child) => paragraph.append(child));
        const listNode = this.getParentOrThrow();
        const listNodeParent = listNode.getParentOrThrow();
        const isIndented = $isListItemNode(listNodeParent);

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
                const key = paragraph.getKey();

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
        const parent = this.getParent();
        if (parent === null) {
            return this.getLatest().__indent;
        }
        // ListItemNode should always have a ListNode for a parent.
        let listNodeParent = parent.getParentOrThrow();
        let indentLevel = 0;
        while ($isListItemNode(listNodeParent)) {
            listNodeParent = listNodeParent.getParentOrThrow().getParentOrThrow();
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
        return $isListItemNode(node);
    }

    canReplaceWith(replacement: LexicalNode): boolean {
        return $isListItemNode(replacement);
    }

    canMergeWith(node: LexicalNode): boolean {
        return $isParagraphNode(node) || $isListItemNode(node);
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
        return $createListNode("bullet");
    }
}

const $setListItemThemeClassNames = (
    dom: HTMLElement,
    editorThemeClasses: EditorThemeClasses,
    node: ListItemNode,
): void => {
    const classesToAdd: (string | undefined)[] = [];
    const classesToRemove: (string | undefined)[] = [];
    const listTheme = editorThemeClasses.list;
    const listItemClassName = listTheme ? listTheme.listitem : undefined;
    let nestedListItemClassName;

    if (listTheme && listTheme.nested) {
        nestedListItemClassName = listTheme.nested.listitem;
    }

    if (listItemClassName !== undefined) {
        classesToAdd.push(...normalizeClassNames(listItemClassName));
    }

    if (listTheme) {
        const parentNode = node.getParent();
        const isCheckList =
            $isListNode(parentNode) && parentNode.getListType() === "check";
        const checked = node.getChecked();

        if (!isCheckList || checked) {
            classesToRemove.push(listTheme.listitemUnchecked);
        }

        if (!isCheckList || !checked) {
            classesToRemove.push(listTheme.listitemChecked);
        }

        if (isCheckList) {
            classesToAdd.push(
                checked ? listTheme.listitemChecked : listTheme.listitemUnchecked,
            );
        }
    }

    if (nestedListItemClassName !== undefined) {
        const nestedListItemClasses = normalizeClassNames(nestedListItemClassName);

        if (node.getChildren().some((child) => $isListNode(child))) {
            classesToAdd.push(...nestedListItemClasses);
        } else {
            classesToRemove.push(...nestedListItemClasses);
        }
    }

    if (classesToRemove.length > 0) {
        removeClassNamesFromElement(dom, ...classesToRemove);
    }

    if (classesToAdd.length > 0) {
        addClassNamesToElement(dom, ...classesToAdd);
    }
};

function updateListItemChecked(
    dom: HTMLElement,
    listItemNode: ListItemNode,
    prevListItemNode: ListItemNode | null,
    listNode: ListNode,
): void {
    // Only add attributes for leaf list items
    if ($isListNode(listItemNode.getFirstChild())) {
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

const convertListItemElement = (domNode: Node): DOMConversionOutput => {
    const checked =
        isHTMLElement(domNode) && domNode.getAttribute("aria-checked") === "true";
    return { node: $createListItemNode(checked) };
};

/**
 * Creates a new List Item node, passing true/false will convert it to a checkbox input.
 * @param checked - Is the List Item a checkbox and, if so, is it checked? undefined/null: not a checkbox, true/false is a checkbox and checked/unchecked, respectively.
 * @returns The new List Item.
 */
export const $createListItemNode = (checked?: boolean): ListItemNode => {
    return $applyNodeReplacement(new ListItemNode(undefined, checked));
};

/**
 * Checks to see if the node is a ListItemNode.
 * @param node - The node to be checked.
 * @returns true if the node is a ListItemNode, false otherwise.
 */
export const $isListItemNode = (
    node: LexicalNode | null | undefined,
): node is ListItemNode => {
    return node instanceof ListItemNode;
};

/**
 * Adds an empty ListNode/ListItemNode chain at listItemNode, so as to
 * create an indent effect. Won't indent ListItemNodes that have a ListNode as
 * a child, but does merge sibling ListItemNodes if one has a nested ListNode.
 * @param listItemNode - The ListItemNode to be indented.
 */
export const $handleIndent = (listItemNode: ListItemNode): void => {
    // go through each node and decide where to move it.
    const removed = new Set<NodeKey>();

    if (isNestedListNode(listItemNode) || removed.has(listItemNode.getKey())) {
        return;
    }

    const parent = listItemNode.getParent();

    // We can cast both of the below `isNestedListNode` only returns a boolean type instead of a user-defined type guards
    const nextSibling =
        listItemNode.getNextSibling<ListItemNode>() as ListItemNode;
    const previousSibling =
        listItemNode.getPreviousSibling<ListItemNode>() as ListItemNode;
    // if there are nested lists on either side, merge them all together.

    if (isNestedListNode(nextSibling) && isNestedListNode(previousSibling)) {
        const innerList = previousSibling.getFirstChild();

        if ($isListNode(innerList)) {
            innerList.append(listItemNode);
            const nextInnerList = nextSibling.getFirstChild();

            if ($isListNode(nextInnerList)) {
                const children = nextInnerList.getChildren();
                append(innerList, children);
                nextSibling.remove();
                removed.add(nextSibling.getKey());
            }
        }
    } else if (isNestedListNode(nextSibling)) {
        // if the ListItemNode is next to a nested ListNode, merge them
        const innerList = nextSibling.getFirstChild();

        if ($isListNode(innerList)) {
            const firstChild = innerList.getFirstChild();

            if (firstChild !== null) {
                firstChild.insertBefore(listItemNode);
            }
        }
    } else if (isNestedListNode(previousSibling)) {
        const innerList = previousSibling.getFirstChild();

        if ($isListNode(innerList)) {
            innerList.append(listItemNode);
        }
    } else {
        // otherwise, we need to create a new nested ListNode

        if ($isListNode(parent)) {
            const newListItem = $createListItemNode();
            const newList = $createListNode(parent.getListType());
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
};

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
    const parentList = listItemNode.getParent();
    const grandparentListItem = parentList ? parentList.getParent() : undefined;
    const greatGrandparentList = grandparentListItem
        ? grandparentListItem.getParent()
        : undefined;
    // If it doesn't have these ancestors, it's not indented.

    if (
        $isListNode(greatGrandparentList) &&
        $isListItemNode(grandparentListItem) &&
        $isListNode(parentList)
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
            const previousSiblingsListItem = $createListItemNode();
            const previousSiblingsList = $createListNode(listType);
            previousSiblingsListItem.append(previousSiblingsList);
            listItemNode
                .getPreviousSiblings()
                .forEach((sibling) => previousSiblingsList.append(sibling));
            const nextSiblingsListItem = $createListItemNode();
            const nextSiblingsList = $createListNode(listType);
            nextSiblingsListItem.append(nextSiblingsList);
            append(nextSiblingsList, listItemNode.getNextSiblings());
            // put the sibling nested lists on either side of the grandparent list item in the great grandparent.
            grandparentListItem.insertBefore(previousSiblingsListItem);
            grandparentListItem.insertAfter(nextSiblingsListItem);
            // replace the grandparent list item (now between the siblings) with the outdented list item.
            grandparentListItem.replace(listItemNode);
        }
    }
}
