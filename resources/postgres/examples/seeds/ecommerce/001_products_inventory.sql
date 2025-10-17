-- Ecommerce Seed Data: Products and Inventory
-- Description: Sample product catalog and inventory data for ecommerce automation

-- Create categories table
CREATE TABLE IF NOT EXISTS categories (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    parent_id INTEGER REFERENCES categories(id),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create products table
CREATE TABLE IF NOT EXISTS products (
    id SERIAL PRIMARY KEY,
    sku VARCHAR(100) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) UNIQUE NOT NULL,
    description TEXT,
    short_description TEXT,
    price DECIMAL(10,2) NOT NULL,
    cost_price DECIMAL(10,2),
    category_id INTEGER REFERENCES categories(id),
    brand VARCHAR(100),
    weight DECIMAL(8,2),
    dimensions JSONB,
    is_active BOOLEAN DEFAULT true,
    is_featured BOOLEAN DEFAULT false,
    meta_title VARCHAR(255),
    meta_description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create inventory table
CREATE TABLE IF NOT EXISTS inventory (
    id SERIAL PRIMARY KEY,
    product_id INTEGER REFERENCES products(id) ON DELETE CASCADE,
    quantity_available INTEGER NOT NULL DEFAULT 0,
    quantity_reserved INTEGER NOT NULL DEFAULT 0,
    reorder_point INTEGER DEFAULT 10,
    reorder_quantity INTEGER DEFAULT 50,
    last_restocked TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Insert sample categories
INSERT INTO categories (name, slug, description) VALUES
('Electronics', 'electronics', 'Electronic devices and accessories'),
('Computers', 'computers', 'Laptops, desktops, and computer accessories'),
('Smartphones', 'smartphones', 'Mobile phones and accessories'),
('Home & Garden', 'home-garden', 'Home improvement and garden supplies'),
('Clothing', 'clothing', 'Apparel and fashion accessories');

-- Insert sample products
INSERT INTO products (sku, name, slug, description, short_description, price, cost_price, category_id, brand, weight, dimensions) VALUES
('LAPTOP001', 'UltraBook Pro 15"', 'ultrabook-pro-15', 'Professional laptop with high-performance processor and stunning display', 'High-performance laptop for professionals', 1299.99, 899.00, 2, 'TechPro', 2.1, '{"length": 35, "width": 24, "height": 2}'),
('PHONE001', 'SmartPhone X1', 'smartphone-x1', 'Latest smartphone with advanced camera and all-day battery', 'Premium smartphone with excellent camera', 899.99, 599.00, 3, 'MobileTech', 0.18, '{"length": 15, "width": 7, "height": 0.8}'),
('DESK001', 'Ergonomic Office Desk', 'ergonomic-office-desk', 'Height-adjustable desk perfect for home office setup', 'Height-adjustable standing desk', 399.99, 249.00, 4, 'HomeOffice', 25.0, '{"length": 140, "width": 70, "height": 75}'),
('TSHIRT001', 'Premium Cotton T-Shirt', 'premium-cotton-tshirt', 'Soft, comfortable cotton t-shirt in various colors', 'Premium quality cotton t-shirt', 29.99, 12.00, 5, 'ComfortWear', 0.2, '{"length": 70, "width": 50, "height": 1}'),
('HEADPHONE001', 'Wireless Noise-Canceling Headphones', 'wireless-noise-canceling-headphones', 'Professional-grade headphones with active noise cancellation', 'Premium wireless headphones', 249.99, 149.00, 1, 'AudioPro', 0.3, '{"length": 20, "width": 18, "height": 8}');

-- Insert sample inventory
INSERT INTO inventory (product_id, quantity_available, quantity_reserved, reorder_point, reorder_quantity, last_restocked) VALUES
(1, 25, 2, 5, 20, NOW() - INTERVAL '10 days'),
(2, 50, 5, 10, 30, NOW() - INTERVAL '5 days'),
(3, 8, 1, 3, 10, NOW() - INTERVAL '15 days'),
(4, 150, 10, 25, 100, NOW() - INTERVAL '7 days'),
(5, 35, 3, 8, 25, NOW() - INTERVAL '12 days');

-- Create product attributes table for variant management
CREATE TABLE IF NOT EXISTS product_attributes (
    id SERIAL PRIMARY KEY,
    product_id INTEGER REFERENCES products(id) ON DELETE CASCADE,
    attribute_name VARCHAR(100) NOT NULL,
    attribute_value VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Insert sample product attributes
INSERT INTO product_attributes (product_id, attribute_name, attribute_value) VALUES
(1, 'Color', 'Silver'),
(1, 'Storage', '512GB SSD'),
(1, 'RAM', '16GB'),
(2, 'Color', 'Black'),
(2, 'Storage', '128GB'),
(4, 'Size', 'Medium'),
(4, 'Color', 'Navy Blue'),
(5, 'Color', 'Black'),
(5, 'Connectivity', 'Bluetooth 5.0');