import { ChatParticipant } from "@local/shared";
import { Avatar, IconButton, Stack, Typography, useTheme } from "@mui/material";
import { RelationshipItemUser } from "components/lists/types";
import { TextShrink } from "components/text/TextShrink/TextShrink";
import { useField, useFormikContext } from "formik";
import { AddIcon, BotIcon, UserIcon } from "icons";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { extractImageUrl } from "utils/display/imageTools";
import { getDisplay, placeholderColor } from "utils/display/listTools";
import { ParticipantManageView } from "views/ParticipantManageView/ParticipantManageView";
import { ParticipantManageViewProps } from "views/types";
import { commonLabelProps, smallButtonProps } from "../styles";
import { ParticipantsButtonProps } from "../types";

const maxIconsDisplayed = 4;

export const ParticipantsButton = ({
    isEditing,
    objectType,
}: ParticipantsButtonProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();

    const formikContext = useFormikContext();
    const [participantsField, , participantsFieldHelpers] = useField("participants");

    const isAvailable = useMemo(() => ["Chat"].includes(objectType), [objectType]);

    const [isDialogOpen, setDialogOpen] = useState<boolean>(false);
    const openDialog = useCallback(() => { setDialogOpen(true); }, []);
    const closeDialog = useCallback(() => { setDialogOpen(false); }, []);
    const handleParticipantSelect = useCallback((participant: RelationshipItemUser) => {
        participantsFieldHelpers.setValue([...(participantsField.value ?? []), participant]);
        closeDialog();
    }, [participantsFieldHelpers, participantsField.value, closeDialog]);

    const icons = useMemo<(JSX.Element | null)[]>(() => {
        let newIcons: (JSX.Element | null)[] = [];
        let maxUserIcons = isEditing ? maxIconsDisplayed - 1 : maxIconsDisplayed;
        // If there are more members than allowed, add a "+X" icon
        const hasMoreParticipants = (participantsField.value ?? []).length > maxUserIcons;
        if (hasMoreParticipants) maxUserIcons--;
        // Add the first X members
        newIcons = (participantsField.value ?? []).slice(0, maxUserIcons).map((p: ChatParticipant) => {
            const colors = placeholderColor();
            return <Avatar
                key={p.id}
                src={extractImageUrl(p.user?.profileImage, p.user?.updated_at, 50)}
                alt={`${getDisplay(p.user).title}'s profile picture`}
                sx={{
                    backgroundColor: colors[0],
                    width: "24px",
                    height: "24px",
                    pointerEvents: "none",
                    ...(p.user?.isBot ? { borderRadius: "4px" } : {}),
                }}
            >
                {p.user?.isBot ? <BotIcon width="75%" height="75%" fill={colors[1]} /> : <UserIcon width="75%" height="75%" fill={colors[1]} />}
            </Avatar>;
        });
        // Add the "+X" icon if there are more members than allowed
        if (hasMoreParticipants) {
            newIcons.push(
                <Typography variant="body2" key="more" sx={{ width: 24, height: 24 }}>
                    {`+${participantsField.value.length - maxUserIcons}`}
                </Typography>,
            );
        }
        // Add the "Add" icon if editing
        if (isEditing) {
            newIcons.push(<AddIcon key="add" width={24} height={24} fill={palette.primary.contrastText} />);
        }
        // Add null icons to fill the remaining space up to the max
        while (newIcons.length < maxIconsDisplayed) newIcons.push(null);
        return newIcons;
    }, [isEditing, participantsField.value, palette.primary.contrastText]);


    if (!isAvailable || (!isEditing && (!Array.isArray(participantsField.value) || participantsField.value.length === 0))) return null;

    return (
        <>
            {/* Dialog for managing participants */}
            <ParticipantManageView
                display="dialog"
                isOpen={isDialogOpen}
                onClose={closeDialog}
                chat={formikContext.values as ParticipantManageViewProps["chat"]}
            />
            <Stack
                direction="column"
                alignItems="center"
                justifyContent="center"
            >
                <TextShrink id="participants" sx={{ ...commonLabelProps() }}>{t("Participant", { count: 2 })}</TextShrink>
                <IconButton
                    sx={{
                        ...smallButtonProps(isEditing, true),
                        background: palette.primary.light,
                        borderRadius: "12px",
                        color: palette.primary.contrastText,
                        position: "relative",
                    }}
                    onClick={openDialog}
                >
                    {/* Participants & add participants icons */}
                    <Stack direction="column" justifyContent="center" alignItems="center" style={{ height: "100%", width: "100%" }}>
                        <Stack direction="row" justifyContent="space-around" sx={{ gap: "2px" }}>
                            {icons[0]}
                            {icons[1]}
                        </Stack>
                        <Stack direction="row" justifyContent="space-around" sx={{ gap: "2px" }}>
                            {icons[2]}
                            {icons[3]}
                        </Stack>
                    </Stack>
                </IconButton>
            </Stack>
        </>
    );
};
