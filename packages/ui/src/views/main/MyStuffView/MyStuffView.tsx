import { GqlModelType, ListObject, getObjectUrlBase, uuidValidate } from "@local/shared";
import { IconButton, ListItemIcon, ListItemText, Menu, MenuItem, Tooltip, useTheme } from "@mui/material";
import { PageTabs } from "components/PageTabs/PageTabs";
import { SideActionsButtons } from "components/buttons/SideActionsButtons/SideActionsButtons";
import { SearchList } from "components/lists/SearchList/SearchList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { SessionContext } from "contexts/SessionContext";
import { useBulkObjectActions } from "hooks/useBulkObjectActions";
import { useFindMany } from "hooks/useFindMany";
import { usePopover } from "hooks/usePopover";
import { useSelectableList } from "hooks/useSelectableList";
import { useTabs } from "hooks/useTabs";
import { ActionIcon, AddIcon, CancelIcon, DeleteIcon, SearchIcon } from "icons";
import { useCallback, useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { BulkObjectAction } from "utils/actions/bulkObjectActions";
import { getCurrentUser } from "utils/authentication/session";
import { scrollIntoFocusedView } from "utils/display/scroll";
import { SearchType, myStuffTabParams } from "utils/search/objectToSearch";
import { MyStuffViewProps } from "../types";

export function MyStuffView({
    display,
    onClose,
}: MyStuffViewProps) {
    const session = useContext(SessionContext);
    const [, setLocation] = useLocation();
    const { palette } = useTheme();
    const { t } = useTranslation();
    const {
        id: userId,
        apisCount,
        codesCount,
        membershipsCount,
        questionsAskedCount,
        standardsCount,
    } = useMemo(() => getCurrentUser(session), [session]);

    /**
     * Filter out certain tabs that we don't have any data for, 
     * so user isn't overwhelmed with tabs for objects they never worked with. 
     * Always keeps routines, projects, and notes
     */
    const filteredTabs = useMemo(() => myStuffTabParams.filter(tab => {
        switch (tab.key) {
            case "Api":
                return Boolean(apisCount);
            case "Code":
                return Boolean(codesCount);
            case "Question":
                return Boolean(questionsAskedCount);
            case "Standard":
                return Boolean(standardsCount);
            case "Team":
                return Boolean(membershipsCount);
        }
        return true;
    }), [apisCount, codesCount, membershipsCount, questionsAskedCount, standardsCount]);
    const {
        currTab,
        handleTabChange,
        searchType,
        tabs,
        where,
    } = useTabs({ id: "my-stuff-tabs", tabParams: filteredTabs, display });

    const findManyData = useFindMany<ListObject>({
        canSearch: () => uuidValidate(userId),
        controlsUrl: display === "page",
        searchType,
        take: 20,
        where: where({ userId: userId ?? "" }),
    });

    const {
        isSelecting,
        handleToggleSelecting,
        handleToggleSelect,
        selectedData,
        setIsSelecting,
        setSelectedData,
    } = useSelectableList<ListObject>();
    const { onBulkActionStart, BulkDeleteDialogComponent } = useBulkObjectActions<ListObject>({
        ...findManyData,
        selectedData,
        setSelectedData: (data) => {
            setSelectedData(data);
            setIsSelecting(false);
        },
        setLocation,
    });

    // Menu for selection object type to create
    const [selectCreateTypeAnchorEl, openSelectCreateType, closeSelectCreateType] = usePopover();
    const onCreateStart = useCallback((event: React.MouseEvent<HTMLElement>) => {
        // If tab is 'All', open menu to select type
        if (searchType === SearchType.Popular) {
            openSelectCreateType(event);
            return;
        }
        // Navigate to object's add page
        setLocation(`${getObjectUrlBase({ __typename: searchType as `${GqlModelType}` })}/add`);
    }, [openSelectCreateType, searchType, setLocation]);
    const onSelectCreateTypeClose = useCallback((type?: SearchType | `${SearchType}`) => {
        if (type) setLocation(`${getObjectUrlBase({ __typename: type as `${GqlModelType}` })}/add`);
        else closeSelectCreateType();
    }, [closeSelectCreateType, setLocation]);

    function focusSearch() { scrollIntoFocusedView("search-bar-my-stuff-list"); }

    const actionIconProps = useMemo(() => ({ fill: palette.secondary.contrastText, width: "36px", height: "36px" }), [palette.secondary.contrastText]);

    return (
        <>
            {BulkDeleteDialogComponent}
            <Menu
                id="select-create-type-menu"
                anchorEl={selectCreateTypeAnchorEl}
                disableScrollLock={true}
                open={Boolean(selectCreateTypeAnchorEl)}
                onClose={() => onSelectCreateTypeClose()}
            >
                {/* Never show 'All' */}
                {myStuffTabParams
                    .filter((t) => ![SearchType.Popular]
                        .includes(t.searchType as SearchType)).map(({ Icon, key, searchType }) => (
                            <MenuItem
                                key={key}
                                onClick={() => onSelectCreateTypeClose(searchType)}
                            >
                                {Icon && <ListItemIcon>
                                    <Icon fill={palette.background.textPrimary} />
                                </ListItemIcon>}
                                <ListItemText primary={t(searchType, { count: 1, defaultValue: searchType })} />
                            </MenuItem>
                        ))}
            </Menu>
            <TopBar
                display={display}
                onClose={onClose}
                title={t("MyStuff")}
                titleBehaviorDesktop="ShowIn"
                below={<PageTabs
                    ariaLabel="my-stuff-tabs"
                    fullWidth
                    id="my-stuff-tabs"
                    ignoreIcons
                    currTab={currTab}
                    onChange={handleTabChange}
                    tabs={tabs}
                />}
            />
            {searchType && <SearchList
                {...findManyData}
                id="my-stuff-list"
                display={display}
                handleToggleSelect={handleToggleSelect}
                isSelecting={isSelecting}
                selectedItems={selectedData}
                sxs={{ search: { marginTop: 2 } }}
            />}
            <SideActionsButtons display={display}>
                {isSelecting && selectedData.length > 0 ? <Tooltip title={t("Delete")}>
                    <IconButton aria-label={t("Delete")} onClick={() => { onBulkActionStart(BulkObjectAction.Delete); }} sx={{ background: palette.secondary.main }}>
                        <DeleteIcon {...actionIconProps} />
                    </IconButton>
                </Tooltip> : null}
                <Tooltip title={t(isSelecting ? "Cancel" : "Select")}>
                    <IconButton aria-label={t(isSelecting ? "Cancel" : "Select")} onClick={handleToggleSelecting} sx={{ background: palette.secondary.main }}>
                        {isSelecting ? <CancelIcon {...actionIconProps} /> : <ActionIcon {...actionIconProps} />}
                    </IconButton>
                </Tooltip>
                {!isSelecting ? <IconButton aria-label={t("FilterList")} onClick={focusSearch} sx={{ background: palette.secondary.main }}>
                    <SearchIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                </IconButton> : null}
                {userId ? (
                    <IconButton aria-label={t("Add")} onClick={onCreateStart} sx={{ background: palette.secondary.main }}>
                        <AddIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                    </IconButton>
                ) : null}
            </SideActionsButtons>
        </>
    );
}
