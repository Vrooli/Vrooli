/**
 * E-COMMERCE APPLICATION EXAMPLE
 * This example shows how to implement OG metadata for an online store
 */

// ============================================================================
// API Response Types
// ============================================================================

interface Product {
    id: string;
    sku: string;
    name: string;
    description: string;
    shortDescription?: string;
    price: {
        amount: number;
        currency: string;
        compareAt?: number; // Original price if on sale
        salePercentage?: number;
    };
    images: {
        url: string;
        alt?: string;
        isPrimary?: boolean;
    }[];
    category: {
        id: string;
        name: string;
        slug: string;
    };
    brand: {
        name: string;
        logo?: string;
    };
    inventory: {
        inStock: boolean;
        quantity?: number;
        lowStock?: boolean;
    };
    rating: {
        average: number;
        count: number;
    };
    variants?: {
        id: string;
        name: string;
        options: string[];
    }[];
    seo?: {
        title?: string;
        description?: string;
        keywords?: string[];
    };
}

interface Category {
    id: string;
    slug: string;
    name: string;
    description: string;
    image?: string;
    productCount: number;
    parent?: {
        id: string;
        name: string;
        slug: string;
    };
}

interface Brand {
    slug: string;
    name: string;
    description: string;
    logo: string;
    website?: string;
    productCount: number;
}

interface Collection {
    id: string;
    slug: string;
    name: string;
    description: string;
    featuredImage: string;
    products: string[]; // Product IDs
    validUntil?: string; // For seasonal collections
}

// ============================================================================
// Route Configuration
// ============================================================================

export const ECOMMERCE_ROUTES = [
    { pattern: "/product/:id", handler: getProductMeta },
    { pattern: "/p/:sku", handler: getProductBySKUMeta }, // Alternative URL
    { pattern: "/category/:slug", handler: getCategoryMeta },
    { pattern: "/brand/:slug", handler: getBrandMeta },
    { pattern: "/collection/:slug", handler: getCollectionMeta },
    { pattern: "/sale", handler: getSaleMeta },
    { pattern: "/new-arrivals", handler: getNewArrivalsMeta },
];

// ============================================================================
// Metadata Fetchers
// ============================================================================

/**
 * Generate metadata for a product page
 */
export async function getProductMeta(
    params: { id: string },
    env: any,
    baseUrl: string
): Promise<any> {
    try {
        const apiUrl = env.API_URL || `https://${env.ORIGIN_HOST}`;
        const response = await fetch(`${apiUrl}/api/products/${params.id}`, {
            cf: { resolveOverride: env.ORIGIN_HOST }
        });
        
        if (!response.ok) throw new Error("Product not found");
        
        const product = await response.json() as Product;
        
        // Use SEO overrides if available
        const title = product.seo?.title || 
            `${product.name} - ${product.brand.name}`;
        
        const description = product.seo?.description || 
            product.shortDescription || 
            product.description.substring(0, 160);
        
        // Get primary image or first image
        const primaryImage = product.images.find(img => img.isPrimary) || product.images[0];
        
        // Build price string
        const priceString = formatPrice(product.price);
        const availability = product.inventory.inStock ? "in stock" : "out of stock";
        
        return {
            title: `${title} - ${priceString}`,
            description,
            image: primaryImage?.url || `${baseUrl}/default-product.jpg`,
            url: `${baseUrl}/product/${product.id}`,
            type: "product",
            siteName: env.SITE_NAME || "Store",
            
            // Product-specific metadata
            extra: {
                "product:price:amount": product.price.amount.toString(),
                "product:price:currency": product.price.currency,
                "product:availability": availability,
                "product:condition": "new",
                "product:brand": product.brand.name,
                "product:category": product.category.name,
                
                // Add sale price if applicable
                ...(product.price.compareAt && {
                    "product:sale_price:amount": product.price.amount.toString(),
                    "product:sale_price:currency": product.price.currency,
                    "product:original_price:amount": product.price.compareAt.toString(),
                    "product:original_price:currency": product.price.currency,
                }),
                
                // Rich pins for Pinterest
                "og:price:amount": product.price.amount.toString(),
                "og:price:currency": product.price.currency,
                
                // Twitter labels
                "twitter:label1": "Price",
                "twitter:data1": priceString,
                "twitter:label2": "Availability",
                "twitter:data2": availability,
                
                // Rating if available
                ...(product.rating.count > 0 && {
                    "product:rating:average": product.rating.average.toString(),
                    "product:rating:count": product.rating.count.toString(),
                })
            }
        };
    } catch (error) {
        console.error("Error fetching product:", error);
        return {
            title: "Product",
            description: "View product details",
            image: `${baseUrl}/default-product.jpg`,
            url: `${baseUrl}/product/${params.id}`
        };
    }
}

/**
 * Alternative product route using SKU
 */
export async function getProductBySKUMeta(
    params: { sku: string },
    env: any,
    baseUrl: string
): Promise<any> {
    try {
        const apiUrl = env.API_URL || `https://${env.ORIGIN_HOST}`;
        const response = await fetch(`${apiUrl}/api/products/sku/${params.sku}`, {
            cf: { resolveOverride: env.ORIGIN_HOST }
        });
        
        if (!response.ok) throw new Error("Product not found");
        
        const product = await response.json() as Product;
        
        // Reuse the main product meta function logic
        return getProductMeta({ id: product.id }, env, baseUrl);
    } catch (error) {
        return {
            title: "Product",
            description: "View product details",
            image: `${baseUrl}/default-product.jpg`,
            url: `${baseUrl}/p/${params.sku}`
        };
    }
}

/**
 * Generate metadata for a category page
 */
export async function getCategoryMeta(
    params: { slug: string },
    env: any,
    baseUrl: string
): Promise<any> {
    try {
        const apiUrl = env.API_URL || `https://${env.ORIGIN_HOST}`;
        const response = await fetch(`${apiUrl}/api/categories/${params.slug}`, {
            cf: { resolveOverride: env.ORIGIN_HOST }
        });
        
        if (!response.ok) throw new Error("Category not found");
        
        const category = await response.json() as Category;
        
        const title = category.parent 
            ? `${category.name} - ${category.parent.name}`
            : category.name;
        
        return {
            title: `${title} - Shop Now`,
            description: category.description || 
                `Browse ${category.productCount} products in ${category.name}`,
            image: category.image || `${baseUrl}/default-category.jpg`,
            url: `${baseUrl}/category/${category.slug}`,
            type: "website",
            
            extra: {
                "product:category": category.name,
                ...(category.parent && {
                    "product:category:parent": category.parent.name
                })
            }
        };
    } catch (error) {
        return {
            title: "Shop by Category",
            description: "Browse our product categories",
            image: `${baseUrl}/default-category.jpg`,
            url: `${baseUrl}/category/${params.slug}`
        };
    }
}

/**
 * Generate metadata for a brand page
 */
export async function getBrandMeta(
    params: { slug: string },
    env: any,
    baseUrl: string
): Promise<any> {
    try {
        const apiUrl = env.API_URL || `https://${env.ORIGIN_HOST}`;
        const response = await fetch(`${apiUrl}/api/brands/${params.slug}`, {
            cf: { resolveOverride: env.ORIGIN_HOST }
        });
        
        if (!response.ok) throw new Error("Brand not found");
        
        const brand = await response.json() as Brand;
        
        return {
            title: `${brand.name} - Official Store`,
            description: brand.description || 
                `Shop ${brand.productCount} ${brand.name} products`,
            image: brand.logo || `${baseUrl}/default-brand.jpg`,
            url: `${baseUrl}/brand/${brand.slug}`,
            type: "website",
            
            extra: {
                "og:type": "business.business",
                "business:contact_data:website": brand.website || baseUrl,
            }
        };
    } catch (error) {
        return {
            title: "Shop by Brand",
            description: "Browse products by brand",
            image: `${baseUrl}/default-brand.jpg`,
            url: `${baseUrl}/brand/${params.slug}`
        };
    }
}

/**
 * Generate metadata for a collection page
 */
export async function getCollectionMeta(
    params: { slug: string },
    env: any,
    baseUrl: string
): Promise<any> {
    try {
        const apiUrl = env.API_URL || `https://${env.ORIGIN_HOST}`;
        const response = await fetch(`${apiUrl}/api/collections/${params.slug}`, {
            cf: { resolveOverride: env.ORIGIN_HOST }
        });
        
        if (!response.ok) throw new Error("Collection not found");
        
        const collection = await response.json() as Collection;
        
        const urgency = collection.validUntil 
            ? ` - Available until ${new Date(collection.validUntil).toLocaleDateString()}`
            : '';
        
        return {
            title: `${collection.name}${urgency}`,
            description: collection.description,
            image: collection.featuredImage || `${baseUrl}/default-collection.jpg`,
            url: `${baseUrl}/collection/${collection.slug}`,
            type: "website"
        };
    } catch (error) {
        return {
            title: "Featured Collection",
            description: "Shop our curated collection",
            image: `${baseUrl}/default-collection.jpg`,
            url: `${baseUrl}/collection/${params.slug}`
        };
    }
}

/**
 * Generate metadata for sale page
 */
export async function getSaleMeta(
    _params: any,
    env: any,
    baseUrl: string
): Promise<any> {
    return {
        title: "Sale - Up to 70% Off",
        description: "Shop amazing deals on your favorite products. Limited time offers!",
        image: `${baseUrl}/sale-banner.jpg`,
        url: `${baseUrl}/sale`,
        type: "website",
        
        extra: {
            "og:type": "website",
            "twitter:label1": "Sale Event",
            "twitter:data1": "Now Live",
            "twitter:label2": "Discount",
            "twitter:data2": "Up to 70% Off"
        }
    };
}

/**
 * Generate metadata for new arrivals page
 */
export async function getNewArrivalsMeta(
    _params: any,
    env: any,
    baseUrl: string
): Promise<any> {
    return {
        title: "New Arrivals - Latest Products",
        description: "Discover the latest additions to our collection. Shop new arrivals now!",
        image: `${baseUrl}/new-arrivals-banner.jpg`,
        url: `${baseUrl}/new-arrivals`,
        type: "website"
    };
}

// ============================================================================
// Helper Functions
// ============================================================================

/**
 * Format price for display
 */
function formatPrice(price: { amount: number; currency: string; compareAt?: number }): string {
    const formatter = new Intl.NumberFormat('en-US', {
        style: 'currency',
        currency: price.currency
    });
    
    if (price.compareAt && price.compareAt > price.amount) {
        const savings = Math.round(((price.compareAt - price.amount) / price.compareAt) * 100);
        return `${formatter.format(price.amount)} (${savings}% OFF)`;
    }
    
    return formatter.format(price.amount);
}

/**
 * Generate structured data for products (JSON-LD)
 */
export function generateProductStructuredData(product: Product, baseUrl: string): any {
    return {
        "@context": "https://schema.org",
        "@type": "Product",
        "name": product.name,
        "description": product.description,
        "image": product.images.map(img => img.url),
        "brand": {
            "@type": "Brand",
            "name": product.brand.name
        },
        "offers": {
            "@type": "Offer",
            "url": `${baseUrl}/product/${product.id}`,
            "priceCurrency": product.price.currency,
            "price": product.price.amount,
            "availability": product.inventory.inStock 
                ? "https://schema.org/InStock" 
                : "https://schema.org/OutOfStock",
            "seller": {
                "@type": "Organization",
                "name": "Your Store Name"
            }
        },
        ...(product.rating.count > 0 && {
            "aggregateRating": {
                "@type": "AggregateRating",
                "ratingValue": product.rating.average,
                "reviewCount": product.rating.count
            }
        })
    };
}