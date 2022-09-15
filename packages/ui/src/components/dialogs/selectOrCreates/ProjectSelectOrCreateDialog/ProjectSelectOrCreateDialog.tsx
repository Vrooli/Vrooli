import {
    Dialog,
    DialogContent,
    IconButton,
    Stack,
    Tooltip,
    Typography,
    useTheme
} from '@mui/material';
import { BaseObjectDialog, DialogTitle } from 'components';
import { useCallback, useEffect, useRef, useState } from 'react';
import { ProjectSelectOrCreateDialogProps } from '../types';
import { Project } from 'types';
import { SearchList } from 'components/lists';
import { projectQuery } from 'graphql/query';
import { useLazyQuery } from '@apollo/client';
import { project, projectVariables } from 'graphql/generated/project';
import { ProjectCreate } from 'components/views/Project/ProjectCreate/ProjectCreate';
import { SearchType, projectSearchSchema, removeSearchParams } from 'utils';
import { useLocation } from '@shared/route';
import { AddIcon } from '@shared/icons';

const helpText =
    `This dialog allows you to connect a new or existing project to an object.

If you do not own the project, the owner will have to approve of the request.`

const titleAria = "select-or-create-project-dialog-title"

export const ProjectSelectOrCreateDialog = ({
    handleAdd,
    handleClose,
    isOpen,
    session,
    zIndex,
}: ProjectSelectOrCreateDialogProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();

    /**
     * Before closing, remove all URL search params for advanced search
     */
    const onClose = useCallback(() => {
        // Clear search params
        removeSearchParams(setLocation, [
            ...projectSearchSchema.fields.map(f => f.fieldName),
            'advanced',
            'sort',
            'time',
        ]);
        handleClose();
    }, [handleClose, setLocation]);

    // Create new project dialog
    const [isCreateOpen, setIsCreateOpen] = useState(false);
    const handleCreateOpen = useCallback(() => { setIsCreateOpen(true); }, [setIsCreateOpen]);
    const handleCreated = useCallback((project: Project) => {
        setIsCreateOpen(false);
        handleAdd(project);
        onClose();
    }, [handleAdd, onClose]);
    const handleCreateClose = useCallback(() => {
        setIsCreateOpen(false);
    }, [setIsCreateOpen]);

    // If project selected from search, query for full data
    const [getProject, { data: projectData }] = useLazyQuery<project, projectVariables>(projectQuery);
    const queryingRef = useRef(false);
    const handleProjectSelect = useCallback((project: Project) => {
        // Query for full project data, if not already known (would be known if the same project was selected last time)
        if (projectData?.project?.id === project.id) {
            handleAdd(projectData?.project);
            onClose();
        } else {
            queryingRef.current = true;
            getProject({ variables: { input: { id: project.id } } });
        }
    }, [getProject, projectData, handleAdd, onClose]);
    useEffect(() => {
        if (projectData?.project && queryingRef.current) {
            handleAdd(projectData.project);
            onClose();
        }
        queryingRef.current = false;
    }, [handleAdd, onClose, handleCreateClose, projectData]);

    return (
        <Dialog
            open={isOpen}
            onClose={onClose}
            scroll="body"
            aria-labelledby={titleAria}
            sx={{
                zIndex,
                '& .MuiDialogContent-root': { overflow: 'visible', background: palette.background.default },
                '& .MuiDialog-paper': { overflow: 'visible' }
            }}
        >
            {/* Popup for creating a new project */}
            <BaseObjectDialog
                onAction={handleCreateClose}
                open={isCreateOpen}
                zIndex={zIndex + 1}
            >
                <ProjectCreate
                    onCreated={handleCreated}
                    onCancel={handleCreateClose}
                    session={session}
                    zIndex={zIndex + 1}
                />
            </BaseObjectDialog>
            <DialogTitle
                ariaLabel={titleAria}
                title={'Add Project'}
                helpText={helpText}
                onClose={onClose}
            />
            <DialogContent>
                <Stack direction="column" spacing={2}>
                    <Stack direction="row" alignItems="center" justifyContent="center">
                        <Typography component="h2" variant="h4">Projects</Typography>
                        <Tooltip title="Add new" placement="top">
                            <IconButton
                                size="medium"
                                onClick={handleCreateOpen}
                                sx={{ padding: 1 }}
                            >
                                <AddIcon fill={palette.secondary.main} width='1.5em' height='1.5em' />
                            </IconButton>
                        </Tooltip>
                    </Stack>
                    <SearchList
                        id="project-select-or-create-list"
                        itemKeyPrefix='project-list-item'
                        noResultsText={"None found. Maybe you should create one?"}
                        onObjectSelect={(newValue) => handleProjectSelect(newValue)}
                        searchType={SearchType.Project}
                        searchPlaceholder={'Select existing projects...'}
                        session={session}
                        take={20}
                        where={{ userId: session?.id }}
                        zIndex={zIndex}
                    />
                </Stack>
            </DialogContent>
        </Dialog>
    )
}