import { useEffect } from "react";
import { INSERT_ORDERED_LIST_COMMAND, INSERT_PARAGRAPH_COMMAND, INSERT_UNORDERED_LIST_COMMAND, REMOVE_LIST_COMMAND } from "../commands.js";
import { COMMAND_PRIORITY_LOW } from "../consts.js";
import { useLexicalComposerContext } from "../context.js";
import { $handleListInsertParagraph, insertList, removeList } from "../nodes/ListNode.js";
import { mergeRegister } from "../utils.js";

export function ListPlugin(): null {
    const editor = useLexicalComposerContext();

    useEffect(() => {
        if (!editor) return;
        return mergeRegister(
            editor.registerCommand(
                INSERT_ORDERED_LIST_COMMAND,
                () => {
                    insertList(editor, "number");
                    return true;
                },
                COMMAND_PRIORITY_LOW,
            ),
            editor.registerCommand(
                INSERT_UNORDERED_LIST_COMMAND,
                () => {
                    insertList(editor, "bullet");
                    return true;
                },
                COMMAND_PRIORITY_LOW,
            ),
            editor.registerCommand(
                REMOVE_LIST_COMMAND,
                () => {
                    removeList(editor);
                    return true;
                },
                COMMAND_PRIORITY_LOW,
            ),
            editor.registerCommand(
                INSERT_PARAGRAPH_COMMAND,
                () => {
                    const hasHandledInsertParagraph = $handleListInsertParagraph();

                    if (hasHandledInsertParagraph) {
                        return true;
                    }

                    return false;
                },
                COMMAND_PRIORITY_LOW,
            ),
        );
    }, [editor]);

    return null;
}
