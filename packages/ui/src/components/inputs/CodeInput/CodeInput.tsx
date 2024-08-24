import { ChatInviteStatus, CodeLanguage, DUMMY_ID, LangsKey, Status, isEqual, uuid } from "@local/shared";
import { Box, Grid, IconButton, Stack, Tooltip, Typography, styled, useTheme } from "@mui/material";
import { HelpButton } from "components/buttons/HelpButton/HelpButton";
import { StatusButton } from "components/buttons/StatusButton/StatusButton";
import { SelectorBase } from "components/inputs/Selector/Selector";
import { useField } from "formik";
import { Suspense, useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
// import { isJson } from "@local/shared"; // Update this so that we can lint JSON standard input type (different from normal JSON)
import { ChatShape } from "@local/shared";
import { SessionContext } from "contexts/SessionContext";
import { useDebounce } from "hooks/useDebounce";
import { CopyIcon, MagicIcon, OpenThreadIcon, RedoIcon, RefreshIcon, UndoIcon } from "icons";
import React from "react";
import { SvgComponent } from "types";
import { getCurrentUser } from "utils/authentication/session";
import { getCookieMatchingChat } from "utils/cookies";
import { generateContext } from "utils/display/stringTools";
import { PubSub } from "utils/pubsub";
import { ChatCrud, VALYXA_INFO } from "views/objects/chat/ChatCrud/ChatCrud";
import { ChatCrudProps } from "views/objects/chat/types";
import { CodeInputBaseProps, CodeInputProps } from "../types";

// Stub types for code splitting
type Extension = {
    extension: Extension;
} | readonly Extension[];
type BlockInfo = {
    from: number;
    length: number;
    top: number;
    height: number;
    to: number;
}
type Range<T> = {
    from: number;
    to: number;
    value: T;
}
type Diagnostic = {
    from: number;
    to: number;
    severity: string;
    markClass?: string;
    source?: string;
    message: string;
    renderMessage?: () => Node;
    actions?: readonly object[];
}
type LanguageSupport = object;
type StreamLanguage = {
    define(spec: unknown): unknown;
}
type EditorSelection = {
    readonly ranges: readonly Range<number>[];
}
type EditorState = {
    readonly doc: {
        sliceString(from: number, to: number): string;
    };
    readonly selection: EditorSelection;
}
type EditorView = {
    state: EditorState;
}
type ReactCodeMirrorRef = {
    editor?: HTMLDivElement | null;
    state?: EditorState;
    view?: EditorView;
}

const LazyCodeMirror = React.lazy(() => import("@uiw/react-codemirror"));

async function loadDecorations() {
    const { Decoration, EditorView, showTooltip } = await import("@codemirror/view");
    const { StateField } = await import("@codemirror/state");
    const underlineMarkVariable = Decoration.mark({ class: "variable-decoration" });
    const underlineMarkOptional = Decoration.mark({ class: "optional-decoration" });
    const underlineMarkWildcard = Decoration.mark({ class: "wildcard-decoration" });
    const underlineMarkError = Decoration.mark({ class: "error-decoration" });

    function underlineVariables(doc) {
        const decorations: Range<typeof underlineMarkVariable>[] = [];
        // Regexes for each type of variable
        const variableRegex = /"<[a-zA-Z0-9_]+>"/g;
        const optionalRegex = /"\?[a-zA-Z0-9_]+":/g;
        const wildcardRegex = /"\[[a-zA-Z0-9_]+\]"/g;
        // Get the document text
        const docText = doc.sliceString(0, doc.length);
        // Match and decorate variables
        let match = variableRegex.exec(docText);
        while (match) {
            const start = match.index + 1;
            const end = start + match[0].length - 2;
            decorations.push(underlineMarkVariable.range(start, end));
            match = variableRegex.exec(docText);
        }
        // Match and decorate optionals
        match = optionalRegex.exec(docText);
        while (match) {
            const start = match.index + 1;
            const end = start + match[0].length - 3;
            decorations.push(underlineMarkOptional.range(start, end));
            match = optionalRegex.exec(docText);
        }
        // Match and decorate wildcards
        match = wildcardRegex.exec(docText);
        while (match) {
            const start = match.index + 1;
            const end = start + match[0].length - 2;
            decorations.push(underlineMarkWildcard.range(start, end));
            match = wildcardRegex.exec(docText);
        }
        // Sort the decorations by their start position. Decoration.set 
        // requires this for some reason.
        decorations.sort((a, b) => a.from - b.from);
        // Return the decorations as a set
        return Decoration.set(decorations);
    }

    const underlineDecorationField = StateField.define({
        create(state) {
            return underlineVariables(state.doc);
        },

        update(decorations, tr) {
            if (tr.docChanged) {
                return underlineVariables(tr.newDoc);
            }
            return decorations;
        },

        provide: f => EditorView.decorations.from(f),
    });

    function getCursorTooltips(state) {
        const tooltips: any[] = [];
        const docText = state.doc.sliceString(0, state.doc.length);
        const regex = /("<[a-zA-Z0-9_]+>")|("\?[a-zA-Z0-9_]+":)|("\[[a-zA-Z0-9_]+\]")/g;

        for (const r of state.selection.ranges) {
            if (!r.empty) continue;

            let match;
            while (match = regex.exec(docText)) {
                const variableText = match[0].slice(1, match[0].includes("?") ? -2 : -1); // Remove the quotes
                const start = match.index + 1; // Ignore the first quote
                const end = start + variableText.length; // We already ignored the quotes

                // If the cursor isn't within this match, skip it
                if (r.from < start || r.from > end) continue;

                let tooltipText = "";

                if (variableText.startsWith("<") && variableText.endsWith(">")) {
                    tooltipText = "Variable: This key represents a variable which will be compared against the value at this key in the data JSON object.";
                } else if (variableText.startsWith("?")) {
                    tooltipText = "Optional: This key is optional. If it is present in the data JSON object, its value will be compared against the value at this key in the data JSON object.";
                } else if (variableText.startsWith("[") && variableText.endsWith("]")) {
                    tooltipText = "Wildcard: This key allows additional keys to be added to the object, and its value type specifies the type of the added values.";
                }

                if (tooltipText) {
                    tooltips.push({
                        pos: start,
                        end,
                        above: true,
                        create() {
                            const dom = document.createElement("div");
                            dom.textContent = tooltipText;
                            return { dom };
                        },
                    });
                }
            }
        }

        return tooltips;
    }

    const cursorTooltipField = StateField.define({
        create: getCursorTooltips,

        update(value, tr) {
            if (!tr.docChanged && !tr.selection) return value;
            return getCursorTooltips(tr.state);
        },

        provide: f => showTooltip.computeN([f], state => state.field(f)),
    });

    return [cursorTooltipField, underlineDecorationField];
}

/**
 * Dynamically imports language packages.
 */
const languageMap: { [x in CodeLanguage]: (() => Promise<{
    main: LanguageSupport | StreamLanguage | Extension,
    linter?: ((view: EditorView) => Diagnostic[]) | Extension,
}>) } = {
    [CodeLanguage.Angular]: async () => {
        const { angular } = await import("@codemirror/lang-angular");
        return { main: angular() };
    },
    [CodeLanguage.Cpp]: async () => {
        const { cpp } = await import("@codemirror/lang-cpp");
        return { main: cpp() };
    },
    [CodeLanguage.Css]: async () => {
        const { css } = await import("@codemirror/lang-css");
        return { main: css() };
    },
    [CodeLanguage.Dockerfile]: async () => {
        const { dockerFile } = await import("@codemirror/legacy-modes/mode/dockerfile");
        const { StreamLanguage } = await import("@codemirror/language");
        return { main: StreamLanguage.define(dockerFile) };
    },
    [CodeLanguage.Go]: async () => {
        const { go } = await import("@codemirror/legacy-modes/mode/go");
        const { StreamLanguage } = await import("@codemirror/language");
        return { main: StreamLanguage.define(go) };
    },
    [CodeLanguage.Graphql]: async () => {
        const { graphql } = await import("cm6-graphql");
        return { main: graphql() };
    },
    [CodeLanguage.Groovy]: async () => {
        const { groovy } = await import("@codemirror/legacy-modes/mode/groovy");
        const { StreamLanguage } = await import("@codemirror/language");
        return { main: StreamLanguage.define(groovy) };
    },
    [CodeLanguage.Haskell]: async () => {
        const { haskell } = await import("@codemirror/legacy-modes/mode/haskell");
        const { StreamLanguage } = await import("@codemirror/language");
        return { main: StreamLanguage.define(haskell) };
    },
    [CodeLanguage.Html]: async () => {
        const { html } = await import("@codemirror/lang-html");
        return { main: html() };
    },
    [CodeLanguage.Java]: async () => {
        const { java } = await import("@codemirror/lang-java");
        return { main: java() };
    },
    [CodeLanguage.Javascript]: async () => {
        const { javascript } = await import("@codemirror/lang-javascript");
        return { main: javascript({ jsx: true }) };
    },
    [CodeLanguage.Json]: async () => {
        const { json, jsonParseLinter } = await import("@codemirror/lang-json");
        return { main: json(), linter: jsonParseLinter() as unknown as ((view: EditorView) => Diagnostic[]) };
    },
    [CodeLanguage.JsonStandard]: async () => {
        const { json, jsonParseLinter } = await import("@codemirror/lang-json");
        return { main: json(), linter: jsonParseLinter() as unknown as ((view: EditorView) => Diagnostic[]) };
    },
    [CodeLanguage.Nginx]: async () => {
        const { nginx } = await import("@codemirror/legacy-modes/mode/nginx");
        const { StreamLanguage } = await import("@codemirror/language");
        return { main: StreamLanguage.define(nginx) };
    },
    [CodeLanguage.Nix]: async () => {
        const { nix } = await import("@replit/codemirror-lang-nix");
        return { main: nix() };
    },
    [CodeLanguage.Php]: async () => {
        const { php } = await import("@codemirror/lang-php");
        return { main: php() };
    },
    [CodeLanguage.Powershell]: async () => {
        const { powerShell } = await import("@codemirror/legacy-modes/mode/powershell");
        const { StreamLanguage } = await import("@codemirror/language");
        return { main: StreamLanguage.define(powerShell) };
    },
    [CodeLanguage.Protobuf]: async () => {
        const { protobuf } = await import("@codemirror/legacy-modes/mode/protobuf");
        const { StreamLanguage } = await import("@codemirror/language");
        return { main: StreamLanguage.define(protobuf) };
    },
    [CodeLanguage.Puppet]: async () => {
        const { puppet } = await import("@codemirror/legacy-modes/mode/puppet");
        const { StreamLanguage } = await import("@codemirror/language");
        return { main: StreamLanguage.define(puppet) };
    },
    [CodeLanguage.Python]: async () => {
        const { python } = await import("@codemirror/lang-python");
        return { main: python() };
    },
    [CodeLanguage.R]: async () => {
        const { r } = await import("codemirror-lang-r");
        return { main: r() };
    },
    [CodeLanguage.Ruby]: async () => {
        const { ruby } = await import("@codemirror/legacy-modes/mode/ruby");
        const { StreamLanguage } = await import("@codemirror/language");
        return { main: StreamLanguage.define(ruby) };
    },
    [CodeLanguage.Rust]: async () => {
        const { rust } = await import("@codemirror/lang-rust");
        return { main: rust() };
    },
    [CodeLanguage.Sass]: async () => {
        const { sass } = await import("@codemirror/lang-sass");
        return { main: sass() };
    },
    [CodeLanguage.Shell]: async () => {
        const { shell } = await import("@codemirror/legacy-modes/mode/shell");
        const { StreamLanguage } = await import("@codemirror/language");
        return { main: StreamLanguage.define(shell) };
    },
    [CodeLanguage.Solidity]: async () => {
        const { solidity } = await import("@replit/codemirror-lang-solidity");
        return { main: solidity };
    },
    [CodeLanguage.Spreadsheet]: async () => {
        const { spreadsheet } = await import("@codemirror/legacy-modes/mode/spreadsheet");
        const { StreamLanguage } = await import("@codemirror/language");
        return { main: StreamLanguage.define(spreadsheet) };
    },
    [CodeLanguage.Sql]: async () => {
        const { standardSQL } = await import("@codemirror/legacy-modes/mode/sql");
        const { StreamLanguage } = await import("@codemirror/language");
        return { main: StreamLanguage.define(standardSQL) };
    },
    [CodeLanguage.Svelte]: async () => {
        const { svelte } = await import("@replit/codemirror-lang-svelte");
        return { main: svelte() };
    },
    [CodeLanguage.Swift]: async () => {
        const { swift } = await import("@codemirror/legacy-modes/mode/swift");
        const { StreamLanguage } = await import("@codemirror/language");
        return { main: StreamLanguage.define(swift) };
    },
    [CodeLanguage.Typescript]: async () => {
        const { javascript } = await import("@codemirror/lang-javascript");
        return { main: javascript({ jsx: true, typescript: true }) };
    },
    [CodeLanguage.Vb]: async () => {
        const { vb } = await import("@codemirror/legacy-modes/mode/vb");
        const { StreamLanguage } = await import("@codemirror/language");
        return { main: StreamLanguage.define(vb) };
    },
    [CodeLanguage.Vbscript]: async () => {
        const { vbScript } = await import("@codemirror/legacy-modes/mode/vbscript");
        const { StreamLanguage } = await import("@codemirror/language");
        return { main: StreamLanguage.define(vbScript) };
    },
    [CodeLanguage.Verilog]: async () => {
        const { verilog } = await import("@codemirror/legacy-modes/mode/verilog");
        const { StreamLanguage } = await import("@codemirror/language");
        return { main: StreamLanguage.define(verilog) };
    },
    [CodeLanguage.Vhdl]: async () => {
        const { vhdl } = await import("@codemirror/legacy-modes/mode/vhdl");
        const { StreamLanguage } = await import("@codemirror/language");
        return { main: StreamLanguage.define(vhdl) };
    },
    [CodeLanguage.Vue]: async () => {
        const { vue } = await import("@codemirror/lang-vue");
        return { main: vue() };
    },
    [CodeLanguage.Xml]: async () => {
        const { xml } = await import("@codemirror/lang-xml");
        return { main: xml() };
    },
    [CodeLanguage.Yacas]: async () => {
        const { yacas } = await import("@codemirror/legacy-modes/mode/yacas");
        const { StreamLanguage } = await import("@codemirror/language");
        return { main: StreamLanguage.define(yacas) };
    },
    [CodeLanguage.Yaml]: async () => {
        const { yaml } = await import("@codemirror/legacy-modes/mode/yaml");
        const { StreamLanguage } = await import("@codemirror/language");
        return { main: StreamLanguage.define(yaml) };
    },
};

function getSeverityForLine(line: BlockInfo, errors: Diagnostic[], view: EditorView) {
    const linePosStart = line.from;
    const linePosEnd = line.to;
    for (const error of errors) {
        if (error.from >= linePosStart && error.to <= linePosEnd) {
            return error.severity;
        }
    }
    return null;
}

/**
 * Maps languages to their labels and help texts.
 */
const languageDisplayMap: { [x in CodeLanguage]: [LangsKey, LangsKey] } = {
    [CodeLanguage.Angular]: ["Angular", "AngularHelp"],
    [CodeLanguage.Cpp]: ["Cpp", "CppHelp"],
    [CodeLanguage.Css]: ["Css", "CssHelp"],
    [CodeLanguage.Dockerfile]: ["Dockerfile", "DockerfileHelp"],
    [CodeLanguage.Go]: ["Go", "GoHelp"],
    [CodeLanguage.Graphql]: ["Graphql", "GraphqlHelp"],
    [CodeLanguage.Groovy]: ["Groovy", "GroovyHelp"],
    [CodeLanguage.Haskell]: ["Haskell", "HaskellHelp"],
    [CodeLanguage.Html]: ["Html", "HtmlHelp"],
    [CodeLanguage.Java]: ["Java", "JavaHelp"],
    [CodeLanguage.Javascript]: ["Javascript", "JavascriptHelp"],
    [CodeLanguage.Json]: ["Json", "JsonHelp"],
    [CodeLanguage.JsonStandard]: ["JsonStandard", "JsonStandardHelp"],
    [CodeLanguage.Nginx]: ["Nginx", "NginxHelp"],
    [CodeLanguage.Nix]: ["Nix", "NixHelp"],
    [CodeLanguage.Php]: ["Php", "PhpHelp"],
    [CodeLanguage.Powershell]: ["Powershell", "PowershellHelp"],
    [CodeLanguage.Protobuf]: ["Protobuf", "ProtobufHelp"],
    [CodeLanguage.Puppet]: ["Puppet", "PuppetHelp"],
    [CodeLanguage.Python]: ["Python", "PythonHelp"],
    [CodeLanguage.R]: ["R", "RHelp"],
    [CodeLanguage.Ruby]: ["Ruby", "RubyHelp"],
    [CodeLanguage.Rust]: ["Rust", "RustHelp"],
    [CodeLanguage.Sass]: ["Sass", "SassHelp"],
    [CodeLanguage.Shell]: ["Shell", "ShellHelp"],
    [CodeLanguage.Solidity]: ["Solidity", "SolidityHelp"],
    [CodeLanguage.Spreadsheet]: ["Spreadsheet", "SpreadsheetHelp"],
    [CodeLanguage.Sql]: ["Sql", "SqlHelp"],
    [CodeLanguage.Svelte]: ["Svelte", "SvelteHelp"],
    [CodeLanguage.Swift]: ["Swift", "SwiftHelp"],
    [CodeLanguage.Typescript]: ["Typescript", "TypescriptHelp"],
    [CodeLanguage.Vb]: ["Vb", "VbHelp"],
    [CodeLanguage.Vbscript]: ["Vbscript", "VbscriptHelp"],
    [CodeLanguage.Verilog]: ["Verilog", "VerilogHelp"],
    [CodeLanguage.Vhdl]: ["Vhdl", "VhdlHelp"],
    [CodeLanguage.Vue]: ["Vue", "VueHelp"],
    [CodeLanguage.Xml]: ["Xml", "XmlHelp"],
    [CodeLanguage.Yacas]: ["Yacas", "YacasHelp"],
    [CodeLanguage.Yaml]: ["Yaml", "YamlHelp"],
};

const InfoBar = styled(Box)(({ theme }) => ({
    display: "flex",
    width: "100%",
    padding: "0.5rem",
    borderBottom: "1px solid #e0e0e0",
    background: theme.palette.primary.light,
    color: theme.palette.primary.contrastText,
    alignItems: "center",
    flexDirection: "row",
    [theme.breakpoints.down("sm")]: {
        flexDirection: "column",
    },
}));

const outerStackStyle = {
    borderRadius: 1.5,
    overflow: "hidden",
} as const;
const lazyCodeMirrorStyle = {
    position: "relative",
    zIndex: 2,
} as const;
const refreshIconStyle = {
    position: "absolute",
    top: "50%",
    left: "50%",
    transform: "translate(-50%, -50%)",
    zIndex: 1,
} as const;
const statusButtonStyle = {
    marginRight: "auto",
    height: "fit-content",
} as const;
const languageSelectorStyle = {
    root: {
        width: "fit-content",
        minWidth: "200px",
    },
} as const;
const copyButtonStyle = { marginLeft: "auto" } as const;

const CHANGE_DEBOUNCE_MS = 250;
const REFRESH_ICON_SHOW_AFTER_MS = 3000;

// TODO May be able to combine CodeInputBase and JsonInput into one component. To do this, make "JSON Standard" a 
// new language option. Also need to add support for format (which is JSON Standard) which, if provided, limits the language to JSON 
// and only makes input valid if it matches the format. Doing this will make this component stand out from the other 
// "standard input" components, but the duplicate code prevention may be worth it.
export function CodeInputBase({
    codeLanguage,
    content,
    defaultValue,
    disabled,
    format,
    handleCodeLanguageChange,
    handleContentChange,
    limitTo,
    name,
    variables,
}: CodeInputBaseProps) {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const session = useContext(SessionContext);
    const { credits } = useMemo(() => getCurrentUser(session), [session]);

    const availableLanguages = Array.isArray(limitTo) && limitTo.length > 0 ? limitTo : Object.values(CodeLanguage);

    const codeMirrorRef = useRef<ReactCodeMirrorRef | null>(null);
    const commandFunctionsRef = useRef<{ undo: ((view: EditorView) => unknown) | null, redo: ((view: EditorView) => unknown) | null }>({ undo: null, redo: null });

    async function loadCommandFunctions() {
        if (!commandFunctionsRef.current.undo || !commandFunctionsRef.current.redo) {
            const commands = await import("@codemirror/commands");
            commandFunctionsRef.current.undo = commands.undo as unknown as ((view: EditorView) => unknown);
            commandFunctionsRef.current.redo = commands.redo as unknown as ((view: EditorView) => unknown);
        }
    }

    // // Last valid schema format
    // const [internalValue, setInternalValue] = useState<string>(jsonToString(format) ?? "");
    const debounceContentHandler = useCallback((content: string) => { handleContentChange(content); }, [handleContentChange]);
    const [debounceContent] = useDebounce(debounceContentHandler, CHANGE_DEBOUNCE_MS);
    const updateContent = useCallback((newContent: string) => {
        if (disabled) return;
        // setInternalValue(newContent);
        debounceContent(newContent);
    }, [disabled, debounceContent]);

    // Track errors
    const [errors, setErrors] = useState<Diagnostic[]>([]);
    const wrappedLinter = useCallback(async (nextLinter) => {
        const { linter } = await import("@codemirror/lint");
        return linter(view => {
            // Run the existing linter and get its diagnostics.
            const diagnostics = nextLinter(view);
            // Count the number of errors.
            const updatedErrors = diagnostics.filter(d => d.severity === "error");
            // Update state
            if (!isEqual(updatedErrors, errors)) {
                setErrors(updatedErrors);
            }
            // Return the diagnostics so they still get displayed.
            return diagnostics;
        });
    }, [errors]);

    // Handle language selection
    const changeCodeLanguage = useCallback((newCodeLanguage: CodeLanguage) => {
        setErrors([]);
        handleCodeLanguageChange(newCodeLanguage);
    }, [handleCodeLanguageChange]);

    const [extensions, setExtensions] = useState<Extension[]>([]);
    const [supportsValidation, setSupportsValidation] = useState<boolean>(false);
    useEffect(function codeExtensionsEffect() {
        let isMounted = true;

        async function loadGutter() {
            const { gutter } = await import("@codemirror/view");
            const { ErrorMarker, WarnMarker } = await import("./codeMirrorMarkers");
            return gutter({
                lineMarker: (view, line) => {
                    const severity = getSeverityForLine(line, errors, view as unknown as EditorView);
                    if (severity === "error") {
                        return new ErrorMarker();
                    } else if (severity === "warning") {
                        return new WarnMarker();
                    }
                    return null;
                },
                class: "cm-errorGutter",
            });
        }

        async function updateExtensions() {
            try {
                const updatedExtensions: Extension[] = [];

                // Load and add base theme
                const { EditorView } = await import("@codemirror/view");
                const base = EditorView.baseTheme({
                    ".error-decoration": {
                        textDecoration: "underline",
                        textDecorationStyle: "wavy",
                        textDecorationColor: palette.error.main,
                    },
                    ".variable-decoration": {
                        textDecoration: "underline",
                        textDecorationStyle: "double",
                        textDecorationColor: "blue",
                    },
                    ".optional-decoration": {
                        textDecoration: "underline",
                        textDecorationStyle: "double",
                        textDecorationColor: "magenta",
                    },
                    ".wildcard-decoration": {
                        textDecoration: "underline",
                        textDecorationStyle: "double",
                        textDecorationColor: "lightseagreen",
                    },
                });
                updatedExtensions.push(base);

                // Load and add gutter
                const errorGutterExtension = await loadGutter();
                updatedExtensions.push(errorGutterExtension);

                // Load language extensions
                if (codeLanguage in languageMap) {
                    const { main, linter } = await languageMap[codeLanguage]();
                    updatedExtensions.push(main as Extension);
                    if (linter) {
                        const linterExtension = await wrappedLinter(linter);
                        updatedExtensions.push(linterExtension as Extension);
                        if (isMounted) setSupportsValidation(true);
                    } else if (isMounted) {
                        setSupportsValidation(false);
                    }
                    // If language is JSON standard, add additional extensions for variables
                    if (codeLanguage === CodeLanguage.JsonStandard) {
                        const [cursorTooltipField, underlineDecorationField] = await loadDecorations();
                        updatedExtensions.push(
                            cursorTooltipField, // Handle tooltips for JSON variables
                            underlineDecorationField, // Underline JSON variables
                        );
                    }
                }

                if (isMounted) setExtensions(updatedExtensions);
            } catch (error) {
                console.error("Error updating extensions:", error);
            }
        }

        updateExtensions();

        return () => {
            isMounted = false;
        };
    }, [errors, codeLanguage, palette.error.main, wrappedLinter]);

    const id = useMemo(() => `code-container-${name}`, [name]);

    // Handle assistant dialog
    const closeAssistantDialog = useCallback(() => {
        setAssistantDialogProps(props => ({ ...props, context: undefined, isOpen: false, overrideObject: undefined } as ChatCrudProps));
        PubSub.get().publish("sideMenu", { id: "chat-side-menu", idPrefix: "standard", isOpen: false });
    }, []);
    const [assistantDialogProps, setAssistantDialogProps] = useState<ChatCrudProps>({
        context: undefined,
        display: "dialog",
        isCreate: true,
        isOpen: false,
        task: "standard",
        onCancel: closeAssistantDialog,
        onCompleted: closeAssistantDialog,
        onClose: closeAssistantDialog,
        onDeleted: closeAssistantDialog,
    });
    const openAssistantDialog = useCallback(() => {
        if (disabled) return;
        const userId = getCurrentUser(session)?.id;
        if (!userId) return;

        if (!codeMirrorRef.current || !codeMirrorRef.current.view) {
            console.error("CodeMirror not found");
            return;
        }
        const codeDoc = codeMirrorRef.current.view.state.doc;
        const selectionRanges = codeMirrorRef.current.view.state.selection.ranges;
        // Only use the first selection range, if it exists
        const selection = selectionRanges.length > 0 ? codeDoc.sliceString(selectionRanges[0].from, selectionRanges[0].to) : "";
        const fullText = codeDoc.sliceString(0, Number.MAX_SAFE_INTEGER);
        const context = generateContext(selection, fullText);

        // Now we'll try to find an existing chat with Valyxa for this task
        const existingChatId = getCookieMatchingChat([userId, VALYXA_INFO.id], "standard");
        const overrideObject = {
            __typename: "Chat" as const,
            id: existingChatId ?? DUMMY_ID,
            openToAnyoneWithInvite: false,
            invites: [{
                __typename: "ChatInvite" as const,
                id: uuid(),
                status: ChatInviteStatus.Pending,
                user: VALYXA_INFO,
            }],
        } as unknown as ChatShape;

        // Open the assistant dialog
        setAssistantDialogProps(props => ({ ...props, isCreate: !existingChatId, isOpen: true, context, overrideObject } as ChatCrudProps));
    }, [disabled, session]);

    // Handle action buttons
    type Action = {
        Icon: SvgComponent,
        key: string,
        label: string,
        onClick: () => unknown,
    }
    const actions = useMemo(() => {
        const actionsList: Action[] = [];
        // If user has premium, add button for AI assistant
        if (credits && BigInt(credits) > 0) {
            actionsList.push({
                Icon: MagicIcon,
                key: "assistant",
                label: "AI assistant",
                onClick: () => { openAssistantDialog(); },
            });
        }
        // Always add undo and redo buttons
        actionsList.push({
            Icon: UndoIcon,
            key: "undo",
            label: t("Undo"),
            onClick: async () => {
                await loadCommandFunctions();
                if (codeMirrorRef.current?.view) {
                    commandFunctionsRef.current.undo?.(codeMirrorRef.current.view);
                }
            },
        });
        actionsList.push({
            Icon: RedoIcon,
            key: "redo",
            label: t("Redo"),
            onClick: async () => {
                await loadCommandFunctions();
                if (codeMirrorRef.current?.view) {
                    commandFunctionsRef.current.redo?.(codeMirrorRef.current.view);
                }
            },
        });
        // For json and jsonStandard, add "pretty print" button to format JSON
        if (codeLanguage === CodeLanguage.Json || codeLanguage === CodeLanguage.JsonStandard) {
            actionsList.push({
                Icon: OpenThreadIcon,
                key: "format",
                label: t("Format"),
                onClick: () => {
                    try {
                        const parsed = JSON.parse(content);
                        updateContent(JSON.stringify(parsed, null, 4));
                    } catch (error) {
                        PubSub.get().publish("snack", { message: "Invalid JSON", severity: "Error", data: { error } });
                    }
                },
            });
        }
        return actionsList;
    }, [credits, codeLanguage, content, openAssistantDialog, t, updateContent]);

    const [, helpKey] = useMemo<[LangsKey, LangsKey]>(() => languageDisplayMap[codeLanguage] ?? ["Json", "JsonHelp"], [codeLanguage]);

    // Handle refreshing the editor (in case is fails to appear, which happens occasionally)
    const [editorKey, setEditorKey] = useState(0);
    const [showRefresh, setShowRefresh] = useState(false);
    function refreshEditor() {
        setEditorKey(prevKey => prevKey + 1); // Increment the key to force re-render
        setShowRefresh(false); // Hide the refresh icon so it can appear again after a delay
    }
    // Show the refresh icon after a short delay
    useEffect(function clearCodeTimeoutEffect() {
        const timer = setTimeout(() => { setShowRefresh(true); }, REFRESH_ICON_SHOW_AFTER_MS);
        return () => clearTimeout(timer);
    }, [editorKey]);

    const handleCopy = useCallback(function handleCopyCallback() {
        if (!content) return;
        navigator.clipboard.writeText(content);
        PubSub.get().publish("snack", { message: "Copied to clipboard", severity: "Success" });
    }, [content]);

    return (
        <>
            {/* Assistant dialog for generating text */}
            <ChatCrud {...assistantDialogProps} />
            <Stack direction="column" spacing={0} sx={outerStackStyle}>
                <InfoBar>
                    {/* Select or display language */}
                    {availableLanguages.length > 1 ?
                        <Grid item xs={12} sm={6}>
                            <SelectorBase
                                name="codeLanguage"
                                value={codeLanguage}
                                onChange={changeCodeLanguage}
                                disabled={disabled}
                                options={availableLanguages}
                                getOptionLabel={(r) => {
                                    const [labelKey] = languageDisplayMap[r as CodeLanguage] ?? ["Unknown"];
                                    return t(labelKey, { ns: "langs" });
                                }}
                                fullWidth
                                inputAriaLabel="select language"
                                label={"Language"}
                                sxs={languageSelectorStyle}
                            />
                        </Grid> :
                        disabled ?
                            <Tooltip title={t(helpKey, { ns: "langs" })}>
                                <Typography variant="body1" sx={{ marginLeft: 1, marginRight: "auto" }}>
                                    {t(languageDisplayMap[codeLanguage][0], { ns: "langs" })}
                                </Typography>
                            </Tooltip> :
                            null
                    }
                    <Grid item xs={12} sm={availableLanguages.length > 1 ? 6 : 12} sx={{
                        marginLeft: { xs: 0, sm: "auto" },
                        ...(availableLanguages.length <= 1 && {
                            display: "flex",
                            justifyContent: "flex-end",
                        }),
                    }}>
                        {/* Actions, Help button, Status */}
                        {!disabled && <Box sx={{
                            display: "flex",
                            flexDirection: "row",
                            justifyContent: "flex-start",
                            flexWrap: "wrap",
                            alignItems: "center",
                        }}>
                            {actions.map(({ Icon, key, label, onClick }, i) => <Tooltip key={key} title={label}>
                                <IconButton
                                    onClick={onClick}
                                    disabled={disabled}
                                >
                                    <Icon fill={palette.primary.contrastText} />
                                </IconButton>
                            </Tooltip>)}
                            <HelpButton
                                markdown={t(helpKey, { ns: "langs" })}
                                sx={{ fill: palette.secondary.contrastText }}
                            />
                            {supportsValidation && <StatusButton
                                status={errors.length === 0 ? Status.Valid : Status.Invalid}
                                messages={errors.map(e => e.message)}
                                sx={statusButtonStyle}
                            />}
                        </Box>}
                        {/* Copy button */}
                        {disabled && Boolean(content) && <IconButton
                            onClick={handleCopy}
                            sx={copyButtonStyle}
                        >
                            <CopyIcon fill={palette.primary.contrastText} />
                        </IconButton>}
                    </Grid>
                </InfoBar>
                <Box sx={{
                    position: "relative",
                    height: "400px",
                    background: palette.mode === "dark" ? "#282c34" : "#ffffff",
                }}>
                    <Suspense fallback={
                        <Typography variant="body1" sx={{
                            color: palette.background.textSecondary,
                            padding: 1,
                        }}>
                            Loading editor...
                        </Typography>
                    }>
                        <LazyCodeMirror
                            key={`code-editor-${editorKey}`}
                            id={id}
                            readOnly={disabled}
                            ref={codeMirrorRef as any}
                            value={content}
                            theme={palette.mode === "dark" ? "dark" : "light"}
                            extensions={extensions}
                            onChange={updateContent}
                            height={"400px"}
                            style={lazyCodeMirrorStyle}
                        />
                    </Suspense>
                    {showRefresh && <IconButton
                        onClick={refreshEditor}
                        sx={refreshIconStyle}
                    >
                        <RefreshIcon fill={palette.background.textPrimary} width={48} height={48} />
                    </IconButton>}
                </Box>
                {/* Bottom bar containing arrow buttons to switch to different incomplete/incorrect
             parts of the JSON, and an input for entering the currently-selected section of JSON */}
                {/* TODO */}
                {/* Helper text label TODO */}
                {/* {
                    formatMeta.error &&
                    <Typography variant="body1" sx={{ color: "red" }}>
                        {formatMeta.error}
                    </Typography>
                } */}
            </Stack>
        </>
    );
}

const DEFAULT_CODE_LANGUAGE = CodeLanguage.Javascript;
const DEFAULT_CODE_LANGUAGE_FIELD = "codeLanguage";
const DEFAULT_CONTENT = "";
const DEFAULT_NAME = "content";

export function CodeInput({
    codeLanguageField,
    name,
    ...props
}: CodeInputProps) {
    const [languageField, , codeLanguageHelpers] = useField<CodeInputBaseProps["codeLanguage"]>(codeLanguageField ?? DEFAULT_CODE_LANGUAGE_FIELD);
    const [contentField, , contentHelpers] = useField<CodeInputBaseProps["content"]>(name ?? DEFAULT_NAME);
    const [defaultValueField] = useField<CodeInputBaseProps["defaultValue"]>("defaultValue");
    const [formatField] = useField<CodeInputBaseProps["format"]>("format");
    const [variablesField] = useField<CodeInputBaseProps["variables"]>("variables");

    console.log("in CodeInput", languageField.value, contentField.value);

    return (
        <CodeInputBase
            {...props}
            codeLanguage={languageField.value ?? DEFAULT_CODE_LANGUAGE}
            content={contentField.value ?? DEFAULT_CONTENT}
            defaultValue={defaultValueField.value}
            format={formatField.value}
            handleCodeLanguageChange={codeLanguageHelpers.setValue}
            handleContentChange={contentHelpers.setValue}
            name={name ?? DEFAULT_NAME}
            variables={variablesField.value}
        />
    );
}
