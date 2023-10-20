import { GqlModelType } from "@local/shared";
import { IconButton, useTheme } from "@mui/material";
import { SideActionsButtons } from "components/buttons/SideActionsButtons/SideActionsButtons";
import { SearchList } from "components/lists/SearchList/SearchList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { PageTabs } from "components/PageTabs/PageTabs";
import { useTabs } from "hooks/useTabs";
import { AddIcon } from "icons";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { toDisplay } from "utils/display/pageTools";
import { getObjectUrlBase } from "utils/navigation/openObject";
import { HistoryPageTabOption, historyTabParams, SearchType } from "utils/search/objectToSearch";
import { HistoryViewProps } from "../types";

export const HistoryView = ({
    isOpen,
    onClose,
}: HistoryViewProps) => {
    const display = toDisplay(isOpen);
    const { t } = useTranslation();
    const { palette } = useTheme();
    const [, setLocation] = useLocation();

    const {
        currTab,
        handleTabChange,
        searchType,
        tabs,
        where,
    } = useTabs<HistoryPageTabOption>({ id: "history-tabs", tabParams: historyTabParams, display });

    return (
        <>
            <TopBar
                display={display}
                hideTitleOnDesktop={true}
                onClose={onClose}
                title={currTab.label}
                below={<PageTabs
                    ariaLabel="history-tabs"
                    currTab={currTab}
                    fullWidth
                    onChange={handleTabChange}
                    tabs={tabs}
                />}
            />
            {searchType && <SearchList
                id="history-page-list"
                display={display}
                dummyLength={display === "page" ? 5 : 3}
                take={20}
                searchType={searchType}
                sxs={{
                    search: {
                        marginTop: 2,
                    },
                }}
                where={where()}
            />}
            {searchType === SearchType.BookmarkList && <SideActionsButtons
                display={display}
                sx={{ position: "fixed" }} // TODO for morning: using "fixed" needed to show at bottom, but it doesn't stay within content-wrap div. Maybe use "sticky" instead? But it doesn't show at the bottom if the content is too short.
            // sx={{
            //    position: "sticky",
            //    justifyContent: "flex-end",
            // }}
            >
                <IconButton
                    aria-label={t("Add")}
                    onClick={() => { setLocation(`${getObjectUrlBase({ __typename: GqlModelType.BookmarkList })}/add`); }}
                    sx={{ background: palette.secondary.main }}
                >
                    <AddIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                </IconButton>
            </SideActionsButtons>}
        </>
    );
};
