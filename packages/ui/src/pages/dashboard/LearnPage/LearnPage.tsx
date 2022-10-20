import { useLazyQuery, useQuery } from '@apollo/client';
import { APP_LINKS } from '@shared/consts';
import { Stack } from '@mui/material';
import { ResourceListHorizontal, ListTitleContainer, PageTitle, PageContainer } from 'components';
import { ResourceListUsedFor } from 'graphql/generated/globalTypes';
import { learnPage } from 'graphql/generated/learnPage';
import { profile } from 'graphql/generated/profile';
import { learnPageQuery, profileQuery } from 'graphql/query';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { ResourceList } from 'types';
import { listToListItems, SearchPageTabOption, stringifySearchParams } from 'utils';
import { useLocation } from '@shared/route';
import { LearnPageProps } from '../types';
import { getCurrentUser } from 'utils/authentication';

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

const zIndex = 200;

export const LearnPage = ({
    session,
}: LearnPageProps) => {
    const [, setLocation] = useLocation();
    const [getProfile, { data: profileData, loading: resourcesLoading }] = useLazyQuery<profile>(profileQuery, { errorPolicy: 'all' });
    useEffect(() => { if (getCurrentUser(session).id) getProfile() }, [getProfile, session])
    const [resourceList, setResourceList] = useState<ResourceList | null>(null);
    useEffect(() => {
        if (!profileData?.profile?.resourceLists) return;
        const list = profileData.profile.resourceLists.find(list => list.usedFor === ResourceListUsedFor.Learn) ?? null;
        setResourceList(list);
    }, [profileData]);
    const handleResourcesUpdate = useCallback((updatedList: ResourceList) => {
        setResourceList(updatedList);
    }, []);

    const { data: learnPageData, loading: learnPageLoading } = useQuery<learnPage>(learnPageQuery, { errorPolicy: 'all' });

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
        setLocation(`${APP_LINKS.Search}${stringifySearchParams({
            tags: ['Learn'],
            type: SearchPageTabOption.Projects,
        })}`);
    }, [setLocation]);

    /**
     * Navigates to "Routine Search" page, with "Learn" tag as default
     */
    const toSeeAllTutorials = useCallback((event: any) => {
        event?.stopPropagation();
        setLocation(`${APP_LINKS.Search}${stringifySearchParams({
            tags: ['Learn'],
            type: SearchPageTabOption.Routines,
        })}`);
    }, [setLocation]);

    const courses = useMemo(() => listToListItems({
        dummyItems: new Array(5).fill('Project'),
        items: learnPageData?.learnPage?.courses,
        keyPrefix: 'course-list-item',
        loading: learnPageLoading,
        session,
        zIndex,
    }), [learnPageData?.learnPage?.courses, learnPageLoading, session])

    const tutorials = useMemo(() => listToListItems({
        dummyItems: new Array(5).fill('Routine'),
        items: learnPageData?.learnPage?.tutorials,
        keyPrefix: 'tutorial-list-item',
        loading: learnPageLoading,
        session,
        zIndex,
    }), [learnPageData?.learnPage?.tutorials, learnPageLoading, session])

    return (
        <PageContainer>
            <PageTitle title='Learn Dashboard' helpText={learnPageText} />
            <Stack direction="column" spacing={10}>
                {/* Resources */}
                {session?.isLoggedIn && <ResourceListHorizontal
                    title={'📌 Resources'}
                    list={resourceList}
                    canEdit={true}
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
        </PageContainer>
    )
}