import { ListObject, uuidValidate } from "@local/shared";
import { Checkbox, FormControlLabel, Stack } from "@mui/material";
import { PageTabs } from "components/PageTabs/PageTabs";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { SearchList } from "components/lists/SearchList/SearchList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Field } from "formik";
import { useFindMany } from "hooks/useFindMany";
import { useTabs } from "hooks/useTabs";
import { useCallback, useEffect, useState } from "react";
import { memberTabParams } from "utils/search/objectToSearch";
import { MemberManageViewProps } from "../types";

/**
 * View members and invited members of a team
 */
export function MemberManageView({
    display,
    onClose,
    isOpen,
    team,
}: MemberManageViewProps) {
    console.log("in MemberManageView", team);

    const {
        currTab,
        handleTabChange,
        searchType,
        tabs,
        where,
    } = useTabs({ id: "member-manage-tabs", tabParams: memberTabParams, display });

    const findManyData = useFindMany<ListObject>({
        canSearch: () => uuidValidate(team.id),
        controlsUrl: display === "page",
        searchType,
        take: 20,
        where: where({ teamId: team.id }),
    });

    // const handleMemberSelect = useCallback((member: RelationshipItemUser) => {
    //     membersFieldHelpers.setValue([...(membersField.value ?? []), member]);
    //     closeDialog();
    // }, [membersFieldHelpers, membersField.value, closeDialog]);

    // const searchData = useMemo(() => ({
    //     searchType: "User" as const,
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
    //            __typename: "MemberInvite",
    //            id: DUMMY_ID,
    //            created_at: new Date().toISOString(),
    //            updated_at: new Date().toISOString(),
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

    const [showSearchFilters, setShowSearchFilters] = useState<boolean>(false);
    const toggleSearchFilters = useCallback(() => setShowSearchFilters(!showSearchFilters), [showSearchFilters]);
    useEffect(() => {
        if (!showSearchFilters) return;
        const searchInput = document.getElementById("search-bar-member-manage-list");
        searchInput?.focus();
    }, [showSearchFilters]);

    return (
        <MaybeLargeDialog
            display={display}
            id="member-manage-dialog"
            isOpen={isOpen}
            onClose={onClose}
            sxs={{
                paper: {
                    minHeight: "min(100vh - 64px, 600px)",
                    width: "min(100%, 500px)",
                },
            }}
        >
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
                below={<PageTabs
                    ariaLabel="search-tabs"
                    currTab={currTab}
                    fullWidth
                    onChange={handleTabChange}
                    tabs={tabs}
                />}
            />
            <Stack direction="row" alignItems="center" justifyContent="flex-start" sx={{ padding: 2 }}>
                <FormControlLabel
                    control={<Field
                        name="isOpenToNewMembers"
                        type="checkbox"
                        as={Checkbox}
                        size="large"
                        color="secondary"
                    />}
                    label="Can anyone ask to join?"
                />
            </Stack >
            {searchType && <SearchList
                {...findManyData}
                id="member-manage-list"
                display={display}
                sxs={showSearchFilters ? {
                    search: { marginTop: 2 },
                    listContainer: { borderRadius: 0 },
                } : {
                    search: { display: "none" },
                    buttons: { display: "none" },
                    listContainer: { borderRadius: 0 },
                }}
            />}
            {/* <SideActionsButtons display={display}>
                <SideActionsButton aria-label={t("FilterList")} onClick={toggleSearchFilters}>
                    <SearchIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                </SideActionsButton>
                <SideActionsButton aria-label={t("CreateInvite")} onClick={onInviteStart}>
                    <AddIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                </SideActionsButton>
            </SideActionsButtons> */}
        </MaybeLargeDialog>
    );
}
