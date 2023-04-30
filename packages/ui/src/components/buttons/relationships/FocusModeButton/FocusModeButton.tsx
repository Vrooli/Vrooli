import { AddIcon, exists, FocusModeIcon, LINKS, useLocation } from "@local/shared";
import { IconButton, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { buttonSx } from "components/buttons/ColorIconButton/ColorIconButton";
import { FindObjectDialog } from "components/dialogs/FindObjectDialog/FindObjectDialog";
import { SelectOrCreateObjectType } from "components/dialogs/types";
import { RelationshipItemFocusMode } from "components/lists/types";
import { useField } from "formik";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { largeButtonProps } from "../styles";
import { FocusModeButtonProps } from "../types";

export function FocusModeButton({
    isEditing,
    objectType,
    zIndex,
}: FocusModeButtonProps) {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const { t } = useTranslation();

    const [field, , helpers] = useField("focusMode");

    const isAvailable = useMemo(() => ["Schedule"].includes(objectType) && ["boolean", "object"].includes(typeof field.value), [objectType, field.value]);

    // Focus mode dialog
    const [isDialogOpen, setDialogOpen] = useState<boolean>(false); const handleClick = useCallback((ev: React.MouseEvent<any>) => {
        if (!isAvailable) return;
        ev.stopPropagation();
        const focusMode = field?.value;
        // If not editing, navigate to display settings
        if (!isEditing) {
            if (focusMode) setLocation(LINKS.SettingsFocusModes);
        }
        else {
            // If focus mode was set, remove
            if (focusMode) {
                exists(helpers) && helpers.setValue(null);
            }
            // Otherwise, open select dialog
            else setDialogOpen(true);
        }
    }, [isAvailable, field?.value, isEditing, setLocation, helpers]);
    const closeDialog = useCallback(() => { setDialogOpen(false); }, [setDialogOpen]);
    const handleSelect = useCallback((focusMode: RelationshipItemFocusMode) => {
        const focusModeId = field?.value?.id;
        if (focusMode?.id === focusModeId) return;
        exists(helpers) && helpers.setValue(focusMode);
        closeDialog();
    }, [field?.value?.id, helpers, closeDialog]);

    // FindObjectDialog
    const [findType, findHandleAdd, findHandleClose] = useMemo<[SelectOrCreateObjectType | null, (item: any) => any, () => void]>(() => {
        if (isDialogOpen) return ["FocusMode", handleSelect, closeDialog];
        return [null, () => { }, () => { }];
    }, [isDialogOpen, handleSelect, closeDialog]);

    const { Icon, tooltip } = useMemo(() => {
        const focusMode = field?.value;
        // If no data, marked as unset
        if (!focusMode) return {
            Icon: AddIcon,
            tooltip: isEditing ? "" : "Press to assign to a focus mode",
        };
        const focusModeName = focusMode?.name ?? "";
        return {
            Icon: FocusModeIcon,
            tooltip: `Focus Mode: ${focusModeName}`,
        };
    }, [isEditing, field?.value]);

    // If not available, return null
    if (!isAvailable || (!isEditing && !Icon)) return null;
    return (
        <>
            {/* Popup for selecting focus mode */}
            {findType && <FindObjectDialog
                find="List"
                isOpen={Boolean(findType)}
                handleCancel={findHandleClose}
                handleComplete={findHandleAdd}
                limitTo={[findType]}
                zIndex={zIndex + 1}
            />}
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
                            ...buttonSx(palette.primary.light, !isEditing),
                        }}
                    >
                        {Icon && (
                            <IconButton>
                                <Icon width={"48px"} height={"48px"} fill="white" />
                            </IconButton>
                        )}
                        <Typography variant="body1" sx={{ color: "white" }}>
                            {field?.value?.name ?? t("FocusMode", { count: 1 })}
                        </Typography>
                    </Stack>
                </Tooltip>
            </Stack>
        </>
    );
}
