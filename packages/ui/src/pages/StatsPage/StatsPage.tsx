import { Box, Tab, Tabs, Typography } from '@mui/material';
import { useState } from 'react';
import { PageContainer, StatsList } from 'components';

const tabProps = (index: number) => ({
    id: `stats-tab-${index}`,
    'aria-controls': `full-width-tabpanel-${index}`,
});

const tabLabels = ['Daily', 'Weekly', 'Monthly', 'Yearly', 'All time'];

export const StatsPage = () => {
    // Handle tabs
    const [tabIndex, setTabIndex] = useState(0);
    const handleTabChange = (event, newValue) => {
        setTabIndex(newValue);
    };

    return (
        <PageContainer>
            <Box display="flex" justifyContent="center" width="100%">
                <Tabs
                    value={tabIndex}
                    onChange={handleTabChange}
                    indicatorColor="secondary"
                    textColor="inherit"
                    variant="scrollable"
                    scrollButtons="auto"
                    allowScrollButtonsMobile
                    aria-label="site-statistics-tabs"
                    sx={{
                        marginBottom: 1,
                        paddingLeft: '1em',
                        paddingRight: '1em',
                    }}
                >
                    {tabLabels.map((label, index) => (
                        <Tab key={index} label={label} {...tabProps(index)} />
                    ))}
                </Tabs>
            </Box>
            <Typography component="h1" variant="h3" textAlign="center">Quick Overview</Typography>
            <Typography component="h1" variant="h3" textAlign="center">The Pretty Pictures</Typography>
            <StatsList data={[{}, {}, {}, {}, {}, {}]} />
        </PageContainer>
    )
};
