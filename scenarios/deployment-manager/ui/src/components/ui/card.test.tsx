import { describe, it, expect } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from "@vrooli/api-base/testing";
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from './card';

describe('Card Component', () => {
  it('renders card with all sections', () => {
    renderWithProviders(
      <Card>
        <CardHeader>
          <CardTitle>Test Title</CardTitle>
          <CardDescription>Test Description</CardDescription>
        </CardHeader>
        <CardContent>
          <p>Test Content</p>
        </CardContent>
      </Card>
    );

    expect(screen.getByText('Test Title')).toBeInTheDocument();
    expect(screen.getByText('Test Description')).toBeInTheDocument();
    expect(screen.getByText('Test Content')).toBeInTheDocument();
  });

  it('renders card with custom className', () => {
    const { container } = renderWithProviders(
      <Card className="custom-class">
        <CardContent>Content</CardContent>
      </Card>
    );

    expect(container.firstChild).toHaveClass('custom-class');
  });

  it('renders CardTitle without description', () => {
    renderWithProviders(
      <Card>
        <CardHeader>
          <CardTitle>Solo Title</CardTitle>
        </CardHeader>
      </Card>
    );

    expect(screen.getByText('Solo Title')).toBeInTheDocument();
  });
});
