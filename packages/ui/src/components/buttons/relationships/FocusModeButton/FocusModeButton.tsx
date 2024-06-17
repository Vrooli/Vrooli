import { LINKS, noop } from "@local/shared";
import { IconButton, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { FindObjectDialog } from "components/dialogs/FindObjectDialog/FindObjectDialog";
import { SelectOrCreateObjectType } from "components/dialogs/types";
import { RelationshipItemFocusMode } from "components/lists/types";
import { SessionContext } from "contexts/SessionContext";
import { useField } from "formik";
import { AddIcon, FocusModeIcon } from "icons";
import { useCallback, useContext, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { highlightStyle } from "styles";
import { getFocusModeInfo } from "utils/authentication/session";
import { largeButtonProps } from "../styles";
import { FocusModeButtonProps } from "../types";

export const FocusModeButton = ({
    isEditing,
    objectType,
}: FocusModeButtonProps) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const { t } = useTranslation();

    const { all: allFocusModes } = useMemo(() => getFocusModeInfo(session), [session]);

    // Schedules set the focus mode directly, while reminders use the focus mode to set the reminder list
    const [focusModeField, , focusModeHelpers] = useField("focusMode");
    const [reminderListField, , reminderListHelpers] = useField("reminderList");

    const isAvailable = useMemo(() =>
        (objectType === "Reminder" && reminderListField.value !== undefined) ||
        (objectType === "Schedule" && focusModeField.value !== undefined)
        , [objectType, reminderListField.value, focusModeField.value]);

    // Focus mode dialog
    const [isDialogOpen, setDialogOpen] = useState<boolean>(false); const handleClick = useCallback((ev: React.MouseEvent<Element>) => {
        if (!isAvailable) return;
        ev.stopPropagation();
        // Find current focus mode, either from focusModeField or by finding the correct focus mode in allFocusModes that has the reminder list
        const focusMode = focusModeField?.value ?? allFocusModes.find(focusMode => focusMode.reminderList?.id === reminderListField?.value?.id);
        // If not editing, navigate to display settings
        if (!isEditing) {
            if (focusMode) setLocation(LINKS.SettingsFocusModes);
        }
        else {
            // If focus mode was set, and this is a schedule, remove focus mode. 
            // We don't remove for reminders because they are required to have a focus mode
            if (focusMode && focusModeField.value !== undefined && focusModeHelpers) {
                focusModeHelpers.setValue(null);
            }
            // Otherwise, open select dialog
            else setDialogOpen(true);
        }
    }, [isAvailable, focusModeField.value, allFocusModes, isEditing, reminderListField?.value, setLocation, focusModeHelpers]);
    const closeDialog = useCallback(() => { setDialogOpen(false); }, [setDialogOpen]);
    const handleSelect = useCallback((focusMode: RelationshipItemFocusMode) => {
        if (focusModeField.value !== undefined && focusModeHelpers) focusModeHelpers.setValue(focusMode);
        else if (reminderListField.value !== undefined && reminderListHelpers) {
            // Add focus mode to reminder list
            const { reminderList, ...rest } = focusMode;
            reminderListHelpers.setValue({ ...reminderList, focusMode: rest });
        }
        closeDialog();
    }, [focusModeField.value, focusModeHelpers, reminderListField.value, reminderListHelpers, closeDialog]);

    // FindObjectDialog
    const [findType, findHandleAdd, findHandleClose] = useMemo<[SelectOrCreateObjectType | null, (item: any) => unknown, () => unknown]>(() => {
        if (isDialogOpen) return ["FocusMode", handleSelect, closeDialog];
        return [null, noop, noop];
    }, [isDialogOpen, handleSelect, closeDialog]);

    const { Icon, tooltip } = useMemo(() => {
        const focusMode = focusModeField?.value ?? reminderListField?.value?.focusMode ?? allFocusModes.find(focusMode => focusMode.reminderList?.id === reminderListField?.value?.id);
        // If no data, marked as unset
        if (!focusMode) return {
            Icon: AddIcon,
            tooltip: t(`FocusModeNoneTogglePress${isEditing ? "Editable" : ""}`),
        };
        const focusModeName = focusMode?.name ?? "";
        return {
            Icon: FocusModeIcon,
            tooltip: t(`FocusModeTogglePress${isEditing ? "Editable" : ""}`, { focusMode: focusModeName }),
        };
    }, [focusModeField?.value, reminderListField?.value?.focusMode, reminderListField?.value?.id, allFocusModes, t, isEditing]);

    // If not available, return null
    if (!isAvailable || (!isEditing && !Icon)) return null;
    return (
        <>
            {/* Popup for selecting focus mode */}
            {findType && <FindObjectDialog
                find="List"
                isOpen={Boolean(findType)}
                handleCancel={findHandleClose}
                handleComplete={findHandleAdd}
                limitTo={[findType]}
            />}
            <Stack
                direction="column"
                alignItems="center"
                justifyContent="center"
                sx={{
                    marginTop: "auto",
                    cursor: "pointer",
                }}
            >
                <Tooltip title={tooltip}>
                    <Stack
                        direction="row"
                        justifyContent="center"
                        alignItems="center"
                        onClick={handleClick}
                        sx={{
                            borderRadius: 8,
                            paddingRight: 2,
                            ...largeButtonProps(isEditing, true),
                            ...highlightStyle(palette.primary.light, !isEditing),
                        }}
                    >
                        {Icon && (
                            <IconButton>
                                <Icon width={"48px"} height={"48px"} fill="white" />
                            </IconButton>
                        )}
                        <Typography variant="body1" sx={{ color: "white" }}>
                            {focusModeField?.value?.name ?? allFocusModes.find(focusMode => focusMode.reminderList?.id === reminderListField?.value?.id)?.name ?? t("FocusMode", { count: 1 })}
                        </Typography>
                    </Stack>
                </Tooltip>
            </Stack>
        </>
    );
};
