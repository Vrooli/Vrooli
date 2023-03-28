import { useTheme } from '@mui/material';
import { ScheduleRecurrenceType, ScheduleSearchResult } from '@shared/consts';
import { addSearchParams, parseSearchParams, useLocation } from '@shared/route';
import { CommonKey } from '@shared/translations';
import { calculateOccurrences } from '@shared/utils';
import { uuid } from '@shared/uuid';
import { FullPageSpinner } from 'components/FullPageSpinner/FullPageSpinner';
import { TopBar } from 'components/navigation/TopBar/TopBar';
import { PageTabs } from 'components/PageTabs/PageTabs';
import { PageTab } from 'components/types';
import { add, endOfMonth, format, getDay, startOfMonth, startOfWeek } from 'date-fns';
import { useCallback, useContext, useEffect, useMemo, useState } from 'react';
import { Calendar, dateFnsLocalizer, DateLocalizer } from 'react-big-calendar';
import 'react-big-calendar/lib/css/react-big-calendar.css';
import { useTranslation } from 'react-i18next';
import { CalendarEvent } from 'types';
import { getDisplay } from 'utils/display/listTools';
import { getUserLanguages, getUserLocale, loadLocale } from 'utils/display/translationTools';
import { useFindMany } from 'utils/hooks/useFindMany';
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
const sampleSchedules = [
    {
        id: uuid(),
        title: 'Meeting',
        startTime: new Date(),
        endTime: add(new Date(), { hours: 1 }),
        recurrences: [
            {
                __typename: 'ScheduleRecurrence' as const,
                id: uuid(),
                recurrenceType: ScheduleRecurrenceType.Weekly,
                interval: 1,
                dayOfWeek: 3,
            },
        ],
        exceptions: [
            {
                __typename: 'ScheduleException' as const,
                id: uuid(),
                originalStartTime: add(new Date(), { weeks: 1 }),
                newStartTime: add(add(new Date(), { weeks: 1 }), { days: 1 }),
                newEndTime: add(add(add(new Date(), { weeks: 1 }), { days: 1 }), { hours: 1 }),
            },
        ],
        labels: [
            {
                __typename: 'Label' as const,
                id: uuid(),
                color: '#4caf50',
                label: 'Work',
            },
            {
                __typename: 'Label' as const,
                id: uuid(),
                label: 'Important',
            },
        ],
        // Dummy data for reminders
        // reminders: ['10 minutes before', '1 hour before'],
    },
    {
        id: uuid(),
        title: 'Monthly Report',
        startTime: add(new Date(), { days: 5 }),
        endTime: add(add(new Date(), { days: 5 }), { hours: 2 }),
        recurrences: [
            {
                __typename: 'ScheduleRecurrence' as const,
                id: uuid(),
                recurrenceType: ScheduleRecurrenceType.Monthly,
                interval: 1,
                dayOfMonth: 10,
            },
        ],
        exceptions: [],
        labels: [
            {
                __typename: 'Label' as const,
                id: uuid(),
                color: '#2196f3',
                label: 'Reports',
            },
        ],
    },
    {
        id: uuid(),
        title: 'Bi-weekly Team Lunch',
        startTime: add(new Date(), { days: 6 }),
        endTime: add(add(new Date(), { days: 6 }), { hours: 1 }),
        recurrences: [
            {
                __typename: 'ScheduleRecurrence' as const,
                id: uuid(),
                recurrenceType: ScheduleRecurrenceType.Weekly,
                interval: 2,
                dayOfWeek: 6,
            },
        ],
        exceptions: [
            {
                __typename: 'ScheduleException' as const,
                id: uuid(),
                originalStartTime: add(new Date(), { weeks: 2 }),
                newStartTime: add(add(new Date(), { weeks: 2 }), { days: 2 }),
                newEndTime: add(add(add(new Date(), { weeks: 2 }), { days: 2 }), { hours: 1 }),
            },
        ],
        labels: [
            {
                __typename: 'Label' as const,
                id: uuid(),
                color: '#ff9800',
                label: 'Social',
            },
        ],
    },
    {
        id: uuid(),
        title: 'Daily Stand-up',
        startTime: add(new Date(), { days: 1 }),
        endTime: add(add(new Date(), { days: 1 }), { minutes: 15 }),
        recurrences: [
            {
                __typename: 'ScheduleRecurrence' as const,
                id: uuid(),
                recurrenceType: ScheduleRecurrenceType.Daily,
                interval: 1,
                endDate: add(new Date(), { days: 15 }),
            },
        ],
        exceptions: [
            {
                __typename: 'ScheduleException' as const,
                id: uuid(),
                originalStartTime: add(new Date(), { days: 2 }),
                newStartTime: null,
                newEndTime: null,
            },
        ],
        labels: [
            {
                __typename: 'Label' as const,
                id: uuid(),
                color: '#f44336',
                label: 'Stand-up',
            }
        ],
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
    // Defaults to current month
    const [dateRange, setDateRange] = useState<{ start: Date, end: Date }>({
        start: startOfMonth(new Date()),
        end: endOfMonth(new Date()),
    });

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

    // Track heights used to calculate calendar height
    const [heights, setHeights] = useState<{
        topBar: number;
        bottomNav: number;
    }>({
        topBar: document.getElementById('navbar')?.clientHeight ?? 0,
        bottomNav: document.getElementById('bottom-nav')?.clientHeight ?? 0,
    });
    useEffect(() => {
        const handleResize = () => {
            const topBarHeight = document.getElementById('navbar')?.clientHeight ?? 0;
            const bottomNavHeight = document.getElementById('bottom-nav')?.clientHeight ?? 0;
            console.log('topBarHeight', document.getElementById('navbar'));
            console.log('bottomNavHeight', document.getElementById('bottom-nav'));
            setHeights({ topBar: topBarHeight, bottomNav: bottomNavHeight });
        };
        window.addEventListener('resize', handleResize);
        return () => {
            window.removeEventListener('resize', handleResize);
        };
    }, []);
    console.log('heights', heights);

    const handleDateRangeChange = useCallback((range: Date[] | { start: Date, end: Date }) => {
        if (Array.isArray(range)) {
            setDateRange({ start: range[0], end: range[1] });
        } else {
            setDateRange(range);
        }
    }, []);

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

    // Find schedules
    const {
        allData: schedules,
        loading,
        loadMore,
    } = useFindMany<ScheduleSearchResult>({
        canSearch: true,
        searchType: 'Schedule',
        where: {
            // Only find schedules that hav not ended, 
            // and will start before the date range ends
            endTimeFrame: (dateRange.start && dateRange.end) ? {
                after: dateRange.start.toISOString(),
                before: add(dateRange.end, { years: 1000 }).toISOString(),
            } : undefined,
            startTimeFrame: (dateRange.start && dateRange.end) ? {
                after: add(dateRange.start, { years: -1000 }).toISOString(),
                before: dateRange.end.toISOString(),
            } : undefined,
            // TODO specify your own schedules here
        },
    });
    // Load more schedules when date range changes
    useEffect(() => {
        if (!loading && dateRange.start && dateRange.end) {
            loadMore();
        }
    }, [dateRange, loadMore, loading]);

    // Handle events, which are created from schedule data.
    // Events represent each occurrence of a schedule within a date range
    const events = useMemo<CalendarEvent[]>(() => {
        console.log('calculating events start', dateRange, sampleSchedules)
        if (!dateRange.start || !dateRange.end) return [];
        // Initialize result
        const result: CalendarEvent[] = [];
        // Loop through schedules
        sampleSchedules.forEach((schedule: any) => {
            console.log('calculating events schedule', schedule)
            // Get occurrences (i.e. start and end times)
            const occurrences = calculateOccurrences(schedule, dateRange.start!, dateRange.end!);
            console.log('calculating events occurrences', occurrences)
            // Create events
            const events: CalendarEvent[] = occurrences.map(occurrence => ({
                __typename: 'CalendarEvent',
                id: `${schedule.id}|${occurrence.start.getTime()}|${occurrence.end.getTime()}`,
                title: getDisplay(schedule, getUserLanguages(session)).title,
                start: occurrence.start,
                end: occurrence.end,
                allDay: false,
                schedule,
            }));
            console.log('calculating events events', events)
            // Add events to result
            result.push(...events);
        });
        console.log('calculating events end', result)
        return result;
    }, [dateRange.end, dateRange.start, schedules, session]);


    const openEvent = useCallback((event: any) => {
        console.log('CalendarEvent clicked:', event);
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
            <Calendar
                localizer={localizer}
                events={events}
                onRangeChange={handleDateRangeChange}
                onSelectEvent={openEvent}
                startAccessor="start"
                endAccessor="end"
                views={['month', 'week', 'day']}
                style={{
                    height: `calc(100vh - ${heights.topBar}px - ${heights.bottomNav}px - env(safe-area-inset-bottom))`,
                    background: palette.background.paper,
                }}
            />
        </>
    )
}