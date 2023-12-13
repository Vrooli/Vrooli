import { Diagnostic, linter } from "@codemirror/lint";
import { Range } from "@codemirror/state";
import { ChatInviteStatus, DUMMY_ID, LangsKey, uuid } from "@local/shared";
import { Box, Grid, IconButton, Stack, Tooltip, useTheme } from "@mui/material";
import { HelpButton } from "components/buttons/HelpButton/HelpButton";
import { StatusButton } from "components/buttons/StatusButton/StatusButton";
import { SelectorBase } from "components/inputs/SelectorBase/SelectorBase";
import { useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Status } from "utils/consts";
import { jsonToString } from "utils/shape/general";
// import { isJson } from "utils/shape/general"; // Update this so that we can lint JSON standard input type (different from normal JSON)
import { redo, undo } from "@codemirror/commands";
import { StateField } from "@codemirror/state";
import { BlockInfo, Decoration, EditorView, GutterMarker, gutter, showTooltip } from "@codemirror/view";
import CodeMirror from "codemirror";
import "codemirror/addon/mode/loadmode";
import "codemirror/lib/codemirror.css";
import "codemirror/mode/meta";
import { SessionContext } from "contexts/SessionContext";
import { ErrorIcon, MagicIcon, OpenThreadIcon, RedoIcon, UndoIcon, WarningIcon } from "icons";
import ReactDOMServer from "react-dom/server";
import { SvgComponent } from "types";
import { getCurrentUser } from "utils/authentication/session";
import { getCookieMatchingChat } from "utils/cookies";
import { PubSub } from "utils/pubsub";
import { ChatShape } from "utils/shape/models/chat";
import { ChatCrud, VALYXA_INFO } from "views/objects/chat/ChatCrud/ChatCrud";
import { ChatCrudProps } from "views/objects/chat/types";
import { CodeInputBaseProps } from "../types";

// declare module "codemirror" {
//     interface Editor {
//         // Add methods that are missing in the typings
//         autoLoadMode: (instance: CodeMirror.Editor, mode: string) => void;
//         // ... other methods
//     }
// }

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

const underlineMarkVariable = Decoration.mark({ class: "variable-decoration" });
const underlineMarkOptional = Decoration.mark({ class: "optional-decoration" });
const underlineMarkWildcard = Decoration.mark({ class: "wildcard-decoration" });
const underlineMarkError = Decoration.mark({ class: "error-decoration" });
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

function underlineVariables(doc) {
    const decorations: Range<Decoration>[] = [];
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

// /** Maps language types to their import location */
// const languageImportMap: { [x in StandardLanguage]: string } = {
//     [StandardLanguage.Angular]: "@codemirror/lang-angular",
//     [StandardLanguage.Cpp]: "@codemirror/lang-cpp",
//     [StandardLanguage.Css]: "@codemirror/lang-css",
//     [StandardLanguage.Dockerfile]: "@codemirror/legacy-modes/mode/dockerfile",
//     [StandardLanguage.Go]: "@codemirror/legacy-modes/mode/go",
//     [StandardLanguage.Graphql]: "cm6-graphql",
//     [StandardLanguage.Groovy]: "@codemirror/legacy-modes/mode/groovy",
//     [StandardLanguage.Haskell]: "@codemirror/legacy-modes/mode/haskell",
//     [StandardLanguage.Html]: "@codemirror/lang-html",
//     [StandardLanguage.Java]: "@codemirror/lang-java",
//     [StandardLanguage.Javascript]: "@codemirror/lang-javascript",
//     [StandardLanguage.Json]: "@codemirror/lang-json",
//     [StandardLanguage.JsonStandard]: "@codemirror/lang-json",
//     [StandardLanguage.Nginx]: "@codemirror/legacy-modes/mode/nginx",
//     [StandardLanguage.Nix]: "@replit/codemirror-lang-nix",
//     [StandardLanguage.Php]: "@codemirror/lang-php",
//     [StandardLanguage.Powershell]: "@codemirror/legacy-modes/mode/powershell",
//     [StandardLanguage.Protobuf]: "@codemirror/legacy-modes/mode/protobuf",
//     [StandardLanguage.Puppet]: "@codemirror/legacy-modes/mode/puppet",
//     [StandardLanguage.Python]: "@codemirror/lang-python",
//     [StandardLanguage.R]: "codemirror-lang-r",
//     [StandardLanguage.Ruby]: "@codemirror/legacy-modes/mode/ruby",
//     [StandardLanguage.Rust]: "@codemirror/lang-rust",
//     [StandardLanguage.Sass]: "@codemirror/lang-sass",
//     [StandardLanguage.Shell]: "@codemirror/legacy-modes/mode/shell",
//     [StandardLanguage.Solidity]: "@replit/codemirror-lang-solidity",
//     [StandardLanguage.Spreadsheet]: "@codemirror/legacy-modes/mode/spreadsheet",
//     [StandardLanguage.Sql]: "@codemirror/legacy-modes/mode/sql",
//     [StandardLanguage.Svelte]: "@replit/codemirror-lang-svelte",
//     [StandardLanguage.Swift]: "@codemirror/legacy-modes/mode/swift",
//     [StandardLanguage.Typescript]: "@codemirror/lang-javascript",
//     [StandardLanguage.Vb]: "@codemirror/legacy-modes/mode/vb",
//     [StandardLanguage.Vbscript]: "@codemirror/legacy-modes/mode/vbscript",
//     [StandardLanguage.Verilog]: "@codemirror/legacy-modes/mode/verilog",
//     [StandardLanguage.Vhdl]: "@codemirror/legacy-modes/mode/vhdl",
//     [StandardLanguage.Vue]: "@codemirror/lang-vue",
//     [StandardLanguage.Xml]: "@codemirror/lang-xml",
//     [StandardLanguage.Yacas]: "@codemirror/legacy-modes/mode/yacas",
//     [StandardLanguage.Yaml]: "@codemirror/legacy-modes/mode/yaml",
// };

class ErrorMarker extends GutterMarker {
    toDOM() {
        const marker = document.createElement("div");
        marker.innerHTML = ReactDOMServer.renderToString(<ErrorIcon fill="red" />);
        return marker;
    }
}

class WarnMarker extends GutterMarker {
    toDOM() {
        const marker = document.createElement("div");
        marker.innerHTML = ReactDOMServer.renderToString(<WarningIcon fill="yellow" />);
        return marker;
    }
}
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
    console.log("codeinputbase", limitTo);
    const { palette } = useTheme();
    const { t } = useTranslation();
    const session = useContext(SessionContext);
    const { hasPremium } = useMemo(() => getCurrentUser(session), [session]);

    // const codeMirrorRef = useRef<ReactCodeMirrorRef | null>(null);
    const codeMirrorRef = useRef<Editor | null>(null);
    useEffect(() => {
        // Assuming you have a textarea with a specific ID in your render method
        const textAreaElement = document.getElementById("code-editor-textarea");

        if (textAreaElement) {
            // Initialize CodeMirror on the textarea
            const editor = CodeMirror.fromTextArea(textAreaElement, {
                value: internalValue,
                lineNumbers: true,
                mode: "text/plain", // Default mode, you can set this based on your requirements
                // Add other options as needed
            });

            // Store the editor instance in a ref for later use
            codeMirrorRef.current = editor;

            // Optional: Add an event listener if you want to handle changes
            editor.on("change", (cm) => {
                updateInternalValue(cm.getValue());
            });

            // Set the initial mode based on the component's state
            const info = CodeMirror.findModeByName(mode);
            if (info) {
                CodeMirror.autoLoadMode(editor, info.mode);
                editor.setOption("mode", info.mime);
            }
        }

        // Cleanup function to detach CodeMirror from the DOM when the component unmounts
        return () => {
            if (codeMirrorRef.current) {
                codeMirrorRef.current.toTextArea();
            }
        };
    }, []); // Empty dependency array to ensure this runs only once on mount

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
    const wrappedLinter = (nextLinter) => {
        return linter(view => {
            // Run the existing linter and get its diagnostics.
            const diagnostics = nextLinter(view);
            // Count the number of errors.
            const errors = diagnostics.filter(d => d.severity === "error");
            // Update state
            console.log("setting errors", errors);
            setErrors(errors);
            // Return the diagnostics so they still get displayed.
            return diagnostics;
        });
    };
    // Locate errors
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
    // Display errors in gutter
    const errorGutter = gutter({
        lineMarker: (view, line) => {
            const severity = getSeverityForLine(line, errors, view);
            if (severity === "error") {
                return new ErrorMarker();
            } else if (severity === "warning") {
                return new WarnMarker();
            }
            return null;
        },
        class: "cm-errorGutter",
    });

    // Handle language selection
    // CodeMirror.modeURL = "/codemirror/%N/%N.js";
    const [mode, setMode] = useState<StandardLanguage>(limitTo && limitTo.length > 0 ? limitTo[0] : StandardLanguage.Json);
    const [extensions, setExtensions] = useState<any[]>([]);
    const [supportsValidation, setSupportsValidation] = useState<boolean>(false);
    // useEffect(() => {
    //     if (codeMirrorRef.current && mode in languageMap) {
    //         const editor = codeMirrorRef.current.editor;
    //         const info = CodeMirror.findModeByName(mode);

    //         if (info) {
    //             CodeMirror.autoLoadMode(editor, info.mode);
    //             editor.setOption("mode", info.mime);
    //         }

    //         languageMap[mode]().then(({ main, linter }) => {
    //             const updatedExtensions = [main];
    //             if (linter) {
    //                 updatedExtensions.push(wrappedLinter(linter));
    //                 setSupportsValidation(true);
    //             } else {
    //                 setErrors([]);
    //                 setSupportsValidation(false);
    //             }
    //             // If mode is JSON standard, add additional extensions for variables
    //             if (mode === StandardLanguage.JsonStandard) {
    //                 updatedExtensions.push(
    //                     cursorTooltipField, // Handle tooltips for JSON variables
    //                     underlineDecorationField, // Underline JSON variables
    //                 );
    //             }
    //             setExtensions(updatedExtensions);
    //         });
    //     }
    // }, [mode]);

    useEffect(() => {
        if (codeMirrorRef.current) {
            // Find the mode information based on the selected mode
            const info = CodeMirror.findModeByName(mode);

            if (info) {
                // Dynamically load the mode script if it's not already loaded
                CodeMirror.requireMode(info.mode, () => {
                    // Once the mode script is loaded, set the mode on the editor
                    codeMirrorRef.current.setOption("mode", info.mime);
                });
            }

            // If the mode has an associated linter or other extensions, load them here
            if (mode in languageMap) {
                languageMap[mode]().then(({ main, linter }) => {
                    // Assuming 'main' is a function that applies the necessary
                    // extensions or configurations to the CodeMirror instance
                    main(codeMirrorRef.current);

                    if (linter) {
                        // Apply the linter or other extensions
                        // This part depends on how your linters are set up
                        // For example, you might do something like:
                        codeMirrorRef.current.setOption("lint", linter);
                        setSupportsValidation(true);
                    } else {
                        setErrors([]);
                        setSupportsValidation(false);
                    }
                });
            }
        }
    }, [mode]);

    useEffect(() => {
        if (codeMirrorRef.current) {
            codeMirrorRef.current.on("change", (cm) => {
                updateInternalValue(cm.getValue());
            });
        }
    }, []);

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

        // We want to provide the assistant with the most relevant context
        let context: string | undefined = undefined;
        const maxContextLength = 1500;
        // Get highlighted text
        let highlightedText = "";
        const selection = codeMirrorRef.current?.view?.state?.selection;
        if (selection) {
            const { from, to } = selection.ranges[0];
            highlightedText = codeMirrorRef.current?.view?.state?.doc?.sliceString(from, to) ?? "";
            console.log("got selection", from, to, highlightedText);
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
        const existingChatId = getCookieMatchingChat([VALYXA_INFO.id], "standard");
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
        const actions: Action[] = [];
        // If user has premium, add button for AI assistant
        if (hasPremium) {
            actions.push({
                label: "AI assistant",
                Icon: MagicIcon,
                onClick: () => { openAssistantDialog(); },
            });
        }
        // Always add undo and redo buttons
        actions.push({
            label: t("Undo"),
            Icon: UndoIcon,
            onClick: () => {
                codeMirrorRef.current?.view !== undefined && undo(codeMirrorRef.current.view);
            },
        }, {
            label: t("Redo"),
            Icon: RedoIcon,
            onClick: () => {
                codeMirrorRef.current?.view !== undefined && redo(codeMirrorRef.current.view);
            },
        });
        // For json and jsonStandard, add "pretty print" button to format JSON
        if (mode === StandardLanguage.Json || mode === StandardLanguage.JsonStandard) {
            actions.push({
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
        return actions;
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
                                onChange={setMode}
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
                <textarea
                    id="your-textarea-id"
                    ref={codeMirrorRef}
                    defaultValue={internalValue}
                ></textarea>
                {/* <CodeMirror
                    ref={codeMirrorRef}
                    value={internalValue}
                    theme={palette.mode === "dark" ? "dark" : "light"}
                    extensions={[
                        ...extensions, // Language-specific extensions
                        errorGutter, // Display warnings and errors in gutter
                        EditorView.baseTheme({ // Custom theme
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
                        })]}
                    onChange={updateInternalValue}
                    height={"400px"}
                /> */}
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
