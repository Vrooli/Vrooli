import { Schedule, ScheduleSearchResult } from "@local/consts";
import { AddIcon, FocusModeIcon, OrganizationIcon, ProjectIcon, RoutineIcon, SvgProps } from "@local/icons";
import { CommonKey } from "@local/translations";
import { calculateOccurrences } from "@local/utils";
import { useTheme } from "@mui/material";
import { add, endOfMonth, format, getDay, startOfMonth, startOfWeek } from "date-fns";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { Calendar, dateFnsLocalizer, DateLocalizer } from "react-big-calendar";
import "react-big-calendar/lib/css/react-big-calendar.css";
import { useTranslation } from "react-i18next";
import { ColorIconButton } from "../../components/buttons/ColorIconButton/ColorIconButton";
import { SideActionButtons } from "../../components/buttons/SideActionButtons/SideActionButtons";
import { ScheduleDialog } from "../../components/dialogs/ScheduleDialog/ScheduleDialog";
import { FullPageSpinner } from "../../components/FullPageSpinner/FullPageSpinner";
import { TopBar } from "../../components/navigation/TopBar/TopBar";
import { PageTabs } from "../../components/PageTabs/PageTabs";
import { PageTab } from "../../components/types";
import { CalendarEvent } from "../../types";
import { getCurrentUser } from "../../utils/authentication/session";
import { getDisplay } from "../../utils/display/listTools";
import { getUserLanguages, getUserLocale, loadLocale } from "../../utils/display/translationTools";
import { useDimensions } from "../../utils/hooks/useDimensions";
import { useFindMany } from "../../utils/hooks/useFindMany";
import { useWindowSize } from "../../utils/hooks/useWindowSize";
import { addSearchParams, parseSearchParams, useLocation } from "../../utils/route";
import { CalendarPageTabOption } from "../../utils/search/objectToSearch";
import { SessionContext } from "../../utils/SessionContext";
import { CalendarViewProps } from "../types";

// Tab data type
type BaseParams = {
    Icon: (props: SvgProps) => JSX.Element,
    titleKey: CommonKey;
    tabType: CalendarPageTabOption;
}

// Data for each tab
const tabParams: BaseParams[] = [{
    Icon: OrganizationIcon,
    titleKey: "Meeting",
    tabType: CalendarPageTabOption.Meetings,
}, {
    Icon: RoutineIcon,
    titleKey: "Routine",
    tabType: CalendarPageTabOption.Routines,
}, {
    Icon: ProjectIcon,
    titleKey: "Project",
    tabType: CalendarPageTabOption.Projects,
}, {
    Icon: FocusModeIcon,
    titleKey: "FocusMode",
    tabType: CalendarPageTabOption.FocusModes,
}];

// // Replace this with your own events data
// const sampleSchedules = [
//     {
//         id: uuid(),
//         title: 'Meeting',
//         startTime: new Date(),
//         endTime: add(new Date(), { hours: 1 }),
//         recurrences: [
//             {
//                 __typename: 'ScheduleRecurrence' as const,
//                 id: uuid(),
//                 recurrenceType: ScheduleRecurrenceType.Weekly,
//                 interval: 1,
//                 dayOfWeek: 3,
//             },
//         ],
//         exceptions: [
//             {
//                 __typename: 'ScheduleException' as const,
//                 id: uuid(),
//                 originalStartTime: add(new Date(), { weeks: 1 }),
//                 newStartTime: add(add(new Date(), { weeks: 1 }), { days: 1 }),
//                 newEndTime: add(add(add(new Date(), { weeks: 1 }), { days: 1 }), { hours: 1 }),
//             },
//         ],
//         labels: [
//             {
//                 __typename: 'Label' as const,
//                 id: uuid(),
//                 color: '#4caf50',
//                 label: 'Work',
//             },
//             {
//                 __typename: 'Label' as const,
//                 id: uuid(),
//                 label: 'Important',
//             },
//         ],
//         // Dummy data for reminders
//         // reminders: ['10 minutes before', '1 hour before'],
//     },
//     {
//         id: uuid(),
//         title: 'Monthly Report',
//         startTime: add(new Date(), { days: 5 }),
//         endTime: add(add(new Date(), { days: 5 }), { hours: 2 }),
//         recurrences: [
//             {
//                 __typename: 'ScheduleRecurrence' as const,
//                 id: uuid(),
//                 recurrenceType: ScheduleRecurrenceType.Monthly,
//                 interval: 1,
//                 dayOfMonth: 10,
//             },
//         ],
//         exceptions: [],
//         labels: [
//             {
//                 __typename: 'Label' as const,
//                 id: uuid(),
//                 color: '#2196f3',
//                 label: 'Reports',
//             },
//         ],
//     },
//     {
//         id: uuid(),
//         title: 'Bi-weekly Team Lunch',
//         startTime: add(new Date(), { days: 6 }),
//         endTime: add(add(new Date(), { days: 6 }), { hours: 1 }),
//         recurrences: [
//             {
//                 __typename: 'ScheduleRecurrence' as const,
//                 id: uuid(),
//                 recurrenceType: ScheduleRecurrenceType.Weekly,
//                 interval: 2,
//                 dayOfWeek: 6,
//             },
//         ],
//         exceptions: [
//             {
//                 __typename: 'ScheduleException' as const,
//                 id: uuid(),
//                 originalStartTime: add(new Date(), { weeks: 2 }),
//                 newStartTime: add(add(new Date(), { weeks: 2 }), { days: 2 }),
//                 newEndTime: add(add(add(new Date(), { weeks: 2 }), { days: 2 }), { hours: 1 }),
//             },
//         ],
//         labels: [
//             {
//                 __typename: 'Label' as const,
//                 id: uuid(),
//                 color: '#ff9800',
//                 label: 'Social',
//             },
//         ],
//     },
//     {
//         id: uuid(),
//         title: 'Daily Stand-up',
//         startTime: add(new Date(), { days: 1 }),
//         endTime: add(add(new Date(), { days: 1 }), { minutes: 15 }),
//         recurrences: [
//             {
//                 __typename: 'ScheduleRecurrence' as const,
//                 id: uuid(),
//                 recurrenceType: ScheduleRecurrenceType.Daily,
//                 interval: 1,
//                 endDate: add(new Date(), { days: 15 }),
//             },
//         ],
//         exceptions: [
//             {
//                 __typename: 'ScheduleException' as const,
//                 id: uuid(),
//                 originalStartTime: add(new Date(), { days: 2 }),
//                 newStartTime: null,
//                 newEndTime: null,
//             },
//         ],
//         labels: [
//             {
//                 __typename: 'Label' as const,
//                 id: uuid(),
//                 color: '#f44336',
//                 label: 'Stand-up',
//             }
//         ],
//     },
// ];

export const CalendarView = ({
    display = "page",
}: CalendarViewProps) => {
    const session = useContext(SessionContext);
    const { breakpoints, palette } = useTheme();
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
                const localeModule = await loadLocale(locale as any);

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
                console.error("Failed to load locale:", error);
            }
        };

        localeLoader();
    }, [locale]);

    // Data for calculating calendar height
    const { dimensions, ref } = useDimensions();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);
    const calendarHeight = useMemo(() => `calc(100vh - ${dimensions.height}px - ${isMobile ? 56 : 0}px - env(safe-area-inset-bottom))`, [dimensions, isMobile]);

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
            Icon: tab.Icon,
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
        setCurrTab(tab);
    }, [setLocation]);

    // Find schedules
    const {
        allData: schedules,
        hasMore,
        loading,
        loadMore,
    } = useFindMany<ScheduleSearchResult>({
        canSearch: true,
        searchType: "Schedule",
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
            scheduleForUserId: getCurrentUser(session)?.id,
        },
    });
    // Load more schedules when date range changes
    useEffect(() => {
        if (!loading && hasMore && dateRange.start && dateRange.end) {
            loadMore();
        }
    }, [dateRange, loadMore, loading, hasMore]);

    // Handle events, which are created from schedule data.
    // Events represent each occurrence of a schedule within a date range
    const events = useMemo<CalendarEvent[]>(() => {
        if (!dateRange.start || !dateRange.end) return [];
        // Initialize result
        const result: CalendarEvent[] = [];
        // Loop through schedules
        schedules.forEach((schedule: any) => {
            // Get occurrences (i.e. start and end times)
            const occurrences = calculateOccurrences(schedule, dateRange.start!, dateRange.end!);
            // Create events
            const events: CalendarEvent[] = occurrences.map(occurrence => ({
                __typename: "CalendarEvent",
                id: `${schedule.id}|${occurrence.start.getTime()}|${occurrence.end.getTime()}`,
                title: getDisplay(schedule, getUserLanguages(session)).title,
                start: occurrence.start,
                end: occurrence.end,
                allDay: false,
                schedule,
            }));
            // Add events to result
            result.push(...events);
        });
        return result;
    }, [dateRange.end, dateRange.start, schedules, session]);


    const openEvent = useCallback((event: any) => {
        console.log("CalendarEvent clicked:", event);
    }, []);

    // Handle scheduling
    const [isScheduleDialogOpen, setIsScheduleDialogOpen] = useState(false);
    const [editingSchedule, setEditingSchedule] = useState<Schedule | null>(null);
    const handleAddSchedule = () => { setIsScheduleDialogOpen(true); };
    const handleUpdateSchedule = (schedule: Schedule) => {
        setEditingSchedule(schedule);
        setIsScheduleDialogOpen(true);
    };
    const handleCloseScheduleDialog = () => { setIsScheduleDialogOpen(false); };
    const handleScheduleCreated = (created: Schedule) => {
        //TODO
        setIsScheduleDialogOpen(false);
    };
    const handleScheduleUpdated = (updated: Schedule) => {
        //TODO
        setIsScheduleDialogOpen(false);
    };
    const handleDeleteSchedule = () => {
        //TODO
    };

    if (!localizer) return <FullPageSpinner />;
    return (
        <>
            {/* Dialog for creating/updating schedules */}
            <ScheduleDialog
                isCreate={editingSchedule === null}
                isMutate={true}
                isOpen={isScheduleDialogOpen}
                onClose={handleCloseScheduleDialog}
                onCreated={handleScheduleCreated}
                onUpdated={handleScheduleUpdated}
                zIndex={202}
            />
            {/* Add event button */}
            <SideActionButtons
                // Treat as a dialog when build view is open
                display={display}
                zIndex={201}
            >
                <ColorIconButton
                    aria-label="create event"
                    background={palette.secondary.main}
                    onClick={handleAddSchedule}
                    sx={{
                        padding: 0,
                        width: "54px",
                        height: "54px",
                    }}
                >
                    <AddIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                </ColorIconButton>
            </SideActionButtons>
            <TopBar
                ref={ref}
                display={display}
                onClose={() => { }}
                titleData={{ title: currTab.label }}
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
                views={["month", "week", "day"]}
                style={{
                    height: calendarHeight,
                    background: palette.background.paper,
                }}
            />
        </>
    );
};