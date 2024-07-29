import { ChatInvite, ChatInviteShape, ChatInviteStatus, DUMMY_ID, ListObject, noop, ParticipantManagePageTabOption, User, uuidValidate } from "@local/shared";
import { Box, Tooltip, useTheme } from "@mui/material";
import { SideActionsButtons } from "components/buttons/SideActionsButtons/SideActionsButtons";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { SearchList } from "components/lists/SearchList/SearchList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { PageTabs } from "components/PageTabs/PageTabs";
import { useBulkObjectActions } from "hooks/useBulkObjectActions";
import { useFindMany } from "hooks/useFindMany";
import { useSelectableList } from "hooks/useSelectableList";
import { useTabs } from "hooks/useTabs";
import { ActionIcon, AddIcon, CancelIcon, DeleteIcon, EditIcon } from "icons";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { SideActionsButton } from "styles";
import { BulkObjectAction } from "utils/actions/bulkObjectActions";
import { participantTabParams } from "utils/search/objectToSearch";
import { ChatInvitesUpsert } from "views/objects/chatInvite";
import { ParticipantManageViewProps } from "../types";

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
    } = useSelectableList();
    // Remove selection when tab changes
    useEffect(() => {
        setSelectedData([]);
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
        const actionIconProps = { fill: palette.secondary.contrastText, width: "36px", height: "36px" };
        const buttons: JSX.Element[] = [];
        // If not selecting, show select button
        if (!isSelecting) {
            buttons.push(<Tooltip title={t("Select")}>
                <SideActionsButton aria-label={t("Select")} onClick={handleToggleSelecting} sx={{ background: palette.secondary.main }}>
                    <ActionIcon {...actionIconProps} />
                </SideActionsButton>
            </Tooltip>);
            return buttons;
        }
        // If there are selected items, show relevant actions depending on tab
        if (selectedData.length > 0) {
            if ([ParticipantManagePageTabOption.ChatParticipant, ParticipantManagePageTabOption.ChatInvite].includes(currTab.key as ParticipantManagePageTabOption)) {
                buttons.push(<Tooltip title={t("Delete")}>
                    <SideActionsButton aria-label={t("Delete")} onClick={() => { onBulkActionStart(BulkObjectAction.Delete); }}>
                        <DeleteIcon {...actionIconProps} />
                    </SideActionsButton>
                </Tooltip>);
            }
            if (currTab.key === ParticipantManagePageTabOption.ChatInvite) {
                buttons.push(<Tooltip title={t("Edit")}>
                    <SideActionsButton aria-label={t("Edit")} onClick={handleInvitesUpdate}>
                        <EditIcon {...actionIconProps} />
                    </SideActionsButton>
                </Tooltip>);
            }
            if (currTab.key === ParticipantManagePageTabOption.Add) {
                buttons.push(<Tooltip title={t("Add")}>
                    <SideActionsButton aria-label={t("Add")} onClick={handleInvitesCreate}>
                        <AddIcon {...actionIconProps} />
                    </SideActionsButton>
                </Tooltip>);
            }
        }
        // Show cancel button to exit selection mode
        buttons.push(<Tooltip title={t("Cancel")}>
            <SideActionsButton aria-label={t("Cancel")} onClick={handleToggleSelecting}>
                <CancelIcon {...actionIconProps} />
            </SideActionsButton>
        </Tooltip>);
        return buttons;
    }, [palette.secondary.contrastText, palette.secondary.main, isSelecting, selectedData.length, t, handleToggleSelecting, currTab.key, onBulkActionStart, handleInvitesUpdate, handleInvitesCreate]);

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
                below={<PageTabs
                    ariaLabel="search-tabs"
                    currTab={currTab}
                    fullWidth
                    onChange={handleTabChange}
                    tabs={tabs}
                />}
            />
            <Box sx={{ flexGrow: 1, overflowY: "auto" }} >
                {searchType && <SearchList
                    {...findManyData}
                    id="participant-manage-list"
                    display={display}
                    handleToggleSelect={handleToggleSelect}
                    isSelecting={isSelecting}
                    selectedItems={selectedData}
                    sxs={{
                        search: { marginTop: 2 },
                        listContainer: { borderRadius: 0 },
                    }}
                />}
                {/* <ListContainer
                    emptyText={t("NoResults", { ns: "error" })}
                    isEmpty={allData.length === 0 && !loading}
                    sx={{ borderRadius: 0 }}
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
        </MaybeLargeDialog>
    );
}
