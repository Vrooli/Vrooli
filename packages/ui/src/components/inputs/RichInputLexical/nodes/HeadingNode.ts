import { LexicalEditor } from "../editor";
import { RangeSelection } from "../selection";
import { DOMConversionMap, DOMConversionOutput, DOMExportOutput, EditorConfig, ElementFormatType, HeadingTagType, NodeKey, SerializedHeadingNode } from "../types";
import { $applyNodeReplacement, addClassNamesToElement, isHTMLElement } from "../utils";
import { ElementNode } from "./ElementNode";
import { LexicalNode } from "./LexicalNode";
import { $createParagraphNode, ParagraphNode } from "./ParagraphNode";

export class HeadingNode extends ElementNode {
    /** @internal */
    __tag: HeadingTagType;

    static getType(): string {
        return "heading";
    }

    static clone(node: HeadingNode): HeadingNode {
        return new HeadingNode(node.__tag, node.__key);
    }

    constructor(tag: HeadingTagType, key?: NodeKey) {
        super(key);
        this.__tag = tag;
    }

    getTag(): HeadingTagType {
        return this.__tag;
    }

    // View

    createDOM(config: EditorConfig): HTMLElement {
        const tag = this.__tag;
        const element = document.createElement(tag);
        const theme = config.theme;
        const classNames = theme.heading;
        if (classNames !== undefined) {
            const className = classNames[tag];
            addClassNamesToElement(element, className);
        }
        return element;
    }

    updateDOM(prevNode: HeadingNode, dom: HTMLElement): boolean {
        return false;
    }

    static importDOM(): DOMConversionMap | null {
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

    exportDOM(editor: LexicalEditor): DOMExportOutput {
        const { element } = super.exportDOM(editor);

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

    static importJSON(serializedNode: SerializedHeadingNode): HeadingNode {
        const node = $createHeadingNode(serializedNode.tag);
        node.setFormat(serializedNode.format);
        node.setIndent(serializedNode.indent);
        node.setDirection(serializedNode.direction);
        return node;
    }

    exportJSON(): SerializedHeadingNode {
        return {
            ...super.exportJSON(),
            tag: this.getTag(),
            type: "heading",
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
                ? $createParagraphNode()
                : $createHeadingNode(this.getTag());
        const direction = this.getDirection();
        newElement.setDirection(direction);
        this.insertAfter(newElement, restoreSelection);
        if (anchorOffet === 0 && !this.isEmpty() && selection) {
            const paragraph = $createParagraphNode();
            paragraph.select();
            this.replace(paragraph, true);
        }
        return newElement;
    }

    collapseAtStart(): true {
        const newElement = !this.isEmpty()
            ? $createHeadingNode(this.getTag())
            : $createParagraphNode();
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
        node = $createHeadingNode(nodeName);
        if (element.style !== null) {
            node.setFormat(element.style.textAlign as ElementFormatType);
        }
    }
    return { node };
};


export const $createHeadingNode = (headingTag: HeadingTagType): HeadingNode => {
    return $applyNodeReplacement(new HeadingNode(headingTag));
};

export const $isHeadingNode = (
    node: LexicalNode | null | undefined,
): node is HeadingNode => {
    return node instanceof HeadingNode;
};
