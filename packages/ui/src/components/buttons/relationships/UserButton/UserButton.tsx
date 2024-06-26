import { exists, noop } from "@local/shared";
import { Avatar, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { FindObjectDialog } from "components/dialogs/FindObjectDialog/FindObjectDialog";
import { RelationshipItemUser } from "components/lists/types";
import { useField } from "formik";
import { AddIcon, BotIcon, UserIcon } from "icons";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { highlightStyle } from "styles";
import { extractImageUrl } from "utils/display/imageTools";
import { getDisplay, placeholderColor } from "utils/display/listTools";
import { openObject } from "utils/navigation/openObject";
import { largeButtonProps } from "../styles";
import { UserButtonProps } from "../types";

export const UserButton = ({
    isEditing,
    objectType,
}: UserButtonProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const { t } = useTranslation();

    const [field, , helpers] = useField("user");

    const isAvailable = useMemo(() => ["ChatInvite", "MemberInvite"].includes(objectType) && ["boolean", "object"].includes(typeof field.value), [objectType, field.value]);

    // Select object dialog
    const [isDialogOpen, setDialogOpen] = useState<boolean>(false); const handleClick = useCallback((ev: React.MouseEvent<Element>) => {
        if (!isAvailable) return;
        ev.stopPropagation();
        const relation = field?.value;
        // If not editing, navigate to object's page
        if (!isEditing) {
            if (relation) openObject(relation, setLocation);
        }
        else {
            // If relation was set, remove
            if (relation) {
                exists(helpers) && helpers.setValue(null);
            }
            // Otherwise, open select dialog
            else setDialogOpen(true);
        }
    }, [isAvailable, field?.value, isEditing, setLocation, helpers]);
    const closeDialog = useCallback(() => { setDialogOpen(false); }, [setDialogOpen]);
    const handleSelect = useCallback((relation: RelationshipItemUser) => {
        const relationId = field?.value?.id;
        if (relation?.id === relationId) return;
        exists(helpers) && helpers.setValue(relation);
        closeDialog();
    }, [field?.value?.id, helpers, closeDialog]);

    // FindObjectDialog
    const [findHandleAdd, findHandleClose] = useMemo<[(item: any) => unknown, () => unknown]>(() => {
        if (isDialogOpen) return [handleSelect, closeDialog];
        return [noop, noop];
    }, [isDialogOpen, handleSelect, closeDialog]);

    const tooltip = useMemo(() => {
        const relation = field?.value;
        // If no data, marked as unset
        if (!relation) return t(`UserNoneTogglePress${isEditing ? "Editable" : ""}`);
        return t(`UserTogglePress${isEditing ? "Editable" : ""}`, { user: getDisplay(relation).title ?? "" });
    }, [isEditing, field?.value, t]);

    // If not available, return null
    if (!isAvailable || (!isEditing && !field?.value)) return null;
    return (
        <>
            {/* Popup for selecting relation */}
            <FindObjectDialog
                find="List"
                isOpen={isDialogOpen}
                handleCancel={findHandleClose}
                handleComplete={findHandleAdd}
                limitTo={["User"]}
            />
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
                        {field?.value ? (
                            <Avatar
                                src={extractImageUrl(field.value.profileImage, field.value.updated_at, 50)}
                                alt={`${getDisplay(field.value).title}'s profile picture`}
                                sx={{
                                    backgroundColor: placeholderColor()[0],
                                    width: "48px",
                                    height: "48px",
                                    pointerEvents: "none",
                                    marginLeft: field.value.isBot ? 1.5 : 1,
                                    marginRight: 1,
                                    ...(field.value.isBot ? { borderRadius: "8px" } : {}),
                                }}
                            >
                                {field.value.isBot ? <BotIcon width="75%" height="75%" fill="white" /> : <UserIcon width="75%" height="75%" fill="white" />}
                            </Avatar>
                        ) : <AddIcon width={"48px"} height={"48px"} fill="white" />}
                        <Typography variant="body1" sx={{ color: "white" }}>
                            {field?.value ? getDisplay(field?.value).title : t("User")}
                        </Typography>
                    </Stack>
                </Tooltip>
            </Stack>
        </>
    );
};
