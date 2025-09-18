// Drag and Drop Handler Module
export class DragDropHandler {
    constructor(onDrop) {
        this.onDrop = onDrop;
        this.draggedElement = null;
        this.draggedTaskId = null;
        this.draggedFromStatus = null;
    }

    initializeDragDrop() {
        // Set up drop zones
        document.querySelectorAll('.kanban-column').forEach(column => {
            column.addEventListener('dragover', (e) => this.handleDragOver(e));
            column.addEventListener('drop', (e) => this.handleDrop(e));
            column.addEventListener('dragleave', (e) => this.handleDragLeave(e));
        });
    }

    setupTaskCardDragHandlers(card, taskId, status) {
        card.addEventListener('dragstart', (e) => this.handleDragStart(e, taskId, status));
        card.addEventListener('dragend', (e) => this.handleDragEnd(e));
    }

    handleDragStart(e, taskId, status) {
        this.draggedElement = e.target;
        this.draggedTaskId = taskId;
        this.draggedFromStatus = status;
        
        e.dataTransfer.effectAllowed = 'move';
        e.dataTransfer.setData('text/html', e.target.innerHTML);
        
        // Add dragging class
        e.target.classList.add('dragging');
        
        // Set drag image
        const dragImage = e.target.cloneNode(true);
        dragImage.style.transform = 'rotate(-2deg)';
        dragImage.style.opacity = '0.8';
        document.body.appendChild(dragImage);
        dragImage.style.position = 'absolute';
        dragImage.style.top = '-1000px';
        e.dataTransfer.setDragImage(dragImage, e.offsetX, e.offsetY);
        setTimeout(() => document.body.removeChild(dragImage), 0);
    }

    handleDragOver(e) {
        if (e.preventDefault) {
            e.preventDefault();
        }
        
        e.dataTransfer.dropEffect = 'move';
        
        const column = e.target.closest('.kanban-column');
        if (column && !column.classList.contains('drag-over')) {
            // Remove drag-over class from all columns
            document.querySelectorAll('.kanban-column').forEach(col => {
                col.classList.remove('drag-over');
            });
            // Add drag-over class to current column
            column.classList.add('drag-over');
        }
        
        return false;
    }

    handleDragLeave(e) {
        const column = e.target.closest('.kanban-column');
        if (column && !column.contains(e.relatedTarget)) {
            column.classList.remove('drag-over');
        }
    }

    async handleDrop(e) {
        if (e.stopPropagation) {
            e.stopPropagation();
        }
        
        const column = e.target.closest('.kanban-column');
        if (!column) return false;
        
        const targetStatus = column.dataset.status;
        
        // Remove drag-over class
        column.classList.remove('drag-over');
        
        if (this.draggedElement && this.draggedTaskId && targetStatus !== this.draggedFromStatus) {
            if (this.onDrop) {
                await this.onDrop(this.draggedTaskId, this.draggedFromStatus, targetStatus);
            }
        }
        
        return false;
    }

    handleDragEnd(e) {
        // Remove dragging class
        e.target.classList.remove('dragging');
        
        // Remove drag-over class from all columns
        document.querySelectorAll('.kanban-column').forEach(column => {
            column.classList.remove('drag-over');
        });
        
        // Reset drag state
        this.draggedElement = null;
        this.draggedTaskId = null;
        this.draggedFromStatus = null;
    }
}