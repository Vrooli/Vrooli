/**
 * Search page for organizations, projects, routines, standards, and users
 */
import { Box, Button, IconButton, Stack, Tab, Tabs, Tooltip, Typography, useTheme } from "@mui/material";
import { PageContainer, SearchList, ShareSiteDialog, SnackSeverity } from "components";
import { useCallback, useMemo, useState } from "react";
import { centeredDiv } from "styles";
import { useLocation } from '@shared/route';
import { SearchPageProps } from "../types";
import { getObjectUrlBase, PubSub, parseSearchParams, stringifySearchParams, SearchType, SearchPageTabOption as TabOption, addSearchParams, getUserLanguages } from "utils";
import { APP_LINKS, GqlModelType } from "@shared/consts";
import { AddIcon } from "@shared/icons";
import { getCurrentUser } from "utils/authentication";
import { CommonKey } from "types";
import { useTranslation } from "react-i18next";

// Tab data type
type BaseParams = {
    popupTitleKey: CommonKey;
    popupTooltipKey: CommonKey;
    searchType: SearchType;
    where: { [x: string]: any };
}

// Data for each tab
const tabParams: { [key in TabOption]: BaseParams } = {
    [TabOption.Apis]: {
        popupTitleKey: 'Add',
        popupTooltipKey: 'AddTooltip',
        searchType: SearchType.Api,
        where: {},
    },
    [TabOption.Notes]: {
        popupTitleKey: 'Add',
        popupTooltipKey: 'AddTooltip',
        searchType: SearchType.Note,
        where: {},
    },
    [TabOption.Organizations]: {
        popupTitleKey: 'Invite',
        popupTooltipKey: 'InviteTooltip',
        searchType: SearchType.Organization,
        where: {},
    },
    [TabOption.Projects]: {
        popupTitleKey: 'Add',
        popupTooltipKey: 'AddTooltip',
        searchType: SearchType.Project,
        where: {},
    },
    [TabOption.Questions]: {
        popupTitleKey: 'Invite',
        popupTooltipKey: 'InviteTooltip',
        searchType: SearchType.Question,
        where: {},
    },
    [TabOption.Routines]: {
        popupTitleKey: 'Add',
        popupTooltipKey: 'AddTooltip',
        searchType: SearchType.Routine,
        where: { isInternal: false },
    },
    [TabOption.SmartContracts]: {
        popupTitleKey: 'Invite',
        popupTooltipKey: 'InviteTooltip',
        searchType: SearchType.SmartContract,
        where: {},
    },
    [TabOption.Standards]: {
        popupTitleKey: 'Add',
        popupTooltipKey: 'AddTooltip',
        searchType: SearchType.Standard,
        where: {},
    },
    [TabOption.Users]: {
        popupTitleKey: 'Invite',
        popupTooltipKey: 'InviteTooltip',
        searchType: SearchType.User,
        where: {},
    },
}

const tabOptions: [SearchType, TabOption][] = Object.entries(tabParams).map(([key, value]) => [value.searchType, key as TabOption]);

export function SearchPage({
    session,
}: SearchPageProps) {
    const [, setLocation] = useLocation();
    const { palette } = useTheme();
    const { t } = useTranslation();
    const lng = useMemo(() => getUserLanguages(session)[0], [session]);

    // Popup button, which opens either an add or invite dialog
    const [popupButton, setPopupButton] = useState<boolean>(false);

    const [shareDialogOpen, setShareDialogOpen] = useState(false);
    const closeShareDialog = useCallback(() => setShareDialogOpen(false), []);

    // Handle tabs
    const [tabIndex, setTabIndex] = useState<number>(() => {
        const searchParams = parseSearchParams();
        const availableTypes: TabOption[] = tabOptions.map(t => t[1]);
        const index = availableTypes.indexOf(searchParams.type as TabOption);
        // Return valid index, or default to Routines
        return index < 0 ? 2 : index;
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
    const { popupTitleKey, popupTooltipKey, searchType, where } = useMemo<BaseParams>(() => {
        // Update tab title
        document.title = t(`common:Search${tabOptions[tabIndex][0]}`, { lng });
        // Get object type
        const searchType: TabOption = tabOptions[tabIndex][1];
        // Return base params
        return tabParams[searchType]
    }, [lng, t, tabIndex]);

    const onAddClick = useCallback((ev: any) => {
        const addUrl = `${getObjectUrlBase({ __typename: searchType as `${GqlModelType}` })}/add`
        // If not logged in, redirect to login page
        if (!getCurrentUser(session).id) {
            PubSub.get().publishSnack({ messageKey: 'MustBeLoggedIn', severity: SnackSeverity.Error });
            setLocation(`${APP_LINKS.Start}${stringifySearchParams({
                redirect: addUrl
            })}`);
            return;
        }
        // If search type is a routine, open create routine page
        if (searchType === SearchType.Routine) {
            setLocation(`${APP_LINKS.Routine}/add`);
        }
        // If search type is a user, open start page
        else if (searchType === SearchType.User) {
            setLocation(`${APP_LINKS.Start}`);
        }
        // Otherwise, navigate to add page
        else {
            setLocation(addUrl)
        }
    }, [searchType, session, setLocation]);

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
            <Tooltip title={t(`common:${popupTooltipKey}`, { lng })}>
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
                    {t(`common:${popupTitleKey}`, { lng })}
                </Button>
            </Tooltip>
        </Box>
    ), [lng, onPopupButtonClick, popupButton, popupTitleKey, popupTooltipKey, t]);

    return (
        <PageContainer>
            {/* Invite dialog for organizations and users */}
            <ShareSiteDialog
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
                        paddingLeft: '1em',
                        paddingRight: '1em',
                    }}
                >
                    {tabOptions.map((option, index) => (
                        <Tab
                            key={index}
                            id={`search-tab-${index}`}
                            {...{ 'aria-controls': `search-tabpanel-${index}` }}
                            label={t(`common:${option[0]}`, { lng })}
                            color={index === 0 ? '#ce6c12' : 'default'}
                            component="a"
                            href={option[1]}
                        />
                    ))}
                </Tabs>
            </Box>
            <Stack direction="row" alignItems="center" justifyContent="center" sx={{ paddingTop: 2 }}>
                <Typography component="h2" variant="h4">{t(`common:${searchType}`, { lng })}</Typography>
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
                itemKeyPrefix={`${searchType}-list-item`}
                searchPlaceholder={'Search...'}
                take={20}
                searchType={searchType}
                onScrolledFar={handleScrolledFar}
                session={session}
                zIndex={200}
                where={where}
            />}
            {popupButtonContainer}
        </PageContainer>
    )
}