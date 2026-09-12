package main

import (
	feedbackhttp "landing-page-business-suite-api/handlers/feedback"
	"landing-page-business-suite-api/internal/experimentation"
	"landing-page-business-suite-api/internal/logx"
	domainmetrics "landing-page-business-suite-api/internal/metrics"
)

// feedbackEmailNotifier is the production notification adapter. Feedback is
// already durable when this asynchronous best-effort email work begins.
type feedbackEmailNotifier struct {
	configStore  *experimentation.ConfigStore
	emailService *EmailService
}

func (n feedbackEmailNotifier) Notify(feedback *domainmetrics.FeedbackRequest) {
	go func() {
		branding := n.configStore.GetBranding()
		if branding == nil {
			logx.Info("feedback_email_skipped", map[string]interface{}{"reason": "branding is not configured"})
			return
		}
		if err := n.emailService.SendFeedbackNotification(branding, feedback); err != nil {
			logx.Error("feedback_email_send_failed", map[string]interface{}{"error": err.Error()})
		}
	}()
}

var _ feedbackhttp.Notifier = feedbackEmailNotifier{}
