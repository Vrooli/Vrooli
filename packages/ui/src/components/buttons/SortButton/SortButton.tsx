import { TranslationKeyCommon } from "@local/shared";
import { Box, Menu, MenuItem, Tooltip, Typography, useTheme } from "@mui/material";
import i18next from "i18next";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { usePopover } from "../../../hooks/usePopover.js";
import { SortIcon } from "../../../icons/common.js";
import { searchButtonStyle } from "../../../styles.js";
import { SortButtonProps } from "../types.js";

export type LabelledSortOption<SortBy> = { label: string, value: SortBy };

interface SortMenuProps {
    anchorEl: Element | null;
    onClose: (label?: string, value?: string) => unknown;
    sortOptions: LabelledSortOption<string>[];
}

function SortMenu({
    anchorEl,
    onClose,
    sortOptions,
}: SortMenuProps) {
    const { t } = useTranslation();
    const open = Boolean(anchorEl);

    const menuItems = useMemo(() => {
        const menuItems: JSX.Element[] = [];
        sortOptions.forEach(option => {
            const optionLabel = t(`${option.value}` as TranslationKeyCommon);
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
}

export function SortButton({
    options,
    setSortBy,
    sortBy,
}: SortButtonProps) {
    const { palette } = useTheme();
    const { t } = useTranslation();

    const [sortAnchorEl, openSort, closeSort] = usePopover();
    function handleSortClose(_label?: string, selected?: string) {
        closeSort();
        if (selected) setSortBy(selected);
    }

    /** Wrap options with labels */
    const sortOptionsLabelled = useMemo<LabelledSortOption<string>[]>(() => {
        if (!options) return [];
        return Object.keys(options).map((key) => ({
            label: (i18next.t(key as TranslationKeyCommon, key)) as unknown as string,
            value: key,
        }));
    }, [options]);

    /** Find sort by label when sortBy changes */
    const sortByLabel = useMemo(() => t(sortBy as TranslationKeyCommon, sortBy), [sortBy, t]);

    return (
        <>
            <SortMenu
                sortOptions={sortOptionsLabelled}
                anchorEl={sortAnchorEl}
                onClose={handleSortClose}
            />
            <Tooltip title={t("SortBy")} placement="top">
                <Box
                    onClick={openSort}
                    sx={searchButtonStyle(palette)}
                >
                    <SortIcon fill={palette.secondary.main} />
                    <Typography variant="body2" sx={{ marginLeft: 0.5 }}>{sortByLabel}</Typography>
                </Box>
            </Tooltip>
        </>
    );
}
