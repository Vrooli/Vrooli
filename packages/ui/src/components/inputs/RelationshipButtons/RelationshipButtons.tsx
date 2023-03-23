/**
 * Horizontal button list for assigning owner, project, and parent 
 * to objects
 */
import { Stack, useTheme } from '@mui/material';
import { Session } from '@shared/consts';
import { IsCompleteButton, IsPrivateButton, OwnerButton, ParentButton, ProjectButton } from 'components/buttons/relationships';
import { noSelect } from 'styles';
import { getCurrentUser } from 'utils/authentication/session';
import { OwnerShape } from 'utils/shape/models/types';
import { RelationshipButtonsProps } from '../types';

/**
 * Converts session to user object
 * @param session Current user session
 * @returns User object
 */
export const userFromSession = (session: Session): Exclude<OwnerShape, null> => ({
    __typename: 'User',
    id: getCurrentUser(session).id as string,
    handle: null,
    name: 'Self',
})

export function RelationshipButtons(props: RelationshipButtonsProps) {
    const { palette } = useTheme();

    return (
        <Stack
            spacing={{ xs: 1, sm: 1.5, md: 2 }}
            direction="row"
            alignItems="center"
            justifyContent="center"
            p={1}
            sx={{
                borderRadius: '12px',
                background: palette.mode === 'dark' ? palette.background.paper : palette.background.default,
                overflowX: 'auto',
                zIndex: props.zIndex,
                ...noSelect,
                ...(props.sx ?? {}),
            }}
        >
            <OwnerButton {...props} />
            <ProjectButton {...props} />
            <ParentButton {...props} />
            <IsPrivateButton {...props} />
            <IsCompleteButton {...props} />
        </Stack>
    )
}