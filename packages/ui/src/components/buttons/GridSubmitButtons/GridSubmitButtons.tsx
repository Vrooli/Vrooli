/**
 * Prompts user to select which link the new node should be added on
 */
import { CancelIcon, CreateIcon, exists, SaveIcon } from "@local/shared";
import { Box, Button, CircularProgress, Grid } from "@mui/material";
import { useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useErrorPopover } from "utils/hooks/useErrorPopover";
import { GridActionButtons } from "../GridActionButtons/GridActionButtons";
import { SideActionButtons } from "../SideActionButtons/SideActionButtons";
import { GridSubmitButtonsProps } from "../types";

export const GridSubmitButtons = ({
    disabledCancel,
    disabledSubmit,
    display,
    errors,
    isCreate,
    loading = false,
    onCancel,
    onSetSubmitting,
    onSubmit,
    sideActionButtons,
}: GridSubmitButtonsProps) => {
    const { t } = useTranslation();

    // Errors popup
    const { openPopover, Popover } = useErrorPopover({ errors, onSetSubmitting });

    const hasErrors = useMemo(() => Object.values(errors ?? {}).some((value) => exists(value)), [errors]);
    const isSubmitDisabled = useMemo(() => loading || hasErrors || (disabledSubmit === true), [disabledSubmit, hasErrors, loading]);


    const handleSubmit = useCallback((ev: React.MouseEvent | React.TouchEvent) => {
        // If formik invalid, display errors in popup
        if (hasErrors) openPopover(ev);
        else if (!disabledSubmit && typeof onSubmit === "function") onSubmit();
    }, [hasErrors, openPopover, disabledSubmit, onSubmit]);

    return (
        <GridActionButtons display={display}>
            {/* We display side actions in this component because positioning is easier */}
            {sideActionButtons ? <SideActionButtons hasGridActions={true} {...sideActionButtons} /> : null}
            <Popover />
            {/* Create/Save button. On hover or press, displays formik errors if disabled */}
            <Grid item xs={6}>
                <Box onClick={handleSubmit}>
                    <Button
                        disabled={isSubmitDisabled}
                        fullWidth
                        startIcon={loading ? <CircularProgress size={24} sx={{ color: "white" }} /> : (isCreate ? <CreateIcon /> : <SaveIcon />)}
                        variant="contained"
                    >{t(isCreate ? "Create" : "Save")}</Button>
                </Box>
            </Grid>
            {/* Cancel button */}
            <Grid item xs={6}>
                <Button
                    disabled={loading || (disabledCancel !== undefined ? disabledCancel : false)}
                    fullWidth
                    onClick={onCancel}
                    startIcon={<CancelIcon />}
                    variant="outlined"
                >{t("Cancel")}</Button>
            </Grid>
        </GridActionButtons>
    );
};
