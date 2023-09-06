import { CommonKey } from "@local/shared";
import { IconButton, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { SideActionsButtons } from "components/buttons/SideActionsButtons/SideActionsButtons";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { SearchList } from "components/lists/SearchList/SearchList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { PageTabs } from "components/PageTabs/PageTabs";
import { useField } from "formik";
import { useTabs } from "hooks/useTabs";
import { AddIcon, LockIcon, SearchIcon, UnlockIcon } from "icons";
import { useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import { toDisplay } from "utils/display/pageTools";
import { scrollIntoFocusedView } from "utils/display/scroll";
import { MemberManagePageTabOption, memberTabParams } from "utils/search/objectToSearch";
import { MemberManageViewProps } from "../types";

/**
 * View members and invited members of an organization
 */
export const MemberManageView = ({
    onClose,
    organizationId,
    isOpen,
}: MemberManageViewProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const display = toDisplay(isOpen);

    const {
        currTab,
        handleTabChange,
        searchType,
        tabs,
        where,
    } = useTabs<MemberManagePageTabOption>({ tabParams: memberTabParams, display });

    const [isOpenToNewMembersField, , isOpenToNewMembersHelpers] = useField("isOpenToNewMembers");

    const [isInviteDialogOpen, setInviteDialogOpen] = useState(false);
    const onInviteStart = useCallback(() => {
        setInviteDialogOpen(true);
    }, []);

    const focusSearch = () => { scrollIntoFocusedView("search-bar-member-manage-list"); };

    return (
        <MaybeLargeDialog
            display={display}
            id="member-manage-dialog"
            isOpen={isOpen}
            onClose={onClose}
            sxs={{
                paper: {
                    minHeight: "min(100vh - 64px, 600px)",
                    maxWidth: "min(100%, 500px)",
                },
            }}
        >
            {/* Dialog for creating new member invite TODO */}
            {/* <MemberInviteUpsert
                    display="dialog"
                    isCreate={true}
                    isOpen={isInviteDialogOpen}
                    onCompleted={handleCreated}
                    onCancel={handleCreateClose}
                /> */}
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
            <Stack direction="row" alignItems="center" justifyContent="center" sx={{ paddingTop: 2 }}>
                <Typography component="h2" variant="h4">{t(searchType as CommonKey, { count: 2, defaultValue: searchType })}</Typography>
                <Tooltip title={t("Invite")} placement="top">
                    <IconButton
                        size="medium"
                        onClick={onInviteStart}
                        sx={{
                            padding: 1,
                        }}
                    >
                        <AddIcon fill={palette.secondary.main} width='1.5em' height='1.5em' />
                    </IconButton>
                </Tooltip>
                <Tooltip title="Indicates if this organization should be displayed when users are looking for an organization to join" placement="top">
                    <IconButton
                        size="medium"
                        onClick={() => isOpenToNewMembersHelpers.setValue(!isOpenToNewMembersField.value)}
                        sx={{
                            padding: 1,
                        }}
                    >
                        {isOpenToNewMembersField.value === true ? <UnlockIcon fill={palette.secondary.main} width='1.5em' height='1.5em' /> : <LockIcon fill={palette.secondary.main} width='1.5em' height='1.5em' />}
                    </IconButton>
                </Tooltip>
            </Stack >
            {searchType && <SearchList
                id="member-manage-list"
                display={display}
                dummyLength={display === "page" ? 5 : 3}
                take={20}
                searchType={searchType}
                where={where(organizationId)}
                sxs={{ search: { marginTop: 2 } }}
            />}
            <SideActionsButtons
                display={display}
                sx={{ position: "fixed" }}
            >
                <IconButton aria-label={t("FilterList")} onClick={focusSearch} sx={{ background: palette.secondary.main }}>
                    <SearchIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                </IconButton>
                <IconButton aria-label={t("CreateInvite")} onClick={onInviteStart} sx={{ background: palette.secondary.main }}>
                    <AddIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                </IconButton>
            </SideActionsButtons>
        </MaybeLargeDialog>
    );
};
