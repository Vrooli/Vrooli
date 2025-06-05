import { AvatarGroup, Tooltip } from "@mui/material";
import { type ChatParticipantShape } from "@vrooli/shared";
import { useField, useFormikContext } from "formik";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Icon, IconCommon } from "../../../icons/Icons.js";
import { extractImageUrl } from "../../../utils/display/imageTools.js";
import { ParticipantManageView } from "../../../views/ParticipantManageView/ParticipantManageView.js";
import { type ParticipantManageViewProps } from "../../../views/types.js";
import { RelationshipAvatar, RelationshipButton, RelationshipChip } from "./styles.js";
import { type ParticipantsButtonProps } from "./types.js";

const MAX_LABEL_LENGTH = 20;
const TARGET_IMAGE_SIZE = 100;
const MAX_AVATARS = 4;

export function ParticipantsButton({
    isEditing,
}: ParticipantsButtonProps) {
    const { t } = useTranslation();

    const formikContext = useFormikContext();
    const [participantsField] = useField("participants");

    const [isDialogOpen, setDialogOpen] = useState<boolean>(false);
    const openDialog = useCallback(() => { setDialogOpen(true); }, []);
    const closeDialog = useCallback(() => { setDialogOpen(false); }, []);

    const { avatars, iconInfo, participantsCount, label, tooltip } = useMemo(() => {
        const participants = (participantsField.value || []) as ChatParticipantShape[];
        if (!Array.isArray(participants) || participants.some(member => typeof member !== "object")) {
            return {
                avatars: [],
                iconInfo: null,
                participantsCount: 0,
                label: "",
                tooltip: "",
            };
        }
        const avatars = participants.slice(0, MAX_AVATARS).map(participant => {
            const imageUrl = extractImageUrl(participant.user?.profileImage, participant.user?.updatedAt, TARGET_IMAGE_SIZE);
            const isBot = participant.user?.isBot ?? false;
            return (
                <RelationshipAvatar
                    key={participant.id}
                    isBot={isBot}
                    src={imageUrl}
                >
                    {!imageUrl && <IconCommon
                        decorative
                        name={isBot ? "Bot" : "Profile"}
                    />}
                </RelationshipAvatar>
            );
        });

        const participantsCount = participants.length;
        const label = (participantsCount > 0 || !isEditing) ? `${t("Member", { count: participantsCount })}: ${participantsCount}` : "Manage participants";
        const truncatedLabel = label.length > MAX_LABEL_LENGTH ? `${label.slice(0, MAX_LABEL_LENGTH)}...` : label;
        const tooltip = isEditing ? "Manage participants" : "View participants";

        return {
            avatars,
            iconInfo: avatars.length > 0 ? undefined : { name: "Settings", type: "Common" } as const,
            participantsCount,
            label: truncatedLabel,
            tooltip,
        };
    }, [participantsField.value, isEditing, t]);

    const Avatars = useMemo(function AvatarsMemo() {
        return avatars.length > 0 ? (
            <AvatarGroup max={MAX_AVATARS} total={participantsCount}>
                {avatars}
            </AvatarGroup>
        ) : null;
    }, [avatars, participantsCount]);

    if (!isEditing && avatars.length === 0) return null;
    if (isEditing) {
        return (
            <>
                <ParticipantManageView
                    display="Dialog"
                    isEditing={isEditing}
                    isOpen={isDialogOpen}
                    onClose={closeDialog}
                    chat={formikContext.values as ParticipantManageViewProps["chat"]}
                />
                <Tooltip title={tooltip}>
                    <RelationshipButton
                        onClick={openDialog}
                        startIcon={Avatars || (iconInfo && <Icon
                            info={iconInfo}
                        />)}
                        variant="outlined"
                    >
                        {label}
                    </RelationshipButton>
                </Tooltip>
            </>
        );
    }
    return (
        <RelationshipChip
            icon={Avatars || (iconInfo && <Icon
                info={iconInfo}
            />) || undefined}
            label={label}
            onClick={openDialog}
        />
    );
}
