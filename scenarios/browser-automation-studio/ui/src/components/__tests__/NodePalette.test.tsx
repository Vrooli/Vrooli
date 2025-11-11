import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { fireEvent } from '@testing-library/react';
import NodePalette from '../NodePalette';

describe('NodePalette [REQ:BAS-WORKFLOW-BUILDER-DRAG-DROP] [REQ:BAS-WORKFLOW-NODE-TYPES]', () => {
  it('renders key node types [REQ:BAS-WORKFLOW-NODE-TYPES]', () => {
    render(<NodePalette />);

    // Verify representative node types are rendered
    expect(screen.getByText('Navigate')).toBeInTheDocument();
    expect(screen.getByText('Click')).toBeInTheDocument();
    expect(screen.getByText('Type')).toBeInTheDocument();
    expect(screen.getByText('Shortcut')).toBeInTheDocument();
    expect(screen.getByText('Screenshot')).toBeInTheDocument();
    expect(screen.getByText('Wait')).toBeInTheDocument();
    expect(screen.getByText('Extract')).toBeInTheDocument();
    expect(screen.getByText('Assert')).toBeInTheDocument();
    expect(screen.getByText('Focus')).toBeInTheDocument();
    expect(screen.getByText('Blur')).toBeInTheDocument();
    expect(screen.getByText('Call Workflow')).toBeInTheDocument();
    expect(screen.getByText('Drag & Drop')).toBeInTheDocument();
    expect(screen.getByText('Upload File')).toBeInTheDocument();
    expect(screen.getByText('Rotate')).toBeInTheDocument();
    expect(screen.getByText('Gesture')).toBeInTheDocument();
    expect(screen.getByText('Conditional')).toBeInTheDocument();
    expect(screen.getByText('Loop')).toBeInTheDocument();
  });

  it('renders node descriptions', () => {
    render(<NodePalette />);

    expect(screen.getByText('Navigate to URL or scenario')).toBeInTheDocument();
    expect(screen.getByText('Click an element')).toBeInTheDocument();
    expect(screen.getByText('Type text in input')).toBeInTheDocument();
    expect(screen.getByText('Capture screenshot')).toBeInTheDocument();
  });

  it('renders pro tip section', () => {
    render(<NodePalette />);

    expect(screen.getByText('PRO TIP')).toBeInTheDocument();
    expect(screen.getByText(/Connect nodes by dragging/i)).toBeInTheDocument();
  });

  it('makes all nodes draggable [REQ:BAS-WORKFLOW-BUILDER-DRAG-DROP]', () => {
    const { container } = render(<NodePalette />);

    const draggableNodes = container.querySelectorAll('[draggable="true"]');

    // Should have draggable cards for every palette entry
    expect(draggableNodes).toHaveLength(25);
  });

  it('sets correct data on drag start for navigate node [REQ:BAS-WORKFLOW-BUILDER-DRAG-DROP]', () => {
    const { container } = render(<NodePalette />);

    const navigateNode = container.querySelector('[draggable="true"]');
    expect(navigateNode).toBeTruthy();

    const setDataMock = vi.fn();
    const dataTransfer = {
      setData: setDataMock,
      effectAllowed: '',
    } as unknown as DataTransfer;

    fireEvent.dragStart(navigateNode!, {
      dataTransfer,
    });

    expect(setDataMock).toHaveBeenCalledWith('nodeType', 'navigate');
    expect(dataTransfer.effectAllowed).toBe('move');
  });

  it('sets correct data on drag start for click node [REQ:BAS-WORKFLOW-BUILDER-DRAG-DROP]', () => {
    const { container } = render(<NodePalette />);

    const draggableNodes = container.querySelectorAll('[draggable="true"]');
    const clickNode = draggableNodes[1]; // Second node is 'click'

    const setDataMock = vi.fn();
    const dataTransfer = {
      setData: setDataMock,
      effectAllowed: '',
    } as unknown as DataTransfer;

    fireEvent.dragStart(clickNode, {
      dataTransfer,
    });

    expect(setDataMock).toHaveBeenCalledWith('nodeType', 'click');
    expect(dataTransfer.effectAllowed).toBe('move');
  });

  it('renders icons for each node type', () => {
    const { container } = render(<NodePalette />);

    // Check that icons are rendered (lucide-react icons render as SVGs)
    const icons = container.querySelectorAll('svg');

    // Should have at least 10 icons (9 node types + 1 pro tip icon)
    expect(icons.length).toBeGreaterThanOrEqual(10);
  });

  it('applies hover styles to node cards', () => {
    const { container } = render(<NodePalette />);

    const firstNode = container.querySelector('[draggable="true"]');
    expect(firstNode).toBeTruthy();

    // Check that hover classes are present
    expect(firstNode?.className).toContain('hover:border-flow-accent');
    expect(firstNode?.className).toContain('cursor-move');
  });

  it('renders with proper section heading', () => {
    render(<NodePalette />);

    const heading = screen.getByText('Drag nodes to canvas');
    expect(heading).toBeInTheDocument();
    expect(heading.tagName).toBe('H3');
  });

  it('displays node colors correctly', () => {
    const { container } = render(<NodePalette />);

    // Navigate should have blue color
    const navigateIcon = container.querySelector('.text-blue-400');
    expect(navigateIcon).toBeInTheDocument();

    // Click should have green color
    const clickIcon = container.querySelector('.text-green-400');
    expect(clickIcon).toBeInTheDocument();

    // Type should have yellow color
    const typeIcon = container.querySelector('.text-yellow-400');
    expect(typeIcon).toBeInTheDocument();
  });

  it('all node items have proper structure [REQ:BAS-WORKFLOW-NODE-TYPES]', () => {
    const { container } = render(<NodePalette />);

    const nodeCards = container.querySelectorAll('[draggable="true"]');

    nodeCards.forEach((card) => {
      // Each card should have icon, label, and description
      const icon = card.querySelector('svg');
      expect(icon).toBeTruthy();

      const label = card.querySelector('.font-medium');
      expect(label).toBeTruthy();

      const description = card.querySelector('.text-xs.text-gray-500');
      expect(description).toBeTruthy();
    });
  });
});
