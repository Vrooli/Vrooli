/**
 * Displays nodes associated with a routine, but that are not linked to any other nodes.
 */
import { getTranslation, Node, NodeRoutineList, NodeType, noop } from "@local/shared";
import { Box, IconButton, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { EndNode } from "components/graphs/NodeGraph/nodes/EndNode/EndNode";
import { RedirectNode } from "components/graphs/NodeGraph/nodes/RedirectNode/RedirectNode";
import { RoutineListNode } from "components/graphs/NodeGraph/nodes/RoutineListNode/RoutineListNode";
import { DeleteIcon, ExpandLessIcon, ExpandMoreIcon, UnlinkedNodesIcon } from "icons";
import { useCallback } from "react";
import { useTranslation } from "react-i18next";
import { noSelect } from "styles";
import { NodeWithEnd } from "views/objects/node/types";
import { UnlinkedNodesDialogProps } from "../types";

export function UnlinkedNodesDialog({
    handleNodeDelete,
    handleToggleOpen,
    language,
    nodes,
    open,
}: UnlinkedNodesDialogProps) {
    const { palette } = useTheme();
    const { t } = useTranslation();

    /**
     * Generates a simple node from a node type
     */
    const createNode = useCallback((node: Node) => {
        // Common node props
        const nodeProps = {
            canDrag: true,
            handleAction: noop,
            isEditing: false,
            isLinked: false,
            key: `unlinked-node-${node.id}`,
            label: getTranslation(node, [language], false).name ?? null,
            labelVisible: false,
            scale: -0.5,
        };
        // Determine node to display based on nodeType
        switch (node.nodeType) {
            case NodeType.End:
                return <EndNode
                    {...nodeProps}
                    handleUpdate={noop}
                    handleDelete={noop}
                    language={language}
                    linksIn={[]}
                    node={node as NodeWithEnd}
                />;
            case NodeType.Redirect:
                return <RedirectNode
                    {...nodeProps}
                    node={node as any}//as NodeRedirect}
                />;
            case NodeType.RoutineList:
                return <RoutineListNode
                    {...nodeProps}
                    canExpand={false}
                    labelVisible={true}
                    language={language}
                    handleDelete={noop}
                    handleUpdate={noop}
                    linksIn={[]}
                    linksOut={[]}
                    node={node as Node & { routineList: NodeRoutineList }}
                />;
            default:
                return null;
        }
    }, [language]);

    return (
        <Tooltip title={t("NodeUnlinked", { count: 2 })}>
            <Box id="unlinked-nodes-dialog" sx={{
                alignSelf: open ? "baseline" : "auto",
                borderRadius: 3,
                background: palette.secondary.main,
                color: palette.secondary.contrastText,
                paddingLeft: 1,
                paddingRight: 1,
                paddingBottom: open ? 1 : 0,
                marginRight: 1,
                marginTop: open ? "4px" : "unset",
                maxHeight: { xs: "50vh", sm: "65vh", md: "72vh" },
                overflowX: "hidden",
                overflowY: "auto",
                width: open ? { xs: "100%", sm: "375px" } : "fit-content",
                transition: "height 1s ease-in-out",
                zIndex: 1500,
            }}>
                <Stack direction="row" onClick={handleToggleOpen} sx={{
                    display: "flex",
                    alignItems: "center",
                    justifyContent: "space-between",
                    width: "100%",
                }}>
                    <UnlinkedNodesIcon fill={palette.secondary.contrastText} />
                    <Typography variant="h6" sx={{ ...noSelect, marginLeft: "8px" }}>{open ? (t("Unlinked") + " ") : ""}({nodes.length})</Typography>
                    <Tooltip title={t(open ? "Shrink" : "Expand")}>
                        <IconButton edge="end" color="inherit" aria-label={t(open ? "Shrink" : "Expand")}>
                            {open ? <ExpandLessIcon fill={palette.secondary.contrastText} /> : <ExpandMoreIcon fill={palette.secondary.contrastText} />}
                        </IconButton>
                    </Tooltip>
                </Stack>
                {open ? (
                    <Stack direction="column" spacing={1}>
                        {nodes.map((node) => (
                            <Box key={node.id} sx={{ display: "flex", alignItems: "center" }}>
                                {/* Miniature version of node */}
                                <Box>
                                    {createNode(node)}
                                </Box>
                                {/* Node title */}
                                {node.nodeType === NodeType.RoutineList ? null : (<Typography variant="body1" sx={{ marginLeft: 1 }}>{getTranslation(node, [language], true).name}</Typography>)}
                                {/* Delete node icon */}
                                <Tooltip title={t("NodeDeleteWithName", { nodeName: getTranslation(node, [language], true).name })} placement="left">
                                    <Box sx={{ marginLeft: "auto" }}>
                                        <IconButton
                                            color="inherit"
                                            onClick={() => handleNodeDelete(node.id)}
                                            aria-label={t("NodeUnlinkedDelete")}
                                        >
                                            <DeleteIcon fill={palette.background.textPrimary} />
                                        </IconButton>
                                    </Box>
                                </Tooltip>
                            </Box>
                        ))}
                    </Stack>
                ) : null}
            </Box>
        </Tooltip>
    );
}
