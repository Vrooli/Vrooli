import { exists, OrganizationIcon, useLocation } from "@local/shared";
import { Stack, Tooltip, useTheme } from "@mui/material";
import { ColorIconButton } from "components/buttons/ColorIconButton/ColorIconButton";
import { FindObjectDialog } from "components/dialogs/FindObjectDialog/FindObjectDialog";
import { SelectOrCreateObjectType } from "components/dialogs/types";
import { RelationshipItemMeeting } from "components/lists/types";
import { TextShrink } from "components/text/TextShrink/TextShrink";
import { useField } from "formik";
import { useCallback, useContext, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { firstString } from "utils/display/stringTools";
import { getTranslation, getUserLanguages } from "utils/display/translationTools";
import { openObject } from "utils/navigation/openObject";
import { SessionContext } from "utils/SessionContext";
import { commonButtonProps, commonIconProps, commonLabelProps } from "../styles";
import { MeetingButtonProps } from "../types";

export function MeetingButton({
    isEditing,
    objectType,
    zIndex,
}: MeetingButtonProps) {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const languages = useMemo(() => getUserLanguages(session), [session]);

    const [field, , helpers] = useField("meeting");

    const isAvailable = useMemo(() => ["Schedule"].includes(objectType) && exists(field.value), [objectType, field.value]);

    // Meeting dialog
    const [isDialogOpen, setDialogOpen] = useState<boolean>(false); const handleClick = useCallback((ev: React.MouseEvent<any>) => {
        if (!isAvailable) return;
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
    }, [isAvailable, field?.value, isEditing, setLocation, helpers]);
    const closeDialog = useCallback(() => { setDialogOpen(false); }, [setDialogOpen]);
    const handleSelect = useCallback((meeting: RelationshipItemMeeting) => {
        const meetingId = field?.value?.id;
        if (meeting?.id === meetingId) return;
        exists(helpers) && helpers.setValue(meeting);
        closeDialog();
    }, [field?.value?.id, helpers, closeDialog]);

    // FindObjectDialog
    const [findType, findHandleAdd, findHandleClose] = useMemo<[SelectOrCreateObjectType | null, (item: any) => any, () => void]>(() => {
        if (isDialogOpen) return ["Meeting", handleSelect, closeDialog];
        return [null, () => { }, () => { }];
    }, [isDialogOpen, handleSelect, closeDialog]);

    const { Icon, tooltip } = useMemo(() => {
        const meeting = field?.value;
        // If no data, marked as unset
        if (!meeting) return {
            Icon: null,
            tooltip: isEditing ? "" : "Press to assign to a meeting",
        };
        const meetingName = firstString(getTranslation(meeting as RelationshipItemMeeting, languages, true).name, "Focus Mode");
        return {
            Icon: OrganizationIcon,
            tooltip: `Focus Mode: ${meetingName}`,
        };
    }, [isEditing, languages, field?.value]);

    // If not available, return null
    if (!isAvailable || (!isEditing && !Icon)) return null;
    // Return button with label on top
    return (
        <>
            {/* Popup for selecting meeting */}
            {findType && <FindObjectDialog
                find="List"
                isOpen={Boolean(findType)}
                handleCancel={findHandleClose}
                handleComplete={findHandleAdd}
                limitTo={[findType]}
                zIndex={zIndex + 1}
            />}
            <Stack
                direction="column"
                alignItems="center"
                justifyContent="center"
            >
                <TextShrink id="meeting" sx={{ ...commonLabelProps() }}>
                    {t("Meeting", { count: 1 })}
                </TextShrink>
                <Tooltip title={tooltip}>
                    <ColorIconButton
                        background={palette.primary.light}
                        sx={{ ...commonButtonProps(isEditing, true) }}
                        onClick={handleClick}
                    >
                        {Icon && <Icon {...commonIconProps()} />}
                    </ColorIconButton>
                </Tooltip>
            </Stack>
        </>
    );
}
