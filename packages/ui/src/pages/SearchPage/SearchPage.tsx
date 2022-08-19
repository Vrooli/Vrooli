/**
 * Search page for organizations, projects, routines, standards, and users
 */
import { Box, Button, IconButton, Stack, Tab, Tabs, Tooltip, Typography } from "@mui/material";
import { SearchList, ShareDialog } from "components";
import { useCallback, useMemo, useState } from "react";
import { centeredDiv } from "styles";
import { useLocation } from '@shared/route';
import { SearchPageProps } from "../types";
import { Add as AddIcon } from '@mui/icons-material';
import { getObjectUrlBase, ObjectType, PubSub, parseSearchParams, stringifySearchParams, openObject } from "utils";
import { organizationsQuery, projectsQuery, routinesQuery, standardsQuery, usersQuery } from "graphql/query";
import { ListOrganization, ListProject, ListRoutine, ListStandard, ListUser } from "types";
import { validate as uuidValidate } from 'uuid';
import { APP_LINKS } from "@shared/consts";

type BaseParams = {
    itemKeyPrefix: string;
    objectType: ObjectType | undefined;
    popupTitle: string;
    popupTooltip: string;
    query: any | undefined;
    title: string;
    where: { [x: string]: any };
}

type SearchObject = ListOrganization | ListProject | ListRoutine | ListStandard | ListUser;

const tabOptions: [string, ObjectType][] = [
    ['Organizations', ObjectType.Organization],
    ['Projects', ObjectType.Project],
    ['Routines', ObjectType.Routine],
    ['Standards', ObjectType.Standard],
    ['Users', ObjectType.User],
];

/**
 * Maps object types to popup titles and tooltips.
 */
const popupMap = {
    [ObjectType.Organization]: ['Invite', `Can't find who you're looking for? Invite them!😊`],
    [ObjectType.Project]: ['Add', `Can't find what you're looking for? Create it!😎`],
    [ObjectType.Routine]: ['Add', `Can't find what you're looking for? Create it!😎`],
    [ObjectType.Standard]: ['Add', `Can't find what you're looking for? Create it!😎`],
    [ObjectType.User]: ['Invite', `Can't find who you're looking for? Invite them!😊`],
}

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

/**
 * Maps object types to wheres (additional queries for search)
 */
const whereMap = {
    [ObjectType.Project]: { isComplete: true },
    [ObjectType.Routine]: { isComplete: true, isInternal: false },
    [ObjectType.Standard]: { type: 'JSON' },
}

export function SearchPage({
    session,
}: SearchPageProps) {
    const [, setLocation] = useLocation();

    const handleSelected = useCallback((selected: SearchObject) => { openObject(selected, setLocation) }, [setLocation]);

    // Popup button, which opens either an add or invite dialog
    const [popupButton, setPopupButton] = useState<boolean>(false);

    const [shareDialogOpen, setShareDialogOpen] = useState(false);
    const closeShareDialog = useCallback(() => setShareDialogOpen(false), []);

    // Handle tabs
    const [tabIndex, setTabIndex] = useState<number>(() => {
        const searchParams = parseSearchParams(window.location.search);
        console.log('finding tab index', window.location.search, searchParams)
        const availableTypes: string[] = tabOptions.map(t => t[1]);
        const objectType: ObjectType | undefined = availableTypes.includes(searchParams.type as string) ? searchParams.type as ObjectType : undefined;
        const index = tabOptions.findIndex(t => t[1] === objectType);
        return Math.max(0, index);
    });
    const handleTabChange = (_e, newIndex: number) => { setTabIndex(newIndex) };

    // On tab change, update BaseParams, document title, where, and URL
    const { itemKeyPrefix, objectType, popupTitle, popupTooltip, query, title, where } = useMemo<BaseParams>(() => {
        // Close dialogs if they're open
        setPopupButton(false);
        setShareDialogOpen(false);
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
        const popupTitle = popupMap[objectType][0];
        const popupTooltip = popupMap[objectType][1];
        const query = queryMap[objectType];
        const title = objectType ? titleMap[objectType] : 'Search';
        const where = whereMap[objectType];
        return { itemKeyPrefix, objectType, popupTitle, popupTooltip, query, title, where };
    }, [setLocation, tabIndex]);

    const onAddClick = useCallback(() => {
        const loggedIn = session?.isLoggedIn === true && uuidValidate(session?.id ?? '');
        const objectType: ObjectType = tabOptions[tabIndex][1];
        const addUrl = `${getObjectUrlBase({ __typename: objectType })}/add`
        if (loggedIn) {
            setLocation(addUrl)
        }
        else {
            PubSub.get().publishSnack({ message: 'Must be logged in.', severity: 'error' });
            setLocation(`${APP_LINKS.Start}${stringifySearchParams({
                redirect: addUrl
            })}`);
        }
    }, [session?.id, session?.isLoggedIn, setLocation, tabIndex]);

    const onPopupButtonClick = useCallback(() => {
        const objectType = tabOptions[tabIndex][1];
        if (objectType === ObjectType.Organization || objectType === ObjectType.User) {
            setShareDialogOpen(true);
        } else {
            onAddClick();
        }
    }, [onAddClick, tabIndex])

    const handleScrolledFar = useCallback(() => { setPopupButton(true) }, [])
    const popupButtonContainer = useMemo(() => (
        <Box sx={{ ...centeredDiv, paddingTop: 1 }}>
            <Tooltip title={popupTooltip}>
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
                    {popupTitle}
                </Button>
            </Tooltip>
        </Box>
    ), [onPopupButtonClick, popupButton, popupTitle, popupTooltip]);

    return (
        <Box id='page' sx={{
            padding: '0.5em',
            paddingTop: { xs: '64px', md: '80px' },
        }}>
            {/* Invite dialog for organizations and users */}
            <ShareDialog
                onClose={closeShareDialog}
                open={shareDialogOpen}
                zIndex={200}
            />
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
                <Tooltip title="Add new" placement="top">
                    <IconButton
                        size="large"
                        onClick={onAddClick}
                        sx={{
                            padding: 1,
                        }}
                    >
                        <AddIcon color="secondary" sx={{ width: '1.5em', height: '1.5em' }} />
                    </IconButton>
                </Tooltip>
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
                where={where}
            />}
            {popupButtonContainer}
        </Box >
    )
}