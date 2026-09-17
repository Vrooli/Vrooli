package main

import "context"

type securityNotifier struct{ email *EmailService }

func (n securityNotifier) Notify(ctx context.Context, to, event, detail string) error {
	if n.email == nil {
		return nil
	}
	return n.email.SendSecurityNotification(ctx, to, event, detail)
}
