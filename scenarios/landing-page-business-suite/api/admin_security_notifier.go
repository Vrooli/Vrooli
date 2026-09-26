package main

import "context"

import "fmt"

type securityNotifier struct{ email *EmailService }

func (n securityNotifier) Notify(ctx context.Context, to, event, detail string) error {
	if n.email == nil {
		return fmt.Errorf("security email service is unavailable")
	}
	return n.email.SendSecurityNotification(ctx, to, event, detail)
}
