import { PIXEL_VALUE_REG_EXP } from "../consts";
import { DOMConversionMap, DOMConversionOutput, NodeConstructorPayloads, NodeType, SerializedTableRowNode } from "../types";
import { $createNode } from "../utils";
import { ElementNode } from "./ElementNode";

export class TableRowNode extends ElementNode {
    static __type: NodeType = "TableRow";
    __height?: number;

    static clone(node: TableRowNode): TableRowNode {
        const { __height, __key } = node;
        return $createNode("TableRow", { height: __height, key: __key });
    }

    static importDOM(): DOMConversionMap {
        return {
            tr: (node: Node) => ({
                conversion: convertTableRowElement,
                priority: 0,
            }),
        };
    }

    static importJSON({ height }: SerializedTableRowNode): TableRowNode {
        return $createNode("TableRow", { height });
    }

    constructor({ height, ...rest }: NodeConstructorPayloads["TableRow"]) {
        super(rest);
        this.__height = height;
    }

    exportJSON(): SerializedTableRowNode {
        return {
            ...super.exportJSON(),
            ...(this.getHeight() && { height: this.getHeight() }),
            __type: "TableRow",
            version: 1,
        };
    }

    createDOM(): HTMLElement {
        const element = document.createElement("tr");

        if (this.__height) {
            element.style.height = `${this.__height}px`;
        }

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

    return { node: $createNode("TableRow", { height }) };
}
