import { calculateOccurrences, CalendarEvent, Schedule } from "@local/shared";
import { Box, Breakpoints, IconButton, Tooltip, useTheme } from "@mui/material";
import { SideActionsButtons } from "components/buttons/SideActionsButtons/SideActionsButtons";
import { FullPageSpinner } from "components/FullPageSpinner/FullPageSpinner";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { PageTabs } from "components/PageTabs/PageTabs";
import { SessionContext } from "contexts/SessionContext";
import { add, endOfMonth, format, getDay, startOfMonth, startOfWeek } from "date-fns";
import { useDimensions } from "hooks/useDimensions";
import { useFindMany } from "hooks/useFindMany";
import { useTabs } from "hooks/useTabs";
import { useWindowSize } from "hooks/useWindowSize";
import { AddIcon, ArrowLeftIcon, ArrowRightIcon, DayIcon, MonthIcon, TodayIcon, WeekIcon } from "icons";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { Calendar, dateFnsLocalizer, DateLocalizer, Navigate, Views } from "react-big-calendar";
import "react-big-calendar/lib/css/react-big-calendar.css";
import { useTranslation } from "react-i18next";
import { getCurrentUser } from "utils/authentication/session";
import { getDisplay } from "utils/display/listTools";
import { getShortenedLabel, getUserLanguages, getUserLocale, loadLocale } from "utils/display/translationTools";
import { CalendarPageTabOption, calendarTabParams } from "utils/search/objectToSearch";
import { ScheduleUpsert } from "views/objects/schedule";
import { CalendarViewProps } from "views/types";

const sectionStyle = (breakpoints: Breakpoints, spacing: any) => ({
    display: "flex",
    alignItems: "center",
    [breakpoints.down("sm")]: {
        width: "100%",
        justifyContent: "space-evenly",
        marginBottom: spacing(1),
    },
});

/**
 * Toolbar for changing calendar view and navigating between dates
 */
const CustomToolbar = (props) => {
    const { breakpoints, palette, spacing } = useTheme();
    const { t } = useTranslation();
    const { label, onView, view, onNavigate } = props;

    const navigate = (action) => {
        onNavigate(action);
    };

    const changeView = (nextView) => {
        onView(nextView);
    };

    return (
        <Box sx={{
            display: "flex",
            justifyContent: "space-between",
            alignItems: "center",
            padding: "0 16px",
            [breakpoints.down(400)]: {
                flexDirection: "column",
            },
        }}>
            <Box sx={sectionStyle(breakpoints, spacing)}>
                <Tooltip title={t("Today")}>
                    <IconButton onClick={() => navigate(Navigate.TODAY)}>
                        <TodayIcon fill={palette.secondary.main} />
                    </IconButton>
                </Tooltip>
                <Tooltip title={t("Previous")}>
                    <IconButton onClick={() => navigate(Navigate.PREVIOUS)}>
                        <ArrowLeftIcon fill={palette.secondary.main} />
                    </IconButton>
                </Tooltip>
                <Tooltip title={t("Next")}>
                    <IconButton onClick={() => navigate(Navigate.NEXT)}>
                        <ArrowRightIcon fill={palette.secondary.main} />
                    </IconButton>
                </Tooltip>
            </Box>

            <Box sx={sectionStyle(breakpoints, spacing)}>
                <span>{label}</span>
            </Box>

            <Box sx={sectionStyle(breakpoints, spacing)}>
                <Tooltip title={t("Month")}>
                    <IconButton onClick={() => changeView(Views.MONTH)}>
                        <MonthIcon fill={palette.secondary.main} />
                    </IconButton>
                </Tooltip>
                <Tooltip title={t("Week")}>
                    <IconButton onClick={() => changeView(Views.WEEK)}>
                        <WeekIcon fill={palette.secondary.main} />
                    </IconButton>
                </Tooltip>
                <Tooltip title={t("Day")}>
                    <IconButton onClick={() => changeView(Views.DAY)}>
                        <DayIcon fill={palette.secondary.main} />
                    </IconButton>
                </Tooltip>
            </Box>
        </Box >
    );
};

/**
 * Day header for Month view. Use
 */
const DayColumnHeader = ({ label }) => {
    const { breakpoints } = useTheme();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.sm);

    return (
        <Box sx={{
            textAlign: "center",
            fontWeight: "bold",
        }}>
            {isMobile ? getShortenedLabel(label) : label}
        </Box>
    );
};

export const CalendarView = ({
    display,
    onClose,
}: CalendarViewProps) => {
    const session = useContext(SessionContext);
    const { breakpoints, palette } = useTheme();
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
                const localeModule = await loadLocale(locale);

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
    const { dimensions, ref, refreshDimensions } = useDimensions();
    // Refresh dimensions once after the component is mounted
    useEffect(() => {
        setTimeout(refreshDimensions, 1000);
    }, [refreshDimensions]);
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);
    const calendarHeight = useMemo(() => `calc(100vh - ${dimensions.height}px - ${isMobile ? 56 : 0}px - env(safe-area-inset-bottom))`, [dimensions, isMobile]);

    const handleDateRangeChange = useCallback((range: Date[] | { start: Date, end: Date }) => {
        if (Array.isArray(range)) {
            setDateRange({ start: range[0], end: range[1] });
        } else {
            setDateRange(range);
        }
    }, []);

    const {
        currTab,
        handleTabChange,
        searchType,
        tabs,
        where,
    } = useTabs({ id: "calendar-tabs", tabParams: calendarTabParams, display });

    // Find schedules
    const {
        allData: schedules,
        loading,
        loadMore,
    } = useFindMany<Schedule>({
        searchType,
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
            ...where(),
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
    const [events, setEvents] = useState<CalendarEvent[]>([]);
    useEffect(() => {
        let isCancelled = false;

        const fetchEvents = async () => {
            if (!dateRange.start || !dateRange.end) {
                setEvents([]);
                return;
            }

            const result: CalendarEvent[] = [];
            for (const schedule of schedules) {
                const occurrences = await calculateOccurrences(schedule, dateRange.start!, dateRange.end!);
                const events: CalendarEvent[] = occurrences.map(occurrence => ({
                    __typename: "CalendarEvent",
                    id: `${schedule.id}|${occurrence.start.getTime()}|${occurrence.end.getTime()}`,
                    title: getDisplay(schedule, getUserLanguages(session)).title,
                    start: occurrence.start,
                    end: occurrence.end,
                    allDay: false,
                    schedule,
                }));
                if (!isCancelled) {
                    result.push(...events);
                }
            }
            if (!isCancelled) {
                setEvents(result);
            }
        };

        fetchEvents();

        return () => {
            isCancelled = true; // Cleanup function to avoid setting state on unmounted component
        };
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
    const handleScheduleCompleted = (created: Schedule) => {
        //TODO update schedule
        setIsScheduleDialogOpen(false);
    };
    const handleScheduleDeleted = (deleted: Schedule) => {
        //TODO delete schedule
        setIsScheduleDialogOpen(false);
    };

    const activeDayStyle = {
        background: palette.mode === "dark" ? palette.primary.main : undefined,
        color: palette.mode === "dark" ? palette.primary.contrastText : undefined,
    };
    const outOfRangeDayStyle = {
        background: palette.mode === "dark" ? palette.background.default : palette.grey[400],
    };

    const dayPropGetter = (date) => {
        const now = new Date();
        // Check if the date is the current day
        if (date.getDate() === now.getDate() &&
            date.getMonth() === now.getMonth() &&
            date.getFullYear() === now.getFullYear()) {
            return {
                style: activeDayStyle,
            };
        }
        // If the date is not in the current month, apply the outOfRangeDayStyle
        if (dateRange.start && dateRange.end) {
            const midRange = new Date((dateRange.start.getTime() + dateRange.end.getTime()) / 2);
            if (date.getMonth() !== midRange.getMonth()) {
                return {
                    style: outOfRangeDayStyle,
                };
            }
        }
        return {};
    };

    if (!localizer) return <FullPageSpinner />;
    return (
        <Box sx={{ maxHeight: "100vh", overflow: "hidden" }}>
            <ScheduleUpsert
                canChangeTab
                canSetScheduleFor
                defaultTab={currTab.key === "All" ? CalendarPageTabOption.Meeting : currTab.key}
                display="dialog"
                isCreate={editingSchedule === null}
                isMutate={true}
                isOpen={isScheduleDialogOpen}
                onCancel={handleCloseScheduleDialog}
                onClose={handleCloseScheduleDialog}
                onCompleted={handleScheduleCompleted}
                onDeleted={handleScheduleDeleted}
                overrideObject={editingSchedule ?? { __typename: "Schedule" }}
            />
            <TopBar
                ref={ref}
                display={display}
                hideTitleOnDesktop
                onClose={onClose}
                title={t("Schedule", { count: 1 })}
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
                components={{
                    toolbar: CustomToolbar,
                    month: {
                        header: DayColumnHeader,
                    },
                }}
                dayPropGetter={dayPropGetter}
                style={{
                    height: calendarHeight,
                    maxHeight: calendarHeight,
                    background: palette.background.paper,
                }}
            />
            {/* Add event button */}
            <SideActionsButtons display={display}>
                <IconButton
                    aria-label={t("CreateEvent")}
                    onClick={handleAddSchedule}
                    sx={{
                        background: palette.secondary.main,
                        padding: 0,
                        width: "54px",
                        height: "54px",
                    }}
                >
                    <AddIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                </IconButton>
            </SideActionsButtons>
        </Box>
    );
};
