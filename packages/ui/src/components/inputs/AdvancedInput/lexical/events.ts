/* eslint-disable @typescript-eslint/ban-ts-comment */
import { BLUR_COMMAND, CLICK_COMMAND, CONTROLLED_TEXT_INSERTION_COMMAND, COPY_COMMAND, CUT_COMMAND, DELETE_CHARACTER_COMMAND, DELETE_LINE_COMMAND, DELETE_WORD_COMMAND, DRAGEND_COMMAND, DRAGOVER_COMMAND, DRAGSTART_COMMAND, DROP_COMMAND, FOCUS_COMMAND, FORMAT_TEXT_COMMAND, INSERT_LINE_BREAK_COMMAND, KEY_ARROW_DOWN_COMMAND, KEY_ARROW_LEFT_COMMAND, KEY_ARROW_RIGHT_COMMAND, KEY_ARROW_UP_COMMAND, KEY_BACKSPACE_COMMAND, KEY_DELETE_COMMAND, KEY_ENTER_COMMAND, KEY_ESCAPE_COMMAND, KEY_MODIFIER_COMMAND, KEY_SPACE_COMMAND, KEY_TAB_COMMAND, MOVE_TO_END, MOVE_TO_START, PASTE_COMMAND, REDO_COMMAND, SELECTION_CHANGE_COMMAND, SELECT_ALL_COMMAND, UNDO_COMMAND } from "./commands.js";
import { ANDROID_COMPOSITION_LATENCY, CAN_USE_BEFORE_INPUT, COMPOSITION_START_CHAR, DOM_ELEMENT_TYPE, DOM_TEXT_TYPE, IS_APPLE_WEBKIT, IS_FIREFOX, IS_IOS, IS_SAFARI } from "./consts.js";
import { type LexicalEditor } from "./editor.js";
import { flushRootMutations } from "./mutations.js";
import { $getPreviousSelection, type RangeSelection, internalCreateRangeSelection } from "./selection.js";
import { type CustomDomElement, type CustomLexicalEvent, type NodeKey, type RootElementEvents, type RootElementRemoveHandles } from "./types.js";
import { dispatchCommand, errorOnReadOnly, getActiveEditor, updateEditor } from "./updates.js";
import { $getNodeByKey, $getRoot, $getSelection, $isNode, $isNodeSelection, $isRangeSelection, $isSelectionCapturedInDecorator, $isTokenOrSegmented, $setCompositionKey, $setSelection, $shouldInsertTextAfterOrBeforeTextNode, $updateSelectedTextFromDOM, $updateTextNodeFromDOMContent, IS_KEY, doesContainGrapheme, getAnchorTextFromDOM, getDOMSelection, getDOMTextNode, getNearestEditorFromDOMNode, getParent, getTopLevelElementOrThrow, getWindow, isSelectionWithinEditor } from "./utils.js";

let lastKeyDownTimeStamp = 0;
const lastBeforeInputInsertTextTimeStamp = 0;
const rootElementsRegistered = new WeakMap<Document, number>();
let isSelectionChangeFromDOMUpdate = false;
let isSelectionChangeFromMouseDown = false;
let isFirefoxEndingComposition = false;
let collapsedSelectionFormat: [number, string, number, NodeKey, number] = [
    0,
    "",
    0,
    "root",
    0,
];
// Mapping root editors to their active nested editors, contains nested editors
// mapping only, so if root editor is selected map will have no reference to free up memory
const activeNestedEditorsMap: Map<string, LexicalEditor> = new Map();

function onKeyDown(event: KeyboardEvent, editor: LexicalEditor): void {
    lastKeyDownTimeStamp = event.timeStamp;
    if (editor.isComposing()) {
        return;
    }

    const { key } = event;

    if (IS_KEY.moveForward(event)) {
        dispatchCommand(editor, KEY_ARROW_RIGHT_COMMAND, event);
    } else if (IS_KEY.moveToEnd(event)) {
        dispatchCommand(editor, MOVE_TO_END, event);
    } else if (IS_KEY.moveBackward(event)) {
        dispatchCommand(editor, KEY_ARROW_LEFT_COMMAND, event);
    } else if (IS_KEY.moveToStart(event)) {
        dispatchCommand(editor, MOVE_TO_START, event);
    } else if (IS_KEY.moveUp(event)) {
        dispatchCommand(editor, KEY_ARROW_UP_COMMAND, event);
    } else if (IS_KEY.moveDown(event)) {
        dispatchCommand(editor, KEY_ARROW_DOWN_COMMAND, event);
    } else if (IS_KEY.lineBreak(event)) {
        dispatchCommand(editor, KEY_ENTER_COMMAND, event);
    } else if (IS_KEY.space(event)) {
        dispatchCommand(editor, KEY_SPACE_COMMAND, event);
    } else if (IS_KEY.openLineBreak(event)) {
        event.preventDefault();
        dispatchCommand(editor, INSERT_LINE_BREAK_COMMAND, true);
    } else if (IS_KEY.paragraph(event)) {
        dispatchCommand(editor, KEY_ENTER_COMMAND, event);
    } else if (IS_KEY.deleteBackward(event)) {
        if (IS_KEY.backspace(event)) {
            dispatchCommand(editor, KEY_BACKSPACE_COMMAND, event);
        } else {
            event.preventDefault();
            dispatchCommand(editor, DELETE_CHARACTER_COMMAND, true);
        }
    } else if (IS_KEY.escape(event)) {
        dispatchCommand(editor, KEY_ESCAPE_COMMAND, event);
    } else if (IS_KEY.deleteForward(event)) {
        if (IS_KEY.delete(event)) {
            dispatchCommand(editor, KEY_DELETE_COMMAND, event);
        } else {
            event.preventDefault();
            dispatchCommand(editor, DELETE_CHARACTER_COMMAND, false);
        }
    } else if (IS_KEY.deleteWordBackward(event)) {
        event.preventDefault();
        dispatchCommand(editor, DELETE_WORD_COMMAND, true);
    } else if (IS_KEY.deleteWordForward(event)) {
        event.preventDefault();
        dispatchCommand(editor, DELETE_WORD_COMMAND, false);
    } else if (IS_KEY.deleteLineBackward(event)) {
        event.preventDefault();
        dispatchCommand(editor, DELETE_LINE_COMMAND, true);
    } else if (IS_KEY.deleteLineForward(event)) {
        event.preventDefault();
        dispatchCommand(editor, DELETE_LINE_COMMAND, false);
    } else if (IS_KEY.bold(event)) {
        event.preventDefault();
        dispatchCommand(editor, FORMAT_TEXT_COMMAND, "BOLD");
    } else if (IS_KEY.underline(event)) {
        event.preventDefault();
        dispatchCommand(editor, FORMAT_TEXT_COMMAND, "UNDERLINE_TAGS");
    } else if (IS_KEY.italic(event)) {
        event.preventDefault();
        dispatchCommand(editor, FORMAT_TEXT_COMMAND, "ITALIC");
    } else if (IS_KEY.tab(event)) {
        dispatchCommand(editor, KEY_TAB_COMMAND, event);
    } else if (IS_KEY.undo(event)) {
        event.preventDefault();
        dispatchCommand(editor, UNDO_COMMAND, undefined);
    } else if (IS_KEY.redo(event)) {
        event.preventDefault();
        dispatchCommand(editor, REDO_COMMAND, undefined);
    } else {
        const prevSelection = editor._editorState?._selection;
        if ($isNodeSelection(prevSelection)) {
            if (IS_KEY.copy(event)) {
                event.preventDefault();
                dispatchCommand(editor, COPY_COMMAND, event);
            } else if (IS_KEY.cut(event)) {
                event.preventDefault();
                dispatchCommand(editor, CUT_COMMAND, event);
            } else if (IS_KEY.selectAll(event)) {
                event.preventDefault();
                dispatchCommand(editor, SELECT_ALL_COMMAND, event);
            }
            // FF does it well (no need to override behavior)
        } else if (!IS_FIREFOX && IS_KEY.selectAll(event)) {
            event.preventDefault();
            dispatchCommand(editor, SELECT_ALL_COMMAND, event);
        }
    }

    if (IS_KEY.modifier(event)) {
        dispatchCommand(editor, KEY_MODIFIER_COMMAND, event);
    }
}

function onPointerDown(event: PointerEvent, editor: LexicalEditor) {
    // TODO implement text drag & drop
    const target = event.target;
    const pointerType = event.pointerType;
    if (target instanceof Node && pointerType !== "touch") {
        updateEditor(editor, () => {
            // Drag & drop should not recompute selection until mouse up; otherwise the initially
            // selected content is lost.
            if (!$isSelectionCapturedInDecorator(target)) {
                isSelectionChangeFromMouseDown = true;
            }
        });
    }
}

function onInput(event: InputEvent, editor: LexicalEditor): void {
    // We don't want the onInput to bubble, in the case of nested editors.
    event.stopPropagation();
    updateEditor(editor, () => {
        const selection = $getSelection();
        const data = event.data;
        const targetRange = getTargetRange(event);

        if (
            data !== null &&
            $isRangeSelection(selection) &&
            $shouldPreventDefaultAndInsertText(
                selection,
                targetRange,
                data,
                event.timeStamp,
                false,
            )
        ) {
            // Given we're over-riding the default behavior, we will need
            // to ensure to disable composition before dispatching the
            // insertText command for when changing the sequence for FF.
            if (isFirefoxEndingComposition) {
                onCompositionEndImpl(editor, data);
                isFirefoxEndingComposition = false;
            }
            const anchor = selection.anchor;
            const anchorNode = anchor.getNode();
            const domSelection = getDOMSelection(editor._window);
            if (domSelection === null) {
                return;
            }
            const offset = anchor.offset;
            // If the content is the same as inserted, then don't dispatch an insertion.
            // Given onInput doesn't take the current selection (it uses the previous)
            // we can compare that against what the DOM currently says.
            if (
                !CAN_USE_BEFORE_INPUT ||
                selection.isCollapsed() ||
                !$isNode("Text", anchorNode) ||
                domSelection.anchorNode === null ||
                anchorNode.getTextContent().slice(0, offset) +
                data +
                anchorNode.getTextContent().slice(offset + selection.focus.offset) !==
                getAnchorTextFromDOM(domSelection.anchorNode)
            ) {
                dispatchCommand(editor, CONTROLLED_TEXT_INSERTION_COMMAND, data);
            }

            const textLength = data.length;

            // Another hack for FF, as it's possible that the IME is still
            // open, even though compositionend has already fired (sigh).
            if (
                IS_FIREFOX &&
                textLength > 1 &&
                event.inputType === "insertCompositionText" &&
                !editor.isComposing()
            ) {
                selection.anchor.offset -= textLength;
            }

            // This ensures consistency on Android.
            if (!IS_SAFARI && !IS_IOS && !IS_APPLE_WEBKIT && editor.isComposing()) {
                lastKeyDownTimeStamp = 0;
                $setCompositionKey(null);
            }
        } else {
            const characterData = data !== null ? data : undefined;
            $updateSelectedTextFromDOM(false, editor, characterData);

            // onInput always fires after onCompositionEnd for FF.
            if (isFirefoxEndingComposition) {
                onCompositionEndImpl(editor, data || undefined);
                isFirefoxEndingComposition = false;
            }
        }

        // Also flush any other mutations that might have occurred
        // since the change.
        errorOnReadOnly();
        const activeEditor = getActiveEditor();
        flushRootMutations(activeEditor);
    });
}

function onCompositionStart(
    event: CompositionEvent,
    editor: LexicalEditor,
) {
    updateEditor(editor, () => {
        const selection = $getSelection();

        if ($isRangeSelection(selection) && !editor.isComposing()) {
            const anchor = selection.anchor;
            const node = selection.anchor.getNode();
            $setCompositionKey(anchor.key);

            if (
                // If it has been 30ms since the last keydown, then we should
                // apply the empty space heuristic. We can't do this for Safari,
                // as the keydown fires after composition start.
                event.timeStamp < lastKeyDownTimeStamp + ANDROID_COMPOSITION_LATENCY ||
                // FF has issues around composing multibyte characters, so we also
                // need to invoke the empty space heuristic below.
                anchor.type === "element" ||
                !selection.isCollapsed() ||
                node.getFormat() !== selection.format ||
                ($isNode("Text", node) && node.getStyle() !== selection.style)
            ) {
                // We insert a zero width character, ready for the composition
                // to get inserted into the new node we create. If
                // we don't do this, Safari will fail on us because
                // there is no text node matching the selection.
                dispatchCommand(
                    editor,
                    CONTROLLED_TEXT_INSERTION_COMMAND,
                    COMPOSITION_START_CHAR,
                );
            }
        }
    });
}

function onCompositionEndImpl(editor: LexicalEditor, data?: string): void {
    const compositionKey = editor._compositionKey;
    $setCompositionKey(null);

    // Handle termination of composition.
    if (compositionKey !== null && data !== null && data !== undefined) {
        // Composition can sometimes move to an adjacent DOM node when backspacing.
        // So check for the empty case.
        if (data === "") {
            const node = $getNodeByKey(compositionKey);
            const textNode = getDOMTextNode(editor.getElementByKey(compositionKey));

            if (
                textNode !== null &&
                textNode.nodeValue !== null &&
                $isNode("Text", node)
            ) {
                $updateTextNodeFromDOMContent(
                    node,
                    textNode.nodeValue,
                    null,
                    null,
                    true,
                );
            }

            return;
        }

        // Composition can sometimes be that of a new line. In which case, we need to
        // handle that accordingly.
        if (data[data.length - 1] === "\n") {
            const selection = $getSelection();

            if ($isRangeSelection(selection)) {
                // If the last character is a line break, we also need to insert
                // a line break.
                const focus = selection.focus;
                selection.anchor.set(focus.key, focus.offset, focus.type);
                dispatchCommand(editor, KEY_ENTER_COMMAND, null);
                return;
            }
        }
    }

    $updateSelectedTextFromDOM(true, editor, data);
}

function onCompositionEnd(
    event: CompositionEvent,
    editor: LexicalEditor,
) {
    // Firefox fires onCompositionEnd before onInput, but Chrome/Webkit,
    // fire onInput before onCompositionEnd. To ensure the sequence works
    // like Chrome/Webkit we use the isFirefoxEndingComposition flag to
    // defer handling of onCompositionEnd in Firefox till we have processed
    // the logic in onInput.
    if (IS_FIREFOX) {
        isFirefoxEndingComposition = true;
    } else {
        updateEditor(editor, () => {
            onCompositionEndImpl(editor, event.data);
        });
    }
}

// This is a work-around is mainly Chrome specific bug where if you select
// the contents of an empty block, you cannot easily unselect anything.
// This results in a tiny selection box that looks buggy/broken. This can
// also help other browsers when selection might "appear" lost, when it
// really isn't.
function onClick(event: PointerEvent, editor: LexicalEditor) {
    updateEditor(editor, () => {
        const selection = $getSelection();
        const domSelection = getDOMSelection(editor._window);
        const lastSelection = $getPreviousSelection();

        if (domSelection) {
            if ($isRangeSelection(selection)) {
                const anchor = selection.anchor;
                const anchorNode = anchor.getNode();

                if (
                    anchor.type === "element" &&
                    anchor.offset === 0 &&
                    selection.isCollapsed() &&
                    !$isNode("Root", anchorNode) &&
                    $getRoot().getChildrenSize() === 1 &&
                    getTopLevelElementOrThrow(anchorNode).isEmpty() &&
                    lastSelection !== null &&
                    selection.is(lastSelection)
                ) {
                    domSelection.removeAllRanges();
                    selection.dirty = true;
                } else if (event.detail === 3 && !selection.isCollapsed()) {
                    // Tripple click causing selection to overflow into the nearest element. In that
                    // case visually it looks like a single element content is selected, focus node
                    // is actually at the beginning of the next element (if present) and any manipulations
                    // with selection (formatting) are affecting second element as well
                    const focus = selection.focus;
                    const focusNode = focus.getNode();
                    if (anchorNode !== focusNode) {
                        if ($isNode("Element", anchorNode)) {
                            anchorNode.select(0);
                        } else {
                            getParent(anchorNode, { throwIfNull: true }).select(0);
                        }
                    }
                }
            } else if (event.pointerType === "touch") {
                // This is used to update the selection on touch devices when the user clicks on text after a
                // node selection. See isSelectionChangeFromMouseDown for the inverse
                const domAnchorNode = domSelection.anchorNode;
                if (domAnchorNode !== null) {
                    const nodeType = domAnchorNode.nodeType;
                    // If the user is attempting to click selection back onto text, then
                    // we should attempt create a range selection.
                    // When we click on an empty paragraph node or the end of a paragraph that ends
                    // with an image/poll, the nodeType will be ELEMENT_NODE
                    if (nodeType === DOM_ELEMENT_TYPE || nodeType === DOM_TEXT_TYPE) {
                        const newSelection = internalCreateRangeSelection(
                            lastSelection,
                            domSelection,
                            editor,
                            event,
                        );
                        $setSelection(newSelection);
                    }
                }
            }
        }

        dispatchCommand(editor, CLICK_COMMAND, event);
    });
}

const PASS_THROUGH_COMMAND = Object.freeze({});
const rootElementEvents: RootElementEvents = [
    ["keydown", onKeyDown],
    ["pointerdown", onPointerDown],
    ["compositionstart", onCompositionStart],
    ["compositionend", onCompositionEnd],
    ["input", onInput],
    ["click", onClick],
    ["cut", PASS_THROUGH_COMMAND],
    ["copy", PASS_THROUGH_COMMAND],
    ["dragstart", PASS_THROUGH_COMMAND],
    ["dragover", PASS_THROUGH_COMMAND],
    ["dragend", PASS_THROUGH_COMMAND],
    ["paste", PASS_THROUGH_COMMAND],
    ["focus", PASS_THROUGH_COMMAND],
    ["blur", PASS_THROUGH_COMMAND],
    ["drop", PASS_THROUGH_COMMAND],
];

/**
 * Determines if Lexical should prevent the default behavior of the browser
 * and insert text using its own internal heuristics.
 * 
 * This is an extremely important function, and makes much of Lexical
 * work as intended between different browsers and across word, line and character
 * boundary/formats. It also is important for text replacement, node schemas and
 * composition mechanics.
 */
function $shouldPreventDefaultAndInsertText(
    selection: RangeSelection,
    domTargetRange: null | StaticRange,
    text: string,
    timeStamp: number,
    isBeforeInput: boolean,
): boolean {
    const anchor = selection.anchor;
    const focus = selection.focus;
    const anchorNode = anchor.getNode();
    const editor = getActiveEditor();
    const domSelection = getDOMSelection(editor._window);
    const domAnchorNode = domSelection !== null ? domSelection.anchorNode : null;
    const anchorKey = anchor.key;
    const backingAnchorElement = editor.getElementByKey(anchorKey);
    const textLength = text.length;

    return (
        anchorKey !== focus.key ||
        // If we're working with a non-text node.
        !$isNode("Text", anchorNode) ||
        // If we are replacing a range with a single character or grapheme, and not composing.
        (((!isBeforeInput &&
            (!CAN_USE_BEFORE_INPUT ||
                // We check to see if there has been
                // a recent beforeinput event for "textInput". If there has been one in the last
                // 50ms then we proceed as normal. However, if there is not, then this is likely
                // a dangling `input` event caused by execCommand('insertText').
                lastBeforeInputInsertTextTimeStamp < timeStamp + 50)) ||
            (anchorNode.isDirty() && textLength < 2) ||
            doesContainGrapheme(text)) &&
            anchor.offset !== focus.offset &&
            !anchorNode.isComposing()) ||
        // Any non standard text node.
        $isTokenOrSegmented(anchorNode) ||
        // If the text length is more than a single character and we're either
        // dealing with this in "beforeinput" or where the node has already recently
        // been changed (thus is dirty).
        (anchorNode.isDirty() && textLength > 1) ||
        // If the DOM selection element is not the same as the backing node during beforeinput.
        ((isBeforeInput || !CAN_USE_BEFORE_INPUT) &&
            backingAnchorElement !== null &&
            !anchorNode.isComposing() &&
            domAnchorNode !== getDOMTextNode(backingAnchorElement)) ||
        // If TargetRange is not the same as the DOM selection; browser trying to edit random parts
        // of the editor.
        (domSelection !== null &&
            domTargetRange !== null &&
            (!domTargetRange.collapsed ||
                domTargetRange.startContainer !== domSelection.anchorNode ||
                domTargetRange.startOffset !== domSelection.anchorOffset)) ||
        // Check if we're changing from bold to italics, or some other format.
        anchorNode.getFormat() !== selection.format ||
        anchorNode.getStyle() !== selection.style ||
        // One last set of heuristics to check against.
        $shouldInsertTextAfterOrBeforeTextNode(selection, anchorNode)
    );
}

function getTargetRange(event: InputEvent): null | StaticRange {
    if (!event.getTargetRanges) {
        return null;
    }
    const targetRanges = event.getTargetRanges();
    if (targetRanges.length === 0) {
        return null;
    }
    return targetRanges[0];
}

function getRootElementRemoveHandles(
    rootElement: HTMLElement,
): RootElementRemoveHandles {
    let eventHandles = (rootElement as CustomDomElement).__lexicalEventHandles;

    if (eventHandles === undefined) {
        eventHandles = [];
        (rootElement as CustomDomElement).__lexicalEventHandles = eventHandles;
    }

    return eventHandles;
}

function onDocumentSelectionChange(event: Event): void {
    const target = event.target as null | Element | Document;
    const targetWindow =
        target === null
            ? null
            : target.nodeType === 9
                ? (target as Document).defaultView
                : (target as Element).ownerDocument.defaultView;
    const domSelection = getDOMSelection(targetWindow);
    if (domSelection === null) {
        return;
    }
    const nextActiveEditor = getNearestEditorFromDOMNode(domSelection.anchorNode);
    if (!nextActiveEditor) {
        return;
    }

    if (isSelectionChangeFromMouseDown) {
        isSelectionChangeFromMouseDown = false;
        updateEditor(nextActiveEditor, () => {
            const lastSelection = $getPreviousSelection();
            const domAnchorNode = domSelection.anchorNode;
            if (domAnchorNode === null) {
                return;
            }
            const nodeType = domAnchorNode.nodeType;
            // If the user is attempting to click selection back onto text, then
            // we should attempt create a range selection.
            // When we click on an empty paragraph node or the end of a paragraph that ends
            // with an image/poll, the nodeType will be ELEMENT_NODE
            if (nodeType !== DOM_ELEMENT_TYPE && nodeType !== DOM_TEXT_TYPE) {
                return;
            }
            const newSelection = internalCreateRangeSelection(
                lastSelection,
                domSelection,
                nextActiveEditor,
                event,
            );
            $setSelection(newSelection);
        });
    }

    // When editor receives selection change event, we're checking if
    // it has any sibling editors (within same parent editor) that were active
    // before, and trigger selection change on it to nullify selection.
    const editors = [nextActiveEditor];
    const rootEditor = editors[editors.length - 1];
    const rootEditorKey = rootEditor._key;
    const activeNestedEditor = activeNestedEditorsMap.get(rootEditorKey);
    const prevActiveEditor = activeNestedEditor || rootEditor;

    if (prevActiveEditor !== nextActiveEditor) {
        onSelectionChange(domSelection, prevActiveEditor, false);
    }

    onSelectionChange(domSelection, nextActiveEditor, true);

    // If newly selected editor is nested, then add it to the map, clean map otherwise
    if (nextActiveEditor !== rootEditor) {
        activeNestedEditorsMap.set(rootEditorKey, nextActiveEditor);
    } else if (activeNestedEditor) {
        activeNestedEditorsMap.delete(rootEditorKey);
    }
}

function stopLexicalPropagation(event: Event): void {
    // We attach a special property to ensure the same event doesn't re-fire for parent editors.
    (event as CustomLexicalEvent).__lexicalHandled = true;
}

function hasStoppedLexicalPropagation(event: Event): boolean {
    return (event as CustomLexicalEvent).__lexicalHandled === true;
}

export function addRootElementEvents(
    rootElement: HTMLElement,
    editor: LexicalEditor,
) {
    // We only want to have a single global selectionchange event handler, shared
    // between all editor instances.
    const doc = rootElement.ownerDocument;
    const documentRootElementsCount = rootElementsRegistered.get(doc);
    if (
        documentRootElementsCount === undefined ||
        documentRootElementsCount < 1
    ) {
        doc.addEventListener("selectionchange", onDocumentSelectionChange);
    }
    rootElementsRegistered.set(doc, documentRootElementsCount || 0 + 1);

    (rootElement as CustomDomElement).__lexicalEditor = editor;
    const removeHandles = getRootElementRemoveHandles(rootElement);

    for (let i = 0; i < rootElementEvents.length; i++) {
        const [eventName, onEvent] = rootElementEvents[i];
        let eventHandler: (event: Event) => void;
        if (typeof onEvent === "function") {
            eventHandler = (event: Event) => {
                if (hasStoppedLexicalPropagation(event)) {
                    return;
                }
                stopLexicalPropagation(event);
                if (editor.isEditable() || eventName === "click") {
                    onEvent(event, editor);
                }
            };
        } else {
            eventHandler = (event: Event) => {
                if (hasStoppedLexicalPropagation(event)) {
                    return;
                }
                stopLexicalPropagation(event);
                if (editor.isEditable()) {
                    const handlers = {
                        blur: () => { dispatchCommand(editor, BLUR_COMMAND, event as FocusEvent); },
                        cut: () => { dispatchCommand(editor, CUT_COMMAND, event as ClipboardEvent); },
                        copy: () => { dispatchCommand(editor, COPY_COMMAND, event as ClipboardEvent); },
                        paste: () => { dispatchCommand(editor, PASTE_COMMAND, event as ClipboardEvent); },
                        dragstart: () => { dispatchCommand(editor, DRAGSTART_COMMAND, event as DragEvent); },
                        dragover: () => { dispatchCommand(editor, DRAGOVER_COMMAND, event as DragEvent); },
                        dragend: () => { dispatchCommand(editor, DRAGEND_COMMAND, event as DragEvent); },
                        drop: () => { dispatchCommand(editor, DROP_COMMAND, event as DragEvent); },
                        focus: () => { dispatchCommand(editor, FOCUS_COMMAND, event as FocusEvent); },
                    } as const;
                    if (handlers[eventName]) {
                        return handlers[eventName]();
                    }
                }
            };
        }
        rootElement.addEventListener(eventName, eventHandler);
        removeHandles.push(() => {
            rootElement.removeEventListener(eventName, eventHandler);
        });
    }
}

function cleanActiveNestedEditorsMap(editor: LexicalEditor) {
    activeNestedEditorsMap.delete(editor._key);
}

export function removeRootElementEvents(rootElement: HTMLElement): void {
    const doc = rootElement.ownerDocument;
    const documentRootElementsCount = rootElementsRegistered.get(doc);
    if (documentRootElementsCount === undefined) {
        throw new Error("Root element not registered");
    }

    // We only want to have a single global selectionchange event handler, shared
    // between all editor instances.
    rootElementsRegistered.set(doc, documentRootElementsCount - 1);
    if (rootElementsRegistered.get(doc) === 0) {
        doc.removeEventListener("selectionchange", onDocumentSelectionChange);
    }

    const editor = (rootElement as CustomDomElement).__lexicalEditor;

    if (editor) {
        cleanActiveNestedEditorsMap(editor);
        (rootElement as CustomDomElement).__lexicalEditor = null;
    }

    const removeHandles = getRootElementRemoveHandles(rootElement);

    for (let i = 0; i < removeHandles.length; i++) {
        removeHandles[i]();
    }

    (rootElement as CustomDomElement).__lexicalEventHandles = [];
}

export function markSelectionChangeFromDOMUpdate(): void {
    isSelectionChangeFromDOMUpdate = true;
}

function shouldSkipSelectionChange(
    domNode: null | Node,
    offset: number,
): boolean {
    return (
        domNode !== null &&
        domNode.nodeValue !== null &&
        domNode.nodeType === DOM_TEXT_TYPE &&
        offset !== 0 &&
        offset !== domNode.nodeValue.length
    );
}

function onSelectionChange(
    domSelection: Selection,
    editor: LexicalEditor,
    isActive: boolean,
): void {
    const {
        anchorNode: anchorDOM,
        anchorOffset,
        focusNode: focusDOM,
        focusOffset,
    } = domSelection;
    if (isSelectionChangeFromDOMUpdate) {
        isSelectionChangeFromDOMUpdate = false;

        // If native DOM selection is on a DOM element, then
        // we should continue as usual, as Lexical's selection
        // may have normalized to a better child. If the DOM
        // element is a text node, we can safely apply this
        // optimization and skip the selection change entirely.
        // We also need to check if the offset is at the boundary,
        // because in this case, we might need to normalize to a
        // sibling instead.
        if (
            shouldSkipSelectionChange(anchorDOM, anchorOffset) &&
            shouldSkipSelectionChange(focusDOM, focusOffset)
        ) {
            return;
        }
    }
    updateEditor(editor, () => {
        // Non-active editor don't need any extra logic for selection, it only needs update
        // to reconcile selection (set it to null) to ensure that only one editor has non-null selection.
        if (!isActive) {
            $setSelection(null);
            return;
        }

        if (!isSelectionWithinEditor(editor, anchorDOM, focusDOM)) {
            return;
        }

        const selection = $getSelection();

        // Update the selection format
        if ($isRangeSelection(selection)) {
            const anchor = selection.anchor;
            const anchorNode = anchor.getNode();

            if (selection.isCollapsed()) {
                // Badly interpreted range selection when collapsed - #1482
                if (
                    domSelection.type === "Range" &&
                    domSelection.anchorNode === domSelection.focusNode
                ) {
                    selection.dirty = true;
                }

                // If we have marked a collapsed selection format, and we're
                // within the given time range – then attempt to use that format
                // instead of getting the format from the anchor node.
                const windowEvent = getWindow(editor).event;
                const currentTimeStamp = windowEvent
                    ? windowEvent.timeStamp
                    : performance.now();
                const [lastFormat, lastStyle, lastOffset, lastKey, timeStamp] =
                    collapsedSelectionFormat;

                const root = $getRoot();
                const isRootTextContentEmpty =
                    editor.isComposing() === false && root.getTextContent() === "";

                if (
                    currentTimeStamp < timeStamp + 200 &&
                    anchor.offset === lastOffset &&
                    anchor.key === lastKey
                ) {
                    selection.format = lastFormat;
                    selection.style = lastStyle;
                } else {
                    if (anchor.type === "text") {
                        if (!$isNode("Text", anchorNode)) {
                            throw new Error("Point.getNode() must return TextNode when type is text");
                        }
                        selection.format = anchorNode.getFormat();
                        selection.style = anchorNode.getStyle();
                    } else if (anchor.type === "element" && !isRootTextContentEmpty) {
                        const lastNode = anchor.getNode();
                        if (
                            lastNode.getType() === "Paragraph" &&
                            lastNode.getChildrenSize() === 0
                        ) {
                            selection.format = lastNode.__format;
                        } else {
                            selection.format = 0;
                        }
                        selection.style = "";
                    }
                }
            } else {
                const anchorKey = anchor.key;
                const focus = selection.focus;
                const focusKey = focus.key;
                const nodes = selection.getNodes();
                const nodesLength = nodes.length;
                const isBackward = selection.isBackward();
                const startOffset = isBackward ? focusOffset : anchorOffset;
                const endOffset = isBackward ? anchorOffset : focusOffset;
                const startKey = isBackward ? focusKey : anchorKey;
                const endKey = isBackward ? anchorKey : focusKey;
                let combinedFormat = 0; // Start with all formats (boolean flags) set to false
                let hasTextNodes = false;
                for (let i = 0; i < nodesLength; i++) {
                    const node = nodes[i];
                    const textContentSize = node.getTextContentSize();
                    if (
                        $isNode("Text", node) &&
                        textContentSize !== 0 &&
                        // Exclude empty text nodes at boundaries resulting from user's selection
                        !(
                            (i === 0 &&
                                node.__key === startKey &&
                                startOffset === textContentSize) ||
                            (i === nodesLength - 1 &&
                                node.__key === endKey &&
                                endOffset === 0)
                        )
                    ) {
                        // TODO: what about style?
                        hasTextNodes = true;
                        // Apply the format to the combined format
                        combinedFormat |= node.getFormat();
                    }
                }

                selection.format = combinedFormat;
            }
        }

        console.log("triggering editor selection_change_command");
        dispatchCommand(editor, SELECTION_CHANGE_COMMAND, undefined);
    });
}

export function markCollapsedSelectionFormat(
    format: number,
    style: string,
    offset: number,
    key: NodeKey,
    timeStamp: number,
) {
    collapsedSelectionFormat = [format, style, offset, key, timeStamp];
}
