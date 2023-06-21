import { CopyIcon, endpointGetApi, endpointGetChat, endpointGetComment, endpointGetNote, endpointGetOrganization, endpointGetProject, endpointGetQuestion, endpointGetQuiz, endpointGetReport, endpointGetRoutine, endpointGetSmartContract, endpointGetStandard, endpointGetTag, endpointGetUser, exists, LINKS } from "@local/shared";
import { Box, CircularProgress, IconButton, Link } from "@mui/material";
import { PopoverWithArrow } from "components/dialogs/PopoverWithArrow/PopoverWithArrow";
import hljs from "highlight.js";
import "highlight.js/styles/monokai-sublime.css";
import Markdown from "markdown-to-jsx";
import { useCallback, useEffect, useRef, useState } from "react";
import { getDisplay } from "utils/display/listTools";
import { useDisplayServerError } from "utils/hooks/useDisplayServerError";
import { useLazyFetch } from "utils/hooks/useLazyFetch";
import usePress from "utils/hooks/usePress";
import { parseSingleItemUrl } from "utils/navigation/urlTools";
import { PubSub } from "utils/pubsub";
import { MarkdownDisplayProps } from "../types";

/** Pretty code block with copy button */
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

// Vrooli pages that show up as special links
const specialRoutes = ["Api", "Chat", "Comment", "Note", "Organization", "Project", "Question", "Quiz", "Report", "Routine", "SmartContract", "Standard", "Tag", "User"].map(key => LINKS[key]);

// Maps URL slugs to endpoints
const routeToEndpoint = {
    [LINKS.Api]: endpointGetApi,
    [LINKS.Chat]: endpointGetChat,
    [LINKS.Comment]: endpointGetComment,
    [LINKS.Note]: endpointGetNote,
    [LINKS.Organization]: endpointGetOrganization,
    [LINKS.Project]: endpointGetProject,
    [LINKS.Question]: endpointGetQuestion,
    [LINKS.Quiz]: endpointGetQuiz,
    [LINKS.Report]: endpointGetReport,
    [LINKS.Routine]: endpointGetRoutine,
    [LINKS.SmartContract]: endpointGetSmartContract,
    [LINKS.Standard]: endpointGetStandard,
    [LINKS.Tag]: endpointGetTag,
    [LINKS.User]: endpointGetUser,
};

/** Creates custom links for Vrooli objects, and normal links otherwise */
const CustomLink = ({ children, href }) => {
    // Check if this is a special link
    let url;
    try {
        url = new URL(href);
    } catch (_) { /* empty */ }

    const matchingRoute: string | undefined = url ? specialRoutes.find(route => url.pathname.startsWith(route)) : undefined;
    const isSpecialLink: boolean = url.hostname === new URL(window.location.href).hostname && matchingRoute !== undefined;
    const endpoint = (isSpecialLink && matchingRoute) ? routeToEndpoint[matchingRoute] : null;

    // Fetch hook
    const [getData, { data, loading: isLoading, errors }] = useLazyFetch<any, any>(endpoint ?? endpointGetUser);
    useDisplayServerError(errors);

    // Get display data
    const { title, subtitle } = getDisplay(data, ["en"]);

    // Popover to display more info
    const [anchorEl, setAnchorEl] = useState<any | null>(null);
    const open = useCallback((target: EventTarget) => {
        setAnchorEl(target);
        const urlParams = parseSingleItemUrl({ url: href });
        if (exists(urlParams.handle)) getData({ handle: urlParams.handle });
        else if (exists(urlParams.handleRoot)) getData({ handleRoot: urlParams.handleRoot });
        else if (exists(urlParams.id)) getData({ id: urlParams.id });
        else if (exists(urlParams.idRoot)) getData({ idRoot: urlParams.idRoot });
        else PubSub.get().publishSnack({ message: "Invalid URL", severity: "Error" });
    }, [getData, href]);
    const close = useCallback(() => setAnchorEl(null), []);

    const pressEvents = usePress({
        onHover: open,
        onLongPress: open,
        onClick: open,
    });

    if (isSpecialLink) {
        return (
            <>
                <Link
                    href={href}
                    {...pressEvents}
                    sx={{
                        backgroundColor: "#f8f9fa",
                        border: "1px solid #ced4da",
                        borderRadius: "4px",
                        padding: "2px",
                    }}
                >
                    {children}
                </Link>
                <PopoverWithArrow
                    anchorEl={anchorEl}
                    handleClose={close}
                >
                    <Box p={2}>
                        {isLoading
                            ? <CircularProgress />
                            : <>
                                <Link href={href}><strong>{title}</strong></Link>
                                <br />
                                <MarkdownDisplay content={subtitle} />
                            </>
                        }
                    </Box>
                </PopoverWithArrow>
            </>
        );
    } else {
        return <Link href={href}>{children}</Link>;
    }
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
            a: CustomLink,
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
