import { User } from "@local/shared";
import { Avatar, IconButton, Stack, Typography, useTheme } from "@mui/material";
import { RelationshipItemUser } from "components/lists/types";
import { TextShrink } from "components/text/TextShrink/TextShrink";
import { useField, useFormikContext } from "formik";
import { AddIcon, BotIcon, LockIcon, UserIcon } from "icons";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { extractImageUrl } from "utils/display/imageTools";
import { getDisplay, placeholderColor } from "utils/display/listTools";
import { MemberManageView } from "views/MemberManageView/MemberManageView";
import { MemberManageViewProps } from "views/types";
import { commonLabelProps, smallButtonProps } from "../styles";
import { MembersButtonProps } from "../types";

const maxIconsDisplayed = 4;

export const MembersButton = ({
    isEditing,
    objectType,
}: MembersButtonProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();

    const formikContext = useFormikContext();
    const [membersField, , membersFieldHelpers] = useField("members");
    const [isOpenToNewMembersField] = useField("isOpenToNewMembers");

    const isAvailable = useMemo(() => ["Organization"].includes(objectType), [objectType]);

    const [isDialogOpen, setDialogOpen] = useState<boolean>(false);
    const openDialog = useCallback(() => { setDialogOpen(true); }, []);
    const closeDialog = useCallback(() => { setDialogOpen(false); }, []);
    const handleMemberSelect = useCallback((member: RelationshipItemUser) => {
        membersFieldHelpers.setValue([...(membersField.value ?? []), member]);
        closeDialog();
    }, [membersFieldHelpers, membersField.value, closeDialog]);

    const searchData = useMemo(() => ({
        searchType: "User" as const,
        where: { memberInOrganizationId: membersField?.value?.id },
    }), [membersField?.value?.id]);

    const icons = useMemo<(JSX.Element | null)[]>(() => {
        let newIcons: (JSX.Element | null)[] = [];
        let maxUserIcons = isEditing ? maxIconsDisplayed - 1 : maxIconsDisplayed;
        // If there are more members than allowed, add a "+X" icon
        const hasMoreMembers = (membersField.value ?? []).length > maxUserIcons;
        if (hasMoreMembers) maxUserIcons--;
        // Add the first X members
        newIcons = (membersField.value ?? []).slice(0, maxUserIcons).map((user: User) => (
            <Avatar
                src={extractImageUrl(user.profileImage, user.updated_at, 50)}
                alt={`${getDisplay(user).title}'s profile picture`}
                sx={{
                    backgroundColor: placeholderColor()[0],
                    width: "24px",
                    height: "24px",
                    pointerEvents: "none",
                    ...(user.isBot ? { borderRadius: "4px" } : {}),
                }}
            >
                {user.isBot ? <BotIcon width="75%" height="75%" fill={palette.background.textPrimary} /> : <UserIcon width="75%" height="75%" fill={palette.background.textPrimary} />}
            </Avatar>
        ));
        // Add the "+X" icon if there are more members than allowed
        if (hasMoreMembers) {
            newIcons.push(
                <Typography variant="body2" key="more" sx={{ width: 24, height: 24 }}>
                    {`+${membersField.value.length - maxUserIcons}`}
                </Typography>,
            );
        }
        // Add the "Add" or "Lock" icon if editing
        if (isEditing) {
            if (isOpenToNewMembersField.value) newIcons.push(<AddIcon key="add" width={24} height={24} fill={palette.background.textPrimary} />);
            else newIcons.push(<LockIcon key="lock" width={24} height={24} fill={palette.background.textPrimary} />);
        }
        // Add null icons to fill the remaining space up to the max
        while (newIcons.length < maxIconsDisplayed) newIcons.push(null);
        return newIcons;
    }, [isEditing, isOpenToNewMembersField.value, membersField.value, palette.background.textPrimary]);

    if (!isAvailable || (!isEditing && (!Array.isArray(membersField.value) || membersField.value.length === 0))) return null;

    return (
        <>
            {/* Dialog for managing members */}
            <MemberManageView
                isOpen={isDialogOpen}
                onClose={closeDialog}
                organization={formikContext.values as MemberManageViewProps["organization"]}
            />
            <Stack
                direction="column"
                alignItems="center"
                justifyContent="center"
            >
                <TextShrink id="members" sx={{ ...commonLabelProps() }}>{t("Member", { count: 2 })}</TextShrink>
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
                    {/* Members & add members icons */}
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
