import { CommonKey } from "@local/shared";
import { Palette, useTheme } from "@mui/material";
import { ChangeEvent, useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { addSearchParams, parseSearchParams, useLocation } from "route";
import { SvgComponent } from "types";
import { SearchType } from "utils/search/objectToSearch";
import { ViewDisplayType } from "views/types";

export type TabParam<T, S extends boolean = true> = {
    color?: (palette: Palette) => (string | { active: string, inactive: string })
    href?: string;
    Icon?: SvgComponent,
    titleKey: CommonKey;
    tabType: T;
} & (S extends true ? {
    searchPlaceholderKey?: CommonKey;
    searchType: SearchType;
    where: (params?: any) => { [x: string]: any };
} : object);

export type PageTab<T, S extends boolean = true> = {
    color?: string | { active: string, inactive: string };
    href?: string;
    index: number;
    Icon?: SvgComponent;
    label: string;
    tabType: T;
} & (S extends true ? {
    searchPlaceholder: string;
    searchType: SearchType;
} : object);

type UseTabsParam<T, S extends boolean = true> = TabParam<T, S>;

/**
 * Contains logic for displaying tabs and handling tab changes.
 */
export const useTabs = <T, S extends boolean = true>({
    defaultTab = 0,
    display,
    tabParams,
}: {
    defaultTab?: number,
    display: ViewDisplayType,
    tabParams: readonly UseTabsParam<T, S>[],
}) => {
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const { palette } = useTheme();

    const tabs = useMemo<PageTab<T, S>[]>(() => {
        return tabParams.map((tab, i) => ({
            color: typeof tab.color === "function" ? tab.color(palette) : tab.color,
            href: tab.href,
            Icon: tab.Icon,
            index: i,
            label: t(tab.titleKey, { count: 2, defaultValue: tab.titleKey }),
            searchPlaceholder: t((tab as UseTabsParam<T, true>).searchPlaceholderKey ?? "Search"),
            searchType: (tab as UseTabsParam<T, true>).searchType,
            tabType: tab.tabType,
        }));
    }, [palette, t, tabParams]);

    const [currTab, setCurrTab] = useState<PageTab<T, S>>(() => {
        // If this is not for a page, we can't use URL params
        if (display !== "page") return tabs[defaultTab];
        const searchParams = parseSearchParams();
        const index = tabs.findIndex(tab => tab.tabType === searchParams.type);
        if (index === -1) return tabs[defaultTab];
        return tabs[index];
    });

    const handleTabChange = useCallback((e: ChangeEvent<unknown>, tab: PageTab<T, S>) => {
        e.preventDefault();
        if (display === "page") addSearchParams(setLocation, { type: tab.tabType });
        setCurrTab(tab);
    }, [display, setLocation]);

    const currTabParams = useMemo(() => tabParams[currTab.index], [currTab.index, tabParams]);

    return { tabs, currTab, setCurrTab, handleTabChange, ...currTabParams };
};
