package appviewx

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"terraform-provider-appviewx/appviewx/config"
	"terraform-provider-appviewx/appviewx/constants"
	"terraform-provider-appviewx/appviewx/logger"
)

// constructGCPCertificateID builds the GCP Certificate Manager resource id in the
// same format as google_certificate_manager_certificate.id:
// projects/{project}/locations/{location}/certificates/{name}
func constructGCPCertificateID(project, location, name string) string {
	return fmt.Sprintf("projects/%s/locations/%s/certificates/%s", project, location, name)
}

// gcpPushOptions holds all inputs for buildGCPPushPayload.
type gcpPushOptions struct {
	CertificateID     string
	CertificateName   string
	Location          string
	ConnectorName     string
	IsNewCertificate  bool
	PushAutomatically bool
	SelectedProfiles  []string
	// Optional – Certificate Map fields for Load Balancer binding.
	// Set CertMapName to enable cert-map fields in the payload.
	CertMapName           string
	IsNewCertMap          bool
	CertMapEntryName      string
	IsNewCertMapEntry     bool
	IsPrimaryCertMapEntry bool
	Hostname              string // used when IsPrimaryCertMapEntry == false
}

// buildGCPPushPayload builds the /avxapi/certificate/pushToDevice request body.
// The transformer ApplicationConnectorSaveAndPushTransformer routes this to save-application-connector.
//
// certificateLocation is always "Certificate Manager" so CertGCPPushHelper routes
// Load Balancer profiles to the Certificate Manager push path
// (it reads fileProperties.certificateLocation, not profileFilterSelection).
//
// When opts.CertMapName is non-empty the cert-map and cert-map-entry fields are
// included so the backend can create/update the Certificate Map and bind it to a
// Load Balancer proxy.
func buildGCPPushPayload(opts gcpPushOptions) map[string]interface{} {
	certDetails := map[string]interface{}{
		"isNewCertificate":    opts.IsNewCertificate,
		"certificateName":     opts.CertificateName,
		"region":              opts.Location,
		"certificateLocation": "Certificate Manager",
	}

	// Include cert-map fields only when the caller supplies a certificate map name.
	// These are required by CertificateManagerHelper for the create/update-cert-map
	// and bind-to-proxy paths used by Load Balancer profiles.
	if opts.CertMapName != "" {
		certDetails["isNewCertificateMap"] = fmt.Sprintf("%v", opts.IsNewCertMap)
		certDetails["certificateMapName"] = opts.CertMapName
		certDetails["isNewCertificateMapEntry"] = fmt.Sprintf("%v", opts.IsNewCertMapEntry)
		certDetails["certificateMapEntryName"] = opts.CertMapEntryName
		certDetails["isPrimaryCertificateMapEntry"] = fmt.Sprintf("%v", opts.IsPrimaryCertMapEntry)
		if !opts.IsPrimaryCertMapEntry && opts.Hostname != "" {
			certDetails["hostName"] = opts.Hostname
		}
	}

	return map[string]interface{}{
		"generalInformation": map[string]interface{}{
			"category":               "cloud",
			"vendor":                 "GCP",
			"profileFilterSelection": ":Certificate Manager",
			"name":                   opts.ConnectorName,
		},
		"certificateDetails": certDetails,
		"pushDetails": map[string]interface{}{
			"scriptLocation":    "appviewx",
			"pushAutomatically": opts.PushAutomatically,
		},
		"certificateId":    opts.CertificateID,
		"selectedProfiles": opts.SelectedProfiles,
	}
}

// extractPushIDs pulls requestId/connectorId from the pushToDevice "response"
// field, which AppViewX returns either as [{"requestId":..,"connectorId":..}]
// or as a bare ["<connectorId>"] array.
func extractPushIDs(response interface{}) (requestID, connectorID string) {
	arr, ok := response.([]interface{})
	if !ok || len(arr) == 0 {
		return "", ""
	}
	switch first := arr[0].(type) {
	case map[string]interface{}:
		if v, ok := first["requestId"].(string); ok {
			requestID = v
		}
		if v, ok := first["connectorId"].(string); ok {
			connectorID = v
		}
	case string:
		connectorID = first
	}
	return requestID, connectorID
}

func ResourcePushGCPCertificateManager() *schema.Resource {
	return &schema.Resource{
		Create: resourcePushGCPCertificateManagerCreate,
		Read:   resourcePushGCPCertificateManagerRead,
		Update: resourcePushGCPCertificateManagerUpdate,
		Delete: resourcePushGCPCertificateManagerDelete,

		Schema: map[string]*schema.Schema{
			constants.CERTIFICATE_ID: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "AppViewX certificate resource id (resource_id from appviewx_create_certificate).",
			},
			constants.CERTIFICATE_NAME: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Name of the certificate in GCP Certificate Manager.",
			},
			constants.GCP_PROJECT: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "GCP project id.",
			},
			constants.GCP_LOCATION: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "GCP location: a region (e.g. us-central1) or 'global'.",
			},
			constants.GCP_CONNECTOR_NAME: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "AppViewX GCP connector name stored as the application-connector name.",
			},
			constants.SELECTED_PROFILES: {
				Type:        schema.TypeList,
				Required:    true,
				ForceNew:    true,
				MinItems:    1,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "AppViewX selected profile IDs, e.g. [\"GCP.Single.All:<project>:Certificate Manager\"]. Use a 'Load Balancer' type profile to also bind to an LB.",
			},
			constants.IS_NEW_CERTIFICATE: {
				Type:        schema.TypeBool,
				Optional:    true,
				ForceNew:    true,
				Default:     true,
				Description: "true to create the certificate in GCP CM; false to update an existing one.",
			},
			constants.PUSH_AUTOMATICALLY: {
				Type:        schema.TypeBool,
				Optional:    true,
				ForceNew:    true,
				Default:     true,
				Description: "Trigger the push immediately without waiting for manual approval.",
			},
			// --- Certificate Map / Load Balancer binding (optional) ---
			constants.GCP_CERT_MAP_NAME: {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "GCP Certificate Map name. Required when binding to a Load Balancer via a cert map.",
			},
			constants.GCP_IS_NEW_CERT_MAP: {
				Type:        schema.TypeBool,
				Optional:    true,
				ForceNew:    true,
				Default:     true,
				Description: "true to create a new Certificate Map; false to use an existing one.",
			},
			constants.GCP_CERT_MAP_ENTRY_NAME: {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "GCP Certificate Map Entry name.",
			},
			constants.GCP_IS_NEW_CERT_MAP_ENTRY: {
				Type:        schema.TypeBool,
				Optional:    true,
				ForceNew:    true,
				Default:     true,
				Description: "true to create a new Certificate Map Entry; false to update an existing one.",
			},
			constants.GCP_IS_PRIMARY_CERT_MAP_ENTRY: {
				Type:        schema.TypeBool,
				Optional:    true,
				ForceNew:    true,
				Default:     true,
				Description: "true for a PRIMARY matcher; false for a hostname matcher (requires hostname).",
			},
			constants.GCP_HOSTNAME: {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "Hostname for the Certificate Map Entry when is_primary_certificate_map_entry is false.",
			},
			// --- Polling ---
			constants.WAIT_FOR_COMPLETION: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Block until the GCP push workflow completes.",
			},
			constants.WAIT_TIMEOUT_SECONDS: {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      600,
				ValidateFunc: validation.IntAtLeast(1),
				Description:  "Maximum seconds to wait for push completion.",
			},
			constants.POLL_INTERVAL_SECONDS: {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      10,
				ValidateFunc: validation.IntAtLeast(1),
				Description:  "Seconds between status-poll attempts.",
			},
			// --- Computed outputs ---
			constants.CERT_MANAGER_CERTIFICATE_ID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "GCP Certificate Manager resource id: projects/{project}/locations/{location}/certificates/{name}.",
			},
			constants.REQUEST_ID: {
				Type:     schema.TypeString,
				Computed: true,
			},
			constants.CONNECTOR_ID: {
				Type:     schema.TypeString,
				Computed: true,
			},
			constants.STATUS_CODE: {
				Type:     schema.TypeInt,
				Computed: true,
			},
			constants.SUCCESS: {
				Type:     schema.TypeBool,
				Computed: true,
			},
		},
	}
}

func resourcePushGCPCertificateManagerRead(d *schema.ResourceData, m interface{}) error {
	logger.Info("**************** READ (no-op, preserving state) - GCP CERT MANAGER PUSH ****************")
	return nil
}

func resourcePushGCPCertificateManagerUpdate(d *schema.ResourceData, m interface{}) error {
	logger.Info("**************** UPDATE (no-op) - GCP CERT MANAGER PUSH ****************")
	return nil
}

func resourcePushGCPCertificateManagerDelete(d *schema.ResourceData, m interface{}) error {
	logger.Info("**************** DELETE (state only; GCP cert not deleted from GCP) - GCP CERT MANAGER PUSH ****************")
	d.SetId("")
	return nil
}

func resourcePushGCPCertificateManagerCreate(d *schema.ResourceData, m interface{}) error {
	logger.Info("**************** CREATE - GCP CERT MANAGER PUSH ****************")
	configAppViewXEnvironment := m.(*config.AppViewXEnvironment)

	appviewxEnvironmentIP := configAppViewXEnvironment.AppViewXEnvironmentIP
	appviewxEnvironmentPort := configAppViewXEnvironment.AppViewXEnvironmentPort
	appviewxEnvironmentIsHTTPS := configAppViewXEnvironment.AppViewXIsHTTPS
	appviewxGwSource := "WEB"

	// Authenticate (session id or access token).
	var appviewxSessionID, accessToken string
	var err error
	if configAppViewXEnvironment.AppViewXUserName != "" && configAppViewXEnvironment.AppViewXPassword != "" {
		appviewxSessionID, err = GetSession(configAppViewXEnvironment.AppViewXUserName, configAppViewXEnvironment.AppViewXPassword, appviewxEnvironmentIP, appviewxEnvironmentPort, appviewxGwSource, appviewxEnvironmentIsHTTPS)
		if err != nil {
			logger.Error("Error getting session: ", err)
		}
	}
	if appviewxSessionID == "" && configAppViewXEnvironment.AppViewXClientId != "" && configAppViewXEnvironment.AppViewXClientSecret != "" {
		accessToken, err = GetAccessTokenWithRotation(configAppViewXEnvironment, appviewxGwSource)
		if err != nil {
			logger.Error("Error getting access token: ", err)
			return err
		}
	}
	if appviewxSessionID == "" && accessToken == "" {
		return errors.New("authentication failed - provide either username/password or client ID/secret")
	}

	// Gather inputs.
	rawProfiles := d.Get(constants.SELECTED_PROFILES).([]interface{})
	selectedProfiles := make([]string, len(rawProfiles))
	for i, v := range rawProfiles {
		selectedProfiles[i] = v.(string)
	}

	opts := gcpPushOptions{
		CertificateID:     d.Get(constants.CERTIFICATE_ID).(string),
		CertificateName:   d.Get(constants.CERTIFICATE_NAME).(string),
		Location:          d.Get(constants.GCP_LOCATION).(string),
		ConnectorName:     d.Get(constants.GCP_CONNECTOR_NAME).(string),
		IsNewCertificate:  d.Get(constants.IS_NEW_CERTIFICATE).(bool),
		PushAutomatically: d.Get(constants.PUSH_AUTOMATICALLY).(bool),
		SelectedProfiles:  selectedProfiles,
		// Cert-map / LB binding fields (empty string = not used)
		CertMapName:           d.Get(constants.GCP_CERT_MAP_NAME).(string),
		IsNewCertMap:          d.Get(constants.GCP_IS_NEW_CERT_MAP).(bool),
		CertMapEntryName:      d.Get(constants.GCP_CERT_MAP_ENTRY_NAME).(string),
		IsNewCertMapEntry:     d.Get(constants.GCP_IS_NEW_CERT_MAP_ENTRY).(bool),
		IsPrimaryCertMapEntry: d.Get(constants.GCP_IS_PRIMARY_CERT_MAP_ENTRY).(bool),
		Hostname:              d.Get(constants.GCP_HOSTNAME).(string),
	}

	// Build and send the pushToDevice request.
	payload := buildGCPPushPayload(opts)
	requestBody, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	payloadBytes, _ := json.MarshalIndent(payload, "", "  ")
	logger.Debug("\nGCP push payload:\n%s\n", string(payloadBytes))

	queryParams := map[string]string{constants.GW_SOURCE: appviewxGwSource}
	url := GetURL(appviewxEnvironmentIP, appviewxEnvironmentPort, "certificate/pushToDevice", queryParams, appviewxEnvironmentIsHTTPS)

	client := &http.Client{Transport: HTTPTransport()}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(requestBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if appviewxSessionID != "" {
		req.Header.Set(constants.SESSION_ID, appviewxSessionID)
	} else {
		req.Header.Set(constants.TOKEN, accessToken)
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	d.Set(constants.STATUS_CODE, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	logger.Info("\nGCP push response:\n%s\n", string(body))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("gcp certificate push failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse the response. AppViewX may return an application-level error via
	// "appStatusCode"/"message" or a null "response" even on a 2xx body.
	var responseObj map[string]interface{}
	if err := json.Unmarshal(body, &responseObj); err != nil {
		return fmt.Errorf("unable to parse push response: %v", err)
	}

	message, _ := responseObj["message"].(string)
	appStatusCode, _ := responseObj["appStatusCode"].(string)
	// appStatusCode "" or "0" both indicate success for the submission response.
	// Any other non-empty value is an application-level error.
	if appStatusCode != "" && appStatusCode != "0" {
		return fmt.Errorf("gcp certificate push failed: %s (appStatusCode=%s)", message, appStatusCode)
	}
	if responseObj["response"] == nil {
		return fmt.Errorf("gcp certificate push failed: %s", message)
	}

	// AppViewX returns the "response" array as either
	//   [{"requestId":..,"connectorId":..}]  – trackable via polling, or
	//   ["<connectorId>"]                    – accepted but not trackable
	//                                          (often accompanied by an exception message).
	requestID, connectorID := extractPushIDs(responseObj["response"])
	d.Set(constants.REQUEST_ID, requestID)
	d.Set(constants.CONNECTOR_ID, connectorID)
	logger.Info("GCP push message: %s", message)

	// Optionally wait for completion by polling the request status.
	if d.Get(constants.WAIT_FOR_COMPLETION).(bool) {
		if requestID == "" {
			logger.Warn("wait_for_completion is set but AppViewX did not return a requestId; the push was accepted but is not trackable - skipping wait")
		} else {
			waitTimeout := d.Get(constants.WAIT_TIMEOUT_SECONDS).(int)
			pollInterval := d.Get(constants.POLL_INTERVAL_SECONDS).(int)
			if err := waitForRequestCompletion(configAppViewXEnvironment, appviewxSessionID, accessToken, appviewxGwSource, requestID, "gcp push", waitTimeout, pollInterval); err != nil {
				return err
			}
		}
	}

	// Construct and expose the GCP certificate id.
	project := d.Get(constants.GCP_PROJECT).(string)
	gcpCertID := constructGCPCertificateID(project, opts.Location, opts.CertificateName)
	d.Set(constants.CERT_MANAGER_CERTIFICATE_ID, gcpCertID)
	d.Set(constants.SUCCESS, true)
	d.SetId(gcpCertID)

	return nil
}

// waitForRequestCompletion polls an AppViewX workflow requestId until it completes
// (success or failure) or the timeout elapses.
func waitForRequestCompletion(cfg *config.AppViewXEnvironment, sessionID, accessToken, gwSource, requestID, label string, waitTimeoutSeconds, pollIntervalSeconds int) error {
	logger.Info("Waiting for %s completion (requestId=%s, timeout=%ds)", label, requestID, waitTimeoutSeconds)
	deadline := time.Now().Add(time.Duration(waitTimeoutSeconds) * time.Second)

	for {
		statusCode, body, err := pollWorkflowStatus(cfg.AppViewXEnvironmentIP, cfg.AppViewXEnvironmentPort, cfg.AppViewXIsHTTPS, sessionID, accessToken, requestID, gwSource)
		if err != nil {
			return fmt.Errorf("error polling %s status: %v", label, err)
		}
		if statusCode < 200 || statusCode >= 300 {
			return fmt.Errorf("error polling %s status, http %d: %s", label, statusCode, string(body))
		}

		var responseObj map[string]interface{}
		if err := json.Unmarshal(body, &responseObj); err != nil {
			return fmt.Errorf("error parsing %s status response: %v", label, err)
		}
		code, completed := getWorkflowStatusCode(responseObj)
		if completed {
			if code == STATUS_SUCCESS {
				logger.Info("%s completed successfully (requestId=%s)", label, requestID)
				return nil
			}
			return fmt.Errorf("%s failed with status code %d (requestId=%s): %s", label, code, requestID, string(body))
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("timed out after %ds waiting for %s to complete (requestId=%s)", waitTimeoutSeconds, label, requestID)
		}
		time.Sleep(time.Duration(pollIntervalSeconds) * time.Second)
	}
}
