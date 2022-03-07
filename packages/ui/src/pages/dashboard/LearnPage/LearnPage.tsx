import { useQuery } from '@apollo/client';
import { Box, Stack, Typography } from '@mui/material';
import { HelpButton, ProjectListItem, ResourceListHorizontal, RoutineListItem, TitleContainer } from 'components';
import { ProjectSortBy, RoutineSortBy } from 'graphql/generated/globalTypes';
import { projects, projectsVariables } from 'graphql/generated/projects';
import { routines, routinesVariables } from 'graphql/generated/routines';
import { projectsQuery, routinesQuery } from 'graphql/query';
import { useEffect, useMemo, useState } from 'react';
import { Project, Routine } from 'types';
import { LearnPageProps } from '../types';
import courseMarkdown from './courseHelp.md';
import learnPageMarkdown from './learnPageHelp.md';
import tutorialMarkdown from './tutorialHelp.md';

export const LearnPage = ({
    session,
}: LearnPageProps) => {

    const { data: coursesData, loading: coursesLoading } = useQuery<projects, projectsVariables>(projectsQuery, { 
        variables: { input: { 
            take: 5,
            sortBy: ProjectSortBy.VotesDesc,
            tags: ['learn'],
        } } 
    });
    const { data: tutorialsData, loading: tutorialsLoading } = useQuery<routines, routinesVariables>(routinesQuery, { 
        variables: { input: { 
            take: 5,
            sortBy: RoutineSortBy.VotesDesc,
            tags: ['learn'],
        } } 
    });

    const courses = useMemo(() => coursesData?.projects?.edges?.map((o, index) => (
        <ProjectListItem
            key={`course-list-item-${index}`}
            index={index}
            session={session}
            data={o.node as Project}
            onClick={() => { }}
        />
    )) ?? [], [coursesData]);

    const tutorials = useMemo(() => tutorialsData?.routines?.edges?.map((o, index) => (
        <RoutineListItem
            key={`tutorial-list-item-${index}`}
            index={index}
            session={session}
            data={o.node as Routine}
            onClick={() => { }}
        />
    )) ?? [], [tutorialsData]);

    // Parse help button markdown
    const [courseText, setCourseText] = useState('');
    const [learnPageText, setLearnPageText] = useState('');
    const [tutorialText, setTutorialText] = useState('');   
    useEffect(() => {
        fetch(courseMarkdown).then((r) => r.text()).then((text) => { setCourseText(text) });
        fetch(learnPageMarkdown).then((r) => r.text()).then((text) => { setLearnPageText(text) });
        fetch(tutorialMarkdown).then((r) => r.text()).then((text) => { setTutorialText(text) });
    }, []);

    return (
        <Box id="page">
            {/* Title and help button */}
            <Stack 
                direction="row" 
                justifyContent="center" 
                alignItems="center" 
                sx={{ marginTop: 2, marginBottom: 2}}
            >
                <Typography component="h1" variant="h3">Learn Dashboard</Typography>
                <HelpButton markdown={learnPageText} sx={{width: '40px', height: '40px'}} />
            </Stack>
            <Stack direction="column" spacing={10}>
                {/* Resources */}
                <ResourceListHorizontal />
                {/* Available courses */}
                <TitleContainer
                    title={"Courses"}
                    helpText={courseText}
                    loading={coursesLoading}
                    onClick={() => { }}
                    options={[['Create', () => { }], ['See all', () => { }]]}
                >
                    {courses}
                </TitleContainer>
                {/* Available tutorials */}
                <TitleContainer
                    title={"Tutorials"}
                    helpText={tutorialText}
                    loading={tutorialsLoading}
                    onClick={() => { }}
                    options={[['Create', () => { }], ['See all', () => { }]]}
                >
                    {tutorials}
                </TitleContainer>
            </Stack>
        </Box>
    )
}