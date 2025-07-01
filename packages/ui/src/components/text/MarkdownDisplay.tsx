/* eslint-disable import/extensions */

import Box from "@mui/material/Box";
import Checkbox from "@mui/material/Checkbox";
import CircularProgress from "@mui/material/CircularProgress";
import { IconButton } from "../buttons/IconButton.js";
import Link from "@mui/material/Link";
import { Tooltip } from "../Tooltip/Tooltip.js";
import Typography from "@mui/material/Typography";
import { styled } from "@mui/material/styles";
import { useTheme } from "@mui/material/styles";
import { nanoid, type ListObject } from "@vrooli/shared";
import { type HLJSApi } from "highlight.js";
import Markdown from "markdown-to-jsx";
import { Suspense, lazy, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { usePress, type UsePressEvent } from "../../hooks/gestures.js";
import { uiPathToApi, useLazyFetch } from "../../hooks/useFetch.js";
import { IconCommon, IconText } from "../../icons/Icons.js";
import { type SxType } from "../../types.js";
import { getDisplay } from "../../utils/display/listTools.js";
import { PubSub } from "../../utils/pubsub.js";
import { PopoverWithArrow } from "../dialogs/PopoverWithArrow/PopoverWithArrow.js";

const H_TAG_MAX = 6;
const H_TAG_DISPLAY_OFFSET = 2;

type HeadingLevel = "h1" | "h2" | "h3" | "h4" | "h5" | "h6";

type HeadingProps = {
    children: React.ReactNode;
    // eslint-disable-next-line no-magic-numbers
    level: 1 | 2 | 3 | 4 | 5 | 6;
};

function Heading({ children, level, ...props }: HeadingProps) {
    return (
        <Typography
            variant={`h${Math.min(level + H_TAG_DISPLAY_OFFSET, H_TAG_MAX)}` as HeadingLevel}
            component={`h${level}` as HeadingLevel}
            // eslint-disable-next-line no-magic-numbers
            marginTop={level === 1 ? 0 : 4}
            {...props}
        >
            {children}
        </Typography>
    );
}

type ParagraphProps = {
    children: React.ReactNode;
};

function Paragraph({ children, ...props }: ParagraphProps) {
    console.log("in Paragraph", props);
    return (
        <Typography
            variant="body1"
            component="p"
            marginBottom={2}
            {...props}
        >
            {children}
        </Typography>
    );
}

const StyledListItem = styled("li")({
    "& p": {
        marginBottom: 0,
    },
});
function ListItem({ children, ...props }) {
    console.log("in ListItem", props, children);
    return (
        <StyledListItem {...props}>
            {children}
        </StyledListItem>
    );
}

const CodeBlockOuter = styled("div")(({ theme }) => ({
    position: "relative",
    borderRadius: theme.spacing(2),
    boxShadow: theme.shadows[2],
    overflow: "hidden",
}));
const CodeBlockTopBar = styled(Box)(({ theme }) => ({
    display: "flex",
    alignItems: "center",
    padding: theme.spacing(1),
    backgroundColor: theme.palette.primary.dark,
}));
const CodeBlockTopBarButton = styled(IconButton)(({ theme }) => ({
    color: theme.palette.primary.contrastText,
    backgroundColor: theme.palette.primary.dark,
}));
const LanguageLabel = styled(Typography)(({ theme }) => ({
    color: theme.palette.primary.contrastText,
    flexGrow: 1,
    fontStyle: "italic",
}));
const CodeContainer = styled(Box)<{ isCollapsed?: boolean }>(({ theme, isCollapsed }) => ({
    position: "relative",
    maxHeight: isCollapsed ? "0px" : "none",
    overflow: "hidden",
    transition: "max-height 0.3s ease-in-out",
    "& pre": {
        margin: 0,
        padding: theme.spacing(2),
    },
    "& code": {
        fontFamily: "monospace",
        whiteSpace: isCollapsed ? "normal" : "pre",
        wordBreak: "break-word",
    },
}));

const CollapsedText = styled(Typography)(({ theme }) => ({
    padding: theme.spacing(1),
    color: theme.palette.text.secondary,
    fontStyle: "italic",
}));

// Module-level variable to store the Highlight.js module
let hljsModule: HLJSApi | null = null;
// Keep track of registered languages to avoid redundant registrations
const registeredLanguages = new Set<string>();

// Maps alternative language names to Highlight.js language names
const aliasMap: Record<string, string> = {
    "c++": "cpp",
    "c#": "csharp",
    html: "xml", // Highlight.js uses 'xml' for HTML highlighting
    js: "javascript",
    py: "python",
    rb: "ruby",
    sh: "shell",
    ts: "typescript",
    yml: "yaml",
};

// Limited to commonly used programming languages to reduce bundle size
const languageImportMap = {
    // Web languages (most common)
    javascript: () => import("highlight.js/lib/languages/javascript"),
    typescript: {
        importFunc: () => import("highlight.js/lib/languages/typescript"),
        dependencies: ["javascript"],
    },
    html: () => import("highlight.js/lib/languages/xml"), // HTML uses XML highlighting
    css: () => import("highlight.js/lib/languages/css"),
    scss: () => import("highlight.js/lib/languages/scss"),
    less: () => import("highlight.js/lib/languages/less"),
    json: () => import("highlight.js/lib/languages/json"),
    
    // Popular backend languages
    python: () => import("highlight.js/lib/languages/python"),
    java: () => import("highlight.js/lib/languages/java"),
    csharp: () => import("highlight.js/lib/languages/csharp"),
    cpp: {
        importFunc: () => import("highlight.js/lib/languages/cpp"),
        dependencies: ["c"],
    },
    c: () => import("highlight.js/lib/languages/c"),
    go: () => import("highlight.js/lib/languages/go"),
    rust: () => import("highlight.js/lib/languages/rust"),
    php: () => import("highlight.js/lib/languages/php"),
    ruby: () => import("highlight.js/lib/languages/ruby"),
    
    // Shell and config
    bash: () => import("highlight.js/lib/languages/bash"),
    shell: () => import("highlight.js/lib/languages/shell"),
    powershell: () => import("highlight.js/lib/languages/powershell"),
    dockerfile: () => import("highlight.js/lib/languages/dockerfile"),
    yaml: () => import("highlight.js/lib/languages/yaml"),
    ini: () => import("highlight.js/lib/languages/ini"),
    
    // Database
    sql: () => import("highlight.js/lib/languages/sql"),
    
    // Popular functional/modern languages
    haskell: () => import("highlight.js/lib/languages/haskell"),
    clojure: () => import("highlight.js/lib/languages/clojure"),
    scala: () => import("highlight.js/lib/languages/scala"),
    kotlin: () => import("highlight.js/lib/languages/kotlin"),
    swift: () => import("highlight.js/lib/languages/swift"),
    dart: () => import("highlight.js/lib/languages/dart"),
    
    // Data science
    r: () => import("highlight.js/lib/languages/r"),
    julia: () => import("highlight.js/lib/languages/julia"),
    matlab: () => import("highlight.js/lib/languages/matlab"),
    
    // Documentation
    markdown: () => import("highlight.js/lib/languages/markdown"),
    latex: () => import("highlight.js/lib/languages/latex"),
    xml: () => import("highlight.js/lib/languages/xml"),
    
    // Build tools
    makefile: () => import("highlight.js/lib/languages/makefile"),
    cmake: () => import("highlight.js/lib/languages/cmake"),
    gradle: () => import("highlight.js/lib/languages/gradle"),
    
    // Basic support
    plaintext: () => import("highlight.js/lib/languages/plaintext"),
    diff: () => import("highlight.js/lib/languages/diff"),
} as const;

// Lazy-load the TeX component and CSS when needed
const TeXComponent = lazy(async () => {
    await import("katex/dist/katex.min.css");
    const module = await import("@matejmazur/react-katex");
    return { default: module.default };
});

const preStyle = {
    padding: 0,
} as const;

function codeStyle(isHighlighted: boolean, isWrapped: boolean) {
    return {
        opacity: isHighlighted ? 1 : 0,
        whiteSpace: isWrapped ? "pre-wrap" : "pre",
    } as const;
}

/** Pretty code block with copy button */
function CodeBlock({ children, className }) {
    const { t } = useTranslation();
    const textRef = useRef<HTMLElement | null>(null);
    const [isHighlighted, setIsHighlighted] = useState(false);
    const [isCollapsed, setIsCollapsed] = useState(false);
    const [isWrapped, setIsWrapped] = useState(false);
    const [lineCount, setLineCount] = useState(0);

    // Extract language from className
    const language = className ? className.replace("language-", "").replace("lang-", "") : "";

    // Ensure children is a string
    const codeString = String(children).trim();

    // Check if the language is LaTeX
    const isLatex = language === "latex" || language === "tex";

    // Count lines
    useEffect(() => {
        const lines = codeString.split("\n").length;
        setLineCount(lines);
    }, [codeString]);

    useEffect(function loadAndApplyHighlighting() {
        if (isLatex) return; // Skip Highlight.js logic for LaTeX

        let isMounted = true;

        async function loadLanguage(lang: string) {
            if (!hljsModule) {
                console.warn("Highlight.js is not loaded.");
                return;
            }
            if (registeredLanguages.has(lang)) {
                // Language is already registered
                return;
            }

            const languageInfo = languageImportMap[lang];

            if (!languageInfo) {
                console.warn(`Language '${lang}' is not supported.`);
                throw new Error(`Language '${lang}' is not supported.`);
            }

            let importFunc = languageInfo;
            // Load dependencies first
            if (
                typeof languageInfo !== "function"
                && Object.prototype.hasOwnProperty.call(languageInfo, "importFunc")
                && Object.prototype.hasOwnProperty.call(languageInfo, "dependencies")
                && languageInfo.dependencies.length > 0
            ) {
                importFunc = languageInfo.importFunc;
                for (const dep of languageInfo.dependencies) {
                    await loadLanguage(dep);
                }
            }

            // Import and register the language
            const langModule = await importFunc();
            hljsModule.registerLanguage(lang, langModule.default || langModule);
            registeredLanguages.add(lang);
        }

        async function loadHighlightJs() {
            // Load Highlight.js core if not already loaded
            if (!hljsModule) {
                const hljs = await import("highlight.js/lib/core");
                hljsModule = hljs.default || hljs;
                // Load the CSS only once
                import("highlight.js/styles/monokai-sublime.css");
            }

            // Check if the component is ready to load
            const isReadyToLoad = textRef.current && isMounted && hljsModule;
            if (!isReadyToLoad) return;

            // Determine the language to use
            let lang = language;
            lang = aliasMap[lang] ?? lang;
            if (languageImportMap[lang]) {
                await loadLanguage(lang);
            } else {
                await loadLanguage("plaintext");
            }

            hljsModule.highlightElement(textRef.current as HTMLElement);
            setIsHighlighted(true);
        }

        loadHighlightJs();

        return () => {
            isMounted = false;
        };
    }, [codeString, isLatex, language]);

    const copyCode = useCallback(() => {
        if (isLatex) {
            navigator.clipboard.writeText(codeString);
        } else if (textRef && textRef.current) {
            navigator.clipboard.writeText(textRef.current.textContent ?? "");
        }
        PubSub.get().publish("snack", { messageKey: "CopiedToClipboard", severity: "Success" });
    }, [codeString, isLatex]);

    const toggleCollapse = useCallback(() => setIsCollapsed(prev => !prev), []);
    const toggleWrap = useCallback(() => setIsWrapped(prev => !prev), []);

    return (
        <CodeBlockOuter>
            <Box position="sticky" top={0} zIndex={1}>
                <Box display="flex" gap={1} position="absolute" right={1}>
                    <Tooltip title={isWrapped ? t("WordWrapDisable") : t("WordWrapEnable")}>
                        <CodeBlockTopBarButton
                            aria-label={isWrapped ? t("WordWrapDisable") : t("WordWrapEnable")}
                            size="sm"
                            variant="transparent"
                            onClick={toggleWrap}
                        >
                            {
                                isWrapped
                                    ? <IconText
                                        decorative
                                        name="List"
                                    />
                                    : <IconText
                                        decorative
                                        name="WrapText"
                                    />
                            }
                        </CodeBlockTopBarButton>
                    </Tooltip>
                    <Tooltip title={t("Copy")}>
                        <CodeBlockTopBarButton
                            aria-label={t("Copy")}
                            size="sm"
                            variant="transparent"
                            onClick={copyCode}
                        >
                            <IconCommon
                                decorative
                                name="Copy"
                            />
                        </CodeBlockTopBarButton>
                    </Tooltip>
                    <Tooltip title={isCollapsed ? t("Expand") : t("Collapse")}>
                        <CodeBlockTopBarButton
                            aria-label={isCollapsed ? t("Expand") : t("Collapse")}
                            size="sm"
                            variant="transparent"
                            onClick={toggleCollapse}
                        >
                            {
                                isCollapsed
                                    ? <IconCommon
                                        decorative
                                        name="ExpandMore"
                                    />
                                    : <IconCommon
                                        decorative
                                        name="ExpandLess"
                                    />
                            }
                        </CodeBlockTopBarButton>
                    </Tooltip>
                </Box>
            </Box>
            <CodeBlockTopBar>
                <LanguageLabel variant="body2">{language || "text"}</LanguageLabel>
            </CodeBlockTopBar>
            {isLatex ? (
                <Suspense fallback={<div>Loading...</div>}>
                    <div>
                        <TeXComponent block>{codeString}</TeXComponent>
                    </div>
                </Suspense>
            ) : (
                <>
                    <CodeContainer isCollapsed={isCollapsed}>
                        <pre style={preStyle}>
                            <code
                                ref={textRef}
                                className={className}
                                style={codeStyle(isHighlighted, isWrapped)}
                            >
                                {codeString}
                            </code>
                        </pre>
                    </CodeContainer>
                    {isCollapsed && (
                        <CollapsedText>
                            {lineCount} lines collapsed
                        </CollapsedText>
                    )}
                </>
            )}
        </CodeBlockOuter>
    );
}

const blockquoteStyle = {
    position: "relative",
    paddingLeft: "1.5em",
    marginLeft: "1em",
    borderLeft: "3px solid #ccc",
} as const;
/** Custom Blockquote component */
function Blockquote({ children }) {
    return (
        <blockquote style={blockquoteStyle}>
            {children}
        </blockquote>
    );
}


/**
 * Preprocess Markdown text to:
 * 1. Replace single newline characters with double newlines, except those inside code blocks.
 * 2. Convert custom `||spoiler||` syntax to an HTML-like `<spoiler></spoiler>` for further rendering.
 * These adjustments make the behavior of Markdown more intuitive for users and add custom functionality.
 *
 * @param {string} content - The input Markdown text.
 * @returns {string} - The processed Markdown text.
 */
function processMarkdown(content: string): string {
    // Initialize state variables
    let isInCodeBlock = false;
    let result = "";

    // Convert ||spoiler|| to <spoiler>spoiler</spoiler>
    content = content.replace(/\|\|([\s\S]+?)\|\|/g, "<spoiler>$1</spoiler>");

    // Iterate over each character in the processed content
    for (let i = 0; i < content.length; i++) {
        if (content[i] === "\n") {
            // If it's a newline not preceded by a newline and we're not inside a code block, add an extra newline
            if (content[i - 1] !== "\n" && !isInCodeBlock) {
                result += "\n";
            }
            // Add the newline itself
            result += "\n";
        } else if (content[i] === "`") {
            // If it's a backtick and the two preceding characters are also backticks, toggle the isInCodeBlock flag.
            // This assumes that code blocks are initiated and closed with triple backticks (```).
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
}

/** Creates custom links for Vrooli objects, and normal links otherwise */
function CustomLink({ children, href }) {
    // Check if this is a special link
    const { isInternal, apiUrl } = useMemo(() => {
        try {
            const link = new URL(href, window.location.href);
            const here = window.location;
            const isInternal = link.host === here.host;
            if (!isInternal) return { isInternal, apiUrl: null };
            return { isInternal: true, apiUrl: uiPathToApi(link.pathname) };
        } catch {
            return { isInternal: false, apiUrl: null };
        }
    }, [href]);

    // Fetch hook
    const [getData, { data, loading: isLoading }] = useLazyFetch<undefined, unknown>({ endpoint: undefined, method: "GET" });

    // Get display data
    const { title, subtitle } = getDisplay(data as unknown as ListObject, ["en"]);

    // Popover to display more info
    const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(null);
    const open = useCallback(({ target }: UsePressEvent) => {
        if (!apiUrl) return;
        setAnchorEl(target as HTMLElement);
        getData(undefined, {
            endpointOverride: apiUrl,
            onError: () => PubSub.get().publish("snack", { message: "Invalid URL", severity: "Error" }),
        });
    }, [getData, apiUrl]);
    const close = useCallback(() => setAnchorEl(null), []);

    const pressEvents = usePress({
        onHover: open,
        onLongPress: open,
        onClick: open,
    });

    if (isInternal) {
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
}

/** HOC for rendering links */
function withCustomLinkProps(additionalProps) {
    return function withCustomLinkPropsFunc({ href, children }) {
        return <CustomLink href={href} {...additionalProps}>{children}</CustomLink>;
    };
}

/** Custom checkbox component editable checkboxes */
function CustomCheckbox({ checked, onChange, ...otherProps }) {
    const id = useMemo(() => nanoid(), []);
    function toggleCheckbox() {
        onChange(id, !checked);
    }
    return <Checkbox checked={checked} id={id} onChange={toggleCheckbox} {...otherProps} />;
}

/** HOC for rendering inputs. Required so we can pass onChange handler */
function withCustomCheckboxProps(additionalProps) {
    return function withCustomCheckboxPropsFunc({ type, checked, onChange }) {
        if (type === "checkbox") {
            return <CustomCheckbox checked={checked} onChange={onChange} {...additionalProps} />;
        }
        return null;
    };
}

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

const spoilerStyles = {
    cursor: "pointer",
    transition: "color 0.4s, background 0.4s",
};
const revealedStyles = {
    color: "inherit",
    background: "rgba(0, 0, 0, 0.3)",
};
const hiddenStyles = {
    color: "transparent",
    background: "black",
};
function Spoiler({ children }) {
    const [revealed, setRevealed] = useState(false);
    const currentStyles = revealed ? revealedStyles : hiddenStyles;
    return (
        <span
            style={{ ...spoilerStyles, ...currentStyles }}
            onClick={() => setRevealed(!revealed)}
        >
            {children}
        </span>
    );
}

export function MarkdownDisplay({
    content,
    isEditable,
    onChange,
    sx,
    headingLevelOffset = 0,
}: {
    content: string | undefined;
    isEditable?: boolean;
    onChange?: (content: string) => unknown;
    sx?: SxType;
    headingLevelOffset?: number;
}) {
    const { palette, typography } = useTheme();
    const id = useMemo(() => nanoid(), []);

    // Add overrides for custom components
    const options = useMemo(function optionsMemo() {
        return {
            overrides: {
                code: CodeBlock,
                blockquote: Blockquote,
                a: withCustomLinkProps({}),
                spoiler: {
                    component: Spoiler,
                },
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
                h1: { component: Heading, props: { level: Math.max(1, Math.min(6, 1 + headingLevelOffset)) } },
                h2: { component: Heading, props: { level: Math.max(1, Math.min(6, 2 + headingLevelOffset)) } },
                h3: { component: Heading, props: { level: Math.max(1, Math.min(6, 3 + headingLevelOffset)) } },
                h4: { component: Heading, props: { level: Math.max(1, Math.min(6, 4 + headingLevelOffset)) } },
                h5: { component: Heading, props: { level: Math.max(1, Math.min(6, 5 + headingLevelOffset)) } },
                h6: { component: Heading, props: { level: Math.max(1, Math.min(6, 6 + headingLevelOffset)) } },
                p: Paragraph,
                li: ListItem,
            },
        };
    }, [content, headingLevelOffset, id, onChange]);

    // Preprocess the Markdown content
    const processedContent = processMarkdown(content ?? "");

    const markdownStyle = useMemo(function markdownStyleMemo() {
        return {
            fontFamily: typography.fontFamily,
            fontSize: typography.fontSize + 2,
            lineHeight: `${Math.round(typography.fontSize * 1.5)}px`,
            color: "inherit",
            display: "block",
            ...sx,
        } as const;
    }, [sx, typography.fontSize, typography.fontFamily]);

    return (
        <Markdown id={id} options={options} style={markdownStyle}>
            {processedContent}
        </Markdown>
    );
}
