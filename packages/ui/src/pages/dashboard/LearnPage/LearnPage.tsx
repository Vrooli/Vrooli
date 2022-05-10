import { useLazyQuery, useMutation, useQuery } from '@apollo/client';
import { Box, Stack, Typography, useTheme } from '@mui/material';
import { HelpButton, ProjectListItem, ResourceListHorizontal, RoutineListItem, TitleContainer } from 'components';
import { ProjectSortBy, ResourceListUsedFor, ResourceUsedFor, RoutineSortBy } from 'graphql/generated/globalTypes';
import { learnPage } from 'graphql/generated/learnPage';
import { profile } from 'graphql/generated/profile';
import { projects, projectsVariables } from 'graphql/generated/projects';
import { routines, routinesVariables } from 'graphql/generated/routines';
import { profileUpdateMutation } from 'graphql/mutation';
import { learnPageQuery, profileQuery, projectsQuery, routinesQuery } from 'graphql/query';
import { useCallback, useEffect, useMemo } from 'react';
import { Project, ResourceList, Routine } from 'types';
import { listToListItems, openObject } from 'utils';
import { useLocation } from 'wouter';
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

export const LearnPage = ({
    session,
}: LearnPageProps) => {
    const { breakpoints } = useTheme();
    const [, setLocation] = useLocation();
    const [getProfile, { data: profileData, loading: resourcesLoading }] = useLazyQuery<profile>(profileQuery);
    useEffect(() => { if (session?.id) getProfile() }, [getProfile, session])

    const resourceList: ResourceList = useMemo(() => {
        return (profileData?.profile?.resourceLists?.find(list => list.usedFor === ResourceListUsedFor.Learn) ?? []) as ResourceList;
    }, [profileData]);
    const [updateResources] = useMutation<profile>(profileUpdateMutation);
    const handleResourcesUpdate = useCallback((updatedList: ResourceList) => {
        getProfile();
    }, [updateResources]);

    const { data: learnPageData, loading: learnPageLoading } = useQuery<learnPage>(learnPageQuery);

    /**
     * Opens page for list item
     */
     const toItemPage = useCallback((item: Project | Routine, event: any) => {
        event?.stopPropagation();
        // Navigate to item page
        openObject(item, setLocation);
    }, [setLocation]);

    const courses = useMemo(() => listToListItems({
        dummyItems: new Array(5).fill('Project'),
        items: learnPageData?.learnPage?.courses,
        keyPrefix: 'course-list-item',
        loading: learnPageLoading,
        onClick: toItemPage,
        session,
    }), [learnPageData, session])

    const tutorials = useMemo(() => listToListItems({
        dummyItems: new Array(5).fill('Routine'),
        items: learnPageData?.learnPage?.tutorials,
        keyPrefix: 'tutorial-list-item',
        loading: learnPageLoading,
        onClick: toItemPage,
        session,
    }), [learnPageData, session])

    return (
        <Box id='page' sx={{
            [breakpoints.up('md')]: {
                paddingTop: '10vh',
            },
        }}>
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
                {session?.id && <ResourceListHorizontal
                    title={Boolean(session?.id) ? '📌 Resources' : 'Example Resources'}
                    list={resourceList}
                    canEdit={Boolean(session?.id)}
                    handleUpdate={handleResourcesUpdate}
                    mutate={true}
                    session={session}
                />}
                {/* Available courses */}
                <TitleContainer
                    title={"Courses"}
                    helpText={courseText}
                    loading={learnPageLoading}
                    onClick={() => { }}
                    options={[['Create', () => { }], ['See all', () => { }]]}
                >
                    {courses}
                </TitleContainer>
                {/* Available tutorials */}
                <TitleContainer
                    title={"Tutorials"}
                    helpText={tutorialText}
                    loading={learnPageLoading}
                    onClick={() => { }}
                    options={[['Create', () => { }], ['See all', () => { }]]}
                >
                    {tutorials}
                </TitleContainer>
            </Stack>
        </Box>
    )
}