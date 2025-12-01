import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { Card, CardHeader, CardTitle, CardDescription, CardContent, CardFooter } from './card';

/**
 * Card Component Tests
 * Validates card structure and composition
 */

describe('Card Components', () => {
  it('should render a basic card', () => {
    const { container } = render(<Card>Card content</Card>);
    const card = container.querySelector('.rounded-xl');
    expect(card).toBeDefined();
    expect(card?.textContent).toBe('Card content');
  });

  it('should render card with all sections', () => {
    render(
      <Card>
        <CardHeader>
          <CardTitle>Card Title</CardTitle>
          <CardDescription>Card description</CardDescription>
        </CardHeader>
        <CardContent>
          Card content goes here
        </CardContent>
        <CardFooter>
          Card footer
        </CardFooter>
      </Card>
    );

    expect(screen.getByText('Card Title')).toBeDefined();
    expect(screen.getByText('Card description')).toBeDefined();
    expect(screen.getByText('Card content goes here')).toBeDefined();
    expect(screen.getByText('Card footer')).toBeDefined();
  });

  it('should apply custom className to card', () => {
    const { container } = render(<Card className="custom-card">Content</Card>);
    const card = container.querySelector('.custom-card');
    expect(card).toBeDefined();
  });

  it('should apply custom className to card header', () => {
    const { container } = render(
      <Card>
        <CardHeader className="custom-header">Header</CardHeader>
      </Card>
    );
    const header = container.querySelector('.custom-header');
    expect(header).toBeDefined();
  });

  it('should render card title as h3', () => {
    const { container } = render(
      <Card>
        <CardHeader>
          <CardTitle>Title</CardTitle>
        </CardHeader>
      </Card>
    );
    const title = container.querySelector('h3');
    expect(title?.textContent).toBe('Title');
  });

  it('should render card description as paragraph', () => {
    const { container } = render(
      <Card>
        <CardHeader>
          <CardDescription>Description</CardDescription>
        </CardHeader>
      </Card>
    );
    const description = container.querySelector('p');
    expect(description?.textContent).toBe('Description');
  });

  it('should support nested content in card content', () => {
    render(
      <Card>
        <CardContent>
          <div>
            <p>Paragraph 1</p>
            <p>Paragraph 2</p>
          </div>
        </CardContent>
      </Card>
    );

    expect(screen.getByText('Paragraph 1')).toBeDefined();
    expect(screen.getByText('Paragraph 2')).toBeDefined();
  });

  it('should render card footer with flex layout', () => {
    const { container } = render(
      <Card>
        <CardFooter>Footer</CardFooter>
      </Card>
    );
    const footer = container.querySelector('.flex.items-center');
    expect(footer?.textContent).toBe('Footer');
  });

  it('should forward refs correctly', () => {
    const ref = { current: null };
    render(<Card ref={ref}>Content</Card>);
    expect(ref.current).not.toBeNull();
  });

  it('should support data attributes', () => {
    render(<Card data-testid="test-card">Content</Card>);
    expect(screen.getByTestId('test-card')).toBeDefined();
  });
});
