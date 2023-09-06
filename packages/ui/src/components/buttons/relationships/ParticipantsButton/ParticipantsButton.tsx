import { User } from "@local/shared";
import { Box, IconButton, Stack, Typography, useTheme } from "@mui/material";
import { RelationshipItemUser } from "components/lists/types";
import { TextShrink } from "components/text/TextShrink/TextShrink";
import { useField } from "formik";
import { AddIcon, BotIcon, UserIcon } from "icons";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { SvgComponent } from "types";
import { commonIconProps, commonLabelProps, smallButtonProps } from "../styles";
import { ParticipantsButtonProps } from "../types";

const maxIconsDisplayed = 4;

type IconType = SvgComponent | string | null;

const renderIcon = (Icon: IconType, key: number) => {
    if (Icon === null) return <Box sx={{ width: 24, height: 24 }} key={key} />;
    if (typeof Icon === "string") return <Typography variant="body2" key={key} sx={{ width: 24, height: 24 }}>{Icon}</Typography>;
    return <Icon key={key} {...commonIconProps()} width={24} height={24} />;
};

export const ParticipantsButton = ({
    isEditing,
    objectType,
}: ParticipantsButtonProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();

    const [idField] = useField("id");
    const [participantsField, , participantsFieldHelpers] = useField("participants");

    const isAvailable = useMemo(() => ["Chat"].includes(objectType), [objectType]);

    const [isDialogOpen, setDialogOpen] = useState<boolean>(false);
    const openDialog = useCallback(() => { setDialogOpen(true); }, []);
    const closeDialog = useCallback(() => { setDialogOpen(false); }, []);
    const handleParticipantSelect = useCallback((participant: RelationshipItemUser) => {
        participantsFieldHelpers.setValue([...(participantsField.value ?? []), participant]);
        closeDialog();
    }, [participantsFieldHelpers, participantsField.value, closeDialog]);

    const searchData = useMemo(() => ({
        searchType: "User" as const,
        where: { participantInChatId: participantsField?.value?.id },
    }), [participantsField?.value?.id]);

    const icons = useMemo<[IconType, IconType, IconType, IconType]>(() => {
        let newIcons: IconType[] = [];
        let maxUserIcons = isEditing ? maxIconsDisplayed - 1 : maxIconsDisplayed;
        // If there are more participants than allowed, add a "+X" icon
        const hasMoreParticipants = (participantsField.value ?? []).length > maxUserIcons;
        if (hasMoreParticipants) maxUserIcons--;
        // Add the first X participants
        newIcons = (participantsField.value ?? []).slice(0, maxUserIcons).map((user: User) => user.isBot ? BotIcon : UserIcon); // TODO Replace with User's profile pic in the future
        // Add the "+X" icon if there are more participants than allowed
        if (hasMoreParticipants) newIcons.push(`+${participantsField.value.length - maxUserIcons}`);
        // Add the "Add" or "Lock" icon if editing
        if (isEditing) {
            newIcons.push(AddIcon);
        }
        // Add null icons to fill the remaining space up to the max
        while (newIcons.length < maxIconsDisplayed) newIcons.push(null);
        return newIcons as [IconType, IconType, IconType, IconType];
    }, [participantsField.value, isEditing]);

    if (!isAvailable || (!isEditing && (!Array.isArray(participantsField.value) || participantsField.value.length === 0))) return null;

    return (
        <>
            {/* Dialog for managing participants */}
            {/* <ParticipantManageView
                isOpen={isDialogOpen}
                onClose={closeDialog}
                chatId={idField.value ?? ""}
            /> */}
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
                        color: "white",
                        position: "relative",
                    }}
                    onClick={openDialog}
                >
                    {/* Participants & add participants icons */}
                    <Stack direction="column" justifyContent="center" alignItems="center" style={{ height: "100%", width: "100%" }}>
                        <Stack direction="row" justifyContent="space-around">
                            {renderIcon(icons[0], 0)}
                            {renderIcon(icons[1], 1)}
                        </Stack>
                        <Stack direction="row" justifyContent="space-around">
                            {renderIcon(icons[2], 2)}
                            {renderIcon(icons[3], 3)}
                        </Stack>
                    </Stack>
                </IconButton>
            </Stack>
        </>
    );
};
