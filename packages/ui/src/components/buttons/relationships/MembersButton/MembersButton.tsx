import { User } from "@local/shared";
import { Box, IconButton, Stack, Typography, useTheme } from "@mui/material";
import { RelationshipItemUser } from "components/lists/types";
import { TextShrink } from "components/text/TextShrink/TextShrink";
import { useField, useFormikContext } from "formik";
import { AddIcon, BotIcon, LockIcon, UserIcon } from "icons";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { SvgComponent } from "types";
import { MemberManageView } from "views/MemberManageView/MemberManageView";
import { MemberManageViewProps } from "views/types";
import { commonIconProps, commonLabelProps, smallButtonProps } from "../styles";
import { MembersButtonProps } from "../types";

const maxIconsDisplayed = 4;

type IconType = SvgComponent | string | null;

const renderIcon = (Icon: IconType, key: number) => {
    if (Icon === null) return <Box sx={{ width: 24, height: 24 }} key={key} />;
    if (typeof Icon === "string") return <Typography variant="body2" key={key} sx={{ width: 24, height: 24 }}>{Icon}</Typography>;
    return <Icon key={key} {...commonIconProps()} width={24} height={24} />;
};

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

    const icons = useMemo<[IconType, IconType, IconType, IconType]>(() => {
        let newIcons: IconType[] = [];
        let maxUserIcons = isEditing ? maxIconsDisplayed - 1 : maxIconsDisplayed;
        // If there are more members than allowed, add a "+X" icon
        const hasMoreMembers = (membersField.value ?? []).length > maxUserIcons;
        if (hasMoreMembers) maxUserIcons--;
        // Add the first X members
        newIcons = (membersField.value ?? []).slice(0, maxUserIcons).map((user: User) => user.isBot ? BotIcon : UserIcon); // TODO Replace with User's profile pic in the future
        // Add the "+X" icon if there are more members than allowed
        if (hasMoreMembers) newIcons.push(`+${membersField.value.length - maxUserIcons}`);
        // Add the "Add" or "Lock" icon if editing
        if (isEditing) {
            if (isOpenToNewMembersField.value) newIcons.push(AddIcon);
            else newIcons.push(LockIcon);
        }
        // Add null icons to fill the remaining space up to the max
        while (newIcons.length < maxIconsDisplayed) newIcons.push(null);
        return newIcons as [IconType, IconType, IconType, IconType];
    }, [membersField.value, isEditing, isOpenToNewMembersField.value]);

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
