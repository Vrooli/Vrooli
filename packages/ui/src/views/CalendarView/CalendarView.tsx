import { Box, useTheme } from '@mui/material';
import { addSearchParams, parseSearchParams, useLocation } from '@shared/route';
import { CommonKey } from '@shared/translations';
import { FullPageSpinner } from 'components/FullPageSpinner/FullPageSpinner';
import { TopBar } from 'components/navigation/TopBar/TopBar';
import { PageTabs } from 'components/PageTabs/PageTabs';
import { PageTab } from 'components/types';
import { add, format, getDay, startOfWeek } from 'date-fns';
import { useCallback, useContext, useEffect, useMemo, useState } from 'react';
import { Calendar, dateFnsLocalizer, DateLocalizer } from 'react-big-calendar';
import 'react-big-calendar/lib/css/react-big-calendar.css';
import { useTranslation } from 'react-i18next';
import { getUserLocale, loadLocale } from 'utils/display/translationTools';
import { CalendarPageTabOption } from 'utils/search/objectToSearch';
import { SessionContext } from 'utils/SessionContext';
import { CalendarViewProps } from 'views/types';

// Tab data type
type BaseParams = {
    titleKey: CommonKey;
    tabType: CalendarPageTabOption;
}

// Data for each tab
const tabParams: BaseParams[] = [{
    titleKey: 'Meeting',
    tabType: CalendarPageTabOption.Meetings,
}, {
    titleKey: 'Routine',
    tabType: CalendarPageTabOption.Routines,
}, {
    titleKey: 'Project',
    tabType: CalendarPageTabOption.Projects,
}, {
    titleKey: 'FocusModeShort',
    tabType: CalendarPageTabOption.FocusModes,
}];

// Replace this with your own events data
const sampleEvents = [
    {
        id: 1,
        title: 'Meeting',
        start: new Date(),
        end: add(new Date(), { hours: 1 }),
    },
    {
        id: 2,
        title: 'Conference',
        start: add(new Date(), { days: 3 }),
        end: add(add(new Date(), { days: 3 }), { hours: 2 }),
    },
];


export const CalendarView = ({
    display = 'page',
}: CalendarViewProps) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const locale = useMemo(() => getUserLocale(session), [session]);
    const [localizer, setLocalizer] = useState<DateLocalizer | null>(null);

    useEffect(() => {
        const localeLoader = async () => {
            try {
                const localeModule = await loadLocale(locale as any)

                const newLocalizer = dateFnsLocalizer({
                    format,
                    startOfWeek,
                    getDay,
                    locales: {
                        [locale]: localeModule,
                    },
                });

                setLocalizer(newLocalizer);
            } catch (error) {
                console.error('Failed to load locale:', error);
            }
        };

        localeLoader();
    }, [locale]);

    // Handle tabs
    const tabs = useMemo<PageTab<CalendarPageTabOption>[]>(() => {
        return tabParams.map((tab, i) => ({
            index: i,
            label: t(tab.titleKey, { count: 2, defaultValue: tab.titleKey }),
            value: tab.tabType,
        }));
    }, [t]);
    const [currTab, setCurrTab] = useState<PageTab<CalendarPageTabOption>>(() => {
        const searchParams = parseSearchParams();
        const index = tabParams.findIndex(tab => tab.tabType === searchParams.type);
        // Default to bookmarked tab
        if (index === -1) return tabs[0];
        // Return tab
        return tabs[index];
    });
    const handleTabChange = useCallback((e: any, tab: PageTab<CalendarPageTabOption>) => {
        e.preventDefault();
        // Update search params
        addSearchParams(setLocation, { type: tab.value });
        // Update curr tab
        setCurrTab(tab)
    }, [setLocation]);

    const [events, setEvents] = useState(sampleEvents);

    // const [eventsData, { data: pageData, loading, error }] = useCustomlazyQuery<QueryResult, QueryVariables>(query, {
    //     variables: ({
    //         after: after.current,
    //         take,
    //         sortBy,
    //         searchString,
    //         createdTimeFrame: (timeFrame && Object.keys(timeFrame).length > 0) ? {
    //             after: timeFrame.after?.toISOString(),
    //             before: timeFrame.before?.toISOString(),
    //         } : undefined,
    //         ...where,
    //         ...advancedSearchParams
    //     } as any),
    //     errorPolicy: 'all',
    // });

    const openEvent = useCallback((event: any) => {
        console.log('Event clicked:', event);
    }, []);

    if (!localizer) return <FullPageSpinner />
    return (
        <>
            <TopBar
                display={display}
                onClose={() => { }}
                titleData={{
                    titleKey: 'Calendar',
                }}
                below={<PageTabs
                    ariaLabel="calendar-tabs"
                    currTab={currTab}
                    fullWidth
                    onChange={handleTabChange}
                    tabs={tabs}
                />}
            />
            <Box mt={2} p={2} style={{
                height: '500px',
                background: palette.background.paper,
            }}>
                <Calendar
                    localizer={localizer}
                    events={events}
                    onSelectEvent={openEvent}
                    startAccessor="start"
                    endAccessor="end"
                    views={['month', 'week', 'day']}
                    style={{ height: '100%' }}
                />
            </Box>
        </>
    )
}