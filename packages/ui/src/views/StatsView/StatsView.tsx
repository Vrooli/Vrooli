import { Box, Card, CardContent, Grid, Typography, useTheme } from '@mui/material';
import { StatPeriodType, StatsSite, StatsSiteSearchInput, StatsSiteSearchResult } from '@shared/consts';
import { useCustomLazyQuery } from 'api';
import { statsSiteFindMany } from 'api/generated/endpoints/statsSite_findMany';
import { ContentCollapse } from 'components/containers/ContentCollapse/ContentCollapse';
import { CardGrid } from 'components/lists/CardGrid/CardGrid';
import { DateRangeMenu } from 'components/lists/DateRangeMenu/DateRangeMenu';
import { LineGraphCard } from 'components/lists/LineGraphCard/LineGraphCard';
import { TopBar } from 'components/navigation/TopBar/TopBar';
import { PageTabs } from 'components/PageTabs/PageTabs';
import { PageTab } from 'components/types';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { statsDisplay } from 'utils/display/statsDisplay';
import { displayDate } from 'utils/display/stringTools';
import { StatsViewProps } from '../types';

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
    display = 'page',
}: StatsViewProps) => {
    const { palette } = useTheme();
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
            // Uppercase first letter of field name
            const fieldName = field.charAt(0).toUpperCase() + field.slice(1);
            const title = t(fieldName, { count: 2, ns: 'common', defaultValue: field });
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
                        title={title}
                    />
                </Box>
            )
        })
    ), [t, visual]);

    return (
        <>
            <TopBar
                display={display}
                onClose={() => { }}
                titleData={{
                    titleKey: 'StatisticsShort',
                }}
                below={<PageTabs
                    ariaLabel="stats-period-tabs"
                    currTab={currTab}
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
            <ContentCollapse
                isOpen={true}
                titleKey="Overview"
                sxs={{
                    root: {
                        marginBottom: 4
                    }
                }}
            >
                <Grid container spacing={2}>
                    {Object.entries(aggregate).map(([field, value], index) => {
                        // Uppercase first letter of field name
                        const fieldName = field.charAt(0).toUpperCase() + field.slice(1);
                        const title = t(fieldName, { count: 2, ns: "common", defaultValue: field });
                        return (
                            <Grid key={index} item xs={12} sm={6} md={4} lg={3}>
                                <Card sx={{
                                    background: palette.primary.light,
                                    color: palette.primary.contrastText,
                                    height: '100%'
                                }}>
                                    <CardContent>
                                        <Typography variant="h6" textAlign="center" gutterBottom>
                                            {title}
                                        </Typography>
                                        <Typography variant="body1" textAlign="center">
                                            {value}
                                        </Typography>
                                    </CardContent>
                                </Card>
                            </Grid>
                        );
                    })}
                </Grid>
            </ContentCollapse>
            {/* Line graph cards */}
            <ContentCollapse
                isOpen={true}
                titleKey="Visual"
            >
                <CardGrid minWidth={275}>
                    {cards}
                </CardGrid>
            </ContentCollapse>
        </>
    )
};
