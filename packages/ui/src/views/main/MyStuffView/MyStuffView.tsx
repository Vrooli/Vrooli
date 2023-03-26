/**
 * Search page for organizations, projects, routines, standards, and users
 */
import { Box, Button, IconButton, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { GqlModelType, LINKS } from "@shared/consts";
import { AddIcon, ApiIcon, HelpIcon, NoteIcon, OrganizationIcon, ProjectIcon, RoutineIcon, SmartContractIcon, StandardIcon, SvgProps } from "@shared/icons";
import { addSearchParams, parseSearchParams, useLocation } from '@shared/route';
import { CommonKey } from "@shared/translations";
import { SearchList } from "components/lists/SearchList/SearchList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { PageTabs } from "components/PageTabs/PageTabs";
import { PageTab } from "components/types";
import { useCallback, useContext, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { centeredDiv } from "styles";
import { getCurrentUser } from "utils/authentication/session";
import { getObjectUrlBase } from "utils/navigation/openObject";
import { PubSub } from "utils/pubsub";
import { SearchPageTabOption, SearchType } from "utils/search/objectToSearch";
import { SessionContext } from "utils/SessionContext";
import { MyStuffViewProps } from "../types";

// Tab data type
type BaseParams = {
    Icon: (props: SvgProps) => JSX.Element,
    searchType: SearchType;
    tabType: SearchPageTabOption;
    where: (userId: string) => { [x: string]: any };
}

// Data for each tab
const tabParams: BaseParams[] = [{
    Icon: RoutineIcon,
    searchType: SearchType.Routine,
    tabType: SearchPageTabOption.Routines,
    where: (userId) => ({ isInternal: false, ownedByUserId: userId }),
}, {
    Icon: ProjectIcon,
    searchType: SearchType.Project,
    tabType: SearchPageTabOption.Projects,
    where: (userId) => ({ ownedByUserId: userId }),
}, {
    Icon: HelpIcon,
    searchType: SearchType.Question,
    tabType: SearchPageTabOption.Questions,
    where: (userId) => ({ createdById: userId }),
}, {
    Icon: NoteIcon,
    searchType: SearchType.Note,
    tabType: SearchPageTabOption.Notes,
    where: (userId) => ({ ownedByUserId: userId }),
}, {
    Icon: OrganizationIcon,
    searchType: SearchType.Organization,
    tabType: SearchPageTabOption.Organizations,
    where: (userId) => ({ memberUserIds: [userId] }),
}, {
    Icon: StandardIcon,
    searchType: SearchType.Standard,
    tabType: SearchPageTabOption.Standards,
    where: (userId) => ({ ownedByUserId: userId }),
}, {
    Icon: ApiIcon,
    searchType: SearchType.Api,
    tabType: SearchPageTabOption.Apis,
    where: (userId) => ({ ownedByUserId: userId }),
}, {
    Icon: SmartContractIcon,
    searchType: SearchType.SmartContract,
    tabType: SearchPageTabOption.SmartContracts,
    where: (userId) => ({ ownedByUserId: userId }),
}];

export const MyStuffView = ({
    display = 'page',
}: MyStuffViewProps) => {
    const session = useContext(SessionContext);
    const [, setLocation] = useLocation();
    const { palette } = useTheme();
    const { t } = useTranslation();
    const { id: userId } = useMemo(() => getCurrentUser(session), [session]);

    // Popup button, which opens either an add or invite dialog
    const [popupButton, setPopupButton] = useState<boolean>(false);

    // Handle tabs
    const tabs = useMemo<PageTab<SearchPageTabOption>[]>(() => {
        return tabParams.map((tab, i) => ({
            index: i,
            Icon: tab.Icon,
            label: t(tab.searchType, { count: 2, defaultValue: tab.searchType }),
            value: tab.tabType,
        }));
    }, [t]);
    const [currTab, setCurrTab] = useState<PageTab<SearchPageTabOption>>(() => {
        const searchParams = parseSearchParams();
        const index = tabParams.findIndex(tab => tab.tabType === searchParams.type);
        // Default to routine tab
        if (index === -1) return tabs[0];
        // Return tab
        return tabs[index];
    });
    const handleTabChange = useCallback((e: any, tab: PageTab<SearchPageTabOption>) => {
        e.preventDefault();
        // Update search params
        addSearchParams(setLocation, { type: tab.value });
        // Update curr tab
        setCurrTab(tab)
    }, [setLocation]);

    // On tab change, update BaseParams, document title, where, and URL
    const { searchType, where } = useMemo<Omit<BaseParams, 'where'> & { where: { [x: string]: any } }>(() => {
        // Update tab title
        document.title = `${t(`Search`)} | ${currTab.label}`;
        const params = tabParams[currTab.index];
        return {
            ...params,
            where: params.where(userId!),
        }
    }, [currTab.index, currTab.label, t, userId]);

    const onAddClick = useCallback((ev: any) => {
        const addUrl = `${getObjectUrlBase({ __typename: searchType as `${GqlModelType}` })}/add`
        // If not logged in, redirect to login page
        if (!userId) {
            PubSub.get().publishSnack({ messageKey: 'MustBeLoggedIn', severity: 'Error' });
            setLocation(LINKS.Start, { searchParams: { redirect: addUrl } });
            return;
        }
        // If search type is a routine, open create routine page
        if (searchType === SearchType.Routine) {
            setLocation(`${LINKS.Routine}/add`);
        }
        // If search type is a user, open start page
        else if (searchType === SearchType.User) {
            setLocation(`${LINKS.Start}`);
        }
        // Otherwise, navigate to add page
        else {
            setLocation(addUrl)
        }
    }, [searchType, setLocation, userId]);

    const handleScrolledFar = useCallback(() => { setPopupButton(true) }, [])
    const popupButtonContainer = useMemo(() => (
        <Box sx={{ ...centeredDiv, paddingTop: 1 }}>
            <Tooltip title={t('AddTooltip')}>
                <Button
                    onClick={onAddClick}
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
                    {t('Add')}
                </Button>
            </Tooltip>
        </Box>
    ), [onAddClick, popupButton, t]);

    console.log('search typeeee', searchType, currTab)

    return (
        <>
            <TopBar
                display={display}
                onClose={() => { }}
                titleData={{
                    hideOnDesktop: true,
                    titleKey: 'MyStuff',
                }}
                below={<PageTabs
                    ariaLabel="search-tabs"
                    currTab={currTab}
                    onChange={handleTabChange}
                    tabs={tabs}
                />}
            />
            <Stack direction="row" alignItems="center" justifyContent="center" sx={{ paddingTop: 2 }}>
                <Typography component="h2" variant="h4">{t(searchType as CommonKey, { count: 2, defaultValue: searchType })}</Typography>
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
                zIndex={200}
                where={where}
            />}
            {popupButtonContainer}
        </>
    )
}