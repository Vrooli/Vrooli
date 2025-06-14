import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Checkbox from "@mui/material/Checkbox";
import DialogContent from "@mui/material/DialogContent";
import Divider from "@mui/material/Divider";
import FormControlLabel from "@mui/material/FormControlLabel";
import FormGroup from "@mui/material/FormGroup";
import IconButton from "@mui/material/IconButton";
import InputAdornment from "@mui/material/InputAdornment";
import List from "@mui/material/List";
import ListItem from "@mui/material/ListItem";
import ListItemText from "@mui/material/ListItemText";
import Paper from "@mui/material/Paper";
import Tab from "@mui/material/Tab";
import Tabs from "@mui/material/Tabs";
import TextField from "@mui/material/TextField";
import Typography from "@mui/material/Typography";
import type { BoxProps } from "@mui/material";
import { styled, useTheme } from "@mui/material";
import { ScheduleFor, calculateOccurrences, type CalendarEvent, type Schedule } from "@vrooli/shared";
import { add, endOfMonth, format, getDay, startOfMonth, startOfWeek } from "date-fns";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { Calendar, Views, dateFnsLocalizer, type HeaderProps as CalendarHeaderProps, type Components, type DateLocalizer, type SlotInfo, type View } from "react-big-calendar";
import "react-big-calendar/lib/css/react-big-calendar.css";
import { useTranslation } from "react-i18next";
import { CalendarPreviewToolbar } from "../../components/CalendarPreviewToolbar.js";
import { FullPageSpinner } from "../../components/Spinners.js";
import { SideActionsButtons } from "../../components/buttons/SideActionsButtons.js";
import { LargeDialog } from "../../components/dialogs/LargeDialog/LargeDialog.js";
import { useIsBottomNavVisible } from "../../components/navigation/BottomNav.js";
import { APP_BAR_HEIGHT_PX, Navbar } from "../../components/navigation/Navbar.js";
import { SessionContext } from "../../contexts/session.js";
import { useFindMany } from "../../hooks/useFindMany.js";
import { useHistoryState } from "../../hooks/useHistoryState.js";
import { IconCommon } from "../../icons/Icons.js";
import { useLocation } from "../../route/router.js";
import { bottomNavHeight } from "../../styles.js";
import { type PartialWithType } from "../../types.js";
import { getCurrentUser } from "../../utils/authentication/session.js";
import { getDisplay } from "../../utils/display/listTools.js";
import { getShortenedLabel, getUserLanguages, getUserLocale, loadLocale } from "../../utils/display/translationTools.js";
import { lazy, Suspense } from "react";

const ScheduleUpsert = lazy(() => import("../objects/schedule/ScheduleUpsert.js").then(module => ({ default: module.ScheduleUpsert })));
import { type CalendarViewProps } from "../types.js";

// Workaround typing issues: wrap Calendar as any to satisfy JSX
const BigCalendar: any = Calendar;

// Define the tab values
export enum CalendarTabs {
    CALENDAR = "calendar",
    TRIGGER = "trigger",
}

// Define schedule types for filtering using the ScheduleFor enum
const SCHEDULE_TYPES: ScheduleFor[] = [
    ScheduleFor.Meeting,
    ScheduleFor.Run,
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
 * Filter dialog component for filtering events and triggers
 */
interface FilterDialogProps {
    isOpen: boolean;
    onClose: () => void;
    searchQuery: string;
    setSearchQuery: (query: string) => void;
    selectedTypes: ScheduleFor[];
    setSelectedTypes: (updater: (prev: ScheduleFor[]) => ScheduleFor[]) => void;
    selectedTriggerEvents?: TriggerEvent[];
    setSelectedTriggerEvents?: (updater: (prev: TriggerEvent[]) => TriggerEvent[]) => void;
    selectedTriggerActions?: TriggerAction[];
    setSelectedTriggerActions?: (updater: (prev: TriggerAction[]) => TriggerAction[]) => void;
    filteredEvents: CalendarEvent[];
    filteredTriggers?: Trigger[];
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
    selectedTriggerEvents = [],
    setSelectedTriggerEvents,
    selectedTriggerActions = [],
    setSelectedTriggerActions,
    filteredEvents,
    filteredTriggers = [],
    activeTab,
    onAddNew,
}: FilterDialogProps) {
    const dialogId = "filter-dialog";
    const { t } = useTranslation();

    const handleSearchChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
        setSearchQuery(e.target.value);
    }, [setSearchQuery]);

    // Calendar handlers
    const handleSelectAll = useCallback(() => {
        if (selectedTypes.length === SCHEDULE_TYPES.length) {
            setSelectedTypes(() => []);
        } else {
            setSelectedTypes(() => [...SCHEDULE_TYPES]);
        }
    }, [selectedTypes.length, setSelectedTypes]);

    const handleTypeChange = useCallback((type: ScheduleFor) => () => {
        setSelectedTypes(prev => {
            if (prev.includes(type)) {
                return prev.filter(t => t !== type);
            } else {
                return [...prev, type];
            }
        });
    }, [setSelectedTypes]);

    // Trigger handlers
    const triggerEventTypes: TriggerEvent[] = ["RunCompleted", "RunStarted", "RunFailed", "NoteCreated", "NoteUpdated", "ProjectUpdated", "DailyAt", "WeeklyOn"];
    const triggerActionTypes: TriggerAction[] = ["StartRun", "CreateMeeting", "SendNotification", "CreateNote", "UpdateProject"];

    const handleSelectAllTriggerEvents = useCallback(() => {
        if (!setSelectedTriggerEvents) return;
        if (selectedTriggerEvents.length === triggerEventTypes.length) {
            setSelectedTriggerEvents(() => []);
        } else {
            setSelectedTriggerEvents(() => [...triggerEventTypes]);
        }
    }, [selectedTriggerEvents.length, setSelectedTriggerEvents]);

    const handleTriggerEventChange = useCallback((event: TriggerEvent) => () => {
        if (!setSelectedTriggerEvents) return;
        setSelectedTriggerEvents(prev => {
            if (prev.includes(event)) {
                return prev.filter(e => e !== event);
            } else {
                return [...prev, event];
            }
        });
    }, [setSelectedTriggerEvents]);

    const handleSelectAllTriggerActions = useCallback(() => {
        if (!setSelectedTriggerActions) return;
        if (selectedTriggerActions.length === triggerActionTypes.length) {
            setSelectedTriggerActions(() => []);
        } else {
            setSelectedTriggerActions(() => [...triggerActionTypes]);
        }
    }, [selectedTriggerActions.length, setSelectedTriggerActions]);

    const handleTriggerActionChange = useCallback((action: TriggerAction) => () => {
        if (!setSelectedTriggerActions) return;
        setSelectedTriggerActions(prev => {
            if (prev.includes(action)) {
                return prev.filter(a => a !== action);
            } else {
                return [...prev, action];
            }
        });
    }, [setSelectedTriggerActions]);

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
                        : t("FindTriggers", { defaultValue: "Find automation triggers..." })}
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
                                <Box sx={{ display: "flex", flexDirection: "column", gap: 2 }}>
                                    {/* Trigger Events Filter */}
                                    <Box>
                                        <Typography variant="subtitle2" gutterBottom>
                                            {t("TriggerEvents", { defaultValue: "Trigger Events" })}
                                        </Typography>
                                        <FormGroup>
                                            <FormControlLabel
                                                control={
                                                    <Checkbox
                                                        color="secondary"
                                                        checked={selectedTriggerEvents.length === triggerEventTypes.length}
                                                        indeterminate={selectedTriggerEvents.length > 0 && selectedTriggerEvents.length < triggerEventTypes.length}
                                                        onChange={handleSelectAllTriggerEvents}
                                                    />
                                                }
                                                label={t("All", { defaultValue: "All" })}
                                            />
                                            <Divider sx={filterDialogDividerSx} />
                                            {triggerEventTypes.map((event) => {
                                                const eventInfo = getTriggerEventInfo(event);
                                                return (
                                                    <FormControlLabel
                                                        key={event}
                                                        control={
                                                            <Checkbox
                                                                color="secondary"
                                                                checked={selectedTriggerEvents.includes(event)}
                                                                onChange={handleTriggerEventChange(event)}
                                                            />
                                                        }
                                                        label={eventInfo.label}
                                                    />
                                                );
                                            })}
                                        </FormGroup>
                                    </Box>

                                    <Divider />

                                    {/* Trigger Actions Filter */}
                                    <Box>
                                        <Typography variant="subtitle2" gutterBottom>
                                            {t("TriggerActions", { defaultValue: "Trigger Actions" })}
                                        </Typography>
                                        <FormGroup>
                                            <FormControlLabel
                                                control={
                                                    <Checkbox
                                                        color="secondary"
                                                        checked={selectedTriggerActions.length === triggerActionTypes.length}
                                                        indeterminate={selectedTriggerActions.length > 0 && selectedTriggerActions.length < triggerActionTypes.length}
                                                        onChange={handleSelectAllTriggerActions}
                                                    />
                                                }
                                                label={t("All", { defaultValue: "All" })}
                                            />
                                            <Divider sx={filterDialogDividerSx} />
                                            {triggerActionTypes.map((action) => {
                                                const actionInfo = getTriggerActionInfo(action);
                                                return (
                                                    <FormControlLabel
                                                        key={action}
                                                        control={
                                                            <Checkbox
                                                                color="secondary"
                                                                checked={selectedTriggerActions.includes(action)}
                                                                onChange={handleTriggerActionChange(action)}
                                                            />
                                                        }
                                                        label={actionInfo.label}
                                                    />
                                                );
                                            })}
                                        </FormGroup>
                                    </Box>
                                </Box>
                            </Paper>
                        )}
                    </Box>

                    <Box width="50%" pl={1}>
                        <Typography variant="subtitle1" gutterBottom>
                            {activeTab === CalendarTabs.CALENDAR 
                                ? t("Result", { count: filteredEvents.length, defaultValue: "Results" }) 
                                : t("Result", { count: filteredTriggers.length, defaultValue: "Results" })} 
                            ({activeTab === CalendarTabs.CALENDAR ? filteredEvents.length : filteredTriggers.length})
                        </Typography>
                        <Paper elevation={1} sx={filterSectionPaperSx}>
                            <List dense sx={resultsListSx}>
                                {activeTab === CalendarTabs.CALENDAR ? (
                                    filteredEvents.length > 0 ? (
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
                                    )
                                ) : (
                                    filteredTriggers.length > 0 ? (
                                        filteredTriggers.map((trigger) => {
                                            const eventInfo = getTriggerEventInfo(trigger.triggerEvent);
                                            const actionInfo = getTriggerActionInfo(trigger.action);
                                            return (
                                                <ListItem key={trigger.id}>
                                                    <ListItemText
                                                        primary={trigger.name}
                                                        secondary={
                                                            <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
                                                                <Typography variant="caption">
                                                                    {eventInfo.label} → {actionInfo.label}
                                                                </Typography>
                                                                {trigger.enabled ? (
                                                                    <Typography variant="caption" color="success.main">
                                                                        {t("Enabled", { defaultValue: "Enabled" })}
                                                                    </Typography>
                                                                ) : (
                                                                    <Typography variant="caption" color="text.disabled">
                                                                        {t("Disabled", { defaultValue: "Disabled" })}
                                                                    </Typography>
                                                                )}
                                                            </Box>
                                                        }
                                                    />
                                                </ListItem>
                                            );
                                        })
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
                                                    {t("AddTrigger", { defaultValue: "Add Trigger" })}
                                                </Button>
                                            </Box>
                                        </ListItem>
                                    )
                                )}
                            </List>
                        </Paper>
                    </Box>
                </Box>
            </DialogContent>
        </LargeDialog>
    );
}

// Trigger types for the UI
type TriggerEvent = "RunCompleted" | "RunStarted" | "RunFailed" | "NoteCreated" | "NoteUpdated" | "ProjectUpdated" | "UserLogin" | "UserSignup" | "DailyAt" | "WeeklyOn" | "WebhookReceived" | "EmailReceived";
type TriggerAction = "StartRun" | "CreateMeeting" | "SendNotification" | "CreateNote" | "UpdateProject";

type TriggerCondition = {
    field: string;
    operator: "equals" | "contains" | "greater_than" | "less_than";
    value: any;
};

type Trigger = {
    __typename: "Trigger";
    id: string;
    name: string;
    description?: string;
    enabled: boolean;
    triggerEvent: TriggerEvent;
    triggerConditions?: TriggerCondition[];
    action: TriggerAction;
    actionConfig: Record<string, any>;
    createdAt: string;
    updatedAt: string;
    you: {
        __typename: "TriggerYou";
        canDelete: boolean;
        canUpdate: boolean;
        canRead: boolean;
    };
};

// Helper function to get trigger event display info
const getTriggerEventInfo = (event: TriggerEvent) => {
    const eventMap = {
        RunCompleted: { icon: "CheckCircle", label: "Run Completed", color: "success" },
        RunStarted: { icon: "PlayArrow", label: "Run Started", color: "info" },
        RunFailed: { icon: "Error", label: "Run Failed", color: "error" },
        NoteCreated: { icon: "Add", label: "Note Created", color: "primary" },
        NoteUpdated: { icon: "Edit", label: "Note Updated", color: "warning" },
        ProjectUpdated: { icon: "Update", label: "Project Updated", color: "warning" },
        UserLogin: { icon: "Login", label: "User Login", color: "info" },
        UserSignup: { icon: "PersonAdd", label: "User Signup", color: "success" },
        DailyAt: { icon: "Schedule", label: "Daily At", color: "primary" },
        WeeklyOn: { icon: "DateRange", label: "Weekly On", color: "primary" },
        WebhookReceived: { icon: "Webhook", label: "Webhook Received", color: "secondary" },
        EmailReceived: { icon: "Email", label: "Email Received", color: "secondary" },
    };
    return eventMap[event] || { icon: "Help", label: event, color: "default" };
};

// Helper function to get trigger action display info
const getTriggerActionInfo = (action: TriggerAction) => {
    const actionMap = {
        StartRun: { icon: "PlayArrow", label: "Start Run", color: "success" },
        CreateMeeting: { icon: "Event", label: "Create Meeting", color: "primary" },
        SendNotification: { icon: "Notifications", label: "Send Notification", color: "warning" },
        CreateNote: { icon: "Note", label: "Create Note", color: "info" },
        UpdateProject: { icon: "Update", label: "Update Project", color: "secondary" },
    };
    return actionMap[action] || { icon: "Help", label: action, color: "default" };
};

/**
 * Trigger tab component showing automation triggers
 */
interface TriggerViewProps {
    triggers: Trigger[];
    loading: boolean;
    onTriggerUpdate: (triggers: Trigger[]) => void;
}

function TriggerView({ triggers, loading, onTriggerUpdate }: TriggerViewProps) {
    const { t } = useTranslation();

    const handleTriggerToggle = useCallback((triggerId: string, enabled: boolean) => {
        onTriggerUpdate(triggers.map(trigger => 
            trigger.id === triggerId ? { ...trigger, enabled } : trigger
        ));
    }, [triggers, onTriggerUpdate]);

    const handleEditTrigger = useCallback((trigger: Trigger) => {
        console.log("Edit trigger:", trigger);
        // TODO: Open trigger edit dialog
    }, []);

    const handleDeleteTrigger = useCallback((triggerId: string) => {
        console.log("Delete trigger:", triggerId);
        onTriggerUpdate(triggers.filter(trigger => trigger.id !== triggerId));
    }, [triggers, onTriggerUpdate]);

    const handleAddTrigger = useCallback(() => {
        console.log("Add new trigger");
        // TODO: Open trigger create dialog
    }, []);

    if (loading) {
        return (
            <Box sx={triggerViewBoxSx}>
                <FullPageSpinner />
            </Box>
        );
    }

    if (triggers.length === 0) {
        return (
            <Box sx={triggerViewBoxSx}>
                <IconCommon name="History" size={64} fill="text.secondary" />
                <Typography variant="h5" sx={triggerViewTitleSx}>
                    {t("NoTriggersYet", { defaultValue: "No Automation Triggers Yet" })}
                </Typography>
                <Typography variant="body1" color="text.secondary" sx={triggerViewDescSx}>
                    {t("TriggerDescription", { defaultValue: "Create triggers to automate routine tasks based on events like completing runs, updating notes, or scheduled times." })}
                </Typography>
                <Button
                    variant="contained"
                    color="primary"
                    onClick={handleAddTrigger}
                    startIcon={<IconCommon name="Add" />}
                    sx={{ mt: 2 }}
                >
                    {t("CreateFirstTrigger", { defaultValue: "Create Your First Trigger" })}
                </Button>
            </Box>
        );
    }

    return (
        <Box sx={{ p: 2, height: "100%", overflow: "auto" }}>
            <Box sx={{ display: "flex", justifyContent: "space-between", alignItems: "center", mb: 2 }}>
                <Typography variant="h6">
                    {t("AutomationTriggers", { defaultValue: "Automation Triggers" })} ({triggers.length})
                </Typography>
                <Button
                    variant="contained"
                    color="primary"
                    onClick={handleAddTrigger}
                    startIcon={<IconCommon name="Add" />}
                    size="small"
                >
                    {t("AddTrigger", { defaultValue: "Add Trigger" })}
                </Button>
            </Box>

            <List>
                {triggers.map((trigger) => {
                    const eventInfo = getTriggerEventInfo(trigger.triggerEvent);
                    const actionInfo = getTriggerActionInfo(trigger.action);
                    
                    return (
                        <ListItem
                            key={trigger.id}
                            divider
                            sx={{
                                border: 1,
                                borderColor: "divider",
                                borderRadius: 1,
                                mb: 1,
                                backgroundColor: trigger.enabled ? "background.paper" : "action.disabled",
                                opacity: trigger.enabled ? 1 : 0.6,
                            }}
                        >
                            <Box sx={{ width: "100%" }}>
                                {/* Header */}
                                <Box sx={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start", mb: 1 }}>
                                    <Box sx={{ flex: 1 }}>
                                        <Typography variant="subtitle1" sx={{ fontWeight: "bold" }}>
                                            {trigger.name}
                                        </Typography>
                                        {trigger.description && (
                                            <Typography variant="body2" color="text.secondary">
                                                {trigger.description}
                                            </Typography>
                                        )}
                                    </Box>
                                    <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
                                        <FormControlLabel
                                            control={
                                                <Checkbox
                                                    checked={trigger.enabled}
                                                    onChange={(e) => handleTriggerToggle(trigger.id, e.target.checked)}
                                                    color="primary"
                                                />
                                            }
                                            label={t("Enabled", { defaultValue: "Enabled" })}
                                            sx={{ mr: 1 }}
                                        />
                                        <IconButton
                                            size="small"
                                            onClick={() => handleEditTrigger(trigger)}
                                            disabled={!trigger.you.canUpdate}
                                        >
                                            <IconCommon name="Edit" />
                                        </IconButton>
                                        <IconButton
                                            size="small"
                                            onClick={() => handleDeleteTrigger(trigger.id)}
                                            disabled={!trigger.you.canDelete}
                                            color="error"
                                        >
                                            <IconCommon name="Delete" />
                                        </IconButton>
                                    </Box>
                                </Box>

                                {/* Trigger Flow */}
                                <Box sx={{ display: "flex", alignItems: "center", gap: 2, mt: 2 }}>
                                    {/* Event */}
                                    <Box sx={{ display: "flex", alignItems: "center", gap: 1, flex: 1 }}>
                                        <IconCommon name={eventInfo.icon as any} color={eventInfo.color as any} />
                                        <Box>
                                            <Typography variant="caption" color="text.secondary">
                                                {t("When", { defaultValue: "WHEN" })}
                                            </Typography>
                                            <Typography variant="body2" sx={{ fontWeight: "medium" }}>
                                                {eventInfo.label}
                                            </Typography>
                                        </Box>
                                    </Box>

                                    {/* Arrow */}
                                    <IconCommon name="ArrowForward" color="text.secondary" />

                                    {/* Action */}
                                    <Box sx={{ display: "flex", alignItems: "center", gap: 1, flex: 1 }}>
                                        <IconCommon name={actionInfo.icon as any} color={actionInfo.color as any} />
                                        <Box>
                                            <Typography variant="caption" color="text.secondary">
                                                {t("Then", { defaultValue: "THEN" })}
                                            </Typography>
                                            <Typography variant="body2" sx={{ fontWeight: "medium" }}>
                                                {actionInfo.label}
                                            </Typography>
                                        </Box>
                                    </Box>
                                </Box>

                                {/* Conditions */}
                                {trigger.triggerConditions && trigger.triggerConditions.length > 0 && (
                                    <Box sx={{ mt: 1, pl: 2, borderLeft: 2, borderColor: "divider" }}>
                                        <Typography variant="caption" color="text.secondary">
                                            {t("Conditions", { defaultValue: "CONDITIONS" })}
                                        </Typography>
                                        {trigger.triggerConditions.map((condition, index) => (
                                            <Typography key={index} variant="body2" color="text.secondary">
                                                {condition.field} {condition.operator} {condition.value}
                                            </Typography>
                                        ))}
                                    </Box>
                                )}
                            </Box>
                        </ListItem>
                    );
                })}
            </List>
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
    const [location] = useLocation();

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
    const [selectedTriggerEvents, setSelectedTriggerEvents] = useState<TriggerEvent[]>(["RunCompleted", "RunStarted", "RunFailed", "NoteCreated", "NoteUpdated", "ProjectUpdated", "DailyAt", "WeeklyOn"]);
    const [selectedTriggerActions, setSelectedTriggerActions] = useState<TriggerAction[]>(["StartRun", "CreateMeeting", "SendNotification", "CreateNote", "UpdateProject"]);

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

    // Trigger data state
    const [triggers, setTriggers] = useState<Trigger[]>([]);
    const [triggersLoading, setTriggersLoading] = useState(true);
    
    // Load mock triggers - in real implementation would use useFindMany
    useEffect(() => {
        if (activeTab === CalendarTabs.TRIGGER) {
            setTimeout(() => {
                const mockTriggersData: Trigger[] = [
                    {
                        __typename: "Trigger",
                        id: "trigger-1",
                        name: "Weekly Planning Notification",
                        description: "Send reminder to plan the week every Monday",
                        enabled: true,
                        triggerEvent: "WeeklyOn",
                        triggerConditions: [
                            { field: "dayOfWeek", operator: "equals", value: 1 },
                            { field: "timeOfDay", operator: "equals", value: "09:00" }
                        ],
                        action: "SendNotification",
                        actionConfig: {
                            title: "Weekly Planning Time",
                            message: "Don't forget to plan your week ahead!",
                            type: "reminder"
                        },
                        createdAt: new Date().toISOString(),
                        updatedAt: new Date().toISOString(),
                        you: { __typename: "TriggerYou", canDelete: true, canUpdate: true, canRead: true },
                    },
                    {
                        __typename: "Trigger",
                        id: "trigger-2",
                        name: "Project Completion Follow-up",
                        description: "Start retrospective routine when a project is completed",
                        enabled: true,
                        triggerEvent: "RunCompleted",
                        triggerConditions: [
                            { field: "routineType", operator: "equals", value: "project" }
                        ],
                        action: "StartRun",
                        actionConfig: {
                            routineId: "retrospective-routine-id",
                            delay: 30
                        },
                        createdAt: new Date().toISOString(),
                        updatedAt: new Date().toISOString(),
                        you: { __typename: "TriggerYou", canDelete: true, canUpdate: true, canRead: true },
                    },
                    {
                        __typename: "Trigger",
                        id: "trigger-3",
                        name: "Team Meeting After Sprint",
                        description: "Schedule team meeting automatically after sprint completion",
                        enabled: false,
                        triggerEvent: "RunCompleted",
                        triggerConditions: [
                            { field: "tags", operator: "contains", value: "sprint" }
                        ],
                        action: "CreateMeeting",
                        actionConfig: {
                            title: "Sprint Retrospective Meeting",
                            duration: 60,
                            teamId: "team-id",
                            scheduledOffset: "1 day"
                        },
                        createdAt: new Date().toISOString(),
                        updatedAt: new Date().toISOString(),
                        you: { __typename: "TriggerYou", canDelete: true, canUpdate: true, canRead: true },
                    },
                    {
                        __typename: "Trigger",
                        id: "trigger-4",
                        name: "Daily Standup Auto-Start",
                        description: "Automatically start daily standup routine every weekday",
                        enabled: true,
                        triggerEvent: "DailyAt",
                        triggerConditions: [
                            { field: "timeOfDay", operator: "equals", value: "09:30" },
                            { field: "weekdaysOnly", operator: "equals", value: true }
                        ],
                        action: "StartRun",
                        actionConfig: {
                            routineId: "daily-standup-routine-id"
                        },
                        createdAt: new Date().toISOString(),
                        updatedAt: new Date().toISOString(),
                        you: { __typename: "TriggerYou", canDelete: true, canUpdate: true, canRead: true },
                    },
                    {
                        __typename: "Trigger",
                        id: "trigger-5",
                        name: "Note Update Documentation",
                        description: "Create documentation note when important project notes are updated",
                        enabled: true,
                        triggerEvent: "NoteUpdated",
                        triggerConditions: [
                            { field: "tags", operator: "contains", value: "important" }
                        ],
                        action: "CreateNote",
                        actionConfig: {
                            title: "Documentation Update Required",
                            template: "project-doc-template",
                            parentProject: "auto-detect"
                        },
                        createdAt: new Date().toISOString(),
                        updatedAt: new Date().toISOString(),
                        you: { __typename: "TriggerYou", canDelete: true, canUpdate: true, canRead: true },
                    },
                ];
                setTriggers(mockTriggersData);
                setTriggersLoading(false);
            }, 500);
        }
    }, [activeTab]);

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
                        if (schedule.meetings?.[0]?.translations?.[0]?.name) {
                            title = schedule.meetings[0].translations[0].name;
                        } else if (schedule.runs?.[0]?.name) {
                            title = schedule.runs[0].name;
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
            // Determine schedule type by checking meetings or runs array
            let scheduleType: ScheduleFor | undefined;
            if (event.schedule?.meetings?.length) {
                scheduleType = ScheduleFor.Meeting;
            } else if (event.schedule?.runs?.length) {
                scheduleType = ScheduleFor.Run;
            }
            const matchesType = scheduleType ? selectedTypes.includes(scheduleType) : true;

            return matchesSearch && matchesType;
        });
    }, [events, searchQuery, selectedTypes]);

    // Filter triggers based on search query and selected event/action types
    const filteredTriggers = useMemo(() => {
        return triggers.filter(trigger => {
            // Filter by search query
            const matchesSearch = searchQuery.trim() === "" ||
                trigger.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
                (trigger.description?.toLowerCase().includes(searchQuery.toLowerCase()) ?? false);

            // Filter by selected trigger events
            const matchesEvent = selectedTriggerEvents.includes(trigger.triggerEvent);

            // Filter by selected trigger actions
            const matchesAction = selectedTriggerActions.includes(trigger.action);

            return matchesSearch && matchesEvent && matchesAction;
        });
    }, [triggers, searchQuery, selectedTriggerEvents, selectedTriggerActions]);

    // Scheduling state and handlers: manage opening and closing of schedule dialog
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
        setEditingSchedule(null);
    }, []);
    const handleScheduleCompleted = useCallback((_created: Schedule) => {
        handleCloseScheduleDialog();
    }, [handleCloseScheduleDialog]);
    const handleScheduleDeleted = useCallback((_deleted: Schedule) => {
        handleCloseScheduleDialog();
    }, [handleCloseScheduleDialog]);

    // Handler for clicking on a calendar event: open the schedule dialog populated with the clicked event's schedule
    const openEvent = useCallback((event: CalendarEvent) => {
        console.log("CalendarEvent clicked:", event);
        handleUpdateSchedule(event.schedule);
    }, [handleUpdateSchedule]);

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
        // Determine schedule type by checking meetings or runs array
        let scheduleType: ScheduleFor | undefined;
        if (event.schedule?.meetings?.length) {
            scheduleType = ScheduleFor.Meeting;
        } else if (event.schedule?.runs?.length) {
            scheduleType = ScheduleFor.Run;
        }

        let backgroundColor = palette.primary.main; // Default color
        let color = palette.primary.contrastText; // Default text color

        // Assign colors based on schedule type
        switch (scheduleType) {
            case ScheduleFor.Meeting:
                backgroundColor = palette.info.light;
                color = palette.info.contrastText;
                break;
            case ScheduleFor.Run:
                backgroundColor = palette.warning.light;
                color = palette.warning.contrastText;
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
    }, [palette.primary.main, palette.primary.contrastText, palette.info.light, palette.info.contrastText, palette.warning.light, palette.warning.contrastText, typography.caption.fontSize, typography.caption.lineHeight]);

    const calendarComponents = useMemo(function calendarComponentsMemo() {
        return {
            toolbar: (props) => <CalendarPreviewToolbar {...props} onSelectDate={handleSelectDate} />,
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
                {isScheduleDialogOpen && (
                    <Suspense fallback={null}>
                        <ScheduleUpsert
                            canSetScheduleFor={true}
                            defaultScheduleFor={activeTab === CalendarTabs.CALENDAR ? ScheduleFor.Meeting : ScheduleFor.Meeting}
                            display="Dialog"
                            isCreate={editingSchedule === null}
                            isMutate={true}
                            isOpen={isScheduleDialogOpen}
                            onCancel={handleCloseScheduleDialog}
                            onClose={handleCloseScheduleDialog}
                            onCompleted={handleScheduleCompleted}
                            onDeleted={handleScheduleDeleted}
                            overrideObject={scheduleOverrideObject}
                        />
                    </Suspense>
                )}

                <FilterDialog
                    isOpen={isFilterDialogOpen}
                    onClose={closeFilterDialog}
                    searchQuery={searchQuery}
                    setSearchQuery={setSearchQuery}
                    selectedTypes={selectedTypes}
                    setSelectedTypes={setSelectedTypes}
                    selectedTriggerEvents={selectedTriggerEvents}
                    setSelectedTriggerEvents={setSelectedTriggerEvents}
                    selectedTriggerActions={selectedTriggerActions}
                    setSelectedTriggerActions={setSelectedTriggerActions}
                    filteredEvents={filteredEvents}
                    filteredTriggers={filteredTriggers}
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
                    <BigCalendar<CalendarEvent, object>
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
                    <TriggerView 
                        triggers={filteredTriggers} 
                        loading={triggersLoading}
                        onTriggerUpdate={setTriggers}
                    />
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
