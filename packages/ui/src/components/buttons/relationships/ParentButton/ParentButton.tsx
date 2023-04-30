import { exists, ProjectIcon, RoutineIcon, useLocation } from "@local/shared";
import { Stack, Tooltip, useTheme } from "@mui/material";
import { ColorIconButton } from "components/buttons/ColorIconButton/ColorIconButton";
import { FindObjectDialog } from "components/dialogs/FindObjectDialog/FindObjectDialog";
import { SelectOrCreateObjectType } from "components/dialogs/types";
import { RelationshipItemProjectVersion, RelationshipItemRoutineVersion } from "components/lists/types";
import { TextShrink } from "components/text/TextShrink/TextShrink";
import { useField } from "formik";
import { useCallback, useContext, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { firstString } from "utils/display/stringTools";
import { getTranslation, getUserLanguages } from "utils/display/translationTools";
import { openObject } from "utils/navigation/openObject";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { commonIconProps, commonLabelProps, smallButtonProps } from "../styles";
import { ParentButtonProps } from "../types";

export function ParentButton({
    isEditing,
    isFormDirty,
    objectType,
    zIndex,
}: ParentButtonProps) {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const languages = useMemo(() => getUserLanguages(session), [session]);

    const [versionField, , versionHelpers] = useField("parent");
    const [rootField, , rootHelpers] = useField("root.parent");

    const isAvailable = false; // TODO add when copying/pull requests are implemented

    // Parent dialog
    const [isParentDialogOpen, setParentDialogOpen] = useState<boolean>(false);
    const handleParentClick = useCallback((ev: React.MouseEvent<any>) => {
        if (!isAvailable) return;
        ev.stopPropagation();
        const parent = versionField?.value ?? rootField?.value;
        // If not editing, navigate to parent
        if (!isEditing) {
            if (parent) openObject(parent, setLocation);
        }
        else {
            // If parent was set, remove
            if (parent) {
                exists(versionHelpers) && versionHelpers.setValue(null);
                exists(rootHelpers) && rootHelpers.setValue(null);
            }
            else {
                // If form is dirty, prompt to confirm (since data will be lost)
                if (isFormDirty) {
                    PubSub.get().publishAlertDialog({
                        messageKey: "ParentOverrideConfirm",
                        buttons: [
                            { labelKey: "Yes", onClick: () => { setParentDialogOpen(true); } },
                            { labelKey: "No", onClick: () => { } },
                        ],
                    });
                }
                // Otherwise, open parent select dialog
                else setParentDialogOpen(true);
            }
        }
    }, [isAvailable, versionField?.value, rootField?.value, isEditing, setLocation, versionHelpers, rootHelpers, isFormDirty]);
    const closeParentDialog = useCallback(() => { setParentDialogOpen(false); }, [setParentDialogOpen]);
    const handleParentSelect = useCallback((parent: any) => {
        const parentId = versionField?.value?.id ?? rootField?.value?.id;
        if (parent?.id === parentId) return;
        exists(versionHelpers) && versionHelpers.setValue(parent);
        exists(rootHelpers) && rootHelpers.setValue(parent);
        closeParentDialog();
    }, [closeParentDialog, rootField?.value?.id, rootHelpers, versionField?.value?.id, versionHelpers]);

    // FindObjectDialog
    const [findType, findHandleAdd, findHandleClose] = useMemo<[SelectOrCreateObjectType | null, (item: any) => any, () => void]>(() => {
        if (isParentDialogOpen) return [objectType as SelectOrCreateObjectType, handleParentSelect, closeParentDialog];
        return [null, () => { }, () => { }];
    }, [isParentDialogOpen, objectType, handleParentSelect, closeParentDialog]);

    const { Icon, tooltip } = useMemo(() => {
        const parent = versionField?.value ?? rootField?.value;
        if (!parent) return {
            Icon: null,
            tooltip: isEditing ? "" : "Press to copy from a parent (will override entered data)",
        };
        // If parent is project, use project icon
        if (parent.__typename === "ProjectVersion") {
            const Icon = ProjectIcon;
            const parentName = firstString(getTranslation(parent as RelationshipItemProjectVersion, languages, true).name, "project");
            return {
                Icon,
                tooltip: `${t("Parent")}: ${parentName}`,
            };
        }
        // If parent is routine, use routine icon
        const Icon = RoutineIcon;
        const parentName = firstString(getTranslation(parent as RelationshipItemRoutineVersion, languages, true).name, "routine");
        return {
            Icon,
            tooltip: `${t("Parent")}: ${parentName}`,
        };
    }, [isEditing, languages, rootField?.value, t, versionField?.value]);

    // If not available, return null
    if (!isAvailable || (!isEditing && !Icon)) return null;
    // Return button with label on top
    return (
        <>
            {/* Popup for selecting organization, user, etc. */}
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
                <TextShrink id="parent" sx={{ ...commonLabelProps() }}>{t("Parent")}</TextShrink>
                <Tooltip title={tooltip}>
                    <ColorIconButton
                        background={palette.primary.light}
                        sx={{ ...smallButtonProps(isEditing, true) }}
                        onClick={handleParentClick}
                    >
                        {Icon && <Icon {...commonIconProps()} />}
                    </ColorIconButton>
                </Tooltip>
            </Stack>
        </>
    );
}
