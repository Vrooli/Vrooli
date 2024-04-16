import { DOM_TEXT_TYPE } from "../consts";
import { DOMConversionMap, DOMConversionOutput, SerializedLexicalNode, SerializedLineBreakNode } from "../types";
import { $applyNodeReplacement } from "../utils";
import { LexicalNode } from "./LexicalNode";


export class LineBreakNode extends LexicalNode {
    static getType(): string {
        return "linebreak";
    }

    static clone(node: LineBreakNode): LineBreakNode {
        return new LineBreakNode(node.__key);
    }

    getTextContent(): "\n" {
        return "\n";
    }

    createDOM(): HTMLElement {
        return document.createElement("br");
    }

    updateDOM(): false {
        return false;
    }

    static importDOM(): DOMConversionMap | null {
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
        return $createLineBreakNode();
    }

    exportJSON(): SerializedLexicalNode {
        return {
            type: "linebreak",
            version: 1,
        };
    }
}

const convertLineBreakElement = (node: Node): DOMConversionOutput => {
    return { node: $createLineBreakNode() };
};

export const $createLineBreakNode = (): LineBreakNode => {
    return $applyNodeReplacement(new LineBreakNode());
};

export const $isLineBreakNode = (
    node: LexicalNode | null | undefined,
): node is LineBreakNode => {
    return node instanceof LineBreakNode;
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
