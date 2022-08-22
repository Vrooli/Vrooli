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
import { getObjectUrlBase, PubSub, parseSearchParams, stringifySearchParams, openObject, SearchType, SearchPageTabOption as TabOption } from "utils";
import { ListOrganization, ListProject, ListRoutine, ListStandard, ListUser } from "types";
import { validate as uuidValidate } from 'uuid';
import { APP_LINKS } from "@shared/consts";

// Tab data type
type BaseParams = {
    itemKeyPrefix: string;
    searchType: SearchType;
    popupTitle: string;
    popupTooltip: string;
    title: string;
    where: { [x: string]: any };
}

// Data for each tab
const tabParams: { [key in TabOption]: BaseParams } = {
    [TabOption.Organizations]: {
        itemKeyPrefix: 'organization-list-item',
        popupTitle: 'Invite',
        popupTooltip: `Can't find who you're looking for? Invite them!😊`,
        searchType: SearchType.Organization,
        title: 'Organizations',
        where: { },
    },
    [TabOption.Projects]: {
        itemKeyPrefix: 'project-list-item',
        popupTitle: 'Add',
        popupTooltip: `Can't find what you're looking for? Create it!😎`,
        searchType: SearchType.Project,
        title: 'Projects',
        where: { },
    },
    [TabOption.Routines]: {
        itemKeyPrefix: 'routine-list-item',
        popupTitle: 'Add',
        popupTooltip: `Can't find what you're looking for? Create it!😎`,
        searchType: SearchType.Routine,
        title: 'Routines',
        where: { isInternal: false },
    },
    [TabOption.Standards]: {
        itemKeyPrefix: 'standard-list-item',
        popupTitle: 'Add',
        popupTooltip: `Can't find what you're looking for? Create it!😎`,
        searchType: SearchType.Standard,
        title: 'Standards',
        where: { },
    },
    [TabOption.Users]: {
        itemKeyPrefix: 'user-list-item',
        popupTitle: 'Invite',
        popupTooltip: `Can't find who you're looking for? Invite them!😊`,
        searchType: SearchType.User,
        title: 'Users',
        where: { },
    },
}

// [title, searchType] for each tab
const tabOptions: [string, TabOption][] = Object.entries(tabParams).map(([key, value]) => [value.title, key as TabOption]);

type SearchObject = ListOrganization | ListProject | ListRoutine | ListStandard | ListUser;

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
        const availableTypes: TabOption[] = tabOptions.map(t => t[1]);
        const index = availableTypes.indexOf(searchParams.type as TabOption);
        console.log('getting tab index', searchParams, availableTypes, index);
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
        }), { replace: true });
        // Update tab index
        setTabIndex(newIndex) 
    };

    // On tab change, update BaseParams, document title, where, and URL
    const { itemKeyPrefix, popupTitle, popupTooltip, searchType, title, where } = useMemo<BaseParams>(() => {
        // Update tab title
        document.title = `Search ${tabOptions[tabIndex][0]}`;
        // Get object type
        const searchType: TabOption = tabOptions[tabIndex][1];
        // Return base params
        return tabParams[searchType]
    }, [tabIndex]);

    const onAddClick = useCallback(() => {
        const loggedIn = session?.isLoggedIn === true && uuidValidate(session?.id ?? '');
        const tabType: TabOption = tabOptions[tabIndex][1];
        const addUrl = `${getObjectUrlBase({ __typename: tabType as string })}/add`
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
        const tabType = tabOptions[tabIndex][1];
        if (tabType === TabOption.Organizations || tabType === TabOption.Users) {
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
            {searchType && <SearchList
                itemKeyPrefix={itemKeyPrefix}
                searchPlaceholder={'Search...'}
                take={20}
                searchType={searchType}
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