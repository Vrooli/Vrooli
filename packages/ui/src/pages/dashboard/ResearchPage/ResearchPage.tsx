import { useLazyQuery, useMutation, useQuery } from '@apollo/client';
import { ResourceListUsedFor, RoutineSortBy } from '@local/shared';
import { Box, Stack, Typography } from '@mui/material';
import { HelpButton, OrganizationListItem, ProjectListItem, ResourceListHorizontal, RoutineListItem, TitleContainer } from 'components';
import { OrganizationSortBy, ProjectSortBy, ResourceUsedFor } from 'graphql/generated/globalTypes';
import { organizations, organizationsVariables } from 'graphql/generated/organizations';
import { profile } from 'graphql/generated/profile';
import { projects, projectsVariables } from 'graphql/generated/projects';
import { researchPage } from 'graphql/generated/researchPage';
import { routines, routinesVariables } from 'graphql/generated/routines';
import { profileUpdateMutation } from 'graphql/mutation';
import { organizationsQuery, profileQuery, projectsQuery, researchPageQuery, routinesQuery } from 'graphql/query';
import { useCallback, useEffect, useMemo } from 'react';
import { Organization, Project, ResourceList, Routine } from 'types';
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

    const { data: researchPageData, loading: researchPageLoading } = useQuery<researchPage>(researchPageQuery);

    const processes = useMemo(() => researchPageData?.researchPage?.processes?.map((o, index) => (
        <RoutineListItem
            key={`research-processes-list-item-${index}`}
            index={index}
            session={session}
            data={o as Routine}
            onClick={() => {}}
        />
    )) ?? [], []);

    const newlyCompleted = useMemo(() => researchPageData?.researchPage?.newlyCompleted?.map((o, index) => (
        <ProjectListItem
            key={`recently-completed-projects-list-item-${index}`}
            index={index}
            session={session}
            data={o as Project}
            onClick={() => {}}
        />
    )) ?? [], []);

    const needVotes = useMemo(() => researchPageData?.researchPage?.needVotes?.map((o, index) => (
        <ProjectListItem
            key={`projects-that-need-votes-list-item-${index}`}
            index={index}
            session={session}
            data={o as Project}
            onClick={() => {}}
        />
    )) ?? [], []);

    const donateOrInvest = useMemo(() => researchPageData?.researchPage?.needInvestments?.map((o, index) => ( 
        <ProjectListItem
            key={`projects-that-need-funding-list-item-${index}`}
            index={index}
            session={session}
            data={o as Project}
            onClick={() => {}}
        />
    )) ?? [], []);

    const needMembers = useMemo(() => researchPageData?.researchPage?.needMembers?.map((o, index) => (
        <OrganizationListItem
            key={`looking-for-members-list-item-${index}`}
            index={index}
            session={session}
            data={o as Organization}
            onClick={() => {}}
        />
    )) ?? [], []);

    return (
        <Box id="page">
            {/* Title and help button */}
            <Stack 
                direction="row" 
                justifyContent="center" 
                alignItems="center" 
                sx={{ marginTop: 2, marginBottom: 2}}
            >
                <Typography component="h1" variant="h3" sx={{ fontSize: { xs: '2rem', sm: '3rem' }}}>Research Dashboard</Typography>
                <HelpButton markdown={researchPageText} sx={{width: '40px', height: '40px'}} />
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
                    {donateOrInvest}
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