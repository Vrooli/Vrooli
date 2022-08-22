/**
 * Search page for personal objects (active runs, completed runs, views, stars)
 */
import { Box, Stack, Tab, Tabs, Typography } from "@mui/material";
import { SearchList } from "components";
import { useCallback, useMemo, useState } from "react";
import { useLocation } from '@shared/route';
import { DevelopSearchPageProps } from "../types";
import { parseSearchParams, stringifySearchParams, openObject, SearchType, DevelopSearchPageTabOption as TabOption } from "utils";
import { ListProject, ListRoutine } from "types";

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
        where: { routineIsInternal: false },
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
        where: { routineIsInternal: false },
    },
}

// [title, searchType] for each tab
const tabOptions: [string, TabOption][] = Object.entries(tabParams).map(([key, value]) => [value.title, key as TabOption]);

type SearchObject = ListProject | ListRoutine;

export function DevelopSearchPage({
    session,
}: DevelopSearchPageProps) {
    const [, setLocation] = useLocation();

    const handleSelected = useCallback((selected: SearchObject) => { openObject(selected, setLocation) }, [setLocation]);

    // Handle tabs
    const [tabIndex, setTabIndex] = useState<number>(() => {
        const searchParams = parseSearchParams(window.location.search);
        const availableTypes: TabOption[] = tabOptions.map(t => t[1]);
        const index = availableTypes.indexOf(searchParams.type as TabOption);
        return Math.max(0, index);
    });
    const handleTabChange = (_e, newIndex: number) => { 
        // Update "type" in URL and remove all search params not shared by all tabs
        const { search, sort, time } = parseSearchParams(window.location.search);
        setLocation(stringifySearchParams({
            search,
            sort,
            time,
            type: tabOptions[tabIndex][1],
        }), { replace: true })
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
        <Box id='page' sx={{
            padding: '0.5em',
            paddingTop: { xs: '64px', md: '80px' },
        }}>
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
                        />
                    ))}
                </Tabs>
            </Box>
            <Stack direction="row" alignItems="center" justifyContent="center" sx={{ paddingTop: 2 }}>
                <Typography component="h2" variant="h4">{title}</Typography>
            </Stack>
            {searchType && <SearchList
                itemKeyPrefix={itemKeyPrefix}
                searchPlaceholder={'Search...'}
                take={20}
                searchType={searchType}
                onObjectSelect={handleSelected}
                session={session}
                zIndex={200}
                where={where}
            />}
        </Box >
    )
}