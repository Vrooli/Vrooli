import { LINKS } from "@local/shared";
import { Tooltip } from "@mui/material";
import { useField } from "formik";
import { useCallback, useContext, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { SessionContext } from "../../../contexts/session.js";
import { Icon } from "../../../icons/Icons.js";
import { useLocation } from "../../../route/router.js";
import { useFocusModes } from "../../../stores/focusModeStore.js";
import { FindObjectDialog } from "../../dialogs/FindObjectDialog/FindObjectDialog.js";
import { RelationshipItemFocusMode } from "../../lists/types.js";
import { RelationshipButton, RelationshipChip } from "./styles.js";
import { FocusModeButtonProps } from "./types.js";

const limitTo = ["FocusMode"] as const;

export function FocusModeButton({
    isEditing,
}: FocusModeButtonProps) {
    const session = useContext(SessionContext);
    const [, setLocation] = useLocation();
    const { t } = useTranslation();

    const focusModeInfo = useFocusModes(session);

    // Schedules set the focus mode directly, while reminders use the focus mode to set the reminder list
    const [focusModeField, , focusModeHelpers] = useField("focusMode");
    const [reminderListField, , reminderListHelpers] = useField("reminderList");

    // Focus mode dialog
    const [isDialogOpen, setDialogOpen] = useState<boolean>(false);
    const handleClick = useCallback((ev: React.MouseEvent<Element>) => {
        ev.stopPropagation();
        // Find current focus mode, either from focusModeField or by finding the correct focus mode in allFocusModes that has the reminder list
        const focusMode = focusModeField?.value ?? focusModeInfo.all.find(focusMode => focusMode.reminderList?.id === reminderListField?.value?.id);
        // If not editing, navigate to display settings
        if (!isEditing) {
            if (focusMode) setLocation(LINKS.SettingsFocusModes);
        }
        else {
            setDialogOpen(true);
        }
    }, [focusModeField?.value, focusModeInfo.all, isEditing, reminderListField?.value, setLocation]);
    const closeDialog = useCallback(() => { setDialogOpen(false); }, [setDialogOpen]);
    const handleSelect = useCallback((focusMode: RelationshipItemFocusMode | null | undefined) => {
        if (!focusMode) return;
        if (focusModeField.value !== undefined && focusModeHelpers) focusModeHelpers.setValue(focusMode);
        else if (reminderListField.value !== undefined && reminderListHelpers) {
            // Add focus mode to reminder list
            const { reminderList, ...rest } = focusMode;
            reminderListHelpers.setValue({ ...reminderList, focusMode: rest });
        }
        closeDialog();
    }, [focusModeField.value, focusModeHelpers, reminderListField.value, reminderListHelpers, closeDialog]);

    const { iconInfo, label, tooltip } = useMemo(() => {
        const focusMode = focusModeField?.value ?? reminderListField?.value?.focusMode ?? focusModeInfo.all.find(focusMode => focusMode.reminderList?.id === reminderListField?.value?.id);
        // If no data
        if (!focusMode) {
            // If not editing, don't show anything
            if (!isEditing) return {
                Icon: undefined,
                label: null,
                tooltip: null,
            };
            // Otherwise, mark as unset
            return {
                iconInfo: { name: "Add", type: "Common" } as const,
                label: "Add focus mode",
                tooltip: t(`FocusModeNoneTogglePress${isEditing ? "Editable" : ""}`),
            };
        }
        const focusModeName = focusMode?.name ?? null;
        return {
            iconInfo: { name: "FocusMode", type: "Common" } as const,
            label: focusModeName ? `Focus mode: ${focusModeName}` : "Focus mode",
            tooltip: t(`FocusModeTogglePress${isEditing ? "Editable" : ""}`, { focusMode: focusModeName || "" }),
        };
    }, [focusModeField?.value, reminderListField?.value?.focusMode, reminderListField?.value?.id, focusModeInfo.all, t, isEditing]);

    // If not editing and no focus mode, return null
    if (!isEditing && !iconInfo) return null;
    // If editing, return button and popups for choosing owner type and owner
    if (isEditing) {
        return (
            <>
                {/* Popup for selecting focus mode */}
                {isDialogOpen && <FindObjectDialog
                    find="List"
                    isOpen={isDialogOpen}
                    handleCancel={closeDialog}
                    handleComplete={handleSelect as (data: object) => unknown}
                    limitTo={limitTo}
                />}
                <Tooltip title={tooltip}>
                    <RelationshipButton
                        onClick={handleClick}
                        startIcon={iconInfo && <Icon info={iconInfo} />}
                        variant="outlined"
                    >
                        {label}
                    </RelationshipButton>
                </Tooltip>
            </>
        );
    }
    // Otherwise, return chip
    return (
        <RelationshipChip
            icon={iconInfo && <Icon info={iconInfo} />}
            label={label}
            onClick={handleClick}
        />
    );
}
