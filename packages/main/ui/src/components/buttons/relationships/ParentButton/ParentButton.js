import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { ProjectIcon, RoutineIcon } from "@local/icons";
import { exists } from "@local/utils";
import { Stack, Tooltip, useTheme } from "@mui/material";
import { useField } from "formik";
import { useCallback, useContext, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { firstString } from "../../../../utils/display/stringTools";
import { getTranslation, getUserLanguages } from "../../../../utils/display/translationTools";
import { openObject } from "../../../../utils/navigation/openObject";
import { PubSub } from "../../../../utils/pubsub";
import { useLocation } from "../../../../utils/route";
import { SessionContext } from "../../../../utils/SessionContext";
import { FindObjectDialog } from "../../../dialogs/FindObjectDialog/FindObjectDialog";
import { TextShrink } from "../../../text/TextShrink/TextShrink";
import { ColorIconButton } from "../../ColorIconButton/ColorIconButton";
import { commonButtonProps, commonIconProps, commonLabelProps } from "../styles";
export function ParentButton({ isEditing, isFormDirty, objectType, zIndex, }) {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const languages = useMemo(() => getUserLanguages(session), [session]);
    const [versionField, , versionHelpers] = useField("parent");
    const [rootField, , rootHelpers] = useField("root.parent");
    const isAvailable = false;
    const [isParentDialogOpen, setParentDialogOpen] = useState(false);
    const handleParentClick = useCallback((ev) => {
        if (!isAvailable)
            return;
        ev.stopPropagation();
        const parent = versionField?.value ?? rootField?.value;
        if (!isEditing) {
            if (parent)
                openObject(parent, setLocation);
        }
        else {
            if (parent) {
                exists(versionHelpers) && versionHelpers.setValue(null);
                exists(rootHelpers) && rootHelpers.setValue(null);
            }
            else {
                if (isFormDirty) {
                    PubSub.get().publishAlertDialog({
                        messageKey: "ParentOverrideConfirm",
                        buttons: [
                            { labelKey: "Yes", onClick: () => { setParentDialogOpen(true); } },
                            { labelKey: "No", onClick: () => { } },
                        ],
                    });
                }
                else
                    setParentDialogOpen(true);
            }
        }
    }, [isAvailable, versionField?.value, rootField?.value, isEditing, setLocation, versionHelpers, rootHelpers, isFormDirty]);
    const closeParentDialog = useCallback(() => { setParentDialogOpen(false); }, [setParentDialogOpen]);
    const handleParentSelect = useCallback((parent) => {
        const parentId = versionField?.value?.id ?? rootField?.value?.id;
        if (parent?.id === parentId)
            return;
        exists(versionHelpers) && versionHelpers.setValue(parent);
        exists(rootHelpers) && rootHelpers.setValue(parent);
        closeParentDialog();
    }, [closeParentDialog, rootField?.value?.id, rootHelpers, versionField?.value?.id, versionHelpers]);
    const [findType, findHandleAdd, findHandleClose] = useMemo(() => {
        if (isParentDialogOpen)
            return [objectType, handleParentSelect, closeParentDialog];
        return [null, () => { }, () => { }];
    }, [isParentDialogOpen, objectType, handleParentSelect, closeParentDialog]);
    const { Icon, tooltip } = useMemo(() => {
        const parent = versionField?.value ?? rootField?.value;
        if (!parent)
            return {
                Icon: null,
                tooltip: isEditing ? "" : "Press to copy from a parent (will override entered data)",
            };
        if (parent.__typename === "ProjectVersion") {
            const Icon = ProjectIcon;
            const parentName = firstString(getTranslation(parent, languages, true).name, "project");
            return {
                Icon,
                tooltip: `${t("Parent")}: ${parentName}`,
            };
        }
        const Icon = RoutineIcon;
        const parentName = firstString(getTranslation(parent, languages, true).name, "routine");
        return {
            Icon,
            tooltip: `${t("Parent")}: ${parentName}`,
        };
    }, [isEditing, languages, rootField?.value, t, versionField?.value]);
    if (!isAvailable || (!isEditing && !Icon))
        return null;
    return (_jsxs(_Fragment, { children: [findType && _jsx(FindObjectDialog, { find: "List", isOpen: Boolean(findType), handleCancel: findHandleClose, handleComplete: findHandleAdd, limitTo: [findType], zIndex: zIndex + 1 }), _jsxs(Stack, { direction: "column", alignItems: "center", justifyContent: "center", children: [_jsx(TextShrink, { id: "parent", sx: { ...commonLabelProps() }, children: t("Parent") }), _jsx(Tooltip, { title: tooltip, children: _jsx(ColorIconButton, { background: palette.primary.light, sx: { ...commonButtonProps(isEditing, true) }, onClick: handleParentClick, children: Icon && _jsx(Icon, { ...commonIconProps() }) }) })] })] }));
}
//# sourceMappingURL=ParentButton.js.map