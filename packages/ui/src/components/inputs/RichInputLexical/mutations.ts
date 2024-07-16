/* eslint-disable @typescript-eslint/ban-ts-comment */
import { DOM_TEXT_TYPE, IS_FIREFOX, TEXT_MUTATION_VARIANCE } from "./consts";
import { LexicalEditor } from "./editor";
import { type RootNode } from "./nodes/RootNode";
import { type TextNode } from "./nodes/TextNode";
import { BaseSelection, CustomDomElement } from "./types";
import { updateEditor } from "./updates";
import { $getNearestNodeFromDOMNode, $getSelection, $isNode, $isRangeSelection, $setSelection, $updateTextNodeFromDOMContent, getDOMSelection, getNodeFromDOMNode, getWindow, isAttachedToRoot, isFirefoxClipboardEvents } from "./utils";

let isProcessingMutations = false;
let lastTextEntryTimeStamp = 0;

export const getIsProcessingMutations = (): boolean => {
    return isProcessingMutations;
};

const updateTimeStamp = (event: Event) => {
    lastTextEntryTimeStamp = event.timeStamp;
};

const initTextEntryListener = (editor: LexicalEditor) => {
    if (lastTextEntryTimeStamp === 0) {
        getWindow(editor).addEventListener("textInput", updateTimeStamp, true);
    }
};

export const initMutationObserver = (editor: LexicalEditor) => {
    initTextEntryListener(editor);
    editor._observer = new MutationObserver(
        (mutations: MutationRecord[], observer: MutationObserver) => {
            $flushMutations(editor, mutations, observer);
        },
    );
};

const isManagedLineBreak = (
    dom: Node,
    target: Node,
    editor: LexicalEditor,
): boolean => {
    return (
        (target as CustomDomElement).__lexicalLineBreak === dom ||
        (dom as CustomDomElement)[`__lexicalKey_${editor._key}`] !== undefined
    );
};

const getLastSelection = (editor: LexicalEditor): null | BaseSelection => {
    return editor.getEditorState()?.read(() => {
        const selection = $getSelection();
        return selection !== null ? selection.clone() : null;
    }) ?? null;
};

const shouldUpdateTextNodeFromMutation = (
    selection: null | BaseSelection,
    targetDOM: Node,
    targetNode: TextNode,
): boolean => {
    if ($isRangeSelection(selection)) {
        const anchorNode = selection.anchor.getNode();
        if (
            anchorNode.__key === targetNode.__key &&
            selection.format !== anchorNode.getFormat()
        ) {
            return false;
        }
    }
    return targetDOM.nodeType === DOM_TEXT_TYPE && isAttachedToRoot(targetNode);
};

const handleTextMutation = (
    target: Text,
    node: TextNode,
    editor: LexicalEditor,
): void => {
    const domSelection = getDOMSelection(editor._window);
    let anchorOffset: number | null = null;
    let focusOffset: number | null = null;

    if (domSelection !== null && domSelection.anchorNode === target) {
        anchorOffset = domSelection.anchorOffset;
        focusOffset = domSelection.focusOffset;
    }

    const text = target.nodeValue;
    if (text !== null) {
        $updateTextNodeFromDOMContent(node, text, anchorOffset, focusOffset, false);
    }
};

export function $flushMutations(
    editor: LexicalEditor,
    mutations: MutationRecord[],
    observer: MutationObserver,
): void {
    isProcessingMutations = true;
    const shouldFlushTextMutations = performance.now() - lastTextEntryTimeStamp > TEXT_MUTATION_VARIANCE;

    try {
        updateEditor(editor, () => {
            const selection = $getSelection() || getLastSelection(editor);
            const badDOMTargets = new Map();
            const rootElement = editor.getRootElement();
            // We use the current editor state, as that reflects what is
            // actually "on screen".
            const currentEditorState = editor._editorState;
            const blockCursorElement = editor._blockCursorElement;
            let shouldRevertSelection = false;
            let possibleTextForFirefoxPaste = "";

            for (let i = 0; i < mutations.length; i++) {
                const mutation = mutations[i];
                const type = mutation.type;
                const targetDOM = mutation.target;
                let targetNode = $getNearestNodeFromDOMNode(
                    targetDOM,
                    currentEditorState,
                );

                if (
                    (targetNode === null && targetDOM !== rootElement) ||
                    $isNode("Decorator", targetNode)
                ) {
                    continue;
                }

                if (type === "characterData") {
                    // Text mutations are deferred and passed to mutation listeners to be
                    // processed outside of the Lexical engine.
                    if (
                        shouldFlushTextMutations &&
                        $isNode("Text", targetNode) &&
                        shouldUpdateTextNodeFromMutation(selection, targetDOM, targetNode)
                    ) {
                        handleTextMutation(
                            // nodeType === DOM_TEXT_TYPE is a Text DOM node
                            targetDOM as Text,
                            targetNode,
                            editor,
                        );
                    }
                } else if (type === "childList") {
                    shouldRevertSelection = true;
                    // We attempt to "undo" any changes that have occurred outside
                    // of Lexical. We want Lexical's editor state to be source of truth.
                    // To the user, these will look like no-ops.
                    const addedDOMs = mutation.addedNodes;

                    for (let s = 0; s < addedDOMs.length; s++) {
                        const addedDOM = addedDOMs[s];
                        const node = getNodeFromDOMNode(addedDOM);
                        const parentDOM = addedDOM.parentNode;

                        if (
                            parentDOM !== null &&
                            addedDOM !== blockCursorElement &&
                            node === null &&
                            (addedDOM.nodeName !== "BR" ||
                                !isManagedLineBreak(addedDOM, parentDOM, editor))
                        ) {
                            if (IS_FIREFOX) {
                                const possibleText =
                                    (addedDOM as HTMLElement).innerText || addedDOM.nodeValue;

                                if (possibleText) {
                                    possibleTextForFirefoxPaste += possibleText;
                                }
                            }

                            parentDOM.removeChild(addedDOM);
                        }
                    }

                    const removedDOMs = mutation.removedNodes;
                    const removedDOMsLength = removedDOMs.length;

                    if (removedDOMsLength > 0) {
                        let unremovedBRs = 0;

                        for (let s = 0; s < removedDOMsLength; s++) {
                            const removedDOM = removedDOMs[s];

                            if (
                                (removedDOM.nodeName === "BR" &&
                                    isManagedLineBreak(removedDOM, targetDOM, editor)) ||
                                blockCursorElement === removedDOM
                            ) {
                                targetDOM.appendChild(removedDOM);
                                unremovedBRs++;
                            }
                        }

                        if (removedDOMsLength !== unremovedBRs) {
                            if (targetDOM === rootElement) {
                                targetNode = (currentEditorState?._nodeMap.get("root") ?? null) as RootNode | null;
                            }

                            badDOMTargets.set(targetDOM, targetNode);
                        }
                    }
                }
            }

            // Now we process each of the unique target nodes, attempting
            // to restore their contents back to the source of truth, which
            // is Lexical's "current" editor state. This is basically like
            // an internal revert on the DOM.
            if (badDOMTargets.size > 0) {
                for (const [targetDOM, targetNode] of badDOMTargets) {
                    if ($isNode("Element", targetNode)) {
                        const childKeys = targetNode.getChildrenKeys();
                        let currentDOM = targetDOM.firstChild;

                        for (let s = 0; s < childKeys.length; s++) {
                            const key = childKeys[s];
                            const correctDOM = editor.getElementByKey(key);

                            if (correctDOM === null) {
                                continue;
                            }

                            if (currentDOM === null) {
                                targetDOM.appendChild(correctDOM);
                                currentDOM = correctDOM;
                            } else if (currentDOM !== correctDOM) {
                                targetDOM.replaceChild(correctDOM, currentDOM);
                            }

                            currentDOM = currentDOM.nextSibling;
                        }
                    } else if ($isNode("Text", targetNode)) {
                        targetNode.markDirty();
                    }
                }
            }

            // Capture all the mutations made during this function. This
            // also prevents us having to process them on the next cycle
            // of onMutation, as these mutations were made by us.
            const records = observer.takeRecords();

            // Check for any random auto-added <br> elements, and remove them.
            // These get added by the browser when we undo the above mutations
            // and this can lead to a broken UI.
            if (records.length > 0) {
                for (let i = 0; i < records.length; i++) {
                    const record = records[i];
                    const addedNodes = record.addedNodes;
                    const target = record.target;

                    for (let s = 0; s < addedNodes.length; s++) {
                        const addedDOM = addedNodes[s];
                        const parentDOM = addedDOM.parentNode;

                        if (
                            parentDOM !== null &&
                            addedDOM.nodeName === "BR" &&
                            !isManagedLineBreak(addedDOM, target, editor)
                        ) {
                            parentDOM.removeChild(addedDOM);
                        }
                    }
                }

                // Clear any of those removal mutations
                observer.takeRecords();
            }

            if (selection !== null) {
                if (shouldRevertSelection) {
                    selection.dirty = true;
                    $setSelection(selection);
                }

                if (IS_FIREFOX && isFirefoxClipboardEvents(editor)) {
                    selection.insertRawText(possibleTextForFirefoxPaste);
                }
            }
        });
    } finally {
        isProcessingMutations = false;
    }
}

export const flushRootMutations = (editor: LexicalEditor): void => {
    const observer = editor._observer;

    if (observer !== null) {
        const mutations = observer.takeRecords();
        $flushMutations(editor, mutations, observer);
    }
};
