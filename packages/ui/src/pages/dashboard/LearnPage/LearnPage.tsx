import { useLazyQuery, useQuery } from '@apollo/client';
import { APP_LINKS } from '@local/shared';
import { Box, Stack, Typography } from '@mui/material';
import { HelpButton, ResourceListHorizontal, ListTitleContainer } from 'components';
import { ResourceListUsedFor } from 'graphql/generated/globalTypes';
import { learnPage } from 'graphql/generated/learnPage';
import { profile } from 'graphql/generated/profile';
import { learnPageQuery, profileQuery } from 'graphql/query';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { Project, ResourceList, Routine } from 'types';
import { listToListItems, openObject, stringifySearchParams } from 'utils';
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
    const [, setLocation] = useLocation();
    const [getProfile, { data: profileData, loading: resourcesLoading }] = useLazyQuery<profile>(profileQuery);
    useEffect(() => { if (session?.id) getProfile() }, [getProfile, session])
    const [resourceList, setResourceList] = useState<ResourceList | null>(null);
    useEffect(() => {
        if (!profileData?.profile?.resourceLists) return;
        const list = profileData.profile.resourceLists.find(list => list.usedFor === ResourceListUsedFor.Learn) ?? null;
        setResourceList(list);
    }, [profileData]);
    const handleResourcesUpdate = useCallback((updatedList: ResourceList) => {
        setResourceList(updatedList);
    }, []);

    const { data: learnPageData, loading: learnPageLoading } = useQuery<learnPage>(learnPageQuery);

    /**
     * Opens page for list item
     */
     const toItemPage = useCallback((item: Project | Routine, event: any) => {
        event?.stopPropagation();
        // Navigate to item page
        openObject(item, setLocation);
    }, [setLocation]);

    /**
     * Navigates to "New Project" page, with "Learn" tag as default
     */
    const toCreateCourse = useCallback((event: any) => {
        event?.stopPropagation();
        setLocation(`${APP_LINKS.Project}/add${stringifySearchParams({ tags: ['Learn'] })}`);
    }, [setLocation]);

    /**
     * Navigates to "New Routine" page, with "Learn" tag as default
     */
    const toCreateTutorial = useCallback((event: any) => {
        event?.stopPropagation();
        setLocation(`${APP_LINKS.Routine}/add${stringifySearchParams({ tags: ['Learn'] })}`);
    }, [setLocation]);

    /**
     * Navigates to "Project Search" page, with "Learn" tag as default
     */
    const toSeeAllCourses = useCallback((event: any) => {
        event?.stopPropagation();
        setLocation(`${APP_LINKS.SearchProjects}${stringifySearchParams({ tags: ['Learn'] })}`);
    }, [setLocation]);

    /**
     * Navigates to "Routine Search" page, with "Learn" tag as default
     */
    const toSeeAllTutorials = useCallback((event: any) => {
        event?.stopPropagation();
        setLocation(`${APP_LINKS.SearchRoutines}${stringifySearchParams({ tags: ['Learn'] })}`);
    }, [setLocation]);

    const courses = useMemo(() => listToListItems({
        dummyItems: new Array(5).fill('Project'),
        items: learnPageData?.learnPage?.courses,
        keyPrefix: 'course-list-item',
        loading: learnPageLoading,
        onClick: toItemPage,
        session,
    }), [learnPageData?.learnPage?.courses, learnPageLoading, session, toItemPage])

    const tutorials = useMemo(() => listToListItems({
        dummyItems: new Array(5).fill('Routine'),
        items: learnPageData?.learnPage?.tutorials,
        keyPrefix: 'tutorial-list-item',
        loading: learnPageLoading,
        onClick: toItemPage,
        session,
    }), [learnPageData?.learnPage?.tutorials, learnPageLoading, session, toItemPage])

    return (
        <Box id='page' sx={{
            padding: '0.5em',
            paddingTop: { xs: '64px', md: '80px' },
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
                    loading={resourcesLoading}
                    mutate={true}
                    session={session}
                    zIndex={1}
                />}
                {/* Available courses */}
                <ListTitleContainer
                    title={"Courses"}
                    helpText={courseText}
                    isEmpty={courses.length === 0}
                    onClick={toSeeAllCourses}
                    options={[['Create', toCreateCourse], ['See all', toSeeAllCourses]]}
                >
                    {courses}
                </ListTitleContainer>
                {/* Available tutorials */}
                <ListTitleContainer
                    title={"Tutorials"}
                    helpText={tutorialText}
                    isEmpty={tutorials.length === 0}
                    onClick={toSeeAllTutorials}
                    options={[['Create', toCreateTutorial], ['See all', toSeeAllTutorials]]}
                >
                    {tutorials}
                </ListTitleContainer>
            </Stack>
        </Box>
    )
}