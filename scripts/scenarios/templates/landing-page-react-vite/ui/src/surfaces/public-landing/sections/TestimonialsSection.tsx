import { Star, Quote } from 'lucide-react';

interface Testimonial {
  name: string;
  role: string;
  company: string;
  content: string;
  rating?: number;
  avatar_url?: string;
}

interface TestimonialsSectionProps {
  content: {
    title?: string;
    subtitle?: string;
    testimonials?: Testimonial[];
  };
}

export function TestimonialsSection({ content }: TestimonialsSectionProps) {
  const testimonials = content.testimonials || [
    {
      name: 'Sarah Johnson',
      role: 'Marketing Director',
      company: 'TechCorp',
      content: 'This platform transformed how we build landing pages. We launched 5 campaigns in the time it used to take us to launch 1.',
      rating: 5,
    },
    {
      name: 'Michael Chen',
      role: 'Founder',
      company: 'StartupXYZ',
      content: 'The A/B testing and analytics are game-changing. We increased our conversion rate by 40% in the first month.',
      rating: 5,
    },
    {
      name: 'Emma Davis',
      role: 'Product Manager',
      company: 'SaaSCo',
      content: 'Simple, powerful, and elegant. The agent customization feature saves us countless hours of development time.',
      rating: 5,
    },
  ];

  return (
    <section className="border-y border-white/5 bg-[#07090F] py-24">
      <div className="container mx-auto px-6">
        <div className="max-w-4xl text-left">
          <p className="text-xs uppercase tracking-[0.35em] text-slate-500">Operators on record</p>
          <h2 className="mt-3 text-4xl font-semibold text-white">{content.title || 'Proof from the factory floor'}</h2>
          <p className="mt-3 text-lg text-slate-300">
            {content.subtitle || 'Teams treat this template like a case-study-ready deliverable. Here is what they highlight in handoffs.'}
          </p>
        </div>
        <div className="mt-12 grid gap-8 md:grid-cols-2 lg:grid-cols-3">
          {testimonials.map((testimonial, index) => (
            <article
              key={index}
              className="relative rounded-[28px] border border-white/8 bg-[#0F172A] p-8 shadow-[0_25px_50px_rgba(0,0,0,0.35)]"
              data-testid={`testimonial-${index}`}
            >
              <Quote className="absolute right-6 top-6 h-8 w-8 text-[#F97316]/30" />
              {testimonial.rating && (
                <div className="mb-4 flex gap-1">
                  {Array.from({ length: testimonial.rating }).map((_, i) => (
                    <Star key={i} className="h-4 w-4 fill-[#FBBF24] text-[#FBBF24]" />
                  ))}
                </div>
              )}
              <p className="text-base leading-relaxed text-slate-200">“{testimonial.content}”</p>
              <div className="mt-6 flex items-center gap-4 border-t border-white/10 pt-4">
                {testimonial.avatar_url ? (
                  <img src={testimonial.avatar_url} alt={testimonial.name} className="h-12 w-12 rounded-full object-cover" />
                ) : (
                  <div className="flex h-12 w-12 items-center justify-center rounded-full bg-white/10 text-lg font-semibold text-white">
                    {testimonial.name.charAt(0)}
                  </div>
                )}
                <div>
                  <p className="font-semibold text-white">{testimonial.name}</p>
                  <p className="text-sm text-slate-400">
                    {testimonial.role} at {testimonial.company}
                  </p>
                </div>
              </div>
            </article>
          ))}
        </div>
      </div>
    </section>
  );
}
