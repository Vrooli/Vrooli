import { useEffect, useMemo, useState } from 'react';
import { Calendar, dateFnsLocalizer } from 'react-big-calendar';
import { format, startOfWeek, getDay, add } from 'date-fns';
import 'react-big-calendar/lib/css/react-big-calendar.css';
import { CalendarViewProps } from 'views/types';
import { Box, useTheme } from '@mui/material';
import { getUserLocale } from 'utils';
import { FullPageSpinner, TopBar } from 'components';
import { useTranslation } from 'react-i18next';

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
    session,
}: CalendarViewProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const locale = useMemo(() => getUserLocale(session), [session]);
    const [localizer, setLocalizer] = useState(null);

    // useEffect(() => {
    //     const loadLocale = async () => {
    //         try {
    //             const localeModule = await import(
    //                 /* @vite-ignore */
    //                 `date-fns/esm/locale/${locale}/index.js`
    //             );
    //             const currentLocale = localeModule.default;

    //             const newLocalizer = dateFnsLocalizer({
    //                 format,
    //                 startOfWeek,
    //                 getDay,
    //                 locales: {
    //                     [locale]: currentLocale,
    //                 },
    //             });

    //             setLocalizer(newLocalizer);
    //         } catch (error) {
    //             console.error('Failed to load locale:', error);
    //         }
    //     };

    //     loadLocale();
    // }, [locale]);

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

    if (!localizer) return <FullPageSpinner />
    return (
        <>
            <TopBar
                display={display}
                onClose={() => { }}
                session={session}
                titleData={{
                    titleKey: 'Calendar',
                }}
            />
            <Box style={{ height: '500px' }}>
                <Calendar
                    localizer={localizer}
                    events={events}
                    startAccessor="start"
                    endAccessor="end"
                    views={['month', 'week', 'day']}
                    style={{ height: '100%' }}
                />
            </Box>
        </>
    )
}