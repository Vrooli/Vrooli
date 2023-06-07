import { InvisibleIcon, VisibleIcon } from "@local/shared";
import { Box, IconButton, Stack, TextField, Tooltip, Typography, useTheme } from "@mui/material";
import { HelpButton } from "components/buttons/HelpButton/HelpButton";
import { StatusButton } from "components/buttons/StatusButton/StatusButton";
import { useField } from "formik";
import { JsonProps } from "forms/types";
import { useEffect, useMemo, useState } from "react";
import { Status } from "utils/consts";
import { isEqualJSON, isJson, jsonHelpText, jsonToMarkdown, jsonToString } from "utils/shape/general";
import { MarkdownDisplay } from "../../../../../../../packages/ui/src/components/text/MarkdownDisplay/MarkdownDisplay";
import { JsonStandardInputProps } from "../types";

export const JsonStandardInput = ({
    isEditing,
}: JsonStandardInputProps) => {
    const { palette } = useTheme();

    const [defaultValueField, , defaultValueHelpers] = useField<JsonProps["defaultValue"]>("defaultValue");
    const [formatField, formatMeta, formatHelpers] = useField<JsonProps["format"]>("format");
    const [variablesField, , variablesHelpers] = useField<JsonProps["variables"]>("variables");

    // Last valid schema format
    const [internalValue, setInternalValue] = useState<string>(jsonToString(formatField.value ?? {}) ?? "");

    /**
    * Set internal value when formatField.value changes.
    */
    useEffect(() => {
        if (!isEditing) return;
        // Compare to current internal value
        if (isEqualJSON(formatField.value, internalValue)) return;
        setInternalValue(jsonToString(formatField.value ?? {}) ?? "");
    }, [formatField.value, internalValue, isEditing]);

    // // Update formatField only when internalValue is valid TODO
    // useEffect(() => {
    //     if (isJson(internalValue)) {
    //         formatHelpers.setValue(JSON.parse(internalValue));
    //     }
    // }, [formatField.value, formatHelpers, internalValue]);


    const [isPreviewOn, setIsPreviewOn] = useState<boolean>(false);
    const togglePreview = () => setIsPreviewOn(!isPreviewOn);
    const { previewMarkdown, isValueValid } = useMemo(() => ({
        previewMarkdown: jsonToMarkdown(formatField.value ?? {}),
        isValueValid: internalValue.length > 2 && isJson(formatField.value),
    }), [formatField.value, internalValue.length]);

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
                <StatusButton
                    status={isValueValid ? Status.Valid : Status.Invalid}
                    messages={isValueValid ? ["JSON is valid"] : ["JSON is empty or could not be parsed"]}
                    sx={{
                        marginLeft: 1,
                        marginRight: "auto",
                    }}
                />
                {/* Toggle preview */}
                <Tooltip title={isPreviewOn ? "Preview mode" : "Edit mode"} placement="top" sx={{ marginLeft: "auto" }}>
                    <IconButton size="small" onClick={togglePreview}>
                        {
                            isPreviewOn ?
                                <InvisibleIcon fill={palette.primary.contrastText} /> :
                                <VisibleIcon fill={palette.primary.contrastText} />
                        }
                    </IconButton>
                </Tooltip>
                <HelpButton
                    markdown={jsonHelpText}
                    sxRoot={{ marginRight: 1 }}
                />
            </Box>
            {/* Displays inputted JSON to the left, and info about the current variable being edited to the right */}
            {/* TextField for entering markdown, or markdown display if previewing */}
            {
                isPreviewOn ?
                    <Box sx={{
                        border: `1px solid ${palette.background.textPrimary}`,
                        color: isValueValid ? palette.background.textPrimary : palette.error.main,
                    }}>
                        {
                            previewMarkdown ?
                                <MarkdownDisplay content={previewMarkdown} /> :
                                <p>{`Error: Invalid JSON - ${internalValue}`}</p>
                        }
                    </Box> :
                    <TextField
                        name="format"
                        disabled={!isEditing}
                        placeholder={"Enter JSON format. Click the '?' button for help."}
                        multiline
                        value={internalValue}
                        onChange={(e) => setInternalValue(e.target.value)}
                        style={{
                            minWidth: "-webkit-fill-available",
                            maxWidth: "-webkit-fill-available",
                            minHeight: "50px",
                            maxHeight: "800px",
                            background: "transparent",
                            borderColor: formatMeta.error ? "red" : "unset",
                            borderRadius: "0 0 0.5rem 0.5rem",
                            borderTop: "none",
                            fontFamily: "inherit",
                            fontSize: "inherit",
                        }}
                    />
            }
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
