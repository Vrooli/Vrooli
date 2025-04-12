import { NO_DIRTY_NODES } from "../consts.js";
import { NodeConstructorPayloads, NodeType, SerializedRootNode } from "../types.js";
import { getActiveEditor, isCurrentlyReadOnlyMode } from "../updates.js";
import { $createNode, $getRoot, $isNode } from "../utils.js";
import { ElementNode } from "./ElementNode.js";
import { type LexicalNode } from "./LexicalNode.js";

export class RootNode extends ElementNode {
    static __type: NodeType = "Root";
    /** Unstyled cached text */
    __cachedText: null | string;
    /** Styled (i.e. markdown) cached text */
    __cachedMarkdown: null | string;

    static clone(node: RootNode): RootNode {
        const { __key } = node;
        return $createNode("Root", { key: __key });
    }

    constructor({ ...rest }: NodeConstructorPayloads["Root"]) {
        super(rest);
        this.__cachedText = null;
        this.__cachedMarkdown = null;
    }

    getTopLevelElementOrThrow(): never {
        throw new Error("getTopLevelElementOrThrow: root nodes are not top level elements");
    }

    getMarkdownContent() {
        const cachedMarkdown = this.__cachedMarkdown;
        if (
            cachedMarkdown &&
            (isCurrentlyReadOnlyMode() || getActiveEditor()._dirtyType === NO_DIRTY_NODES)
        ) {
            return cachedMarkdown;
        }
        return super.getMarkdownContent();
    }

    getTextContent() {
        const cachedText = this.__cachedText;
        if (
            cachedText &&
            (isCurrentlyReadOnlyMode() || getActiveEditor()._dirtyType === NO_DIRTY_NODES)
        ) {
            return cachedText;
        }
        return super.getTextContent();
    }

    getTextContentSize() {
        const cachedText = this.__cachedText;
        if (
            cachedText &&
            (isCurrentlyReadOnlyMode() || getActiveEditor()._dirtyType === NO_DIRTY_NODES)
        ) {
            return cachedText.length;
        }
        return super.getTextContentSize();
    }

    remove(): never {
        throw new Error("remove: cannot be called on root nodes");
    }

    replace<N = LexicalNode>(node: N): never {
        throw new Error("replace: cannot be called on root nodes");
    }

    insertBefore(nodeToInsert: LexicalNode): LexicalNode {
        throw new Error("insertBefore: cannot be called on root nodes");
    }

    insertAfter(nodeToInsert: LexicalNode): LexicalNode {
        throw new Error("insertAfter: cannot be called on root nodes");
    }

    // View

    updateDOM(prevNode: RootNode, dom: HTMLElement): false {
        return false;
    }

    // Mutate

    append(...nodesToAppend: LexicalNode[]): this {
        for (let i = 0; i < nodesToAppend.length; i++) {
            const node = nodesToAppend[i];
            if (!$isNode("Element", node) && !$isNode("Decorator", node)) {
                throw new Error("rootNode.append: Only element or decorator nodes can be appended to the root node");
            }
        }
        return super.append(...nodesToAppend);
    }

    static importJSON(serializedNode: SerializedRootNode): RootNode {
        // We don't create a root, and instead use the existing root.
        const node = $getRoot();
        node.setFormat(serializedNode.format);
        node.setIndent(serializedNode.indent);
        node.setDirection(serializedNode.direction);
        return node;
    }

    exportJSON(): SerializedRootNode {
        return {
            __type: "Root",
            children: [],
            direction: this.getDirection(),
            format: this.getFormatType(),
            indent: this.getIndent(),
            version: 1,
        };
    }

    collapseAtStart(): true {
        return true;
    }
}
