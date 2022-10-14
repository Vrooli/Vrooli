/**
 * Search page for organizations, projects, routines, standards, and users
 */
import { Box, Button, IconButton, Stack, Tab, Tabs, Tooltip, Typography, useTheme } from "@mui/material";
import { PageContainer, SearchList, SelectRoutineTypeMenu, ShareSiteDialog, SnackSeverity } from "components";
import { useCallback, useMemo, useState } from "react";
import { centeredDiv } from "styles";
import { useLocation } from '@shared/route';
import { SearchPageProps } from "../types";
import { getObjectUrlBase, PubSub, parseSearchParams, stringifySearchParams, openObject, SearchType, SearchPageTabOption as TabOption, addSearchParams } from "utils";
import { ListOrganization, ListProject, ListRoutine, ListStandard, ListUser } from "types";
import { APP_LINKS } from "@shared/consts";
import { AddIcon } from "@shared/icons";
import { getCurrentUser } from "utils/authentication";

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
        where: {},
    },
    [TabOption.Projects]: {
        itemKeyPrefix: 'project-list-item',
        popupTitle: 'Add',
        popupTooltip: `Can't find what you're looking for? Create it!😎`,
        searchType: SearchType.Project,
        title: 'Projects',
        where: {},
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
        where: {},
    },
    [TabOption.Users]: {
        itemKeyPrefix: 'user-list-item',
        popupTitle: 'Invite',
        popupTooltip: `Can't find who you're looking for? Invite them!😊`,
        searchType: SearchType.User,
        title: 'Users',
        where: {},
    },
}

// [title, searchType] for each tab
const tabOptions: [string, TabOption][] = Object.entries(tabParams).map(([key, value]) => [value.title, key as TabOption]);

type SearchObject = ListOrganization | ListProject | ListRoutine | ListStandard | ListUser;

export function SearchPage({
    session,
}: SearchPageProps) {
    const [, setLocation] = useLocation();
    const { palette } = useTheme();

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
        // Return valid index, or default to Routines
        return index < 0 ? 2 : index;
    });
    const handleTabChange = (_e, newIndex: number) => {
        // Update search params
        addSearchParams(setLocation, {
            type: tabOptions[newIndex][1],
        });
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

    // Menu for picking which routine type to add
    const [addRoutineAnchor, setAddRoutineAnchor] = useState<any>(null);
    const openAddRoutine = useCallback((ev: React.MouseEvent<HTMLElement>) => {
        setAddRoutineAnchor(ev.currentTarget)
    }, []);
    const closeAddRoutine = useCallback(() => setAddRoutineAnchor(null), []);

    const onAddClick = useCallback((ev: any) => {
        console.log('onaddclick')
        const addUrl = `${getObjectUrlBase({ __typename: searchType as string })}/add`
        // If not logged in, redirect to login page
        if (!getCurrentUser(session).id) {
            PubSub.get().publishSnack({ message: 'Must be logged in.', severity: SnackSeverity.Error });
            setLocation(`${APP_LINKS.Start}${stringifySearchParams({
                redirect: addUrl
            })}`);
            return;
        }
        // If search type is a routine, open dialog to select routine type
        if (searchType === SearchType.Routine) {
            openAddRoutine(ev);
        }
        // If search type is a user, open start page
        else if (searchType === SearchType.User) {
            setLocation(`${APP_LINKS.Start}`);
        }
        // Otherwise, navigate to add page
        else {
            setLocation(addUrl)
        }
    }, [openAddRoutine, searchType, session, setLocation]);

    const onPopupButtonClick = useCallback((ev: any) => {
        const tabType = tabOptions[tabIndex][1];
        if (tabType === TabOption.Organizations || tabType === TabOption.Users) {
            setShareDialogOpen(true);
        } else {
            onAddClick(ev);
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
        <PageContainer>
            {/* Invite dialog for organizations and users */}
            <ShareSiteDialog
                onClose={closeShareDialog}
                open={shareDialogOpen}
                zIndex={200}
            />
            {/* Add menu for selecting between single-step and multi-step routines */}
            <SelectRoutineTypeMenu
                anchorEl={addRoutineAnchor}
                handleClose={closeAddRoutine}
                session={session}
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
                        size="medium"
                        onClick={onAddClick}
                        sx={{
                            padding: 1,
                        }}
                    >
                        <AddIcon fill={palette.secondary.main} width='1.5em' height='1.5em' />
                    </IconButton>
                </Tooltip>
            </Stack>
            {searchType && <SearchList
                id="main-search-page-list"
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
        </PageContainer>
    )
}