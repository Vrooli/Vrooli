import { CommonKey } from "@local/shared";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { addSearchParams, parseSearchParams, useLocation } from "route";
import { SvgComponent } from "types";
import { SearchType } from "utils/search/objectToSearch";

type TabParams<T> = {
    Icon?: SvgComponent,
    titleKey: CommonKey;
    searchType: SearchType;
    tabType: T;
    where: { [x: string]: any };
}

type UseTabsParams<T, U extends Record<string, any> = object> = TabParams<T> & U;

/**
 * Contains logic for displaying tabs and handling tab changes.
 */
export const useTabs = <T, U extends Record<string, any> = object>(tabParams: readonly UseTabsParams<T, U>[], defaultTab: number) => {
    const [, setLocation] = useLocation();
    const { t } = useTranslation();

    const tabs = useMemo(() => {
        return tabParams.map((tab, i) => ({
            index: i,
            Icon: tab.Icon,
            label: t(tab.titleKey, { count: 2, defaultValue: tab.titleKey }),
            value: tab.tabType,
        }));
    }, [t, tabParams]);

    const [currTab, setCurrTab] = useState(() => {
        const searchParams = parseSearchParams();
        const index = tabParams.findIndex(tab => tab.tabType === searchParams.type);

        if (index === -1) return tabs[defaultTab];
        return tabs[index];
    });

    const handleTabChange = useCallback((e, tab) => {
        e.preventDefault();
        addSearchParams(setLocation, { type: tab.value });
        setCurrTab(tab);
    }, [setLocation]);

    const { searchType, title, where, ...rest } = useMemo(() => {
        return {
            ...tabParams[currTab.index],
            title: currTab.label,
        };
    }, [currTab.index, currTab.label, tabParams]);

    return { searchType, title, where, tabs, currTab, handleTabChange, ...rest };
};
