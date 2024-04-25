import { DOM_TEXT_TYPE } from "../consts";
import { DOMConversionMap, DOMConversionOutput, NodeType, SerializedLexicalNode, SerializedLineBreakNode } from "../types";
import { $createNode } from "../utils";
import { LexicalNode } from "./LexicalNode";


export class LineBreakNode extends LexicalNode {
    static __type: NodeType = "LineBreak";

    static clone(node: LineBreakNode): LineBreakNode {
        return new LineBreakNode(node.__key);
    }

    getMarkdownContent() {
        return "\n";
    }

    getTextContent() {
        return "\n";
    }

    getTextContentSize() {
        return 1;
    }

    createDOM() {
        return document.createElement("br");
    }

    updateDOM() {
        return false;
    }

    static importDOM(): DOMConversionMap {
        return {
            br: (node: Node) => {
                if (isOnlyChild(node)) {
                    return null;
                }
                return {
                    conversion: convertLineBreakElement,
                    priority: 0,
                };
            },
        };
    }

    static importJSON(
        serializedLineBreakNode: SerializedLineBreakNode,
    ): LineBreakNode {
        return $createNode("LineBreak", {});
    }

    exportJSON(): SerializedLexicalNode {
        return {
            __type: "LineBreak",
            version: 1,
        };
    }
}

const convertLineBreakElement = (node: Node): DOMConversionOutput => {
    return { node: $createNode("LineBreak", {}) };
};

const isOnlyChild = (node: Node): boolean => {
    const parentElement = node.parentElement;
    if (parentElement !== null) {
        const firstChild = parentElement.firstChild;
        if (
            firstChild !== null &&
            (firstChild === node ||
                (firstChild.nextSibling === node && isWhitespaceDomTextNode(firstChild)))
        ) {
            const lastChild = parentElement.lastChild;
            if (
                lastChild !== null &&
                (lastChild === node ||
                    (lastChild.previousSibling === node &&
                        isWhitespaceDomTextNode(lastChild)))
            ) {
                return true;
            }
        }
    }
    return false;
};

const isWhitespaceDomTextNode = (node: Node): boolean => {
    return (
        node.nodeType === DOM_TEXT_TYPE &&
        /^( |\t|\r?\n)+$/.test(node.textContent || "")
    );
};
