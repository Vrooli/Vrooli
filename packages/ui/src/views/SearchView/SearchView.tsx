import { CommonKey, GqlModelType, LINKS } from "@local/shared";
import { ListItemIcon, ListItemText, Menu, MenuItem, useTheme } from "@mui/material";
import { ColorIconButton } from "components/buttons/ColorIconButton/ColorIconButton";
import { SideActionButtons } from "components/buttons/SideActionButtons/SideActionButtons";
import { SearchList } from "components/lists/SearchList/SearchList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { PageTabs } from "components/PageTabs/PageTabs";
import { AddIcon, ApiIcon, HelpIcon, NoteIcon, OrganizationIcon, ProjectIcon, RoutineIcon, SearchIcon, SmartContractIcon, StandardIcon, UserIcon, VisibleIcon } from "icons";
import { useCallback, useContext, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { getCurrentUser } from "utils/authentication/session";
import { toDisplay } from "utils/display/pageTools";
import { useTabs } from "utils/hooks/useTabs";
import { getObjectUrlBase } from "utils/navigation/openObject";
import { PubSub } from "utils/pubsub";
import { SearchPageTabOption, SearchType } from "utils/search/objectToSearch";
import { SessionContext } from "utils/SessionContext";
import { SearchViewProps } from "../types";

// Data for each tab
export const searchViewTabParams = [{
    Icon: VisibleIcon,
    titleKey: "All" as CommonKey,
    searchType: SearchType.Popular,
    tabType: SearchPageTabOption.All,
    where: () => ({}),
}, {
    Icon: RoutineIcon,
    titleKey: "Routine" as CommonKey,
    searchType: SearchType.Routine,
    tabType: SearchPageTabOption.Routine,
    where: () => ({ isInternal: false }),
}, {
    Icon: ProjectIcon,
    titleKey: "Project" as CommonKey,
    searchType: SearchType.Project,
    tabType: SearchPageTabOption.Project,
    where: () => ({}),
}, {
    Icon: HelpIcon,
    titleKey: "Question" as CommonKey,
    searchType: SearchType.Question,
    tabType: SearchPageTabOption.Question,
    where: () => ({}),
}, {
    Icon: NoteIcon,
    titleKey: "Note" as CommonKey,
    searchType: SearchType.Note,
    tabType: SearchPageTabOption.Note,
    where: () => ({}),
}, {
    Icon: OrganizationIcon,
    titleKey: "Organization" as CommonKey,
    searchType: SearchType.Organization,
    tabType: SearchPageTabOption.Organization,
    where: () => ({}),
}, {
    Icon: UserIcon,
    titleKey: "User" as CommonKey,
    searchType: SearchType.User,
    tabType: SearchPageTabOption.User,
    where: () => ({}),
}, {
    Icon: StandardIcon,
    titleKey: "Standard" as CommonKey,
    searchType: SearchType.Standard,
    tabType: SearchPageTabOption.Standard,
    where: () => ({ isInternal: false, type: "JSON" }),
}, {
    Icon: ApiIcon,
    titleKey: "Api" as CommonKey,
    searchType: SearchType.Api,
    tabType: SearchPageTabOption.Api,
    where: () => ({}),
}, {
    Icon: SmartContractIcon,
    titleKey: "SmartContract" as CommonKey,
    searchType: SearchType.SmartContract,
    tabType: SearchPageTabOption.SmartContract,
    where: () => ({}),
}];

/**
 * Search page for organizations, projects, routines, standards, users, and other main objects
 */
export const SearchView = ({
    isOpen,
    onClose,
    zIndex,
}: SearchViewProps) => {
    const session = useContext(SessionContext);
    const [, setLocation] = useLocation();
    const { palette } = useTheme();
    const { t } = useTranslation();
    const display = toDisplay(isOpen);
    const { id: userId } = useMemo(() => getCurrentUser(session), [session]);

    const {
        currTab,
        handleTabChange,
        searchType,
        tabs,
        where,
    } = useTabs<SearchPageTabOption>({ tabParams: searchViewTabParams, display });

    // Menu for selection object type to create
    const [selectCreateTypeAnchorEl, setSelectCreateTypeAnchorEl] = useState<null | HTMLElement>(null);

    const onCreateStart = useCallback((e: React.MouseEvent<HTMLElement>) => {
        // If tab is 'All', open menu to select type
        if (searchType === SearchType.Popular) {
            setSelectCreateTypeAnchorEl(e.currentTarget);
            return;
        }
        const addUrl = `${getObjectUrlBase({ __typename: searchType as `${GqlModelType}` })}/add`;
        // If not logged in, redirect to login page
        if (!userId) {
            PubSub.get().publishSnack({ messageKey: "MustBeLoggedIn", severity: "Error" });
            setLocation(LINKS.Start, { searchParams: { redirect: addUrl } });
            return;
        }
        // Otherwise, navigate to object's add page
        else setLocation(addUrl);
    }, [searchType, setLocation, userId]);
    const onSelectCreateTypeClose = useCallback((type?: SearchType) => {
        if (type) setLocation(`${getObjectUrlBase({ __typename: type as `${GqlModelType}` })}/add`);
        else setSelectCreateTypeAnchorEl(null);
    }, [setLocation]);

    const focusSearch = useCallback(() => {
        const searchInput = document.getElementById("search-bar-main-search-page-list");
        searchInput?.focus();
    }, []);

    return (
        <>
            <Menu
                id="select-create-type-menu"
                anchorEl={selectCreateTypeAnchorEl}
                disableScrollLock={true}
                open={Boolean(selectCreateTypeAnchorEl)}
                onClose={() => onSelectCreateTypeClose()}
            >
                {/* Never show 'All' */}
                {searchViewTabParams.filter((t) => ![SearchType.Popular].includes(t.searchType)).map(tab => (
                    <MenuItem
                        key={tab.searchType}
                        onClick={() => onSelectCreateTypeClose(tab.searchType as SearchType)}
                    >
                        <ListItemIcon>
                            <tab.Icon fill={palette.background.textPrimary} />
                        </ListItemIcon>
                        <ListItemText primary={t(tab.searchType, { count: 1, defaultValue: tab.searchType })} />
                    </MenuItem>
                ))}
            </Menu>
            <TopBar
                display={display}
                hideTitleOnDesktop={true}
                onClose={onClose}
                title={t("Search")}
                below={<PageTabs
                    ariaLabel="search-tabs"
                    fullWidth
                    id="search-tabs"
                    ignoreIcons
                    currTab={currTab}
                    onChange={handleTabChange}
                    tabs={tabs}
                />}
                zIndex={zIndex}
            />
            {searchType && <SearchList
                id="main-search-page-list"
                display={display}
                dummyLength={display === "page" ? 5 : 3}
                take={20}
                searchType={searchType}
                zIndex={zIndex}
                where={where()}
                sxs={{ search: { marginTop: 2 } }}
            />}
            <SideActionButtons
                display={display}
                zIndex={zIndex + 2}
                sx={{ position: "fixed" }}
            >
                <ColorIconButton aria-label="filter-list" background={palette.secondary.main} onClick={focusSearch} >
                    <SearchIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                </ColorIconButton>
                {userId ? (
                    <ColorIconButton aria-label="edit-routine" background={palette.secondary.main} onClick={onCreateStart} >
                        <AddIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                    </ColorIconButton>
                ) : null}
            </SideActionButtons>
        </>
    );
};
