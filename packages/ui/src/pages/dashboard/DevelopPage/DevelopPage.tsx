import { useLazyQuery, useQuery } from '@apollo/client';
import { ResourceListUsedFor } from '@local/shared';
import { Box, Stack, Typography } from '@mui/material';
import { HelpButton, ListTitleContainer, ResourceListHorizontal } from 'components';
import { developPage } from 'graphql/generated/developPage';
import { profile } from 'graphql/generated/profile';
import { developPageQuery, profileQuery } from 'graphql/query';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { Organization, Project, ResourceList, Routine, Standard, User } from 'types';
import { listToListItems, openObject } from 'utils';
import { useLocation } from 'wouter';
import { DevelopPageProps } from '../types';

const completedText =
    `Find projects and routines that you've recently completed
`

const developPageText =
    `The **Develop Dashboard** is designed to help you implement new projects and routines.

Currently, the page is bare-bones. It contains sections for:  
- Projects and routines you've recently completed
- Projects and routines you're still working on
- Projects and routines you've recently updated

The top of this page also contains a list of resources, which you can update with your favorite work-related links. 
If you are not logged in, default resources will be displayed.`

const inProgressText =
    `Continue working on projects and routines you've recently started`

const recentText =
    `Recently updated projects and routines`

export const DevelopPage = ({
    session
}: DevelopPageProps) => {
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

    const { data: developPageData, loading: developPageLoading } = useQuery<developPage>(developPageQuery);

    /**
     * Opens page for list item
     */
    const toItemPage = useCallback((item: Organization | Project | Routine | Standard | User, event: any) => {
        event?.stopPropagation();
        // Navigate to item page
        openObject(item, setLocation);
    }, [setLocation]);

    const inProgress = useMemo(() => listToListItems({
        dummyItems: new Array(5).fill('Routine'),
        items: developPageData?.developPage?.inProgress,
        keyPrefix: 'in-progress-list-item',
        loading: developPageLoading,
        onClick: toItemPage,
        session,
    }), [developPageData?.developPage?.inProgress, developPageLoading, session, toItemPage])

    const recent = useMemo(() => listToListItems({
        dummyItems: new Array(5).fill('Routine'),
        items: developPageData?.developPage?.recent,
        keyPrefix: 'recent-list-item',
        loading: developPageLoading,
        onClick: toItemPage,
        session,
    }), [developPageData?.developPage?.recent, developPageLoading, session, toItemPage])

    const completed = useMemo(() => listToListItems({
        dummyItems: new Array(5).fill('Routine'),
        items: developPageData?.developPage?.completed,
        keyPrefix: 'completed-list-item',
        loading: developPageLoading,
        onClick: toItemPage,
        session,
    }), [developPageData?.developPage?.completed, developPageLoading, session, toItemPage])

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
                <Typography component="h1" variant="h3" sx={{ fontSize: { xs: '2rem', sm: '3rem' } }}>Develop Dashboard</Typography>
                <HelpButton markdown={developPageText} sx={{ width: '40px', height: '40px' }} />
            </Stack>
            <Stack direction="column" spacing={10}>
                {/* TODO my organizations list */}
                {/* Resources */}
                <ResourceListHorizontal
                    title={Boolean(session?.id) ? '📌 Resources' : 'Example Resources'}
                    list={resourceList}
                    canEdit={Boolean(session?.id)}
                    handleUpdate={handleResourcesUpdate}
                    loading={resourcesLoading}
                    mutate={true}
                    session={session}
                />
                <ListTitleContainer
                    title={"In Progress"}
                    helpText={inProgressText}
                    isEmpty={inProgress.length === 0}
                    onClick={() => { }}
                    options={[['Create', () => { }], ['See all', () => { }]]}
                >
                    {inProgress}
                </ListTitleContainer>
                <ListTitleContainer
                    title={"Recent"}
                    helpText={recentText}
                    isEmpty={recent.length === 0}
                    onClick={() => { }}
                    options={[['See all', () => { }]]}
                >
                    {recent}
                </ListTitleContainer>
                <ListTitleContainer
                    title={"Completed"}
                    helpText={completedText}
                    isEmpty={completed.length === 0}
                    onClick={() => { }}
                    options={[['See all', () => { }]]}
                >
                    {completed}
                </ListTitleContainer>
            </Stack>
        </Box>
    )
}