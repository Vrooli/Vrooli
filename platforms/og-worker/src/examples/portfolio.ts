/**
 * PORTFOLIO/PERSONAL WEBSITE EXAMPLE
 * This example shows how to implement OG metadata for portfolios, personal sites, and professional profiles
 */

// ============================================================================
// API Response Types
// ============================================================================

interface Project {
    id: string;
    slug: string;
    title: string;
    subtitle?: string;
    description: string;
    thumbnail: string;
    images: string[];
    client?: {
        name: string;
        logo?: string;
        industry?: string;
    };
    role: string;
    technologies: string[];
    skills: string[];
    timeline: {
        startDate: string;
        endDate?: string; // Optional for ongoing projects
        duration?: string; // e.g., "3 months"
    };
    links: {
        live?: string;
        github?: string;
        casestudy?: string;
        behance?: string;
        dribbble?: string;
    };
    metrics?: {
        label: string;
        value: string;
    }[];
    featured: boolean;
    category: 'web' | 'mobile' | 'design' | 'research' | 'other';
}

interface Profile {
    name: string;
    title: string;
    tagline?: string;
    bio: string;
    avatar: string;
    coverImage?: string;
    location?: {
        city?: string;
        country?: string;
        remote?: boolean;
    };
    contact: {
        email?: string;
        phone?: string;
        website?: string;
    };
    social: {
        linkedin?: string;
        github?: string;
        twitter?: string;
        instagram?: string;
        behance?: string;
        dribbble?: string;
        medium?: string;
    };
    skills: {
        category: string;
        items: string[];
    }[];
    experience: {
        company: string;
        role: string;
        duration: string;
        current?: boolean;
    }[];
    availability?: {
        status: 'available' | 'busy' | 'not-available';
        message?: string;
    };
}

interface Service {
    id: string;
    slug: string;
    name: string;
    description: string;
    icon?: string;
    image?: string;
    features: string[];
    process?: {
        step: number;
        title: string;
        description: string;
    }[];
    pricing?: {
        type: 'fixed' | 'hourly' | 'project' | 'custom';
        amount?: number;
        currency?: string;
        note?: string;
    };
    deliverables?: string[];
    timeline?: string;
}

interface BlogPost {
    slug: string;
    title: string;
    excerpt: string;
    content: string;
    featuredImage?: string;
    publishedAt: string;
    readingTime: number;
    tags: string[];
}

interface Testimonial {
    id: string;
    client: {
        name: string;
        role: string;
        company: string;
        avatar?: string;
    };
    content: string;
    rating?: number;
    project?: string; // Project ID reference
    date: string;
}

// ============================================================================
// Route Configuration
// ============================================================================

export const PORTFOLIO_ROUTES = [
    { pattern: "/", handler: getHomeMeta },
    { pattern: "/about", handler: getAboutMeta },
    { pattern: "/portfolio", handler: getPortfolioMeta },
    { pattern: "/project/:slug", handler: getProjectMeta },
    { pattern: "/work/:slug", handler: getProjectMeta }, // Alternative URL
    { pattern: "/case-study/:slug", handler: getCaseStudyMeta },
    { pattern: "/service/:slug", handler: getServiceMeta },
    { pattern: "/services", handler: getServicesMeta },
    { pattern: "/blog/:slug", handler: getBlogPostMeta },
    { pattern: "/contact", handler: getContactMeta },
    { pattern: "/resume", handler: getResumeMeta },
];

// ============================================================================
// Metadata Fetchers
// ============================================================================

/**
 * Generate metadata for homepage
 */
export async function getHomeMeta(
    _params: any,
    env: any,
    baseUrl: string
): Promise<any> {
    try {
        const apiUrl = env.API_URL || `https://${env.ORIGIN_HOST}`;
        const response = await fetch(`${apiUrl}/api/profile`, {
            cf: { resolveOverride: env.ORIGIN_HOST }
        });
        
        if (!response.ok) throw new Error("Profile not found");
        
        const profile = await response.json() as Profile;
        
        return {
            title: `${profile.name} - ${profile.title}`,
            description: profile.tagline || profile.bio.substring(0, 160),
            image: profile.coverImage || profile.avatar || `${baseUrl}/og-image.jpg`,
            url: baseUrl,
            type: "website",
            
            extra: {
                "og:type": "profile",
                "profile:first_name": profile.name.split(' ')[0],
                "profile:last_name": profile.name.split(' ').slice(1).join(' '),
                ...(profile.social.twitter && {
                    "twitter:creator": `@${profile.social.twitter}`
                })
            }
        };
    } catch (error) {
        return {
            title: "Portfolio",
            description: "Welcome to my portfolio",
            image: `${baseUrl}/og-image.jpg`,
            url: baseUrl
        };
    }
}

/**
 * Generate metadata for about page
 */
export async function getAboutMeta(
    _params: any,
    env: any,
    baseUrl: string
): Promise<any> {
    try {
        const apiUrl = env.API_URL || `https://${env.ORIGIN_HOST}`;
        const response = await fetch(`${apiUrl}/api/profile`, {
            cf: { resolveOverride: env.ORIGIN_HOST }
        });
        
        if (!response.ok) throw new Error("Profile not found");
        
        const profile = await response.json() as Profile;
        
        const locationStr = profile.location 
            ? `${profile.location.city}, ${profile.location.country}`
            : '';
        
        return {
            title: `About ${profile.name}`,
            description: profile.bio,
            image: profile.avatar || `${baseUrl}/about-og.jpg`,
            url: `${baseUrl}/about`,
            type: "profile",
            
            extra: {
                "og:type": "profile",
                "profile:first_name": profile.name.split(' ')[0],
                "profile:last_name": profile.name.split(' ').slice(1).join(' '),
                ...(locationStr && {
                    "og:locale": locationStr
                }),
                ...(profile.social.linkedin && {
                    "article:author": `https://linkedin.com/in/${profile.social.linkedin}`
                })
            }
        };
    } catch (error) {
        return {
            title: "About",
            description: "Learn more about me",
            image: `${baseUrl}/about-og.jpg`,
            url: `${baseUrl}/about`
        };
    }
}

/**
 * Generate metadata for portfolio page
 */
export async function getPortfolioMeta(
    _params: any,
    env: any,
    baseUrl: string
): Promise<any> {
    try {
        const apiUrl = env.API_URL || `https://${env.ORIGIN_HOST}`;
        const response = await fetch(`${apiUrl}/api/projects?featured=true&limit=3`, {
            cf: { resolveOverride: env.ORIGIN_HOST }
        });
        
        if (!response.ok) throw new Error("Projects not found");
        
        const projects = await response.json() as Project[];
        const projectNames = projects.map(p => p.title).join(', ');
        
        return {
            title: "Portfolio - Featured Work",
            description: `Explore my portfolio featuring projects like ${projectNames} and more`,
            image: projects[0]?.thumbnail || `${baseUrl}/portfolio-og.jpg`,
            url: `${baseUrl}/portfolio`,
            type: "website"
        };
    } catch (error) {
        return {
            title: "Portfolio",
            description: "View my latest work and projects",
            image: `${baseUrl}/portfolio-og.jpg`,
            url: `${baseUrl}/portfolio`
        };
    }
}

/**
 * Generate metadata for a project page
 */
export async function getProjectMeta(
    params: { slug: string },
    env: any,
    baseUrl: string
): Promise<any> {
    try {
        const apiUrl = env.API_URL || `https://${env.ORIGIN_HOST}`;
        const response = await fetch(`${apiUrl}/api/projects/${params.slug}`, {
            cf: { resolveOverride: env.ORIGIN_HOST }
        });
        
        if (!response.ok) throw new Error("Project not found");
        
        const project = await response.json() as Project;
        
        const techStack = project.technologies.slice(0, 3).join(', ');
        const clientInfo = project.client 
            ? ` for ${project.client.name}` 
            : '';
        
        return {
            title: `${project.title}${clientInfo}`,
            description: project.subtitle || 
                `${project.description.substring(0, 100)}... Built with ${techStack}`,
            image: project.thumbnail || project.images[0] || `${baseUrl}/project-og.jpg`,
            url: `${baseUrl}/project/${project.slug}`,
            type: "article",
            
            extra: {
                "article:author": env.AUTHOR_NAME || "Portfolio Owner",
                "article:published_time": project.timeline.startDate,
                "article:tag": project.technologies.join(", "),
                "twitter:label1": "Role",
                "twitter:data1": project.role,
                "twitter:label2": "Technologies",
                "twitter:data2": techStack,
                ...(project.links.live && {
                    "og:see_also": project.links.live
                })
            }
        };
    } catch (error) {
        return {
            title: "Project",
            description: "View project details",
            image: `${baseUrl}/project-og.jpg`,
            url: `${baseUrl}/project/${params.slug}`
        };
    }
}

/**
 * Generate metadata for a case study
 */
export async function getCaseStudyMeta(
    params: { slug: string },
    env: any,
    baseUrl: string
): Promise<any> {
    // Similar to project but with different framing
    const projectMeta = await getProjectMeta(params, env, baseUrl);
    
    return {
        ...projectMeta,
        title: `Case Study: ${projectMeta.title}`,
        type: "article",
        extra: {
            ...projectMeta.extra,
            "og:type": "article",
            "article:section": "Case Studies"
        }
    };
}

/**
 * Generate metadata for a service page
 */
export async function getServiceMeta(
    params: { slug: string },
    env: any,
    baseUrl: string
): Promise<any> {
    try {
        const apiUrl = env.API_URL || `https://${env.ORIGIN_HOST}`;
        const response = await fetch(`${apiUrl}/api/services/${params.slug}`, {
            cf: { resolveOverride: env.ORIGIN_HOST }
        });
        
        if (!response.ok) throw new Error("Service not found");
        
        const service = await response.json() as Service;
        
        const pricingInfo = service.pricing 
            ? ` - ${formatPricing(service.pricing)}` 
            : '';
        
        return {
            title: `${service.name}${pricingInfo}`,
            description: service.description,
            image: service.image || `${baseUrl}/service-og.jpg`,
            url: `${baseUrl}/service/${service.slug}`,
            type: "website",
            
            extra: {
                "og:type": "business.business",
                ...(service.pricing && {
                    "twitter:label1": "Pricing",
                    "twitter:data1": formatPricing(service.pricing),
                    "twitter:label2": "Timeline",
                    "twitter:data2": service.timeline || "Custom"
                })
            }
        };
    } catch (error) {
        return {
            title: "Service",
            description: "Learn about our services",
            image: `${baseUrl}/service-og.jpg`,
            url: `${baseUrl}/service/${params.slug}`
        };
    }
}

/**
 * Generate metadata for services overview page
 */
export async function getServicesMeta(
    _params: any,
    env: any,
    baseUrl: string
): Promise<any> {
    return {
        title: "Services - What I Offer",
        description: "Explore my professional services including web development, design, consulting, and more",
        image: `${baseUrl}/services-og.jpg`,
        url: `${baseUrl}/services`,
        type: "website"
    };
}

/**
 * Generate metadata for blog post
 */
export async function getBlogPostMeta(
    params: { slug: string },
    env: any,
    baseUrl: string
): Promise<any> {
    try {
        const apiUrl = env.API_URL || `https://${env.ORIGIN_HOST}`;
        const response = await fetch(`${apiUrl}/api/blog/${params.slug}`, {
            cf: { resolveOverride: env.ORIGIN_HOST }
        });
        
        if (!response.ok) throw new Error("Post not found");
        
        const post = await response.json() as BlogPost;
        
        return {
            title: post.title,
            description: post.excerpt,
            image: post.featuredImage || `${baseUrl}/blog-og.jpg`,
            url: `${baseUrl}/blog/${post.slug}`,
            type: "article",
            
            extra: {
                "article:published_time": post.publishedAt,
                "article:tag": post.tags.join(", "),
                "twitter:label1": "Reading time",
                "twitter:data1": `${post.readingTime} min read`
            }
        };
    } catch (error) {
        return {
            title: "Blog Post",
            description: "Read the latest blog post",
            image: `${baseUrl}/blog-og.jpg`,
            url: `${baseUrl}/blog/${params.slug}`
        };
    }
}

/**
 * Generate metadata for contact page
 */
export async function getContactMeta(
    _params: any,
    env: any,
    baseUrl: string
): Promise<any> {
    try {
        const apiUrl = env.API_URL || `https://${env.ORIGIN_HOST}`;
        const response = await fetch(`${apiUrl}/api/profile`, {
            cf: { resolveOverride: env.ORIGIN_HOST }
        });
        
        const profile = await response.json() as Profile;
        
        const availabilityMsg = profile.availability?.status === 'available'
            ? "Available for new projects!"
            : profile.availability?.message || "Get in touch";
        
        return {
            title: `Contact ${profile.name}`,
            description: `${availabilityMsg} Let's discuss your project.`,
            image: profile.avatar || `${baseUrl}/contact-og.jpg`,
            url: `${baseUrl}/contact`,
            type: "website"
        };
    } catch (error) {
        return {
            title: "Contact",
            description: "Get in touch for collaborations and projects",
            image: `${baseUrl}/contact-og.jpg`,
            url: `${baseUrl}/contact`
        };
    }
}

/**
 * Generate metadata for resume/CV page
 */
export async function getResumeMeta(
    _params: any,
    env: any,
    baseUrl: string
): Promise<any> {
    try {
        const apiUrl = env.API_URL || `https://${env.ORIGIN_HOST}`;
        const response = await fetch(`${apiUrl}/api/profile`, {
            cf: { resolveOverride: env.ORIGIN_HOST }
        });
        
        const profile = await response.json() as Profile;
        
        const currentRole = profile.experience.find(e => e.current);
        const roleInfo = currentRole 
            ? `${currentRole.role} at ${currentRole.company}` 
            : profile.title;
        
        return {
            title: `${profile.name} - Resume`,
            description: `${roleInfo}. View my professional experience, skills, and qualifications.`,
            image: profile.avatar || `${baseUrl}/resume-og.jpg`,
            url: `${baseUrl}/resume`,
            type: "profile"
        };
    } catch (error) {
        return {
            title: "Resume",
            description: "View my professional experience and qualifications",
            image: `${baseUrl}/resume-og.jpg`,
            url: `${baseUrl}/resume`
        };
    }
}

// ============================================================================
// Helper Functions
// ============================================================================

/**
 * Format pricing information for display
 */
function formatPricing(pricing: Service['pricing']): string {
    if (!pricing) return 'Contact for pricing';
    
    switch (pricing.type) {
        case 'fixed':
            return pricing.amount 
                ? `$${pricing.amount} ${pricing.currency || 'USD'}` 
                : 'Fixed price';
        case 'hourly':
            return pricing.amount 
                ? `$${pricing.amount}/hour` 
                : 'Hourly rate';
        case 'project':
            return 'Project-based pricing';
        case 'custom':
        default:
            return pricing.note || 'Custom pricing';
    }
}

/**
 * Generate a call-to-action based on availability
 */
export function generateCTA(availability?: Profile['availability']): string {
    if (!availability) return "Let's work together";
    
    switch (availability.status) {
        case 'available':
            return "Available for new projects - Let's talk!";
        case 'busy':
            return "Limited availability - Book early";
        case 'not-available':
            return availability.message || "Currently unavailable";
        default:
            return "Get in touch";
    }
}