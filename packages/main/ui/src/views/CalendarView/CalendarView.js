import { jsx as _jsx, Fragment as _Fragment, jsxs as _jsxs } from "react/jsx-runtime";
import { AddIcon, FocusModeIcon, OrganizationIcon, ProjectIcon, RoutineIcon } from "@local/icons";
import { calculateOccurrences } from "@local/utils";
import { useTheme } from "@mui/material";
import { add, endOfMonth, format, getDay, startOfMonth, startOfWeek } from "date-fns";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { Calendar, dateFnsLocalizer } from "react-big-calendar";
import "react-big-calendar/lib/css/react-big-calendar.css";
import { useTranslation } from "react-i18next";
import { ColorIconButton } from "../../components/buttons/ColorIconButton/ColorIconButton";
import { SideActionButtons } from "../../components/buttons/SideActionButtons/SideActionButtons";
import { ScheduleDialog } from "../../components/dialogs/ScheduleDialog/ScheduleDialog";
import { FullPageSpinner } from "../../components/FullPageSpinner/FullPageSpinner";
import { TopBar } from "../../components/navigation/TopBar/TopBar";
import { PageTabs } from "../../components/PageTabs/PageTabs";
import { getCurrentUser } from "../../utils/authentication/session";
import { getDisplay } from "../../utils/display/listTools";
import { getUserLanguages, getUserLocale, loadLocale } from "../../utils/display/translationTools";
import { useDimensions } from "../../utils/hooks/useDimensions";
import { useFindMany } from "../../utils/hooks/useFindMany";
import { useWindowSize } from "../../utils/hooks/useWindowSize";
import { addSearchParams, parseSearchParams, useLocation } from "../../utils/route";
import { CalendarPageTabOption } from "../../utils/search/objectToSearch";
import { SessionContext } from "../../utils/SessionContext";
const tabParams = [{
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
export const CalendarView = ({ display = "page", }) => {
    const session = useContext(SessionContext);
    const { breakpoints, palette } = useTheme();
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const locale = useMemo(() => getUserLocale(session), [session]);
    const [localizer, setLocalizer] = useState(null);
    const [dateRange, setDateRange] = useState({
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
            }
            catch (error) {
                console.error("Failed to load locale:", error);
            }
        };
        localeLoader();
    }, [locale]);
    const { dimensions, ref } = useDimensions();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);
    const calendarHeight = useMemo(() => `calc(100vh - ${dimensions.height}px - ${isMobile ? 56 : 0}px - env(safe-area-inset-bottom))`, [dimensions, isMobile]);
    const handleDateRangeChange = useCallback((range) => {
        if (Array.isArray(range)) {
            setDateRange({ start: range[0], end: range[1] });
        }
        else {
            setDateRange(range);
        }
    }, []);
    const tabs = useMemo(() => {
        return tabParams.map((tab, i) => ({
            index: i,
            Icon: tab.Icon,
            label: t(tab.titleKey, { count: 2, defaultValue: tab.titleKey }),
            value: tab.tabType,
        }));
    }, [t]);
    const [currTab, setCurrTab] = useState(() => {
        const searchParams = parseSearchParams();
        const index = tabParams.findIndex(tab => tab.tabType === searchParams.type);
        if (index === -1)
            return tabs[0];
        return tabs[index];
    });
    const handleTabChange = useCallback((e, tab) => {
        e.preventDefault();
        addSearchParams(setLocation, { type: tab.value });
        setCurrTab(tab);
    }, [setLocation]);
    const { allData: schedules, hasMore, loading, loadMore, } = useFindMany({
        canSearch: true,
        searchType: "Schedule",
        where: {
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
    useEffect(() => {
        if (!loading && hasMore && dateRange.start && dateRange.end) {
            loadMore();
        }
    }, [dateRange, loadMore, loading, hasMore]);
    const events = useMemo(() => {
        if (!dateRange.start || !dateRange.end)
            return [];
        const result = [];
        schedules.forEach((schedule) => {
            const occurrences = calculateOccurrences(schedule, dateRange.start, dateRange.end);
            const events = occurrences.map(occurrence => ({
                __typename: "CalendarEvent",
                id: `${schedule.id}|${occurrence.start.getTime()}|${occurrence.end.getTime()}`,
                title: getDisplay(schedule, getUserLanguages(session)).title,
                start: occurrence.start,
                end: occurrence.end,
                allDay: false,
                schedule,
            }));
            result.push(...events);
        });
        return result;
    }, [dateRange.end, dateRange.start, schedules, session]);
    const openEvent = useCallback((event) => {
        console.log("CalendarEvent clicked:", event);
    }, []);
    const [isScheduleDialogOpen, setIsScheduleDialogOpen] = useState(false);
    const [editingSchedule, setEditingSchedule] = useState(null);
    const handleAddSchedule = () => { setIsScheduleDialogOpen(true); };
    const handleUpdateSchedule = (schedule) => {
        setEditingSchedule(schedule);
        setIsScheduleDialogOpen(true);
    };
    const handleCloseScheduleDialog = () => { setIsScheduleDialogOpen(false); };
    const handleScheduleCreated = (created) => {
        setIsScheduleDialogOpen(false);
    };
    const handleScheduleUpdated = (updated) => {
        setIsScheduleDialogOpen(false);
    };
    const handleDeleteSchedule = () => {
    };
    if (!localizer)
        return _jsx(FullPageSpinner, {});
    return (_jsxs(_Fragment, { children: [_jsx(ScheduleDialog, { isCreate: editingSchedule === null, isMutate: true, isOpen: isScheduleDialogOpen, onClose: handleCloseScheduleDialog, onCreated: handleScheduleCreated, onUpdated: handleScheduleUpdated, zIndex: 202 }), _jsx(SideActionButtons, { display: display, zIndex: 201, children: _jsx(ColorIconButton, { "aria-label": "create event", background: palette.secondary.main, onClick: handleAddSchedule, sx: {
                        padding: 0,
                        width: "54px",
                        height: "54px",
                    }, children: _jsx(AddIcon, { fill: palette.secondary.contrastText, width: '36px', height: '36px' }) }) }), _jsx(TopBar, { ref: ref, display: display, onClose: () => { }, titleData: { title: currTab.label }, below: _jsx(PageTabs, { ariaLabel: "calendar-tabs", currTab: currTab, fullWidth: true, onChange: handleTabChange, tabs: tabs }) }), _jsx(Calendar, { localizer: localizer, events: events, onRangeChange: handleDateRangeChange, onSelectEvent: openEvent, startAccessor: "start", endAccessor: "end", views: ["month", "week", "day"], style: {
                    height: calendarHeight,
                    background: palette.background.paper,
                } })] }));
};
//# sourceMappingURL=CalendarView.js.map