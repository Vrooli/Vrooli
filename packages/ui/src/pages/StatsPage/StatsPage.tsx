import { Box, Tab, Tabs, Typography } from '@mui/material';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { LineGraphCard, DateRangeMenu, PageContainer } from 'components';
import { useTranslation } from 'react-i18next';
import { displayDate, getUserLanguages, statsDisplay } from 'utils';
import { StatsPageProps } from 'pages/types';
import { StatPeriodType, StatsSite, StatsSiteSearchInput, StatsSiteSearchResult } from '@shared/consts';
import { useLazyQuery } from 'api';
import { statsSiteFindMany } from 'api/generated/endpoints/statsSite';

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
const tabPeriodTypes: { [key in typeof TabOptions[number]]: `${StatPeriodType}` } = {
    Daily: 'Hourly',
    Weekly: 'Daily',
    Monthly: 'Weekly',
    Yearly: 'Monthly',
    AllTime: 'Yearly',
};

// Stats should not be earlier than February 2023.
const MIN_DATE = new Date(2023, 1, 1);

const tabProps = (index: number) => ({
    id: `stats-tab-${index}`,
    'aria-controls': `full-width-tabpanel-${index}`,
});

/**
 * Displays site-wide statistics, organized by time period.
 */
export const StatsPage = ({
    session,
}: StatsPageProps) => {
    const { t } = useTranslation();
    const lng = useMemo(() => getUserLanguages(session)[0], [session]);

    // Period time frame. Defaults to past 24 hours.
    const [period, setPeriod] = useState<{ after: Date, before: Date }>({
        after: new Date(Date.now() - 24 * 60 * 60 * 1000),
        before: new Date()
    });
    // Menu for picking date range.
    const [dateRangeAnchorEl, setCustomRangeAnchorEl] = useState<HTMLElement | null>(null);
    const handleDateRangeOpen = (event) => setCustomRangeAnchorEl(event.currentTarget);
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

    // Handle tabs. Defaults to "Daily" tab.
    const [tabIndex, setTabIndex] = useState(0);
    const handleTabChange = useCallback((event, newValue) => {
        setTabIndex(newValue);
        // Reset date range based on tab selection.
        const period = tabPeriods[TabOptions[newValue]];
        const newAfter = new Date(Math.max(Date.now() - period, MIN_DATE.getTime()));
        const newBefore = new Date(Math.min(Date.now(), newAfter.getTime() + period));
        console.log('yeeties', newAfter, newBefore)
        setPeriod({ after: newAfter, before: newBefore });
    }, []);

    // Handle querying stats data.
    const [getStats, { data: statsData, loading }] = useLazyQuery<StatsSiteSearchResult, StatsSiteSearchInput, 'statsSite'>(statsSiteFindMany, 'statsSite', {
        variables: ({
            periodType: tabPeriodTypes[TabOptions[tabIndex]] as StatPeriodType,
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
            setStats(statsData.statsSite.edges.map(edge => edge.node));
        }
    }, [statsData]);
    useEffect(() => {
        getStats();
    }, [tabIndex, period, getStats]);

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
        <PageContainer>
            {/* Tabs to switch time frame size */}
            <Box display="flex" justifyContent="center" width="100%">
                <Tabs
                    value={tabIndex}
                    onChange={handleTabChange}
                    indicatorColor="secondary"
                    textColor="inherit"
                    variant="scrollable"
                    scrollButtons="auto"
                    allowScrollButtonsMobile
                    aria-label="site-statistics-tabs"
                    sx={{
                        marginBottom: 1,
                        paddingLeft: '1em',
                        paddingRight: '1em',
                    }}
                >
                    {TabOptions.map((label, index) => (
                        <Tab key={index} label={t(`common:${label}`, { lng })} {...tabProps(index)} />
                    ))}
                </Tabs>
            </Box>
            {/* Date range picker */}
            <DateRangeMenu
                anchorEl={dateRangeAnchorEl}
                minDate={MIN_DATE}
                maxDate={new Date()}
                onClose={handleDateRangeClose}
                onSubmit={handleDateRangeSubmit}
                range={period}
                session={session}
                strictIntervalRange={tabPeriods[TabOptions[tabIndex]]}
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
            <Typography component="h1" variant="h4" textAlign="center">{t(`common:Overview`, { lng })}</Typography>
            {/* Line graph cards */}
            <Typography component="h1" variant="h4" textAlign="center">Visual</Typography>
            <Box
                sx={{
                    display: 'grid',
                    gridTemplateColumns: 'repeat(auto-fill, minmax(275px, 1fr))',
                    gridAutoRows: '275px',
                    alignItems: 'stretch',
                }}
            >
                {cards}
            </Box>
        </PageContainer>
    )
};
