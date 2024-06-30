import { GqlModelType, LINKS, ListObject, getObjectUrlBase } from "@local/shared";
import { IconButton, useTheme } from "@mui/material";
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
import { getCurrentUser } from "utils/authentication/session";
import { scrollIntoFocusedView } from "utils/display/scroll";
import { PubSub } from "utils/pubsub";
import { searchVersionViewTabParams } from "utils/search/objectToSearch";
import { SearchVersionViewProps } from "../types";

/**
 * Uncommon search page for versioned objects
 */
export function SearchVersionView({
    display,
    onClose,
}: SearchVersionViewProps) {
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
    } = useTabs({ id: "search-version-tabs", tabParams: searchVersionViewTabParams, display });

    const findManyData = useFindMany<ListObject>({
        controlsUrl: display === "page",
        searchType,
        take: 20,
        where: where(),
    });

    const onCreateStart = useCallback(function onCreateStartCallback(e: React.MouseEvent<HTMLElement>) {
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

    function focusSearch() { scrollIntoFocusedView("search-bar-version-search-page-list"); }

    return (
        <>
            <TopBar
                display={display}
                hideTitleOnDesktop={true}
                onClose={onClose}
                title={t("SearchVersions")}
                below={<PageTabs
                    ariaLabel="search-version-tabs"
                    fullWidth
                    id="search-version-tabs"
                    ignoreIcons
                    currTab={currTab}
                    onChange={handleTabChange}
                    tabs={tabs}
                />}
            />
            {searchType && <SearchList
                {...findManyData}
                id="version-search-page-list"
                display={display}
                sxs={{ search: { marginTop: 2 } }}
            />}
            <SideActionsButtons display={display}>
                <IconButton aria-label={t("FilterList")} onClick={focusSearch} sx={{ background: palette.secondary.main }}>
                    <SearchIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                </IconButton>
                {userId ? (
                    <IconButton aria-label={t("Add")} onClick={onCreateStart} sx={{ background: palette.secondary.main }}>
                        <AddIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                    </IconButton>
                ) : null}
            </SideActionsButtons>
        </>
    );
}
