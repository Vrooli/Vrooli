import { useLazyQuery, useQuery } from '@apollo/client';
import { APP_LINKS, ProjectOrRoutineSortBy, ResourceListUsedFor } from '@shared/consts';
import { Box, Stack } from '@mui/material';
import { ListMenu, ListTitleContainer, PageTitle, ResourceListHorizontal } from 'components';
import { developPage } from 'graphql/generated/developPage';
import { profile } from 'graphql/generated/profile';
import { developPageQuery, profileQuery } from 'graphql/query';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { ResourceList } from 'types';
import { DevelopSearchPageTabOption, listToListItems, openObject, OpenObjectProps, stringifySearchParams } from 'utils';
import { useLocation } from '@shared/route';
import { DevelopPageProps } from '../types';
import { ListMenuItemData } from 'components/dialogs/types';

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

const createPopupOptions: ListMenuItemData<string>[] = [
    { label: 'Project', value: `${APP_LINKS.Project}/add` },
    { label: 'Routine (Single Step)', value: `${APP_LINKS.Routine}/add` },
    { label: 'Routine (Multi Step)', value: `${APP_LINKS.Routine}/add?build=true` },
]

export const DevelopPage = ({
    session
}: DevelopPageProps) => {
    const [, setLocation] = useLocation();
    const [getProfile, { data: profileData, loading: resourcesLoading }] = useLazyQuery<profile>(profileQuery, { errorPolicy: 'all' });
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

    const { data: developPageData, loading: developPageLoading } = useQuery<developPage>(developPageQuery, { errorPolicy: 'all' });

    /**
     * Opens page for list item
     */
    const toItemPage = useCallback((item: OpenObjectProps['object'], event: any) => {
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

    const toSeeAllInProgress = useCallback((event: any) => {
        event?.stopPropagation();
        setLocation(`${APP_LINKS.DevelopSearch}${stringifySearchParams({
            isComplete: false,
            type: DevelopSearchPageTabOption.InProgress,
            sort: ProjectOrRoutineSortBy.DateUpdatedDesc,
        })}`);
    }, [setLocation]);

    const toSeeAllRecent = useCallback((event: any) => {
        event?.stopPropagation();
        setLocation(`${APP_LINKS.DevelopSearch}${stringifySearchParams({
            type: DevelopSearchPageTabOption.Recent,
            sort: ProjectOrRoutineSortBy.DateUpdatedDesc,
        })}`);
    }, [setLocation]);

    const toSeeAllCompleted = useCallback((event: any) => {
        event?.stopPropagation();
        setLocation(`${APP_LINKS.DevelopSearch}${stringifySearchParams({
            isComplete: true,
            type: DevelopSearchPageTabOption.Completed,
            sort: ProjectOrRoutineSortBy.DateUpdatedDesc,
        })}`);
    }, [setLocation]);

    // Dialog for selecting whether to create a new project or routine
    const [createAnchor, setCreateAnchor] = useState<any>(null);
    const openCreateSelect = useCallback((ev: React.MouseEvent<any>) => {
        ev.stopPropagation();
        setCreateAnchor(ev.currentTarget)
    }, [setCreateAnchor]);
    const closeCreateSelect = useCallback(() => setCreateAnchor(null), []);
    const handleAdvancedSearchSelect = useCallback((path: string) => {
        setLocation(path);
    }, [setLocation]);

    return (
        <Box id='page' sx={{
            padding: '0.5em',
            paddingTop: { xs: '64px', md: '80px' },
            width: 'min(100%, 700px)',
            margin: 'auto',
        }}>
            {/* Create new dialog */}
            <ListMenu
                id={`create-project-or-routine-menu`}
                anchorEl={createAnchor}
                title='Select Object Type'
                data={createPopupOptions}
                onSelect={handleAdvancedSearchSelect}
                onClose={closeCreateSelect}
                zIndex={200}
            />
            <PageTitle title="Develop Dashboard" helpText={developPageText} />
            <Stack direction="column" spacing={10}>
                {/* TODO my organizations list */}
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
                <ListTitleContainer
                    title={"In Progress"}
                    helpText={inProgressText}
                    isEmpty={inProgress.length === 0}
                    onClick={toSeeAllInProgress}
                    options={[['Create', openCreateSelect], ['See all', toSeeAllInProgress]]}
                >
                    {inProgress}
                </ListTitleContainer>
                <ListTitleContainer
                    title={"Recent"}
                    helpText={recentText}
                    isEmpty={recent.length === 0}
                    onClick={toSeeAllRecent}
                    options={[['See all', toSeeAllRecent]]}
                >
                    {recent}
                </ListTitleContainer>
                <ListTitleContainer
                    title={"Completed"}
                    helpText={completedText}
                    isEmpty={completed.length === 0}
                    onClick={toSeeAllCompleted}
                    options={[['See all', toSeeAllCompleted]]}
                >
                    {completed}
                </ListTitleContainer>
            </Stack>
        </Box>
    )
}