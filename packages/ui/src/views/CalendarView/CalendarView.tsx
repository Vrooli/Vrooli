import { calculateOccurrences, CalendarEvent, Schedule, ScheduleFor } from "@local/shared";
import { Box, BoxProps, Button, Checkbox, DialogContent, Divider, FormControlLabel, FormGroup, IconButton, InputAdornment, List, ListItem, ListItemText, Paper, styled, Tab, Tabs, TextField, Tooltip, Typography, useTheme } from "@mui/material";
import { add, endOfMonth, format, getDay, startOfMonth, startOfWeek } from "date-fns";
import { useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { Calendar, HeaderProps as CalendarHeaderProps, ToolbarProps as CalendarToolbarProps, Components, dateFnsLocalizer, DateLocalizer, Navigate, SlotInfo, View, Views } from "react-big-calendar";
import "react-big-calendar/lib/css/react-big-calendar.css";
import { useTranslation } from "react-i18next";
import { SideActionsButtons } from "../../components/buttons/SideActionsButtons.js";
import { LargeDialog } from "../../components/dialogs/LargeDialog/LargeDialog.js";
import { useIsBottomNavVisible } from "../../components/navigation/BottomNav.js";
import { APP_BAR_HEIGHT_PX, Navbar } from "../../components/navigation/Navbar.js";
import { FullPageSpinner } from "../../components/Spinners.js";
import { SessionContext } from "../../contexts/session.js";
import { useFindMany } from "../../hooks/useFindMany.js";
import { useHistoryState } from "../../hooks/useHistoryState.js";
import { IconCommon } from "../../icons/Icons.js";
import { bottomNavHeight } from "../../styles.js";
import { PartialWithType } from "../../types.js";
import { getCurrentUser } from "../../utils/authentication/session.js";
import { getDisplay } from "../../utils/display/listTools.js";
import { getShortenedLabel, getUserLanguages, getUserLocale, loadLocale } from "../../utils/display/translationTools.js";
import { ScheduleUpsert } from "../objects/schedule/ScheduleUpsert.js";
import { CalendarViewProps } from "../types.js";

// Define the tab values
export enum CalendarTabs {
    CALENDAR = "calendar",
    TRIGGER = "trigger",
}

// Define schedule types for filtering using the ScheduleFor enum
const SCHEDULE_TYPES: ScheduleFor[] = [
    ScheduleFor.FocusMode,
    ScheduleFor.Meeting,
    ScheduleFor.RunProject,
    ScheduleFor.RunRoutine,
];

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
    height: `calc(100vh - ${isBottomNavVisible ? bottomNavHeight : "0px"} - ${APP_BAR_HEIGHT_PX}px - env(safe-area-inset-bottom))`,
}));

const outerBoxStyle = {
    maxHeight: "100vh",
    overflow: "hidden",
} as const;

// Define styles and props outside the component for performance
const searchInputAdornment = (
    <InputAdornment position="start">
        <IconCommon name="Search" />
    </InputAdornment>
);
const searchInputProps = { startAdornment: searchInputAdornment };
const filterDialogDividerSx = { my: 1 };
const filterSectionPaperSx = {
    p: 2,
    flex: 1,
    display: "flex",
    flexDirection: "column",
    height: "100%",
    overflowY: "auto",
};
const resultsListSx = {
    flexGrow: 1,
};
const noResultsTextProps = { color: "text.secondary" };
const triggerViewBoxSx = {
    display: "flex",
    flexDirection: "column",
    alignItems: "center",
    justifyContent: "center",
    height: "100%",
    p: 3,
};
const triggerViewIconProps = { name: "History" as const, size: 64, fill: "text.secondary" };
const triggerViewTitleSx = { mt: 2 };
const triggerViewDescSx = { mt: 1, textAlign: "center", maxWidth: 600 };

/**
 * Filter dialog component for filtering events
 */
interface FilterDialogProps {
    isOpen: boolean;
    onClose: () => void;
    searchQuery: string;
    setSearchQuery: (query: string) => void;
    selectedTypes: ScheduleFor[];
    setSelectedTypes: (updater: (prev: ScheduleFor[]) => ScheduleFor[]) => void;
    filteredEvents: CalendarEvent[];
    activeTab: CalendarTabs;
    onAddNew: () => void;
}

function FilterDialog({
    isOpen,
    onClose,
    searchQuery,
    setSearchQuery,
    selectedTypes,
    setSelectedTypes,
    filteredEvents,
    activeTab,
    onAddNew,
}: FilterDialogProps) {
    const dialogId = "filter-dialog";
    const { t } = useTranslation();

    const handleSearchChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
        setSearchQuery(e.target.value);
    }, [setSearchQuery]);

    const handleSelectAll = useCallback(() => {
        if (selectedTypes.length === SCHEDULE_TYPES.length) {
            setSelectedTypes(() => []);
        } else {
            setSelectedTypes(() => [...SCHEDULE_TYPES]);
        }
    }, [selectedTypes.length, setSelectedTypes]);

    // Memoize the handler creation for each type checkbox
    const handleTypeChange = useCallback((type: ScheduleFor) => () => {
        setSelectedTypes(prev => {
            if (prev.includes(type)) {
                return prev.filter(t => t !== type);
            } else {
                return [...prev, type];
            }
        });
    }, [setSelectedTypes]);

    return (
        <LargeDialog
            isOpen={isOpen}
            onClose={onClose}
            id={dialogId}
            sxs={{
                paper: {
                    width: "min(100%, 800px)",
                },
                content: {
                    height: "100%",
                },
            }}
            titleId={`${dialogId}-title`}
        >
            <DialogContent sx={{ display: "flex", flexDirection: "column" }}>
                <TextField
                    fullWidth
                    placeholder={activeTab === CalendarTabs.CALENDAR
                        ? t("FindScheduledEvents", { defaultValue: "Find scheduled events..." })
                        : t("FindTriggeredEvents", { defaultValue: "Find triggered events..." })}
                    value={searchQuery}
                    onChange={handleSearchChange}
                    margin="normal"
                    variant="outlined"
                    InputProps={searchInputProps}
                />

                <Box display="flex" mt={2} gap={2} sx={{ flexGrow: 1, alignItems: "stretch", overflow: "hidden" }}>
                    <Box width="50%" pr={1}>
                        <Typography variant="subtitle1" gutterBottom>
                            {t("Option", { count: 2, defaultValue: "Options" })}
                        </Typography>

                        {activeTab === CalendarTabs.CALENDAR && (
                            <Paper elevation={1} sx={filterSectionPaperSx}>
                                <FormGroup>
                                    <FormControlLabel
                                        control={
                                            <Checkbox
                                                color="secondary"
                                                checked={selectedTypes.length === SCHEDULE_TYPES.length}
                                                indeterminate={selectedTypes.length > 0 && selectedTypes.length < SCHEDULE_TYPES.length}
                                                onChange={handleSelectAll}
                                            />
                                        }
                                        label={t("All", { defaultValue: "All" })}
                                    />
                                    <Divider sx={filterDialogDividerSx} />
                                    {SCHEDULE_TYPES.map((type) => (
                                        <FormControlLabel
                                            key={type}
                                            control={
                                                <Checkbox
                                                    color="secondary"
                                                    checked={selectedTypes.includes(type)}
                                                    onChange={handleTypeChange(type)}
                                                />
                                            }
                                            label={t(type, { defaultValue: type })}
                                        />
                                    ))}
                                </FormGroup>
                            </Paper>
                        )}

                        {activeTab === CalendarTabs.TRIGGER && (
                            <Paper elevation={1} sx={filterSectionPaperSx}>
                                <Typography variant="body2" color="text.secondary">
                                    {t("ComingSoon", { defaultValue: "Coming Soon" })}
                                </Typography>
                            </Paper>
                        )}
                    </Box>

                    <Box width="50%" pl={1}>
                        <Typography variant="subtitle1" gutterBottom>
                            {t("Result", { count: filteredEvents.length, defaultValue: "Results" })} ({filteredEvents.length})
                        </Typography>
                        <Paper elevation={1} sx={filterSectionPaperSx}>
                            <List dense sx={resultsListSx}>
                                {filteredEvents.length > 0 ? (
                                    filteredEvents.map((event) => (
                                        <ListItem key={event.id}>
                                            <ListItemText
                                                primary={event.title}
                                                secondary={`${format(event.start, "PPp")} - ${format(event.end, "PPp")}`}
                                            />
                                        </ListItem>
                                    ))
                                ) : (
                                    <ListItem>
                                        <Box sx={{ display: "flex", flexDirection: "column", alignItems: "center", justifyContent: "center", height: "100%", width: "100%", textAlign: "center", p: 2 }}>
                                            <ListItemText
                                                primary={t("NoResults", { defaultValue: "No results found" })}
                                                primaryTypographyProps={noResultsTextProps}
                                            />
                                            <Button
                                                variant="outlined"
                                                color="secondary"
                                                onClick={onAddNew}
                                                startIcon={<IconCommon name="Add" />}
                                                sx={{ mt: 2 }}
                                            >
                                                {t("AddEvent", { defaultValue: "Add Event" })}
                                            </Button>
                                        </Box>
                                    </ListItem>
                                )}
                            </List>
                        </Paper>
                    </Box>
                </Box>
            </DialogContent>
        </LargeDialog>
    );
}

/**
 * Simple trigger tab component
 */
function TriggerView() {
    const { t } = useTranslation();

    return (
        <Box sx={triggerViewBoxSx}>
            <IconCommon {...triggerViewIconProps} />
            <Typography variant="h5" sx={triggerViewTitleSx}>
                {t("ComingSoon", { defaultValue: "Coming Soon" })}
            </Typography>
            <Typography variant="body1" color="text.secondary" sx={triggerViewDescSx}>
                {t("TriggerDescription", { defaultValue: "Trigger functionality is coming soon!" })}
            </Typography>
        </Box>
    );
}

// Update the CalendarViewProps interface
export interface CalendarViewOwnProps {
    /**
     * The initial tab to display (for testing purposes)
     */
    initialTab?: CalendarTabs;
}

export function CalendarView({
    display,
    initialTab = CalendarTabs.CALENDAR,
}: CalendarViewProps & CalendarViewOwnProps) {
    const session = useContext(SessionContext);
    const { palette, typography } = useTheme();
    const isBottomNavVisible = useIsBottomNavVisible();
    const { t } = useTranslation();
    const locale = useMemo(() => getUserLocale(session), [session]);
    const [localizer, setLocalizer] = useState<DateLocalizer | null>(null);
    const { palette: themePalette } = useTheme();

    // Active tab state
    const [activeTab, setActiveTab] = useHistoryState("calendar-tab", initialTab);
    const handleTabChange = useCallback((_event: React.SyntheticEvent, newValue: CalendarTabs) => {
        setActiveTab(newValue);
    }, [setActiveTab]);

    // Filter dialog state
    const [isFilterDialogOpen, setIsFilterDialogOpen] = useState(false);
    const openFilterDialog = useCallback(() => setIsFilterDialogOpen(true), []);
    const closeFilterDialog = useCallback(() => setIsFilterDialogOpen(false), []);
    const [searchQuery, setSearchQuery] = useState("");
    const [selectedTypes, setSelectedTypes] = useState<ScheduleFor[]>([...SCHEDULE_TYPES]);

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

    // Find schedules
    const {
        allData: schedules,
        loading,
        loadMore,
    } = useFindMany<Schedule>({
        searchType: "Schedule",
        where: {
            // Only find schedules that have not ended, 
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

            console.log("Fetching events with schedules:", schedules);
            console.log("Date range:", dateRange);

            const result: CalendarEvent[] = [];
            for (const schedule of schedules) {
                console.log("Processing schedule:", schedule);
                try {
                    const occurrences = await calculateOccurrences(schedule, dateRange.start!, dateRange.end!);
                    console.log("Generated occurrences:", occurrences);

                    // Try to get a display title
                    let title = "Untitled";
                    try {
                        title = getDisplay(schedule, getUserLanguages(session)).title || "Untitled";
                    } catch (displayError) {
                        console.error("Error getting display title:", displayError);
                        // Try to extract title/name from available data
                        if (schedule.meetings?.[0]?.name) {
                            title = schedule.meetings[0].name;
                        } else if (schedule.meetings?.[0]?.translations?.[0]?.title) {
                            title = schedule.meetings[0].translations[0].title;
                        } else if (schedule.meetings?.[0]?.translations?.[0]?.name) {
                            title = schedule.meetings[0].translations[0].name;
                        } else if (schedule.focusModes?.[0]?.name) {
                            title = schedule.focusModes[0].name;
                        } else if (schedule.runProjects?.[0]?.name) {
                            title = schedule.runProjects[0].name;
                        } else if (schedule.runRoutines?.[0]?.name) {
                            title = schedule.runRoutines[0].name;
                        }
                    }

                    const events: CalendarEvent[] = occurrences.map(occurrence => ({
                        __typename: "CalendarEvent",
                        id: `${schedule.id}|${occurrence.start.getTime()}|${occurrence.end.getTime()}`,
                        title,
                        start: occurrence.start,
                        end: occurrence.end,
                        allDay: false,
                        schedule,
                    }));

                    if (!isCancelled) {
                        result.push(...events);
                    }
                } catch (error) {
                    console.error("Error calculating occurrences:", error);
                }
            }

            console.log("Final events for calendar:", result);

            if (!isCancelled) {
                setEvents(result);
            }
        }

        fetchEvents();

        return () => {
            isCancelled = true; // Cleanup function to avoid setting state on unmounted component
        };
    }, [dateRange, dateRange.end, dateRange.start, schedules, session]);

    // Filter events based on search query and selected types
    const filteredEvents = useMemo(() => {
        return events.filter(event => {
            // Filter by search query
            const matchesSearch = searchQuery.trim() === "" ||
                event.title.toLowerCase().includes(searchQuery.toLowerCase());

            // Filter by selected types
            const scheduleType = event.schedule?.scheduleFor as ScheduleFor | undefined;
            const matchesType = scheduleType ? selectedTypes.includes(scheduleType) : true;

            return matchesSearch && matchesType;
        });
    }, [events, searchQuery, selectedTypes]);

    const openEvent = useCallback((event: any) => {
        console.log("CalendarEvent clicked:", event);
    }, []);

    // Handle scheduling
    const [isScheduleDialogOpen, setIsScheduleDialogOpen] = useState(false);
    const [editingSchedule, setEditingSchedule] = useState<Schedule | null>(null);
    const handleAddSchedule = useCallback(() => {
        setIsScheduleDialogOpen(true);
    }, []);
    const handleUpdateSchedule = useCallback((schedule: Schedule) => {
        setEditingSchedule(schedule);
        setIsScheduleDialogOpen(true);
    }, []);
    const handleCloseScheduleDialog = useCallback(() => {
        setIsScheduleDialogOpen(false);
        setEditingSchedule(null); // Clear editing state on close
    }, []);
    const handleScheduleCompleted = useCallback((_created: Schedule) => {
        //TODO update schedule list/refetch
        handleCloseScheduleDialog();
    }, [handleCloseScheduleDialog]);
    const handleScheduleDeleted = useCallback((_deleted: Schedule) => {
        //TODO update schedule list/refetch
        handleCloseScheduleDialog();
    }, [handleCloseScheduleDialog]);

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

    // Event styling based on type
    const eventPropGetter = useCallback((event: CalendarEvent) => {
        const scheduleType = event.schedule?.scheduleFor as ScheduleFor | undefined;
        let backgroundColor = palette.primary.main; // Default color
        let color = palette.primary.contrastText; // Default text color

        // Assign colors based on schedule type
        switch (scheduleType) {
            case ScheduleFor.FocusMode:
                backgroundColor = palette.success.light;
                color = palette.success.contrastText;
                break;
            case ScheduleFor.Meeting:
                backgroundColor = palette.info.light;
                color = palette.info.contrastText;
                break;
            case ScheduleFor.RunProject:
                backgroundColor = palette.warning.light;
                color = palette.warning.contrastText;
                break;
            case ScheduleFor.RunRoutine:
                backgroundColor = palette.secondary.light;
                color = palette.secondary.contrastText;
                break;
            // Add more cases if needed
            default:
                // Use default colors
                break;
        }

        const style: React.CSSProperties = {
            backgroundColor,
            color,
            borderRadius: "5px",
            opacity: 0.8,
            border: "0px",
            display: "block",
            fontSize: typography.caption.fontSize, // Use caption size
            lineHeight: typography.caption.lineHeight, // Use caption line height
            padding: "2px 4px", // Add some padding
        };
        return {
            style,
        };
    }, [palette.primary.main, palette.primary.contrastText, palette.success.light, palette.success.contrastText, palette.info.light, palette.info.contrastText, palette.warning.light, palette.warning.contrastText, palette.secondary.light, palette.secondary.contrastText, typography.caption.fontSize, typography.caption.lineHeight]);

    const calendarComponents = useMemo(function calendarComponentsMemo() {
        return {
            toolbar: (props: CalendarToolbarProps<CalendarEvent, object>) => <CustomToolbar {...props} onSelectDate={handleSelectDate} />,
            month: {
                header: (props: CalendarHeaderProps) => <DayColumnHeader isBottomNavVisible={isBottomNavVisible} {...props} />,
            },
        } satisfies Components<CalendarEvent, object>;
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

    // Function to handle closing filter and opening add dialog
    const handleAddNewEvent = useCallback(() => {
        closeFilterDialog();
        handleAddSchedule();
    }, [closeFilterDialog, handleAddSchedule]);

    if (!localizer) return <FullPageSpinner />;
    return (
        <Box sx={outerBoxStyle}>
            <Navbar keepVisible title={t("Schedule", { count: 1, defaultValue: "Schedule" })} />
            <FlexContainer isBottomNavVisible={isBottomNavVisible}>
                <ScheduleUpsert
                    canSetScheduleFor={true}
                    defaultScheduleFor={activeTab === CalendarTabs.CALENDAR ? ScheduleFor.Meeting : ScheduleFor.Meeting}
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

                <FilterDialog
                    isOpen={isFilterDialogOpen}
                    onClose={closeFilterDialog}
                    searchQuery={searchQuery}
                    setSearchQuery={setSearchQuery}
                    selectedTypes={selectedTypes}
                    setSelectedTypes={setSelectedTypes}
                    filteredEvents={filteredEvents}
                    key={activeTab}
                    onAddNew={handleAddNewEvent}
                    activeTab={activeTab}
                />

                <Tabs
                    value={activeTab}
                    onChange={handleTabChange}
                    variant="fullWidth"
                >
                    <Tab
                        label={t("Calendar", { defaultValue: "Calendar" })}
                        value={CalendarTabs.CALENDAR}
                        icon={<IconCommon name="Schedule" />}
                        iconPosition="start"
                    />
                    <Tab
                        label={t("Trigger", { defaultValue: "Trigger" })}
                        value={CalendarTabs.TRIGGER}
                        icon={<IconCommon name="History" />}
                        iconPosition="start"
                    />
                </Tabs>

                {activeTab === CalendarTabs.CALENDAR && (
                    <Calendar<CalendarEvent, object>
                        localizer={localizer}
                        longPressThreshold={20}
                        events={filteredEvents}
                        onRangeChange={handleDateRangeChange}
                        onSelectEvent={openEvent}
                        onSelectSlot={handleSelectSlot}
                        onView={handleViewChange}
                        startAccessor="start"
                        endAccessor="end"
                        components={calendarComponents}
                        dayPropGetter={dayPropGetter}
                        eventPropGetter={eventPropGetter}
                        selectable={true}
                        slotPropGetter={slotPropGetter}
                        style={calendarStyle}
                        view={view}
                        views={views}
                    />
                )}

                {activeTab === CalendarTabs.TRIGGER && (
                    <TriggerView />
                )}
            </FlexContainer>
            <SideActionsButtons display={display}>
                <IconButton
                    aria-label={t("CreateEvent", { defaultValue: "Create Event" })}
                    onClick={handleAddSchedule}
                >
                    <IconCommon name="Add" />
                </IconButton>
                <IconButton
                    aria-label={t("Filter", { defaultValue: "Filter" })}
                    onClick={openFilterDialog}
                >
                    <IconCommon name="Filter" />
                </IconButton>
            </SideActionsButtons>
        </Box>
    );
}
