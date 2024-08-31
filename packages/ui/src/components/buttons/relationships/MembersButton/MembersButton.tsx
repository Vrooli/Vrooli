import { MemberShape } from "@local/shared";
import { AvatarGroup, Tooltip } from "@mui/material";
import { useField, useFormikContext } from "formik";
import { SettingsIcon, UserIcon } from "icons";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { extractImageUrl } from "utils/display/imageTools";
import { placeholderColor } from "utils/display/listTools";
import { MemberManageView } from "views/MemberManageView/MemberManageView";
import { MemberManageViewProps } from "views/types";
import { RelationshipAvatar, RelationshipButton, RelationshipChip } from "../styles";
import { MembersButtonProps } from "../types";

const MAX_LABEL_LENGTH = 20;
const TARGET_IMAGE_SIZE = 100;
const MAX_AVATARS = 4;

export function MembersButton({
    isEditing,
}: MembersButtonProps) {
    const { t } = useTranslation();

    const formikContext = useFormikContext();
    const [membersField] = useField("members");

    const [isDialogOpen, setDialogOpen] = useState<boolean>(false);
    const openDialog = useCallback(() => { setDialogOpen(true); }, []);
    const closeDialog = useCallback(() => { setDialogOpen(false); }, []);

    const { avatars, Icon, membersCount, label, tooltip } = useMemo(() => {
        const members = (membersField.value || []) as MemberShape[];
        if (!Array.isArray(members) || members.some(member => typeof member !== "object")) {
            return {
                avatars: [],
                Icon: null,
                membersCount: 0,
                label: "",
                tooltip: "",
            };
        }
        const avatars = members.slice(0, MAX_AVATARS).map(member => {
            const imageUrl = extractImageUrl(member.user?.profileImage, member.user?.updated_at, TARGET_IMAGE_SIZE);
            const isBot = member.user?.isBot ?? false;
            return (
                <RelationshipAvatar
                    key={member.id}
                    isBot={isBot}
                    src={imageUrl}
                    profileColors={placeholderColor(member.user?.id)}
                >
                    {!imageUrl && <UserIcon />}
                </RelationshipAvatar>
            );
        });

        const membersCount = members.length;
        const label = (membersCount > 0 || !isEditing) ? `${t("Member", { count: membersCount })}: ${membersCount}` : "Manage members";
        const truncatedLabel = label.length > MAX_LABEL_LENGTH ? `${label.slice(0, MAX_LABEL_LENGTH)}...` : label;
        const tooltip = isEditing ? "Manage members" : "View members";

        return {
            avatars,
            Icon: avatars.length > 0 ? undefined : SettingsIcon,
            membersCount,
            label: truncatedLabel,
            tooltip,
        };
    }, [isEditing, membersField.value, t]);

    const Avatars = useMemo(function AvatarsMemo() {
        return avatars.length > 0 ? (
            <AvatarGroup max={MAX_AVATARS} total={membersCount}>
                {avatars}
            </AvatarGroup>
        ) : null;
    }, [avatars, membersCount]);

    if (!isEditing && avatars.length === 0) return null;
    if (isEditing) {
        return (
            <>
                <MemberManageView
                    display="dialog"
                    isEditing={isEditing}
                    isOpen={isDialogOpen}
                    onClose={closeDialog}
                    team={formikContext.values as MemberManageViewProps["team"]}
                />
                <Tooltip title={tooltip}>
                    <RelationshipButton
                        onClick={openDialog}
                        startIcon={Avatars || (Icon && <Icon />)}
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
            icon={Avatars || (Icon && <Icon />) || undefined}
            label={label}
            onClick={openDialog}
        />
    );
}
