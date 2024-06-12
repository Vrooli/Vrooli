import { LexicalEditor } from "../editor";
import { EditorConfig, NodeType } from "../types";
import { LexicalNode } from "./LexicalNode";

export class DecoratorNode<T> extends LexicalNode {
    static __type: NodeType = "Decorator";

    /**
     * The returned value is added to the LexicalEditor._decorators
     */
    decorate(_editor: LexicalEditor, _config: EditorConfig): T {
        throw new Error("decorate: base method not extended");
    }

    isIsolated(): boolean {
        return false;
    }

    isInline(): boolean {
        return true;
    }

    isKeyboardSelectable(): boolean {
        return true;
    }
}
