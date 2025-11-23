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
    <section className="py-24 bg-slate-950/80 relative overflow-hidden">
      {/* Background decoration */}
      <div className="absolute top-0 right-0 w-96 h-96 bg-purple-500/5 rounded-full blur-3xl"></div>
      <div className="absolute bottom-0 left-0 w-96 h-96 bg-blue-500/5 rounded-full blur-3xl"></div>

      <div className="container relative z-10 mx-auto px-6">
        <div className="max-w-6xl mx-auto">
          {/* Header */}
          <div className="text-center mb-16 space-y-4">
            <h2 className="text-4xl lg:text-5xl font-bold bg-gradient-to-r from-white to-slate-300 bg-clip-text text-transparent">
              {content.title || 'Loved by Teams Everywhere'}
            </h2>
            <p className="text-xl text-slate-400 max-w-2xl mx-auto">
              {content.subtitle || 'See what our customers have to say about their experience.'}
            </p>
          </div>

          {/* Testimonials Grid */}
          <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-8">
            {testimonials.map((testimonial, index) => (
              <div
                key={index}
                className="group relative rounded-2xl border border-white/10 bg-gradient-to-br from-white/5 to-white/0 p-8 hover:border-purple-500/50 hover:bg-white/10 transition-all duration-300"
                data-testid={`testimonial-${index}`}
              >
                {/* Quote icon */}
                <Quote className="absolute top-6 right-6 w-8 h-8 text-purple-500/20 group-hover:text-purple-500/40 transition-colors" />

                <div className="relative space-y-4">
                  {/* Rating */}
                  {testimonial.rating && (
                    <div className="flex gap-1">
                      {Array.from({ length: testimonial.rating }).map((_, i) => (
                        <Star key={i} className="w-4 h-4 fill-yellow-400 text-yellow-400" />
                      ))}
                    </div>
                  )}

                  {/* Content */}
                  <p className="text-slate-300 leading-relaxed italic">
                    "{testimonial.content}"
                  </p>

                  {/* Author */}
                  <div className="flex items-center gap-4 pt-4 border-t border-white/10">
                    {testimonial.avatar_url ? (
                      <img
                        src={testimonial.avatar_url}
                        alt={testimonial.name}
                        className="w-12 h-12 rounded-full"
                      />
                    ) : (
                      <div className="w-12 h-12 rounded-full bg-gradient-to-br from-purple-500 to-blue-500 flex items-center justify-center text-white font-semibold">
                        {testimonial.name.charAt(0)}
                      </div>
                    )}
                    <div>
                      <div className="font-semibold text-white">{testimonial.name}</div>
                      <div className="text-sm text-slate-400">
                        {testimonial.role} at {testimonial.company}
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </section>
  );
}
