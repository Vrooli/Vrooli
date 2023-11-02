import { ChatInviteStatus, DUMMY_ID, GqlModelType, noop, User } from "@local/shared";
import { Box, IconButton, Tooltip, useTheme } from "@mui/material";
import { SideActionsButtons } from "components/buttons/SideActionsButtons/SideActionsButtons";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { SearchList } from "components/lists/SearchList/SearchList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { PageTabs } from "components/PageTabs/PageTabs";
import { useBulkObjectActions } from "hooks/useBulkObjectActions";
import { useTabs } from "hooks/useTabs";
import { ActionIcon, AddIcon, CancelIcon, DeleteIcon, EditIcon } from "icons";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { BulkObjectAction } from "utils/actions/bulkObjectActions";
import { ListObject } from "utils/display/listTools";
import { toDisplay } from "utils/display/pageTools";
import { ParticipantManagePageTabOption, participantTabParams } from "utils/search/objectToSearch";
import { ChatInviteShape } from "utils/shape/models/chatInvite";
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
        changeTab,
        currTab,
        handleTabChange,
        searchType,
        tabs,
        where,
    } = useTabs<ParticipantManagePageTabOption>({ id: "participant-manage-tabs", tabParams: participantTabParams, display });

    // const {
    //     allData,
    //     loading,
    //     removeItem,
    //     loadMore,
    //     setAllData,
    //     updateItem,
    // } = useFindMany<ListObject>({
    //     searchType,
    //     where: where(chat.id),
    // });

    const [isSelecting, setIsSelecting] = useState(true);
    const [selectedData, setSelectedData] = useState<ListObject[]>([]);
    const handleToggleSelecting = useCallback(() => {
        if (isSelecting) { setSelectedData([]); }
        setIsSelecting(is => !is);
    }, [isSelecting]);
    const handleToggleSelect = useCallback((item: ListObject) => {
        setSelectedData(items => {
            const newItems = [...items];
            const index = newItems.findIndex(i => i.id === item.id);
            if (index === -1) {
                newItems.push(item as ListObject);
            } else {
                newItems.splice(index, 1);
            }
            return newItems;
        });
    }, []);
    // Remove selection when tab changes
    useEffect(() => {
        setSelectedData([]);
    }, [currTab.tabType]);

    // Handle add/update invite dialog
    const [invitesToUpsert, setInvitesToUpsert] = useState<ChatInviteShape[]>([]);
    const handleInvitesUpdate = useCallback(() => {
        if (currTab.tabType !== ParticipantManagePageTabOption.ChatInvite) return;
        setInvitesToUpsert(selectedData as ChatInviteShape[]);
    }, [currTab.tabType, selectedData]);
    const handleInvitesCreate = useCallback(() => {
        if (currTab.tabType !== ParticipantManagePageTabOption.Add) return;
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
    }, [chat.id, currTab.tabType, selectedData]);
    const onInviteCompleted = () => {
        // TODO Handle any post-completion tasks here, if necessary
        setInvitesToUpsert([]);
    };

    // Handle deleting participants
    const { onBulkActionStart, BulkDeleteDialogComponent } = useBulkObjectActions<ListObject>({
        allData: [] as any, //TODO
        selectedData,
        objectType: searchType as unknown as GqlModelType,
        setAllData: noop, //TODO
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
                <IconButton aria-label={t("Select")} onClick={handleToggleSelecting} sx={{ background: palette.secondary.main }}>
                    <ActionIcon {...actionIconProps} />
                </IconButton>
            </Tooltip>);
            return buttons;
        }
        // If there are selected items, show relevant actions depending on tab
        if (selectedData.length > 0) {
            if ([ParticipantManagePageTabOption.ChatParticipant, ParticipantManagePageTabOption.ChatInvite].includes(currTab.tabType)) {
                buttons.push(<Tooltip title={t("Delete")}>
                    <IconButton aria-label={t("Delete")} onClick={() => { onBulkActionStart(BulkObjectAction.Delete); }} sx={{ background: palette.secondary.main }}>
                        <DeleteIcon {...actionIconProps} />
                    </IconButton>
                </Tooltip>);
            }
            if (currTab.tabType === ParticipantManagePageTabOption.ChatInvite) {
                buttons.push(<Tooltip title={t("Edit")}>
                    <IconButton aria-label={t("Edit")} onClick={handleInvitesUpdate} sx={{ background: palette.secondary.main }}>
                        <EditIcon {...actionIconProps} />
                    </IconButton>
                </Tooltip>);
            }
            if (currTab.tabType === ParticipantManagePageTabOption.Add) {
                buttons.push(<Tooltip title={t("Add")}>
                    <IconButton aria-label={t("Add")} onClick={handleInvitesCreate} sx={{ background: palette.secondary.main }}>
                        <AddIcon {...actionIconProps} />
                    </IconButton>
                </Tooltip>);
            }
        }
        // Show cancel button to exit selection mode
        buttons.push(<Tooltip title={t("Cancel")}>
            <IconButton aria-label={t("Cancel")} onClick={handleToggleSelecting} sx={{ background: palette.secondary.main }}>
                <CancelIcon {...actionIconProps} />
            </IconButton>
        </Tooltip>);
        return buttons;
    }, [palette.secondary.contrastText, palette.secondary.main, isSelecting, selectedData.length, t, handleToggleSelecting, currTab.tabType, onBulkActionStart, handleInvitesUpdate, handleInvitesCreate]);

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
            <ChatInviteUpsert
                invites={invitesToUpsert}
                isCreate={true}
                isOpen={invitesToUpsert.length > 0}
                onCompleted={onInviteCompleted}
                onCancel={() => setInvitesToUpsert([])}
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
                    id="participant-manage-list"
                    display={display}
                    dummyLength={display === "page" ? 5 : 3}
                    handleToggleSelect={handleToggleSelect}
                    isSelecting={isSelecting}
                    take={20}
                    searchType={searchType}
                    where={where(chat.id)}
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
};
