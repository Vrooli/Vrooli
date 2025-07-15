import React from "react";
import { vi } from "vitest";

// Mock implementations for @hello-pangea/dnd components
// These provide the essential structure and props needed for tests
// without the complexity of actual drag and drop functionality

export const helloPangeaDndMock = {
    DragDropContext: ({ children, onDragEnd }: any) => {
        // Simple wrapper that provides DnD context without actual functionality
        return React.createElement("div", { 
            "data-testid": "drag-drop-context",
            "data-mock": "hello-pangea-dnd", 
        }, children);
    },
    
    Droppable: ({ children, droppableId, direction, type }: any) => {
        // Mock droppable that provides the expected provided object
        const provided = {
            innerRef: vi.fn(),
            droppableProps: {
                "data-testid": `droppable-${droppableId}`,
                "data-droppable-id": droppableId,
                "data-direction": direction,
                "data-type": type,
            },
            placeholder: React.createElement("div", { 
                "data-testid": "droppable-placeholder",
                style: { display: "none" },
            }),
        };
        
        return React.createElement("div", {
            "data-testid": `droppable-${droppableId}`,
            "data-mock": "droppable",
        }, children(provided));
    },
    
    Draggable: ({ children, draggableId, index }: any) => {
        // Mock draggable that provides the expected provided object
        const provided = {
            innerRef: vi.fn(),
            draggableProps: {
                "data-testid": `draggable-${draggableId}`,
                "data-draggable-id": draggableId,
                "data-index": index,
            },
            dragHandleProps: {
                "data-testid": `drag-handle-${draggableId}`,
                "data-drag-handle": "true",
                // Add role and tabIndex for accessibility testing
                role: "button",
                tabIndex: 0,
                "aria-label": `Drag handle for ${draggableId}`,
            },
        };
        
        const snapshot = {
            isDragging: false,
            isDropAnimating: false,
            dropAnimation: null,
            draggingOver: null,
            combineWith: null,
            combineTargetFor: null,
            mode: null,
        };
        
        return React.createElement("div", {
            "data-testid": `draggable-${draggableId}`,
            "data-mock": "draggable",
        }, children(provided, snapshot));
    },
};

// Mock the entire module
export default helloPangeaDndMock;
