import { SearchIcon } from "@local/shared";
import { Box, Stack, TextField, Tooltip, useTheme } from "@mui/material";
import { ColorIconButton } from "components/buttons/ColorIconButton/ColorIconButton";
import { FindObjectDialog } from "components/dialogs/FindObjectDialog/FindObjectDialog";
import { MarkdownDisplay } from "components/text/MarkdownDisplay/MarkdownDisplay";
import { Field, useField } from "formik";
import { useCallback, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { getDisplay } from "utils/display/listTools";
import { LinkInputProps } from "../types";

export const LinkInput = ({
    label,
    name = "link",
    onObjectData,
    zIndex,
}: LinkInputProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();

    const textFieldRef = useRef<HTMLDivElement | null>(null);
    const [field, , helpers] = useField<string>(name);

    // Search dialog to find objects. to link to
    const hasSelectedObject = useRef(false);
    const [searchOpen, setSearchOpen] = useState(false);
    const openSearch = useCallback(() => { setSearchOpen(true); }, []);
    const closeSearch = useCallback((selectedUrl?: string) => {
        setSearchOpen(false);
        hasSelectedObject.current = !!selectedUrl;
        if (selectedUrl) {
            helpers.setValue(selectedUrl);
        }
    }, [helpers]);

    // If there is a link for this website and we have display 
    // data stored for it, display it below the link
    const { title, subtitle } = useMemo(() => {
        if (!field.value) return {};
        // Check if the link is for this website
        if (field.value.startsWith(window.location.origin)) {
            // Check local storage for display data
            const displayDataJson = localStorage.getItem(`objectFromUrl:${field.value}`);
            let displayData;
            try {
                displayData = displayDataJson ? JSON.parse(displayDataJson) : null;
            } catch (e) { /* empty */ }
            if (displayData) {
                let { title, subtitle } = getDisplay(displayData);
                // Pass data to parent, in case we want to use it.
                // Only do this if a new URL was selected, not for the initial value
                if (hasSelectedObject.current) {
                    onObjectData?.({ title, subtitle });
                    hasSelectedObject.current = false;
                }
                // Limit subtitle to 100 characters
                if (subtitle && subtitle.length > 100) {
                    subtitle = subtitle.substring(0, 100) + "...";
                }
                return { title, subtitle };
            }
        }
        return {};
    }, [field.value, onObjectData]);

    return (
        <>
            {/* Search dialog */}
            <FindObjectDialog
                find="Url"
                isOpen={searchOpen}
                handleCancel={closeSearch}
                handleComplete={closeSearch}
                zIndex={zIndex + 1}
            />
            <Box>
                {/* Text field with button to open search dialog */}
                <Stack direction="row" spacing={0}>
                    <Field
                        fullWidth
                        name={name}
                        label={label ?? t("Link")}
                        as={TextField}
                        ref={textFieldRef}
                        sx={{
                            "& .MuiInputBase-root": {
                                borderRadius: "5px 0 0 5px",
                            },
                        }}
                    />
                    <ColorIconButton
                        aria-label='find URL'
                        onClick={openSearch}
                        background={palette.secondary.main}
                        sx={{
                            borderRadius: "0 5px 5px 0",
                            height: `${textFieldRef.current?.clientHeight ?? 56}px)`,
                        }}>
                        <SearchIcon />
                    </ColorIconButton>
                </Stack>
                {/* Title/Subtitle */}
                {title && (
                    <Tooltip title={subtitle}>
                        <MarkdownDisplay sx={{ marginLeft: "8px" }} content={`${title}${subtitle ? " - " + subtitle : ""}`} />
                    </Tooltip>
                )}
            </Box>
        </>

    );
};
