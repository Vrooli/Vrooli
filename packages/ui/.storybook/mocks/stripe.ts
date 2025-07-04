// Mock Stripe for Storybook to prevent network errors and console warnings

// Mock Stripe object that won't make network requests
export const mockStripe = {
    elements: () => ({
        create: () => ({
            mount: () => {},
            unmount: () => {},
            on: () => {},
            off: () => {},
            update: () => {},
            clear: () => {},
            focus: () => {},
            blur: () => {},
            destroy: () => {},
        }),
    }),
    confirmPayment: () => Promise.resolve({ error: null }),
    confirmCardPayment: () => Promise.resolve({ error: null }),
    createToken: () => Promise.resolve({ token: null, error: null }),
    createPaymentMethod: () => Promise.resolve({ paymentMethod: null, error: null }),
    retrievePaymentIntent: () => Promise.resolve({ paymentIntent: null, error: null }),
};

// Prevent the Stripe SDK from loading its external scripts
if (typeof window !== 'undefined') {
    // Override document.createElement to prevent Stripe script injection
    const originalCreateElement = document.createElement.bind(document);
    document.createElement = function(tagName: string) {
        const element = originalCreateElement(tagName);
        if (tagName.toLowerCase() === 'script') {
            const originalSetAttribute = element.setAttribute.bind(element);
            element.setAttribute = function(name: string, value: string) {
                // Block Stripe SDK scripts
                if (name === 'src' && value.includes('stripe.com')) {
                    console.info('Blocked Stripe SDK script:', value);
                    return;
                }
                return originalSetAttribute(name, value);
            };
        }
        return element;
    };
}

// Mock loadStripe function
export const loadStripe = () => {
    console.info('Stripe mock: Using mocked Stripe in Storybook');
    return Promise.resolve(null);
};

// Export Stripe constructor for any direct usage
export const Stripe = () => mockStripe;