import { CodeNode } from "@lexical/code";
import { LinkNode } from "@lexical/link";
import { ListItemNode, ListNode } from "@lexical/list";
import { $convertFromMarkdownString, TRANSFORMERS } from "@lexical/markdown";
import { InitialEditorStateType } from "@lexical/react/LexicalComposer";
import { createLexicalComposerContext, LexicalComposerContext, LexicalComposerContextType } from "@lexical/react/LexicalComposerContext";
import { ContentEditable } from "@lexical/react/LexicalContentEditable";
import LexicalErrorBoundary from "@lexical/react/LexicalErrorBoundary";
import { HistoryPlugin } from "@lexical/react/LexicalHistoryPlugin";
import { HorizontalRuleNode } from "@lexical/react/LexicalHorizontalRuleNode";
import { MarkdownShortcutPlugin } from "@lexical/react/LexicalMarkdownShortcutPlugin";
import { RichTextPlugin } from "@lexical/react/LexicalRichTextPlugin";
import { HeadingNode, QuoteNode } from "@lexical/rich-text";
import { Box, useTheme } from "@mui/material";
import "highlight.js/styles/monokai-sublime.css";
import { $createParagraphNode, $getRoot, $getSelection, createEditor, FORMAT_TEXT_COMMAND, LexicalEditor } from "lexical";
import { CSSProperties, useCallback, useLayoutEffect, useMemo } from "react";
import { linkColors } from "styles";
import { ListObject } from "utils/display/listTools";
import { RichInputTagDropdown, useTagDropdown } from "../RichInputTagDropdown/RichInputTagDropdown";
import { RichInputAction } from "../RichInputToolbar/RichInputToolbar";
import { RichInputChildView, RichInputLexicalProps } from "../types";

export const CAN_USE_DOM: boolean =
    typeof window !== "undefined" &&
    typeof window.document !== "undefined" &&
    typeof window.document.createElement !== "undefined";
const HISTORY_MERGE_OPTIONS = { tag: "history-merge" };

function initializeEditor(
    editor: LexicalEditor,
    initialEditorState?: InitialEditorStateType,
): void {
    if (initialEditorState === null) {
        return;
    } else if (initialEditorState === undefined) {
        editor.update(() => {
            const root = $getRoot();
            if (root.isEmpty()) {
                const paragraph = $createParagraphNode();
                root.append(paragraph);
                const activeElement = CAN_USE_DOM ? document.activeElement : null;
                if (
                    $getSelection() !== null ||
                    (activeElement !== null && activeElement === editor.getRootElement())
                ) {
                    paragraph.select();
                }
            }
        }, HISTORY_MERGE_OPTIONS);
    } else if (initialEditorState !== null) {
        switch (typeof initialEditorState) {
            case "string": {
                const parsedEditorState = editor.parseEditorState(initialEditorState);
                editor.setEditorState(parsedEditorState, HISTORY_MERGE_OPTIONS);
                break;
            }
            case "object": {
                editor.setEditorState(initialEditorState, HISTORY_MERGE_OPTIONS);
                break;
            }
            case "function": {
                editor.update(() => {
                    const root = $getRoot();
                    if (root.isEmpty()) {
                        initialEditorState(editor);
                    }
                }, HISTORY_MERGE_OPTIONS);
                break;
            }
        }
    }
}

const theme = {

};

/** TextField for entering WYSIWYG text */
export const RichInputLexical = ({
    autoFocus = false,
    disabled = false,
    error = false,
    getTaggableItems,
    maxRows,
    minRows = 4,
    name,
    onBlur,
    onChange,
    openAssistantDialog,
    placeholder = "",
    redo,
    tabIndex,
    toggleMarkdown,
    undo,
    value,
    sx,
}: RichInputLexicalProps) => {
    const { palette, spacing } = useTheme();

    const onError = (error: Error) => {
        console.error(error);
    };

    const initialConfig = useMemo(() => ({
        editable: true,
        editorState: () => $convertFromMarkdownString(value, TRANSFORMERS),
        namespace: "MyEditor",
        nodes: [CodeNode, HeadingNode, HorizontalRuleNode, LinkNode, ListNode, ListItemNode, QuoteNode],
        onError,
        theme,
    }), [value]);

    const composerContext: [LexicalEditor, LexicalComposerContextType] = useMemo(() => {
        const {
            editorState: initialEditorState,
            namespace,
            nodes,
            onError,
            theme,
        } = initialConfig;
        const context: LexicalComposerContextType = createLexicalComposerContext(
            null,
            theme,
        );
        const newEditor = createEditor({
            editable: initialConfig.editable,
            namespace,
            nodes,
            onError: (error) => onError(error),
            theme,
        });
        initializeEditor(newEditor, initialEditorState);
        const editor = newEditor;
        return [editor, context];
        // We only do this for init
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    useLayoutEffect(() => {
        const isEditable = initialConfig.editable;
        const [editor] = composerContext;
        editor.setEditable(isEditable !== undefined ? isEditable : true);
        // We only do this for init
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    const tagData = useTagDropdown({ getTaggableItems });
    const selectDropdownItem = useCallback((item: ListObject) => {
        //TODO
    }, []);

    (RichInputLexical as unknown as RichInputChildView).handleAction = (action: RichInputAction) => {
        console.log("in RichInputlExical handleAction", action);
        //TODO
        const actionMap: { [key in RichInputAction]: (() => unknown) } = {
            "Assistant": () => openAssistantDialog(""),
            "Bold": () => composerContext[0].dispatchCommand(FORMAT_TEXT_COMMAND, "bold"),
            "Header1": () => { },
            "Header2": () => { },
            "Header3": () => { },
            "Italic": () => composerContext[0].dispatchCommand(FORMAT_TEXT_COMMAND, "italic"),
            "Link": () => { },
            "ListBullet": () => { },//composerContext[0].dispatchCommand(INSERT_UNORDERED_LIST_COMMAND),
            "ListCheckbox": () => { },
            "ListNumber": () => { },
            "Mode": toggleMarkdown,
            "Redo": redo,
            "Strikethrough": () => { },
            "Undo": undo,
        };
        const actionFunction = actionMap[action];
        if (actionFunction) actionFunction();
    };

    return (
        <LexicalComposerContext.Provider value={composerContext}>
            <Box className="editor-container" sx={{
                border: `1px solid ${error ? "red" : "black"}`,
                borderRadius: "0 0 0.5rem 0.5rem",
                padding: "12px",
                wordBreak: "break-word",
                overflow: "auto",
                backgroundColor: palette.background.paper,
                color: palette.text.primary,
                position: "relative",
                ...linkColors(palette),
                ...sx,
            }}>
                <RichTextPlugin
                    contentEditable={<ContentEditable
                        className="editor-input"
                        // content={content ?? ""}
                        style={{
                            outline: "none",
                            border: "1px solid red",
                        } as CSSProperties}
                    />}
                    placeholder={
                        <div style={{
                            position: "absolute",
                            pointerEvents: "none",
                            top: 0,
                            left: 0,
                            padding: spacing(1),
                        }}
                        >
                            {placeholder ?? "Enter some text..."}
                        </div>
                    }
                    ErrorBoundary={LexicalErrorBoundary}
                />
                {/* <OnChangePlugin onChange={onChange} /> */}
                <HistoryPlugin />
                <RichInputTagDropdown {...tagData} selectDropdownItem={selectDropdownItem} />
                <MarkdownShortcutPlugin transformers={TRANSFORMERS} />
            </Box>
        </LexicalComposerContext.Provider>
    );
};
