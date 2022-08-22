import { useLazyQuery, useQuery } from '@apollo/client';
import { APP_LINKS, ResourceListUsedFor, ResourceUsedFor } from '@shared/consts';
import { Box, Stack, Typography } from '@mui/material';
import { HelpButton, ResourceListHorizontal, ListTitleContainer } from 'components';
import { profile } from 'graphql/generated/profile';
import { researchPage } from 'graphql/generated/researchPage';
import { profileQuery, researchPageQuery } from 'graphql/query';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { ResourceList } from 'types';
import { listToListItems, openObject, OpenObjectProps, stringifySearchParams, SearchPageTabOption } from 'utils';
import { useLocation } from '@shared/route';
import { ResearchPageProps } from '../types';

const donateOrInvestText =
    ``

const joinATeamText =
    `Organizations listed here have indicated that they are open to new members. Check them out and contact them if you have anything to contribute!`

const newlyCompletedText =
    `Projects listed here have been recently completed, which means their main routine(s) are ready to run.`

const processesText =
    ``

const researchPageText =
    `The **Research Dashboard** is designed to help you validate ideas and discover new projects and routines.

Currently, the page is bare-bones. It contains sections for:  
- Projects looking for funding/investments
- Organizations looking for new members
- Projects recently completed
- Routines to help you validate ideas or perform general research
- Projects looking for votes

The top of this page also contains a list of resources, which you can update with your favorite research-related links. 
If you are not logged in, default resources will be displayed.`

const voteText =
    `Projects listed here are requesting your vote on Project Catalyst! Click on their proposal resources to learn more.`

export const ResearchPage = ({
    session
}: ResearchPageProps) => {
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

    const { data: researchPageData, loading: researchPageLoading } = useQuery<researchPage>(researchPageQuery, { errorPolicy: 'all' });

    /**
     * Opens page for list item
     */
    const toItemPage = useCallback((item: OpenObjectProps['object'], event: any) => {
        event?.stopPropagation();
        // Navigate to item page
        openObject(item, setLocation);
    }, [setLocation]);

    const processes = useMemo(() => listToListItems({
        dummyItems: new Array(5).fill('Routine'),
        items: researchPageData?.researchPage?.processes,
        keyPrefix: 'research-process-list-item',
        loading: researchPageLoading,
        onClick: toItemPage,
        session,
    }), [researchPageData?.researchPage?.processes, researchPageLoading, session, toItemPage])

    const newlyCompleted = useMemo(() => listToListItems({
        dummyItems: new Array(5).fill('Project'),
        items: researchPageData?.researchPage?.newlyCompleted,
        keyPrefix: 'newly-completed-list-item',
        loading: researchPageLoading,
        onClick: toItemPage,
        session,
    }), [researchPageData?.researchPage?.newlyCompleted, researchPageLoading, session, toItemPage])

    const needVotes = useMemo(() => listToListItems({
        dummyItems: new Array(5).fill('Project'),
        items: researchPageData?.researchPage?.needVotes,
        keyPrefix: 'need-votes-list-item',
        loading: researchPageLoading,
        onClick: toItemPage,
        session,
    }), [researchPageData?.researchPage?.needVotes, researchPageLoading, session, toItemPage])

    const needInvestments = useMemo(() => listToListItems({
        dummyItems: new Array(5).fill('Project'),
        items: researchPageData?.researchPage?.needInvestments,
        keyPrefix: 'need-investments-list-item',
        loading: researchPageLoading,
        onClick: toItemPage,
        session,
    }), [researchPageData?.researchPage?.needInvestments, researchPageLoading, session, toItemPage])

    const needMembers = useMemo(() => listToListItems({
        dummyItems: new Array(5).fill('Organization'),
        items: researchPageData?.researchPage?.needMembers,
        keyPrefix: 'need-members-list-item',
        loading: researchPageLoading,
        onClick: toItemPage,
        session,
    }), [researchPageData?.researchPage?.needMembers, researchPageLoading, session, toItemPage])

    const toCreateProcess = useCallback((event: any) => {
        event?.stopPropagation();
        setLocation(`${APP_LINKS.Routine}/add${stringifySearchParams({ tags: ['Research'] })}`);
    }, [setLocation]);

    const toCreateProject = useCallback((event: any) => {
        event?.stopPropagation();
        setLocation(`${APP_LINKS.Project}/add`);
    }, [setLocation]);

    const toCreateOrganization = useCallback((event: any) => {
        event?.stopPropagation();
        setLocation(`${APP_LINKS.Organization}/add`);
    }, [setLocation]);

    const toUpdateProfile = useCallback((event: any) => {
        event?.stopPropagation();
        setLocation(`${APP_LINKS.Settings}?page="profile"`);
    }, [setLocation]);

    const toSeeAllProcesses = useCallback((event: any) => {
        event?.stopPropagation();
        setLocation(`${APP_LINKS.Search}${stringifySearchParams({ 
            tags: ['Research'], 
            type: SearchPageTabOption.Routines,
        })}`);
    }, [setLocation]);

    const toSeeAllNewlyCompleted = useCallback((event: any) => {
        event?.stopPropagation();
        setLocation(`${APP_LINKS.Search}${stringifySearchParams({
            isComplete: true,
            sort: 'DateUpdatedDesc',
            type: SearchPageTabOption.Projects,
        })}`);
    }, [setLocation]);

    const toSeeAllJoinATeam = useCallback((event: any) => {
        event?.stopPropagation();
        setLocation(`${APP_LINKS.Search}${stringifySearchParams({
            isOpenToNewMembers: true,
            type: SearchPageTabOption.Organizations,
        })}`);
    }, [setLocation]);

    const toSeeAllNeedVotes = useCallback((event: any) => {
        event?.stopPropagation();
        setLocation(`${APP_LINKS.Search}${stringifySearchParams({
            resourceTypes: [ResourceUsedFor.Proposal], 
            type: SearchPageTabOption.Projects,
        })}`);
    }, [setLocation]);

    const toSeeAllNeedInvestments = useCallback((event: any) => {
        event?.stopPropagation();
        setLocation(`${APP_LINKS.Search}${stringifySearchParams({
            resourceTypes: [ResourceUsedFor.Donation],
            type: SearchPageTabOption.Projects,
        })}`);
    } , [setLocation]);

    return (
        <Box id='page' sx={{
            padding: '0.5em',
            paddingTop: { xs: '64px', md: '80px' },
            width: 'min(100%, 700px)',
            margin: 'auto',
        }}>
            {/* Title and help button */}
            <Stack
                direction="row"
                justifyContent="center"
                alignItems="center"
                sx={{ marginTop: 2, marginBottom: 2 }}
            >
                <Typography component="h1" variant="h3" sx={{ fontSize: { xs: '2rem', sm: '3rem' } }}>Research Dashboard</Typography>
                <HelpButton markdown={researchPageText} sx={{ width: '40px', height: '40px' }} />
            </Stack>
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
                <ListTitleContainer
                    title={"Processes"}
                    helpText={processesText}
                    isEmpty={processes.length === 0}
                    onClick={toSeeAllProcesses}
                    options={[['Create', toCreateProcess], ['See all', toSeeAllProcesses]]}
                >
                    {processes}
                </ListTitleContainer>
                <ListTitleContainer
                    title={"Newly Completed"}
                    helpText={newlyCompletedText}
                    isEmpty={newlyCompleted.length === 0}
                    onClick={toSeeAllNewlyCompleted}
                    options={[['Create', toCreateProject], ['See all', toSeeAllNewlyCompleted]]}
                >
                    {newlyCompleted}
                </ListTitleContainer>
                <ListTitleContainer
                    title={"Vote"}
                    helpText={voteText}
                    isEmpty={needVotes.length === 0}
                    onClick={toSeeAllNeedVotes}
                    options={[['Create', toCreateProject], ['See all', toSeeAllNeedVotes]]}
                >
                    {needVotes}
                </ListTitleContainer>
                <ListTitleContainer
                    title={"Donate or Invest"}
                    helpText={donateOrInvestText}
                    isEmpty={needInvestments.length === 0}
                    onClick={toSeeAllNeedInvestments}
                    options={[['Create', toCreateProject], ['See all', toSeeAllNeedInvestments]]}
                >
                    {needInvestments}
                </ListTitleContainer>
                <ListTitleContainer
                    title={"Join a Team"}
                    helpText={joinATeamText}
                    isEmpty={needMembers.length === 0}
                    onClick={toSeeAllJoinATeam}
                    options={[['Create', toCreateOrganization], ['Update profile', toUpdateProfile], ['See all', toSeeAllJoinATeam]]}
                >
                    {needMembers}
                </ListTitleContainer>
            </Stack>
        </Box>
    )
}