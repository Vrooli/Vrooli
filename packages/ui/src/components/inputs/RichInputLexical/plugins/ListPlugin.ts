import { useEffect } from "react";
import { INSERT_ORDERED_LIST_COMMAND, INSERT_PARAGRAPH_COMMAND, INSERT_UNORDERED_LIST_COMMAND, REMOVE_LIST_COMMAND } from "../commands";
import { COMMAND_PRIORITY_LOW } from "../consts";
import { useLexicalComposerContext } from "../context";
import { LexicalEditor } from "../editor";
import { ListItemNode } from "../nodes/ListItemNode";
import { $handleListInsertParagraph, ListNode, insertList, removeList } from "../nodes/ListNode";
import { mergeRegister } from "../utils";

export const useList = (editor: LexicalEditor) => {
    useEffect(() => {
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
};

export const ListPlugin = (): null => {
    const [editor] = useLexicalComposerContext();

    useEffect(() => {
        if (!editor.hasNodes([ListNode, ListItemNode])) {
            throw new Error(
                "ListPlugin: ListNode and/or ListItemNode not registered on editor",
            );
        }
    }, [editor]);

    useList(editor);

    return null;
};
