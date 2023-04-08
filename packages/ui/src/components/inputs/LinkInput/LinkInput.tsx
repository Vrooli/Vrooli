import { Box, Stack, TextField, Tooltip, Typography, useTheme } from "@mui/material";
import { SearchIcon } from "@shared/icons";
import { ColorIconButton } from "components/buttons/ColorIconButton/ColorIconButton";
import { FindObjectDialog } from "components/dialogs/FindObjectDialog/FindObjectDialog";
import { Field, useField } from "formik";
import { useCallback, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { getDisplay } from "utils/display/listTools";
import { LinkInputProps } from "../types";

export const LinkInput = ({
    label,
    name = 'link',
    zIndex,
}: LinkInputProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();

    const textFieldRef = useRef<HTMLDivElement | null>(null);
    const [field, , helpers] = useField<string>(name);

    // Search dialog to find objects. to link to
    const [searchOpen, setSearchOpen] = useState(false);
    const openSearch = useCallback(() => { setSearchOpen(true) }, []);
    const closeSearch = useCallback((selectedUrl?: string) => {
        setSearchOpen(false);
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
            } catch (e) { }
            if (displayData) {
                let { title, subtitle } = getDisplay(displayData);
                // Limit subtitle to 100 characters
                if (subtitle && subtitle.length > 100) {
                    subtitle = subtitle.substring(0, 100) + '...';
                }
                return { title, subtitle };
            }
        }
        return {};
    }, [field.value]);

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
                        label={label ?? t('Link')}
                        as={TextField}
                        ref={textFieldRef}
                        sx={{
                            '& .MuiInputBase-root': {
                                borderRadius: '5px 0 0 5px',
                            }
                        }}
                    />
                    <ColorIconButton
                        aria-label='find URL'
                        onClick={openSearch}
                        background={palette.secondary.main}
                        sx={{
                            borderRadius: '0 5px 5px 0',
                            height: `${textFieldRef.current?.clientHeight ?? 56}px)`,
                        }}>
                        <SearchIcon />
                    </ColorIconButton>
                </Stack>
                {/* Title/Subtitle */}
                {title && (
                    <Tooltip title={subtitle}>
                        <Typography variant='body2' ml={1}>{title}{subtitle ? ' - ' + subtitle : ''}</Typography>
                    </Tooltip>
                )}
            </Box>
        </>

    )
}