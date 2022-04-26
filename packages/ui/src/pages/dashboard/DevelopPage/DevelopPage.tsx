import { useLazyQuery, useMutation, useQuery } from '@apollo/client';
import { ResourceListUsedFor } from '@local/shared';
import { Box, Stack, Typography } from '@mui/material';
import { HelpButton, ResourceListHorizontal, TitleContainer } from 'components';
import { developPage } from 'graphql/generated/developPage';
import { LogType } from 'graphql/generated/globalTypes';
import { logs, logsVariables } from 'graphql/generated/logs';
import { profile } from 'graphql/generated/profile';
import { profileUpdateMutation } from 'graphql/mutation';
import { developPageQuery, logsQuery, profileQuery } from 'graphql/query';
import { useCallback, useEffect, useMemo } from 'react';
import { ResourceList } from 'types';
import { listToListItems } from 'utils';
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

/**
 * Default develop resources TODO
 */
const defaultResourceList: ResourceList = {
    usedFor: ResourceListUsedFor.Develop,
    resources: [
    ]
} as any;

export const DevelopPage = ({
    session
}: DevelopPageProps) => {

    const [getProfile, { data: profileData, loading: resourcesLoading }] = useLazyQuery<profile>(profileQuery);
    useEffect(() => { if (session?.id) getProfile() }, [getProfile, session])

    const resourceList = useMemo(() => {
        if (!profileData?.profile?.resourceLists) return defaultResourceList;
        return profileData.profile.resourceLists.find(list => list.usedFor === ResourceListUsedFor.Research) ?? null;
    }, [profileData]);
    const [updateResources] = useMutation<profile>(profileUpdateMutation);
    const handleResourcesUpdate = useCallback((updatedList: ResourceList) => {
        getProfile();
    }, [updateResources]);

    const { data: developPageData, loading: developPageLoading } = useQuery<developPage>(developPageQuery);

    const inProgress = useMemo(() => listToListItems(
        developPageData?.developPage?.inProgress ?? [],
        session,
        'in-progress-list-item',
        () => {}
    ), [developPageData, session])

    const recent = useMemo(() => listToListItems(
        developPageData?.developPage?.recent ?? [],
        session,
        'recently-updated-list-item',
        () => {}
    ), [developPageData, session])

    const completed = useMemo(() => listToListItems(
        developPageData?.developPage?.completed ?? [],
        session,
        'completed-list-item',
        () => {}
    ), [developPageData, session])

    return (
        <Box id="page">
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
                    mutate={true}
                    session={session}
                />
                <TitleContainer
                    title={"In Progress"}
                    helpText={inProgressText}
                    loading={developPageLoading}
                    onClick={() => { }}
                    options={[['Create', () => { }], ['See all', () => { }]]}
                >
                    {inProgress}
                </TitleContainer>
                <TitleContainer
                    title={"Recent"}
                    helpText={recentText}
                    loading={developPageLoading}
                    onClick={() => { }}
                    options={[['See all', () => { }]]}
                >
                    {recent}
                </TitleContainer>
                <TitleContainer
                    title={"Completed"}
                    helpText={completedText}
                    loading={developPageLoading}
                    onClick={() => { }}
                    options={[['See all', () => { }]]}
                >
                    {completed}
                </TitleContainer>
            </Stack>
        </Box>
    )
}