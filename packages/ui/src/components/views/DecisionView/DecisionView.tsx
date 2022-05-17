import { ListItem, ListItemButton, ListItemText, Stack, Typography, useTheme } from "@mui/material";
import {
    OpenInNew as OpenLinkIcon,
} from "@mui/icons-material";
import { useCallback, useMemo } from "react";
import { containerShadow, multiLineEllipsis } from "styles";
import { Node, NodeLink } from "types";
import { getTranslation, getUserLanguages } from "utils";
import { DecisionViewProps } from "../types";

type Decision = {
    node: Node;
    link: NodeLink;
};

export const DecisionView = ({
    data,
    handleDecisionSelect,
    nodes,
    session,
}: DecisionViewProps) => {
    const { palette } = useTheme();

    /**
     * Pair each link with its "to" node
     */
    const decisions = useMemo<Decision[]>(() => {
        return data.links.map(link => {
            const node = nodes.find(n => n.id === link.toId);
            return { node, link } as Decision;
        });
    }, [data.links, nodes]);

    /**
     * Navigate to chosen option
     */
    const toDecision = useCallback((index: number) => {
        // Find decision
        const decision = decisions[index];
        handleDecisionSelect(decision.node);
    }, [decisions, handleDecisionSelect]);

    return (
        <Stack direction="column" spacing={4}>
            <Typography variant="h4">What would you like to do next?</Typography>
            {/* Each decision as its own ListItem, with title and description */}
            {decisions.map((decision, index) => {
                const languages = getUserLanguages(session);
                const title = getTranslation(decision.node, 'title', languages, true)
                const description = getTranslation(decision.node, 'description', languages, true);
                return (<ListItem
                    disablePadding
                    onClick={() => { toDecision(index); }}
                    sx={{
                        display: 'flex',
                        background: palette.secondary.light,
                        color: 'black',
                        ...containerShadow,
                        borderRadius: '12px',
                    }}
                >
                    <ListItemButton component="div" onClick={() => { toDecision(index); }}>
                        <Stack direction="column" spacing={1} pl={2} sx={{ width: '-webkit-fill-available', alignItems: 'center' }}>
                            {/* Name/Title */}
                            <ListItemText
                                primary={title}
                                sx={{ ...multiLineEllipsis(1) }}
                            />
                            {/* Bio/Description */}
                            {description && <ListItemText
                                primary={description}
                                sx={{ ...multiLineEllipsis(2), color: palette.text.secondary }}
                            />}
                        </Stack>
                        <OpenLinkIcon />
                    </ListItemButton>
                </ListItem>)
            })}
        </Stack>
    )
}