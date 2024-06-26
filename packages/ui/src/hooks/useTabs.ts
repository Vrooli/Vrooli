import { CommonKey, parseSearchParams } from "@local/shared";
import { useTheme } from "@mui/material";
import { ChangeEvent, useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { addSearchParams, useLocation } from "route";
import { getCookie, setCookie } from "utils/cookies";
import { TabParam, TabsInfo } from "utils/search/objectToSearch";
import { ViewDisplayType } from "views/types";

export type PageTab<TabList extends TabsInfo> = Omit<TabParam<TabList>, "color" | "searchPlaceholderKey" | "titleKey"> & {
    color: string,
    index: number,
    label: string,
    searchPlaceholder: string,
};

/**
 * Contains logic for displaying tabs and handling tab changes.
 */
export const useTabs = <TabList extends TabsInfo>({
    defaultTab,
    display,
    id,
    tabParams,
}: {
    defaultTab?: TabList["Key"] | `${TabList["Key"]}`,
    display: ViewDisplayType,
    id: string,
    tabParams: readonly TabParam<TabList>[],
}) => {
    const [location, setLocation] = useLocation();
    const { t } = useTranslation();
    const { palette } = useTheme();

    const tabs = useMemo(() => {
        return tabParams.map((tab, i) => ({
            ...tab,
            color: typeof tab.color === "function" ? tab.color(palette) : tab.color,
            index: i,
            label: t(tab.titleKey, { count: 2, defaultValue: tab.titleKey }),
            searchPlaceholder: t((tab as { searchPlaceholderKey?: CommonKey }).searchPlaceholderKey ?? "Search"),
        })) as PageTab<TabList>[];
    }, [palette, t, tabParams]);

    const [currTab, setCurrTab] = useState<PageTab<TabList>>(() => {
        const storedKey = getCookie("LastTab", id);
        if (display !== "page") {
            const defaultIndex = tabs.findIndex(tab => tab.key === (storedKey || defaultTab));
            return tabs[defaultIndex !== -1 ? defaultIndex : 0];
        }
        const searchParams = parseSearchParams();
        const tabFromParams = tabs.find(tab => tab.key === searchParams.type);
        return tabFromParams || tabs.find(tab => tab.key === storedKey || defaultTab) || tabs[0];
    });

    useEffect(() => {
        if (display === "page") {
            const searchParams = parseSearchParams();
            const tabFromParams = tabs.find(tab => tab.key === searchParams.type);
            if (tabFromParams) {
                setCurrTab(tabFromParams);
            }
        }
    }, [location, display, tabs]);

    const handleTabChange = useCallback((e: ChangeEvent<unknown> | undefined, tab: PageTab<TabList>) => {
        e?.preventDefault();
        if (display === "page") addSearchParams(setLocation, { type: tab.key });
        setCookie("LastTab", tab.key, id);
        setCurrTab(tab);
    }, [display, setLocation, id]);

    const changeTab = useCallback((key: TabList["Key"]) => {
        const tab = tabs.find(tab => tab.key === key);
        if (tab) handleTabChange(undefined, tab);
    }, [handleTabChange, tabs]);

    const currTabParams = useMemo(() => tabParams[currTab.index] as TabParam<TabList>, [currTab, tabParams]);

    return { tabs, currTab, setCurrTab, handleTabChange, changeTab, ...currTabParams };
};
