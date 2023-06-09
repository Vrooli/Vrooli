import { LanguageSupport, StreamLanguage } from "@codemirror/language";
import { Diagnostic, linter } from "@codemirror/lint";
import { ErrorIcon, LangsKey, WarningIcon } from "@local/shared";
import { Box, Stack, Typography, useTheme } from "@mui/material";
import CodeMirror from "@uiw/react-codemirror";
import { HelpButton } from "components/buttons/HelpButton/HelpButton";
import { StatusButton } from "components/buttons/StatusButton/StatusButton";
import { SelectorBase } from "components/inputs/SelectorBase/SelectorBase";
import { useField } from "formik";
import { JsonProps } from "forms/types";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Status } from "utils/consts";
import { jsonToString } from "utils/shape/general";
// import { isJson } from "utils/shape/general"; // Update this so that we can lint JSON standard input type (different from normal JSON)
import { Extension } from "@codemirror/state";
import { BlockInfo, EditorView, gutter, GutterMarker } from "@codemirror/view";
import ReactDOMServer from "react-dom/server";
import { JsonStandardInputProps } from "../types";

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
    Json = "json",
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

/**
 * Dynamically imports language packages.
 */
const languageMap: { [x in StandardLanguage]: (() => Promise<{
    main: LanguageSupport | StreamLanguage<any> | Extension,
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
        return { main: StreamLanguage.define(dockerFile) };
    },
    [StandardLanguage.Go]: async () => {
        const { go } = await import("@codemirror/legacy-modes/mode/go");
        return { main: StreamLanguage.define(go) };
    },
    [StandardLanguage.Graphql]: async () => {
        const { graphql } = await import("cm6-graphql");
        return { main: graphql() };
    },
    [StandardLanguage.Groovy]: async () => {
        const { groovy } = await import("@codemirror/legacy-modes/mode/groovy");
        return { main: StreamLanguage.define(groovy) };
    },
    [StandardLanguage.Haskell]: async () => {
        const { haskell } = await import("@codemirror/legacy-modes/mode/haskell");
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
        return { main: json(), linter: jsonParseLinter() };
    },
    [StandardLanguage.Nginx]: async () => {
        const { nginx } = await import("@codemirror/legacy-modes/mode/nginx");
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
        return { main: StreamLanguage.define(powerShell) };
    },
    [StandardLanguage.Protobuf]: async () => {
        const { protobuf } = await import("@codemirror/legacy-modes/mode/protobuf");
        return { main: StreamLanguage.define(protobuf) };
    },
    [StandardLanguage.Puppet]: async () => {
        const { puppet } = await import("@codemirror/legacy-modes/mode/puppet");
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
        return { main: StreamLanguage.define(shell) };
    },
    [StandardLanguage.Solidity]: async () => {
        const { solidity } = await import("@replit/codemirror-lang-solidity");
        return { main: solidity };
    },
    [StandardLanguage.Spreadsheet]: async () => {
        const { spreadsheet } = await import("@codemirror/legacy-modes/mode/spreadsheet");
        return { main: StreamLanguage.define(spreadsheet) };
    },
    [StandardLanguage.Sql]: async () => {
        const { standardSQL } = await import("@codemirror/legacy-modes/mode/sql");
        return { main: StreamLanguage.define(standardSQL) };
    },
    [StandardLanguage.Svelte]: async () => {
        const { svelte } = await import("@replit/codemirror-lang-svelte");
        return { main: svelte() };
    },
    [StandardLanguage.Swift]: async () => {
        const { swift } = await import("@codemirror/legacy-modes/mode/swift");
        return { main: StreamLanguage.define(swift) };
    },
    [StandardLanguage.Typescript]: async () => {
        const { javascript } = await import("@codemirror/lang-javascript");
        return { main: javascript({ jsx: true, typescript: true }) };
    },
    [StandardLanguage.Vb]: async () => {
        const { vb } = await import("@codemirror/legacy-modes/mode/vb");
        return { main: StreamLanguage.define(vb) };
    },
    [StandardLanguage.Vbscript]: async () => {
        const { vbScript } = await import("@codemirror/legacy-modes/mode/vbscript");
        return { main: StreamLanguage.define(vbScript) };
    },
    [StandardLanguage.Verilog]: async () => {
        const { verilog } = await import("@codemirror/legacy-modes/mode/verilog");
        return { main: StreamLanguage.define(verilog) };
    },
    [StandardLanguage.Vhdl]: async () => {
        const { vhdl } = await import("@codemirror/legacy-modes/mode/vhdl");
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
        return { main: StreamLanguage.define(yacas) };
    },
    [StandardLanguage.Yaml]: async () => {
        const { yaml } = await import("@codemirror/legacy-modes/mode/yaml");
        return { main: StreamLanguage.define(yaml) };
    },
};

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

// TODO for morning: this component is supposed to be for creating a standard for inputting JSON in JSONInput. 
// I think all of this multi-language stuff should be in JsonInput instead, and then rename that component to CodeInput 
// or something, with a prop to limit the language.
// TODO 1.5: Instead of first TODO, may be able to combine JsonStandardInput and JsonInput into one component. To do this, make "JSON Standard" a 
// new language option. Also need to add support for format (which is JSON Standard) which, if provided, limits the language to JSON 
// and only makes input valid if it matches the format. Doing this will make this component stand out from the other 
// "standard input" components, but the duplicate code prevention may be worth it.
export const JsonStandardInput = ({
    isEditing,
    limitTo,
}: JsonStandardInputProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();

    const [defaultValueField, , defaultValueHelpers] = useField<JsonProps["defaultValue"]>("defaultValue");
    const [formatField, formatMeta, formatHelpers] = useField<JsonProps["format"]>("format");
    const [variablesField, , variablesHelpers] = useField<JsonProps["variables"]>("variables");

    // Last valid schema format
    const [internalValue, setInternalValue] = useState<string>(jsonToString(formatField.value ?? {}) ?? "");
    const updateInternalValue = useCallback((value: string) => {
        if (!isEditing) return;
        setInternalValue(value);
    }, [isEditing]);

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
    const [mode, setMode] = useState<StandardLanguage>(StandardLanguage.Json);
    const [extensions, setExtensions] = useState<any[]>([]);
    const [supportsValidation, setSupportsValidation] = useState<boolean>(false);
    useEffect(() => {
        if (mode in languageMap) {
            languageMap[mode]().then(({ main, linter }) => {
                console.log("imported extensions", main, linter);
                const updatedExtensions = [main];
                if (linter) {
                    updatedExtensions.push(wrappedLinter(linter));
                    setSupportsValidation(true);
                }
                else {
                    setErrors([]);
                    setSupportsValidation(false);
                }
                setExtensions(updatedExtensions);
            });
        }
    }, [mode]);

    // Find language label and help text
    const [label, help] = useMemo<[LangsKey, LangsKey]>(() => languageDisplayMap[mode] ?? ["Json", "JsonHelp"], [mode]);

    return (
        <Stack direction="column" spacing={0} sx={{
            borderRadius: 1.5,
            overflow: "hidden",
        }}>
            {/* Bar above TextField, for status and HelpButton */}
            <Box sx={{
                display: "flex",
                width: "100%",
                padding: "0.5rem",
                borderBottom: "1px solid #e0e0e0",
                background: palette.primary.light,
                color: palette.primary.contrastText,
                alignItems: "center",
            }}>
                {/* Select language */}
                {availableLanguages.length > 0 && <SelectorBase
                    name="mode"
                    value={mode}
                    onChange={setMode}
                    disabled={!isEditing}
                    options={availableLanguages}
                    getOptionLabel={(r) => t(languageDisplayMap[r as StandardLanguage][0], { ns: "langs" })}
                    fullWidth
                    inputAriaLabel="select language"
                    label={"Language"}
                    sx={{
                        width: "fit-content",
                        minWidth: "200px",
                    }}
                />}
                {/* Help button */}
                <HelpButton
                    markdown={t(help, { ns: "langs" })}
                    sxRoot={{ marginRight: 1, fill: palette.secondary.light }}
                />
                {/* Status */}
                {supportsValidation && <StatusButton
                    status={errors.length === 0 ? Status.Valid : Status.Invalid}
                    messages={errors.map(e => e.message)}
                    sx={{
                        marginLeft: 1,
                        marginRight: "auto",
                        height: "fit-content",
                    }}
                />}
            </Box>
            <CodeMirror
                value={internalValue}
                theme={palette.mode === "dark" ? "dark" : "light"}
                extensions={[...extensions, errorGutter]}
                onChange={updateInternalValue}
                height={"400px"}
            />
            {/* Bottom bar containing arrow buttons to switch to different incomplete/incorrect
             parts of the JSON, and an input for entering the currently-selected section of JSON */}
            {/* TODO */}
            {/* Helper text label */}
            {
                formatMeta.error &&
                <Typography variant="body1" sx={{ color: "red" }}>
                    {formatMeta.error}
                </Typography>
            }
        </Stack>
    );
};
