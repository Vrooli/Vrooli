import { useEffect } from "react";
import { PASTE_COMMAND, TOGGLE_LINK_COMMAND } from "../commands.js";
import { COMMAND_PRIORITY_LOW } from "../consts.js";
import { useLexicalComposerContext } from "../context.js";
import { ElementNode } from "../nodes/ElementNode.js";
import { type LinkNode } from "../nodes/LinkNode.js";
import { LinkAttributes } from "../types.js";
import { $createNode, $getAncestor, $getSelection, $isNode, $isRangeSelection, getParent, getPreviousSibling, mergeRegister, objectKlassEquals } from "../utils.js";

type Props = {
    validateUrl?: (url: string) => boolean;
};

/**
 * Generates or updates a LinkNode.
 */
export function addLink(params: LinkAttributes): void {
    const { target, title, url } = params;
    const rel = params.rel === undefined ? "noreferrer" : params.rel;
    const selection = $getSelection();

    if (!$isRangeSelection(selection)) {
        return;
    }
    const nodes = selection.extract();

    // Add or merge LinkNodes
    if (nodes.length === 1) {
        const firstNode = nodes[0];
        // if the first node is a LinkNode or if its
        // parent is a LinkNode, we update the URL, target and rel.
        const linkNode = $getAncestor(firstNode, (node): node is LinkNode => $isNode("Link", node));
        if (linkNode !== null) {
            linkNode.setURL(url);
            if (target !== undefined) {
                linkNode.setTarget(target);
            }
            if (rel !== null) {
                linkNode.setRel(rel);
            }
            if (title !== undefined) {
                linkNode.setTitle(title);
            }
            return;
        }
    }

    let prevParent: ElementNode | LinkNode | null = null;
    let linkNode: LinkNode | null = null;

    nodes.forEach((node) => {
        const parent = getParent(node);

        if (
            parent === linkNode ||
            parent === null ||
            ($isNode("Element", node) && !node.isInline())
        ) {
            return;
        }

        if ($isNode("Link", parent)) {
            linkNode = parent;
            parent.setURL(url);
            if (target !== undefined) {
                parent.setTarget(target);
            }
            if (rel !== null) {
                linkNode.setRel(rel);
            }
            if (title !== undefined) {
                linkNode.setTitle(title);
            }
            return;
        }

        if (parent.__key !== prevParent?.__key) {
            prevParent = parent;
            linkNode = $createNode("Link", { rel, target, title, url });

            if ($isNode("Link", parent)) {
                if (getPreviousSibling(node) === null) {
                    parent.insertBefore(linkNode);
                } else {
                    parent.insertAfter(linkNode);
                }
            } else {
                node.insertBefore(linkNode);
            }
        }

        if ($isNode("Link", node)) {
            if (node.is(linkNode)) {
                return;
            }
            if (linkNode !== null) {
                const children = node.getChildren();

                for (let i = 0; i < children.length; i++) {
                    linkNode.append(children[i]);
                }
            }

            node.remove();
            return;
        }

        if (linkNode !== null) {
            linkNode.append(node);
        }
    });
}

/**
 * Delete a LinkNode, and saves any children and brings them up to the parent node.
 */
export function removeLink(): void {
    const selection = $getSelection();

    if (!$isRangeSelection(selection)) {
        return;
    }
    const nodes = selection.extract();

    // Remove LinkNodes
    nodes.forEach((node) => {
        const parent = getParent(node);

        if ($isNode("Link", parent)) {
            const children = parent.getChildren();

            for (let i = 0; i < children.length; i++) {
                parent.insertBefore(children[i]);
            }

            parent.remove();
        }
    });
}

export function LinkPlugin({ validateUrl }: Props) {
    const editor = useLexicalComposerContext();

    useEffect(() => {
        if (!editor) return;
        return mergeRegister(
            editor.registerCommand(
                TOGGLE_LINK_COMMAND,
                (payload) => {
                    if (payload === null) {
                        removeLink();
                        return true;
                    } else if (typeof payload === "string") {
                        if (validateUrl === undefined || validateUrl(payload)) {
                            addLink({ url: payload });
                            return true;
                        }
                        return false;
                    } else {
                        const { url, target, rel, title } = payload;
                        addLink({ rel, target, title, url });
                        return true;
                    }
                },
                COMMAND_PRIORITY_LOW,
            ),
            validateUrl !== undefined
                ? editor.registerCommand(
                    PASTE_COMMAND,
                    (event) => {
                        const selection = $getSelection();
                        if (
                            !$isRangeSelection(selection) ||
                            selection.isCollapsed() ||
                            !objectKlassEquals(event, ClipboardEvent)
                        ) {
                            return false;
                        }
                        const clipboardEvent = event as ClipboardEvent;
                        if (clipboardEvent.clipboardData === null) {
                            return false;
                        }
                        const clipboardText =
                            clipboardEvent.clipboardData.getData("text");
                        if (!validateUrl(clipboardText)) {
                            return false;
                        }
                        // If we select nodes that are elements then avoid applying the link.
                        if (!selection.getNodes().some((node) => $isNode("Element", node))) {
                            editor.dispatchCommand(TOGGLE_LINK_COMMAND, clipboardText);
                            event.preventDefault();
                            return true;
                        }
                        return false;
                    },
                    COMMAND_PRIORITY_LOW,
                )
                : () => {
                    // Don't paste arbritrary text as a link when there's no validate function
                },
        );
    }, [editor, validateUrl]);

    return null;
}
