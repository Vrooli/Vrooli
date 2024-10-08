import { useLayoutEffect } from "react";
import { LexicalErrorBoundary } from "../LexicalErrorBoundary";
import { $insertDataTransferForRichText, copyToClipboard } from "../clipboard";
import { CLICK_COMMAND, CONTROLLED_TEXT_INSERTION_COMMAND, COPY_COMMAND, CUT_COMMAND, DELETE_CHARACTER_COMMAND, DELETE_LINE_COMMAND, DELETE_WORD_COMMAND, DRAGOVER_COMMAND, DRAGSTART_COMMAND, DRAG_DROP_PASTE, DROP_COMMAND, FORMAT_ELEMENT_COMMAND, FORMAT_TEXT_COMMAND, INDENT_CONTENT_COMMAND, INSERT_LINE_BREAK_COMMAND, INSERT_PARAGRAPH_COMMAND, INSERT_TAB_COMMAND, KEY_ARROW_DOWN_COMMAND, KEY_ARROW_LEFT_COMMAND, KEY_ARROW_RIGHT_COMMAND, KEY_ARROW_UP_COMMAND, KEY_BACKSPACE_COMMAND, KEY_DELETE_COMMAND, KEY_ENTER_COMMAND, KEY_ESCAPE_COMMAND, OUTDENT_CONTENT_COMMAND, PASTE_COMMAND, REMOVE_TEXT_COMMAND, SELECT_ALL_COMMAND } from "../commands";
import { CAN_USE_BEFORE_INPUT, COMMAND_PRIORITY_EDITOR, IS_APPLE_WEBKIT, IS_IOS, IS_SAFARI } from "../consts";
import { useLexicalComposerContext } from "../context";
import { LexicalEditor } from "../editor";
import { ElementNode } from "../nodes/ElementNode";
import { $createRangeSelection, $insertNodes, $moveCharacter, $shouldOverrideDefaultCharacterSelection, caretFromPoint } from "../selection";
import { ElementFormatType, TextFormatType } from "../types";
import { $createNode, $findMatchingParent, $getAdjacentNode, $getNearestBlockElementAncestorOrThrow, $getNearestNodeFromDOMNode, $getSelection, $isNode, $isNodeSelection, $isRangeSelection, $isSelectionAtEndOfRoot, $isTargetWithinDecorator, $normalizeSelection, $selectAll, $setSelection, eventFiles, getIndexWithinParent, getParent, handleIndentAndOutdent, isSelectionCapturedInDecoratorInput, mergeRegister, objectKlassEquals, onCutForRichText, onPasteForRichText } from "../utils";

//TODO this file is a good place to store markdown string and selection positions. Then we can edit this in real-time instead of exporting everything to markdown on every change

function registerRichText(editor: LexicalEditor): (() => void) {
    const removeListener = mergeRegister(
        editor.registerCommand(
            CLICK_COMMAND,
            (payload) => {
                const selection = $getSelection();
                if ($isNodeSelection(selection)) {
                    selection.clear();
                    return true;
                }
                return false;
            },
            0,
        ),
        editor.registerCommand<boolean>(
            DELETE_CHARACTER_COMMAND,
            (isBackward) => {
                const selection = $getSelection();
                if (!$isRangeSelection(selection)) {
                    return false;
                }
                selection.deleteCharacter(isBackward);
                return true;
            },
            COMMAND_PRIORITY_EDITOR,
        ),
        editor.registerCommand<boolean>(
            DELETE_WORD_COMMAND,
            (isBackward) => {
                const selection = $getSelection();
                if (!$isRangeSelection(selection)) {
                    return false;
                }
                selection.deleteWord(isBackward);
                return true;
            },
            COMMAND_PRIORITY_EDITOR,
        ),
        editor.registerCommand<boolean>(
            DELETE_LINE_COMMAND,
            (isBackward) => {
                const selection = $getSelection();
                if (!$isRangeSelection(selection)) {
                    return false;
                }
                selection.deleteLine(isBackward);
                return true;
            },
            COMMAND_PRIORITY_EDITOR,
        ),
        editor.registerCommand(
            CONTROLLED_TEXT_INSERTION_COMMAND,
            (eventOrText) => {
                const selection = $getSelection();

                if (typeof eventOrText === "string") {
                    if (selection !== null) {
                        selection.insertText(eventOrText);
                    }
                } else {
                    if (selection === null) {
                        return false;
                    }

                    const dataTransfer = eventOrText.dataTransfer;
                    if (dataTransfer !== null) {
                        $insertDataTransferForRichText(dataTransfer, selection, editor);
                    } else if ($isRangeSelection(selection)) {
                        const data = eventOrText.data;
                        if (data) {
                            selection.insertText(data);
                        }
                        return true;
                    }
                }
                return true;
            },
            COMMAND_PRIORITY_EDITOR,
        ),
        editor.registerCommand(
            REMOVE_TEXT_COMMAND,
            () => {
                const selection = $getSelection();
                if (!$isRangeSelection(selection)) {
                    return false;
                }
                selection.removeText();
                return true;
            },
            COMMAND_PRIORITY_EDITOR,
        ),
        editor.registerCommand<TextFormatType>(
            FORMAT_TEXT_COMMAND,
            (format) => {
                const selection = $getSelection();
                console.log("in format_text_command register", format, selection);
                if (!$isRangeSelection(selection)) {
                    return false;
                }
                selection.formatText(format);
                return true;
            },
            COMMAND_PRIORITY_EDITOR,
        ),
        editor.registerCommand<ElementFormatType>(
            FORMAT_ELEMENT_COMMAND,
            (format) => {
                const selection = $getSelection();
                if (!$isRangeSelection(selection) && !$isNodeSelection(selection)) {
                    return false;
                }
                const nodes = selection.getNodes();
                for (const node of nodes) {
                    const element = $findMatchingParent(
                        node,
                        (parentNode): parentNode is ElementNode =>
                            $isNode("Element", parentNode) && !parentNode.isInline(),
                    );
                    if (element !== null) {
                        element.setFormat(format);
                    }
                }
                return true;
            },
            COMMAND_PRIORITY_EDITOR,
        ),
        editor.registerCommand<boolean>(
            INSERT_LINE_BREAK_COMMAND,
            (selectStart) => {
                const selection = $getSelection();
                if (!$isRangeSelection(selection)) {
                    return false;
                }
                selection.insertLineBreak(selectStart);
                return true;
            },
            COMMAND_PRIORITY_EDITOR,
        ),
        editor.registerCommand(
            INSERT_PARAGRAPH_COMMAND,
            () => {
                const selection = $getSelection();
                if (!$isRangeSelection(selection)) {
                    return false;
                }
                selection.insertParagraph();
                return true;
            },
            COMMAND_PRIORITY_EDITOR,
        ),
        editor.registerCommand(
            INSERT_TAB_COMMAND,
            () => {
                $insertNodes([$createNode("Tab", {})]);
                return true;
            },
            COMMAND_PRIORITY_EDITOR,
        ),
        editor.registerCommand(
            INDENT_CONTENT_COMMAND,
            () => {
                return handleIndentAndOutdent((block) => {
                    const indent = block.getIndent();
                    block.setIndent(indent + 1);
                });
            },
            COMMAND_PRIORITY_EDITOR,
        ),
        editor.registerCommand(
            OUTDENT_CONTENT_COMMAND,
            () => {
                return handleIndentAndOutdent((block) => {
                    const indent = block.getIndent();
                    if (indent > 0) {
                        block.setIndent(indent - 1);
                    }
                });
            },
            COMMAND_PRIORITY_EDITOR,
        ),
        editor.registerCommand<KeyboardEvent>(
            KEY_ARROW_UP_COMMAND,
            (event) => {
                const selection = $getSelection();
                if (
                    $isNodeSelection(selection) &&
                    !$isTargetWithinDecorator(event.target as HTMLElement)
                ) {
                    // If selection is on a node, let's try and move selection
                    // back to being a range selection.
                    const nodes = selection.getNodes();
                    if (nodes.length > 0) {
                        nodes[0].selectPrevious();
                        return true;
                    }
                } else if ($isRangeSelection(selection)) {
                    const possibleNode = $getAdjacentNode(selection.focus, true);
                    if (
                        !event.shiftKey &&
                        $isNode("Decorator", possibleNode) &&
                        !possibleNode.isIsolated() &&
                        !possibleNode.isInline()
                    ) {
                        possibleNode.selectPrevious();
                        event.preventDefault();
                        return true;
                    }
                }
                return false;
            },
            COMMAND_PRIORITY_EDITOR,
        ),
        editor.registerCommand<KeyboardEvent>(
            KEY_ARROW_DOWN_COMMAND,
            (event) => {
                const selection = $getSelection();
                if ($isNodeSelection(selection)) {
                    // If selection is on a node, let's try and move selection
                    // back to being a range selection.
                    const nodes = selection.getNodes();
                    if (nodes.length > 0) {
                        nodes[0].selectNext(0, 0);
                        return true;
                    }
                } else if ($isRangeSelection(selection)) {
                    if ($isSelectionAtEndOfRoot(selection)) {
                        event.preventDefault();
                        return true;
                    }
                    const possibleNode = $getAdjacentNode(selection.focus, false);
                    if (
                        !event.shiftKey &&
                        $isNode("Decorator", possibleNode) &&
                        !possibleNode.isIsolated() &&
                        !possibleNode.isInline()
                    ) {
                        possibleNode.selectNext();
                        event.preventDefault();
                        return true;
                    }
                }
                return false;
            },
            COMMAND_PRIORITY_EDITOR,
        ),
        editor.registerCommand<KeyboardEvent>(
            KEY_ARROW_LEFT_COMMAND,
            (event) => {
                const selection = $getSelection();
                if ($isNodeSelection(selection)) {
                    // If selection is on a node, let's try and move selection
                    // back to being a range selection.
                    const nodes = selection.getNodes();
                    if (nodes.length > 0) {
                        event.preventDefault();
                        nodes[0].selectPrevious();
                        return true;
                    }
                }
                if (!$isRangeSelection(selection)) {
                    return false;
                }
                if ($shouldOverrideDefaultCharacterSelection(selection, true)) {
                    const isHoldingShift = event.shiftKey;
                    event.preventDefault();
                    $moveCharacter(selection, isHoldingShift, true);
                    return true;
                }
                return false;
            },
            COMMAND_PRIORITY_EDITOR,
        ),
        editor.registerCommand<KeyboardEvent>(
            KEY_ARROW_RIGHT_COMMAND,
            (event) => {
                const selection = $getSelection();
                if (
                    $isNodeSelection(selection) &&
                    !$isTargetWithinDecorator(event.target as HTMLElement)
                ) {
                    // If selection is on a node, let's try and move selection
                    // back to being a range selection.
                    const nodes = selection.getNodes();
                    if (nodes.length > 0) {
                        event.preventDefault();
                        nodes[0].selectNext(0, 0);
                        return true;
                    }
                }
                if (!$isRangeSelection(selection)) {
                    return false;
                }
                const isHoldingShift = event.shiftKey;
                if ($shouldOverrideDefaultCharacterSelection(selection, false)) {
                    event.preventDefault();
                    $moveCharacter(selection, isHoldingShift, false);
                    return true;
                }
                return false;
            },
            COMMAND_PRIORITY_EDITOR,
        ),
        editor.registerCommand<KeyboardEvent>(
            KEY_BACKSPACE_COMMAND,
            (event) => {
                if ($isTargetWithinDecorator(event.target as HTMLElement)) {
                    return false;
                }
                const selection = $getSelection();
                if (!$isRangeSelection(selection)) {
                    return false;
                }
                event.preventDefault();
                const { anchor } = selection;
                const anchorNode = anchor.getNode();

                if (
                    selection.isCollapsed() &&
                    anchor.offset === 0 &&
                    !$isNode("Root", anchorNode)
                ) {
                    const element = $getNearestBlockElementAncestorOrThrow(anchorNode);
                    if (element.getIndent() > 0) {
                        return editor.dispatchCommand(OUTDENT_CONTENT_COMMAND, undefined);
                    }
                }
                return editor.dispatchCommand(DELETE_CHARACTER_COMMAND, true);
            },
            COMMAND_PRIORITY_EDITOR,
        ),
        editor.registerCommand<KeyboardEvent>(
            KEY_DELETE_COMMAND,
            (event) => {
                if ($isTargetWithinDecorator(event.target as HTMLElement)) {
                    return false;
                }
                const selection = $getSelection();
                if (!$isRangeSelection(selection)) {
                    return false;
                }
                event.preventDefault();
                return editor.dispatchCommand(DELETE_CHARACTER_COMMAND, false);
            },
            COMMAND_PRIORITY_EDITOR,
        ),
        editor.registerCommand<KeyboardEvent | null>(
            KEY_ENTER_COMMAND,
            (event) => {
                const selection = $getSelection();
                if (!$isRangeSelection(selection)) {
                    return false;
                }
                if (event !== null) {
                    // If we have beforeinput, then we can avoid blocking
                    // the default behavior. This ensures that the iOS can
                    // intercept that we're actually inserting a paragraph,
                    // and autocomplete, autocapitalize etc work as intended.
                    // This can also cause a strange performance issue in
                    // Safari, where there is a noticeable pause due to
                    // preventing the key down of enter.
                    if (
                        (IS_IOS || IS_SAFARI || IS_APPLE_WEBKIT) &&
                        CAN_USE_BEFORE_INPUT
                    ) {
                        return false;
                    }
                    event.preventDefault();
                    if (event.shiftKey) {
                        return editor.dispatchCommand(INSERT_LINE_BREAK_COMMAND, false);
                    }
                }
                return editor.dispatchCommand(INSERT_PARAGRAPH_COMMAND, undefined);
            },
            COMMAND_PRIORITY_EDITOR,
        ),
        editor.registerCommand(
            KEY_ESCAPE_COMMAND,
            () => {
                const selection = $getSelection();
                if (!$isRangeSelection(selection)) {
                    return false;
                }
                editor.blur();
                return true;
            },
            COMMAND_PRIORITY_EDITOR,
        ),
        editor.registerCommand<DragEvent>(
            DROP_COMMAND,
            (event) => {
                // If we're dealing with plaintext, then we should just

                const [, files] = eventFiles(event);
                console.log("in drop command richtextplugin", event, files, event.dataTransfer?.getData("text/plain"));
                if (files.length > 0) {
                    const x = event.clientX;
                    const y = event.clientY;
                    const eventRange = caretFromPoint(x, y);
                    console.log("drop command event range:", eventRange);
                    if (eventRange) {
                        const { offset: domOffset, node: domNode } = eventRange;
                        const node = $getNearestNodeFromDOMNode(domNode);
                        if (node) {
                            const selection = $createRangeSelection();
                            if ($isNode("Text", node)) {
                                selection.anchor.set(node.__key, domOffset, "text");
                                selection.focus.set(node.__key, domOffset, "text");
                            } else {
                                const parentKey = getParent(node, { throwIfNull: true }).__key;
                                const offset = getIndexWithinParent(node) + 1;
                                selection.anchor.set(parentKey, offset, "element");
                                selection.focus.set(parentKey, offset, "element");
                            }
                            const normalizedSelection =
                                $normalizeSelection(selection);
                            $setSelection(normalizedSelection);
                        }
                        editor.dispatchCommand(DRAG_DROP_PASTE, files);
                    }
                    event.preventDefault();
                    return true;
                }

                const selection = $getSelection();
                console.log("selection in drop command:", selection);
                if ($isRangeSelection(selection)) {
                    return true;
                }

                return false;
            },
            COMMAND_PRIORITY_EDITOR,
        ),
        editor.registerCommand<DragEvent>(
            DRAGSTART_COMMAND,
            (event) => {
                const [isFileTransfer] = eventFiles(event);
                const selection = $getSelection();
                if (isFileTransfer && !$isRangeSelection(selection)) {
                    return false;
                }
                return true;
            },
            COMMAND_PRIORITY_EDITOR,
        ),
        editor.registerCommand<DragEvent>(
            DRAGOVER_COMMAND,
            (event) => {
                const [isFileTransfer] = eventFiles(event);
                const selection = $getSelection();
                if (isFileTransfer && !$isRangeSelection(selection)) {
                    return false;
                }
                const x = event.clientX;
                const y = event.clientY;
                const eventRange = caretFromPoint(x, y);
                if (eventRange !== null) {
                    const node = $getNearestNodeFromDOMNode(eventRange.node);
                    if ($isNode("Decorator", node)) {
                        // Show browser caret as the user is dragging the media across the screen. Won't work
                        // for DecoratorNode nor it's relevant.
                        event.preventDefault();
                    }
                }
                return true;
            },
            COMMAND_PRIORITY_EDITOR,
        ),
        editor.registerCommand(
            SELECT_ALL_COMMAND,
            () => {
                $selectAll();

                return true;
            },
            COMMAND_PRIORITY_EDITOR,
        ),
        editor.registerCommand(
            COPY_COMMAND,
            (event) => {
                copyToClipboard(
                    editor,
                    objectKlassEquals(event, ClipboardEvent)
                        ? (event as ClipboardEvent)
                        : null,
                );
                return true;
            },
            COMMAND_PRIORITY_EDITOR,
        ),
        editor.registerCommand(
            CUT_COMMAND,
            (event) => {
                onCutForRichText(event, editor);
                return true;
            },
            COMMAND_PRIORITY_EDITOR,
        ),
        editor.registerCommand(
            PASTE_COMMAND,
            (event) => {
                const [, files, hasTextContent] = eventFiles(event);
                console.log("in richtextplugin paste_command", event, files, hasTextContent);
                if (files.length > 0 && !hasTextContent) {
                    editor.dispatchCommand(DRAG_DROP_PASTE, files);
                    return true;
                }

                // if inputs then paste within the input ignore creating a new node on paste event
                if (isSelectionCapturedInDecoratorInput(event.target as Node)) {
                    return false;
                }

                const selection = $getSelection();
                if (selection !== null) {
                    onPasteForRichText(event, editor);
                    return true;
                }

                return false;
            },
            COMMAND_PRIORITY_EDITOR,
        ),
    );
    return removeListener;
}

/**
 * Adds support for Dragon, which is an extension that allows users to dictate text.
 */
function registerDragonSupport(editor: LexicalEditor): (() => void) {
    const origin = window.location.origin;
    function handler(event: MessageEvent) {
        if (event.origin !== origin) {
            return;
        }
        const rootElement = editor.getRootElement();

        if (document.activeElement !== rootElement) {
            return;
        }

        const data = event.data;

        if (typeof data === "string") {
            let parsedData;

            try {
                parsedData = JSON.parse(data);
            } catch (e) {
                return;
            }

            if (
                parsedData &&
                parsedData.protocol === "nuanria_messaging" &&
                parsedData.type === "request"
            ) {
                const payload = parsedData.payload;

                if (payload && payload.functionId === "makeChanges") {
                    const args = payload.args;

                    if (args) {
                        const [
                            elementStart,
                            elementLength,
                            text,
                            selStart,
                            selLength,
                            formatCommand,
                        ] = args;
                        // TODO: we should probably handle formatCommand somehow?
                        // eslint-disable-next-line no-unused-expressions
                        formatCommand;
                        editor.update(() => {
                            const selection = $getSelection();

                            if ($isRangeSelection(selection)) {
                                const anchor = selection.anchor;
                                let anchorNode = anchor.getNode();
                                let setSelStart = 0;
                                let setSelEnd = 0;

                                if ($isNode("Text", anchorNode)) {
                                    // set initial selection
                                    if (elementStart >= 0 && elementLength >= 0) {
                                        setSelStart = elementStart;
                                        setSelEnd = elementStart + elementLength;
                                        // If the offset is more than the end, make it the end
                                        selection.setTextNodeRange(
                                            anchorNode,
                                            setSelStart,
                                            anchorNode,
                                            setSelEnd,
                                        );
                                    }
                                }

                                if (setSelStart !== setSelEnd || text !== "") {
                                    selection.insertRawText(text);
                                    anchorNode = anchor.getNode();
                                }

                                if ($isNode("Text", anchorNode)) {
                                    // set final selection
                                    setSelStart = selStart;
                                    setSelEnd = selStart + selLength;
                                    const anchorNodeTextLength = anchorNode.getTextContentSize();
                                    // If the offset is more than the end, make it the end
                                    setSelStart =
                                        setSelStart > anchorNodeTextLength
                                            ? anchorNodeTextLength
                                            : setSelStart;
                                    setSelEnd =
                                        setSelEnd > anchorNodeTextLength
                                            ? anchorNodeTextLength
                                            : setSelEnd;
                                    selection.setTextNodeRange(
                                        anchorNode,
                                        setSelStart,
                                        anchorNode,
                                        setSelEnd,
                                    );
                                }

                                // block the chrome extension from handling this event
                                event.stopImmediatePropagation();
                            }
                        });
                    }
                }
            }
        }
    }

    window.addEventListener("message", handler, true);
    return () => {
        window.removeEventListener("message", handler, true);
    };
}

export function RichTextPlugin({
    contentEditable,
}: {
    contentEditable: JSX.Element;
}): JSX.Element {
    const editor = useLexicalComposerContext();

    useLayoutEffect(() => {
        if (!editor) return;
        return mergeRegister(
            registerRichText(editor),
            registerDragonSupport(editor),
        );
    }, [editor]);

    return (
        <LexicalErrorBoundary>
            {contentEditable}
        </LexicalErrorBoundary>
    );
}
