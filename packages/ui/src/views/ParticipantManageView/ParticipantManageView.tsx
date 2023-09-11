import { ChatInvite } from "@local/shared";
import { IconButton, useTheme } from "@mui/material";
import { SideActionsButtons } from "components/buttons/SideActionsButtons/SideActionsButtons";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { SearchList } from "components/lists/SearchList/SearchList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { PageTabs } from "components/PageTabs/PageTabs";
import { useTabs } from "hooks/useTabs";
import { AddIcon, SearchIcon } from "icons";
import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { toDisplay } from "utils/display/pageTools";
import { ParticipantManagePageTabOption, participantTabParams } from "utils/search/objectToSearch";
import { ChatInviteUpsert } from "views/objects/chatInvite";
import { ParticipantManageViewProps } from "../types";

/**
 * View participants and invited participants of an chat
 */
export const ParticipantManageView = ({
    chat,
    onClose,
    isOpen,
}: ParticipantManageViewProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const display = toDisplay(isOpen);

    const {
        currTab,
        handleTabChange,
        searchType,
        tabs,
        where,
    } = useTabs<ParticipantManagePageTabOption>({ id: "participant-manage-tabs", tabParams: participantTabParams, display });

    const [isInviteDialogOpen, setInviteDialogOpen] = useState(false);
    const onInviteStart = useCallback(() => {
        setInviteDialogOpen(true);
    }, []);
    const onInviteCompleted = useCallback((invite: ChatInvite) => {
        setInviteDialogOpen(false);
        // TODO add or update list
    }, []);

    const [showSearchFilters, setShowSearchFilters] = useState<boolean>(false);
    const toggleSearchFilters = useCallback(() => setShowSearchFilters(!showSearchFilters), [showSearchFilters]);
    useEffect(() => {
        if (!showSearchFilters) return;
        const searchInput = document.getElementById("search-bar-participant-manage-list");
        searchInput?.focus();
    }, [showSearchFilters]);

    return (
        <MaybeLargeDialog
            display={display}
            id="participant-manage-dialog"
            isOpen={isOpen}
            onClose={onClose}
            sxs={{
                paper: {
                    minHeight: "min(100vh - 64px, 600px)",
                    width: "min(100%, 500px)",
                },
            }}
        >
            {/* Dialog for creating new participant invite */}
            <ChatInviteUpsert
                isCreate={true}
                isOpen={isInviteDialogOpen}
                onCompleted={onInviteCompleted}
                onCancel={() => setInviteDialogOpen(false)}
                overrideObject={{ chat }}
            />
            {/* Main dialog */}
            <TopBar
                display={display}
                onClose={onClose}
                below={<PageTabs
                    ariaLabel="search-tabs"
                    currTab={currTab}
                    fullWidth
                    onChange={handleTabChange}
                    tabs={tabs}
                />}
            />
            {searchType && <SearchList
                id="participant-manage-list"
                display={display}
                dummyLength={display === "page" ? 5 : 3}
                take={20}
                searchType={searchType}
                where={where(chat.id)}
                sxs={showSearchFilters ? {
                    search: { marginTop: 2 },
                    listContainer: { borderRadius: 0 },
                } : {
                    search: { display: "none" },
                    buttons: { display: "none" },
                    listContainer: { borderRadius: 0 },
                }}
            />}
            <SideActionsButtons display={display}>
                <IconButton aria-label={t("FilterList")} onClick={toggleSearchFilters} sx={{ background: palette.secondary.main }}>
                    <SearchIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                </IconButton>
                <IconButton aria-label={t("CreateInvite")} onClick={onInviteStart} sx={{ background: palette.secondary.main }}>
                    <AddIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                </IconButton>
            </SideActionsButtons>
        </MaybeLargeDialog>
    );
};
