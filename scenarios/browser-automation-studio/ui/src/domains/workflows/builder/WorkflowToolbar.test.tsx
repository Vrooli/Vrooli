import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ReactFlowProvider } from 'reactflow';
import WorkflowToolbar from './WorkflowToolbar';

// Mock usePopoverPosition hook
vi.mock('@hooks/usePopoverPosition', () => ({
  usePopoverPosition: () => ({
    floatingStyles: { position: 'absolute', top: 0, left: 0 },
  }),
}));

// Mock ReactFlow's useReactFlow hook
vi.mock('reactflow', async () => {
  const actual = await vi.importActual('reactflow');
  return {
    ...actual,
    useReactFlow: () => ({
      zoomIn: vi.fn(),
      zoomOut: vi.fn(),
      fitView: vi.fn(),
    }),
  };
});

describe('WorkflowToolbar [REQ:BAS-WORKFLOW-BUILDER-ZOOM] [REQ:BAS-WORKFLOW-BUILDER-UNDO-REDO]', () => {
  const defaultProps = {
    locked: false,
    onToggleLock: vi.fn(),
    onUndo: vi.fn(),
    onRedo: vi.fn(),
    canUndo: true,
    canRedo: true,
    onDuplicate: vi.fn(),
    onDelete: vi.fn(),
    graphWidth: 800,
    onConfigureViewport: vi.fn(),
    executionViewport: { width: 1920, height: 1080, preset: 'desktop' as const },
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  const renderToolbar = (props = {}) => {
    return render(
      <ReactFlowProvider>
        <WorkflowToolbar {...defaultProps} {...props} />
      </ReactFlowProvider>
    );
  };

  describe('Zoom Controls [REQ:BAS-WORKFLOW-BUILDER-ZOOM]', () => {
    it('renders zoom in button [REQ:BAS-WORKFLOW-BUILDER-ZOOM]', () => {
      renderToolbar();

      const zoomInButton = screen.getByTitle('Zoom in');
      expect(zoomInButton).toBeInTheDocument();
    });

    it('renders zoom out button [REQ:BAS-WORKFLOW-BUILDER-ZOOM]', () => {
      renderToolbar();

      const zoomOutButton = screen.getByTitle('Zoom out');
      expect(zoomOutButton).toBeInTheDocument();
    });

    it('renders fit view button in non-compact mode [REQ:BAS-WORKFLOW-BUILDER-ZOOM]', () => {
      renderToolbar({ graphWidth: 800 });

      const fitViewButton = screen.getByTitle('Fit view');
      expect(fitViewButton).toBeInTheDocument();
    });
  });

  describe('Undo/Redo Controls [REQ:BAS-WORKFLOW-BUILDER-UNDO-REDO]', () => {
    it('renders undo button [REQ:BAS-WORKFLOW-BUILDER-UNDO-REDO]', () => {
      renderToolbar();

      const undoButton = screen.getByTitle('Undo');
      expect(undoButton).toBeInTheDocument();
    });

    it('renders redo button [REQ:BAS-WORKFLOW-BUILDER-UNDO-REDO]', () => {
      renderToolbar();

      const redoButton = screen.getByTitle('Redo');
      expect(redoButton).toBeInTheDocument();
    });

    it('calls onUndo when undo button clicked [REQ:BAS-WORKFLOW-BUILDER-UNDO-REDO]', async () => {
      const user = userEvent.setup();
      const onUndo = vi.fn();

      renderToolbar({ onUndo });

      const undoButton = screen.getByTitle('Undo');
      await user.click(undoButton);

      expect(onUndo).toHaveBeenCalledTimes(1);
    });

    it('calls onRedo when redo button clicked [REQ:BAS-WORKFLOW-BUILDER-UNDO-REDO]', async () => {
      const user = userEvent.setup();
      const onRedo = vi.fn();

      renderToolbar({ onRedo });

      const redoButton = screen.getByTitle('Redo');
      await user.click(redoButton);

      expect(onRedo).toHaveBeenCalledTimes(1);
    });

    it('disables undo button when canUndo is false [REQ:BAS-WORKFLOW-BUILDER-UNDO-REDO]', () => {
      renderToolbar({ canUndo: false });

      const undoButton = screen.getByTitle('Undo');
      expect(undoButton).toBeDisabled();
      expect(undoButton.className).toContain('opacity-50');
    });

    it('disables redo button when canRedo is false [REQ:BAS-WORKFLOW-BUILDER-UNDO-REDO]', () => {
      renderToolbar({ canRedo: false });

      const redoButton = screen.getByTitle('Redo');
      expect(redoButton).toBeDisabled();
      expect(redoButton.className).toContain('opacity-50');
    });
  });

  describe('Lock/Unlock Controls', () => {
    it('renders unlock icon when not locked', () => {
      renderToolbar({ locked: false, graphWidth: 800 });

      const lockButton = screen.getByTitle('Lock editing');
      expect(lockButton).toBeInTheDocument();
    });

    it('renders lock icon when locked', () => {
      renderToolbar({ locked: true, graphWidth: 800 });

      const lockButton = screen.getByTitle('Unlock editing');
      expect(lockButton).toBeInTheDocument();
    });

    it('calls onToggleLock when lock button clicked', async () => {
      const user = userEvent.setup();
      const onToggleLock = vi.fn();

      renderToolbar({ onToggleLock, graphWidth: 800 });

      const lockButton = screen.getByTitle('Lock editing');
      await user.click(lockButton);

      expect(onToggleLock).toHaveBeenCalledTimes(1);
    });
  });

  describe('Viewport Configuration', () => {
    it('renders viewport configuration button', () => {
      renderToolbar({ graphWidth: 800 });

      const viewportButton = screen.getByTitle(/Set execution dimensions/i);
      expect(viewportButton).toBeInTheDocument();
    });

    it('displays viewport summary in title when viewport is set', () => {
      renderToolbar({
        graphWidth: 800,
        executionViewport: { width: 1920, height: 1080, preset: 'desktop' as const },
      });

      const viewportButton = screen.getByTitle('Set execution dimensions (1920 × 1080)');
      expect(viewportButton).toBeInTheDocument();
    });

    it('calls onConfigureViewport when viewport button clicked', async () => {
      const user = userEvent.setup();
      const onConfigureViewport = vi.fn();

      renderToolbar({ onConfigureViewport, graphWidth: 800 });

      const viewportButton = screen.getByTitle(/Set execution dimensions/i);
      await user.click(viewportButton);

      expect(onConfigureViewport).toHaveBeenCalledTimes(1);
    });
  });

  describe('Compact Mode', () => {
    it('shows compact layout when graphWidth < 560', () => {
      renderToolbar({ graphWidth: 500 });

      // In compact mode, fit view is in a menu instead of a direct button
      const fitViewButton = screen.queryByTitle('Fit view');
      expect(fitViewButton).not.toBeInTheDocument();

      // Should have canvas options button instead
      const canvasOptionsButton = screen.getByTitle(/Canvas options/i);
      expect(canvasOptionsButton).toBeInTheDocument();
    });

    it('shows full layout when graphWidth >= 560', () => {
      renderToolbar({ graphWidth: 800 });

      // In full mode, fit view is a direct button
      const fitViewButton = screen.getByTitle('Fit view');
      expect(fitViewButton).toBeInTheDocument();

      // Should not have canvas options dropdown
      const canvasOptionsButton = screen.queryByTitle(/Canvas options/i);
      expect(canvasOptionsButton).not.toBeInTheDocument();
    });

    it('opens layout menu in compact mode when canvas options clicked', async () => {
      const user = userEvent.setup();
      renderToolbar({ graphWidth: 500 });

      const canvasOptionsButton = screen.getByTitle(/Canvas options/i);
      await user.click(canvasOptionsButton);

      // Menu items should appear
      expect(screen.getByText('Fit to view')).toBeInTheDocument();
      expect(screen.getByText('Set dimensions')).toBeInTheDocument();
    });

    it('opens selection menu in compact mode when selection actions clicked', async () => {
      const user = userEvent.setup();
      renderToolbar({ graphWidth: 500 });

      const selectionActionsButton = screen.getByTitle('Selection actions');
      await user.click(selectionActionsButton);

      // Menu items should appear
      expect(screen.getByText('Duplicate selected')).toBeInTheDocument();
      expect(screen.getByText('Delete selected')).toBeInTheDocument();
    });
  });

  describe('Selection Actions', () => {
    it('calls onDuplicate when duplicate button clicked', async () => {
      const user = userEvent.setup();
      const onDuplicate = vi.fn();

      renderToolbar({ onDuplicate, graphWidth: 500 }); // Compact mode

      const selectionActionsButton = screen.getByTitle('Selection actions');
      await user.click(selectionActionsButton);

      const duplicateButton = screen.getByText('Duplicate selected');
      await user.click(duplicateButton);

      expect(onDuplicate).toHaveBeenCalledTimes(1);
    });

    it('calls onDelete when delete button clicked', async () => {
      const user = userEvent.setup();
      const onDelete = vi.fn();

      renderToolbar({ onDelete, graphWidth: 500 }); // Compact mode

      const selectionActionsButton = screen.getByTitle('Selection actions');
      await user.click(selectionActionsButton);

      const deleteButton = screen.getByText('Delete selected');
      await user.click(deleteButton);

      expect(onDelete).toHaveBeenCalledTimes(1);
    });
  });

  describe('Menu Close Behavior', () => {
    it('closes layout menu when Escape key pressed', async () => {
      const user = userEvent.setup();
      renderToolbar({ graphWidth: 500 });

      const canvasOptionsButton = screen.getByTitle(/Canvas options/i);
      await user.click(canvasOptionsButton);

      expect(screen.getByText('Fit to view')).toBeInTheDocument();

      await user.keyboard('{Escape}');

      // Menu should be closed (items not in document)
      expect(screen.queryByText('Fit to view')).not.toBeInTheDocument();
    });

    it('closes selection menu when Escape key pressed', async () => {
      const user = userEvent.setup();
      renderToolbar({ graphWidth: 500 });

      const selectionActionsButton = screen.getByTitle('Selection actions');
      await user.click(selectionActionsButton);

      expect(screen.getByText('Duplicate selected')).toBeInTheDocument();

      await user.keyboard('{Escape}');

      // Menu should be closed
      expect(screen.queryByText('Duplicate selected')).not.toBeInTheDocument();
    });
  });

  describe('Responsive Behavior', () => {
    it('closes menus when switching from compact to full mode', () => {
      const { rerender } = render(
        <ReactFlowProvider>
          <WorkflowToolbar {...defaultProps} graphWidth={500} />
        </ReactFlowProvider>
      );

      // Menus exist in compact mode
      expect(screen.queryByTitle(/Canvas options/i)).toBeInTheDocument();

      // Switch to full mode
      rerender(
        <ReactFlowProvider>
          <WorkflowToolbar {...defaultProps} graphWidth={800} />
        </ReactFlowProvider>
      );

      // Canvas options menu should not exist
      expect(screen.queryByTitle(/Canvas options/i)).not.toBeInTheDocument();
      // Direct buttons should exist
      expect(screen.getByTitle('Fit view')).toBeInTheDocument();
    });
  });
});
