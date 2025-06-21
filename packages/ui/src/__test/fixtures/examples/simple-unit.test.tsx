/**
 * Simple Unit Test Example
 * 
 * This test demonstrates that simple component tests run fast without database.
 * No special configuration needed - just write normal tests.
 */

import React from 'react';
import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

// Simple counter component
const Counter = ({ initialValue = 0, onValueChange }: { 
    initialValue?: number; 
    onValueChange?: (value: number) => void;
}) => {
    const [count, setCount] = React.useState(initialValue);

    const handleIncrement = () => {
        const newValue = count + 1;
        setCount(newValue);
        onValueChange?.(newValue);
    };

    const handleDecrement = () => {
        const newValue = count - 1;
        setCount(newValue);
        onValueChange?.(newValue);
    };

    const handleReset = () => {
        setCount(0);
        onValueChange?.(0);
    };

    return (
        <div>
            <h2>Counter: {count}</h2>
            <button onClick={handleIncrement}>Increment</button>
            <button onClick={handleDecrement}>Decrement</button>
            <button onClick={handleReset}>Reset</button>
        </div>
    );
};

// Form component with validation
const ContactForm = ({ onSubmit }: { 
    onSubmit: (data: { name: string; email: string; message: string }) => void;
}) => {
    const [formData, setFormData] = React.useState({
        name: '',
        email: '',
        message: ''
    });
    const [errors, setErrors] = React.useState<Record<string, string>>({});
    const [isSubmitting, setIsSubmitting] = React.useState(false);

    const validate = () => {
        const newErrors: Record<string, string> = {};
        
        if (!formData.name.trim()) {
            newErrors.name = 'Name is required';
        }
        
        if (!formData.email.trim()) {
            newErrors.email = 'Email is required';
        } else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(formData.email)) {
            newErrors.email = 'Invalid email format';
        }
        
        if (!formData.message.trim()) {
            newErrors.message = 'Message is required';
        } else if (formData.message.length < 10) {
            newErrors.message = 'Message must be at least 10 characters';
        }
        
        return newErrors;
    };

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        
        const newErrors = validate();
        if (Object.keys(newErrors).length > 0) {
            setErrors(newErrors);
            return;
        }
        
        setIsSubmitting(true);
        setErrors({});
        
        try {
            // Simulate async submission
            await new Promise(resolve => setTimeout(resolve, 100));
            onSubmit(formData);
            
            // Reset form
            setFormData({ name: '', email: '', message: '' });
        } finally {
            setIsSubmitting(false);
        }
    };

    const handleChange = (field: string, value: string) => {
        setFormData(prev => ({ ...prev, [field]: value }));
        // Clear error for this field when user starts typing
        if (errors[field]) {
            setErrors(prev => ({ ...prev, [field]: '' }));
        }
    };

    return (
        <form onSubmit={handleSubmit}>
            <div>
                <label htmlFor="name">Name</label>
                <input
                    id="name"
                    value={formData.name}
                    onChange={(e) => handleChange('name', e.target.value)}
                    aria-invalid={!!errors.name}
                    aria-describedby={errors.name ? 'name-error' : undefined}
                />
                {errors.name && (
                    <span id="name-error" role="alert">{errors.name}</span>
                )}
            </div>

            <div>
                <label htmlFor="email">Email</label>
                <input
                    id="email"
                    type="email"
                    value={formData.email}
                    onChange={(e) => handleChange('email', e.target.value)}
                    aria-invalid={!!errors.email}
                    aria-describedby={errors.email ? 'email-error' : undefined}
                />
                {errors.email && (
                    <span id="email-error" role="alert">{errors.email}</span>
                )}
            </div>

            <div>
                <label htmlFor="message">Message</label>
                <textarea
                    id="message"
                    value={formData.message}
                    onChange={(e) => handleChange('message', e.target.value)}
                    aria-invalid={!!errors.message}
                    aria-describedby={errors.message ? 'message-error' : undefined}
                />
                {errors.message && (
                    <span id="message-error" role="alert">{errors.message}</span>
                )}
            </div>

            <button type="submit" disabled={isSubmitting}>
                {isSubmitting ? 'Sending...' : 'Send Message'}
            </button>
        </form>
    );
};

describe('Counter Component', () => {
    it('should render with initial value', () => {
        render(<Counter initialValue={5} />);
        expect(screen.getByText('Counter: 5')).toBeInTheDocument();
    });

    it('should increment counter', async () => {
        const user = userEvent.setup();
        const handleChange = vi.fn();
        
        render(<Counter onValueChange={handleChange} />);
        
        const incrementButton = screen.getByRole('button', { name: /increment/i });
        
        await user.click(incrementButton);
        expect(screen.getByText('Counter: 1')).toBeInTheDocument();
        expect(handleChange).toHaveBeenCalledWith(1);
        
        await user.click(incrementButton);
        expect(screen.getByText('Counter: 2')).toBeInTheDocument();
        expect(handleChange).toHaveBeenCalledWith(2);
    });

    it('should decrement counter', async () => {
        const user = userEvent.setup();
        render(<Counter initialValue={3} />);
        
        const decrementButton = screen.getByRole('button', { name: /decrement/i });
        
        await user.click(decrementButton);
        expect(screen.getByText('Counter: 2')).toBeInTheDocument();
    });

    it('should reset counter to zero', async () => {
        const user = userEvent.setup();
        const handleChange = vi.fn();
        
        render(<Counter initialValue={10} onValueChange={handleChange} />);
        
        const resetButton = screen.getByRole('button', { name: /reset/i });
        
        await user.click(resetButton);
        expect(screen.getByText('Counter: 0')).toBeInTheDocument();
        expect(handleChange).toHaveBeenCalledWith(0);
    });
});

describe('ContactForm Component', () => {
    it('should render all form fields', () => {
        render(<ContactForm onSubmit={vi.fn()} />);
        
        expect(screen.getByLabelText(/name/i)).toBeInTheDocument();
        expect(screen.getByLabelText(/email/i)).toBeInTheDocument();
        expect(screen.getByLabelText(/message/i)).toBeInTheDocument();
        expect(screen.getByRole('button', { name: /send message/i })).toBeInTheDocument();
    });

    it('should validate required fields', async () => {
        const user = userEvent.setup();
        const handleSubmit = vi.fn();
        
        render(<ContactForm onSubmit={handleSubmit} />);
        
        // Try to submit empty form
        await user.click(screen.getByRole('button', { name: /send message/i }));
        
        // Check for validation errors
        expect(screen.getByText('Name is required')).toBeInTheDocument();
        expect(screen.getByText('Email is required')).toBeInTheDocument();
        expect(screen.getByText('Message is required')).toBeInTheDocument();
        
        // Verify form wasn't submitted
        expect(handleSubmit).not.toHaveBeenCalled();
    });

    it('should validate email format', async () => {
        const user = userEvent.setup();
        render(<ContactForm onSubmit={vi.fn()} />);
        
        // Enter invalid email
        await user.type(screen.getByLabelText(/email/i), 'invalid-email');
        await user.click(screen.getByRole('button', { name: /send message/i }));
        
        expect(screen.getByText('Invalid email format')).toBeInTheDocument();
    });

    it('should validate message length', async () => {
        const user = userEvent.setup();
        render(<ContactForm onSubmit={vi.fn()} />);
        
        // Enter short message
        await user.type(screen.getByLabelText(/message/i), 'Too short');
        await user.click(screen.getByRole('button', { name: /send message/i }));
        
        expect(screen.getByText('Message must be at least 10 characters')).toBeInTheDocument();
    });

    it('should clear errors when user types', async () => {
        const user = userEvent.setup();
        render(<ContactForm onSubmit={vi.fn()} />);
        
        // Submit empty form to show errors
        await user.click(screen.getByRole('button', { name: /send message/i }));
        expect(screen.getByText('Name is required')).toBeInTheDocument();
        
        // Start typing in name field
        await user.type(screen.getByLabelText(/name/i), 'J');
        
        // Error should be cleared
        expect(screen.queryByText('Name is required')).not.toBeInTheDocument();
    });

    it('should submit valid form data', async () => {
        const user = userEvent.setup();
        const handleSubmit = vi.fn();
        
        render(<ContactForm onSubmit={handleSubmit} />);
        
        // Fill out form
        await user.type(screen.getByLabelText(/name/i), 'John Doe');
        await user.type(screen.getByLabelText(/email/i), 'john@example.com');
        await user.type(screen.getByLabelText(/message/i), 'This is a test message');
        
        // Submit
        await user.click(screen.getByRole('button', { name: /send message/i }));
        
        // Check submission
        await waitFor(() => {
            expect(handleSubmit).toHaveBeenCalledWith({
                name: 'John Doe',
                email: 'john@example.com',
                message: 'This is a test message'
            });
        });
        
        // Form should be reset
        expect(screen.getByLabelText(/name/i)).toHaveValue('');
        expect(screen.getByLabelText(/email/i)).toHaveValue('');
        expect(screen.getByLabelText(/message/i)).toHaveValue('');
    });

    it('should show loading state during submission', async () => {
        const user = userEvent.setup();
        render(<ContactForm onSubmit={vi.fn()} />);
        
        // Fill valid data
        await user.type(screen.getByLabelText(/name/i), 'John Doe');
        await user.type(screen.getByLabelText(/email/i), 'john@example.com');
        await user.type(screen.getByLabelText(/message/i), 'This is a test message');
        
        // Submit
        await user.click(screen.getByRole('button', { name: /send message/i }));
        
        // Should show loading state
        expect(screen.getByRole('button', { name: /sending/i })).toBeDisabled();
        
        // Wait for submission to complete
        await waitFor(() => {
            expect(screen.getByRole('button', { name: /send message/i })).toBeEnabled();
        });
    });
});

/**
 * These tests demonstrate:
 * 
 * 1. ✅ Simple unit tests that run fast without database
 * 2. ✅ No special configuration needed - just write tests
 * 3. ✅ Full React Testing Library support
 * 4. ✅ Vitest's built-in DOM assertions (toBeInTheDocument, etc.)
 * 5. ✅ Component state and interaction testing
 * 6. ✅ Form validation and submission testing
 * 
 * These tests run in milliseconds because they don't need testcontainers.
 */