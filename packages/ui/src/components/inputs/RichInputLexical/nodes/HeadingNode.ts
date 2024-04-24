import { RangeSelection } from "../selection";
import { DOMConversionMap, DOMConversionOutput, DOMExportOutput, ElementFormatType, HeadingTagType, NodeConstructorPayloads, NodeType, SerializedHeadingNode } from "../types";
import { $createNode, isHTMLElement } from "../utils";
import { ElementNode } from "./ElementNode";
import { type ParagraphNode } from "./ParagraphNode";

export class HeadingNode extends ElementNode {
    static __type: NodeType = "Heading";
    __tag: HeadingTagType;

    constructor({ tag, ...rest }: NodeConstructorPayloads["Heading"]) {
        super(rest);
        this.__tag = tag;
    }

    static clone(node: HeadingNode): HeadingNode {
        const { __tag, __key } = node;
        return $createNode("Heading", { tag: __tag, key: __key });
    }

    getTag(): HeadingTagType {
        return this.__tag;
    }

    // View

    createDOM(): HTMLElement {
        const tag = this.__tag;
        const element = document.createElement(tag);
        return element;
    }

    updateDOM(prevNode: HeadingNode, dom: HTMLElement): boolean {
        return false;
    }

    static importDOM(): DOMConversionMap {
        return {
            h1: (node: Node) => ({
                conversion: convertHeadingElement,
                priority: 0,
            }),
            h2: (node: Node) => ({
                conversion: convertHeadingElement,
                priority: 0,
            }),
            h3: (node: Node) => ({
                conversion: convertHeadingElement,
                priority: 0,
            }),
            h4: (node: Node) => ({
                conversion: convertHeadingElement,
                priority: 0,
            }),
            h5: (node: Node) => ({
                conversion: convertHeadingElement,
                priority: 0,
            }),
            h6: (node: Node) => ({
                conversion: convertHeadingElement,
                priority: 0,
            }),
            p: (node: Node) => {
                return null;
            },
            span: (node: Node) => {
                return null;
            },
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
        }

        return {
            element,
        };
    }

    static importJSON({ direction, format, indent, tag }: SerializedHeadingNode): HeadingNode {
        const node = $createNode("Heading", { tag });
        node.setFormat(format);
        node.setIndent(indent);
        node.setDirection(direction);
        return node;
    }

    exportJSON(): SerializedHeadingNode {
        return {
            ...super.exportJSON(),
            __type: "Heading",
            tag: this.getTag(),
            version: 1,
        };
    }

    // Mutation
    insertNewAfter(
        selection?: RangeSelection,
        restoreSelection = true,
    ): ParagraphNode | HeadingNode {
        const anchorOffet = selection ? selection.anchor.offset : 0;
        const newElement =
            anchorOffet === this.getTextContentSize() || !selection
                ? $createNode("Paragraph", {})
                : $createNode("Heading", { tag: this.getTag() });
        const direction = this.getDirection();
        newElement.setDirection(direction);
        this.insertAfter(newElement, restoreSelection);
        if (anchorOffet === 0 && !this.isEmpty() && selection) {
            const paragraph = $createNode("Paragraph", {});
            paragraph.select();
            this.replace(paragraph, true);
        }
        return newElement;
    }

    collapseAtStart(): true {
        const newElement = !this.isEmpty()
            ? $createNode("Heading", { tag: this.getTag() })
            : $createNode("Paragraph", {});
        const children = this.getChildren();
        children.forEach((child) => newElement.append(child));
        this.replace(newElement);
        return true;
    }

    extractWithChild(): boolean {
        return true;
    }
}

const convertHeadingElement = (element: HTMLElement): DOMConversionOutput => {
    const nodeName = element.nodeName.toLowerCase();
    let node: HeadingNode | null = null;
    if (
        nodeName === "h1" ||
        nodeName === "h2" ||
        nodeName === "h3" ||
        nodeName === "h4" ||
        nodeName === "h5" ||
        nodeName === "h6"
    ) {
        node = $createNode("Heading", { tag: nodeName });
        if (element.style !== null) {
            node.setFormat(element.style.textAlign as ElementFormatType);
        }
    }
    return { node };
};
