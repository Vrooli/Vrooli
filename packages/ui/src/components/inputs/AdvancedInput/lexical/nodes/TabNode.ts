import { IS_UNMERGEABLE } from "../consts.js";
import { type DOMConversionMap, type NodeConstructorPayloads, type NodeType, type SerializedTabNode, type TextDetailType, type TextModeType } from "../types.js";
import { $createNode } from "../utils.js";
import { TextNode } from "./TextNode.js";

export class TabNode extends TextNode {
    static __type: NodeType = "Tab";

    constructor({ ...rest }: NodeConstructorPayloads["Tab"]) {
        super({ text: "\t", ...rest });
        this.__detail = IS_UNMERGEABLE;
    }

    static clone(node: TabNode): TabNode {
        const { __key } = node;
        const newNode = new TabNode({ key: __key });
        // TabNode __text can be either '\t' or ''. insertText will remove the empty Node
        newNode.__text = node.__text;
        newNode.__format = node.__format;
        newNode.__style = node.__style;
        return newNode;
    }

    static importDOM(): DOMConversionMap {
        return {};
    }

    static importJSON(serializedTabNode: SerializedTabNode): TabNode {
        const node = $createNode("Tab", {});
        node.setFormat(serializedTabNode.format);
        node.setStyle(serializedTabNode.style);
        return node;
    }

    exportJSON(): SerializedTabNode {
        return {
            ...super.exportJSON(),
            __type: "Tab",
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
