import { Box, Tooltip, Typography, useTheme } from "@mui/material";
import { SortIcon } from "@shared/icons";
import { CommonKey } from "@shared/translations";
import { SortMenu } from "components/lists";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { labelledSortOptions } from "utils";
import { searchButtonStyle } from "../styles";
import { SortButtonProps } from "../types";

export const SortButton = ({
    options,
    setSortBy,
    sortBy,
}: SortButtonProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();

    const [sortAnchorEl, setSortAnchorEl] = useState<HTMLElement | null>(null);

    const handleSortOpen = (event: any) => setSortAnchorEl(event.currentTarget);
    const handleSortClose = (label?: string, selected?: string) => {
        setSortAnchorEl(null);
        if (selected) setSortBy(selected);
    };

    /**
     * Wrap options with labels
     */
    const sortOptionsLabelled = useMemo(() => labelledSortOptions(options), [options]);

    /**
     * Find sort by label when sortBy changes
     */
    const sortByLabel = useMemo(() => t(sortBy as CommonKey, sortBy), [sortBy, t]);

    return (
        <>
            <SortMenu
                sortOptions={sortOptionsLabelled}
                anchorEl={sortAnchorEl}
                onClose={handleSortClose}
            />
            <Tooltip title={t(`SortBy`)} placement="top">
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