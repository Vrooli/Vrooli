import hljs from "highlight.js";
import "highlight.js/styles/monokai-sublime.css";
import Markdown from "markdown-to-jsx";
import { useEffect } from "react";
import { MarkdownDisplayProps } from "../types";

export const MarkdownDisplay = ({
    content,
    sx,
    variant, //TODO
}: MarkdownDisplayProps) => {
    useEffect(() => {
        document.querySelectorAll("pre code").forEach((block) => {
            hljs.highlightBlock(block as HTMLElement);
        });
    }, [content]);

    return (
        <Markdown style={{ ...sx }}>
            {content ?? ""}
        </Markdown>
    );
};
