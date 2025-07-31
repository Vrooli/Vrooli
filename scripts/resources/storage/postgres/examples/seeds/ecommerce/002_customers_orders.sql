-- Ecommerce Seed Data: Customers and Orders
-- Description: Sample customer and order data for ecommerce automation

-- Create customers table
CREATE TABLE IF NOT EXISTS customers (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    phone VARCHAR(20),
    date_of_birth DATE,
    registration_date TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_login TIMESTAMP WITH TIME ZONE,
    customer_group VARCHAR(50) DEFAULT 'regular',
    total_spent DECIMAL(12,2) DEFAULT 0.00,
    order_count INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT true
);

-- Create addresses table
CREATE TABLE IF NOT EXISTS addresses (
    id SERIAL PRIMARY KEY,
    customer_id INTEGER REFERENCES customers(id) ON DELETE CASCADE,
    type VARCHAR(20) NOT NULL, -- 'billing', 'shipping'
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    company VARCHAR(100),
    address_line1 VARCHAR(255) NOT NULL,
    address_line2 VARCHAR(255),
    city VARCHAR(100) NOT NULL,
    state VARCHAR(100),
    postal_code VARCHAR(20),
    country VARCHAR(100) NOT NULL,
    is_default BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create orders table
CREATE TABLE IF NOT EXISTS orders (
    id SERIAL PRIMARY KEY,
    order_number VARCHAR(50) UNIQUE NOT NULL,
    customer_id INTEGER REFERENCES customers(id),
    status VARCHAR(50) DEFAULT 'pending',
    subtotal DECIMAL(12,2) NOT NULL,
    tax_amount DECIMAL(10,2) DEFAULT 0.00,
    shipping_amount DECIMAL(10,2) DEFAULT 0.00,
    discount_amount DECIMAL(10,2) DEFAULT 0.00,
    total_amount DECIMAL(12,2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'USD',
    payment_status VARCHAR(50) DEFAULT 'pending',
    payment_method VARCHAR(100),
    shipping_method VARCHAR(100),
    notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create order items table
CREATE TABLE IF NOT EXISTS order_items (
    id SERIAL PRIMARY KEY,
    order_id INTEGER REFERENCES orders(id) ON DELETE CASCADE,
    product_id INTEGER REFERENCES products(id),
    sku VARCHAR(100),
    product_name VARCHAR(255) NOT NULL,
    quantity INTEGER NOT NULL,
    unit_price DECIMAL(10,2) NOT NULL,
    total_price DECIMAL(12,2) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Insert sample customers
INSERT INTO customers (email, first_name, last_name, phone, customer_group, total_spent, order_count, last_login) VALUES
('alice.cooper@email.com', 'Alice', 'Cooper', '555-2001', 'vip', 2599.97, 3, NOW() - INTERVAL '2 hours'),
('bob.martin@email.com', 'Bob', 'Martin', '555-2002', 'regular', 459.98, 2, NOW() - INTERVAL '1 day'),
('carol.williams@email.com', 'Carol', 'Williams', '555-2003', 'regular', 29.99, 1, NOW() - INTERVAL '3 days'),
('david.brown@email.com', 'David', 'Brown', '555-2004', 'regular', 1549.98, 2, NOW() - INTERVAL '1 week'),
('eva.johnson@email.com', 'Eva', 'Johnson', '555-2005', 'vip', 1899.96, 4, NOW() - INTERVAL '1 hour');

-- Insert sample addresses
INSERT INTO addresses (customer_id, type, first_name, last_name, address_line1, city, state, postal_code, country, is_default) VALUES
(1, 'billing', 'Alice', 'Cooper', '123 Main Street', 'New York', 'NY', '10001', 'USA', true),
(1, 'shipping', 'Alice', 'Cooper', '123 Main Street', 'New York', 'NY', '10001', 'USA', true),
(2, 'billing', 'Bob', 'Martin', '456 Oak Avenue', 'Los Angeles', 'CA', '90001', 'USA', true),
(2, 'shipping', 'Bob', 'Martin', '456 Oak Avenue', 'Los Angeles', 'CA', '90001', 'USA', true),
(3, 'billing', 'Carol', 'Williams', '789 Pine Road', 'Chicago', 'IL', '60601', 'USA', true),
(4, 'billing', 'David', 'Brown', '321 Elm Drive', 'Houston', 'TX', '77001', 'USA', true),
(5, 'billing', 'Eva', 'Johnson', '654 Maple Lane', 'Phoenix', 'AZ', '85001', 'USA', true);

-- Insert sample orders
INSERT INTO orders (order_number, customer_id, status, subtotal, tax_amount, shipping_amount, total_amount, payment_status, payment_method, shipping_method) VALUES
('ORD-2025-001', 1, 'completed', 1299.99, 130.00, 19.99, 1449.98, 'paid', 'Credit Card', 'Standard Shipping'),
('ORD-2025-002', 2, 'shipped', 399.99, 40.00, 19.99, 459.98, 'paid', 'PayPal', 'Standard Shipping'),
('ORD-2025-003', 3, 'completed', 29.99, 3.00, 9.99, 42.98, 'paid', 'Credit Card', 'Economy Shipping'),
('ORD-2025-004', 1, 'processing', 1149.99, 115.00, 0.00, 1264.99, 'paid', 'Credit Card', 'Express Shipping'),
('ORD-2025-005', 4, 'completed', 1299.99, 130.00, 19.99, 1449.98, 'paid', 'Bank Transfer', 'Standard Shipping'),
('ORD-2025-006', 5, 'pending', 249.99, 25.00, 9.99, 284.98, 'pending', 'Credit Card', 'Standard Shipping');

-- Insert sample order items
INSERT INTO order_items (order_id, product_id, sku, product_name, quantity, unit_price, total_price) VALUES
(1, 1, 'LAPTOP001', 'UltraBook Pro 15"', 1, 1299.99, 1299.99),
(2, 3, 'DESK001', 'Ergonomic Office Desk', 1, 399.99, 399.99),
(3, 4, 'TSHIRT001', 'Premium Cotton T-Shirt', 1, 29.99, 29.99),
(4, 2, 'PHONE001', 'SmartPhone X1', 1, 899.99, 899.99),
(4, 5, 'HEADPHONE001', 'Wireless Noise-Canceling Headphones', 1, 249.99, 249.99),
(5, 1, 'LAPTOP001', 'UltraBook Pro 15"', 1, 1299.99, 1299.99),
(6, 5, 'HEADPHONE001', 'Wireless Noise-Canceling Headphones', 1, 249.99, 249.99);

-- Create shopping cart table for abandoned cart automation
CREATE TABLE IF NOT EXISTS shopping_carts (
    id SERIAL PRIMARY KEY,
    customer_id INTEGER REFERENCES customers(id) ON DELETE CASCADE,
    product_id INTEGER REFERENCES products(id) ON DELETE CASCADE,
    quantity INTEGER NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Insert sample abandoned cart items
INSERT INTO shopping_carts (customer_id, product_id, quantity) VALUES
(3, 2, 1), -- Carol has a phone in cart
(4, 5, 2); -- David has 2 headphones in cart