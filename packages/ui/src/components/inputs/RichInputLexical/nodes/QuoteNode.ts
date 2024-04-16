import { LexicalEditor } from "../editor";
import { RangeSelection } from "../selection";
import { DOMConversionMap, DOMConversionOutput, DOMExportOutput, EditorConfig, ElementFormatType, SerializedElementNode, SerializedQuoteNode } from "../types";
import { $applyNodeReplacement, addClassNamesToElement, isHTMLElement } from "../utils";
import { ElementNode } from "./ElementNode";
import { LexicalNode } from "./LexicalNode";
import { $createParagraphNode, ParagraphNode } from "./ParagraphNode";

export class QuoteNode extends ElementNode {
    static getType(): string {
        return "quote";
    }

    static clone(node: QuoteNode): QuoteNode {
        return new QuoteNode(node.__key);
    }

    // View

    createDOM(config: EditorConfig): HTMLElement {
        const element = document.createElement("blockquote");
        addClassNamesToElement(element, config.theme.quote);
        return element;
    }
    updateDOM(prevNode: QuoteNode, dom: HTMLElement): boolean {
        return false;
    }

    static importDOM(): DOMConversionMap | null {
        return {
            blockquote: (node: Node) => ({
                conversion: convertBlockquoteElement,
                priority: 0,
            }),
        };
    }

    exportDOM(editor: LexicalEditor): DOMExportOutput {
        const { element } = super.exportDOM(editor);

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
        const node = $createQuoteNode();
        node.setFormat(serializedNode.format);
        node.setIndent(serializedNode.indent);
        node.setDirection(serializedNode.direction);
        return node;
    }

    exportJSON(): SerializedElementNode {
        return {
            ...super.exportJSON(),
            type: "quote",
        };
    }

    // Mutation

    insertNewAfter(_: RangeSelection, restoreSelection?: boolean): ParagraphNode {
        const newBlock = $createParagraphNode();
        const direction = this.getDirection();
        newBlock.setDirection(direction);
        this.insertAfter(newBlock, restoreSelection);
        return newBlock;
    }

    collapseAtStart(): true {
        const paragraph = $createParagraphNode();
        const children = this.getChildren();
        children.forEach((child) => paragraph.append(child));
        this.replace(paragraph);
        return true;
    }
}

export const $createQuoteNode = (): QuoteNode => {
    return $applyNodeReplacement(new QuoteNode());
};

export const $isQuoteNode = (
    node: LexicalNode | null | undefined,
): node is QuoteNode => {
    return node instanceof QuoteNode;
};

const convertBlockquoteElement = (element: HTMLElement): DOMConversionOutput => {
    const node = $createQuoteNode();
    if (element.style !== null) {
        node.setFormat(element.style.textAlign as ElementFormatType);
    }
    return { node };
};
