import { CommonKey } from "@local/shared";
import { Box, Menu, MenuItem, Tooltip, Typography, useTheme } from "@mui/material";
import i18next from "i18next";
import { SortIcon } from "icons";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { searchButtonStyle } from "../styles";
import { SortButtonProps } from "../types";

export type LabelledSortOption<SortBy> = { label: string, value: SortBy };

const SortMenu = ({
    sortOptions,
    anchorEl,
    onClose,
}: {
    sortOptions: LabelledSortOption<string>[];
    anchorEl: HTMLElement | null;
    onClose: (label?: string, value?: string) => void;
}) => {
    const { t } = useTranslation();
    const open = Boolean(anchorEl);

    const menuItems = useMemo(() => {
        const menuItems: JSX.Element[] = [];
        sortOptions.forEach(option => {
            const optionLabel = t(`${option.value}` as CommonKey);
            if (optionLabel) {
                menuItems.push(
                    <MenuItem
                        key={option.value}
                        value={option.value}
                        onClick={() => onClose(optionLabel, option.value)}
                    >
                        {optionLabel}
                    </MenuItem>,
                );
            }
        });
        return menuItems;
    }, [sortOptions, t, onClose]);

    return (
        <Menu
            id="sort-results-menu"
            disableScrollLock={true}
            anchorEl={anchorEl}
            open={open}
            onClose={() => onClose()}
            MenuListProps={{ "aria-labelledby": "sort-results-menu-list" }}
        >
            {menuItems}
        </Menu>
    );
};

export const SortButton = ({
    options,
    setSortBy,
    sortBy,
}: SortButtonProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();

    const [sortAnchorEl, setSortAnchorEl] = useState<HTMLElement | null>(null);

    const handleSortOpen = (event: { currentTarget: HTMLElement }) => setSortAnchorEl(event.currentTarget);
    const handleSortClose = (_label?: string, selected?: string) => {
        setSortAnchorEl(null);
        if (selected) setSortBy(selected);
    };

    /** Wrap options with labels */
    const sortOptionsLabelled = useMemo<LabelledSortOption<string>[]>(() => {
        if (!options) return [];
        return Object.keys(options).map((key) => ({
            label: (i18next.t(key as CommonKey, key)) as unknown as string,
            value: key,
        }));
    }, [options]);

    /** Find sort by label when sortBy changes */
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
