import { Box, Stack, Typography } from '@mui/material';
import { HelpButton, ProjectListItem, ResourceListHorizontal, RoutineListItem, TitleContainer } from 'components';
import { useEffect, useMemo, useState } from 'react';
import { LearnPageProps } from '../types';
import courseMarkdown from './courseHelp.md';
import learnPageMarkdown from './learnPageHelp.md';
import tutorialMarkdown from './tutorialHelp.md';

export const LearnPage = ({
    session,
}: LearnPageProps) => {

    const courses = useMemo(() => [].map((o, index) => (
        <ProjectListItem
            key={`course-list-item-${'TODO'}`}
            index={index}
            session={session}
            data={o}
            isOwn={false}
            onClick={() => { }}
        />
    )), []);

    const tutorials = useMemo(() => [].map((o, index) => (
        <RoutineListItem
            key={`tutorial-list-item-${'TODO'}`}
            index={index}
            session={session}
            data={o}
            isOwn={false}
            onClick={() => { }}
        />
    )), []);

    // Parse help button markdown
    const [courseText, setCourseText] = useState('');
    const [learnPageText, setLearnPageText] = useState('');
    const [tutorialText, setTutorialText] = useState('');   
    useEffect(() => {
        fetch(courseMarkdown).then((r) => r.text()).then((text) => { setCourseText(text) });
        fetch(learnPageMarkdown).then((r) => r.text()).then((text) => { setLearnPageText(text) });
        fetch(tutorialMarkdown).then((r) => r.text()).then((text) => { setTutorialText(text) });
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
                <Typography component="h1" variant="h3">Learn Dashboard</Typography>
                <HelpButton markdown={learnPageText} sx={{width: '40px', height: '40px'}} />
            </Stack>
            <Stack direction="column" spacing={10}>
                {/* Resources */}
                <ResourceListHorizontal />
                {/* Available courses */}
                <TitleContainer
                    title={"Courses"}
                    helpText={courseText}
                    loading={true}
                    onClick={() => { }}
                    options={[['Create', () => { }], ['See all', () => { }]]}
                >
                    {courses}
                </TitleContainer>
                {/* Available tutorials */}
                <TitleContainer
                    title={"Tutorials"}
                    helpText={tutorialText}
                    loading={true}
                    onClick={() => { }}
                    options={[['Create', () => { }], ['See all', () => { }]]}
                >
                    {tutorials}
                </TitleContainer>
            </Stack>
        </Box>
    )
}