/* eslint-disable @typescript-eslint/ban-ts-comment */
import { SELECTION_CHANGE_COMMAND } from "./commands.js";
import { CSS_TO_STYLES, FULL_RECONCILE, NO_DIRTY_NODES } from "./consts.js";
import { type EditorState, type LexicalEditor, createEmptyEditorState, editorStateHasDirtySelection, resetEditor } from "./editor.js";
import { $garbageCollectDetachedDecorators, $garbageCollectDetachedNodes } from "./garbageCollector.js";
import { initMutationObserver } from "./mutations.js";
import { type LexicalNode } from "./nodes/LexicalNode.js";
import { type TextNode } from "./nodes/TextNode.js";
import { reconcileRoot } from "./reconcile.js";
import { applySelectionTransforms, internalCreateSelection, updateDOMSelection } from "./selection.js";
import { type CommandPayloadType, type EditorListenerPayload, type EditorListeners, type EditorUpdateOptions, type LexicalCommand, type MutatedNodes, type NodeType, type RegisteredNode, type RegisteredNodes, type SerializedEditorState, type SerializedLexicalNode, type Transform } from "./types.js";
import { $getCompositionKey, $getRoot, $isNode, $isNodeSelection, $isRangeSelection, getDOMSelection, getNextSibling, getPreviousSibling, getStyleObjectFromRawCSS, isAttachedToRoot, removeDOMBlockCursorElement, scheduleMicroTask, updateDOMBlockCursorElement } from "./utils.js";

let activeEditorState: null | EditorState = null;
let activeEditor: null | LexicalEditor = null;
let isReadOnlyMode = false;
let isAttemptingToRecoverFromReconcilerError = false;
let infiniteTransformCount = 0;

const observerOptions = {
    characterData: true,
    childList: true,
    subtree: true,
};

export function isCurrentlyReadOnlyMode(): boolean {
    return (
        isReadOnlyMode ||
        (activeEditorState !== null && activeEditorState._readOnly)
    );
}

export function getActiveEditorState(): EditorState {
    if (activeEditorState === null) {
        throw new Error("Unable to find an active editor state. State helpers or node methods can only be used synchronously during the callback of editor.update() or editorState.read().");
    }

    return activeEditorState;
}

export function getActiveEditor(): LexicalEditor {
    if (activeEditor === null) {
        throw new Error("Unable to find an active editor. This method can only be used synchronously during the callback of editor.update().");
    }

    return activeEditor;
}

export function internalGetActiveEditor(): LexicalEditor | null {
    return activeEditor;
}

export function setActiveEditor(editor: LexicalEditor): void {
    activeEditor = editor;
}

/**
 * Triggers the command listeners for the given command type.
 * @param editor The editor to trigger the command listeners on.
 * @param type The command type to trigger.
 * @param payload The payload to pass to the command listeners.
 * @returns True if the command was handled by a listener (meaning no 
 * further listeners should be called), false otherwise.
 */
export function dispatchCommand<
    TCommand extends LexicalCommand<unknown>,
>(
    editor: LexicalEditor,
    type: TCommand,
    payload: CommandPayloadType<TCommand>,
): boolean {
    if (editor._updating === false || activeEditor !== editor) {
        let returnVal = false;
        editor.update(() => {
            returnVal = dispatchCommand(editor, type, payload);
        });
        return returnVal;
    }

    const commandListeners = editor._commands;
    const listenersInPriorityOrder = commandListeners.get(type);
    if (listenersInPriorityOrder === undefined) {
        return false;
    }

    // Loop backwards through the listeners, since it goes from least to most important
    for (let i = listenersInPriorityOrder.length - 1; i >= 0; i--) {
        const listeners = listenersInPriorityOrder[i];
        if (listeners === undefined) {
            continue;
        }

        for (const listener of listeners) {
            if (listener(payload, editor) === true) {
                return true;
            }
        }
    }

    return false;
}

function triggerMutationListeners(
    editor: LexicalEditor,
    mutatedNodes: MutatedNodes,
    updateTags: Set<string>,
    dirtyLeaves: Set<string>,
    prevEditorState: EditorState,
) {
    // Loop through every mutated node type
    for (const [nodeType, mutatedNodesByType] of Object.entries(mutatedNodes)) {
        // Check if there are any listeners for this node type
        const listeners = editor._listeners[nodeType] as EditorListeners[NodeType];
        // If there are no listeners for this node type, skip to the next node type
        if (!listeners) {
            continue;
        }
        // Call every listener in the set
        for (const listener of listeners) {
            listener(mutatedNodesByType, {
                dirtyLeaves,
                prevEditorState,
                updateTags,
            });
        }
    }
}

export function triggerListeners<T extends keyof EditorListenerPayload>(
    type: T,
    editor: LexicalEditor,
    isCurrentlyEnqueuingUpdates: boolean,
    payload: EditorListenerPayload[T],
) {
    const previouslyUpdating = editor._updating;
    editor._updating = isCurrentlyEnqueuingUpdates;

    try {
        const listeners = editor._listeners[type];
        if (!listeners) {
            return;
        }
        for (const listener of listeners) {
            listener(payload as never);
        }
    } finally {
        editor._updating = previouslyUpdating;
    }
}

/**
 * Triggers the text content listeners if the markdown content has changed. 
 * This includes text changes and formatting changes
 */
function triggerTextContentListeners(
    editor: LexicalEditor,
    currentEditorState: EditorState,
    pendingEditorState: EditorState,
) {
    const currentMarkdownContent = currentEditorState.read(() => $getRoot().getMarkdownContent());
    const latestMarkdownContent = pendingEditorState.read(() => $getRoot().getMarkdownContent());
    console.log("in triggerTextContentListeners💗 - markdown comparison", currentMarkdownContent.length, latestMarkdownContent.length, currentEditorState._nodeMap, pendingEditorState._nodeMap);

    if (currentMarkdownContent !== latestMarkdownContent) {
        triggerListeners("textcontent", editor, true, latestMarkdownContent);
    }
}

export function readEditorState<V>(
    editorState: EditorState,
    callbackFn: () => V,
): V {
    const previousActiveEditorState = activeEditorState;
    const previousReadOnlyMode = isReadOnlyMode;
    const previousActiveEditor = activeEditor;

    activeEditorState = editorState;
    isReadOnlyMode = true;
    activeEditor = null;

    try {
        return callbackFn();
    } finally {
        activeEditorState = previousActiveEditorState;
        isReadOnlyMode = previousReadOnlyMode;
        activeEditor = previousActiveEditor;
    }
}

/**
 * Gets the TextNode's style object and adds the styles to the CSS.
 * @param node - The TextNode to add styles to.
 */
export function $addNodeStyle(node: TextNode): void {
    const CSSText = node.getStyle();
    const styles = getStyleObjectFromRawCSS(CSSText);
    CSS_TO_STYLES.set(CSSText, styles);
}

type InternalSerializedNode = {
    children?: Array<InternalSerializedNode>;
    __type: NodeType;
    version: number;
};

export function $parseSerializedNode(
    serializedNode: SerializedLexicalNode,
): LexicalNode {
    const internalSerializedNode: InternalSerializedNode = serializedNode;
    return $parseSerializedNodeImpl(
        internalSerializedNode,
        getActiveEditor()._nodes,
    );
}

function $parseSerializedNodeImpl<
    SerializedNode extends InternalSerializedNode,
>(
    serializedNode: SerializedNode,
    registeredNodes: RegisteredNodes,
): LexicalNode {
    const type = serializedNode.__type;
    const registeredNode = registeredNodes[type];

    if (registeredNode === undefined) {
        throw new Error(`parseEditorState: type "${type}" not found`);
    }

    const nodeClass = registeredNode.klass;

    const node = nodeClass.importJSON(serializedNode);
    const children = serializedNode.children;

    if ($isNode("Element", node) && Array.isArray(children)) {
        for (let i = 0; i < children.length; i++) {
            const serializedJSONChildNode = children[i];
            const childNode = $parseSerializedNodeImpl(
                serializedJSONChildNode,
                registeredNodes,
            );
            node.append(childNode);
        }
    }

    return node;
}

export function parseEditorState(
    serializedEditorState: SerializedEditorState,
    editor: LexicalEditor,
    updateFn: void | (() => void),
): EditorState {
    const editorState = createEmptyEditorState();
    const previousActiveEditorState = activeEditorState;
    const previousReadOnlyMode = isReadOnlyMode;
    const previousActiveEditor = activeEditor;
    const previousDirtyElements = editor._dirtyElements;
    const previousDirtyLeaves = editor._dirtyLeaves;
    const previousCloneNotNeeded = editor._cloneNotNeeded;
    const previousDirtyType = editor._dirtyType;
    editor._dirtyElements = new Map();
    editor._dirtyLeaves = new Set();
    editor._cloneNotNeeded = new Set();
    editor._dirtyType = 0;
    activeEditorState = editorState;
    isReadOnlyMode = false;
    activeEditor = editor;

    try {
        const registeredNodes = editor._nodes;
        const serializedNode = serializedEditorState.root;
        $parseSerializedNodeImpl(serializedNode, registeredNodes);
        if (updateFn) {
            updateFn();
        }

        // Make the editorState immutable
        editorState._readOnly = true;

    } catch (error) {
        if (error instanceof Error) {
            editor._onError(error);
        }
    } finally {
        editor._dirtyElements = previousDirtyElements;
        editor._dirtyLeaves = previousDirtyLeaves;
        editor._cloneNotNeeded = previousCloneNotNeeded;
        editor._dirtyType = previousDirtyType;
        activeEditorState = previousActiveEditorState;
        isReadOnlyMode = previousReadOnlyMode;
        activeEditor = previousActiveEditor;
    }

    return editorState;
}

function triggerDeferredUpdateCallbacks(
    editor: LexicalEditor,
    deferred: Array<() => void>,
) {
    editor._deferred = [];

    if (deferred.length !== 0) {
        const previouslyUpdating = editor._updating;
        editor._updating = true;

        try {
            for (let i = 0; i < deferred.length; i++) {
                deferred[i]();
            }
        } finally {
            editor._updating = previouslyUpdating;
        }
    }
}

function triggerEnqueuedUpdates(editor: LexicalEditor): void {
    const queuedUpdates = editor._updates;

    if (queuedUpdates.length !== 0) {
        const queuedUpdate = queuedUpdates.shift();
        if (queuedUpdate) {
            const [updateFn, options] = queuedUpdate;
            beginUpdate(editor, updateFn, options);
        }
    }
}

export function commitPendingUpdates(
    editor: LexicalEditor,
    recoveryEditorState?: EditorState | null,
) {
    const pendingEditorState = editor._pendingEditorState;
    const rootElement = editor._rootElement;
    const shouldSkipDOM = rootElement === null;

    if (pendingEditorState === null) {
        return;
    }

    // ======
    // Reconciliation has started.
    // ======

    const currentEditorState = editor._editorState;
    const currentSelection = currentEditorState?._selection;
    const pendingSelection = pendingEditorState._selection;
    const needsUpdate = editor._dirtyType !== NO_DIRTY_NODES;
    const previousActiveEditorState = activeEditorState;
    const previousReadOnlyMode = isReadOnlyMode;
    const previousActiveEditor = activeEditor;
    const previouslyUpdating = editor._updating;
    const observer = editor._observer;
    let mutatedNodes: MutatedNodes | null = null;
    editor._pendingEditorState = null;
    editor._editorState = pendingEditorState;

    if (!shouldSkipDOM && needsUpdate && observer !== null && currentEditorState) {
        activeEditor = editor;
        activeEditorState = pendingEditorState;
        isReadOnlyMode = false;
        // We don't want updates to sync block the reconciliation.
        editor._updating = true;
        try {
            const dirtyType = editor._dirtyType;
            const dirtyElements = editor._dirtyElements;
            const dirtyLeaves = editor._dirtyLeaves;
            observer.disconnect();

            mutatedNodes = reconcileRoot( //TODO this is probably causing root node to duplicate
                currentEditorState,
                pendingEditorState,
                editor,
                dirtyType,
                dirtyElements,
                dirtyLeaves,
            );
        } catch (error) {
            // Report errors
            if (error instanceof Error) {
                editor._onError(error);
            }

            // Reset editor and restore incoming editor state to the DOM
            if (!isAttemptingToRecoverFromReconcilerError) {
                resetEditor(editor, null, rootElement, pendingEditorState);
                initMutationObserver(editor);
                editor._dirtyType = FULL_RECONCILE;
                isAttemptingToRecoverFromReconcilerError = true;
                commitPendingUpdates(editor, currentEditorState);
                isAttemptingToRecoverFromReconcilerError = false;
            } else {
                // To avoid a possible situation of infinite loops, lets throw
                throw error;
            }

            return;
        } finally {
            observer.observe(rootElement as Node, observerOptions);
            editor._updating = previouslyUpdating;
            activeEditorState = previousActiveEditorState;
            isReadOnlyMode = previousReadOnlyMode;
            activeEditor = previousActiveEditor;
        }
    }

    if (!pendingEditorState._readOnly) {
        pendingEditorState._readOnly = true;
    }

    const dirtyLeaves = editor._dirtyLeaves;
    const dirtyElements = editor._dirtyElements;
    const normalizedNodes = editor._normalizedNodes;
    const tags = editor._updateTags;
    const deferred = editor._deferred;
    const nodeCount = pendingEditorState._nodeMap.size;

    if (needsUpdate) {
        editor._dirtyType = NO_DIRTY_NODES;
        editor._cloneNotNeeded.clear();
        editor._dirtyLeaves = new Set();
        editor._dirtyElements = new Map();
        editor._normalizedNodes = new Set();
        editor._updateTags = new Set();
    }
    $garbageCollectDetachedDecorators(editor, pendingEditorState);

    // ======
    // Reconciliation has finished. Now update selection and trigger listeners.
    // ======

    const domSelection = shouldSkipDOM ? null : getDOMSelection(editor._window);

    // Attempt to update the DOM selection, including focusing of the root element,
    // and scroll into view if needed.
    if (
        editor._editable &&
        // domSelection will be null in headless
        domSelection !== null &&
        (needsUpdate || pendingSelection === null || pendingSelection.dirty)
    ) {
        activeEditor = editor;
        activeEditorState = pendingEditorState;
        try {
            if (observer !== null) {
                observer.disconnect();
            }
            if (needsUpdate || pendingSelection === null || pendingSelection.dirty) {
                const blockCursorElement = editor._blockCursorElement;
                if (blockCursorElement !== null) {
                    removeDOMBlockCursorElement(
                        blockCursorElement,
                        editor,
                        rootElement as HTMLElement,
                    );
                }
                updateDOMSelection(
                    currentSelection,
                    pendingSelection,
                    editor,
                    domSelection,
                    tags,
                    rootElement as HTMLElement,
                    nodeCount,
                );
            }
            updateDOMBlockCursorElement(
                editor,
                rootElement as HTMLElement,
                pendingSelection,
            );
            if (observer !== null) {
                observer.observe(rootElement as Node, observerOptions);
            }
        } finally {
            activeEditor = previousActiveEditor;
            activeEditorState = previousActiveEditorState;
        }
    }

    if (mutatedNodes !== null && currentEditorState) {
        triggerMutationListeners(
            editor,
            mutatedNodes,
            tags,
            dirtyLeaves,
            currentEditorState,
        );
    }
    if (
        !$isRangeSelection(pendingSelection) &&
        pendingSelection !== null &&
        (currentSelection === null || currentSelection === undefined || !currentSelection.is(pendingSelection))
    ) {
        editor.dispatchCommand(SELECTION_CHANGE_COMMAND, undefined);
    }
    /**
     * Capture pendingDecorators after garbage collecting detached decorators
     */
    const pendingDecorators = editor._pendingDecorators;
    if (pendingDecorators !== null) {
        editor._decorators = pendingDecorators;
        editor._pendingDecorators = null;
        triggerListeners("decorator", editor, true, pendingDecorators as Record<string, never>);
    }

    // If reconciler fails, we reset whole editor (so current editor state becomes empty)
    // and attempt to re-render pendingEditorState. If that goes through we trigger
    // listeners, but instead use recoverEditorState which is current editor state before reset
    // This specifically important for collab that relies on prevEditorState from update
    // listener to calculate delta of changed nodes/properties
    if (recoveryEditorState || currentEditorState) {
        triggerTextContentListeners(
            editor,
            (recoveryEditorState || currentEditorState) as EditorState,
            pendingEditorState,
        );
    }
    triggerListeners("update", editor, true, {
        dirtyElements,
        dirtyLeaves,
        editorState: pendingEditorState,
        normalizedNodes,
        prevEditorState: (recoveryEditorState || currentEditorState) as EditorState,
        tags,
    });
    triggerDeferredUpdateCallbacks(editor, deferred);
    triggerEnqueuedUpdates(editor);
}

function processNestedUpdates(
    editor: LexicalEditor,
    initialSkipTransforms?: boolean,
): boolean {
    const queuedUpdates = editor._updates;
    let skipTransforms = initialSkipTransforms || false;

    // Updates might grow as we process them, we so we'll need
    // to handle each update as we go until the updates array is
    // empty.
    while (queuedUpdates.length !== 0) {
        const queuedUpdate = queuedUpdates.shift();
        if (queuedUpdate) {
            const [nextUpdateFn, options] = queuedUpdate;

            let onUpdate;
            let tag;

            if (options !== undefined) {
                onUpdate = options.onUpdate;
                tag = options.tag;

                if (options.skipTransforms) {
                    skipTransforms = true;
                }

                if (onUpdate) {
                    editor._deferred.push(onUpdate);
                }

                if (tag) {
                    editor._updateTags.add(tag);
                }
            }

            nextUpdateFn();
        }
    }

    return skipTransforms;
}

export function getRegisteredNodeOrThrow(
    editor: LexicalEditor,
    nodeType: NodeType,
): RegisteredNode {
    const registeredNode = editor._nodes[nodeType];
    if (registeredNode === undefined) {
        throw new Error("registeredNode: Type not found");
    }
    return registeredNode;
}

export function $applyTransforms(
    editor: LexicalEditor,
    node: LexicalNode,
    transformsCache: Map<string, Array<Transform<LexicalNode>>>,
) {
    const type = node.getType();
    const registeredNode = getRegisteredNodeOrThrow(editor, type);
    let transformsArr = transformsCache.get(type);

    if (transformsArr === undefined) {
        transformsArr = Array.from(registeredNode.transforms);
        transformsCache.set(type, transformsArr);
    }

    const transformsArrLength = transformsArr.length;

    for (let i = 0; i < transformsArrLength; i++) {
        transformsArr[i](node);

        if (!isAttachedToRoot(node)) {
            break;
        }
    }
}

function $isNodeValidForTransform(
    node: LexicalNode,
    compositionKey: null | string,
): boolean {
    return (
        node !== undefined &&
        // We don't want to transform nodes being composed
        node.__key !== compositionKey &&
        isAttachedToRoot(node)
    );
}

/**
 * Transform heuristic:
 * 1. We transform leaves first. If transforms generate additional dirty nodes we repeat step 1.
 * The reasoning behind this is that marking a leaf as dirty marks all its parent elements as dirty too.
 * 2. We transform elements. If element transforms generate additional dirty nodes we repeat step 1.
 * If element transforms only generate additional dirty elements we only repeat step 2.
 *
 * Note that to keep track of newly dirty nodes and subtrees we leverage the editor._dirtyNodes and
 * editor._subtrees which we reset in every loop.
 */
function $applyAllTransforms(
    editorState: EditorState,
    editor: LexicalEditor,
) {
    const dirtyLeaves = editor._dirtyLeaves;
    const dirtyElements = editor._dirtyElements;
    const nodeMap = editorState._nodeMap;
    const compositionKey = $getCompositionKey();
    const transformsCache = new Map();

    let untransformedDirtyLeaves = dirtyLeaves;
    let untransformedDirtyLeavesLength = untransformedDirtyLeaves.size;
    let untransformedDirtyElements = dirtyElements;
    let untransformedDirtyElementsLength = untransformedDirtyElements.size;

    while (
        untransformedDirtyLeavesLength > 0 ||
        untransformedDirtyElementsLength > 0
    ) {
        if (untransformedDirtyLeavesLength > 0) {
            // We leverage editor._dirtyLeaves to track the new dirty leaves after the transforms
            editor._dirtyLeaves = new Set();

            for (const nodeKey of untransformedDirtyLeaves) {
                const node = nodeMap.get(nodeKey);

                if (
                    $isNode("Text", node) &&
                    isAttachedToRoot(node) &&
                    node.isSimpleText() &&
                    !node.isUnmergeable()
                ) {
                    $normalizeTextNode(node);
                }

                if (
                    node !== undefined &&
                    $isNodeValidForTransform(node, compositionKey)
                ) {
                    $applyTransforms(editor, node, transformsCache);
                }

                dirtyLeaves.add(nodeKey);
            }

            untransformedDirtyLeaves = editor._dirtyLeaves;
            untransformedDirtyLeavesLength = untransformedDirtyLeaves.size;

            // We want to prioritize node transforms over element transforms
            if (untransformedDirtyLeavesLength > 0) {
                infiniteTransformCount++;
                continue;
            }
        }

        // All dirty leaves have been processed. Let's do elements!
        // We have previously processed dirty leaves, so let's restart the editor leaves Set to track
        // new ones caused by element transforms
        editor._dirtyLeaves = new Set();
        editor._dirtyElements = new Map();

        for (const currentUntransformedDirtyElement of untransformedDirtyElements) {
            const nodeKey = currentUntransformedDirtyElement[0];
            const intentionallyMarkedAsDirty = currentUntransformedDirtyElement[1];
            if (nodeKey !== "root" && !intentionallyMarkedAsDirty) {
                continue;
            }

            const node = nodeMap.get(nodeKey);

            if (
                node !== undefined &&
                $isNodeValidForTransform(node, compositionKey)
            ) {
                $applyTransforms(editor, node, transformsCache);
            }

            dirtyElements.set(nodeKey, intentionallyMarkedAsDirty);
        }

        untransformedDirtyLeaves = editor._dirtyLeaves;
        untransformedDirtyLeavesLength = untransformedDirtyLeaves.size;
        untransformedDirtyElements = editor._dirtyElements;
        untransformedDirtyElementsLength = untransformedDirtyElements.size;
        infiniteTransformCount++;
    }

    editor._dirtyLeaves = dirtyLeaves;
    editor._dirtyElements = dirtyElements;
}

function $canSimpleTextNodesBeMerged(
    node1: TextNode,
    node2: TextNode,
): boolean {
    const node1Mode = node1.__mode;
    const node1Format = node1.__format;
    const node1Style = node1.__style;
    const node2Mode = node2.__mode;
    const node2Format = node2.__format;
    const node2Style = node2.__style;
    return (
        (node1Mode === null || node1Mode === node2Mode) &&
        (node1Format === null || node1Format === node2Format) &&
        (node1Style === null || node1Style === node2Style)
    );
}

function $mergeTextNodes(node1: TextNode, node2: TextNode): TextNode {
    const writableNode1 = node1.mergeWithSibling(node2);

    const normalizedNodes = getActiveEditor()._normalizedNodes;

    normalizedNodes.add(node1.__key);
    normalizedNodes.add(node2.__key);
    return writableNode1;
}

export function $normalizeTextNode(textNode: TextNode) {
    let node = textNode;

    if (node.__text === "" && node.isSimpleText() && !node.isUnmergeable()) {
        node.remove();
        return;
    }

    // Backward
    let previousNode;

    while (
        (previousNode = getPreviousSibling(node)) !== null &&
        $isNode("Text", previousNode) &&
        previousNode.isSimpleText() &&
        !previousNode.isUnmergeable()
    ) {
        if (previousNode.__text === "") {
            previousNode.remove();
        } else if ($canSimpleTextNodesBeMerged(previousNode, node)) {
            node = $mergeTextNodes(previousNode, node);
            break;
        } else {
            break;
        }
    }

    // Forward
    let nextNode;

    while (
        (nextNode = getNextSibling(node)) !== null &&
        $isNode("Text", nextNode) &&
        nextNode.isSimpleText() &&
        !nextNode.isUnmergeable()
    ) {
        if (nextNode.__text === "") {
            nextNode.remove();
        } else if ($canSimpleTextNodesBeMerged(node, nextNode)) {
            node = $mergeTextNodes(node, nextNode);
            break;
        } else {
            break;
        }
    }
}

function $normalizeAllDirtyTextNodes(
    editorState: EditorState,
    editor: LexicalEditor,
) {
    const dirtyLeaves = editor._dirtyLeaves;
    const nodeMap = editorState._nodeMap;

    for (const nodeKey of dirtyLeaves) {
        const node = nodeMap.get(nodeKey);

        if (
            $isNode("Text", node) &&
            isAttachedToRoot(node) &&
            node.isSimpleText() &&
            !node.isUnmergeable()
        ) {
            $normalizeTextNode(node);
        }
    }
}

function beginUpdate(
    editor: LexicalEditor,
    updateFn: () => void,
    options?: EditorUpdateOptions,
) {
    const updateTags = editor._updateTags;
    let onUpdate: (() => void) | null = null;
    let tag: string | null = null;
    let skipTransforms = false;
    let discrete = false;

    if (options) {
        onUpdate = options.onUpdate || null;
        tag = options.tag || null;

        if (tag) {
            updateTags.add(tag);
        }

        skipTransforms = options.skipTransforms || false;
        discrete = options.discrete || false;
    }

    if (onUpdate) {
        editor._deferred.push(onUpdate);
    }

    const currentEditorState = editor._editorState;
    let pendingEditorState = editor._pendingEditorState;
    let editorStateWasCloned = false;

    if ((pendingEditorState === null || pendingEditorState._readOnly) && currentEditorState) {
        // Set pendingEditorState and editor._pendingEditorState at the same time
        pendingEditorState = editor._pendingEditorState = (pendingEditorState || currentEditorState).clone();
        editorStateWasCloned = true;
    }
    if (!pendingEditorState) {
        throw new Error("Editor state is null");
    }
    pendingEditorState._flushSync = discrete;

    const previousActiveEditorState = activeEditorState;
    const previousReadOnlyMode = isReadOnlyMode;
    const previousActiveEditor = activeEditor;
    const previouslyUpdating = editor._updating;
    activeEditorState = pendingEditorState;
    isReadOnlyMode = false;
    editor._updating = true;
    activeEditor = editor;

    try {
        if (editorStateWasCloned) {
            pendingEditorState._selection = internalCreateSelection(editor);
        }

        const startingCompositionKey = editor._compositionKey;
        updateFn();
        skipTransforms = processNestedUpdates(editor, skipTransforms);
        applySelectionTransforms(pendingEditorState, editor);

        if (editor._dirtyType !== NO_DIRTY_NODES) {
            if (skipTransforms) {
                $normalizeAllDirtyTextNodes(pendingEditorState, editor);
            } else {
                $applyAllTransforms(pendingEditorState, editor);
            }

            processNestedUpdates(editor);
            if (currentEditorState) {
                $garbageCollectDetachedNodes(
                    currentEditorState,
                    pendingEditorState,
                    editor._dirtyLeaves,
                    editor._dirtyElements,
                );
            }
        }

        const endingCompositionKey = editor._compositionKey;

        if (startingCompositionKey !== endingCompositionKey) {
            pendingEditorState._flushSync = true;
        }

        const pendingSelection = pendingEditorState._selection;

        if ($isRangeSelection(pendingSelection)) {
            const pendingNodeMap = pendingEditorState._nodeMap;
            const anchorKey = pendingSelection.anchor.key;
            const focusKey = pendingSelection.focus.key;

            if (
                pendingNodeMap.get(anchorKey) === undefined ||
                pendingNodeMap.get(focusKey) === undefined
            ) {
                throw new Error("updateEditor: selection has been lost because the previously selected nodes have been removed and selection wasn't moved to another node. Ensure selection changes after removing/replacing a selected node.");
            }
        } else if ($isNodeSelection(pendingSelection)) {
            // TODO: we should also validate node selection?
            if (pendingSelection._nodes.size === 0) {
                pendingEditorState._selection = null;
            }
        }
    } catch (error) {
        // Report errors
        if (error instanceof Error) {
            editor._onError(error);
        }

        // Restore existing editor state to the DOM
        editor._pendingEditorState = currentEditorState;
        editor._dirtyType = FULL_RECONCILE;

        editor._cloneNotNeeded.clear();

        editor._dirtyLeaves = new Set();

        editor._dirtyElements.clear();

        commitPendingUpdates(editor);
        return;
    } finally {
        activeEditorState = previousActiveEditorState;
        isReadOnlyMode = previousReadOnlyMode;
        activeEditor = previousActiveEditor;
        editor._updating = previouslyUpdating;
        infiniteTransformCount = 0;
    }

    const shouldUpdate =
        editor._dirtyType !== NO_DIRTY_NODES ||
        editorStateHasDirtySelection(pendingEditorState, editor);

    if (shouldUpdate) {
        if (pendingEditorState._flushSync) {
            pendingEditorState._flushSync = false;
            commitPendingUpdates(editor);
        } else if (editorStateWasCloned) {
            scheduleMicroTask(() => {
                commitPendingUpdates(editor);
            });
        }
    } else {
        pendingEditorState._flushSync = false;

        if (editorStateWasCloned) {
            updateTags.clear();
            editor._deferred = [];
            editor._pendingEditorState = null;
        }
    }
}

export function updateEditor(
    editor: LexicalEditor,
    updateFn: () => void,
    options?: EditorUpdateOptions,
) {
    if (editor._updating) {
        editor._updates.push([updateFn, options]);
    } else {
        beginUpdate(editor, updateFn, options);
    }
}

export function errorOnReadOnly(): void {
    if (isReadOnlyMode) {
        throw new Error("Cannot use method in read-only mode.");
    }
}

const TRANSFORMS_BEFORE_INFINITE_LOOP = 100;

export function errorOnInfiniteTransforms(): void {
    if (infiniteTransformCount > TRANSFORMS_BEFORE_INFINITE_LOOP) {
        throw new Error("One or more transforms are endlessly triggering additional transforms. May have encountered infinite recursion caused by transforms that have their preconditions too lose and/or conflict with each other.");
    }
}
