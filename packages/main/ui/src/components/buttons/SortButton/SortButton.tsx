import { SortIcon } from "@local/icons";
import { CommonKey } from "@local/translations";
import { Box, Tooltip, Typography, useTheme } from "@mui/material";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { labelledSortOptions } from "../../../utils/display/sorting";
import { SortMenu } from "../../lists/SortMenu/SortMenu";
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
            <Tooltip title={t("SortBy")} placement="top">
                <Box
                    onClick={handleSortOpen}
                    sx={searchButtonStyle(palette)}
                >
                    <SortIcon fill={palette.secondary.main} />
                    <Typography variant="body2" sx={{ marginLeft: 0.5 }}>{sortByLabel}</Typography>
                </Box>
            </Tooltip>
        </>
    );
};