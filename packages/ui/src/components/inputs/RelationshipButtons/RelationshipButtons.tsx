/**
 * Horizontal button list for assigning owner, project, and parent 
 * to objects
 */
import { Box, IconButton, Palette, Stack, Tooltip, useTheme } from '@mui/material';
import { useCallback, useMemo, useState } from 'react';
import { RelationshipButtonsProps, RelationshipItemOrganization, RelationshipItemProject, RelationshipItemRoutine, RelationshipItemUser, RelationshipOwner } from '../types';
import { getTranslation, getUserLanguages, ObjectType, openObject, PubSub } from 'utils';
import { ListMenu, OrganizationSelectOrCreateDialog, ProjectSelectOrCreateDialog, RoutineSelectOrCreateDialog, UserSelectDialog } from 'components/dialogs';
import { Session } from 'types';
import { ListMenuItemData } from 'components/dialogs/types';
import { TextShrink } from 'components/text/TextShrink/TextShrink';
import { CompleteIcon as CompletedIcon, InvisibleIcon, OrganizationIcon, ProjectIcon as ProjIcon, RoutineIcon, UserIcon as SelfIcon, VisibleIcon } from '@shared/icons';
import { useLocation } from '@shared/route';

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
    overflow: 'hidden',
    background: palette.primary.light,
    '&:hover': {
        background: palette.primary.light,
        filter: 'brightness(120%)',
    },
})

const commonIconProps = {
    width: "69px",
    height: "69px",
    color: "white",
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
    isFormDirty = false,
    objectType,
    onRelationshipsChange,
    relationships,
    session,
    zIndex,
}: RelationshipButtonsProps) {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const languages = useMemo(() => getUserLanguages(session), [session])

    // Determine which relationship buttons are available
    const { isCompleteAvailable, isOwnerAvailable, isPrivateAvailable, isProjectAvailable, isParentAvailable } = useMemo(() => {
        // isComplete available for projects and routines
        const isCompleteAvailable = [ObjectType.Project, ObjectType.Routine].includes(objectType);
        // Owner available for projects, routines, and standards
        const isOwnerAvailable = [ObjectType.Project, ObjectType.Routine, ObjectType.Standard].includes(objectType);
        // Roles available for organizations
        //TODO
        // isPrivate always available for now
        const isPrivateAvailable = true;
        // Project available for projects, routines, and standards
        const isProjectAvailable = [ObjectType.Project, ObjectType.Routine, ObjectType.Standard].includes(objectType);
        // Projects (i.e. setting projects assigned to object instead of project object is assigned to) available for organizations and projects
        //TODO
        // Parent available for routines TODO
        const isParentAvailable = false;//[ObjectType.Routine].includes(objectType);
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
        if (!isOwnerAvailable) return;
        ev.stopPropagation();
        // If disabled, navigate to owner
        if (disabled) {
            if (relationships.owner) openObject(relationships.owner, setLocation);
        }
        // Otherwise, open dialog
        else setOwnerDialogAnchor(ev.currentTarget)
    }, [disabled, isOwnerAvailable, relationships.owner, setLocation]);
    const closeOwnerDialog = useCallback(() => setOwnerDialogAnchor(null), []);
    const handleOwnerDialogSelect = useCallback((ownerType: OwnerTypesEnum) => {
        if (ownerType === OwnerTypesEnum.Organization) {
            openOrganizationDialog();
        } else if (ownerType === OwnerTypesEnum.AnotherUser) {
            openAnotherUserDialog();
        } else {
            onRelationshipsChange({ owner: userFromSession(session) });
        }
        closeOwnerDialog();
    }, [closeOwnerDialog, onRelationshipsChange, openAnotherUserDialog, openOrganizationDialog, session]);
    const handleOwnerSelect = useCallback((owner: RelationshipOwner) => {
        onRelationshipsChange({ owner });
    }, [onRelationshipsChange]);

    // Project dialog
    const [isProjectDialogOpen, setProjectDialogOpen] = useState<boolean>(false);
    const handleProjectClick = useCallback((ev: React.MouseEvent<any>) => {
        if (!isProjectAvailable) return;
        ev.stopPropagation();
        // If disabled, navigate to project
        if (disabled) {
            if (relationships.project) openObject(relationships.project, setLocation);
        }
        else {
            // If project was set, remove
            if (relationships.project) onRelationshipsChange({ project: null });
            // Otherwise, open project select dialog
            else setProjectDialogOpen(true);
        }
    }, [disabled, isProjectAvailable, onRelationshipsChange, relationships.project, setLocation]);
    const closeProjectDialog = useCallback(() => { setProjectDialogOpen(false); }, [setProjectDialogOpen]);
    const handleProjectSelect = useCallback((project: RelationshipItemProject) => {
        onRelationshipsChange({ project });
    }, [onRelationshipsChange]);

    // Parent dialog
    const [isParentDialogOpen, setParentDialogOpen] = useState<boolean>(false);
    const handleParentClick = useCallback((ev: React.MouseEvent<any>) => {
        if (!isParentAvailable) return;
        ev.stopPropagation();
        // If disabled, navigate to parent
        if (disabled) {
            if (relationships.parent) openObject(relationships.parent, setLocation);
        }
        else {
            // If parent was set, remove
            if (relationships.parent) onRelationshipsChange({ parent: null });
            else {
                // If form is dirty, prompt to confirm (since data will be lost)
                if (isFormDirty) {
                    PubSub.get().publishAlertDialog({
                        message: 'Selecting a parent to copy will override existing data. Continue?',
                        buttons: [
                            {
                                text: 'Yes', onClick: () => { setParentDialogOpen(true); }
                            },
                            {
                                text: "No", onClick: () => { }
                            },
                        ]
                    });
                }
                // Otherwise, open parent select dialog
                else setParentDialogOpen(true);
            }
        }
    }, [disabled, isFormDirty, isParentAvailable, onRelationshipsChange, relationships.parent, setLocation]);
    const closeParentDialog = useCallback(() => { setParentDialogOpen(false); }, [setParentDialogOpen]);
    const handleParentProjectSelect = useCallback((parent: RelationshipItemProject) => {
        onRelationshipsChange({ parent });
    }, [onRelationshipsChange]);
    const handleParentRoutineSelect = useCallback((parent: RelationshipItemRoutine) => {
        onRelationshipsChange({ parent });
    }, [onRelationshipsChange]);

    // Handle private click
    const handlePrivateClick = useCallback((ev: React.MouseEvent<any>) => {
        if (disabled || !isPrivateAvailable) return;
        onRelationshipsChange({ isPrivate: !relationships.isPrivate });
    }, [disabled, relationships.isPrivate, isPrivateAvailable, onRelationshipsChange]);

    // Handle complete click
    const handleCompleteClick = useCallback((ev: React.MouseEvent<any>) => {
        if (disabled || !isCompleteAvailable) return;
        onRelationshipsChange({ isComplete: !relationships.isComplete });
    }, [disabled, relationships.isComplete, isCompleteAvailable, onRelationshipsChange]);

    // Current owner icon
    const { OwnerIcon, ownerTooltip } = useMemo(() => {
        // If no owner data, marked as anonymous
        if (!relationships.owner) return {
            OwnerIcon: null,
            ownerTooltip: `Marked as anonymous${disabled ? '' : '. Press to set owner'}`
        };
        // If owner is organization, use organization icon
        if (relationships.owner.__typename === 'Organization') {
            const OwnerIcon = OrganizationIcon;
            // Button color indicates if you can modify the organization, or if someone will have to approve it
            const canEdit = (relationships.owner as RelationshipItemOrganization).permissionsOrganization?.canEdit === true;
            const ownerName = getTranslation(relationships.owner as RelationshipItemOrganization, 'name', languages, true) ?? 'organization';
            return {
                OwnerIcon,
                ownerTooltip: `Owner: ${ownerName}${!canEdit ? ' (requires approval)' : ''}`
            };
        }
        // If owner is user, use self icon
        const OwnerIcon = SelfIcon;
        // Button color indicates if it's your own account, or if someone will have to approve it
        const canEdit = relationships.owner.id === session.id;
        const ownerName = (relationships.owner as RelationshipItemUser).name;
        return {
            OwnerIcon,
            ownerTooltip: `Owner: ${canEdit ? 'Self' : (ownerName + ' (requires approval)')}`
        };
    }, [disabled, languages, relationships.owner, session.id]);

    // Current project icon
    const { ProjectIcon, projectTooltip } = useMemo(() => {
        // If no project data, marked as unset
        if (!relationships.project) return {
            ProjectIcon: null,
            projectTooltip: disabled ? '' : 'Press to assign to a project'
        };
        const canEdit = relationships.project.permissionsProject?.canEdit === true;
        const projectName = getTranslation(relationships.project as RelationshipItemProject, 'name', languages, true) ?? 'project';
        return {
            ProjectIcon: ProjIcon,
            projectTooltip: `Project: ${projectName}${!canEdit ? ' (requires approval)' : ''}`
        };
    }, [disabled, languages, relationships.project]);

    // Current parent icon
    const { ParentIcon, parentTooltip } = useMemo(() => {
        // If no parent data, marked as unset
        if (!relationships.parent) return {
            ParentIcon: null,
            parentTooltip: disabled ? '' : 'Press to copy from a parent (will override entered data)'
        };
        // If parent is project, use project icon
        if (relationships.parent.__typename === 'Project') {
            const ParentIcon = ProjIcon;
            // Button color indicates if you can modify the project, or if someone will have to approve it
            const canEdit = (relationships.parent as RelationshipItemProject).permissionsProject?.canEdit === true;
            const parentName = getTranslation(relationships.parent as RelationshipItemProject, 'name', languages, true) ?? 'project';
            return {
                ParentIcon,
                parentTooltip: `Parent: ${parentName}${!canEdit ? ' (requires approval)' : ''}`
            };
        }
        // If parent is routine, use routine icon
        const ParentIcon = RoutineIcon;
        // Button color indicates if you can modify the routine, or if someone will have to approve it
        const canEdit = (relationships.parent as RelationshipItemRoutine).permissionsRoutine?.canEdit === true;
        const parentName = getTranslation(relationships.parent as RelationshipItemRoutine, 'title', languages, true) ?? 'routine';
        return {
            ParentIcon,
            parentTooltip: `Parent: ${parentName}${!canEdit ? ' (requires approval)' : ''}`
        };
    }, [disabled, languages, relationships.parent]);

    // Current privacy icon (public or private)
    const { PrivacyIcon, privacyTooltip } = useMemo(() => {
        return {
            PrivacyIcon: relationships.isPrivate ? InvisibleIcon : VisibleIcon,
            privacyTooltip: relationships.isPrivate ? `Only you or your organization can see this${disabled ? '' : '. Press to make public'}` : `Anyone can see this${disabled ? '' : '. Press to make private'}`
        }
    }, [disabled, relationships.isPrivate]);

    // Current complete icon (complete or incomplete)
    const { CompleteIcon, completeTooltip } = useMemo(() => {
        return {
            CompleteIcon: relationships.isComplete ? CompletedIcon : null,
            completeTooltip: relationships.isComplete ? `This is complete${disabled ? '' : '. Press to mark incomplete'}` : `This is incomplete${disabled ? '' : '. Press to mark complete'}`
        }
    }, [disabled, relationships.isComplete]);

    return (
        <Box
            padding={1}
            zIndex={zIndex}
            sx={{
                borderRadius: '12px',
                background: palette.mode === 'dark' ? palette.background.paper : palette.background.default,
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
                {isOwnerAvailable && <TextShrink id="owner" sx={{ ...commonLabelProps }}>Owner</TextShrink>}
                {isProjectAvailable && <TextShrink id="project" sx={{ ...commonLabelProps }}>Project</TextShrink>}
                {isParentAvailable && <TextShrink id="parent" sx={{ ...commonLabelProps }}>Parent</TextShrink>}
                {isPrivateAvailable && <TextShrink id="privacy" sx={{ ...commonLabelProps }}>Privacy</TextShrink>}
                {isCompleteAvailable && <TextShrink id="complete" sx={{ ...commonLabelProps }}>Complete?</TextShrink>}
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
                        {OwnerIcon && <OwnerIcon { ...commonIconProps } />}
                    </IconButton>
                </Tooltip>}
                {/* Project button */}
                {isProjectAvailable && <Tooltip title={projectTooltip}>
                    <IconButton sx={{ ...commonButtonProps(palette) }} onClick={handleProjectClick}>
                        {ProjectIcon && <ProjectIcon{ ...commonIconProps } />}
                    </IconButton>
                </Tooltip>}
                {/* Parent button */}
                {isParentAvailable && <Tooltip title={parentTooltip}>
                    <IconButton sx={{ ...commonButtonProps(palette) }} onClick={handleParentClick}>
                        {ParentIcon && <ParentIcon { ...commonIconProps } />}
                    </IconButton>
                </Tooltip>}
                {/* Privacy button */}
                {isPrivateAvailable && <Tooltip title={privacyTooltip}>
                    <IconButton sx={{ ...commonButtonProps(palette) }} onClick={handlePrivateClick}>
                        {PrivacyIcon && <PrivacyIcon {...commonIconProps} />}
                    </IconButton>
                </Tooltip>}
                {/* Complete button */}
                {isCompleteAvailable && <Tooltip title={completeTooltip}>
                    <IconButton sx={{ ...commonButtonProps(palette) }} onClick={handleCompleteClick}>
                        {CompleteIcon && <CompleteIcon {...commonIconProps} />}
                    </IconButton>
                </Tooltip>}
            </Stack>
        </Box>
    )
}
