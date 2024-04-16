import { useCallback, useEffect } from "react";
import { CLICK_COMMAND, KEY_BACKSPACE_COMMAND, KEY_DELETE_COMMAND } from "../commands";
import { COMMAND_PRIORITY_LOW } from "../consts";
import { useLexicalComposerContext } from "../context";
import { useLexicalNodeSelection } from "../selection";
import { DOMConversionMap, DOMConversionOutput, DOMExportOutput, NodeKey, SerializedHorizontalRuleNode, SerializedLexicalNode } from "../types";
import { $applyNodeReplacement, $getNodeByKey, $getSelection, $isNodeSelection, mergeRegister } from "../utils";
import { DecoratorNode } from "./DecoratorNode";
import { LexicalNode } from "./LexicalNode";

const HorizontalRuleComponent = ({ nodeKey }: { nodeKey: NodeKey }) => {
    const [editor] = useLexicalComposerContext();
    const [isSelected, setSelected, clearSelection] =
        useLexicalNodeSelection(nodeKey);

    const onDelete = useCallback(
        (event: KeyboardEvent) => {
            if (isSelected && $isNodeSelection($getSelection())) {
                event.preventDefault();
                const node = $getNodeByKey(nodeKey);
                if ($isHorizontalRuleNode(node)) {
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
    static getType(): string {
        return "horizontalrule";
    }

    static clone(node: HorizontalRuleNode): HorizontalRuleNode {
        return new HorizontalRuleNode(node.__key);
    }

    static importJSON(
        serializedNode: SerializedHorizontalRuleNode,
    ): HorizontalRuleNode {
        return $createHorizontalRuleNode();
    }

    static importDOM(): DOMConversionMap | null {
        return {
            hr: () => ({
                conversion: convertHorizontalRuleElement,
                priority: 0,
            }),
        };
    }

    exportJSON(): SerializedLexicalNode {
        return {
            type: "horizontalrule",
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

function convertHorizontalRuleElement(): DOMConversionOutput {
    return { node: $createHorizontalRuleNode() };
}

export function $createHorizontalRuleNode(): HorizontalRuleNode {
    return $applyNodeReplacement(new HorizontalRuleNode());
}

export function $isHorizontalRuleNode(
    node: LexicalNode | null | undefined,
): node is HorizontalRuleNode {
    return node instanceof HorizontalRuleNode;
}
