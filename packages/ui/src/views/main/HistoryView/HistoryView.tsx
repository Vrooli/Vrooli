import { CommonKey, RunStatus } from "@local/shared";
import { SearchList } from "components/lists/SearchList/SearchList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { PageTabs } from "components/PageTabs/PageTabs";
import { toDisplay } from "utils/display/pageTools";
import { useTabs } from "utils/hooks/useTabs";
import { HistoryPageTabOption, SearchType } from "utils/search/objectToSearch";
import { HistoryViewProps } from "../types";

const tabParams = [{
    titleKey: "View" as CommonKey,
    searchType: SearchType.View,
    tabType: HistoryPageTabOption.Viewed,
    where: () => ({}),
}, {
    titleKey: "Bookmark" as CommonKey,
    searchType: SearchType.BookmarkList,
    tabType: HistoryPageTabOption.Bookmarked,
    where: () => ({}),
}, {
    titleKey: "Active" as CommonKey,
    searchType: SearchType.RunProjectOrRunRoutine,
    tabType: HistoryPageTabOption.RunsActive,
    where: () => ({ statuses: [RunStatus.InProgress, RunStatus.Scheduled] }),
}, {
    titleKey: "Complete" as CommonKey,
    searchType: SearchType.RunProjectOrRunRoutine,
    tabType: HistoryPageTabOption.RunsCompleted,
    where: () => ({ statuses: [RunStatus.Cancelled, RunStatus.Completed, RunStatus.Failed] }),
}];

/**
 * Shows items you've bookmarked, viewed, or run recently.
 */
export const HistoryView = ({
    isOpen,
    onClose,
    zIndex,
}: HistoryViewProps) => {
    const display = toDisplay(isOpen);
    const {
        currTab,
        handleTabChange,
        searchType,
        tabs,
        where,
    } = useTabs<HistoryPageTabOption>({ tabParams, display });

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
                zIndex={zIndex}
            />
            {searchType && <SearchList
                id="history-page-list"
                display={display}
                dummyLength={display === "page" ? 5 : 3}
                take={20}
                searchType={searchType}
                zIndex={zIndex}
                sxs={{
                    search: {
                        marginTop: 2,
                    },
                }}
                where={where()}
            />}
        </>
    );
};
