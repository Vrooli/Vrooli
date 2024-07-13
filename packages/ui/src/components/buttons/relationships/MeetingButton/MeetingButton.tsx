import { exists, noop } from "@local/shared";
import { Tooltip } from "@mui/material";
import { FindObjectDialog } from "components/dialogs/FindObjectDialog/FindObjectDialog";
import { SelectOrCreateObjectType } from "components/dialogs/types";
import { RelationshipItemMeeting } from "components/lists/types";
import { SessionContext } from "contexts/SessionContext";
import { useField } from "formik";
import { AddIcon, TeamIcon } from "icons";
import { useCallback, useContext, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { firstString } from "utils/display/stringTools";
import { getTranslation, getUserLanguages } from "utils/display/translationTools";
import { openObject } from "utils/navigation/openObject";
import { RelationshipButton, RelationshipChip } from "../styles";
import { MeetingButtonProps } from "../types";

export function MeetingButton({
    isEditing,
}: MeetingButtonProps) {
    const session = useContext(SessionContext);
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const languages = useMemo(() => getUserLanguages(session), [session]);

    const [field, , helpers] = useField("meeting");

    // Meeting dialog
    const [isDialogOpen, setDialogOpen] = useState<boolean>(false);
    const handleClick = useCallback((ev: React.MouseEvent<Element>) => {
        ev.stopPropagation();
        const meeting = field?.value;
        // If not editing, navigate to meeting
        if (!isEditing) {
            if (meeting) openObject(meeting, setLocation);
        }
        else {
            // If meeting was set, remove
            if (meeting) {
                exists(helpers) && helpers.setValue(null);
            }
            // Otherwise, open select dialog
            else setDialogOpen(true);
        }
    }, [field?.value, isEditing, setLocation, helpers]);
    const closeDialog = useCallback(() => { setDialogOpen(false); }, [setDialogOpen]);
    const handleSelect = useCallback((meeting: RelationshipItemMeeting) => {
        const meetingId = field?.value?.id;
        if (meeting?.id === meetingId) return;
        exists(helpers) && helpers.setValue(meeting);
        closeDialog();
    }, [field?.value?.id, helpers, closeDialog]);

    // FindObjectDialog
    const [findType, findHandleAdd, findHandleClose] = useMemo<[SelectOrCreateObjectType | null, (item: any) => unknown, () => unknown]>(() => {
        if (isDialogOpen) return ["Meeting", handleSelect, closeDialog];
        return [null, noop, noop];
    }, [isDialogOpen, handleSelect, closeDialog]);
    const limitTo = useMemo(function limitToMemo() {
        return findType ? [findType] : [];
    }, [findType]);

    const { Icon, label, tooltip } = useMemo(() => {
        const meeting = field?.value;
        // If no data,
        if (!meeting) {
            // If not editing, don't show anything
            if (!isEditing) return {
                Icon: undefined,
                label: null,
                tooltip: null,
            };
            // Otherwise, mark as unset
            return {
                Icon: AddIcon,
                label: "Add meeting",
                tooltip: t(`MeetingTogglePress${isEditing ? "Editable" : ""}`),
            };
        }
        const meetingName = firstString(getTranslation(meeting as RelationshipItemMeeting, languages, true).name, t("Meeting", { count: 1 }));
        return {
            Icon: TeamIcon,
            label: firstString(getTranslation(field?.value as RelationshipItemMeeting, languages, true).name, t("Meeting", { count: 1 })),
            tooltip: t(`MeetingTogglePress${isEditing ? "Editable" : ""}`, { meeting: meetingName }),
        };
    }, [field?.value, isEditing, languages, t]);

    // If not editing and no focus mode, return null
    if (!isEditing && !Icon) return null;
    // If editing, return button and popups for choosing owner type and owner
    if (isEditing) {
        return (
            <>
                {/* Popup for selecting focus mode */}
                {findType && <FindObjectDialog
                    find="List"
                    isOpen={Boolean(findType)}
                    handleCancel={findHandleClose}
                    handleComplete={findHandleAdd}
                    limitTo={limitTo}
                />}
                <Tooltip title={tooltip}>
                    <RelationshipButton
                        onClick={handleClick}
                        startIcon={Icon && <Icon />}
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
            icon={Icon && <Icon />}
            label={label}
            onClick={handleClick}
        />
    );
}
