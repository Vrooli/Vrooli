import { Box, Tab, Tabs, Typography } from '@mui/material';
import { useCallback, useState } from 'react';
import { StatsList } from 'components';
import SwipeableViews from 'react-swipeable-views';

interface TabPanelProps {
    children?: React.ReactNode[] | React.ReactNode;
    index: number;
    value: number;
}

const TabPanel = ({
    children,
    index,
    value,
}: TabPanelProps) => {
  
    return (
      <div
        role="tabpanel"
        hidden={value !== index}
        id={`full-width-tabpanel-${index}`}
        aria-labelledby={`full-width-tab-${index}`}
      >
        {value === index && (
          <Box sx={{ p: 3 }}>
            <Typography>{children}</Typography>
          </Box>
        )}
      </div>
    );
  }

const tabProps = (index: number) => ({
    id: `full-width-tab-${index}`,
    'aria-controls': `full-width-tabpanel-${index}`,
});

const tabLabels = ['Daily', 'Weekly', 'Monthly', 'Yearly', 'All time'];

export const StatsPage = () => {
    const [tabIndex, setTabIndex] = useState(0);

    const handleTabChange = (event, newValue) => {
        console.log('setting tab index', newValue);
        setTabIndex(newValue);
    };

    const handleTabSwipe = (index) => {
        setTabIndex(index);
    };

    // Each tab contains an overview of the current stats (plus the % change from the previous interval).
    // For example, it might say "Daily active users: 5,000 (+10%)"
    // Below, it displays bar charts for each of the stats.
    const createTab = useCallback((index: number) => (
        <TabPanel value={index} index={index}>
            <Typography component="h1" variant="h3" textAlign="center">Quick Overview</Typography>
            <Typography component="h1" variant="h3" textAlign="center">The Pretty Pictures</Typography>
            <StatsList data={[{}, {}, {}, {}, {}, {}]} />
        </TabPanel>
    ), [])

    return (
        <Box id='page'>
            <Tabs
                value={tabIndex}
                onChange={handleTabChange}
                indicatorColor="secondary"
                textColor="inherit"
                variant="fullWidth"
                aria-label="site-statistics-tabs"
            >
                <Tab label="Daily" {...tabProps(0)} />
                <Tab label="Weekly" {...tabProps(1)} />
                <Tab label="Monthly" {...tabProps(2)} />
                <Tab label="Yearly" {...tabProps(3)} />
                <Tab label="All Time" {...tabProps(4)} />
            </Tabs>
            <SwipeableViews
                axis={'x'}
                index={tabIndex}
                onChangeIndex={handleTabSwipe}
            >
                {tabLabels.map((label, index) => createTab(index))}
            </SwipeableViews>
        </Box>
    )
};
