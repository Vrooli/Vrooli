import { exists, noop } from "@local/shared";
import { IconButton, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { FindObjectDialog } from "components/dialogs/FindObjectDialog/FindObjectDialog";
import { SelectOrCreateObjectType } from "components/dialogs/types";
import { RelationshipItemMeeting } from "components/lists/types";
import { SessionContext } from "contexts/SessionContext";
import { useField } from "formik";
import { AddIcon, TeamIcon } from "icons";
import { useCallback, useContext, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { highlightStyle } from "styles";
import { firstString } from "utils/display/stringTools";
import { getTranslation, getUserLanguages } from "utils/display/translationTools";
import { openObject } from "utils/navigation/openObject";
import { largeButtonProps } from "../styles";
import { MeetingButtonProps } from "../types";

export const MeetingButton = ({
    isEditing,
    objectType,
}: MeetingButtonProps) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const languages = useMemo(() => getUserLanguages(session), [session]);

    const [field, , helpers] = useField("meeting");

    const isAvailable = useMemo(() => ["Schedule"].includes(objectType) && ["boolean", "object"].includes(typeof field.value), [objectType, field.value]);

    // Meeting dialog
    const [isDialogOpen, setDialogOpen] = useState<boolean>(false); const handleClick = useCallback((ev: React.MouseEvent<Element>) => {
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
    const [findType, findHandleAdd, findHandleClose] = useMemo<[SelectOrCreateObjectType | null, (item: any) => unknown, () => unknown]>(() => {
        if (isDialogOpen) return ["Meeting", handleSelect, closeDialog];
        return [null, noop, noop];
    }, [isDialogOpen, handleSelect, closeDialog]);

    const { Icon, tooltip } = useMemo(() => {
        const meeting = field?.value;
        // If no data, marked as unset
        if (!meeting) return {
            Icon: AddIcon,
            tooltip: t(`MeetingTogglePress${isEditing ? "Editable" : ""}`),
        };
        const meetingName = firstString(getTranslation(meeting as RelationshipItemMeeting, languages, true).name, t("Meeting", { count: 1 }));
        return {
            Icon: TeamIcon,
            tooltip: t(`MeetingTogglePress${isEditing ? "Editable" : ""}`, { meeting: meetingName }),
        };
    }, [field?.value, isEditing, languages, t]);

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
                            {firstString(getTranslation(field?.value as RelationshipItemMeeting, languages, true).name, t("Meeting", { count: 1 }))}
                        </Typography>
                    </Stack>
                </Tooltip>
            </Stack>
        </>
    );
};
