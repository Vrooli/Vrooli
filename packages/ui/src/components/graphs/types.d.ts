import { NodeData } from '@local/shared';

export interface BarGraphProps {
    className?: string;
    data?: any;
    dimensions?: Dimensions;
    margins?: Margins;
}

export interface NodeGraphProps {
    scale?: number;
    isEditable?: boolean;
    labelVisible?: boolean;
    nodes?: NodeData[]
}

/**
 * Props for the Node Column (a container for displaying nodes on separate branches)
 */
 export interface NodeGraphColumnProps {
    scale?: number;
    isEditable?: boolean;
    labelVisible?: boolean;
    columnNumber: number;
    nodes: NodeData[];
}