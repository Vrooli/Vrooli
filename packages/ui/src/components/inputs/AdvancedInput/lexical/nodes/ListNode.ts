/* eslint-disable @typescript-eslint/ban-ts-comment */
import { LIST_INDENT_SIZE } from "../consts.js";
import { type LexicalEditor } from "../editor.js";
import { type CustomDomElement, type DOMConversionMap, type DOMConversionOutput, type DOMExportOutput, type EditorConfig, type ElementTransformer, type ListNodeTagType, type ListType, type NodeConstructorPayloads, type NodeType, type SerializedListNode } from "../types.js";
import { $createNode, $getAllListItems, $getListDepth, $getNearestNodeOfType, $getSelection, $getTopListNode, $isLeafNode, $isNode, $isRangeSelection, $isRootOrShadowRoot, $removeHighestEmptyListParent, addClassNamesToElement, append, getNextSibling, getNextSiblings, getParent, getPreviousSibling, isHTMLElement, isNestedListNode, normalizeClassNames, removeClassNamesFromElement, wrapInListItem } from "../utils.js";
import { ElementNode } from "./ElementNode.js";
import { type LexicalNode } from "./LexicalNode.js";
import { type ListItemNode } from "./ListItemNode.js";
import { type ParagraphNode } from "./ParagraphNode.js";

export class ListNode extends ElementNode {
    static __type: NodeType = "List";
    __tag: ListNodeTagType;
    __start: number;
    __listType: ListType;

    static clone(node: ListNode): ListNode {
        const { __listType, __start, __key } = node;
        return $createNode("List", { listType: __listType, start: __start, key: __key });
    }

    constructor({ listType, start, ...rest }: NodeConstructorPayloads["List"]) {
        super(rest);
        const _listType = TAG_TO_LIST_TYPE[listType] || listType;
        this.__listType = _listType;
        this.__tag = _listType === "number" ? "ol" : "ul";
        this.__start = start === undefined ? 1 : start;
    }

    getTag(): ListNodeTagType {
        return this.__tag;
    }

    setListType(type: ListType): void {
        const writable = this.getWritable();
        writable.__listType = type;
        writable.__tag = type === "number" ? "ol" : "ul";
    }

    getListType(): ListType {
        return this.__listType;
    }

    getStart(): number {
        return this.__start;
    }

    // View

    createDOM(): HTMLElement {
        const tag = this.__tag;
        const dom = document.createElement(tag) as CustomDomElement<HTMLOListElement | HTMLUListElement>;

        if (this.__start !== 1) {
            dom.__lexicalListStart = String(this.__start);
        }
        dom.__lexicalListType = this.__listType;

        return dom;
    }

    updateDOM(
        prevNode: ListNode,
        dom: HTMLElement,
        config: EditorConfig,
    ): boolean {
        if (prevNode.__tag !== this.__tag) {
            return true;
        }

        return false;
    }

    static transform(): (node: LexicalNode) => void {
        return (node: LexicalNode) => {
            if (!$isNode("List", node)) {
                throw new Error("node is not a ListNode");
            }
            mergeNextSiblingListIfSameType(node);
            updateChildrenListItemValue(node);
        };
    }

    static importDOM(): DOMConversionMap {
        return {
            ol: (node: Node) => ({
                conversion: convertListNode,
                priority: 0,
            }),
            ul: (node: Node) => ({
                conversion: convertListNode,
                priority: 0,
            }),
        };
    }

    static importJSON({ direction, format, indent, listType, start }: SerializedListNode): ListNode {
        const node = $createNode("List", { listType, start });
        node.setFormat(format);
        node.setIndent(indent);
        node.setDirection(direction);
        return node;
    }

    exportDOM(): DOMExportOutput {
        const { element } = super.exportDOM();
        if (element && isHTMLElement(element)) {
            if (this.__start !== 1) {
                (element as CustomDomElement).__lexicalListStart = String(this.__start);
            }
            if (this.__listType === "check") {
                (element as CustomDomElement).__lexicalListType = "check";
            }
        }
        return {
            element,
        };
    }

    exportJSON(): SerializedListNode {
        return {
            ...super.exportJSON(),
            __type: "List",
            listType: this.getListType(),
            start: this.getStart(),
            tag: this.getTag(),
            version: 1,
        };
    }

    canBeEmpty(): false {
        return false;
    }

    canIndent(): false {
        return false;
    }

    append(...nodesToAppend: LexicalNode[]): this {
        for (let i = 0; i < nodesToAppend.length; i++) {
            const currentNode = nodesToAppend[i];

            if ($isNode("ListItem", currentNode)) {
                super.append(currentNode);
            } else {
                const listItemNode = $createNode("ListItem", {});

                if ($isNode("List", currentNode)) {
                    listItemNode.append(currentNode);
                } else if ($isNode("Element", currentNode)) {
                    const textNode = $createNode("Text", { text: currentNode.getTextContent() });
                    listItemNode.append(textNode);
                } else {
                    listItemNode.append(currentNode);
                }
                super.append(listItemNode);
            }
        }
        return this;
    }

    extractWithChild(child: LexicalNode): boolean {
        return $isNode("ListItem", child);
    }
}

/**
 * Takes the value of a child ListItemNode and makes it the value the ListItemNode
 * should be if it isn't already. Also ensures that checked is undefined if the
 * parent does not have a list type of 'check'.
 * @param list - The list whose children are updated.
 */
export function updateChildrenListItemValue(list: ListNode): void {
    const isNotChecklist = list.getListType() !== "check";
    let value = list.getStart();
    for (const child of list.getChildren()) {
        if ($isNode("ListItem", child)) {
            if (child.getValue() !== value) {
                child.setValue(value);
            }
            if (isNotChecklist && child.getChecked() !== null) {
                child.setChecked(undefined);
            }
            if (!$isNode("List", child.getFirstChild())) {
                value++;
            }
        }
    }
}

function setListThemeClassNames(
    dom: HTMLElement,
    node: ListNode,
): void {
    const classesToAdd: string[] = [];
    const classesToRemove: string[] = [];
    const listTheme = {} as any;

    if (listTheme !== undefined) {
        const listLevelsClassNames = listTheme[`${node.__tag}Depth`] || [];
        const listDepth = $getListDepth(node) - 1;
        const normalizedListDepth = listDepth % listLevelsClassNames.length;
        const listLevelClassName = listLevelsClassNames[normalizedListDepth];
        const listClassName = listTheme[node.__tag];
        let nestedListClassName;
        const nestedListTheme = listTheme.nested;
        const checklistClassName = listTheme.checklist;

        if (nestedListTheme !== undefined && nestedListTheme.list) {
            nestedListClassName = nestedListTheme.list;
        }

        if (listClassName !== undefined) {
            classesToAdd.push(listClassName);
        }

        if (checklistClassName !== undefined && node.__listType === "check") {
            classesToAdd.push(checklistClassName);
        }

        if (listLevelClassName !== undefined) {
            classesToAdd.push(...normalizeClassNames(listLevelClassName));
            for (let i = 0; i < listLevelsClassNames.length; i++) {
                if (i !== normalizedListDepth) {
                    classesToRemove.push(node.__tag + i);
                }
            }
        }

        if (nestedListClassName !== undefined) {
            const nestedListItemClasses = normalizeClassNames(nestedListClassName);

            if (listDepth > 1) {
                classesToAdd.push(...nestedListItemClasses);
            } else {
                classesToRemove.push(...nestedListItemClasses);
            }
        }
    }

    if (classesToRemove.length > 0) {
        removeClassNamesFromElement(dom, ...classesToRemove);
    }

    if (classesToAdd.length > 0) {
        addClassNamesToElement(dom, ...classesToAdd);
    }
}

/*
 * This function normalizes the children of a ListNode after the conversion from HTML,
 * ensuring that they are all ListItemNodes and contain either a single nested ListNode
 * or some other inline content.
 */
function normalizeChildren(nodes: LexicalNode[]): Array<ListItemNode> {
    const normalizedListItems: ListItemNode[] = [];
    for (let i = 0; i < nodes.length; i++) {
        const node = nodes[i];
        if ($isNode("ListItem", node)) {
            normalizedListItems.push(node);
            const children = node.getChildren();
            if (children.length > 1) {
                children.forEach((child) => {
                    if ($isNode("List", child)) {
                        normalizedListItems.push(wrapInListItem(child));
                    }
                });
            }
        } else {
            normalizedListItems.push(wrapInListItem(node));
        }
    }
    return normalizedListItems;
}

function convertListNode(domNode: Node): DOMConversionOutput {
    const nodeName = domNode.nodeName.toLowerCase();
    let node: ListNode | null = null;
    if (nodeName === "ol") {
        const start = +((domNode as CustomDomElement).__lexicalListStart ?? "0");
        node = $createNode("List", { listType: "number", start });
    } else if (nodeName === "ul") {
        if (
            isHTMLElement(domNode) &&
            (domNode as CustomDomElement).__lexicalListType === "check"
        ) {
            node = $createNode("List", { listType: "check" });
        } else {
            node = $createNode("List", { listType: "bullet" });
        }
    }

    return {
        after: normalizeChildren,
        node,
    };
}

const TAG_TO_LIST_TYPE: Record<string, ListType> = {
    ol: "number",
    ul: "bullet",
};

function createListOrMerge(node: ElementNode, listType: ListType): ListNode {
    if ($isNode("List", node)) {
        return node;
    }

    const previousSibling = getPreviousSibling(node);
    const nextSibling = getNextSibling(node);
    const listItem = $createNode("ListItem", {});
    listItem.setFormat(node.getFormatType());
    listItem.setIndent(node.getIndent());
    append(listItem, node.getChildren());

    if (
        $isNode("List", previousSibling) &&
        listType === previousSibling.getListType()
    ) {
        previousSibling.append(listItem);
        node.remove();
        // if the same type of list is on both sides, merge them.

        if ($isNode("List", nextSibling) && listType === nextSibling.getListType()) {
            append(previousSibling, nextSibling.getChildren());
            nextSibling.remove();
        }
        return previousSibling;
    } else if (
        $isNode("List", nextSibling) &&
        listType === nextSibling.getListType()
    ) {
        nextSibling.getFirstChildOrThrow().insertBefore(listItem);
        node.remove();
        return nextSibling;
    } else {
        const list = $createNode("List", { listType });
        list.append(listItem);
        node.replace(list);
        return list;
    }
}

/**
 * Inserts a new ListNode. If the selection's anchor node is an empty ListItemNode and is a child of
 * the root/shadow root, it will replace the ListItemNode with a ListNode and the old ListItemNode.
 * Otherwise it will replace its parent with a new ListNode and re-insert the ListItemNode and any previous children.
 * If the selection's anchor node is not an empty ListItemNode, it will add a new ListNode or merge an existing ListNode,
 * unless the the node is a leaf node, in which case it will attempt to find a ListNode up the branch and replace it with
 * a new ListNode, or create a new ListNode at the nearest root/shadow root.
 * @param editor - The lexical editor.
 * @param listType - The type of list, "number" | "bullet" | "check".
 */
export function insertList(editor: LexicalEditor, listType: ListType): void {
    editor.update(() => {
        const selection = $getSelection();

        if (selection !== null) {
            const nodes = selection.getNodes();
            if ($isRangeSelection(selection)) {
                const anchorAndFocus = selection.getStartEndPoints();
                if (anchorAndFocus === null) {
                    throw new Error("insertList: anchor should be defined");
                }
                const [anchor] = anchorAndFocus;
                const anchorNode = anchor.getNode();
                const anchorNodeParent = getParent(anchorNode);

                if ($isSelectingEmptyListItem(anchorNode, nodes)) {
                    const list = $createNode("List", { listType });

                    if ($isRootOrShadowRoot(anchorNodeParent)) {
                        anchorNode.replace(list);
                        const listItem = $createNode("ListItem", {});
                        if ($isNode("Element", anchorNode)) {
                            listItem.setFormat(anchorNode.getFormatType());
                            listItem.setIndent(anchorNode.getIndent());
                        }
                        list.append(listItem);
                    } else if ($isNode("ListItem", anchorNode)) {
                        const parent = getParent(anchorNode, { throwIfNull: true });
                        append(list, parent.getChildren());
                        parent.replace(list);
                    }

                    return;
                }
            }

            const handled = new Set();
            for (let i = 0; i < nodes.length; i++) {
                const node = nodes[i];

                if (
                    $isNode("Element", node) &&
                    node.isEmpty() &&
                    !$isNode("ListItem", node) &&
                    !handled.has(node.__key)
                ) {
                    createListOrMerge(node, listType);
                    continue;
                }

                if ($isLeafNode(node)) {
                    let parent = getParent(node);
                    while (parent !== null) {
                        const parentKey = parent.__key;

                        if ($isNode("List", parent)) {
                            if (!handled.has(parentKey)) {
                                const newListNode = $createNode("List", { listType });
                                append(newListNode, parent.getChildren());
                                parent.replace(newListNode);
                                handled.add(parentKey);
                            }

                            break;
                        } else {
                            const nextParent = getParent(parent);

                            if ($isRootOrShadowRoot(nextParent) && !handled.has(parentKey)) {
                                handled.add(parentKey);
                                createListOrMerge(parent, listType);
                                break;
                            }

                            parent = nextParent;
                        }
                    }
                }
            }
        }
    });
}

/**
 * Attempts to insert a ParagraphNode at selection and selects the new node. The selection must contain a ListItemNode
 * or a node that does not already contain text. If its grandparent is the root/shadow root, it will get the ListNode
 * (which should be the parent node) and insert the ParagraphNode as a sibling to the ListNode. If the ListNode is
 * nested in a ListItemNode instead, it will add the ParagraphNode after the grandparent ListItemNode.
 * Throws an invariant if the selection is not a child of a ListNode.
 * @returns true if a ParagraphNode was inserted succesfully, false if there is no selection
 * or the selection does not contain a ListItemNode or the node already holds text.
 */
export function $handleListInsertParagraph(): boolean {
    const selection = $getSelection();

    if (!$isRangeSelection(selection) || !selection.isCollapsed()) {
        return false;
    }
    // Only run this code on empty list items
    const anchor = selection.anchor.getNode();

    if (!$isNode("ListItem", anchor) || anchor.getChildrenSize() !== 0) {
        return false;
    }
    const topListNode = $getTopListNode(anchor);
    const parent = getParent(anchor);

    if (!$isNode("List", parent)) {
        throw new Error("A ListItemNode must have a ListNode for a parent.");
    }

    const grandparent = getParent(parent);

    let replacementNode;

    if ($isRootOrShadowRoot(grandparent)) {
        replacementNode = $createNode("Paragraph", {});
        topListNode.insertAfter(replacementNode);
    } else if ($isNode("ListItem", grandparent)) {
        replacementNode = $createNode("ListItem", {});
        grandparent.insertAfter(replacementNode);
    } else {
        return false;
    }
    replacementNode.select();

    const nextSiblings = getNextSiblings(anchor);

    if (nextSiblings.length > 0) {
        const newList = $createNode("List", { listType: parent.getListType() });

        if ($isNode("Paragraph", replacementNode)) {
            replacementNode.insertAfter(newList);
        } else {
            const newListItem = $createNode("ListItem", {});
            newListItem.append(newList);
            replacementNode.insertAfter(newListItem);
        }
        nextSiblings.forEach((sibling) => {
            sibling.remove();
            newList.append(sibling);
        });
    }

    // Don't leave hanging nested empty lists
    $removeHighestEmptyListParent(anchor);

    return true;
}

/**
 * Searches for the nearest ancestral ListNode and removes it. If selection is an empty ListItemNode
 * it will remove the whole list, including the ListItemNode. For each ListItemNode in the ListNode,
 * removeList will also generate new ParagraphNodes in the removed ListNode's place. Any child node
 * inside a ListItemNode will be appended to the new ParagraphNodes.
 * @param editor - The lexical editor.
 */
export function removeList(editor: LexicalEditor): void {
    editor.update(() => {
        const selection = $getSelection();

        if ($isRangeSelection(selection)) {
            const listNodes = new Set<ListNode>();
            const nodes = selection.getNodes();
            const anchorNode = selection.anchor.getNode();

            if ($isSelectingEmptyListItem(anchorNode, nodes)) {
                listNodes.add($getTopListNode(anchorNode));
            } else {
                for (let i = 0; i < nodes.length; i++) {
                    const node = nodes[i];

                    if ($isLeafNode(node)) {
                        const listItemNode = $getNearestNodeOfType("ListItem", node);

                        if (listItemNode !== null) {
                            listNodes.add($getTopListNode(listItemNode));
                        }
                    }
                }
            }

            for (const listNode of listNodes) {
                let insertionPoint: ListNode | ParagraphNode = listNode;

                const listItems = $getAllListItems(listNode);

                for (const listItemNode of listItems) {
                    const paragraph = $createNode("Paragraph", {});

                    append(paragraph, listItemNode.getChildren());

                    insertionPoint.insertAfter(paragraph);
                    insertionPoint = paragraph;

                    // When the anchor and focus fall on the textNode
                    // we don't have to change the selection because the textNode will be appended to
                    // the newly generated paragraph.
                    // When selection is in empty nested list item, selection is actually on the listItemNode.
                    // When the corresponding listItemNode is deleted and replaced by the newly generated paragraph
                    // we should manually set the selection's focus and anchor to the newly generated paragraph.
                    if (listItemNode.__key === selection.anchor.key) {
                        selection.anchor.set(paragraph.__key, 0, "element");
                    }
                    if (listItemNode.__key === selection.focus.key) {
                        selection.focus.set(paragraph.__key, 0, "element");
                    }

                    listItemNode.remove();
                }
                listNode.remove();
            }
        }
    });
}

function $isSelectingEmptyListItem(
    anchorNode: ListItemNode | LexicalNode,
    nodes: Array<LexicalNode>,
): boolean {
    return (
        $isNode("ListItem", anchorNode) &&
        (nodes.length === 0 ||
            (nodes.length === 1 &&
                anchorNode.__key !== nodes[0]?.__key &&
                anchorNode.getChildrenSize() === 0))
    );
}

/**
 * A recursive function that goes through each list and their children, including nested lists,
 * appending list2 children after list1 children and updating ListItemNode values.
 * @param list1 - The first list to be merged.
 * @param list2 - The second list to be merged.
 */
export function mergeLists(list1: ListNode, list2: ListNode): void {
    const listItem1 = list1.getLastChild();
    const listItem2 = list2.getFirstChild();

    if (
        listItem1 &&
        listItem2 &&
        isNestedListNode(listItem1) &&
        isNestedListNode(listItem2)
    ) {
        mergeLists(listItem1.getFirstChild(), listItem2.getFirstChild());
        listItem2.remove();
    }

    const toMerge = list2.getChildren();
    if (toMerge.length > 0) {
        list1.append(...toMerge);
    }

    list2.remove();
}

/**
 * Merge the next sibling list if same type.
 * <ul> will merge with <ul>, but NOT <ul> with <ol>.
 * @param list - The list whose next sibling should be potentially merged
 */
export function mergeNextSiblingListIfSameType(list: ListNode): void {
    const nextSibling = getNextSibling(list);
    if (
        $isNode("List", nextSibling) &&
        list.getListType() === nextSibling.getListType()
    ) {
        mergeLists(list, nextSibling);
    }
}

function getIndent(whitespaces: string): number {
    const tabs = whitespaces.match(/\t/g);
    const spaces = whitespaces.match(/ /g);

    let indent = 0;

    if (tabs) {
        indent += tabs.length;
    }

    if (spaces) {
        indent += Math.floor(spaces.length / LIST_INDENT_SIZE);
    }

    return indent;
}

export function listExport(
    listNode: ListNode,
    exportChildren: (node: ElementNode) => string,
    depth: number,
): string {
    const output: string[] = [];
    const children = listNode.getChildren();
    let index = 0;
    for (const listItemNode of children) {
        if ($isNode("ListItem", listItemNode)) {
            if (listItemNode.getChildrenSize() === 1) {
                const firstChild = listItemNode.getFirstChild();
                if ($isNode("List", firstChild)) {
                    output.push(listExport(firstChild, exportChildren, depth + 1));
                    continue;
                }
            }
            const indent = " ".repeat(depth * LIST_INDENT_SIZE);
            const listType = listNode.getListType();
            const prefix =
                listType === "number"
                    ? `${listNode.getStart() + index}. `
                    : listType === "check"
                        ? `- [${listItemNode.getChecked() ? "x" : " "}] `
                        : "- ";
            output.push(indent + prefix + exportChildren(listItemNode));
            index++;
        }
    }

    return output.join("\n");
}

export function listReplace(listType: ListType): ElementTransformer["replace"] {
    return (parentNode, children, match) => {
        const previousNode = getPreviousSibling(parentNode);
        const nextNode = getNextSibling(parentNode);
        const checked = listType === "check" ? match[3] === "x" : undefined;
        const listItem = $createNode("ListItem", { checked });
        if ($isNode("List", nextNode) && nextNode.getListType() === listType) {
            const firstChild = nextNode.getFirstChild();
            if (firstChild !== null) {
                firstChild.insertBefore(listItem);
            } else {
                // should never happen, but let's handle gracefully, just in case.
                nextNode.append(listItem);
            }
            parentNode.remove();
        } else if (
            $isNode("List", previousNode) &&
            previousNode.getListType() === listType
        ) {
            previousNode.append(listItem);
            parentNode.remove();
        } else {
            const start = listType === "number" ? Number(match[1]) : 1;
            const list = $createNode("List", { listType, start });
            list.append(listItem);
            parentNode.replace(list);
        }
        listItem.append(...children);
        listItem.select(0, 0);
        const indent = getIndent(match[1]);
        if (indent) {
            listItem.setIndent(indent);
        }
    };
}
