/**
 * Horizontal button list for assigning owner, project, and parent 
 * to objects
 */
import { Box, Stack, Tooltip, useTheme } from '@mui/material';
import { useCallback, useMemo, useState } from 'react';
import { RelationshipButtonsProps, RelationshipItemOrganization, RelationshipItemProjectVersion, RelationshipItemRoutineVersion, RelationshipItemUser, RelationshipOwner } from '../types';
import { firstString, getTranslation, getUserLanguages, openObject, PubSub } from 'utils';
import { ListMenu, SelectOrCreateDialog } from 'components/dialogs';
import { ListMenuItemData } from 'components/dialogs/types';
import { TextShrink } from 'components/text/TextShrink/TextShrink';
import { CompleteIcon as CompletedIcon, InvisibleIcon, OrganizationIcon, ProjectIcon as ProjIcon, RoutineIcon, UserIcon as SelfIcon, VisibleIcon } from '@shared/icons';
import { useLocation } from '@shared/route';
import { getCurrentUser } from 'utils/authentication';
import { ColorIconButton } from 'components/buttons';
import { noSelect } from 'styles';
import { GqlModelType, Session } from '@shared/consts';
import { SelectOrCreateObjectType } from 'components/dialogs/selectOrCreates/types';

/**
 * Converts session to user object
 * @param session Current user session
 * @returns User object
 */
export const userFromSession = (session: Session): Exclude<RelationshipOwner, null> => ({
    type: GqlModelType.User,
    id: getCurrentUser(session).id as string,
    handle: null,
    name: 'Self',
})

const commonButtonProps = (isEditing: boolean, canPressWhenNotEditing: boolean) => ({
    width: { xs: '58px', md: '69px' },
    height: { xs: '58px', md: '69px' },
    overflow: 'hidden',
    boxShadow: !isEditing && !canPressWhenNotEditing ? 0 : 4,
    pointerEvents: !isEditing && !canPressWhenNotEditing ? 'none' : 'auto',
})

const commonIconProps = () => ({
    width: '69px',
    height: '69px',
    color: "white",
})

const commonLabelProps = () => ({
    width: { xs: '58px', md: '69px' },
    textAlign: 'center',
})

enum OwnerTypesEnum {
    Self = 'Self',
    Organization = 'Organization',
    AnotherUser = 'AnotherUser',
}

const ownerTypes: ListMenuItemData<OwnerTypesEnum>[] = [
    { label: 'Self', value: OwnerTypesEnum.Self },
    { label: 'Organization', value: OwnerTypesEnum.Organization },
    // { label: 'Another User (Requires Permission)', value: OwnerTypesEnum.AnotherUser },
]

export function RelationshipButtons({
    isEditing,
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

    // Determine which relationship buttons are available on this object type
    // when in edit mode
    const { isCompleteAvailable, isOwnerAvailable, isPrivateAvailable, isProjectAvailable, isParentAvailable } = useMemo(() => {
        // isComplete available for projects and routines
        const isCompleteAvailable = ['Project', 'Routine'].includes(objectType);
        // Owner available for projects, routines, and standards
        const isOwnerAvailable = ['Project', 'Routine', 'Standard'].includes(objectType);
        // Roles available for organizations
        //TODO
        // isPrivate always available for now
        const isPrivateAvailable = true;
        // Project available for projects, routines, and standards
        const isProjectAvailable = ['Project', 'Routine', 'Standard'].includes(objectType);
        // Projects (i.e. setting projects assigned to object instead of project object is assigned to) available for organizations and projects
        //TODO
        // Parent available for routines TODO
        const isParentAvailable = false;//['Routine'].includes(objectType);
        return { isCompleteAvailable, isOwnerAvailable, isPrivateAvailable, isProjectAvailable, isParentAvailable }
    }, [objectType])

    // Organization/User owner dialogs (displayed after selecting owner dialog)
    const [isOrganizationDialogOpen, setOrganizationDialogOpen] = useState<boolean>(false);
    const openOrganizationDialog = useCallback(() => { setOrganizationDialogOpen(true) }, [setOrganizationDialogOpen]);
    const closeOrganizationDialog = useCallback(() => { setOrganizationDialogOpen(false); }, [setOrganizationDialogOpen]);
    const [isAnotherUserDialogOpen, setAnotherUserDialogOpen] = useState<boolean>(false);
    const openAnotherUserDialog = useCallback(() => { setAnotherUserDialogOpen(true) }, [setAnotherUserDialogOpen]);
    const closeAnotherUserDialog = useCallback(() => { setAnotherUserDialogOpen(false); }, [setAnotherUserDialogOpen]);
    const handleOwnerSelect = useCallback((owner: RelationshipOwner) => {
        if (owner?.id === relationships.owner?.id) return;
        onRelationshipsChange({ owner });
    }, [onRelationshipsChange, relationships.owner?.id]);
    // Project dialog
    const [isProjectDialogOpen, setProjectDialogOpen] = useState<boolean>(false);
    const handleProjectClick = useCallback((ev: React.MouseEvent<any>) => {
        if (!isProjectAvailable) return;
        ev.stopPropagation();
        // If not editing, navigate to project
        if (!isEditing) {
            if (relationships.project) openObject(relationships.project, setLocation);
        }
        else {
            // If project was set, remove
            if (relationships.project) onRelationshipsChange({ project: null });
            // Otherwise, open project select dialog
            else setProjectDialogOpen(true);
        }
    }, [isEditing, isProjectAvailable, onRelationshipsChange, relationships.project, setLocation]);
    const closeProjectDialog = useCallback(() => { setProjectDialogOpen(false); }, [setProjectDialogOpen]);
    const handleProjectSelect = useCallback((project: RelationshipItemProjectVersion) => {
        if (project?.id === relationships.project?.id) return;
        onRelationshipsChange({ project });
    }, [onRelationshipsChange, relationships.project?.id]);
    // Parent dialog
    const [isParentDialogOpen, setParentDialogOpen] = useState<boolean>(false);
    const handleParentClick = useCallback((ev: React.MouseEvent<any>) => {
        if (!isParentAvailable) return;
        ev.stopPropagation();
        // If not editing, navigate to parent
        if (!isEditing) {
            if (relationships.parent) openObject(relationships.parent, setLocation);
        }
        else {
            // If parent was set, remove
            if (relationships.parent) onRelationshipsChange({ parent: null });
            else {
                // If form is dirty, prompt to confirm (since data will be lost)
                if (isFormDirty) {
                    PubSub.get().publishAlertDialog({
                        messageKey: 'ParentOverrideConfirm',
                        buttons: [
                            { labelKey: 'Yes', onClick: () => { setParentDialogOpen(true); } },
                            { labelKey: "No", onClick: () => { } },
                        ]
                    });
                }
                // Otherwise, open parent select dialog
                else setParentDialogOpen(true);
            }
        }
    }, [isEditing, isFormDirty, isParentAvailable, onRelationshipsChange, relationships.parent, setLocation]);
    const closeParentDialog = useCallback(() => { setParentDialogOpen(false); }, [setParentDialogOpen]);
    const handleParentSelect = useCallback((parent: any) => {
        if (parent?.id === relationships.parent?.id) return;
        onRelationshipsChange({ parent });
    }, [onRelationshipsChange, relationships.parent?.id]);
    // SelectOrCreateDialog
    const [selectOrCreateType, selectOrCreateHandleAdd, selectOrCreateHandleClose] = useMemo<[SelectOrCreateObjectType | null, (item: any) => any, () => void]>(() => {
        if (isOrganizationDialogOpen) return ['Organization', handleOwnerSelect, closeOrganizationDialog];
        else if (isAnotherUserDialogOpen) return ['User', handleOwnerSelect, closeAnotherUserDialog];
        else if (isProjectDialogOpen) return ['ProjectVersion', handleProjectSelect, closeProjectDialog];
        else if (isParentDialogOpen) return [objectType as SelectOrCreateObjectType, handleParentSelect, closeParentDialog];
        return [null, () => { }, () => { }];
    }, [isOrganizationDialogOpen, handleOwnerSelect, closeOrganizationDialog, isAnotherUserDialogOpen, closeAnotherUserDialog, isProjectDialogOpen, handleProjectSelect, closeProjectDialog, isParentDialogOpen, objectType, handleParentSelect, closeParentDialog]);

    // Owner list dialog (select self, organization, or another user)
    const [ownerDialogAnchor, setOwnerDialogAnchor] = useState<any>(null);
    const handleOwnerClick = useCallback((ev: React.MouseEvent<any>) => {
        if (!isOwnerAvailable) return;
        ev.stopPropagation();
        // If not editing, navigate to owner
        if (!isEditing) {
            if (relationships.owner) openObject(relationships.owner, setLocation);
        }
        // Otherwise, open dialog
        else setOwnerDialogAnchor(ev.currentTarget)
    }, [isEditing, isOwnerAvailable, relationships.owner, setLocation]);
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

    // Handle private click
    const handlePrivateClick = useCallback((ev: React.MouseEvent<any>) => {
        if (!isEditing || !isPrivateAvailable) return;
        onRelationshipsChange({ isPrivate: !relationships.isPrivate });
    }, [isEditing, relationships.isPrivate, isPrivateAvailable, onRelationshipsChange]);

    // Handle complete click
    const handleCompleteClick = useCallback((ev: React.MouseEvent<any>) => {
        if (!isEditing || !isCompleteAvailable) return;
        onRelationshipsChange({ isComplete: !relationships.isComplete });
    }, [isEditing, relationships.isComplete, isCompleteAvailable, onRelationshipsChange]);

    // Current owner icon
    const { OwnerIcon, ownerTooltip } = useMemo(() => {
        // If no owner data, marked as anonymous
        if (!relationships.owner) return {
            OwnerIcon: null,
            ownerTooltip: `Marked as anonymous${isEditing ? '' : '. Press to set owner'}`
        };
        // If owner is organization, use organization icon
        if (relationships.owner.type === 'Organization') {
            const OwnerIcon = OrganizationIcon;
            const ownerName = firstString(getTranslation(relationships.owner as RelationshipItemOrganization, languages, true).name, 'organization');
            return {
                OwnerIcon,
                ownerTooltip: `Owner: ${ownerName}`
            };
        }
        // If owner is user, use self icon
        const OwnerIcon = SelfIcon;
        const isSelf = relationships.owner.id === getCurrentUser(session).id;
        const ownerName = (relationships.owner as RelationshipItemUser).name;
        return {
            OwnerIcon,
            ownerTooltip: `Owner: ${isSelf ? 'Self' : ownerName}`
        };
    }, [isEditing, languages, relationships.owner, session]);

    // Current project icon
    const { ProjectIcon, projectTooltip } = useMemo(() => {
        // If no project data, marked as unset
        if (!relationships.project) return {
            ProjectIcon: null,
            projectTooltip: isEditing ? '' : 'Press to assign to a project'
        };
        const projectName = firstString(getTranslation(relationships.project as RelationshipItemProjectVersion, languages, true).name, 'project');
        return {
            ProjectIcon: ProjIcon,
            projectTooltip: `Project: ${projectName}`
        };
    }, [isEditing, languages, relationships.project]);

    // Current parent icon
    const { ParentIcon, parentTooltip } = useMemo(() => {
        // If no parent data, marked as unset
        if (!relationships.parent) return {
            ParentIcon: null,
            parentTooltip: isEditing ? '' : 'Press to copy from a parent (will override entered data)'
        };
        // If parent is project, use project icon
        if (relationships.parent.type === 'ProjectVersion') {
            const ParentIcon = ProjIcon;
            const parentName = firstString(getTranslation(relationships.parent as RelationshipItemProjectVersion, languages, true).name, 'project');
            return {
                ParentIcon,
                parentTooltip: `Parent: ${parentName}`
            };
        }
        // If parent is routine, use routine icon
        const ParentIcon = RoutineIcon;
        const parentName = firstString(getTranslation(relationships.parent as RelationshipItemRoutineVersion, languages, true).name, 'routine');
        return {
            ParentIcon,
            parentTooltip: `Parent: ${parentName}`
        };
    }, [isEditing, languages, relationships.parent]);

    // Current privacy icon (public or private)
    const { PrivacyIcon, privacyTooltip } = useMemo(() => {
        return {
            PrivacyIcon: relationships.isPrivate ? InvisibleIcon : VisibleIcon,
            privacyTooltip: relationships.isPrivate ? `Only you or your organization can see this${isEditing ? '' : '. Press to make public'}` : `Anyone can see this${isEditing ? '' : '. Press to make private'}`
        }
    }, [isEditing, relationships.isPrivate]);

    // Current complete icon (complete or incomplete)
    const { CompleteIcon, completeTooltip } = useMemo(() => {
        return {
            CompleteIcon: relationships.isComplete ? CompletedIcon : null,
            completeTooltip: relationships.isComplete ? `This is complete${isEditing ? '' : '. Press to mark incomplete'}` : `This is incomplete${isEditing ? '' : '. Press to mark complete'}`
        }
    }, [isEditing, relationships.isComplete]);

    return (
        <Box
            padding={1}
            zIndex={zIndex}
            sx={{
                borderRadius: '12px',
                background: palette.mode === 'dark' ? palette.background.paper : palette.background.default,
                overflowX: 'auto',
                ...noSelect,
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
            {/* Popup for selecting organization, user, etc. */}
            <SelectOrCreateDialog
                isOpen={Boolean(selectOrCreateType)}
                handleAdd={selectOrCreateHandleAdd}
                handleClose={selectOrCreateHandleClose}
                objectType={selectOrCreateType ?? 'User'} // Default can be anything. Only here to satisfy TS
                session={session}
                zIndex={zIndex + 1}
            />
            {/* Row of button labels */}
            <Stack
                spacing={{ xs: 1, sm: 1.5, md: 2 }}
                direction="row"
                alignItems="center"
                justifyContent="center"
            >
                {isOwnerAvailable && (isEditing || Boolean(OwnerIcon)) && <TextShrink id="owner" sx={{ ...commonLabelProps() }}>Owner</TextShrink>}
                {isProjectAvailable && (isEditing || Boolean(ProjectIcon)) && <TextShrink id="project" sx={{ ...commonLabelProps() }}>Project</TextShrink>}
                {isParentAvailable && (isEditing || Boolean(ParentIcon)) && <TextShrink id="parent" sx={{ ...commonLabelProps() }}>Parent</TextShrink>}
                {isPrivateAvailable && (isEditing || Boolean(PrivacyIcon)) && <TextShrink id="privacy" sx={{ ...commonLabelProps() }}>Privacy</TextShrink>}
                {isCompleteAvailable && (isEditing || Boolean(CompleteIcon)) && <TextShrink id="complete" sx={{ ...commonLabelProps() }}>Complete?</TextShrink>}
            </Stack>
            {/* Buttons row */}
            <Stack
                spacing={{ xs: 1, sm: 1.5, md: 2 }}
                direction="row"
                alignItems="center"
                justifyContent="center"
            >

                {/* Owner button */}
                {isOwnerAvailable && (isEditing || Boolean(OwnerIcon)) && <Tooltip title={ownerTooltip}>
                    <ColorIconButton background={palette.primary.light} sx={{ ...commonButtonProps(isEditing, true) }} onClick={handleOwnerClick}>
                        {OwnerIcon && <OwnerIcon {...commonIconProps()} />}
                    </ColorIconButton>
                </Tooltip>}
                {/* Project button */}
                {isProjectAvailable && (isEditing || Boolean(ProjectIcon)) && <Tooltip title={projectTooltip}>
                    <ColorIconButton background={palette.primary.light} sx={{ ...commonButtonProps(isEditing, true) }} onClick={handleProjectClick}>
                        {ProjectIcon && <ProjectIcon {...commonIconProps()} />}
                    </ColorIconButton>
                </Tooltip>}
                {/* Parent button */}
                {isParentAvailable && (isEditing || Boolean(ParentIcon)) && <Tooltip title={parentTooltip}>
                    <ColorIconButton background={palette.primary.light} sx={{ ...commonButtonProps(isEditing, true) }} onClick={handleParentClick}>
                        {ParentIcon && <ParentIcon {...commonIconProps()} />}
                    </ColorIconButton>
                </Tooltip>}
                {/* Privacy button */}
                {isPrivateAvailable && (isEditing || Boolean(PrivacyIcon)) && <Tooltip title={privacyTooltip}>
                    <ColorIconButton background={palette.primary.light} sx={{ ...commonButtonProps(isEditing, false) }} onClick={handlePrivateClick}>
                        {PrivacyIcon && <PrivacyIcon {...commonIconProps()} />}
                    </ColorIconButton>
                </Tooltip>}
                {/* Complete button */}
                {isCompleteAvailable && (isEditing || Boolean(CompleteIcon)) && <Tooltip title={completeTooltip}>
                    <ColorIconButton background={palette.primary.light} sx={{ ...commonButtonProps(isEditing, false) }} onClick={handleCompleteClick}>
                        {CompleteIcon && <CompleteIcon {...commonIconProps()} />}
                    </ColorIconButton>
                </Tooltip>}
            </Stack>
        </Box>
    )
}
