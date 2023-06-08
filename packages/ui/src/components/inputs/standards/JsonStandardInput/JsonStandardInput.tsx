import { StreamLanguage } from "@codemirror/language";
import { Diagnostic, linter } from "@codemirror/lint";
import { LangsKey } from "@local/shared";
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
import { JsonStandardInputProps } from "../types";

enum LanguageType {
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
const languageMap: { [x in LanguageType]: (() => Promise<{ main: any, linter?: any }>) } = {
    [LanguageType.Angular]: async () => {
        const { angular } = await import("@codemirror/lang-angular");
        return { main: angular() };
    },
    [LanguageType.Cpp]: async () => {
        const { cpp } = await import("@codemirror/lang-cpp");
        return { main: cpp() };
    },
    [LanguageType.Css]: async () => {
        const { css } = await import("@codemirror/lang-css");
        return { main: css() };
    },
    [LanguageType.Dockerfile]: async () => {
        const { dockerFile } = await import("@codemirror/legacy-modes/mode/dockerfile");
        return { main: StreamLanguage.define(dockerFile) };
    },
    [LanguageType.Go]: async () => {
        const { go } = await import("@codemirror/legacy-modes/mode/go");
        return { main: StreamLanguage.define(go) };
    },
    [LanguageType.Graphql]: async () => {
        const { graphql } = await import("cm6-graphql");
        return { main: graphql() };
    },
    [LanguageType.Groovy]: async () => {
        const { groovy } = await import("@codemirror/legacy-modes/mode/groovy");
        return { main: StreamLanguage.define(groovy) };
    },
    [LanguageType.Haskell]: async () => {
        const { haskell } = await import("@codemirror/legacy-modes/mode/haskell");
        return { main: StreamLanguage.define(haskell) };
    },
    [LanguageType.Html]: async () => {
        const { html } = await import("@codemirror/lang-html");
        return { main: html() };
    },
    [LanguageType.Java]: async () => {
        const { java } = await import("@codemirror/lang-java");
        return { main: java() };
    },
    [LanguageType.Javascript]: async () => {
        const { javascript } = await import("@codemirror/lang-javascript");
        return { main: javascript({ jsx: true }) };
    },
    [LanguageType.Json]: async () => {
        const { json, jsonParseLinter } = await import("@codemirror/lang-json");
        return { main: json(), linter: jsonParseLinter() };
    },
    [LanguageType.Nginx]: async () => {
        const { nginx } = await import("@codemirror/legacy-modes/mode/nginx");
        return { main: StreamLanguage.define(nginx) };
    },
    [LanguageType.Nix]: async () => {
        const { nix } = await import("@replit/codemirror-lang-nix");
        return { main: nix() };
    },
    [LanguageType.Php]: async () => {
        const { php } = await import("@codemirror/lang-php");
        return { main: php() };
    },
    [LanguageType.Powershell]: async () => {
        const { powerShell } = await import("@codemirror/legacy-modes/mode/powershell");
        return { main: StreamLanguage.define(powerShell) };
    },
    [LanguageType.Protobuf]: async () => {
        const { protobuf } = await import("@codemirror/legacy-modes/mode/protobuf");
        return { main: StreamLanguage.define(protobuf) };
    },
    [LanguageType.Puppet]: async () => {
        const { puppet } = await import("@codemirror/legacy-modes/mode/puppet");
        return { main: StreamLanguage.define(puppet) };
    },
    [LanguageType.Python]: async () => {
        const { python } = await import("@codemirror/lang-python");
        return { main: python() };
    },
    [LanguageType.R]: async () => {
        const { r } = await import("codemirror-lang-r");
        return { main: r() };
    },
    [LanguageType.Ruby]: async () => {
        const { ruby } = await import("@codemirror/legacy-modes/mode/ruby");
        return { main: StreamLanguage.define(ruby) };
    },
    [LanguageType.Rust]: async () => {
        const { rust } = await import("@codemirror/lang-rust");
        return { main: rust() };
    },
    [LanguageType.Sass]: async () => {
        const { sass } = await import("@codemirror/lang-sass");
        return { main: sass() };
    },
    [LanguageType.Shell]: async () => {
        const { shell } = await import("@codemirror/legacy-modes/mode/shell");
        return { main: StreamLanguage.define(shell) };
    },
    [LanguageType.Solidity]: async () => {
        const { solidity } = await import("@replit/codemirror-lang-solidity");
        return { main: solidity };
    },
    [LanguageType.Spreadsheet]: async () => {
        const { spreadsheet } = await import("@codemirror/legacy-modes/mode/spreadsheet");
        return { main: StreamLanguage.define(spreadsheet) };
    },
    [LanguageType.Sql]: async () => {
        const { standardSQL } = await import("@codemirror/legacy-modes/mode/sql");
        return { main: StreamLanguage.define(standardSQL) };
    },
    [LanguageType.Svelte]: async () => {
        const { svelte } = await import("@replit/codemirror-lang-svelte");
        return { main: svelte() };
    },
    [LanguageType.Swift]: async () => {
        const { swift } = await import("@codemirror/legacy-modes/mode/swift");
        return { main: StreamLanguage.define(swift) };
    },
    [LanguageType.Typescript]: async () => {
        const { javascript } = await import("@codemirror/lang-javascript");
        return { main: javascript({ jsx: true, typescript: true }) };
    },
    [LanguageType.Vb]: async () => {
        const { vb } = await import("@codemirror/legacy-modes/mode/vb");
        return { main: StreamLanguage.define(vb) };
    },
    [LanguageType.Vbscript]: async () => {
        const { vbScript } = await import("@codemirror/legacy-modes/mode/vbscript");
        return { main: StreamLanguage.define(vbScript) };
    },
    [LanguageType.Verilog]: async () => {
        const { verilog } = await import("@codemirror/legacy-modes/mode/verilog");
        return { main: StreamLanguage.define(verilog) };
    },
    [LanguageType.Vhdl]: async () => {
        const { vhdl } = await import("@codemirror/legacy-modes/mode/vhdl");
        return { main: StreamLanguage.define(vhdl) };
    },
    [LanguageType.Vue]: async () => {
        const { vue } = await import("@codemirror/lang-vue");
        return { main: vue() };
    },
    [LanguageType.Xml]: async () => {
        const { xml } = await import("@codemirror/lang-xml");
        return { main: xml() };
    },
    [LanguageType.Yacas]: async () => {
        const { yacas } = await import("@codemirror/legacy-modes/mode/yacas");
        return { main: StreamLanguage.define(yacas) };
    },
    [LanguageType.Yaml]: async () => {
        const { yaml } = await import("@codemirror/legacy-modes/mode/yaml");
        return { main: StreamLanguage.define(yaml) };
    },
};

/**
 * Maps languages to their labels and help texts.
 */
const languageDisplayMap: { [x in LanguageType]: [LangsKey, LangsKey] } = {
    [LanguageType.Angular]: ["Angular", "AngularHelp"],
    [LanguageType.Cpp]: ["Cpp", "CppHelp"],
    [LanguageType.Css]: ["Css", "CssHelp"],
    [LanguageType.Dockerfile]: ["Dockerfile", "DockerfileHelp"],
    [LanguageType.Go]: ["Go", "GoHelp"],
    [LanguageType.Graphql]: ["Graphql", "GraphqlHelp"],
    [LanguageType.Groovy]: ["Groovy", "GroovyHelp"],
    [LanguageType.Haskell]: ["Haskell", "HaskellHelp"],
    [LanguageType.Html]: ["Html", "HtmlHelp"],
    [LanguageType.Java]: ["Java", "JavaHelp"],
    [LanguageType.Javascript]: ["Javascript", "JavascriptHelp"],
    [LanguageType.Json]: ["Json", "JsonHelp"],
    [LanguageType.Nginx]: ["Nginx", "NginxHelp"],
    [LanguageType.Nix]: ["Nix", "NixHelp"],
    [LanguageType.Php]: ["Php", "PhpHelp"],
    [LanguageType.Powershell]: ["Powershell", "PowershellHelp"],
    [LanguageType.Protobuf]: ["Protobuf", "ProtobufHelp"],
    [LanguageType.Puppet]: ["Puppet", "PuppetHelp"],
    [LanguageType.Python]: ["Python", "PythonHelp"],
    [LanguageType.R]: ["R", "RHelp"],
    [LanguageType.Ruby]: ["Ruby", "RubyHelp"],
    [LanguageType.Rust]: ["Rust", "RustHelp"],
    [LanguageType.Sass]: ["Sass", "SassHelp"],
    [LanguageType.Shell]: ["Shell", "ShellHelp"],
    [LanguageType.Solidity]: ["Solidity", "SolidityHelp"],
    [LanguageType.Spreadsheet]: ["Spreadsheet", "SpreadsheetHelp"],
    [LanguageType.Sql]: ["Sql", "SqlHelp"],
    [LanguageType.Svelte]: ["Svelte", "SvelteHelp"],
    [LanguageType.Swift]: ["Swift", "SwiftHelp"],
    [LanguageType.Typescript]: ["Typescript", "TypescriptHelp"],
    [LanguageType.Vb]: ["Vb", "VbHelp"],
    [LanguageType.Vbscript]: ["Vbscript", "VbscriptHelp"],
    [LanguageType.Verilog]: ["Verilog", "VerilogHelp"],
    [LanguageType.Vhdl]: ["Vhdl", "VhdlHelp"],
    [LanguageType.Vue]: ["Vue", "VueHelp"],
    [LanguageType.Xml]: ["Xml", "XmlHelp"],
    [LanguageType.Yacas]: ["Yacas", "YacasHelp"],
    [LanguageType.Yaml]: ["Yaml", "YamlHelp"],
};

// TODO for morning: this component is supposed to be for creating a standard for inputting JSON in JSONInput. 
// I think all of this multi-language stuff should be in JsonInput instead, and then rename that component to CodeInput 
// or something, with a prop to limit the language.
// TODO 1.5: Instead of first TODO, may be able to combine JsonStandardInput and JsonInput into one component. To do this, make "JSON Standard" a 
// new language option. Also need to add support for format (which is JSON Standard) which, if provided, limits the language to JSON 
// and only makes input valid if it matches the format. Doing this will make this component stand out from the other 
// "standard input" components, but the duplicate code prevention may be worth it.
// TODO 2: After this stuff is done, update TopBar to support icon and action buttons, like the Header and Subheader components
export const JsonStandardInput = ({
    isEditing,
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

    // Handle language selection
    const [mode, setMode] = useState<LanguageType>(LanguageType.Json);
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
        <Stack direction="column" spacing={0}>
            {/* Bar above TextField, for status and HelpButton */}
            <Box sx={{
                display: "flex",
                width: "100%",
                padding: "0.5rem",
                borderBottom: "1px solid #e0e0e0",
                background: palette.primary.light,
                color: palette.primary.contrastText,
                borderRadius: "0.5rem 0.5rem 0 0",
                alignItems: "center",
            }}>
                {/* Select language */}
                <SelectorBase
                    name="mode"
                    value={mode}
                    onChange={setMode}
                    disabled={!isEditing}
                    options={Object.keys(languageMap) as LanguageType[]}
                    getOptionLabel={(r) => t(languageDisplayMap[r as LanguageType][0], { ns: "langs" })}
                    fullWidth
                    inputAriaLabel="select language"
                    label={"Language"}
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
                {/* Help button */}
                <HelpButton
                    markdown={t(help, { ns: "langs" })}
                    sxRoot={{ marginRight: 1, fill: palette.secondary.light }}
                />
            </Box>
            <CodeMirror
                value={internalValue}
                theme={palette.mode === "dark" ? "dark" : "light"}
                extensions={extensions}
                onChange={updateInternalValue}
                height={"400px"}
                style={{
                    borderRadius: "0 0 0.5rem 0.5rem",
                }}
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
