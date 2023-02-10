/**
 * Search page for personal objects (active runs, completed runs, views, stars)
 */
import { Box, Stack, Tab, Tabs, Typography } from "@mui/material";
import { PageContainer, SearchList } from "components";
import { useMemo, useState } from "react";
import { useLocation } from '@shared/route';
import { DevelopSearchPageProps } from "../types";
import { parseSearchParams, SearchType, DevelopSearchPageTabOption as TabOption, addSearchParams } from "utils";

// Tab data type
type BaseParams = {
    itemKeyPrefix: string;
    searchType: SearchType;
    title: string;
    where: { [x: string]: any };
}

// Data for each tab
const tabParams: { [key in TabOption]: BaseParams } = {
    [TabOption.InProgress]: {
        itemKeyPrefix: 'inProgress-list-item',
        searchType: SearchType.ProjectOrRoutine,
        title: 'In Progress',
        where: { isComplete: false, routineIsInternal: false },
    },
    [TabOption.Recent]: {
        itemKeyPrefix: 'recent-list-item',
        searchType: SearchType.ProjectOrRoutine,
        title: 'Recent',
        where: { routineIsInternal: false },
    },
    [TabOption.Completed]: {
        itemKeyPrefix: 'completed-list-item',
        searchType: SearchType.ProjectOrRoutine,
        title: 'Completed',
        where: { isComplete: true, routineIsInternal: false },
    },
}

// [title, searchType] for each tab
const tabOptions: [string, TabOption][] = Object.entries(tabParams).map(([key, value]) => [value.title, key as TabOption]);

//TODO possibly deprecated
export function DevelopSearchPage({
    session,
}: DevelopSearchPageProps) {
    const [, setLocation] = useLocation();

    // Handle tabs
    const [tabIndex, setTabIndex] = useState<number>(() => {
        const searchParams = parseSearchParams();
        const availableTypes: TabOption[] = tabOptions.map(t => t[1]);
        const index = availableTypes.indexOf(searchParams.type as TabOption);
        return Math.max(0, index);
    });
    const handleTabChange = (e, newIndex: number) => {
        e.preventDefault();
        // Update search params
        addSearchParams(setLocation, {
            type: tabOptions[newIndex][1],
        });
        // Update tab index
        setTabIndex(newIndex)
    };

    // On tab change, update BaseParams, document title, where, and URL
    const { itemKeyPrefix, searchType, title, where } = useMemo<BaseParams>(() => {
        // Update tab title
        document.title = `Search ${tabOptions[tabIndex][0]}`;
        // Get object type
        const searchType: TabOption = tabOptions[tabIndex][1];
        // Return base params
        return tabParams[searchType]
    }, [tabIndex]);

    return (
        <PageContainer>
            {/* Navigate between search pages */}
            <Box display="flex" justifyContent="center" width="100%">
                <Tabs
                    value={tabIndex}
                    onChange={handleTabChange}
                    indicatorColor="secondary"
                    textColor="inherit"
                    variant="scrollable"
                    scrollButtons="auto"
                    allowScrollButtonsMobile
                    aria-label="search-type-tabs"
                    sx={{
                        marginBottom: 1,
                    }}
                >
                    {tabOptions.map((option, index) => (
                        <Tab
                            key={index}
                            id={`search-tab-${index}`}
                            {...{ 'aria-controls': `search-tabpanel-${index}` }}
                            label={option[0]}
                            color={index === 0 ? '#ce6c12' : 'default'}
                            component="a"
                            href={option[1]}
                        />
                    ))}
                </Tabs>
            </Box>
            <Stack direction="row" alignItems="center" justifyContent="center" sx={{ paddingTop: 2 }}>
                <Typography component="h2" variant="h4">{title}</Typography>
            </Stack>
            {searchType && <SearchList
                id="develop-search-page-list"
                itemKeyPrefix={itemKeyPrefix}
                searchPlaceholder={'Search...'}
                take={20}
                searchType={searchType}
                session={session}
                zIndex={200}
                where={where}
            />}
        </PageContainer>
    )
}