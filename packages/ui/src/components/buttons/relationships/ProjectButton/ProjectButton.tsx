import { Stack, Tooltip, useTheme } from '@mui/material';
import { ProjectIcon } from '@shared/icons';
import { useLocation } from '@shared/route';
import { exists } from '@shared/utils';
import { ColorIconButton } from 'components/buttons/ColorIconButton/ColorIconButton';
import { SelectOrCreateDialog } from 'components/dialogs/selectOrCreates';
import { SelectOrCreateObjectType } from 'components/dialogs/selectOrCreates/types';
import { RelationshipItemProjectVersion } from 'components/inputs/types';
import { TextShrink } from 'components/text/TextShrink/TextShrink';
import { useField } from 'formik';
import { useCallback, useContext, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { firstString } from 'utils/display/stringTools';
import { getTranslation, getUserLanguages } from 'utils/display/translationTools';
import { openObject } from 'utils/navigation/openObject';
import { SessionContext } from 'utils/SessionContext';
import { commonButtonProps, commonIconProps, commonLabelProps } from '../styles';
import { ProjectButtonProps } from '../types';

export function ProjectButton({
    isEditing,
    objectType,
    zIndex,
}: ProjectButtonProps) {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const languages = useMemo(() => getUserLanguages(session), [session])

    const [versionField, , versionHelpers] = useField('project');
    const [rootField, , rootHelpers] = useField('root.project');

    const isAvailable = useMemo(() => ['Project', 'Routine', 'Standard'].includes(objectType), [objectType]);

    // Project dialog
    const [isProjectDialogOpen, setProjectDialogOpen] = useState<boolean>(false);
    const handleProjectClick = useCallback((ev: React.MouseEvent<any>) => {
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
    }, [versionField?.value?.id, rootField?.value?.id, versionHelpers, rootHelpers]);

    // SelectOrCreateDialog
    const [selectOrCreateType, selectOrCreateHandleAdd, selectOrCreateHandleClose] = useMemo<[SelectOrCreateObjectType | null, (item: any) => any, () => void]>(() => {
        if (isProjectDialogOpen) return ['ProjectVersion', handleProjectSelect, closeProjectDialog];
        return [null, () => { }, () => { }];
    }, [isProjectDialogOpen, handleProjectSelect, closeProjectDialog]);

    const { Icon, tooltip } = useMemo(() => {
        const project = versionField?.value ?? rootField?.value;
        // If no project data, marked as unset
        if (!project) return {
            Icon: null,
            tooltip: isEditing ? '' : 'Press to assign to a project'
        };
        const projectName = firstString(getTranslation(project as RelationshipItemProjectVersion, languages, true).name, 'project');
        return {
            Icon: ProjectIcon,
            tooltip: `Project: ${projectName}`
        };
    }, [isEditing, languages, rootField?.value, versionField?.value]);

    // If not available, return null
    if (!isAvailable || !isEditing) return null;
    // Return button with label on top
    return (
        <>
            {/* Popup for selecting project */}
            {selectOrCreateType && <SelectOrCreateDialog
                isOpen={Boolean(selectOrCreateType)}
                handleAdd={selectOrCreateHandleAdd}
                handleClose={selectOrCreateHandleClose}
                objectType={selectOrCreateType}
                zIndex={zIndex + 1}
            />}
            <Stack
                direction="column"
                alignItems="center"
                justifyContent="center"
            >
                <TextShrink id="project" sx={{ ...commonLabelProps() }}>Project</TextShrink>
                <Tooltip title={tooltip}>
                    <ColorIconButton background={palette.primary.light} sx={{ ...commonButtonProps(isEditing, true) }} onClick={handleProjectClick}>
                        {Icon && <Icon {...commonIconProps()} />}
                    </ColorIconButton>
                </Tooltip>
            </Stack>
        </>
    )
}