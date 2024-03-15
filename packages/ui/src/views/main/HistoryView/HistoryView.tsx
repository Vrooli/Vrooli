import { GqlModelType } from "@local/shared";
import { IconButton, useTheme } from "@mui/material";
import { PageTabs } from "components/PageTabs/PageTabs";
import { SideActionsButtons } from "components/buttons/SideActionsButtons/SideActionsButtons";
import { SearchList } from "components/lists/SearchList/SearchList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { useFindMany } from "hooks/useFindMany";
import { useTabs } from "hooks/useTabs";
import { AddIcon } from "icons";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { ListObject } from "utils/display/listTools";
import { getObjectUrlBase } from "utils/navigation/openObject";
import { HistoryPageTabOption, SearchType, historyTabParams } from "utils/search/objectToSearch";
import { HistoryViewProps } from "../types";

export const HistoryView = ({
    display,
    isOpen,
    onClose,
}: HistoryViewProps) => {
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

    const findManyData = useFindMany<ListObject>({
        controlsUrl: display === "page",
        searchType,
        take: 20,
        where: where(),
    });

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
            {
                searchType && <SearchList
                    {...findManyData}
                    id="history-page-list"
                    display={display}
                    dummyLength={display === "page" ? 5 : 3}
                    sxs={{ search: { marginTop: 2 } }}
                />
            }
            {
                searchType === SearchType.BookmarkList && <SideActionsButtons display={display}>
                    <IconButton
                        aria-label={t("Add")}
                        onClick={() => { setLocation(`${getObjectUrlBase({ __typename: GqlModelType.BookmarkList })}/add`); }}
                        sx={{ background: palette.secondary.main }}
                    >
                        <AddIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                    </IconButton>
                </SideActionsButtons>
            }
        </>
    );
};
