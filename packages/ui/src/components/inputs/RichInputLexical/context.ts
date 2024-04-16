import { createContext as createReactContext, useContext } from "react";
import { LexicalEditor } from "./editor";
import { EditorThemeClasses } from "./types";

export type LexicalComposerContextType = {
    getTheme: () => EditorThemeClasses | null | undefined;
};

export type LexicalComposerContextWithEditor = [
    LexicalEditor,
    LexicalComposerContextType,
];

export const LexicalComposerContext: React.Context<
    LexicalComposerContextWithEditor | null | undefined
> = createReactContext<LexicalComposerContextWithEditor | null | undefined>(
    null,
);

export function createLexicalComposerContext(
    parent: LexicalComposerContextWithEditor | null | undefined,
    theme: EditorThemeClasses | null | undefined,
): LexicalComposerContextType {
    let parentContext: LexicalComposerContextType | null = null;

    if (parent != null) {
        parentContext = parent[1];
    }

    function getTheme() {
        if (theme != null) {
            return theme;
        }

        return parentContext != null ? parentContext.getTheme() : null;
    }

    return {
        getTheme,
    };
}

export const useLexicalComposerContext = () => {
    const composerContext = useContext(LexicalComposerContext);

    if (composerContext == null) {
        throw new Error("LexicalComposerContext.useLexicalComposerContext: cannot find a LexicalComposerContext");
    }

    return composerContext;
};
