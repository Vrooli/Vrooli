/**
 * Horizontal button list for assigning owner, project, and parent 
 * to objects
 */
import { Box, IconButton, Palette, Stack, Tooltip, useTheme } from '@mui/material';
import { useCallback, useMemo, useState } from 'react';
import { RelationshipButtonsProps, RelationshipItemOrganization, RelationshipItemProject, RelationshipItemRoutine, RelationshipItemUser, RelationshipOwner } from '../types';
import { getTranslation, getUserLanguages, ObjectType } from 'utils';
import { ListMenu, OrganizationSelectOrCreateDialog, ProjectSelectOrCreateDialog, RoutineSelectOrCreateDialog, UserSelectDialog } from 'components/dialogs';
import {
    Apartment as OrganizationIcon,
    DeviceHub as RoutineIcon,
    DoneAll as CompletedIcon,
    Person as SelfIcon,
    RemoveDone as IncompleteIcon,
    ViewQuilt as ProjIcon,
    Visibility as PublicIcon,
    VisibilityOff as PrivateIcon,
} from '@mui/icons-material';
import { Session } from 'types';
import { ListMenuItemData } from 'components/dialogs/types';
import { TextShrink } from 'components/text/TextShrink/TextShrink';

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

const commonButtonProps = (palette: Palette) => ({
    width: '69px',
    height: '69px',
    background: palette.primary.light,
})

const commonIconProps = {
    width: 'unset',
    height: 'unset',
    '&:hover': {
        filter: `brightness(120%)`,
        transition: 'filter 0.2s',
    },
}

const commonLabelProps = {
    width: '69px',
    textAlign: 'center',
}

enum OwnerTypesEnum {
    Self = 'Self',
    Organization = 'Organization',
    AnotherUser = 'AnotherUser',
}

const ownerTypes: ListMenuItemData<OwnerTypesEnum>[] = [
    { label: 'Self', value: OwnerTypesEnum.Self },
    { label: 'Organization', value: OwnerTypesEnum.Organization },
    { label: 'Another User (Requires Permission)', value: OwnerTypesEnum.AnotherUser },
]

export function RelationshipButtons({
    disabled = false,
    isComplete,
    isPrivate,
    objectType,
    onIsCompleteChange,
    onIsPrivateChange,
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

    // Determine which relationship buttons are available
    const { isCompleteAvailable, isOwnerAvailable, isPrivateAvailable, isProjectAvailable, isParentAvailable } = useMemo(() => {
        // isComplete available for projects and routines
        const isCompleteAvailable = [ObjectType.Project, ObjectType.Routine].includes(objectType);
        // Owner available for organizations, projects, routines, and standards
        const isOwnerAvailable = [ObjectType.Organization, ObjectType.Project, ObjectType.Routine, ObjectType.Standard].includes(objectType);
        // isPrivate always available for now
        const isPrivateAvailable = true;
        // Project available for organizations, routines, and standards
        const isProjectAvailable = [ObjectType.Organization, ObjectType.Project, ObjectType.Standard].includes(objectType);
        // Parent available for projects and routines
        const isParentAvailable = [ObjectType.Project, ObjectType.Routine].includes(objectType);
        return { isCompleteAvailable, isOwnerAvailable, isPrivateAvailable, isProjectAvailable, isParentAvailable }
    }, [objectType])

    // Organization owner dialog (displayed after selecting owner dialog)
    const [isOrganizationDialogOpen, setOrganizationDialogOpen] = useState<boolean>(false);
    const openOrganizationDialog = useCallback(() => { setOrganizationDialogOpen(true) }, [setOrganizationDialogOpen]);
    const closeOrganizationDialog = useCallback(() => { setOrganizationDialogOpen(false); }, [setOrganizationDialogOpen]);

    // Another user owner dialog (displayed after selecting owner dialog)
    const [isAnotherUserDialogOpen, setAnotherUserDialogOpen] = useState<boolean>(false);
    const openAnotherUserDialog = useCallback(() => { setAnotherUserDialogOpen(true) }, [setAnotherUserDialogOpen]);
    const closeAnotherUserDialog = useCallback(() => { setAnotherUserDialogOpen(false); }, [setAnotherUserDialogOpen]);

    // Owner dialog (select self, organization, or another user)
    const [ownerDialogAnchor, setOwnerDialogAnchor] = useState<any>(null);
    const handleOwnerClick = useCallback((ev: React.MouseEvent<any>) => {
        if (disabled || !isOwnerAvailable) return;
        ev.stopPropagation();
        setOwnerDialogAnchor(ev.currentTarget)
    }, [disabled, isOwnerAvailable]);
    const closeOwnerDialog = useCallback(() => setOwnerDialogAnchor(null), []);
    const handleOwnerDialogSelect = useCallback((ownerType: OwnerTypesEnum) => {
        if (ownerType === OwnerTypesEnum.Organization) {
            openOrganizationDialog();
        } else if (ownerType === OwnerTypesEnum.AnotherUser) {
            openAnotherUserDialog();
        } else {
            onOwnerChange(userFromSession(session));
        }
        closeOwnerDialog();
    }, [closeOwnerDialog, onOwnerChange, openAnotherUserDialog, openOrganizationDialog, session]);
    const handleOwnerSelect = useCallback((owner: RelationshipOwner) => {
        onOwnerChange(owner);
    }, [onOwnerChange]);

    // Project dialog
    const [isProjectDialogOpen, setProjectDialogOpen] = useState<boolean>(false);
    const handleProjectClick = useCallback((ev: React.MouseEvent<any>) => {
        if (disabled || !isProjectAvailable) return;
        // If project was set, remove
        if (project) onProjectChange(null);
        // Otherwise, open project select dialog
        else setProjectDialogOpen(true);
    }, [disabled, isProjectAvailable, onProjectChange, project]);
    const closeProjectDialog = useCallback(() => { setProjectDialogOpen(false); }, [setProjectDialogOpen]);
    const handleProjectSelect = useCallback((project: RelationshipItemProject) => {
        onProjectChange(project);
    }, [onProjectChange]);

    // Parent dialog
    const [isParentDialogOpen, setParentDialogOpen] = useState<boolean>(false);
    const handleParentClick = useCallback((ev: React.MouseEvent<any>) => {
        if (disabled || !isParentAvailable) return;
        // If parent was set, remove
        if (parent) onParentChange(null);
        // Otherwise, open parent select dialog
        else setParentDialogOpen(true);
    }, [disabled, isParentAvailable, onParentChange, parent]);
    const closeParentDialog = useCallback(() => { setParentDialogOpen(false); }, [setParentDialogOpen]);
    const handleParentProjectSelect = useCallback((parent: RelationshipItemProject) => {
        onParentChange(parent);
    }, [onParentChange]);
    const handleParentRoutineSelect = useCallback((parent: RelationshipItemRoutine) => {
        onParentChange(parent);
    }, [onParentChange]);

    // Handle private click
    const handlePrivateClick = useCallback((ev: React.MouseEvent<any>) => {
        if (disabled || !isPrivateAvailable) return;
        onIsPrivateChange(!isPrivate);
    }, [disabled, isPrivate, isPrivateAvailable, onIsPrivateChange]);

    // Handle complete click
    const handleCompleteClick = useCallback((ev: React.MouseEvent<any>) => {
        if (disabled || !isCompleteAvailable) return;
        onIsCompleteChange(!isComplete);
    }, [disabled, isComplete, isCompleteAvailable, onIsCompleteChange]);

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

    // Current privacy icon (public or private)
    const { PrivacyIcon, privacyTooltip } = useMemo(() => {
        return {
            PrivacyIcon: isPrivate ? PrivateIcon : PublicIcon,
            privacyTooltip: isPrivate ? 'Only you or your organization can see this. Press to make public' : 'Anyone can see this. Press to make private'
        }
    }, [isPrivate]);

    // Current complete icon (complete or incomplete)
    const { CompleteIcon, completeTooltip } = useMemo(() => {
        return {
            CompleteIcon: isComplete ? CompletedIcon : IncompleteIcon,
            completeTooltip: isComplete ? 'This is complete. Press to mark incomplete' : 'This is incomplete. Press to mark complete'
        }
    }, [isComplete]);

    return (
        <Box
            padding={1}
            zIndex={zIndex}
            sx={{
                borderRadius: '12px',
                background: palette.background.paper,
                overflowX: 'auto',
            }}
        >
            {/* Popup for selecting type of owner */}
            <ListMenu
                id={'select-owner-type-menu'}
                anchorEl={ownerDialogAnchor}
                title='Owner Type'
                data={ownerTypes}
                onSelect={handleOwnerDialogSelect}
                onClose={closeOwnerDialog}
                zIndex={zIndex + 1}
            />
            {/* Popup for selecting organization (yours or another) */}
            <OrganizationSelectOrCreateDialog
                isOpen={isOrganizationDialogOpen}
                handleAdd={handleOwnerSelect}
                handleClose={closeOrganizationDialog}
                session={session}
                zIndex={zIndex + 1}
            />
            {/* Popup for selecting a user that's not you */}
            <UserSelectDialog
                isOpen={isAnotherUserDialogOpen}
                handleAdd={handleOwnerSelect}
                handleClose={closeAnotherUserDialog}
                session={session}
                zIndex={zIndex + 1}
            />
            {/* Popup for selecting a project (yours or another) */}
            <ProjectSelectOrCreateDialog
                isOpen={isProjectDialogOpen}
                handleAdd={handleProjectSelect}
                handleClose={closeProjectDialog}
                session={session}
                zIndex={zIndex + 1}
            />
            {/* Popups for selecting a parent (yours or another) */}
            {objectType === ObjectType.Routine && <RoutineSelectOrCreateDialog
                isOpen={isParentDialogOpen}
                handleAdd={handleParentRoutineSelect}
                handleClose={closeParentDialog}
                session={session}
                zIndex={zIndex + 1}
            />}
            {objectType === ObjectType.Project && <ProjectSelectOrCreateDialog
                isOpen={isParentDialogOpen}
                handleAdd={handleParentProjectSelect}
                handleClose={closeParentDialog}
                session={session}
                zIndex={zIndex + 1}
            />}
            {/* Row of button labels */}
            <Stack
                spacing={2}
                direction="row"
                alignItems="center"
                justifyContent="center"
            >
                {isOwnerAvailable && <TextShrink id="owner" sx={{...commonLabelProps}}>Owner</TextShrink>}
                {isProjectAvailable && <TextShrink id="project" sx={{...commonLabelProps}}>Project</TextShrink>}
                {isParentAvailable && <TextShrink id="parent" sx={{...commonLabelProps}}>Parent</TextShrink>}
                {isPrivateAvailable && <TextShrink id="privacy" sx={{...commonLabelProps}}>Privacy</TextShrink>}
                {isCompleteAvailable && <TextShrink id="complete" sx={{...commonLabelProps}}>Complete?</TextShrink>}
            </Stack>
            {/* Buttons row */}
            <Stack
                spacing={2}
                direction="row"
                alignItems="center"
                justifyContent="center"
            >

                {/* Owner button */}
                {isOwnerAvailable && <Tooltip title={ownerTooltip}>
                    <IconButton sx={{ ...commonButtonProps(palette) }} onClick={handleOwnerClick}>
                        {OwnerIcon && <OwnerIcon sx={{ ...commonIconProps }} />}
                    </IconButton>
                </Tooltip>}
                {/* Project button */}
                {isProjectAvailable && <Tooltip title={projectTooltip}>
                    <IconButton sx={{ ...commonButtonProps(palette) }} onClick={handleProjectClick}>
                        {ProjectIcon && <ProjectIcon sx={{ ...commonIconProps }} />}
                    </IconButton>
                </Tooltip>}
                {/* Parent button */}
                {isParentAvailable && <Tooltip title={parentTooltip}>
                    <IconButton sx={{ ...commonButtonProps(palette) }} onClick={handleParentClick}>
                        {ParentIcon && <ParentIcon sx={{ ...commonIconProps }} />}
                    </IconButton>
                </Tooltip>}
                {/* Privacy button */}
                {isPrivateAvailable && <Tooltip title={privacyTooltip}>
                    <IconButton sx={{ ...commonButtonProps(palette) }} onClick={handlePrivateClick}>
                        {PrivacyIcon && <PrivacyIcon sx={{ ...commonIconProps }} />}
                    </IconButton>
                </Tooltip>}
                {/* Complete button */}
                {isCompleteAvailable && <Tooltip title={completeTooltip}>
                    <IconButton sx={{ ...commonButtonProps(palette) }} onClick={handleCompleteClick}>
                        {CompleteIcon && <CompleteIcon sx={{ ...commonIconProps }} />}
                    </IconButton>
                </Tooltip>}
            </Stack>
        </Box>
    )
}
