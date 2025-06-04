import { Box, type BoxProps, Typography, styled, useTheme } from "@mui/material";
import { type FormikProps } from "formik";
import { type RefObject, useCallback, useEffect, useState } from "react";
import { useWindowSize } from "../../hooks/useWindowSize.js";
import { Icon, type IconInfo } from "../../icons/Icons.js";

type AutoSaveIndicatorProps = {
    formikRef: RefObject<FormikProps<object>>;
    savedIndicatorTimeoutMs?: number;
}

type SaveStatus = "Saving" | "Saved" | "Unsaved";

type StatusDisplay = {
    backgroundColor: string;
    iconColor: string;
    iconInfo: IconInfo;
    label: string;
    labelColor: string;
}

const DEFAULT_SAVED_INDICATOR_TIMEOUT_MS = 3000;
const CHECK_SAVE_STATUS_INTERVAL_MS = 1000;

const statusToDisplay = {
    Saving: {
        backgroundColor: "#e5f6fdbb",
        iconColor: "#0288d1",
        iconInfo: { name: "Refresh", type: "Common" },
        label: "Saving...",
        labelColor: "#014361",
    },
    Saved: {
        backgroundColor: "#edf7edbb",
        iconColor: "#2e7d32",
        iconInfo: { name: "Save", type: "Common" },
        label: "Saved",
        labelColor: "#1e4620",
    },
    Unsaved: {
        backgroundColor: "#fff4e5bb",
        iconColor: "#ed6c02",
        iconInfo: { name: "Warning", type: "Common" },
        label: "Not saved",
        labelColor: "#663c00",
    },
} as const;

interface AutoSaveAlertProps extends BoxProps {
    isLabelVisible: boolean;
    isVisible: boolean;
    status: SaveStatus;
}

const AutoSaveAlert = styled(Box, {
    shouldForwardProp: (prop) => prop !== "isLabelVisible" && prop !== "isVisible" && prop !== "status",
})<AutoSaveAlertProps>(({ isLabelVisible, isVisible, status, theme }) => ({
    display: isVisible ? "flex" : "none",
    alignItems: "center",
    justifyContent: "center",
    background: statusToDisplay[status].backgroundColor,
    borderRadius: "4px",
    // eslint-disable-next-line no-magic-numbers
    padding: isLabelVisible ? theme.spacing(0.75) : `${theme.spacing(0.5)} ${theme.spacing(1)}`,
    "& .save-alert-icon": {
        paddingTop: 0,
        paddingBottom: 0,
    },
    "& .save-alert-label": {
        display: isLabelVisible ? "block" : "none",
        padding: 0,
        marginLeft: theme.spacing(1),
    },
}));

export function AutoSaveIndicator({
    formikRef,
    savedIndicatorTimeoutMs = DEFAULT_SAVED_INDICATOR_TIMEOUT_MS,
}: AutoSaveIndicatorProps) {
    const { breakpoints } = useTheme();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);

    const [saveStatus, setSaveStatus] = useState<SaveStatus>("Saved");
    const [isVisible, setIsVisible] = useState(false);
    const [showLabelOnMobile, setShowLabelOnMobile] = useState(false);
    const toggleShowLabelOnMobile = useCallback(function toggleShowLabelOnMobileCallback() {
        setShowLabelOnMobile((prev) => !prev);
    }, []);
    const handleKeyDown = useCallback(function handleKeyDownCallback(event: React.KeyboardEvent<HTMLDivElement>) {
        if (event.key === "Enter" || event.key === " ") {
            toggleShowLabelOnMobile();
        }
    }, [toggleShowLabelOnMobile]);

    useEffect(function checkStatusTimeout() {
        let timeoutId: NodeJS.Timeout;

        function checkFormStatus() {
            if (formikRef.current) {
                const { dirty, isSubmitting } = formikRef.current;

                if (isSubmitting) {
                    setSaveStatus("Saving");
                    setIsVisible(true);
                } else if (dirty) {
                    setSaveStatus("Unsaved");
                    setIsVisible(true);
                } else {
                    setSaveStatus("Saved");
                    // Hide the indicator after 3 seconds when saved
                    timeoutId = setTimeout(() => setIsVisible(false), savedIndicatorTimeoutMs);
                }
            }
        }

        const intervalId = setInterval(checkFormStatus, CHECK_SAVE_STATUS_INTERVAL_MS);

        return () => {
            clearInterval(intervalId);
            clearTimeout(timeoutId);
        };
    }, [formikRef, savedIndicatorTimeoutMs]);

    return (
        <AutoSaveAlert
            aria-label={statusToDisplay[saveStatus].label}
            aria-pressed={saveStatus === "Saved"}
            isLabelVisible={!isMobile || showLabelOnMobile}
            isVisible={isVisible}
            onClick={toggleShowLabelOnMobile}
            onKeyDown={handleKeyDown}
            role="button"
            status={saveStatus}
        >
            <Icon
                className="save-alert-icon"
                decorative
                fill={statusToDisplay[saveStatus].iconColor}
                info={statusToDisplay[saveStatus].iconInfo}
                size={20}
            />
            <Typography
                className="save-alert-label"
                color={statusToDisplay[saveStatus].labelColor}
                variant="body2"
            >
                {statusToDisplay[saveStatus].label}
            </Typography>
        </AutoSaveAlert>
    );
}
