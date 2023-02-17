import { Box, Tab, Tabs, Typography } from '@mui/material';
import { useCallback, useMemo, useState } from 'react';
import { DateRangeMenu, PageContainer, StatsList } from 'components';
import { useTranslation } from 'react-i18next';
import { displayDate, getUserLanguages } from 'utils';
import { StatsPageProps } from 'pages/types';

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
    const [fromDate, setFromDate] = useState(new Date(Date.now() - 24 * 60 * 60 * 1000));
    const [toDate, setToDate] = useState(new Date());
    const [resetDateRange, setResetDateRange] = useState(false);
    // Menu for picking date range.
    const [dateRangeAnchorEl, setCustomRangeAnchorEl] = useState<HTMLElement | null>(null);
    const handleDateRangeOpen = (event) => setCustomRangeAnchorEl(event.currentTarget);
    const handleDateRangeClose = () => {
        setCustomRangeAnchorEl(null)
    };
    const handleDateRangeSubmit = (after?: Date | undefined, before?: Date | undefined) => {
        if (after) setFromDate(after);
        if (before) setToDate(before);
        handleDateRangeClose();
    };

    // Handle tabs. Defaults to "Daily" tab.
    const [tabIndex, setTabIndex] = useState(0);
    const handleTabChange = useCallback((event, newValue) => {
        setTabIndex(newValue);
        // Set date range based on tab selection.
        const period = tabPeriods[TabOptions[newValue]];
        let newFromDate = new Date(Date.now() - period);
        if (newFromDate < MIN_DATE) newFromDate = MIN_DATE;
        const newToDate = new Date(newFromDate.getTime() + period);
        setFromDate(newFromDate);
        setToDate(newToDate);
        // Toggle resetDateRange to force DateRangeMenu to reset its date range.
        setResetDateRange(true);
        setResetDateRange(false);
    }, []);

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
                resetDateRange={resetDateRange}
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
            >{displayDate(fromDate.getTime(), false) + " - " + displayDate(toDate.getTime(), false)}</Typography>
            {/* Aggregate stats for the time period */}
            <Typography component="h1" variant="h4" textAlign="center">Quick Overview</Typography>
            {/* Stats as bar graphs */}
            <Typography component="h1" variant="h4" textAlign="center">The Pretty Pictures</Typography>
            <StatsList data={[{}, {}, {}, {}, {}, {}]} />
        </PageContainer>
    )
};
