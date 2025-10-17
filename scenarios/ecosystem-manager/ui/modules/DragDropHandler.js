// Drag and Drop Handler Module
export class DragDropHandler {
    constructor(onDrop) {
        this.onDrop = onDrop;
        this.draggedElement = null;
        this.draggedTaskId = null;
        this.draggedFromStatus = null;
        this.autoScrollEdgeThreshold = 80;
        this.autoScrollEdgeThresholdHorizontal = 100;
        this.autoScrollVelocity = { x: 0, y: 0 };
        this.autoScrollAnimationFrame = null;
        this.touchDragState = null;
        this.kanbanBoard = null;
        this.touchDragThreshold = 12;

        this.handlePointerMove = this.handlePointerMove.bind(this);
        this.handlePointerUp = this.handlePointerUp.bind(this);
    }

    initializeDragDrop() {
        this.kanbanBoard = document.querySelector('.kanban-board');

        // Set up drop zones
        document.querySelectorAll('.kanban-column').forEach(column => {
            column.addEventListener('dragover', (e) => this.handleDragOver(e));
            column.addEventListener('drop', (e) => this.handleDrop(e));
            column.addEventListener('dragleave', (e) => this.handleDragLeave(e));
        });
    }

    setupTaskCardDragHandlers(card, taskId, status) {
        card.draggable = false;
        card.addEventListener('dragstart', (e) => this.handleDragStart(e, taskId, status));
        card.addEventListener('dragend', (e) => this.handleDragEnd(e));
        card.addEventListener('pointerdown', (e) => this.handlePointerDown(e, card, taskId, status));

        const dragHandle = card.querySelector('.task-drag-handle');
        if (dragHandle) {
            dragHandle.addEventListener('click', (event) => {
                event.preventDefault();
                event.stopPropagation();
            });
        }
    }

    handleDragStart(e, taskId, status) {
        const card = e.currentTarget;
        if (card.dataset.mouseDragState !== 'pending') {
            e.preventDefault();
            return;
        }

        card.dataset.mouseDragState = 'active';
        this.draggedElement = card;
        this.draggedTaskId = taskId;
        this.draggedFromStatus = status;
        
        e.dataTransfer.effectAllowed = 'move';
        e.dataTransfer.setData('text/html', card.innerHTML);
        
        // Add dragging class
        card.classList.add('dragging');
        
        // Set drag image
        const dragImage = card.cloneNode(true);
        dragImage.style.transform = 'rotate(-2deg)';
        dragImage.style.opacity = '0.8';
        document.body.appendChild(dragImage);
        dragImage.style.position = 'absolute';
        dragImage.style.top = '-1000px';
        const rect = card.getBoundingClientRect();
        const offsetX = Number.isFinite(e.clientX) ? e.clientX - rect.left : rect.width / 2;
        const offsetY = Number.isFinite(e.clientY) ? e.clientY - rect.top : rect.height / 2;
        e.dataTransfer.setDragImage(dragImage, offsetX, offsetY);
        setTimeout(() => document.body.removeChild(dragImage), 0);
    }

    handleDragOver(e) {
        if (e.preventDefault) {
            e.preventDefault();
        }

        e.dataTransfer.dropEffect = 'move';

        if (typeof e.clientX === 'number' || typeof e.clientY === 'number') {
            this.maybeAutoScroll(e.clientX, e.clientY);
        }

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

        if (!column) {
            this.stopAutoScroll();
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
        this.stopAutoScroll();

        if (this.draggedElement && this.draggedTaskId && targetStatus !== this.draggedFromStatus) {
            if (this.onDrop) {
                await this.onDrop(this.draggedTaskId, this.draggedFromStatus, targetStatus);
            }
        }
        
        return false;
    }

    handleDragEnd(e) {
        const card = e.currentTarget;
        // Remove dragging class
        card.classList.remove('dragging');

        // Remove drag-over class from all columns
        document.querySelectorAll('.kanban-column').forEach(column => {
            column.classList.remove('drag-over');
        });

        // Reset drag state
        this.draggedElement = null;
        this.draggedTaskId = null;
        this.draggedFromStatus = null;
        this.stopAutoScroll();

        delete card.dataset.mouseDragState;
        card.draggable = false;
    }

    handlePointerDown(e, card, taskId, status) {
        const handle = e.target.closest('.task-drag-handle');
        if (!handle) {
            return;
        }

        e.stopPropagation();

        if (e.pointerType !== 'touch') {
            card.dataset.mouseDragState = 'pending';
            card.draggable = true;

            const onPointerEnd = () => {
                if (card.dataset.mouseDragState === 'pending') {
                    delete card.dataset.mouseDragState;
                    card.draggable = false;
                }
                card.removeEventListener('pointerup', onPointerEnd);
                card.removeEventListener('pointercancel', onPointerEnd);
            };

            card.addEventListener('pointerup', onPointerEnd, { once: true });
            card.addEventListener('pointercancel', onPointerEnd, { once: true });
            return;
        }

        if (!this.kanbanBoard) {
            this.kanbanBoard = document.querySelector('.kanban-board');
        }

        const rect = card.getBoundingClientRect();
        const offsetX = e.clientX - rect.left;
        const offsetY = e.clientY - rect.top;

        this.touchDragState = {
            pointerId: e.pointerId,
            card,
            taskId,
            fromStatus: status,
            startX: e.clientX,
            startY: e.clientY,
            offsetX,
            offsetY,
            ghost: null,
            dragging: false
        };

        card.setPointerCapture?.(e.pointerId);

        document.addEventListener('pointermove', this.handlePointerMove, { passive: false });
        document.addEventListener('pointerup', this.handlePointerUp, { passive: false });
        document.addEventListener('pointercancel', this.handlePointerUp, { passive: false });
    }

    handlePointerMove(e) {
        if (!this.touchDragState || e.pointerId !== this.touchDragState.pointerId) {
            return;
        }

        const state = this.touchDragState;
        const deltaX = Math.abs(e.clientX - state.startX);
        const deltaY = Math.abs(e.clientY - state.startY);

        if (!state.dragging) {
            if (deltaX <= this.touchDragThreshold && deltaY <= this.touchDragThreshold) {
                return;
            }
            this.activateTouchDrag(e);
        }

        if (!state.dragging) {
            return;
        }

        e.preventDefault();

        const { ghost, offsetX, offsetY } = state;
        this.updateTouchGhostPosition(ghost, e.clientX, e.clientY, offsetX, offsetY);
        this.highlightColumnAtPoint(e.clientX, e.clientY);
        this.maybeAutoScroll(e.clientX, e.clientY);
    }

    async handlePointerUp(e) {
        if (!this.touchDragState || e.pointerId !== this.touchDragState.pointerId) {
            return;
        }

        const state = this.touchDragState;
        const wasDragging = state.dragging;

        if (wasDragging) {
            e.preventDefault();
        }

        const { card, taskId, fromStatus, ghost } = state;
        card.classList.remove('dragging');
        card.releasePointerCapture?.(e.pointerId);
        card.draggable = false;

        if (ghost && ghost.parentElement) {
            ghost.parentElement.removeChild(ghost);
        }

        if (wasDragging) {
            const dropTarget = document.elementFromPoint(e.clientX, e.clientY);
            const column = dropTarget ? dropTarget.closest('.kanban-column') : null;

            this.clearDragHighlights();
            this.stopAutoScroll();

            if (column) {
                const targetStatus = column.dataset.status;
                if (targetStatus && targetStatus !== fromStatus && this.onDrop) {
                    await this.onDrop(taskId, fromStatus, targetStatus);
                }
            }
        }

        document.removeEventListener('pointermove', this.handlePointerMove);
        document.removeEventListener('pointerup', this.handlePointerUp);
        document.removeEventListener('pointercancel', this.handlePointerUp);

        this.touchDragState = null;
    }

    activateTouchDrag(e) {
        if (!this.touchDragState) {
            return;
        }

        const state = this.touchDragState;
        if (state.dragging) {
            return;
        }

        const { card, offsetX, offsetY } = state;
        e.preventDefault();
        const rect = card.getBoundingClientRect();
        const ghost = this.createTouchGhost(card, rect, {
            initialX: e.clientX,
            initialY: e.clientY,
            offsetX,
            offsetY
        });

        card.classList.add('dragging');

        state.ghost = ghost;
        state.dragging = true;

        this.updateTouchGhostPosition(ghost, e.clientX, e.clientY, offsetX, offsetY);
        this.highlightColumnAtPoint(e.clientX, e.clientY);
        this.maybeAutoScroll(e.clientX, e.clientY);
    }

    maybeAutoScroll(clientX, clientY) {
        const viewportHeight = window.innerHeight || document.documentElement.clientHeight;
        const viewportWidth = window.innerWidth || document.documentElement.clientWidth;

        let vertical = 0;
        if (typeof clientY === 'number' && !Number.isNaN(clientY)) {
            if (clientY < this.autoScrollEdgeThreshold) {
                vertical = -1;
            } else if (clientY > viewportHeight - this.autoScrollEdgeThreshold) {
                vertical = 1;
            }
        }

        let horizontal = 0;
        if (typeof clientX === 'number' && !Number.isNaN(clientX)) {
            if (clientX < this.autoScrollEdgeThresholdHorizontal) {
                horizontal = -1;
            } else if (clientX > viewportWidth - this.autoScrollEdgeThresholdHorizontal) {
                horizontal = 1;
            }
        }

        if (horizontal === 0 && vertical === 0) {
            this.stopAutoScroll();
        } else {
            this.startAutoScroll(horizontal, vertical);
        }
    }

    startAutoScroll(horizontal, vertical) {
        this.autoScrollVelocity = { x: horizontal, y: vertical };

        if (this.autoScrollAnimationFrame) {
            return;
        }

        const step = () => {
            const { x, y } = this.autoScrollVelocity;
            if (x === 0 && y === 0) {
                this.stopAutoScroll();
                return;
            }

            if (y !== 0) {
                const scrollAmountY = y * 14;
                window.scrollBy({ top: scrollAmountY, left: 0, behavior: 'auto' });
            }

            if (x !== 0 && (this.kanbanBoard || document.querySelector('.kanban-board'))) {
                if (!this.kanbanBoard) {
                    this.kanbanBoard = document.querySelector('.kanban-board');
                }
                if (this.kanbanBoard) {
                    const scrollAmountX = x * 18;
                    this.kanbanBoard.scrollLeft += scrollAmountX;
                }
            }

            this.autoScrollAnimationFrame = window.requestAnimationFrame(step);
        };

        this.autoScrollAnimationFrame = window.requestAnimationFrame(step);
    }

    stopAutoScroll() {
        this.autoScrollVelocity = { x: 0, y: 0 };
        if (this.autoScrollAnimationFrame) {
            window.cancelAnimationFrame(this.autoScrollAnimationFrame);
            this.autoScrollAnimationFrame = null;
        }
    }

    createTouchGhost(card, rect, options = {}) {
        const ghost = card.cloneNode(true);
        ghost.removeAttribute('id');
        ghost.style.width = `${rect.width}px`;
        ghost.style.height = `${rect.height}px`;
        ghost.style.opacity = '0.9';

        const { initialX, initialY, offsetX = 0, offsetY = 0 } = options;
        if (typeof initialX === 'number' && typeof initialY === 'number') {
            ghost.style.transform = `translate3d(${initialX - offsetX}px, ${initialY - offsetY}px, 0)`;
        } else {
            ghost.style.transform = 'translate3d(-9999px, -9999px, 0)';
        }

        ghost.classList.add('touch-drag-ghost');
        ghost.style.transition = 'none';
        document.body.appendChild(ghost);
        requestAnimationFrame(() => {
            ghost.style.transition = 'transform 0.05s ease-out';
        });
        return ghost;
    }

    updateTouchGhostPosition(ghost, clientX, clientY, offsetX = 0, offsetY = 0) {
        if (!ghost) return;
        const x = clientX - offsetX;
        const y = clientY - offsetY;
        ghost.style.transform = `translate3d(${x}px, ${y}px, 0)`;
    }

    highlightColumnAtPoint(clientX, clientY) {
        const element = document.elementFromPoint(clientX, clientY);
        const column = element ? element.closest('.kanban-column') : null;
        this.clearDragHighlights();
        if (column) {
            column.classList.add('drag-over');
        }
    }

    clearDragHighlights() {
        document.querySelectorAll('.kanban-column.drag-over').forEach(column => {
            column.classList.remove('drag-over');
        });
    }
}
