import { PIXEL_VALUE_REG_EXP } from "../consts";
import { DOMConversionMap, DOMConversionOutput, EditorConfig, NodeKey, SerializedTableRowNode } from "../types";
import { $applyNodeReplacement, addClassNamesToElement } from "../utils";
import { ElementNode } from "./ElementNode";
import { LexicalNode } from "./LexicalNode";

export class TableRowNode extends ElementNode {
    /** @internal */
    __height?: number;

    static getType(): string {
        return "tablerow";
    }

    static clone(node: TableRowNode): TableRowNode {
        return new TableRowNode(node.__height, node.__key);
    }

    static importDOM(): DOMConversionMap | null {
        return {
            tr: (node: Node) => ({
                conversion: convertTableRowElement,
                priority: 0,
            }),
        };
    }

    static importJSON(serializedNode: SerializedTableRowNode): TableRowNode {
        return $createTableRowNode(serializedNode.height);
    }

    constructor(height?: number, key?: NodeKey) {
        super(key);
        this.__height = height;
    }

    exportJSON(): SerializedTableRowNode {
        return {
            ...super.exportJSON(),
            ...(this.getHeight() && { height: this.getHeight() }),
            type: "tablerow",
            version: 1,
        };
    }

    createDOM(config: EditorConfig): HTMLElement {
        const element = document.createElement("tr");

        if (this.__height) {
            element.style.height = `${this.__height}px`;
        }

        addClassNamesToElement(element, config.theme.tableRow);

        return element;
    }

    isShadowRoot(): boolean {
        return true;
    }

    setHeight(height: number): number | null | undefined {
        const self = this.getWritable();
        self.__height = height;
        return this.__height;
    }

    getHeight(): number | undefined {
        return this.getLatest().__height;
    }

    updateDOM(prevNode: TableRowNode): boolean {
        return prevNode.__height !== this.__height;
    }

    canBeEmpty(): false {
        return false;
    }

    canIndent(): false {
        return false;
    }
}

export function convertTableRowElement(domNode: Node): DOMConversionOutput {
    const domNode_ = domNode as HTMLTableCellElement;
    let height: number | undefined = undefined;

    if (PIXEL_VALUE_REG_EXP.test(domNode_.style.height)) {
        height = parseFloat(domNode_.style.height);
    }

    return { node: $createTableRowNode(height) };
}

export function $createTableRowNode(height?: number): TableRowNode {
    return $applyNodeReplacement(new TableRowNode(height));
}

export function $isTableRowNode(
    node: LexicalNode | null | undefined,
): node is TableRowNode {
    return node instanceof TableRowNode;
}
