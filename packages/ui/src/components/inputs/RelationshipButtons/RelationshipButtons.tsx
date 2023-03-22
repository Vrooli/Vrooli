/**
 * Horizontal button list for assigning owner, project, and parent 
 * to objects
 */
import { Box, Stack, Tooltip, useTheme } from '@mui/material';
import { Session } from '@shared/consts';
import { CompleteIcon as CompletedIcon, InvisibleIcon, OrganizationIcon, ProjectIcon as ProjIcon, RoutineIcon, UserIcon as SelfIcon, VisibleIcon } from '@shared/icons';
import { useLocation } from '@shared/route';
import { isOfType } from '@shared/utils';
import { ColorIconButton } from 'components/buttons/ColorIconButton/ColorIconButton';
import { ListMenu } from 'components/dialogs/ListMenu/ListMenu';
import { SelectOrCreateDialog } from 'components/dialogs/selectOrCreates';
import { SelectOrCreateObjectType } from 'components/dialogs/selectOrCreates/types';
import { ListMenuItemData } from 'components/dialogs/types';
import { TextShrink } from 'components/text/TextShrink/TextShrink';
import { useField } from 'formik';
import { useCallback, useContext, useMemo, useState } from 'react';
import { noSelect } from 'styles';
import { getCurrentUser } from 'utils/authentication/session';
import { firstString } from 'utils/display/stringTools';
import { getTranslation, getUserLanguages } from 'utils/display/translationTools';
import { openObject } from 'utils/navigation/openObject';
import { PubSub } from 'utils/pubsub';
import { SessionContext } from 'utils/SessionContext';
import { RelationshipButtonsProps, RelationshipItemOrganization, RelationshipItemProjectVersion, RelationshipItemRoutineVersion, RelationshipItemUser, RelationshipOwner } from '../types';

/**
 * Converts session to user object
 * @param session Current user session
 * @returns User object
 */
export const userFromSession = (session: Session): Exclude<RelationshipOwner, null> => ({
    __typename: 'User',
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
    zIndex,
    sx,
}: RelationshipButtonsProps) {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const languages = useMemo(() => getUserLanguages(session), [session])

    // Formik fields. Some may be undefined if not relevant to object type
    const [isCompleteField, , isCompleteHelpers] = useField('isComplete');
    const [isPrivateField, , isPrivateHelpers] = useField('isPrivate');
    const [ownerField, , ownerHelpers] = useField('owner');
    const [ownerRootField, , ownerRootHelpers] = useField('root.owner');
    const [parentField, , parentHelpers] = useField('parent');
    const [parentRootField, , parentRootHelpers] = useField('root.parent');
    const [projectField, , projectHelpers] = useField('project');
    const [projectRootField, , projectRootHelpers] = useField('root.project');

    // Determine which relationship buttons are available on this object type
    // when in edit mode
    const { isCompleteAvailable, isOwnerAvailable, isPrivateAvailable, isProjectAvailable, isParentAvailable } = useMemo(() => {
        // isComplete available for projects and routines
        const isCompleteAvailable = isOfType(objectType, 'Project', 'Routine');
        // Owner available for projects, routines, and standards
        const isOwnerAvailable = isOfType(objectType, 'Project', 'Routine', 'Standard');
        // Roles available for organizations
        //TODO
        // isPrivate always available for now
        const isPrivateAvailable = true;
        // Project available for projects, routines, and standards
        const isProjectAvailable = isOfType(objectType, 'Project', 'Routine', 'Standard');
        // Projects (i.e. setting projects assigned to object instead of project object is assigned to) available for organizations and projects
        //TODO
        // Parent available for routines TODO
        const isParentAvailable = false;//ifOfType(objectType, 'Routine');
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
        const ownerId = ownerField?.value?.id ?? ownerRootField?.value?.id;
        if (owner?.id === ownerId) return;
        //TODO determine which to use
        ownerHelpers.setValue(owner);
    }, [ownerField?.value?.id, ownerHelpers, ownerRootField?.value?.id]);
    // Project dialog
    const [isProjectDialogOpen, setProjectDialogOpen] = useState<boolean>(false);
    const handleProjectClick = useCallback((ev: React.MouseEvent<any>) => {
        if (!isProjectAvailable) return;
        ev.stopPropagation();
        const project = projectField?.value ?? projectRootField?.value;
        // If not editing, navigate to project
        if (!isEditing) {
            if (project) openObject(project, setLocation);
        }
        else {
            // If project was set, remove
            if (project) {
                //TODO determine which to use
                projectHelpers.setValue(null);
            }
            // Otherwise, open project select dialog
            else setProjectDialogOpen(true);
        }
    }, [isEditing, isProjectAvailable, projectField?.value, projectHelpers, projectRootField?.value, setLocation]);
    const closeProjectDialog = useCallback(() => { setProjectDialogOpen(false); }, [setProjectDialogOpen]);
    const handleProjectSelect = useCallback((project: RelationshipItemProjectVersion) => {
        const projectId = projectField?.value?.id ?? projectRootField?.value?.id;
        if (project?.id === projectId) return;
        //TODO determine which to use
        projectHelpers.setValue(project);
    }, [projectField?.value?.id, projectHelpers, projectRootField?.value?.id]);
    // Parent dialog
    const [isParentDialogOpen, setParentDialogOpen] = useState<boolean>(false);
    const handleParentClick = useCallback((ev: React.MouseEvent<any>) => {
        if (!isParentAvailable) return;
        ev.stopPropagation();
        const parent = parentField?.value ?? parentRootField?.value;
        // If not editing, navigate to parent
        if (!isEditing) {
            if (parent) openObject(parent, setLocation);
        }
        else {
            // If parent was set, remove
            if (parent) {
                //TODO determine which to use
                parentHelpers.setValue(null);
            }
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
    }, [isEditing, isFormDirty, isParentAvailable, parentField?.value, parentHelpers, parentRootField?.value, setLocation]);
    const closeParentDialog = useCallback(() => { setParentDialogOpen(false); }, [setParentDialogOpen]);
    const handleParentSelect = useCallback((parent: any) => {
        const parentId = parentField?.value?.id ?? parentRootField?.value?.id;
        if (parent?.id === parentId) return;
        //TODO determine which to use
        parentHelpers.setValue(parent);
    }, [parentField?.value?.id, parentHelpers, parentRootField?.value?.id]);
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
        const owner = ownerField?.value ?? ownerRootField?.value;
        // If not editing, navigate to owner
        if (!isEditing) {
            if (owner) openObject(owner, setLocation);
        }
        // Otherwise, open dialog
        else setOwnerDialogAnchor(ev.currentTarget)
    }, [isEditing, isOwnerAvailable, ownerField?.value, ownerRootField?.value, setLocation]);
    const closeOwnerDialog = useCallback(() => setOwnerDialogAnchor(null), []);
    const handleOwnerDialogSelect = useCallback((ownerType: OwnerTypesEnum) => {
        if (ownerType === OwnerTypesEnum.Organization) {
            openOrganizationDialog();
        } else if (ownerType === OwnerTypesEnum.AnotherUser) {
            openAnotherUserDialog();
        } else {
            //TODO determine which to use
            ownerHelpers.setValue(session ? userFromSession(session) : undefined);
        }
        closeOwnerDialog();
    }, [closeOwnerDialog, openAnotherUserDialog, openOrganizationDialog, ownerHelpers, session]);

    // Handle private click
    const handlePrivateClick = useCallback((ev: React.MouseEvent<any>) => {
        if (!isEditing || !isPrivateAvailable) return;
        isPrivateHelpers.setValue(!isPrivateField?.value);
    }, [isEditing, isPrivateAvailable, isPrivateHelpers, isPrivateField?.value]);

    // Handle complete click
    const handleCompleteClick = useCallback((ev: React.MouseEvent<any>) => {
        if (!isEditing || !isCompleteAvailable) return;
        isCompleteHelpers.setValue(!isCompleteField?.value);
    }, [isEditing, isCompleteAvailable, isCompleteHelpers, isCompleteField?.value]);

    // Current owner icon
    const { OwnerIcon, ownerTooltip } = useMemo(() => {
        const owner = ownerField?.value ?? ownerRootField?.value;
        // If no owner data, marked as anonymous
        if (!owner) return {
            OwnerIcon: null,
            ownerTooltip: `Marked as anonymous${isEditing ? '' : '. Press to set owner'}`
        };
        // If owner is organization, use organization icon
        if (owner.__typename === 'Organization') {
            const OwnerIcon = OrganizationIcon;
            const ownerName = firstString(getTranslation(owner as RelationshipItemOrganization, languages, true).name, 'organization');
            return {
                OwnerIcon,
                ownerTooltip: `Owner: ${ownerName}`
            };
        }
        // If owner is user, use self icon
        const OwnerIcon = SelfIcon;
        const isSelf = owner.id === getCurrentUser(session).id;
        const ownerName = (owner as RelationshipItemUser).name;
        return {
            OwnerIcon,
            ownerTooltip: `Owner: ${isSelf ? 'Self' : ownerName}`
        };
    }, [isEditing, languages, ownerField?.value, ownerRootField?.value, session]);

    // Current project icon
    const { ProjectIcon, projectTooltip } = useMemo(() => {
        const project = projectField?.value ?? projectRootField?.value;
        // If no project data, marked as unset
        if (!project) return {
            ProjectIcon: null,
            projectTooltip: isEditing ? '' : 'Press to assign to a project'
        };
        const projectName = firstString(getTranslation(project as RelationshipItemProjectVersion, languages, true).name, 'project');
        return {
            ProjectIcon: ProjIcon,
            projectTooltip: `Project: ${projectName}`
        };
    }, [isEditing, languages, projectField?.value, projectRootField?.value]);

    // Current parent icon
    const { ParentIcon, parentTooltip } = useMemo(() => {
        // If no parent data, marked as unset
        const parent = parentField?.value ?? parentRootField?.value;
        if (!parent) return {
            ParentIcon: null,
            parentTooltip: isEditing ? '' : 'Press to copy from a parent (will override entered data)'
        };
        // If parent is project, use project icon
        if (parent.__typename === 'ProjectVersion') {
            const ParentIcon = ProjIcon;
            const parentName = firstString(getTranslation(parent as RelationshipItemProjectVersion, languages, true).name, 'project');
            return {
                ParentIcon,
                parentTooltip: `Parent: ${parentName}`
            };
        }
        // If parent is routine, use routine icon
        const ParentIcon = RoutineIcon;
        const parentName = firstString(getTranslation(parent as RelationshipItemRoutineVersion, languages, true).name, 'routine');
        return {
            ParentIcon,
            parentTooltip: `Parent: ${parentName}`
        };
    }, [isEditing, languages, parentField?.value, parentRootField?.value]);

    // Current privacy icon (public or private)
    const { PrivacyIcon, privacyTooltip } = useMemo(() => {
        const isPrivate = isPrivateField?.value;
        return {
            PrivacyIcon: isPrivate ? InvisibleIcon : VisibleIcon,
            privacyTooltip: isPrivate ? `Only you or your organization can see this${isEditing ? '' : '. Press to make public'}` : `Anyone can see this${isEditing ? '' : '. Press to make private'}`
        }
    }, [isEditing, isPrivateField?.value]);

    // Current complete icon (complete or incomplete)
    const { CompleteIcon, completeTooltip } = useMemo(() => {
        const isComplete = isCompleteField?.value;
        return {
            CompleteIcon: isComplete ? CompletedIcon : null,
            completeTooltip: isComplete ? `This is complete${isEditing ? '' : '. Press to mark incomplete'}` : `This is incomplete${isEditing ? '' : '. Press to mark complete'}`
        }
    }, [isCompleteField?.value, isEditing]);

    return (
        <Box
            padding={1}
            zIndex={zIndex}
            sx={{
                borderRadius: '12px',
                background: palette.mode === 'dark' ? palette.background.paper : palette.background.default,
                overflowX: 'auto',
                ...noSelect,
                ...(sx ?? {}),
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
            {selectOrCreateType && <SelectOrCreateDialog
                isOpen={Boolean(selectOrCreateType)}
                handleAdd={selectOrCreateHandleAdd}
                handleClose={selectOrCreateHandleClose}
                objectType={selectOrCreateType}
                zIndex={zIndex + 1}
            />}
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