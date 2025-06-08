import Box from "@mui/material/Box";
import IconButton from "@mui/material/IconButton";
import { styled } from "@mui/material/styles";
import { type GraphActionsProps } from "../types.js";

const ActionsContainer = styled(Box)(({ theme }) => ({
    display: "flex",
    flexDirection: "row",
    gap: theme.spacing(1),
    zIndex: 2,
    height: "48px",
    // Make background semi-transparent
    background: theme.palette.primary.main + "80",
    color: theme.palette.primary.contrastText,
    justifyContent: "center",
    alignItems: "center",
    paddingTop: theme.spacing(1),
    paddingBottom: theme.spacing(1),
    "@media print": {
        display: "none",
    },
}));

const ActionsButton = styled(IconButton)(({ theme }) => ({
    background: theme.palette.secondary.main,
}));

export function GraphActions({
    // canRedo,
    // canUndo,
    // handleCleanUpGraph,
    // handleNodeDelete,
    // handleOpenLinkDialog,
    // handleRedo,
    // handleUndo,
    // isEditing,
    // language,
    // nodesOffGraph,
}: GraphActionsProps) {
    // const { t } = useTranslation();
    // const { breakpoints, palette } = useTheme();
    // const isMobile = useWindowSize(({ width }) => width < breakpoints.values.sm);

    // const [unlinkedNodesOpen, setIsUnlinkedNodesOpen] = useState(false);
    // const handleUnlinkedToggle = useCallback(() => { setIsUnlinkedNodesOpen(!unlinkedNodesOpen); }, [unlinkedNodesOpen, setIsUnlinkedNodesOpen]);

    // const show = useMemo(() => (isEditing || nodesOffGraph.length > 0) && !(isMobile && unlinkedNodesOpen), [isEditing, isMobile, nodesOffGraph.length, unlinkedNodesOpen]);

    // if (!show) return null;
    // return (
    //     <ActionsContainer>
    //         <Tooltip title={canUndo ? t("Undo") : ""}>
    //             <ActionsButton
    //                 id="undo-button"
    //                 disabled={!canUndo}
    //                 onClick={handleUndo}
    //                 aria-label={t("Undo")}
    //             >
    //                 <UndoIcon id="redo-button-icon" fill={palette.secondary.contrastText} />
    //             </ActionsButton>
    //         </Tooltip>
    //         <Tooltip title={canRedo ? t("Redo") : ""}>
    //             <ActionsButton
    //                 id="redo-button"
    //                 disabled={!canRedo}
    //                 onClick={handleRedo}
    //                 aria-label={t("Redo")}
    //             >
    //                 <RedoIcon id="redo-button-icon" fill={palette.secondary.contrastText} />
    //             </ActionsButton>
    //         </Tooltip>
    //         <Tooltip title={t("CleanGraph")}>
    //             <ActionsButton
    //                 id="clean-graph-button"
    //                 onClick={handleCleanUpGraph}
    //                 aria-label={t("CleanGraph")}
    //             >
    //                 <CompressIcon id="clean-up-button-icon" fill={palette.secondary.contrastText} />
    //             </ActionsButton>
    //         </Tooltip>
    //         <Tooltip title={t("AddNewLink")}>
    //             <ActionsButton
    //                 id="add-link-button"
    //                 onClick={handleOpenLinkDialog}
    //                 aria-label={t("AddNewLink")}
    //             >
    //                 <AddLinkIcon id="add-link-button-icon" fill={palette.secondary.contrastText} />
    //             </ActionsButton>
    //         </Tooltip>
    //     </ActionsContainer>
    // );
    return null;
}
