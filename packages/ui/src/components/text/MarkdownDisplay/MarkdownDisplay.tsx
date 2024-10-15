import { endpointGetApi, endpointGetChat, endpointGetCode, endpointGetComment, endpointGetNote, endpointGetProject, endpointGetQuestion, endpointGetQuiz, endpointGetReport, endpointGetRoutine, endpointGetStandard, endpointGetTag, endpointGetTeam, endpointGetUser, exists, LINKS, uuid } from "@local/shared";
import { Box, Checkbox, CircularProgress, IconButton, Link, styled, Typography, TypographyProps, useTheme } from "@mui/material";
import { PopoverWithArrow } from "components/dialogs/PopoverWithArrow/PopoverWithArrow";
import { type HLJSApi } from "highlight.js";
import { usePress, UsePressEvent } from "hooks/gestures";
import { useLazyFetch } from "hooks/useLazyFetch";
import { CopyIcon } from "icons";
import Markdown from "markdown-to-jsx";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { SxType } from "types";
import { getDisplay } from "utils/display/listTools";
import { parseSingleItemUrl } from "utils/navigation/urlTools";
import { PubSub } from "utils/pubsub";

type HeadingLevel = "h1" | "h2" | "h3" | "h4" | "h5" | "h6";

type HeadingProps = {
    children: string;
    // eslint-disable-next-line no-magic-numbers
    level: 1 | 2 | 3 | 4 | 5 | 6;
};

function Heading({ children, level, ...props }: HeadingProps) {
    return (
        <Typography
            variant={`h${Math.min(level + 2, 6)}` as HeadingLevel}
            component={`h${level}` as HeadingLevel}
            marginTop={level === 1 ? 0 : 4}
            {...props}
        >
            {children}
        </Typography>
    );
}

const CodeBlockOuter = styled("div")(() => ({
    position: "relative",
}));

const CopyButton = styled(IconButton)(() => ({
    position: "absolute",
    top: "0px",
    right: "0px",
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

const languageImportMap = {
    "1c": () => import("highlight.js/lib/languages/1c"),
    abnf: () => import("highlight.js/lib/languages/abnf"),
    accesslog: () => import("highlight.js/lib/languages/accesslog"),
    actionscript: () => import("highlight.js/lib/languages/actionscript"),
    ada: () => import("highlight.js/lib/languages/ada"),
    angelscript: () => import("highlight.js/lib/languages/angelscript"),
    apache: () => import("highlight.js/lib/languages/apache"),
    applescript: () => import("highlight.js/lib/languages/applescript"),
    arcade: () => import("highlight.js/lib/languages/arcade"),
    arduino: () => import("highlight.js/lib/languages/arduino"),
    armasm: () => import("highlight.js/lib/languages/armasm"),
    asciidoc: () => import("highlight.js/lib/languages/asciidoc"),
    aspectj: () => import("highlight.js/lib/languages/aspectj"),
    autohotkey: () => import("highlight.js/lib/languages/autohotkey"),
    autoit: () => import("highlight.js/lib/languages/autoit"),
    avrasm: () => import("highlight.js/lib/languages/avrasm"),
    awk: () => import("highlight.js/lib/languages/awk"),
    axapta: () => import("highlight.js/lib/languages/axapta"),
    bash: () => import("highlight.js/lib/languages/bash"),
    basic: () => import("highlight.js/lib/languages/basic"),
    bnf: () => import("highlight.js/lib/languages/bnf"),
    brainfuck: () => import("highlight.js/lib/languages/brainfuck"),
    c: () => import("highlight.js/lib/languages/c"),
    cal: () => import("highlight.js/lib/languages/cal"),
    capnproto: () => import("highlight.js/lib/languages/capnproto"),
    ceylon: () => import("highlight.js/lib/languages/ceylon"),
    clean: () => import("highlight.js/lib/languages/clean"),
    clojure: () => import("highlight.js/lib/languages/clojure"),
    "clojure-repl": () => import("highlight.js/lib/languages/clojure-repl"),
    cmake: () => import("highlight.js/lib/languages/cmake"),
    coffeescript: () => import("highlight.js/lib/languages/coffeescript"),
    coq: () => import("highlight.js/lib/languages/coq"),
    cos: () => import("highlight.js/lib/languages/cos"),
    cpp: {
        importFunc: () => import("highlight.js/lib/languages/cpp"),
        dependencies: ["c"],
    },
    crmsh: () => import("highlight.js/lib/languages/crmsh"),
    crystal: () => import("highlight.js/lib/languages/crystal"),
    csharp: () => import("highlight.js/lib/languages/csharp"),
    csp: () => import("highlight.js/lib/languages/csp"),
    css: () => import("highlight.js/lib/languages/css"),
    d: () => import("highlight.js/lib/languages/d"),
    dart: () => import("highlight.js/lib/languages/dart"),
    delphi: () => import("highlight.js/lib/languages/delphi"),
    diff: () => import("highlight.js/lib/languages/diff"),
    django: () => import("highlight.js/lib/languages/django"),
    dns: () => import("highlight.js/lib/languages/dns"),
    dockerfile: () => import("highlight.js/lib/languages/dockerfile"),
    dos: () => import("highlight.js/lib/languages/dos"),
    dsconfig: () => import("highlight.js/lib/languages/dsconfig"),
    dts: () => import("highlight.js/lib/languages/dts"),
    dust: () => import("highlight.js/lib/languages/dust"),
    ebnf: () => import("highlight.js/lib/languages/ebnf"),
    elixir: () => import("highlight.js/lib/languages/elixir"),
    elm: () => import("highlight.js/lib/languages/elm"),
    erb: () => import("highlight.js/lib/languages/erb"),
    erlang: () => import("highlight.js/lib/languages/erlang"),
    "erlang-repl": () => import("highlight.js/lib/languages/erlang-repl"),
    excel: () => import("highlight.js/lib/languages/excel"),
    fix: () => import("highlight.js/lib/languages/fix"),
    flix: () => import("highlight.js/lib/languages/flix"),
    fortran: () => import("highlight.js/lib/languages/fortran"),
    fsharp: () => import("highlight.js/lib/languages/fsharp"),
    gams: () => import("highlight.js/lib/languages/gams"),
    gauss: () => import("highlight.js/lib/languages/gauss"),
    gcode: () => import("highlight.js/lib/languages/gcode"),
    gherkin: () => import("highlight.js/lib/languages/gherkin"),
    glsl: () => import("highlight.js/lib/languages/glsl"),
    gml: () => import("highlight.js/lib/languages/gml"),
    go: () => import("highlight.js/lib/languages/go"),
    golo: () => import("highlight.js/lib/languages/golo"),
    gradle: () => import("highlight.js/lib/languages/gradle"),
    graphql: () => import("highlight.js/lib/languages/graphql"),
    groovy: () => import("highlight.js/lib/languages/groovy"),
    haml: () => import("highlight.js/lib/languages/haml"),
    handlebars: () => import("highlight.js/lib/languages/handlebars"),
    haskell: () => import("highlight.js/lib/languages/haskell"),
    haxe: () => import("highlight.js/lib/languages/haxe"),
    hsp: () => import("highlight.js/lib/languages/hsp"),
    http: () => import("highlight.js/lib/languages/http"),
    hy: () => import("highlight.js/lib/languages/hy"),
    inform7: () => import("highlight.js/lib/languages/inform7"),
    ini: () => import("highlight.js/lib/languages/ini"),
    irpf90: () => import("highlight.js/lib/languages/irpf90"),
    isbl: () => import("highlight.js/lib/languages/isbl"),
    java: () => import("highlight.js/lib/languages/java"),
    javascript: () => import("highlight.js/lib/languages/javascript"),
    "jboss-cli": () => import("highlight.js/lib/languages/jboss-cli"),
    json: () => import("highlight.js/lib/languages/json"),
    julia: () => import("highlight.js/lib/languages/julia"),
    "julia-repl": () => import("highlight.js/lib/languages/julia-repl"),
    kotlin: () => import("highlight.js/lib/languages/kotlin"),
    lasso: () => import("highlight.js/lib/languages/lasso"),
    latex: () => import("highlight.js/lib/languages/latex"),
    ldif: () => import("highlight.js/lib/languages/ldif"),
    leaf: () => import("highlight.js/lib/languages/leaf"),
    less: () => import("highlight.js/lib/languages/less"),
    lisp: () => import("highlight.js/lib/languages/lisp"),
    livecodeserver: () => import("highlight.js/lib/languages/livecodeserver"),
    livescript: () => import("highlight.js/lib/languages/livescript"),
    llvm: () => import("highlight.js/lib/languages/llvm"),
    lsl: () => import("highlight.js/lib/languages/lsl"),
    lua: () => import("highlight.js/lib/languages/lua"),
    makefile: () => import("highlight.js/lib/languages/makefile"),
    markdown: () => import("highlight.js/lib/languages/markdown"),
    mathematica: () => import("highlight.js/lib/languages/mathematica"),
    matlab: () => import("highlight.js/lib/languages/matlab"),
    maxima: () => import("highlight.js/lib/languages/maxima"),
    mel: () => import("highlight.js/lib/languages/mel"),
    mercury: () => import("highlight.js/lib/languages/mercury"),
    mipsasm: () => import("highlight.js/lib/languages/mipsasm"),
    mizar: () => import("highlight.js/lib/languages/mizar"),
    mojolicious: () => import("highlight.js/lib/languages/mojolicious"),
    monkey: () => import("highlight.js/lib/languages/monkey"),
    moonscript: () => import("highlight.js/lib/languages/moonscript"),
    n1ql: () => import("highlight.js/lib/languages/n1ql"),
    nestedtext: () => import("highlight.js/lib/languages/nestedtext"),
    nginx: () => import("highlight.js/lib/languages/nginx"),
    nim: () => import("highlight.js/lib/languages/nim"),
    nix: () => import("highlight.js/lib/languages/nix"),
    "node-repl": () => import("highlight.js/lib/languages/node-repl"),
    nsis: () => import("highlight.js/lib/languages/nsis"),
    objectivec: () => import("highlight.js/lib/languages/objectivec"),
    ocaml: () => import("highlight.js/lib/languages/ocaml"),
    openscad: () => import("highlight.js/lib/languages/openscad"),
    oxygene: () => import("highlight.js/lib/languages/oxygene"),
    parser3: () => import("highlight.js/lib/languages/parser3"),
    perl: () => import("highlight.js/lib/languages/perl"),
    pf: () => import("highlight.js/lib/languages/pf"),
    php: () => import("highlight.js/lib/languages/php"),
    "php-template": () => import("highlight.js/lib/languages/php-template"),
    plaintext: () => import("highlight.js/lib/languages/plaintext"),
    pony: () => import("highlight.js/lib/languages/pony"),
    powershell: () => import("highlight.js/lib/languages/powershell"),
    processing: () => import("highlight.js/lib/languages/processing"),
    profile: () => import("highlight.js/lib/languages/profile"),
    prolog: () => import("highlight.js/lib/languages/prolog"),
    properties: () => import("highlight.js/lib/languages/properties"),
    protobuf: () => import("highlight.js/lib/languages/protobuf"),
    puppet: () => import("highlight.js/lib/languages/puppet"),
    purebasic: () => import("highlight.js/lib/languages/purebasic"),
    python: () => import("highlight.js/lib/languages/python"),
    "python-repl": () => import("highlight.js/lib/languages/python-repl"),
    q: () => import("highlight.js/lib/languages/q"),
    qml: () => import("highlight.js/lib/languages/qml"),
    r: () => import("highlight.js/lib/languages/r"),
    reasonml: () => import("highlight.js/lib/languages/reasonml"),
    rib: () => import("highlight.js/lib/languages/rib"),
    roboconf: () => import("highlight.js/lib/languages/roboconf"),
    routeros: () => import("highlight.js/lib/languages/routeros"),
    rsl: () => import("highlight.js/lib/languages/rsl"),
    ruby: () => import("highlight.js/lib/languages/ruby"),
    ruleslanguage: () => import("highlight.js/lib/languages/ruleslanguage"),
    rust: () => import("highlight.js/lib/languages/rust"),
    sas: () => import("highlight.js/lib/languages/sas"),
    scala: () => import("highlight.js/lib/languages/scala"),
    scheme: () => import("highlight.js/lib/languages/scheme"),
    scilab: () => import("highlight.js/lib/languages/scilab"),
    scss: () => import("highlight.js/lib/languages/scss"),
    shell: () => import("highlight.js/lib/languages/shell"),
    smali: () => import("highlight.js/lib/languages/smali"),
    smalltalk: () => import("highlight.js/lib/languages/smalltalk"),
    sml: () => import("highlight.js/lib/languages/sml"),
    sqf: () => import("highlight.js/lib/languages/sqf"),
    sql: () => import("highlight.js/lib/languages/sql"),
    stan: () => import("highlight.js/lib/languages/stan"),
    stata: () => import("highlight.js/lib/languages/stata"),
    step21: () => import("highlight.js/lib/languages/step21"),
    stylus: () => import("highlight.js/lib/languages/stylus"),
    subunit: () => import("highlight.js/lib/languages/subunit"),
    swift: () => import("highlight.js/lib/languages/swift"),
    taggerscript: () => import("highlight.js/lib/languages/taggerscript"),
    tap: () => import("highlight.js/lib/languages/tap"),
    tcl: () => import("highlight.js/lib/languages/tcl"),
    thrift: () => import("highlight.js/lib/languages/thrift"),
    tp: () => import("highlight.js/lib/languages/tp"),
    twig: () => import("highlight.js/lib/languages/twig"),
    typescript: {
        importFunc: () => import("highlight.js/lib/languages/typescript"),
        dependencies: ["javascript"],
    },
    vala: () => import("highlight.js/lib/languages/vala"),
    vbnet: () => import("highlight.js/lib/languages/vbnet"),
    vbscript: () => import("highlight.js/lib/languages/vbscript"),
    "vbscript-html": () => import("highlight.js/lib/languages/vbscript-html"),
    verilog: () => import("highlight.js/lib/languages/verilog"),
    vhdl: () => import("highlight.js/lib/languages/vhdl"),
    vim: () => import("highlight.js/lib/languages/vim"),
    wasm: () => import("highlight.js/lib/languages/wasm"),
    wren: () => import("highlight.js/lib/languages/wren"),
    x86asm: () => import("highlight.js/lib/languages/x86asm"),
    xl: () => import("highlight.js/lib/languages/xl"),
    xml: () => import("highlight.js/lib/languages/xml"),
    xquery: () => import("highlight.js/lib/languages/xquery"),
    yaml: () => import("highlight.js/lib/languages/yaml"),
    zephir: () => import("highlight.js/lib/languages/zephir"),
} as const;

/** Pretty code block with copy button */
function CodeBlock({ children, className }) {
    const textRef = useRef<HTMLElement | null>(null);
    const [isHighlighted, setIsHighlighted] = useState(false);

    // Extract language from className
    const language = className ? className.replace("language-", "") : "";

    // Ensure children is a string
    const codeString = String(children).trim();

    useEffect(() => {
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
            let lang = language.replace("language-", "").replace("lang-", "");
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
    }, [codeString, language]);

    function copyCode() {
        if (textRef && textRef.current) {
            // Copy the text content of the code block
            navigator.clipboard.writeText(textRef.current.textContent ?? "");
            PubSub.get().publish("snack", { messageKey: "CopiedToClipboard", severity: "Success" });
        }
    }

    return (
        <CodeBlockOuter>
            <CopyButton
                onClick={copyCode}
            >
                <CopyIcon fill="white" />
            </CopyButton>
            <pre>
                <code
                    ref={textRef}
                    className={className}
                    style={{ paddingRight: "40px", opacity: isHighlighted ? 1 : 0 }}
                >
                    {codeString}
                </code>
            </pre>
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

// Vrooli pages that show up as special links
const specialRoutes = [
    "Api",
    "Chat",
    "Code",
    "Comment",
    "Note",
    "Project",
    "Question",
    "Quiz",
    "Report",
    "Routine",
    "Standard",
    "Tag",
    "Team",
    "User",
].map(key => LINKS[key]);

// Maps URL slugs to endpoints
const routeToEndpoint = {
    [LINKS.Api]: endpointGetApi,
    [LINKS.Chat]: endpointGetChat,
    [LINKS.DataConverter]: endpointGetCode,
    [LINKS.DataStructure]: endpointGetStandard,
    [LINKS.Comment]: endpointGetComment,
    [LINKS.Note]: endpointGetNote,
    [LINKS.Project]: endpointGetProject,
    [LINKS.Prompt]: endpointGetStandard,
    [LINKS.Question]: endpointGetQuestion,
    [LINKS.Quiz]: endpointGetQuiz,
    [LINKS.Report]: endpointGetReport,
    [LINKS.Routine]: endpointGetRoutine,
    [LINKS.SmartContract]: endpointGetCode,
    [LINKS.Tag]: endpointGetTag,
    [LINKS.Team]: endpointGetTeam,
    [LINKS.User]: endpointGetUser,
};

/** Creates custom links for Vrooli objects, and normal links otherwise */
function CustomLink({ children, href }) {
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
    const [getData, { data, loading: isLoading }] = useLazyFetch<any, any>(endpoint ?? endpointGetUser);

    // Get display data
    const { title, subtitle } = getDisplay(data, ["en"]);

    // Popover to display more info
    const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(null);
    const open = useCallback(({ target }: UsePressEvent) => {
        setAnchorEl(target as HTMLElement);
        const urlParams = parseSingleItemUrl({ href });
        if (exists(urlParams.handle)) getData({ handle: urlParams.handle });
        else if (exists(urlParams.handleRoot)) getData({ handleRoot: urlParams.handleRoot });
        else if (exists(urlParams.id)) getData({ id: urlParams.id });
        else if (exists(urlParams.idRoot)) getData({ idRoot: urlParams.idRoot });
        else PubSub.get().publish("snack", { message: "Invalid URL", severity: "Error" });
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
}

/** HOC for rendering links */
function withCustomLinkProps(additionalProps) {
    return function withCustomLinkPropsFunc({ href, children }) {
        return <CustomLink href={href} {...additionalProps}>{children}</CustomLink>;
    };
}

/** Custom checkbox component editable checkboxes */
function CustomCheckbox({ checked, onChange, ...otherProps }) {
    const id = useMemo(() => uuid(), []);
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
    variant, //TODO
}: {
    content: string | undefined;
    isEditable?: boolean;
    onChange?: (content: string) => unknown;
    sx?: SxType;
    variant?: TypographyProps["variant"];
}) {
    const { palette, typography } = useTheme();
    const id = useMemo(() => uuid(), []);

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
                h1: { component: Heading, props: { level: 1 } },
                h2: { component: Heading, props: { level: 2 } },
                h3: { component: Heading, props: { level: 3 } },
                h4: { component: Heading, props: { level: 4 } },
                h5: { component: Heading, props: { level: 5 } },
                h6: { component: Heading, props: { level: 6 } },
            },
        };
    }, [content, id, onChange]);

    // Preprocess the Markdown content
    const processedContent = processMarkdown(content ?? "");

    const markdownStyle = useMemo(function markdownStyleMemo() {
        return {
            fontFamily: typography.fontFamily,
            fontSize: typography.fontSize + 2,
            lineHeight: `${Math.round(typography.fontSize * 1.5)}px`,
            color: palette.background.textPrimary,
            display: "block",
            ...sx,
        } as const;
    }, [palette.background.textPrimary, sx, typography.fontSize, typography.fontFamily]);

    return (
        <Markdown id={id} options={options} style={markdownStyle}>
            {processedContent}
        </Markdown>
    );
}
