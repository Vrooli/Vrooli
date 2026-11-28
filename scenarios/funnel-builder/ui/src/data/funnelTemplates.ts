import { FunnelStep, FunnelSettings } from '../types'

export type TemplateIcon =
  | 'users'
  | 'rocket'
  | 'calendar'
  | 'zap'
  | 'gift'
  | 'shoppingCart'
  | 'fileText'
  | 'trendingUp'

export interface FunnelTemplateDefinition {
  id: string
  name: string
  description: string
  category: string
  color: string
  conversionRate: string
  icon: TemplateIcon
  funnel: {
    name: string
    description: string
    steps: FunnelStep[]
    settings: FunnelSettings
  }
}

const templateStep = (step: Omit<FunnelStep, 'position'>, position: number): FunnelStep => ({
  ...step,
  position,
})

export const funnelTemplates: FunnelTemplateDefinition[] = [
  {
    id: 'lead-gen',
    name: 'Lead Generation',
    description: 'Capture high-quality leads with qualifying questions',
    category: 'Marketing',
    color: 'bg-blue-500',
    conversionRate: '32%',
    icon: 'users',
    funnel: {
      name: 'Lead Generation Funnel',
      description: 'Guide prospects through qualification and capture contact details.',
      settings: {
        theme: { primaryColor: '#2563eb', fontFamily: 'Inter' },
        progressBar: true,
        exitIntent: true,
        tracking: { googleAnalytics: '', facebookPixel: '' },
      },
      steps: [
        templateStep(
          {
            id: 'lead-gen-step-1',
            type: 'quiz',
            title: 'Tell us about your goals',
            content: {
              question: 'What is your primary marketing goal?',
              options: [
                { id: 'goal-leads', text: 'Generate more leads', icon: '📈' },
                { id: 'goal-nurture', text: 'Nurture existing audience', icon: '💡' },
                { id: 'goal-automation', text: 'Automate follow-up', icon: '🤖' },
              ],
            },
          },
          0,
        ),
        templateStep(
          {
            id: 'lead-gen-step-2',
            type: 'content',
            title: 'See what top performers do',
            content: {
              headline: 'Top 3 campaigns driving 4x ROI',
              body: 'Successful teams personalize lead capture journeys and instantly route hot leads to sales.',
              media: {
                type: 'image',
                url: 'https://source.unsplash.com/random/800x600/?marketing',
                alt: 'Marketing strategy discussion',
              },
              buttonText: 'Continue',
            },
          },
          1,
        ),
        templateStep(
          {
            id: 'lead-gen-step-3',
            type: 'form',
            title: 'Share your contact details',
            content: {
              fields: [
                { id: 'name', type: 'text', label: 'Full Name', required: true },
                { id: 'email', type: 'email', label: 'Work Email', required: true },
                { id: 'company', type: 'text', label: 'Company', required: false },
              ],
              submitText: 'Get Custom Plan',
            },
          },
          2,
        ),
        templateStep(
          {
            id: 'lead-gen-step-4',
            type: 'quiz',
            title: 'How big is your team?',
            content: {
              question: 'Team size helps us tailor recommendations.',
              options: [
                { id: 'team-small', text: '1-5 people', icon: '👥' },
                { id: 'team-mid', text: '6-20 people', icon: '👨‍👩‍👧‍👦' },
                { id: 'team-enterprise', text: '21+ people', icon: '🏢' },
              ],
            },
          },
          3,
        ),
        templateStep(
          {
            id: 'lead-gen-step-5',
            type: 'cta',
            title: 'Schedule your strategy session',
            content: {
              headline: 'We hand off qualified leads in under 24 hours',
              subheadline: 'Pick a time with our growth team to see how the funnel performs.',
              buttonText: 'Schedule Call',
              buttonUrl: 'https://cal.com',
              urgency: 'Only 5 strategy slots remain this week',
            },
          },
          4,
        ),
      ],
    },
  },
  {
    id: 'product-launch',
    name: 'Product Launch',
    description: 'Build excitement and drive sales for new products',
    category: 'Sales',
    color: 'bg-purple-500',
    conversionRate: '28%',
    icon: 'rocket',
    funnel: {
      name: 'Product Launch Blueprint',
      description: 'Warm prospects and convert them during launch week.',
      settings: {
        theme: { primaryColor: '#7c3aed', fontFamily: 'Poppins' },
        progressBar: true,
        exitIntent: true,
        tracking: { googleAnalytics: '', facebookPixel: '' },
      },
      steps: [
        templateStep(
          {
            id: 'product-launch-1',
            type: 'content',
            title: 'Launch Announcement',
            content: {
              headline: 'Introducing the Velocity Suite',
              body: 'A modern toolkit that compresses your go-to-market from 6 months to 6 weeks.',
              media: {
                type: 'video',
                url: 'https://www.youtube.com/embed/dQw4w9WgXcQ',
                alt: 'Product teaser',
              },
              buttonText: 'See what is inside',
            },
          },
          0,
        ),
        templateStep(
          {
            id: 'product-launch-2',
            type: 'quiz',
            title: 'Where are you stuck today?',
            content: {
              question: 'Choose the biggest bottleneck you face right now.',
              options: [
                { id: 'bottleneck-awareness', text: 'Customers discovering us', icon: '🔍' },
                { id: 'bottleneck-engagement', text: 'Keeping leads engaged', icon: '💬' },
                { id: 'bottleneck-conversion', text: 'Closing deals faster', icon: '🏁' },
              ],
            },
          },
          1,
        ),
        templateStep(
          {
            id: 'product-launch-3',
            type: 'content',
            title: 'Personalized Playbook',
            content: {
              headline: 'How Velocity Suite fixes it',
              body: 'We tailor campaigns, nurture tracks, and launch-day offers for each bottleneck.',
              buttonText: 'Continue to Early Access',
            },
          },
          2,
        ),
        templateStep(
          {
            id: 'product-launch-4',
            type: 'form',
            title: 'Request Early Access',
            content: {
              fields: [
                { id: 'name', type: 'text', label: 'Full Name', required: true },
                { id: 'email', type: 'email', label: 'Work Email', required: true },
                { id: 'launch-date', type: 'text', label: 'Upcoming Launch Date', required: false },
              ],
              submitText: 'Reserve Early Access',
            },
          },
          3,
        ),
        templateStep(
          {
            id: 'product-launch-5',
            type: 'content',
            title: 'Testimonials',
            content: {
              headline: 'Teams that already accelerated launch velocity',
              body: '“Velocity Suite gave us a repeatable launch playbook.” — NovaOps',
              buttonText: 'Continue',
            },
          },
          4,
        ),
        templateStep(
          {
            id: 'product-launch-6',
            type: 'quiz',
            title: 'Pick your launch bonus',
            content: {
              question: 'Select the bonus that would help your launch the most.',
              options: [
                { id: 'bonus-coaching', text: 'Launch strategy coaching', icon: '🧠' },
                { id: 'bonus-ads', text: 'Done-for-you ad set', icon: '📣' },
                { id: 'bonus-copy', text: 'Conversion copy kit', icon: '📝' },
              ],
            },
          },
          5,
        ),
        templateStep(
          {
            id: 'product-launch-7',
            type: 'cta',
            title: 'Confirm your launch bonus',
            content: {
              headline: 'Launch Week: Claim your bonus now',
              subheadline: 'We only accept 25 companies into each launch cohort.',
              buttonText: 'Complete Reservation',
              buttonUrl: 'https://cal.com',
              urgency: 'Cohort resets in 3 days',
            },
          },
          6,
        ),
      ],
    },
  },
  {
    id: 'webinar-reg',
    name: 'Webinar Registration',
    description: 'Get attendees for your online events and webinars',
    category: 'Events',
    color: 'bg-green-500',
    conversionRate: '45%',
    icon: 'calendar',
    funnel: {
      name: 'Webinar Registration Flow',
      description: 'Boost attendance through reminders and value stacking.',
      settings: {
        theme: { primaryColor: '#10b981', fontFamily: 'Inter' },
        progressBar: true,
        exitIntent: false,
        tracking: { googleAnalytics: '', facebookPixel: '' },
      },
      steps: [
        templateStep(
          {
            id: 'webinar-step-1',
            type: 'content',
            title: 'Event Overview',
            content: {
              headline: 'Automation Playbooks Live Workshop',
              body: 'Join us to learn how to automate acquisition funnels using AI and workflows.',
              media: {
                type: 'image',
                url: 'https://source.unsplash.com/random/800x600/?webinar',
                alt: 'Webinar presentation',
              },
              buttonText: 'Reserve Seat',
            },
          },
          0,
        ),
        templateStep(
          {
            id: 'webinar-step-2',
            type: 'form',
            title: 'Reserve Your Seat',
            content: {
              fields: [
                { id: 'name', type: 'text', label: 'Full Name', required: true },
                { id: 'email', type: 'email', label: 'Email Address', required: true },
                { id: 'role', type: 'select', label: 'Role', options: ['Founder', 'Marketing Lead', 'Sales Leader', 'Operations'], required: true },
              ],
              submitText: 'Save My Seat',
            },
          },
          1,
        ),
        templateStep(
          {
            id: 'webinar-step-3',
            type: 'quiz',
            title: 'What do you want to learn?',
            content: {
              question: 'Pick the outcomes you care about most.',
              options: [
                { id: 'learn-funnels', text: 'Building funnels faster', icon: '⚙️' },
                { id: 'learn-ads', text: 'Automating paid acquisition', icon: '📊' },
                { id: 'learn-crm', text: 'Improving CRM follow-up', icon: '💬' },
              ],
              multiSelect: true,
            },
          },
          2,
        ),
        templateStep(
          {
            id: 'webinar-step-4',
            type: 'cta',
            title: 'Get calendar invite',
            content: {
              headline: 'You are almost confirmed! 📅',
              subheadline: 'Add the webinar to your calendar to get reminders.',
              buttonText: 'Add to Calendar',
              buttonUrl: 'https://calendar.google.com',
            },
          },
          3,
        ),
      ],
    },
  },
  {
    id: 'quiz-funnel',
    name: 'Interactive Quiz',
    description: 'Engage visitors with personalized quiz results',
    category: 'Engagement',
    color: 'bg-yellow-500',
    conversionRate: '38%',
    icon: 'zap',
    funnel: {
      name: 'Interactive Quiz Funnel',
      description: 'Deliver tailored outcomes based on quiz responses.',
      settings: {
        theme: { primaryColor: '#f59e0b', fontFamily: 'Rubik' },
        progressBar: true,
        exitIntent: false,
        tracking: { googleAnalytics: '', facebookPixel: '' },
      },
      steps: [
        templateStep(
          {
            id: 'quiz-funnel-1',
            type: 'quiz',
            title: 'Start the quiz',
            content: {
              question: 'How experienced are you with funnel building?',
              options: [
                { id: 'experience-new', text: 'Just getting started', icon: '🌱' },
                { id: 'experience-growing', text: 'Growing my funnels', icon: '🌿' },
                { id: 'experience-pro', text: 'Scaling advanced funnels', icon: '🌳' },
              ],
            },
          },
          0,
        ),
        templateStep(
          {
            id: 'quiz-funnel-2',
            type: 'quiz',
            title: 'Choose your channel',
            content: {
              question: 'Which channel drives most of your traffic?',
              options: [
                { id: 'channel-ads', text: 'Paid Ads', icon: '💸' },
                { id: 'channel-organic', text: 'Organic / SEO', icon: '🌐' },
                { id: 'channel-partners', text: 'Partnerships', icon: '🤝' },
              ],
            },
          },
          1,
        ),
        templateStep(
          {
            id: 'quiz-funnel-3',
            type: 'quiz',
            title: 'Pick your KPI',
            content: {
              question: 'What metric matters most right now?',
              options: [
                { id: 'kpi-cpl', text: 'Cost Per Lead', icon: '💰' },
                { id: 'kpi-conv', text: 'Conversion Rate', icon: '🎯' },
                { id: 'kpi-cltv', text: 'Customer Lifetime Value', icon: '♾️' },
              ],
            },
          },
          2,
        ),
        templateStep(
          {
            id: 'quiz-funnel-4',
            type: 'content',
            title: 'Your tailored playbook',
            content: {
              headline: 'Built for your quiz responses',
              body: 'See how to optimize the channel and KPI you selected with actionable steps.',
              buttonText: 'Unlock Recommendations',
            },
          },
          3,
        ),
        templateStep(
          {
            id: 'quiz-funnel-5',
            type: 'form',
            title: 'Get your personalized report',
            content: {
              fields: [
                { id: 'email', type: 'email', label: 'Work Email', required: true },
                { id: 'company', type: 'text', label: 'Company', required: false },
              ],
              submitText: 'Email My Plan',
            },
          },
          4,
        ),
        templateStep(
          {
            id: 'quiz-funnel-6',
            type: 'cta',
            title: 'Book a strategy session',
            content: {
              headline: 'Ready to implement your personalized plan?',
              buttonText: 'Talk to a Specialist',
              buttonUrl: 'https://cal.com',
            },
          },
          5,
        ),
      ],
    },
  },
  {
    id: 'free-trial',
    name: 'Free Trial Signup',
    description: 'Convert visitors into trial users for SaaS products',
    category: 'SaaS',
    color: 'bg-pink-500',
    conversionRate: '25%',
    icon: 'gift',
    funnel: {
      name: 'Free Trial Accelerator',
      description: 'Educate and convert ideal SaaS prospects.',
      settings: {
        theme: { primaryColor: '#ec4899', fontFamily: 'Inter' },
        progressBar: true,
        exitIntent: true,
        tracking: { googleAnalytics: '', facebookPixel: '' },
      },
      steps: [
        templateStep(
          {
            id: 'free-trial-1',
            type: 'content',
            title: 'Product Tour',
            content: {
              headline: 'See the platform in action',
              body: 'Watch a 2-minute overview to understand how we streamline your operations.',
              media: {
                type: 'video',
                url: 'https://www.youtube.com/embed/oHg5SJYRHA0',
                alt: 'Product walkthrough',
              },
              buttonText: 'Next',
            },
          },
          0,
        ),
        templateStep(
          {
            id: 'free-trial-2',
            type: 'quiz',
            title: 'Choose your priority',
            content: {
              question: 'What do you want the trial to solve first?',
              options: [
                { id: 'priority-automation', text: 'Automate manual tasks', icon: '⚙️' },
                { id: 'priority-reporting', text: 'Better reporting', icon: '📊' },
                { id: 'priority-collaboration', text: 'Team collaboration', icon: '🧑‍🤝‍🧑' },
              ],
            },
          },
          1,
        ),
        templateStep(
          {
            id: 'free-trial-3',
            type: 'content',
            title: 'Feature Match',
            content: {
              headline: 'Perfect for your priority',
              body: 'We pre-configure your workspace to highlight the features that best match your needs.',
              buttonText: 'Continue to Trial Signup',
            },
          },
          2,
        ),
        templateStep(
          {
            id: 'free-trial-4',
            type: 'form',
            title: 'Start Your Free Trial',
            content: {
              fields: [
                { id: 'name', type: 'text', label: 'Full Name', required: true },
                { id: 'email', type: 'email', label: 'Business Email', required: true },
                { id: 'team-size', type: 'select', label: 'Team Size', options: ['1-10', '11-50', '51-200', '201+'], required: true },
              ],
              submitText: 'Create Trial Account',
            },
          },
          3,
        ),
        templateStep(
          {
            id: 'free-trial-5',
            type: 'cta',
            title: 'Book Onboarding',
            content: {
              headline: 'Launch with expert guidance',
              subheadline: 'Reserve an onboarding call so we can tailor the first week of your trial.',
              buttonText: 'Book Onboarding',
              buttonUrl: 'https://cal.com',
            },
          },
          4,
        ),
      ],
    },
  },
  {
    id: 'ecommerce',
    name: 'E-commerce Checkout',
    description: 'Optimize your checkout process for higher sales',
    category: 'E-commerce',
    color: 'bg-indigo-500',
    conversionRate: '42%',
    icon: 'shoppingCart',
    funnel: {
      name: 'E-commerce Checkout Funnel',
      description: 'Reduce cart abandonment and recover more revenue.',
      settings: {
        theme: { primaryColor: '#4f46e5', fontFamily: 'Inter' },
        progressBar: true,
        exitIntent: true,
        tracking: { googleAnalytics: '', facebookPixel: '' },
      },
      steps: [
        templateStep(
          {
            id: 'ecommerce-1',
            type: 'content',
            title: 'Cart Summary',
            content: {
              headline: 'Here is what you are getting',
              body: 'Review your cart before checkout and see bonus items we have added for you.',
              buttonText: 'Proceed to Checkout',
            },
          },
          0,
        ),
        templateStep(
          {
            id: 'ecommerce-2',
            type: 'form',
            title: 'Shipping Details',
            content: {
              fields: [
                { id: 'full-name', type: 'text', label: 'Full Name', required: true },
                { id: 'address', type: 'text', label: 'Shipping Address', required: true },
                { id: 'city', type: 'text', label: 'City', required: true },
                { id: 'postal', type: 'text', label: 'Postal Code', required: true },
              ],
              submitText: 'Continue to Payment',
            },
          },
          1,
        ),
        templateStep(
          {
            id: 'ecommerce-3',
            type: 'form',
            title: 'Payment Method',
            content: {
              fields: [
                { id: 'card-number', type: 'text', label: 'Card Number', required: true },
                { id: 'expiry', type: 'text', label: 'Expiration Date', required: true },
                { id: 'cvc', type: 'text', label: 'CVC', required: true },
              ],
              submitText: 'Review Order',
            },
          },
          2,
        ),
        templateStep(
          {
            id: 'ecommerce-4',
            type: 'content',
            title: 'Order Review & Upsell',
            content: {
              headline: 'Add expedited shipping and save 10%',
              body: 'Customers who select expedited shipping see 2-day delivery. You can opt in below.',
              buttonText: 'Add & Complete Order',
            },
          },
          3,
        ),
      ],
    },
  },
  {
    id: 'survey',
    name: 'Customer Survey',
    description: 'Collect valuable feedback from your customers',
    category: 'Feedback',
    color: 'bg-red-500',
    conversionRate: '55%',
    icon: 'fileText',
    funnel: {
      name: 'Customer Feedback Loop',
      description: 'Capture qualitative insights and follow up automatically.',
      settings: {
        theme: { primaryColor: '#ef4444', fontFamily: 'Inter' },
        progressBar: true,
        exitIntent: false,
        tracking: { googleAnalytics: '', facebookPixel: '' },
      },
      steps: [
        templateStep(
          {
            id: 'survey-1',
            type: 'content',
            title: 'We value your feedback',
            content: {
              headline: 'Help us build better experiences',
              body: 'Answer a few quick questions so we can prioritize the improvements that matter most to you.',
              buttonText: 'Start Survey',
            },
          },
          0,
        ),
        templateStep(
          {
            id: 'survey-2',
            type: 'quiz',
            title: 'Overall satisfaction',
            content: {
              question: 'How satisfied are you with our product so far?',
              options: [
                { id: 'sat-very', text: 'Very satisfied', icon: '😍' },
                { id: 'sat-ok', text: 'Somewhat satisfied', icon: '🙂' },
                { id: 'sat-unhappy', text: 'Not satisfied', icon: '😞' },
              ],
            },
          },
          1,
        ),
        templateStep(
          {
            id: 'survey-3',
            type: 'form',
            title: 'Share more context',
            content: {
              fields: [
                { id: 'favorite', type: 'textarea', label: 'What do you love most?', required: false },
                { id: 'improve', type: 'textarea', label: 'What could we improve?', required: false },
              ],
              submitText: 'Next',
            },
          },
          2,
        ),
        templateStep(
          {
            id: 'survey-4',
            type: 'quiz',
            title: 'Would you recommend us?',
            content: {
              question: 'How likely are you to recommend us to a friend or colleague?',
              options: [
                { id: 'recommend-yes', text: 'Definitely', icon: '🚀' },
                { id: 'recommend-maybe', text: 'Maybe', icon: '🤔' },
                { id: 'recommend-no', text: 'Not right now', icon: '🙅' },
              ],
            },
          },
          3,
        ),
        templateStep(
          {
            id: 'survey-5',
            type: 'cta',
            title: 'Thank you!',
            content: {
              headline: 'We appreciate your input',
              subheadline: 'Want to shape our roadmap? Join the beta group.',
              buttonText: 'Join Beta Group',
              buttonUrl: 'https://example.com/beta',
            },
          },
          4,
        ),
      ],
    },
  },
  {
    id: 'roi-calculator',
    name: 'ROI Calculator',
    description: 'Help prospects calculate their potential returns',
    category: 'Tools',
    color: 'bg-orange-500',
    conversionRate: '35%',
    icon: 'trendingUp',
    funnel: {
      name: 'ROI Calculator Funnel',
      description: 'Quantify the impact of adopting your solution.',
      settings: {
        theme: { primaryColor: '#f97316', fontFamily: 'Inter' },
        progressBar: true,
        exitIntent: false,
        tracking: { googleAnalytics: '', facebookPixel: '' },
      },
      steps: [
        templateStep(
          {
            id: 'roi-1',
            type: 'content',
            title: 'Calculate your ROI',
            content: {
              headline: 'See the projected impact in < 60 seconds',
              body: 'Answer a few quick questions to forecast ROI using your own numbers.',
              buttonText: 'Start Calculation',
            },
          },
          0,
        ),
        templateStep(
          {
            id: 'roi-2',
            type: 'form',
            title: 'Baseline Metrics',
            content: {
              fields: [
                { id: 'current-leads', type: 'number', label: 'Monthly Leads', required: true },
                { id: 'conversion-rate', type: 'number', label: 'Current Conversion Rate (%)', required: true },
              ],
              submitText: 'Next',
            },
          },
          1,
        ),
        templateStep(
          {
            id: 'roi-3',
            type: 'quiz',
            title: 'Goal Alignment',
            content: {
              question: 'Pick the goal you want to achieve first.',
              options: [
                { id: 'roi-more-leads', text: 'Increase qualified leads', icon: '🧲' },
                { id: 'roi-higher-value', text: 'Raise average deal size', icon: '💼' },
                { id: 'roi-speed', text: 'Shorten sales cycle', icon: '⚡' },
              ],
            },
          },
          2,
        ),
        templateStep(
          {
            id: 'roi-4',
            type: 'content',
            title: 'Projected Results',
            content: {
              headline: 'Here is the ROI you can expect',
              body: 'Customers with similar goals see a 3.4x increase in revenue within the first quarter.',
              buttonText: 'Continue',
            },
          },
          3,
        ),
        templateStep(
          {
            id: 'roi-5',
            type: 'form',
            title: 'Send Detailed Report',
            content: {
              fields: [
                { id: 'email', type: 'email', label: 'Work Email', required: true },
                { id: 'phone', type: 'tel', label: 'Phone Number', required: false },
              ],
              submitText: 'Email Me the Report',
            },
          },
          4,
        ),
        templateStep(
          {
            id: 'roi-6',
            type: 'cta',
            title: 'Schedule ROI Review',
            content: {
              headline: 'Break down the ROI together',
              subheadline: 'Walk through the report with one of our specialists.',
              buttonText: 'Book ROI Call',
              buttonUrl: 'https://cal.com',
            },
          },
          5,
        ),
      ],
    },
  },
]

export const getFunnelTemplate = (id: string) =>
  funnelTemplates.find((template) => template.id === id)
