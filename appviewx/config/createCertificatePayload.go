package config

type CreateCertificatePayload struct {
	CaConnectorInfo  CAConnectionInfo `json:"caConnectorInfo"`
	CertificateGroup CertificateGroup `json:"certificateGroup"`
}

type CertificateGroup struct {
	CertificateGroupName string `json:"name"`
}

type CAConnectionInfo struct {
	CertificateAuthority  string            `json:"certificateAuthority"`
	CertificateType       string            `json:"certificateType"`
	CASettingName         string            `json:"caSettingName"`
	CAConnectorName       string            `json:"name"`
	CSRParameters         CSRParameters     `json:"csrParameters"`
	ValidityInDays        int               `json:"validityInDays"`
	ValidityUnit          string            `json:"validityUnit"`
	ValidityUnitValue     int               `json:"validityUnitValue"`
	CustomAttributes      map[string]string `json:"certAttributes"`
	VendorSpecificfields  map[string]string `json:"vendorSpecificDetails"`
	IsAutoRenewal         string            `json:"isAutoRenewal,omitempty"`
	RenewBefore           string            `json:"renewBefore,omitempty"`
	AutoRegenerateEnabled *bool             `json:"autoRegenerateEnabled,omitempty"`
}

type CSRParameters struct {
	CommonName            string           `json:"commonName"`
	HashFunction          string           `json:"hashFunction"`
	KeyType               string           `json:"keyType"`
	BitLength             string           `json:"bitLength"`
	CertificateCategories []string         `json:"certificateCategories"`
	EnhancedSANTypes      EnhancedSANTypes `json:"enhancedSANTypes"`
}

type EnhancedSANTypes struct {
	DNSNames []string `json:"dNSNames"`
}

type AppviewxResponse struct {
	Response      string            `json:"response"`
	Message       string            `json:"message"`
	AppStatusCode string            `json:"appStatusCode"`
	Tags          map[string]string `json:"tags"`
	Headers       string            `json:"headers"`
}

type AppviewxCreateCertResponse struct {
	Response      map[string]string `json:"response"`
	Message       string            `json:"message"`
	AppStatusCode string            `json:"appStatusCode"`
	Tags          map[string]string `json:"tags"`
	Headers       string            `json:"headers"`
}

var CreateCertificateActionId = "certificate/create"
var HTTPMethodPost = "POST"
var HTTPMethodGet = "GET"
