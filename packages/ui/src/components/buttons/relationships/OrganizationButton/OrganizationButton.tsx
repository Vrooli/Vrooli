import { exists } from "@local/shared";
import { IconButton, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { FindObjectDialog } from "components/dialogs/FindObjectDialog/FindObjectDialog";
import { RelationshipItemOrganization } from "components/lists/types";
import { useField } from "formik";
import { AddIcon, OrganizationIcon } from "icons";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { highlightStyle } from "styles";
import { getDisplay } from "utils/display/listTools";
import { openObject } from "utils/navigation/openObject";
import { largeButtonProps } from "../styles";
import { OrganizationButtonProps } from "../types";

export function OrganizationButton({
    isEditing,
    objectType,
}: OrganizationButtonProps) {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const { t } = useTranslation();

    const [field, , helpers] = useField("organization");

    const isAvailable = useMemo(() => ["MemberInvite"].includes(objectType) && ["boolean", "object"].includes(typeof field.value), [objectType, field.value]);

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
    const handleSelect = useCallback((relation: RelationshipItemOrganization) => {
        const relationId = field?.value?.id;
        if (relation?.id === relationId) return;
        exists(helpers) && helpers.setValue(relation);
        closeDialog();
    }, [field?.value?.id, helpers, closeDialog]);

    // FindObjectDialog
    const [findHandleAdd, findHandleClose] = useMemo<[(item: any) => unknown, () => unknown]>(() => {
        if (isDialogOpen) return [handleSelect, closeDialog];
        // eslint-disable-next-line @typescript-eslint/no-empty-function
        return [() => { }, () => { }];
    }, [isDialogOpen, handleSelect, closeDialog]);

    const { Icon, tooltip } = useMemo(() => {
        const relation = field?.value;
        // If no data, marked as unset
        if (!relation) return {
            Icon: AddIcon,
            tooltip: t(`OrganizationNoneTogglePress${isEditing ? "Editable" : ""}`),
        };
        return {
            Icon: OrganizationIcon,
            tooltip: t(`OrganizationTogglePress${isEditing ? "Editable" : ""}`, { organization: getDisplay(relation).title ?? "" }),
        };
    }, [isEditing, field?.value, t]);

    // If not available, return null
    if (!isAvailable || (!isEditing && !Icon)) return null;
    return (
        <>
            {/* Popup for selecting relation */}
            <FindObjectDialog
                find="List"
                isOpen={isDialogOpen}
                handleCancel={findHandleClose}
                handleComplete={findHandleAdd}
                limitTo={["Organization"]}
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
                        <IconButton>
                            <Icon width={"48px"} height={"48px"} fill="white" />
                        </IconButton>
                        <Typography variant="body1" sx={{ color: "white" }}>
                            {field?.value ? getDisplay(field?.value).title : t("Organization")}
                        </Typography>
                    </Stack>
                </Tooltip>
            </Stack>
        </>
    );
}
