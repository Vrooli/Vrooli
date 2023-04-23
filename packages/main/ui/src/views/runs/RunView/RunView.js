import { NodeType } from "@local/consts";
import { RoutineStepType } from "../../../utils/consts";
import { getTranslation } from "../../../utils/display/translationTools";
import { routineVersionHasSubroutines } from "../../../utils/runUtils";
const MAX_NESTING = 20;
const insertStep = (stepData, steps) => {
    const step = steps;
    for (let i = 0; i < step.steps.length; i++) {
        const currentStep = step.steps[i];
        if (currentStep.type === RoutineStepType.Subroutine) {
            if (currentStep.routineVersion.id === stepData.routineVersionId) {
                step.steps[i] = stepData;
            }
        }
        else if (currentStep.type === RoutineStepType.RoutineList) {
            step.steps[i] = insertStep(stepData, currentStep);
        }
    }
    return step;
};
const locationFromNodeId = (nodeId, step, location = []) => {
    if (!step)
        return null;
    if (step.type === RoutineStepType.RoutineList) {
        if (step?.nodeId === nodeId)
            return location;
        const stepList = step;
        for (let i = 1; i <= stepList.steps.length; i++) {
            const currStep = stepList.steps[i - 1];
            if (currStep.type === RoutineStepType.RoutineList) {
                const currLocation = locationFromNodeId(nodeId, currStep, [...location, i]);
                if (currLocation)
                    return currLocation;
            }
        }
    }
    return null;
};
const locationFromRoutineId = (routineId, step, location = []) => {
    if (!step)
        return null;
    if (step.type === RoutineStepType.Subroutine) {
        if (step?.routineVersion?.id === routineId)
            return [...location, 1];
    }
    else if (step.type === RoutineStepType.RoutineList) {
        const stepList = step;
        for (let i = 1; i <= stepList.steps.length; i++) {
            const currStep = stepList.steps[i - 1];
            if (currStep.type === RoutineStepType.RoutineList) {
                const currLocation = locationFromRoutineId(routineId, currStep, [...location, i]);
                if (currLocation)
                    return currLocation;
            }
        }
    }
    return null;
};
const stepFromLocation = (locationArray, steps) => {
    if (!steps)
        return null;
    let currNestedSteps = steps;
    if (locationArray.length > MAX_NESTING) {
        console.error(`Location array too large in findStep: ${locationArray}`);
        return null;
    }
    for (let i = 0; i < locationArray.length; i++) {
        if (currNestedSteps !== null && currNestedSteps.type === RoutineStepType.RoutineList) {
            currNestedSteps = currNestedSteps.steps.length > Math.max(locationArray[i] - 1, 0) ?
                currNestedSteps.steps[Math.max(locationArray[i] - 1, 0)] :
                null;
        }
    }
    return currNestedSteps;
};
const subroutineNeedsQuerying = (step) => {
    if (!step || step.type !== RoutineStepType.Subroutine)
        return false;
    const currSubroutine = step.routineVersion;
    return routineVersionHasSubroutines(currSubroutine);
};
const getStepComplexity = (step) => {
    switch (step.type) {
        case RoutineStepType.Decision:
            return 1;
        case RoutineStepType.Subroutine:
            return step.routineVersion.complexity;
        case RoutineStepType.RoutineList:
            return step.steps.reduce((acc, curr) => acc + getStepComplexity(curr), 0);
        default:
            return 0;
    }
};
const convertRoutineVersionToStep = (routineVersion, languages) => {
    if (!routineVersion || !routineVersion.nodes || !routineVersion.nodeLinks) {
        console.log("routineVersion does not have enough data to calculate steps");
        return null;
    }
    let routineListNodes = routineVersion.nodes.filter(({ nodeType }) => nodeType === NodeType.RoutineList);
    const startNode = routineVersion.nodes.find((node) => node.nodeType === NodeType.Start);
    routineListNodes = routineListNodes.sort((a, b) => {
        const aCol = a.columnIndex ?? 0;
        const bCol = b.columnIndex ?? 0;
        if (aCol !== bCol)
            return aCol - bCol;
        const aRow = a.rowIndex ?? 0;
        const bRow = b.rowIndex ?? 0;
        return aRow - bRow;
    });
    const resultSteps = [];
    const startLinks = routineVersion.nodeLinks.filter((link) => link.from.id === startNode?.id);
    if (startLinks.length > 1) {
        resultSteps.push({
            type: RoutineStepType.Decision,
            links: startLinks,
            name: "Decision",
            description: "Select a subroutine to run next",
        });
    }
    for (const node of routineListNodes) {
        const subroutineSteps = [...node.routineList.items]
            .sort((r1, r2) => r1.index - r2.index)
            .map((item) => ({
            type: RoutineStepType.Subroutine,
            index: item.index,
            routineVersion: item.routineVersion,
            name: getTranslation(item.routineVersion, languages, true).name ?? "Untitled",
            description: getTranslation(item.routineVersion, languages, true).description ?? "Description not found matching selected language",
        }));
        const links = routineVersion.nodeLinks.filter((link) => link.from.id === node.id);
        const decisionSteps = links.length > 1 ? [{
                type: RoutineStepType.Decision,
                links,
                name: "Decision",
                description: "Select a subroutine to run next",
            }] : [];
        resultSteps.push({
            type: RoutineStepType.RoutineList,
            nodeId: node.id,
            isOrdered: node.routineList?.isOrdered ?? false,
            name: getTranslation(node, languages, true).name ?? "Untitled",
            description: getTranslation(node, languages, true).description ?? "Description not found matching selected language",
            steps: [...subroutineSteps, ...decisionSteps],
        });
    }
    return {
        type: RoutineStepType.RoutineList,
        routineVersionId: routineVersion.id,
        isOrdered: true,
        name: getTranslation(routineVersion, languages, true).name ?? "Untitled",
        description: getTranslation(routineVersion, languages, true).description ?? "Description not found matching selected language",
        steps: resultSteps,
    };
};
export const RunView = ({ display = "page", handleClose, runnableObject, zIndex, }) => {
    return {};
};
//# sourceMappingURL=RunView.js.map