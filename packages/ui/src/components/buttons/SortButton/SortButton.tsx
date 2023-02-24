import { Box, Tooltip, Typography, useTheme } from "@mui/material";
import { SortIcon } from "@shared/icons";
import { SortMenu } from "components/lists";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { getUserLanguages, labelledSortOptions } from "utils";
import { searchButtonStyle } from "../styles";
import { SortButtonProps } from "../types";

export const SortButton = ({
    options,
    setSortBy,
    session,
    sortBy,
}: SortButtonProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const lng = useMemo(() => getUserLanguages(session)[0], [session]);

    const [sortAnchorEl, setSortAnchorEl] = useState<HTMLElement | null>(null);

    const handleSortOpen = (event) => setSortAnchorEl(event.currentTarget);
    const handleSortClose = (label?: string, selected?: string) => {
        setSortAnchorEl(null);
        if (selected) setSortBy(selected);
    };

    /**
     * Wrap options with labels
     */
    const sortOptionsLabelled = useMemo(() => labelledSortOptions(options, lng), [options, lng]);

    /**
     * Find sort by label when sortBy changes
     */
    const sortByLabel = useMemo(() => t(`common:${sortBy}`, { lng }) ?? sortBy, [lng, sortBy, t]);

    return (
        <>
            <SortMenu
                sortOptions={sortOptionsLabelled}
                anchorEl={sortAnchorEl}
                lng={getUserLanguages(session)[0]}
                onClose={handleSortClose}
            />
            <Tooltip title={t(`common:SortBy`, { lng })} placement="top">
                <Box
                    onClick={handleSortOpen}
                    sx={searchButtonStyle(palette)}
                >
                    <SortIcon fill={palette.secondary.main} />
                    <Typography variant="body2" sx={{ marginLeft: 0.5 }}>{sortByLabel}</Typography>
                </Box>
            </Tooltip>
        </>
    )
}