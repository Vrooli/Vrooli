import { Box, Button, Typography } from '@mui/material';
import { useQuery } from '@apollo/client';
import { projectsQuery } from 'graphql/query';
import { useCallback, useState } from 'react';
import { NewProjectDialog } from 'components';
import { projects, projectsVariables } from 'graphql/generated/projects';
import { Session } from 'types';
import { centeredText } from 'styles';

interface Props {
    session?: Session
}

export const ProjectsPage = ({
    session
}: Props) => {
    const { data: projects } = useQuery<projects, projectsVariables>(projectsQuery, { variables: { input: { userId: session?.id } } })
    // const [newProject] = useMutation<any>(asdf);
    // const [deleteProject] = useMutation<any>(asdf);
    const [newProjectOpen, setNewProjectOpen] = useState(false);
    const openNewProjectDialog = useCallback(() => setNewProjectOpen(true), []);
    const closeNewProjectDialog = useCallback(() => setNewProjectOpen(false), []);

    // const cards = useMemo(() => (
    //     projects?.projects?.edges?.map((edge, index) =>
    //         <ProjectCard
    //             key={index}
    //             data={edge.node}
    //         />)
    // ), [projects])

    return (
        <Box id="page">
            <NewProjectDialog open={newProjectOpen} onClose={closeNewProjectDialog} />
            <Box sx={{...centeredText}}>
                <Typography variant="h3" component="h1">My Projects</Typography>
                <Button onClick={openNewProjectDialog}>New Project</Button>
            </Box>
            <Box
                sx={{
                    display: 'grid',
                    gridTemplateColumns: 'repeat(auto-fill, minmax(300px, .5fr))',
                    gridGap: '20px',
                }}
            >
                {/* {cards} */}
            </Box>
        </Box>
    );
}