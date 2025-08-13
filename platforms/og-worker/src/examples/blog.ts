/**
 * BLOG APPLICATION EXAMPLE
 * This example shows how to implement OG metadata for a blog or content site
 */

// ============================================================================
// API Response Types
// ============================================================================

interface BlogPost {
    id: string;
    slug: string;
    title: string;
    excerpt: string;
    content: string;
    featuredImage?: string;
    author: {
        name: string;
        bio?: string;
        avatar?: string;
        twitter?: string;
    };
    category: {
        name: string;
        slug: string;
    };
    tags: string[];
    publishedAt: string;
    updatedAt: string;
    readingTime: number; // in minutes
}

interface Author {
    id: string;
    username: string;
    name: string;
    bio: string;
    avatar?: string;
    social: {
        twitter?: string;
        linkedin?: string;
        github?: string;
    };
    postCount: number;
}

interface Category {
    slug: string;
    name: string;
    description: string;
    postCount: number;
    featuredImage?: string;
}

// ============================================================================
// Route Configuration
// ============================================================================

export const BLOG_ROUTES = [
    { pattern: "/blog/:slug", handler: getBlogPostMeta },
    { pattern: "/posts/:slug", handler: getBlogPostMeta }, // Alternative URL structure
    { pattern: "/author/:username", handler: getAuthorMeta },
    { pattern: "/category/:slug", handler: getCategoryMeta },
    { pattern: "/tag/:tag", handler: getTagMeta },
];

// ============================================================================
// Metadata Fetchers
// ============================================================================

/**
 * Generate metadata for a blog post
 */
export async function getBlogPostMeta(
    params: { slug: string },
    env: any,
    baseUrl: string
): Promise<any> {
    try {
        const apiUrl = env.API_URL || `https://${env.ORIGIN_HOST}`;
        const response = await fetch(`${apiUrl}/api/posts/${params.slug}`, {
            cf: { resolveOverride: env.ORIGIN_HOST }
        });
        
        if (!response.ok) throw new Error("Post not found");
        
        const post = await response.json() as BlogPost;
        
        // Generate description from excerpt or content
        const description = post.excerpt || 
            post.content.substring(0, 160).replace(/<[^>]*>/g, '') + '...';
        
        return {
            title: post.title,
            description,
            image: post.featuredImage || `${baseUrl}/default-blog-image.jpg`,
            url: `${baseUrl}/blog/${post.slug}`,
            type: "article",
            siteName: env.SITE_NAME || "Blog",
            locale: env.LOCALE || "en_US",
            
            // Article-specific metadata
            extra: {
                "article:author": post.author.name,
                "article:published_time": post.publishedAt,
                "article:modified_time": post.updatedAt,
                "article:section": post.category.name,
                "article:tag": post.tags.join(", "),
                "twitter:label1": "Written by",
                "twitter:data1": post.author.name,
                "twitter:label2": "Reading time",
                "twitter:data2": `${post.readingTime} min read`,
                ...(post.author.twitter && {
                    "twitter:creator": `@${post.author.twitter}`
                })
            }
        };
    } catch (error) {
        console.error("Error fetching blog post:", error);
        return {
            title: "Blog Post",
            description: "Read our latest blog post",
            image: `${baseUrl}/default-blog-image.jpg`,
            url: `${baseUrl}/blog/${params.slug}`
        };
    }
}

/**
 * Generate metadata for an author page
 */
export async function getAuthorMeta(
    params: { username: string },
    env: any,
    baseUrl: string
): Promise<any> {
    try {
        const apiUrl = env.API_URL || `https://${env.ORIGIN_HOST}`;
        const response = await fetch(`${apiUrl}/api/authors/${params.username}`, {
            cf: { resolveOverride: env.ORIGIN_HOST }
        });
        
        if (!response.ok) throw new Error("Author not found");
        
        const author = await response.json() as Author;
        
        return {
            title: `${author.name} - Author`,
            description: author.bio || `Read articles by ${author.name}`,
            image: author.avatar || `${baseUrl}/default-avatar.jpg`,
            url: `${baseUrl}/author/${author.username}`,
            type: "profile",
            
            extra: {
                "profile:first_name": author.name.split(' ')[0],
                "profile:last_name": author.name.split(' ').slice(1).join(' '),
                "profile:username": author.username,
                ...(author.social.twitter && {
                    "twitter:creator": `@${author.social.twitter}`
                })
            }
        };
    } catch (error) {
        return {
            title: "Author Profile",
            description: "View author's articles",
            image: `${baseUrl}/default-avatar.jpg`,
            url: `${baseUrl}/author/${params.username}`
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
        
        return {
            title: `${category.name} Articles`,
            description: category.description || 
                `Browse ${category.postCount} articles in ${category.name}`,
            image: category.featuredImage || `${baseUrl}/default-category.jpg`,
            url: `${baseUrl}/category/${category.slug}`,
            type: "website"
        };
    } catch (error) {
        return {
            title: "Category",
            description: "Browse articles by category",
            image: `${baseUrl}/default-category.jpg`,
            url: `${baseUrl}/category/${params.slug}`
        };
    }
}

/**
 * Generate metadata for a tag page
 */
export async function getTagMeta(
    params: { tag: string },
    env: any,
    baseUrl: string
): Promise<any> {
    // Tags are usually simpler - might not need API call
    const tagName = params.tag.replace(/-/g, ' ')
        .replace(/\b\w/g, l => l.toUpperCase());
    
    return {
        title: `Articles tagged "${tagName}"`,
        description: `Browse all articles tagged with ${tagName}`,
        image: `${baseUrl}/default-tag.jpg`,
        url: `${baseUrl}/tag/${params.tag}`,
        type: "website"
    };
}

// ============================================================================
// Helper Functions
// ============================================================================

/**
 * Generate a dynamic OG image for blog posts
 * This could call an image generation service
 */
export function generateBlogOGImage(
    post: BlogPost,
    env: any
): string {
    // Example using Cloudinary's text overlay feature
    if (env.CLOUDINARY_CLOUD_NAME) {
        const text = encodeURIComponent(post.title);
        const author = encodeURIComponent(post.author.name);
        
        return `https://res.cloudinary.com/${env.CLOUDINARY_CLOUD_NAME}/image/upload/` +
               `w_1200,h_630,c_fit,q_auto,f_auto/` +
               `l_text:Roboto_72_bold:${text},co_rgb:FFFFFF,c_fit,w_1000,h_400/` +
               `fl_layer_apply,g_center/` +
               `l_text:Roboto_36:${author},co_rgb:CCCCCC,c_fit,w_1000/` +
               `fl_layer_apply,g_south,y_100/` +
               `blog-og-template.jpg`;
    }
    
    // Fallback to featured image or default
    return post.featuredImage || `${env.SITE_URL}/default-blog-og.jpg`;
}

/**
 * Format reading time for display
 */
export function formatReadingTime(minutes: number): string {
    if (minutes < 1) return "Less than 1 min read";
    if (minutes === 1) return "1 min read";
    return `${Math.round(minutes)} min read`;
}

/**
 * Generate RSS feed URL for author/category
 */
export function getRSSUrl(type: 'author' | 'category' | 'tag', slug: string, baseUrl: string): string {
    return `${baseUrl}/rss/${type}/${slug}.xml`;
}