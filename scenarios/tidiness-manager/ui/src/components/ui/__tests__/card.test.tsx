import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
  CardFooter,
} from '../card';

describe('Card', () => {
  it('renders correctly', () => {
    render(<Card>Card content</Card>);
    const card = screen.getByRole('region');
    expect(card).toBeInTheDocument();
    expect(card).toHaveTextContent('Card content');
  });

  it('applies default styling classes', () => {
    render(<Card>Content</Card>);
    const card = screen.getByRole('region');
    expect(card).toHaveClass(
      'rounded-xl',
      'border',
      'border-white/10',
      'bg-white/5',
      'backdrop-blur-sm'
    );
  });

  it('applies custom className', () => {
    render(<Card className="custom-card">Content</Card>);
    const card = screen.getByRole('region');
    expect(card).toHaveClass('custom-card');
  });

  it('includes role="region"', () => {
    render(<Card>Test</Card>);
    const card = screen.getByText('Test');
    expect(card).toHaveAttribute('role', 'region');
  });
});

describe('CardHeader', () => {
  it('renders correctly', () => {
    render(
      <Card>
        <CardHeader>Header content</CardHeader>
      </Card>
    );
    expect(screen.getByText('Header content')).toBeInTheDocument();
  });

  it('applies default styling', () => {
    render(
      <Card>
        <CardHeader>Header</CardHeader>
      </Card>
    );
    const header = screen.getByText('Header');
    expect(header).toHaveClass('flex', 'flex-col', 'space-y-1.5', 'p-6');
  });

  it('applies custom className', () => {
    render(
      <Card>
        <CardHeader className="custom-header">Header</CardHeader>
      </Card>
    );
    const header = screen.getByText('Header');
    expect(header).toHaveClass('custom-header');
  });
});

describe('CardTitle', () => {
  it('renders correctly', () => {
    render(
      <Card>
        <CardHeader>
          <CardTitle>Card Title</CardTitle>
        </CardHeader>
      </Card>
    );
    expect(screen.getByText('Card Title')).toBeInTheDocument();
  });

  it('applies default styling', () => {
    render(
      <Card>
        <CardHeader>
          <CardTitle>Title</CardTitle>
        </CardHeader>
      </Card>
    );
    const title = screen.getByText('Title');
    expect(title).toHaveClass('text-lg', 'font-semibold', 'leading-none', 'tracking-tight');
  });

  it('renders as h2 element for proper heading hierarchy', () => {
    render(
      <Card>
        <CardHeader>
          <CardTitle>Title</CardTitle>
        </CardHeader>
      </Card>
    );
    const title = screen.getByText('Title');
    expect(title.tagName).toBe('H2');
  });

  it('is a valid heading element', () => {
    render(
      <Card>
        <CardHeader>
          <CardTitle>Title</CardTitle>
        </CardHeader>
      </Card>
    );
    const title = screen.getByRole('heading', { level: 2 });
    expect(title).toBeInTheDocument();
    expect(title).toHaveTextContent('Title');
  });

  it('applies custom className', () => {
    render(
      <Card>
        <CardHeader>
          <CardTitle className="custom-title">Title</CardTitle>
        </CardHeader>
      </Card>
    );
    const title = screen.getByText('Title');
    expect(title).toHaveClass('custom-title');
  });
});

describe('CardDescription', () => {
  it('renders correctly', () => {
    render(
      <Card>
        <CardHeader>
          <CardDescription>Card description text</CardDescription>
        </CardHeader>
      </Card>
    );
    expect(screen.getByText('Card description text')).toBeInTheDocument();
  });

  it('applies default styling', () => {
    render(
      <Card>
        <CardHeader>
          <CardDescription>Description</CardDescription>
        </CardHeader>
      </Card>
    );
    const description = screen.getByText('Description');
    expect(description).toHaveClass('text-sm', 'text-slate-400');
  });

  it('renders as p element', () => {
    render(
      <Card>
        <CardHeader>
          <CardDescription>Description</CardDescription>
        </CardHeader>
      </Card>
    );
    const description = screen.getByText('Description');
    expect(description.tagName).toBe('P');
  });

  it('applies custom className', () => {
    render(
      <Card>
        <CardHeader>
          <CardDescription className="custom-desc">Description</CardDescription>
        </CardHeader>
      </Card>
    );
    const description = screen.getByText('Description');
    expect(description).toHaveClass('custom-desc');
  });
});

describe('CardContent', () => {
  it('renders correctly', () => {
    render(
      <Card>
        <CardContent>Content text</CardContent>
      </Card>
    );
    expect(screen.getByText('Content text')).toBeInTheDocument();
  });

  it('applies default styling', () => {
    render(
      <Card>
        <CardContent>Content</CardContent>
      </Card>
    );
    const content = screen.getByText('Content');
    expect(content).toHaveClass('p-6', 'pt-0');
  });

  it('applies custom className', () => {
    render(
      <Card>
        <CardContent className="custom-content">Content</CardContent>
      </Card>
    );
    const content = screen.getByText('Content');
    expect(content).toHaveClass('custom-content');
  });
});

describe('CardFooter', () => {
  it('renders correctly', () => {
    render(
      <Card>
        <CardFooter>Footer content</CardFooter>
      </Card>
    );
    expect(screen.getByText('Footer content')).toBeInTheDocument();
  });

  it('applies default styling', () => {
    render(
      <Card>
        <CardFooter>Footer</CardFooter>
      </Card>
    );
    const footer = screen.getByText('Footer');
    expect(footer).toHaveClass('flex', 'items-center', 'p-6', 'pt-0');
  });

  it('applies custom className', () => {
    render(
      <Card>
        <CardFooter className="custom-footer">Footer</CardFooter>
      </Card>
    );
    const footer = screen.getByText('Footer');
    expect(footer).toHaveClass('custom-footer');
  });
});

describe('Card composition', () => {
  it('renders complete card with all sections', () => {
    render(
      <Card>
        <CardHeader>
          <CardTitle>Test Card</CardTitle>
          <CardDescription>This is a test card</CardDescription>
        </CardHeader>
        <CardContent>
          <p>Card body content goes here</p>
        </CardContent>
        <CardFooter>
          <button>Action</button>
        </CardFooter>
      </Card>
    );

    expect(screen.getByText('Test Card')).toBeInTheDocument();
    expect(screen.getByText('This is a test card')).toBeInTheDocument();
    expect(screen.getByText('Card body content goes here')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Action' })).toBeInTheDocument();
  });

  it('supports nested content structures', () => {
    render(
      <Card>
        <CardHeader>
          <CardTitle>Stats</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 gap-4">
            <div>Stat 1</div>
            <div>Stat 2</div>
          </div>
        </CardContent>
      </Card>
    );

    expect(screen.getByText('Stats')).toBeInTheDocument();
    expect(screen.getByText('Stat 1')).toBeInTheDocument();
    expect(screen.getByText('Stat 2')).toBeInTheDocument();
  });

  it('supports card without header', () => {
    render(
      <Card>
        <CardContent>Simple card content</CardContent>
      </Card>
    );

    expect(screen.getByText('Simple card content')).toBeInTheDocument();
  });

  it('supports card without footer', () => {
    render(
      <Card>
        <CardHeader>
          <CardTitle>Title Only</CardTitle>
        </CardHeader>
        <CardContent>Content only</CardContent>
      </Card>
    );

    expect(screen.getByText('Title Only')).toBeInTheDocument();
    expect(screen.getByText('Content only')).toBeInTheDocument();
  });

  it('supports responsive layout classes', () => {
    render(
      <Card className="w-full sm:w-1/2 lg:w-1/3">
        <CardContent>Responsive card</CardContent>
      </Card>
    );

    const card = screen.getByRole('region');
    expect(card).toHaveClass('w-full', 'sm:w-1/2', 'lg:w-1/3');
  });
});
