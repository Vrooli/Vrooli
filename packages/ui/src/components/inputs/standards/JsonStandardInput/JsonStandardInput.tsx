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
    Html = "html",
    Java = "java",
    Javascript = "javascript",
    Json = "json",
    Php = "php",
    Python = "python",
    Rust = "rust",
    Sass = "sass",
    Typescript = "typescript",
    Vue = "vue",
    Xml = "xml",
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
    [LanguageType.Php]: async () => {
        const { php } = await import("@codemirror/lang-php");
        return php();
    },
    [LanguageType.Python]: async () => {
        const { python } = await import("@codemirror/lang-python");
        return python();
    },
    [LanguageType.Rust]: async () => {
        const { rust } = await import("@codemirror/lang-rust");
        return rust();
    },
    [LanguageType.Sass]: async () => {
        const { sass } = await import("@codemirror/lang-sass");
        return sass();
    },
    [LanguageType.Typescript]: async () => {
        const { javascript } = await import("@codemirror/lang-javascript");
        return javascript({ jsx: true, typescript: true });
    },
    [LanguageType.Vue]: async () => {
        const { vue } = await import("@codemirror/lang-vue");
        return vue();
    },
    [LanguageType.Xml]: async () => {
        const { xml } = await import("@codemirror/lang-xml");
        return xml();
    },
};

/**
 * Maps languages to their labels and help texts.
 */
const languageDisplayMap: { [x in LanguageType]: [LangsKey, LangsKey] } = {
    [LanguageType.Angular]: ["Angular", "AngularHelp"],
    [LanguageType.Cpp]: ["Cpp", "CppHelp"],
    [LanguageType.Css]: ["Css", "CssHelp"],
    [LanguageType.Html]: ["Html", "HtmlHelp"],
    [LanguageType.Java]: ["Java", "JavaHelp"],
    [LanguageType.Javascript]: ["Javascript", "JavascriptHelp"],
    [LanguageType.Json]: ["Json", "JsonHelp"],
    [LanguageType.Php]: ["Php", "PhpHelp"],
    [LanguageType.Python]: ["Python", "PythonHelp"],
    [LanguageType.Rust]: ["Rust", "RustHelp"],
    [LanguageType.Sass]: ["Sass", "SassHelp"],
    [LanguageType.Typescript]: ["Typescript", "TypescriptHelp"],
    [LanguageType.Vue]: ["Vue", "VueHelp"],
    [LanguageType.Xml]: ["Xml", "XmlHelp"],
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
