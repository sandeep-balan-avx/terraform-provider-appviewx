package appviewx

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"terraform-provider-appviewx/appviewx/config"
	"terraform-provider-appviewx/appviewx/constants"
)

func ResourceAutomationServer() *schema.Resource {
	return &schema.Resource{
		Create: resourceAutomationServerCreate,
		Read:   resourceAutomationServerRead,
		Update: resourceAutomationServerUpdate,
		Delete: resourceAutomationServerDelete,

		Schema: map[string]*schema.Schema{
			constants.APPVIEWX_ACTION_ID: {
				Type:     schema.TypeString,
				Required: true,
			},
			constants.PAYLOAD: {
				Type:      schema.TypeString,
				Optional:  true,
				Sensitive: true,
			},
			constants.HEADERS: {
				Type:      schema.TypeMap,
				Optional:  true,
				Sensitive: true,
			},
			constants.MASTER_PAYLOAD: {
				Type:      schema.TypeString,
				Optional:  true,
				Sensitive: true,
			},
			constants.QUERY_PARAMS: {
				Type:     schema.TypeMap,
				Optional: true,
			},
			constants.CERTIFICATE_DOWNLOAD_PATH: {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func resourceAutomationServerRead(d *schema.ResourceData, m interface{}) error {
	log.Println("[INFO] **************** GET OPERATION NOT SUPPORTED FOR THIS RESOURCE **************** ")
	// Since the resource is for stateless operation, only nil returned
	return nil
}

func resourceAutomationServerUpdate(d *schema.ResourceData, m interface{}) error {
	log.Println("[INFO]  **************** UPDATE OPERATION NOT SUPPORTED FOR THIS RESOURCE **************** ")
	//Update implementation is empty since this resource is for the stateless generic api invocation
	return errors.New("Update not supported")
}

func resourceAutomationServerDelete(d *schema.ResourceData, m interface{}) error {
	log.Println("[INFO]  **************** DELETE OPERATION NOT SUPPORTED FOR THIS RESOURCE **************** ")
	// Delete implementation is empty since this resource is for the stateless generic api invocation
	d.SetId("")
	return nil
}

func resourceAutomationServerCreate(d *schema.ResourceData, m interface{}) error {
	configAppViewXEnvironment := m.(*config.AppViewXEnvironment)

	d.Partial(true)

	log.Println("[DEBUG] *********************** Request received to create")
	appviewxUserName := configAppViewXEnvironment.AppViewXUserName
	appviewxPassword := configAppViewXEnvironment.AppViewXPassword
	appviewxClientId := configAppViewXEnvironment.AppViewXClientId
	appviewxClientSecret := configAppViewXEnvironment.AppViewXClientSecret
	appviewxEnvironmentIP := configAppViewXEnvironment.AppViewXEnvironmentIP
	appviewxEnvironmentPort := configAppViewXEnvironment.AppViewXEnvironmentPort
	appviewxEnvironmentIsHTTPS := configAppViewXEnvironment.AppViewXIsHTTPS
	appviewxGwSource := "WEB"

	var appviewxSessionID, accessToken string
	var err error

	// Try username/password authentication first
	if appviewxUserName != "" && appviewxPassword != "" {
		appviewxSessionID, err = GetSession(appviewxUserName, appviewxPassword, appviewxEnvironmentIP, appviewxEnvironmentPort, appviewxGwSource, appviewxEnvironmentIsHTTPS)
		if err != nil {
			log.Println("[ERROR] Error in getting the session due to : ", err)
			// Don't return error here, try client ID/secret authentication
		}
	}

	// If username/password authentication failed or wasn't provided, try client ID/secret
	if appviewxSessionID == "" && appviewxClientId != "" && appviewxClientSecret != "" {
		accessToken, err = GetAccessTokenWithRotation(configAppViewXEnvironment, appviewxGwSource)
		if err != nil {
			log.Println("[ERROR] Error in getting the access token due to : ", err)
			return err
		}
	}

	// If both authentication methods failed, return error
	if appviewxSessionID == "" && accessToken == "" {
		return errors.New("authentication failed - provide either username/password or client ID/secret in Terraform File or in the Environment Variables:[APPVIEWX_TERRAFORM_CLIENT_ID, APPVIEWX_TERRAFORM_CLIENT_SECRET]")
	}

	types := constants.POST

	actionID := d.Get(constants.APPVIEWX_ACTION_ID).(string)
	payloadString := d.Get(constants.PAYLOAD).(string)

	var masterPayloadFileName string
	if v, ok := d.GetOk(constants.MASTER_PAYLOAD); ok {
		masterPayloadFileName = v.(string)
	} else {
		masterPayloadFileName = "./payload.json"
	}

	log.Println("[DEBUG] Input minimal payload : ", payloadString)

	payloadMinimal := make(map[string]interface{})
	err = json.Unmarshal([]byte(payloadString), &payloadMinimal)
	if err != nil {
		log.Println("[ERROR] error in unmarshalling the payloadString", payloadString)
		return err
	}

	masterPayload := GetMasterPayloadApplyingMinimalPayload(masterPayloadFileName, payloadMinimal)
	log.Println("[DEBUG] masterPayload : ", masterPayload)

	queryParams := make(map[string]string)
	queryParams[constants.GW_SOURCE] = appviewxGwSource

	var queryParamReceived map[string]interface{}
	if v, ok := d.GetOk(constants.QUERY_PARAMS); ok {
		queryParamReceived = v.(map[string]interface{})
		for k, v := range queryParamReceived {
			queryParams[k] = v.(string)
		}
	}

	url := GetURL(appviewxEnvironmentIP, appviewxEnvironmentPort, actionID, queryParams, appviewxEnvironmentIsHTTPS)

	headers := make(map[string]interface{})
	if v, ok := d.GetOk(constants.HEADERS); ok {
		headers = v.(map[string]interface{})
	}

	if len(headers) == 0 {
		headers["Content-Type"] = "application/json"
		headers["Accept"] = "application/json"
	}

	client := &http.Client{Transport: HTTPTransport()}
	masterPayload[constants.APPVIEWX_ACTION_ID] = actionID
	requestBody, err := json.Marshal(masterPayload)
	if err != nil {
		log.Println("[ERROR] error in Marshalling the masterPayload", masterPayload, err)
		return err
	}

	printRequest(types, url, headers, requestBody)

	req, err := http.NewRequest(types, url, bytes.NewBuffer(requestBody))
	if err != nil {
		log.Println("[ERROR] error in creating the new request ", err)
		return err
	}

	for key, value := range headers {
		value1 := fmt.Sprintf("%v", value)
		key1 := fmt.Sprintf("%v", key)
		req.Header.Add(key1, value1)
	}

	// Add appropriate authentication header
	if appviewxSessionID != "" {
		log.Printf("[DEBUG] Using session ID for authentication")
		req.Header.Set(constants.SESSION_ID, appviewxSessionID)
	} else if accessToken != "" {
		log.Printf("[DEBUG] Using access token for authentication")
		req.Header.Set(constants.TOKEN, accessToken)
	}

	// Debug headers (already a map, so just print as is)
	// headersBytes, _ := json.MarshalIndent(req.Header, "", "  ")
	// log.Println("[DEBUG] 🏷️  Request headers:\n", string(headersBytes))

	resp, err := client.Do(req)
	if err != nil {
		log.Println("[ERROR] error in making http request", err)
		return err
	} else {
		log.Println("[DEBUG] Request success : url :", url)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("\n[ERROR] ❌ Error in reading the response body:")
		log.Println("   ", err)
		log.Println("--------------------------------------------------------------")
		return err
	}

	// Safely access the download path
	var downloadFilePath string
	if v, ok := d.GetOk(constants.CERTIFICATE_DOWNLOAD_PATH); ok {
		downloadFilePath = v.(string)
	}

	if downloadFilePath != "" {
		log.Println("downloadFilePath : ", downloadFilePath)
		err = ioutil.WriteFile(downloadFilePath, body, 0777)
		if err != nil {
			log.Println("[ERROR] error in writing the contents to file", err)
			return err
		}
	} else {
		log.Println("[DEBUG] downloadFilePath is empty")
	}

	// Pretty print response body if JSON
	var prettyResp bytes.Buffer
	if json.Indent(&prettyResp, body, "", "  ") == nil {
		log.Println("\n[DEBUG] 📦 Response body:\n", prettyResp.String())
	} else {
		log.Println("\n[DEBUG] 📦 Response body (raw):\n", string(body))
	}

	log.Println("[DEBUG] API invoke success")
	d.SetId(strconv.Itoa(rand.Int()))

	return resourceAutomationServerRead(d, m)
}
