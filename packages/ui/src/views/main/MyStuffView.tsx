import IconButton from "@mui/material/IconButton";
import ListItemIcon from "@mui/material/ListItemIcon";
import ListItemText from "@mui/material/ListItemText";
import Menu from "@mui/material/Menu";
import MenuItem from "@mui/material/MenuItem";
import Tooltip from "@mui/material/Tooltip";
import { useTheme } from "@mui/material/styles";
import { getObjectUrlBase, validatePK, type ListObject, type ModelType, type SearchType } from "@vrooli/shared";
import { useCallback, useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { PageContainer } from "../../components/Page/Page.js";
import { PageTabs } from "../../components/PageTabs/PageTabs.js";
import { SideActionsButtons } from "../../components/buttons/SideActionsButtons.js";
import { SearchList, SearchListScrollContainer } from "../../components/lists/SearchList/SearchList.js";
import { Navbar } from "../../components/navigation/Navbar.js";
import { SessionContext } from "../../contexts/session.js";
import { useBulkObjectActions } from "../../hooks/objectActions.js";
import { useFindMany } from "../../hooks/useFindMany.js";
import { usePopover } from "../../hooks/usePopover.js";
import { useSelectableList } from "../../hooks/useSelectableList.js";
import { useTabs } from "../../hooks/useTabs.js";
import { Icon, IconCommon } from "../../icons/Icons.js";
import { useLocation } from "../../route/router.js";
import { BulkObjectAction } from "../../utils/actions/bulkObjectActions.js";
import { getCurrentUser } from "../../utils/authentication/session.js";
import { ELEMENT_CLASSES, ELEMENT_IDS } from "../../utils/consts.js";
import { scrollIntoFocusedView } from "../../utils/display/scroll.js";
import { myStuffTabParams } from "../../utils/search/objectToSearch.js";
import { type MyStuffViewProps } from "./types.js";

const scrollContainerId = "my-stuff-search-scroll";
const pageContainerStyle = {
    [`& .${ELEMENT_CLASSES.SearchBar}`]: {
        margin: 2,
    },
} as const;

export function MyStuffView({
    display,
}: MyStuffViewProps) {
    const session = useContext(SessionContext);
    const [, setLocation] = useLocation();
    const { palette } = useTheme();
    const { t } = useTranslation();
    const currentUser = useMemo(() => getCurrentUser(session), [session]);

    const {
        currTab,
        handleTabChange,
        searchType,
        tabs,
        where,
    } = useTabs({ id: ELEMENT_IDS.MyStuffTabs, tabParams: myStuffTabParams, display });

    const findManyData = useFindMany<ListObject>({
        canSearch: () => validatePK(currentUser.id),
        controlsUrl: display === "Page",
        searchType,
        take: 20,
        where: where({ userId: currentUser.id ?? "" }),
    });

    const {
        isSelecting,
        handleToggleSelecting,
        handleToggleSelect,
        selectedData,
        setIsSelecting,
        setSelectedData,
    } = useSelectableList<ListObject>(findManyData.allData);
    const { onBulkActionStart, BulkDeleteDialogComponent } = useBulkObjectActions<ListObject>({
        ...findManyData,
        selectedData,
        setSelectedData: (data) => {
            setSelectedData(data);
            setIsSelecting(false);
        },
        setLocation,
    });
    function startBulkDelete() {
        onBulkActionStart(BulkObjectAction.Delete);
    }

    // Menu for selection object type to create
    const [selectCreateTypeAnchorEl, openSelectCreateType, closeSelectCreateType] = usePopover();
    const onCreateStart = useCallback((event: React.MouseEvent<HTMLElement>) => {
        // If tab is 'All', open menu to select type
        if (searchType === "Popular") {
            openSelectCreateType(event);
            return;
        }
        // Navigate to object's add page
        setLocation(`${getObjectUrlBase({ __typename: searchType as `${ModelType}` })}/add`);
    }, [openSelectCreateType, searchType, setLocation]);
    const onSelectCreateTypeClose = useCallback((type?: SearchType | `${SearchType}`) => {
        if (type) setLocation(`${getObjectUrlBase({ __typename: type as `${ModelType}` })}/add`);
        else closeSelectCreateType();
    }, [closeSelectCreateType, setLocation]);

    function focusSearch() { scrollIntoFocusedView("search-bar-my-stuff-list"); }

    return (
        <PageContainer size="fullSize" sx={pageContainerStyle}>
            <Navbar title={t("MyStuff")} />
            <PageTabs<typeof myStuffTabParams>
                ariaLabel="Search tabs"
                fullWidth
                id={ELEMENT_IDS.MyStuffTabs}
                ignoreIcons
                currTab={currTab}
                onChange={handleTabChange}
                tabs={tabs}
            />
            <SearchListScrollContainer id={scrollContainerId}>
                {BulkDeleteDialogComponent}
                <Menu
                    id="select-create-type-menu"
                    anchorEl={selectCreateTypeAnchorEl}
                    disableScrollLock={true}
                    open={Boolean(selectCreateTypeAnchorEl)}
                    onClose={closeSelectCreateType}
                >
                    {/* Never show 'All' */}
                    {myStuffTabParams
                        .filter((t) => !["Popular"]
                            .includes(t.searchType as SearchType)).map(({ iconInfo, key, searchType }) => {
                                function handleClick() {
                                    onSelectCreateTypeClose(searchType);
                                }

                                return (
                                    <MenuItem
                                        key={key}
                                        onClick={handleClick}
                                    >
                                        {iconInfo && <ListItemIcon>
                                            <Icon
                                                decorative
                                                fill={palette.background.textPrimary}
                                                info={iconInfo}
                                            />
                                        </ListItemIcon>}
                                        <ListItemText primary={t(searchType, { count: 1, defaultValue: searchType })} />
                                    </MenuItem>
                                );
                            })}
                </Menu>
                {searchType && <SearchList
                    {...findManyData}
                    display={display}
                    handleToggleSelect={handleToggleSelect}
                    isSelecting={isSelecting}
                    scrollContainerId={scrollContainerId}
                    selectedItems={selectedData}
                />}
                <SideActionsButtons display={display}>
                    {isSelecting && selectedData.length > 0 ? <Tooltip title={t("Delete")}>
                        <IconButton
                            aria-label={t("Delete")}
                            onClick={startBulkDelete}
                        >
                            <IconCommon name="Delete" />
                        </IconButton>
                    </Tooltip> : null}
                    <Tooltip title={t(isSelecting ? "Cancel" : "Select")}>
                        <IconButton
                            aria-label={t(isSelecting ? "Cancel" : "Select")}
                            onClick={handleToggleSelecting}
                        >
                            <IconCommon name={isSelecting ? "Cancel" : "Action"} />
                        </IconButton>
                    </Tooltip>
                    {!isSelecting ? <IconButton aria-label={t("FilterList")} onClick={focusSearch}>
                        <IconCommon name="Search" />
                    </IconButton> : null}
                    {currentUser.id ? (
                        <IconButton aria-label={t("Add")} onClick={onCreateStart}>
                            <IconCommon name="Add" />
                        </IconButton>
                    ) : null}
                </SideActionsButtons>
            </SearchListScrollContainer>
        </PageContainer>
    );
}
