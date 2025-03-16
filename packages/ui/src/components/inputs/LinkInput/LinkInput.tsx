import { Box, IconButton, InputAdornment, Stack, TextField, Tooltip, useTheme } from "@mui/material";
import { FindObjectDialog } from "components/dialogs/FindObjectDialog/FindObjectDialog.js";
import { MarkdownDisplay } from "components/text/MarkdownDisplay.js";
import { useField } from "formik";
import { LinkIcon, SearchIcon } from "icons/common.js";
import { useCallback, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { getDisplay } from "utils/display/listTools";
import { LinkInputBaseProps, LinkInputProps } from "../types.js";

const MAX_LINK_TITLE_CHARS = 100;

const linkInputProps = {
    startAdornment: (
        <InputAdornment position="start">
            <LinkIcon />
        </InputAdornment>
    ),
} as const;

export function LinkInputBase({
    autoFocus,
    disabled,
    error,
    fullWidth,
    helperText,
    label,
    limitTo,
    name = "link",
    onBlur,
    onChange,
    onObjectData,
    placeholder,
    sxs,
    tabIndex,
    value,
}: LinkInputBaseProps) {
    const { palette } = useTheme();
    const { t } = useTranslation();

    // Search dialog to find objects to link to
    const hasSelectedObject = useRef(false);
    const [searchOpen, setSearchOpen] = useState(false);
    const openSearch = useCallback(() => { setSearchOpen(true); }, []);
    const closeSearch = useCallback((selectedUrl?: string) => {
        setSearchOpen(false);
        hasSelectedObject.current = !!selectedUrl;
        if (selectedUrl) {
            onChange(selectedUrl);
        }
    }, [onChange]);

    // If there is a link for this website and we have display 
    // data stored for it, display it below the link
    const { title, subtitle } = useMemo(() => {
        if (!value || typeof value !== "string") return {};
        // Check if the link is for this website
        if (value.startsWith(window.location.origin)) {
            // Check local storage for display data
            const displayDataJson = localStorage.getItem(`objectFromUrl:${value}`);
            let displayData;
            try {
                displayData = displayDataJson ? JSON.parse(displayDataJson) : null;
            } catch (e) { /* empty */ }
            if (displayData) {
                const { title, subtitle } = getDisplay(displayData);
                // Pass data to parent, in case we want to use it.
                // Only do this if a new URL was selected, not for the initial value
                if (hasSelectedObject.current) {
                    onObjectData?.({ title, subtitle });
                    hasSelectedObject.current = false;
                }
                return {
                    title,
                    // Limit subtitle to 100 characters
                    subtitle: subtitle && subtitle.length > MAX_LINK_TITLE_CHARS ? subtitle.substring(0, MAX_LINK_TITLE_CHARS) + "..." : subtitle,
                };
            }
        }
        return {};
    }, [value, onObjectData]);

    return (
        <>
            {/* Search dialog */}
            <FindObjectDialog
                find="Url"
                isOpen={searchOpen}
                limitTo={limitTo}
                handleCancel={closeSearch}
                handleComplete={closeSearch}
            />
            <Box sx={sxs?.root}>
                {/* Text field with button to open search dialog */}
                <Stack direction="row" spacing={0}>
                    <TextField
                        autoFocus={autoFocus}
                        disabled={disabled}
                        error={error}
                        fullWidth={fullWidth}
                        helperText={helperText}
                        label={label ?? t("Link", { count: 1 })}
                        name={name}
                        placeholder={placeholder ?? "https://example.com"}
                        onBlur={onBlur}
                        onChange={(e) => onChange(e.target.value)}
                        tabIndex={tabIndex}
                        value={value}
                        InputProps={linkInputProps}
                        sx={{
                            "& .MuiInputBase-root": {
                                borderRadius: "5px 0 0 5px",
                            },
                        }}
                    />
                    <IconButton
                        aria-label={t("SearchObjectLink")}
                        disabled={disabled}
                        onClick={openSearch}
                        sx={{
                            background: palette.secondary.main,
                            borderRadius: "0 5px 5px 0",
                            color: palette.secondary.contrastText,
                        }}>
                        <SearchIcon />
                    </IconButton>
                </Stack>
                {title && (
                    <Tooltip title={subtitle}>
                        <MarkdownDisplay
                            sx={{ marginLeft: "8px", paddingTop: "8px" }}
                            content={`${title}${subtitle ? " - " + subtitle : ""}`}
                        />
                    </Tooltip>
                )}
                {helperText && (
                    <Box sx={{ paddingTop: "8px", color: palette.error.main, fontSize: "0.75rem" }}>
                        {typeof helperText === "string" ? helperText : JSON.stringify(helperText)}
                    </Box>
                )}
            </Box>
        </>
    );
}

export function LinkInput({
    name,
    ...props
}: LinkInputProps) {
    const [field, meta, helpers] = useField<string>(name);

    function handleChange(value) {
        helpers.setValue(value);
    }

    return (
        <LinkInputBase
            {...props}
            name={name}
            value={field.value}
            error={meta.touched && !!meta.error}
            helperText={meta.touched && meta.error}
            onBlur={field.onBlur}
            onChange={handleChange}
        />
    );
}
