import { Box, Stack, Typography } from '@mui/material';
import { HelpButton, OrganizationListItem, ProjectListItem, RoutineListItem, TitleContainer } from 'components';
import { useEffect, useMemo, useState } from 'react';
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
    const processes = useMemo(() => [].map((o, index) => (
        <RoutineListItem
            key={`research-processes-list-item-${'TODO'}`}
            index={index}
            session={session}
            data={o}
            onClick={() => {}}
        />
    )), []);

    const newlyCompleted = useMemo(() => [].map((o, index) => (
        <ProjectListItem
            key={`recently-completed-projects-list-item-${'TODO'}`}
            index={index}
            session={session}
            data={o}
            onClick={() => {}}
        />
    )), []);

    const needVotes = useMemo(() => [].map((o, index) => (
        <ProjectListItem
            key={`projects-that-need-votes-list-item-${'TODO'}`}
            index={index}
            session={session}
            data={o}
            onClick={() => {}}
        />
    )), []);

    const needFunding = useMemo(() => [].map((o, index) => (
        <ProjectListItem
            key={`projects-that-need-funding-list-item-${'TODO'}`}
            index={index}
            session={session}
            data={o}
            onClick={() => {}}
        />
    )), []);

    const needMembers = useMemo(() => [].map((o, index) => (
        <OrganizationListItem
            key={`looking-for-members-list-item-${'TODO'}`}
            index={index}
            session={session}
            data={o}
            onClick={() => {}}
        />
    )), []);

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
                    loading={true}
                    onClick={() => { }}
                    options={[['Create', () => { }], ['See all', () => { }]]}
                >
                    {processes}
                </TitleContainer>
                <TitleContainer
                    title={"Newly Completed"}
                    helpText={newlyCompletedText}
                    loading={true}
                    onClick={() => { }}
                    options={[['See all', () => { }]]}
                >
                    {newlyCompleted}
                </TitleContainer>
                <TitleContainer
                    title={"Vote"}
                    helpText={voteText}
                    loading={true}
                    onClick={() => { }}
                    options={[['See all', () => { }]]}
                >
                    {needVotes}
                </TitleContainer>
                <TitleContainer
                    title={"Donate or Invest"}
                    helpText={donateOrInvestText}
                    loading={true}
                    onClick={() => { }}
                    options={[['See all', () => { }]]}
                >
                    {needFunding}
                </TitleContainer>
                <TitleContainer
                    title={"Join a Team"}
                    helpText={joinATeamText}
                    loading={true}
                    onClick={() => { }}
                    options={[['Update profile', () => { }], ['See all', () => { }]]}
                >
                    {needMembers}
                </TitleContainer>
            </Stack>
        </Box>
    )
}