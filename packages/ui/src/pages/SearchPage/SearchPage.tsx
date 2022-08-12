import { Box, Button, IconButton, Stack, Tab, Tabs, Tooltip, Typography } from "@mui/material";
import { SearchList } from "components";
import { useCallback, useMemo, useState } from "react";
import { centeredDiv } from "styles";
import { useLocation } from "wouter";
import { SearchPageProps } from "../types";
import { Add as AddIcon } from '@mui/icons-material';
import { ObjectType, parseSearchParams, stringifySearchParams } from "utils";
import { organizationsQuery, projectsQuery, routinesQuery, standardsQuery, usersQuery } from "graphql/query";
import { lazily } from "react-lazily";
import { Organization, Project, Routine, Standard, User } from "types";

type BaseParams = {
    itemKeyPrefix: string;
    objectType: ObjectType | undefined;
    query: any | undefined;
    title: string;
}

type SearchObject = Organization | Project | Routine | Standard | User;

const tabOptions: [string, ObjectType][] = [
    ['Organizations', ObjectType.Organization],
    ['Projects', ObjectType.Project],
    ['Routines', ObjectType.Routine],
    ['Standards', ObjectType.Standard],
    ['Users', ObjectType.User],
];

/**
 * Maps object types to queries.
 */
const queryMap = {
    [ObjectType.Organization]: organizationsQuery,
    [ObjectType.Project]: projectsQuery,
    [ObjectType.Routine]: routinesQuery,
    [ObjectType.Standard]: standardsQuery,
    [ObjectType.User]: usersQuery,
}

/**
 * Maps object types to titles
 */
const titleMap = {
    [ObjectType.Organization]: 'Organizations',
    [ObjectType.Project]: 'Projects',
    [ObjectType.Routine]: 'Routines',
    [ObjectType.Standard]: 'Standards',
    [ObjectType.User]: 'Users',
}

export function SearchPage({
    session,
}: SearchPageProps) {
    const [, setLocation] = useLocation();

    // Handles item add/select/edit
    const [selectedItem, setSelectedItem] = useState<SearchObject | undefined>(undefined);
    const handleSelected = useCallback((selected: SearchObject) => {
        setSelectedItem(selected);
    }, []);

    // Handle tabs
    const [tabIndex, setTabIndex] = useState<number>(() => {
        const searchParams = parseSearchParams(window.location.search);
        const availableTypes: string[] = tabOptions.map(t => t[1]);
        const objectType: ObjectType | undefined = availableTypes.includes(searchParams.type as string) ? searchParams.type as ObjectType : undefined;
        const index = tabOptions.findIndex(t => t[1] === objectType);
        return Math.max(0, index);
    });
    const handleTabChange = (_e, newIndex: number) => { setTabIndex(newIndex) };

    // On tab change, update BaseParams, document title, and URL
    const { itemKeyPrefix, objectType, query, title } = useMemo<BaseParams>(() => {
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
        const title = objectType ? titleMap[objectType] : 'Search';
        return { itemKeyPrefix, objectType, query, title };
    }, [setLocation, tabIndex]);

    // Popup button
    const [popupButton, setPopupButton] = useState<boolean>(false);
    const handleScrolledFar = useCallback(() => { setPopupButton(true) }, [])
    const popupButtonContainer = useMemo(() => (
        <Box sx={{ ...centeredDiv, paddingTop: 1 }}>
            {/* <Tooltip title={popupButtonTooltip}>
                <Button
                    onClick={onPopupButtonClick}
                    size="large"
                    sx={{
                        zIndex: 100,
                        minWidth: 'min(100%, 200px)',
                        height: '48px',
                        borderRadius: 3,
                        position: 'fixed',
                        bottom: 'calc(5em + env(safe-area-inset-bottom))',
                        transform: popupButton ? 'translateY(0)' : 'translateY(calc(10em + env(safe-area-inset-bottom)))',
                        transition: 'transform 1s ease-in-out',
                    }}
                >
                    {popupButtonText}
                </Button>
            </Tooltip> */}
        </Box>
    ), [popupButton]);

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
                {/* <Tooltip title="Add new" placement="top">
                    <IconButton
                        size="large"
                        onClick={onAddClick}
                        sx={{
                            padding: 1,
                        }}
                    >
                        <AddIcon color="secondary" sx={{ width: '1.5em', height: '1.5em' }} />
                    </IconButton>
                </Tooltip> */}
            </Stack>
            {objectType && <SearchList
                itemKeyPrefix={itemKeyPrefix}
                searchPlaceholder={'Search...'}
                query={query}
                take={20}
                objectType={objectType}
                onObjectSelect={handleSelected}
                onScrolledFar={handleScrolledFar}
                session={session}
                zIndex={200}
            />}
            {popupButtonContainer}
        </Box >
    )
}