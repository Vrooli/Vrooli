import { NODE_TYPES } from "@local/shared";

export interface AddNodeProps {
    options: NODE_TYPES[];
    onAdd: (type: NODE_TYPES) => void;
}

export interface CombineNodeProps {}

export interface DecisionNodeProps {
    label?: string;
    text?: string;
    labelVisible?: boolean;
}

export interface EndNodeProps {
    label?: string;
    labelVisible?: boolean;
}

export interface LoopNodeProps {}

export interface RedirectNodeProps {}

export interface RoutineListNodeProps {}

export interface StartNodeProps {
    label?: string;
    labelVisible?: boolean;
}