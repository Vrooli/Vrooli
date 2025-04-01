import { createContext as createReactContext, useContext } from "react";
import { LexicalEditor } from "./editor.js";

export const LexicalComposerContext: React.Context<LexicalEditor | null> =
    createReactContext<LexicalEditor | null>(null);

export function useLexicalComposerContext() {
    const composerContext = useContext(LexicalComposerContext);
    return composerContext;
}
