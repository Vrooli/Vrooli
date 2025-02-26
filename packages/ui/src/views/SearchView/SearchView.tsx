import { LINKS, ListObject, ModelType, getObjectUrlBase } from "@local/shared";
import { useTheme } from "@mui/material";
import { PageTabs } from "components/PageTabs/PageTabs";
import { SideActionsButtons } from "components/buttons/SideActionsButtons/SideActionsButtons";
import { SearchList, SearchListScrollContainer } from "components/lists/SearchList/SearchList";
import { TopBar } from "components/navigation/TopBar/TopBar.js";
import { SessionContext } from "contexts";
import { useFindMany } from "hooks/useFindMany";
import { useTabs } from "hooks/useTabs";
import { AddIcon, SearchIcon } from "icons";
import { useCallback, useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { SideActionsButton } from "styles";
import { getCurrentUser } from "utils/authentication/session.js";
import { ELEMENT_IDS } from "utils/consts.js";
import { scrollIntoFocusedView } from "utils/display/scroll";
import { PubSub } from "utils/pubsub.js";
import { searchViewTabParams } from "utils/search/objectToSearch";
import { SearchViewProps } from "views/types";

const scrollContainerId = "main-search-scroll";

const searchListStyle = { search: { marginTop: 2 } } as const;

/**
 * Search page for teams, projects, routines, standards, users, and other main objects
 */
export function SearchView({
    display,
    onClose,
}: SearchViewProps) {
    const session = useContext(SessionContext);
    const [, setLocation] = useLocation();
    const { palette } = useTheme();
    const { t } = useTranslation();
    const { id: userId } = useMemo(() => getCurrentUser(session), [session]);

    const {
        currTab,
        handleTabChange,
        searchType,
        tabs,
        where,
    } = useTabs({ id: ELEMENT_IDS.SearchTabs, tabParams: searchViewTabParams, display });

    const findManyData = useFindMany<ListObject>({
        controlsUrl: display === "page",
        searchType,
        take: 20,
        where: where(undefined),
    });

    const onCreateStart = useCallback(function onCreateStartCallback() {
        // If tab is 'All', go to "Create" page
        if (searchType === "Popular") {
            setLocation(LINKS.Create);
            return;
        }
        const addUrl = `${getObjectUrlBase({ __typename: searchType as `${ModelType}` })}/add`;
        // If not logged in, redirect to login page
        if (!userId) {
            PubSub.get().publish("snack", { messageKey: "NotLoggedIn", severity: "Error" });
            setLocation(LINKS.Login, { searchParams: { redirect: addUrl } });
            return;
        }
        // Otherwise, navigate to object's add page
        else setLocation(addUrl);
    }, [searchType, setLocation, userId]);

    function focusSearch() { scrollIntoFocusedView("search-bar-main-search-page-list"); }

    return (
        <>
            <SearchListScrollContainer id={scrollContainerId}>
                <TopBar
                    display={display}
                    onClose={onClose}
                    title={t("Search")}
                    titleBehaviorDesktop="ShowIn"
                    below={<PageTabs<typeof searchViewTabParams>
                        ariaLabel="Search tabs"
                        fullWidth
                        id={ELEMENT_IDS.SearchTabs}
                        ignoreIcons
                        currTab={currTab}
                        onChange={handleTabChange}
                        tabs={tabs}
                    />}
                />
                {searchType && <SearchList
                    {...findManyData}
                    display={display}
                    scrollContainerId={scrollContainerId}
                    sxs={searchListStyle}
                />}
            </SearchListScrollContainer>
            <SideActionsButtons display={display}>
                <SideActionsButton aria-label={t("FilterList")} onClick={focusSearch}>
                    <SearchIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                </SideActionsButton>
                {userId ? (
                    <SideActionsButton aria-label={t("Add")} onClick={onCreateStart}>
                        <AddIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                    </SideActionsButton>
                ) : null}
            </SideActionsButtons>
        </>
    );
}
