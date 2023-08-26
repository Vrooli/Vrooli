import { CodeNode } from "@lexical/code";
import { LinkNode } from "@lexical/link";
import { ListItemNode, ListNode } from "@lexical/list";
import { $convertFromMarkdownString, TRANSFORMERS } from "@lexical/markdown";
import { LexicalComposer } from "@lexical/react/LexicalComposer";
import { ContentEditable } from "@lexical/react/LexicalContentEditable";
import LexicalErrorBoundary from "@lexical/react/LexicalErrorBoundary";
import { HistoryPlugin } from "@lexical/react/LexicalHistoryPlugin";
import { HorizontalRuleNode } from "@lexical/react/LexicalHorizontalRuleNode";
import { MarkdownShortcutPlugin } from "@lexical/react/LexicalMarkdownShortcutPlugin";
import { RichTextPlugin } from "@lexical/react/LexicalRichTextPlugin";
import { HeadingNode, QuoteNode } from "@lexical/rich-text";
import { Box, useTheme } from "@mui/material";
import "highlight.js/styles/monokai-sublime.css";
import { CSSProperties, useCallback } from "react";
import { linkColors } from "styles";
import { ListObject } from "utils/display/listTools";
import { RichInputTagDropdown, useTagDropdown } from "../RichInputTagDropdown/RichInputTagDropdown";
import { RichInputAction } from "../RichInputToolbar/RichInputToolbar";
import { RichInputChildView, RichInputLexicalProps } from "../types";


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

    const theme = {

    };

    const onError = (error: Error) => {
        console.error(error);
    };

    const initialConfig = {
        namespace: "MyEditor",
        // To find missing nodes, check each plugin's file in the lexical repo for the "dependencies" array
        nodes: [CodeNode, HeadingNode, HorizontalRuleNode, LinkNode, ListNode, ListItemNode, QuoteNode],
        theme,
        onError,
        editorState: () => $convertFromMarkdownString(value, TRANSFORMERS),
    };

    const tagData = useTagDropdown({ getTaggableItems });
    const selectDropdownItem = useCallback((item: ListObject) => {
        //TODO
    }, []);

    (RichInputLexical as unknown as RichInputChildView).handleAction = (action: RichInputAction) => {
        //TODO
        const actionMap: { [key in RichInputAction]: (() => unknown) } = {
            "Assistant": () => openAssistantDialog(""),
            "Bold": () => { },
            "Header1": () => { },
            "Header2": () => { },
            "Header3": () => { },
            "Italic": () => { },
            "Link": () => { },
            "ListBullet": () => { },
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
        <LexicalComposer initialConfig={initialConfig}>
            <Box className="editor-container" sx={{
                border: `1px solid ${error ? "red" : "black"}`,
                borderRadius: "0 0 0.5rem 0.5rem",
                padding: "12px",
                wordBreak: "break-word",
                overflow: "auto",
                backgroundColor: palette.background.paper,
                color: palette.text.primary,
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
        </LexicalComposer>
    );
};
