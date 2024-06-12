import { createContext as createReactContext, useContext } from "react";
import { LexicalEditor } from "./editor";

export const LexicalComposerContext: React.Context<LexicalEditor | null> =
    createReactContext<LexicalEditor | null>(null);

export const useLexicalComposerContext = () => {
    const composerContext = useContext(LexicalComposerContext);
    return composerContext;
};
