import { createContext as createReactContext, useContext } from "react";
import { LexicalEditor } from "./editor";

export const LexicalComposerContext: React.Context<LexicalEditor | null> =
    createReactContext<LexicalEditor | null>(null);

export const useLexicalComposerContext = () => {
    const composerContext = useContext(LexicalComposerContext);

    if (composerContext == null) {
        throw new Error("LexicalComposerContext.useLexicalComposerContext: cannot find a LexicalComposerContext");
    }

    return composerContext;
};
