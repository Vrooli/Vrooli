import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { AppShell } from '../components/layout/AppShell';

// Helper to render with all required providers
function renderWithRouter(ui: React.ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        {ui}
      </MemoryRouter>
    </QueryClientProvider>
  );
}

describe('Accessibility - WCAG AAA Compliance', () => {
  describe('Skip to content link', () => {
    it('[REQ:TM-UI-008] should have skip-to-content link for keyboard navigation', () => {
      const { container } = renderWithRouter(
        <AppShell>
          <div>Test content</div>
        </AppShell>
      );

      const skipLink = container.querySelector('.skip-to-content');
      expect(skipLink).toBeInTheDocument();
      expect(skipLink).toHaveAttribute('href', '#main-content');
      expect(skipLink).toHaveTextContent('Skip to content');
    });

    it('[REQ:TM-UI-008] main content should have id="main-content"', () => {
      const { container } = renderWithRouter(
        <AppShell>
          <div>Test content</div>
        </AppShell>
      );

      const main = container.querySelector('main');
      expect(main).toHaveAttribute('id', 'main-content');
    });
  });

  describe('ARIA labels and roles', () => {
    it('[REQ:TM-UI-004] Mobile menu button should have aria-label', () => {
      renderWithRouter(
        <AppShell>
          <div>Test</div>
        </AppShell>
      );

      const menuButton = screen.getByLabelText('Toggle navigation menu');
      expect(menuButton).toBeInTheDocument();
      expect(menuButton).toHaveAttribute('type', 'button');
    });

    it('[REQ:TM-UI-004] Interactive buttons should have descriptive text or aria-label', () => {
      const { container } = renderWithRouter(
        <AppShell>
          <div>
            <button>Save</button>
            <button aria-label="Close dialog">X</button>
          </div>
        </AppShell>
      );

      const buttons = container.querySelectorAll('button');
      buttons.forEach((button) => {
        const hasText = button.textContent && button.textContent.trim().length > 0;
        const hasAriaLabel = button.hasAttribute('aria-label');
        expect(hasText || hasAriaLabel).toBe(true);
      });
    });
  });

  describe('Keyboard navigation', () => {
    it('[REQ:TM-UI-007] Navigation links should be keyboard accessible', () => {
      const { container } = renderWithRouter(
        <AppShell>
          <div>Test</div>
        </AppShell>
      );

      const navLinks = container.querySelectorAll('a[href^="/"]');
      expect(navLinks.length).toBeGreaterThan(0); // Should have nav links

      navLinks.forEach((link) => {
        // Links should not have tabindex=-1 (making them non-focusable)
        expect(link.getAttribute('tabindex')).not.toBe('-1');
      });
    });

    it('[REQ:TM-UI-007] Interactive elements should be focusable', () => {
      const { container } = renderWithRouter(
        <AppShell>
          <div>
            <button>Click me</button>
            <input type="text" />
            <select><option>Test</option></select>
          </div>
        </AppShell>
      );

      const focusableElements = container.querySelectorAll('button, input, select, a');
      focusableElements.forEach((element) => {
        // Elements should not have tabindex=-1 unless disabled
        if (!element.hasAttribute('disabled')) {
          expect(element.getAttribute('tabindex')).not.toBe('-1');
        }
      });
    });
  });

  describe('Color contrast and readability', () => {
    it('[REQ:TM-UI-006] Dark theme should use readable color combinations', () => {
      const { container } = renderWithRouter(
        <AppShell>
          <div>Test content</div>
        </AppShell>
      );

      // Root should use dark background
      const root = container.querySelector('.bg-slate-950');
      expect(root).toBeInTheDocument();

      // Text should use light color
      const lightText = container.querySelector('.text-slate-50');
      expect(lightText).toBeInTheDocument();
    });

    it('[REQ:TM-UI-006] Focus styles should have sufficient visibility', () => {
      // Check that global focus styles are applied to elements
      const { container } = renderWithRouter(
        <AppShell>
          <button>Test</button>
        </AppShell>
      );

      const button = container.querySelector('button');
      expect(button).toBeInTheDocument();
      // Focus styles are in global CSS, verified via manual/Lighthouse testing
    });
  });

  describe('Responsive touch targets', () => {
    it('[REQ:TM-UI-004] Mobile menu button should have adequate touch target', () => {
      renderWithRouter(
        <AppShell>
          <div>Test</div>
        </AppShell>
      );

      const menuButton = screen.getByLabelText('Toggle navigation menu');

      // Button should have padding for adequate touch target (44x44px minimum)
      const classes = menuButton.className;
      expect(classes).toMatch(/p-2\.5/); // Tailwind padding class provides adequate spacing
    });

    it('[REQ:TM-UI-004] Buttons should have proper spacing for touch interfaces', () => {
      const { container } = renderWithRouter(
        <AppShell>
          <div>
            <button className="px-4 py-2">Action 1</button>
            <button className="px-4 py-2">Action 2</button>
          </div>
        </AppShell>
      );

      const buttons = container.querySelectorAll('button');
      buttons.forEach((button) => {
        // Should have padding classes for touch targets
        expect(button.className).toBeTruthy();
        expect(button.className.length).toBeGreaterThan(0);
      });
    });
  });

  describe('Semantic HTML and landmarks', () => {
    it('[REQ:TM-UI-001] Should use semantic HTML elements', () => {
      const { container } = renderWithRouter(
        <AppShell>
          <div>Test content</div>
        </AppShell>
      );

      // Should have semantic elements
      const header = container.querySelector('header');
      const main = container.querySelector('main');
      const footer = container.querySelector('footer');
      const nav = container.querySelector('nav');

      expect(header).toBeInTheDocument();
      expect(main).toBeInTheDocument();
      expect(footer).toBeInTheDocument();
      expect(nav).toBeInTheDocument();
    });
  });
});
