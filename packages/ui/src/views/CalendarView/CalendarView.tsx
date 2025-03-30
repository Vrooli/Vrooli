import { calculateOccurrences, CalendarEvent, Schedule } from "@local/shared";
import { Box, BoxProps, IconButton, styled, Tooltip, useTheme } from "@mui/material";
import { add, endOfMonth, format, getDay, startOfMonth, startOfWeek } from "date-fns";
import { useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { Calendar, HeaderProps as CalendarHeaderProps, ToolbarProps as CalendarToolbarProps, dateFnsLocalizer, DateLocalizer, Navigate, SlotInfo, View, Views } from "react-big-calendar";
import "react-big-calendar/lib/css/react-big-calendar.css";
import { useTranslation } from "react-i18next";
import { SideActionsButtons } from "../../components/buttons/SideActionsButtons/SideActionsButtons.js";
import { TopBar } from "../../components/navigation/TopBar.js";
import { PageTabs } from "../../components/PageTabs/PageTabs.js";
import { FullPageSpinner } from "../../components/Spinners.js";
import { SessionContext } from "../../contexts/session.js";
import { useFindMany } from "../../hooks/useFindMany.js";
import { useTabs } from "../../hooks/useTabs.js";
import { useWindowSize } from "../../hooks/useWindowSize.js";
import { IconCommon } from "../../icons/Icons.js";
import { bottomNavHeight } from "../../styles.js";
import { PartialWithType } from "../../types.js";
import { getCurrentUser } from "../../utils/authentication/session.js";
import { getDisplay } from "../../utils/display/listTools.js";
import { getShortenedLabel, getUserLanguages, getUserLocale, loadLocale } from "../../utils/display/translationTools.js";
import { calendarTabParams } from "../../utils/search/objectToSearch.js";
import { ScheduleUpsert } from "../objects/schedule/ScheduleUpsert.js";
import { ScheduleForType } from "../objects/schedule/types.js";
import { CalendarViewProps } from "../types.js";

const views = {
    month: true,
    week: true,
    day: true,
} as const;

const DEFAULT_START_HOUR = 9;
const DEFAULT_DURATION_HOURS = 1;

const ToolbarBox = styled(Box)(({ theme }) => ({
    display: "flex",
    justifyContent: "space-between",
    alignItems: "center",
    padding: "0 16px",
    // eslint-disable-next-line no-magic-numbers
    [theme.breakpoints.down(400)]: {
        flexDirection: "column",
    },
}));

const ToolbarSection = styled(Box)(({ theme }) => ({
    display: "flex",
    alignItems: "center",
    [theme.breakpoints.down("sm")]: {
        width: "100%",
        justifyContent: "space-evenly",
        marginBottom: theme.spacing(1),
    },
}));

const dateLabelBoxStyle = {
    cursor: "pointer",
    position: "relative",
    display: "inline-block",
} as const;

const dateInputStyle = {
    position: "absolute",
    top: 0,
    left: 0,
    width: "100%",
    height: "100%",
    opacity: 0,
    cursor: "pointer",
} as const;

type CustomToolbarProps = CalendarToolbarProps & {
    onSelectDate: (date: Date) => unknown;
}

/**
 * Toolbar for changing calendar view and navigating between dates
 */
function CustomToolbar({
    date,
    label,
    onNavigate,
    onSelectDate,
    onView,
}: CustomToolbarProps) {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const dateInputRef = useRef<HTMLInputElement>(null);

    const handleDateChange = useCallback(function handleDateChangeCallback(event: React.ChangeEvent<HTMLInputElement>) {
        const inputDate = event.target.value; // This is in YYYY-MM-DD format
        const [year, month, day] = inputDate.split("-").map(Number);
        const newDate = new Date(year, month - 1, day); // month is 0-indexed in Date constructor

        if (!isNaN(newDate.getTime())) {
            onNavigate(Navigate.DATE, newDate);
            onSelectDate(newDate);
        }
    }, [onNavigate, onSelectDate]);

    const openDatePicker = useCallback(function openDatePickerCallback() {
        if (dateInputRef.current) {
            dateInputRef.current.showPicker();
        }
    }, []);

    const navigate = useCallback(function navigateCallback(action) {
        onNavigate(action);
    }, [onNavigate]);

    const changeView = useCallback(function changeViewCallback(nextView) {
        onView(nextView);
    }, [onView]);

    function toToday() {
        navigate(Navigate.TODAY);
    }
    function toPrevious() {
        navigate(Navigate.PREVIOUS);
    }
    function toNext() {
        navigate(Navigate.NEXT);
    }

    function toMonth() {
        changeView(Views.MONTH);
    }
    function toWeek() {
        changeView(Views.WEEK);
    }
    function toDay() {
        changeView(Views.DAY);
    }

    return (
        <ToolbarBox>
            <ToolbarSection>
                <Tooltip title={t("Today")}>
                    <IconButton onClick={toToday}>
                        <IconCommon
                            decorative
                            fill={palette.secondary.main}
                            name="Today"
                        />
                    </IconButton>
                </Tooltip>
                <Tooltip title={t("Previous")}>
                    <IconButton onClick={toPrevious}>
                        <IconCommon
                            decorative
                            fill={palette.secondary.main}
                            name="ArrowLeft"
                        />
                    </IconButton>
                </Tooltip>
                <Tooltip title={t("Next")}>
                    <IconButton onClick={toNext}>
                        <IconCommon
                            decorative
                            fill={palette.secondary.main}
                            name="ArrowRight"
                        />
                    </IconButton>
                </Tooltip>
            </ToolbarSection>

            <ToolbarSection>
                <Box
                    onClick={openDatePicker}
                    sx={dateLabelBoxStyle}
                >
                    {label}
                    <input
                        ref={dateInputRef}
                        type="date"
                        value={date.toISOString().split("T")[0]}
                        onChange={handleDateChange}
                        style={dateInputStyle}
                    />
                </Box>
            </ToolbarSection>
            <ToolbarSection>
                <Tooltip title={t("Month")}>
                    <IconButton onClick={toMonth}>
                        <IconCommon
                            decorative
                            fill={palette.secondary.main}
                            name="Month"
                        />
                    </IconButton>
                </Tooltip>
                <Tooltip title={t("Week")}>
                    <IconButton onClick={toWeek}>
                        <IconCommon
                            decorative
                            fill={palette.secondary.main}
                            name="Week"
                        />
                    </IconButton>
                </Tooltip>
                <Tooltip title={t("Day")}>
                    <IconButton onClick={toDay}>
                        <IconCommon
                            decorative
                            fill={palette.secondary.main}
                            name="Day"
                        />
                    </IconButton>
                </Tooltip>
            </ToolbarSection>
        </ToolbarBox>
    );
}

const dayColumnHeaderBoxStyle = {
    textAlign: "center",
    fontWeight: "bold",
} as const;

type DayColumnHeaderProps = {
    isBottomNavVisible: boolean;
    label: string;
}

/**
 * Day header for Month view. Use
 */
function DayColumnHeader({
    isBottomNavVisible,
    label,
}: DayColumnHeaderProps) {
    return (
        <Box sx={dayColumnHeaderBoxStyle}>
            {isBottomNavVisible ? getShortenedLabel(label) : label}
        </Box>
    );
}

interface FlexContainerProps extends BoxProps {
    isBottomNavVisible: boolean;
}

const FlexContainer = styled(Box, {
    shouldForwardProp: (prop) => prop !== "isBottomNavVisible",
})<FlexContainerProps>(({ isBottomNavVisible }) => ({
    display: "flex",
    flexDirection: "column",
    height: `calc(100vh - ${isBottomNavVisible ? bottomNavHeight : "0px"} - env(safe-area-inset-bottom))`,

}));

const outerBoxStyle = {
    maxHeight: "100vh",
    overflow: "hidden",
} as const;

export function CalendarView({
    display,
    onClose,
}: CalendarViewProps) {
    const session = useContext(SessionContext);
    const { breakpoints, palette } = useTheme();
    const isBottomNavVisible = useWindowSize(({ width }) => width <= breakpoints.values.md);
    const { t } = useTranslation();
    const locale = useMemo(() => getUserLocale(session), [session]);
    const [localizer, setLocalizer] = useState<DateLocalizer | null>(null);

    // Defaults to current month
    const [dateRange, setDateRange] = useState<{ start: Date, end: Date }>({
        start: startOfMonth(new Date()),
        end: endOfMonth(new Date()),
    });
    const [view, setView] = useState<View>(Views.MONTH);
    const handleViewChange = useCallback((nextView: View) => {
        setView(nextView);
    }, []);

    const [selectedDateTime, setSelectedDateTime] = useState<Date | null>(null);
    const handleSelectSlot = useCallback((slot: SlotInfo) => {
        console.log("Slot selected:", slot);
        setSelectedDateTime(slot.start);
    }, []);
    const handleSelectDate = useCallback((date: Date) => {
        setSelectedDateTime(date);
    }, []);

    useEffect(() => {
        async function localeLoader() {
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
        }

        localeLoader();
    }, [locale]);

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
            ...where(undefined),
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

        async function fetchEvents() {
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
        }

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
    const handleAddSchedule = useCallback(function handleAddScheduleCallback() {
        setIsScheduleDialogOpen(true);
    }, []);
    const handleUpdateSchedule = useCallback(function handleUpdateScheduleCallback(schedule: Schedule) {
        setEditingSchedule(schedule);
        setIsScheduleDialogOpen(true);
    }, []);
    const handleCloseScheduleDialog = useCallback(function handleCloseScheduleDialogCallback() {
        setIsScheduleDialogOpen(false);
    }, []);
    const handleScheduleCompleted = useCallback(function handleScheduleCompletedCallback(created: Schedule) {
        //TODO update schedule
        setIsScheduleDialogOpen(false);
    }, []);
    const handleScheduleDeleted = useCallback(function handleScheduleDeletedCallback(deleted: Schedule) {
        //TODO delete schedule
        setIsScheduleDialogOpen(false);
    }, []);

    const activeDayStyle = useMemo(function activeDayStyleMemo() {
        return {
            background: palette.mode === "dark" ? palette.primary.main : undefined,
            color: palette.mode === "dark" ? palette.primary.contrastText : undefined,
        } as const;
    }, [palette.mode, palette.primary.contrastText, palette.primary.main]);
    const outOfRangeDayStyle = useMemo(function outOfRangeDayStyleMemo() {
        return {
            background: palette.mode === "dark" ? palette.background.default : palette.grey[400],
        } as const;
    }, [palette.background.default, palette.grey, palette.mode]);

    const dayPropGetter = useCallback(function dayPropGetterCallback(date: Date) {
        const now = new Date();
        let style: React.CSSProperties = { cursor: "pointer " };
        // Handle styling for the current date
        const isCurrentDate = date.getDate() === now.getDate() && date.getMonth() === now.getMonth() && date.getFullYear() === now.getFullYear();
        if (isCurrentDate) {
            style = { ...style, ...activeDayStyle };
        }
        // Handle styling for selected date. This only applies for the "month" view
        const isSelectedDate = selectedDateTime && date.toDateString() === selectedDateTime.toDateString();
        if (isSelectedDate && view === "month") {
            style = { ...style, border: `2px solid ${palette.secondary.main}` };
        }
        // Handle styling for dates outside of the current month
        if (dateRange.start && dateRange.end) {
            const midRange = new Date((dateRange.start.getTime() + dateRange.end.getTime()) / 2);
            if (date.getMonth() !== midRange.getMonth()) {
                style = { ...style, ...outOfRangeDayStyle };
            }
        }
        return { style };
    }, [activeDayStyle, dateRange.end, dateRange.start, outOfRangeDayStyle, palette.secondary.main, selectedDateTime, view]);

    const slotPropGetter = useCallback((date: Date) => {
        let style: React.CSSProperties = {};
        // Handle selected hour styling for the day and week views
        if (selectedDateTime && view !== "month") {
            const selectedHour = selectedDateTime.getHours();
            const slotHour = date.getHours();
            const isSameDay = date.toDateString() === selectedDateTime.toDateString();

            if (isSameDay && slotHour === selectedHour) {
                style = {
                    ...style,
                    backgroundColor: palette.secondary.light,
                    borderLeft: `4px solid ${palette.secondary.main}`,
                };
            }
        }

        return { style };
    }, [selectedDateTime, view, palette.secondary]);

    const calendarComponents = useMemo(function calendarComponentsMemo() {
        return {
            toolbar: (props: CalendarToolbarProps) => <CustomToolbar {...props} onSelectDate={handleSelectDate} />,
            month: {
                header: (props: CalendarHeaderProps) => <DayColumnHeader isBottomNavVisible={isBottomNavVisible} {...props} />,
            },
        };
    }, [handleSelectDate, isBottomNavVisible]);

    const calendarStyle = useMemo(function calendarStyleMemo() {
        return {
            background: palette.background.paper,
            flexGrow: 1,
        } as const;
    }, [palette.background.paper]);

    const scheduleOverrideObject = useMemo(function scheduleOverrideObjectMemo() {
        if (editingSchedule) {
            return editingSchedule;
        }
        const defaultSchedule: PartialWithType<Schedule> = { __typename: "Schedule" } as const;
        if (selectedDateTime) {
            const startDate = new Date(selectedDateTime);
            const endDate = new Date(selectedDateTime);

            if (view === "month") {
                startDate.setHours(DEFAULT_START_HOUR, 0, 0, 0);
                endDate.setHours(DEFAULT_START_HOUR + DEFAULT_DURATION_HOURS, 0, 0, 0);
            } else if (view === "week") {
                endDate.setHours(startDate.getHours() + DEFAULT_DURATION_HOURS);
            } else {
                endDate.setHours(startDate.getHours() + DEFAULT_DURATION_HOURS);
            }

            defaultSchedule.startTime = startDate.toISOString();
            defaultSchedule.endTime = endDate.toISOString();
        }
        return defaultSchedule;
    }, [editingSchedule, selectedDateTime, view]);

    if (!localizer) return <FullPageSpinner />;
    return (
        <Box sx={outerBoxStyle}>
            <FlexContainer isBottomNavVisible={isBottomNavVisible}>
                <ScheduleUpsert
                    canSetScheduleFor={true}
                    defaultScheduleFor={currTab.key === "All" ? "Meeting" : currTab.key as ScheduleForType}
                    display="dialog"
                    isCreate={editingSchedule === null}
                    isMutate={true}
                    isOpen={isScheduleDialogOpen}
                    onCancel={handleCloseScheduleDialog}
                    onClose={handleCloseScheduleDialog}
                    onCompleted={handleScheduleCompleted}
                    onDeleted={handleScheduleDeleted}
                    overrideObject={scheduleOverrideObject}
                />
                <TopBar
                    // ref={ref}
                    display={display}
                    onClose={onClose}
                    title={t("Schedule", { count: 1 })}
                    titleBehaviorDesktop="ShowIn"
                    below={<PageTabs<typeof calendarTabParams>
                        ariaLabel="calendar-tabs"
                        currTab={currTab}
                        fullWidth
                        onChange={handleTabChange}
                        tabs={tabs}
                    />}
                />
                {/* TODO Remove when weird type error is fixed */}
                {/* @ts-expect-error Incompatible JSX type definitions */}
                <Calendar
                    localizer={localizer}
                    longPressThreshold={20}
                    events={events}
                    onRangeChange={handleDateRangeChange}
                    onSelectEvent={openEvent}
                    onSelectSlot={handleSelectSlot}
                    onView={handleViewChange}
                    startAccessor="start"
                    endAccessor="end"
                    components={calendarComponents}
                    dayPropGetter={dayPropGetter}
                    selectable={true}
                    slotPropGetter={slotPropGetter}
                    style={calendarStyle}
                    view={view}
                    views={views}
                />
            </FlexContainer>
            <SideActionsButtons display={display}>
                <IconButton
                    aria-label={t("CreateEvent")}
                    onClick={handleAddSchedule}
                >
                    <IconCommon name="Add" />
                </IconButton>
            </SideActionsButtons>
        </Box>
    );
}
