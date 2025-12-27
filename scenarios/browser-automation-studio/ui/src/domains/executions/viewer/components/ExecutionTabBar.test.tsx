/**
 * ExecutionTabBar Component Tests
 *
 * Tests for the multi-page execution tab bar component.
 */

import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { ExecutionTabBar, getPageColor, getPageColorById } from './ExecutionTabBar';
import type { ExecutionPage } from '../../store';

describe('ExecutionTabBar', () => {
  const createPage = (overrides: Partial<ExecutionPage> = {}): ExecutionPage => ({
    id: 'page-1',
    url: 'https://example.com',
    title: 'Example Page',
    isInitial: false,
    ...overrides,
  });

  describe('visibility', () => {
    it('renders nothing when only one page', () => {
      const pages = [createPage({ id: 'page-1' })];
      const { container } = render(
        <ExecutionTabBar pages={pages} activePageId="page-1" />
      );
      expect(container.firstChild).toBeNull();
    });

    it('renders nothing when show is false', () => {
      const pages = [
        createPage({ id: 'page-1' }),
        createPage({ id: 'page-2' }),
      ];
      const { container } = render(
        <ExecutionTabBar pages={pages} activePageId="page-1" show={false} />
      );
      expect(container.firstChild).toBeNull();
    });

    it('renders tabs when multiple pages exist', () => {
      const pages = [
        createPage({ id: 'page-1', title: 'Main Page', isInitial: true }),
        createPage({ id: 'page-2', title: 'Popup Page' }),
      ];
      render(<ExecutionTabBar pages={pages} activePageId="page-1" />);

      expect(screen.getByText('Main Page')).toBeInTheDocument();
      expect(screen.getByText('Popup Page')).toBeInTheDocument();
    });
  });

  describe('tab styling', () => {
    it('shows main badge for initial page', () => {
      const pages = [
        createPage({ id: 'page-1', title: 'Main Page', isInitial: true }),
        createPage({ id: 'page-2', title: 'Popup Page' }),
      ];
      render(<ExecutionTabBar pages={pages} activePageId="page-1" />);

      expect(screen.getByText('main')).toBeInTheDocument();
    });

    it('shows closed badge for closed pages', () => {
      const pages = [
        createPage({ id: 'page-1', title: 'Main Page', isInitial: true }),
        createPage({ id: 'page-2', title: 'Closed Page', closed: true }),
      ];
      render(<ExecutionTabBar pages={pages} activePageId="page-1" />);

      expect(screen.getByText('closed')).toBeInTheDocument();
    });

    it('shows tab count badge', () => {
      const pages = [
        createPage({ id: 'page-1', isInitial: true }),
        createPage({ id: 'page-2' }),
        createPage({ id: 'page-3', closed: true }),
      ];
      render(<ExecutionTabBar pages={pages} activePageId="page-1" />);

      // 2 open out of 3 total
      expect(screen.getByText('2 / 3 tabs')).toBeInTheDocument();
    });
  });

  describe('interaction', () => {
    it('calls onTabClick when tab is clicked', () => {
      const onTabClick = vi.fn();
      const pages = [
        createPage({ id: 'page-1', title: 'Main Page', isInitial: true }),
        createPage({ id: 'page-2', title: 'Popup Page' }),
      ];
      render(
        <ExecutionTabBar
          pages={pages}
          activePageId="page-1"
          onTabClick={onTabClick}
        />
      );

      fireEvent.click(screen.getByText('Popup Page'));

      expect(onTabClick).toHaveBeenCalledWith('page-2');
    });

    it('does not call onTabClick for already active tab', () => {
      const onTabClick = vi.fn();
      const pages = [
        createPage({ id: 'page-1', title: 'Main Page', isInitial: true }),
        createPage({ id: 'page-2', title: 'Popup Page' }),
      ];
      render(
        <ExecutionTabBar
          pages={pages}
          activePageId="page-1"
          onTabClick={onTabClick}
        />
      );

      fireEvent.click(screen.getByText('Main Page'));

      expect(onTabClick).not.toHaveBeenCalled();
    });
  });

  describe('running indicator', () => {
    it('shows running indicator when isRunning is true', () => {
      const pages = [
        createPage({ id: 'page-1', title: 'Main Page', isInitial: true }),
        createPage({ id: 'page-2', title: 'Popup Page' }),
      ];
      render(
        <ExecutionTabBar pages={pages} activePageId="page-1" isRunning={true} />
      );

      expect(screen.getByText('Running...')).toBeInTheDocument();
    });

    it('does not show running indicator when isRunning is false', () => {
      const pages = [
        createPage({ id: 'page-1', title: 'Main Page', isInitial: true }),
        createPage({ id: 'page-2', title: 'Popup Page' }),
      ];
      render(<ExecutionTabBar pages={pages} activePageId="page-1" />);

      expect(screen.queryByText('Running...')).not.toBeInTheDocument();
    });
  });
});

describe('getPageColor', () => {
  it('returns consistent colors for same index', () => {
    const color1 = getPageColor(0);
    const color2 = getPageColor(0);
    expect(color1).toEqual(color2);
  });

  it('returns different colors for different indices', () => {
    const color0 = getPageColor(0);
    const color1 = getPageColor(1);
    expect(color0.bg).not.toEqual(color1.bg);
  });

  it('wraps around for indices beyond color palette', () => {
    const color0 = getPageColor(0);
    const color8 = getPageColor(8);
    expect(color0).toEqual(color8);
  });
});

describe('getPageColorById', () => {
  it('returns color for existing page', () => {
    const pages: ExecutionPage[] = [
      { id: 'page-1', url: '', title: '', isInitial: true },
      { id: 'page-2', url: '', title: '', isInitial: false },
    ];
    const color = getPageColorById('page-2', pages);
    expect(color).toBeTruthy();
    expect(color).toEqual(getPageColor(1));
  });

  it('returns first color for non-existing page', () => {
    const pages: ExecutionPage[] = [
      { id: 'page-1', url: '', title: '', isInitial: true },
    ];
    const color = getPageColorById('non-existent', pages);
    expect(color).toEqual(getPageColor(0));
  });
});
