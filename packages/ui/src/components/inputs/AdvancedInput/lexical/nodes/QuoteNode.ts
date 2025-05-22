import { type RangeSelection } from "../selection.js";
import { type DOMConversionMap, type DOMConversionOutput, type DOMExportOutput, type ElementFormatType, type NodeType, type SerializedElementNode, type SerializedQuoteNode } from "../types.js";
import { $createNode, isHTMLElement } from "../utils.js";
import { ElementNode } from "./ElementNode.js";
import { type ParagraphNode } from "./ParagraphNode.js";

export class QuoteNode extends ElementNode {
    static __type: NodeType = "Quote";

    static clone(node: QuoteNode): QuoteNode {
        const { __key } = node;
        return new QuoteNode({ key: __key });
    }

    // View

    createDOM(): HTMLElement {
        const element = document.createElement("blockquote");
        return element;
    }
    updateDOM(): boolean {
        return false;
    }

    static importDOM(): DOMConversionMap {
        return {
            blockquote: (node: Node) => ({
                conversion: convertBlockquoteElement,
                priority: 0,
            }),
        };
    }

    exportDOM(): DOMExportOutput {
        const { element } = super.exportDOM();

        if (element && isHTMLElement(element)) {
            if (this.isEmpty()) {
                (element as HTMLElement).append(document.createElement("br"));
            }

            const formatType = this.getFormatType();
            element.style.textAlign = formatType;

            const direction = this.getDirection();
            if (direction) {
                element.dir = direction;
            }
        }

        return {
            element,
        };
    }

    static importJSON(serializedNode: SerializedQuoteNode): QuoteNode {
        const node = $createNode("Quote", {});
        node.setFormat(serializedNode.format);
        node.setIndent(serializedNode.indent);
        node.setDirection(serializedNode.direction);
        return node;
    }

    exportJSON(): SerializedElementNode {
        return {
            ...super.exportJSON(),
            __type: "Quote",
        };
    }

    // Mutation

    insertNewAfter(_: RangeSelection, restoreSelection?: boolean): ParagraphNode {
        const newBlock = $createNode("Paragraph", {});
        const direction = this.getDirection();
        newBlock.setDirection(direction);
        this.insertAfter(newBlock, restoreSelection);
        return newBlock;
    }

    collapseAtStart(): true {
        const paragraph = $createNode("Paragraph", {});
        const children = this.getChildren();
        children.forEach((child) => paragraph.append(child));
        this.replace(paragraph);
        return true;
    }
}

function convertBlockquoteElement(element: HTMLElement): DOMConversionOutput {
    const node = $createNode("Quote", {});
    if (element.style !== null) {
        node.setFormat(element.style.textAlign as ElementFormatType);
    }
    return { node };
}
