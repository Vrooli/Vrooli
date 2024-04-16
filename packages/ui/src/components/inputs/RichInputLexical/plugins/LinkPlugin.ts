import { useEffect } from "react";
import { PASTE_COMMAND, TOGGLE_LINK_COMMAND } from "../commands";
import { COMMAND_PRIORITY_LOW } from "../consts";
import { useLexicalComposerContext } from "../context";
import { ElementNode } from "../nodes/ElementNode";
import { $createLinkNode, $isLinkNode, LinkNode } from "../nodes/LinkNode";
import { LinkAttributes } from "../types";
import { $getAncestor, $getSelection, $isElementNode, $isRangeSelection, mergeRegister, objectKlassEquals } from "../utils";

type Props = {
    validateUrl?: (url: string) => boolean;
};

/**
 * Generates or updates a LinkNode. It can also delete a LinkNode if the URL is null,
 * but saves any children and brings them up to the parent node.
 * @param url - The URL the link directs to.
 * @param attributes - Optional HTML a tag attributes. { target, rel, title }
 */
export const toggleLink = (
    url: null | string,
    attributes: LinkAttributes = {},
) => {
    const { target, title } = attributes;
    const rel = attributes.rel === undefined ? "noreferrer" : attributes.rel;
    const selection = $getSelection();

    if (!$isRangeSelection(selection)) {
        return;
    }
    const nodes = selection.extract();

    if (url === null) {
        // Remove LinkNodes
        nodes.forEach((node) => {
            const parent = node.getParent();

            if ($isLinkNode(parent)) {
                const children = parent.getChildren();

                for (let i = 0; i < children.length; i++) {
                    parent.insertBefore(children[i]);
                }

                parent.remove();
            }
        });
    } else {
        // Add or merge LinkNodes
        if (nodes.length === 1) {
            const firstNode = nodes[0];
            // if the first node is a LinkNode or if its
            // parent is a LinkNode, we update the URL, target and rel.
            const linkNode = $getAncestor(firstNode, $isLinkNode);
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
            const parent = node.getParent();

            if (
                parent === linkNode ||
                parent === null ||
                ($isElementNode(node) && !node.isInline())
            ) {
                return;
            }

            if ($isLinkNode(parent)) {
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

            if (!parent.is(prevParent)) {
                prevParent = parent;
                linkNode = $createLinkNode(url, { rel, target, title });

                if ($isLinkNode(parent)) {
                    if (node.getPreviousSibling() === null) {
                        parent.insertBefore(linkNode);
                    } else {
                        parent.insertAfter(linkNode);
                    }
                } else {
                    node.insertBefore(linkNode);
                }
            }

            if ($isLinkNode(node)) {
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
};

export const LinkPlugin = ({ validateUrl }: Props) => {
    const [editor] = useLexicalComposerContext();

    useEffect(() => {
        if (!editor.hasNodes([LinkNode])) {
            throw new Error("LinkPlugin: LinkNode not registered on editor");
        }
        return mergeRegister(
            editor.registerCommand(
                TOGGLE_LINK_COMMAND,
                (payload) => {
                    if (payload === null) {
                        toggleLink(payload);
                        return true;
                    } else if (typeof payload === "string") {
                        if (validateUrl === undefined || validateUrl(payload)) {
                            toggleLink(payload);
                            return true;
                        }
                        return false;
                    } else {
                        const { url, target, rel, title } = payload;
                        toggleLink(url, { rel, target, title });
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
                        if (!selection.getNodes().some((node) => $isElementNode(node))) {
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
};
