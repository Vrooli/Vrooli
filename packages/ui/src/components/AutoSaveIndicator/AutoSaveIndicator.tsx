import { Box, BoxProps, Typography, styled, useTheme } from "@mui/material";
import { FormikProps } from "formik";
import { useWindowSize } from "hooks/useWindowSize.js";
import { RefreshIcon, SaveIcon, WarningIcon } from "icons/common.js";
import { RefObject, useCallback, useEffect, useState } from "react";

type AutoSaveIndicatorProps = {
    formikRef: RefObject<FormikProps<object>>;
    savedIndicatorTimeoutMs?: number;
}

type SaveStatus = "Saving" | "Saved" | "Unsaved";

const DEFAULT_SAVED_INDICATOR_TIMEOUT_MS = 3000;
const CHECK_SAVE_STATUS_INTERVAL_MS = 1000;

const statusToIconColor = {
    Saving: "#0288d1",
    Saved: "#2e7d32",
    Unsaved: "#ed6c02",
} as const;
const statusToBackgroundColor = {
    Saving: "#e5f6fdbb",
    Saved: "#edf7edbb",
    Unsaved: "#fff4e5bb",
} as const;
const statusToLabelColor = {
    Saving: "#014361",
    Saved: "#1e4620",
    Unsaved: "#663c00",
} as const;
const statusToLabel = {
    Saving: "Saving...",
    Saved: "Saved",
    Unsaved: "Not saved",
} as const;
const statusToIcon = {
    Saving: RefreshIcon,
    Saved: SaveIcon,
    Unsaved: WarningIcon,
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
    background: statusToBackgroundColor[status],
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

    const Icon = statusToIcon[saveStatus];
    return (
        <AutoSaveAlert
            isLabelVisible={!isMobile || showLabelOnMobile}
            isVisible={isVisible}
            onClick={toggleShowLabelOnMobile}
            onKeyDown={handleKeyDown}
            role="button"
            status={saveStatus}
        >
            <Icon
                className="save-alert-icon"
                width={20}
                height={20}
                fill={statusToIconColor[saveStatus]}
            />
            <Typography
                className="save-alert-label"
                variant="body2"
                color={statusToLabelColor[saveStatus]}
            >
                {statusToLabel[saveStatus]}
            </Typography>
        </AutoSaveAlert>
    );
}
