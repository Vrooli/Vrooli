import { Box, IconButton, Tooltip, styled } from "@mui/material";
import { useTheme } from "@mui/material/styles";
import { type CalendarEvent } from "@vrooli/shared";
import React, { useCallback, useRef } from "react";
import { Navigate, Views, type ToolbarProps } from "react-big-calendar";
import { useTranslation } from "react-i18next";
import { IconCommon } from "../icons/Icons.js";

const ToolbarBox = styled(Box)(({ theme }) => ({
    display: "flex",
    justifyContent: "space-between",
    alignItems: "center",
    padding: "0 16px",
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

/**
 * ToolbarProps provides: date, view, views, label, onNavigate, onView, localizer, messages
 */
export interface CalendarPreviewToolbarProps
    extends ToolbarProps<CalendarEvent, object> {
    /** Called when the user picks a date via the date input */
    onSelectDate: (date: Date) => void;
}

/**
 * Reusable toolbar for calendar previews: navigates date, selects view, and picks a date.
 */
export function CalendarPreviewToolbar({
    date,
    view,
    label,
    onNavigate,
    onView,
    onSelectDate,
}: CalendarPreviewToolbarProps) {
    const theme = useTheme();
    const { palette } = theme;
    const { t } = useTranslation();
    const dateInputRef = useRef<HTMLInputElement>(null);

    const handleDateChange = useCallback(
        (event: React.ChangeEvent<HTMLInputElement>) => {
            const inputDate = event.target.value; // YYYY-MM-DD
            const [year, month, day] = inputDate.split("-").map(Number);
            const newDate = new Date(year, month - 1, day);
            if (!isNaN(newDate.getTime())) {
                onNavigate(Navigate.DATE as any, newDate);
                onSelectDate(newDate);
            }
        },
        [onNavigate, onSelectDate],
    );

    const openDatePicker = useCallback(() => {
        if (dateInputRef.current) {
            dateInputRef.current.showPicker();
        }
    }, []);

    const navigate = useCallback(
        (action) => onNavigate(action),
        [onNavigate],
    );

    const changeView = useCallback(
        (nextView) => onView(nextView),
        [onView],
    );

    return (
        <ToolbarBox>
            <ToolbarSection>
                <Tooltip title={t("Today")}>
                    <IconButton onClick={() => navigate(Navigate.TODAY)}>
                        <IconCommon decorative fill={palette.secondary.main} name="Today" />
                    </IconButton>
                </Tooltip>
                <Tooltip title={t("Previous")}>
                    <IconButton onClick={() => navigate(Navigate.PREVIOUS)}>
                        <IconCommon decorative fill={palette.secondary.main} name="ArrowLeft" />
                    </IconButton>
                </Tooltip>
                <Tooltip title={t("Next")}>
                    <IconButton onClick={() => navigate(Navigate.NEXT)}>
                        <IconCommon decorative fill={palette.secondary.main} name="ArrowRight" />
                    </IconButton>
                </Tooltip>
            </ToolbarSection>
            <ToolbarSection>
                <Box onClick={openDatePicker} sx={dateLabelBoxStyle}>
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
                    <IconButton onClick={() => changeView(Views.MONTH)}>
                        <IconCommon decorative fill={palette.secondary.main} name="Month" />
                    </IconButton>
                </Tooltip>
                <Tooltip title={t("Week")}>
                    <IconButton onClick={() => changeView(Views.WEEK)}>
                        <IconCommon decorative fill={palette.secondary.main} name="Week" />
                    </IconButton>
                </Tooltip>
                <Tooltip title={t("Day")}>
                    <IconButton onClick={() => changeView(Views.DAY)}>
                        <IconCommon decorative fill={palette.secondary.main} name="Day" />
                    </IconButton>
                </Tooltip>
            </ToolbarSection>
        </ToolbarBox>
    );
} 
