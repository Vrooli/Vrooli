import Box from "@mui/material/Box";
import Grid from "@mui/material/Grid";
import { styled, useTheme } from "@mui/material/styles";
import type { BoxProps } from "@mui/material/Box";
import { exists } from "@vrooli/shared";
import { useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useErrorPopover } from "../../hooks/useErrorPopover.js";
import { useKeyboardOpen } from "../../hooks/useKeyboardOpen.js";
import { useWindowSize } from "../../hooks/useWindowSize.js";
import { IconCommon } from "../../icons/Icons.js";
import { type SxType } from "../../types.js";
import { BottomActionsGrid } from "./BottomActionsGrid.js";
import { Button } from "./Button.js";
import { type BottomActionsButtonsProps } from "./types.js";

interface SideActionsBoxProps extends BoxProps {
    hideButtons: boolean;
    isKeyboardOpen: boolean;
}

const SideActionsBox = styled(Box, {
    shouldForwardProp: (prop) => prop !== "hideButtons" && prop !== "isKeyboardOpen",
})<SideActionsBoxProps>(({ hideButtons, isKeyboardOpen }) => ({
    position: "absolute",
    top: isKeyboardOpen ? 0 : "-64px",
    justifyContent: "flex-end",
    display: "flex",
    flexDirection: "row",
    gap: "16px",
    alignItems: "end",
    paddingRight: "calc(32px + env(safe-area-inset-left))",
    paddingLeft: "calc(32px + env(safe-area-inset-right))",
    height: "calc(64px)",
    width: "100%",
    pointerEvents: "none",
    "& > *": {
        marginBottom: !hideButtons ? "4px !important" : "calc(16px + env(safe-area-inset-bottom)) !important",
        pointerEvents: "auto",
    },
}));

export function BottomActionsButtons({
    disabledCancel,
    disabledSubmit,
    display,
    errors,
    hideButtons = false,
    hideTextOnMobile = false,
    isCreate,
    loading = false,
    onCancel,
    onSetSubmitting,
    onSubmit,
    sideActionButtons,
}: BottomActionsButtonsProps) {
    const { t } = useTranslation();
    const { breakpoints } = useTheme();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.sm);
    const isKeyboardOpen = useKeyboardOpen();

    const buttonStyle = useMemo<SxType>(function buttonStyleMemo() {
        // Removing padding when the button doesn't display text
        const spanStyle = hideTextOnMobile && isMobile ? { marginLeft: 0, marginRight: 0 } : {};
        return { "& span": spanStyle } as SxType;
    }, [hideTextOnMobile, isMobile]);

    const { openPopover, Popover } = useErrorPopover({ errors, onSetSubmitting });

    const hasErrors = useMemo(() => Object.values(errors ?? {}).some((value) => exists(value)), [errors]);
    const isSubmitDisabled = useMemo(() => loading || hasErrors || (disabledSubmit === true), [disabledSubmit, hasErrors, loading]);


    const handleSubmit = useCallback(function handleSubmitCallback(ev: React.MouseEvent | React.TouchEvent) {
        // If formik invalid, display errors in popup
        if (hasErrors) openPopover(ev);
        else if (!disabledSubmit && typeof onSubmit === "function") onSubmit();
    }, [hasErrors, openPopover, disabledSubmit, onSubmit]);

    const handleCancel = useCallback(function handleCancelCallback(event: React.MouseEvent | React.TouchEvent) {
        event.preventDefault();
        event.stopPropagation();
        onCancel();
    }, [onCancel]);

    return (
        <BottomActionsGrid display={display} data-testid="bottom-actions-grid">
            <Popover />
            {/* Side action buttons displayed above grid options */}
            {sideActionButtons ? <SideActionsBox
                hideButtons={hideButtons}
                isKeyboardOpen={isKeyboardOpen}
                data-testid="side-actions-box"
            >
                {sideActionButtons}
            </SideActionsBox> : null}
            {/* Create/Save button. On hover or press, displays formik errors if disabled */}
            {!hideButtons ? <Grid item xs={6}>
                <Box onClick={handleSubmit} data-testid="submit-button-wrapper">
                    <Button
                        aria-label={t(isCreate ? "Create" : "Save")}
                        disabled={isSubmitDisabled}
                        isLoading={loading}
                        startIcon={<IconCommon
                            decorative
                            name={isCreate ? "Create" : "Save"}
                        />}
                        style={buttonStyle}
                        variant="primary"
                        fullWidth
                        data-testid="submit-button"
                    >{hideTextOnMobile && isMobile ? "" : t(isCreate ? "Create" : "Save")}</Button>
                </Box>
            </Grid> : null}
            {/* Cancel button */}
            {!hideButtons ? <Grid item xs={6}>
                <Button
                    aria-label={t("Cancel")}
                    disabled={loading || (disabledCancel !== undefined ? disabledCancel : false)}
                    fullWidth
                    onClick={handleCancel}
                    startIcon={<IconCommon
                        decorative
                        name="Cancel"
                    />}
                    variant="outline"
                    style={buttonStyle}
                    data-testid="cancel-button"
                >{hideTextOnMobile && isMobile ? "" : t("Cancel")}</Button>
            </Grid> : null}
        </BottomActionsGrid>
    );
}
