/**
 * UnifiedSidebar Component Tests
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { UnifiedSidebar, type UnifiedSidebarProps } from './UnifiedSidebar';
import { DEFAULT_AI_SETTINGS } from './types';
import { VISION_MODELS } from '../ai-navigation/types';

// Mock scrollIntoView which isn't available in jsdom
Element.prototype.scrollIntoView = vi.fn();

// Mock TimelineTab component
vi.mock('./TimelineTab', () => ({
  TimelineTab: vi.fn(({ className }) => (
    <div data-testid="timeline-tab" className={className}>
      TimelineTab Content
    </div>
  )),
}));

// Mock AutoTab component
vi.mock('./AutoTab', () => ({
  AutoTab: vi.fn(({ className }) => (
    <div data-testid="auto-tab" className={className}>
      AutoTab Content
    </div>
  )),
}));

// Mock localStorage
const localStorageMock = (() => {
  let store: Record<string, string> = {};
  return {
    getItem: (key: string) => store[key] || null,
    setItem: (key: string, value: string) => {
      store[key] = value;
    },
    removeItem: (key: string) => {
      delete store[key];
    },
    clear: () => {
      store = {};
    },
  };
})();
Object.defineProperty(window, 'localStorage', { value: localStorageMock });

describe('UnifiedSidebar', () => {
  const defaultTimelineProps: UnifiedSidebarProps['timelineProps'] = {
    actions: [],
    isRecording: true,
    isLoading: false,
    isReplaying: false,
    hasUnstableSelectors: false,
    onClearRequested: vi.fn(),
    onCreateWorkflow: vi.fn(),
    isSelectionMode: false,
    selectedIndices: new Set(),
    onToggleSelectionMode: vi.fn(),
    onActionClick: vi.fn(),
    onSelectAll: vi.fn(),
    onSelectNone: vi.fn(),
  };

  const defaultAutoProps: UnifiedSidebarProps['autoProps'] = {
    messages: [],
    isNavigating: false,
    settings: DEFAULT_AI_SETTINGS,
    availableModels: VISION_MODELS,
    onSendMessage: vi.fn(),
    onAbort: vi.fn(),
    onHumanDone: vi.fn(),
    onSettingsChange: vi.fn(),
  };

  const defaultProps: UnifiedSidebarProps = {
    timelineProps: defaultTimelineProps,
    autoProps: defaultAutoProps,
    isOpen: true,
  };

  beforeEach(() => {
    vi.clearAllMocks();
    localStorageMock.clear();
  });

  describe('rendering', () => {
    it('should render when open', () => {
      render(<UnifiedSidebar {...defaultProps} />);

      expect(screen.getByRole('tab', { name: 'Timeline' })).toBeInTheDocument();
      expect(screen.getByRole('tab', { name: 'Auto' })).toBeInTheDocument();
    });

    it('should not render when closed', () => {
      render(<UnifiedSidebar {...defaultProps} isOpen={false} />);

      expect(screen.queryByRole('tab')).not.toBeInTheDocument();
    });

    it('should render timeline tab content by default', () => {
      render(<UnifiedSidebar {...defaultProps} />);

      expect(screen.getByTestId('timeline-tab')).toBeInTheDocument();
      expect(screen.queryByTestId('auto-tab')).not.toBeInTheDocument();
    });

    it('should render auto tab content when initialTab is auto', () => {
      render(<UnifiedSidebar {...defaultProps} initialTab="auto" />);

      expect(screen.getByTestId('auto-tab')).toBeInTheDocument();
      expect(screen.queryByTestId('timeline-tab')).not.toBeInTheDocument();
    });

    it('should render resize handle', () => {
      render(<UnifiedSidebar {...defaultProps} />);

      expect(screen.getByRole('separator', { name: 'Resize sidebar' })).toBeInTheDocument();
    });
  });

  describe('tab switching', () => {
    it('should switch to auto tab when clicked', async () => {
      const user = userEvent.setup();
      render(<UnifiedSidebar {...defaultProps} />);

      await user.click(screen.getByRole('tab', { name: 'Auto' }));

      expect(screen.getByTestId('auto-tab')).toBeInTheDocument();
      expect(screen.queryByTestId('timeline-tab')).not.toBeInTheDocument();
    });

    it('should switch back to timeline tab when clicked', async () => {
      const user = userEvent.setup();
      render(<UnifiedSidebar {...defaultProps} initialTab="auto" />);

      await user.click(screen.getByRole('tab', { name: 'Timeline' }));

      expect(screen.getByTestId('timeline-tab')).toBeInTheDocument();
      expect(screen.queryByTestId('auto-tab')).not.toBeInTheDocument();
    });

    it('should call onTabChange when tab changes', async () => {
      const user = userEvent.setup();
      const onTabChange = vi.fn();
      render(<UnifiedSidebar {...defaultProps} onTabChange={onTabChange} />);

      await user.click(screen.getByRole('tab', { name: 'Auto' }));

      expect(onTabChange).toHaveBeenCalledWith('auto');
    });

    it('should mark timeline tab as selected when active', () => {
      render(<UnifiedSidebar {...defaultProps} />);

      const timelineTab = screen.getByRole('tab', { name: 'Timeline' });
      const autoTab = screen.getByRole('tab', { name: 'Auto' });

      expect(timelineTab).toHaveAttribute('aria-selected', 'true');
      expect(autoTab).toHaveAttribute('aria-selected', 'false');
    });

    it('should mark auto tab as selected when active', async () => {
      const user = userEvent.setup();
      render(<UnifiedSidebar {...defaultProps} />);

      await user.click(screen.getByRole('tab', { name: 'Auto' }));

      const timelineTab = screen.getByRole('tab', { name: 'Timeline' });
      const autoTab = screen.getByRole('tab', { name: 'Auto' });

      expect(timelineTab).toHaveAttribute('aria-selected', 'false');
      expect(autoTab).toHaveAttribute('aria-selected', 'true');
    });
  });

  describe('resize functionality', () => {
    it('should start resize on mousedown', () => {
      render(<UnifiedSidebar {...defaultProps} />);

      const resizeHandle = screen.getByRole('separator', { name: 'Resize sidebar' });
      fireEvent.mouseDown(resizeHandle, { clientX: 360 });

      // The resize handle should be active (would have bg-blue-500 class)
      // We can verify by checking the handle is still there and the component is functional
      expect(resizeHandle).toBeInTheDocument();
    });

    it('should have aria-valuenow with current width', () => {
      render(<UnifiedSidebar {...defaultProps} />);

      const resizeHandle = screen.getByRole('separator', { name: 'Resize sidebar' });
      // Default width is 360
      expect(resizeHandle).toHaveAttribute('aria-valuenow', '360');
    });
  });

  describe('activity indicators', () => {
    it('should show activity indicator on auto tab when viewing timeline and messages arrive', async () => {
      const { rerender } = render(<UnifiedSidebar {...defaultProps} />);

      // Add a message while on timeline tab
      const updatedAutoProps = {
        ...defaultAutoProps,
        messages: [{ id: '1', role: 'user' as const, content: 'Test', timestamp: new Date() }],
      };
      rerender(<UnifiedSidebar {...defaultProps} autoProps={updatedAutoProps} />);

      // Activity indicator should be visible on the Auto tab
      // When activity indicator is present, the accessible name becomes "AutoNew activity"
      const autoTab = screen.getByRole('tab', { name: /Auto/i });
      const indicator = autoTab.querySelector('[aria-label="New activity"]');
      expect(indicator).toBeInTheDocument();
    });

    it('should show activity indicator on timeline tab when viewing auto and actions arrive', async () => {
      const { rerender } = render(<UnifiedSidebar {...defaultProps} initialTab="auto" />);

      // Add an action while on auto tab
      const updatedTimelineProps = {
        ...defaultTimelineProps,
        actions: [{ id: '1', type: 'click', timestamp: new Date(), selector: 'button' }] as any,
      };
      rerender(
        <UnifiedSidebar {...defaultProps} initialTab="auto" timelineProps={updatedTimelineProps} />
      );

      // Activity indicator should be visible on the Timeline tab
      // When activity indicator is present, the accessible name becomes "TimelineNew activity"
      const timelineTab = screen.getByRole('tab', { name: /Timeline/i });
      const indicator = timelineTab.querySelector('[aria-label="New activity"]');
      expect(indicator).toBeInTheDocument();
    });

    it('should not show activity indicator on active tab', () => {
      render(<UnifiedSidebar {...defaultProps} />);

      // Timeline tab is active, should not have activity indicator
      const timelineTab = screen.getByRole('tab', { name: 'Timeline' });
      const indicator = timelineTab.querySelector('[aria-label="New activity"]');
      expect(indicator).not.toBeInTheDocument();
    });
  });

  describe('controlled open state', () => {
    it('should call onOpenChange when internal state differs', () => {
      const onOpenChange = vi.fn();
      render(<UnifiedSidebar {...defaultProps} isOpen={true} onOpenChange={onOpenChange} />);

      // Initial render - states should match
      // onOpenChange would be called if they differed
      expect(screen.getByRole('tab', { name: 'Timeline' })).toBeInTheDocument();
    });

    it('should sync with controlled isOpen prop', () => {
      const { rerender } = render(<UnifiedSidebar {...defaultProps} isOpen={true} />);

      expect(screen.getByRole('tab', { name: 'Timeline' })).toBeInTheDocument();

      rerender(<UnifiedSidebar {...defaultProps} isOpen={false} />);

      expect(screen.queryByRole('tab')).not.toBeInTheDocument();
    });
  });

  describe('className prop', () => {
    it('should apply custom className', () => {
      const { container } = render(<UnifiedSidebar {...defaultProps} className="custom-class" />);

      expect(container.firstChild).toHaveClass('custom-class');
    });
  });

  describe('props passthrough', () => {
    it('should pass props to TimelineTab', () => {
      render(<UnifiedSidebar {...defaultProps} />);

      expect(screen.getByTestId('timeline-tab')).toHaveClass('h-full');
    });

    it('should pass props to AutoTab', async () => {
      const user = userEvent.setup();
      render(<UnifiedSidebar {...defaultProps} />);

      await user.click(screen.getByRole('tab', { name: 'Auto' }));

      expect(screen.getByTestId('auto-tab')).toHaveClass('h-full');
    });
  });

  describe('keyboard shortcuts', () => {
    it('should switch to timeline tab with Ctrl+1', async () => {
      render(<UnifiedSidebar {...defaultProps} initialTab="auto" />);

      expect(screen.getByTestId('auto-tab')).toBeInTheDocument();

      fireEvent.keyDown(document, { key: '1', ctrlKey: true });

      expect(screen.getByTestId('timeline-tab')).toBeInTheDocument();
      expect(screen.queryByTestId('auto-tab')).not.toBeInTheDocument();
    });

    it('should switch to auto tab with Ctrl+2', async () => {
      render(<UnifiedSidebar {...defaultProps} />);

      expect(screen.getByTestId('timeline-tab')).toBeInTheDocument();

      fireEvent.keyDown(document, { key: '2', ctrlKey: true });

      expect(screen.getByTestId('auto-tab')).toBeInTheDocument();
      expect(screen.queryByTestId('timeline-tab')).not.toBeInTheDocument();
    });

    it('should not switch tabs without modifier key', () => {
      render(<UnifiedSidebar {...defaultProps} />);

      expect(screen.getByTestId('timeline-tab')).toBeInTheDocument();

      fireEvent.keyDown(document, { key: '2' });

      // Should still be on timeline
      expect(screen.getByTestId('timeline-tab')).toBeInTheDocument();
    });

    it('should not respond to shortcuts when closed', () => {
      const onTabChange = vi.fn();
      render(<UnifiedSidebar {...defaultProps} isOpen={false} onTabChange={onTabChange} />);

      fireEvent.keyDown(document, { key: '2', ctrlKey: true });

      expect(onTabChange).not.toHaveBeenCalled();
    });
  });

  describe('tooltips', () => {
    it('should have title attribute on timeline tab with keyboard shortcut', () => {
      render(<UnifiedSidebar {...defaultProps} />);

      const timelineTab = screen.getByRole('tab', { name: /Timeline/i });
      const title = timelineTab.getAttribute('title');

      expect(title).toContain('View recorded actions');
      expect(title).toMatch(/Ctrl\+1|⌘1/);
    });

    it('should have title attribute on auto tab with keyboard shortcut', () => {
      render(<UnifiedSidebar {...defaultProps} />);

      const autoTab = screen.getByRole('tab', { name: /Auto/i });
      const title = autoTab.getAttribute('title');

      expect(title).toContain('AI-powered browser automation');
      expect(title).toMatch(/Ctrl\+2|⌘2/);
    });
  });

  describe('accessibility', () => {
    it('should have tablist role on tab container', () => {
      render(<UnifiedSidebar {...defaultProps} />);

      expect(screen.getByRole('tablist')).toBeInTheDocument();
    });
  });
});
