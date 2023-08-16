import { CommonKey } from "@local/shared";
import { Palette, useTheme } from "@mui/material";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { addSearchParams, parseSearchParams, useLocation } from "route";
import { SvgComponent } from "types";
import { SearchType } from "utils/search/objectToSearch";
import { ViewDisplayType } from "views/types";

export type TabParam<T> = {
    color?: (palette: Palette) => string;
    href?: string;
    Icon?: SvgComponent,
    titleKey: CommonKey;
    searchType: SearchType;
    tabType: T;
    where: (params?: any) => { [x: string]: any };
}

export type PageTab<T> = {
    color?: string
    href?: string;
    index: number;
    Icon?: SvgComponent;
    label: string;
    searchType: SearchType;
    tabType: T;
};

type UseTabsParam<T, U extends Record<string, any> = object> = TabParam<T> & U;

/**
 * Contains logic for displaying tabs and handling tab changes.
 */
export const useTabs = <T, U extends Record<string, any> = object>({
    defaultTab = 0,
    display,
    tabParams,
}: {
    defaultTab?: number,
    display: ViewDisplayType,
    tabParams: readonly UseTabsParam<T, U>[],
}) => {
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const { palette } = useTheme();

    const tabs = useMemo<PageTab<T>[]>(() => {
        return tabParams.map((tab, i) => ({
            color: typeof tab.color === "function" ? tab.color(palette) : tab.color,
            href: tab.href,
            Icon: tab.Icon,
            index: i,
            label: t(tab.titleKey, { count: 2, defaultValue: tab.titleKey }),
            searchType: tab.searchType,
            tabType: tab.tabType,
        }));
    }, [palette, t, tabParams]);

    const [currTab, setCurrTab] = useState(() => {
        // If this is not for a page, we can't use URL params
        if (display !== "page") return tabs[defaultTab];
        const searchParams = parseSearchParams();
        const index = tabs.findIndex(tab => tab.tabType === searchParams.type);
        if (index === -1) return tabs[defaultTab];
        return tabs[index];
    });

    const handleTabChange = useCallback((e, tab) => {
        e.preventDefault();
        if (display === "page") addSearchParams(setLocation, { type: tab.value });
        setCurrTab(tab);
    }, [display, setLocation]);

    const { searchType, title, where, ...rest } = useMemo(() => {
        return {
            ...tabParams[currTab.index],
            title: currTab.label,
        };
    }, [currTab.index, currTab.label, tabParams]);

    return { searchType, title, where, tabs, currTab, handleTabChange, ...rest };
};
