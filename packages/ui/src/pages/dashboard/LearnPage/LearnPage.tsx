import { useLazyQuery, useMutation, useQuery } from '@apollo/client';
import { Box, Stack, Typography } from '@mui/material';
import { HelpButton, ProjectListItem, ResourceListHorizontal, RoutineListItem, TitleContainer } from 'components';
import { ProjectSortBy, ResourceListUsedFor, ResourceUsedFor, RoutineSortBy } from 'graphql/generated/globalTypes';
import { profile } from 'graphql/generated/profile';
import { projects, projectsVariables } from 'graphql/generated/projects';
import { routines, routinesVariables } from 'graphql/generated/routines';
import { profileUpdateMutation } from 'graphql/mutation';
import { profileQuery, projectsQuery, routinesQuery } from 'graphql/query';
import { useCallback, useEffect, useMemo } from 'react';
import { Project, ResourceList, Routine } from 'types';
import { LearnPageProps } from '../types';

const courseText =
`Courses are community-created projects, each designed to teach a specific skill. Any project associated with the "learn" tag will be listed here.

In the long term, we would like to be able to generate digital certificates for completing courses. These certificates would be issued on [Atala Prism](https://atalaprism.io/app). The legitimacy of a certificate would be generated through a [web-of-trust](https://en.wikipedia.org/wiki/Web_of_trust).`

const learnPageText =
`The **Learn Dashboard** is designed to help you achieve your goals of self-actualization and self-improvement. 

Currently, the page is bare-bones. It contains a section to pin your learning resources, and lists of popular courses and tutorials.

If you are not logged in, default resources will be displayed.

In the future, we will add many new learning features, such as:  
- [Atala Prism](https://atalaprism.io/app) certifications to courses, so you can display your skills
- The ability to set time and skill-based goals for learning, with reminders to keep you on track`

const tutorialText =
`Tutorials are community-created routines, each designed to teach a specific skill. Any routine associated with the "learn" tag will be listed here.`

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
    const handleResourcesUpdate = useCallback((updatedList: ResourceList) => {
        getProfile();
    }, [updateResources]);

    const { data: coursesData, loading: coursesLoading } = useQuery<projects, projectsVariables>(projectsQuery, {
        variables: {
            input: {
                take: 5,
                sortBy: ProjectSortBy.VotesDesc,
                tags: ['learn'],
            }
        }
    });
    const { data: tutorialsData, loading: tutorialsLoading } = useQuery<routines, routinesVariables>(routinesQuery, {
        variables: {
            input: {
                take: 5,
                sortBy: RoutineSortBy.VotesDesc,
                tags: ['learn'],
            }
        }
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

    return (
        <Box id="page">
            {/* Title and help button */}
            <Stack
                direction="row"
                justifyContent="center"
                alignItems="center"
                sx={{ marginTop: 2, marginBottom: 2 }}
            >
                <Typography component="h1" variant="h3" sx={{ fontSize: { xs: '2rem', sm: '3rem' }}}>Learn Dashboard</Typography>
                <HelpButton markdown={learnPageText} sx={{ width: '40px', height: '40px' }} />
            </Stack>
            <Stack direction="column" spacing={10}>
                {/* Resources */}
                <ResourceListHorizontal
                    title={Boolean(session?.id) ? '📌 Resources' : 'Example Resources'}
                    list={resourceList}
                    canEdit={Boolean(session?.id)}
                    handleUpdate={handleResourcesUpdate}
                    mutate={true}
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