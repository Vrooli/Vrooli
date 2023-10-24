import { ChatInvite, ChatParticipant, User } from "@local/shared";
import { Box, useTheme } from "@mui/material";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { RichInputBase } from "components/inputs/RichInputBase/RichInputBase";
import { SearchList } from "components/lists/SearchList/SearchList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { PageTabs } from "components/PageTabs/PageTabs";
import { useTabs } from "hooks/useTabs";
import { AddIcon } from "icons";
import { useCallback } from "react";
import { useTranslation } from "react-i18next";
import { pagePaddingBottom } from "styles";
import { toDisplay } from "utils/display/pageTools";
import { ParticipantManagePageTabOption, participantTabParams } from "utils/search/objectToSearch";
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
        changeTab,
        currTab,
        handleTabChange,
        searchType,
        tabs,
        where,
    } = useTabs<ParticipantManagePageTabOption>({ id: "participant-manage-tabs", tabParams: participantTabParams, display });

    const handleItemClick = useCallback((item: ChatParticipant | ChatInvite | User) => {
        // If members or invites tab, toggle selected
        //TODO
        // If add tab, add to invites 
    }, []);

    return (
        <MaybeLargeDialog
            display={display}
            id="participant-manage-dialog"
            isOpen={isOpen}
            onClose={onClose}
            sxs={{
                paper: {
                    minHeight: "min(100vh - 64px, 800px)",
                    width: "min(100%, 500px)",
                    display: "flex",
                },
            }}
        >
            {/* Dialog for creating new participant invite */}
            {/* <ChatInviteUpsert
                isCreate={true}
                isOpen={isInviteDialogOpen}
                onCompleted={onInviteCompleted}
                onCancel={() => setInviteDialogOpen(false)}
                overrideObject={{ chat }}
            /> */}
            {/* Main dialog */}
            <TopBar
                display={display}
                onClose={onClose}
                title={t("Participant", { count: 2 })}
                below={<PageTabs
                    ariaLabel="search-tabs"
                    currTab={currTab}
                    fullWidth
                    onChange={handleTabChange}
                    tabs={tabs}
                />}
            />
            <Box sx={{ flexGrow: 1, overflowY: "auto" }} >
                {/* TODO for invites tab, show newly invited first */}
                {searchType && <SearchList
                    id="participant-manage-list"
                    display={display}
                    dummyLength={display === "page" ? 5 : 3}
                    onItemClick={handleItemClick}
                    take={20}
                    searchType={searchType}
                    where={where(chat.id)}
                    sxs={{
                        search: { marginTop: 2 },
                        listContainer: { borderRadius: 0 },
                    }}
                />}
            </Box>
            {/* Text input to set message for selected invite(s) */}
            {/* TODO need way to put message text input inbetween side and bottom buttons */}
            {currTab.tabType === ParticipantManagePageTabOption.ChatInvite ? <RichInputBase
                actionButtons={[{
                    Icon: AddIcon,
                    onClick: () => {
                        //TODO
                    },
                }]}
                fullWidth
                maxChars={1500}
                minRows={1}
                onChange={() => { }} //TODO
                name="inviteMessage"
                sxs={{
                    root: {
                        background: palette.primary.dark,
                        color: palette.primary.contrastText,
                        maxHeight: "min(50vh, 500px)",
                        width: "min(700px, 100%)",
                        margin: "auto",
                        marginBottom: { xs: display === "page" ? pagePaddingBottom : "0", md: "0" },
                    },
                    bar: { borderRadius: 0 },
                    textArea: { paddingRight: 4, border: "none" },
                }}
                value={""} //TODO
            /> : null}
            <BottomActionsButtons
                display={display}
                errors={{}} //TODO need to add formik
                isCreate={false} //TODO
                loading={false} //TODO
                onCancel={() => { }} //TODO
                onSetSubmitting={() => { }}//TODO
                onSubmit={() => { }} //TODO
            />
        </MaybeLargeDialog>
    );
};
