import { DOM_TEXT_TYPE } from "../consts";
import { DOMConversionMap, DOMConversionOutput, NodeConstructorPayloads, NodeType, SerializedLexicalNode } from "../types";
import { $createNode } from "../utils";
import { LexicalNode } from "./LexicalNode";

/**
 * Used when a line is empty
 */
export class LineBreakNode extends LexicalNode {
    static __type: NodeType = "LineBreak";

    constructor({ key }: NodeConstructorPayloads["LineBreak"]) {
        super(key);
    }

    static clone(node: LineBreakNode): LineBreakNode {
        return new LineBreakNode({ key: node.__key });
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

    static importJSON(): LineBreakNode {
        return $createNode("LineBreak", {});
    }

    exportJSON(): SerializedLexicalNode {
        return {
            __type: "LineBreak",
            version: 1,
        };
    }
}

const convertLineBreakElement = (): DOMConversionOutput => {
    return { node: $createNode("LineBreak", {}) };
};

const isOnlyChild = (node: Node): boolean => {
    const parentElement = node.parentElement;
    if (!parentElement) return false;
    const firstChild = parentElement.firstChild;
    if (
        firstChild &&
        (firstChild === node ||
            (firstChild.nextSibling === node && isWhitespaceDomTextNode(firstChild)))
    ) {
        const lastChild = parentElement.lastChild;
        if (
            lastChild &&
            (lastChild === node ||
                (lastChild.previousSibling === node &&
                    isWhitespaceDomTextNode(lastChild)))
        ) {
            return true;
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
