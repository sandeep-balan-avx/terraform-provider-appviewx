package appviewx

import (
	"testing"

	"terraform-provider-appviewx/appviewx/constants"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestFrameCertificatePayload_AutoRenewalEnabled(t *testing.T) {
	d := schema.TestResourceDataRaw(t, ResourceCertificateServer().Schema, map[string]interface{}{
		constants.COMMON_NAME:             "dev.api.example.com",
		constants.IS_AUTO_RENEWAL:         true,
		constants.RENEW_BEFORE:            "30",
		constants.AUTO_REGENERATE_ENABLED: false,
	})

	payload := frameCertificatePayload(d)

	if payload.CaConnectorInfo.IsAutoRenewal != "true" {
		t.Fatalf("expected isAutoRenewal \"true\", got %q", payload.CaConnectorInfo.IsAutoRenewal)
	}
	if payload.CaConnectorInfo.RenewBefore != "30" {
		t.Fatalf("expected renewBefore \"30\", got %q", payload.CaConnectorInfo.RenewBefore)
	}
	if payload.CaConnectorInfo.AutoRegenerateEnabled == nil || *payload.CaConnectorInfo.AutoRegenerateEnabled != false {
		t.Fatalf("expected autoRegenerateEnabled pointer to false, got %v", payload.CaConnectorInfo.AutoRegenerateEnabled)
	}
}

func TestFrameCertificatePayload_AutoRenewalDisabledByDefault(t *testing.T) {
	d := schema.TestResourceDataRaw(t, ResourceCertificateServer().Schema, map[string]interface{}{
		constants.COMMON_NAME: "dev.api.example.com",
	})

	payload := frameCertificatePayload(d)

	if payload.CaConnectorInfo.IsAutoRenewal != "" {
		t.Fatalf("expected empty isAutoRenewal when not enabled, got %q", payload.CaConnectorInfo.IsAutoRenewal)
	}
	if payload.CaConnectorInfo.AutoRegenerateEnabled != nil {
		t.Fatalf("expected nil autoRegenerateEnabled when not enabled, got %v", *payload.CaConnectorInfo.AutoRegenerateEnabled)
	}
}

func TestResourceCertificateSchema_ForceNew(t *testing.T) {
	s := ResourceCertificateServer().Schema

	// Attributes that shape the issued certificate (or the artifacts written at
	// create time) must force a re-create when changed — an issued cert cannot be
	// mutated in place.
	forceNew := []string{
		constants.COMMON_NAME,
		constants.HASH_FUNCTION,
		constants.KEY_TYPE,
		constants.BIT_LENGTH,
		constants.DNS_NAMES,
		constants.CUSTOM_FIELDS,
		constants.VENDOR_SPECIFIC_FIELDS,
		constants.CERTIFICATE_AUTHORITY,
		constants.CERTIFICATE_GROUP_NAME,
		constants.CA_SETTING_NAME,
		constants.CERTIFICATE_TYPE,
		constants.VALIDITY,
		constants.VALIDITY_UNIT,
		constants.VALIDITY_UNIT_VALUE,
		constants.IS_SYNC,
		constants.CERTIFICATE_DOWNLOAD_PATH,
		constants.CERTIFICATE_DOWNLOAD_FORMAT,
		constants.CERTIFICATE_DOWNLOAD_PASSWORD,
		constants.CERTIFICATE_CHAIN_REQUIRED,
		constants.KEY_DOWNLOAD_PATH,
		constants.KEY_DOWNLOAD_PASSWORD,
		constants.DOWNLOAD_PASSWORD_PROTECTED_KEY,
		constants.IS_AUTO_RENEWAL,
		constants.RENEW_BEFORE,
		constants.AUTO_REGENERATE_ENABLED,
		constants.WAIT_FOR_ISSUANCE,
		constants.ISSUANCE_TIMEOUT_SECONDS,
		constants.ISSUANCE_POLL_INTERVAL_SECONDS,
	}
	for _, name := range forceNew {
		field, ok := s[name]
		if !ok {
			t.Fatalf("schema is missing attribute %q", name)
		}
		if !field.ForceNew {
			t.Errorf("attribute %q must have ForceNew: true", name)
		}
	}

	// Destroy-time knobs take effect on Delete and must NOT re-issue the
	// certificate, so they are updatable in place.
	inPlace := []string{
		constants.REVOKE_ON_DESTROY,
		constants.REVOKE_REASON,
		constants.REVOKE_COMMENTS,
	}
	for _, name := range inPlace {
		field, ok := s[name]
		if !ok {
			t.Fatalf("schema is missing attribute %q", name)
		}
		if field.ForceNew {
			t.Errorf("attribute %q must be updatable in place (ForceNew: false)", name)
		}
	}
}

func TestBuildRevokePayload_WithComments(t *testing.T) {
	got := buildRevokePayload("6a47c392e9692037365e62e1", "Cessation of operation", "decommissioned")
	if got["resourceId"] != "6a47c392e9692037365e62e1" {
		t.Fatalf("unexpected resourceId: %v", got["resourceId"])
	}
	if got["reason"] != "Cessation of operation" {
		t.Fatalf("unexpected reason: %v", got["reason"])
	}
	if got["comments"] != "decommissioned" {
		t.Fatalf("expected comments to be set, got: %v", got["comments"])
	}
}

func TestBuildRevokePayload_WithoutComments(t *testing.T) {
	got := buildRevokePayload("abc", "Superseded", "")
	if _, ok := got["comments"]; ok {
		t.Fatalf("expected comments to be omitted when empty, got: %v", got["comments"])
	}
}
