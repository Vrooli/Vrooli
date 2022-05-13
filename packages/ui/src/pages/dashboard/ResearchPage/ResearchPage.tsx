import { useLazyQuery, useMutation, useQuery } from '@apollo/client';
import { ResourceListUsedFor } from '@local/shared';
import { Box, Stack, Typography } from '@mui/material';
import { HelpButton, ResourceListHorizontal, TitleContainer } from 'components';
import { ResourceUsedFor } from 'graphql/generated/globalTypes';
import { profile } from 'graphql/generated/profile';
import { researchPage } from 'graphql/generated/researchPage';
import { profileUpdateMutation } from 'graphql/mutation';
import { profileQuery, researchPageQuery } from 'graphql/query';
import { useCallback, useEffect, useMemo } from 'react';
import { Organization, Project, ResourceList, Routine, Standard, User } from 'types';
import { listToListItems, openObject } from 'utils';
import { useLocation } from 'wouter';
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

/**
 * Default research resources
 */
const defaultResourceList: ResourceList = {
    usedFor: ResourceListUsedFor.Research,
    resources: [
        {
            link: 'https://cardano.stackexchange.com/',
            usedFor: ResourceUsedFor.Community,
            translations: [
                {
                    language: 'en',
                    title: 'Cardano Stack Exchange',
                    description: 'Browse and ask technical questions about Cardano',
                }
            ]
        } as any,
        {
            link: 'https://www.wolframalpha.com/',
            usedFor: ResourceUsedFor.Researching,
            translations: [
                {
                    language: 'en',
                    title: 'Wolfram Alpha',
                    description: 'Find answers to math questions',
                }
            ]
        } as any,
        {
            link: 'https://www.improvethenews.org/',
            usedFor: ResourceUsedFor.Feed,
            translations: [
                {
                    language: 'en',
                    title: 'News Feed',
                    description: 'Catch up on the latest news, with sliders for biases',
                }
            ]
        } as any,
        {
            link: 'https://scholar.google.com/',
            usedFor: ResourceUsedFor.Researching,
            translations: [
                {
                    language: 'en',
                    title: 'Google Scholar',
                    description: 'Research scientific articles',
                }
            ]
        } as any,
    ]
} as any;

export const ResearchPage = ({
    session
}: ResearchPageProps) => {
    const [, setLocation] = useLocation();
    const [getProfile, { data: profileData, loading: resourcesLoading }] = useLazyQuery<profile>(profileQuery);
    useEffect(() => { if (session?.id) getProfile() }, [getProfile, session])

    const resourceList = useMemo(() => {
        if (!profileData?.profile?.resourceLists) return defaultResourceList;
        return profileData.profile.resourceLists.find(list => list.usedFor === ResourceListUsedFor.Research) ?? null;
    }, [profileData]);
    const [updateResources] = useMutation<profile>(profileUpdateMutation);
    const handleResourcesUpdate = useCallback((updatedList: ResourceList) => {
        getProfile();
    }, [getProfile]);

    const { data: researchPageData, loading: researchPageLoading } = useQuery<researchPage>(researchPageQuery);

    /**
     * Opens page for list item
     */
    const toItemPage = useCallback((item: Organization | Project | Routine | Standard | User, event: any) => {
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
                <Typography component="h1" variant="h3" sx={{ fontSize: { xs: '2rem', sm: '3rem' } }}>Research Dashboard</Typography>
                <HelpButton markdown={researchPageText} sx={{ width: '40px', height: '40px' }} />
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
                <TitleContainer
                    title={"Processes"}
                    helpText={processesText}
                    loading={researchPageLoading}
                    onClick={() => { }}
                    options={[['Create', () => { }], ['See all', () => { }]]}
                >
                    {processes}
                </TitleContainer>
                <TitleContainer
                    title={"Newly Completed"}
                    helpText={newlyCompletedText}
                    loading={researchPageLoading}
                    onClick={() => { }}
                    options={[['See all', () => { }]]}
                >
                    {newlyCompleted}
                </TitleContainer>
                <TitleContainer
                    title={"Vote"}
                    helpText={voteText}
                    loading={researchPageLoading}
                    onClick={() => { }}
                    options={[['See all', () => { }]]}
                >
                    {needVotes}
                </TitleContainer>
                <TitleContainer
                    title={"Donate or Invest"}
                    helpText={donateOrInvestText}
                    loading={researchPageLoading}
                    onClick={() => { }}
                    options={[['See all', () => { }]]}
                >
                    {needInvestments}
                </TitleContainer>
                <TitleContainer
                    title={"Join a Team"}
                    helpText={joinATeamText}
                    loading={researchPageLoading}
                    onClick={() => { }}
                    options={[['Update profile', () => { }], ['See all', () => { }]]}
                >
                    {needMembers}
                </TitleContainer>
            </Stack>
        </Box>
    )
}