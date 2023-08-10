import { calculateOccurrences, CommonKey, Schedule, ScheduleFor, ScheduleSearchResult } from "@local/shared";
import { Box, Breakpoints, IconButton, Tooltip, useTheme } from "@mui/material";
import { ColorIconButton } from "components/buttons/ColorIconButton/ColorIconButton";
import { SideActionButtons } from "components/buttons/SideActionButtons/SideActionButtons";
import { LargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { FullPageSpinner } from "components/FullPageSpinner/FullPageSpinner";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { PageTabs } from "components/PageTabs/PageTabs";
import { PageTab } from "components/types";
import { add, endOfMonth, format, getDay, startOfMonth, startOfWeek } from "date-fns";
import { AddIcon, ArrowLeftIcon, ArrowRightIcon, DayIcon, FocusModeIcon, MonthIcon, OrganizationIcon, ProjectIcon, RoutineIcon, TodayIcon, VisibleIcon, WeekIcon } from "icons";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { Calendar, dateFnsLocalizer, DateLocalizer, Navigate, Views } from "react-big-calendar";
import "react-big-calendar/lib/css/react-big-calendar.css";
import { useTranslation } from "react-i18next";
import { addSearchParams, parseSearchParams, useLocation } from "route";
import { CalendarEvent, SvgComponent } from "types";
import { getCurrentUser } from "utils/authentication/session";
import { getDisplay } from "utils/display/listTools";
import { getShortenedLabel, getUserLanguages, getUserLocale, loadLocale } from "utils/display/translationTools";
import { useDimensions } from "utils/hooks/useDimensions";
import { useFindMany } from "utils/hooks/useFindMany";
import { useWindowSize } from "utils/hooks/useWindowSize";
import { CalendarPageTabOption } from "utils/search/objectToSearch";
import { SessionContext } from "utils/SessionContext";
import { ScheduleUpsert } from "views/objects/schedule";
import { CalendarViewProps } from "views/types";

// Tab data type
type CalendarBaseParams = {
    Icon: SvgComponent;
    titleKey: CommonKey;
    tabType: CalendarPageTabOption;
    filterType?: ScheduleFor;
}

type CalendarPageTab = PageTab<CalendarPageTabOption> & { filterType: ScheduleFor | undefined };

// Data for each tab. Ordered by tab index
export const calendarTabParams: CalendarBaseParams[] = [{
    Icon: VisibleIcon,
    titleKey: "All",
    tabType: CalendarPageTabOption.All,
}, {
    Icon: OrganizationIcon,
    titleKey: "Meeting",
    tabType: CalendarPageTabOption.Meetings,
    filterType: ScheduleFor.Meeting,
}, {
    Icon: RoutineIcon,
    titleKey: "Routine",
    tabType: CalendarPageTabOption.RunRoutines,
    filterType: ScheduleFor.RunRoutine,
}, {
    Icon: ProjectIcon,
    titleKey: "Project",
    tabType: CalendarPageTabOption.RunProjects,
    filterType: ScheduleFor.RunProject,
}, {
    Icon: FocusModeIcon,
    titleKey: "FocusMode",
    tabType: CalendarPageTabOption.FocusModes,
    filterType: ScheduleFor.FocusMode,
}];

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
    display = "page",
    onClose,
    zIndex,
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

    // Handle tabs
    const tabs = useMemo<CalendarPageTab[]>(() => {
        return calendarTabParams.map((tab, i) => ({
            index: i,
            Icon: tab.Icon,
            label: t(tab.titleKey, { count: 2, defaultValue: tab.titleKey }),
            value: tab.tabType,
            filterType: tab.filterType,
        }));
    }, [t]);
    const [currTab, setCurrTab] = useState<CalendarPageTab>(() => {
        const searchParams = parseSearchParams();
        const index = calendarTabParams.findIndex(tab => tab.tabType === searchParams.type);
        // Default to bookmarked tab
        if (index === -1) return tabs[0];
        // Return tab
        return tabs[index];
    });
    const handleTabChange = useCallback((e: any, tab: CalendarPageTab) => {
        e.preventDefault();
        // Update search params
        addSearchParams(setLocation, { type: tab.value });
        // Update curr tab
        setCurrTab(tab);
    }, [setLocation]);

    // Find schedules
    const {
        allData: schedules,
        loading,
        loadMore,
    } = useFindMany<ScheduleSearchResult>({
        canSearch: () => true,
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
            scheduleFor: currTab.filterType,
            scheduleForUserId: getCurrentUser(session)?.id,
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
        console.log("calculating events...", schedules);
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
    const handleScheduleCompleted = (created: Schedule) => {
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
            <LargeDialog
                id="schedule-dialog"
                onClose={handleCloseScheduleDialog}
                isOpen={isScheduleDialogOpen}
                titleId={""}
                zIndex={zIndex + 2}
            >
                <ScheduleUpsert
                    defaultTab={currTab.value === "All" ? CalendarPageTabOption.Meetings : currTab.value}
                    handleDelete={handleDeleteSchedule}
                    isCreate={editingSchedule === null}
                    isMutate={true}
                    onCancel={handleCloseScheduleDialog}
                    onCompleted={handleScheduleCompleted}
                    overrideObject={editingSchedule ?? { __typename: "Schedule" }}
                    zIndex={zIndex + 1002}
                />
            </LargeDialog>
            {/* Add event button */}
            <SideActionButtons
                // Treat as a dialog when build view is open
                display={display}
                zIndex={zIndex + 1}
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
                onClose={onClose}
                title={currTab.label}
                below={<PageTabs
                    ariaLabel="calendar-tabs"
                    currTab={currTab}
                    fullWidth
                    onChange={handleTabChange}
                    tabs={tabs}
                />}
                zIndex={zIndex}
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
                style={{
                    height: calendarHeight,
                    maxHeight: calendarHeight,
                    background: palette.background.paper,
                }}
            />
        </>
    );
};
