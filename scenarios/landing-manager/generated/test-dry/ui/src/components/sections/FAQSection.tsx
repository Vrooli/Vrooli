import { useState } from 'react';
import { ChevronDown } from 'lucide-react';

interface FAQItem {
  question: string;
  answer: string;
}

interface FAQSectionProps {
  content: {
    title?: string;
    subtitle?: string;
    faqs?: FAQItem[];
  };
}

export function FAQSection({ content }: FAQSectionProps) {
  const [openIndex, setOpenIndex] = useState<number | null>(0);

  const faqs = content.faqs || [
    {
      question: 'How quickly can I launch a landing page?',
      answer: 'You can have a fully functional landing page live in minutes. Our templates are pre-built and optimized, so you just need to customize the content and deploy.',
    },
    {
      question: 'Do I need coding skills?',
      answer: 'No coding required! Our visual editor and agent customization features allow you to create professional landing pages without writing any code.',
    },
    {
      question: 'Can I run A/B tests?',
      answer: 'Yes! A/B testing is built-in. You can create multiple variants, set traffic weights, and track performance metrics to optimize your conversion rates.',
    },
    {
      question: 'What kind of analytics do you provide?',
      answer: 'We provide comprehensive analytics including visitor counts, conversion rates, click-through rates, scroll depth, and variant performance comparisons.',
    },
    {
      question: 'How does pricing work?',
      answer: 'We offer flexible pricing tiers based on your needs. Start with our free trial, then choose a plan that matches your scale. No hidden fees, cancel anytime.',
    },
  ];

  return (
    <section className="py-24 bg-slate-950 relative overflow-hidden">
      {/* Background decoration */}
      <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[800px] h-[800px] bg-purple-500/5 rounded-full blur-3xl"></div>

      <div className="container relative z-10 mx-auto px-6">
        <div className="max-w-4xl mx-auto">
          {/* Header */}
          <div className="text-center mb-16 space-y-4">
            <h2 className="text-4xl lg:text-5xl font-bold bg-gradient-to-r from-white to-slate-300 bg-clip-text text-transparent">
              {content.title || 'Frequently Asked Questions'}
            </h2>
            <p className="text-xl text-slate-400">
              {content.subtitle || 'Everything you need to know about our platform.'}
            </p>
          </div>

          {/* FAQ Accordion */}
          <div className="space-y-4">
            {faqs.map((faq, index) => (
              <div
                key={index}
                className="rounded-xl border border-white/10 bg-white/5 overflow-hidden transition-all duration-300 hover:border-purple-500/50"
                data-testid={`faq-${index}`}
              >
                <button
                  onClick={() => setOpenIndex(openIndex === index ? null : index)}
                  className="w-full flex items-center justify-between p-6 text-left hover:bg-white/5 transition-colors"
                  data-testid={`faq-question-${index}`}
                >
                  <span className="text-lg font-semibold text-white pr-8">
                    {faq.question}
                  </span>
                  <ChevronDown
                    className={`w-5 h-5 text-purple-400 flex-shrink-0 transition-transform duration-300 ${
                      openIndex === index ? 'rotate-180' : ''
                    }`}
                  />
                </button>
                <div
                  className={`overflow-hidden transition-all duration-300 ${
                    openIndex === index ? 'max-h-96' : 'max-h-0'
                  }`}
                  data-testid={`faq-answer-${index}`}
                >
                  <div className="px-6 pb-6 pt-2 text-slate-300 leading-relaxed">
                    {faq.answer}
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
