import { Box, Typography } from '@mui/material';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { LineGraphCard, DateRangeMenu, PageTabs, CardGrid } from 'components';
import { useTranslation } from 'react-i18next';
import { displayDate, statsDisplay } from 'utils';
import { StatsViewProps } from '../types';
import { StatPeriodType, StatsSite, StatsSiteSearchInput, StatsSiteSearchResult } from '@shared/consts';
import { useCustomLazyQuery } from 'api';
import { statsSiteFindMany } from 'api/generated/endpoints/statsSite_findMany';
import { PageTab } from 'components/types';

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
const TabOptions = ['Daily', 'Weekly', 'Monthly', 'Yearly', 'AllTime'] as const;

/**
 * Stores the time frame interval for each tab.
 */
const tabPeriods: { [key in typeof TabOptions[number]]: number } = {
    Daily: 24 * 60 * 60 * 1000, // Past 24 hours
    Weekly: 7 * 24 * 60 * 60 * 1000, // Past 7 days
    Monthly: 30 * 24 * 60 * 60 * 1000, // Past 30 days
    Yearly: 365 * 24 * 60 * 60 * 1000, // Past 365 days
    AllTime: Number.MAX_SAFE_INTEGER, // All time
};

/**
 * Maps tab options to PeriodType.
 */
const tabPeriodTypes = {
    Daily: 'Hourly',
    Weekly: 'Daily',
    Monthly: 'Weekly',
    Yearly: 'Monthly',
    AllTime: 'Yearly',
} as const;

// Stats should not be earlier than February 2023.
const MIN_DATE = new Date(2023, 1, 1);

/**
 * Displays site-wide statistics, organized by time period.
 */
export const StatsView = ({
    session,
}: StatsViewProps) => {
    const { t } = useTranslation();

    // Period time frame. Defaults to past 24 hours.
    const [period, setPeriod] = useState<{ after: Date, before: Date }>({
        after: new Date(Date.now() - 24 * 60 * 60 * 1000),
        before: new Date()
    });
    // Menu for picking date range.
    const [dateRangeAnchorEl, setCustomRangeAnchorEl] = useState<HTMLElement | null>(null);
    const handleDateRangeOpen = (event: any) => setCustomRangeAnchorEl(event.currentTarget);
    const handleDateRangeClose = () => {
        setCustomRangeAnchorEl(null)
    };
    const handleDateRangeSubmit = useCallback((newAfter?: Date | undefined, newBefore?: Date | undefined) => {
        setPeriod({
            after: newAfter || period.after,
            before: newBefore || period.before,
        })
        handleDateRangeClose();
    }, [period.after, period.before]);

    // Handle tabs
    const tabs = useMemo<PageTab<typeof TabOptions[number]>[]>(() => {
        let tabs = TabOptions;
        // Return tabs shaped for the tab component
        return tabs.map((tab, i) => ({
            index: i,
            label: t(tab, { count: 2 }),
            value: tab,
        }));
    }, [t]);
    const [currTab, setCurrTab] = useState<PageTab<typeof TabOptions[number]>>(tabs[0]);
    const handleTabChange = useCallback((e: any, tab: PageTab<typeof TabOptions[number]>) => {
        setCurrTab(tab);
        // Reset date range based on tab selection.
        const period = tabPeriods[tab.value];
        const newAfter = new Date(Math.max(Date.now() - period, MIN_DATE.getTime()));
        const newBefore = new Date(Math.min(Date.now(), newAfter.getTime() + period));
        console.log('yeeties', newAfter, newBefore)
        setPeriod({ after: newAfter, before: newBefore });
    }, []);

    // Handle querying stats data.
    const [getStats, { data: statsData, loading }] = useCustomLazyQuery<StatsSiteSearchResult, StatsSiteSearchInput>(statsSiteFindMany, {
        variables: ({
            periodType: tabPeriodTypes[currTab.value] as StatPeriodType,
            periodTimeFrame: {
                after: period.after.toISOString(),
                before: period.before.toISOString(),
            },
        }),
        errorPolicy: 'all',
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

    // Create a line graph card for each visual stat
    const cards = useMemo(() => (
        Object.entries(visual).map(([field, data], index) => {
            if (data.length === 0) return null;
            return (
                <Box
                    key={index}
                    sx={{
                        margin: 2,
                        display: 'flex',
                        justifyContent: 'center',
                        alignItems: 'center',
                    }}
                >
                    <LineGraphCard
                        data={data}
                        index={index}
                        lineColor='white'
                        title={field}
                    />
                </Box>
            )
        })
    ), [visual]);

    return (
        <>
            <PageTabs
                ariaLabel="stats-period-tabs"
                currTab={currTab}
                onChange={handleTabChange}
                tabs={tabs}
            />
            {/* Date range picker */}
            <DateRangeMenu
                anchorEl={dateRangeAnchorEl}
                minDate={MIN_DATE}
                maxDate={new Date()}
                onClose={handleDateRangeClose}
                onSubmit={handleDateRangeSubmit}
                range={period}
                strictIntervalRange={tabPeriods[currTab.value]}
            />
            {/* Date range diplay */}
            <Typography
                component="h3"
                variant="body1"
                textAlign="center"
                onClick={handleDateRangeOpen}
                sx={{ cursor: 'pointer' }}
            >{displayDate(period.after.getTime(), false) + " - " + displayDate(period.before.getTime(), false)}</Typography>
            {/* Aggregate stats for the time period */}
            <Typography component="h1" variant="h4" textAlign="center">{t(`Overview`)}</Typography>
            {/* Line graph cards */}
            <Typography component="h1" variant="h4" textAlign="center">Visual</Typography>
            <CardGrid minWidth={275}>
                {cards}
            </CardGrid>
        </>
    )
};
