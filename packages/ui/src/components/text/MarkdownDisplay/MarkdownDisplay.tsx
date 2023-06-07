import { CopyIcon } from "@local/shared";
import { IconButton } from "@mui/material";
import hljs from "highlight.js";
import "highlight.js/styles/monokai-sublime.css";
import Markdown from "markdown-to-jsx";
import { useEffect, useRef } from "react";
import { PubSub } from "utils/pubsub";
import { MarkdownDisplayProps } from "../types";

/**
 * Pretty code block, with copy button 
 */
const CodeBlock = ({ children }) => {
    const textRef = useRef<HTMLElement | null>(null);

    useEffect(() => {
        if (textRef && textRef.current) {
            hljs.highlightBlock(textRef.current);
        }
    }, []);

    const copyCode = () => {
        if (textRef && textRef.current) {
            // Copy the text content of the code block
            navigator.clipboard.writeText(textRef.current.textContent ?? "");
            PubSub.get().publishSnack({ messageKey: "CopiedToClipboard", severity: "Success" });
        }
    };

    return (
        <div style={{ position: "relative" }}>
            <IconButton onClick={copyCode} sx={{
                position: "absolute",
                top: "0px",
                right: "0px",
            }}>
                <CopyIcon fill="white" />
            </IconButton>
            <pre>
                <code
                    ref={textRef}
                    style={{ paddingRight: "40px" }}
                >{children}</code>
            </pre>
        </div>
    );
};

/**
 * Preprocess Markdown text to replace single newline characters with double newlines, 
 * except those inside code blocks and blockquotes. This makes the behavior of Markdown 
 * more intuitive for users.
 *
 * @param {string} content - The input Markdown text.
 * @returns {string} - The processed Markdown text.
 */
const processMarkdown = (content: string): string => {
    // Initialize state variables
    let isInCodeBlock = false;
    let isInQuote = false;
    let result = "";
    // Iterate over each character in the input
    for (let i = 0; i < content.length; i++) {
        if (content[i] === "\n") {
            // If it's a newline not preceded by a newline and we're not inside a code block or quote, add an extra newline
            if (content[i - 1] !== "\n" && !isInCodeBlock && !isInQuote) {
                result += "\n";
            }
            // Add the newline itself
            result += "\n";
        } else if (content[i] === "`") {
            // If it's a backtick and the two preceding characters are also backticks, toggle the isInCodeBlock flag
            if (content[i - 1] === "`" && content[i - 2] === "`") {
                isInCodeBlock = !isInCodeBlock;
            }
            // Add the backtick itself
            result += "`";
        } else if (content[i] === ">") {
            // If it's a '>' and it's the start of a line (preceded by a newline or at the start of the text), set isInQuote to true
            if (content[i - 1] === "\n" || i === 0) {
                isInQuote = true;
            }
            // Add the '>' itself
            result += ">";
        } else {
            // If it's a newline and we're inside a quote, set isInQuote to false
            if (content[i] === "\n" && isInQuote) {
                isInQuote = false;
            }
            // Add the character itself
            result += content[i];
        }
    }
    return result;
};

export const MarkdownDisplay = ({
    content,
    sx,
    variant, //TODO
}: MarkdownDisplayProps) => {
    // Add overrides for custom components
    const options = {
        overrides: {
            code: CodeBlock,
        },
    };

    // Preprocess the Markdown content
    const processedContent = processMarkdown(content ?? "");

    return (
        <Markdown options={options} style={{ ...sx }}>
            {processedContent}
        </Markdown>
    );
};
