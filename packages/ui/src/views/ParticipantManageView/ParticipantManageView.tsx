import { ChatInvite, ChatInviteShape, ChatInviteStatus, DUMMY_ID, ListObject, noop, ParticipantManagePageTabOption, User, uuidValidate } from "@local/shared";
import { Box, IconButton, Tooltip, useTheme } from "@mui/material";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { SideActionsButtons } from "../../components/buttons/SideActionsButtons/SideActionsButtons.js";
import { MaybeLargeDialog } from "../../components/dialogs/LargeDialog/LargeDialog.js";
import { SearchList, SearchListScrollContainer } from "../../components/lists/SearchList/SearchList.js";
import { TopBar } from "../../components/navigation/TopBar.js";
import { PageTabs } from "../../components/PageTabs/PageTabs.js";
import { useBulkObjectActions } from "../../hooks/objectActions.js";
import { useFindMany } from "../../hooks/useFindMany.js";
import { useSelectableList } from "../../hooks/useSelectableList.js";
import { useTabs } from "../../hooks/useTabs.js";
import { IconCommon } from "../../icons/Icons.js";
import { BulkObjectAction } from "../../utils/actions/bulkObjectActions.js";
import { participantTabParams } from "../../utils/search/objectToSearch.js";
import { ChatInvitesUpsert } from "../../views/objects/chatInvite/ChatInvitesUpsert.js";
import { ParticipantManageViewProps } from "../types.js";

const scrollContainerId = "participant-search-scroll";
const dialogStyle = {
    paper: {
        minHeight: "min(100vh - 64px, 600px)",
        width: "min(100%, 500px)",
    },
} as const;

/**
 * View participants and invited participants of an chat
 */
export function ParticipantManageView({
    chat,
    display,
    onClose,
    isOpen,
}: ParticipantManageViewProps) {
    const { palette } = useTheme();
    const { t } = useTranslation();

    const {
        currTab,
        handleTabChange,
        searchType,
        tabs,
        where,
    } = useTabs({ id: "participant-manage-tabs", tabParams: participantTabParams, display });

    const findManyData = useFindMany<ListObject>({
        canSearch: () => uuidValidate(chat.id),
        controlsUrl: display === "page",
        searchType,
        take: 20,
        where: where({ chatId: chat.id }),
    });

    const {
        isSelecting,
        handleToggleSelecting,
        handleToggleSelect,
        selectedData,
        setIsSelecting,
        setSelectedData,
    } = useSelectableList(findManyData.allData);
    // Remove selection when tab changes
    useEffect(() => {
        setSelectedData([]);
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [currTab.key]);

    // Handle add/update invite dialog
    const [invitesToUpsert, setInvitesToUpsert] = useState<ChatInviteShape[]>([]);
    const handleInvitesUpdate = useCallback(() => {
        if (currTab.key !== ParticipantManagePageTabOption.ChatInvite) return;
        setInvitesToUpsert(selectedData as ChatInviteShape[]);
    }, [currTab.key, selectedData]);
    const handleInvitesCreate = useCallback(() => {
        if (currTab.key !== ParticipantManagePageTabOption.Add) return;
        const asInvites: ChatInviteShape[] = (selectedData as User[]).map(user => ({
            __typename: "ChatInvite",
            id: DUMMY_ID,
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
            status: ChatInviteStatus.Pending,
            chat: { __typename: "Chat", id: chat.id },
            user,
        } as const));
        setInvitesToUpsert(asInvites);
    }, [chat.id, currTab.key, selectedData]);
    function onInviteCompleted(invites: ChatInvite[]) {
        setInvitesToUpsert([]);
    }

    // Handle deleting participants
    const { onBulkActionStart, BulkDeleteDialogComponent } = useBulkObjectActions<ListObject>({
        ...findManyData,
        selectedData,
        setSelectedData: (data) => {
            setSelectedData(data);
            setIsSelecting(false);
        },
        setLocation: noop,
    });

    const sideActionButtons = useMemo(() => {
        const buttons: JSX.Element[] = [];
        // If not selecting, show select button
        if (!isSelecting) {
            buttons.push(<Tooltip title={t("Select")}>
                <IconButton aria-label={t("Select")} onClick={handleToggleSelecting} sx={{ background: palette.secondary.main }}>
                    <IconCommon name="Action" />
                </IconButton>
            </Tooltip>);
            return buttons;
        }
        // If there are selected items, show relevant actions depending on tab
        if (selectedData.length > 0) {
            if ([ParticipantManagePageTabOption.ChatParticipant, ParticipantManagePageTabOption.ChatInvite].includes(currTab.key as ParticipantManagePageTabOption)) {
                buttons.push(<Tooltip title={t("Delete")}>
                    <IconButton aria-label={t("Delete")} onClick={() => { onBulkActionStart(BulkObjectAction.Delete); }}>
                        <IconCommon name="Delete" />
                    </IconButton>
                </Tooltip>);
            }
            if (currTab.key === ParticipantManagePageTabOption.ChatInvite) {
                buttons.push(<Tooltip title={t("Edit")}>
                    <IconButton aria-label={t("Edit")} onClick={handleInvitesUpdate}>
                        <IconCommon name="Edit" />
                    </IconButton>
                </Tooltip>);
            }
            if (currTab.key === ParticipantManagePageTabOption.Add) {
                buttons.push(<Tooltip title={t("Add")}>
                    <IconButton aria-label={t("Add")} onClick={handleInvitesCreate}>
                        <IconCommon name="Add" />
                    </IconButton>
                </Tooltip>);
            }
        }
        // Show cancel button to exit selection mode
        buttons.push(<Tooltip title={t("Cancel")}>
            <IconButton aria-label={t("Cancel")} onClick={handleToggleSelecting}>
                <IconCommon name="Cancel" />
            </IconButton>
        </Tooltip>);
        return buttons;
    }, [palette.secondary.main, isSelecting, selectedData.length, t, handleToggleSelecting, currTab.key, onBulkActionStart, handleInvitesUpdate, handleInvitesCreate]);

    return (
        <MaybeLargeDialog
            display={display}
            id="participant-manage-dialog"
            isOpen={isOpen}
            onClose={onClose}
            sxs={dialogStyle}
        >
            <SearchListScrollContainer id={scrollContainerId}>
                {/* Dialog for creating/updating invites */}
                <ChatInvitesUpsert
                    display="dialog"
                    invites={invitesToUpsert}
                    isCreate={true}
                    isMutate={true}
                    isOpen={invitesToUpsert.length > 0}
                    onCancel={() => setInvitesToUpsert([])}
                    onClose={() => setInvitesToUpsert([])}
                    onCompleted={onInviteCompleted}
                    onDeleted={() => setInvitesToUpsert([])}
                />
                {BulkDeleteDialogComponent}
                {/* Main dialog */}
                <TopBar
                    display={display}
                    onClose={onClose}
                    title={t("Participant", { count: 2 })}
                    below={<PageTabs<typeof participantTabParams>
                        ariaLabel="Search tabs"
                        currTab={currTab}
                        fullWidth
                        onChange={handleTabChange}
                        tabs={tabs}
                    />}
                />
                <Box overflow="auto" flexGrow={1}>
                    {searchType && <SearchList
                        {...findManyData}
                        display={display}
                        handleToggleSelect={handleToggleSelect}
                        isSelecting={isSelecting}
                        selectedItems={selectedData}
                        scrollContainerId={scrollContainerId}
                        sxs={{
                            listContainer: { borderRadius: 0 },
                        }}
                    />}
                    {/* <ListContainer
                    borderRadius={0}
                    emptyText={t("NoResults", { ns: "error" })}
                    isEmpty={allData.length === 0 && !loading}
                >
                    <ObjectList
                        dummyItems={new Array(display === "page" ? 5 : 3).fill(searchType)}
                        handleToggleSelect={handleToggleSelect}
                        isSelecting={isSelecting}
                        items={allData as ListObject[]}
                        keyPrefix={`${searchType}-list-item`}
                        loading={loading}
                        onAction={noop}
                        selectedItems={selectedData}
                    />
                </ListContainer> */}
                </Box>
                <SideActionsButtons display={display} sx={{ position: "absolute" }}>
                    {sideActionButtons}
                </SideActionsButtons>
            </SearchListScrollContainer>
        </MaybeLargeDialog>
    );
}
