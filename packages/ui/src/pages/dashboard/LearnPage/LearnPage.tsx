import { useLazyQuery, useMutation, useQuery } from '@apollo/client';
import { Box, Stack, Typography } from '@mui/material';
import { HelpButton, ProjectListItem, ResourceListHorizontal, RoutineListItem, TitleContainer } from 'components';
import { ProjectSortBy, ResourceListUsedFor, ResourceUsedFor, RoutineSortBy } from 'graphql/generated/globalTypes';
import { profile } from 'graphql/generated/profile';
import { projects, projectsVariables } from 'graphql/generated/projects';
import { routines, routinesVariables } from 'graphql/generated/routines';
import { profileUpdateMutation } from 'graphql/mutation';
import { profileQuery, projectsQuery, routinesQuery } from 'graphql/query';
import { mutationWrapper } from 'graphql/utils/wrappers';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { Project, Resource, ResourceList, Routine } from 'types';
import { formatForUpdate, Pubs } from 'utils';
import { LearnPageProps } from '../types';
import courseMarkdown from './courseHelp.md';
import learnPageMarkdown from './learnPageHelp.md';
import tutorialMarkdown from './tutorialHelp.md';

/**
 * Default learning resources
 */
const defaultResourceList: ResourceList = {    
    usedFor: ResourceListUsedFor.Learn,
    resources: [
        {
            link: 'https://www.goodreads.com/review/list/138131443?shelf=must-reads-for-everyone',
            usedFor: ResourceUsedFor.Learning,
            translations: [
                { 
                    language: 'en',
                    title: 'Reading list',
                    description: 'Must-read books',
                }
            ]
        } as any,
        {
            link: 'https://www.notion.so/',
            usedFor: ResourceUsedFor.Notes,
            translations: [
                {
                    language: 'en',
                    title: 'Notion',
                    description: 'Jot down your thoughts and ideas',
                }
            ]
        } as any,
        {
            link: 'https://calendar.google.com/calendar/u/0/r',
            usedFor: ResourceUsedFor.Scheduling,
            translations: [
                {
                    language: 'en',
                    title: 'My Schedule',
                    description: 'Schedule study sessions',
                }
            ]
        } as any,
        {
            link: 'https://lifeat.io/',
            usedFor: ResourceUsedFor.Learning,
            translations: [
                {
                    language: 'en',
                    title: 'Study Session',
                    description: 'Start a study session on LifeAt',
                }
            ]
        } as any,
        {
            link: 'https://github.com/MattHalloran/Vrooli',
            usedFor: ResourceUsedFor.Learning,
            translations: [
                {
                    language: 'en',
                    title: 'TED Talks',
                    description: 'Find some inspiration by watching educational videos',
                }
            ]
        } as any,
        {
            link: 'https://www.duolingo.com/learn',
            usedFor: ResourceUsedFor.Learning,
            translations: [
                {
                    language: 'en',
                    title: 'Duolingo',
                    description: 'Learn a new language',
                }
            ]
        } as any,
        {
            link: 'https://www.khanacademy.org/profile/me/courses',
            usedFor: ResourceUsedFor.Learning,
            translations: [
                {
                    language: 'en',
                    title: 'Khan Academy',
                    description: 'Brush up on your math skills',
                }
            ]
        } as any
    ]
} as any;

export const LearnPage = ({
    session,
}: LearnPageProps) => {

    const [getProfile, { data: profileData, loading: resourcesLoading }] = useLazyQuery<profile>(profileQuery);
    useEffect(() => { if (session?.id) getProfile() }, [getProfile, session])
    
    const resourceList = useMemo(() => {
        if (!profileData?.profile?.resourceLists) return defaultResourceList;
        return profileData.profile.resourceLists.find(list => list.usedFor === ResourceListUsedFor.Learn) ?? null;
    }, [profileData]);
    const [updateResources] = useMutation<profile>(profileUpdateMutation);
    const handleResourcesUpdate = useCallback((updatedList: Resource[]) => {
        mutationWrapper({
            mutation: updateResources,
            input: formatForUpdate(profileData?.profile, {
                ...profileData?.profile,
                resourcesLearn: updatedList
            }, [], []),
            onError: (response) => {
                PubSub.publish(Pubs.Snack, { message: 'Error occurred.', severity: 'error', data: { error: response } });
            }
        })
    }, [updateResources]);

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
                <ResourceListHorizontal 
                    title={Boolean(session?.id) ? '📌 Resources' : 'Example Resources'}
                    list={resourceList}
                    canEdit={Boolean(session?.id)}
                    handleUpdate={handleResourcesUpdate}
                    session={session}
                />
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