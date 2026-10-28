package appviewx

import (
	"reflect"
	"testing"
)

func TestConstructGCPCertificateID(t *testing.T) {
	got := constructGCPCertificateID("my-project", "us-central1", "apigee-ingress-usc1-cert")
	want := "projects/my-project/locations/us-central1/certificates/apigee-ingress-usc1-cert"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestBuildGCPPushPayload_PushOnly(t *testing.T) {
	profiles := []string{"GCP.Single.All:my-project:Certificate Manager"}
	payload := buildGCPPushPayload(gcpPushOptions{
		CertificateID:     "6a47c392e9692037365e62e1",
		CertificateName:   "apigee-ingress-usc1-cert",
		Location:          "us-central1",
		ConnectorName:     "GCP connector",
		IsNewCertificate:  true,
		PushAutomatically: true,
		SelectedProfiles:  profiles,
	})

	if payload["certificateId"] != "6a47c392e9692037365e62e1" {
		t.Fatalf("unexpected certificateId: %v", payload["certificateId"])
	}
	if !reflect.DeepEqual(payload["selectedProfiles"], profiles) {
		t.Fatalf("unexpected selectedProfiles: %v", payload["selectedProfiles"])
	}

	gi := payload["generalInformation"].(map[string]interface{})
	if gi["vendor"] != "GCP" || gi["category"] != "cloud" {
		t.Fatalf("unexpected generalInformation: %v", gi)
	}
	if gi["name"] != "GCP connector" {
		t.Fatalf("unexpected connector name: %v", gi["name"])
	}
	if _, has := gi["profileType"]; has {
		t.Fatalf("generalInformation must not contain profileType (not read by backend)")
	}
	if gi["profileFilterSelection"] != ":Certificate Manager" {
		t.Fatalf("unexpected generalInformation.profileFilterSelection: %v", gi["profileFilterSelection"])
	}

	cd := payload["certificateDetails"].(map[string]interface{})
	if cd["certificateName"] != "apigee-ingress-usc1-cert" || cd["region"] != "us-central1" {
		t.Fatalf("unexpected certificateDetails: %v", cd)
	}
	if cd["isNewCertificate"] != true {
		t.Fatalf("expected isNewCertificate true, got: %v", cd["isNewCertificate"])
	}
	if cd["certificateLocation"] != "Certificate Manager" {
		t.Fatalf("unexpected certificateDetails.certificateLocation: %v", cd["certificateLocation"])
	}
	if _, has := cd["profileFilterSelection"]; has {
		t.Fatalf("certificateDetails must not contain profileFilterSelection (unused by backend)")
	}
	// No cert-map fields when CertMapName is empty.
	for _, f := range []string{"isNewCertificateMap", "certificateMapName", "isNewCertificateMapEntry", "certificateMapEntryName", "isPrimaryCertificateMapEntry"} {
		if _, has := cd[f]; has {
			t.Fatalf("push-only payload must not contain %s", f)
		}
	}

	pd := payload["pushDetails"].(map[string]interface{})
	if pd["scriptLocation"] != "appviewx" || pd["pushAutomatically"] != true {
		t.Fatalf("unexpected pushDetails: %v", pd)
	}

	if _, has := payload["pushToDevice"]; has {
		t.Fatalf("payload must not contain pushToDevice (not checked by save-application-connector)")
	}
}

func TestBuildGCPPushPayload_WithLBBinding(t *testing.T) {
	profiles := []string{"GCP.Single.All:my-project:Load Balancer"}
	payload := buildGCPPushPayload(gcpPushOptions{
		CertificateID:         "6a47c392e9692037365e62e1",
		CertificateName:       "apigee-ingress-usc1-cert",
		Location:              "us-central1",
		ConnectorName:         "GCP connector",
		IsNewCertificate:      true,
		PushAutomatically:     true,
		SelectedProfiles:      profiles,
		CertMapName:           "my-cert-map",
		IsNewCertMap:          true,
		CertMapEntryName:      "my-cert-map-entry",
		IsNewCertMapEntry:     true,
		IsPrimaryCertMapEntry: true,
	})

	cd := payload["certificateDetails"].(map[string]interface{})
	if cd["certificateMapName"] != "my-cert-map" {
		t.Fatalf("unexpected certificateMapName: %v", cd["certificateMapName"])
	}
	if cd["isNewCertificateMap"] != "true" {
		t.Fatalf("unexpected isNewCertificateMap: %v", cd["isNewCertificateMap"])
	}
	if cd["certificateMapEntryName"] != "my-cert-map-entry" {
		t.Fatalf("unexpected certificateMapEntryName: %v", cd["certificateMapEntryName"])
	}
	if cd["isNewCertificateMapEntry"] != "true" {
		t.Fatalf("unexpected isNewCertificateMapEntry: %v", cd["isNewCertificateMapEntry"])
	}
	if cd["isPrimaryCertificateMapEntry"] != "true" {
		t.Fatalf("unexpected isPrimaryCertificateMapEntry: %v", cd["isPrimaryCertificateMapEntry"])
	}
	// No hostname for a PRIMARY entry.
	if _, has := cd["hostName"]; has {
		t.Fatalf("hostName must be absent for a PRIMARY cert map entry")
	}
}

func TestBuildGCPPushPayload_WithHostname(t *testing.T) {
	profiles := []string{"GCP.Single.All:my-project:Load Balancer"}
	payload := buildGCPPushPayload(gcpPushOptions{
		CertificateID:         "6a47c392e9692037365e62e1",
		CertificateName:       "apigee-ingress-usc1-cert",
		Location:              "us-central1",
		ConnectorName:         "GCP connector",
		IsNewCertificate:      true,
		PushAutomatically:     true,
		SelectedProfiles:      profiles,
		CertMapName:           "my-cert-map",
		IsNewCertMap:          false,
		CertMapEntryName:      "api-entry",
		IsNewCertMapEntry:     true,
		IsPrimaryCertMapEntry: false,
		Hostname:              "api.example.com",
	})

	cd := payload["certificateDetails"].(map[string]interface{})
	if cd["hostName"] != "api.example.com" {
		t.Fatalf("unexpected hostName: %v", cd["hostName"])
	}
	if cd["isPrimaryCertificateMapEntry"] != "false" {
		t.Fatalf("unexpected isPrimaryCertificateMapEntry: %v", cd["isPrimaryCertificateMapEntry"])
	}
}

func TestExtractPushIDs_ObjectShape(t *testing.T) {
	response := []interface{}{
		map[string]interface{}{"requestId": "2710282", "connectorId": "1783753921336"},
	}
	requestID, connectorID := extractPushIDs(response)
	if requestID != "2710282" {
		t.Fatalf("unexpected requestID: %q", requestID)
	}
	if connectorID != "1783753921336" {
		t.Fatalf("unexpected connectorID: %q", connectorID)
	}
}

func TestExtractPushIDs_StringShape(t *testing.T) {
	response := []interface{}{"1783800974255"}
	requestID, connectorID := extractPushIDs(response)
	if requestID != "" {
		t.Fatalf("expected empty requestID, got %q", requestID)
	}
	if connectorID != "1783800974255" {
		t.Fatalf("unexpected connectorID: %q", connectorID)
	}
}

func TestExtractPushIDs_Empty(t *testing.T) {
	requestID, connectorID := extractPushIDs(nil)
	if requestID != "" || connectorID != "" {
		t.Fatalf("expected empty ids, got %q / %q", requestID, connectorID)
	}
}
