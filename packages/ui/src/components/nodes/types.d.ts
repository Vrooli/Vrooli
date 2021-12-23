import { NODE_TYPES } from "@local/shared";

export interface BaseNodeProps {
    scale?: number;
}

export interface AddNodeProps extends BaseNodeProps {
    options: NODE_TYPES[];
    onAdd: (type: NODE_TYPES) => void;
}

export interface CombineNodeProps extends BaseNodeProps {}

export interface DecisionNodeProps extends BaseNodeProps {
    label?: string;
    text?: string;
    labelVisible?: boolean;
}

export interface EndNodeProps extends BaseNodeProps {
    label?: string;
    labelVisible?: boolean;
}

export interface LoopNodeProps extends BaseNodeProps {
    label?: string;
    labelVisible?: boolean;
}

export interface RedirectNodeProps extends BaseNodeProps {}

export interface RoutineListNodeProps extends BaseNodeProps {}

export interface StartNodeProps extends BaseNodeProps {
    label?: string;
    labelVisible?: boolean;
}