import { CopyIcon, endpointGetApi, endpointGetChat, endpointGetComment, endpointGetNote, endpointGetOrganization, endpointGetProject, endpointGetQuestion, endpointGetQuiz, endpointGetReport, endpointGetRoutine, endpointGetSmartContract, endpointGetStandard, endpointGetTag, endpointGetUser, exists, LINKS, uuid } from "@local/shared";
import { Box, Checkbox, CircularProgress, IconButton, Link, useTheme } from "@mui/material";
import { PopoverWithArrow } from "components/dialogs/PopoverWithArrow/PopoverWithArrow";
import hljs from "highlight.js";
import "highlight.js/styles/monokai-sublime.css";
import Markdown from "markdown-to-jsx";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
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

/** Custom Blockquote component */
const Blockquote = ({ children }) => {
    return (
        <blockquote style={{
            position: "relative",
            paddingLeft: "1.5em",
            marginLeft: "1em",
            borderLeft: "3px solid #ccc",
        }}>
            {children}
        </blockquote>
    );
};


/**
 * Preprocess Markdown text to replace single newline characters with double newlines, 
 * except those inside code blocks. This makes the behavior of Markdown 
 * more intuitive for users.
 *
 * @param {string} content - The input Markdown text.
 * @returns {string} - The processed Markdown text.
 */
const processMarkdown = (content: string): string => {
    // Initialize state variables
    let isInCodeBlock = false;
    let result = "";
    // Iterate over each character in the input
    for (let i = 0; i < content.length; i++) {
        if (content[i] === "\n") {
            // If it's a newline not preceded by a newline and we're not inside a code block, add an extra newline
            if (content[i - 1] !== "\n" && !isInCodeBlock) {
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
        } else {
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
const CustomLink = ({ children, href, zIndex }) => {
    // Check if this is a special link
    let linkUrl, windowUrl;
    try {
        linkUrl = new URL(href);
        windowUrl = new URL(window.location.href);
    } catch (_) {
        console.error("CustomLink failed to parse url", href);
    }

    const matchingRoute: string | undefined = linkUrl ? specialRoutes.find(route => linkUrl.pathname.startsWith(route)) : undefined;
    const isSpecialLink: boolean = linkUrl && linkUrl.hostname === windowUrl.hostname && matchingRoute !== undefined;
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
                    zIndex={zIndex + 1}
                >
                    <Box p={2}>
                        {isLoading
                            ? <CircularProgress />
                            : <>
                                <Link href={href}><strong>{title}</strong></Link>
                                <br />
                                <MarkdownDisplay content={subtitle} zIndex={zIndex + 1} />
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

/** HOC for rendering links. Required so we can pass zIndex */
const withCustomLinkProps = (additionalProps) => {
    return ({ href, children }) => {
        return <CustomLink href={href} {...additionalProps}>{children}</CustomLink>;
    };
};

/** Custom checkbox component editable checkboxes */
const CustomCheckbox = ({ checked, onChange, ...otherProps }) => {
    const id = useMemo(() => uuid(), []);
    return <Checkbox checked={checked} id={id} onChange={() => { onChange(id, !checked); }} {...otherProps} />;
};

/** HOC for rendering inputs. Required so we can pass onChange handler */
const withCustomCheckboxProps = (additionalProps) => {
    return ({ type, checked, onChange }) => {
        if (type === "checkbox") {
            return <CustomCheckbox checked={checked} onChange={onChange} {...additionalProps} />;
        }
        return null;
    };
};

/** State machine to locate checkboxes */
function parseMarkdownCheckboxes(content: string) {
    const STATE_NORMAL = 0;
    const STATE_CODE_BLOCK = 1;

    let state = STATE_NORMAL;
    const checkboxIndices: number[] = [];
    const potentialCheckboxIndices: number[] = [];

    // use a buffer to keep track of significant characters
    let buffer = "";

    for (let i = 0; i < content.length; i++) {
        const char = content[i];

        // update state based on buffer content
        if (buffer.endsWith("```")) {
            state = state === STATE_CODE_BLOCK ? STATE_NORMAL : STATE_CODE_BLOCK;
            buffer = ""; // reset buffer
            if (state === STATE_NORMAL) {
                potentialCheckboxIndices.length = 0; // clear the list of potential checkboxes
            }
        }

        // update buffer
        buffer += char;
        if (buffer.length > 3) {
            buffer = buffer.slice(1); // keep buffer at most 3 characters
        }

        // record checkbox start index
        if (buffer === "[ ]" || buffer === "[x]") {
            if (state === STATE_NORMAL) {
                checkboxIndices.push(i - 2);
            } else {
                potentialCheckboxIndices.push(i - 2);
            }
        }
    }

    // if the text ends while still inside a block, consider the potential checkboxes as actual ones
    if (state !== STATE_NORMAL) {
        checkboxIndices.push(...potentialCheckboxIndices);
    }

    return checkboxIndices;
}



export const MarkdownDisplay = ({
    content,
    isEditable,
    onChange,
    sx,
    variant, //TODO
    zIndex,
}: MarkdownDisplayProps) => {
    const { palette, typography } = useTheme();
    const id = useMemo(() => uuid(), []);

    // Add overrides for custom components
    const options = {
        overrides: {
            code: CodeBlock,
            blockquote: Blockquote,
            a: withCustomLinkProps({ zIndex }),
            input: withCustomCheckboxProps({
                onChange: (checkboxId: string, updatedState: boolean) => {
                    if (!content || !onChange) return;
                    // Find location of each checkbox in rendered markdown. Used to find corresponding checkbox in markdown string
                    const markdownComponent = document.getElementById(id);
                    if (!markdownComponent) return;
                    // Use a tree walker to find all checkboxes
                    const treeWalker = document.createTreeWalker(
                        markdownComponent,
                        NodeFilter.SHOW_ELEMENT,
                        {
                            acceptNode: (node: Node) => {
                                // Check if the node is an input element before checking its type
                                if ((node as HTMLInputElement).nodeName === "INPUT" && (node as HTMLInputElement).type === "checkbox") {
                                    return NodeFilter.FILTER_ACCEPT;
                                } else {
                                    return NodeFilter.FILTER_SKIP;
                                }
                            },
                        },
                    );
                    const checkboxes: Node[] = [];
                    while (treeWalker.nextNode()) {
                        checkboxes.push(treeWalker.currentNode);
                    }
                    // Extract id from each checkbox, so we know the order of the checkboxes in the markdown string
                    const checkboxIds = checkboxes.map(checkbox => checkbox.id);
                    // Find the index of the checkbox that was clicked
                    const checkboxIndex = checkboxIds.findIndex(cId => cId === checkboxId);
                    // Find location of each checkbox in content (i.e. plaintext), both checked and unchecked
                    const checkboxLocations = parseMarkdownCheckboxes(content);
                    if (checkboxIndex >= checkboxLocations.length) {
                        console.error("Checkbox index out of range. Checkboxes:", checkboxes, "Checkbox index:", checkboxIndex, "Checkbox locations:", checkboxLocations);
                        return;
                    }
                    // Replace the checkbox in the content with the updated checkbox
                    const checkboxStart = checkboxLocations[checkboxIndex];
                    const newCheckbox = updatedState ? "[x]" : "[ ]";
                    const newContent = content.substring(0, checkboxStart) + newCheckbox + content.substring(checkboxStart + 3);
                    onChange(newContent);
                },
            }),
        },
    };

    // Preprocess the Markdown content
    const processedContent = processMarkdown(content ?? "");

    return (
        <Markdown id={id} options={options} style={{
            fontFamily: typography.fontFamily,
            fontSize: typography.fontSize + 2,
            lineHeight: `${Math.round(typography.fontSize * 1.5)}px`,
            color: palette.background.textPrimary,
            ...sx,
        }}>
            {processedContent}
        </Markdown>
    );
};
