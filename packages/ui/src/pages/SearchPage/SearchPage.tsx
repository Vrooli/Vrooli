/**
 * Search page for organizations, projects, routines, standards, and users
 */
import { Box, Button, IconButton, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { PageContainer, PageTabs, SearchList, ShareSiteDialog } from "components";
import { useCallback, useMemo, useState } from "react";
import { centeredDiv } from "styles";
import { addSearchParams, parseSearchParams, useLocation } from '@shared/route';
import { SearchPageProps } from "../types";
import { getObjectUrlBase, PubSub, SearchType, SearchPageTabOption as TabOptions, getUserLanguages } from "utils";
import { APP_LINKS, GqlModelType } from "@shared/consts";
import { AddIcon } from "@shared/icons";
import { getCurrentUser } from "utils/authentication";
import { CommonKey } from "types";
import { useTranslation } from "react-i18next";
import { PageTab } from "components/types";

// Tab data type
type BaseParams = {
    popupTitleKey: CommonKey;
    popupTooltipKey: CommonKey;
    searchType: SearchType;
    tabType: TabOptions;
    where: { [x: string]: any };
}

// Data for each tab
const tabParams: BaseParams[] = [{
    popupTitleKey: 'Add',
    popupTooltipKey: 'AddTooltip',
    searchType: SearchType.Api,
    tabType: TabOptions.Apis,
    where: {},
}, {
    popupTitleKey: 'Add',
    popupTooltipKey: 'AddTooltip',
    searchType: SearchType.Note,
    tabType: TabOptions.Notes,
    where: {},
}, {
    popupTitleKey: 'Invite',
    popupTooltipKey: 'InviteTooltip',
    searchType: SearchType.Organization,
    tabType: TabOptions.Organizations,
    where: {},
}, {
    popupTitleKey: 'Add',
    popupTooltipKey: 'AddTooltip',
    searchType: SearchType.Project,
    tabType: TabOptions.Projects,
    where: {},
}, {
    popupTitleKey: 'Invite',
    popupTooltipKey: 'InviteTooltip',
    searchType: SearchType.Question,
    tabType: TabOptions.Questions,
    where: {},
}, {
    popupTitleKey: 'Add',
    popupTooltipKey: 'AddTooltip',
    searchType: SearchType.Routine,
    tabType: TabOptions.Routines,
    where: { isInternal: false },
}, {
    popupTitleKey: 'Invite',
    popupTooltipKey: 'InviteTooltip',
    searchType: SearchType.SmartContract,
    tabType: TabOptions.SmartContracts,
    where: {},
}, {
    popupTitleKey: 'Add',
    popupTooltipKey: 'AddTooltip',
    searchType: SearchType.Standard,
    tabType: TabOptions.Standards,
    where: {},
}, {
    popupTitleKey: 'Invite',
    popupTooltipKey: 'InviteTooltip',
    searchType: SearchType.User,
    tabType: TabOptions.Users,
    where: {},
}];

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
    const tabs = useMemo<PageTab<TabOptions>[]>(() => {
        return tabParams.map((tab, i) => ({
            index: i,
            label: t(`common:${tab.searchType}`, { lng: getUserLanguages(session)[0], count: 2 }),
            value: tab.tabType,
        }));
    }, [session, t]);
    const [currTab, setCurrTab] = useState<PageTab<TabOptions>>(() => {
        const searchParams = parseSearchParams();
        const index = tabParams.findIndex(tab => tab.tabType === searchParams.type);
        return tabs[Math.max(0, index)];
    });
    const handleTabChange = useCallback((e: any, tab: PageTab<TabOptions>) => {
        e.preventDefault();
        // Update search params
        addSearchParams(setLocation, { type: tab.value });
        // Update curr tab
        setCurrTab(tab)
    }, [setLocation]);

    // On tab change, update BaseParams, document title, where, and URL
    const { popupTitleKey, popupTooltipKey, searchType, where } = useMemo<BaseParams>(() => {
        // Update tab title
        document.title = `${t(`common:Search`, { lng })} | ${currTab.label}`;
        return tabParams[currTab.index];
    }, [currTab.index, currTab.label, lng, t]);

    const onAddClick = useCallback((ev: any) => {
        const addUrl = `${getObjectUrlBase({ __typename: searchType as `${GqlModelType}` })}/add`
        // If not logged in, redirect to login page
        if (!getCurrentUser(session).id) {
            PubSub.get().publishSnack({ messageKey: 'MustBeLoggedIn', severity: 'Error' });
            setLocation(APP_LINKS.Start, { searchParams: { redirect: addUrl } });
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
        if ([TabOptions.Organizations, TabOptions.Users].includes(currTab.value)) {
            setShareDialogOpen(true);
        } else {
            onAddClick(ev);
        }
    }, [currTab.value, onAddClick])

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
            <PageTabs
                ariaLabel="search-tabs"
                currTab={currTab}
                onChange={handleTabChange}
                tabs={tabs}
            />
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