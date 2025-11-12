import { describe, it, expect, beforeEach, vi } from 'vitest';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import NodePalette from '../NodePalette';
import { ALL_NODE_TYPES } from '../../constants/nodeCategories';

const getNodeLabel = (label: string) =>
  screen.getByText((_, element) => {
    if (!element) {
      return false;
    }
    const text = element.textContent?.trim().toLowerCase();
    const className = (element as HTMLElement).className ?? '';
    return text === label.toLowerCase() && className.includes('font-medium');
  });

describe(
  'NodePalette [REQ:BAS-UX-PALETTE-CATEGORIES] [REQ:BAS-UX-PALETTE-SEARCH] [REQ:BAS-UX-PALETTE-QUICK-ACCESS] [REQ:BAS-WORKFLOW-BUILDER-DRAG-DROP] [REQ:BAS-WORKFLOW-NODE-TYPES]',
  () => {
    beforeEach(() => {
      window.localStorage.clear();
    });

    it('renders categories and exposes every node definition [REQ:BAS-UX-PALETTE-CATEGORIES] [REQ:BAS-WORKFLOW-NODE-TYPES]', () => {
      const { container } = render(<NodePalette />);

      expect(screen.getByText('Navigation & Context')).toBeInTheDocument();
      expect(screen.getByText('Pointer & Gestures')).toBeInTheDocument();
      expect(screen.getByText('Forms & Input')).toBeInTheDocument();

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
      expect(screen.getByText('Frame Switch')).toBeInTheDocument();
      expect(screen.getByText('Conditional')).toBeInTheDocument();
      expect(screen.getByText('Loop')).toBeInTheDocument();
      expect(screen.getByText('Set Cookie')).toBeInTheDocument();
      expect(screen.getByText('Get Storage')).toBeInTheDocument();
      expect(screen.getByText('Network Mock')).toBeInTheDocument();

      const nodeCards = Array.from(container.querySelectorAll('[data-node-type]'));
      const uniqueTypes = new Set(nodeCards.map((card) => card.getAttribute('data-node-type')));
      expect(uniqueTypes.size).toBe(ALL_NODE_TYPES.length);
    });

    it('filters nodes with search and highlights matches [REQ:BAS-UX-PALETTE-SEARCH]', () => {
      render(<NodePalette />);

      const searchInput = screen.getByPlaceholderText(/Search nodes/);
      fireEvent.change(searchInput, { target: { value: 'cookie' } });

      expect(screen.getByText('Storage & Network')).toBeInTheDocument();
      expect(screen.queryByText('Navigation & Context')).not.toBeInTheDocument();

      expect(getNodeLabel('Set Cookie')).toBeInTheDocument();
      expect(getNodeLabel('Get Cookie')).toBeInTheDocument();
      expect(getNodeLabel('Clear Cookie')).toBeInTheDocument();
      expect(screen.queryByText('Navigate')).not.toBeInTheDocument();

      const highlights = screen.getAllByText(
        (content, element) => element?.tagName === 'MARK' && content.toLowerCase() === 'cookie',
      );
      expect(highlights.length).toBeGreaterThanOrEqual(1);
    });

    it('records recents on drag and clears search [REQ:BAS-UX-PALETTE-QUICK-ACCESS] [REQ:BAS-WORKFLOW-BUILDER-DRAG-DROP]', async () => {
      const { container } = render(<NodePalette />);

      const searchInput = screen.getByPlaceholderText(/Search nodes/);
      fireEvent.change(searchInput, { target: { value: 'cookie' } });

      const setCookieCard = container.querySelector('[data-node-type="setCookie"]');
      expect(setCookieCard).toBeTruthy();

      const dataTransfer = {
        setData: vi.fn(),
        effectAllowed: '',
      } as unknown as DataTransfer;

      fireEvent.dragStart(setCookieCard!, { dataTransfer });

      await waitFor(() => expect(screen.getByText('Quick Access')).toBeInTheDocument());
      expect((searchInput as HTMLInputElement).value).toBe('');
      expect(screen.getByText('Recent')).toBeInTheDocument();
      const recentsHeading = screen.getByText('Recent');
      const recentsContainer = recentsHeading.parentElement?.parentElement;
      expect(recentsContainer?.querySelector('[data-node-type="setCookie"]')).toBeTruthy();
      expect(dataTransfer.setData).toHaveBeenCalledWith('nodeType', 'setCookie');
      expect(dataTransfer.effectAllowed).toBe('move');
    });

    it('limits favorites to five entries and persists ordering [REQ:BAS-UX-PALETTE-QUICK-ACCESS]', async () => {
      render(<NodePalette />);

      const favoriteTargets = [
        'Navigate',
        'Click',
        'Hover',
        'Drag & Drop',
        'Focus',
        'Blur',
      ];

      favoriteTargets.forEach((label) => {
        const toggle = screen.getByLabelText(`Add ${label} to favorites`);
        fireEvent.click(toggle);
      });

      const favoritesSection = await screen.findByText('Favorites');
      expect(favoritesSection).toBeInTheDocument();
      const favoritesHeading = await screen.findByText('Favorites');
      const favoritesContainer = favoritesHeading.parentElement?.querySelectorAll('[data-node-type]') ?? [];
      const favoriteTypes = Array.from(favoritesContainer)
        .map((card) => card.getAttribute('data-node-type'))
        .filter((value): value is string => Boolean(value));
      expect(favoriteTypes).toHaveLength(5);
      expect(favoriteTypes).not.toContain('navigate');
      expect(favoriteTypes).toContain('blur');

      await waitFor(() => {
        const stored = window.localStorage.getItem('bas.palette.favorites');
        expect(stored).toBeTruthy();
        const parsed = JSON.parse(stored ?? '[]');
        expect(parsed).toHaveLength(5);
        expect(parsed.includes('blur')).toBe(true);
      });
    });
  },
);
