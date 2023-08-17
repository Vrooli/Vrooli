import { NodeLink } from "@local/shared";
import { ListItem, ListItemButton, ListItemText, Stack, Typography, useTheme } from "@mui/material";
import { OpenInNewIcon } from "icons";
import { useCallback, useMemo } from "react";
import { multiLineEllipsis } from "styles";
import { EndStep, RoutineStep } from "types";
import { RoutineStepType } from "utils/consts";
import { toDisplay } from "utils/display/pageTools";
import { DecisionViewProps } from "../types";

type Decision = {
    step: RoutineStep | EndStep;
    link: NodeLink;
    color: string;
};

export const DecisionView = ({
    data,
    isOpen,
    handleDecisionSelect,
    routineList,
    zIndex,
}: DecisionViewProps) => {
    const { palette } = useTheme();
    const display = toDisplay(isOpen);

    /**
     * Pair each link with its "to" node
     */
    const decisions = useMemo<Decision[]>(() => {
        const decisions: Decision[] = [];
        // Find corresponding step for each link
        for (const link of (data?.links ?? [])) {
            // If link points to step in "steps", it's not the end of the routine
            const step = routineList?.steps?.find(s =>
                (s.type === RoutineStepType.Subroutine ||
                    s.type === RoutineStepType.RoutineList) &&
                s.nodeId === link.to.id);
            if (step) {
                decisions.push({ color: palette.primary.dark, step, link });
            }
            // If link points to step in "endSteps", it's the end of the routine
            const endStep = routineList.endSteps.find(s => s.nodeId === link.to.id);
            if (endStep) {
                decisions.push({
                    color: endStep?.wasSuccessful === false ? "#7c262a" : "#387e30",
                    step: endStep,
                    link,
                });
            }
            // If not found, it's an error
            console.error(`Could not find step for link ${link.id}`);
        }
        return decisions;
    }, [data.links, routineList.steps, routineList.endSteps, palette.primary.dark]);


    /**
     * Navigate to chosen option
     */
    const toDecision = useCallback((index: number) => {
        const decision = decisions[index];
        handleDecisionSelect(decision.step);
    }, [decisions, handleDecisionSelect]);

    return (
        <Stack direction="column" spacing={4} p={2}>
            {/* Title and help button */}
            <Stack
                direction="row"
                justifyContent="center"
                alignItems="center"
            >
                <Typography variant="h4" sx={{ textAlign: "center" }}>What would you like to do next?</Typography>
            </Stack>
            {/* Each decision as its own ListItem, with name and description */}
            {decisions.map((decision, index) => {
                const { description, name } = decision.step;
                return (<ListItem
                    disablePadding
                    onClick={() => { toDecision(index); }}
                    sx={{
                        display: "flex",
                        background: decision.color,
                        color: "white",
                        boxShadow: 12,
                        borderRadius: "12px",
                    }}
                >
                    <ListItemButton component="div" onClick={() => { toDecision(index); }}>
                        <Stack direction="column" spacing={1} pl={2} sx={{ width: "-webkit-fill-available", alignItems: "center" }}>
                            {/* Name/Title */}
                            <ListItemText
                                primary={name}
                                sx={{
                                    ...multiLineEllipsis(1),
                                    fontWeight: "bold",
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
                </ListItem>);
            })}
        </Stack>
    );
};
