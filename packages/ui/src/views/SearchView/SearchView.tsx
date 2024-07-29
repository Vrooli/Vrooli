import { GqlModelType, LINKS, ListObject, SearchType, getObjectUrlBase } from "@local/shared";
import { useTheme } from "@mui/material";
import { PageTabs } from "components/PageTabs/PageTabs";
import { SideActionsButtons } from "components/buttons/SideActionsButtons/SideActionsButtons";
import { SearchList } from "components/lists/SearchList/SearchList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { SessionContext } from "contexts/SessionContext";
import { useFindMany } from "hooks/useFindMany";
import { useTabs } from "hooks/useTabs";
import { AddIcon, SearchIcon } from "icons";
import { useCallback, useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { SideActionsButton } from "styles";
import { getCurrentUser } from "utils/authentication/session";
import { scrollIntoFocusedView } from "utils/display/scroll";
import { PubSub } from "utils/pubsub";
import { searchViewTabParams } from "utils/search/objectToSearch";
import { SearchViewProps } from "views/types";

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
    } = useTabs({ id: "search-tabs", tabParams: searchViewTabParams, display });

    const findManyData = useFindMany<ListObject>({
        controlsUrl: display === "page",
        searchType,
        take: 20,
        where: where(),
    });

    const onCreateStart = useCallback(function onCreateStartCallback() {
        // If tab is 'All', go to "Create" page
        if (searchType === SearchType.Popular) {
            setLocation(LINKS.Create);
            return;
        }
        const addUrl = `${getObjectUrlBase({ __typename: searchType as `${GqlModelType}` })}/add`;
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
            <TopBar
                display={display}
                onClose={onClose}
                title={t("Search")}
                titleBehaviorDesktop="ShowIn"
                below={<PageTabs
                    ariaLabel="search-tabs"
                    fullWidth
                    id="search-tabs"
                    ignoreIcons
                    currTab={currTab}
                    onChange={handleTabChange}
                    tabs={tabs}
                />}
            />
            {searchType && <SearchList
                {...findManyData}
                id="main-search-page-list"
                display={display}
                sxs={searchListStyle}
            />}
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
