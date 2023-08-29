import { exists } from "@local/shared";
import { IconButton, Stack, Tooltip, useTheme } from "@mui/material";
import { FindObjectDialog } from "components/dialogs/FindObjectDialog/FindObjectDialog";
import { SelectOrCreateObjectType } from "components/dialogs/types";
import { RelationshipItemProjectVersion } from "components/lists/types";
import { TextShrink } from "components/text/TextShrink/TextShrink";
import { SessionContext } from "contexts/SessionContext";
import { useField } from "formik";
import { ProjectIcon } from "icons";
import { useCallback, useContext, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { firstString } from "utils/display/stringTools";
import { getTranslation, getUserLanguages } from "utils/display/translationTools";
import { openObject } from "utils/navigation/openObject";
import { commonIconProps, commonLabelProps, smallButtonProps } from "../styles";
import { ProjectButtonProps } from "../types";

export function ProjectButton({
    isEditing,
    objectType,
}: ProjectButtonProps) {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const languages = useMemo(() => getUserLanguages(session), [session]);

    const [versionField, , versionHelpers] = useField("project");
    const [rootField, , rootHelpers] = useField("root.project");

    const isAvailable = useMemo(() => ["Project", "Routine", "Standard"].includes(objectType), [objectType]);

    // Project dialog
    const [isProjectDialogOpen, setProjectDialogOpen] = useState<boolean>(false);
    const handleProjectClick = useCallback((ev: React.MouseEvent<Element>) => {
        if (!isAvailable) return;
        ev.stopPropagation();
        const project = versionField?.value ?? rootField?.value;
        // If not editing, navigate to project
        if (!isEditing) {
            if (project) openObject(project, setLocation);
        }
        else {
            // If project was set, remove
            if (project) {
                exists(versionHelpers) && versionHelpers.setValue(null);
                exists(rootHelpers) && rootHelpers.setValue(null);
            }
            // Otherwise, open project select dialog
            else setProjectDialogOpen(true);
        }
    }, [isAvailable, versionField?.value, rootField?.value, isEditing, setLocation, versionHelpers, rootHelpers]);
    const closeProjectDialog = useCallback(() => { setProjectDialogOpen(false); }, [setProjectDialogOpen]);
    const handleProjectSelect = useCallback((project: RelationshipItemProjectVersion) => {
        const projectId = versionField?.value?.id ?? rootField?.value?.id;
        if (project?.id === projectId) return;
        exists(versionHelpers) && versionHelpers.setValue(project);
        exists(rootHelpers) && rootHelpers.setValue(project);
        closeProjectDialog();
    }, [versionField?.value?.id, rootField?.value?.id, versionHelpers, rootHelpers, closeProjectDialog]);

    // FindObjectDialog
    const [findType, findHandleAdd, findHandleClose] = useMemo<[SelectOrCreateObjectType | null, (item: any) => unknown, () => unknown]>(() => {
        if (isProjectDialogOpen) return ["ProjectVersion", handleProjectSelect, closeProjectDialog];
        // eslint-disable-next-line @typescript-eslint/no-empty-function
        return [null, () => { }, () => { }];
    }, [isProjectDialogOpen, handleProjectSelect, closeProjectDialog]);

    const { Icon, tooltip } = useMemo(() => {
        const project = versionField?.value ?? rootField?.value;
        // If no project data, marked as unset
        if (!project) return {
            Icon: null,
            tooltip: t(`ProjectNoneTogglePress${isEditing ? "Editable" : ""}`),
        };
        const projectName = firstString(getTranslation(project as RelationshipItemProjectVersion, languages, true).name, "project");
        return {
            Icon: ProjectIcon,
            tooltip: t(`ProjectTogglePress${isEditing ? "Editable" : ""}`, { project: projectName }),
        };
    }, [isEditing, languages, rootField?.value, t, versionField?.value]);

    // If not available, return null
    if (!isAvailable || (!isEditing && !Icon)) return null;
    // Return button with label on top
    return (
        <>
            {/* Popup for selecting project */}
            {findType && <FindObjectDialog
                find="List"
                isOpen={Boolean(findType)}
                handleCancel={findHandleClose}
                handleComplete={findHandleAdd}
                limitTo={[findType]}
            />}
            <Stack
                direction="column"
                alignItems="center"
                justifyContent="center"
            >
                <TextShrink id="project" sx={{ ...commonLabelProps() }}>{t("Project", { count: 1 })}</TextShrink>
                <Tooltip title={tooltip}>
                    <IconButton
                        onClick={handleProjectClick}
                        sx={{ ...smallButtonProps(isEditing, true), background: palette.primary.light }}
                    >
                        {Icon && <Icon {...commonIconProps()} />}
                    </IconButton>
                </Tooltip>
            </Stack>
        </>
    );
}
