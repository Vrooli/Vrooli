import { AddIcon, UserIcon } from "@local/shared";
import { Box, Stack, useTheme } from "@mui/material";
import { ColorIconButton } from "components/buttons/ColorIconButton/ColorIconButton";
import { FindObjectDialog } from "components/dialogs/FindObjectDialog/FindObjectDialog";
import { RelationshipItemUser } from "components/lists/types";
import { TextShrink } from "components/text/TextShrink/TextShrink";
import { useField } from "formik";
import { useCallback, useContext, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { SessionContext } from "utils/SessionContext";
import { commonIconProps, commonLabelProps, smallButtonProps } from "../styles";
import { MembersButtonProps } from "../types";

const maxIconsDisplayed = 4;

type IconType = typeof UserIcon | typeof AddIcon | string | null;

const renderIcon = (Icon: IconType, key: number) => {
    if (Icon === null) return <Box sx={{ width: 24, height: 24 }} key={key} />;
    if (typeof Icon === "string") return <TextShrink key={key} id="members-additional">{Icon}</TextShrink>;
    return <Icon key={key} {...commonIconProps()} width={24} height={24} />;
};

export const MembersButton = ({
    isEditing,
    objectType,
    zIndex,
}: MembersButtonProps) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { t } = useTranslation();

    const [field, , fieldHelpers] = useField("members");

    const isAvailable = useMemo(() => ["Organization"].includes(objectType), [objectType]);

    const [isDialogOpen, setDialogOpen] = useState<boolean>(false);
    const openDialog = useCallback(() => { setDialogOpen(true); }, []);
    const closeDialog = useCallback(() => { setDialogOpen(false); }, []);
    const handleMemberSelect = useCallback((member: RelationshipItemUser) => {
        fieldHelpers.setValue([...(field.value ?? []), member]);
        closeDialog();
    }, [fieldHelpers, closeDialog]);

    const searchData = useMemo(() => ({
        searchType: "User" as const,
        where: { memberInOrganizationId: field?.value?.id },
    }), [field?.value?.id]);

    const icons = useMemo<[IconType, IconType, IconType, IconType]>(() => {
        let newIcons: IconType[] = [];
        let maxUserIcons = isEditing ? maxIconsDisplayed - 1 : maxIconsDisplayed;
        // If there are more members than allowed, add a "+X" icon
        const hasMoreMembers = (field.value ?? []).length > maxUserIcons;
        if (hasMoreMembers) maxUserIcons--;
        // Add the first X members
        newIcons = (field.value ?? []).slice(0, maxUserIcons).map(user => UserIcon); // TODO Replace with User's profile pic in the future
        // Add the "+X" icon if there are more members than allowed
        if (hasMoreMembers) newIcons.push(`+${field.value.length - maxUserIcons}`);
        // Add the "Add" icon if editing
        if (isEditing) newIcons.push(AddIcon);
        // Add null icons to fill the remaining space up to the max
        while (newIcons.length < maxIconsDisplayed) newIcons.push(null);
        return newIcons as [IconType, IconType, IconType, IconType];
    }, [field.value, isEditing]);

    if (!isAvailable || (!isEditing && (!Array.isArray(field.value) || field.value.length == 0))) return null;

    return (
        <>
            {isDialogOpen && <FindObjectDialog
                find="List"
                isOpen={isDialogOpen}
                handleCancel={closeDialog}
                handleComplete={handleMemberSelect}
                searchData={searchData}
                limitTo={["User"]}
                zIndex={zIndex + 1}
            />}
            <Stack
                direction="column"
                alignItems="center"
                justifyContent="center"
            >
                <TextShrink id="members" sx={{ ...commonLabelProps() }}>{t("Member", { count: 2 })}</TextShrink>
                <ColorIconButton
                    background={palette.primary.light}
                    sx={{
                        ...smallButtonProps(isEditing, true),
                        borderRadius: "12px",
                        color: "white",
                    }}
                    onClick={openDialog}
                >
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
                </ColorIconButton>
            </Stack>
        </>
    );
};
