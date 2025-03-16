import { TranslationKeyCommon, parseSearchParams } from "@local/shared";
import { useTheme } from "@mui/material";
import { ChangeEvent, useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "../route/router.js";
import { addSearchParams } from "../route/searchParams.js";
import { ViewDisplayType } from "../types.js";
import { getCookie, setCookie } from "../utils/localStorage.js";
import { TabListType } from "../utils/search/objectToSearch.js";

export type PageTab<TabItem extends TabListType[number]> = Omit<TabItem, "color" | "searchPlaceholderKey" | "titleKey"> & {
    color: string,
    index: number,
    label: string,
    searchPlaceholder: string,
};

type UseTabsProps<TabList extends TabListType = TabListType> = {
    defaultTab?: TabList[number]["key"]
    disableHistory?: boolean,
    display: ViewDisplayType,
    id: string,
    tabParams: TabList
}

/**
 * Contains logic for displaying tabs and handling tab changes.
 */
export function useTabs<TabList extends TabListType = TabListType>({
    defaultTab,
    disableHistory,
    display,
    id,
    tabParams,
}: UseTabsProps<TabList>) {
    const [location, setLocation] = useLocation();
    const { t } = useTranslation();
    const { palette } = useTheme();

    const tabs = useMemo(() => {
        return tabParams.map((tab, i) => ({
            ...tab,
            color: typeof tab.color === "function" ? tab.color(palette) : tab.color,
            index: i,
            label: t(tab.titleKey, { count: 2, defaultValue: tab.titleKey }),
            searchPlaceholder: t((tab as { searchPlaceholderKey?: TranslationKeyCommon }).searchPlaceholderKey ?? "Search"),
        })) as PageTab<TabList[number]>[];
    }, [palette, t, tabParams]);

    const [currTab, setCurrTab] = useState<PageTab<TabList[number]>>(() => {
        const storedKey = disableHistory ? undefined : getCookie("LastTab", id);
        if (display !== "page") {
            const defaultIndex = tabs.findIndex(tab => tab.key === (storedKey || defaultTab));
            return tabs[defaultIndex !== -1 ? defaultIndex : 0];
        }
        const searchParams = parseSearchParams();
        const tabFromParams = tabs.find(tab => tab.key === searchParams.type);
        return tabFromParams || tabs.find(tab => tab.key === storedKey || defaultTab) || tabs[0];
    });

    useEffect(function getTabDataFromUrlEffect() {
        if (display === "page") {
            const searchParams = parseSearchParams();
            const tabFromParams = tabs.find(tab => tab.key === searchParams.type);
            if (tabFromParams) {
                setCurrTab(tabFromParams);
            }
        }
    }, [location, display, tabs]);

    const handleTabChange = useCallback(function handleTabChangeCallback(e: ChangeEvent<unknown> | undefined, tab: PageTab<TabList[number]>) {
        e?.preventDefault();
        if (display === "page") addSearchParams(setLocation, { type: tab.key });
        if (!disableHistory) {
            setCookie("LastTab", tab.key, id);
        }
        setCurrTab(tab);
    }, [display, setLocation, disableHistory, id]);

    const changeTab = useCallback(function changeTabCallback(key: TabList[number]["key"]) {
        const tab = tabs.find(tab => tab.key === key);
        if (tab) handleTabChange(undefined, tab);
    }, [handleTabChange, tabs]);

    const currTabParams = useMemo(() => tabParams[currTab.index] as TabList[number], [currTab, tabParams]);

    return { tabs, currTab, setCurrTab, handleTabChange, changeTab, ...currTabParams };
}
