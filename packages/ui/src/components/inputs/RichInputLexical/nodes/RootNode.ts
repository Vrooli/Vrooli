import { NO_DIRTY_NODES } from "../consts";
import { SerializedRootNode } from "../types";
import { getActiveEditor, isCurrentlyReadOnlyMode } from "../updates";
import { $getRoot, $isDecoratorNode, $isElementNode } from "../utils";
import { ElementNode } from "./ElementNode";
import { LexicalNode } from "./LexicalNode";

export class RootNode extends ElementNode {
    /** @internal */
    __cachedText: null | string;

    static getType(): string {
        return "root";
    }

    static clone(): RootNode {
        return new RootNode();
    }

    constructor() {
        super("root");
        this.__cachedText = null;
    }

    getTopLevelElementOrThrow(): never {
        throw new Error("getTopLevelElementOrThrow: root nodes are not top level elements");
    }

    getTextContent(): string {
        const cachedText = this.__cachedText;
        if (
            isCurrentlyReadOnlyMode() ||
            getActiveEditor()._dirtyType === NO_DIRTY_NODES
        ) {
            if (cachedText !== null) {
                return cachedText;
            }
        }
        return super.getTextContent();
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
            if (!$isElementNode(node) && !$isDecoratorNode(node)) {
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
            children: [],
            direction: this.getDirection(),
            format: this.getFormatType(),
            indent: this.getIndent(),
            type: "root",
            version: 1,
        };
    }

    collapseAtStart(): true {
        return true;
    }
}

export const $createRootNode = (): RootNode => {
    return new RootNode();
};

export const $isRootNode = (
    node: RootNode | LexicalNode | null | undefined,
): node is RootNode => {
    return node instanceof RootNode;
};
