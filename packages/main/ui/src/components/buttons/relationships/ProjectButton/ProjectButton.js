import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { ProjectIcon } from "@local/icons";
import { exists } from "@local/utils";
import { Stack, Tooltip, useTheme } from "@mui/material";
import { useField } from "formik";
import { useCallback, useContext, useMemo, useState } from "react";
import { firstString } from "../../../../utils/display/stringTools";
import { getTranslation, getUserLanguages } from "../../../../utils/display/translationTools";
import { openObject } from "../../../../utils/navigation/openObject";
import { useLocation } from "../../../../utils/route";
import { SessionContext } from "../../../../utils/SessionContext";
import { FindObjectDialog } from "../../../dialogs/FindObjectDialog/FindObjectDialog";
import { TextShrink } from "../../../text/TextShrink/TextShrink";
import { ColorIconButton } from "../../ColorIconButton/ColorIconButton";
import { commonButtonProps, commonIconProps, commonLabelProps } from "../styles";
export function ProjectButton({ isEditing, objectType, zIndex, }) {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const languages = useMemo(() => getUserLanguages(session), [session]);
    const [versionField, , versionHelpers] = useField("project");
    const [rootField, , rootHelpers] = useField("root.project");
    const isAvailable = useMemo(() => ["Project", "Routine", "Standard"].includes(objectType), [objectType]);
    const [isProjectDialogOpen, setProjectDialogOpen] = useState(false);
    const handleProjectClick = useCallback((ev) => {
        if (!isAvailable)
            return;
        ev.stopPropagation();
        const project = versionField?.value ?? rootField?.value;
        if (!isEditing) {
            if (project)
                openObject(project, setLocation);
        }
        else {
            if (project) {
                exists(versionHelpers) && versionHelpers.setValue(null);
                exists(rootHelpers) && rootHelpers.setValue(null);
            }
            else
                setProjectDialogOpen(true);
        }
    }, [isAvailable, versionField?.value, rootField?.value, isEditing, setLocation, versionHelpers, rootHelpers]);
    const closeProjectDialog = useCallback(() => { setProjectDialogOpen(false); }, [setProjectDialogOpen]);
    const handleProjectSelect = useCallback((project) => {
        const projectId = versionField?.value?.id ?? rootField?.value?.id;
        if (project?.id === projectId)
            return;
        exists(versionHelpers) && versionHelpers.setValue(project);
        exists(rootHelpers) && rootHelpers.setValue(project);
        closeProjectDialog();
    }, [versionField?.value?.id, rootField?.value?.id, versionHelpers, rootHelpers, closeProjectDialog]);
    const [findType, findHandleAdd, findHandleClose] = useMemo(() => {
        if (isProjectDialogOpen)
            return ["ProjectVersion", handleProjectSelect, closeProjectDialog];
        return [null, () => { }, () => { }];
    }, [isProjectDialogOpen, handleProjectSelect, closeProjectDialog]);
    const { Icon, tooltip } = useMemo(() => {
        const project = versionField?.value ?? rootField?.value;
        if (!project)
            return {
                Icon: null,
                tooltip: isEditing ? "" : "Press to assign to a project",
            };
        const projectName = firstString(getTranslation(project, languages, true).name, "project");
        return {
            Icon: ProjectIcon,
            tooltip: `Project: ${projectName}`,
        };
    }, [isEditing, languages, rootField?.value, versionField?.value]);
    if (!isAvailable || (!isEditing && !Icon))
        return null;
    return (_jsxs(_Fragment, { children: [findType && _jsx(FindObjectDialog, { find: "List", isOpen: Boolean(findType), handleCancel: findHandleClose, handleComplete: findHandleAdd, limitTo: [findType], zIndex: zIndex + 1 }), _jsxs(Stack, { direction: "column", alignItems: "center", justifyContent: "center", children: [_jsx(TextShrink, { id: "project", sx: { ...commonLabelProps() }, children: "Project" }), _jsx(Tooltip, { title: tooltip, children: _jsx(ColorIconButton, { background: palette.primary.light, sx: { ...commonButtonProps(isEditing, true) }, onClick: handleProjectClick, children: Icon && _jsx(Icon, { ...commonIconProps() }) }) })] })] }));
}
//# sourceMappingURL=ProjectButton.js.map