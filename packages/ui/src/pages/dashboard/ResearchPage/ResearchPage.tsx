import { useQuery } from '@apollo/client';
import { RoutineSortBy } from '@local/shared';
import { Box, Stack, Typography } from '@mui/material';
import { HelpButton, OrganizationListItem, ProjectListItem, RoutineListItem, TitleContainer } from 'components';
import { OrganizationSortBy, ProjectSortBy, ResourceUsedFor } from 'graphql/generated/globalTypes';
import { organizations, organizationsVariables } from 'graphql/generated/organizations';
import { projects, projectsVariables } from 'graphql/generated/projects';
import { routines, routinesVariables } from 'graphql/generated/routines';
import { organizationsQuery, projectsQuery, routinesQuery } from 'graphql/query';
import { useEffect, useMemo, useState } from 'react';
import { Organization, Project, Routine } from 'types';
import { ResearchPageProps } from '../types';
import donateOrInvestMarkdown from './donateOrInvestHelp.md';
import joinATeamMarkdown from './joinATeamHelp.md';
import newlyCompletedMarkdown from './newlyCompletedHelp.md';
import processesMarkdown from './processesHelp.md';
import researchPageMarkdown from './researchPageHelp.md';
import voteMarkdown from './voteHelp.md';

export const ResearchPage = ({
    session
}: ResearchPageProps) => {

    const { data: processesData, loading: processesLoading } = useQuery<routines, routinesVariables>(routinesQuery, { 
        variables: { input: { 
            take: 5,
            sortBy: RoutineSortBy.AlphabeticalAsc,
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