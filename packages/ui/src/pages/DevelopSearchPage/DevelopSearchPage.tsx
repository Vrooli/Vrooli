/**
 * Search page for personal objects (active runs, completed runs, views, stars)
 */
import { Box, Stack, Tab, Tabs, Typography } from "@mui/material";
import { SearchList } from "components";
import { useCallback, useMemo, useState } from "react";
import { useLocation } from '@shared/route';
import { DevelopSearchPageProps } from "../types";
import { parseSearchParams, stringifySearchParams, openObject } from "utils";
import { runsQuery, starsQuery, viewsQuery } from "graphql/query";
import { ListRun, ListStar, ListView } from "types";
import { DocumentNode } from "graphql";

enum DevelopSearchTypes {
    InProgress = 'InProgress',
    Recent = 'Recent',
    Completed = 'Completed',
}

type BaseParams = {
    itemKeyPrefix: string;
    objectType: DevelopSearchTypes | undefined;
    query: any | undefined;
    title: string;
    where: { [x: string]: any };
}

type SearchObject = ListRun | ListView | ListStar;

const tabOptions: [string, ObjectType][] = [
    ['In Progress', ObjectType.Run],
    ['Recent', ObjectType.View],
    ['Completed', ObjectType.Star],
];

/**
 * Maps object types to queries.
 */
const queryMap: { [key in ObjectType]?: DocumentNode } = {
    [ObjectType.Run]: runsQuery,
    [ObjectType.View]: viewsQuery,
    [ObjectType.Star]: starsQuery,
}

/**
 * Maps object types to titles
 */
const titleMap: { [key in ObjectType]?: string } = {
    [ObjectType.Run]: 'Runs',
    [ObjectType.View]: 'Views',
    [ObjectType.Star]: 'Stars',
}

/**
 * Maps object types to wheres (additional queries for search)
 */
const whereMap: { [key in ObjectType]?: { [x: string]: any } } = {
    [ObjectType.Run]: {},
    [ObjectType.View]: {},
    [ObjectType.Star]: {},
}

export function DevelopSearchPage({
    session,
}: DevelopSearchPageProps) {
    const [, setLocation] = useLocation();

    const handleSelected = useCallback((selected: SearchObject) => { openObject(selected, setLocation) }, [setLocation]);

    // Handle tabs
    const [tabIndex, setTabIndex] = useState<number>(() => {
        const searchParams = parseSearchParams(window.location.search);
        console.log('finding tab index', window.location.search, searchParams)
        const availableTypes: ObjectType[] = tabOptions.map(t => t[1]);
        const objectType: ObjectType | undefined = availableTypes.includes(searchParams.type as ObjectType) ? searchParams.type as ObjectType : undefined;
        const index = tabOptions.findIndex(t => t[1] === objectType);
        return Math.max(0, index);
    });
    const handleTabChange = (_e, newIndex: number) => { setTabIndex(newIndex) };

    // On tab change, update BaseParams, document title, where, and URL
    const { itemKeyPrefix, objectType, query, title, where } = useMemo<BaseParams>(() => {
        // Update tab title
        document.title = `Search ${tabOptions[tabIndex][0]}`;
        // Get object type
        const objectType: ObjectType = tabOptions[tabIndex][1];
        // Update URL
        const params = parseSearchParams(window.location.search);
        params.type = objectType;
        setLocation(stringifySearchParams(params), { replace: true });
        // Get other BaseParams
        const itemKeyPrefix = `${objectType}-list-item`;
        const query = queryMap[objectType];
        const title = (objectType in titleMap ? titleMap[objectType] : 'Search') as string;
        const where = (objectType in whereMap ? whereMap[objectType] : {}) as { [x: string]: any };
        return { itemKeyPrefix, objectType, query, title, where };
    }, [setLocation, tabIndex]);

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
            {objectType && <SearchList
                itemKeyPrefix={itemKeyPrefix}
                searchPlaceholder={'Search...'}
                query={query}
                take={20}
                objectType={objectType}
                onObjectSelect={handleSelected}
                session={session}
                zIndex={200}
                where={where}
            />}
        </Box >
    )
}