import { Box, Tab, Tabs, Typography, useTheme } from '@mui/material';
import { useState } from 'react';
import { StatsList } from 'components';

const tabProps = (index: number) => ({
    id: `stats-tab-${index}`,
    'aria-controls': `full-width-tabpanel-${index}`,
});

const tabLabels = ['Daily', 'Weekly', 'Monthly', 'Yearly', 'All time'];

export const StatsPage = () => {
    const { breakpoints } = useTheme();
    
    // Handle tabs
    const [tabIndex, setTabIndex] = useState(0);
    const handleTabChange = (event, newValue) => {
        setTabIndex(newValue);
    };

    return (
        <Box id='page' sx={{
            padding: '0.5em',
            paddingTop: '64px',
            [breakpoints.up('md')]: {
                paddingTop: '10vh',
            },
        }}>
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
                    '& .MuiTabs-flexContainer': {
                        justifyContent: 'space-between',
                    },
                }}
            >
                {tabLabels.map((label, index) => (
                    <Tab key={index} label={label} {...tabProps(index)} />
                ))}
            </Tabs>
            <Typography component="h1" variant="h3" textAlign="center">Quick Overview</Typography>
            <Typography component="h1" variant="h3" textAlign="center">The Pretty Pictures</Typography>
            <StatsList data={[{}, {}, {}, {}, {}, {}]} />
        </Box>
    )
};
