import { useLayoutEffect } from "react";
import { useLexicalComposerContext } from "../context";
import { EditorState, LexicalEditor } from "../editor";

export const OnChangePlugin = ({
    ignoreHistoryMergeTagChange = true,
    ignoreSelectionChange = false,
    onChange,
}: {
    ignoreHistoryMergeTagChange?: boolean;
    ignoreSelectionChange?: boolean;
    onChange: (
        editorState: EditorState,
        editor: LexicalEditor,
        tags: Set<string>,
    ) => void;
}): null => {
    const [editor] = useLexicalComposerContext();

    useLayoutEffect(() => {
        if (onChange) {
            return editor.registerUpdateListener(
                ({ editorState, dirtyElements, dirtyLeaves, prevEditorState, tags }) => {
                    if (
                        (ignoreSelectionChange &&
                            dirtyElements.size === 0 &&
                            dirtyLeaves.size === 0) ||
                        (ignoreHistoryMergeTagChange && tags.has("history-merge")) ||
                        prevEditorState.isEmpty()
                    ) {
                        return;
                    }

                    onChange(editorState, editor, tags);
                },
            );
        }
    }, [editor, ignoreHistoryMergeTagChange, ignoreSelectionChange, onChange]);

    return null;
};
