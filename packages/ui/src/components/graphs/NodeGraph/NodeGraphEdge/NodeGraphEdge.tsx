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
    // Determines 
    const dimensions = {
        width: Math.abs(end.x - start.x),
        height: Math.abs(end.y - start.y),
    }

    return (
        <svg 
            width={dimensions.width}
            // Extra height to make sure straight lines are drawn
            height={dimensions.height + 2} 
            style={{ 
                zIndex:2, 
                position: "absolute",
                transform: `translate(${Math.min(start.x, end.x)}px,${Math.min(start.y, end.y)}px)`,
            }}
        >
            <line x1="0" y1="0" x2={dimensions.width} y2={dimensions.height} stroke="black" strokeWidth={3}/>
        </svg>
    )
}