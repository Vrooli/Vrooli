import { MemberInvite } from "@local/shared";
import { Checkbox, FormControlLabel, Stack } from "@mui/material";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { SearchList } from "components/lists/SearchList/SearchList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { PageTabs } from "components/PageTabs/PageTabs";
import { Field } from "formik";
import { useTabs } from "hooks/useTabs";
import { useCallback, useEffect, useState } from "react";
import { toDisplay } from "utils/display/pageTools";
import { MemberManagePageTabOption, memberTabParams } from "utils/search/objectToSearch";
import { MemberInvitesUpsert } from "views/objects/memberInvite";
import { MemberManageViewProps } from "../types";

/**
 * View members and invited members of an organization
 */
export const MemberManageView = ({
    onClose,
    organization,
    isOpen,
}: MemberManageViewProps) => {
    console.log("in MemberManageView", organization);
    const display = toDisplay(isOpen);

    const {
        currTab,
        handleTabChange,
        searchType,
        tabs,
        where,
    } = useTabs<MemberManagePageTabOption>({ id: "member-manage-tabs", tabParams: memberTabParams, display });

    const [isInviteDialogOpen, setInviteDialogOpen] = useState(false);
    const onInviteStart = useCallback(() => {
        setInviteDialogOpen(true);
    }, []);
    const onInviteCompleted = useCallback((invite: MemberInvite) => {
        setInviteDialogOpen(false);
        // TODO add or update list
    }, []);

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
            <MemberInvitesUpsert
                isCreate={true}
                isOpen={isInviteDialogOpen}
                onCompleted={onInviteCompleted}
                onCancel={() => setInviteDialogOpen(false)}
                overrideObject={{ organization }}
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
                id="member-manage-list"
                display={display}
                dummyLength={display === "page" ? 5 : 3}
                take={20}
                searchType={searchType}
                where={where(organization.id)}
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
                <IconButton aria-label={t("FilterList")} onClick={toggleSearchFilters} sx={{ background: palette.secondary.main }}>
                    <SearchIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                </IconButton>
                <IconButton aria-label={t("CreateInvite")} onClick={onInviteStart} sx={{ background: palette.secondary.main }}>
                    <AddIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                </IconButton>
            </SideActionsButtons> */}
        </MaybeLargeDialog>
    );
};
