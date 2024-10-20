/* eslint-disable @typescript-eslint/ban-ts-comment */
// NOTE: Much of this file is taken from the lexical package, which is licensed under the MIT license. 
// We have copied the code here for customization purposes, such as replacing the default code block component

import { uuid } from "@local/shared";
import { FULL_RECONCILE, NO_DIRTY_NODES } from "./consts";
import { addRootElementEvents, removeRootElementEvents } from "./events";
import { flushRootMutations, initMutationObserver } from "./mutations";
import { LexicalNodes } from "./nodes";
import { type LexicalNode } from "./nodes/LexicalNode";
import { BaseSelection, CommandListener, CommandListenerPriority, CommandPayloadType, CommandsMap, CreateEditorArgs, EditorConfig, EditorFocusOptions, EditorListener, EditorListeners, EditorSetOptions, EditorUpdateOptions, ErrorHandler, IntentionallyMarkedAsDirtyElement, LexicalCommand, NodeKey, NodeMap, NodeType, RegisteredNodes, SerializedEditor, SerializedEditorState, SerializedElementNode, SerializedLexicalNode, Transform } from "./types";
import { commitPendingUpdates, dispatchCommand, parseEditorState, readEditorState, setActiveEditor, triggerListeners, updateEditor } from "./updates";
import { $createNode, $getRoot, $getSelection, $isNode, getDOMSelection, getDefaultView, markAllNodesAsDirty } from "./utils";

function exportNodeToJSON<SerializedNode extends SerializedLexicalNode>(
    node: LexicalNode,
): SerializedNode {
    const serializedNode = node.exportJSON();

    if (serializedNode.__type !== node.getType()) {
        throw new Error(`LexicalNode: Node ${node.getType()} does not match the serialized type. Check if .exportJSON() is implemented and it is returning the correct type.`);
    }

    if ($isNode("Element", node)) {
        const serializedChildren = (serializedNode as SerializedElementNode).children;
        if (!Array.isArray(serializedChildren)) {
            throw new Error(`LexicalNode: Node ${node.getType()} is an element but .exportJSON() does not have a children array.`);
        }

        const children = node.getChildren();

        for (let i = 0; i < children.length; i++) {
            const child = children[i];
            const serializedChildNode = exportNodeToJSON(child);
            serializedChildren.push(serializedChildNode);
        }
    }

    return serializedNode as SerializedNode;
}

export class EditorState {
    _nodeMap: NodeMap;
    _selection: null | BaseSelection;
    _flushSync: boolean;
    _readOnly: boolean;

    constructor(nodeMap: NodeMap, selection?: null | BaseSelection) {
        this._nodeMap = nodeMap;
        this._selection = selection || null;
        this._flushSync = false;
        this._readOnly = false;
    }

    isEmpty(): boolean {
        return this._nodeMap.size === 0;
    }

    read<V>(callbackFn: () => V): V {
        return readEditorState(this, callbackFn);
    }

    clone(selection?: null | BaseSelection): EditorState {
        const editorState = new EditorState(
            new Map(this._nodeMap),
            selection === undefined ? this._selection : selection,
        );

        return editorState;
    }
    toJSON(): SerializedEditorState {
        return readEditorState(this, () => ({
            root: exportNodeToJSON($getRoot()),
        }));
    }
}

/**
 * Creates a new EditorState with a single root node.
 */
export function createEmptyEditorState(): EditorState {
    const rootKey = "root" as const;
    return new EditorState(new Map([[rootKey, $createNode("Root", { key: rootKey })]]));
}

export function resetEditor(
    editor: LexicalEditor,
    prevRootElement: null | HTMLElement,
    nextRootElement: null | HTMLElement,
    pendingEditorState: EditorState,
): void {
    const keyNodeMap = editor._keyToDOMMap;
    keyNodeMap.clear();
    editor._editorState = createEmptyEditorState();
    editor._pendingEditorState = pendingEditorState;
    editor._compositionKey = null;
    editor._dirtyType = NO_DIRTY_NODES;
    editor._cloneNotNeeded.clear();
    editor._dirtyLeaves = new Set();
    editor._dirtyElements.clear();
    editor._normalizedNodes = new Set();
    editor._updateTags = new Set();
    editor._updates = [];
    editor._blockCursorElement = null;

    const observer = editor._observer;

    if (observer !== null) {
        observer.disconnect();
        editor._observer = null;
    }

    // Remove all the DOM nodes from the root element
    if (prevRootElement !== null) {
        prevRootElement.textContent = "";
    }

    if (nextRootElement !== null) {
        nextRootElement.textContent = "";
        keyNodeMap.set("root", nextRootElement);
    }
}

/**
 * Creates a new LexicalEditor attached to a single contentEditable (provided in the config). This is
 * the lowest-level initialization API for a LexicalEditor. If you're using React or another framework,
 * consider using the appropriate abstractions, such as LexicalComposer
 * @param editorConfig - the editor configuration.
 * @returns a LexicalEditor instance
 */
export async function createEditor(editorConfig?: CreateEditorArgs): Promise<LexicalEditor> {
    // Make sure node information is loaded before creating the editor
    await LexicalNodes.init();

    // Return active editor if it exists
    const config = editorConfig || {};
    const namespace = config.namespace || uuid();

    const editor = new LexicalEditor(
        { namespace },
        console.error,
    );
    setActiveEditor(editor);
    const editorState = createEmptyEditorState();
    editor.setEditorState(editorState);

    return editor;
}

export class LexicalEditor {
    /**
     * The root element associated with this editor, as elements are 
     * stored in a tree structure.
     */
    _rootElement: null | HTMLElement;
    /** The state of the editor */
    _editorState: EditorState | null;
    /**
     * Map of registered nodes, by node type. Allows you to attach additional logic to nodes,
     * such as triggering a transform when a node is marked dirty.
     */
    _nodes: RegisteredNodes;
    /** Handles drafts and updates */
    _pendingEditorState: null | EditorState;
    /** Helps co-ordinate selection and events */
    _compositionKey: null | NodeKey;
    _deferred: (() => void)[];
    /** Used during reconciliation */
    _keyToDOMMap: Map<NodeKey, HTMLElement>;
    _updates: [() => void, EditorUpdateOptions | undefined][];
    _updating: boolean;
    _listeners: EditorListeners;
    _commands: CommandsMap;
    _decorators: Record<NodeKey, unknown>;
    _pendingDecorators: null | Record<NodeKey, unknown>;
    _config: EditorConfig;
    _dirtyType: 0 | 1 | 2;
    _cloneNotNeeded: Set<NodeKey>;
    _dirtyLeaves: Set<NodeKey>;
    _dirtyElements: Map<NodeKey, IntentionallyMarkedAsDirtyElement>;
    _normalizedNodes: Set<NodeKey>;
    _updateTags: Set<string>;
    _observer: null | MutationObserver;
    _key: string;
    _onError: ErrorHandler;
    _window: null | Window;
    _editable: boolean;
    _blockCursorElement: null | HTMLDivElement;

    /** @internal */
    constructor(
        config: EditorConfig,
        onError: ErrorHandler,
    ) {
        this._rootElement = null;
        this._editorState = null;
        this._nodes = Object.fromEntries(Object.entries(LexicalNodes.getAll() ?? {}).map(([key, klass]) => [key, { klass, transforms: new Set() }])) as unknown as RegisteredNodes;
        this._pendingEditorState = null;
        this._compositionKey = null;
        this._deferred = [];
        this._keyToDOMMap = new Map();
        this._updates = [];
        this._updating = false;
        this._listeners = {};
        // Commands
        this._commands = new Map();
        // Editor configuration for theme/context.
        this._config = config;
        // React node decorators for portals
        this._decorators = {};
        this._pendingDecorators = null;
        // Used to optimize reconciliation
        this._dirtyType = NO_DIRTY_NODES;
        this._cloneNotNeeded = new Set();
        this._dirtyLeaves = new Set();
        this._dirtyElements = new Map();
        this._normalizedNodes = new Set();
        this._updateTags = new Set();
        // Handling of DOM mutations
        this._observer = null;
        // Used for identifying owning editors
        this._key = uuid();

        this._onError = onError;
        this._editable = true;
        this._window = null;
        this._blockCursorElement = null;
    }

    /**
     *
     * @returns true if the editor is currently in "composition" mode due to receiving input
     * through an IME, or 3P extension, for example. Returns false otherwise.
     */
    isComposing(): boolean {
        return this._compositionKey !== null;
    }
    /**
     * Registers a listener for an editor event.
     * @param event - the event to listen for.
     * @param listener - the function to run when the event is triggered.
     * @returns a teardown function that can be used to cleanup the listener.
     */
    registerListener<T extends keyof EditorListeners>(
        event: T,
        listener: NonNullable<EditorListener[T]>,
    ): () => void {
        // All node class listeners are stored in the mutation map. 
        // Other listeners are stored in their own sets.
        let listenerSet = this._listeners[event];
        // If set has not been created, create it.
        if (!listenerSet) {
            this._listeners[event] = new Set() as EditorListeners[T];
            listenerSet = this._listeners[event];
        }
        // Add the listener to the set.
        listenerSet!.add(listener);
        return () => {
            listenerSet!.delete(listener);
        };
    }
    /**
     * Registers a listener that will trigger anytime the provided command
     * is dispatched, subject to priority. Listeners that run at a higher priority can "intercept"
     * commands and prevent them from propagating to other handlers by returning true.
     *
     * Listeners registered at the same priority level will run deterministically in the order of registration.
     *
     * @param command - the command that will trigger the callback.
     * @param listener - the function that will execute when the command is dispatched.
     * @param priority - the relative priority of the listener. 0 | 1 | 2 | 3 | 4
     * @returns a teardown function that can be used to cleanup the listener.
     */
    registerCommand<P>(
        command: LexicalCommand<P>,
        listener: CommandListener<P>,
        priority: CommandListenerPriority,
    ): () => void {
        const commandsMap = this._commands;

        if (!commandsMap.has(command)) {
            commandsMap.set(command, [
                new Set(),
                new Set(),
                new Set(),
                new Set(),
                new Set(),
            ]);
        }

        const listenersInPriorityOrder = commandsMap.get(command);

        if (listenersInPriorityOrder === undefined) {
            throw new Error(`registerCommand: Command ${String(command)} not found in command map`);
        }

        const listeners = listenersInPriorityOrder[priority];
        listeners.add(listener as CommandListener<unknown>);
        return () => {
            listeners.delete(listener as CommandListener<unknown>);

            if (
                listenersInPriorityOrder.every(
                    (listenersSet) => listenersSet.size === 0,
                )
            ) {
                commandsMap.delete(command);
            }
        };
    }

    /**
     * Registers a listener that will run when a Lexical node of the provided class is
     * marked dirty during an update. The listener will continue to run as long as the node
     * is marked dirty. There are no guarantees around the order of transform execution!
     *
     * Watch out for infinite loops. See [Node Transforms](https://lexical.dev/docs/concepts/transforms)
     * @param nodeType - The type of node that you want to run transforms on.
     * @param listener - The logic you want to run when the node is updated.
     * @returns a teardown function that can be used to cleanup the listener.
     */
    registerNodeTransform<T extends LexicalNode>(
        nodeType: NodeType,
        listener: Transform<T>,
    ): () => void {
        const registeredNode = this._nodes[nodeType];
        if (!registeredNode) {
            this._onError(new Error(`Node ${nodeType} has not been registered. Ensure node has been passed to createEditor.`));
            // eslint-disable-next-line @typescript-eslint/no-empty-function
            return () => { };
        }
        registeredNode.transforms.add(listener as Transform<LexicalNode>);

        markAllNodesAsDirty(this, nodeType);
        return () => {
            registeredNode.transforms.delete(listener as Transform<LexicalNode>);
        };
    }

    /**
     * Dispatches a command of the specified type with the specified payload.
     * This triggers all command listeners (set by {@link LexicalEditor.registerCommand})
     * for this type, passing them the provided payload.
     * @param type - the type of command listeners to trigger.
     * @param payload - the data to pass as an argument to the command listeners.
     */
    dispatchCommand<TCommand extends LexicalCommand<unknown>>(
        type: TCommand,
        payload: CommandPayloadType<TCommand>,
    ): boolean {
        return dispatchCommand(this, type, payload);
    }

    /**
     * Gets a map of all decorators in the editor.
     * @returns A mapping of call decorator keys to their decorated content
     */
    getDecorators<T>(): Record<NodeKey, T> {
        return this._decorators as Record<NodeKey, T>;
    }

    /**
     *
     * @returns the current root element of the editor. If you want to register
     * an event listener, do it via {@link LexicalEditor.registerListener}, since
     * this reference may not be stable.
     */
    getRootElement(): null | HTMLElement {
        return this._rootElement;
    }

    /**
     * Imperatively set the root contenteditable element that Lexical listens
     * for events on.
     */
    setRootElement(nextRootElement: null | HTMLElement): void {
        const prevRootElement = this._rootElement;

        if (nextRootElement !== prevRootElement) {
            const pendingEditorState = this._pendingEditorState || this._editorState;
            if (!pendingEditorState) {
                throw new Error("setRootElement: editor state is null. Ensure the editor has been initialized.");
            }
            this._rootElement = nextRootElement;
            resetEditor(this, prevRootElement, nextRootElement, pendingEditorState);

            if (prevRootElement) {
                if (this._editable) {
                    removeRootElementEvents(prevRootElement);
                }
            }

            if (nextRootElement) {
                const windowObj = getDefaultView(nextRootElement);
                const style = nextRootElement.style;
                style.userSelect = "text";
                style.whiteSpace = "pre-wrap";
                style.wordBreak = "break-word";
                nextRootElement.setAttribute("data-lexical-editor", "true");
                this._window = windowObj;
                this._dirtyType = FULL_RECONCILE;
                initMutationObserver(this);

                this._updateTags.add("history-merge");

                commitPendingUpdates(this);

                if (this._editable) {
                    addRootElementEvents(nextRootElement, this);
                }
            } else {
                // If content editable is unmounted we'll reset editor state back to original
                // (or pending) editor state since there will be no reconciliation
                this._editorState = pendingEditorState;
                this._pendingEditorState = null;
                this._window = null;
            }

            triggerListeners("root", this, false, {
                prevRootElement,
                rootElement: nextRootElement,
            });
        }
    }

    /**
     * Gets the underlying HTMLElement associated with the LexicalNode for the given key.
     * @returns the HTMLElement rendered by the LexicalNode associated with the key.
     * @param key - the key of the LexicalNode.
     */
    getElementByKey(key: NodeKey): HTMLElement | null {
        return this._keyToDOMMap.get(key) || null;
    }

    /**
     * Gets the active editor state.
     * @returns The editor state
     */
    getEditorState(): EditorState | null {
        return this._editorState;
    }

    /**
     * Imperatively set the EditorState. Triggers reconciliation like an update.
     * @param editorState - the state to set the editor
     * @param options - options for the update.
     */
    setEditorState(editorState: EditorState, options?: EditorSetOptions): void {
        if (editorState.isEmpty()) {
            throw new Error("setEditorState: the editor state is empty. It should at least contain a root node.");
        }

        flushRootMutations(this);
        const pendingEditorState = this._pendingEditorState;
        const tags = this._updateTags;
        const tag = options !== undefined ? options.tag : null;

        if (pendingEditorState && !pendingEditorState.isEmpty()) {
            if (tag) {
                tags.add(tag);
            }

            commitPendingUpdates(this);
        }

        this._pendingEditorState = editorState;
        this._dirtyType = FULL_RECONCILE;
        this._dirtyElements.set("root", false);
        this._compositionKey = null;

        if (tag) {
            tags.add(tag);
        }

        commitPendingUpdates(this);
    }

    /**
     * Parses a SerializedEditorState (usually produced by {@link EditorState.toJSON}) and returns
     * and EditorState object that can be, for example, passed to {@link LexicalEditor.setEditorState}. Typically,
     * deserliazation from JSON stored in a database uses this method.
     * @param maybeStringifiedEditorState
     * @param updateFn
     * @returns
     */
    parseEditorState(
        maybeStringifiedEditorState: string | SerializedEditorState,
        updateFn?: () => void,
    ): EditorState {
        const serializedEditorState =
            typeof maybeStringifiedEditorState === "string"
                ? JSON.parse(maybeStringifiedEditorState)
                : maybeStringifiedEditorState;
        return parseEditorState(serializedEditorState, this, updateFn);
    }

    /**
     * Executes an update to the editor state. The updateFn callback is the ONLY place
     * where Lexical editor state can be safely mutated.
     * @param updateFn - A function that has access to writable editor state.
     * @param options - A bag of options to control the behavior of the update.
     * @param options.onUpdate - A function to run once the update is complete.
     * Useful for synchronizing updates in some cases.
     * @param options.skipTransforms - Setting this to true will suppress all node
     * transforms for this update cycle.
     * @param options.tag - A tag to identify this update, in an update listener, for instance.
     * Some tags are reserved by the core and control update behavior in different ways.
     * @param options.discrete - If true, prevents this update from being batched, forcing it to
     * run synchronously.
     */
    update(updateFn: () => void, options?: EditorUpdateOptions): void {
        updateEditor(this, updateFn, options);
    }

    /**
     * Focuses the editor
     * @param callbackFn - A function to run after the editor is focused.
     * @param options - A bag of options
     * @param options.defaultSelection - Where to move selection when the editor is
     * focused. Can be rootStart, rootEnd, or undefined. Defaults to rootEnd.
     */
    focus(callbackFn?: () => void, options: EditorFocusOptions = {}): void {
        const rootElement = this._rootElement;
        if (!rootElement) return;

        // This ensures that iOS does not trigger caps lock upon focus
        rootElement.setAttribute("autocapitalize", "off");
        updateEditor(
            this,
            () => {
                const selection = $getSelection();
                const root = $getRoot();

                if (selection) {
                    // Marking the selection dirty will force the selection back to it
                    selection.dirty = true;
                } else if (root.getChildrenSize() !== 0) {
                    if (options.defaultSelection === "rootStart") {
                        root.selectStart();
                    } else {
                        root.selectEnd();
                    }
                }
            },
            {
                onUpdate: () => {
                    rootElement.removeAttribute("autocapitalize");
                    if (callbackFn) {
                        callbackFn();
                    }
                },
                tag: "focus",
            },
        );
        // In the case where onUpdate doesn't fire (due to the focus update not
        // occuring).
        if (!this._pendingEditorState) {
            rootElement.removeAttribute("autocapitalize");
        }
    }

    /**
     * Removes focus from the editor.
     */
    blur(): void {
        const rootElement = this._rootElement;

        if (rootElement !== null) {
            rootElement.blur();
        }

        const domSelection = getDOMSelection(this._window);

        if (domSelection !== null) {
            domSelection.removeAllRanges();
        }
    }
    /**
     * Returns true if the editor is editable, false otherwise.
     * @returns True if the editor is editable, false otherwise.
     */
    isEditable(): boolean {
        return this._editable;
    }
    /**
     * Sets the editable property of the editor. When false, the
     * editor will not listen for user events on the underling contenteditable.
     * @param editable - the value to set the editable mode to.
     */
    setEditable(editable: boolean): void {
        if (this._editable !== editable) {
            this._editable = editable;
            triggerListeners("editable", this, true, editable);
        }
    }
    /**
     * Returns a JSON-serializable javascript object NOT a JSON string.
     * You still must call JSON.stringify (or something else) to turn the
     * state into a string you can transfer over the wire and store in a database.
     *
     * See {@link LexicalNode.exportJSON}
     *
     * @returns A JSON-serializable javascript object
     */
    toJSON(): SerializedEditor {
        return {
            editorState: this._editorState?.toJSON() ?? { root: { __type: "Root", version: 1, children: [], direction: null, format: "", indent: 0 } },
        };
    }
}

export function editorStateHasDirtySelection(
    editorState: EditorState,
    editor: LexicalEditor,
): boolean {
    const currentSelection = editor.getEditorState()?._selection;

    const pendingSelection = editorState._selection;

    // Check if we need to update because of changes in selection
    if (pendingSelection !== null) {
        if (pendingSelection.dirty || !currentSelection || !pendingSelection.is(currentSelection)) {
            return true;
        }
    } else if (currentSelection !== null) {
        return true;
    }

    return false;
}
