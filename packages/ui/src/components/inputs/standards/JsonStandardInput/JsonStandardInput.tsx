import { StreamLanguage } from "@codemirror/language";
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
import { isJson, jsonToString } from "utils/shape/general";
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
const languageMap: { [x in LanguageType]: (() => Promise<any>) } = {
    [LanguageType.Angular]: async () => {
        const { angular } = await import("@codemirror/lang-angular");
        return angular();
    },
    [LanguageType.Cpp]: async () => {
        const { cpp } = await import("@codemirror/lang-cpp");
        return cpp();
    },
    [LanguageType.Css]: async () => {
        const { css } = await import("@codemirror/lang-css");
        return css();
    },
    [LanguageType.Dockerfile]: async () => {
        const { dockerFile } = await import("@codemirror/legacy-modes/mode/dockerfile");
        return StreamLanguage.define(dockerFile);
    },
    [LanguageType.Go]: async () => {
        const { go } = await import("@codemirror/legacy-modes/mode/go");
        return StreamLanguage.define(go);
    },
    [LanguageType.Graphql]: async () => {
        const { graphql } = await import("cm6-graphql");
        return graphql();
    },
    [LanguageType.Groovy]: async () => {
        const { groovy } = await import("@codemirror/legacy-modes/mode/groovy");
        return StreamLanguage.define(groovy);
    },
    [LanguageType.Haskell]: async () => {
        const { haskell } = await import("@codemirror/legacy-modes/mode/haskell");
        return StreamLanguage.define(haskell);
    },
    [LanguageType.Html]: async () => {
        const { html } = await import("@codemirror/lang-html");
        return html();
    },
    [LanguageType.Java]: async () => {
        const { java } = await import("@codemirror/lang-java");
        return java();
    },
    [LanguageType.Javascript]: async () => {
        const { javascript } = await import("@codemirror/lang-javascript");
        return javascript({ jsx: true });
    },
    [LanguageType.Json]: async () => {
        const { json } = await import("@codemirror/lang-json");
        return json();
    },
    [LanguageType.Nginx]: async () => {
        const { nginx } = await import("@codemirror/legacy-modes/mode/nginx");
        return StreamLanguage.define(nginx);
    },
    [LanguageType.Nix]: async () => {
        const { nix } = await import("@replit/codemirror-lang-nix");
        return nix();
    },
    [LanguageType.Php]: async () => {
        const { php } = await import("@codemirror/lang-php");
        return php();
    },
    [LanguageType.Powershell]: async () => {
        const { powerShell } = await import("@codemirror/legacy-modes/mode/powershell");
        return StreamLanguage.define(powerShell);
    },
    [LanguageType.Protobuf]: async () => {
        const { protobuf } = await import("@codemirror/legacy-modes/mode/protobuf");
        return StreamLanguage.define(protobuf);
    },
    [LanguageType.Puppet]: async () => {
        const { puppet } = await import("@codemirror/legacy-modes/mode/puppet");
        return StreamLanguage.define(puppet);
    },
    [LanguageType.Python]: async () => {
        const { python } = await import("@codemirror/lang-python");
        return python();
    },
    [LanguageType.R]: async () => {
        const { r } = await import("codemirror-lang-r");
        return r();
    },
    [LanguageType.Ruby]: async () => {
        const { ruby } = await import("@codemirror/legacy-modes/mode/ruby");
        return StreamLanguage.define(ruby);
    },
    [LanguageType.Rust]: async () => {
        const { rust } = await import("@codemirror/lang-rust");
        return rust();
    },
    [LanguageType.Sass]: async () => {
        const { sass } = await import("@codemirror/lang-sass");
        return sass();
    },
    [LanguageType.Shell]: async () => {
        const { shell } = await import("@codemirror/legacy-modes/mode/shell");
        return StreamLanguage.define(shell);
    },
    [LanguageType.Solidity]: async () => {
        const { solidity } = await import("@replit/codemirror-lang-solidity");
        return solidity;
    },
    [LanguageType.Spreadsheet]: async () => {
        const { spreadsheet } = await import("@codemirror/legacy-modes/mode/spreadsheet");
        return StreamLanguage.define(spreadsheet);
    },
    [LanguageType.Sql]: async () => {
        const { standardSQL } = await import("@codemirror/legacy-modes/mode/sql");
        return StreamLanguage.define(standardSQL);
    },
    [LanguageType.Svelte]: async () => {
        const { svelte } = await import("@replit/codemirror-lang-svelte");
        return svelte();
    },
    [LanguageType.Swift]: async () => {
        const { swift } = await import("@codemirror/legacy-modes/mode/swift");
        return StreamLanguage.define(swift);
    },
    [LanguageType.Typescript]: async () => {
        const { javascript } = await import("@codemirror/lang-javascript");
        return javascript({ jsx: true, typescript: true });
    },
    [LanguageType.Vb]: async () => {
        const { vb } = await import("@codemirror/legacy-modes/mode/vb");
        return StreamLanguage.define(vb);
    },
    [LanguageType.Vbscript]: async () => {
        const { vbScript } = await import("@codemirror/legacy-modes/mode/vbscript");
        return StreamLanguage.define(vbScript);
    },
    [LanguageType.Verilog]: async () => {
        const { verilog } = await import("@codemirror/legacy-modes/mode/verilog");
        return StreamLanguage.define(verilog);
    },
    [LanguageType.Vhdl]: async () => {
        const { vhdl } = await import("@codemirror/legacy-modes/mode/vhdl");
        return StreamLanguage.define(vhdl);
    },
    [LanguageType.Vue]: async () => {
        const { vue } = await import("@codemirror/lang-vue");
        return vue();
    },
    [LanguageType.Xml]: async () => {
        const { xml } = await import("@codemirror/lang-xml");
        return xml();
    },
    [LanguageType.Yacas]: async () => {
        const { yacas } = await import("@codemirror/legacy-modes/mode/yacas");
        return StreamLanguage.define(yacas);
    },
    [LanguageType.Yaml]: async () => {
        const { yaml } = await import("@codemirror/legacy-modes/mode/yaml");
        return StreamLanguage.define(yaml);
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

/**
 * Maps languages to validators, if any.
 */
const validatorMap: { [x in LanguageType]?: (input: string) => boolean } = {
    [LanguageType.Json]: (input) => isJson(input),
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

    // Handle language selection
    const [mode, setMode] = useState<LanguageType>(LanguageType.Json);
    const [languageExtension, setLanguageExtension] = useState(null);
    useEffect(() => {
        if (mode in languageMap) {
            languageMap[mode]().then((language) => {
                console.log("imported language", language);
                setLanguageExtension(language);
            });
        }
    }, [mode]);

    // Find language label and help text
    const [label, help] = useMemo<[LangsKey, LangsKey]>(() => languageDisplayMap[mode] ?? ["Json", "JsonHelp"], [mode]);

    // Handle valid indicator for supported languages
    const [isValid, supportsValidation] = useMemo<[boolean, boolean]>(() => {
        if (mode in validatorMap) {
            return [validatorMap[mode]!(internalValue), true];
        }
        return [true, false];
    }, [mode, internalValue]);

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
                    status={isValid ? Status.Valid : Status.Invalid}
                    messages={isValid ? ["JSON is valid"] : ["JSON is empty or could not be parsed"]}
                    sx={{
                        marginLeft: 1,
                        marginRight: "auto",
                    }}
                />}
                {/* Help button */}
                <HelpButton
                    markdown={t(help, { ns: "langs" })}
                    sxRoot={{ marginRight: 1 }}
                />
            </Box>
            <CodeMirror
                value={internalValue}
                theme={palette.mode === "dark" ? "dark" : "light"}
                extensions={languageExtension ? [languageExtension] : []}
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
