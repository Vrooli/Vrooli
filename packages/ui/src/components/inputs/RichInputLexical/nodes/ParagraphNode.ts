import { TEXT_FLAGS } from "../consts";
import { RangeSelection } from "../selection";
import { DOMConversionMap, DOMConversionOutput, DOMExportOutput, EditorConfig, ElementFormatType, NodeConstructorPayloads, NodeType, SerializedParagraphNode, TextFormatType } from "../types";
import { $createNode, $isNode, getNextSibling, getPreviousSibling, isHTMLElement } from "../utils";
import { ElementNode } from "./ElementNode";

export class ParagraphNode extends ElementNode {
    static __type: NodeType = "Paragraph";
    __textFormat: number;

    constructor({ ...rest }: NodeConstructorPayloads["Paragraph"]) {
        super(rest);
        this.__textFormat = 0;
    }

    getTextFormat(): number {
        const self = this.getLatest();
        return self.__textFormat;
    }

    setTextFormat(type: number): this {
        const self = this.getWritable();
        self.__textFormat = type;
        return self;
    }

    hasTextFormat(type: TextFormatType): boolean {
        return (this.getTextFormat() & TEXT_FLAGS[type]) !== 0;
    }

    static clone(node: ParagraphNode): ParagraphNode {
        const { __key } = node;
        return $createNode("Paragraph", { key: __key });
    }

    // View

    createDOM(): HTMLElement {
        const dom = document.createElement("p");
        dom.classList.add("RichInput__paragraph");
        return dom;
    }
    updateDOM(
        prevNode: ParagraphNode,
        dom: HTMLElement,
        config: EditorConfig,
    ): boolean {
        return false;
    }

    static importDOM(): DOMConversionMap {
        return {
            p: (node: Node) => ({
                conversion: convertParagraphElement,
                priority: 0,
            }),
        };
    }

    exportDOM(): DOMExportOutput {
        const { element } = super.exportDOM();

        if (element && isHTMLElement(element)) {
            if (this.isEmpty()) {
                element.append(document.createElement("br"));
            }

            const formatType = this.getFormatType();
            element.style.textAlign = formatType;

            const direction = this.getDirection();
            if (direction) {
                element.dir = direction;
            }
            const indent = this.getIndent();
            if (indent > 0) {
                // padding-inline-start is not widely supported in email HTML, but
                // Lexical Reconciler uses padding-inline-start. Using text-indent instead.
                element.style.textIndent = `${indent * 20}px`;
            }
        }

        return {
            element,
        };
    }

    static importJSON(serializedNode: SerializedParagraphNode): ParagraphNode {
        const node = $createNode("Paragraph", {});
        node.setFormat(serializedNode.format);
        node.setIndent(serializedNode.indent);
        node.setDirection(serializedNode.direction);
        node.setTextFormat(serializedNode.textFormat);
        return node;
    }

    exportJSON(): SerializedParagraphNode {
        return {
            ...super.exportJSON(),
            __type: "Paragraph",
            textFormat: this.getTextFormat(),
            version: 1,
        };
    }

    // Mutation

    insertNewAfter(
        rangeSelection: RangeSelection,
        restoreSelection: boolean,
    ): ParagraphNode {
        const newElement = $createNode("Paragraph", {});
        newElement.setTextFormat(rangeSelection.format);
        const direction = this.getDirection();
        newElement.setDirection(direction);
        newElement.setFormat(this.getFormatType());
        this.insertAfter(newElement, restoreSelection);
        return newElement;
    }

    collapseAtStart(): boolean {
        const children = this.getChildren();
        // If we have an empty (trimmed) first paragraph and try and remove it,
        // delete the paragraph as long as we have another sibling to go to
        if (
            children.length === 0 ||
            ($isNode("Text", children[0]) && children[0].getTextContent().trim() === "")
        ) {
            const nextSibling = getNextSibling(this);
            if (nextSibling !== null) {
                this.selectNext();
                this.remove();
                return true;
            }
            const prevSibling = getPreviousSibling(this);
            if (prevSibling !== null) {
                this.selectPrevious();
                this.remove();
                return true;
            }
        }
        return false;
    }
}

const convertParagraphElement = (element: HTMLElement): DOMConversionOutput => {
    const node = $createNode("Paragraph", {});
    if (element.style) {
        node.setFormat(element.style.textAlign as ElementFormatType);
        const indent = parseInt(element.style.textIndent, 10) / 20;
        if (indent > 0) {
            node.setIndent(indent);
        }
    }
    return { node };
};
