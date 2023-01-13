import { ListItem, ListItemButton, ListItemText, Stack, Typography, useTheme } from "@mui/material";
import { useCallback, useMemo } from "react";
import { multiLineEllipsis } from "styles";
import { getTranslation, getUserLanguages } from "utils";
import { DecisionViewProps } from "../types";
import { HelpButton } from "components/buttons";
import { OpenInNewIcon } from "@shared/icons";
import { Node, NodeLink, NodeType } from "@shared/consts";

const helpText = 
`The routine has encountered multiple possible paths to take, with no way to decide automatically which one to take. 

Paths which lead to another subroutine are shown in blue. Paths that end the routine are either red or green, depending on whether the routine will be marked as successful or not.

Please select the path you would like to take. You can always navigate back to this decision later through the "Steps" side menu.`;

type Decision = {
    node: Node;
    link: NodeLink;
    color: string;
};

export const DecisionView = ({
    data,
    handleDecisionSelect,
    nodes,
    session,
    zIndex,
}: DecisionViewProps) => {
    const { palette } = useTheme();

    /**
     * Pair each link with its "to" node
     */
    const decisions = useMemo<Decision[]>(() => {
        return data.links.map(link => {
            const node = nodes.find(n => n.id === link.to.id);
            let color = palette.primary.dark;
            if (node?.nodeType === NodeType.End) {
                color = node.end?.wasSuccessful === false ? '#7c262a' : '#387e30'
            }
            return { node, link, color } as Decision;
        });
    }, [data.links, nodes, palette.primary.dark]);

    /**
     * Navigate to chosen option
     */
    const toDecision = useCallback((index: number) => {
        // Find decision
        const decision = decisions[index];
        handleDecisionSelect(decision.node);
    }, [decisions, handleDecisionSelect]);

    return (
        <Stack direction="column" spacing={4} p={2}>
            {/* Title and help button */}
            <Stack
                direction="row"
                justifyContent="center"
                alignItems="center"
            >
                <Typography variant="h4" sx={{ textAlign: 'center' }}>What would you like to do next?</Typography>
                <HelpButton markdown={helpText} sx={{ width: '40px', height: '40px' }} />
            </Stack>
            {/* Each decision as its own ListItem, with name and description */}
            {decisions.map((decision, index) => {
                const languages = getUserLanguages(session);
                const { description, name } = getTranslation(decision.node, languages, true);
                return (<ListItem
                    disablePadding
                    onClick={() => { toDecision(index); }}
                    sx={{
                        display: 'flex',
                        background: decision.color,
                        color: 'white',
                        boxShadow: 12,
                        borderRadius: '12px',
                    }}
                >
                    <ListItemButton component="div" onClick={() => { toDecision(index); }}>
                        <Stack direction="column" spacing={1} pl={2} sx={{ width: '-webkit-fill-available', alignItems: 'center' }}>
                            {/* Name/Title */}
                            <ListItemText
                                primary={name}
                                sx={{ 
                                    ...multiLineEllipsis(1), 
                                    fontWeight: 'bold',
                                }}
                            />
                            {/* Bio/Description */}
                            {description && <ListItemText
                                primary={description}
                                sx={{ 
                                    ...multiLineEllipsis(2), 
                                }}
                            />}
                        </Stack>
                        <OpenInNewIcon fill="white" />
                    </ListItemButton>
                </ListItem>)
            })}
        </Stack>
    )
}