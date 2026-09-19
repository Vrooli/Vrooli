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
    <section className="bg-[#07090F] py-24">
      <div className="container mx-auto px-6">
        <div className="max-w-3xl">
          <p className="text-xs uppercase tracking-[0.35em] text-slate-500">FAQ</p>
          <h2 className="mt-3 text-4xl font-semibold text-white">
            {content.title || 'Guardrails teams ask about before launch'}
          </h2>
          <p className="mt-3 text-lg text-slate-300">
            {content.subtitle || 'From staging folders to download gating, here are the answers ops leaders reference most often.'}
          </p>
        </div>

        <div className="mt-10 space-y-4">
          {faqs.map((faq, index) => (
            <div
              key={index}
              className="rounded-2xl border border-white/10 bg-[#0F172A] transition-all duration-200"
              data-testid={`faq-${index}`}
            >
              <button
                onClick={() => setOpenIndex(openIndex === index ? null : index)}
                className="flex w-full items-center justify-between px-6 py-5 text-left"
                data-testid={`faq-question-${index}`}
              >
                <span className="pr-8 text-lg font-semibold text-white">{faq.question}</span>
                <ChevronDown
                  className={`h-5 w-5 text-[#F97316] transition-transform ${openIndex === index ? 'rotate-180' : ''}`}
                />
              </button>
              <div
                className={`overflow-hidden transition-all duration-300 ${
                  openIndex === index ? 'max-h-96' : 'max-h-0'
                }`}
                data-testid={`faq-answer-${index}`}
              >
                <div className="px-6 pb-6 text-sm text-slate-300">{faq.answer}</div>
              </div>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
