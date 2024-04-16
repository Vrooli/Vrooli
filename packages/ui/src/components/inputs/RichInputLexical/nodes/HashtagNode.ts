import { EditorConfig, SerializedTextNode } from "../types";
import { $applyNodeReplacement, addClassNamesToElement } from "../utils";
import { LexicalNode } from "./LexicalNode";
import { TextNode } from "./TextNode";

export class HashtagNode extends TextNode {
    static getType(): string {
        return "hashtag";
    }

    static clone(node: HashtagNode): HashtagNode {
        return new HashtagNode(node.__text, node.__key);
    }

    createDOM(config: EditorConfig): HTMLElement {
        const element = super.createDOM(config);
        addClassNamesToElement(element, config.theme.hashtag);
        return element;
    }

    static importJSON(serializedNode: SerializedTextNode): HashtagNode {
        const node = $createHashtagNode(serializedNode.text);
        node.setFormat(serializedNode.format);
        node.setDetail(serializedNode.detail);
        node.setMode(serializedNode.mode);
        node.setStyle(serializedNode.style);
        return node;
    }

    exportJSON(): SerializedTextNode {
        return {
            ...super.exportJSON(),
            type: "hashtag",
        };
    }

    canInsertTextBefore(): boolean {
        return false;
    }

    isTextEntity(): true {
        return true;
    }
}

/**
 * Generates a HashtagNode, which is a string following the format of a # followed by some text, eg. #lexical.
 * @param text - The text used inside the HashtagNode.
 * @returns - The HashtagNode with the embedded text.
 */
export const $createHashtagNode = (text = ""): HashtagNode => {
    return $applyNodeReplacement(new HashtagNode(text));
};

/**
 * Determines if node is a HashtagNode.
 * @param node - The node to be checked.
 * @returns true if node is a HashtagNode, false otherwise.
 */
export const $isHashtagNode = (
    node: LexicalNode | null | undefined,
): node is HashtagNode => {
    return node instanceof HashtagNode;
};
