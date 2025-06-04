/* eslint-disable import/extensions */
import { CodeLanguage, Status, type TranslationKeyLangs, isEqual } from "@local/shared";
import { Box, IconButton, Stack, Tooltip, Typography, styled, useTheme } from "@mui/material";
import { useField } from "formik";
import { Suspense, useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { HelpButton } from "../../buttons/HelpButton.js";
import { StatusButton } from "../../buttons/StatusButton.js";
import { SelectorBase } from "../../inputs/Selector/Selector.js";
// import { isJson } from "@local/shared"; // Update this so that we can lint JSON standard input type (different from normal JSON)
import React from "react";
import { SessionContext } from "../../../contexts/session.js";
import { useDebounce } from "../../../hooks/useDebounce.js";
import { Icon, IconCommon, type IconInfo, IconText } from "../../../icons/Icons.js";
import { getCurrentUser } from "../../../utils/authentication/session.js";
import { PubSub } from "../../../utils/pubsub.js";
import { type CodeInputBaseProps, type CodeInputProps } from "../types.js";

const ICON_SIZE_PX = 20;

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
const languageDisplayMap: { [x in CodeLanguage]: [TranslationKeyLangs, TranslationKeyLangs] } = {
    [CodeLanguage.Angular]: ["Angular_Label", "Angular_Help"],
    [CodeLanguage.Cpp]: ["Cpp_Label", "Cpp_Help"],
    [CodeLanguage.Css]: ["Css_Label", "Css_Help"],
    [CodeLanguage.Dockerfile]: ["Dockerfile_Label", "Dockerfile_Help"],
    [CodeLanguage.Go]: ["Go_Label", "Go_Help"],
    [CodeLanguage.Graphql]: ["Graphql_Label", "Graphql_Help"],
    [CodeLanguage.Groovy]: ["Groovy_Label", "Groovy_Help"],
    [CodeLanguage.Haskell]: ["Haskell_Label", "Haskell_Help"],
    [CodeLanguage.Html]: ["Html_Label", "Html_Help"],
    [CodeLanguage.Java]: ["Java_Label", "Java_Help"],
    [CodeLanguage.Javascript]: ["Javascript_Label", "Javascript_Help"],
    [CodeLanguage.Json]: ["Json_Label", "Json_Help"],
    [CodeLanguage.JsonStandard]: ["JsonStandard_Label", "JsonStandard_Help"],
    [CodeLanguage.Nginx]: ["Nginx_Label", "Nginx_Help"],
    [CodeLanguage.Nix]: ["Nix_Label", "Nix_Help"],
    [CodeLanguage.Php]: ["Php_Label", "Php_Help"],
    [CodeLanguage.Powershell]: ["Powershell_Label", "Powershell_Help"],
    [CodeLanguage.Protobuf]: ["Protobuf_Label", "Protobuf_Help"],
    [CodeLanguage.Puppet]: ["Puppet_Label", "Puppet_Help"],
    [CodeLanguage.Python]: ["Python_Label", "Python_Help"],
    [CodeLanguage.R]: ["R_Label", "R_Help"],
    [CodeLanguage.Ruby]: ["Ruby_Label", "Ruby_Help"],
    [CodeLanguage.Rust]: ["Rust_Label", "Rust_Help"],
    [CodeLanguage.Sass]: ["Sass_Label", "Sass_Help"],
    [CodeLanguage.Shell]: ["Shell_Label", "Shell_Help"],
    [CodeLanguage.Solidity]: ["Solidity_Label", "Solidity_Help"],
    [CodeLanguage.Spreadsheet]: ["Spreadsheet_Label", "Spreadsheet_Help"],
    [CodeLanguage.Sql]: ["Sql_Label", "Sql_Help"],
    [CodeLanguage.Svelte]: ["Svelte_Label", "Svelte_Help"],
    [CodeLanguage.Swift]: ["Swift_Label", "Swift_Help"],
    [CodeLanguage.Typescript]: ["Typescript_Label", "Typescript_Help"],
    [CodeLanguage.Vb]: ["Vb_Label", "Vb_Help"],
    [CodeLanguage.Vbscript]: ["Vbscript_Label", "Vbscript_Help"],
    [CodeLanguage.Verilog]: ["Verilog_Label", "Verilog_Help"],
    [CodeLanguage.Vhdl]: ["Vhdl_Label", "Vhdl_Help"],
    [CodeLanguage.Vue]: ["Vue_Label", "Vue_Help"],
    [CodeLanguage.Xml]: ["Xml_Label", "Xml_Help"],
    [CodeLanguage.Yacas]: ["Yacas_Label", "Yacas_Help"],
    [CodeLanguage.Yaml]: ["Yaml_Label", "Yaml_Help"],
};

const InfoBar = styled(Box)(({ theme }) => ({
    display: "flex",
    width: "100%",
    padding: theme.spacing(1),
    backgroundColor: theme.palette.primary.dark,
    color: theme.palette.primary.contrastText,
    alignItems: "center",
    flexDirection: "row",
}));

const LanguageLabel = styled(Typography)(({ theme }) => ({
    color: theme.palette.primary.contrastText,
    flexGrow: 1,
    fontStyle: "italic",
    marginLeft: theme.spacing(1),
}));

const InfoBarButton = styled(IconButton)(({ theme }) => ({
    color: theme.palette.primary.contrastText,
    backgroundColor: theme.palette.primary.dark,
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
    height: "fit-content",
} as const;

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
    // const { chat } = useActiveChat();

    const availableLanguages = Array.isArray(limitTo) && limitTo.length > 0 ? limitTo : Object.values(CodeLanguage);

    const codeMirrorRef = useRef<ReactCodeMirrorRef | null>(null);
    const commandFunctionsRef = useRef<{ undo: ((view: EditorView) => unknown) | null, redo: ((view: EditorView) => unknown) | null }>({ undo: null, redo: null });

    // Add state for expand/collapse and word wrap
    const [isCollapsed, setIsCollapsed] = useState(false);
    const [isWrapped, setIsWrapped] = useState(false);

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
            const { ErrorMarker, WarnMarker } = await import("./codeMirrorMarkers.js");
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

    // const openAssistantDialog = useCallback(() => {
    //     if (disabled) return;
    //     const userId = getCurrentUser(session)?.id;
    //     if (!userId) return;
    //     const chatId = chat?.id;
    //     if (!chatId) return;

    //     if (!codeMirrorRef.current || !codeMirrorRef.current.view) {
    //         console.error("CodeMirror not found");
    //         return;
    //     }
    //     const codeDoc = codeMirrorRef.current.view.state.doc;
    //     const selectionRanges = codeMirrorRef.current.view.state.selection.ranges;
    //     // Only use the first selection range, if it exists
    //     const selection = selectionRanges.length > 0 ? codeDoc.sliceString(selectionRanges[0].from, selectionRanges[0].to) : "";
    //     const fullText = codeDoc.sliceString(0, Number.MAX_SAFE_INTEGER);
    //     const contextValue = generateContext(selection, fullText);

    //     // Open the side chat and provide it context
    //     //TODO
    //     // PubSub.get().publish("menu", { id: ELEMENT_IDS.LeftDrawer, isOpen: true, data: { tab: "Chat" } });
    //     const context = {
    //         id: `code-${name}`,
    //         data: contextValue,
    //         label: generateContextLabel(contextValue),
    //         template: `Code:\n\`\`\`${codeLanguage}\n<DATA>\n\`\`\``,
    //         templateVariables: { data: "<DATA>" },
    //     };
    //     PubSub.get().publish("chatTask", {
    //         chatId,
    //         contexts: {
    //             add: {
    //                 behavior: "replace",
    //                 connect: {
    //                     __type: "contextId",
    //                     data: context.id,
    //                 },
    //                 value: [context],
    //             },
    //         },
    //     });
    // }, [chat?.id, codeLanguage, disabled, name, session]);

    // Handle action buttons
    type Action = {
        iconInfo: IconInfo,
        key: string,
        label: string,
        onClick: () => unknown,
    }
    const actions = useMemo(() => {
        const actionsList: Action[] = [];
        // If user has premium, add button for AI assistant
        if (credits && BigInt(credits) > 0) {
            actionsList.push({
                iconInfo: { name: "Magic", type: "Common" },
                key: "assistant",
                label: "AI assistant",
                onClick: () => { /**openAssistantDialog();*/ },
            });
        }
        // Always add undo and redo buttons
        actionsList.push({
            iconInfo: { name: "Undo", type: "Common" },
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
            iconInfo: { name: "Redo", type: "Common" },
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
                iconInfo: { name: "OpenThread", type: "Common" },
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
    }, [credits, codeLanguage, content, t, updateContent]);

    const [, helpKey] = useMemo<[TranslationKeyLangs, TranslationKeyLangs]>(() => languageDisplayMap[codeLanguage] ?? ["Json", "JsonHelp"], [codeLanguage]);

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

    const getOptionLabel = useCallback(function getOptionLabelMemo(option: CodeLanguage) {
        const [labelKey] = languageDisplayMap[option] ?? ["Unknown"];
        return t(labelKey, { ns: "langs" });
    }, [t]);

    // Toggle collapse and wrap functions
    const toggleCollapse = useCallback(() => setIsCollapsed(prev => !prev), []);
    const toggleWrap = useCallback(() => setIsWrapped(prev => !prev), []);

    return (
        <>
            <Stack direction="column" spacing={0} sx={outerStackStyle}>
                <InfoBar>
                    {/* Display language - either as selector or label */}
                    {availableLanguages.length > 1 && !disabled ?
                        <SelectorBase
                            name="codeLanguage"
                            value={codeLanguage}
                            onChange={changeCodeLanguage}
                            disabled={disabled}
                            options={availableLanguages}
                            getOptionLabel={getOptionLabel}
                            fullWidth
                            inputAriaLabel="select language"
                            sxs={{ root: { width: "fit-content" } }}
                        /> :
                        <LanguageLabel variant="body2">
                            {t(languageDisplayMap[codeLanguage][0], { ns: "langs" })}
                        </LanguageLabel>
                    }

                    {/* Right side action buttons */}
                    <Box display="flex" gap={0}>
                        {/* Word wrap toggle */}
                        <Tooltip title={isWrapped ? t("WordWrapDisable") : t("WordWrapEnable")}>
                            <InfoBarButton
                                aria-label={isWrapped ? t("WordWrapDisable") : t("WordWrapEnable")}
                                size="small"
                                onClick={toggleWrap}
                            >
                                {isWrapped ?
                                    <IconText decorative name="List" size={ICON_SIZE_PX} /> :
                                    <IconText decorative name="WrapText" size={ICON_SIZE_PX} />
                                }
                            </InfoBarButton>
                        </Tooltip>

                        {/* Copy button */}
                        <Tooltip title={t("Copy")}>
                            <InfoBarButton
                                aria-label={t("Copy")}
                                size="small"
                                onClick={handleCopy}
                            >
                                <IconCommon decorative name="Copy" size={ICON_SIZE_PX} />
                            </InfoBarButton>
                        </Tooltip>

                        {/* Expand/Collapse button */}
                        <Tooltip title={isCollapsed ? t("Expand") : t("Collapse")}>
                            <InfoBarButton
                                aria-label={isCollapsed ? t("Expand") : t("Collapse")}
                                size="small"
                                onClick={toggleCollapse}
                            >
                                {isCollapsed ?
                                    <IconCommon decorative name="ExpandMore" size={ICON_SIZE_PX} /> :
                                    <IconCommon decorative name="ExpandLess" size={ICON_SIZE_PX} />
                                }
                            </InfoBarButton>
                        </Tooltip>

                        {/* Show additional actions and help button when editable */}
                        {!disabled && (
                            <>
                                {actions.map(({ iconInfo, key, label, onClick }) => (
                                    <Tooltip key={key} title={label}>
                                        <InfoBarButton
                                            aria-label={label}
                                            size="small"
                                            onClick={onClick}
                                        >
                                            <Icon
                                                decorative
                                                fill={palette.primary.contrastText}
                                                info={iconInfo}
                                                size={ICON_SIZE_PX}
                                            />
                                        </InfoBarButton>
                                    </Tooltip>
                                ))}
                                <HelpButton
                                    markdown={t(helpKey, { ns: "langs" })}
                                    sx={{ fill: palette.primary.contrastText }}
                                />
                                {supportsValidation && (
                                    <StatusButton
                                        status={errors.length === 0 ? Status.Valid : Status.Invalid}
                                        messages={errors.map(e => e.message)}
                                        sx={statusButtonStyle}
                                    />
                                )}
                            </>
                        )}
                    </Box>
                </InfoBar>
                <Box sx={{
                    position: "relative",
                    height: isCollapsed ? "0px" : "400px",
                    maxHeight: isCollapsed ? "0px" : "400px",
                    background: palette.mode === "dark" ? "#282c34" : "#ffffff",
                    overflow: "hidden",
                    transition: "max-height 0.3s ease-in-out, height 0.3s ease-in-out",
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
                            style={{
                                ...lazyCodeMirrorStyle,
                                whiteSpace: isWrapped ? "pre-wrap" : "pre",
                            }}
                        />
                    </Suspense>
                    {showRefresh && <IconButton
                        onClick={refreshEditor}
                        sx={refreshIconStyle}
                    >
                        <IconCommon
                            decorative
                            fill={palette.background.textPrimary}
                            name="Refresh"
                            size={48}
                        />
                    </IconButton>}
                </Box>
                {/* Display message when collapsed */}
                {isCollapsed && (
                    <Typography variant="body2" sx={{
                        padding: 1,
                        color: palette.text.secondary,
                        fontStyle: "italic",
                        backgroundColor: palette.mode === "dark" ? "#282c34" : "#ffffff",
                    }}>
                        {t("CodeEditorCollapsed")}
                    </Typography>
                )}
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
