import { MemberShape } from "@local/shared";
import { AvatarGroup, Tooltip } from "@mui/material";
import { useField, useFormikContext } from "formik";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Icon, IconCommon } from "../../../icons/Icons.js";
import { extractImageUrl } from "../../../utils/display/imageTools.js";
import { MemberManageView } from "../../../views/MemberManageView/MemberManageView.js";
import { MemberManageViewProps } from "../../../views/types.js";
import { RelationshipAvatar, RelationshipButton, RelationshipChip } from "./styles.js";
import { MembersButtonProps } from "./types.js";

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

    const { avatars, iconInfo, membersCount, label, tooltip } = useMemo(() => {
        const members = (membersField.value || []) as MemberShape[];
        if (!Array.isArray(members) || members.some(member => typeof member !== "object")) {
            return {
                avatars: [],
                iconInfo: null,
                membersCount: 0,
                label: "",
                tooltip: "",
            };
        }
        const avatars = members.slice(0, MAX_AVATARS).map(member => {
            const imageUrl = extractImageUrl(member.user?.profileImage, member.user?.updatedAt, TARGET_IMAGE_SIZE);
            const isBot = member.user?.isBot ?? false;
            return (
                <RelationshipAvatar
                    key={member.id}
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

        const membersCount = members.length;
        const label = (membersCount > 0 || !isEditing) ? `${t("Member", { count: membersCount })}: ${membersCount}` : "Manage members";
        const truncatedLabel = label.length > MAX_LABEL_LENGTH ? `${label.slice(0, MAX_LABEL_LENGTH)}...` : label;
        const tooltip = isEditing ? "Manage members" : "View members";

        return {
            avatars,
            iconInfo: avatars.length > 0 ? undefined : { name: "Settings", type: "Common" } as const,
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
                    display="Dialog"
                    isEditing={isEditing}
                    isOpen={isDialogOpen}
                    onClose={closeDialog}
                    team={formikContext.values as MemberManageViewProps["team"]}
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
