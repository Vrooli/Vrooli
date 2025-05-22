import { type EditorState, type LexicalEditor } from "./editor.js";
import { type ElementNode } from "./nodes/ElementNode.js";
import { type IntentionallyMarkedAsDirtyElement, type NodeKey, type NodeMap } from "./types.js";
import { $isNode, cloneDecorators, getNextSibling, isAttachedToRoot } from "./utils.js";

export function $garbageCollectDetachedDecorators(
    editor: LexicalEditor,
    pendingEditorState: EditorState,
): void {
    const currentDecorators = editor._decorators;
    const pendingDecorators = editor._pendingDecorators;
    let decorators = pendingDecorators || currentDecorators;
    const nodeMap = pendingEditorState._nodeMap;
    let key;

    for (key in decorators) {
        if (!nodeMap.has(key)) {
            if (decorators === currentDecorators) {
                decorators = cloneDecorators(editor);
            }

            delete decorators[key];
        }
    }
}

function $garbageCollectDetachedDeepChildNodes(
    node: ElementNode,
    parentKey: NodeKey,
    prevNodeMap: NodeMap,
    nodeMap: NodeMap,
    nodeMapDelete: NodeKey[],
    dirtyNodes: Map<NodeKey, IntentionallyMarkedAsDirtyElement>,
): void {
    let child = node.getFirstChild();

    while (child !== null) {
        const childKey = child.__key;
        // TODO Revise condition below, redundant? LexicalNode already cleans up children when moving Nodes
        if (child.__parent === parentKey) {
            if ($isNode("Element", child)) {
                $garbageCollectDetachedDeepChildNodes(
                    child,
                    childKey,
                    prevNodeMap,
                    nodeMap,
                    nodeMapDelete,
                    dirtyNodes,
                );
            }

            // If we have created a node and it was dereferenced, then also
            // remove it from out dirty nodes Set.
            if (!prevNodeMap.has(childKey)) {
                dirtyNodes.delete(childKey);
            }
            nodeMapDelete.push(childKey);
        }
        child = getNextSibling(child);
    }
}

export function $garbageCollectDetachedNodes(
    prevEditorState: EditorState,
    editorState: EditorState,
    dirtyLeaves: Set<NodeKey>,
    dirtyElements: Map<NodeKey, IntentionallyMarkedAsDirtyElement>,
): void {
    const prevNodeMap = prevEditorState._nodeMap;
    const nodeMap = editorState._nodeMap;
    // Store dirtyElements in a queue for later deletion; deleting dirty subtrees too early will
    // hinder accessing .__next on child nodes
    const nodeMapDelete: NodeKey[] = [];

    for (const [nodeKey] of dirtyElements) {
        const node = nodeMap.get(nodeKey);
        if (node !== undefined) {
            // Garbage collect node and its children if they exist
            if (!isAttachedToRoot(node)) {
                if ($isNode("Element", node)) {
                    $garbageCollectDetachedDeepChildNodes(
                        node,
                        nodeKey,
                        prevNodeMap,
                        nodeMap,
                        nodeMapDelete,
                        dirtyElements,
                    );
                }
                // If we have created a node and it was dereferenced, then also
                // remove it from out dirty nodes Set.
                if (!prevNodeMap.has(nodeKey)) {
                    dirtyElements.delete(nodeKey);
                }
                nodeMapDelete.push(nodeKey);
            }
        }
    }
    for (const nodeKey of nodeMapDelete) {
        nodeMap.delete(nodeKey);
    }

    for (const nodeKey of dirtyLeaves) {
        const node = nodeMap.get(nodeKey);
        if (node !== undefined && !isAttachedToRoot(node)) {
            if (!prevNodeMap.has(nodeKey)) {
                dirtyLeaves.delete(nodeKey);
            }
            nodeMap.delete(nodeKey);
        }
    }
}
