package delivery

import (
	"context"
	"errors"
	"fmt"
	"strings"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

// CredentialState is the canonical presence vocabulary shared by the admin API
// and every consumer that gates on delivery credentials. A missing value, a
// store that cannot be reached now, and an outright authority fault are
// distinct outcomes: collapsing them is how a broken host gets reported as "not
// configured" and silently falls back to anonymous access.
type CredentialState string

const (
	CredentialConfigured     CredentialState = "configured"
	CredentialMissing        CredentialState = "missing"
	CredentialUnavailable    CredentialState = "unavailable"
	CredentialAuthorityError CredentialState = "authority_error"
)

// CredentialSource names the single authority that owns delivery secrets.
const CredentialSourceAuthority = "authority"

// CredentialPresence reports, without retrieving or exposing values, whether
// each delivery credential field is configured. Secret contents never appear
// here; only presence flags and states do.
type CredentialPresence struct {
	AccessKeyID          CredentialState `json:"access_key_id"`
	SecretAccessKey      CredentialState `json:"secret_access_key"`
	SessionToken         CredentialState `json:"session_token"`
	AccessKeyIDSet       bool            `json:"access_key_id_set"`
	SecretAccessKeySet   bool            `json:"secret_access_key_set"`
	SessionTokenSet      bool            `json:"session_token_set"`
	CredentialsSource    string          `json:"credentials_source"`
	SessionTokenOptional bool            `json:"session_token_optional"`
	Detail               string          `json:"detail,omitempty"`
}

// RequiredConfigured reports whether every required delivery credential has a
// value. The session token is intentionally excluded because it applies only
// to temporary STS credentials.
func (p CredentialPresence) RequiredConfigured() bool {
	return p.AccessKeyIDSet && p.SecretAccessKeySet
}

func (p CredentialPresence) source() string {
	if p.CredentialsSource != "" {
		return p.CredentialsSource
	}
	return CredentialSourceAuthority
}

// CredentialPresence inspects the authority for each delivery field. A provider
// fault is reported as unavailable or authority_error rather than missing so a
// caller never treats an outage as an absent credential.
func (s *Service) CredentialPresence(ctx context.Context) CredentialPresence {
	presence := CredentialPresence{
		CredentialsSource:    CredentialSourceAuthority,
		SessionTokenOptional: true,
	}
	accessState, accessDetail := s.credentialState(ctx, CredentialFieldAccessKeyID)
	secretState, secretDetail := s.credentialState(ctx, CredentialFieldSecretAccessKey)
	tokenState, tokenDetail := s.credentialState(ctx, CredentialFieldSessionToken)

	presence.AccessKeyID = accessState
	presence.SecretAccessKey = secretState
	presence.SessionToken = tokenState
	presence.AccessKeyIDSet = accessState == CredentialConfigured
	presence.SecretAccessKeySet = secretState == CredentialConfigured
	presence.SessionTokenSet = tokenState == CredentialConfigured
	presence.Detail = firstNonEmpty(accessDetail, secretDetail, tokenDetail)
	if hasAuthorityFault(accessState, secretState, tokenState) {
		presence.CredentialsSource = "unavailable"
	}
	return presence
}

func (s *Service) credentialState(ctx context.Context, field string) (CredentialState, string) {
	if s.credentials.Read == nil {
		return CredentialUnavailable, "credential authority boundary is not installed on this host"
	}
	value, err := s.credentials.Read(ctx, field)
	if err == nil {
		if strings.TrimSpace(value) != "" {
			return CredentialConfigured, ""
		}
		return CredentialMissing, ""
	}
	if errors.Is(err, credentialauthority.ErrUnconfigured) {
		return CredentialMissing, ""
	}
	if errors.Is(err, credentialauthority.ErrProviderUnavailable) || errors.Is(err, credentialauthority.ErrProviderAbsent) {
		return CredentialUnavailable, fmt.Sprintf("credential authority is unavailable while checking %s", field)
	}
	return CredentialAuthorityError, fmt.Sprintf("credential authority returned an error while checking %s", field)
}

func hasAuthorityFault(states ...CredentialState) bool {
	for _, state := range states {
		if state == CredentialUnavailable || state == CredentialAuthorityError {
			return true
		}
	}
	return false
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

// ProvisioningGuide is the complete operator guidance returned when delivery
// credentials are missing or unverified. It never contains secret values and is
// safe to serialize to the admin UI.
type ProvisioningGuide struct {
	Identity           string             `json:"identity"`
	Bucket             string             `json:"bucket"`
	Region             string             `json:"region"`
	PolicyName         string             `json:"policy_name"`
	PolicyDocument     string             `json:"policy_document"`
	IAMUsers           IAMUserGuidance    `json:"iam_users"`
	Steps              []ProvisioningStep `json:"steps"`
	Warnings           []string           `json:"warnings"`
	LocalCommands      []string           `json:"local_commands"`
	ProductionCommands []string           `json:"production_commands"`
	RotationSteps      []string           `json:"rotation_steps"`
	SessionTokenNote   string             `json:"session_token_note"`
	PrivateBucketNote  string             `json:"private_bucket_note"`
}

// IAMUserGuidance names the separate local and production identities so their
// credentials can be audited, revoked, and rotated independently.
type IAMUserGuidance struct {
	Local      string `json:"local"`
	Production string `json:"production"`
}

// ProvisioningStep is one ordered action in the AWS Console flow.
type ProvisioningStep struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
}

// DeliveryIdentity is the credential-authority identity LPBS resolves delivery
// credentials under.
const DeliveryIdentity = "vrooli/landing-page-business-suite"

// DefaultPolicyName is the required narrow IAM policy.
const DefaultPolicyName = "VrooliDeliveryBucketAccess"

// BuildProvisioningGuide assembles the full AWS and Vrooli instructions for a
// configured bucket. The bucket name is substituted into both policy ARNs so
// the pasted document matches the deployment. A prefix restriction is not
// applied because LPBS's per-app object-key layout is not proven stable enough
// to narrow the policy safely.
func BuildProvisioningGuide(bucket, region string) ProvisioningGuide {
	bucket = strings.TrimSpace(bucket)
	if bucket == "" {
		bucket = DefaultBucketFor("business_suite")
	}
	region = strings.TrimSpace(region)
	if region == "" {
		region = DefaultRegion
	}
	policy := fmt.Sprintf(`{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "InspectVrooliDeliveryBucket",
      "Effect": "Allow",
      "Action": [
        "s3:GetBucketLocation",
        "s3:ListBucket",
        "s3:ListBucketMultipartUploads"
      ],
      "Resource": "arn:aws:s3:::%[1]s"
    },
    {
      "Sid": "ManageVrooliDeliveryObjects",
      "Effect": "Allow",
      "Action": [
        "s3:GetObject",
        "s3:PutObject",
        "s3:DeleteObject",
        "s3:AbortMultipartUpload",
        "s3:ListMultipartUploadParts"
      ],
      "Resource": "arn:aws:s3:::%[1]s/*"
    }
  ]
}`, bucket)

	return ProvisioningGuide{
		Identity:       DeliveryIdentity,
		Bucket:         bucket,
		Region:         region,
		PolicyName:     DefaultPolicyName,
		PolicyDocument: policy,
		IAMUsers: IAMUserGuidance{
			Local:      "vrooli-lpbs-local",
			Production: "vrooli-lpbs-prod",
		},
		Steps: []ProvisioningStep{
			{Number: 1, Title: "Sign in to AWS", Detail: "Personal account owners normally select Root user and enter the account email. An email address is not an AWS account ID or IAM alias. Root access may administer IAM, but never create a root access key."},
			{Number: 2, Title: "Create the policy", Detail: fmt.Sprintf("Open IAM -> Policies and create %s using the policy document below. Both ARNs must name the configured bucket %q.", DefaultPolicyName, bucket)},
			{Number: 3, Title: "Create the IAM user", Detail: "Open IAM -> Users -> Create user and create either vrooli-lpbs-local or vrooli-lpbs-prod."},
			{Number: 4, Title: "Disable console access", Detail: "Do not enable AWS Console access for these workload identities; they need programmatic access only."},
			{Number: 5, Title: "Attach the policy", Detail: fmt.Sprintf("Attach %s directly. Do not attach AdministratorAccess or AmazonS3FullAccess.", DefaultPolicyName)},
			{Number: 6, Title: "Open Security credentials", Detail: "Open the user's Security credentials tab."},
			{Number: 7, Title: "Create the access key", Detail: "Under Access keys, select Create access key, then choose the application-running-outside-AWS use case."},
			{Number: 8, Title: "Tag the key", Detail: "Enter the matching description: \"Vrooli LPBS local deployment credential\" or \"Vrooli LPBS production deployment credential\"."},
			{Number: 9, Title: "Save the key once", Detail: "Create the key and copy the access key ID and secret access key into a secure location. AWS shows the secret only once."},
		},
		Warnings: []string{
			"The secret access key is displayed only once; if it is lost, create a replacement and deactivate/delete the old key.",
			"Never paste either credential into chat, logs, screenshots, or issue reports.",
			"Never commit credentials to Git or place them in service.json, application configuration, source files, or shell history.",
			"Never create or use an access key for the AWS root user.",
			"Do not attach AdministratorAccess or AmazonS3FullAccess; attach only VrooliDeliveryBucketAccess.",
			"AWS allows at most two access keys per IAM user.",
			"Use separate credential pairs for local and production; they should not be identical.",
		},
		LocalCommands: []string{
			fmt.Sprintf("vrooli credentials provision \\\n  --identity %s \\\n  --field delivery-s3-access-key-id", DeliveryIdentity),
			fmt.Sprintf("vrooli credentials provision \\\n  --identity %s \\\n  --field delivery-s3-secret-access-key", DeliveryIdentity),
			"vrooli credentials doctor --format json",
		},
		ProductionCommands: []string{
			fmt.Sprintf("# Run on the production host with the vrooli-lpbs-prod pair\nvrooli credentials provision \\\n  --identity %s \\\n  --field delivery-s3-access-key-id", DeliveryIdentity),
			fmt.Sprintf("vrooli credentials provision \\\n  --identity %s \\\n  --field delivery-s3-secret-access-key", DeliveryIdentity),
			"vrooli credentials doctor --format json",
		},
		RotationSteps: []string{
			"1. Create a second access key for the same IAM user.",
			"2. Provision the new pair into Vrooli with `vrooli credentials provision`.",
			"3. Restart or reload the affected service if necessary.",
			"4. Run presence and operational validation (`vrooli credentials doctor --format json` and Test storage access).",
			"5. Complete a real presign/upload/download check.",
			"6. Deactivate the old access key in IAM.",
			"7. Verify deployment still works.",
			"8. Delete the old key after a short observation period.",
		},
		SessionTokenNote:  "delivery-s3-session-token is optional. Provision it only when deliberately using temporary AWS STS credentials; leave it unset for ordinary long-lived IAM-user credentials.",
		PrivateBucketNote: "The delivery bucket stays private. Free and paid downloads both use short-lived presigned URLs issued by LPBS, so a free download never requires a public S3 object or AWS credentials on the client.",
	}
}

// MissingCredentialsDiagnostic builds the diagnostic returned when the required
// delivery credentials are absent. The guidance field is filled by the caller
// from BuildProvisioningGuide.
func MissingCredentialsDiagnostic(presence CredentialPresence, bucket, region string) *DiagnosticError {
	summary := "delivery storage credentials are not configured"
	code := CodeAccessKeyIDMissing
	switch {
	case presence.AccessKeyID != CredentialConfigured && presence.SecretAccessKey != CredentialConfigured:
		code = CodeAccessKeyIDMissing
		summary = "delivery-s3-access-key-id and delivery-s3-secret-access-key are not configured"
	case presence.AccessKeyID != CredentialConfigured:
		code = CodeAccessKeyIDMissing
		summary = "delivery-s3-access-key-id is not configured"
	case !presence.SecretAccessKeySet:
		code = CodeSecretAccessKeyMissing
		summary = "delivery-s3-secret-access-key is not configured"
	}
	diagnostic := newDiagnosticError(code, summary, "ResolveCredentials", bucket, region,
		"Provision both delivery-s3-access-key-id and delivery-s3-secret-access-key through the credential authority, then retest.", false)
	return diagnostic
}
