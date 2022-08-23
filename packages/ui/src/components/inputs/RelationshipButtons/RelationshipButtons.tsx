/**
 * Horizontal button list for assigning owner, project, and parent 
 * to objects
 */
import { Box, Button, IconButton, Stack, Tooltip, Typography, useTheme } from '@mui/material';
import { useCallback, useMemo, useState } from 'react';
import { RelationshipButtonsProps, RelationshipItemOrganization, RelationshipItemProject, RelationshipItemRoutine, RelationshipItemUser, RelationshipOwner } from '../types';
import { noSelect } from 'styles';
import { getTranslation, getUserLanguages, ObjectType } from 'utils';
import { OrganizationSelectOrCreateDialog } from 'components/dialogs';
import {
    Apartment as OrganizationIcon,
    DeviceHub as RoutineIcon,
    Person as SelfIcon,
    ViewQuilt as ProjIcon,
} from '@mui/icons-material';
import { Session } from 'types';

/**
 * Converts session to user object
 * @param session Current user session
 * @returns User object
 */
export const userFromSession = (session: Session): RelationshipOwner => ({
    __typename: 'User',
    id: session.id,
    handle: null,
    name: 'Self',
})

export function RelationshipButtons({
    disabled = false,
    objectType,
    onOwnerChange,
    onProjectChange,
    onParentChange,
    owner,
    parent,
    project,
    session,
    zIndex,
}: RelationshipButtonsProps) {
    const { palette } = useTheme();
    const languages = useMemo(() => getUserLanguages(session), [session])

    // Owner dialog (select self or organization)
    const [isOwnerDialogOpen, setOwnerDialogOpen] = useState<boolean>(false);
    const openOwnerDialog = useCallback(() => { setOwnerDialogOpen(true) }, [setOwnerDialogOpen]);
    const closeOwnerDialog = useCallback(() => { setOwnerDialogOpen(false); }, [setOwnerDialogOpen]);

    // Organization owner dialog
    const [isOrganizationDialogOpen, setOrganizationDialogOpen] = useState<boolean>(false);
    const openOrganizationDialog = useCallback(() => { setOrganizationDialogOpen(true) }, [setOrganizationDialogOpen]);
    const closeOrganizationDialog = useCallback(() => { setOrganizationDialogOpen(false); }, [setOrganizationDialogOpen]);

    // Project dialog
    const [isProjectDialogOpen, setProjectDialogOpen] = useState<boolean>(false);
    const openProjectDialog = useCallback(() => { setProjectDialogOpen(true) }, [setProjectDialogOpen]);
    const closeProjectDialog = useCallback(() => { setProjectDialogOpen(false); }, [setProjectDialogOpen]);

    // Parent dialog
    const [isParentDialogOpen, setParentDialogOpen] = useState<boolean>(false);
    const openParentDialog = useCallback(() => { setParentDialogOpen(true) }, [setParentDialogOpen]);
    const closeParentDialog = useCallback(() => { setParentDialogOpen(false); }, [setParentDialogOpen]);

    // Determine which relationship buttons are available
    const { isOwnerAvailable, isProjectAvailable, isParentAvailable } = useMemo(() => {
        // Owner available for organizations, projects, routines, and standards
        const isOwnerAvailable = [ObjectType.Organization, ObjectType.Project, ObjectType.Routine, ObjectType.Standard].includes(objectType);
        // Project available for organizations, routines, and standards
        const isProjectAvailable = [ObjectType.Organization, ObjectType.Project, ObjectType.Standard].includes(objectType);
        // Parent available for projects and routines
        const isParentAvailable = [ObjectType.Project, ObjectType.Routine].includes(objectType);
        return { isOwnerAvailable, isProjectAvailable, isParentAvailable }
    }, [objectType])

    // Handle owner click
    const handleOwnerClick = useCallback((ev: React.MouseEvent<any>) => {
        if (disabled || !isOwnerAvailable) return;
        // No matter what the owner value was, open the owner select dialog
        openOwnerDialog();
    }, [disabled, isOwnerAvailable, openOwnerDialog]);

    // Handle project click
    const handleProjectClick = useCallback((ev: React.MouseEvent<any>) => {
        if (disabled || !isProjectAvailable) return;
        // If project was set, remove
        if (project) onProjectChange(null);
        // Otherwise, open project select dialog
        else openProjectDialog();
    }, [disabled, isProjectAvailable, onProjectChange, openProjectDialog, project]);

    // Handle parent click
    const handleParentClick = useCallback((ev: React.MouseEvent<any>) => {
        if (disabled || !isParentAvailable) return;
        // If parent was set, remove
        if (parent) onParentChange(null);
        // Otherwise, open parent select dialog
        else openParentDialog();
    }, [disabled, isParentAvailable, onParentChange, openParentDialog, parent]);

    // Current owner icon
    const { OwnerIcon, ownerTooltip } = useMemo(() => {
        // If no owner data, marked as anonymous
        if (!owner) return {
            OwnerIcon: null,
            ownerTooltip: 'Marked as anonymous. Press to set owner'
        };
        // If owner is organization, use organization icon
        if (owner.__typename === 'Organization') {
            const OwnerIcon = OrganizationIcon;
            // Button color indicates if you can modify the organization, or if someone will have to approve it
            const canEdit = (owner as RelationshipItemOrganization).permissionsOrganization?.canEdit === true;
            const ownerName = getTranslation(owner as RelationshipItemOrganization, 'name', languages, true) ?? 'organization';
            return {
                OwnerIcon,
                ownerTooltip: `Owner: ${ownerName}${!canEdit ? ' (requires approval)' : ''}`
            };
        }
        // If owner is user, use self icon
        const OwnerIcon = SelfIcon;
        // Button color indicates if it's your own account, or if someone will have to approve it
        const canEdit = owner.id === session.id;
        const ownerName = (owner as RelationshipItemUser).name;
        return {
            OwnerIcon,
            ownerTooltip: `Owner: ${canEdit ? 'Self' : (ownerName + ' (requires approval)')}`
        };
    }, [languages, owner, session.id]);

    // Current project icon
    const { ProjectIcon, projectTooltip } = useMemo(() => {
        // If no project data, marked as unset
        if (!project) return {
            ProjectIcon: null,
            projectTooltip: 'Press to assign to a project'
        };
        const canEdit = project.permissionsProject?.canEdit === true;
        const projectName = getTranslation(project as RelationshipItemProject, 'name', languages, true) ?? 'project';
        return {
            ProjectIcon: ProjIcon,
            projectTooltip: `Project: ${projectName}${!canEdit ? ' (requires approval)' : ''}`
        };
    }, [languages, project]);

    // Current parent icon
    const { ParentIcon, parentTooltip } = useMemo(() => {
        // If no parent data, marked as unset
        if (!parent) return {
            ParentIcon: null,
            parentTooltip: 'Press to copy from a parent (will override entered data)'
        };
        // If parent is project, use project icon
        if (parent.__typename === 'Project') {
            const ParentIcon = ProjIcon;
            // Button color indicates if you can modify the project, or if someone will have to approve it
            const canEdit = (parent as RelationshipItemProject).permissionsProject?.canEdit === true;
            const parentName = getTranslation(parent as RelationshipItemProject, 'name', languages, true) ?? 'project';
            return {
                ParentIcon,
                parentTooltip: `Parent: ${parentName}${!canEdit ? ' (requires approval)' : ''}`
            };
        }
        // If parent is routine, use routine icon
        const ParentIcon = RoutineIcon;
        // Button color indicates if you can modify the routine, or if someone will have to approve it
        const canEdit = (parent as RelationshipItemRoutine).permissionsRoutine?.canEdit === true;
        const parentName = getTranslation(parent as RelationshipItemRoutine, 'title', languages, true) ?? 'routine';
        return {
            ParentIcon,
            parentTooltip: `Parent: ${parentName}${!canEdit ? ' (requires approval)' : ''}`
        };
    }, [languages, parent]);

    const commonButtonProps = {
        width: '60px',
        height: '60px',
        background: palette.primary.light,
    }

    const commonIconProps = {
        width: 'unset',
        height: 'unset',
        '&:hover': {
            filter: `brightness(120%)`,
            transition: 'filter 0.2s',
        },
    }

    return (
        <Stack
            spacing={2}
            padding={1}
            direction="row"
            alignItems="center"
            justifyContent="center"
            zIndex={zIndex}
            sx={{
                borderRadius: '12px',
                background: palette.background.paper,
            }}
        >
            {/* Popup for adding/connecting a new organization */}
            {/* <OrganizationSelectOrCreateDialog
                isOpen={isOrganizationSelectDialogOpen}
                handleAdd={onOwnerOrganizationAdd}
                handleClose={closeOrganizationSelectDialog}
                session={session}
                zIndex={zIndex + 1}
            /> */}
            {/* Owner button */}
            <Tooltip title={ownerTooltip}>
                <IconButton sx={{ ...commonButtonProps }} onClick={handleOwnerClick}>
                    {OwnerIcon && <OwnerIcon sx={{ ...commonIconProps }} />}
                </IconButton>
            </Tooltip>
            {/* Project button */}
            <Tooltip title={projectTooltip}>
                <IconButton sx={{ ...commonButtonProps }}>
                    {ProjectIcon && <ProjectIcon sx={{ ...commonIconProps }} />}
                </IconButton>
            </Tooltip>
            {/* Parent button */}
            <Tooltip title={parentTooltip}>
                <IconButton sx={{ ...commonButtonProps }}>
                    {ParentIcon && <ParentIcon sx={{ ...commonIconProps }} />}
                </IconButton>
            </Tooltip>
        </Stack>
    )
}
