import { NodeType, SerializedTextNode } from "../types.js";
import { $createNode } from "../utils.js";
import { TextNode } from "./TextNode.js";

export class HashtagNode extends TextNode {
    static __type: NodeType = "Hashtag";

    static clone(node: HashtagNode): HashtagNode {
        const { __text, __key } = node;
        return $createNode("Hashtag", { text: __text, key: __key });
    }

    createDOM(): HTMLElement {
        const element = super.createDOM();
        return element;
    }

    static importJSON({ detail, format, mode, style, text }: SerializedTextNode): HashtagNode {
        const node = $createNode("Hashtag", { text });
        node.setFormat(format);
        node.setDetail(detail);
        node.setMode(mode);
        node.setStyle(style);
        return node;
    }

    exportJSON(): SerializedTextNode {
        return {
            ...super.exportJSON(),
            __type: "Hashtag",
        };
    }

    canInsertTextBefore(): boolean {
        return false;
    }

    isTextEntity(): true {
        return true;
    }
}
