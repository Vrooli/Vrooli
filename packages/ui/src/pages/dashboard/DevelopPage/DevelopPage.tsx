import { Box, Stack, Typography } from '@mui/material';
import { HelpButton, ProjectListItem, TitleContainer } from 'components';
import { useEffect, useMemo, useState } from 'react';
import { DevelopPageProps } from '../types';
import completedMarkdown from './completedHelp.md';
import developPageMarkdown from './developPageHelp.md';
import inProgressMarkdown from './inProgressHelp.md';
import recentMarkdown from './recentHelp.md';

export const DevelopPage = ({
    session
}: DevelopPageProps) => {
    const inProgress = useMemo(() => [].map((o, index) => (
        <ProjectListItem
            key={`in-progress-list-item-${'TODO'}`}
            index={index}
            session={session}
            data={o}
            onClick={() => { }}
        />
    )), []);

    const recent = useMemo(() => [].map((o, index) => (
        <ProjectListItem
            key={`recently-projects-list-item-${'TODO'}`}
            index={index}
            session={session}
            data={o}
            onClick={() => { }}
        />
    )), []);

    const completed = useMemo(() => [].map((o, index) => (
        <ProjectListItem
            key={`completed-projects-list-item-${'TODO'}`}
            index={index}
            session={session}
            data={o}
            onClick={() => { }}
        />
    )), []);

    // Parse help button markdown
    const [completedText, setCompletedText] = useState('');
    const [developPageText, setDevelopPageText] = useState('');
    const [inProgressText, setInProgressText] = useState('');
    const [recentText, setRecentText] = useState('');
    useEffect(() => {
        fetch(completedMarkdown).then((r) => r.text()).then((text) => { setCompletedText(text) });
        fetch(developPageMarkdown).then((r) => r.text()).then((text) => { setDevelopPageText(text) });
        fetch(inProgressMarkdown).then((r) => r.text()).then((text) => { setInProgressText(text) });
        fetch(recentMarkdown).then((r) => r.text()).then((text) => { setRecentText(text) });
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
                <Typography component="h1" variant="h3">Develop Dashboard</Typography>
                <HelpButton markdown={developPageText} sx={{width: '40px', height: '40px'}} />
            </Stack>
            <Stack direction="column" spacing={10}>
                {/* TODO my organizations list */}
                <TitleContainer
                    title={"In Progress"}
                    helpText={inProgressText}
                    loading={true}
                    onClick={() => { }}
                    options={[['Create', () => { }], ['See all', () => { }]]}
                >
                    {inProgress}
                </TitleContainer>
                <TitleContainer
                    title={"Recent"}
                    helpText={recentText}
                    loading={true}
                    onClick={() => { }}
                    options={[['See all', () => { }]]}
                >
                    {recent}
                </TitleContainer>
                <TitleContainer
                    title={"Completed"}
                    helpText={completedText}
                    loading={true}
                    onClick={() => { }}
                    options={[['See all', () => { }]]}
                >
                    {completed}
                </TitleContainer>
            </Stack>
        </Box>
    )
}