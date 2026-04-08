# Evidence excerpts

## API monolith (Server deps)

From `scenarios/landing-page-business-suite/api/main.go`:

```
33  type Server struct {
35    config               *Config
36    db                   *sql.DB
37    router               *mux.Router
38    variantSpace         *VariantSpace
39    configStore          *ConfigStore
40    metricsService       *MetricsService
41    stripeService        *StripeService
42    planService          *PlanService
43    downloadService      *DownloadService
44    downloadHosting      *DownloadHostingService
45    downloadAuthorizer   *DownloadAuthorizer
46    accountService       *AccountService
47    landingConfigService *LandingConfigService
48    paymentSettings      *PaymentSettingsService
49    assetsService        *AssetsService
50    seoService           *SEOService
51    feedbackService      *FeedbackService
52    emailService         *EmailService
53    waitlistService      *WaitlistService
54    apiKeyService *APIKeyService
55    limitsService *LimitsService
56    usageService  *UsageService
59    remoteProfileService *RemoteProfileService
61    userAuthService  *UserAuthService
62    magicLinkLimiter *RateLimiter
64    aiGatewayService *AIGatewayService
65    aiGatewayDeps    *AIGatewayDeps
67    sessionManager SessionManager
68  }
```

## Route registration breadth

From `scenarios/landing-page-business-suite/api/routes.go`:

```
9   func (s *Server) setupRoutes() {
10    s.router.Use(loggingMiddleware)
12    registerHealthRoutes(s)
13    registerLandingRoutes(s)
14    registerAuthRoutes(s)
15    registerAccountRoutes(s)
16    registerBillingRoutes(s)
17    registerAdminCoreRoutes(s)
18    registerRemoteProfileRoutes(s)
19    registerCommerceAdminRoutes(s)
20    registerVariantRoutes(s)
21    registerContentRoutes(s)
22    registerMetricsRoutes(s)
23    registerFeedbackRoutes(s)
24    registerWaitlistRoutes(s)
25    registerCreditsRoutes(s)
26    registerAIRoutes(s)
27    registerDocsRoutes(s)
28    registerAdminUserRoutes(s)
29    registerUpdateRoutes(s)
30  }
```

## Architecture doc UI tree mismatch

From `scenarios/landing-page-business-suite/docs/concepts/ARCHITECTURE.md`:

```
90  ui/src/
91  ├── app/
92  │   ├── App.tsx              # Root component, routing
93  │   └── providers/           # Context providers
94  │       ├── VariantProvider  # A/B test variant context
95  │       └── AuthProvider     # Admin authentication
96  ├── surfaces/
97  │   ├── public-landing/
98  │   │   ├── routes/
99  │   │   ├── sections/
100 │   │   └── components/
101 │   └── admin-portal/
102 │       ├── routes/
103 │       ├── components/
104 │       └── controllers/
105 └── shared/
106     ├── api/
111     ├── hooks/
112     ├── lib/
115     └── components/
```

## Backend initialization path mismatch

From `scenarios/landing-page-business-suite/docs/concepts/ARCHITECTURE.md`:

```
138 └── initialization/
139     └── postgres/
140         ├── schema.sql
141         └── seed.sql
```

Actual path is `scenarios/landing-page-business-suite/initialization/postgres/`.

## UI entrypoint reality

From `scenarios/landing-page-business-suite/ui/src/App.tsx`:

```
1  // DOC: docs/concepts/ARCHITECTURE.md - UI architecture and component structure
6  import { AdminAuthProvider } from './app/providers/AdminAuthProvider';
8  import { LandingVariantProvider } from './app/providers/LandingVariantProvider';
10 import { ErrorBoundary } from './shared/ui/ErrorBoundary';
11 import { ToastProvider } from './shared/ui/Toast';
43 import { UserLogin, VerifyMagicLink } from './surfaces/user-auth';
```
