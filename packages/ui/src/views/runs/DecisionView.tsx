import { type DecisionOption } from "@local/shared";
import { ListItem, type ListItemProps, Stack, Typography, styled, useTheme } from "@mui/material";
import { useCallback, useMemo } from "react";
import { multiLineEllipsis } from "../../styles.js";
import { type DecisionViewProps } from "./types.js";

type Decision = DecisionOption & {
    color: string;
};

interface DecisionListItemProps extends ListItemProps {
    color: string;
}

const DecisionListItem = styled(ListItem, {
    shouldForwardProp: (prop) => prop !== "color",
})<DecisionListItemProps>(({ color, theme }) => ({
    display: "flex",
    background: color,
    color: "white",
    boxShadow: theme.shadows[4],
    borderRadius: "12px",
}));

const titleLabelStyle = { textAlign: "center" } as const;
const decisionTextStackStyle = { width: "-webkit-fill-available", alignItems: "center" } as const;
const decisionNameStyle = { ...multiLineEllipsis(1), fontWeight: "bold" } as const;
const decisionDescriptionStyle = { ...multiLineEllipsis(2) } as const;

export function DecisionView({
    data,
    handleDecisionSelect,
}: DecisionViewProps) {
    const { palette } = useTheme();

    /** Pair each link with its "to" node */
    const decisions = useMemo<Decision[]>(function decisionsMemo() {
        // const decisions: Decision[] = [];
        if (!data || !Array.isArray(data.options)) return [];
        return data.options.map((option) => {
            const color = palette.primary.dark;
            // // End steps are colored based on success
            // if (option.step.__type === RunStepType.End) {
            //     color = option.step.wasSuccessful ? "#387e30" : "#7c262a";
            // }
            return { color, ...option };
        });
    }, [data, palette.primary.dark]);

    /**
     * Navigate to chosen option
     */
    const toDecision = useCallback(function toDecisionCallback(index: number) {
        const decision = decisions[index];
        // handleDecisionSelect(decision.step);
    }, [decisions, handleDecisionSelect]);

    return (
        <Stack direction="column" spacing={4} p={2}>
            {/* Title and help button */}
            <Stack
                direction="row"
                justifyContent="center"
                alignItems="center"
            >
                <Typography variant="h4" sx={titleLabelStyle}>What would you like to do next?</Typography>
            </Stack>
            {/* Each decision as its own ListItem, with name and description */}
            {decisions.map((decision, index) => {
                // const { description, name } = decision.step;
                // function handleClick() {
                //     toDecision(index);
                // }

                // return (<DecisionListItem
                //     color={decision.color}
                //     disablePadding
                //     key={`decision-${decision.nodeId}`}
                //     onClick={handleClick}
                // >
                //     <ListItemButton component="div" onClick={handleClick}>
                //         <Stack direction="column" spacing={1} pl={2} sx={decisionTextStackStyle}>
                //             <ListItemText primary={name} sx={decisionNameStyle} />
                //             {description && <ListItemText primary={description} sx={decisionDescriptionStyle} />}
                //         </Stack>
                //         <OpenInNewIcon fill="white" />
                //     </ListItemButton>
                // </DecisionListItem>);
                return null;
            })}
        </Stack>
    );
}
