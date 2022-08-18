/**
 * Horizontal button list for assigning owner, project, and parent 
 * to objects
 */
import { Box, IconButton, Stack, Typography, useTheme } from '@mui/material';
import { useCallback, useMemo, useState } from 'react';
import { RelationshipButtonsProps } from '../types';
import { noSelect } from 'styles';
import { getTranslation, getUserLanguages, ObjectType } from 'utils';
import { OrganizationSelectOrCreateDialog } from 'components/dialogs';
import {
    Apartment as OrganizationIcon,
    Person as SelfIcon
} from '@mui/icons-material';
import { Organization, Project, Routine } from 'types';

const grey = {
    400: '#BFC7CF',
    800: '#2F3A45',
};

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
    const handleOnwerClick = useCallback((ev: React.MouseEvent<any>) => {
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
    

    return (
        <Box sx={{
            position: 'relative',
            zIndex: zIndex,
            display: 'flex',
            flexDirection: 'row',
            alignItems: 'center',
            justifyContent: 'space-between',
            borderRadius: '12px',
            background: palette.background.paper,
        }}>
            {/* Owner button */}
            <Box
                width="50px"
                minWidth="50px"
                height="50px"
                borderRadius='100%'
                // bgcolor={profileColors[0]}
                justifyContent='center'
                alignItems='center'
                sx={{
                    display: 'flex',
                }}
            >
                <OrganizationIcon sx={{
                    // fill: profileColors[1],
                    width: '80%',
                    height: '80%',
                }} />
            </Box>
        </Box>
        // <>
        //     {/* Popup for adding/connecting a new organization */}
        //     <OrganizationSelectOrCreateDialog
        //         isOpen={isCreateDialogOpen}
        //         handleAdd={onChange}
        //         handleClose={closeCreateDialog}
        //         session={session}
        //         zIndex={zIndex + 1}
        //     />
        //     {/* Main component */}
        //     <Stack direction="row" spacing={1} justifyContent="center">
        //         <Typography variant="h6" sx={{ ...noSelect }}>For:</Typography>
        //         <Box component="span" sx={{
        //             display: 'inline-block',
        //             position: 'relative',
        //             width: '64px',
        //             height: '36px',
        //             padding: '8px',
        //         }}>
        //             {/* Track */}
        //             <Box component="span" sx={{
        //                 backgroundColor: palette.mode === 'dark' ? grey[800] : grey[400],
        //                 borderRadius: '16px',
        //                 width: '100%',
        //                 height: '65%',
        //                 display: 'block',
        //             }}>
        //                 {/* Thumb */}

        //                 <IconButton sx={{
        //                     backgroundColor: palette.secondary.main,
        //                     display: 'inline-flex',
        //                     width: '30px',
        //                     height: '30px',
        //                     position: 'absolute',
        //                     top: 0,
        //                     transition: 'transform 150ms cubic-bezier(0.4, 0, 0.2, 1)',
        //                     transform: `translateX(${Boolean(selected) ? '24' : '0'}px)`,
        //                 }}>
        //                     <Icon sx={{
        //                         position: 'absolute',
        //                         display: 'block',
        //                         fill: 'white',
        //                         borderRadius: '8px',
        //                     }} />
        //                 </IconButton>
        //             </Box>
        //             {/* Input */}
        //             <input
        //                 type="checkbox"
        //                 checked={Boolean(selected)}
        //                 disabled={disabled}
        //                 aria-label="user-organization-toggle"
        //                 onClick={handleClick}
        //                 style={{
        //                     position: 'absolute',
        //                     width: '100%',
        //                     height: '100%',
        //                     top: '0',
        //                     left: '0',
        //                     opacity: '0',
        //                     zIndex: '1',
        //                     margin: '0',
        //                     cursor: 'pointer',
        //                 }} />
        //         </Box >
        //         <Typography variant="h6" sx={{ ...noSelect }}>{Boolean(selected) ? getTranslation(selected, 'name', languages, true) : 'Self'}</Typography>
        //     </Stack>
        // </>
    )
}
