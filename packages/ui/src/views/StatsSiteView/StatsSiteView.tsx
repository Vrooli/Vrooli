import { DAYS_1_MS, endpointsStatsSite, MONTHS_1_MS, StatPeriodType, StatsSite, StatsSiteSearchInput, StatsSiteSearchResult, WEEKS_1_MS, YEARS_1_MS } from "@local/shared";
import { Card, CardContent, Typography, useTheme } from "@mui/material";
import { ContentCollapse } from "components/containers/ContentCollapse/ContentCollapse.js";
import { CardGrid } from "components/lists/CardGrid/CardGrid";
import { DateRangeMenu } from "components/lists/DateRangeMenu/DateRangeMenu";
import { LineGraphCard } from "components/lists/LineGraphCard/LineGraphCard";
import { TopBar } from "components/navigation/TopBar.js";
import { PageTabs } from "components/PageTabs/PageTabs";
import { useLazyFetch } from "hooks/useLazyFetch.js";
import { PageTab, useTabs } from "hooks/useTabs";
import { ChangeEvent, useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { statsDisplay } from "utils/display/statsDisplay";
import { displayDate } from "utils/display/stringTools";
import { TabParamBase } from "utils/search/objectToSearch";
import { StatsSiteViewProps } from "../types.js";

/**
 * Stats page tabs. While stats data is stored using PeriodType 
 * (i.e "Hourly", "Daily", "Weekly", "Monthly", "Yearly"), the 
 * tabs appear one time frame higher (i.e. "Daily", "Weekly", 
 * "Monthly", "Yearly", "All time"). 
 * 
 * This makes sense because each tab display data which has a 
 * smaller time frame than the tab itself. 
 * For example, the "Daily" tab will display the hourly stats data
 * for the previous 24 hours.
 */
enum StatsTabOption {
    Daily = "Daily",
    Weekly = "Weekly",
    Monthly = "Monthly",
    Yearly = "Yearly",
    AllTime = "AllTime",
}

/** Maps tab options to time frame intervals (in milliseconds) */
const tabPeriods: { [key in StatsTabOption]: number } = {
    Daily: DAYS_1_MS,
    Weekly: WEEKS_1_MS,
    Monthly: MONTHS_1_MS,
    Yearly: YEARS_1_MS,
    AllTime: Number.MAX_SAFE_INTEGER,
};

/** Maps tab options to PeriodType */
const tabPeriodTypes: { [key in StatsTabOption]: StatPeriodType | `${StatPeriodType}` } = {
    Daily: "Hourly",
    Weekly: "Daily",
    Monthly: "Weekly",
    Yearly: "Monthly",
    AllTime: "Yearly",
} as const;

type StatsSiteTabsInfo = {
    IsSearchable: false;
    Key: StatsTabOption;
    Payload: undefined;
    WhereParams: undefined;
};

export const statsSiteTabParams: TabParamBase<StatsSiteTabsInfo>[] = [
    {
        key: StatsTabOption.Daily,
        titleKey: "Daily",
    }, {
        key: StatsTabOption.Weekly,
        titleKey: "Weekly",
    },
    {
        key: StatsTabOption.Monthly,
        titleKey: "Monthly",
    },
    {
        key: StatsTabOption.Yearly,
        titleKey: "Yearly",
    },
    {
        key: StatsTabOption.AllTime,
        titleKey: "AllTime",
    },
];

// Stats should not be earlier than February 2023.
// eslint-disable-next-line no-magic-numbers
const MIN_DATE = new Date(2023, 1, 1);

/**
 * Displays site-wide statistics, organized by time period.
 */
export function StatsSiteView({
    display,
    onClose,
}: StatsSiteViewProps) {
    const { breakpoints, palette } = useTheme();
    const { t } = useTranslation();

    // Period time frame. Defaults to past 24 hours.
    const [period, setPeriod] = useState<{ after: Date, before: Date }>({
        after: new Date(Date.now() - DAYS_1_MS),
        before: new Date(),
    });
    // Menu for picking date range.
    const [dateRangeAnchorEl, setCustomRangeAnchorEl] = useState<Element | null>(null);
    function handleDateRangeOpen(event: any) {
        setCustomRangeAnchorEl(event.currentTarget);
    }
    function handleDateRangeClose() {
        setCustomRangeAnchorEl(null);
    }
    const handleDateRangeSubmit = useCallback((newAfter?: Date | undefined, newBefore?: Date | undefined) => {
        setPeriod({
            after: newAfter || period.after,
            before: newBefore || period.before,
        });
        handleDateRangeClose();
    }, [period.after, period.before]);

    const { currTab, setCurrTab, tabs } = useTabs({ id: "stats-site-tabs", tabParams: statsSiteTabParams, display });
    const handleTabChange = useCallback((_event: ChangeEvent<unknown>, tab: PageTab<TabParamBase<StatsSiteTabsInfo>>) => {
        setCurrTab(tab);
        // Reset date range based on tab selection.
        const period = tabPeriods[tab.key];
        const newAfter = new Date(Math.max(Date.now() - period, MIN_DATE.getTime()));
        const newBefore = new Date(Math.min(Date.now(), newAfter.getTime() + period));
        setPeriod({ after: newAfter, before: newBefore });
    }, [setCurrTab]);

    // Handle querying stats data.
    const [getStats, { data: statsData, loading }] = useLazyFetch<StatsSiteSearchInput, StatsSiteSearchResult>({
        ...endpointsStatsSite.findMany,
        inputs: {
            periodType: tabPeriodTypes[currTab.key] as StatPeriodType,
            periodTimeFrame: {
                after: period.after.toISOString(),
                before: period.before.toISOString(),
            },
        },
    });
    const [stats, setStats] = useState<StatsSite[]>([]);
    useEffect(() => {
        if (statsData) {
            setStats(statsData.edges.map(edge => edge.node));
        }
    }, [statsData]);
    useEffect(() => {
        getStats();
    }, [currTab, period, getStats]);

    // Shape stats data for display.
    const { aggregate, visual } = useMemo(() => statsDisplay(stats), [stats]);

    // Create a card for each aggregate stat
    const aggregateCards = useMemo(() => (
        Object.entries(aggregate).map(([field, value], index) => {
            // Uppercase first letter of field name
            const fieldName = field.charAt(0).toUpperCase() + field.slice(1);
            const title = t(fieldName, { count: 2, ns: "common", defaultValue: field });
            return (
                <Card sx={{
                    width: "100%",
                    height: "100%",
                    background: palette.background.paper,
                    color: palette.background.textPrimary,
                    boxShadow: 0,
                    borderRadius: { xs: 0, sm: 2 },
                    margin: 0,
                    [breakpoints.down("sm")]: {
                        borderBottom: `1px solid ${palette.divider}`,
                    },
                }}>
                    <CardContent sx={{
                        display: "flex",
                        flexDirection: "column",
                        justifyContent: "space-between",
                        height: "100%",
                    }}>
                        {/* TODO add brone, silver, gold, etc. AwardIcon depending on tier */}
                        <Typography
                            variant="h6"
                            component="h2"
                            textAlign="center"
                            mb={2}
                        >{title}</Typography>
                        <Typography
                            variant="body2"
                            component="p"
                            textAlign="center"
                            color="text.secondary"
                        >{value}</Typography>
                    </CardContent>
                </Card>
            );
        })
    ), [aggregate, breakpoints, palette.background.paper, palette.background.textPrimary, palette.divider, t]);

    // Create a line graph card for each visual stat
    const graphCards = useMemo(() => (
        Object.entries(visual).map(([field, data], index) => {
            if (data.length === 0) return null;
            // Uppercase first letter of field name
            const fieldName = field.charAt(0).toUpperCase() + field.slice(1);
            const title = t(fieldName, { count: 2, ns: "common", defaultValue: field });
            return (
                <LineGraphCard
                    data={data}
                    key={`line-graph-card-${field}`}
                    index={index}
                    lineColor='white'
                    title={title}
                />
            );
        })
    ), [t, visual]);

    return (
        <>
            <TopBar
                display={display}
                onClose={onClose}
                title={t("StatisticsShort")}
                titleBehaviorDesktop="ShowIn"
                below={<PageTabs
                    ariaLabel="stats-period-tabs"
                    currTab={currTab}
                    fullWidth={true}
                    onChange={handleTabChange}
                    tabs={tabs}
                />}
            />
            {/* Date range picker */}
            <DateRangeMenu
                anchorEl={dateRangeAnchorEl}
                minDate={MIN_DATE}
                maxDate={new Date()}
                onClose={handleDateRangeClose}
                onSubmit={handleDateRangeSubmit}
                range={period}
                strictIntervalRange={tabPeriods[currTab.key]}
            />
            {/* Date range diplay */}
            <Typography
                component="h3"
                variant="body1"
                textAlign="center"
                onClick={handleDateRangeOpen}
                sx={{ cursor: "pointer", marginBottom: 4, marginTop: 2 }}
            >{displayDate(period.after.getTime(), false) + " - " + displayDate(period.before.getTime(), false)}</Typography>
            {/* Aggregate stats for the time period */}
            <ContentCollapse
                isOpen={true}
                titleKey="Overview"
                sxs={{
                    root: {
                        marginBottom: 4,
                    },
                    titleContainer: {
                        justifyContent: "center",
                    },
                }}
            >
                {stats.length === 0 && <Typography
                    variant="body1"
                    textAlign="center"
                    color="text.secondary"
                    sx={{ marginTop: 4 }}
                >{t("NoData")}</Typography>}
                {aggregateCards.length > 0 && <CardGrid minWidth={300}>
                    {aggregateCards}
                </CardGrid>}
            </ContentCollapse>

            {/* Line graph cards */}
            <ContentCollapse
                isOpen={true}
                titleKey="Visual"
                sxs={{
                    titleContainer: {
                        justifyContent: "center",
                    },
                }}
            >
                {graphCards.length === 0 && <Typography
                    variant="body1"
                    textAlign="center"
                    color="text.secondary"
                    sx={{ marginTop: 4 }}
                >{t("NoData")}</Typography>}
                {graphCards.length > 0 && <CardGrid minWidth={300}>
                    {graphCards}
                </CardGrid>}
            </ContentCollapse>
        </>
    );
}
