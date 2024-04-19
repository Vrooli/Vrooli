import { useCallback, useEffect } from "react";
import { CLICK_COMMAND, KEY_BACKSPACE_COMMAND, KEY_DELETE_COMMAND } from "../commands";
import { COMMAND_PRIORITY_LOW } from "../consts";
import { useLexicalComposerContext } from "../context";
import { useLexicalNodeSelection } from "../selection";
import { DOMConversionMap, DOMExportOutput, NodeKey, NodeType, SerializedHorizontalRuleNode, SerializedLexicalNode } from "../types";
import { $createNode, $getNodeByKey, $getSelection, $isNode, $isNodeSelection, mergeRegister } from "../utils";
import { DecoratorNode } from "./DecoratorNode";

const HorizontalRuleComponent = ({ nodeKey }: { nodeKey: NodeKey }) => {
    const editor = useLexicalComposerContext();
    const [isSelected, setSelected, clearSelection] =
        useLexicalNodeSelection(nodeKey);

    const onDelete = useCallback(
        (event: KeyboardEvent) => {
            if (isSelected && $isNodeSelection($getSelection())) {
                event.preventDefault();
                const node = $getNodeByKey(nodeKey);
                if ($isNode("HorizontalRule", node)) {
                    node.remove();
                    return true;
                }
            }
            return false;
        },
        [isSelected, nodeKey],
    );

    useEffect(() => {
        return mergeRegister(
            editor.registerCommand(
                CLICK_COMMAND,
                (event: MouseEvent) => {
                    const hrElem = editor.getElementByKey(nodeKey);

                    if (event.target === hrElem) {
                        if (!event.shiftKey) {
                            clearSelection();
                        }
                        setSelected(!isSelected);
                        return true;
                    }

                    return false;
                },
                COMMAND_PRIORITY_LOW,
            ),
            editor.registerCommand(
                KEY_DELETE_COMMAND,
                onDelete,
                COMMAND_PRIORITY_LOW,
            ),
            editor.registerCommand(
                KEY_BACKSPACE_COMMAND,
                onDelete,
                COMMAND_PRIORITY_LOW,
            ),
        );
    }, [clearSelection, editor, isSelected, nodeKey, onDelete, setSelected]);

    useEffect(() => {
        const hrElem = editor.getElementByKey(nodeKey);
        if (hrElem !== null) {
            hrElem.className = isSelected ? "selected" : "";
        }
    }, [editor, isSelected, nodeKey]);

    return null;
};

export class HorizontalRuleNode extends DecoratorNode<JSX.Element> {
    static __type: NodeType = "HorizontalRule";

    static clone(node: HorizontalRuleNode): HorizontalRuleNode {
        return new HorizontalRuleNode(node.__key);
    }

    static importJSON(serializedNode: SerializedHorizontalRuleNode): HorizontalRuleNode {
        return $createNode("HorizontalRule", {});
    }

    static importDOM(): DOMConversionMap {
        return {
            hr: () => ({
                conversion: () => ({ node: $createNode("HorizontalRule", {}) }),
                priority: 0,
            }),
        };
    }

    exportJSON(): SerializedLexicalNode {
        return {
            __type: "HorizontalRule",
            version: 1,
        };
    }

    exportDOM(): DOMExportOutput {
        return { element: document.createElement("hr") };
    }

    createDOM(): HTMLElement {
        return document.createElement("hr");
    }

    getTextContent(): string {
        return "\n";
    }

    isInline(): false {
        return false;
    }

    updateDOM(): boolean {
        return false;
    }

    decorate(): JSX.Element {
        return <HorizontalRuleComponent nodeKey={this.__key} />;
    }
}
