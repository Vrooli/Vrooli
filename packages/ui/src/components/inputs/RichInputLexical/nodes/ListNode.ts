/* eslint-disable @typescript-eslint/ban-ts-comment */
import { LIST_INDENT_SIZE } from "../consts";
import { type LexicalEditor } from "../editor";
import { DOMConversionMap, DOMConversionOutput, DOMExportOutput, EditorConfig, EditorThemeClasses, ElementTransformer, ListNodeTagType, ListType, NodeKey, SerializedListNode } from "../types";
import { $applyNodeReplacement, $createTextNode, $getAllListItems, $getListDepth, $getNearestNodeOfType, $getSelection, $getTopListNode, $isElementNode, $isLeafNode, $isRangeSelection, $isRootOrShadowRoot, $removeHighestEmptyListParent, addClassNamesToElement, append, isHTMLElement, isNestedListNode, normalizeClassNames, removeClassNamesFromElement, wrapInListItem } from "../utils";
import { ElementNode } from "./ElementNode";
import { LexicalNode } from "./LexicalNode";
import { $createListItemNode, $isListItemNode, ListItemNode } from "./ListItemNode";
import { $createParagraphNode, $isParagraphNode, ParagraphNode } from "./ParagraphNode";

export class ListNode extends ElementNode {
    /** @internal */
    __tag: ListNodeTagType;
    /** @internal */
    __start: number;
    /** @internal */
    __listType: ListType;

    static getType(): string {
        return "list";
    }

    static clone(node: ListNode): ListNode {
        const listType = node.__listType || TAG_TO_LIST_TYPE[node.__tag];

        return new ListNode(listType, node.__start, node.__key);
    }

    constructor(listType: ListType, start: number, key?: NodeKey) {
        super(key);
        const _listType = TAG_TO_LIST_TYPE[listType] || listType;
        this.__listType = _listType;
        this.__tag = _listType === "number" ? "ol" : "ul";
        this.__start = start;
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

    createDOM(config: EditorConfig, _editor?: LexicalEditor): HTMLElement {
        const tag = this.__tag;
        const dom = document.createElement(tag);

        if (this.__start !== 1) {
            dom.setAttribute("start", String(this.__start));
        }
        // @ts-expect-error Internal field.
        dom.__lexicalListType = this.__listType;
        setListThemeClassNames(dom, config.theme, this);

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

        setListThemeClassNames(dom, config.theme, this);

        return false;
    }

    static transform(): (node: LexicalNode) => void {
        return (node: LexicalNode) => {
            if (!$isListNode(node)) {
                throw new Error("node is not a ListNode");
            }
            mergeNextSiblingListIfSameType(node);
            updateChildrenListItemValue(node);
        };
    }

    static importDOM(): DOMConversionMap | null {
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

    static importJSON(serializedNode: SerializedListNode): ListNode {
        const node = $createListNode(serializedNode.listType, serializedNode.start);
        node.setFormat(serializedNode.format);
        node.setIndent(serializedNode.indent);
        node.setDirection(serializedNode.direction);
        return node;
    }

    exportDOM(editor: LexicalEditor): DOMExportOutput {
        const { element } = super.exportDOM(editor);
        if (element && isHTMLElement(element)) {
            if (this.__start !== 1) {
                element.setAttribute("start", String(this.__start));
            }
            if (this.__listType === "check") {
                element.setAttribute("__lexicalListType", "check");
            }
        }
        return {
            element,
        };
    }

    exportJSON(): SerializedListNode {
        return {
            ...super.exportJSON(),
            listType: this.getListType(),
            start: this.getStart(),
            tag: this.getTag(),
            type: "list",
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

            if ($isListItemNode(currentNode)) {
                super.append(currentNode);
            } else {
                const listItemNode = $createListItemNode();

                if ($isListNode(currentNode)) {
                    listItemNode.append(currentNode);
                } else if ($isElementNode(currentNode)) {
                    const textNode = $createTextNode(currentNode.getTextContent());
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
        return $isListItemNode(child);
    }
}

/**
 * Takes the value of a child ListItemNode and makes it the value the ListItemNode
 * should be if it isn't already. Also ensures that checked is undefined if the
 * parent does not have a list type of 'check'.
 * @param list - The list whose children are updated.
 */
export const updateChildrenListItemValue = (list: ListNode): void => {
    const isNotChecklist = list.getListType() !== "check";
    let value = list.getStart();
    for (const child of list.getChildren()) {
        if ($isListItemNode(child)) {
            if (child.getValue() !== value) {
                child.setValue(value);
            }
            if (isNotChecklist && child.getChecked() != null) {
                child.setChecked(undefined);
            }
            if (!$isListNode(child.getFirstChild())) {
                value++;
            }
        }
    }
};

const setListThemeClassNames = (
    dom: HTMLElement,
    editorThemeClasses: EditorThemeClasses,
    node: ListNode,
): void => {
    const classesToAdd: string[] = [];
    const classesToRemove: string[] = [];
    const listTheme = editorThemeClasses.list;

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
};

/*
 * This function normalizes the children of a ListNode after the conversion from HTML,
 * ensuring that they are all ListItemNodes and contain either a single nested ListNode
 * or some other inline content.
 */
function normalizeChildren(nodes: LexicalNode[]): Array<ListItemNode> {
    const normalizedListItems: ListItemNode[] = [];
    for (let i = 0; i < nodes.length; i++) {
        const node = nodes[i];
        if ($isListItemNode(node)) {
            normalizedListItems.push(node);
            const children = node.getChildren();
            if (children.length > 1) {
                children.forEach((child) => {
                    if ($isListNode(child)) {
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
        // @ts-ignore
        const start = domNode.start;
        node = $createListNode("number", start);
    } else if (nodeName === "ul") {
        if (
            isHTMLElement(domNode) &&
            domNode.getAttribute("__lexicallisttype") === "check"
        ) {
            node = $createListNode("check");
        } else {
            node = $createListNode("bullet");
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

/**
 * Creates a ListNode of listType.
 * @param listType - The type of list to be created. Can be 'number', 'bullet', or 'check'.
 * @param start - Where an ordered list starts its count, start = 1 if left undefined.
 * @returns The new ListNode
 */
export const $createListNode = (listType: ListType, start = 1): ListNode => {
    return $applyNodeReplacement(new ListNode(listType, start));
};

/**
 * Checks to see if the node is a ListNode.
 * @param node - The node to be checked.
 * @returns true if the node is a ListNode, false otherwise.
 */
export const $isListNode = (
    node: LexicalNode | null | undefined,
): node is ListNode => {
    return node instanceof ListNode;
};

const createListOrMerge = (node: ElementNode, listType: ListType): ListNode => {
    if ($isListNode(node)) {
        return node;
    }

    const previousSibling = node.getPreviousSibling();
    const nextSibling = node.getNextSibling();
    const listItem = $createListItemNode();
    listItem.setFormat(node.getFormatType());
    listItem.setIndent(node.getIndent());
    append(listItem, node.getChildren());

    if (
        $isListNode(previousSibling) &&
        listType === previousSibling.getListType()
    ) {
        previousSibling.append(listItem);
        node.remove();
        // if the same type of list is on both sides, merge them.

        if ($isListNode(nextSibling) && listType === nextSibling.getListType()) {
            append(previousSibling, nextSibling.getChildren());
            nextSibling.remove();
        }
        return previousSibling;
    } else if (
        $isListNode(nextSibling) &&
        listType === nextSibling.getListType()
    ) {
        nextSibling.getFirstChildOrThrow().insertBefore(listItem);
        node.remove();
        return nextSibling;
    } else {
        const list = $createListNode(listType);
        list.append(listItem);
        node.replace(list);
        return list;
    }
};

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
export const insertList = (editor: LexicalEditor, listType: ListType): void => {
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
                const anchorNodeParent = anchorNode.getParent();

                if ($isSelectingEmptyListItem(anchorNode, nodes)) {
                    const list = $createListNode(listType);

                    if ($isRootOrShadowRoot(anchorNodeParent)) {
                        anchorNode.replace(list);
                        const listItem = $createListItemNode();
                        if ($isElementNode(anchorNode)) {
                            listItem.setFormat(anchorNode.getFormatType());
                            listItem.setIndent(anchorNode.getIndent());
                        }
                        list.append(listItem);
                    } else if ($isListItemNode(anchorNode)) {
                        const parent = anchorNode.getParentOrThrow();
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
                    $isElementNode(node) &&
                    node.isEmpty() &&
                    !$isListItemNode(node) &&
                    !handled.has(node.getKey())
                ) {
                    createListOrMerge(node, listType);
                    continue;
                }

                if ($isLeafNode(node)) {
                    let parent = node.getParent();
                    while (parent != null) {
                        const parentKey = parent.getKey();

                        if ($isListNode(parent)) {
                            if (!handled.has(parentKey)) {
                                const newListNode = $createListNode(listType);
                                append(newListNode, parent.getChildren());
                                parent.replace(newListNode);
                                handled.add(parentKey);
                            }

                            break;
                        } else {
                            const nextParent = parent.getParent();

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
};

/**
 * Attempts to insert a ParagraphNode at selection and selects the new node. The selection must contain a ListItemNode
 * or a node that does not already contain text. If its grandparent is the root/shadow root, it will get the ListNode
 * (which should be the parent node) and insert the ParagraphNode as a sibling to the ListNode. If the ListNode is
 * nested in a ListItemNode instead, it will add the ParagraphNode after the grandparent ListItemNode.
 * Throws an invariant if the selection is not a child of a ListNode.
 * @returns true if a ParagraphNode was inserted succesfully, false if there is no selection
 * or the selection does not contain a ListItemNode or the node already holds text.
 */
export const $handleListInsertParagraph = (): boolean => {
    const selection = $getSelection();

    if (!$isRangeSelection(selection) || !selection.isCollapsed()) {
        return false;
    }
    // Only run this code on empty list items
    const anchor = selection.anchor.getNode();

    if (!$isListItemNode(anchor) || anchor.getChildrenSize() !== 0) {
        return false;
    }
    const topListNode = $getTopListNode(anchor);
    const parent = anchor.getParent();

    if (!$isListNode(parent)) {
        throw new Error("A ListItemNode must have a ListNode for a parent.");
    }

    const grandparent = parent.getParent();

    let replacementNode;

    if ($isRootOrShadowRoot(grandparent)) {
        replacementNode = $createParagraphNode();
        topListNode.insertAfter(replacementNode);
    } else if ($isListItemNode(grandparent)) {
        replacementNode = $createListItemNode();
        grandparent.insertAfter(replacementNode);
    } else {
        return false;
    }
    replacementNode.select();

    const nextSiblings = anchor.getNextSiblings();

    if (nextSiblings.length > 0) {
        const newList = $createListNode(parent.getListType());

        if ($isParagraphNode(replacementNode)) {
            replacementNode.insertAfter(newList);
        } else {
            const newListItem = $createListItemNode();
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
};

/**
 * Searches for the nearest ancestral ListNode and removes it. If selection is an empty ListItemNode
 * it will remove the whole list, including the ListItemNode. For each ListItemNode in the ListNode,
 * removeList will also generate new ParagraphNodes in the removed ListNode's place. Any child node
 * inside a ListItemNode will be appended to the new ParagraphNodes.
 * @param editor - The lexical editor.
 */
export const removeList = (editor: LexicalEditor): void => {
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
                        const listItemNode = $getNearestNodeOfType(node, ListItemNode);

                        if (listItemNode != null) {
                            listNodes.add($getTopListNode(listItemNode));
                        }
                    }
                }
            }

            for (const listNode of listNodes) {
                let insertionPoint: ListNode | ParagraphNode = listNode;

                const listItems = $getAllListItems(listNode);

                for (const listItemNode of listItems) {
                    const paragraph = $createParagraphNode();

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
                        selection.anchor.set(paragraph.getKey(), 0, "element");
                    }
                    if (listItemNode.__key === selection.focus.key) {
                        selection.focus.set(paragraph.getKey(), 0, "element");
                    }

                    listItemNode.remove();
                }
                listNode.remove();
            }
        }
    });
};

const $isSelectingEmptyListItem = (
    anchorNode: ListItemNode | LexicalNode,
    nodes: Array<LexicalNode>,
): boolean => {
    return (
        $isListItemNode(anchorNode) &&
        (nodes.length === 0 ||
            (nodes.length === 1 &&
                anchorNode.is(nodes[0]) &&
                anchorNode.getChildrenSize() === 0))
    );
};

/**
 * A recursive function that goes through each list and their children, including nested lists,
 * appending list2 children after list1 children and updating ListItemNode values.
 * @param list1 - The first list to be merged.
 * @param list2 - The second list to be merged.
 */
export const mergeLists = (list1: ListNode, list2: ListNode): void => {
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
};

/**
 * Merge the next sibling list if same type.
 * <ul> will merge with <ul>, but NOT <ul> with <ol>.
 * @param list - The list whose next sibling should be potentially merged
 */
export const mergeNextSiblingListIfSameType = (list: ListNode): void => {
    const nextSibling = list.getNextSibling();
    if (
        $isListNode(nextSibling) &&
        list.getListType() === nextSibling.getListType()
    ) {
        mergeLists(list, nextSibling);
    }
};

const getIndent = (whitespaces: string): number => {
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
};

export const listExport = (
    listNode: ListNode,
    exportChildren: (node: ElementNode) => string,
    depth: number,
): string => {
    const output: string[] = [];
    const children = listNode.getChildren();
    let index = 0;
    for (const listItemNode of children) {
        if ($isListItemNode(listItemNode)) {
            if (listItemNode.getChildrenSize() === 1) {
                const firstChild = listItemNode.getFirstChild();
                if ($isListNode(firstChild)) {
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
};

export const listReplace = (listType: ListType): ElementTransformer["replace"] => {
    return (parentNode, children, match) => {
        const previousNode = parentNode.getPreviousSibling();
        const nextNode = parentNode.getNextSibling();
        const listItem = $createListItemNode(
            listType === "check" ? match[3] === "x" : undefined,
        );
        if ($isListNode(nextNode) && nextNode.getListType() === listType) {
            const firstChild = nextNode.getFirstChild();
            if (firstChild !== null) {
                firstChild.insertBefore(listItem);
            } else {
                // should never happen, but let's handle gracefully, just in case.
                nextNode.append(listItem);
            }
            parentNode.remove();
        } else if (
            $isListNode(previousNode) &&
            previousNode.getListType() === listType
        ) {
            previousNode.append(listItem);
            parentNode.remove();
        } else {
            const list = $createListNode(
                listType,
                listType === "number" ? Number(match[2]) : undefined,
            );
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
};
