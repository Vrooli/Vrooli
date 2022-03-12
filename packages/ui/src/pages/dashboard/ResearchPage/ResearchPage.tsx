import { useLazyQuery, useMutation, useQuery } from '@apollo/client';
import { ResourceListUsedFor, RoutineSortBy } from '@local/shared';
import { Box, Stack, Typography } from '@mui/material';
import { HelpButton, OrganizationListItem, ProjectListItem, ResourceListHorizontal, RoutineListItem, TitleContainer } from 'components';
import { OrganizationSortBy, ProjectSortBy, ResourceUsedFor } from 'graphql/generated/globalTypes';
import { organizations, organizationsVariables } from 'graphql/generated/organizations';
import { profile } from 'graphql/generated/profile';
import { projects, projectsVariables } from 'graphql/generated/projects';
import { routines, routinesVariables } from 'graphql/generated/routines';
import { profileUpdateMutation } from 'graphql/mutation';
import { organizationsQuery, profileQuery, projectsQuery, routinesQuery } from 'graphql/query';
import { mutationWrapper } from 'graphql/utils/wrappers';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { Organization, Project, Resource, ResourceList, Routine } from 'types';
import { formatForUpdate, Pubs } from 'utils';
import { ResearchPageProps } from '../types';
import donateOrInvestMarkdown from './donateOrInvestHelp.md';
import joinATeamMarkdown from './joinATeamHelp.md';
import newlyCompletedMarkdown from './newlyCompletedHelp.md';
import processesMarkdown from './processesHelp.md';
import researchPageMarkdown from './researchPageHelp.md';
import voteMarkdown from './voteHelp.md';

/**
 * Default research resources
 */
 const defaultResourceList: ResourceList = {    
    usedFor: ResourceListUsedFor.Develop,
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
        return profileData.profile.resourceLists.find(list => list.usedFor === ResourceListUsedFor.Develop) ?? null;
    }, [profileData]);
    const [updateResources] = useMutation<profile>(profileUpdateMutation);
    const handleResourcesUpdate = useCallback((updatedList: ResourceList) => {
        mutationWrapper({
            mutation: updateResources,
            input: formatForUpdate(profileData?.profile, {
                ...profileData?.profile,
                resourcesLearn: updatedList
            }, [], []),
            onError: (response) => {
                PubSub.publish(Pubs.Snack, { message: 'Error occurred.', severity: 'error', data: { error: response } });
            }
        })
    }, [updateResources]);

    const { data: processesData, loading: processesLoading } = useQuery<routines, routinesVariables>(routinesQuery, { 
        variables: { input: { 
            take: 5,
            sortBy: RoutineSortBy.VotesDesc,
            tags: ['research'],
        } } 
    });

    const { data: newlyCompletedData, loading: newlyCompletedLoading } = useQuery<projects, projectsVariables>(projectsQuery, { 
        variables: { input: { 
            take: 5,
            isComplete: true,
            sortBy: ProjectSortBy.DateCompletedAsc,
        } } 
    });

    // We check if a project needs votes by the existence of a Proposal resource
    const { data: needVotesData, loading: needVotesLoading } = useQuery<projects, projectsVariables>(projectsQuery, { 
        variables: { input: { 
            take: 5,
            resourceTypes: [ResourceUsedFor.Proposal],
        } } 
    });

    // We check if a project needs donations or is an investment opportunity by the existence of a token TODO
    // const { data: donateOrInvestData, loading: donateOrInvestLoading } = useQuery<projects, projectsVariables>(projectsQuery, { 
    //     variables: { input: { 
    //         take: 5,
    //         TODO
    //     } } 
    // });

    const { data: needMembersData, loading: needMembersLoading } = useQuery<organizations, organizationsVariables>(organizationsQuery, { 
        variables: { input: { 
            take: 5,
            isOpenToNewMembers: true,
            sortBy: OrganizationSortBy.StarsDesc
        } } 
    });

    const processes = useMemo(() => processesData?.routines?.edges?.map((o, index) => (
        <RoutineListItem
            key={`research-processes-list-item-${index}`}
            index={index}
            session={session}
            data={o.node as Routine}
            onClick={() => {}}
        />
    )) ?? [], []);

    const newlyCompleted = useMemo(() => newlyCompletedData?.projects?.edges?.map((o, index) => (
        <ProjectListItem
            key={`recently-completed-projects-list-item-${index}`}
            index={index}
            session={session}
            data={o.node as Project}
            onClick={() => {}}
        />
    )) ?? [], []);

    const needVotes = useMemo(() => needVotesData?.projects?.edges?.map((o, index) => (
        <ProjectListItem
            key={`projects-that-need-votes-list-item-${index}`}
            index={index}
            session={session}
            data={o.node as Project}
            onClick={() => {}}
        />
    )) ?? [], []);

    // const donateOrInvest = useMemo(() => donateOrInvestData?.projects?.edges?.map((o, index) => ( 
    //     <ProjectListItem
    //         key={`projects-that-need-funding-list-item-${index}`}
    //         index={index}
    //         session={session}
    //         data={o.node as Project}
    //         onClick={() => {}}
    //     />
    // )) ?? [], []);

    const needMembers = useMemo(() => needMembersData?.organizations?.edges?.map((o, index) => (
        <OrganizationListItem
            key={`looking-for-members-list-item-${index}`}
            index={index}
            session={session}
            data={o.node as Organization}
            onClick={() => {}}
        />
    )) ?? [], []);

    // Parse help button markdown
    const [donateOrInvestText, setDonateOrInvestText] = useState('');
    const [joinATeamText, setJoinATeamText] = useState('');
    const [newlyCompletedText, setNewlyCompletedText] = useState('');
    const [processesText, setProcessesText] = useState('');
    const [researchPageText, setResearchPageText] = useState('');
    const [voteText, setVoteText] = useState('');
    useEffect(() => {
        fetch(donateOrInvestMarkdown).then((r) => r.text()).then((text) => { setDonateOrInvestText(text) });
        fetch(joinATeamMarkdown).then((r) => r.text()).then((text) => { setJoinATeamText(text) });
        fetch(newlyCompletedMarkdown).then((r) => r.text()).then((text) => { setNewlyCompletedText(text) });
        fetch(processesMarkdown).then((r) => r.text()).then((text) => { setProcessesText(text) });
        fetch(researchPageMarkdown).then((r) => r.text()).then((text) => { setResearchPageText(text) });
        fetch(voteMarkdown).then((r) => r.text()).then((text) => { setVoteText(text) });
    }, []);

    return (
        <Box id="page">
            {/* Title and help button */}
            <Stack 
                direction="row" 
                justifyContent="center" 
                alignItems="center" 
                sx={{ marginTop: 2, marginBottom: 2}}
            >
                <Typography component="h1" variant="h3">Research Dashboard</Typography>
                <HelpButton markdown={researchPageText} sx={{width: '40px', height: '40px'}} />
            </Stack>
            <Stack direction="column" spacing={10}>
                {/* Resources */}
                <ResourceListHorizontal 
                    title={Boolean(session?.id) ? '📌 Resources' : 'Example Resources'}
                    list={resourceList}
                    canEdit={Boolean(session?.id)}
                    handleUpdate={handleResourcesUpdate}
                    session={session}
                />
                <TitleContainer
                    title={"Processes"}
                    helpText={processesText}
                    loading={processesLoading}
                    onClick={() => { }}
                    options={[['Create', () => { }], ['See all', () => { }]]}
                >
                    {processes}
                </TitleContainer>
                <TitleContainer
                    title={"Newly Completed"}
                    helpText={newlyCompletedText}
                    loading={newlyCompletedLoading}
                    onClick={() => { }}
                    options={[['See all', () => { }]]}
                >
                    {newlyCompleted}
                </TitleContainer>
                <TitleContainer
                    title={"Vote"}
                    helpText={voteText}
                    loading={needVotesLoading}
                    onClick={() => { }}
                    options={[['See all', () => { }]]}
                >
                    {needVotes}
                </TitleContainer>
                {/* <TitleContainer
                    title={"Donate or Invest"}
                    helpText={donateOrInvestText}
                    loading={donateOrInvestLoading}
                    onClick={() => { }}
                    options={[['See all', () => { }]]}
                >
                    {donateOrInvest}
                </TitleContainer> */}
                <TitleContainer
                    title={"Join a Team"}
                    helpText={joinATeamText}
                    loading={needMembersLoading}
                    onClick={() => { }}
                    options={[['Update profile', () => { }], ['See all', () => { }]]}
                >
                    {needMembers}
                </TitleContainer>
            </Stack>
        </Box>
    )
}