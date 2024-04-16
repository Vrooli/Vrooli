import { IS_UNMERGEABLE } from "../consts";
import { DOMConversionMap, NodeKey, SerializedTabNode, TextDetailType, TextModeType } from "../types";
import { $applyNodeReplacement } from "../utils";
import { LexicalNode } from "./LexicalNode";
import { TextNode } from "./TextNode";

export class TabNode extends TextNode {
    static getType(): string {
        return "tab";
    }

    static clone(node: TabNode): TabNode {
        const newNode = new TabNode(node.__key);
        // TabNode __text can be either '\t' or ''. insertText will remove the empty Node
        newNode.__text = node.__text;
        newNode.__format = node.__format;
        newNode.__style = node.__style;
        return newNode;
    }

    constructor(key?: NodeKey) {
        super("\t", key);
        this.__detail = IS_UNMERGEABLE;
    }

    static importDOM(): DOMConversionMap | null {
        return null;
    }

    static importJSON(serializedTabNode: SerializedTabNode): TabNode {
        const node = $createTabNode();
        node.setFormat(serializedTabNode.format);
        node.setStyle(serializedTabNode.style);
        return node;
    }

    exportJSON(): SerializedTabNode {
        return {
            ...super.exportJSON(),
            type: "tab",
            version: 1,
        };
    }

    setTextContent(_text: string): this {
        throw new Error("TabNode does not support setTextContent");
    }

    setDetail(_detail: TextDetailType | number): this {
        throw new Error("TabNode does not support setDetail");
    }

    setMode(_type: TextModeType): this {
        throw new Error("TabNode does not support setMode");
    }

    canInsertTextBefore(): boolean {
        return false;
    }

    canInsertTextAfter(): boolean {
        return false;
    }
}

export const $createTabNode = (): TabNode => {
    return $applyNodeReplacement(new TabNode());
};

export const $isTabNode = (
    node: LexicalNode | null | undefined,
): node is TabNode => {
    return node instanceof TabNode;
};
