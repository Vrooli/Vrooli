import { exists, FocusModeIcon, LINKS, useLocation } from "@local/shared";
import { Stack, Tooltip, useTheme } from "@mui/material";
import { ColorIconButton } from "components/buttons/ColorIconButton/ColorIconButton";
import { FindObjectDialog } from "components/dialogs/FindObjectDialog/FindObjectDialog";
import { SelectOrCreateObjectType } from "components/dialogs/types";
import { RelationshipItemFocusMode } from "components/lists/types";
import { TextShrink } from "components/text/TextShrink/TextShrink";
import { useField } from "formik";
import { useCallback, useContext, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { getUserLanguages } from "utils/display/translationTools";
import { SessionContext } from "utils/SessionContext";
import { commonButtonProps, commonIconProps, commonLabelProps } from "../styles";
import { FocusModeButtonProps } from "../types";

export function FocusModeButton({
    isEditing,
    objectType,
    zIndex,
}: FocusModeButtonProps) {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const languages = useMemo(() => getUserLanguages(session), [session]);

    const [field, , helpers] = useField("focusMode");

    const isAvailable = useMemo(() => ["Schedule"].includes(objectType) && exists(field.value), [objectType, field.value]);

    // Focus mode dialog
    const [isDialogOpen, setDialogOpen] = useState<boolean>(false); const handleClick = useCallback((ev: React.MouseEvent<any>) => {
        if (!isAvailable) return;
        ev.stopPropagation();
        const focusMode = field?.value;
        // // If not editing, navigate to focus mode
        // If not editing, navigate to display settings
        if (!isEditing) {
            // if (focusMode) openObject(focusMode, setLocation);
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
            Icon: null,
            tooltip: isEditing ? "" : "Press to assign to a focus mode",
        };
        const focusModeName = focusMode?.name ?? "";
        return {
            Icon: FocusModeIcon,
            tooltip: `Focus Mode: ${focusModeName}`,
        };
    }, [isEditing, languages, field?.value]);

    // If not available, return null
    if (!isAvailable || (!isEditing && !Icon)) return null;
    // Return button with label on top
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
            >
                <TextShrink id="focusMode" sx={{ ...commonLabelProps() }}>
                    {t("FocusMode", { count: 1 })}
                </TextShrink>
                <Tooltip title={tooltip}>
                    <ColorIconButton
                        background={palette.primary.light}
                        sx={{ ...commonButtonProps(isEditing, true) }}
                        onClick={handleClick}
                    >
                        {Icon && <Icon {...commonIconProps()} />}
                    </ColorIconButton>
                </Tooltip>
            </Stack>
        </>
    );
}
