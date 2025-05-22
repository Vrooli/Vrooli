import { type ListObject, validatePK } from "@local/shared";
import { Box, Button, Checkbox, Divider, FormControlLabel, IconButton, Stack, useTheme } from "@mui/material";
import { Field } from "formik";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { PageTabs } from "../../components/PageTabs/PageTabs.js";
import { SideActionsButtons } from "../../components/buttons/SideActionsButtons.js";
import { MaybeLargeDialog } from "../../components/dialogs/LargeDialog/LargeDialog.js";
import { SearchList, SearchListScrollContainer } from "../../components/lists/SearchList/SearchList.js";
import { TopBar } from "../../components/navigation/TopBar.js";
import { useFindMany } from "../../hooks/useFindMany.js";
import { useSelectableList } from "../../hooks/useSelectableList.js";
import { useTabs } from "../../hooks/useTabs.js";
import { IconCommon } from "../../icons/Icons.js";
import { memberTabParams } from "../../utils/search/objectToSearch.js";
import { type MemberManageViewProps } from "../types.js";

const scrollContainerId = "member-search-scroll";
const dialogStyle = {
    paper: {
        minHeight: "min(100vh - 64px, 600px)",
        width: "min(100%, 500px)",
    },
} as const;

/**
 * View members and invited members of a team
 */
export function MemberManageView({
    display,
    onClose,
    isEditing,
    isOpen,
    team,
}: MemberManageViewProps) {
    const { palette } = useTheme();
    const { t } = useTranslation();

    const {
        currTab,
        handleTabChange,
        searchType,
        tabs,
        where,
    } = useTabs({ id: "member-manage-tabs", tabParams: memberTabParams, display });
    const toInviteNewMembersTab = useCallback(function toInviteNewMembersTabCallback() {
        const nonMembersTab = tabs.find(tab => tab.key === "NonMembers");
        if (nonMembersTab) {
            handleTabChange(undefined, nonMembersTab);
        } else {
            console.error("NonMembers tab not found");
        }
    }, [handleTabChange, tabs]);

    const findManyData = useFindMany<ListObject>({
        canSearch: () => validatePK(team.id),
        controlsUrl: display === "Page",
        searchType,
        take: 20,
        where: where({ teamId: team.id }),
    });

    const {
        isSelecting,
        handleToggleSelecting,
        handleToggleSelect,
        selectedData,
        setIsSelecting,
        setSelectedData,
    } = useSelectableList<ListObject>(findManyData.allData);
    // Remove selection when tab changes
    useEffect(() => {
        setSelectedData([]);
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [currTab.key]);

    const notIfEditing = useCallback(function notIfEditingCallback() {
        return isEditing !== true;
    }, [isEditing]);

    // const handleMemberSelect = useCallback((member: RelationshipItemUser) => {
    //     membersFieldHelpers.setValue([...(membersField.value ?? []), member]);
    //     closeDialog();
    // }, [membersFieldHelpers, membersField.value, closeDialog]);

    // const searchData = useMemo(() => ({
    //     searchType: "NonMembers" as const,
    //     where: { memberInTeamId: membersField?.value?.id },
    // }), [membersField?.value?.id]);

    //    // Handle add/update invite dialog
    //    const [invitesToUpsert, setInvitesToUpsert] = useState<MemberInviteShape[]>([]);
    //    const handleInvitesUpdate = useCallback(() => {
    //        if (currTab.key !== ParticipantManagePageTabOption.MemberInvite) return;
    //        setInvitesToUpsert(selectedData as MemberInviteShape[]);
    //    }, [currTab.key, selectedData]);
    //    const handleInvitesCreate = useCallback(() => {
    //        if (currTab.key !== ParticipantManagePageTabOption.Add) return;
    //        const asInvites: MemberInviteShape[] = (selectedData as User[]).map(user => ({
    //            __typename: "Invites",
    //            id: DUMMY_ID,
    //            createdAt: new Date().toISOString(),
    //            updatedAt: new Date().toISOString(),
    //            status: MemberInviteStatus.Pending,
    //            team: { __typename: "Team", id: team.id },
    //            user,
    //        } as const));
    //        setInvitesToUpsert(asInvites);
    //    }, [team.id, currTab.key, selectedData]);
    //    const onInviteCompleted = () => {
    //        // TODO Handle any post-completion tasks here, if necessary
    //        setInvitesToUpsert([]);
    //    };

    const handleRemoveMembers = useCallback(() => {
        // TODO Implement remove members logic
        console.log("Removing members:", selectedData);
        setSelectedData([]);
        setIsSelecting(false);
    }, [selectedData, setSelectedData, setIsSelecting]);

    const handleCancelInvites = useCallback(() => {
        // TODO Implement cancel invites logic
        console.log("Cancelling invites:", selectedData);
        setSelectedData([]);
        setIsSelecting(false);
    }, [selectedData, setSelectedData, setIsSelecting]);

    const handleInviteMembers = useCallback(() => {
        // TODO Implement invite members logic
        console.log("Inviting members:", selectedData);
        setSelectedData([]);
        setIsSelecting(false);
    }, [selectedData, setSelectedData, setIsSelecting]);

    const [showSearchFilters, setShowSearchFilters] = useState<boolean>(false);
    const toggleSearchFilters = useCallback(() => setShowSearchFilters(!showSearchFilters), [showSearchFilters]);
    useEffect(() => {
        if (!showSearchFilters) return;
        const searchInput = document.getElementById("search-bar-member-manage-list");
        searchInput?.focus();
    }, [showSearchFilters]);

    const renderTabContent = useMemo(function renderTabContentMemo() {
        switch (currTab.key) {
            case "Members":
                return (
                    <>
                        <Stack direction="row" alignItems="center" justifyContent="flex-start" p={2}>
                            <FormControlLabel
                                control={<Field
                                    name="isOpenToNewMembers"
                                    type="checkbox"
                                    as={Checkbox}
                                    size="large"
                                    color="secondary"
                                />}
                                label={t("CanAnyoneAskToJoin")}
                            />
                        </Stack>
                        <Divider />
                        <SearchList
                            {...findManyData}
                            borderRadius={0}
                            canNavigate={notIfEditing}
                            display={display}
                            scrollContainerId={scrollContainerId}
                            handleToggleSelect={handleToggleSelect}
                            isSelecting={isSelecting}
                            selectedItems={selectedData}
                            variant={showSearchFilters ? "normal" : "minimal"}
                        />
                        {
                            findManyData.allData.length === 0 && <Box m={2}>
                                <Button
                                    fullWidth
                                    onClick={toInviteNewMembersTab}
                                    variant="contained"
                                >
                                    {t("InviteMembers")}
                                </Button>
                            </Box>
                        }
                    </>
                );
            case "Invites":
                return (
                    <>
                        <SearchList
                            {...findManyData}
                            borderRadius={0}
                            canNavigate={notIfEditing}
                            display={display}
                            scrollContainerId={scrollContainerId}
                            handleToggleSelect={handleToggleSelect}
                            isSelecting={isSelecting}
                            selectedItems={selectedData}
                            variant={showSearchFilters ? "normal" : "minimal"}
                        />
                        {
                            findManyData.allData.length === 0 && <Box m={2}>
                                <Button
                                    fullWidth
                                    onClick={toInviteNewMembersTab}
                                    variant="contained"
                                >
                                    {t("InviteMembers")}
                                </Button>
                            </Box>
                        }
                    </>
                );
            case "NonMembers":
                return (
                    <SearchList
                        {...findManyData}
                        borderRadius={0}
                        canNavigate={notIfEditing}
                        display={display}
                        scrollContainerId={scrollContainerId}
                        handleToggleSelect={handleToggleSelect}
                        isSelecting={isSelecting}
                        selectedItems={selectedData}
                        variant={showSearchFilters ? "normal" : "minimal"}
                    />
                );
            default:
                return null;
        }
    }, [currTab.key, t, findManyData, notIfEditing, display, handleToggleSelect, isSelecting, selectedData, showSearchFilters, toInviteNewMembersTab]);

    const renderActionButtons = useMemo(function renderActionButtonsMemo() {
        if (isSelecting && selectedData.length > 0) {
            switch (currTab.key) {
                case "Members":
                    return (
                        <IconButton aria-label={t("RemoveMembers")} onClick={handleRemoveMembers}>
                            <IconCommon name="Delete" />
                        </IconButton>
                    );
                case "Invites":
                    return (
                        <IconButton aria-label={t("CancelInvites")} onClick={handleCancelInvites}>
                            <IconCommon name="Delete" />
                        </IconButton>
                    );
                case "NonMembers":
                    return (
                        <IconButton aria-label={t("InviteMembers")} onClick={handleInviteMembers}>
                            <IconCommon name="Add" />
                        </IconButton>
                    );
            }
        }

        return null;
    }, [currTab.key, isSelecting, selectedData.length, t, handleRemoveMembers, handleCancelInvites, handleInviteMembers]);


    return (
        <MaybeLargeDialog
            display={display}
            id="member-manage-dialog"
            isOpen={isOpen}
            onClose={onClose}
            sxs={dialogStyle}
        >
            <SearchListScrollContainer id={scrollContainerId}>
                {/* Dialog for creating new member invite */}
                {/* <MemberInvitesUpsert
                isCreate={true}
                isMutate={false}
                invites={invites}
                isOpen={isInviteDialogOpen}
                onCompleted={onInviteCompleted}
                onCancel={() => setInviteDialogOpen(false)}
            />  */}
                {/* Main dialog */}
                <TopBar
                    display={display}
                    onClose={onClose}
                    below={<PageTabs<typeof memberTabParams>
                        ariaLabel="Search tabs"
                        currTab={currTab}
                        fullWidth
                        onChange={handleTabChange}
                        tabs={tabs}
                    />}
                />
                {renderTabContent}
                <SideActionsButtons display={display}>
                    {renderActionButtons}
                    <IconButton aria-label={t(isSelecting ? "Cancel" : "Select")} onClick={handleToggleSelecting}>
                        <IconCommon name={isSelecting ? "Cancel" : "Action"} />
                    </IconButton>
                    {!isSelecting ? (
                        <IconButton aria-label={t("Search")} onClick={toggleSearchFilters}>
                            <IconCommon name="Search" />
                        </IconButton>
                    ) : null}
                </SideActionsButtons>
            </SearchListScrollContainer>
        </MaybeLargeDialog>
    );
}
