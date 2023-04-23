import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { Box, Card, CardContent, Grid, Typography, useTheme } from "@mui/material";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useCustomLazyQuery } from "../../api";
import { statsSiteFindMany } from "../../api/generated/endpoints/statsSite_findMany";
import { ContentCollapse } from "../../components/containers/ContentCollapse/ContentCollapse";
import { CardGrid } from "../../components/lists/CardGrid/CardGrid";
import { DateRangeMenu } from "../../components/lists/DateRangeMenu/DateRangeMenu";
import { LineGraphCard } from "../../components/lists/LineGraphCard/LineGraphCard";
import { TopBar } from "../../components/navigation/TopBar/TopBar";
import { PageTabs } from "../../components/PageTabs/PageTabs";
import { statsDisplay } from "../../utils/display/statsDisplay";
import { displayDate } from "../../utils/display/stringTools";
const TabOptions = ["Daily", "Weekly", "Monthly", "Yearly", "AllTime"];
const tabPeriods = {
    Daily: 24 * 60 * 60 * 1000,
    Weekly: 7 * 24 * 60 * 60 * 1000,
    Monthly: 30 * 24 * 60 * 60 * 1000,
    Yearly: 365 * 24 * 60 * 60 * 1000,
    AllTime: Number.MAX_SAFE_INTEGER,
};
const tabPeriodTypes = {
    Daily: "Hourly",
    Weekly: "Daily",
    Monthly: "Weekly",
    Yearly: "Monthly",
    AllTime: "Yearly",
};
const MIN_DATE = new Date(2023, 1, 1);
export const StatsView = ({ display = "page", }) => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [period, setPeriod] = useState({
        after: new Date(Date.now() - 24 * 60 * 60 * 1000),
        before: new Date(),
    });
    const [dateRangeAnchorEl, setCustomRangeAnchorEl] = useState(null);
    const handleDateRangeOpen = (event) => setCustomRangeAnchorEl(event.currentTarget);
    const handleDateRangeClose = () => {
        setCustomRangeAnchorEl(null);
    };
    const handleDateRangeSubmit = useCallback((newAfter, newBefore) => {
        setPeriod({
            after: newAfter || period.after,
            before: newBefore || period.before,
        });
        handleDateRangeClose();
    }, [period.after, period.before]);
    const tabs = useMemo(() => {
        const tabs = TabOptions;
        return tabs.map((tab, i) => ({
            index: i,
            label: t(tab, { count: 2 }),
            value: tab,
        }));
    }, [t]);
    const [currTab, setCurrTab] = useState(tabs[0]);
    const handleTabChange = useCallback((e, tab) => {
        setCurrTab(tab);
        const period = tabPeriods[tab.value];
        const newAfter = new Date(Math.max(Date.now() - period, MIN_DATE.getTime()));
        const newBefore = new Date(Math.min(Date.now(), newAfter.getTime() + period));
        setPeriod({ after: newAfter, before: newBefore });
    }, []);
    const [getStats, { data: statsData, loading }] = useCustomLazyQuery(statsSiteFindMany, {
        variables: ({
            periodType: tabPeriodTypes[currTab.value],
            periodTimeFrame: {
                after: period.after.toISOString(),
                before: period.before.toISOString(),
            },
        }),
        errorPolicy: "all",
    });
    const [stats, setStats] = useState([]);
    useEffect(() => {
        if (statsData) {
            setStats(statsData.edges.map(edge => edge.node));
        }
    }, [statsData]);
    useEffect(() => {
        getStats();
    }, [currTab, period, getStats]);
    const { aggregate, visual } = useMemo(() => statsDisplay(stats), [stats]);
    const cards = useMemo(() => (Object.entries(visual).map(([field, data], index) => {
        if (data.length === 0)
            return null;
        const fieldName = field.charAt(0).toUpperCase() + field.slice(1);
        const title = t(fieldName, { count: 2, ns: "common", defaultValue: field });
        return (_jsx(Box, { sx: {
                margin: 2,
                display: "flex",
                justifyContent: "center",
                alignItems: "center",
            }, children: _jsx(LineGraphCard, { data: data, index: index, lineColor: 'white', title: title }) }, index));
    })), [t, visual]);
    return (_jsxs(_Fragment, { children: [_jsx(TopBar, { display: display, onClose: () => { }, titleData: {
                    titleKey: "StatisticsShort",
                }, below: _jsx(PageTabs, { ariaLabel: "stats-period-tabs", currTab: currTab, onChange: handleTabChange, tabs: tabs }) }), _jsx(DateRangeMenu, { anchorEl: dateRangeAnchorEl, minDate: MIN_DATE, maxDate: new Date(), onClose: handleDateRangeClose, onSubmit: handleDateRangeSubmit, range: period, strictIntervalRange: tabPeriods[currTab.value] }), _jsx(Typography, { component: "h3", variant: "body1", textAlign: "center", onClick: handleDateRangeOpen, sx: { cursor: "pointer" }, children: displayDate(period.after.getTime(), false) + " - " + displayDate(period.before.getTime(), false) }), _jsx(ContentCollapse, { isOpen: true, titleKey: "Overview", sxs: {
                    root: {
                        marginBottom: 4,
                    },
                }, children: _jsx(Grid, { container: true, spacing: 2, children: Object.entries(aggregate).map(([field, value], index) => {
                        const fieldName = field.charAt(0).toUpperCase() + field.slice(1);
                        const title = t(fieldName, { count: 2, ns: "common", defaultValue: field });
                        return (_jsx(Grid, { item: true, xs: 12, sm: 6, md: 4, lg: 3, children: _jsx(Card, { sx: {
                                    background: palette.primary.light,
                                    color: palette.primary.contrastText,
                                    height: "100%",
                                }, children: _jsxs(CardContent, { children: [_jsx(Typography, { variant: "h6", textAlign: "center", gutterBottom: true, children: title }), _jsx(Typography, { variant: "body1", textAlign: "center", children: value })] }) }) }, index));
                    }) }) }), _jsx(ContentCollapse, { isOpen: true, titleKey: "Visual", children: _jsx(CardGrid, { minWidth: 275, children: cards }) })] }));
};
//# sourceMappingURL=StatsView.js.map