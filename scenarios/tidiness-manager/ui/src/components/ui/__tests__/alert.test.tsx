import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { Alert, AlertTitle, AlertDescription } from '../alert';

describe('Alert', () => {
  it('renders with default variant', () => {
    render(<Alert>Default alert</Alert>);
    const alert = screen.getByRole('alert');
    expect(alert).toBeInTheDocument();
    expect(alert).toHaveClass('bg-white/5', 'border-white/10', 'text-slate-200');
  });

  it('renders with destructive variant', () => {
    render(<Alert variant="destructive">Destructive alert</Alert>);
    const alert = screen.getByRole('alert');
    expect(alert).toHaveClass('bg-red-950/50', 'border-red-900', 'text-red-100');
  });

  it('renders with warning variant', () => {
    render(<Alert variant="warning">Warning alert</Alert>);
    const alert = screen.getByRole('alert');
    expect(alert).toHaveClass('bg-yellow-950/50', 'border-yellow-900', 'text-yellow-100');
  });

  it('renders with success variant', () => {
    render(<Alert variant="success">Success alert</Alert>);
    const alert = screen.getByRole('alert');
    expect(alert).toHaveClass('bg-green-950/50', 'border-green-900', 'text-green-100');
  });

  it('renders with info variant', () => {
    render(<Alert variant="info">Info alert</Alert>);
    const alert = screen.getByRole('alert');
    expect(alert).toHaveClass('bg-blue-950/50', 'border-blue-900', 'text-blue-100');
  });

  it('applies custom className', () => {
    render(<Alert className="custom-class">Custom alert</Alert>);
    const alert = screen.getByRole('alert');
    expect(alert).toHaveClass('custom-class');
  });

  it('includes role="alert"', () => {
    render(<Alert>Alert message</Alert>);
    const alert = screen.getByText('Alert message');
    expect(alert).toHaveAttribute('role', 'alert');
  });

  it('includes base styling classes', () => {
    render(<Alert>Styled alert</Alert>);
    const alert = screen.getByRole('alert');
    expect(alert).toHaveClass('relative', 'w-full', 'rounded-lg', 'border', 'p-4');
  });
});

describe('AlertTitle', () => {
  it('renders title correctly', () => {
    render(
      <Alert>
        <AlertTitle>Alert Title</AlertTitle>
      </Alert>
    );
    expect(screen.getByText('Alert Title')).toBeInTheDocument();
  });

  it('applies default styling', () => {
    render(
      <Alert>
        <AlertTitle>Styled Title</AlertTitle>
      </Alert>
    );
    const title = screen.getByText('Styled Title');
    expect(title).toHaveClass('mb-1', 'font-medium', 'leading-none', 'tracking-tight');
  });

  it('applies custom className', () => {
    render(
      <Alert>
        <AlertTitle className="custom-title">Custom Title</AlertTitle>
      </Alert>
    );
    const title = screen.getByText('Custom Title');
    expect(title).toHaveClass('custom-title');
  });

  it('includes aria-level attribute', () => {
    render(
      <Alert>
        <AlertTitle>Accessible Title</AlertTitle>
      </Alert>
    );
    const title = screen.getByText('Accessible Title');
    expect(title).toHaveAttribute('aria-level', '5');
  });

  it('renders as h5 element', () => {
    render(
      <Alert>
        <AlertTitle>Heading</AlertTitle>
      </Alert>
    );
    const title = screen.getByText('Heading');
    expect(title.tagName).toBe('H5');
  });
});

describe('AlertDescription', () => {
  it('renders description correctly', () => {
    render(
      <Alert>
        <AlertDescription>Alert description text</AlertDescription>
      </Alert>
    );
    expect(screen.getByText('Alert description text')).toBeInTheDocument();
  });

  it('applies default styling', () => {
    render(
      <Alert>
        <AlertDescription>Styled description</AlertDescription>
      </Alert>
    );
    const description = screen.getByText('Styled description');
    expect(description).toHaveClass('text-sm', 'opacity-90');
  });

  it('applies custom className', () => {
    render(
      <Alert>
        <AlertDescription className="custom-description">Custom description</AlertDescription>
      </Alert>
    );
    const description = screen.getByText('Custom description');
    expect(description).toHaveClass('custom-description');
  });
});

describe('Alert composition', () => {
  it('renders complete alert with title and description', () => {
    render(
      <Alert variant="warning">
        <AlertTitle>Warning</AlertTitle>
        <AlertDescription>This is a warning message.</AlertDescription>
      </Alert>
    );

    expect(screen.getByRole('alert')).toBeInTheDocument();
    expect(screen.getByText('Warning')).toBeInTheDocument();
    expect(screen.getByText('This is a warning message.')).toBeInTheDocument();
  });

  it('supports multiple descriptions', () => {
    render(
      <Alert>
        <AlertTitle>Title</AlertTitle>
        <AlertDescription>First paragraph</AlertDescription>
        <AlertDescription>Second paragraph</AlertDescription>
      </Alert>
    );

    expect(screen.getByText('First paragraph')).toBeInTheDocument();
    expect(screen.getByText('Second paragraph')).toBeInTheDocument();
  });

  it('works with icon and text content', () => {
    render(
      <Alert variant="success">
        <div className="flex gap-2">
          <span>✓</span>
          <div>
            <AlertTitle>Success</AlertTitle>
            <AlertDescription>Operation completed successfully.</AlertDescription>
          </div>
        </div>
      </Alert>
    );

    expect(screen.getByText('✓')).toBeInTheDocument();
    expect(screen.getByText('Success')).toBeInTheDocument();
    expect(screen.getByText('Operation completed successfully.')).toBeInTheDocument();
  });
});
