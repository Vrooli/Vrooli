import { NodeGraphEdgeProps } from '../types';

// Displays a line between two nodes.
// If in editing mode, an "Add Node" button appears on the line. 
// This button always appears inbetween two node columns, to avoid collisions with nodes.
export const NodeGraphEdge = ({
    start,
    end,
    isEditable = true,
    onAdd,
}: NodeGraphEdgeProps) => {
    console.log('node edge', start, end)

    // Determines 
    const dimensions = {
        width: end.x - start.x,
        height: end.y - start.y,
    }

    return (
        <svg 
            {...dimensions} 
            style={{zIndex:2}}
        >
            <line x1="0" y1="0" x2={dimensions.width} y2={dimensions.height} stroke="black"/>
        </svg>
    )
}