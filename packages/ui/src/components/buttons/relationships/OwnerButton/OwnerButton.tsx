import { Stack, Tooltip, useTheme } from '@mui/material';
import { OrganizationIcon, UserIcon } from '@shared/icons';
import { useLocation } from '@shared/route';
import { exists } from '@shared/utils';
import { ColorIconButton } from 'components/buttons/ColorIconButton/ColorIconButton';
import { ListMenu } from 'components/dialogs/ListMenu/ListMenu';
import { SelectOrCreateDialog } from 'components/dialogs/selectOrCreates';
import { SelectOrCreateObjectType } from 'components/dialogs/selectOrCreates/types';
import { ListMenuItemData } from 'components/dialogs/types';
import { userFromSession } from 'components/lists/RelationshipList/RelationshipList';
import { RelationshipItemOrganization, RelationshipItemUser } from 'components/lists/types';
import { TextShrink } from 'components/text/TextShrink/TextShrink';
import { useField } from 'formik';
import { useCallback, useContext, useMemo, useState } from 'react';
import { getCurrentUser } from 'utils/authentication/session';
import { firstString } from 'utils/display/stringTools';
import { getTranslation, getUserLanguages } from 'utils/display/translationTools';
import { openObject } from 'utils/navigation/openObject';
import { SessionContext } from 'utils/SessionContext';
import { OwnerShape } from 'utils/shape/models/types';
import { commonButtonProps, commonIconProps, commonLabelProps } from '../styles';
import { OwnerButtonProps } from '../types';

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

export function OwnerButton({
    isEditing,
    objectType,
    zIndex,
}: OwnerButtonProps) {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const languages = useMemo(() => getUserLanguages(session), [session])

    const [versionField, , versionHelpers] = useField('owner');
    const [rootField, , rootHelpers] = useField('root.owner');

    const isAvailable = useMemo(() => ['Project', 'Routine', 'Standard'].includes(objectType), [objectType]);

    // Organization/User owner dialogs (displayed after selecting owner dialog)
    const [isOrganizationDialogOpen, setOrganizationDialogOpen] = useState<boolean>(false);
    const openOrganizationDialog = useCallback(() => { setOrganizationDialogOpen(true) }, [setOrganizationDialogOpen]);
    const closeOrganizationDialog = useCallback(() => { setOrganizationDialogOpen(false); }, [setOrganizationDialogOpen]);
    const [isAnotherUserDialogOpen, setAnotherUserDialogOpen] = useState<boolean>(false);
    const openAnotherUserDialog = useCallback(() => { setAnotherUserDialogOpen(true) }, [setAnotherUserDialogOpen]);
    const closeAnotherUserDialog = useCallback(() => { setAnotherUserDialogOpen(false); }, [setAnotherUserDialogOpen]);
    const handleOwnerSelect = useCallback((owner: OwnerShape) => {
        const ownerId = versionField?.value?.id ?? rootField?.value?.id;
        if (owner?.id === ownerId) return;
        exists(versionHelpers) && versionHelpers.setValue(owner);
        exists(rootHelpers) && rootHelpers.setValue(owner);
    }, [versionField?.value?.id, rootField?.value?.id, versionHelpers, rootHelpers]);

    // Owner list dialog (select self, organization, or another user)
    const [ownerDialogAnchor, setOwnerDialogAnchor] = useState<any>(null);
    const handleOwnerClick = useCallback((ev: React.MouseEvent<any>) => {
        if (!isAvailable) return;
        ev.stopPropagation();
        const owner = versionField?.value ?? rootField?.value;
        // If not editing, navigate to owner
        if (!isEditing) {
            if (owner) openObject(owner, setLocation);
        }
        // Otherwise, open dialog
        else setOwnerDialogAnchor(ev.currentTarget)
    }, [isEditing, isAvailable, versionField?.value, rootField?.value, setLocation]);
    const closeOwnerDialog = useCallback(() => setOwnerDialogAnchor(null), []);
    const handleOwnerDialogSelect = useCallback((ownerType: OwnerTypesEnum) => {
        if (ownerType === OwnerTypesEnum.Organization) {
            openOrganizationDialog();
        } else if (ownerType === OwnerTypesEnum.AnotherUser) {
            openAnotherUserDialog();
        } else {
            const owner = session ? userFromSession(session) : undefined
            exists(versionHelpers) && versionHelpers.setValue(owner);
            exists(rootHelpers) && rootHelpers.setValue(owner);
        }
        closeOwnerDialog();
    }, [closeOwnerDialog, openOrganizationDialog, openAnotherUserDialog, session, versionHelpers, rootHelpers]);

    // SelectOrCreateDialog
    const [selectOrCreateType, selectOrCreateHandleAdd, selectOrCreateHandleClose] = useMemo<[SelectOrCreateObjectType | null, (item: any) => any, () => void]>(() => {
        if (isOrganizationDialogOpen) return ['Organization', handleOwnerSelect, closeOrganizationDialog];
        else if (isAnotherUserDialogOpen) return ['User', handleOwnerSelect, closeAnotherUserDialog];
        return [null, () => { }, () => { }];
    }, [isOrganizationDialogOpen, handleOwnerSelect, closeOrganizationDialog, isAnotherUserDialogOpen, closeAnotherUserDialog]);

    const { Icon, tooltip } = useMemo(() => {
        const owner = versionField?.value ?? rootField?.value;
        // If no owner data, marked as anonymous
        if (!owner) return {
            Icon: null,
            tooltip: `Marked as anonymous${isEditing ? '' : '. Press to set owner'}`
        };
        // If owner is organization, use organization icon
        if (owner.__typename === 'Organization') {
            const Icon = OrganizationIcon;
            const ownerName = firstString(getTranslation(owner as RelationshipItemOrganization, languages, true).name, 'organization');
            return {
                Icon,
                tooltip: `Owner: ${ownerName}`
            };
        }
        // If owner is user, use self icon
        const Icon = UserIcon;
        const isSelf = owner.id === getCurrentUser(session).id;
        const ownerName = (owner as RelationshipItemUser).name;
        return {
            Icon,
            tooltip: `Owner: ${isSelf ? 'Self' : ownerName}`
        };
    }, [isEditing, languages, rootField?.value, session, versionField?.value]);

    // If not available, return null
    if (!isAvailable || (!isEditing && !Icon)) return null;
    // Return button with label on top
    return (
        <>
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
            {/* Popup for selecting organization or user */}
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
                <TextShrink id="owner" sx={{ ...commonLabelProps() }}>Owner</TextShrink>
                <Tooltip title={tooltip}>
                    <ColorIconButton
                        background={palette.primary.light}
                        sx={{ ...commonButtonProps(isEditing, true) }}
                        onClick={handleOwnerClick}
                    >
                        {Icon && <Icon {...commonIconProps()} />}
                    </ColorIconButton>
                </Tooltip>
            </Stack>
        </>
    )
}