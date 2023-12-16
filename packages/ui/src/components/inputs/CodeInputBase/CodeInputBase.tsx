import { ChatInviteStatus, DUMMY_ID, LangsKey, isEqual, uuid } from "@local/shared";
import { Box, Grid, IconButton, Stack, Tooltip, useTheme } from "@mui/material";
import { HelpButton } from "components/buttons/HelpButton/HelpButton";
import { StatusButton } from "components/buttons/StatusButton/StatusButton";
import { SelectorBase } from "components/inputs/SelectorBase/SelectorBase";
import { Suspense, useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Status } from "utils/consts";
import { jsonToString } from "utils/shape/general";
// import { isJson } from "utils/shape/general"; // Update this so that we can lint JSON standard input type (different from normal JSON)
import { SessionContext } from "contexts/SessionContext";
import { MagicIcon, OpenThreadIcon, RedoIcon, UndoIcon } from "icons";
import React from "react";
import { SvgComponent } from "types";
import { getCurrentUser } from "utils/authentication/session";
import { getCookieMatchingChat } from "utils/cookies";
import { PubSub } from "utils/pubsub";
import { ChatShape } from "utils/shape/models/chat";
import { ChatCrud, VALYXA_INFO } from "views/objects/chat/ChatCrud/ChatCrud";
import { ChatCrudProps } from "views/objects/chat/types";
import { CodeInputBaseProps } from "../types";

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

export enum StandardLanguage {
    Angular = "angular",
    Cpp = "cpp",
    Css = "css",
    Dockerfile = "dockerfile",
    Go = "go",
    Graphql = "graphql",
    Groovy = "groovy",
    Haskell = "haskell",
    Html = "html",
    Java = "java",
    Javascript = "javascript",
    Json = "json", // JSON which may or may not conform to a standard
    JsonStandard = "jsonStandard", // JSON which defines a standard for some data (could be JSON, a file, etc.)
    Nginx = "nginx",
    Nix = "nix",
    Php = "php",
    Powershell = "powershell",
    Protobuf = "protobuf",
    Puppet = "puppet",
    Python = "python",
    R = "r",
    Ruby = "ruby",
    Rust = "rust",
    Sass = "sass",
    Shell = "shell",
    Solidity = "solidity",
    Spreadsheet = "spreadsheet",
    Sql = "sql",
    Svelte = "svelte",
    Swift = "swift",
    Typescript = "typescript",
    Vb = "vb",
    Vbscript = "vbscript",
    Verilog = "verilog",
    Vhdl = "vhdl",
    Vue = "vue",
    Xml = "xml",
    Yacas = "yacas",
    Yaml = "yaml",
}

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

    const getCursorTooltips = (state) => {
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
    };

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
const languageMap: { [x in StandardLanguage]: (() => Promise<{
    main: LanguageSupport | StreamLanguage | Extension,
    linter?: ((view: EditorView) => Diagnostic[]) | Extension,
}>) } = {
    [StandardLanguage.Angular]: async () => {
        const { angular } = await import("@codemirror/lang-angular");
        return { main: angular() };
    },
    [StandardLanguage.Cpp]: async () => {
        const { cpp } = await import("@codemirror/lang-cpp");
        return { main: cpp() };
    },
    [StandardLanguage.Css]: async () => {
        const { css } = await import("@codemirror/lang-css");
        return { main: css() };
    },
    [StandardLanguage.Dockerfile]: async () => {
        const { dockerFile } = await import("@codemirror/legacy-modes/mode/dockerfile");
        const { StreamLanguage } = await import("@codemirror/language");
        return { main: StreamLanguage.define(dockerFile) };
    },
    [StandardLanguage.Go]: async () => {
        const { go } = await import("@codemirror/legacy-modes/mode/go");
        const { StreamLanguage } = await import("@codemirror/language");
        return { main: StreamLanguage.define(go) };
    },
    [StandardLanguage.Graphql]: async () => {
        const { graphql } = await import("cm6-graphql");
        return { main: graphql() };
    },
    [StandardLanguage.Groovy]: async () => {
        const { groovy } = await import("@codemirror/legacy-modes/mode/groovy");
        const { StreamLanguage } = await import("@codemirror/language");
        return { main: StreamLanguage.define(groovy) };
    },
    [StandardLanguage.Haskell]: async () => {
        const { haskell } = await import("@codemirror/legacy-modes/mode/haskell");
        const { StreamLanguage } = await import("@codemirror/language");
        return { main: StreamLanguage.define(haskell) };
    },
    [StandardLanguage.Html]: async () => {
        const { html } = await import("@codemirror/lang-html");
        return { main: html() };
    },
    [StandardLanguage.Java]: async () => {
        const { java } = await import("@codemirror/lang-java");
        return { main: java() };
    },
    [StandardLanguage.Javascript]: async () => {
        const { javascript } = await import("@codemirror/lang-javascript");
        return { main: javascript({ jsx: true }) };
    },
    [StandardLanguage.Json]: async () => {
        const { json, jsonParseLinter } = await import("@codemirror/lang-json");
        return { main: json(), linter: jsonParseLinter() as unknown as ((view: EditorView) => Diagnostic[]) };
    },
    [StandardLanguage.JsonStandard]: async () => {
        const { json, jsonParseLinter } = await import("@codemirror/lang-json");
        return { main: json(), linter: jsonParseLinter() as unknown as ((view: EditorView) => Diagnostic[]) };
    },
    [StandardLanguage.Nginx]: async () => {
        const { nginx } = await import("@codemirror/legacy-modes/mode/nginx");
        const { StreamLanguage } = await import("@codemirror/language");
        return { main: StreamLanguage.define(nginx) };
    },
    [StandardLanguage.Nix]: async () => {
        const { nix } = await import("@replit/codemirror-lang-nix");
        return { main: nix() };
    },
    [StandardLanguage.Php]: async () => {
        const { php } = await import("@codemirror/lang-php");
        return { main: php() };
    },
    [StandardLanguage.Powershell]: async () => {
        const { powerShell } = await import("@codemirror/legacy-modes/mode/powershell");
        const { StreamLanguage } = await import("@codemirror/language");
        return { main: StreamLanguage.define(powerShell) };
    },
    [StandardLanguage.Protobuf]: async () => {
        const { protobuf } = await import("@codemirror/legacy-modes/mode/protobuf");
        const { StreamLanguage } = await import("@codemirror/language");
        return { main: StreamLanguage.define(protobuf) };
    },
    [StandardLanguage.Puppet]: async () => {
        const { puppet } = await import("@codemirror/legacy-modes/mode/puppet");
        const { StreamLanguage } = await import("@codemirror/language");
        return { main: StreamLanguage.define(puppet) };
    },
    [StandardLanguage.Python]: async () => {
        const { python } = await import("@codemirror/lang-python");
        return { main: python() };
    },
    [StandardLanguage.R]: async () => {
        const { r } = await import("codemirror-lang-r");
        return { main: r() };
    },
    [StandardLanguage.Ruby]: async () => {
        const { ruby } = await import("@codemirror/legacy-modes/mode/ruby");
        const { StreamLanguage } = await import("@codemirror/language");
        return { main: StreamLanguage.define(ruby) };
    },
    [StandardLanguage.Rust]: async () => {
        const { rust } = await import("@codemirror/lang-rust");
        return { main: rust() };
    },
    [StandardLanguage.Sass]: async () => {
        const { sass } = await import("@codemirror/lang-sass");
        return { main: sass() };
    },
    [StandardLanguage.Shell]: async () => {
        const { shell } = await import("@codemirror/legacy-modes/mode/shell");
        const { StreamLanguage } = await import("@codemirror/language");
        return { main: StreamLanguage.define(shell) };
    },
    [StandardLanguage.Solidity]: async () => {
        const { solidity } = await import("@replit/codemirror-lang-solidity");
        return { main: solidity };
    },
    [StandardLanguage.Spreadsheet]: async () => {
        const { spreadsheet } = await import("@codemirror/legacy-modes/mode/spreadsheet");
        const { StreamLanguage } = await import("@codemirror/language");
        return { main: StreamLanguage.define(spreadsheet) };
    },
    [StandardLanguage.Sql]: async () => {
        const { standardSQL } = await import("@codemirror/legacy-modes/mode/sql");
        const { StreamLanguage } = await import("@codemirror/language");
        return { main: StreamLanguage.define(standardSQL) };
    },
    [StandardLanguage.Svelte]: async () => {
        const { svelte } = await import("@replit/codemirror-lang-svelte");
        return { main: svelte() };
    },
    [StandardLanguage.Swift]: async () => {
        const { swift } = await import("@codemirror/legacy-modes/mode/swift");
        const { StreamLanguage } = await import("@codemirror/language");
        return { main: StreamLanguage.define(swift) };
    },
    [StandardLanguage.Typescript]: async () => {
        const { javascript } = await import("@codemirror/lang-javascript");
        return { main: javascript({ jsx: true, typescript: true }) };
    },
    [StandardLanguage.Vb]: async () => {
        const { vb } = await import("@codemirror/legacy-modes/mode/vb");
        const { StreamLanguage } = await import("@codemirror/language");
        return { main: StreamLanguage.define(vb) };
    },
    [StandardLanguage.Vbscript]: async () => {
        const { vbScript } = await import("@codemirror/legacy-modes/mode/vbscript");
        const { StreamLanguage } = await import("@codemirror/language");
        return { main: StreamLanguage.define(vbScript) };
    },
    [StandardLanguage.Verilog]: async () => {
        const { verilog } = await import("@codemirror/legacy-modes/mode/verilog");
        const { StreamLanguage } = await import("@codemirror/language");
        return { main: StreamLanguage.define(verilog) };
    },
    [StandardLanguage.Vhdl]: async () => {
        const { vhdl } = await import("@codemirror/legacy-modes/mode/vhdl");
        const { StreamLanguage } = await import("@codemirror/language");
        return { main: StreamLanguage.define(vhdl) };
    },
    [StandardLanguage.Vue]: async () => {
        const { vue } = await import("@codemirror/lang-vue");
        return { main: vue() };
    },
    [StandardLanguage.Xml]: async () => {
        const { xml } = await import("@codemirror/lang-xml");
        return { main: xml() };
    },
    [StandardLanguage.Yacas]: async () => {
        const { yacas } = await import("@codemirror/legacy-modes/mode/yacas");
        const { StreamLanguage } = await import("@codemirror/language");
        return { main: StreamLanguage.define(yacas) };
    },
    [StandardLanguage.Yaml]: async () => {
        const { yaml } = await import("@codemirror/legacy-modes/mode/yaml");
        const { StreamLanguage } = await import("@codemirror/language");
        return { main: StreamLanguage.define(yaml) };
    },
};

const getSeverityForLine = (line: BlockInfo, errors: Diagnostic[], view: EditorView) => {
    const linePosStart = line.from;
    const linePosEnd = line.to;
    for (const error of errors) {
        if (error.from >= linePosStart && error.to <= linePosEnd) {
            return error.severity;
        }
    }
    return null;
};

/**
 * Maps languages to their labels and help texts.
 */
const languageDisplayMap: { [x in StandardLanguage]: [LangsKey, LangsKey] } = {
    [StandardLanguage.Angular]: ["Angular", "AngularHelp"],
    [StandardLanguage.Cpp]: ["Cpp", "CppHelp"],
    [StandardLanguage.Css]: ["Css", "CssHelp"],
    [StandardLanguage.Dockerfile]: ["Dockerfile", "DockerfileHelp"],
    [StandardLanguage.Go]: ["Go", "GoHelp"],
    [StandardLanguage.Graphql]: ["Graphql", "GraphqlHelp"],
    [StandardLanguage.Groovy]: ["Groovy", "GroovyHelp"],
    [StandardLanguage.Haskell]: ["Haskell", "HaskellHelp"],
    [StandardLanguage.Html]: ["Html", "HtmlHelp"],
    [StandardLanguage.Java]: ["Java", "JavaHelp"],
    [StandardLanguage.Javascript]: ["Javascript", "JavascriptHelp"],
    [StandardLanguage.Json]: ["Json", "JsonHelp"],
    [StandardLanguage.JsonStandard]: ["JsonStandard", "JsonStandardHelp"],
    [StandardLanguage.Nginx]: ["Nginx", "NginxHelp"],
    [StandardLanguage.Nix]: ["Nix", "NixHelp"],
    [StandardLanguage.Php]: ["Php", "PhpHelp"],
    [StandardLanguage.Powershell]: ["Powershell", "PowershellHelp"],
    [StandardLanguage.Protobuf]: ["Protobuf", "ProtobufHelp"],
    [StandardLanguage.Puppet]: ["Puppet", "PuppetHelp"],
    [StandardLanguage.Python]: ["Python", "PythonHelp"],
    [StandardLanguage.R]: ["R", "RHelp"],
    [StandardLanguage.Ruby]: ["Ruby", "RubyHelp"],
    [StandardLanguage.Rust]: ["Rust", "RustHelp"],
    [StandardLanguage.Sass]: ["Sass", "SassHelp"],
    [StandardLanguage.Shell]: ["Shell", "ShellHelp"],
    [StandardLanguage.Solidity]: ["Solidity", "SolidityHelp"],
    [StandardLanguage.Spreadsheet]: ["Spreadsheet", "SpreadsheetHelp"],
    [StandardLanguage.Sql]: ["Sql", "SqlHelp"],
    [StandardLanguage.Svelte]: ["Svelte", "SvelteHelp"],
    [StandardLanguage.Swift]: ["Swift", "SwiftHelp"],
    [StandardLanguage.Typescript]: ["Typescript", "TypescriptHelp"],
    [StandardLanguage.Vb]: ["Vb", "VbHelp"],
    [StandardLanguage.Vbscript]: ["Vbscript", "VbscriptHelp"],
    [StandardLanguage.Verilog]: ["Verilog", "VerilogHelp"],
    [StandardLanguage.Vhdl]: ["Vhdl", "VhdlHelp"],
    [StandardLanguage.Vue]: ["Vue", "VueHelp"],
    [StandardLanguage.Xml]: ["Xml", "XmlHelp"],
    [StandardLanguage.Yacas]: ["Yacas", "YacasHelp"],
    [StandardLanguage.Yaml]: ["Yaml", "YamlHelp"],
};

// TODO May be able to combine CodeInputBase and JsonInput into one component. To do this, make "JSON Standard" a 
// new language option. Also need to add support for format (which is JSON Standard) which, if provided, limits the language to JSON 
// and only makes input valid if it matches the format. Doing this will make this component stand out from the other 
// "standard input" components, but the duplicate code prevention may be worth it.
export const CodeInputBase = ({
    defaultValue,
    disabled,
    format,
    limitTo,
    variables,
}: CodeInputBaseProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const session = useContext(SessionContext);
    const { hasPremium } = useMemo(() => getCurrentUser(session), [session]);

    const codeMirrorRef = useRef<ReactCodeMirrorRef | null>(null);
    const commandFunctionsRef = useRef<{ undo: ((view: EditorView) => unknown) | null, redo: ((view: EditorView) => unknown) | null }>({ undo: null, redo: null });

    const loadCommandFunctions = async () => {
        if (!commandFunctionsRef.current.undo || !commandFunctionsRef.current.redo) {
            const commands = await import("@codemirror/commands");
            commandFunctionsRef.current.undo = commands.undo as unknown as ((view: EditorView) => unknown);
            commandFunctionsRef.current.redo = commands.redo as unknown as ((view: EditorView) => unknown);
        }
    };

    // Last valid schema format
    const [internalValue, setInternalValue] = useState<string>(jsonToString(format) ?? "");
    const updateInternalValue = useCallback((value: string) => {
        if (disabled) return;
        setInternalValue(value);
    }, [disabled]);

    // Limit language options
    const availableLanguages = useMemo(() => {
        if (limitTo) {
            return limitTo;
        } else {
            return Object.values(StandardLanguage);
        }
    }, [limitTo]);

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
    const [mode, setMode] = useState<StandardLanguage>(limitTo && limitTo.length > 0 ? limitTo[0] : StandardLanguage.Json);
    const changeMode = useCallback((mode: StandardLanguage) => {
        setErrors([]); // Reset errors
        setMode(mode);
    }, []);
    const [extensions, setExtensions] = useState<Extension[]>([]);
    const [supportsValidation, setSupportsValidation] = useState<boolean>(false);
    useEffect(() => {
        let isMounted = true;

        const loadGutter = async () => {
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
        };

        const updateExtensions = async () => {
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
                if (mode in languageMap) {
                    const { main, linter } = await languageMap[mode]();
                    updatedExtensions.push(main as Extension);
                    if (linter) {
                        const linterExtension = await wrappedLinter(linter);
                        updatedExtensions.push(linterExtension as Extension);
                        if (isMounted) setSupportsValidation(true);
                    } else if (isMounted) {
                        setSupportsValidation(false);
                    }
                    // If mode is JSON standard, add additional extensions for variables
                    if (mode === StandardLanguage.JsonStandard) {
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
        };

        updateExtensions();

        return () => {
            isMounted = false;
        };
    }, [errors, mode, palette.error.main, wrappedLinter]);

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

        // We want to provide the assistant with the most relevant context
        let context: string | undefined = undefined;
        const maxContextLength = 1500;
        // Get highlighted text
        let highlightedText = "";
        const selection = codeMirrorRef.current?.view?.state?.selection;
        if (selection) {
            const { from, to } = selection.ranges[0];
            highlightedText = codeMirrorRef.current?.view?.state?.doc?.sliceString(from, to) ?? "";
        }
        if (highlightedText.length > maxContextLength) highlightedText = highlightedText.substring(0, maxContextLength);
        if (highlightedText.length > 0) context = highlightedText;
        // If there's not highlighted text, provide the full text if it's not too long or short
        else if (internalValue.length <= maxContextLength && internalValue.length > 2) context = internalValue;
        // Otherwise, provide the last 1500 characters
        else if (internalValue.length > 2) context = internalValue.substring(internalValue.length - maxContextLength, internalValue.length);
        // Put code block around context
        if (context) context = "```\n" + context + "\n```\n\n";

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
    }, [disabled, internalValue]);

    // Handle action buttons
    type Action = {
        label: string,
        Icon: SvgComponent,
        onClick: () => unknown,
    }
    const actions = useMemo(() => {
        const actionsList: Action[] = [];
        // If user has premium, add button for AI assistant
        if (hasPremium) {
            actionsList.push({
                label: "AI assistant",
                Icon: MagicIcon,
                onClick: () => { openAssistantDialog(); },
            });
        }
        // Always add undo and redo buttons
        actionsList.push({
            label: t("Undo"),
            Icon: UndoIcon,
            onClick: async () => {
                await loadCommandFunctions();
                if (codeMirrorRef.current?.view) {
                    commandFunctionsRef.current.undo?.(codeMirrorRef.current.view);
                }
            },
        });
        actionsList.push({
            label: t("Redo"),
            Icon: RedoIcon,
            onClick: async () => {
                await loadCommandFunctions();
                if (codeMirrorRef.current?.view) {
                    commandFunctionsRef.current.redo?.(codeMirrorRef.current.view);
                }
            },
        });
        // For json and jsonStandard, add "pretty print" button to format JSON
        if (mode === StandardLanguage.Json || mode === StandardLanguage.JsonStandard) {
            actionsList.push({
                label: t("Format"),
                Icon: OpenThreadIcon,
                onClick: () => {
                    try {
                        const parsed = JSON.parse(internalValue);
                        updateInternalValue(JSON.stringify(parsed, null, 4));
                    } catch (error) {
                        PubSub.get().publish("snack", { message: "Invalid JSON", severity: "Error", data: { error } });
                    }
                },
            });
        }
        return actionsList;
    }, [hasPremium, internalValue, mode, openAssistantDialog, t, updateInternalValue]);

    // Find language label and help text
    const [label, help] = useMemo<[LangsKey, LangsKey]>(() => languageDisplayMap[mode] ?? ["Json", "JsonHelp"], [mode]);

    return (
        <>
            {/* Assistant dialog for generating text */}
            <ChatCrud {...assistantDialogProps} />
            <Stack direction="column" spacing={0} sx={{
                borderRadius: 1.5,
                overflow: "hidden",
            }}>
                {/* Bar above main input */}
                <Box sx={{
                    display: "flex",
                    width: "100%",
                    padding: "0.5rem",
                    borderBottom: "1px solid #e0e0e0",
                    background: palette.primary.light,
                    color: palette.primary.contrastText,
                    alignItems: "center",
                    flexDirection: { xs: "column", sm: "row" }, // switch to column on xs screens, row on sm and larger
                }}>
                    {/* Select language */}
                    {availableLanguages.length > 1 &&
                        <Grid item xs={12} sm={6}>
                            <SelectorBase
                                name="mode"
                                value={mode}
                                onChange={changeMode}
                                disabled={disabled}
                                options={availableLanguages}
                                getOptionLabel={(r) => t(languageDisplayMap[r as StandardLanguage][0], { ns: "langs" })}
                                fullWidth
                                inputAriaLabel="select language"
                                label={"Language"}
                                sx={{
                                    width: "fit-content",
                                    minWidth: "200px",
                                }}
                            />
                        </Grid>
                    }
                    {/* Actions, Help button, Status */}
                    <Grid item xs={12} sm={availableLanguages.length > 1 ? 6 : 12} sx={{
                        marginLeft: { xs: 0, sm: "auto" },
                        ...(availableLanguages.length <= 1 && {
                            display: "flex",
                            justifyContent: "flex-end",
                        }),
                    }}>
                        <Box sx={{
                            display: "flex",
                            flexDirection: "row",
                            justifyContent: "flex-start",
                            flexWrap: "wrap",
                            alignItems: "center",
                        }}>
                            {actions.map(({ label, Icon, onClick }, i) => <Tooltip key={i} title={label}>
                                <IconButton
                                    onClick={onClick}
                                    disabled={disabled}
                                >
                                    <Icon fill={palette.primary.contrastText} />
                                </IconButton>
                            </Tooltip>)}
                            <HelpButton
                                markdown={t(help, { ns: "langs" })}
                                sx={{ fill: palette.secondary.contrastText }}
                            />
                            {supportsValidation && <StatusButton
                                status={errors.length === 0 ? Status.Valid : Status.Invalid}
                                messages={errors.map(e => e.message)}
                                sx={{
                                    marginRight: "auto",
                                    height: "fit-content",
                                }}
                            />}
                        </Box>
                    </Grid>
                </Box>
                <Suspense fallback={<div>Loading editor...</div>}>
                    <LazyCodeMirror
                        ref={codeMirrorRef as any}
                        value={internalValue}
                        theme={palette.mode === "dark" ? "dark" : "light"}
                        extensions={extensions}
                        onChange={updateInternalValue}
                        height={"400px"}
                    />
                </Suspense>
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
            </Stack >
        </>
    );
};
