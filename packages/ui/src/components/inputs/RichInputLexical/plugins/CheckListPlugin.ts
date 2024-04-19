/* eslint-disable @typescript-eslint/ban-ts-comment */
import { useEffect } from "react";
import { INSERT_CHECK_LIST_COMMAND, KEY_ARROW_DOWN_COMMAND, KEY_ARROW_LEFT_COMMAND, KEY_ARROW_UP_COMMAND, KEY_ESCAPE_COMMAND, KEY_SPACE_COMMAND } from "../commands";
import { COMMAND_PRIORITY_LOW } from "../consts";
import { useLexicalComposerContext } from "../context";
import { LexicalEditor } from "../editor";
import { ListItemNode } from "../nodes/ListItemNode";
import { insertList } from "../nodes/ListNode";
import { $findMatchingParent, $getNearestNodeFromDOMNode, $getSelection, $isNode, $isRangeSelection, calculateZoomLevel, getNextSibling, getParent, getParentOrThrow, getPreviousSibling, isHTMLElement, mergeRegister } from "../utils";

export const CheckListPlugin = (): null => {
    const editor = useLexicalComposerContext();

    useEffect(() => {
        return mergeRegister(
            editor.registerCommand(
                INSERT_CHECK_LIST_COMMAND,
                () => {
                    insertList(editor, "check");
                    return true;
                },
                COMMAND_PRIORITY_LOW,
            ),
            editor.registerCommand<KeyboardEvent>(
                KEY_ARROW_DOWN_COMMAND,
                (event) => {
                    return handleArrownUpOrDown(event, editor, false);
                },
                COMMAND_PRIORITY_LOW,
            ),
            editor.registerCommand<KeyboardEvent>(
                KEY_ARROW_UP_COMMAND,
                (event) => {
                    return handleArrownUpOrDown(event, editor, true);
                },
                COMMAND_PRIORITY_LOW,
            ),
            editor.registerCommand<KeyboardEvent>(
                KEY_ESCAPE_COMMAND,
                (event) => {
                    const activeItem = getActiveCheckListItem();

                    if (activeItem != null) {
                        const rootElement = editor.getRootElement();

                        if (rootElement != null) {
                            rootElement.focus();
                        }

                        return true;
                    }

                    return false;
                },
                COMMAND_PRIORITY_LOW,
            ),
            editor.registerCommand<KeyboardEvent>(
                KEY_SPACE_COMMAND,
                (event) => {
                    const activeItem = getActiveCheckListItem();

                    if (activeItem != null && editor.isEditable()) {
                        editor.update(() => {
                            const listItemNode = $getNearestNodeFromDOMNode(activeItem);

                            if ($isNode("ListItem", listItemNode)) {
                                event.preventDefault();
                                listItemNode!.toggleChecked();
                            }
                        });
                        return true;
                    }

                    return false;
                },
                COMMAND_PRIORITY_LOW,
            ),
            editor.registerCommand<KeyboardEvent>(
                KEY_ARROW_LEFT_COMMAND,
                (event) => {
                    return editor.getEditorState().read(() => {
                        const selection = $getSelection();

                        if ($isRangeSelection(selection) && selection.isCollapsed()) {
                            const { anchor } = selection;
                            const isElement = anchor.type === "element";

                            if (isElement || anchor.offset === 0) {
                                const anchorNode = anchor.getNode();
                                const elementNode = $findMatchingParent(
                                    anchorNode,
                                    (node) => $isNode("Element", node) && !node.isInline(),
                                );
                                if ($isNode("ListItem", elementNode)) {
                                    const parent = getParent(elementNode);
                                    if (
                                        $isNode("List", parent) &&
                                        parent!.getListType() === "check" &&
                                        (isElement ||
                                            elementNode.getFirstDescendant() === anchorNode)
                                    ) {
                                        const domNode = editor.getElementByKey(elementNode.__key);

                                        if (domNode != null && document.activeElement !== domNode) {
                                            domNode.focus();
                                            event.preventDefault();
                                            return true;
                                        }
                                    }
                                }
                            }
                        }

                        return false;
                    });
                },
                COMMAND_PRIORITY_LOW,
            ),
            editor.registerRootListener((rootElement, prevElement) => {
                if (rootElement !== null) {
                    rootElement.addEventListener("click", handleClick);
                    rootElement.addEventListener("pointerdown", handlePointerDown);
                }

                if (prevElement !== null) {
                    prevElement.removeEventListener("click", handleClick);
                    prevElement.removeEventListener("pointerdown", handlePointerDown);
                }
            }),
        );
    });

    return null;
};

const handleCheckItemEvent = (event: PointerEvent, callback: () => void) => {
    const target = event.target;

    if (target === null || !isHTMLElement(target)) {
        return;
    }

    // Ignore clicks on LI that have nested lists
    const firstChild = target.firstChild;

    if (
        firstChild != null &&
        isHTMLElement(firstChild) &&
        (firstChild.tagName === "UL" || firstChild.tagName === "OL")
    ) {
        return;
    }

    const parentNode = target.parentNode;

    // @ts-ignore internal field
    if (!parentNode || parentNode.__lexicalListType !== "check") {
        return;
    }

    const rect = target.getBoundingClientRect();
    const pageX = event.pageX / calculateZoomLevel(target);
    if (
        target.dir === "rtl"
            ? pageX < rect.right && pageX > rect.right - 20
            : pageX > rect.left && pageX < rect.left + 20
    ) {
        callback();
    }
};

const handleClick = (event: Event) => {
    handleCheckItemEvent(event as PointerEvent, () => {
        const domNode = event.target as HTMLElement;
        const editor = findEditor(domNode);

        if (editor != null && editor.isEditable()) {
            editor.update(() => {
                if (event.target) {
                    const node = $getNearestNodeFromDOMNode(domNode);

                    if ($isNode("ListItem", node)) {
                        domNode.focus();
                        node.toggleChecked();
                    }
                }
            });
        }
    });
};

const handlePointerDown = (event: PointerEvent) => {
    handleCheckItemEvent(event, () => {
        // Prevents caret moving when clicking on check mark
        event.preventDefault();
    });
};

const findEditor = (target: Node) => {
    let node: ParentNode | Node | null = target;

    while (node) {
        // @ts-ignore internal field
        if (node.__lexicalEditor) {
            // @ts-ignore internal field
            return node.__lexicalEditor;
        }

        node = node.parentNode;
    }

    return null;
};

const getActiveCheckListItem = (): HTMLElement | null => {
    const activeElement = document.activeElement as HTMLElement;

    return activeElement != null &&
        activeElement.tagName === "LI" &&
        activeElement.parentNode != null &&
        // @ts-ignore internal field
        activeElement.parentNode.__lexicalListType === "check"
        ? activeElement
        : null;
};

const findCheckListItemSibling = (
    node: ListItemNode,
    backward: boolean,
): ListItemNode | null => {
    let sibling = backward ? getPreviousSibling(node) : getNextSibling(node);
    let parent: ListItemNode | null = node;

    // Going up in a tree to get non-null sibling
    while (sibling == null && $isNode("ListItem", parent)) {
        // Get li -> parent ul/ol -> parent li
        parent = getParent(getParentOrThrow(parent));

        if (parent != null) {
            sibling = backward
                ? getPreviousSibling(parent)
                : getNextSibling(parent);
        }
    }

    // Going down in a tree to get first non-nested list item
    while ($isNode("ListItem", sibling)) {
        const firstChild = backward
            ? sibling.getLastChild()
            : sibling.getFirstChild();

        if (!$isNode("List", firstChild)) {
            return sibling;
        }

        sibling = backward ? firstChild.getLastChild() : firstChild.getFirstChild();
    }

    return null;
};

const handleArrownUpOrDown = (
    event: KeyboardEvent,
    editor: LexicalEditor,
    backward: boolean,
) => {
    const activeItem = getActiveCheckListItem();

    if (activeItem != null) {
        editor.update(() => {
            const listItem = $getNearestNodeFromDOMNode(activeItem);

            if (!$isNode("ListItem", listItem)) {
                return;
            }

            const nextListItem = findCheckListItemSibling(listItem, backward);

            if (nextListItem != null) {
                nextListItem.selectStart();
                const dom = editor.getElementByKey(nextListItem.__key);

                if (dom != null) {
                    event.preventDefault();
                    setTimeout(() => {
                        dom.focus();
                    }, 0);
                }
            }
        });
    }

    return false;
};
