/**
 * Input for creating a JSON schema.
 */
import { InvisibleIcon, VisibleIcon } from "@local/shared";
import { Box, IconButton, Stack, TextField, Tooltip, Typography, useTheme } from "@mui/material";
import { HelpButton } from "components/buttons/HelpButton/HelpButton";
import { StatusButton } from "components/buttons/StatusButton/StatusButton";
import { useField } from "formik";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Status } from "utils/consts";
import { isJson, jsonToMarkdown } from "utils/shape/general";
import { MarkdownDisplay } from "../../../../../../packages/ui/src/components/text/MarkdownDisplay/MarkdownDisplay";
import { JsonInputProps } from "../types";

export const JsonInput = ({
    disabled = false,
    format,
    index,
    minRows = 4,
    name,
    placeholder = "",
    variables,
}: JsonInputProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();

    const [field, meta, helpers] = useField<string | null>(name);

    /**
     * If value not set, defaults to format with variables 
     * substituted for their known defaults.
     */
    useEffect(() => {
        if (disabled) return;
        if ((field.value ?? "").length > 0 || !format) return;
        // Initialize with stringified format
        const changedFormat: string = JSON.stringify(format);
        // If variables not set, return stringified format
        if (!variables) {
            helpers.setValue(changedFormat);
            return;
        }
        // Loop through variables and replace all instances of them in format
        // with their default values.
        for (const [variableName, variableData] of Object.entries(variables)) {
            // If variable has no default value, skip it
            if (variableData.defaultValue === undefined) continue;
            // Find locations of variables in format
            // TODO
        }
    }, [disabled, format, field.value, helpers, variables]);

    /**
     * Uses format, variables, and value
     */
    useEffect(() => {
        //TODO
    }, [format, variables]);

    const [isPreviewOn, setIsPreviewOn] = useState<boolean>(false);
    const togglePreview = () => setIsPreviewOn(!isPreviewOn);
    const { previewMarkdown, isValueValid } = useMemo(() => ({
        previewMarkdown: jsonToMarkdown(field.value),
        isValueValid: isJson(field.value),
    }), [field.value]);

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
                    messages={isValueValid ? ["JSON is valid"] : ["JSON could not be parsed"]}
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
                    markdown={t("JsonHelp")}
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
                                <p>{`Error: Invalid JSON - ${field.value}`}</p>
                        }
                    </Box> :
                    <TextField
                        id={`markdown-input-${name}`}
                        autoFocus={index === 0}
                        disabled={disabled}
                        error={meta.touched && !!meta.error}
                        name={name}
                        placeholder={placeholder}
                        rows={minRows}
                        value={field.value ?? ""}
                        onBlur={field.onBlur}
                        onChange={field.onChange}
                        tabIndex={index}
                        style={{
                            minWidth: "-webkit-fill-available",
                            maxWidth: "-webkit-fill-available",
                            minHeight: "50px",
                            maxHeight: "800px",
                            background: "transparent",
                            borderColor: (meta.touched && !!meta.error) ? "red" : "unset",
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
            {/* Error message */}
            {meta.touched && !!meta.error && <Typography variant="body1" sx={{ color: "red" }}>{meta.error}</Typography>}
        </Stack>
    );
};
